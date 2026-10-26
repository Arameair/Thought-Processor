// Package supervisor starts, watches and stops feature processes for
// Thought Processor (owner decisions #20: manual / on-demand / always-running,
// idle stop, controlled recovery).
//
// Rules:
//   - Starting a process grants nothing. Permissions are Identity's grants,
//     enforced by the feature itself.
//   - Each launch gets a fresh random launch key (handed over in a 0600 file
//     the feature deletes) and launch ID. Processor trusts an endpoint only
//     after a status answer signed with that key names the right installation
//     and launch. A port, a PID or an endpoint file alone is never trust.
//   - The feature binds 127.0.0.1:0 (the OS picks a free port), so port
//     collisions cannot happen; the chosen URL is read from endpoint.json.
//   - Process restart is not request retry. A crash during a request leaves
//     that request UNCERTAIN; nothing is resent. Experimental features default
//     to manual recovery; "bounded" allows at most 3 restarts in 10 minutes,
//     then NEEDS_ATTENTION.
//   - Idle stop never happens while a request is in flight.
package supervisor

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	API = "thought.feature-api/1"

	Stopped        = "STOPPED"
	Starting       = "STARTING"
	Running        = "RUNNING"
	Stopping       = "STOPPING"
	Crashed        = "CRASHED"
	Restarting     = "RESTARTING"
	NeedsAttention = "NEEDS_ATTENTION"
	FailedToStart  = "FAILED_TO_START"

	ModeManual   = "manual"
	ModeOnDemand = "on_demand"
	ModeAlways   = "always_running"

	RecoveryManual  = "manual"
	RecoveryBounded = "bounded"

	maxRestarts   = 3
	restartWindow = 10 * time.Minute
	startTimeout  = 10 * time.Second
	stopGrace     = 15 * time.Second
	healthEvery   = 2 * time.Second
)

var endpointURLRE = regexp.MustCompile(`^http://127\.0\.0\.1:[0-9]{1,5}$`)

// Spec is everything needed to launch one installation.
type Spec struct {
	InstallationID   string
	FeatureID        string
	PackageDir       string
	DataDir          string
	RuntimeDir       string
	LogPath          string
	CredentialPath   string
	Command          []string
	IdentityURL      string
	IdentitySystemID string
}

// Policy is the per-installation runtime policy (Processor settings).
type Policy struct {
	Mode     string `json:"startup_mode"`
	Idle     int    `json:"idle_seconds"`
	Recovery string `json:"recovery"`
}

type Exit struct {
	At       time.Time `json:"at"`
	Code     int       `json:"code"`
	Expected bool      `json:"expected"`
	Reason   string    `json:"reason"`
}

type Health struct {
	Ready        bool           `json:"ready"`
	HeartbeatSeq float64        `json:"heartbeat_seq"`
	CheckedAt    time.Time      `json:"checked_at"`
	Error        string         `json:"error,omitempty"`
	Failures     int            `json:"consecutive_failures"`
	Status       map[string]any `json:"status,omitempty"`
}

// View is what the UI shows about one installation's process.
type View struct {
	State      string      `json:"state"`
	PID        int         `json:"pid,omitempty"`
	LaunchID   string      `json:"launch_id,omitempty"`
	URL        string      `json:"url,omitempty"`
	StartedAt  *time.Time  `json:"started_at,omitempty"`
	StartedBy  string      `json:"started_by,omitempty"`
	InFlight   int         `json:"in_flight"`
	IdleStopAt *time.Time  `json:"idle_stop_at,omitempty"`
	LastExit   *Exit       `json:"last_exit,omitempty"`
	Restarts   []time.Time `json:"recent_restarts"`
	Health     *Health     `json:"health,omitempty"`
	Note       string      `json:"note,omitempty"`
}

type proc struct {
	spec      Spec
	cmd       *exec.Cmd
	key       []byte
	launchID  string
	url       string
	state     string
	startedAt time.Time
	startedBy string
	stopping  bool
	inFlight  int
	idleTimer *time.Timer
	idleAt    *time.Time
	exited    chan struct{}
	lastExit  *Exit
	restarts  []time.Time
	health    *Health
	note      string
	pid       int
}

type Supervisor struct {
	mu      sync.Mutex
	procs   map[string]*proc
	Log     func(level, event, installation, detail string)
	Policy  func(installation string) Policy
	SpecFor func(installation string) (Spec, error)
	HTTP    *http.Client
	closed  bool
}

