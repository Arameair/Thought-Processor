package idclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/Arameair/Thought-Processor/processor/internal/store"
)

// Pairing is what Processor keeps after the person approved the connection.
type Pairing struct {
	BaseURL          string    `json:"base_url"`
	ProcessorID      string    `json:"processor_id"`
	IdentitySystemID string    `json:"identity_system_id"`
	IdentityVersion  string    `json:"identity_version"`
	PairedAt         time.Time `json:"paired_at"`
	Token            string    `json:"-"` // the pairing secret: sealed at rest, never sent after pairing
}

type pairingFile struct {
	Pairing
	TokenProtection string `json:"token_protection"`
	SealedToken     string `json:"sealed_token"`
}

var tokenRE = regexp.MustCompile(`^[0-9a-f]{64}$`)

func pairingPath(profileDir string) string { return filepath.Join(profileDir, "identity-pairing.json") }

// LoadPairing returns the saved pairing, or (nil, nil) when there is none.
func LoadPairing(profileDir string) (*Pairing, error) {
	b, err := os.ReadFile(pairingPath(profileDir))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var f pairingFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, errors.New("identity-pairing.json is damaged; connect to Identity again")
	}
	if f.TokenProtection != store.SecretScheme {
		return nil, fmt.Errorf("the saved pairing was protected with %q but this computer uses %q; connect to Identity again", f.TokenProtection, store.SecretScheme)
	}
	sealed, err := base64.StdEncoding.DecodeString(f.SealedToken)
	if err != nil {
		return nil, errors.New("identity-pairing.json is damaged; connect to Identity again")
	}
	plain, err := store.Unprotect(sealed)
	if err != nil {
		return nil, err
	}
	if !tokenRE.Match(plain) {
		return nil, errors.New("the saved pairing could not be read back; connect to Identity again")
	}
	p := f.Pairing
	p.Token = string(plain)
	return &p, nil
}

func SavePairing(profileDir string, p Pairing) error {
	sealed, err := store.Protect([]byte(p.Token))
	if err != nil {
		return err
	}
	return store.WriteJSON(pairingPath(profileDir), pairingFile{Pairing: p, TokenProtection: store.SecretScheme,
		SealedToken: base64.StdEncoding.EncodeToString(sealed)})
}

func DeletePairing(profileDir string) error {
	err := os.Remove(pairingPath(profileDir))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// ---- discovery ----

// Locator is one running Identity's per-user locator file. It only says
// where to look — never trust.
type Locator struct {
	BaseURL          string `json:"base_url"`
	IdentitySystemID string `json:"identity_system_id"`
	PID              int    `json:"pid"`
	Profile          string `json:"profile"`
	ManagedBy        string `json:"managed_by"`
}

// LocatorDir mirrors Identity: %LOCALAPPDATA%\Arameair\Identity\instances on
// Windows; IDENTITY_LOCATOR_DIR overrides ("-" = off).
func LocatorDir() string {
	if v := os.Getenv("IDENTITY_LOCATOR_DIR"); v != "" {
		if v == "-" {
			return ""
		}
		return v
	}
	if runtime.GOOS == "windows" {
		if v := os.Getenv("LOCALAPPDATA"); v != "" {
			return filepath.Join(v, "Arameair", "Identity", "instances")
		}
	}
	if d, err := os.UserConfigDir(); err == nil {
		return filepath.Join(d, "arameair-identity", "instances")
	}
	return ""
}

func ReadLocators() []Locator {
	dir := LocatorDir()
	if dir == "" {
		return nil
	}
	files, _ := filepath.Glob(filepath.Join(dir, "idn_*.json"))
	var out []Locator
	for _, f := range files {
		var l Locator
		b, err := os.ReadFile(f)
		if err != nil || json.Unmarshal(b, &l) != nil {
			continue
		}
		if u, err := LoopbackOnly(l.BaseURL); err == nil {
			l.BaseURL = u
			out = append(out, l)
		}
	}
	return out
}

// Candidate is a running Identity found on this computer (unverified).
type Candidate struct {
	BaseURL          string `json:"base_url"`
	IdentitySystemID string `json:"identity_system_id,omitempty"` // from its locator (a claim)
	Profile          string `json:"profile,omitempty"`
	Version          string `json:"version"`
	API              string `json:"api"`
	Compatible       bool   `json:"compatible"`
}

// Discover lists Identities answering on this computer: every live locator,
// plus the default port and any extra address. Unverified by definition.
// DefaultProbeURL is the well-known address Processor also probes before it
// is paired (besides the locator folder). TP_IDENTITY_DEFAULT_URL overrides
// it; "-" turns the probe off (tests do this so a stranger's Identity on
// 127.0.0.1:7790 is never contacted). Only loopback addresses are accepted.
func DefaultProbeURL() string {
	v := os.Getenv("TP_IDENTITY_DEFAULT_URL")
	switch v {
	case "":
		return DefaultURL
	case "-":
		return ""
	}
	if u, err := LoopbackOnly(v); err == nil {
		return u
	}
	return ""
}

func Discover(ctx context.Context, extra ...string) []Candidate {
	seen := map[string]bool{}
	var out []Candidate
	try := func(u, sys, prof string) {
		if u == "" || seen[u] {
			return
		}
		seen[u] = true
		h, err := ProbeHealth(ctx, u)
		if err != nil {
			return
		}
		out = append(out, Candidate{BaseURL: u, IdentitySystemID: sys, Profile: prof, Version: h.Version, API: h.API, Compatible: Compatible(h.API)})
	}
	for _, l := range ReadLocators() {
		try(l.BaseURL, l.IdentitySystemID, l.Profile)
	}
	try(DefaultProbeURL(), "", "")
	for _, e := range extra {
		if u, err := LoopbackOnly(e); err == nil {
			try(u, "", "")
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].BaseURL < out[j].BaseURL })
	return out
}

