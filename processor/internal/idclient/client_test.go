package idclient

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const tok = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// fakeIdentity verifies signed requests and signs answers the way Identity
// identity/2 does. mode changes behaviour. It records every request so tests
// can prove the secret never travels.
var seenRequests []string

func fakeIdentity(t *testing.T, sys string, mode *string) *httptest.Server {
	rk, sk := DeriveKeys(tok)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") != "" {
			t.Error("client sent a browser Origin header")
		}
		raw, _ := io.ReadAll(r.Body)
		dump := fmt.Sprint(r.Header) + r.URL.String() + string(raw)
		seenRequests = append(seenRequests, dump)
		if r.URL.Path == "/api/health" {
			w.Write([]byte(`{"app":"identity","version":"t","api":"identity/2-alpha"}`))
			return
		}
		sum := sha256.Sum256(raw)
		mac := hmac.New(sha256.New, rk)
		mac.Write([]byte("identity/2 request\n" + r.Method + "\n" + r.URL.RequestURI() + "\n" + r.Header.Get(TimeHeader) + "\n" + r.Header.Get(NonceHeader) + "\n" + hex.EncodeToString(sum[:])))
		if r.Header.Get(AuthHeader) != hex.EncodeToString(mac.Sum(nil)) || r.Header.Get(PIDHeader) != "tp_000000000000" || *mode == "401" {
			w.WriteHeader(401)
			return
		}
		body := []byte(`{"state":"READY","api":"identity/2-alpha","identity_system_id":"` + sys + `"}`)
		code := 200
		if *mode == "refuse" {
			body, code = []byte(`{"error":"NO_RELEASE","message":"no release"}`), 404
		}
		key := sk
		if *mode == "impostor" { // signs with what an impostor could know: nothing secret
			key = []byte(r.Header.Get(AuthHeader))
		}
		m := hmac.New(sha256.New, key)
		status := code
		if *mode == "status" {
			status = 500 // proof made for a different status
		}
		m.Write([]byte("identity/2 response\n" + r.Header.Get(NonceHeader) + "\n" + strconv.Itoa(status) + "\n"))
		m.Write(body)
		proof := hex.EncodeToString(m.Sum(nil))
		switch *mode {
		case "badproof":
			proof = strings.Repeat("0", 64)
		case "noproof":
			proof = ""
		case "tamper":
			body = append(body, ' ')
		}
		w.Header().Set(ProofHeader, proof)
		w.Header().Set(IDHeader, sys)
		if *mode == "otherid" {
			w.Header().Set(IDHeader, "idn_other")
		}
		w.WriteHeader(code)
		w.Write(body)
	}))
}

func TestCallVerifiesEveryAnswer(t *testing.T) {
	mode := ""
	srv := fakeIdentity(t, "idn_aaaaaaaaaaaaaaaaaaaaaaaa", &mode)
	defer srv.Close()
	c := New(Pairing{BaseURL: srv.URL, Token: tok, ProcessorID: "tp_000000000000", IdentitySystemID: "idn_aaaaaaaaaaaaaaaaaaaaaaaa"})
	ctx := context.Background()
	if s, err := c.Status(ctx); err != nil || s.State != "READY" {
		t.Fatalf("good answer refused: %v", err)
	}
	for m, want := range map[string]Kind{"badproof": WrongService, "noproof": WrongService, "tamper": WrongService,
		"otherid": WrongService, "401": TokenRefused, "refuse": Refused, "impostor": WrongService, "status": WrongService} {
		mode = m
		_, err := c.Status(ctx)
		if KindOf(err) != want {
			t.Errorf("%s: got %v want %s", m, err, want)
		}
		if m == "refuse" {
			var e *Error
			if !errorsAs(err, &e) || e.Code != "NO_RELEASE" {
				t.Errorf("signed refusal code lost: %v", err)
			}
		}
	}
	// The pairing secret never appears in anything sent to Identity.
	for _, r := range seenRequests {
		if strings.Contains(r, tok) {
			t.Fatal("the pairing secret was sent on the wire")
		}
	}
	srv.Close()
	if _, err := c.Status(ctx); KindOf(err) != Unavailable {
		t.Errorf("down: %v", err)
	}
	if _, err := New(Pairing{}).Status(ctx); KindOf(err) != NotPaired {
		t.Errorf("unpaired: %v", err)
	}
}

func errorsAs(err error, e **Error) bool {
	x, ok := err.(*Error)
	if ok {
		*e = x
	}
	return ok
}

func TestPairingSavedProtectedAndReloaded(t *testing.T) {
	dir := t.TempDir()
	p := Pairing{BaseURL: "http://127.0.0.1:7790", ProcessorID: "tp_000000000000", IdentitySystemID: "idn_x", Token: tok}
	if err := SavePairing(dir, p); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "identity-pairing.json"))
	var f map[string]any
	json.Unmarshal(b, &f)
	if _, has := f["token"]; has {
		t.Fatal("token stored under a plain field")
	}
	got, err := LoadPairing(dir)
	if err != nil || got.Token != tok || got.IdentitySystemID != "idn_x" {
		t.Fatalf("reload: %v %+v", err, got)
	}
	DeletePairing(dir)
	if got, err := LoadPairing(dir); got != nil || err != nil {
		t.Fatal("pairing not deleted")
	}
}

func TestDiscoveryUsesLocatorsAndLoopbackOnly(t *testing.T) {
	mode := ""
	srv := fakeIdentity(t, "idn_bbbbbbbbbbbbbbbbbbbbbbbb", &mode)
	defer srv.Close()
	loc := t.TempDir()
	t.Setenv("IDENTITY_LOCATOR_DIR", loc)
	os.WriteFile(filepath.Join(loc, "idn_bbbbbbbbbbbbbbbbbbbbbbbb.json"), []byte(`{"base_url":"`+srv.URL+`","identity_system_id":"idn_bbbbbbbbbbbbbbbbbbbbbbbb"}`), 0o600)
	os.WriteFile(filepath.Join(loc, "idn_cccccccccccccccccccccccc.json"), []byte(`{"base_url":"http://evil.example:80","identity_system_id":"idn_c"}`), 0o600)
	cs := Discover(context.Background())
	found := false
	for _, c := range cs {
		if c.BaseURL == srv.URL && c.IdentitySystemID == "idn_bbbbbbbbbbbbbbbbbbbbbbbb" && c.Compatible {
			found = true
		}
		if strings.Contains(c.BaseURL, "evil") {
			t.Fatal("non-loopback locator used")
		}
	}
	if !found {
		t.Fatalf("locator not discovered: %+v", cs)
	}
	// Relocate follows the paired installation to its new address.
	if u := Relocate(context.Background(), Pairing{BaseURL: "http://127.0.0.1:1", IdentitySystemID: "idn_bbbbbbbbbbbbbbbbbbbbbbbb"}); u != srv.URL {
		t.Fatalf("relocate: %q", u)
	}
	for _, bad := range []string{"https://127.0.0.1:7790", "http://10.0.0.1:7790", "http://127.0.0.1", "ftp://x"} {
		if _, err := LoopbackOnly(bad); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
}