func New() *Supervisor {
	return &Supervisor{procs: map[string]*proc{}, HTTP: &http.Client{Timeout: 10 * time.Second,
		Transport: &http.Transport{Proxy: nil}}, Log: func(string, string, string, string) {}}
}

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func (s *Supervisor) get(id string) *proc {
	p := s.procs[id]
	if p == nil {
		p = &proc{state: Stopped}
		s.procs[id] = p
	}
	return p
}

// View returns the current process view (STOPPED if never started).
func (s *Supervisor) View(id string) View {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.get(id)
	v := View{State: p.state, PID: p.pid, LaunchID: p.launchID, URL: p.url, StartedBy: p.startedBy, InFlight: p.inFlight,
		LastExit: p.lastExit, Restarts: append([]time.Time{}, p.restarts...), Note: p.note, IdleStopAt: p.idleAt}
	if !p.startedAt.IsZero() && (p.state == Running || p.state == Starting || p.state == Stopping) {
		t := p.startedAt
		v.StartedAt = &t
	}
	if p.health != nil && p.state == Running {
		h := *p.health
		v.Health = &h
	}
	return v
}

func (s *Supervisor) Running(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.procs[id]
	return p != nil && p.state == Running
}

// Start launches the installation unless it is already running.
func (s *Supervisor) Start(ctx context.Context, id, by string) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return errors.New("Processor is shutting down")
	}
	p := s.get(id)
	switch p.state {
	case Running:
		s.mu.Unlock()
		return nil
	case Starting, Stopping, Restarting:
		st := p.state
		s.mu.Unlock()
		return fmt.Errorf("BUSY: the feature is %s; try again in a moment", st)
	}
	p.state, p.note, p.startedBy = Starting, "", by
	s.mu.Unlock()
	err := s.launch(ctx, id, by)
	if err != nil {
		s.mu.Lock()
		p.state, p.note = FailedToStart, err.Error()
		s.mu.Unlock()
		s.Log("warn", "feature.start_failed", id, err.Error())
	}
	return err
}

