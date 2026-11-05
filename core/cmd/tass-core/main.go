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
	"errors"
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
	"github.com/Arameair/Thought-Processor/core/identitysvc"
	"github.com/Arameair/Thought-Processor/core/tass"
)

//go:embed page.html dev.html cognition.html
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
	tass.BuildVersion = version
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
	if u := os.Getenv("TASS_IDENTITY_URL"); u != "" {
		// One-app mode: the installation's own Identity service decides everything. No
		// built-in owner, no GitHub repository unless the owner turns sync on.
		ids, err := identitysvc.New(u, env("TASS_IDENTITY_KEY_FILE", "/run/tass-secrets/core.key"))
		if err != nil {
			log.Fatal(err)
		}
		cfg.IdentitySvc = ids
		// Chat-only unless the owner turns Autonomous on (page switch, kept in settings.json).
		// Issue #50's checks (same cycle ID, locked prediction with digest) now guard every action
		// either way. TASS_CHAT_ONLY=on forces chat-only whatever the switch says.
		cfg.ChatOnly = strings.EqualFold(os.Getenv("TASS_CHAT_ONLY"), "on")
		if cfg.ChatOnly {
			log.Printf("CHAT_ONLY forced (TASS_CHAT_ONLY=on): no model actions, tasks or work pulse")
		}
		cfg.Identity = "tass-identity at " + u
		// GitHub only when the owner set a repository for THIS installation (nothing is built in).
		cfg.Repo = strings.TrimSpace(os.Getenv("TASS_REPO"))
		cfg.Branch = env("TASS_BRANCH", "live")
		cfg.DocPath = env("TASS_DOC", "tass.md")
		cfg.LessonsDir = env("TASS_LESSONS", "lessons")
		cfg.GitHub = cfg.Repo != ""
		cfg.Messages = false // GitHub issue messages are the prototype's path; app mode talks on the page
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
	gh_ := "GitHub sync every " + every.String()
	if cfg.IdentitySvc != nil {
		gh_ = "no GitHub repository set (local only)"
		if cfg.Repo != "" {
			gh_ = fmt.Sprintf("GitHub %s@%s (%s, lessons in %s/) every %s", cfg.Repo, cfg.Branch, cfg.DocPath, cfg.LessonsDir, every)
		}
	}
	log.Printf("core %s: data in %s · authority %s · engine %s · %s", version, cfg.DataDir, cfg.Identity, eng.URL, gh_)
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
	// Development platform (Arameair/Development ideas/tass-development-platform): same token, same origin.
	mux.HandleFunc("GET /dev", func(w http.ResponseWriter, r *http.Request) {
		if !localHost(r) {
			http.Error(w, "host not allowed", http.StatusForbidden)
			return
		}
		b, _ := page.ReadFile("dev.html")
		b = []byte(strings.Replace(string(b), "__PAGE_TOKEN__", pageToken, 1))
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; frame-ancestors 'none'")
		w.Write(b)
	})
	// Cognition is a read-only, same-origin browser of recorded thoughts/lessons/skills.
	// It reuses the existing authenticated /api/status and /api/brain endpoints.
	mux.HandleFunc("GET /cognition", func(w http.ResponseWriter, r *http.Request) {
		if !localHost(r) {
			http.Error(w, "host not allowed", http.StatusForbidden)
			return
		}
		b, err := page.ReadFile("cognition.html")
		if err != nil {
			http.Error(w, "cognition page unavailable", http.StatusInternalServerError)
			return
		}
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

    // Cognition reads: Core page-token AND current Identity authorization.
    // No browser endpoint mounts Engine/Identity directories or starts work.
    cognitionResponse := func(w http.ResponseWriter, r *http.Request) bool {
        w.Header().Set("Cache-Control","no-store")
        w.Header().Set("X-Content-Type-Options","nosniff")
        if !authed(r) {http.Error(w,"Cognition token required",http.StatusUnauthorized);return false}
        return true
    }
    cognitionErr := func(w http.ResponseWriter,err error) {
        if errors.Is(err,tass.ErrCognitionDenied) {
            http.Error(w,"Identity denied Cognition read",http.StatusForbidden)
        } else {
            http.Error(w,"Cognition evidence NOT RECORDED or temporarily unavailable",http.StatusServiceUnavailable)
        }
    }
    mux.HandleFunc("GET /api/cognition/library",func(w http.ResponseWriter,r *http.Request){
        if !cognitionResponse(w,r){return}
        library,err:=core.CognitionLibrary()
        if err!=nil{cognitionErr(w,err);return}
        w.Header().Set("Content-Type","application/json; charset=utf-8")
        json.NewEncoder(w).Encode(library)
    })
    // #82 P1: the real pulse and Core tick, read-only (one GET to the Engine; never a control call).
    mux.HandleFunc("GET /api/cognition/pulse",func(w http.ResponseWriter,r *http.Request){
        if !cognitionResponse(w,r){return}
        p,err:=core.CognitionPulse()
        if err!=nil{cognitionErr(w,err);return}
        w.Header().Set("Content-Type","application/json; charset=utf-8")
        json.NewEncoder(w).Encode(p)
    })
    mux.HandleFunc("GET /api/cognition/thoughts/{id}",func(w http.ResponseWriter,r *http.Request){
        if !cognitionResponse(w,r){return}
        view,err:=core.CognitionThought(r.PathValue("id"))
        if err!=nil{cognitionErr(w,err);return}
        w.Header().Set("Content-Type","application/json; charset=utf-8")
        json.NewEncoder(w).Encode(view)
    })
    mux.HandleFunc("GET /api/cognition/cycles/{id}/timeline",func(w http.ResponseWriter,r *http.Request){
        if !cognitionResponse(w,r){return}
        view,err:=core.CognitionTimeline(r.PathValue("id"))
        if err!=nil{cognitionErr(w,err);return}
        w.Header().Set("Content-Type","application/json; charset=utf-8")
        json.NewEncoder(w).Encode(view)
    })
    mux.HandleFunc("GET /api/cognition/cycles/{id}/export",func(w http.ResponseWriter,r *http.Request){
        if !cognitionResponse(w,r){return}
        format:=r.URL.Query().Get("format")
        if format!="md"&&format!="json"{
            http.Error(w,"format must be md or json",http.StatusBadRequest);return
        }
        id:=r.PathValue("id")
        if !engine.CognitionID.MatchString(id){
            http.Error(w,"invalid cycle ID",http.StatusBadRequest);return
        }
        raw,media,err:=core.CognitionExport(id,format,version)
        if err!=nil{cognitionErr(w,err);return}
        suffix:=".md"
        if format=="json"{suffix=".json"}
        w.Header().Set("Content-Type",media+"; charset=utf-8")
        w.Header().Set("Content-Disposition","attachment; filename=\"cognition-"+id+suffix+"\"")
        w.Write(raw)
    })

	// pagePost: the key, a local Origin (blocks other websites in your browser) and JSON.
	pagePost := func(w http.ResponseWriter, r *http.Request, into any) bool {
		if !authed(r) {
			http.Error(w, "missing or wrong key (the page's token, or the api-key file)", http.StatusUnauthorized)
			return false
		}
		if o := r.Header.Get("Origin"); o != "" {
			u, err := url.Parse(o)
			if err != nil || (u.Hostname() != "127.0.0.1" && u.Hostname() != "localhost") {
				http.Error(w, "forbidden origin", http.StatusForbidden)
				return false
			}
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			http.Error(w, "JSON only", http.StatusUnsupportedMediaType)
			return false
		}
		limit := int64(16 << 10)
		if strings.HasPrefix(r.URL.Path, "/api/brain/") {
			limit = 512 << 10 // a whole graph
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit)).Decode(into); err != nil {
			http.Error(w, "bad JSON", http.StatusBadRequest)
			return false
		}
		return true
	}
	reply := func(w http.ResponseWriter, code int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(v)
	}
	mux.HandleFunc("POST /api/say", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Text string `json:"text"`
		}
		if !pagePost(w, r, &body) {
			return
		}
		id, err := core.Say(body.Text)
		if errors.Is(err, tass.ErrBusy) { // #46: refused, not silently dropped
			reply(w, http.StatusTooManyRequests, map[string]string{"error": err.Error()})
			return
		}
		if err != nil {
			reply(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		reply(w, http.StatusAccepted, map[string]string{"receipt": id})
	})
	mux.HandleFunc("POST /api/tasks", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Text string `json:"text"`
		}
		if !pagePost(w, r, &body) {
			return
		}
		id, err := core.SubmitTask(body.Text)
		if err != nil {
			reply(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		reply(w, http.StatusAccepted, map[string]string{"job_id": id})
	})
	mux.HandleFunc("POST /api/settings", func(w http.ResponseWriter, r *http.Request) {
		var body tass.Settings
		if !pagePost(w, r, &body) {
			return
		}
		if err := core.SetSettings(body); err != nil {
			reply(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		reply(w, http.StatusOK, core.GetSettings())
	})
	mux.HandleFunc("POST /api/lessons/{action}", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Path string `json:"path"`
		}
		if !pagePost(w, r, &body) {
			return
		}
		var err error
		var receipt string
		switch r.PathValue("action") {
		case "discover": // explicit owner request; no pulse and no autonomous toggle
			err = core.DiscoverLessons()
		case "study": // exactly one selected Markdown lesson, then one pulse step
			receipt, err = core.StudyLesson(body.Path)
		case "approve": // owner's approval is separate from the grading
			err = core.ApproveLesson(body.Path)
		case "retry":
			err = core.RetryLesson(body.Path)
		default:
			reply(w, http.StatusNotFound, map[string]string{"error": "discover, study, approve or retry"})
			return
		}
		if err != nil {
			reply(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		reply(w, http.StatusOK, map[string]any{"learning": core.Learning(), "receipt": receipt})
	})
	// Development workspace (#31): read-only view of an allowlisted project file, and one
	// question to TASS with that file as context.
	mux.HandleFunc("POST /api/workspace/{action}", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Path string `json:"path"`
			Text string `json:"text"`
		}
		if !pagePost(w, r, &body) {
			return
		}
		switch r.PathValue("action") {
		case "file":
			f, err := core.ReadWorkspaceFile(body.Path)
			if err != nil {
				reply(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			reply(w, http.StatusOK, f)
		case "ask":
			id, err := core.SayAboutFile(body.Text, body.Path)
			if errors.Is(err, tass.ErrBusy) {
				reply(w, http.StatusTooManyRequests, map[string]string{"error": err.Error()})
				return
			}
			if err != nil {
				reply(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			reply(w, http.StatusAccepted, map[string]string{"receipt": id})
		default:
			reply(w, http.StatusNotFound, map[string]string{"error": "file or ask"})
		}
	})
	mux.HandleFunc("GET /api/brain", func(w http.ResponseWriter, r *http.Request) {
		if !authed(r) {
			http.Error(w, "missing or wrong key", http.StatusUnauthorized)
			return
		}
		g := core.Brain()
		blocks := core.BrainBlocks(g.Notes)
		reply(w, http.StatusOK, map[string]any{"graph": g, "blocks": blocks, "report": tass.ValidateBrain(g, blocks),
			"history": core.BrainHistory(), "relations": tass.BrainRelations()})
	})
	mux.HandleFunc("POST /api/brain/{action}", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Graph    tass.BrainGraph `json:"graph"`
			Change   string          `json:"change"`
			Revision int             `json:"revision"`
			BasedOn  int             `json:"based_on"`
			Text     string          `json:"text"`
		}
		if !pagePost(w, r, &body) {
			return
		}
		switch r.PathValue("action") {
		case "check":
			reply(w, http.StatusOK, tass.ValidateBrain(body.Graph, core.BrainBlocks(body.Graph.Notes)))
		case "save":
			g, rep, err := core.SaveBrain(body.Graph, "owner", body.Change)
			if errors.Is(err, tass.ErrBrainConflict) {
				reply(w, http.StatusConflict, map[string]any{"error": err.Error(), "graph": g})
				return
			}
			if err != nil {
				reply(w, http.StatusUnprocessableEntity, map[string]any{"error": err.Error(), "report": rep})
				return
			}
			reply(w, http.StatusOK, map[string]any{"graph": g, "report": rep})
		case "restore":
			g, err := core.RestoreBrain(body.Revision, body.BasedOn)
			if err != nil {
				reply(w, http.StatusConflict, map[string]any{"error": err.Error(), "graph": g})
				return
			}
			reply(w, http.StatusOK, map[string]any{"graph": g})
		case "ask":
			id, rev, err := core.AskWithBrain(body.Text)
			if errors.Is(err, tass.ErrBusy) {
				reply(w, http.StatusTooManyRequests, map[string]string{"error": err.Error()})
				return
			}
			if err != nil {
				reply(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			reply(w, http.StatusAccepted, map[string]any{"receipt": id, "revision": rev})
		default:
			reply(w, http.StatusNotFound, map[string]string{"error": "check, save, restore or ask"})
		}
	})
    // #82 P0-A: owner-reviewed link between the Identity entity and Engine history. This is an
    // explicit owner action on Core state; it is not part of playback and touches no Engine data.
    mux.HandleFunc("POST /api/cognition/{action}",func(w http.ResponseWriter,r *http.Request){
        var body struct{ EngineEntityID string `json:"engine_entity_id"` }
        if !pagePost(w,r,&body){return}
        w.Header().Set("Cache-Control","no-store")
        switch r.PathValue("action") {
        case "link":
            l,err:=core.LinkCognition(body.EngineEntityID)
            if err!=nil{reply(w,http.StatusConflict,map[string]string{"error":err.Error()});return}
            reply(w,http.StatusOK,l)
        case "unlink":
            if err:=core.UnlinkCognition();err!=nil{reply(w,http.StatusConflict,map[string]string{"error":err.Error()});return}
            reply(w,http.StatusOK,map[string]string{"ok":"unlinked"})
        default:
            reply(w,http.StatusNotFound,map[string]string{"error":"link or unlink"})
        }
    })
    // #82 P2/P3: junction receipts, alternative cells and comparisons for Development.
    mux.HandleFunc("GET /api/junctions",func(w http.ResponseWriter,r *http.Request){
        if !authed(r){http.Error(w,"missing or wrong key",http.StatusUnauthorized);return}
        w.Header().Set("Cache-Control","no-store")
        reply(w,http.StatusOK,core.Junctions())
    })
    mux.HandleFunc("POST /api/junctions/{action}",func(w http.ResponseWriter,r *http.Request){
        var body struct{
            Path string `json:"path"`
            Text string `json:"text"`
            Note string `json:"note"`
            ProposalID string `json:"proposal_id"`
        }
        if !pagePost(w,r,&body){return}
        switch r.PathValue("action") {
        case "request":
            rc,err:=core.RequestJunction(body.Path,"owner (Development)","","owner requested it in Development")
            if err!=nil{reply(w,http.StatusConflict,map[string]string{"error":err.Error()});return}
            reply(w,http.StatusOK,rc)
        case "propose":
            p,err:=core.ProposeJunctionCell(body.Text,body.Note)
            if err!=nil{reply(w,http.StatusUnprocessableEntity,map[string]string{"error":err.Error()});return}
            reply(w,http.StatusOK,p)
        case "compare":
            cmp,err:=core.CompareJunctionCells(body.ProposalID,body.Path)
            if err!=nil{reply(w,http.StatusConflict,map[string]string{"error":err.Error()});return}
            reply(w,http.StatusOK,cmp)
        default:
            reply(w,http.StatusNotFound,map[string]string{"error":"request, propose or compare"})
        }
    })
	mux.HandleFunc("POST /api/conversation/new", func(w http.ResponseWriter, r *http.Request) {
		var body struct{}
		if !pagePost(w, r, &body) {
			return
		}
		core.NewConversation()
		reply(w, http.StatusOK, map[string]string{"ok": "new conversation"})
	})
	// @ skills (#72): request from Talk; approve/cancel ONLY from this local page (page token).
	mux.HandleFunc("POST /api/skills/{action}", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Receipt string `json:"receipt"`
			Skill   string `json:"skill"`
			Text    string `json:"text"`
		}
		if !pagePost(w, r, &body) {
			return
		}
		var rec tass.SkillReceipt
		var err error
		switch r.PathValue("action") {
		case "request":
			rec, err = core.RequestSkill(body.Skill, firstNonEmptyStr(body.Text, "owner pressed Request on the Skills panel"), "owner (Skills panel)", "you chose it on the Skills panel")
		case "approve":
			rec, err = core.ApproveSkill(body.Receipt)
		case "cancel":
			rec, err = core.CancelSkill(body.Receipt)
		default:
			reply(w, http.StatusNotFound, map[string]string{"error": "request, approve or cancel"})
			return
		}
		if err != nil {
			reply(w, http.StatusConflict, map[string]any{"error": err.Error(), "receipt": rec})
			return
		}
		reply(w, http.StatusOK, map[string]any{"receipt": rec, "skills": core.Skills(true)})
	})
	mux.HandleFunc("POST /api/pulse", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Action string `json:"action"`
		}
		if !pagePost(w, r, &body) {
			return
		}
		out, err := core.PulseControl(body.Action)
		if err != nil {
			reply(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		reply(w, http.StatusOK, out)
	})
	return mux
}

func firstNonEmptyStr(s ...string) string {
	for _, x := range s {
		if strings.TrimSpace(x) != "" {
			return x
		}
	}
	return ""
}
