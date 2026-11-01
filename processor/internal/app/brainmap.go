package app

import (
	"encoding/json"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Brain Map, Screen 1 (#34, D001–D009): a read-only Service Map.
//
// Everything it shows comes from what Processor already knows (/api/state:
// installation records, signed feature status, the Identity pairing, grants,
// request outcomes) plus this run's scrubbed activity log. Two small routes
// are added:
//
//   GET  /api/map/diagnostics  this run's Processor log entries from the last
//                              24 h (already scrubbed when written; scrubbed
//                              again here). Read-only.
//   GET/POST /api/map/layout   where the owner dragged each node. Presentation
//                              data only, in <profile>/ui/map-layout.json.
//                              It never touches installation records, grants,
//                              trust pins, processes or Identity.
//
// No connection can be proposed, authorized or activated from this screen.

const (
	mapLayoutFile  = "map-layout.json"
	mapMaxNodes    = 300
	mapMaxBody     = 32 << 10
	diagWindow     = 24 * time.Hour
	diagMaxEntries = 300
)

var mapKeyRE = regexp.MustCompile(`^[A-Za-z0-9_:./-]{1,120}$`)

type mapPos struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type mapLayout struct {
	Positions map[string]mapPos `json:"positions"`
	SavedAt   time.Time         `json:"saved_at,omitempty"`
}

var mapLayoutMu sync.Mutex

func (a *App) mapLayoutPath() string { return filepath.Join(a.Prof.Dir, "ui", mapLayoutFile) }

func (a *App) loadMapLayout() mapLayout {
	l := mapLayout{Positions: map[string]mapPos{}}
	b, err := os.ReadFile(a.mapLayoutPath())
	if err != nil {
		return l
	}
	if json.Unmarshal(b, &l) != nil || l.Positions == nil {
		return mapLayout{Positions: map[string]mapPos{}}
	}
	return l
}

func validMapLayout(l *mapLayout) bool {
	if l.Positions == nil || len(l.Positions) > mapMaxNodes {
		return false
	}
	for k, p := range l.Positions {
		if !mapKeyRE.MatchString(k) || math.IsNaN(p.X) || math.IsNaN(p.Y) || math.IsInf(p.X, 0) || math.IsInf(p.Y, 0) ||
			math.Abs(p.X) > 1e6 || math.Abs(p.Y) > 1e6 {
			return false
		}
	}
	return true
}

func (a *App) mapRoutes(m *http.ServeMux) {
	m.HandleFunc("GET /api/map/layout", a.wrap(private, func(w http.ResponseWriter, r *http.Request) {
		mapLayoutMu.Lock()
		l := a.loadMapLayout()
		mapLayoutMu.Unlock()
		writeJSON(w, 200, l)
	}))
	m.HandleFunc("POST /api/map/layout", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		var l mapLayout
		r.Body = http.MaxBytesReader(w, r.Body, mapMaxBody)
		if err := decode(r, &l); err != nil || !validMapLayout(&l) {
			a.fail(w, 400, "a layout is up to 300 node positions with plain ids and finite numbers; nothing was saved")
			return
		}
		l.SavedAt = time.Now().UTC()
		b, _ := json.MarshalIndent(l, "", " ")
		mapLayoutMu.Lock()
		defer mapLayoutMu.Unlock()
		if err := os.MkdirAll(filepath.Dir(a.mapLayoutPath()), 0o700); err != nil {
			a.fail(w, 500, "couldn't save the layout")
			return
		}
		if err := writeFileAtomic(a.mapLayoutPath(), b); err != nil {
			a.fail(w, 500, "couldn't save the layout")
			return
		}
		writeJSON(w, 200, map[string]any{"saved": len(l.Positions), "saved_at": l.SavedAt,
			"note": "positions only: no service, grant, pin or process was changed"})
	}))
	m.HandleFunc("GET /api/map/diagnostics", a.wrap(private, func(w http.ResponseWriter, r *http.Request) {
		cut := time.Now().Add(-diagWindow)
		out := []Entry{}
		for _, e := range a.Log.Recent(diagMaxEntries) {
			if e.Time.After(cut) {
				e.Detail = scrub(e.Detail)
				out = append(out, e)
			}
		}
		writeJSON(w, 200, map[string]any{
			"source":    "Thought Processor's own activity log for this run (scrubbed before it is written)",
			"window":    "last 24 hours, up to the last 300 entries in memory",
			"durable":   false,
			"note":      "Not yet built (Brain Map phase 2): durable 24-hour history across restarts, other services' telemetry, and correlation IDs. Missing telemetry is shown as missing.",
			"timezone":  "UTC",
			"entries":   out,
			"processor": strings.TrimSpace(a.Prof.Self.ProcessorID),
		})
	}))
}

// writeFileAtomic: temp file in the same folder, fsync, rename (0600).
func writeFileAtomic(path string, b []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".layout-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
