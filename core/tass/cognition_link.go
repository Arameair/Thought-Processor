package tass

// Cognition identity correlation (#82 P0-A).
//
// The Thought Engine keeps its own entity IDs and finds/creates them by display name; it does not
// record the canonical Identity entity ID. A matching name is therefore NOT evidence that Engine
// history belongs to the current Identity entity (a rename, a recreated entity or a second identity
// with the same name would inherit someone else's history).
//
// So the Core keeps an explicit, owner-reviewed link: canonical Identity entity ID → Engine entity
// ID. Without a link, Cognition shows no Engine Thoughts or cycles (fail closed) and lists the
// Engine entities only as metadata the owner can review and link. Each link records the Identity
// decision that authorized it. Unlink removes it. Old journal files are never rewritten.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Arameair/Thought-Processor/core/engine"
)

type CognitionLink struct {
	IdentityEntityID string    `json:"identity_entity_id"`
	EngineEntityID   string    `json:"engine_entity_id"`
	EngineName       string    `json:"engine_display_name"` // as shown when linked (informational only)
	LinkedAt         time.Time `json:"linked_at"`
	DecisionID       string    `json:"identity_decision_id"`
	By               string    `json:"by"` // always "owner"
}

type cognitionLinks struct {
	Links []CognitionLink `json:"links"`
}

// CognitionLinkState is what Cognition shows about the correlation.
type CognitionLinkState struct {
	State          string            `json:"state"` // LINKED | NOT LINKED | LINK BROKEN
	Link           *CognitionLink    `json:"link,omitempty"`
	Candidates     []CognitionEngine `json:"candidates"` // Engine entities the owner may review (metadata only)
	UnscopedCycles int               `json:"unscoped_cycles"`
	OtherCycles    int               `json:"other_entity_cycles"`
	Explanation    string            `json:"explanation"`
}

type CognitionEngine struct {
	EngineEntityID  string `json:"engine_entity_id"`
	DisplayName     string `json:"display_name"`
	SameName        bool   `json:"same_display_name"` // a hint for review, never proof
	LinkedElsewhere bool   `json:"linked_to_other_identity"`
}

func (c *Core) loadCognitionLinks() cognitionLinks {
	var l cognitionLinks
	if b, err := os.ReadFile(c.file("cognition_links.json")); err == nil {
		_ = json.Unmarshal(b, &l)
	}
	return l
}

