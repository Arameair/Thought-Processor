package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Profile is the isolated data root for one Processor installation/test.
// It holds ONLY Processor's own non-authoritative records: explicitly entered
// candidate addresses, a run record, the per-run operator code and a bounded
// event log. It never holds permissions, identities or Engine data.
type Profile struct {
	Dir      string
	lock     *fileLock
	Previous *RunRecord // run record left by a run that did not shut down cleanly
	LastStop *StopRecord
	Problems []string
}

type RunRecord struct {
	RunID     string    `json:"run_id"`
	PID       int       `json:"pid"`
	Port      int       `json:"port"`
	StartedAt time.Time `json:"started_at"`
	Revision  string    `json:"revision"`
}

type StopRecord struct {
	RunID      string    `json:"run_id"`
	StoppedAt  time.Time `json:"stopped_at"`
	ShutdownMS int64     `json:"shutdown_ms"`
}

type Candidates struct {
	Identity string `json:"identity,omitempty"`
	Engine   string `json:"engine,omitempty"`
}

func OpenProfile(dir string) (*Profile, error) {
	if err := os.MkdirAll(filepath.Join(dir, "runtime"), 0o700); err != nil {
		return nil, fmt.Errorf("can't create profile folder %s: %w", dir, err)
	}
	// Prove the folder is writable before claiming anything about it.
	probe := filepath.Join(dir, "runtime", ".write-check")
	if err := os.WriteFile(probe, []byte("ok"), 0o600); err != nil {
		return nil, fmt.Errorf("profile folder %s is not writable: %w", dir, err)
	}
	os.Remove(probe)

	lk, err := acquireLock(filepath.Join(dir, "runtime", "processor.lock"))
	if err != nil {
		msg := fmt.Sprintf("another Thought Processor is already using profile %s", dir)
		if rr, e := readRun(filepath.Join(dir, "runtime", "run.json")); e == nil {
			msg += fmt.Sprintf(" (run %s, pid %d, http://127.0.0.1:%d). Open that one, or stop it first; this one will not take it over", rr.RunID, rr.PID, rr.Port)
		}
		return nil, errors.New(msg)
	}
	p := &Profile{Dir: dir, lock: lk}
	// A run.json that survives while we hold the lock means the last run ended
	// without a clean shutdown (crash, kill -9, power loss).
	if rr, err := readRun(p.runPath()); err == nil {
		p.Previous = &rr
	} else if !errors.Is(err, os.ErrNotExist) {
		p.Problems = append(p.Problems, "previous run record unreadable; left in place")
	}
	if b, err := os.ReadFile(filepath.Join(dir, "runtime", "last-stop.json")); err == nil {
		var s StopRecord
		if json.Unmarshal(b, &s) == nil {
			p.LastStop = &s
		}
	}
	return p, nil
}

func (p *Profile) runPath() string  { return filepath.Join(p.Dir, "runtime", "run.json") }
func (p *Profile) CodePath() string { return filepath.Join(p.Dir, "runtime", "operator-code") }
func (p *Profile) LogPath() string  { return filepath.Join(p.Dir, "logs", "processor.jsonl") }
func (p *Profile) candPath() string { return filepath.Join(p.Dir, "candidates.json") }

func readRun(path string) (RunRecord, error) {
	var r RunRecord
	b, err := os.ReadFile(path)
	if err != nil {
		return r, err
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return r, err
	}
	return r, nil
}

func (p *Profile) WriteRun(r RunRecord) error { return writeJSONAtomic(p.runPath(), r) }
func (p *Profile) WriteCode(code string) error {
	return writeFileAtomic(p.CodePath(), []byte(code+"\n"))
}

func (p *Profile) RecordCleanShutdown(runID string, d time.Duration) {
	_ = writeJSONAtomic(filepath.Join(p.Dir, "runtime", "last-stop.json"),
		StopRecord{RunID: runID, StoppedAt: time.Now().UTC(), ShutdownMS: d.Milliseconds()})
	os.Remove(p.CodePath())
	os.Remove(p.runPath())
}

// LoadCandidates never rewrites an unreadable file; it reports it instead.
func (p *Profile) LoadCandidates() (Candidates, error) {
	var c Candidates
	b, err := os.ReadFile(p.candPath())
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return Candidates{}, fmt.Errorf("candidates.json is unreadable (left unchanged): %w", err)
	}
	// Re-validate whatever is on disk; a hand-edited file gets no extra trust.
	if c.Identity != "" {
		if _, err := parseCandidate(c.Identity, 0); err != nil {
			return Candidates{}, fmt.Errorf("candidates.json identity address rejected (left unchanged): %w", err)
		}
	}
	if c.Engine != "" {
		if _, err := parseCandidate(c.Engine, 0); err != nil {
			return Candidates{}, fmt.Errorf("candidates.json engine address rejected (left unchanged): %w", err)
		}
	}
	return c, nil
}

func (p *Profile) SaveCandidates(c Candidates) error { return writeJSONAtomic(p.candPath(), c) }

func (p *Profile) Close() {
	if p.lock != nil {
		p.lock.release()
	}
}

func writeJSONAtomic(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(path, append(b, '\n'))
}

func writeFileAtomic(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	if _, err := f.Write(b); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}