func (s *Supervisor) launch(ctx context.Context, id, by string) error {
	spec, err := s.SpecFor(id)
	if err != nil {
		return err
	}
	if spec.IdentityURL == "" || spec.IdentitySystemID == "" {
		return errors.New("NOT_CONNECTED: connect Processor to Identity first; a feature is started knowing which Identity decides for it")
	}
	argv := make([]string, len(spec.Command))
	for i, a := range spec.Command {
		argv[i] = strings.ReplaceAll(a, "{package}", spec.PackageDir)
	}
	exe := argv[0]
	if !strings.ContainsRune(exe, os.PathSeparator) && !strings.Contains(exe, "/") {
		lp, err := exec.LookPath(exe)
		if err != nil {
			return fmt.Errorf("REQUIREMENT_MISSING: %q is not installed or not on PATH (the feature needs it)", exe)
		}
		exe = lp
	}
	for _, d := range []string{spec.DataDir, spec.RuntimeDir, filepath.Dir(spec.LogPath)} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return err
		}
	}
	os.Remove(filepath.Join(spec.RuntimeDir, "endpoint.json"))
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return err
	}
	launchID := "lch_" + randHex(8)
	keyFile := filepath.Join(spec.RuntimeDir, "launch-key-"+launchID)
	f, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	f.WriteString(hex.EncodeToString(key))
	f.Close()
	logf, err := os.OpenFile(spec.LogPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		os.Remove(keyFile)
		return err
	}
	cmd := exec.Command(exe, argv[1:]...)
	cmd.Dir = spec.DataDir
	cmd.Stdout, cmd.Stderr = logf, logf
	cmd.Env = minimalEnv(map[string]string{
		"TF_API": API, "TF_INSTALLATION_ID": spec.InstallationID, "TF_FEATURE_ID": spec.FeatureID, "TF_LAUNCH_ID": launchID,
		"TF_DATA_DIR": spec.DataDir, "TF_RUNTIME_DIR": spec.RuntimeDir, "TF_LAUNCH_KEY_FILE": keyFile,
		"TF_CREDENTIAL_FILE": spec.CredentialPath, "TF_IDENTITY_URL": spec.IdentityURL, "TF_IDENTITY_SYSTEM_ID": spec.IdentitySystemID,
	})
	setProcAttrs(cmd)
	if err := cmd.Start(); err != nil {
		logf.Close()
		os.Remove(keyFile)
		return fmt.Errorf("could not start: %v", err)
	}
	logf.Close()
	exited := make(chan struct{})
	s.mu.Lock()
	p := s.get(id)
	p.spec, p.cmd, p.key, p.launchID, p.exited, p.stopping, p.pid = spec, cmd, key, launchID, exited, false, cmd.Process.Pid
	p.url, p.health = "", nil
	s.mu.Unlock()
	writeLaunchRecord(spec.RuntimeDir, launchRecord{PID: cmd.Process.Pid, LaunchID: launchID, Key: hex.EncodeToString(key), StartedAt: time.Now().UTC()})
	go s.wait(id, p, cmd, exited)
	s.Log("info", "feature.launching", id, fmt.Sprintf("pid %d launch %s (%s); waiting for its signed status", cmd.Process.Pid, launchID, by))

	// Discover the endpoint and authenticate it.
	deadline := time.Now().Add(startTimeout)
	var ep struct {
		API, URL, LaunchID, InstallationID string
		PID                                int
	}
	for {
		select {
		case <-exited:
			os.Remove(keyFile)
			return errors.New("the feature exited during start-up (see its log)")
		default:
		}
		if b, err := os.ReadFile(filepath.Join(spec.RuntimeDir, "endpoint.json")); err == nil {
			var raw map[string]any
			if json.Unmarshal(b, &raw) == nil {
				ep.API, _ = raw["api"].(string)
				ep.URL, _ = raw["url"].(string)
				ep.LaunchID, _ = raw["launch_id"].(string)
				ep.InstallationID, _ = raw["installation_id"].(string)
				if f, ok := raw["pid"].(float64); ok {
					ep.PID = int(f)
				}
				if ep.LaunchID == launchID {
					break
				}
			}
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			s.kill(p)
			os.Remove(keyFile)
			return errors.New("the feature did not publish its endpoint within 10 s")
		}
		time.Sleep(50 * time.Millisecond)
	}
	if ep.API != API || ep.InstallationID != spec.InstallationID || !endpointURLRE.MatchString(ep.URL) || ep.PID != cmd.Process.Pid {
		s.kill(p)
		return errors.New("WRONG_SERVICE: endpoint.json does not describe this launch (api, installation, loopback URL or pid differ)")
	}
	s.mu.Lock()
	p.url = ep.URL
	s.mu.Unlock()
	st, raw, err := s.call(p, "GET", "/v1/status", nil)
	var status map[string]any
	if err == nil {
		json.Unmarshal(raw, &status)
	}
	if err != nil || st != 200 || status["installation_id"] != spec.InstallationID || status["launch_id"] != launchID {
		s.kill(p)
		if err == nil {
			err = errors.New("status did not match this launch")
		}
		return fmt.Errorf("WRONG_SERVICE: could not authenticate the started feature (%v)", err)
	}
	if _, err := os.Stat(keyFile); err == nil {
		os.Remove(keyFile)
		s.Log("warn", "feature.key_file_left", id, "the feature did not delete its launch-key file; Processor removed it")
	}
	now := time.Now()
	s.mu.Lock()
	p.state, p.startedAt = Running, now
	p.health = &Health{Ready: status["ready"] == true, CheckedAt: now, Status: status}
	if f, ok := status["heartbeat_seq"].(float64); ok {
		p.health.HeartbeatSeq = f
	}
	s.mu.Unlock()
	go s.watch(id, p, exited)
	s.Log("info", "feature.running", id, "RUNNING at "+ep.URL+" (authenticated with this launch's key); it holds ZERO grants by itself")
	return nil
}

// minimalEnv passes only what an interpreter needs plus the TF_ contract,
// so Processor's own environment (tokens etc.) never leaks to a feature.
func minimalEnv(tf map[string]string) []string {
	keep := []string{"PATH", "HOME", "LANG", "LC_ALL", "TMPDIR", "TEMP", "TMP", "SYSTEMROOT", "WINDIR", "LOCALAPPDATA", "APPDATA", "USERPROFILE"}
	out := []string{}
	for _, k := range keep {
		if v, ok := os.LookupEnv(k); ok {
			out = append(out, k+"="+v)
		}
	}
	for k, v := range tf {
		out = append(out, k+"="+v)
	}
	return out
}

