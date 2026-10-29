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
  let chats = {};           // installation|identity -> {messages, note, err} (this page only; never stored by Processor)
  let chatSub = {};         // installation -> identity chosen for talking
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
    let h = `<div class="grp"><div class="grp-h">Developer</div>` + item("platform:all", "Developer Platform", "home", "", s.verified ? "ok" : "warn") + `</div>`;
    h += `<div class="grp"><div class="grp-h">Administrator</div>`;
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

    if (!sel || (sel.type === "entity" && !ids.find((e) => e.id === sel.id)) || (sel.type === "feature" && !findFeature(s, sel.id))) sel = { type: "platform", id: "all" };
    const d = $("#detail");
    if (sel.type === "platform") return renderPlatform(s, d);
    if (!sel) return set(d, `<div class="empty"><h2>Choose an identity</h2><p>The Admin, your companions and every installed feature are identities here. Each one starts with no access to anything.</p></div>`);
    if (sel.type === "admin") return renderAdmin(s, d);
    if (sel.type === "entity") return renderEntity(s, d, ids.find((e) => e.id === sel.id));
    if (sel.type === "feature") return renderFeature(s, d, findFeature(s, sel.id));
    if (sel.type === "library") return renderLibrary(s, d);
  }

  // ----- Developer Platform: the whole system on one screen -----
  // Every part with the evidence behind its state, who may use it, and the
  // controls that make sense now. It only shows and does what this paired
  // session can already do; installs and permissions still go to the Admin.
  function modelBadge(st) {
    const m = st && st.model;
    if (!m) return "";
    const ab = (m.abilities || []).filter((x) => x !== "completion");
    return badge("MODEL: " + String(m.state || "?").replace(/_/g, " ") + (m.state === "REACHABLE" && m.name ? " · " + m.name : "") + (ab.length ? " · " + ab.join(" · ") : ""),
      m.state === "REACHABLE" ? "ok" : m.state === "NOT_CONFIGURED" || m.state === "CHECKING" ? "warn" : "bad", m.detail || "");
  }
  function pulseBadge(st) {
    const p = st && st.pulse;
    if (!p) return "";
    const c = p.counters || {}, l = p.last || {};
    const word = p.stage === "born" ? "PULSE: born" : p.stage === "damaged" ? "PULSE: record damaged" : "PULSE: " + p.stage;
    return badge(word, p.stage === "born" ? "ok" : p.stage === "damaged" ? "bad" : "warn",
      `every ${p.every_s} s · ${c.ticks || 0} ticks: ${c.skipped || 0} skipped, ${c.thought || 0} thought, ${c.waiting_for_model || 0} waiting for a model` + (l.outcome ? ` · last: ${l.outcome} (${l.ms} ms) ${l.why || ""}` : ""));
  }
  function svcEvidence(v) {
    const rt = v.runtime, st = (rt.health && rt.health.status) || {};
    if (rt.state !== "RUNNING") return "";
    let h = badge("RUNNING", "ok", "Processor verified this launch's signed status (pid " + rt.pid + ")");
    if (st.entrance) h += badge(st.entrance.connected ? "CONNECTED" : "NOT CONNECTED", st.entrance.connected ? "ok" : "bad", (st.authority || {}).detail || "")
      + badge(st.entrance.ready ? "READY" : "NOT READY", st.entrance.ready ? "ok" : "warn");
    else h += badge(rt.health && rt.health.ready ? "READY" : "NOT READY", rt.health && rt.health.ready ? "ok" : "warn", "the service's own signed health");
    return h + modelBadge(st) + pulseBadge(st);
  }
  function renderPlatform(s, d) {
    const fs = features(s).filter((v) => v.record.state === "INSTALLED"), lib = s.library || [];
    const ids = s.identities || [], a = s.admin || {}, g = s.github || {}, c = s.connection || {};
    const name = (eid) => { const e = ids.find((x) => x.id === eid); return e ? e.name : eid; };
    const running = fs.filter((v) => v.runtime.state === "RUNNING");
    const nm = (r) => r.name.replace(" (TEST FIXTURE)", "");
    let h = `<div class="dhead"><div class="typed">DEVELOPER PLATFORM</div><h1>Your Brain system</h1>
      <p class="muted">Every part, the evidence behind its state, who may use it, and what you can do now. Installs and permissions are always approved by the Admin in Identity's own page.</p></div>`;
    // system strip
    const tile = (label, word, cls, sub) => `<div class="tile"><div class="t-l">${esc(label)}</div><div class="t-w"><span class="dot ${cls}"></span> ${esc(word)}</div><div class="t-s muted small">${sub}</div></div>`;
    h += `<div class="tiles">` + tile("Identity", s.verified ? "Connected" : c.state === "AUTHORITY_UNAVAILABLE" ? "Unavailable" : "Not connected", s.verified ? "ok" : "bad", esc(c.identity_system_id ? short(c.identity_system_id, 14) : c.detail || ""))
      + tile("Admin", a.state === "ENROLLED" ? "Established" : a.state === "DAMAGED" ? "Damaged" : "Not established", a.state === "ENROLLED" ? "ok" : a.state === "DAMAGED" ? "bad" : "warn", g.state === "VERIFIED" ? "GitHub: " + esc(g.account ? g.account.login : "signed in") : "GitHub: " + esc((g.state || "signed out").toLowerCase().replace(/_/g, " ")))
      + tile("Thought Processor", "Running", "ok", "v" + esc((s.processor || {}).version || "") + " · pid " + esc((s.processor || {}).pid || ""))
      + tile("Services", running.length + " of " + fs.length + " running", running.length ? "ok" : "idle", esc(fs.length ? fs.map((v) => nm(v.record)).join(" · ") : "none installed yet")) + `</div>`;
    // the ONE next step while setting up
    const next = !s.verified ? "Connect Processor to Identity (top of the page)."
      : a.state !== "ENROLLED" ? `Establish the Admin Identity. <button class="btn small primary" data-sel="admin:admin">Admin →</button>`
      : !ids.length ? `Add an identity for yourself (<strong>+ Identity</strong> on the left).`
      : !fs.length ? `Install your first service below (each install is approved by the Admin).`
      : !running.length ? `Start a service below, or <strong>Start all</strong>.` : "";
    if (next) h += `<div class="next"><strong>Next:</strong> ${next}</div>`;
    // services
    h += `<div class="row between"><h3>Services</h3><div>${fs.length ? `<button class="btn small" data-startall="1" ${s.verified ? "" : "disabled"}>Start all</button> <button class="btn small" data-stopall="1">Stop all</button>` : ""}</div></div>`;
    h += fs.length ? `<table class="tbl plat"><tr><th>Service</th><th>State and evidence</th><th>Who may use it</th><th>Last self-test</th><th></th></tr>${fs.map((v) => {
      const r = v.record, rt = v.runtime, ins = r.installation_id, iv = v.identity, [word, cls] = phaseOf(rt.state);
      const busy = rt.state === "STARTING" || rt.state === "RESTARTING" || rt.state === "STOPPING";
      const gr = ((iv && iv.grants) || []).filter((x) => x.status === "ACTIVE");
      const who = [...new Set(gr.map((x) => name(x.grant.subject_identity_id)))];
      const t = v.self_test;
      return `<tr><td><button class="linklike" data-sel="feature:${esc(ins)}"><strong>${esc(nm(r))}</strong></button><div class="muted small">${esc(r.feature_id)} v${esc(r.version)} · ${mono(short(ins, 12))}</div></td>
        <td><div><span class="dot ${cls}"></span> ${esc(word)}${rt.state !== "RUNNING" && rt.last_exit ? ` <span class="muted small">last exit: ${esc(rt.last_exit.reason || "")}</span>` : ""}</div><div class="badges">${svcEvidence(v)}${iv ? "" : badge("Identity can't confirm", "warn")}${v.files && v.files !== "OK" ? badge(v.files, "bad") : ""}</div></td>
        <td class="small">${!iv ? `<span class="warn">can't confirm</span>` : who.length ? esc(who.join(", ")) + ` <span class="muted">(${gr.length} permission${gr.length === 1 ? "" : "s"})</span>` : `<span class="muted">nobody yet</span>`}</td>
        <td class="small">${t ? badge(t.result, t.result === "PASS" ? "ok" : "bad") + ` <span class="muted">${esc(new Date(t.at).toLocaleTimeString())}${rt.launch_id && t.launch_id && t.launch_id !== rt.launch_id ? " (earlier launch)" : ""}</span>` : `<span class="muted">not run</span>`}</td>
        <td class="acts">${rt.state === "RUNNING" ? `<button class="btn small" data-stop="${esc(ins)}">Stop</button> <button class="btn small" data-selftest="${esc(ins)}">Self-test</button>`
          : busy ? `<button class="btn small" disabled>${esc(word)}</button>` : `<button class="btn small primary" data-start="${esc(ins)}" ${iv ? "" : "disabled"}>Start</button>`} <button class="btn ghost small" data-sel="feature:${esc(ins)}">Open →</button></td></tr>`;
    }).join("")}</table>` : `<p class="muted">No services installed yet.</p>`;
    // what can be added
    const avail = lib.filter((x) => !x.blocked && !(x.installed_as || []).length);
    if (avail.length) h += `<h3>Ready to install</h3><div class="mini-list">${avail.map((x) => `<div class="mini"><strong>${esc(x.name)}</strong> ${badge("pinned artifact", "info", x.trust_basis || "")} <span class="muted small">${esc((x.capabilities || []).map((c) => c.id).join(", "))}</span> <button class="btn small primary" data-review="${esc(x.feature_id)}" ${s.verified && a.state === "ENROLLED" ? "" : "disabled"}>Install…</button></div>`).join("")}</div>`;
    const blocked = lib.filter((x) => x.blocked);
    if (blocked.length) h += `<p class="small muted">${blocked.length} package(s) in trusted folders are BLOCKED (not the pinned reviewed artifact): ${esc(blocked.map((x) => x.feature_id || x.dir).join(", "))}.</p>`;
    // recent activity: outcomes only, never what was said
    const oc = { COMPLETED: "ok", DENIED_BY_IDENTITY: "bad", DENIED_BY_FEATURE: "bad", AUTHORITY_UNAVAILABLE: "warn", REQUEST_UNCERTAIN: "warn", NOT_RUNNING: "warn", TICKET_REJECTED: "bad", ERROR: "bad" };
    const reqs = fs.flatMap((v) => (v.requests || []).map((q) => [nm(v.record), q])).sort((x, y) => (x[1].at < y[1].at ? 1 : -1)).slice(0, 15);
    h += `<h3>Recent requests</h3>` + (reqs.length ? `<table class="tbl small"><tr><th>When</th><th>Service</th><th>For</th><th>Capability</th><th>Outcome</th></tr>${reqs.map(([n, q]) =>
      `<tr><td>${esc(new Date(q.at).toLocaleTimeString())}</td><td>${esc(n)}</td><td>${esc(name(q.subject_identity_id))}</td><td>${mono(q.capability)}</td><td>${badge(q.outcome.replace(/_/g, " "), oc[q.outcome] || "")}</td></tr>`).join("")}</table>
      <p class="small muted">Outcomes only: Processor keeps no copy of what a service answered.</p>` : `<p class="muted small">None yet this session.</p>`);
    set(d, h);
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
    BAD_INPUT: "The service was allowed to run the request but couldn't use the input. Nothing changed.",
    MODEL_REFUSED: "TASS refused its model settings: the model must be on this computer (http://127.0.0.1:port). Nothing was sent or saved.",
    MODEL_UNAVAILABLE: "TASS couldn't get an answer from the local model (is it running?). Your message was not saved; send it again.",
    MODEL_TIMEOUT: "The local model took too long to answer. Your message was not saved; try again or use a smaller model.",
    RATE_LIMITED: "Too many messages in one minute. Nothing was saved; wait a moment.",
    STORE_DAMAGED: "This identity's saved conversation can't be read. It was left untouched; nothing was done.",
    NO_MODEL_FOR_IMAGE: "TASS can't look at a picture without a local model. Choose a vision model (e.g. qwen3.5:4b) in “TASS's model” below. Nothing was saved.",
    MODEL_CANT_SEE: "The chosen model can't see pictures. Choose a vision model such as qwen3.5:4b. Nothing was saved.",
    BAD_IMAGE: "The picture was refused: only PNG, JPEG or WebP up to 4 MB. Nothing was sent." };
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
        + badge(st.entrance.ready ? "READY" : "NOT READY", st.entrance.ready ? "ok" : "warn")
        + modelBadge(st) + pulseBadge(st) + `</div>`
        + (st.pulse ? `<p class="small muted">Pulse: ${esc(st.pulse.stage)}${st.pulse.born_at ? " · born " + esc(new Date(st.pulse.born_at).toLocaleString()) : ""} · ${esc((st.pulse.counters || {}).ticks || 0)} ticks${st.pulse.last && st.pulse.last.outcome ? " · last tick: " + esc(st.pulse.last.outcome) + " (" + esc(st.pulse.last.ms) + " ms)" + (st.pulse.last.why ? " — " + esc(st.pulse.last.why) : "") : ""}</p>` : "")
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
      else if (caps.some((c) => /\.chat\.send$/.test(c.id))) h += `<div class="next"><strong>Next:</strong> talk to it below ↓</div>`;
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
        <div class="pane" id="pane-overview"><div id="ov-run"></div><div id="ov-chat"><div id="chat-head"></div><div id="chat-log" class="chatlog" aria-live="polite"></div><div id="chat-form"></div></div><div id="ov-model"></div><div id="ov-self"></div><div id="ov-install"></div></div><div class="pane" id="pane-connections"><div id="fx-grants"></div><div id="fx-grant-form"></div><div id="fx-action"></div><div id="fx-action-result"></div><div id="fx-requests"></div></div>
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
    set($("#ov-run"), gone ? "" : `<h3>Run</h3>` + runCard(s, v, iv));
    renderChat(s, v, iv);
    renderSelf(s, v, iv);
    set($("#ov-install"), `<h3>Installation</h3>` + kv([["Installation ID", mono(ins) + ` <span class="muted small">durable · minted by Identity · never reused</span>`], ["Feature", mono(r.feature_id) + " v" + esc(r.version)],
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
  // Talk to a companion feature (one that offers *.chat.send). Every message is
  // its own Identity-checked request for the chosen identity; what was said
  // lives only in the feature's own private file and on this page.
  function renderChat(s, v, iv) {
    const r = v.record, ins = r.installation_id, rt = v.runtime, caps = r.capabilities || [];
    const send = caps.find((c) => /\.chat\.send$/.test(c.id)), hist = caps.find((c) => /\.chat\.history$/.test(c.id));
    const head = $("#chat-head"), logEl = $("#chat-log"), form = $("#chat-form");
    if (!send || r.state !== "INSTALLED") { set(head, ""); set(logEl, ""); form.innerHTML = ""; form.dataset.key = ""; logEl.hidden = true; return; }
    const grants = ((iv && iv.grants) || []).filter((g) => g.status === "ACTIVE");
    const can = (cap, sub) => !!cap && grants.some((g) => g.grant.capability === cap.id && g.grant.subject_identity_id === sub);
    // Anyone allowed to send or to read their history gets the panel; Send works only with the send permission.
    const subs = [...new Set(grants.filter((g) => g.grant.capability === send.id || (hist && g.grant.capability === hist.id)).map((g) => g.grant.subject_identity_id))];
    const senders = subs.filter((x) => can(send, x));
    const name = (eid) => { const e = (s.identities || []).find((x) => x.id === eid); return e ? e.name : eid; };
    if (!subs.includes(chatSub[ins])) chatSub[ins] = subs[0] || "";
    const sub = chatSub[ins], key = ins + "|" + sub, c = chats[key] || {};
    const running = rt.state === "RUNNING", st = (rt.health && rt.health.status) || {}, m = st.model;
    const title = esc((r.name || "it").split(" (")[0]);
    let h = `<h3>Talk to ${title}</h3>`;
    if (m) h += `<p class="small">${modelBadge(st)} ${esc(m.detail || "")}</p>`;
    if (st.privacy) h += `<p class="small muted">${esc(st.privacy)}. Processor passes each message through and keeps only the outcome; Identity sees only a fingerprint (SHA-256) of it.</p>`;
    if (!running) h += `<p class="small muted">Start it first (above).</p>`;
    else if (subs.length && !can(send, sub)) h += `<p class="small warn">${esc(name(sub))} may read the conversation but not send: ${mono(send.id)} isn't allowed (or was revoked).</p>`;
    if (running && !subs.length) h += `<div class="next"><strong>Next:</strong> nobody may talk to it yet. Ask the Admin to allow ${mono(send.id)} for you. <button class="btn small" data-tab="connections">Ask the Admin for a permission →</button></div>`;
    set(head, h);
    logEl.hidden = !(running && subs.length);
    const msgs = c.messages || [];
    set(logEl, (msgs.length ? msgs.map((x) => `<div class="msg ${x.role === "tass" ? "them" : "you"}"><div class="who">${x.role === "tass" ? title : esc(name(sub))} · ${esc(new Date(x.at).toLocaleTimeString())}${x.by ? ` <span class="muted" title="${esc(x.by)}">· ${esc(x.by.split(" (")[0])}</span>` : ""}</div><div class="txt">${esc(x.text)}</div>${x.attachment ? `<div class="small muted">📎 ${esc(x.attachment.name)} · ${esc(Math.max(1, Math.round(x.attachment.bytes / 1024)))} KB · kept privately by TASS</div>` : ""}${x.thinking ? `<details class="small"><summary>Thinking (${esc(x.thinking_label || "the model's reasoning")})</summary><pre>${esc(x.thinking)}</pre></details>` : ""}${x.role === "you" && x.answered === false ? `<div class="small muted">saved · not answered (no local model)</div>` : ""}</div>`).join("")
      : `<p class="small muted">${can(hist, sub) ? "No conversation loaded yet." : "Your conversation shows here after you send a message."}</p>`)
      + (c.note ? `<p class="small"><strong>${esc(c.note)}</strong></p>` : "") + (c.err ? resultBox(c.err) : ""));
    if (logEl.dataset.scrolled !== String(msgs.length)) { logEl.dataset.scrolled = String(msgs.length); logEl.scrollTop = logEl.scrollHeight; }
    // The form is built once per (installation, allowed identities, running) so redraws never wipe what you are typing.
    const fkey = ins + "|" + subs.join(",") + "|" + senders.join(",") + "|" + running;
    if (form.dataset.key !== fkey) {
      form.dataset.key = fkey;
      form.innerHTML = !(running && subs.length) ? "" : `<form class="frm chatform" id="chat-f">
        <div class="frow">${subs.length > 1 ? `<label>Talking as<select id="chat-sub">${subs.map((x) => `<option value="${esc(x)}" ${x === sub ? "selected" : ""}>${esc(name(x))}</option>`).join("")}</select></label>` : `<span class="small muted">Talking as <strong>${esc(name(sub))}</strong></span>`}</div>
        <textarea id="chat-in" maxlength="1000" rows="3" placeholder="Say something to ${title}… (or attach a picture and ask about it)"></textarea>
        <div class="row"><input type="file" id="chat-file" accept="image/png,image/jpeg,image/webp" hidden><button class="btn small" type="button" id="chat-attach">Attach picture</button> <span class="small" id="chat-att"></span>
        <button class="btn primary" type="submit" id="chat-send" ${senders.includes(sub) ? "" : "disabled title=\"Needs the send permission\""}>Send</button>${hist ? ` <button class="btn ghost small" type="button" id="chat-load">Load conversation</button>` : ""}<span class="small muted" id="chat-busy"></span></div></form>`;
      const f = $("#chat-f");
      if (f) {
        const cs = $("#chat-sub");
        if (cs) cs.onchange = () => { chatSub[ins] = cs.value; render(); };
        const load = async () => {
          const who = chatSub[ins], k = ins + "|" + who;
          if (!can(hist, who)) { chats[k] = Object.assign(chats[k] || {}, { note: "Loading needs its own permission: " + hist.id + " for " + name(who) + ".", err: null }); return render(); }
          const j = await act(`/api/installations/${ins}/action`, { subject: who, capability: hist.id, input: "" });
          const res = j && j.result, out = res && res.observation && res.observation.output;
          chats[k] = out ? { messages: out.conversation || [], note: out.headline, err: null } : Object.assign(chats[k] || {}, { err: res || null });
          render();
        };
        const lb = $("#chat-load");
        if (lb) lb.onclick = load;
        $("#chat-in").onkeydown = (e) => { if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); f.requestSubmit(); } };
        let picked = null;
        const showPick = () => { $("#chat-att").innerHTML = picked ? `📎 ${esc(picked.name)} <button class="btn ghost small" type="button" id="chat-unattach">remove</button>` : ""; const u = $("#chat-unattach"); if (u) u.onclick = () => { picked = null; $("#chat-file").value = ""; showPick(); }; };
        $("#chat-attach").onclick = () => $("#chat-file").click();
        $("#chat-file").onchange = () => {
          const file = $("#chat-file").files[0];
          if (!file) return;
          if (!["image/png", "image/jpeg", "image/webp"].includes(file.type) || file.size > 4 * 1024 * 1024) { toast("Only PNG, JPEG or WebP pictures up to 4 MB.", "bad"); $("#chat-file").value = ""; return; }
          const rd = new FileReader();
          rd.onload = () => { picked = { name: file.name.replace(/[^A-Za-z0-9_.() -]/g, "_").slice(0, 128), mime: file.type, base64: String(rd.result).split(",")[1] || "" }; showPick(); };
          rd.readAsDataURL(file);
        };
        f.onsubmit = async (e) => {
          e.preventDefault();
          const ta = $("#chat-in"), text = ta.value.trim(), b = $("#chat-send"), who = chatSub[ins], k = ins + "|" + who;
          if (!text && !picked) return;
          b.disabled = true; $("#chat-busy").textContent = " asking Identity, then " + (r.name || "it").split(" (")[0] + "…";
          const body = { subject: who, capability: send.id, input: text };
          if (picked) body.attachment = picked;
          const j = await act(`/api/installations/${ins}/action`, body);
          b.disabled = false; $("#chat-busy").textContent = "";
          const res = j && j.result, out = res && res.outcome === "COMPLETED" && res.observation && res.observation.output;
          if (out) { chats[k] = { messages: out.conversation || [], note: out.reply ? "" : out.headline, err: null }; ta.value = ""; picked = null; $("#chat-file").value = ""; showPick(); }
          else chats[k] = Object.assign(chats[k] || {}, { err: res || null, note: "" });
          render(); ta.focus();
        };
      }
      // Load the saved conversation once, if that is allowed.
      if (running && sub && can(hist, sub) && !chats[key]) { chats[key] = { messages: [], note: "Loading…" }; const lb2 = $("#chat-load"); if (lb2) lb2.click(); }
    }
  }
  // A companion's own pulse: its seed thoughts, its birth, and what its local
  // model reasoned about itself. Reading it is its own Identity permission.
  let selfViews = {};      // installation -> last answer (this page only)
  function renderModelBox(s, v, iv) {
    const r = v.record, ins = r.installation_id, caps = r.capabilities || [], el = $("#ov-model");
    const cap = caps.find((c) => /\.model\.set$/.test(c.id));
    if (!el) return;
    if (!cap || r.state !== "INSTALLED") { el.innerHTML = ""; el.dataset.key = ""; return; }
    const grants = ((iv && iv.grants) || []).filter((g) => g.status === "ACTIVE" && g.grant.capability === cap.id);
    const subs = [...new Set(grants.map((g) => g.grant.subject_identity_id))];
    const running = v.runtime.state === "RUNNING";
    const key = ins + "|" + subs.join(",") + "|" + running;
    const title = esc((r.name || "it").split(" (")[0]);
    if (el.dataset.key === key) return;
    el.dataset.key = key;
    el.innerHTML = `<h3>${title}'s model</h3>` + (!running ? `<p class="small muted">Start it first.</p>`
      : !subs.length ? `<p class="small muted">Choosing the model needs the ${mono(cap.id)} permission. <button class="btn small" data-tab="connections">Ask the Admin →</button></p>`
      : `<form class="frm" id="model-f"><div class="frow"><label class="grow">Ollama model on this computer<input id="model-in" value="qwen3.5:4b" maxlength="100" autocomplete="off"></label>
         <label><input type="checkbox" id="model-think"> show thinking in chat (slower)</label></div>
         <div class="row"><button class="btn primary" type="submit">Use this model</button> <button class="btn ghost small" type="button" id="model-none">No model</button> <span class="small muted">Only Ollama at http://127.0.0.1:11434 — nothing outside this computer.</span></div></form>`);
    const f = $("#model-f");
    if (!f) return;
    const go = async (input) => {
      const j = await act(`/api/installations/${ins}/action`, { subject: subs[0], capability: cap.id, input });
      const res = j && j.result, out = res && res.observation && res.observation.output;
      if (out) toast(out.headline, "ok"); else if (res) toast(res.outcome.replace(/_/g, " ") + " — " + (explain(res) || res.detail), "bad");
    };
    f.onsubmit = (e) => { e.preventDefault(); const name = $("#model-in").value.trim(); if (!name) return; go($("#model-think").checked ? JSON.stringify({ model: name, chat_think: true }) : name); };
    $("#model-none").onclick = () => go("none");
  }
  function renderSelf(s, v, iv) {
    renderModelBox(s, v, iv);
    const r = v.record, ins = r.installation_id, caps = r.capabilities || [], el = $("#ov-self");
    const cap = caps.find((c) => /\.self\.read$/.test(c.id));
    if (!cap || r.state !== "INSTALLED") return set(el, "");
    const grants = ((iv && iv.grants) || []).filter((g) => g.status === "ACTIVE" && g.grant.capability === cap.id);
    const name = (eid) => { const e = (s.identities || []).find((x) => x.id === eid); return e ? e.name : eid; };
    const title = esc((r.name || "it").split(" (")[0]);
    let h = `<h3>${title}'s pulse</h3>`;
    if (!grants.length) h += `<p class="small muted">Reading its first thoughts and reflections needs the ${mono(cap.id)} permission. <button class="btn small" data-tab="connections">Ask the Admin →</button></p>`;
    else h += `<div class="row">${grants.map((g) => `<button class="btn small" data-selfread="${esc(ins)}" data-sub="${esc(g.grant.subject_identity_id)}" data-cap="${esc(cap.id)}">Read ${title}'s pulse (as ${esc(name(g.grant.subject_identity_id))})</button>`).join(" ")}</div>`;
    const x = selfViews[ins];
    if (x && x.err) h += resultBox(x.err);
    else if (x) {
      const o = x.out;
      h += `<div class="selfview"><p><strong>${esc(o.headline)}</strong></p>
        <ol class="seeds">${(o.seeds || []).map((sd) => `<li><strong>${esc(sd.text)}</strong> <span class="muted small">${esc(new Date(sd.at).toLocaleTimeString())} · ${esc(sd.source)}</span></li>`).join("")}</ol>
        ${o.born ? `<p class="small">Born ${esc(new Date(o.born.at).toLocaleString())} · first question: <strong>${esc(o.born.first_question)}</strong></p>` : ""}
        ${(o.reflections || []).length ? (o.reflections.slice().reverse().map((rf) => `<div class="reflection"><div class="small muted">${esc(new Date(rf.at).toLocaleString())} · ${esc(rf.model)} · ${esc(rf.took_s)} s</div>
          <div><strong>${esc(rf.question)}</strong> ${esc(rf.answer || "(no answer within the thinking limit)")}</div>
          ${rf.thinking ? `<details class="small"><summary>Thinking (${esc(rf.label)})</summary><pre>${esc(rf.thinking)}</pre></details>` : `<div class="small muted">no reasoning trace from this model</div>`}</div>`).join(""))
          : `<p class="small muted">No reflections yet${o.pulse && o.pulse.last && o.pulse.last.why ? " — " + esc(o.pulse.last.why) : ""}.</p>`}
        <p class="small muted">${esc(o.honesty || "")}</p></div>`;
    }
    set(el, h);
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
    else if (ds.startall || ds.stopall) {
      t.disabled = true;
      for (const v of features(state).filter((x) => x.record.state === "INSTALLED")) {
        const st = v.runtime.state;
        if (ds.startall && st !== "RUNNING" && st !== "STARTING" && v.identity) await act(`/api/installations/${v.record.installation_id}/start`, {});
        if (ds.stopall && (st === "RUNNING" || st === "STARTING")) await act(`/api/installations/${v.record.installation_id}/stop`, {});
      }
      t.disabled = false;
    }
    else if (ds.selfread) {
      t.disabled = true;
      const j = await act(`/api/installations/${ds.selfread}/action`, { subject: ds.sub, capability: ds.cap, input: "" });
      const res = j && j.result, out = res && res.outcome === "COMPLETED" && res.observation && res.observation.output;
      selfViews[ds.selfread] = out ? { out } : { err: res || null };
      render();
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
