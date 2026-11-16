package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

// Log is Processor's activity log: a JSON-lines file in the profile plus the
// last 300 entries in memory for Developer Mode. Nothing secret is written:
// tokens never reach it, and anything shaped like a pairing code or a 64-hex
// secret is masked as a second line of defence.
type Log struct {
	mu   sync.Mutex
	f    *os.File
	ring []Entry
	run  string
}

type Entry struct {
	Time      time.Time `json:"time"`
	Level     string    `json:"level"`
	Component string    `json:"component"`
	Event     string    `json:"event"`
	Detail    string    `json:"detail"`
	Run       string    `json:"run"`
}

var (
	codeLike = regexp.MustCompile(`\b[A-HJ-NP-Z2-9]{4}-[A-HJ-NP-Z2-9]{4}\b`)
	keyLink  = regexp.MustCompile(`#(code|key)=[0-9a-f]+`)
	// Identity approval links: /approve#a=apr_…&t=<one-time token>
	approvalToken = regexp.MustCompile(`([#&]t=)[0-9A-Za-z_-]+`)
	// Request tickets: tk1.<payload>.<mac>
	ticketLike = regexp.MustCompile(`\btk1\.[A-Za-z0-9_-]+\.[0-9a-f]+`)
	// Secret-bearing JSON or key=value fields.
	secretField = regexp.MustCompile(`(?i)("?(secret|launch_key|password|passphrase|token|authorization|x-tf-ticket|x-tf-auth|x-identity-auth)"?\s*[:=]\s*"?)[^",\s}]+`)
)

// scrub masks anything secret-shaped. Secrets are never logged on purpose;
// this is the second line of defence for text that came from elsewhere
// (error messages, a feature's own log). SHA-256 digests and IDs stay
// visible: they identify things but unlock nothing.
func scrub(s string) string {
	s = keyLink.ReplaceAllString(s, "#$1=<removed>")
	s = approvalToken.ReplaceAllString(s, "${1}<removed>")
	s = ticketLike.ReplaceAllString(s, "tk1.<removed>")
	s = secretField.ReplaceAllString(s, "${1}<removed>")
	s = codeLike.ReplaceAllString(s, "XXXX-XXXX")
	return s
}

func OpenLog(dir, run string) *Log {
	l := &Log{run: run}
	if err := os.MkdirAll(dir, 0o700); err == nil {
		l.f, _ = os.OpenFile(filepath.Join(dir, "processor.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	}
	return l
}

func (l *Log) Add(level, component, event, detail string) {
	e := Entry{Time: time.Now().UTC(), Level: level, Component: component, Event: event, Detail: scrub(detail), Run: l.run}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.ring = append(l.ring, e)
	if len(l.ring) > 300 {
		l.ring = l.ring[len(l.ring)-300:]
	}
	if l.f != nil {
		b, _ := json.Marshal(e)
		l.f.Write(append(b, '\n'))
	}
}

func (l *Log) Info(c, e, d string) { l.Add("info", c, e, d) }
func (l *Log) Warn(c, e, d string) { l.Add("warn", c, e, d) }

func (l *Log) Recent(n int) []Entry {
	l.mu.Lock()
	defer l.mu.Unlock()
	if n > len(l.ring) {
		n = len(l.ring)
	}
	return append([]Entry(nil), l.ring[len(l.ring)-n:]...)
}

func (l *Log) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f != nil {
		l.f.Close()
	}
}
