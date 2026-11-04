// Package identity is the core's gate to Identity: every GitHub action TASS takes, and every
// person he obeys, is checked against a policy that lives in the Identity repo
// (Arameair/identity: policies/tass.json). Changing what TASS may do is a reviewed change there.
//
// The core keeps the last good policy locally, so it still knows its permissions offline.
// No policy at all means no GitHub actions: TASS keeps thinking locally, but does nothing on GitHub.
package identity

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Actions the core may ask about.
const (
	ModelRequest  = "model.request"        // think with the local model, through the Thought Engine
	ReadMessages  = "github.read_messages" // read issues/comments addressed to TASS
	Reply         = "github.reply"         // post TASS's answers as comments
	SyncTassMD    = "tass_md.sync"         // keep tass.md in step with its GitHub copy
	AskOnConflict = "tass_md.conflict_pr"  // open a pull request when both sides changed the same part
)

// Policy is policies/tass.json in the Identity repo.
type Policy struct {
	Identity string          `json:"identity"` // "TASS"
	Owner    string          `json:"owner"`    // GitHub login that owns this identity
	Obey     []string        `json:"obey"`     // logins whose messages TASS answers
	TassMD   string          `json:"tass_md"`  // owner/repo@branch:path of the GitHub copy
	Allow    map[string]bool `json:"allow"`    // action -> allowed
}

// Parse validates a policy file.
func Parse(b []byte) (*Policy, error) {
	var p Policy
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("identity policy is not valid: %w", err)
	}
	if p.Identity == "" || p.Owner == "" {
		return nil, fmt.Errorf("identity policy needs \"identity\" and \"owner\"")
	}
	if p.Allow == nil {
		p.Allow = map[string]bool{}
	}
	return &p, nil
}

// Allowed reports whether the action is granted. A nil policy allows nothing.
func (p *Policy) Allowed(action string) bool { return p != nil && p.Allow[action] }

// Obeys reports whether TASS answers this GitHub login.
func (p *Policy) Obeys(login string) bool {
	if p == nil {
		return false
	}
	for _, o := range p.Obey {
		if strings.EqualFold(o, login) {
			return true
		}
	}
	return false
}

// Target splits tass_md into repo, branch and path ("Arameair/TASS@live:tass.md").
func (p *Policy) Target() (repo, branch, path string, err error) {
	if p == nil {
		return "", "", "", fmt.Errorf("no identity policy")
	}
	repo, rest, ok1 := strings.Cut(p.TassMD, "@")
	branch, path, ok2 := strings.Cut(rest, ":")
	if !ok1 || !ok2 || repo == "" || branch == "" || path == "" || strings.Contains(path, "..") {
		return "", "", "", fmt.Errorf("identity policy tass_md must look like owner/repo@branch:path")
	}
	if strings.EqualFold(branch, "main") || strings.EqualFold(branch, "alpha") || strings.EqualFold(branch, "stable") {
		return "", "", "", fmt.Errorf("identity policy: tass.md may not sync straight onto %s; use its own branch", branch)
	}
	return repo, branch, path, nil
}