func (s *Supervisor) wait(id string, p *proc, cmd *exec.Cmd, exited chan struct{}) {
	err := cmd.Wait()
	code := 0
	if err != nil {
		code = -1
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			code = ee.ExitCode()
		}
	}
	close(exited)
	s.mu.Lock()
	if p.cmd != cmd { // superseded
		s.mu.Unlock()
		return
	}
	if p.idleTimer != nil {
		p.idleTimer.Stop()
		p.idleTimer, p.idleAt = nil, nil
	}
	ex := &Exit{At: time.Now().UTC(), Code: code, Expected: p.stopping}
	p.lastExit, p.url, p.health = ex, "", nil
	removeLaunchRecord(p.spec.RuntimeDir)
	if p.stopping {
		ex.Reason = "stopped by Processor"
		p.state = Stopped
		s.mu.Unlock()
		s.Log("info", "feature.stopped", id, "process exited cleanly after a stop request; installation identity and data kept")
		return
	}
	if p.state == Starting {
		// Died before it proved itself: a failed start, never "recovered".
		ex.Reason = "exited during start-up"
		p.state, p.note = FailedToStart, fmt.Sprintf("exited during start-up (code %d); see its log", code)
		s.mu.Unlock()
		return
	}
	ex.Reason = "exited unexpectedly"
	inflight := p.inFlight
	pol := s.Policy(id)
	now := time.Now()
	recent := []time.Time{}
	for _, t := range p.restarts {
		if now.Sub(t) < restartWindow {
			recent = append(recent, t)
		}
	}
	p.restarts = recent
	if s.closed || pol.Recovery != RecoveryBounded || len(recent) >= maxRestarts {
		p.state = NeedsAttention
		switch {
		case pol.Recovery != RecoveryBounded:
			p.note = "crashed; recovery is MANUAL for this feature (default for experimental features) — start it again yourself"
		case len(recent) >= maxRestarts:
			p.note = "crashed again after " + strconv.Itoa(len(recent)) + " automatic restarts in 10 minutes — stopped trying (crash loop)"
		}
		note := p.note
		s.mu.Unlock()
		s.Log("warn", "feature.crashed", id, fmt.Sprintf("exit code %d; %d request(s) in flight are UNCERTAIN and are NOT resent; %s", code, inflight, note))
		return
	}
	p.state = Restarting
	backoff := time.Duration(1<<len(recent)) * time.Second
	p.restarts = append(p.restarts, now)
	p.note = fmt.Sprintf("crashed; bounded recovery: restart %d of %d in %s (requests are never resent)", len(p.restarts), maxRestarts, backoff)
	note := p.note
	s.mu.Unlock()
	s.Log("warn", "feature.crashed", id, fmt.Sprintf("exit code %d; %d request(s) UNCERTAIN, not resent; %s", code, inflight, note))
	time.AfterFunc(backoff, func() {
		s.mu.Lock()
		if p.state != Restarting || s.closed {
			s.mu.Unlock()
			return
		}
		p.state = Stopped
		s.mu.Unlock()
		s.Start(context.Background(), id, "bounded recovery")
	})
}

func (s *Supervisor) watch(id string, p *proc, exited chan struct{}) {
	t := time.NewTicker(healthEvery)
	defer t.Stop()
	for {
		select {
		case <-exited:
			return
		case <-t.C:
		}
		st, raw, err := s.call(p, "GET", "/v1/status", nil)
		var status map[string]any
		if err == nil && st == 200 {
			json.Unmarshal(raw, &status)
		}
		s.mu.Lock()
		if p.state != Running || p.health == nil {
			s.mu.Unlock()
			continue
		}
		p.health.CheckedAt = time.Now()
		if status == nil || status["launch_id"] != p.launchID {
			p.health.Failures++
			p.health.Ready = false
			p.health.Error = "no valid signed status"
			if err != nil {
				p.health.Error = err.Error()
			}
		} else {
			p.health.Failures, p.health.Error, p.health.Status = 0, "", status
			p.health.Ready = status["ready"] == true
			if f, ok := status["heartbeat_seq"].(float64); ok {
				p.health.HeartbeatSeq = f
			}
		}
		s.mu.Unlock()
	}
}

