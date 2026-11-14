package tass

// Stage 3C: a human assessment is write-once evidence, not a change to TASS behavior.
// It is private to the experiment's existing Identity owner and Engine link.
import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
	"time"
)

const (
	experimentFeedbackSchema = "thought-processor.experiment-feedback/1"
	experimentFeedbackFile = "feedback.json"
	experimentFeedbackMax = 16 << 10
	experimentCorrectionMax = 500
)

var (
	ErrExperimentFeedbackUnsafe = errors.New("the saved owner assessment is unsafe or damaged; nothing was trusted or changed")
	ErrExperimentFeedbackIncomplete = errors.New("both experiment arms and the frozen snapshot must be verified before owner assessment")
)

// ExperimentFeedbackInput is deliberately not allowed to select an owner, Thought or evidence.
type ExperimentFeedbackInput struct {
	Verdict string `json:"verdict"`
	Correction string `json:"correction"`
}

type ExperimentFeedback struct {
	Schema string `json:"schema"`
	ExperimentID string `json:"experiment_id"`
	OwnerEntityID string `json:"owner_entity_id"`
	EngineEntityID string `json:"engine_entity_id"`
	ThoughtID string `json:"thought_id"`
	SnapshotSHA string `json:"snapshot_sha256"`
	ARequestID string `json:"a_request_id"`
	BRequestID string `json:"b_request_id"`
	AReplySHA string `json:"a_reply_sha256"`
	BReplySHA string `json:"b_reply_sha256"`
	ADecisionID string `json:"a_decision_id"`
	BDecisionID string `json:"b_decision_id"`
	EvidenceSHA string `json:"evidence_sha256"`
	Verdict string `json:"verdict"`
	Correction string `json:"correction"`
	SavedAt time.Time `json:"saved_at"`
}

func (in ExperimentFeedbackInput) validate() error {
	switch in.Verdict {
	case "BETTER", "SAME", "WORSE", "INCONCLUSIVE":
	default:
		return fmt.Errorf("%w: select BETTER, SAME, WORSE or INCONCLUSIVE", ErrExperimentInvalid)
	}
	if !utf8.ValidString(in.Correction) || utf8.RuneCountInString(in.Correction) > experimentCorrectionMax {
		return fmt.Errorf("%w: explanation must be valid text of at most %d characters", ErrExperimentInvalid, experimentCorrectionMax)
	}
	for _, r := range in.Correction {
		if unicode.IsControl(r) && r != '\n' || unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Co, r) || unicode.Is(unicode.Cs, r) || r == utf8.RuneError {
			return fmt.Errorf("%w: explanation contains unsupported control or invisible characters", ErrExperimentInvalid)
		}
	}
	return nil
}

func feedbackEvidence(v ExperimentView) (ExperimentFeedback, error) {
	if v.Frozen.Status != "FROZEN" || !hex64RE.MatchString(v.Frozen.SHA256) || !v.Comparison.Ready || len(v.Arms) != 2 {
		return ExperimentFeedback{}, ErrExperimentFeedbackIncomplete
	}
	a, b := v.Arms[0], v.Arms[1]
	if a.Arm != "A" || b.Arm != "B" || a.State != "COMPLETED" || b.State != "COMPLETED" ||
		a.RequestID == "" || b.RequestID == "" || a.DecisionID == "" || b.DecisionID == "" ||
		a.RequestID == b.RequestID || a.DecisionID == b.DecisionID ||
		!hex64RE.MatchString(a.ReplySHA) || !hex64RE.MatchString(b.ReplySHA) ||
		a.SnapshotSHA != v.Frozen.SHA256 || b.SnapshotSHA != v.Frozen.SHA256 {
		return ExperimentFeedback{}, ErrExperimentFeedbackIncomplete
	}
	f := ExperimentFeedback{Schema: experimentFeedbackSchema, ExperimentID: v.ExperimentID,
		OwnerEntityID: v.OwnerEntityID, EngineEntityID: v.EngineEntityID, ThoughtID: v.ThoughtID,
		SnapshotSHA: v.Frozen.SHA256, ARequestID: a.RequestID, BRequestID: b.RequestID,
		AReplySHA: a.ReplySHA, BReplySHA: b.ReplySHA,
		ADecisionID: a.DecisionID, BDecisionID: b.DecisionID}
	// Hash only identifiers and evidence digests, not Thought/Note/prompt/reply content.
	data := strings.Join([]string{f.ExperimentID, f.OwnerEntityID, f.EngineEntityID, f.ThoughtID,
		f.SnapshotSHA, f.ARequestID, f.BRequestID, f.AReplySHA, f.BReplySHA, f.ADecisionID, f.BDecisionID}, "\x00")
	sum := sha256.Sum256([]byte(data))
	f.EvidenceSHA = hex.EncodeToString(sum[:])
	return f, nil
}