// Relocate finds where the paired installation is answering now: its saved
// address, else its locator (matched by installation ID). "" = not found.
//
// The locator for the paired installation ID wins: something else may be
// answering on the old port (another Identity profile), and a healthy
// answer there says nothing about who it is.
func Relocate(ctx context.Context, p Pairing) string {
	for _, l := range ReadLocators() {
		if l.IdentitySystemID == p.IdentitySystemID {
			if _, err := ProbeHealth(ctx, l.BaseURL); err == nil {
				return l.BaseURL
			}
		}
	}
	if _, err := ProbeHealth(ctx, p.BaseURL); err == nil {
		return p.BaseURL
	}
	return ""
}

// ---- pairing (the two unsigned calls) ----

func postPlain(ctx context.Context, u, path string, in any) (int, http.Header, map[string]any, error) {
	b, _ := json.Marshal(in)
	req, err := http.NewRequestWithContext(ctx, "POST", u+path, bytes.NewReader(b))
	if err != nil {
		return 0, nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return 0, nil, nil, &Error{Kind: Unavailable, Msg: "Identity didn't answer at " + u}
	}
	defer resp.Body.Close()
	var m map[string]any
	json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&m)
	return resp.StatusCode, resp.Header, m, nil
}

// RequestCode asks Identity to show a one-time code in its own window.
func RequestCode(ctx context.Context, u, processorID string) (time.Time, error) {
	st, hdr, m, err := postPlain(ctx, u, "/api/v1/pair/request", map[string]string{"processor_id": processorID, "processor_label": "Thought Processor"})
	if err != nil {
		return time.Time{}, err
	}
	switch st {
	case 202:
		exp, _ := time.Parse(time.RFC3339Nano, fmt.Sprint(m["expires_at"]))
		return exp, nil
	case 429:
		return time.Time{}, &Error{Kind: Refused, Status: 429, Msg: "Identity just showed a code. Wait " + hdr.Get("Retry-After") + " seconds and try again."}
	}
	return time.Time{}, &Error{Kind: Refused, Status: st, Msg: fmt.Sprint(m["error"])}
}

// CompletePairing sends the code the person typed and returns the pairing.
func CompletePairing(ctx context.Context, u, processorID, code string) (Pairing, error) {
	st, _, m, err := postPlain(ctx, u, "/api/v1/pair", map[string]string{"code": code, "processor_id": processorID, "processor_label": "Thought Processor"})
	if err != nil {
		return Pairing{}, err
	}
	if st != 200 {
		msg := fmt.Sprint(m["error"])
		if st == 429 {
			msg = "Too many wrong codes. Wait a minute, then ask Identity for a new code."
		}
		return Pairing{}, &Error{Kind: Refused, Status: st, Msg: msg}
	}
	if _, ok := m["secret"]; !ok {
		return Pairing{}, &Error{Kind: Incompatible, Msg: "this Identity is an older version (identity/1) that sends its secret over the wire; use Identity " + APIMajor + " or newer"}
	}
	p := Pairing{BaseURL: u, ProcessorID: processorID, IdentitySystemID: fmt.Sprint(m["identity_system_id"]),
		IdentityVersion: fmt.Sprint(m["identity_version"]), PairedAt: time.Now().UTC(), Token: fmt.Sprint(m["secret"])}
	if !tokenRE.MatchString(p.Token) || !strings.HasPrefix(p.IdentitySystemID, "idn_") {
		return Pairing{}, &Error{Kind: WrongService, Msg: "Identity's pairing answer didn't look right; nothing was saved"}
	}
	// Prove the new pairing works before keeping it.
	c := New(p)
	var s struct {
		State string `json:"state"`
		API   string `json:"api"`
	}
	if err := c.Get(ctx, "/api/v1/status", &s); err != nil {
		return Pairing{}, err
	}
	if !Compatible(s.API) {
		return Pairing{}, &Error{Kind: Incompatible, Msg: "this Identity speaks " + s.API + "; this Processor needs " + APIMajor}
	}
	return p, nil
}