// Call makes a launcher-signed request to a running feature.
func (s *Supervisor) Call(id, method, path string, body []byte) (int, []byte, error) {
	s.mu.Lock()
	p := s.procs[id]
	if p == nil || p.state != Running {
		s.mu.Unlock()
		return 0, nil, errors.New("NOT_RUNNING")
	}
	s.mu.Unlock()
	return s.call(p, method, path, body)
}

func hmacHex(key []byte, parts ...[]byte) string {
	m := hmac.New(sha256.New, key)
	for _, x := range parts {
		m.Write(x)
	}
	return hex.EncodeToString(m.Sum(nil))
}

func (s *Supervisor) call(p *proc, method, path string, body []byte) (int, []byte, error) {
	s.mu.Lock()
	u, key, launch, ins := p.url, p.key, p.launchID, p.spec.InstallationID
	s.mu.Unlock()
	if u == "" {
		return 0, nil, errors.New("no endpoint")
	}
	req, err := http.NewRequest(method, u+path, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := randHex(12)
	sum := sha256.Sum256(body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TF-Time", ts)
	req.Header.Set("X-TF-Nonce", nonce)
	req.Header.Set("X-TF-Auth", hmacHex(key, []byte(API+" request\n"+method+"\n"+path+"\n"+ts+"\n"+nonce+"\n"+hex.EncodeToString(sum[:]))))
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, nil, err
	}
	want := hmacHex(key, []byte(API+" response\n"+nonce+"\n"+strconv.Itoa(resp.StatusCode)+"\n"), raw)
	if !hmac.Equal([]byte(want), []byte(resp.Header.Get("X-TF-Proof"))) || resp.Header.Get("X-TF-Installation-ID") != ins || resp.Header.Get("X-TF-Launch-ID") != launch {
		return resp.StatusCode, nil, errors.New("WRONG_SERVICE: answer not signed by this launch of the feature")
	}
	return resp.StatusCode, raw, nil
}

// URL returns the running endpoint (for protected requests that carry a ticket).
func (s *Supervisor) URL(id string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p := s.procs[id]; p != nil && p.state == Running {
		return p.url
	}
	return ""
}

// Begin/End track Processor-sent work, so idle stop never interrupts it.
func (s *Supervisor) Begin(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.get(id)
	p.inFlight++
	if p.idleTimer != nil {
		p.idleTimer.Stop()
		p.idleTimer, p.idleAt = nil, nil
	}
}

func (s *Supervisor) End(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.get(id)
	if p.inFlight > 0 {
		p.inFlight--
	}
	s.armIdle(id, p)
}

func (s *Supervisor) armIdle(id string, p *proc) {
	pol := s.Policy(id)
	if pol.Mode != ModeOnDemand || p.inFlight > 0 || p.state != Running {
		return
	}
	idle := time.Duration(pol.Idle) * time.Second
	if idle <= 0 {
		idle = 30 * time.Second
	}
	if p.idleTimer != nil {
		p.idleTimer.Stop()
	}
	at := time.Now().Add(idle)
	p.idleAt = &at
	p.idleTimer = time.AfterFunc(idle, func() {
		s.mu.Lock()
		ok := p.inFlight == 0 && p.state == Running && s.Policy(id).Mode == ModeOnDemand
		p.idleTimer, p.idleAt = nil, nil
		s.mu.Unlock()
		if ok {
			s.Log("info", "feature.idle_stop", id, fmt.Sprintf("on-demand feature idle for %s with no work in flight; stopping it cleanly", idle))
			s.Stop(id, "idle window ended")
		}
	})
}

// ArmIdle is called after an on-demand start that was not followed by work.
func (s *Supervisor) ArmIdle(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.armIdle(id, s.get(id))
}

// Stop asks the feature to stop (it finishes in-flight work first), then
// escalates. It returns when the process is gone.
func (s *Supervisor) Stop(id, reason string) error {
	s.mu.Lock()
	p := s.procs[id]
	if p == nil || p.cmd == nil || (p.state != Running && p.state != Starting && p.state != Restarting) {
		if p != nil && (p.state == NeedsAttention || p.state == FailedToStart || p.state == Restarting) {
			p.state, p.note = Stopped, "cleared by "+reason
		}
		s.mu.Unlock()
		return nil
	}
	if p.idleTimer != nil {
		p.idleTimer.Stop()
		p.idleTimer, p.idleAt = nil, nil
	}
	p.stopping, p.state = true, Stopping
	exited := p.exited
	s.mu.Unlock()
	s.Log("info", "feature.stopping", id, reason)
	s.call(p, "POST", "/v1/shutdown", []byte("{}"))
	select {
	case <-exited:
		return nil
	case <-time.After(stopGrace):
	}
	s.Log("warn", "feature.stop_escalated", id, "did not exit in time; terminating its process group")
	terminate(p.cmd)
	select {
	case <-exited:
		return nil
	case <-time.After(3 * time.Second):
	}
	killHard(p.cmd)
	<-exited
	return nil
}

func (s *Supervisor) kill(p *proc) {
	s.mu.Lock()
	p.stopping = true
	cmd, exited := p.cmd, p.exited
	s.mu.Unlock()
	if cmd == nil {
		return
	}
	terminate(cmd)
	select {
	case <-exited:
	case <-time.After(3 * time.Second):
		killHard(cmd)
		<-exited
	}
}

// StopAll stops every feature this Processor started (Processor exit).
func (s *Supervisor) StopAll(reason string) int {
	s.mu.Lock()
	s.closed = true
	ids := []string{}
	for id, p := range s.procs {
		if p.cmd != nil && (p.state == Running || p.state == Starting || p.state == Restarting) {
			ids = append(ids, id)
		}
	}
	s.mu.Unlock()
	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func(id string) { defer wg.Done(); s.Stop(id, reason) }(id)
	}
	wg.Wait()
	return len(ids)
}