func (c *Core) readExperimentFeedback(v ExperimentView) (*ExperimentFeedback, error) {
	ex, err := c.expTree(v.ExperimentID, false)
	if err != nil { return nil, ErrExperimentFeedbackUnsafe }
	if !ex { return nil, nil }
	b, exists, err := expPrivateRead(c.expRunPath(v.ExperimentID, experimentFeedbackFile), experimentFeedbackMax)
	if err != nil { return nil, ErrExperimentFeedbackUnsafe }
	if !exists { return nil, nil }
	if len(b) < 2 || b[len(b)-1] != '\n' || bytes.Count(b, []byte{'\n'}) != 1 {
		return nil, ErrExperimentFeedbackUnsafe
	}
	var f ExperimentFeedback
	if json.Unmarshal(b[:len(b)-1], &f) != nil || f.Schema != experimentFeedbackSchema || (ExperimentFeedbackInput{f.Verdict, f.Correction}).validate() != nil || f.SavedAt.IsZero() {
		return nil, ErrExperimentFeedbackUnsafe
	}
	// The writer uses json.Marshal + newline. Reject extra, duplicate or unknown keys,
	// altered representation, appended JSON and partial records rather than normalizing them.
	canonical, err := json.Marshal(f)
	if err != nil || !bytes.Equal(b, append(canonical, '\n')) {
		return nil, ErrExperimentFeedbackUnsafe
	}
	want, e := feedbackEvidence(v)
	if e != nil || f.ExperimentID != want.ExperimentID || f.OwnerEntityID != want.OwnerEntityID ||
		f.EngineEntityID != want.EngineEntityID || f.ThoughtID != want.ThoughtID ||
		f.SnapshotSHA != want.SnapshotSHA || f.ARequestID != want.ARequestID || f.BRequestID != want.BRequestID ||
		f.AReplySHA != want.AReplySHA || f.BReplySHA != want.BReplySHA ||
		f.ADecisionID != want.ADecisionID || f.BDecisionID != want.BDecisionID ||
		f.EvidenceSHA != want.EvidenceSHA {
		return nil, ErrExperimentFeedbackUnsafe
	}
	return &f, nil
}

// SaveExperimentFeedback records one assessment. A second identical request is idempotent;
// an attempted change to an existing verdict or correction is a conflict.
func (c *Core) SaveExperimentFeedback(id string, in ExperimentFeedbackInput, rq ...*CognitionReq) (ExperimentFeedback, bool, error) {
	if err := in.validate(); err != nil { return ExperimentFeedback{}, false, err }
	v, err := c.ExperimentByID(id, rq...)
	if err != nil { return ExperimentFeedback{}, false, err }
	expected, err := feedbackEvidence(v)
	if err != nil { return ExperimentFeedback{}, false, err }
	// A Cognition response may have been authorized earlier; check Identity again before saving.
	scope, err := c.cognitionAccess(rq...)
	if err != nil { return ExperimentFeedback{}, false, err }
	if scope.identityID != v.OwnerEntityID || scope.engineID != v.EngineEntityID {
		return ExperimentFeedback{}, false, ErrExperimentNotFound
	}
	if err := c.CognitionRevalidate(scope.attested()); err != nil { return ExperimentFeedback{}, false, err }
	c.expMu.Lock()
	defer c.expMu.Unlock()
	old, err := c.readExperimentFeedback(v)
	if err != nil { return ExperimentFeedback{}, false, err }
	if old != nil {
		if old.Verdict == in.Verdict && old.Correction == in.Correction { return *old, false, nil }
		return ExperimentFeedback{}, false, ErrExperimentConflict
	}
	expected.Verdict, expected.Correction, expected.SavedAt = in.Verdict, in.Correction, c.now().UTC()
	created, err := c.expCreate(id, experimentFeedbackFile, expected)
	if err != nil { return ExperimentFeedback{}, false, fmt.Errorf("%w: %v", ErrExperimentFeedbackUnsafe, err) }
	if !created {
		// Another writer won. Re-read; never silently overwrite or claim a save.
		old, err := c.readExperimentFeedback(v)
		if err != nil || old == nil { return ExperimentFeedback{}, false, ErrExperimentFeedbackUnsafe }
		if old.Verdict == in.Verdict && old.Correction == in.Correction { return *old, false, nil }
		return ExperimentFeedback{}, false, ErrExperimentConflict
	}
	return expected, true, nil
}

// Avoid returning disk contents when the private file or its evidence cannot be trusted.
func (c *Core) experimentFeedbackForView(v *ExperimentView) error {
	// Preserve Stage 3B's read-only UNVERIFIED / RECOVERY_REQUIRED diagnostics.
	// No assessment is trusted or displayed unless the underlying evidence is verified.
	if _, err := feedbackEvidence(*v); err != nil {
		return nil
	}
	f, err := c.readExperimentFeedback(*v)
	if err != nil { return err }
	v.Feedback = f
	return nil
}

