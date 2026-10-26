package app

import (
	"crypto/subtle"
	"net/http"
	"sync"
	"time"

	"github.com/Arameair/Thought-Processor/processor/internal/store"
)

// Sessions: the browser proves it was handed this run's private link (opened
// automatically, also printed in Processor's console). 127.0.0.1 alone is not
// treated as proof. Links and sessions die with the process.
type Sessions struct {
	mu       sync.Mutex
	code     string
	sess     map[string]string // session id -> csrf
	failures int
	lockedTo time.Time
}

const cookieName = "tp_session"

func NewSessions() *Sessions { return &Sessions{code: store.RandHex(24), sess: map[string]string{}} }

func (s *Sessions) Code() string { return s.code }

func (s *Sessions) Exchange(code string) (id, csrf string, ok, locked bool) {
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
	id, csrf = store.RandHex(32), store.RandHex(32)
	s.sess[id] = csrf
	return id, csrf, true, false
}

func (s *Sessions) Lookup(r *http.Request) (string, bool) {
	c, err := r.Cookie(cookieName)
	if err != nil || len(c.Value) != 64 {
		return "", false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	csrf, ok := s.sess[c.Value]
	return csrf, ok
}
