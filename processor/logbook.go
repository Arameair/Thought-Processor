package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logbook records Processor's OWN events only, with fixed, code-authored
// detail strings. It never records request bodies, operator codes, cookies,
// CSRF tokens, environment variables or claims returned by other programs.
type Logbook struct {
	mu   sync.Mutex
	list []LogEntry
	f    *os.File
	err  string
}

type LogEntry struct {
	At     time.Time `json:"at"`
	Kind   string    `json:"kind"`
	Code   string    `json:"code"`
	OpID   string    `json:"op_id,omitempty"`
	Detail string    `json:"detail"`
}

const logMaxBytes = 1 << 20

func OpenLogbook(path string) *Logbook {
	l := &Logbook{}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		l.err = "log folder not writable"
		return l
	}
	if st, err := os.Stat(path); err == nil && st.Size() > logMaxBytes {
		_ = os.Rename(path, path+".1") // keep one previous file, bounded
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		l.err = "log file not writable"
		return l
	}
	l.f = f
	return l
}

func (l *Logbook) Add(kind, code, opID, detail string) {
	e := LogEntry{At: time.Now().UTC(), Kind: kind, Code: code, OpID: opID, Detail: detail}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.list = append(l.list, e)
	if len(l.list) > 200 {
		l.list = l.list[len(l.list)-200:]
	}
	if l.f != nil {
		b, _ := json.Marshal(e)
		if _, err := l.f.Write(append(b, '\n')); err != nil {
			l.err = "log write failed"
		}
	}
}

func (l *Logbook) Recent(n int) []LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.list) > n {
		return append([]LogEntry{}, l.list[len(l.list)-n:]...)
	}
	return append([]LogEntry{}, l.list...)
}

func (l *Logbook) Err() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.err
}

func (l *Logbook) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f != nil {
		l.f.Close()
		l.f = nil
	}
}
