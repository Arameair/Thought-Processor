package workloop

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Arameair/Thought-Processor/core/github"
	"github.com/Arameair/Thought-Processor/core/tassmd"
)

// TASS is a SIMULATED TASS participant: it reads its current assignment at an exact
// ledger revision and prepares one acknowledgment. It runs no model and holds no real
// credentials. Its private memory lives only in Dir and is never published.
//
// It publishes to team/TASS.md only when Granted() says yes. Granted is the stand-in for
// an Identity decision: a local function, never anything read from GitHub.
type TASS struct {
	Team    *Team  // Actor "tass"
	Dir     string // local, private: memory.md, receipts.json, conflicts/
	Watch   string // the workstream whose assignment TASS follows
	Granted func() bool
}

// Receipt is TASS's durable local record of one acknowledgment (metadata only).
type Receipt struct {
	Task      string `json:"task"`
	InputRev  string `json:"input_revision"` // blob sha of the ledger TASS read
	Phase     string `json:"phase"`
	Commit    string `json:"commit,omitempty"`
	Stage     string `json:"stage"` // PREPARED | PUBLISHED | CONFLICT
	OutputRev string `json:"output_revision,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

func (o *TASS) receipts() (map[string]Receipt, error) {
	m := map[string]Receipt{}
	raw, err := os.ReadFile(filepath.Join(o.Dir, "receipts.json"))
	if errors.Is(err, os.ErrNotExist) {
		return m, nil
	}
	if err != nil {
		return nil, err
	}
	return m, json.Unmarshal(raw, &m)
}

func (o *TASS) save(m map[string]Receipt) error {
	raw, _ := json.MarshalIndent(m, "", "  ")
	tmp := filepath.Join(o.Dir, "receipts.json.tmp")
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(o.Dir, "receipts.json"))
}

// Sections of team/TASS.md that TASS writes. Everything else belongs to others.
const (
	statusSection = "Work status"
	outboxSection = "Outbox"
)

// AckText is the public-safe acknowledgment: assignment facts only, never memory.
func AckText(s State, rev string) string {
	return fmt.Sprintf("- TASS read `%s` at ledger revision `%s`: phase `%s`, candidate %s. "+
		"Acknowledged as observer: I report, I don't act. Markdown here grants me nothing.\n",
		s.ID, short(rev), s.Phase, code(s.Commit))
}

func setSection(doc, name, body string) string {
	secs := tassmd.Parse(doc)
	for i := range secs {
		if secs[i].Name == name {
			secs[i].Body = body
			return tassmd.Render(secs)
		}
	}
	if !strings.HasSuffix(doc, "\n") {
		doc += "\n"
	}
	return doc + "\n## " + name + "\n" + body
}

// Pulse is TASS's 20-second tick. It does nothing (no request beyond 304s, no write)
// unless the watched ledger changed and no receipt exists for its revision.
func (o *TASS) Pulse() (r Receipt, acted bool, err error) {
	if _, err := o.Team.Pulse(LedgerPath(o.Watch), TASSPath); err != nil {
		return r, false, err
	}
	evs, s, lf, err := o.Team.Ledger(o.Watch)
	if err != nil {
		return r, false, err
	}
	_ = evs
	if s.Roles[Observer] != o.Team.Actor {
		return r, false, nil // no assignment for TASS here
	}
	all, err := o.receipts()
	if err != nil {
		return r, false, err
	}
	key := o.Watch + "@" + lf.SHA
	if old, ok := all[key]; ok && (old.Stage == "PUBLISHED" || old.Stage == "CONFLICT" || !o.Granted()) {
		return old, false, nil // already handled this exact revision: idempotent across pulses and restarts
	}
	r = Receipt{Task: o.Watch, InputRev: lf.SHA, Phase: s.Phase, Commit: s.Commit, Stage: "PREPARED"}
	ack := AckText(s, lf.SHA)
	if !o.Granted() {
		r.Detail = "no grant: prepared locally, not published"
		all[key] = r
		return r, true, o.save(all)
	}
	r, err = o.publish(r, s, ack)
	all[key] = r
	if serr := o.save(all); err == nil {
		err = serr
	}
	return r, true, err
}

// publish updates only TASS's own sections, merging with others' edits through Core's
// tass.md three-way merge. A section changed on both sides is a CONFLICT: both copies
// are kept locally and nothing is overwritten.
func (o *TASS) publish(r Receipt, s State, ack string) (Receipt, error) {
	basePath := filepath.Join(o.Dir, "tass.base.md")
	for try := 0; try < maxTries; try++ {
		remote, err := o.Team.fresh(TASSPath)
		if err != nil {
			return r, err
		}
		base := remote.Text
		if b, err := os.ReadFile(basePath); err == nil {
			base = string(b)
		}
		local := setSection(setSection(base, statusSection,
			fmt.Sprintf("\n- Task: `%s` · phase `%s` · input revision `%s`\n\n", s.ID, s.Phase, short(r.InputRev))),
			outboxSection, "\n"+ack+"\n")
		merged, mine, conflicts := tassmd.Merge(base, local, remote.Text)
		if len(conflicts) > 0 {
			dir := filepath.Join(o.Dir, "conflicts")
			_ = os.MkdirAll(dir, 0o700)
			_ = os.WriteFile(filepath.Join(dir, short(remote.SHA)+"-github.md"), []byte(remote.Text), 0o600)
			_ = os.WriteFile(filepath.Join(dir, short(remote.SHA)+"-tass.md"), []byte(mine), 0o600)
			r.Stage, r.Detail = "CONFLICT", "both sides changed "+strings.Join(conflicts, ", ")+"; kept both, wrote nothing"
			return r, nil
		}
		sha, err := o.Team.put(TASSPath, merged, remote.SHA, "TASS: acknowledge "+s.ID)
		if errors.Is(err, github.ErrConflict) {
			continue
		}
		if err != nil {
			return r, err
		}
		if err := os.WriteFile(basePath, []byte(merged), 0o600); err != nil {
			return r, err
		}
		r.Stage, r.OutputRev = "PUBLISHED", sha
		return r, nil
	}
	return r, fmt.Errorf("workloop: TASS.md: gave up after %d conflicting writes", maxTries)
}
