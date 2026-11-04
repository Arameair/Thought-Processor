// tass-core: Thought-Processor's side of TASS's core.
//
//	tass-core          run: talk to TASS on 127.0.0.1:8795; he thinks through the Thought Engine (Nemotron);
//	                   GitHub syncs separately every 20 s, backing off while it can't be reached
//	tass-core -once    one GitHub sync, print what happened, exit
//
// Settings (environment):
//
//	TASS_DATA         local home: tass.md, Identity's policy copy, logs     (default ~/.tass)
//	TASS_IDENTITY     where Identity's policy for TASS lives                (default Arameair/identity@main:policies/tass.json)
//	TASS_ENGINE_URL   the Thought Engine                                    (default http://127.0.0.1:7798)
//	TASS_LABEL        issues with this label are messages to TASS           (default tass)
//	TASS_SYNC         GitHub sync interval                                  (default 20s)
//	TASS_ENGINE_KEY   the Thought Engine's api-key file                     (default ~/.thought-engine/tass/api-key)
//	TASS_GITHUB_MESSAGES  "off" to not read GitHub messages (while another program answers them)
//	TASS_TOKEN_FILE   GitHub token file (or GITHUB_TOKEN)                   (default secrets/github_token)
//	TASS_LISTEN       local page address                                    (default 127.0.0.1:8795)
package main

import (
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Arameair/Thought-Processor/core/engine"
	"github.com/Arameair/Thought-Processor/core/github"
	"github.com/Arameair/Thought-Processor/core/tass"
)

//go:embed page.html
var page embed.FS

// version is set at build time: -ldflags "-X main.version=<commit>".
var version = "dev"

func env(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func main() {
	once := flag.Bool("once", false, "one GitHub sync, print the result, exit")
	healthcheck := flag.Bool("healthcheck", false, "ask a running tass-core for /api/health; exit 0 if it answers (container healthcheck)")
	flag.Parse()
	log.SetFlags(log.Ltime)
	if *healthcheck {
		addr := strings.Replace(env("TASS_LISTEN", "127.0.0.1:8795"), "0.0.0.0", "127.0.0.1", 1)
		cl := &http.Client{Timeout: 4 * time.Second}
		resp, err := cl.Get("http://" + addr + "/api/health")
		if err != nil || resp.StatusCode != 200 {
			fmt.Println("not answering:", err)
			os.Exit(1)
		}
		resp.Body.Close()
		return
	}

	home, _ := os.UserHomeDir()
	every, err := time.ParseDuration(env("TASS_SYNC", "20s"))
	if err != nil || every < 5*time.Second {
		log.Fatalf("TASS_SYNC must be a duration of at least 5s (like 20s)")
	}
	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if token == "" {
		tf := env("TASS_TOKEN_FILE", "secrets/github_token")
		b, err := os.ReadFile(tf)
		if err != nil {
			log.Printf("No GitHub token (%s or GITHUB_TOKEN): TASS runs locally only until there is one.", tf)
		}
		token = strings.TrimSpace(string(b))
	}
	cfg := tass.Config{
		DataDir:   env("TASS_DATA", filepath.Join(home, ".tass")),
		Identity:  env("TASS_IDENTITY", "Arameair/identity@main:policies/tass.json"),
		Label:     env("TASS_LABEL", "tass"),
		SyncEvery: every,
		Messages:  !strings.EqualFold(env("TASS_GITHUB_MESSAGES", "on"), "off"),
	}
	gh := github.New(token)
	if b := os.Getenv("TASS_GITHUB_API"); b != "" { // tests and dry runs only
		gh.Base = b
	}
	eng := engine.New(env("TASS_ENGINE_URL", "http://127.0.0.1:7798"))
	keyFile := env("TASS_ENGINE_KEY", filepath.Join(home, ".thought-engine", "tass", "api-key"))
	if b, err := os.ReadFile(keyFile); err == nil {
		eng.Key = strings.TrimSpace(string(b))
	} else {
		log.Printf("No Thought Engine key at %s: start the engine first (it creates the key). TASS can't think until then.", keyFile)
	}
	core, err := tass.New(cfg, gh, eng)
	if err != nil {
		log.Fatal(err)
	}
	if *once {
		core.SyncOnce()
		st := core.Status()
		fmt.Printf("GitHub sync ok: %v  problem: %q\nIdentity policy: %s\nThought Engine reachable: %v\n",
			st.GitHubOnline, st.SyncProblem, st.Policy, st.Engine.Reachable)
		if !st.GitHubOnline {
			os.Exit(1)
		}
		return
	}

	listen := env("TASS_LISTEN", "127.0.0.1:8795")
	ln, err := net.Listen("tcp", listen)
	if err != nil {
		log.Fatalf("can't listen on %s: %v", listen, err)
	}
	// Local API keys: a fresh one for the page each start, and a file for other local programs.
	pageToken := randomKey()
	apiKey, err := loadOrCreateKey(filepath.Join(cfg.DataDir, "api-key"))
	if err != nil {
		log.Fatal(err)
	}
	srv := &http.Server{Handler: handler(core, pageToken, apiKey), ReadHeaderTimeout: 10 * time.Second}
	go srv.Serve(ln)
	log.Printf("TASS core %s: data in %s · Identity policy from %s · engine %s · GitHub sync every %s",
		version, cfg.DataDir, cfg.Identity, eng.URL, every)
	log.Printf("Talk to TASS: http://%s", strings.Replace(listen, "0.0.0.0", "127.0.0.1", 1))

	stop := make(chan struct{})
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() { <-sig; close(stop) }()
	core.Run(stop)
	srv.Close()
	log.Printf("stopped; tass.md and state are on disk in %s", cfg.DataDir)
}

func randomKey() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(b)
}

