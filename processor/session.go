package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

// Local operator session.
//
// What it proves: the browser was given this run's operator code, which is
// printed to the console that started Processor and written to a 0600 file in
// the profile. Codes and sessions are per-run and die with the process.
//
// What it does NOT prove: anything Identity would decide. It is not an
// administrator enrollment, not a GitHub account and not an approval to touch
// any Engine. It only gates Processor's own diagnostics and refusal endpoints.
type Sessions struct {
	mu       sync.Mutex
	code     string
	sess     map[string]sessionRec
	failures int
	lockedTo time.Time
}

type sessionRec struct {
	csrf    string
	created time.Time
}

const cookieName = "tp_session"

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func NewSessions() *Sessions {
	return &Sessions{code: randHex(24), sess: map[string]sessionRec{}}
}

func (s *Sessions) Code() string { return s.code }

// Exchange trades the operator code for a session. Repeated failures pause
// exchanges briefly (the code is 192-bit, so this is belt-and-braces).
func (s *Sessions) Exchange(code string) (id, csrf string, ok bool, locked bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if time.Now().Before(s.lockedTo) {
		return "", "", false, true
	}
	if subtle.ConstantTimeCompare([]byte(code), []byte(s.code)) != 1 {
		s.failures++
		if s.failures >= 10 {
			s.lockedTo, s.failures = time.Now().Add(time.Minute), 0
		}
		return "", "", false, false
	}
	s.failures = 0
	id, csrf = randHex(32), randHex(32)
	s.sess[id] = sessionRec{csrf: csrf, created: time.Now()}
	return id, csrf, true, false
}

func (s *Sessions) Lookup(r *http.Request) (sessionRec, bool) {
	c, err := r.Cookie(cookieName)
	if err != nil || len(c.Value) != 64 {
		return sessionRec{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.sess[c.Value]
	return rec, ok
}

func (s *Sessions) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sess)
}
