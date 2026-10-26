// Package store owns Processor's private profile folder.
//
//	<profile>/processor.json         this Processor's durable ID (processor_id)
//	<profile>/settings.json          display preferences (selected identity, Developer Mode, last Identity address)
//	<profile>/identity-pairing.json  the pairing with Identity (token protected, see secret_*.go)
//	<profile>/features/<id>/...      installed features and their install records
//	<profile>/logs/processor.jsonl   activity log (no secrets)
//	<profile>/runtime/               lock + this run's operator code (removed on exit)
//
// Processor stores no identities, grants or decisions: those belong to Identity.
package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var ErrLocked = errors.New("profile in use")

func RandHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

type Identity struct {
	ProcessorID string    `json:"processor_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type Settings struct {
	SelectedIdentity string `json:"selected_identity,omitempty"`
	DeveloperMode    bool   `json:"developer_mode,omitempty"`
	IdentityURL      string `json:"identity_url,omitempty"` // last address the person chose (before pairing)
}

type Profile struct {
	Dir     string
	Self    Identity
	Created bool

	mu       sync.Mutex
	settings Settings
	lock     *fileLock
}

var pidRE = regexp.MustCompile(`^tp_[0-9a-f]{12}$`)

func Open(dir string) (*Profile, error) {
	dir = filepath.Clean(dir)
	for _, d := range []string{dir, filepath.Join(dir, "runtime"), filepath.Join(dir, "logs"), filepath.Join(dir, "features")} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return nil, fmt.Errorf("can't create profile folder %s: %w", d, err)
		}
	}
	lk, err := acquireLock(filepath.Join(dir, "runtime", "processor.lock"))
	if err != nil {
		return nil, fmt.Errorf("%w: another Thought Processor is already using %s", ErrLocked, dir)
	}
	p := &Profile{Dir: dir, lock: lk}
	fail := func(e error) (*Profile, error) { lk.release(); return nil, e }
	ip := filepath.Join(dir, "processor.json")
	b, err := os.ReadFile(ip)
	switch {
	case errors.Is(err, os.ErrNotExist):
		p.Self = Identity{ProcessorID: "tp_" + RandHex(6), CreatedAt: time.Now().UTC()}
		if err := WriteJSON(ip, p.Self); err != nil {
			return fail(err)
		}
		p.Created = true
	case err != nil:
		return fail(err)
	default:
		if json.Unmarshal(b, &p.Self) != nil || !pidRE.MatchString(p.Self.ProcessorID) {
			return fail(fmt.Errorf("%s is damaged; it was left untouched. Use another -data folder or restore it", ip))
		}
	}
	if b, err := os.ReadFile(filepath.Join(dir, "settings.json")); err == nil {
		json.Unmarshal(b, &p.settings) // display preferences only; damage just resets them
	}
	return p, nil
}

func (p *Profile) Close() {
	os.Remove(p.CodePath())
	if p.lock != nil {
		p.lock.release()
	}
}

func (p *Profile) CodePath() string { return filepath.Join(p.Dir, "runtime", "operator-link.txt") }

func (p *Profile) Settings() Settings { p.mu.Lock(); defer p.mu.Unlock(); return p.settings }

func (p *Profile) UpdateSettings(fn func(*Settings)) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	old := p.settings
	fn(&p.settings)
	if err := WriteJSON(filepath.Join(p.Dir, "settings.json"), p.settings); err != nil {
		p.settings = old
		return err
	}
	return nil
}

// WriteJSON writes atomically with owner-only permissions.
func WriteJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return WriteFile(path, append(b, '\n'))
}

func WriteFile(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, err = f.Write(b)
	if err == nil {
		err = f.Chmod(0o600)
	}
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

// DisplayPath shortens the home folder to ~ so logs don't reveal the user name.
func DisplayPath(p string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if rel, err := filepath.Rel(home, p); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.Join("~", rel)
		}
	}
	return p
}
