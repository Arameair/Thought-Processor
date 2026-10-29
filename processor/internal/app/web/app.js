"use strict";
(() => {
  const $ = (q) => document.querySelector(q);
  const esc = (s) => String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
  const set = (el, h) => { if (el && el.dataset.html !== h) { el.innerHTML = h; el.dataset.html = h; } };
  const short = (s, n = 12) => (s && s.length > n + 4 ? s.slice(0, n) + "…" : s || "");
  const ago = (t) => { if (!t) return ""; const d = (Date.now() - new Date(t)) / 1000; return d < 60 ? Math.round(d) + "s ago" : d < 3600 ? Math.round(d / 60) + " min ago" : new Date(t).toLocaleString(); };
  let csrf = "", state = null, timer = null, exiting = false;
  let sel = JSON.parse(sessionStorage.getItem("tp_sel") || "null"); // {type:'admin'|'entity'|'feature'|'library', id}
  let tab = sessionStorage.getItem("tp_tab") || "overview";
  let approvalWatch = null; // {id, url, kind, summary}
  let lastAction = {};      // installation -> result
  let details = {};         // installation -> /details answer
  let activity = {};        // installation -> copied activity text
  let reviewFor = "";

  function choose(s) { sel = s; sessionStorage.setItem("tp_sel", JSON.stringify(s)); render(); if (s && s.type === "entity") act("/api/identities/select", { id: s.id }); }

  async function api(method, path, body) {
    const opt = { method, headers: {} };
    if (method === "POST") { opt.headers["Content-Type"] = "application/json"; opt.headers["X-TP-CSRF"] = csrf; opt.body = JSON.stringify(body || {}); }
    const r = await fetch(path, opt);
    let j = {};
    try { j = await r.json(); } catch (_) {}
    if (r.status === 401 && path.startsWith("/api/state")) { showGate(j.error); throw new Error(j.error || "not signed in"); }
    if (!r.ok) throw new Error(j.error || ("HTTP " + r.status));
    return j;
  }
  function showGate(msg) { $("#shell").hidden = true; $("#gate").hidden = false; if (msg) $("#gate-msg").textContent = msg; }
  async function signIn() {
    const m = location.hash.match(/code=([0-9a-f]+)/);
    if (m) {
      try {
        const r = await fetch("/api/session", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ code: m[1] }) });
        const j = await r.json();
        if (r.ok) { csrf = j.csrf; sessionStorage.setItem("tp_csrf", csrf); } else { showGate(j.error); return false; }
      } catch (_) { showGate(); return false; }
      history.replaceState(null, "", "/");
    } else csrf = sessionStorage.getItem("tp_csrf") || "";
    return true;
  }
  async function refresh() {
    if (exiting) return;
    try { state = await api("GET", "/api/state"); } catch (_) { return; }
    $("#gate").hidden = true; $("#shell").hidden = false;
    render();
  }
  function schedule() {
    clearTimeout(timer);
    const busy = state && ((state.cards || []).some((c) => c.install && c.install.busy) || approvalWatch);
    timer = setTimeout(async () => { await refresh(); schedule(); }, busy ? 900 : 2000);
  }
  async function act(path, body, errEl) {
    try {
      const j = await api("POST", path, body);
      if (j && j.connection) state = j; else if (j && j.state) state = j.state;
      if (errEl) errEl.hidden = true;
      render();
      return j || true;
    } catch (e) {
      if (errEl) { errEl.textContent = e.message; errEl.hidden = false; } else toast(e.message, "bad");
      return null;
    }
  }
  function toast(msg, cls) {
    let t = $("#toast");
    if (!t) { t = document.createElement("div"); t.id = "toast"; document.body.append(t); }
    t.className = "toast " + (cls || ""); t.textContent = msg; t.hidden = false;
    clearTimeout(t._h); t._h = setTimeout(() => (t.hidden = true), 6000);
  }

  // ---------- theme ----------
  function applyTheme(th) {
    const v = th || "dark";
    if (v === "system") document.documentElement.removeAttribute("data-theme"); else document.documentElement.dataset.theme = v;
    $("#theme").value = v;
  }

  // ---------- render ----------
  const connWords = {
    CONNECTED: ["ok", "Identity: connected"], AUTHORITY_UNAVAILABLE: ["warn", "Identity: AUTHORITY UNAVAILABLE"],
    CONNECTION_REMOVED: ["bad", "Identity: connection removed"], WRONG_SERVICE: ["bad", "Identity: not the one you connected"],
    INCOMPATIBLE: ["bad", "Identity: wrong version"], PAIRING_UNREADABLE: ["bad", "Identity: saved connection unreadable"],
    WAITING_FOR_CODE: ["", "Identity: waiting for code"], FOUND_NOT_CONNECTED: ["", "Identity: not connected"], IDENTITY_NOT_FOUND: ["", "Identity: not running"],
  };
  function render() {
    const s = state; if (!s) return;
    const c = s.connection;
    applyTheme(s.theme);
    const [cls, words] = connWords[c.state] || ["", c.state];
    const chip = $("#chip-identity"); chip.className = "chip " + cls; chip.querySelector(".t").textContent = words; chip.title = c.detail || "";
    $("#devmode").checked = !!s.developer_mode;
    const paired = !!c.identity_system_id && !["CONNECTION_REMOVED", "WRONG_SERVICE", "PAIRING_UNREADABLE", "INCOMPATIBLE"].includes(c.state);
    const showHome = s.verified || (paired && c.state === "AUTHORITY_UNAVAILABLE");
    $("#connect").hidden = showHome; $("#home").hidden = !showHome;
    const ban = $("#banner");
    if (c.state === "AUTHORITY_UNAVAILABLE" && paired) {
      ban.className = "banner"; ban.hidden = false;
      ban.textContent = "AUTHORITY UNAVAILABLE — Identity can't be reached. Nothing that needs Identity's permission can happen (features refuse protected work too). Running features keep their standalone health. Processor reconnects by itself.";
    } else if (s.test_feature_faults) {
      ban.className = "banner"; ban.hidden = false;
      ban.textContent = "TEST MODE — started with -test-feature-faults: test features may obey their fault-injection file (crash/delay after an allowed request). Never use this for normal running.";
    } else ban.hidden = true;
    renderChips(s);
    if (showHome) renderHome(s); else renderConnect(s);
    renderDev(s);
    renderModals(s);
  }

  function renderChips(s) {
    const g = s.github, gc = $("#chip-github");
    gc.hidden = !s.verified || !g;
    if (g) {
      gc.className = "chip link " + (g.state === "VERIFIED" ? "ok" : g.state === "WAITING" ? "warn" : "");
      gc.querySelector(".t").textContent = g.state === "VERIFIED" ? "GitHub: " + (g.account ? g.account.login : "signed in") : g.state === "WAITING" ? "GitHub: waiting for you" : "Sign in to GitHub";
    }
    const a = s.admin, ac = $("#chip-admin");
    ac.hidden = !a;
    if (a) {
      ac.className = "chip link " + (a.state === "ENROLLED" ? "ok" : a.state === "DAMAGED" ? "bad" : "warn");
      ac.querySelector(".t").textContent = a.state === "ENROLLED" ? "Admin: established" : a.state === "DAMAGED" ? "Admin: record damaged" : "Admin: not established";
    }
  }

  // ----- connect (unchanged P0 flow) -----
  function renderConnect(s) {
    const c = s.connection, body = $("#c-body");
    let title = "Connect to Identity", lead = "", h = "";
    switch (c.state) {
      case "IDENTITY_NOT_FOUND":
        title = "Start Identity"; lead = "Thought Processor works together with Identity, a separate program on this computer. Start Identity — Processor will find it by itself.";
        h = `<p class="muted small">Looking for Identity every few seconds…</p>` + advancedAddress(); break;
      case "FOUND_NOT_CONNECTED": {
        lead = "Identity is running. Connect Processor to it once; after that they reconnect by themselves.";
        const cands = c.candidates || [];
        h = cands.map((x) => `<div class="cand"><div><strong>Identity ${esc(x.version)}</strong><div class="muted small">${esc(x.base_url)}${x.profile ? " · " + esc(x.profile) : ""}${x.compatible ? "" : " · <span class='error'>version not supported</span>"}</div></div>
          <button class="btn primary" data-connect="${esc(x.base_url)}" ${x.compatible ? "" : "disabled"}>Connect</button></div>`).join("");
        if (cands.length > 1) h = `<p class="small">More than one Identity is running on this computer. Choose the one you use:</p>` + h;
        h += advancedAddress(); break;
      }
      case "WAITING_FOR_CODE":
        title = "Type the code from Identity"; lead = "Look at the Identity window. It's showing an 8-character code in a box marked PAIRING REQUEST. Type it here.";
        h = `<form id="code-form" class="row"><input id="code-in" class="codein" maxlength="9" placeholder="XXXX-XXXX" autocomplete="off" required><button class="btn primary" type="submit">Connect</button></form>
             <p class="muted small">The code works once and expires in 5 minutes. <button class="btn ghost small" type="button" data-newcode="1">Show a new code</button></p>`; break;
      case "CONNECTION_REMOVED": title = "Identity no longer recognises this Processor"; lead = (c.detail || "") + " You'll need to connect again.";
        h = `<div class="row"><button class="btn primary" data-reconnect="1">Connect again</button></div>`; break;
      case "WRONG_SERVICE": title = "A different Identity is answering";
        lead = "The program answering isn't the Identity this Processor connected to (" + (c.identity_system_id || "") + "), so nothing it says is trusted. If you changed Identity on purpose, connect again.";
        h = `<div class="row"><button class="btn primary" data-reconnect="1">Connect again</button></div>`; break;
      case "INCOMPATIBLE": case "PAIRING_UNREADABLE":
        title = c.state === "INCOMPATIBLE" ? "Identity version doesn't match" : "The saved connection can't be read"; lead = c.detail || "";
        h = `<div class="row"><button class="btn primary" data-reconnect="1">Connect again</button></div>`; break;
      default: title = "Checking Identity…"; lead = c.detail || "";
    }
    $("#c-title").textContent = title; $("#c-lead").textContent = lead;
    if (body.dataset.html !== h) {
      body.innerHTML = h; body.dataset.html = h;
      const f = $("#code-form");
      if (f) { $("#code-in").focus(); f.onsubmit = (e) => { e.preventDefault(); act("/api/connect/code", { code: $("#code-in").value.trim() }, $("#c-error")); }; }
      const af = $("#addr-form");
      if (af) af.onsubmit = (e) => { e.preventDefault(); act("/api/connect/choose", { url: $("#addr-in").value.trim() }, $("#c-error")); };
    }
  }
  function advancedAddress() {
    return `<details class="small advanced"><summary class="muted">Identity uses a different address</summary>
      <form id="addr-form" class="row"><input id="addr-in" placeholder="http://127.0.0.1:7790" autocomplete="off"><button class="btn" type="submit">Look there</button></form></details>`;
  }

  // ----- workspace -----
  const stateDot = { RUNNING: "ok", STARTING: "warn", STOPPING: "warn", RESTARTING: "warn", STOPPED: "idle", CRASHED: "bad", NEEDS_ATTENTION: "bad", FAILED_TO_START: "bad" };
  function features(s) { return s.installations || []; }
  function findFeature(s, id) { return features(s).find((v) => v.record.installation_id === id); }

  function renderHome(s) {
    const ids = s.identities || [], a = s.admin;
    const item = (key, label, kind, extra, dot) => {
      const on = sel && sel.type + ":" + sel.id === key;
      return `<button type="button" class="idrow ${on ? "sel" : ""}" data-sel="${esc(key)}"><span class="dot ${dot || ""}"></span><span class="nm">${esc(label)}</span><span class="kind">${esc(kind)}</span>${extra || ""}</button>`;
    };
    let h = `<div class="grp"><div class="grp-h">Administrator</div>`;
    h += a ? item("admin:admin", a.state === "ENROLLED" ? "Admin Identity" : "Establish Admin…", "admin", "", a.state === "ENROLLED" ? "ok" : "warn") : `<div class="muted small pad">Waiting for Identity…</div>`;
    h += `</div><div class="grp"><div class="grp-h">Companions & people${s.identities_stale ? ` <span class="warn">· last known</span>` : ""}</div>`;
    h += ids.map((e) => item("entity:" + e.id, e.name, e.kind)).join("") || `<div class="muted small pad">None yet — add one below.</div>`;
    h += `</div><div class="grp"><div class="grp-h">Installed features</div>`;
    const fs = features(s).filter((v) => v.record.state === "INSTALLED");
    h += fs.map((v) => item("feature:" + v.record.installation_id, v.record.name.replace(" (TEST FIXTURE)", ""), "feature", `<span class="sub">${esc(short(v.record.installation_id, 10))}</span>`, stateDot[v.runtime.state] || "idle")).join("")
      || `<div class="muted small pad">No features installed.</div>`;
    const tomb = features(s).filter((v) => v.record.state !== "INSTALLED");
    if (tomb.length) h += tomb.map((v) => item("feature:" + v.record.installation_id, v.record.name.replace(" (TEST FIXTURE)", ""), "uninstalled", `<span class="sub">${esc(short(v.record.installation_id, 10))}</span>`, "gone")).join("");
    const txs = s.install_txs || [];
    if (txs.length) h += txs.map((t) => `<div class="txrow"><span class="dot warn"></span><span>${esc(t.name)} — ${esc(t.state === "WAITING_FOR_ADMIN" ? "waiting for Admin" : t.state.toLowerCase())}</span></div>`).join("");
    h += `</div><div class="grp"><div class="grp-h">Library</div>` + item("library:all", "Add a feature…", "library") + `</div>`;
    set($("#id-groups"), h);
    ["#add-id-btn", "#add-feature-btn"].forEach((b) => ($(b).disabled = !s.verified));
    set($("#trusted-note"), (s.trusted_dirs || []).length ? "Trusted local feature folders: " + s.trusted_dirs.map(esc).join(", ") : "No trusted feature folder (start Processor with -trust-feature-dir).");

    if (!sel || (sel.type === "entity" && !ids.find((e) => e.id === sel.id)) || (sel.type === "feature" && !findFeature(s, sel.id))) sel = a && a.state !== "ENROLLED" ? { type: "admin", id: "admin" } : null;
    const d = $("#detail");
    if (!sel) return set(d, `<div class="empty"><h2>Choose an identity</h2><p>The Admin, your companions and every installed feature are identities here. Each one starts with no access to anything.</p></div>`);
    if (sel.type === "admin") return renderAdmin(s, d);
    if (sel.type === "entity") return renderEntity(s, d, ids.find((e) => e.id === sel.id));
    if (sel.type === "feature") return renderFeature(s, d, findFeature(s, sel.id));
    if (sel.type === "library") return renderLibrary(s, d);
  }

  function kv(rows) { return `<dl class="kv">${rows.filter(Boolean).map(([k, v]) => `<dt>${esc(k)}</dt><dd>${v}</dd>`).join("")}</dl>`; }
  const mono = (s) => `<code>${esc(s)}</code>`;
  const badge = (t, c, title) => `<span class="badge ${c || ""}" ${title ? `title="${esc(title)}"` : ""}>${esc(t)}</span>`;

  function renderAdmin(s, d) {
    const a = s.admin || {}, g = s.github || {};
    let h = `<div class="dhead"><div class="typed admin">ADMIN IDENTITY</div><h1>Admin Identity</h1>`;
    if (a.state === "ENROLLED") {
      const ad = a.admin, cur = a.current_github || {};
      h += `<div class="badges">${badge("Established", "ok")}${badge("Local authority — works offline", "info")}${badge(ad.github_binding.mode === "real" ? "GitHub verified" : "GitHub: TEST FIXTURE", ad.github_binding.mode === "real" ? "ok" : "warn")}</div></div>`;
      h += kv([["Admin ID", mono(ad.admin_id)], ["Established", esc(new Date(ad.established_at).toLocaleString())],
        ["Bound GitHub account", esc(ad.github_binding.login_label) + " · numeric id " + mono(ad.github_binding.id) + ` <span class="muted small">(sign-in alias, not the authority)</span>`],
        ["Approves with", esc(ad.authenticator) + ` <span class="muted small">— typed only in Identity's own page</span>`],
        ["Signed-in GitHub now", cur.login ? esc(cur.login) + " (id " + esc(cur.id) + ") " + (cur.is_admin_binding ? badge("this is the Admin's account", "ok") : badge("NOT the Admin's account — no Admin authority", "bad")) : esc(cur.state || "signed out") + ` <span class="muted small">(not needed for local approvals)</span>`],
        ["Recovery", `<span class="warn">${esc(a.recovery || "")}</span>`]]);
      h += `<h3>What needs a fresh Admin approval</h3><ul class="plain"><li>Installing a feature (registers its installation identity)</li><li>Granting a capability / connection</li><li>Uninstalling a feature</li></ul>
        <p class="muted small">Routine actions — viewing, starting or stopping an installed feature, revoking a grant — use this Processor's paired session. Pairing Processor never makes it an Admin.</p>`;
      h += `<h3>Authenticators</h3>` + kv((a.authenticator_options || []).map((o) => [o.kind, badge(o.status, o.status === "AVAILABLE" ? "ok" : "bad") + ` <span class="muted small">${esc(o.note)}</span>`]));
    } else if (a.state === "DAMAGED") {
      h += `</div><div class="problem">${esc(a.error || "The Admin record is damaged.")} Every Admin-approved action is blocked. It was not treated as missing, so nobody can enroll over it.</div>`;
    } else {
      const ok = g.state === "VERIFIED";
      const pend = (s.admin_pending || [])[0];
      h += `<div class="badges">${badge("Not established yet", "warn")}</div></div>
        <p>The Admin Identity is this installation's original administrator. It is created <strong>once</strong>, right after a verified GitHub sign-in, with your explicit confirmation. It is a <strong>local</strong> authority: switching GitHub accounts later never transfers it.</p>
        <ol class="steps2"><li class="${ok ? "done" : "now"}"><strong>Sign in to GitHub</strong> ${ok ? "— " + badge("verified: " + (g.account ? g.account.login : ""), "ok") : `<button class="btn primary small" data-gh="1">Sign in</button>`}</li>
        <li class="${ok ? "now" : ""}"><strong>Establish the Admin Identity in Identity</strong> — you choose an Admin passphrase there (Linux/WSL authenticator). ${ok ? `<button class="btn primary small" data-enroll="1">Establish Admin Identity…</button>` : ""}</li></ol>
        ${pend ? `<p class="small muted">A request is waiting: ${esc(pend.approval_id)}. <button class="btn ghost small" data-watch="${esc(pend.approval_id)}">Open it</button></p>` : ""}
        <p class="warn small">P1 test profiles: recovery is deferred. If the passphrase is lost the profile's Admin can't be recovered — no fallback owner is ever created.</p>`;
    }
    set(d, h);
  }

  function renderEntity(s, d, e) {
    const conns = [];
    for (const v of (s.identity_installations || [])) for (const g of v.grants || []) if (g.grant.subject_identity_id === e.id) conns.push([v.installation, g]);
    let h = `<div class="dhead"><div class="typed">${esc(e.kind.toUpperCase())} IDENTITY</div><h1>${esc(e.name)}</h1><div class="badges">${badge("No information sources", "")}${badge(conns.filter((x) => x[1].status === "ACTIVE").length + " active connection(s) as subject", "")}</div></div>`;
    h += kv([["Identity ID", mono(e.id)], ["Kind", esc(e.kind)], ["Created", esc(new Date(e.created_at).toLocaleString())]]);
    h += `<h3>Connections that act for ${esc(e.name)}</h3>`;
    h += conns.length ? `<table class="tbl"><tr><th>Feature installation</th><th>Capability</th><th>Requester</th><th>Status</th></tr>${conns.map(([i, g]) =>
      `<tr><td>${esc(i.name)} <span class="muted small">${esc(short(i.installation_id, 12))}</span></td><td>${mono(g.grant.capability)}</td><td>${mono(g.grant.requester)}</td><td>${badge(g.status, g.status === "ACTIVE" ? "ok" : "")}</td></tr>`).join("")}</table>`
      : `<p class="muted">None. Every identity starts with no access; connections are approved one capability at a time by the Admin.</p>`;
    const cards = (s.cards || []).filter((c) => c.from_identity);
    if (cards.length) h += `<h3>Identity's feature catalog (P0 release downloads)</h3><p class="muted small">Downloading a published release is P0 behaviour. Running third-party releases is <strong>BLOCKED</strong> in P1 (no verified provenance or OS sandbox).</p>` +
      cards.map((c) => `<div class="mini">${esc(c.name)} · ${badge(c.eligible_to_install ? c.eligible_to_install.state : "", c.installable ? "info" : "")} ${c.install && c.install.installed ? badge("downloaded " + c.install.installed.tag, "ok") : ""} ${badge("execution BLOCKED", "bad")}</div>`).join("");
    set(d, h);
  }

  function renderLibrary(s, d) {
    const lib = s.library || [];
    let h = `<div class="dhead"><div class="typed">LIBRARY</div><h1>Add a feature</h1><p class="muted">Feature packages from local folders you trusted when starting Processor. In P1 only the exact, pinned reviewed test fixture (or an artifact you pinned with <code>-trust-feature-artifact</code>) can be installed — what a package says about itself never counts. Third-party executables stay <strong>BLOCKED</strong> until release provenance and OS sandboxing are verified. Invitations are not built yet.</p></div>`;
    h += lib.map((x) => `<div class="libcard ${x.blocked ? "blocked" : ""}">
      <div class="lc-h"><div><h3>${esc(x.name || x.dir)}</h3><div class="muted small">${esc(x.description || x.error || "")}</div></div>
      ${x.blocked ? badge(x.blocked === "INVALID_PACKAGE" ? "INVALID" : "BLOCKED", "bad", x.blocked)
        : (x.installed_as || []).length ? `<button class="btn primary" data-sel="feature:${esc(x.installed_as[0])}">Open installed feature</button>`
        : `<button class="btn primary" data-review="${esc(x.feature_id)}">Install…</button>`}</div>
      <div class="badges">${x.version ? badge("v" + x.version) : ""}${x.trust_basis ? badge("pinned artifact", "info", x.trust_basis) : x.trust_class ? badge("claims " + x.trust_class + " — not pinned", "bad", "A package's own label can't grant trust") : ""}${x.api ? badge(x.api) : ""}${(x.installed_as || []).length ? badge("installed ×" + x.installed_as.length, "ok") : ""}</div>
      ${x.blocked ? `<p class="small error">${esc(x.blocked)}${x.error ? " — " + esc(x.error) : ""}</p>` : ""}
      <details class="small"><summary>Package details</summary>${kv([["Folder", esc(x.dir)], ["Publisher", esc(x.publisher_claim || "") + " " + badge("claim only — not verified", "warn")], ["Source", esc(x.source_repo || "")],
        ["Artifact SHA-256", mono(x.artifact_sha256 || "")], ["Runs", mono((x.command || []).join(" "))], ["Needs", esc((x.requires || []).join(", "))],
        ["Capabilities", (x.capabilities || []).map((c) => mono(c.id) + " " + badge(c.data_class)).join(" ")], ["Files", (x.files || []).map((f) => mono(f.path) + " " + mono(short(f.sha256, 16))).join("<br>")]])}</details></div>`).join("")
      || `<p class="muted">No packages found in the trusted folders.</p>`;
    set(d, h);
  }

  // The owner-facing run card: one plain state (Stopped / Starting / Running /
  // Stopping / Error), the one button that makes sense now, and the last
  // standalone self-test as the worker actually answered it.
  const phaseOf = (st) => ({ STOPPED: ["Stopped", "idle"], STARTING: ["Starting…", "warn"], RESTARTING: ["Starting…", "warn"], RUNNING: ["Running", "ok"],
    STOPPING: ["Stopping…", "warn"], FAILED_TO_START: ["Error", "bad"], NEEDS_ATTENTION: ["Error", "bad"], CRASHED: ["Error", "bad"] })[st] || [st, ""];
  // Plain words for why a request was refused (the raw reason stays visible too).
  const why = { NO_GRANT: "No permission yet. Ask the Admin to allow it (Connections → Ask the Admin…).",
    GRANT_REVOKED: "The permission was revoked. Ask the Admin again if it should be allowed.",
    GRANT_EXPIRED: "The permission expired (they last 1 hour unless the Admin chose otherwise). Ask the Admin again.",
    TICKET_UNKNOWN: "Identity no longer knows that one-time ticket (for example, after a revocation). Nothing was done.",
    ARTIFACT_NOT_PINNED: "Identity doesn't trust this exact package version. Nothing was started.",
    BAD_INPUT: "The service was allowed to run the request but couldn't use the input. Nothing changed." };
  function explain(r) {
    if (!r) return "";
    if (r.outcome === "AUTHORITY_UNAVAILABLE") return "Identity can't be reached, so nothing was allowed or started. Start Identity; Processor reconnects by itself.";
    if (r.outcome === "NOT_RUNNING") return "The service is stopped. Press Start first; nothing was sent.";
    const m = (r.detail || "").match(/\b([A-Z][A-Z_]{5,})\b/g) || [];
    for (const k of m) if (why[k]) return why[k];
    return "";
  }
  // A service's answer: its own headline and a few plain facts first; the raw
  // answer (IDs, protocol details) stays one click away.
  function obsBlock(o) {
    const out = o.output || {}, il = out.identity_link, pl = out.processor_link, tr = out.this_request;
    const facts = [];
    if (pl) facts.push("Processor link: " + pl.state);
    if (il) facts.push("Identity link: " + il.state + (il.this_installation && il.this_installation.active_grants != null ? " · " + il.this_installation.active_grants + " active grant(s)" : ""));
    if (tr) facts.push("Allowed by: " + tr.allowed_by);
    if (out.cannot) facts.push("Cannot: " + out.cannot.join("; "));
    const raw = `<pre>${esc(JSON.stringify(out, null, 2))}</pre>`;
    return `<div class="obs"><div class="kicker">${esc(o.data_class)}</div>` + (out.headline
      ? `<div><strong>${esc(out.headline)}</strong></div>${facts.length ? `<ul class="plain small">${facts.map((f) => `<li>${esc(f)}</li>`).join("")}</ul>` : ""}<details class="small"><summary>Full answer</summary>${raw}</details>`
      : raw) + `</div>`;
  }
  function resultBox(la) {
    const oc = { COMPLETED: "ok", DENIED_BY_IDENTITY: "bad", DENIED_BY_FEATURE: "bad", AUTHORITY_UNAVAILABLE: "warn", REQUEST_UNCERTAIN: "warn", NOT_RUNNING: "warn", TICKET_REJECTED: "bad", ERROR: "bad" };
    const x = explain(la);
    return `<div class="result ${oc[la.outcome] || ""}"><div><strong>${esc(la.outcome.replace(/_/g, " "))}</strong> ${la.started_feature ? badge("started on demand", "info") : ""} <span class="muted small">${esc(la.request_id)} · ${la.latency_ms} ms ${la.decision_id ? "· decision " + esc(la.decision_id) : ""}</span></div>
      ${x ? `<div class="small"><strong>${esc(x)}</strong></div>` : ""}<div class="small">${esc(la.detail)}</div>${la.observation ? obsBlock(la.observation) : ""}</div>`;
  }
  // The owner-facing run card: one plain state, the evidence behind it, who may
  // use it, ONE next action, and the last self-test as the service answered it.
  function runCard(s, v, iv) {
    const rt = v.runtime, r = v.record, ins = r.installation_id, [word, cls] = phaseOf(rt.state);
    const busy = rt.state === "STARTING" || rt.state === "RESTARTING" || rt.state === "STOPPING";
    const st = (rt.health && rt.health.status) || {};
    let h = `<div class="runcard"><div class="rc-h"><span class="dot ${cls}"></span><strong class="rc-state">${esc(word)}</strong>`;
    if (rt.state === "RUNNING") h += ` <span class="muted small">pid ${esc(rt.pid)} · launch ${esc(rt.launch_id)} · ${rt.health && rt.health.ready ? "health READY" : "health not ready"}</span>`;
    h += `</div>`;
    if (rt.state === "RUNNING" && st.entrance) {
      // Each chip is the service's own launch-signed status, polled every 2 s.
      const a = st.authority || {};
      h += `<div class="badges evid">${badge("RUNNING", "ok", "Processor verified this launch's signed status (pid " + rt.pid + ")")}`
        + badge(st.entrance.connected ? "CONNECTED" : "NOT CONNECTED (" + (a.state || "?") + ")", st.entrance.connected ? "ok" : "bad", a.detail || "")
        + badge(st.entrance.ready ? "READY" : "NOT READY", st.entrance.ready ? "ok" : "warn") + `</div>`
        + `<p class="small muted">Identity link: ${esc(a.detail || "")}${a.checked_at ? " · checked " + esc(new Date(a.checked_at).toLocaleTimeString()) : ""}</p>`;
    }
    if (cls === "bad") h += `<div class="problem small"><strong>${esc(rt.state.replace(/_/g, " "))}:</strong> ${esc(rt.note || (rt.last_exit ? rt.last_exit.reason : ""))} Nothing is left running. Fix the cause, then press Start again.</div>`;
    // Who may use it (Identity's signed grants).
    const caps = r.capabilities || [], grants = ((iv && iv.grants) || []).filter((g) => g.status === "ACTIVE");
    const name = (eid) => { const e = (s.identities || []).find((x) => x.id === eid); return e ? e.name : eid; };
    if (caps.length) h += `<p class="small"><strong>Access</strong> ${!iv ? `<span class="warn">Identity can't confirm grants right now — nothing protected can run.</span>` : caps.map((c) => {
      const gs = grants.filter((g) => g.grant.capability === c.id);
      return `${mono(c.id)}: ${gs.length ? "allowed for " + gs.map((g) => esc(name(g.grant.subject_identity_id))).join(", ") : `<span class="muted">not allowed for anyone yet</span>`}`;
    }).join(" · ")}</p>`;
    h += `<div class="row">`;
    if (rt.state === "RUNNING") h += `<button class="btn" data-stop="${esc(ins)}">Stop</button><button class="btn" data-selftest="${esc(ins)}">Run self-test</button>`;
    else if (busy) h += `<button class="btn" disabled>${esc(word)}</button>`;
    else h += `<button class="btn primary" data-start="${esc(ins)}" ${iv ? "" : "disabled"}>${cls === "bad" ? "Start again" : "Start"}</button><button class="btn" disabled title="The self-test runs inside the service: start it first">Run self-test</button>`;
    h += `<button class="btn ghost small" data-copyact="${esc(ins)}" title="Copies Processor's record of this service: no keys, tickets, codes or private links">Copy activity</button></div>`;
    // ONE next action.
    if (!iv && !busy && rt.state !== "RUNNING") h += `<p class="small warn">Start needs Identity: it must confirm this installation before anything runs.</p>`;
    else if (rt.state === "RUNNING" && caps.length) {
      const quick = r.feature_id === "developer-utility" ? caps[0] : null;
      const allowed = quick ? grants.filter((g) => g.grant.capability === quick.id) : [];
      if (quick && allowed.length) h += `<div class="next"><strong>Next:</strong> ${allowed.map((g) => `<button class="btn primary small" data-quick="${esc(ins)}" data-sub="${esc(g.grant.subject_identity_id)}" data-cap="${esc(quick.id)}">Run ${esc(quick.id.split(".").slice(1).join(" "))} for ${esc(name(g.grant.subject_identity_id))}</button>`).join(" ")}</div>`;
      else if (!grants.length) h += `<div class="next"><strong>Next:</strong> nothing may use it yet. <button class="btn small" data-tab="connections">Ask the Admin for a permission →</button></div>`;
      else h += `<div class="next"><strong>Next:</strong> <button class="btn small" data-tab="connections">Send a request →</button></div>`;
    }
    if (r.feature_id === "developer-utility" && lastAction[ins]) h += resultBox(lastAction[ins]);
    const t = v.self_test;
    if (t) {
      const stale = rt.launch_id && t.launch_id && t.launch_id !== rt.launch_id;
      h += `<div class="result ${t.result === "PASS" ? "ok" : "bad"}"><div><strong>Self-test ${esc(t.result)}</strong> <span class="muted small">${esc(new Date(t.at).toLocaleTimeString())} · launch ${esc(t.launch_id || "?")}${stale ? " (an earlier launch)" : ""}</span></div>`
        + (t.error ? `<div class="small">${esc(t.error)}</div>` : `<div class="small muted">${esc(t.data_class || "")} · external calls: ${esc(t.external_calls ?? "")} · grants used: ${esc(t.grants_used || "")}</div>`
          + (t.output && t.output.checks ? `<ul class="plain small">${t.output.checks.map((c) => `<li>${c.ok ? "✓" : "✗"} ${esc(c.check)}</li>`).join("")}</ul><details class="small"><summary>Full answer</summary><pre>${esc(JSON.stringify(t.output, null, 2))}</pre></details>`
            : `<pre>${esc(JSON.stringify(t.output, null, 2))}</pre>`)) + `</div>`;
    } else if (rt.state === "RUNNING") h += `<p class="small muted">Self-test not run yet for this launch. It uses no grant.</p>`;
    if (activity[ins]) h += `<details class="small" open><summary>Activity (copied to clipboard)</summary><pre class="logbox">${esc(activity[ins])}</pre></details>`;
    return h + `</div>`;
  }

  function renderFeature(s, d, v) {
    const r = v.record, iv = v.identity, rt = v.runtime, ins = r.installation_id;
    const key = "feature:" + ins + ":" + (r.state);
    if (d.dataset.key !== key) {
      d.dataset.key = key; d.dataset.html = "";
      d.innerHTML = `<div id="fx-head"></div><div class="tabs" role="tablist">${["overview", "connections", "runtime", "metrics", "logs"].map((t) =>
        `<button type="button" role="tab" data-tab="${t}" class="${tab === t ? "on" : ""}">${t[0].toUpperCase() + t.slice(1)}</button>`).join("")}</div>
        <div class="pane" id="pane-overview"></div><div class="pane" id="pane-connections"><div id="fx-grants"></div><div id="fx-grant-form"></div><div id="fx-action"></div><div id="fx-action-result"></div><div id="fx-requests"></div></div>
        <div class="pane" id="pane-runtime"><div id="fx-rt"></div><div id="fx-policy"></div></div><div class="pane" id="pane-metrics"></div><div class="pane" id="pane-logs"></div>`;
    }
    document.querySelectorAll(".tabs button").forEach((b) => b.classList.toggle("on", b.dataset.tab === tab));
    document.querySelectorAll(".pane").forEach((p) => (p.hidden = p.id !== "pane-" + tab));
    const idState = iv ? iv.installation.state : s.verified ? "NOT LISTED BY IDENTITY" : "can't confirm (AUTHORITY UNAVAILABLE)";
    const active = iv ? iv.active_grants : 0;
    const gone = r.state !== "INSTALLED";
    set($("#fx-head"), `<div class="dhead"><div class="typed feature">FEATURE INSTALLATION IDENTITY</div><h1>${esc(r.name)}</h1>
      <div class="badges">${badge("Identity: " + idState, idState === "INSTALLED" ? "ok" : gone ? "" : "warn")}${gone ? "" : badge("Process: " + rt.state, stateDot[rt.state] === "ok" ? "ok" : stateDot[rt.state] === "bad" ? "bad" : "")}
      ${rt.state === "RUNNING" ? badge(rt.health && rt.health.ready ? "Health: READY" : "Health: not ready", rt.health && rt.health.ready ? "ok" : "warn") : ""}
      ${iv ? badge(active + " active grant" + (active === 1 ? "" : "s"), active ? "info" : "") : gone ? "" : badge("grants: can't confirm", "warn")}${badge("0 information sources")}${badge("pinned artifact", "info", (r.trust_basis || "artifact pin") + " — publisher not verified")}
      ${v.files && v.files !== "OK" && !gone ? badge(v.files, "bad") : ""}</div>
      ${v.same_artifact_copies && !gone ? `<p class="small warn">${esc(v.same_artifact_copies)} other installation(s) of this same artifact exist in this workspace (made before repeated installs returned the existing one). Nothing was removed; uninstall extras yourself if you don't want them.</p>` : ""}</div>`);
    // Overview: running it comes first.
    set($("#pane-overview"), (gone ? "" : `<h3>Run</h3>` + runCard(s, v, iv)) + `<h3>Installation</h3>` + kv([["Installation ID", mono(ins) + ` <span class="muted small">durable · minted by Identity · never reused</span>`], ["Feature", mono(r.feature_id) + " v" + esc(r.version)],
      ["Publisher", esc(r.publisher_claim) + " " + badge("claim only — not verified", "warn")], ["Source", esc(r.source_repo) + `<br><span class="muted small">installed from ${esc(r.source_dir)}</span>`],
      ["Trusted because", esc(r.trust_basis || "artifact pin (recorded before the cleanup)")], ["Artifact SHA-256", mono(r.artifact_sha256)], ["Manifest SHA-256", mono(r.manifest_sha256)], ["API", mono("thought.feature-api/1")],
      ["Installed", esc(new Date(r.installed_at).toLocaleString())], iv && ["Administered by", mono(iv.installation.administered_by) + " (Admin approval " + mono(short(iv.installation.approval_id, 14)) + ")"],
      ["Files", gone ? "removed" : v.files === "OK" ? badge("verified against the approved SHA-256", "ok") : badge(v.files, "bad")], ["Data folder", esc(v.data_dir) + (gone && r.data_kept ? " (kept)" : "")],
      ["Capabilities declared", (r.capabilities || []).map((c) => mono(c.id) + " " + badge(c.data_class)).join(" ")],
      gone && ["Uninstalled", esc(new Date(r.uninstalled_at).toLocaleString()) + " — tombstone kept; grants revoked; credential removed"]])
      + (gone ? "" : `<div class="danger"><h3>Uninstall</h3><p class="small muted">Needs a fresh Admin approval. Revokes every grant and removes the credential and files; the installation ID stays as a tombstone.</p>
        <label class="small"><input type="checkbox" id="keep-data" checked> keep this installation's data folder</label> <button class="btn danger small" data-uninstall="${esc(ins)}">Uninstall…</button></div>`));
    // Connections
    const grants = (iv && iv.grants) || [];
    const me = "processor:" + s.processor.processor_id;
    set($("#fx-grants"), `<h3>Connections (grants from Identity)</h3>` + (grants.length ? `<table class="tbl"><tr><th>Requester</th><th>Subject</th><th>Capability</th><th>Expires</th><th>Status</th><th></th></tr>${grants.map((g) => {
      const G = g.grant; const ent = (s.identities || []).find((e) => e.id === G.subject_identity_id);
      const mine = G.requester === me || G.requested_by === me;
      const ended = G.revoked_at ? `<div class="small muted">revoked by ${esc(G.revoked_by === me ? "this Processor" : G.revoked_by || "?")} · ${esc(new Date(G.revoked_at).toLocaleString())}</div>` : "";
      return `<tr><td>${mono(G.requester === me ? "this Processor" : G.requester)}</td><td>${esc(ent ? ent.name : G.subject_identity_id)}</td><td>${mono(G.capability)}</td>
        <td class="small">${G.expires_at ? esc(new Date(G.expires_at).toLocaleString()) : G.until_revoked ? "never — until revoked (Admin confirmed)" : "never (granted before expiry defaults)"}</td><td>${badge(g.status, g.status === "ACTIVE" ? "ok" : "")}${ended}</td>
        <td>${g.status === "ACTIVE" && !gone ? (mine ? `<button class="btn ghost small" data-revoke="${esc(G.grant_id)}">Revoke</button>`
          : `<button class="btn ghost small" data-adminrevoke="${esc(G.grant_id)}" title="This grant belongs to another program; only the Admin can end it from here">Ask the Admin to revoke…</button>`) : ""}</td></tr>`;
    }).join("")}</table>` : !iv ? `<p class="warn">AUTHORITY UNAVAILABLE — Identity can't confirm this installation's grants right now, so no protected work can happen.</p>` : `<p class="muted">No grants. This installation can't do any protected work for anyone.</p>`)
      + (v.pending || []).map((p) => `<div class="pending"><span class="dot warn"></span> ${esc(({ grant_capability: "Connection request", uninstall_feature: "Uninstall request", revoke_grant: "Admin revocation request" })[p.kind] || p.kind)} ${mono(p.approval_id)} waiting for the Admin <button class="btn ghost small" data-watch="${esc(p.approval_id)}">Open approval</button></div>`).join(""));
    const ents = s.identities || [], caps = r.capabilities || [];
    const formKey = ins + "|" + ents.map((e) => e.id).join(",") + "|" + caps.map((c) => c.id).join(",") + "|" + gone;
    const gf = $("#fx-grant-form");
    if (gf.dataset.key !== formKey) {
      gf.dataset.key = formKey;
      const opts = ents.map((e) => `<option value="${esc(e.id)}">${esc(e.name)} (${esc(e.kind)})</option>`).join("");
      const copts = caps.map((c) => `<option value="${esc(c.id)}">${esc(c.id)}</option>`).join("");
      gf.innerHTML = gone ? "" : `<form class="frm" id="grant-form"><h3>Request a connection</h3>
        <div class="frow"><label>Requester<input value="This Thought Processor" disabled></label><label>For identity<select id="gf-sub" required>${opts}</select></label>
        <label>Capability<select id="gf-cap">${copts}</select></label><label>Expires<select id="gf-exp"><option value="600">in 10 minutes</option><option value="3600" selected>in 1 hour (default)</option><option value="86400">in 1 day</option><option value="forever">never — until revoked (the Admin must confirm this in Identity)</option></select></label></div>
        <p class="small muted">Information sources: <strong>none</strong>. The Admin approves this exact request in Identity.</p><button class="btn primary" type="submit" ${opts ? "" : "disabled"}>Ask the Admin…</button>${opts ? "" : ` <span class="small muted">Add an identity first.</span>`}</form>`;
      $("#fx-action").innerHTML = gone ? "" : `<form class="frm" id="action-form"><h3>Try the protected capability</h3>
        <p class="small muted">Processor first asks Identity for a one-request ticket. No grant → nothing starts. With a grant, an on-demand feature is started, checks the ticket and asks Identity live before doing anything.</p>
        <div class="frow"><label>For identity<select id="af-sub">${opts}</select></label><label>Capability<select id="af-cap">${copts}</select></label><label class="grow">Input<input id="af-in" maxlength="1000" value="${caps[0] && caps[0].id === "worker.test.synthetic" ? "hello from P1" : ""}"></label></div>
        <p class="small muted" id="af-hint">${esc((caps[0] && caps[0].description) || "")}</p>
        <button class="btn" type="submit" ${opts ? "" : "disabled"}>Send request</button></form>`;
      const f1 = $("#grant-form"), f2 = $("#action-form");
      const afc = $("#af-cap");
      if (afc) afc.onchange = () => { const c = caps.find((x) => x.id === afc.value); $("#af-hint").textContent = (c && c.description) || ""; $("#af-in").placeholder = (c && c.description) || ""; };
      if (f1) f1.onsubmit = async (e) => { e.preventDefault(); const j = await act(`/api/installations/${ins}/grants/request`, { subject: $("#gf-sub").value, capability: $("#gf-cap").value, ...($("#gf-exp").value === "forever" ? { until_revoked: true } : { expires_in_seconds: +$("#gf-exp").value }) }); if (j && j.approval_url) watch(j); };
      if (f2) f2.onsubmit = async (e) => { e.preventDefault(); const b = e.submitter; if (b) b.disabled = true; const j = await act(`/api/installations/${ins}/action`, { subject: $("#af-sub").value, capability: $("#af-cap").value, input: $("#af-in").value }); if (b) b.disabled = false; if (j && j.result) { lastAction[ins] = j.result; render(); } };
    }
    const la = lastAction[ins];
    const oc = { COMPLETED: "ok", DENIED_BY_IDENTITY: "bad", DENIED_BY_FEATURE: "bad", AUTHORITY_UNAVAILABLE: "warn", REQUEST_UNCERTAIN: "warn", NOT_RUNNING: "warn", TICKET_REJECTED: "bad", ERROR: "bad" };
    set($("#fx-action-result"), la ? resultBox(la) : "");
    set($("#fx-requests"), (v.requests || []).length ? `<h3>Recent requests</h3><table class="tbl small"><tr><th>When</th><th>Request</th><th>Outcome</th><th>Decision</th></tr>${v.requests.map((q) =>
      `<tr><td>${esc(new Date(q.at).toLocaleTimeString())}</td><td>${mono(q.request_id)}</td><td>${badge(q.outcome, oc[q.outcome] || "")}</td><td>${q.decision_id ? mono(q.decision_id) : ""}</td></tr>`).join("")}</table>` : "");
    // Runtime
    const h = rt.health || {};
    set($("#fx-rt"), gone ? `<p class="muted">Uninstalled — nothing runs.</p>` : `<h3>Process</h3>` + kv([["State", badge(rt.state, stateDot[rt.state] === "ok" ? "ok" : stateDot[rt.state] === "bad" ? "bad" : "") + (rt.note ? ` <span class="small muted">${esc(rt.note)}</span>` : "")],
      rt.pid && ["PID", mono(rt.pid)], rt.launch_id && ["Launch", mono(rt.launch_id) + ` <span class="muted small">each launch has its own key; the endpoint is trusted only after a status signed with it</span>`],
      rt.url && ["Local API", mono(rt.url) + " (loopback; port chosen by the OS)"], rt.started_at && ["Started", esc(ago(rt.started_at)) + " · by " + esc(rt.started_by || "")],
      rt.state === "RUNNING" && ["Heartbeat", (h.ready ? badge("READY", "ok") : badge("not ready", "warn")) + " seq " + esc(h.heartbeat_seq) + ` <span class="muted small">liveness only — not an observation, grant or result</span>`],
      rt.state === "RUNNING" && h.status && ["Provider", esc(h.status.provider ? h.status.provider.ready : "")], ["In flight", esc(rt.in_flight)],
      rt.idle_stop_at && ["Idle stop", "at " + esc(new Date(rt.idle_stop_at).toLocaleTimeString()) + " if no work arrives"],
      rt.last_exit && ["Last exit", esc(new Date(rt.last_exit.at).toLocaleTimeString()) + " code " + esc(rt.last_exit.code) + " — " + esc(rt.last_exit.reason)],
      (rt.recent_restarts || []).length && ["Automatic restarts", esc(rt.recent_restarts.length) + " in the last 10 min (requests are never resent)"]])
      + (gone ? "" : runCard(s, v, iv)));
    const pkey = ins + JSON.stringify(r.policy) + gone;
    const pf = $("#fx-policy");
    if (pf.dataset.key !== pkey) {
      pf.dataset.key = pkey;
      const modes = [["manual", "Manual — only when you press Start (default for new installs)"], ["on_demand", "On demand — start for an already-authorized request, stop after the idle window"], ["always_running", "Always running — Processor keeps it started"]]
        .filter(([m]) => m === "manual" || (r.startup_modes || []).includes(m));
      pf.innerHTML = gone ? "" : `<form class="frm" id="policy-form"><h3>Startup and recovery</h3><p class="small muted">These change <em>when</em> it runs — never <em>what</em> it may access.</p>
        ${modes.map(([m, t]) => `<label class="choice"><input type="radio" name="pmode" value="${m}" ${r.policy.startup_mode === m ? "checked" : ""}> <span>${esc(t)}</span></label>`).join("")}
        <div class="frow"><label>Idle window (seconds)<input id="p-idle" type="number" min="5" max="3600" value="${esc(r.policy.idle_seconds)}"></label>
        <label>Crash recovery<select id="p-rec"><option value="manual" ${r.policy.recovery === "manual" ? "selected" : ""}>Manual (default for experimental features)</option><option value="bounded" ${r.policy.recovery === "bounded" ? "selected" : ""}>Bounded — up to 3 process restarts in 10 min</option></select></label></div>
        <p class="small muted">Recovery restarts the process, never a request. Uncertain requests are not resent.</p><button class="btn" type="submit">Save</button></form>`;
      const f = $("#policy-form");
      if (f) f.onsubmit = (e) => { e.preventDefault(); act(`/api/installations/${ins}/policy`, { startup_mode: document.querySelector('input[name="pmode"]:checked').value, idle_seconds: +$("#p-idle").value, recovery: $("#p-rec").value }); };
    }
    // Metrics
    const st = h.status || {};
    const c = st.counters || {};
    const outs = {};
    (v.requests || []).forEach((q) => (outs[q.outcome] = (outs[q.outcome] || 0) + 1));
    set($("#pane-metrics"), `<h3>Feature counters <span class="muted small">(reported by the running feature)</span></h3>` + (rt.state === "RUNNING" ? `<div class="stats">${[["requests", c.requests], ["allowed", c.allowed], ["denied", c.denied], ["authority unavailable", c.unavailable], ["refused tickets", c.invalid], ["self-tests", c.self_tests], ["p50 ms", st.latency_ms_p50 ?? "—"], ["uptime s", st.uptime_s]].map(([k, v]) =>
      `<div class="stat"><div class="n">${esc(v ?? 0)}</div><div class="k">${esc(k)}</div></div>`).join("")}</div>` : `<p class="muted">Not running.</p>`)
      + `<h3>Processor-side outcomes (this session)</h3><div class="stats">${Object.keys(outs).length ? Object.entries(outs).map(([k, v]) => `<div class="stat"><div class="n">${v}</div><div class="k">${esc(k.replace(/_/g, " ").toLowerCase())}</div></div>`).join("") : `<span class="muted">No requests yet.</span>`}</div>`);
    // Logs
    const dt = details[ins];
    set($("#pane-logs"), `<h3>Identity decisions for this installation</h3>` + (dt && dt.identity_decisions ? `<table class="tbl small"><tr><th>When</th><th>Decision</th><th>Reason</th><th>Event</th><th>ID</th></tr>${dt.identity_decisions.slice(0, 40).map((x) =>
      `<tr><td>${esc(new Date(x.at).toLocaleTimeString())}</td><td>${badge(x.decision, x.decision.startsWith("ALLOW") ? "ok" : "bad")}</td><td>${esc(x.reason || "")}</td><td>${esc(x.event)}</td><td>${mono(x.decision_id)}</td></tr>`).join("")}</table>` : `<p class="muted">${dt && dt.identity_decisions_error ? "AUTHORITY UNAVAILABLE" : "Loading…"}</p>`)
      + `<h3>What the feature itself hears from Identity</h3><pre class="logbox">${esc(dt ? JSON.stringify(dt.feature_view_of_connections, null, 2) : "")}</pre>`
      + `<h3>Feature log</h3><pre class="logbox">${esc(dt ? (dt.log_tail || []).join("\n") : "")}</pre>`);
    if (tab === "logs" && (!dt || Date.now() - (dt._at || 0) > 3500)) loadDetails(ins);
  }
  async function loadDetails(ins) {
    try { const j = await api("GET", `/api/installations/${ins}/details`); j._at = Date.now(); details[ins] = j; render(); } catch (_) {}
  }

  function renderDev(s) {
    const d = s.dev; $("#dev").hidden = !d; if (!d) return;
    const logs = (d.log || []).map((e) => `<div class="${e.level === "warn" || e.level === "error" ? "warn" : ""}">${esc(new Date(e.time).toLocaleTimeString())} ${esc(e.component)} ${esc(e.event)} — ${esc(e.detail)}</div>`).join("");
    const info = { processor: s.processor, connection: s.connection, lifecycle: s.lifecycle, build: d.build, run_id: d.run_id, profile: d.profile, admin: s.admin, installations: s.identity_installations };
    set($("#dev-body"), `<pre>${esc(JSON.stringify(info, null, 2))}</pre><h3 class="small">Recent activity</h3><div class="log">${logs}</div>`);
  }

  // ---------- approval hand-off ----------
  function watch(j) { approvalWatch = { id: j.approval_id, url: j.approval_url, opened: false }; render(); schedule(); }
  function findApproval(s, id) {
    const ops = [...(s.admin_pending || []), ...(s.installations || []).flatMap((v) => v.pending || [])];
    const op = ops.find((p) => p.approval_id === id);
    const tx = (s.install_txs || []).find((t) => t.approval_id === id);
    return { op, tx };
  }
  function renderModals(s) {
    const g = s.github, mg = $("#m-github");
    if (s.verified && g && g.state === "WAITING") {
      mg.hidden = false; $("#g-code").textContent = g.user_code || "…"; $("#g-link").href = g.verification_uri || "#";
      $("#g-link").textContent = (g.verification_uri || "").replace(/^https?:\/\//, "");
      $("#g-status").textContent = "Waiting for you to approve on GitHub… " + (g.github_said ? "(GitHub says: " + g.github_said + ")" : "");
    } else mg.hidden = true;
    const m = $("#m-approval");
    if (!approvalWatch) { m.hidden = true; } else {
      const { op, tx } = findApproval(s, approvalWatch.id);
      m.hidden = false;
      $("#ap-title").textContent = op ? ({ enroll_admin: "Establish the Admin Identity", grant_capability: "Approve a connection", uninstall_feature: "Approve an uninstall" }[op.kind] || "Approve") : tx ? "Approve installing " + tx.name : "Approval finished";
      set($("#ap-summary"), ((op && op.summary) || (tx ? ["Register a new installation identity for " + tx.name + " v" + tx.version, "Artifact SHA-256 " + tx.artifact_sha256, "Zero grants, zero information sources"] : [])).map((x) => `<li>${esc(x)}</li>`).join(""));
      const done = !op && !tx;
      $("#ap-status").textContent = done ? "Done — see the result in the workspace and the activity notes." : approvalWatch.opened ? "Waiting for the Admin's decision in Identity…" : "Waiting for the Admin…";
      $("#ap-open").hidden = done || approvalWatch.opened;
      $("#ap-open").onclick = () => { window.open(approvalWatch.url, "_blank", "noopener"); approvalWatch.opened = true; render(); };
      $("#ap-close").onclick = () => { approvalWatch = null; render(); };
      if (done && s.admin && s.admin.state === "ENROLLED" && sel && sel.type === "admin") {/* admin view updates by itself */}
    }
    const mi = $("#m-install");
    const it = (s.library || []).find((x) => x.feature_id === reviewFor);
    if (!it) { mi.hidden = true; return; }
    mi.hidden = false;
    $("#i-title").textContent = "Install " + it.name + "?";
    set($("#i-facts"), [["Version", it.version], ["Publisher", it.publisher_claim + " (claim only)"], ["Source", it.source_repo], ["From folder", it.dir], ["Trusted because", it.trust_basis || "NOT PINNED"], ["Package's own label", it.trust_class + " (descriptive only)"], ["Runs", (it.command || []).join(" ")],
      ["Needs", (it.requires || []).join(", ")], ["Artifact SHA-256", it.artifact_sha256], ["Capabilities", (it.capabilities || []).map((c) => c.id + " (" + c.data_class + ")").join(", ")]].map(([k, v]) => `<dt>${esc(k)}</dt><dd>${esc(v)}</dd>`).join(""));
    set($("#i-checks"), `<strong>Checked</strong><ul><li>every file matches the SHA-256 in its manifest; no links, no paths outside the package</li><li>the folder is one you trusted with -trust-feature-dir</li><li>this exact artifact is pinned (${esc(it.trust_basis || "")}); the package's own trust label is not evidence</li><li>files are copied and checked again before registering</li></ul>`);
    set($("#i-not"), `<strong>Not checked</strong><ul>${(it.not_verified || []).map((x) => `<li>${esc(x)}</li>`).join("")}</ul>`);
    $("#i-ok").onclick = async () => { const f = reviewFor; reviewFor = ""; mi.hidden = true; const j = await act(`/api/library/${encodeURIComponent(f)}/install`, {});
      if (j && j.already_installed) { toast("Already installed — opened the existing installation (nothing new was registered).", "ok"); choose({ type: "feature", id: j.installation_id }); }
      else if (j && j.approval_url) watch(j); };
    $("#i-cancel").onclick = () => { reviewFor = ""; mi.hidden = true; };
  }

  // ---------- events ----------
  document.addEventListener("click", async (e) => {
    const t = e.target.closest("button, .chip");
    if (!t) return;
    const ds = t.dataset;
    if (ds.connect) await act("/api/connect/request", { url: ds.connect }, $("#c-error"));
    else if (ds.newcode) await act("/api/connect/request", {}, $("#c-error"));
    else if (ds.reconnect) await act("/api/connect/disconnect", {});
    else if (ds.sel) { const [type, ...rest] = ds.sel.split(":"); choose({ type, id: rest.join(":") }); }
    else if (ds.tab) { tab = ds.tab; sessionStorage.setItem("tp_tab", tab); render(); if (tab === "logs" && sel) loadDetails(sel.id); }
    else if (ds.review) { reviewFor = ds.review; render(); }
    else if (ds.enroll) { const j = await act("/api/admin/enroll", {}); if (j && j.approval_url) watch(j); }
    else if (ds.watch) {
      const ops = [...(state.admin_pending || []), ...(state.installations || []).flatMap((v) => v.pending || [])];
      const op = ops.find((p) => p.approval_id === ds.watch);
      if (op) watch({ approval_id: op.approval_id, approval_url: op.approval_url });
    }
    else if (ds.start) { t.disabled = true; await act(`/api/installations/${ds.start}/start`, {}); }
    else if (ds.stop) { t.disabled = true; await act(`/api/installations/${ds.stop}/stop`, {}); }
    else if (ds.quick) { t.disabled = true; const j = await act(`/api/installations/${ds.quick}/action`, { subject: ds.sub, capability: ds.cap, input: "" }); if (j && j.result) { lastAction[ds.quick] = j.result; render(); } }
    else if (ds.copyact) {
      try { const j = await api("GET", `/api/installations/${ds.copyact}/activity`); activity[ds.copyact] = j.text;
        try { await navigator.clipboard.writeText(j.text); toast("Activity copied — no keys, tickets, codes or private links.", "ok"); } catch (_) { toast("Shown below — select it and copy.", "ok"); }
        render(); } catch (e) { toast(e.message, "bad"); }
    }
    else if (ds.selftest) { t.disabled = true; await act(`/api/installations/${ds.selftest}/self-test`, {}); } // result is kept by Processor and shown in the run card
    else if (ds.revoke) { if (confirm("Revoke this grant? The next request is denied immediately.")) await act(`/api/installations/${sel.id}/grants/${ds.revoke}/revoke`, {}); }
    else if (ds.adminrevoke) { const j = await act(`/api/installations/${sel.id}/grants/${ds.adminrevoke}/admin-revoke`, {}); if (j && j.approval_url) watch(j); }
    else if (ds.uninstall) { const j = await act(`/api/installations/${ds.uninstall}/uninstall`, { keep_data: !!($("#keep-data") && $("#keep-data").checked) }); if (j && j.approval_url) watch(j); }
    else if (ds.gh || (t.id === "chip-github" && state && state.github && state.github.state !== "VERIFIED")) await act("/api/github/connect", {});
    else if (t.id === "chip-admin") choose({ type: "admin", id: "admin" });
  });
  $("#add-id-btn").onclick = () => { $("#new-id").hidden = !$("#new-id").hidden; if (!$("#new-id").hidden) $("#new-name").focus(); };
  $("#add-feature-btn").onclick = () => choose({ type: "library", id: "all" });
  $("#theme").onchange = (e) => { applyTheme(e.target.value); act("/api/theme", { theme: e.target.value }); };
  $("#g-cancel").onclick = () => act("/api/github/cancel", {});
  $("#g-copy").onclick = () => navigator.clipboard && navigator.clipboard.writeText($("#g-code").textContent);
  $("#devmode").onchange = (e) => act("/api/devmode", { on: e.target.checked });
  $("#new-id").onsubmit = async (e) => {
    e.preventDefault();
    if (await act("/api/identities", { name: $("#new-name").value, kind: $("#new-kind").value }, $("#id-error"))) { $("#new-name").value = ""; $("#new-id").hidden = true; }
  };
  $("#exit-btn").onclick = () => {
    const s = state || {}, lc = s.lifecycle;
    $("#x-note").textContent = (s.exit && s.exit.note) || "";
    $("#x-identity-row").hidden = !(s.verified && lc);
    $("#x-identity-why").textContent = lc && lc.ownership === "MANAGED" ? "(Processor started it; it stops with the workspace)" : "(you started it separately; it keeps running unless you tick this)";
    $("#x-identity").checked = !!(lc && lc.ownership === "MANAGED");
    $("#m-exit").hidden = false;
  };
  $("#x-cancel").onclick = () => ($("#m-exit").hidden = true);
  document.querySelectorAll('input[name="exit"]').forEach((r) => (r.onchange = () => { $("#x-identity").disabled = r.value === "keep" && r.checked; if (r.value === "keep" && r.checked) $("#x-identity").checked = false; }));
  $("#x-ok").onclick = async () => {
    const choice = document.querySelector('input[name="exit"]:checked').value;
    exiting = true;
    try {
      const r = await api("POST", "/api/exit", { choice, stop_identity: choice === "shutdown" && $("#x-identity").checked });
      $("#m-exit").hidden = true; $("#m-done").hidden = false;
      $("#d-note").textContent = (r.features ? r.features + ". " : "") + "Identity: " + r.identity + ".";
    } catch (e) { exiting = false; alert(e.message); }
  };
  (async () => { if (await signIn()) { await refresh(); schedule(); } })();
})();
