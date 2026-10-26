// Package idclient is Thought Processor's client for Identity API v1
// (identity/1-alpha; contract: Arameair/identity docs/API_V1.md).
//
// API identity/2: the pairing secret NEVER leaves Processor after pairing.
// Two keys are derived from it. Every protected request is signed with the
// request key (method, path, time, fresh nonce, body hash), and every answer
// is checked with the response key before it is believed:
//
//	X-Identity-Proof     == hex(HMAC-SHA256(response_key, "identity/2 response\n" + nonce + "\n" + status + "\n" + raw body))
//	X-Identity-System-ID == the installation ID saved when pairing
//
// A program that occupies Identity's port therefore learns nothing it could
// reuse, and can't produce an answer Processor would accept.
//
// Anything else is classified (see Kind) so the UI can say exactly what is
// wrong. Nothing here caches a permission: callers ask again every time.
package idclient

import (
	"bytes"
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
	"strconv"
	"strings"
	"time"

	"github.com/Arameair/Thought-Processor/processor/internal/store"
)

const (
	APIMajor    = "identity/2"
	ProofHeader = "X-Identity-Proof"
	IDHeader    = "X-Identity-System-ID"
	NonceHeader = "X-Identity-Nonce"
	TimeHeader  = "X-Identity-Time"
	AuthHeader  = "X-Identity-Auth"
	PIDHeader   = "X-Processor-ID"
	DefaultURL  = "http://127.0.0.1:7790"
)

// Kind classifies a failed call.
type Kind string

const (
	Unavailable  Kind = "AUTHORITY_UNAVAILABLE" // no answer (network, timeout)
	TokenRefused Kind = "CONNECTION_REMOVED"    // 401: Identity doesn't know our token
	WrongService Kind = "WRONG_SERVICE"         // unsigned / bad proof / different installation
	Refused      Kind = "REFUSED"               // a signed refusal (4xx/5xx) from the right Identity
	Incompatible Kind = "INCOMPATIBLE"
	NotPaired    Kind = "NOT_PAIRED"
)

type Error struct {
	Kind   Kind
	Status int
	Code   string // machine-readable "error" from Identity, when present
	Msg    string
}

func (e *Error) Error() string { return string(e.Kind) + ": " + e.Msg }

func KindOf(err error) Kind {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return ""
}

// Client talks to one paired Identity.
type Client struct {
	HTTP *http.Client
	p    Pairing
}

func New(p Pairing) *Client {
	return &Client{HTTP: &http.Client{Timeout: 20 * time.Second}, p: p}
}

func (c *Client) Pairing() Pairing { return c.p }

// SetBase points the client at a new address for the same installation
// (Identity restarted on another port; found through its locator).
func (c *Client) SetBase(u string) { c.p.BaseURL = strings.TrimRight(u, "/") }

func hmacHex(key []byte, parts ...[]byte) string {
	m := hmac.New(sha256.New, key)
	for _, p := range parts {
		m.Write(p)
	}
	return hex.EncodeToString(m.Sum(nil))
}

// DeriveKeys turns the pairing secret into the request and response keys.
func DeriveKeys(secret string) (req, resp []byte) {
	r, _ := hex.DecodeString(hmacHex([]byte(secret), []byte("identity/2 request key")))
	s, _ := hex.DecodeString(hmacHex([]byte(secret), []byte("identity/2 response key")))
	return r, s
}

// Sign adds the identity/2 request headers. It never adds the secret.
func Sign(req *http.Request, processorID, secret string, body []byte) (nonce string) {
	rk, _ := DeriveKeys(secret)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nonce = store.RandHex(12)
	sum := sha256.Sum256(body)
	req.Header.Set(PIDHeader, processorID)
	req.Header.Set(TimeHeader, ts)
	req.Header.Set(NonceHeader, nonce)
	req.Header.Set(AuthHeader, hmacHex(rk, []byte("identity/2 request\n"+req.Method+"\n"+req.URL.RequestURI()+"\n"+ts+"\n"+nonce+"\n"+hex.EncodeToString(sum[:]))))
	return nonce
}

// VerifyProof checks an answer's X-Identity-Proof (exported for tests).
func VerifyProof(secret, nonce string, status int, body []byte, proof string) bool {
	_, sk := DeriveKeys(secret)
	want := hmacHex(sk, []byte("identity/2 response\n"+nonce+"\n"+strconv.Itoa(status)+"\n"), body)
	return hmac.Equal([]byte(want), []byte(proof))
}

