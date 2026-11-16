package app

import (
	"strings"
	"testing"
)

func TestScrubRemovesSecretShapedValues(t *testing.T) {
	cases := []struct{ in, gone, kept string }{
		{"open http://127.0.0.1:7860/#code=0123456789abcdef", "0123456789abcdef", "#code=<removed>"},
		{"approve at http://127.0.0.1:7790/approve#a=apr_0123456789abcdef&t=deadbeefdeadbeefdeadbeef", "deadbeefdeadbeefdeadbeef", "apr_0123456789abcdef"},
		{"sent X-TF-Ticket tk1.eyJ2IjoxfQ.0123456789abcdef0123456789abcdef", "eyJ2IjoxfQ", "tk1.<removed>"},
		{`{"installation_id":"ins_0123456789abcdef0123","secret":"s3cr3tvalue"}`, "s3cr3tvalue", "ins_0123456789abcdef0123"},
		{`launch_key=00ff00ff00ff`, "00ff00ff00ff", "launch_key="},
		{"pairing code ABCD-EFGH shown", "ABCD-EFGH", "XXXX-XXXX"},
	}
	for _, c := range cases {
		out := scrub(c.in)
		if strings.Contains(out, c.gone) {
			t.Errorf("scrub(%q) = %q still contains %q", c.in, out, c.gone)
		}
		if !strings.Contains(out, c.kept) {
			t.Errorf("scrub(%q) = %q lost %q", c.in, out, c.kept)
		}
	}
	// Identifiers and digests stay readable: they identify things but unlock nothing.
	plain := "installed p1-deterministic-worker as ins_0123456789abcdef0123 (artifact 4f1c2a…) grant gnt_00112233445566778899 decision dec_0011223344556677"
	if scrub(plain) != plain {
		t.Errorf("scrub changed harmless text: %q", scrub(plain))
	}
}
