package main

// Identity link: pairing with a local Identity and asking it, as the paired
// program, whether this person may install Thought Engine.
//
// Pairing (Identity side: Arameair/identity internal/pairing):
//   - the person asks Identity for a one-time code and types it here;
//   - Identity returns a 256-bit token; Processor stores it in its profile
//     (identity-pairing.json, 0600) and sends it only as a bearer header to
//     the paired Identity address — never in a URL, log, page or Copy Context;
//   - every Identity answer carries an HMAC proof keyed by that token, which
//     Processor checks. A program that merely claims to be Identity at that
//     address can't produce it, so a valid proof authenticates the answer.

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Pairing struct {
	IdentityAddress  string    `json:"identity_address"`
	Token            string    `json:"token"`
	IdentitySystemID string    `json:"identity_system_id"`
	IdentityVersion  string    `json:"identity_version,omitempty"`
	PairedAt         time.Time `json:"paired_at"`
}

func (p *Profile) pairingPath() string  { return filepath.Join(p.Dir, "identity-pairing.json") }
func (p *Profile) instancePath() string { return filepath.Join(p.Dir, "instance.json") }

var processorIDRE = regexp.MustCompile(`^tp_[0-9a-f]{16}$`)

// ProcessorID is this profile's stable instance ID (created once).
func (p *Profile) ProcessorID() (string, error) {
	var v struct {
		ProcessorID string `json:"processor_id"`
	}
	if b, err := os.ReadFile(p.instancePath()); err == nil {
		if json.Unmarshal(b, &v) == nil && processorIDRE.MatchString(v.ProcessorID) {
			return v.ProcessorID, nil
		}
		return "", errors.New("instance.json is damaged; left unchanged")
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	v.ProcessorID = newID("tp_")
	return v.ProcessorID, writeJSONAtomic(p.instancePath(), v)
}

func (p *Profile) LoadPairing() (*Pairing, error) {
	b, err := os.ReadFile(p.pairingPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var pr Pairing
	if err := json.Unmarshal(b, &pr); err != nil || len(pr.Token) != 64 || pr.IdentityAddress == "" {
		return nil, errors.New("identity-pairing.json is damaged; left unchanged — pair again to replace it")
	}
	return &pr, nil
}

func (p *Profile) SavePairing(pr Pairing) error { return writeJSONAtomic(p.pairingPath(), pr) }
func (p *Profile) RemovePairing() error {
	err := os.Remove(p.pairingPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// ---- talking to Identity ----

var identityHTTP = &http.Client{
	Timeout: 20 * time.Second,
	Transport: &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 2 * time.Second}).DialContext,
		ResponseHeaderTimeout: 20 * time.Second},
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

// IdentityError is a failure with a stable code the UI can explain.
type IdentityError struct {
	Code string // IDENTITY_UNREACHABLE | NOT_PAIRED | PROOF_FAILED | REFUSED | BAD_ANSWER
	Msg  string
	HTTP int
}

func (e *IdentityError) Error() string { return e.Msg }

func identityCall(ctx context.Context, method, base, path, token string, body any) (*http.Response, error) {
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = strings.NewReader(string(b))
	}
	req, err := http.NewRequestWithContext(ctx, method, base+path, rdr)
	if err != nil {
		return nil, &IdentityError{Code: "BAD_ANSWER", Msg: "bad Identity address"}
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := identityHTTP.Do(req)
	if err != nil {
		return nil, &IdentityError{Code: "IDENTITY_UNREACHABLE", Msg: "Processor can't reach Identity at " + strings.TrimPrefix(base, "http://") + "."}
	}
	if resp.StatusCode == 401 && token != "" {
		resp.Body.Close()
		return nil, &IdentityError{Code: "NOT_PAIRED", Msg: "Identity no longer accepts this Processor's pairing (it was removed, or this is a different Identity).", HTTP: 401}
	}
	return resp, nil
}

func readErr(resp *http.Response) string {
	var e struct {
		Error string `json:"error"`
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if json.Unmarshal(b, &e) == nil && e.Error != "" {
		return e.Error
	}
	return fmt.Sprintf("HTTP %d", resp.StatusCode)
}

// IdentityPair exchanges a pairing code for a token.
func IdentityPair(ctx context.Context, base, code, processorID, label string) (Pairing, error) {
	resp, err := identityCall(ctx, http.MethodPost, base, "/api/v1/pair", "",
		map[string]string{"code": code, "processor_id": processorID, "processor_label": label})
	if err != nil {
		return Pairing{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return Pairing{}, &IdentityError{Code: "REFUSED", Msg: "Identity refused pairing: " + readErr(resp), HTTP: resp.StatusCode}
	}
	var r struct {
		Token    string `json:"token"`
		SystemID string `json:"identity_system_id"`
		Version  string `json:"identity_version"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&r); err != nil || len(r.Token) != 64 {
		return Pairing{}, &IdentityError{Code: "BAD_ANSWER", Msg: "Identity's pairing answer was not understood"}
	}
	return Pairing{IdentityAddress: base, Token: r.Token, IdentitySystemID: r.SystemID, IdentityVersion: r.Version, PairedAt: time.Now().UTC()}, nil
}

func proof(token, canonical string) string {
	m := hmac.New(sha256.New, []byte(token))
	m.Write([]byte(canonical))
	return hex.EncodeToString(m.Sum(nil))
}

type AccessAccount struct {
	Login string `json:"login"`
	ID    int64  `json:"id"`
}

// AccessDecision mirrors Identity's answer.
type AccessDecision struct {
	Feature          string         `json:"feature"`
	State            string         `json:"state"`
	Allowed          bool           `json:"allowed"`
	Reason           string         `json:"reason"`
	NextStep         string         `json:"next_step"`
	Hint             string         `json:"hint,omitempty"`
	Account          *AccessAccount `json:"account,omitempty"`
	Rule             string         `json:"rule"`
	CheckedAt        time.Time      `json:"checked_at"`
	DecisionID       string         `json:"decision_id"`
	IdentitySystemID string         `json:"identity_system_id"`
	Proof            string         `json:"proof"`
}

func (d AccessDecision) canonical() string {
	var id int64
	if d.Account != nil {
		id = d.Account.ID
	}
	return strings.Join([]string{"access", d.DecisionID, d.State, strconv.FormatInt(id, 10),
		d.CheckedAt.UTC().Format(time.RFC3339Nano), d.Rule, d.IdentitySystemID}, "|")
}

func verifyProof(token, canonical, got string) bool {
	want := proof(token, canonical)
	return len(got) == len(want) && hmac.Equal([]byte(got), []byte(want))
}

func IdentityAccess(ctx context.Context, pr Pairing, fresh bool) (AccessDecision, error) {
	path := "/api/v1/features/thought-engine/access"
	if fresh {
		path += "?fresh=1"
	}
	resp, err := identityCall(ctx, http.MethodGet, pr.IdentityAddress, path, pr.Token, nil)
	if err != nil {
		return AccessDecision{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return AccessDecision{}, &IdentityError{Code: "REFUSED", Msg: "Identity didn't answer the access question: " + readErr(resp), HTTP: resp.StatusCode}
	}
	var d AccessDecision
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<10)).Decode(&d); err != nil {
		return d, &IdentityError{Code: "BAD_ANSWER", Msg: "Identity's answer was not understood"}
	}
	if !verifyProof(pr.Token, d.canonical(), d.Proof) {
		return AccessDecision{}, &IdentityError{Code: "PROOF_FAILED", Msg: "The answer from " + strings.TrimPrefix(pr.IdentityAddress, "http://") + " could not be verified with this pairing, so it was ignored. Pair again with the Identity you trust."}
	}
	if pr.IdentitySystemID != "" && d.IdentitySystemID != pr.IdentitySystemID {
		return AccessDecision{}, &IdentityError{Code: "PROOF_FAILED", Msg: "The answer came from a different Identity than the one this Processor paired with."}
	}
	d.Allowed = d.State == "ALLOWED"
	return d, nil
}

type ReleaseInfo struct {
	DecisionID  string    `json:"decision_id"`
	Tag         string    `json:"tag"`
	Name        string    `json:"name"`
	PublishedAt time.Time `json:"published_at"`
	Platform    string    `json:"platform"`
	Asset       string    `json:"asset"`
	Size        int64     `json:"size"`
	SHA256      string    `json:"sha256"`
	Proof       string    `json:"proof"`
}

func (ri ReleaseInfo) canonical() string {
	return strings.Join([]string{"release", ri.DecisionID, ri.Tag, ri.Asset, ri.SHA256, strconv.FormatInt(ri.Size, 10)}, "|")
}

func IdentityRelease(ctx context.Context, pr Pairing, platform string) (ReleaseInfo, error) {
	resp, err := identityCall(ctx, http.MethodGet, pr.IdentityAddress, "/api/v1/features/thought-engine/release?platform="+platform, pr.Token, nil)
	if err != nil {
		return ReleaseInfo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		code := "REFUSED"
		if resp.StatusCode == 404 {
			code = "NO_RELEASE"
		}
		return ReleaseInfo{}, &IdentityError{Code: code, Msg: readErr(resp), HTTP: resp.StatusCode}
	}
	var ri ReleaseInfo
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<10)).Decode(&ri); err != nil {
		return ri, &IdentityError{Code: "BAD_ANSWER", Msg: "Identity's release answer was not understood"}
	}
	if !verifyProof(pr.Token, ri.canonical(), ri.Proof) {
		return ReleaseInfo{}, &IdentityError{Code: "PROOF_FAILED", Msg: "The release details could not be verified with this pairing, so nothing will be installed."}
	}
	return ri, nil
}

// downloadClient has no overall timeout (large files); the context bounds it.
var downloadClient = &http.Client{
	Transport: &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 2 * time.Second}).DialContext,
		ResponseHeaderTimeout: 60 * time.Second},
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

func IdentityDownload(ctx context.Context, pr Pairing, platform, tag string) (io.ReadCloser, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, pr.IdentityAddress+"/api/v1/features/thought-engine/release/download?platform="+platform+"&tag="+tag, nil)
	req.Header.Set("Authorization", "Bearer "+pr.Token)
	resp, err := downloadClient.Do(req)
	if err != nil {
		return nil, &IdentityError{Code: "IDENTITY_UNREACHABLE", Msg: "Processor lost contact with Identity during the download."}
	}
	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		if resp.StatusCode == 401 {
			return nil, &IdentityError{Code: "NOT_PAIRED", Msg: "Identity no longer accepts this Processor's pairing.", HTTP: 401}
		}
		return nil, &IdentityError{Code: "REFUSED", Msg: "Identity refused the download: " + readErr(resp), HTTP: resp.StatusCode}
	}
	if resp.Header.Get("X-Release-Tag") != tag {
		resp.Body.Close()
		return nil, &IdentityError{Code: "BAD_ANSWER", Msg: "Identity sent a different release than the one checked."}
	}
	return resp.Body, nil
}