// Call makes a protected request and returns the verified raw body.
func (c *Client) Call(ctx context.Context, method, path string, in any) (int, []byte, error) {
	if c.p.Token == "" {
		return 0, nil, &Error{Kind: NotPaired, Msg: "Processor isn't connected to Identity yet"}
	}
	var b []byte
	if in != nil {
		b, _ = json.Marshal(in)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.p.BaseURL+path, bytes.NewReader(b))
	if err != nil {
		return 0, nil, &Error{Kind: Unavailable, Msg: err.Error()}
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	nonce := Sign(req, c.p.ProcessorID, c.p.Token, b)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return 0, nil, &Error{Kind: Unavailable, Msg: "Identity didn't answer at " + c.p.BaseURL}
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return 0, nil, &Error{Kind: Unavailable, Status: resp.StatusCode, Msg: "Identity's answer was cut off"}
	}
	if resp.StatusCode == 401 {
		var e struct {
			Error string `json:"error"`
		}
		json.Unmarshal(raw, &e)
		msg := "Identity no longer recognises this Processor (it was disconnected, or a different Identity profile is running)"
		if strings.Contains(e.Error, "clock") {
			msg = "Identity refused the request because this computer's clock looks wrong. Check the date and time."
		} else if strings.Contains(e.Error, "older Identity") {
			msg = "This connection was made with an older Identity. Connect again."
		}
		return 401, nil, &Error{Kind: TokenRefused, Status: 401, Msg: msg}
	}
	if !VerifyProof(c.p.Token, nonce, resp.StatusCode, raw, resp.Header.Get(ProofHeader)) {
		actual := resp.Header.Get(IDHeader)
		if actual == "" {
			actual = "an unsigned program"
		} else if actual != c.p.IdentitySystemID {
			actual = "Identity installation " + actual
		} else {
			actual = "a program claiming to be it, without a valid proof"
		}
		return resp.StatusCode, nil, &Error{Kind: WrongService, Status: resp.StatusCode, Msg: "Expected the paired Identity installation " + c.p.IdentitySystemID +
			" at " + c.p.BaseURL + ", but " + actual + " answered there. Processor did not trust it or switch to it. Start your own Identity " +
			"(Processor follows it to a new port through its locator), or choose Connect again only if you really mean to use a different Identity."}
	}
	if resp.Header.Get(IDHeader) != c.p.IdentitySystemID {
		return resp.StatusCode, nil, &Error{Kind: WrongService, Status: resp.StatusCode, Msg: "Expected Identity installation " + c.p.IdentitySystemID +
			" but " + resp.Header.Get(IDHeader) + " answered at " + c.p.BaseURL + ". Processor did not switch."}
	}
	if resp.StatusCode >= 400 {
		var e struct {
			Error   any    `json:"error"`
			Message string `json:"message"`
		}
		json.Unmarshal(raw, &e)
		code, msg := fmt.Sprint(e.Error), e.Message
		if msg == "" {
			msg = code
		}
		if _, isStr := e.Error.(string); !isStr {
			code = ""
		}
		return resp.StatusCode, raw, &Error{Kind: Refused, Status: resp.StatusCode, Code: code, Msg: msg}
	}
	return resp.StatusCode, raw, nil
}

// Get/Post decode a verified JSON answer into out.
func (c *Client) Get(ctx context.Context, path string, out any) error {
	_, raw, err := c.Call(ctx, "GET", path, nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

func (c *Client) Post(ctx context.Context, path string, in, out any) error {
	if in == nil {
		in = map[string]any{}
	}
	_, raw, err := c.Call(ctx, "POST", path, in)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(raw, out)
}

// Download streams a release asset (not signed: its SHA-256 comes from the
// signed release answer and is checked by the installer). The caller closes it.
func (c *Client) Download(ctx context.Context, path string) (io.ReadCloser, int64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.p.BaseURL+path, nil)
	if err != nil {
		return nil, 0, err
	}
	Sign(req, c.p.ProcessorID, c.p.Token, nil)
	dl := &http.Client{Timeout: 30 * time.Minute}
	resp, err := dl.Do(req)
	if err != nil {
		return nil, 0, &Error{Kind: Unavailable, Msg: "Identity didn't answer"}
	}
	if resp.StatusCode != 200 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		resp.Body.Close()
		var e struct {
			Error   any    `json:"error"`
			Message string `json:"message"`
		}
		json.Unmarshal(raw, &e)
		kind := Refused
		if resp.StatusCode == 401 {
			kind = TokenRefused
		}
		msg := e.Message
		if msg == "" {
			msg = fmt.Sprint(e.Error)
		}
		return nil, 0, &Error{Kind: kind, Status: resp.StatusCode, Msg: msg}
	}
	return resp.Body, resp.ContentLength, nil
}

// Health reads the public, unsigned health answer. It only says *something*
// answers at u; it never establishes trust.
type Health struct {
	App      string `json:"app"`
	Version  string `json:"version"`
	Protocol string `json:"protocol"`
	API      string `json:"api"`
}

func ProbeHealth(ctx context.Context, u string) (Health, error) {
	var h Health
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", strings.TrimRight(u, "/")+"/api/health", nil)
	if err != nil {
		return h, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return h, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return h, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&h); err != nil {
		return h, err
	}
	if h.App != "identity" {
		return h, errors.New("something else answers there (not Identity)")
	}
	return h, nil
}

// Compatible reports whether an Identity API label is one this Processor speaks.
func Compatible(api string) bool { return api == APIMajor || strings.HasPrefix(api, APIMajor+"-") }

// LoopbackOnly refuses anything but 127.0.0.1 / localhost http addresses.
func LoopbackOnly(u string) (string, error) {
	u = strings.TrimRight(strings.TrimSpace(u), "/")
	if !strings.HasPrefix(u, "http://") {
		return "", errors.New("Identity's address must start with http://127.0.0.1:")
	}
	host := strings.TrimPrefix(u, "http://")
	h, port, err := net.SplitHostPort(host)
	if err != nil || port == "" || strings.Contains(port, "/") {
		return "", errors.New("Identity's address must look like http://127.0.0.1:7790")
	}
	if h != "127.0.0.1" && h != "localhost" {
		return "", errors.New("Processor only talks to an Identity on this computer (127.0.0.1)")
	}
	return "http://127.0.0.1:" + port, nil
}
