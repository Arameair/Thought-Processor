package supervisor

// These tests start REAL processes: the trusted Python test feature from
// testfeatures/, plus small throwaway scripts that misbehave on purpose.
// They need python3; they are skipped (and say so) without it.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func needPython(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not found: supervisor process tests NOT RUN")
	}
	if goruntime.GOOS == "windows" {
		t.Skip("process-group tests are Linux/WSL only")
	}
}

func workerDir() string {
	_, f, _, _ := goruntime.Caller(0)
	return filepath.Join(filepath.Dir(f), "..", "..", "testfeatures", "deterministic-worker")
}

type env struct {
	sup    *Supervisor
	spec   Spec
	policy Policy
	mu     sync.Mutex
	logs   []string
}

func newEnv(t *testing.T, cmd []string, pkg string) *env {
	root := t.TempDir()
	e := &env{policy: Policy{Mode: ModeManual, Idle: 30, Recovery: RecoveryManual}}
	cred := filepath.Join(root, "credential.json")
	b, _ := json.Marshal(map[string]any{"installation_id": "ins_00000000000000000001", "secret": strings.Repeat("ab", 32), "identity_system_id": "idn_test"})
	os.WriteFile(cred, b, 0o600)
	e.spec = Spec{InstallationID: "ins_00000000000000000001", FeatureID: "p1-deterministic-worker", PackageDir: pkg,
		DataDir: filepath.Join(root, "data"), RuntimeDir: filepath.Join(root, "runtime"), LogPath: filepath.Join(root, "logs", "f.log"),
		CredentialPath: cred, Command: cmd, IdentityURL: "http://127.0.0.1:9", IdentitySystemID: "idn_test"}
	e.sup = New()
	e.sup.SpecFor = func(string) (Spec, error) { return e.spec, nil }
	e.sup.Policy = func(string) Policy { e.mu.Lock(); defer e.mu.Unlock(); return e.policy }
	e.sup.Log = func(level, event, ins, detail string) {
		e.mu.Lock()
		e.logs = append(e.logs, event+": "+detail)
		e.mu.Unlock()
	}
	t.Cleanup(func() { e.sup.StopAll("test cleanup") })
	return e
}

func waitState(t *testing.T, s *Supervisor, id, want string, d time.Duration) View {
	end := time.Now().Add(d)
	for time.Now().Before(end) {
		if v := s.View(id); v.State == want {
			return v
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("state %s never reached (now %+v)", want, s.View(id))
	return View{}
}

const id = "ins_00000000000000000001"

func TestRealWorkerStartStatusStop(t *testing.T) {
	needPython(t)
	e := newEnv(t, []string{"python3", "-I", "{package}/worker.py"}, workerDir())
	if err := e.sup.Start(context.Background(), id, "test"); err != nil {
		t.Fatal(err)
	}
	v := e.sup.View(id)
	if v.State != Running || !strings.HasPrefix(v.LaunchID, "lch_") || !strings.HasPrefix(v.URL, "http://127.0.0.1:") {
		t.Fatalf("%+v", v)
	}
	if _, err := os.Stat(filepath.Join(e.spec.RuntimeDir, "launch-key-"+v.LaunchID)); err == nil {
		t.Fatal("launch key file left on disk")
	}
	st, raw, err := e.sup.Call(id, "POST", "/v1/self-test", []byte("{}"))
	if err != nil || st != 200 || !strings.Contains(string(raw), `"PASS"`) {
		t.Fatalf("self-test %d %s %v", st, raw, err)
	}
	// Unauthenticated caller gets nothing.
	resp, err := http.Get(v.URL + "/v1/status")
	if err != nil || resp.StatusCode != 401 {
		t.Fatalf("unauthenticated status: %v %d", err, resp.StatusCode)
	}
	pid := v.PID
	e.sup.Stop(id, "test")
	if v := e.sup.View(id); v.State != Stopped || v.LastExit == nil || !v.LastExit.Expected {
		t.Fatalf("after stop %+v", v)
	}
	if pidAlive(pid) {
		t.Fatal("process still alive after stop")
	}
}

func script(t *testing.T, body string) string {
	d := t.TempDir()
	os.WriteFile(filepath.Join(d, "f.py"), []byte(body), 0o600)
	return d
}

func TestWrongEndpointAndEarlyExitRefused(t *testing.T) {
	needPython(t)
	// Publishes an endpoint for a DIFFERENT installation.
	liar := script(t, `import json,os,time
p=os.path.join(os.environ["TF_RUNTIME_DIR"],"endpoint.json")
json.dump({"api":"thought.feature-api/1","installation_id":"ins_someoneelse0000000000","launch_id":os.environ["TF_LAUNCH_ID"],"pid":os.getpid(),"url":"http://127.0.0.1:1"},open(p,"w"))
time.sleep(30)`)
	e := newEnv(t, []string{"python3", "-I", "{package}/f.py"}, liar)
	err := e.sup.Start(context.Background(), id, "test")
	if err == nil || !strings.Contains(err.Error(), "WRONG_SERVICE") {
		t.Fatalf("lying endpoint accepted: %v", err)
	}
	if v := e.sup.View(id); v.State != FailedToStart {
		t.Fatalf("%+v", v)
	}
	quitter := script(t, "import sys; sys.exit(3)\n")
	e2 := newEnv(t, []string{"python3", "-I", "{package}/f.py"}, quitter)
	if err := e2.sup.Start(context.Background(), id, "test"); err == nil || !strings.Contains(err.Error(), "exited during start-up") {
		t.Fatalf("early exit: %v", err)
	}
}

// An impostor on the feature's address can't produce answers Processor accepts.
func TestImpostorEndpointIsWrongService(t *testing.T) {
	imp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-TF-Installation-ID", id)
		w.Header().Set("X-TF-Launch-ID", "lch_x")
		w.Header().Set("X-TF-Proof", strings.Repeat("0", 64))
		w.Write([]byte(`{"installation_id":"` + id + `","launch_id":"lch_x","ready":true}`))
	}))
	defer imp.Close()
	s := New()
	p := &proc{url: imp.URL, key: []byte("0123456789abcdef0123456789abcdef"), launchID: "lch_x", spec: Spec{InstallationID: id}}
	if _, _, err := s.call(p, "GET", "/v1/status", nil); err == nil || !strings.Contains(err.Error(), "WRONG_SERVICE") {
		t.Fatalf("impostor accepted: %v", err)
	}
}