func (c *Core) saveCognitionLinks(l cognitionLinks) error {
	b, _ := json.MarshalIndent(l, "", "  ")
	tmp := c.file("cognition_links.json.tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, c.file("cognition_links.json"))
}

func linkFor(l cognitionLinks, identityID string) *CognitionLink {
	for i := range l.Links {
		if l.Links[i].IdentityEntityID == identityID {
			return &l.Links[i]
		}
	}
	return nil
}

var ErrCognitionLink = errors.New("Cognition link refused")

// LinkCognition is an explicit owner action (local page token): link the current, unlocked
// Identity entity to one Engine entity. One Engine entity can belong to only one Identity entity.
func (c *Core) LinkCognition(engineID string) (CognitionLink, error) {
	if c.cfg.IdentitySvc == nil || !engine.CognitionID.MatchString(engineID) {
		return CognitionLink{}, ErrCognitionLink
	}
	if c.NudgeActive() { // Stage 2 (A-11): never relink under a submitted or running nudge
		return CognitionLink{}, fmt.Errorf("%w: a nudge is being processed; wait for it or stop it first", ErrCognitionLink)
	}
	st := c.cfg.IdentitySvc.State()
	if st.State != "UNLOCKED" || st.Entity == nil || st.Entity.EntityID == "" {
		return CognitionLink{}, fmt.Errorf("%w: unlock Identity first", ErrCognitionLink)
	}
	d := c.cfg.IdentitySvc.Authorize("thought.process", "thought-processor:cognition.link "+engineID)
	if !d.Allow || d.DecisionID == "" || d.EntityID != st.Entity.EntityID {
		return CognitionLink{}, fmt.Errorf("%w: Identity did not allow it", ErrCognitionLink)
	}
	status, err := c.engine.CognitionStatus()
	if err != nil {
		return CognitionLink{}, ErrCognitionUnavailable
	}
	name := ""
	found := false
	for _, e := range status.Entities {
		if e.Card.EntityID == engineID && e.Card.EntityType == "ai_companion" {
			found, name = true, e.Card.DisplayName
		}
	}
	if !found {
		return CognitionLink{}, fmt.Errorf("%w: the Engine has no AI companion entity %s", ErrCognitionLink, engineID)
	}
	c.cogMu.Lock()
	defer c.cogMu.Unlock()
	l := c.loadCognitionLinks()
	for _, x := range l.Links {
		if x.EngineEntityID == engineID && x.IdentityEntityID != st.Entity.EntityID {
			return CognitionLink{}, fmt.Errorf("%w: that Engine entity is linked to another Identity entity", ErrCognitionLink)
		}
	}
	kept := l.Links[:0:0]
	for _, x := range l.Links {
		if x.IdentityEntityID != st.Entity.EntityID {
			kept = append(kept, x)
		}
	}
	link := CognitionLink{IdentityEntityID: st.Entity.EntityID, EngineEntityID: engineID, EngineName: cognitionClip(name, 80),
		LinkedAt: c.now().UTC(), DecisionID: d.DecisionID, By: "owner"}
	l.Links = append(kept, link)
	if err := c.saveCognitionLinks(l); err != nil {
		return CognitionLink{}, err
	}
	c.appendLog("cognition_links.jsonl", map[string]any{"at": link.LinkedAt, "action": "link", "link": link})
	return link, nil
}

// UnlinkCognition removes the current Identity entity's link (owner action).
func (c *Core) UnlinkCognition() error {
	if c.NudgeActive() {
		return fmt.Errorf("%w: a nudge is being processed; wait for it or stop it first", ErrCognitionLink)
	}
	if c.cfg.IdentitySvc == nil {
		return ErrCognitionLink
	}
	st := c.cfg.IdentitySvc.State()
	if st.State != "UNLOCKED" || st.Entity == nil {
		return fmt.Errorf("%w: unlock Identity first", ErrCognitionLink)
	}
	c.cogMu.Lock()
	defer c.cogMu.Unlock()
	l := c.loadCognitionLinks()
	kept := l.Links[:0:0]
	for _, x := range l.Links {
		if x.IdentityEntityID != st.Entity.EntityID {
			kept = append(kept, x)
		}
	}
	l.Links = kept
	if err := c.saveCognitionLinks(l); err != nil {
		return err
	}
	c.appendLog("cognition_links.jsonl", map[string]any{"at": c.now().UTC(), "action": "unlink", "identity_entity_id": st.Entity.EntityID})
	return nil
}

// resolveEngineEntity: the linked Engine entity for this Identity entity, verified to exist.
func (c *Core) resolveEngineEntity(identityID, name string, status engine.CognitionStatus) (string, CognitionLinkState) {
	c.cogMu.Lock()
	l := c.loadCognitionLinks()
	c.cogMu.Unlock()
	out := CognitionLinkState{State: "NOT LINKED", Candidates: []CognitionEngine{}}
	linkedTo := map[string]string{}
	for _, x := range l.Links {
		linkedTo[x.EngineEntityID] = x.IdentityEntityID
	}
	for _, e := range status.Entities {
		if e.Card.EntityType != "ai_companion" || !engine.CognitionID.MatchString(e.Card.EntityID) {
			continue
		}
		other := linkedTo[e.Card.EntityID] != "" && linkedTo[e.Card.EntityID] != identityID
		out.Candidates = append(out.Candidates, CognitionEngine{EngineEntityID: e.Card.EntityID, DisplayName: cognitionClip(e.Card.DisplayName, 80),
			SameName: strings.EqualFold(e.Card.DisplayName, name), LinkedElsewhere: other})
	}
	link := linkFor(l, identityID)
	if link == nil {
		out.Explanation = "Engine history is not linked to this Identity entity yet. A matching name is not proof; review the Engine entity below and link it yourself. Until then no Engine Thoughts or cycles are shown."
		return "", out
	}
	cp := *link
	out.Link = &cp
	for _, e := range status.Entities {
		if e.Card.EntityID == link.EngineEntityID && e.Card.EntityType == "ai_companion" {
			out.State = "LINKED"
			out.Explanation = fmt.Sprintf("Linked by the owner on %s (Identity decision %s).", link.LinkedAt.Format("2006-01-02 15:04 MST"), link.DecisionID)
			return link.EngineEntityID, out
		}
	}
	out.State = "LINK BROKEN"
	out.Explanation = "The linked Engine entity no longer exists in the Engine. Nothing is shown until the owner links again."
	return "", out
}
