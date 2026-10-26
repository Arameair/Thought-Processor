package main

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var pairCodeRE = regexp.MustCompile(`^[A-Za-z0-9]{4}-?[A-Za-z0-9]{4}$`)

func (a *App) pairingView() map[string]any {
	a.mu.Lock()
	defer a.mu.Unlock()
	v := map[string]any{"paired": a.pairing != nil, "processor_id": a.procID, "error": a.pairErr}
	if a.pairing != nil {
		v["identity_address"] = a.pairing.IdentityAddress
		v["identity_system_id"] = a.pairing.IdentitySystemID
		v["paired_at"] = a.pairing.PairedAt
	}
	return v
}

func (a *App) currentPairing() (Pairing, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.pairing == nil || a.pairing.IdentityAddress != a.cands.Identity {
		return Pairing{}, false
	}
	return *a.pairing, true
}

func noParams(w http.ResponseWriter, r *http.Request) bool {
	var in struct{}
	if err := readStrict(r, &in); err != nil {
		writeJSON(w, 400, errBody("bad_request", "this action takes no parameters"))
		return false
	}
	return true
}

func (a *App) installRoutes(m *http.ServeMux) {
	m.HandleFunc("POST /api/identity/pair", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Code string `json:"code"`
		}
		if err := readStrict(r, &in); err != nil || !pairCodeRE.MatchString(strings.TrimSpace(in.Code)) {
			writeJSON(w, 400, errBody("bad_code", "Type the 8-character code Identity shows (like ABCD-EFGH)."))
			return
		}
		a.mu.Lock()
		base := a.cands.Identity
		a.mu.Unlock()
		if base == "" {
			writeJSON(w, 409, errBody("no_identity", "Enter Identity's address first."))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		pr, err := IdentityPair(ctx, base, strings.TrimSpace(in.Code), a.procID, fmt.Sprintf("Thought Processor (port %d)", a.port))
		if err != nil {
			code, msg, next := problemFor(err)
			a.log.Add("pairing", "pair_refused", "", "pairing with Identity failed ("+code+")")
			writeJSON(w, 400, map[string]any{"error": strings.ToLower(code), "message": msg, "next_step": next})
			return
		}
		if err := a.prof.SavePairing(pr); err != nil {
			writeJSON(w, 500, errBody("save_failed", "Paired, but the pairing could not be saved to the profile; pair again."))
			return
		}
		a.mu.Lock()
		a.pairing, a.pairErr = &pr, ""
		a.mu.Unlock()
		a.log.Add("pairing", "paired", "", "paired with Identity "+pr.IdentitySystemID+" at "+strings.TrimPrefix(base, "http://")+" (token stored in the profile, never shown)")
		// Ask right away so the person sees the answer.
		cctx, ccancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer ccancel()
		a.inst.Check(cctx, pr)
		writeJSON(w, 200, a.State())
	}))
	m.HandleFunc("POST /api/identity/unpair", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		if !noParams(w, r) {
			return
		}
		if err := a.prof.RemovePairing(); err != nil {
			writeJSON(w, 500, errBody("remove_failed", "Could not remove the saved pairing."))
			return
		}
		a.mu.Lock()
		a.pairing, a.pairErr = nil, ""
		a.mu.Unlock()
		a.log.Add("pairing", "unpaired", "", "pairing with Identity removed from this Processor (also unpair it in Identity to revoke the token there)")
		writeJSON(w, 200, a.State())
	}))
	m.HandleFunc("POST /api/te/check", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		if !noParams(w, r) {
			return
		}
		pr, ok := a.currentPairing()
		if !ok {
			writeJSON(w, 409, errBody("not_paired", "Pair with Identity first."))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		d, err := a.inst.Check(ctx, pr)
		if err != nil {
			a.log.Add("install", "access_check_failed", "", "access check failed before an answer")
		} else {
			a.log.Add("install", "access_"+strings.ToLower(d.State), "", "Identity decision "+d.DecisionID+": "+d.State+" (rule "+d.Rule+")")
		}
		writeJSON(w, 200, a.State())
	}))
	m.HandleFunc("POST /api/te/install", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		if !noParams(w, r) {
			return
		}
		pr, ok := a.currentPairing()
		if !ok {
			writeJSON(w, 409, errBody("not_paired", "Pair with Identity first."))
			return
		}
		a.log.Add("install", "install_requested", "", "operator asked to install Thought Engine")
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
			defer cancel()
			a.inst.Install(ctx, pr, func(code, detail string) { a.log.Add("install", code, "", detail) })
		}()
		writeJSON(w, 202, a.State())
	}))
	m.HandleFunc("POST /api/te/verify", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		if !noParams(w, r) {
			return
		}
		ok, err := a.inst.Verify()
		if err != nil {
			writeJSON(w, 409, errBody("nothing_installed", "Nothing is installed."))
			return
		}
		a.log.Add("install", map[bool]string{true: "verify_ok", false: "verify_files_changed"}[ok], "", "installed files re-hashed")
		writeJSON(w, 200, a.State())
	}))
	m.HandleFunc("POST /api/te/uninstall", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		if !noParams(w, r) {
			return
		}
		if err := a.inst.Uninstall(); err != nil {
			writeJSON(w, 409, errBody("uninstall_failed", err.Error()))
			return
		}
		a.log.Add("install", "uninstalled", "", "installed Thought Engine files removed from the profile (Engine data, if any, untouched)")
		writeJSON(w, 200, a.State())
	}))
}
