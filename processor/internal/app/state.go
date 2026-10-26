package app

import (
	"os"
	"sort"
	"time"

	"github.com/Arameair/Thought-Processor/processor/internal/idclient"
	"github.com/Arameair/Thought-Processor/processor/internal/install"
	"github.com/Arameair/Thought-Processor/processor/internal/store"
)

// Card is one feature as the person sees it: what Identity says (eligibility,
// grants) merged with what Processor knows (installed files).
type Card struct {
	Feature      string                `json:"feature"`
	Name         string                `json:"name"`
	Description  string                `json:"description"`
	Visibility   string                `json:"visibility"`
	Eligible     *idclient.Eligibility `json:"eligible_to_install,omitempty"`
	Enabled      string                `json:"enabled_for_identity,omitempty"`
	Sources      []string              `json:"authorized_sources"`
	Registered   string                `json:"service_registered,omitempty"`
	Install      install.View          `json:"install"`
	Installable  bool                  `json:"installable"`
	Action       string                `json:"action"` // install | update | approve | none | sign_in | wait
	Note         string                `json:"note,omitempty"`
	FromIdentity bool                  `json:"from_identity"`
}

// State is everything the page shows. Developer details only in Developer Mode.
func (a *App) State() map[string]any {
	c := a.Conn
	c.mu.Lock()
	conn := map[string]any{"state": c.state, "detail": c.detail, "since": c.since, "candidates": c.candidates,
		"chosen": c.chosen, "code_expires": c.codeExpires, "last_ok": c.lastOK}
	if c.pairing != nil {
		conn["identity_system_id"] = c.pairing.IdentitySystemID
		conn["base_url"] = c.pairing.BaseURL
		conn["paired_at"] = c.pairing.PairedAt
	}
	var gh *idclient.GitHub
	var lifecycle map[string]any
	if c.status != nil {
		g := c.status.GitHub
		gh = &g
		lifecycle = map[string]any{"ownership": c.status.Lifecycle.Ownership, "pid": c.status.Lifecycle.PID, "started_by": c.status.Lifecycle.StartedBy}
		conn["identity_version"] = c.status.Build.Version
	}
	verified := c.state == CVerified
	ids := append([]idclient.Entity(nil), c.identities...)
	fl := c.features
	featuresFor := c.featuresFor
	dataAt, dataErr := c.dataAt, c.dataErr
	status := c.status
	c.mu.Unlock()

	set := a.Prof.Settings()
	out := map[string]any{
		"processor": map[string]any{"version": a.Build.Version, "processor_id": a.Prof.Self.ProcessorID, "pid": os.Getpid(),
			"port": a.Port, "platform": install.Platform(), "token_protection": store.SecretScheme},
		"connection":     conn,
		"verified":       verified,
		"developer_mode": set.DeveloperMode,
		"lifecycle":      lifecycle,
		"exit":           map[string]any{"note": "Exiting stops every feature process this Processor started (their installation identities, grants and data are kept)."},
		"theme":          set.Theme,
	}
	for k, v := range a.p1State() {
		out[k] = v
	}
	if verified {
		out["github"] = gh
		out["identities"] = ids
	}
	sel := set.SelectedIdentity
	if verified && sel != "" && featuresFor == sel {
		for _, e := range ids {
			if e.ID == sel {
				ee := e
				out["selected"] = &ee
			}
		}
	}
	out["cards"], out["catalog_note"] = a.cards(verified, fl, gh), ""
	if verified && fl != nil {
		out["catalog_note"] = fl.CatalogNote
	}
	out["data_at"], out["data_error"] = dataAt, dataErr
	if set.DeveloperMode {
		dev := map[string]any{"build": a.Build, "run_id": a.RunID, "started": a.Started.UTC(), "profile": store.DisplayPath(a.Prof.Dir),
			"log": a.Log.Recent(60), "data_age_s": int(time.Since(dataAt).Seconds())}
		if status != nil {
			dev["identity_status"] = status
		}
		if fl != nil {
			dev["identity_features"] = fl
		}
		dev["installs"] = a.Inst.Views()
		out["dev"] = dev
	}
	return out
}

func (a *App) cards(verified bool, fl *idclient.FeatureList, gh *idclient.GitHub) []Card {
	out := []Card{}
	seen := map[string]bool{}
	if verified && fl != nil {
		for _, f := range fl.Features {
			if f.Feature == "thought-processor" {
				continue // the app you're using
			}
			seen[f.Feature] = true
			e := f.Eligible
			cd := Card{Feature: f.Feature, Name: f.Name, Description: f.Description, Visibility: f.Visibility, Eligible: &e,
				Enabled: f.EnabledForIdentity, Sources: f.AuthorizedSources, Registered: f.ServiceRegistered,
				Install: a.Inst.View(f.Feature), FromIdentity: true}
			if cd.Sources == nil {
				cd.Sources = []string{}
			}
			cd.Installable = e.State == "ALLOWED"
			switch {
			case cd.Install.Busy:
				cd.Action = "wait"
			case cd.Install.Plan != nil:
				cd.Action = "approve"
			case !cd.Installable:
				cd.Action, cd.Note = "none", e.Reason
			case cd.Install.Installed != nil:
				cd.Action = "update"
			default:
				cd.Action = "install"
			}
			out = append(out, cd)
		}
	}
	// Installed features stay visible (as installed) even when Identity is
	// unreachable or no longer lists them; no actions are offered then.
	for _, v := range a.Inst.Views() {
		if seen[v.Feature] || v.Installed == nil {
			continue
		}
		note := "Installed earlier. Identity isn't confirming this feature right now, so nothing can be done with it."
		if !verified {
			note = "Installed earlier. Waiting for Identity."
		}
		out = append(out, Card{Feature: v.Feature, Name: v.Installed.Name, Install: v, Sources: []string{}, Action: "none", Note: note})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
