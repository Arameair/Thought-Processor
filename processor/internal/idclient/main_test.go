package idclient

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Setenv("TP_IDENTITY_DEFAULT_URL", "-")
	d, _ := os.MkdirTemp("", "tp-test-locators-")
	os.Setenv("IDENTITY_LOCATOR_DIR", d)
	code := m.Run()
	os.RemoveAll(d)
	os.Exit(code)
}

// Regression for Grok M4: an unrelated Identity-looking program on
// 127.0.0.1:7790 must not appear when the default probe is off, and the
// probe can be pointed elsewhere explicitly. If 7790 is already taken by a
// real program, that program is the decoy; this test never stops it.
func TestDefaultProbeIsolation(t *testing.T) {
	if ln, err := net.Listen("tcp", "127.0.0.1:7790"); err == nil {
		srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]string{"app": "identity", "version": "0.0.0-DECOY", "api": "identity/2-alpha"})
		})}
		go srv.Serve(ln)
		defer srv.Close()
	}
	for _, c := range Discover(context.Background()) {
		if c.BaseURL == "http://127.0.0.1:7790" {
			t.Fatalf("default probe is off but %s was contacted and listed", c.BaseURL)
		}
	}
	t.Setenv("TP_IDENTITY_DEFAULT_URL", "http://example.com:7790")
	if DefaultProbeURL() != "" {
		t.Fatal("a non-loopback default probe address was accepted")
	}
	t.Setenv("TP_IDENTITY_DEFAULT_URL", "")
	if DefaultProbeURL() != DefaultURL {
		t.Fatal("production default changed")
	}
}
