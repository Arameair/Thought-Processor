package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Discovery rules (Gate 1, before any approval):
//   - only explicitly configured same-host candidates; no scanning
//   - no credentials sent, no operational session, no control request
//   - only endpoints classifiable as public read-only metadata are called
//   - every field a candidate returns is an UNTRUSTED CLAIM
//
// Identity: some Identity experiments expose GET /api/health returning only
// {"app","version"}; that is public metadata, so it may be read.
// Thought Engine: its only read endpoint (/api/status) returns entity cards,
// activity log and grant state — NOT public metadata — so before approval
// Processor checks only whether the TCP port accepts connections.

const (
	tcpTimeout    = 800 * time.Millisecond
	healthTimeout = 1500 * time.Millisecond
	healthMaxBody = 4096
	staleAfter    = 20 * time.Second
)

// parseCandidate accepts only http://<loopback literal or localhost>:<port>
// with no credentials, path, query or fragment. selfPort (if >0) is refused so
// Processor can't be pointed at itself.
func parseCandidate(raw string, selfPort int) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if len(raw) > 64 {
		return "", errors.New("address too long")
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", errors.New("not a valid address")
	}
	if u.Scheme != "http" {
		return "", errors.New("only http:// on this computer is supported")
	}
	if u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || (u.Path != "" && u.Path != "/") {
		return "", errors.New("give only host and port (no user, path, query or #fragment)")
	}
	host, port := u.Hostname(), u.Port()
	switch host {
	case "127.0.0.1", "::1", "localhost":
	default:
		return "", errors.New("only this computer (127.0.0.1, ::1 or localhost) is allowed; remote targets are refused")
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return "", errors.New("an explicit port 1–65535 is required")
	}
	if selfPort > 0 && n == selfPort {
		return "", errors.New("that is this Processor's own port")
	}
	if host == "localhost" {
		host = "127.0.0.1" // never resolve names at probe time
	}
	return "http://" + net.JoinHostPort(host, port), nil
}

// ProbeResult is evidence from one check at one moment.
type ProbeResult struct {
	Target     string    `json:"target"`
	CheckedAt  time.Time `json:"checked_at"`
	ElapsedMS  int64     `json:"elapsed_ms"`
	Port       string    `json:"port"`     // OPEN | CLOSED | TIMEOUT | ERROR
	Metadata   string    `json:"metadata"` // NOT_REQUESTED | RESPONDED | NO_PUBLIC_METADATA | UNREADABLE | FAILED
	HTTPStatus int       `json:"http_status,omitempty"`
	// Untrusted claims, sanitized and length-limited. Shown in the UI with an
	// UNVERIFIED label; never included in Copy Context.
	ClaimedApp     string `json:"claimed_app,omitempty"`
	ClaimedVersion string `json:"claimed_version,omitempty"`
	ClaimMatches   bool   `json:"claim_matches"`
	Note           string `json:"note"`
}

func probeTCP(ctx context.Context, target string) (string, string) {
	host := strings.TrimPrefix(target, "http://")
	d := net.Dialer{Timeout: tcpTimeout}
	c, err := d.DialContext(ctx, "tcp", host)
	if err != nil {
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			return "TIMEOUT", "no answer within " + tcpTimeout.String()
		}
		if strings.Contains(err.Error(), "refused") {
			return "CLOSED", "nothing is listening on that port"
		}
		return "ERROR", "connection failed"
	}
	c.Close()
	return "OPEN", "something is listening; what it is has not been verified"
}

var probeClient = &http.Client{
	Timeout: healthTimeout,
	Transport: &http.Transport{Proxy: nil, DisableKeepAlives: true,
		DialContext: (&net.Dialer{Timeout: tcpTimeout}).DialContext},
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

func sanitizeClaim(s string) string {
	var b strings.Builder
	for _, r := range s {
		if b.Len() >= 40 {
			break
		}
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func ProbeIdentity(ctx context.Context, target string) ProbeResult {
	t0 := time.Now()
	r := ProbeResult{Target: target, CheckedAt: t0.UTC(), Metadata: "NOT_REQUESTED"}
	r.Port, r.Note = probeTCP(ctx, target)
	if r.Port != "OPEN" {
		r.ElapsedMS = time.Since(t0).Milliseconds()
		return r
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, target+"/api/health", nil)
	req.Header.Set("Accept", "application/json")
	resp, err := probeClient.Do(req)
	r.ElapsedMS = time.Since(t0).Milliseconds()
	if err != nil {
		r.Metadata, r.Note = "FAILED", "port open but the public health request failed or timed out"
		return r
	}
	defer resp.Body.Close()
	r.HTTPStatus = resp.StatusCode
	if resp.StatusCode != http.StatusOK {
		r.Metadata = "NO_PUBLIC_METADATA"
		r.Note = fmt.Sprintf("port open; GET /api/health answered %d, so this program publishes no metadata Processor recognizes", resp.StatusCode)
		return r
	}
	var m map[string]any
	dec := json.NewDecoder(io.LimitReader(resp.Body, healthMaxBody))
	if err := dec.Decode(&m); err != nil {
		r.Metadata, r.Note = "UNREADABLE", "port open; /api/health did not return a small JSON object"
		return r
	}
	app, _ := m["app"].(string)
	ver, _ := m["version"].(string)
	r.ClaimedApp, r.ClaimedVersion = sanitizeClaim(app), sanitizeClaim(ver)
	r.Metadata = "RESPONDED"
	r.ClaimMatches = r.ClaimedApp == "identity"
	if r.ClaimMatches {
		r.Note = "it CLAIMS to be Identity. A claim is not authentication, and no approval service was found."
	} else {
		r.Note = "it answered, but does not claim to be Identity — check the port"
	}
	return r
}

func ProbeEngine(ctx context.Context, target string) ProbeResult {
	t0 := time.Now()
	r := ProbeResult{Target: target, CheckedAt: t0.UTC(), Metadata: "NOT_REQUESTED"}
	r.Port, r.Note = probeTCP(ctx, target)
	r.ElapsedMS = time.Since(t0).Milliseconds()
	if r.Port == "OPEN" {
		r.Note = "something is listening. Processor did NOT read it: the Engine's status endpoint contains private entity and log data, which needs an approved attachment first."
	}
	return r
}

// Freshness is computed at read time so a result can never stay green
// silently: after staleAfter it is reported STALE regardless of content.
func freshness(p *ProbeResult, now time.Time) string {
	if p == nil {
		return "NEVER_CHECKED"
	}
	if now.Sub(p.CheckedAt) > staleAfter {
		return "STALE"
	}
	return "CURRENT"
}
