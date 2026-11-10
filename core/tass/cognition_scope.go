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
// After a read, CognitionRevalidate checks at the response boundary that the scope is still current,
// so Core never answers with the old owner's data after it can see a switch. It is read-only: one
// Identity state request (Identity records no audit line for it) and the local link file.

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

// CognitionRevalidate is the response-boundary check: is the scope a read was authorized under still
// the current one? A lock or refusal is a denial; a different entity or link is a scope change.
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
