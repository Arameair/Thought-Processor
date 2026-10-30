// Thought Processor — P0 prototype: connect to Identity, choose an identity,
// discover features, and install them with the reusable installer.
//
//	thought-processor.exe [-port 7860] [-data <profile folder>] [-no-browser]
//
// No Project Controller needed. Processor is the person's interface; Identity
// (a separate program) is the authority behind it.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"syscall"

	"github.com/Arameair/Thought-Processor/processor/internal/app"
	"github.com/Arameair/Thought-Processor/processor/internal/store"
)

const appVersion = "0.3.0-p1"

var buildBranch = "" // -ldflags "-X main.buildBranch=<branch>"

func build() app.Build {
	b := app.Build{Version: appVersion, Branch: buildBranch, Revision: "UNKNOWN", Modified: "UNKNOWN"}
	if b.Branch == "" {
		b.Branch = "UNKNOWN"
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				b.Revision = s.Value
			case "vcs.modified":
				b.Modified = s.Value
			}
		}
	}
	return b
}

func defaultData() string {
	if v := os.Getenv("THOUGHT_PROCESSOR_DATA"); v != "" {
		return v
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".thought-processor", "p0")
	}
	return "thought-processor-data"
}

func openBrowser(link string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", link)
	case "darwin":
		cmd = exec.Command("open", link)
	default:
		cmd = exec.Command("xdg-open", link)
	}
	return cmd.Start()
}

func main() {
	port := flag.Int("port", 7860, "loopback port (0 = any free port)")
	data := flag.String("data", defaultData(), "profile folder for this Processor")
	noBrowser := flag.Bool("no-browser", false, "don't open the browser; just print the private link")
	var wsRepos multiFlag
	flag.Var(&wsRepos, "workstation-repo", "name=path of a local Git checkout the Development Workstation may show and test (repeatable; read-only)")
	wsTestGitHub := flag.Bool("workstation-allow-test-github", false, "TEST ONLY: let the workstation accept Identity's GitHub TEST FIXTURE as verified. Never use outside tests")
	noLinkFile := flag.Bool("no-link-file", false, "don't save the private link to runtime/operator-link.txt (for launchers that read it from stdout)")
	version := flag.Bool("version", false, "print version and exit")
	defURL := flag.String("identity-default-url", "", "well-known Identity address to probe before pairing (default http://127.0.0.1:7790; \"-\" = don't probe). Same as TP_IDENTITY_DEFAULT_URL")
	var trusted, pins multiFlag
	flag.Var(&pins, "trust-feature-artifact", "pin the exact artifact SHA-256 of a local test feature you reviewed (repeatable). Built-in: the P1 test fixture. The manifest's own trust_class never grants trust")
	testFaults := flag.Bool("test-feature-faults", false, "TEST ONLY: let the trusted test fixture honour its TEST-ONLY fault-injection file (crash/slow after an ALLOW). Never use outside tests")
	flag.Var(&trusted, "trust-feature-dir", "a folder of LOCAL feature packages you trust (repeatable). Only TRUSTED_LOCAL_TEST_FIXTURE packages from such folders can be installed; nothing else can run")
	flag.Parse()
	if *defURL != "" {
		os.Setenv("TP_IDENTITY_DEFAULT_URL", *defURL)
	}
	b := build()
	if *version {
		fmt.Println("thought-processor", appVersion, b.Branch, "commit", b.Revision)
		return
	}
	dir, _ := filepath.Abs(*data)
	prof, err := store.Open(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Thought Processor could not start:", err)
		os.Exit(1)
	}
	defer prof.Close()
	a := app.New(prof, b)
	a.NoLinkFile = *noLinkFile
	a.WS.AllowTestGitHub = *wsTestGitHub
	for _, spec := range wsRepos {
		if err := a.WS.AddRepo(spec); err != nil {
			fmt.Fprintln(os.Stderr, "Thought Processor could not start:", err)
			os.Exit(2)
		}
	}
	a.TrustFeatureDirs(trusted)
	if err := a.TrustArtifacts(pins); err != nil {
		fmt.Fprintln(os.Stderr, "Thought Processor could not start:", err)
		os.Exit(2)
	}
	if *testFaults {
		a.EnableTestFaults()
	}
	a.Port = *port
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	err = a.Run(ctx, nil, func(link string) {
		rev := b.Revision
		if len(rev) > 12 {
			rev = rev[:12]
		}
		fmt.Printf("Thought Processor %s  branch %s  commit %s\n", appVersion, b.Branch, rev)
		fmt.Printf("  profile : %s\n", dir)
		fmt.Printf("  run     : pid %d   port %d (this computer only)   processor %s\n", os.Getpid(), a.Port, prof.Self.ProcessorID)
		fmt.Println("  open    : " + link)
		fmt.Println("            (private link — don't share it; it changes every start)")
		if *testFaults {
			fmt.Println("  TEST MODE: -test-feature-faults is on (test fixture fault injection allowed). Not for normal use.")
		}
		for _, p := range pins {
			fmt.Println("  pinned  : artifact " + p + "  (operator pin)")
		}
		for _, d := range trusted {
			fmt.Println("  trusted : " + d + "  (local feature packages you chose to trust)")
		}
		fmt.Println("  stop    : use Exit in the page, or Ctrl+C here (feature processes it started are stopped too).")
		if !*noBrowser {
			if err := openBrowser(link); err != nil {
				fmt.Println("  (couldn't open the browser; copy the link above into it)")
			}
		}
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Thought Processor stopped with an error:", err)
		os.Exit(1)
	}
	if n := a.ExitNote(); n != "" {
		fmt.Println(n)
	}
	fmt.Println("Thought Processor stopped.")
}

// multiFlag collects a repeatable string flag.
type multiFlag []string

func (m *multiFlag) String() string     { return fmt.Sprint(*m) }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }
