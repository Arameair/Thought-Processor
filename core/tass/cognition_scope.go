package tass

// P0 Cognition lifecycle (#106, contract v2 §2): scope ATTESTATION and NARROWING.
//
// Identity decides who may read; Core resolves the owner-reviewed Identity → Engine link. A Cognition
// response is only meaningful for the scope it was authorized under, so Core says which one:
//
//   Identity  the unlocked Identity entity Core authorized the request for
//   Engine    the linked Engine entity, or UNLINKED
//   Link      a discriminator of the owner's link record (its Identity decision ID), or NONE, so an
//             unlink + relink of the same pair is a different scope
//
// A page may send the scope it believes is current. That is a NARROWING precondition only: Core
// still runs every normal Identity check, and a mismatch is refused (ErrCognitionScopeMismatch, HTTP
// 412) before any record is read or any write starts. It never grants anything; requests without an
// expectation (non-browser callers) are handled as before.
//
// After a read, CognitionRevalidate checks at the response boundary that the read is STILL authorized
// and its scope still current, so Core never answers with data it can see is no longer permitted:
// Identity state (unlocked, same entity), the local link file, and then Identity's own decision for
// both read permissions again. Identity is the only grant authority and its authoritative check is
// Authorize, which records each decision: a successful read costs 4 Identity decisions (2 at
// admission, 2 here). Core does not interpret the grant list in Identity's public state itself.

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
)

const (
	CognitionUnlinked = "UNLINKED"
	CognitionNoLink   = "NONE"
)

type CognitionScope struct {
	Identity, Engine, Link string
}

// CognitionReq optionally carries a page's expected scope (Want) and receives the scope Core
// authorized the request under (Got). Callers that pass none keep the previous behavior.
type CognitionReq struct {
	Want *CognitionScope
	Got  CognitionScope
}

var (
	ErrCognitionScopeMismatch = errors.New("the expected Identity entity or Engine link is not the current one")
	ErrCognitionScopeChanged  = errors.New("the Identity entity or Engine link changed while the read was in progress")
)

var cognitionScopeValue = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,128}$`)

// ValidCognitionScope: every part is a bounded, header-safe token.
func ValidCognitionScope(s CognitionScope) bool {
	return cognitionScopeValue.MatchString(s.Identity) && cognitionScopeValue.MatchString(s.Engine) && cognitionScopeValue.MatchString(s.Link)
}

func cognitionReqOf(rq []*CognitionReq) *CognitionReq {
	if len(rq) > 0 {
		return rq[0]
	}
	return nil
}

// linkDiscriminator names one owner link record; NONE when there is no link.
func linkDiscriminator(l *CognitionLink) string {
	if l == nil {
		return CognitionNoLink
	}
	if cognitionScopeValue.MatchString(l.DecisionID) {
		return l.DecisionID
	}
	h := sha256.Sum256([]byte(l.IdentityEntityID + "|" + l.EngineEntityID + "|" + l.LinkedAt.UTC().String()))
	return "link-" + hex.EncodeToString(h[:8])
}

func (s cognitionScope) attested() CognitionScope {
	eng := s.engineID
	if eng == "" {
		eng = CognitionUnlinked
	}
	return CognitionScope{Identity: s.identityID, Engine: eng, Link: linkDiscriminator(s.link.Link)}
}

// narrow records the authorized scope and refuses a request that expected a different one.
func (r *CognitionReq) narrow(got CognitionScope) error {
	if r == nil {
		return nil
	}
	if r.Want != nil && *r.Want != got {
		return ErrCognitionScopeMismatch
	}
	r.Got = got
	return nil
}

// cognitionReadActions are the Identity permissions every Cognition read needs, at admission and
// again at the response boundary.
var cognitionReadActions = []string{"conversation", "thought.process"}

// CognitionRevalidate is the response-boundary check: is the read still authorized, under the scope
// it was authorized under? A lock, a revoked permission or any refusal is a denial (403); a
// different entity or link is a scope change (412). Checks that need no Identity decision run first,
// so a scope change or lock adds no audit lines.
func (c *Core) CognitionRevalidate(s CognitionScope) error {
	if c.cfg.IdentitySvc == nil {
		return ErrCognitionDenied
	}
	st := c.cfg.IdentitySvc.State()
	if st.State != "UNLOCKED" || st.Entity == nil || st.Entity.EntityID == "" || st.Entity.Name == "" {
		return ErrCognitionDenied
	}
	if st.Entity.EntityID != s.Identity {
		return ErrCognitionScopeChanged
	}
	c.cogMu.Lock()
	l := c.loadCognitionLinks()
	c.cogMu.Unlock()
	if linkDiscriminator(linkFor(l, s.Identity)) != s.Link {
		return ErrCognitionScopeChanged
	}
	// Codex audit of ae344e7: an unlocked entity with an unchanged link can still have lost a read
	// permission while the Engine was answering. Only Identity can say; ask it again, for both.
	for _, action := range cognitionReadActions {
		d := c.cfg.IdentitySvc.Authorize(action, "thought-processor:cognition.read.release")
		if d.EntityID != "" && d.EntityID != s.Identity {
			return ErrCognitionScopeChanged
		}
		if !d.Allow || d.DecisionID == "" || d.EntityID != s.Identity {
			return ErrCognitionDenied
		}
	}
	return nil
}

// currentLinkPrecondition: for owner link changes, the precondition is the Identity entity and the current
// link record (which names its Engine entity). Caller holds cogMu.
func currentLinkPrecondition(rq []*CognitionReq, identityID string, l cognitionLinks) error {
	r := cognitionReqOf(rq)
	if r == nil || r.Want == nil {
		return nil
	}
	if r.Want.Identity != identityID || r.Want.Link != linkDiscriminator(linkFor(l, identityID)) {
		return ErrCognitionScopeMismatch
	}
	return nil
}