func loadOrCreateKey(path string) (string, error) {
	if b, err := os.ReadFile(path); err == nil && len(strings.TrimSpace(string(b))) >= 32 {
		return strings.TrimSpace(string(b)), nil
	}
	k := randomKey()
	return k, os.WriteFile(path, []byte(k+"\n"), 0o600)
}

func localHost(r *http.Request) bool {
	h, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		h = r.Host
	}
	return h == "127.0.0.1" || h == "localhost" || h == "::1"
}

func handler(core *tass.Core, pageToken, apiKey string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		if !localHost(r) {
			http.Error(w, "host not allowed", http.StatusForbidden)
			return
		}
		b, _ := page.ReadFile("page.html")
		b = []byte(strings.Replace(string(b), "__PAGE_TOKEN__", pageToken, 1))
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; frame-ancestors 'none'")
		w.Write(b)
	})
	authed := func(r *http.Request) bool {
		got := r.Header.Get("X-TASS-Token")
		if got == "" {
			got = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		}
		return localHost(r) && got != "" && (subtle.ConstantTimeCompare([]byte(got), []byte(pageToken)) == 1 ||
			subtle.ConstantTimeCompare([]byte(got), []byte(apiKey)) == 1)
	}
	// Operational state only (no conversation, no tass.md), for healthchecks and ./tass.sh status.
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		if !localHost(r) {
			http.Error(w, "host not allowed", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(core.Health(version))
	})
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) {
		if !authed(r) { // it holds conversation and tass.md: same key as writing
			http.Error(w, "missing or wrong key", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(core.Status())
	})
	mux.HandleFunc("POST /api/say", func(w http.ResponseWriter, r *http.Request) {
		if !authed(r) {
			http.Error(w, "missing or wrong key (the page's token, or ~/.tass/api-key)", http.StatusUnauthorized)
			return
		}
		// Only this page may post: JSON, and a local Origin (blocks other websites in your browser).
		if o := r.Header.Get("Origin"); o != "" {
			u, err := url.Parse(o)
			if err != nil || (u.Hostname() != "127.0.0.1" && u.Hostname() != "localhost") {
				http.Error(w, "forbidden origin", http.StatusForbidden)
				return
			}
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			http.Error(w, "JSON only", http.StatusUnsupportedMediaType)
			return
		}
		var body struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&body); err != nil {
			http.Error(w, "bad JSON", http.StatusBadRequest)
			return
		}
		if err := core.Say(body.Text); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	})
	return mux
}