// ---- launch records (orphan cleanup after a Processor crash) ----

type launchRecord struct {
	PID       int       `json:"pid"`
	LaunchID  string    `json:"launch_id"`
	Key       string    `json:"launch_key"`
	StartedAt time.Time `json:"started_at"`
}

func writeLaunchRecord(dir string, r launchRecord) {
	b, _ := json.Marshal(r)
	tmp := filepath.Join(dir, ".launch.json.tmp")
	if os.WriteFile(tmp, b, 0o600) == nil {
		os.Rename(tmp, filepath.Join(dir, "launch.json"))
	}
}

func removeLaunchRecord(dir string) {
	os.Remove(filepath.Join(dir, "launch.json"))
	os.Remove(filepath.Join(dir, "endpoint.json"))
}

// CleanupOrphan handles a launch record left by a Processor that crashed.
// It stops the old process only after that process proves, with the old
// launch key, that it is the launch the record describes.
func (s *Supervisor) CleanupOrphan(id, runtimeDir string) string {
	b, err := os.ReadFile(filepath.Join(runtimeDir, "launch.json"))
	if err != nil {
		return ""
	}
	var r launchRecord
	if json.Unmarshal(b, &r) != nil {
		removeLaunchRecord(runtimeDir)
		return "stale launch record was unreadable; removed"
	}
	key, kerr := hex.DecodeString(r.Key)
	var ep map[string]any
	eb, _ := os.ReadFile(filepath.Join(runtimeDir, "endpoint.json"))
	json.Unmarshal(eb, &ep)
	u, _ := ep["url"].(string)
	if kerr != nil || !endpointURLRE.MatchString(u) {
		removeLaunchRecord(runtimeDir)
		return "stale launch record (no endpoint); removed — nothing was killed"
	}
	p := &proc{url: u, key: key, launchID: r.LaunchID, spec: Spec{InstallationID: id}}
	st, raw, err := s.call(p, "GET", "/v1/status", nil)
	var status map[string]any
	json.Unmarshal(raw, &status)
	if err != nil || st != 200 || status["launch_id"] != r.LaunchID {
		removeLaunchRecord(runtimeDir)
		return "stale launch record: nothing answered as that launch; removed — nothing was killed"
	}
	s.call(p, "POST", "/v1/shutdown", []byte("{}"))
	deadline := time.Now().Add(stopGrace)
	for time.Now().Before(deadline) {
		if !pidAlive(r.PID) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if pidAlive(r.PID) {
		terminatePID(r.PID)
	}
	removeLaunchRecord(runtimeDir)
	return fmt.Sprintf("ORPHAN_STOPPED: launch %s (pid %d) from an earlier Processor run proved itself with its launch key and was stopped", r.LaunchID, r.PID)
}

var _ = runtime.GOOS
