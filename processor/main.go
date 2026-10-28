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
	"errors"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"syscall"

	"github.com/Arameair/Thought-Processor/processor/internal/app"
	"github.com/Arameair/Thought-Processor/processor/internal/store"
)

const appVersion = "0.2.0-p0"

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

// handoffPrivateLink writes the URL ONLY into a caller-owned anonymous pipe.
// A regular file, socket, or a terminal is rejected to prevent persistence.
// The caller must pass an inherited pipe descriptor (3 or greater).
func handoffPrivateLink(fd int, link string) error {
  if fd < 3 { return errors.New("private link requires an inherited pipe fd >= 3") }
  pipe := os.NewFile(uintptr(fd), "private-link-pipe")
  if pipe == nil { return errors.New("private link pipe not available") }
  defer pipe.Close()
  st, err := pipe.Stat()
  if err != nil { return fmt.Errorf("private link pipe unavailable: %w", err) }
  if st.Mode()&os.ModeNamedPipe == 0 { return errors.New("private link target must be an anonymous pipe") }
  if _, err := fmt.Fprintln(pipe, link); err != nil { return fmt.Errorf("private link pipe write failed: %w", err) }
  return nil
}

func main() {
	port := flag.Int("port", 7860, "loopback port (0 = any free port)")
	data := flag.String("data", defaultData(), "profile folder for this Processor")
	noBrowser := flag.Bool("no-browser", false, "don't open the browser; just print the private link")
	version := flag.Bool("version", false, "print version and exit")
	privateLinkFD := flag.Int("private-link-fd", -1, "one-time private session link to inherited anonymous pipe FD (runner mode; never stdout)")
	flag.Parse()
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
	a.Port = *port
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var privateHandoffErr error
	err = a.Run(ctx, nil, func(link string) {
		rev := b.Revision
		if len(rev) > 12 {
			rev = rev[:12]
		}
		fmt.Printf("Thought Processor %s  branch %s  commit %s\n", appVersion, b.Branch, rev)
		fmt.Printf("  profile : %s\n", dir)
		fmt.Printf("  run     : pid %d   port %d (this computer only)   processor %s\n", os.Getpid(), a.Port, prof.Self.ProcessorID)
		// NEVER print the browser credential to stdout/stderr or a file.
		// In runner mode the one-time URL travels only over inherited pipe FD.
		if *privateLinkFD >= 0 {
			if err := handoffPrivateLink(*privateLinkFD, link); err != nil {
				privateHandoffErr = err
				fmt.Fprintln(os.Stderr, "private session handoff failed; stopping without disclosing link")
				stop()
			}
		} else if *noBrowser {
			fmt.Println("  open    : DISABLED (no private link channel provided)")
		}
		fmt.Println("  stop    : use Exit in the page, or Ctrl+C here.")
		if !*noBrowser && *privateLinkFD < 0 {
			if err := openBrowser(link); err != nil {
				fmt.Println("  (couldn't open the browser; copy the link above into it)")
			}
		}
	})
	if privateHandoffErr != nil {
		fmt.Fprintln(os.Stderr, "Thought Processor stopped: private handoff failed")
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Thought Processor stopped with an error:", err)
		os.Exit(1)
	}
	if n := a.ExitNote(); n != "" {
		fmt.Println(n)
	}
	fmt.Println("Thought Processor stopped.")
}
