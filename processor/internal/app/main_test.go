package app

import (
	"os"
	"testing"
)

// Tests never contact an Identity that happens to run on this computer
// (Grok M4): the well-known 127.0.0.1:7790 probe is switched off and every
// test uses its own locator folder.
func TestMain(m *testing.M) {
	os.Setenv("TP_IDENTITY_DEFAULT_URL", "-")
	d, _ := os.MkdirTemp("", "tp-test-locators-")
	os.Setenv("IDENTITY_LOCATOR_DIR", d)
	code := m.Run()
	os.RemoveAll(d)
	os.Exit(code)
}