func TestCrashManualThenBoundedRecovery(t *testing.T) {
	needPython(t)
	// Starts properly (it IS the real worker, signed status), then dies after 1 s.
	w := filepath.Join(workerDir(), "worker.py")
	crasher := script(t, "import os,sys,threading,runpy\nthreading.Timer(1.0, lambda: os._exit(7)).start()\nsys.argv=["+strconvQuote(w)+"]\nrunpy.run_path("+strconvQuote(w)+", run_name='__main__')\n")
	e := newEnv(t, []string{"python3", "{package}/f.py"}, crasher)
	if err := e.sup.Start(context.Background(), id, "test"); err != nil {
		t.Fatal(err)
	}
	v := waitState(t, e.sup, id, NeedsAttention, 10*time.Second)
	if v.LastExit == nil || v.LastExit.Expected || len(v.Restarts) != 0 {
		t.Fatalf("manual recovery must not restart: %+v", v)
	}
	e.mu.Lock()
	e.policy.Recovery = RecoveryBounded
	e.mu.Unlock()
	e.sup.Stop(id, "clear")
	if err := e.sup.Start(context.Background(), id, "test"); err != nil {
		t.Fatal(err)
	}
	v = waitState(t, e.sup, id, NeedsAttention, 40*time.Second)
	if len(v.Restarts) != maxRestarts || !strings.Contains(v.Note, "crash loop") {
		t.Fatalf("bounded recovery: %+v", v)
	}
}

func strconvQuote(s string) string { b, _ := json.Marshal(s); return string(b) }

func TestIdleStopNeverDuringWork(t *testing.T) {
	needPython(t)
	e := newEnv(t, []string{"python3", "-I", "{package}/worker.py"}, workerDir())
	e.policy = Policy{Mode: ModeOnDemand, Idle: 1, Recovery: RecoveryManual}
	if err := e.sup.Start(context.Background(), id, "test"); err != nil {
		t.Fatal(err)
	}
	e.sup.Begin(id) // work in flight
	time.Sleep(2500 * time.Millisecond)
	if v := e.sup.View(id); v.State != Running {
		t.Fatalf("stopped during work: %+v", v)
	}
	e.sup.End(id)
	waitState(t, e.sup, id, Stopped, 10*time.Second)
}

func TestVerifyAnswerRejectsUnsignedOrOtherLaunch(t *testing.T) {
	s := New()
	key := []byte("0123456789abcdef0123456789abcdef")
	s.procs[id] = &proc{key: key, launchID: "lch_1", spec: Spec{InstallationID: id}, state: Running}
	body := []byte(`{"result":"COMPLETED"}`)
	h := http.Header{}
	h.Set("X-TF-Installation-ID", id)
	h.Set("X-TF-Launch-ID", "lch_1")
	h.Set("X-TF-Proof", hmacHex(key, []byte(API+" response\nn1\n200\n"), body))
	if err := s.VerifyAnswer(id, "n1", 200, body, h); err != nil {
		t.Fatalf("genuine answer refused: %v", err)
	}
	if err := s.VerifyAnswer(id, "n2", 200, body, h); err == nil {
		t.Fatal("answer for another nonce accepted")
	}
	if err := s.VerifyAnswer(id, "n1", 200, []byte(`{"result":"FAKED"}`), h); err == nil {
		t.Fatal("changed body accepted")
	}
	h.Set("X-TF-Launch-ID", "lch_old")
	if err := s.VerifyAnswer(id, "n1", 200, body, h); err == nil {
		t.Fatal("answer from another launch accepted")
	}
}
