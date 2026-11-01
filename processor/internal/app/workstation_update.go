package app

// This is the owner-triggered WSL workbench update bridge. It does not
// implement a second updater: the existing launcher enforces branch,
// clean checkout, fast-forward, pinned preflight, restart and rollback.
//
// The browser must have a live signed-in Processor session, current signed
// Identity/GitHub VERIFIED state and CSRF, and the owner must confirm Update.
// No secret-bearing logs, commands, or arbitrary paths are accepted.

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"
)

const wsTrialBranch = "experiment/brain-workstation-controls-20261031"

func (a *App) wsUpdateLauncher() (string, string, error) {
    rp, ok := a.WS.repo("processor")
    if !ok { return "", "", errors.New("processor checkout is not available") }
    if b := gitOut(rp.Dir, "branch", "--show-current"); b != wsTrialBranch {
        return "", "", fmt.Errorf("wrong branch: requires %s", wsTrialBranch)
    }
    script := filepath.Join(rp.Dir, "tools", "p0-launcher", "secure_launch.py")
    real, err := filepath.EvalSymlinks(script)
    if err != nil { return "", "", errors.New("update launcher not available") }
    if !strings.HasPrefix(real, rp.Dir + string(os.PathSeparator)) {
        return "", "", errors.New("launcher outside allowed checkout")
    }
    st, err := os.Stat(real)
    if err != nil || !st.Mode().IsRegular() { return "", "", errors.New("update launcher invalid") }
    return rp.Dir, real, nil
}

// The first trial is explicitly restricted to the owner's verified GitHub
// identity. This is NOT an arbitrary browser user's feature-update grant.
func wsUpdateOwner(g wsGate) bool {
    return g.OK && g.GitHub == "VERIFIED" && g.GitHubMode == "real" &&
        strings.EqualFold(g.Login, "Arameair")
}

func (a *App) wsLastUpdate() map[string]any {
    path := filepath.Join(filepath.Dir(a.Prof.Dir), "launcher-control", "update-history.jsonl")
    f, err := os.Open(path)
    if err != nil { return nil }
    defer f.Close()
    st, err := f.Stat()
    if err != nil || !st.Mode().IsRegular() { return nil }
    // Cap at the end of the file; it contains version receipts, not secrets.
    start := st.Size()-65536
    if start < 0 { start = 0 }
    if _, err := f.Seek(start, 0); err != nil { return nil }
    buf := make([]byte, st.Size()-start)
    n, err := f.Read(buf)
    if err != nil && n == 0 { return nil }
    lines := strings.Split(strings.TrimSpace(string(buf[:n])), "\n")
    for i:=len(lines)-1; i>=0; i-- {
        var raw map[string]any
        if json.Unmarshal([]byte(lines[i]), &raw) == nil {
            // Return only fixed known status fields; never return raw logs.
            out := make(map[string]any)
            for _, key := range []string{"at","branch","result","previous","available","now_running","reason","launcher"} {
                if x, ok := raw[key]; ok { out[key] = scrub(fmt.Sprint(x)) }
            }
            return out
        }
    }
    return nil
}

func (a *App) wsUpdateCheck(ctx context.Context) (map[string]any, error) {
    dir, script, err := a.wsUpdateLauncher()
    if err != nil { return nil, err }
    ctx, cancel := context.WithTimeout(ctx, 40*time.Second)
    defer cancel()
    cmd := exec.CommandContext(ctx, "python3", script, "check")
    cmd.Dir = dir
    cmd.Env = wsEnv()
    cmd.Env = append(cmd.Env, "GIT_TERMINAL_PROMPT=0")
    b, err := cmd.Output()
    if err != nil { return nil, errors.New("update check could not complete") }
    var data map[string]any
    if json.Unmarshal(b, &data) != nil { return nil, errors.New("update checker returned no usable result") }
    return data, nil
}

func (a *App) workstationUpdateRoutes(m *http.ServeMux,
    gated func(level, func(http.ResponseWriter, *http.Request, wsGate)) http.HandlerFunc) {
    m.HandleFunc("GET /api/ws/update", gated(private, func(w http.ResponseWriter, r *http.Request, g wsGate) {
        out := map[string]any{"owner_can_apply": wsUpdateOwner(g), "last_update": a.wsLastUpdate()}
        _, _, err := a.wsUpdateLauncher()
        if err != nil {
            out["check"] = "UNAVAILABLE"
            out["detail"] = err.Error()
            writeJSON(w, 200, out)
            return
        }
        out["branch"] = wsTrialBranch
        out["check"] = "NOT_RUN"
        // Only perform remote network checking when explicitly requested.
        if r.URL.Query().Get("check") == "1" {
            version, err := a.wsUpdateCheck(r.Context())
            if err != nil { out["check"]="UNAVAILABLE"; out["detail"]=err.Error() } else {
                out["check"]="COMPLETED";out["versions"]=version
            }
        }
        writeJSON(w, 200, out)
    }))
    m.HandleFunc("POST /api/ws/update/apply", gated(mutate, func(w http.ResponseWriter, r *http.Request, g wsGate) {
        if !wsUpdateOwner(g) {
            a.fail(w, 403, "only the owner with current Identity-verified real GitHub sign-in can update this experimental Workbench")
            return
        }
        var req struct { Confirm string `json:"confirm"` }
        if err := decode(r, &req); err != nil || req.Confirm != "UPDATE" {
            a.fail(w, 400, "explicit Update confirmation required")
            return
        }
        a.WS.mu.Lock()
        runningTest := a.WS.running
        a.WS.mu.Unlock()
        if runningTest { a.fail(w, 409, "a Workstation test is running; finish the test before updating"); return }
        dir, script, err := a.wsUpdateLauncher()
        if err != nil { a.fail(w, 409, err.Error());return }
        // Detach from the Processor's process group. The launcher stops
        // and restarts Processor during Apply; the updater must survive that.
        if err := wsLaunchDetachedUpdater(dir, script); err != nil {
            a.fail(w, 503, "could not start the trusted Workbench updater")
            return
        }
        a.Log.Info("workstation", "update.requested", "Owner requested guarded launcher update; outcome pending")
        writeJSON(w, 202, map[string]any{
            "state":"STARTED_NOT_FINISHED",
            "detail":"The launcher is checking, updating and restarting if eligible. This page may disconnect. Use launcher Open after the restart; inspect Updates for UPDATED, UP_TO_DATE, REFUSED or ROLLED_BACK.",
        })
    }))
}
