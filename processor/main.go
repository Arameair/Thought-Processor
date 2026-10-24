// Thought Processor — standalone diagnostic/setup shell (experiment).
//
// This build is deliberately small. It reports its own real state, inspects
// explicitly configured same-host Identity / Thought Engine candidates in a
// read-only way, and REFUSES every protected Engine operation (start, attach,
// stop, observation ingest) because no Identity lifecycle-approval producer
// exists yet. It contains no process runner: it cannot start or kill anything.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"
)

const (
	product    = "thought-processor"
	protocol   = "thought-processor.diagnostic/0-experimental"
	appVersion = "0.1.0-experiment"
)

// BuildInfo is what the Go toolchain actually embedded. Anything missing is
// reported as "unknown" — never guessed.
type BuildInfo struct {
	Version   string `json:"version"`
	Revision  string `json:"revision"`
	Modified  string `json:"modified"` // "true" | "false" | "unknown"
	BuiltAt   string `json:"vcs_time"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

func readBuild() BuildInfo {
	b := BuildInfo{Version: appVersion, Revision: "unknown", Modified: "unknown", BuiltAt: "unknown",
		GoVersion: runtime.Version(), OS: runtime.GOOS, Arch: runtime.GOARCH}
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				b.Revision = s.Value
			case "vcs.modified":
				b.Modified = s.Value
			case "vcs.time":
				b.BuiltAt = s.Value
			}
		}
	}
	return b
}

type Config struct {
	DataDir string
	Port    int
}

func main() {
	dataFlag := flag.String("data", "", "profile folder (default: $THOUGHT_PROCESSOR_DATA, then ~/.thought-processor/default)")
	defPort := 7860
	if v := os.Getenv("PORT"); v != "" { // set by Project-controller
		if n, err := strconv.Atoi(v); err == nil {
			defPort = n
		}
	}
	port := flag.Int("port", defPort, "loopback port to listen on (0 = any free port; default $PORT, then 7860)")
	flag.Parse()

	dir := *dataFlag
	if dir == "" {
		dir = os.Getenv("THOUGHT_PROCESSOR_DATA")
	}
	if dir == "" && os.Getenv("PC_DATA") != "" { // Project-controller's per-branch data folder
		dir = filepath.Join(os.Getenv("PC_DATA"), "profile")
	}
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: no -data folder given and home folder unknown:", err)
			os.Exit(2)
		}
		dir = filepath.Join(home, ".thought-processor", "default")
	}
	if *port < 0 || *port > 65535 {
		fmt.Fprintln(os.Stderr, "error: -port must be 0–65535")
		os.Exit(2)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	quit := make(chan struct{})
	go func() { <-stop; close(quit) }()
	if err := run(Config{DataDir: abs, Port: *port}, quit, nil, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func newID(prefix string) string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return prefix + hex.EncodeToString(b)
}

// run serves until quit is closed or an operator requests shutdown.
// ready (tests) receives the actual listening address.
func run(cfg Config, quit <-chan struct{}, ready chan<- *App, out interface{ Write([]byte) (int, error) }) error {
	t0 := time.Now()
	prof, err := OpenProfile(cfg.DataDir)
	if err != nil {
		return err
	}
	defer prof.Close()

	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(cfg.Port)))
	if err != nil {
		var hint string
		if errors.Is(err, syscall.EADDRINUSE) {
			hint = " — another program already uses this port. Choose a free one with -port (Processor will not stop the other program)."
		}
		return fmt.Errorf("can't listen on 127.0.0.1:%d: %w%s", cfg.Port, err, hint)
	}
	port := ln.Addr().(*net.TCPAddr).Port

	app, err := NewApp(prof, port, readBuild(), t0)
	if err != nil {
		ln.Close()
		return err
	}
	srv := &http.Server{Handler: app.Routes(), ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, MaxHeaderBytes: 16 << 10}
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()

	app.startupMS = time.Since(t0).Milliseconds()
	app.log.Add("lifecycle", "started", "", fmt.Sprintf("listening on 127.0.0.1:%d pid %d run %s (startup %d ms)", port, os.Getpid(), app.runID, app.startupMS))
	if out != nil {
		fmt.Fprintf(out, "Thought Processor %s (%s)\n", appVersion, shortRev(app.build.Revision))
		fmt.Fprintf(out, "  profile : %s\n", prof.Dir)
		fmt.Fprintf(out, "  run     : %s   pid %d   started in %d ms\n", app.runID, os.Getpid(), app.startupMS)
		fmt.Fprintf(out, "  open    : http://127.0.0.1:%d/#code=%s\n", port, app.sess.Code())
		fmt.Fprintf(out, "            (the code is also in %s; it changes every run)\n", prof.CodePath())
		fmt.Fprintln(out, "  stop    : Ctrl+C here, or Stop Processor in the page")
	}
	if ready != nil {
		ready <- app
	}

	select {
	case <-quit:
	case <-app.shutdownReq:
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}
	t1 := time.Now()
	app.log.Add("lifecycle", "stopping", "", "graceful shutdown requested")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	prof.RecordCleanShutdown(app.runID, time.Since(t1))
	app.log.Add("lifecycle", "stopped", "", fmt.Sprintf("clean shutdown in %d ms", time.Since(t1).Milliseconds()))
	app.log.Close()
	if out != nil {
		fmt.Fprintf(out, "Thought Processor stopped cleanly (%d ms). Nothing else was stopped.\n", time.Since(t1).Milliseconds())
	}
	return nil
}

func shortRev(r string) string {
	if len(r) > 12 {
		return r[:12]
	}
	return r
}
