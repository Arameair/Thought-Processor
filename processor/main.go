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

func main() {
	port := flag.Int("port", 7860, "loopback port (0 = any free port)")
	data := flag.String("data", defaultData(), "profile folder for this Processor")
	noBrowser := flag.Bool("no-browser", false, "don't open the browser; just print the private link")
	version := flag.Bool("version", false, "print version and exit")
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
		fmt.Println("  stop    : use Exit in the page, or Ctrl+C here.")
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
