"use strict";
// Brain Map, Screen 1 (#34 D001–D009): read-only Service Map.
// Every node is a real installation (or the Processor / paired Identity / a
// model endpoint a feature reports). Every solid line carries fresh signed
// evidence and a timestamp. Nothing here changes a service: dragging only
// saves screen positions (POST /api/map/layout, presentation data).
(() => {
  const $ = (q) => document.querySelector(q);
  const NS = "http://www.w3.org/2000/svg";
  const esc = (s) => String(s == null ? "" : s).replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
  const mono = (s) => `<span class="mono">${esc(s)}</span>`;
  const FRESH_MS = 45000;
  let csrf = sessionStorage.getItem("tp_csrf") || "";
  let state = null, diag = null, nodes = [], edges = [], pos = {}, sel = null, logScope = null;
  let view = { x: 0, y: 0, k: 1 }, saveTimer = 0;

  // ---- session (#33: the token of the session this window's cookie names) ----
  async function refreshCsrf() {
    try {
      const r = await fetch("/api/session/csrf", { headers: { "X-TP-Want-CSRF": "1" } });
      const j = await r.json().catch(() => ({}));
      if (r.ok && j.csrf) { csrf = j.csrf; sessionStorage.setItem("tp_csrf", csrf); return true; }
    } catch (_) {}
    return false;
  }
  async function api(method, path, body, retried) {
    const h = {};
    if (body !== undefined) { h["Content-Type"] = "application/json"; h["X-TP-CSRF"] = csrf; }
    const r = await fetch(path, { method, headers: h, body: body === undefined ? undefined : JSON.stringify(body) });
    const j = await r.json().catch(() => ({}));
    if (body !== undefined && r.status === 403 && j.error === "missing or wrong anti-forgery token" && !retried && await refreshCsrf()) return api(method, path, body, true);
    if (!r.ok) { const e = new Error(j.error || "HTTP " + r.status); e.status = r.status; throw e; }
    return j;
  }

  // ---- time & evidence helpers ----
  const toMs = (t) => { if (t == null || t === "") return NaN; if (typeof t === "number") return t < 1e12 ? t * 1000 : t; return new Date(t).getTime(); };
  const ago = (t) => { const ms = toMs(t); if (isNaN(ms)) return "unknown"; const d = (Date.now() - ms) / 1000; return d < 90 ? Math.max(0, Math.round(d)) + " s ago" : d < 5400 ? Math.round(d / 60) + " min ago" : new Date(ms).toLocaleString(); };
  const fresh = (t) => { const ms = toMs(t); return !isNaN(ms) && Date.now() - ms < FRESH_MS; };
  const unk = (v) => (v == null || v === "" ? `<span class="muted">Not reported</span>` : esc(v));

  // ---- build the graph from real state ----
  function build(s) {
    const N = [], E = [];
    const conn = s.connection || {}, proc = s.processor || {};
    const pkey = "processor:" + (proc.processor_id || "this");
    N.push({ key: pkey, kind: "processor", group: "Control", label: "Thought Processor", sub: "this app · v" + (proc.version || "?"),
      status: { installed: "n/a (this app)", running: "RUNNING (serving this page)", authorized: "n/a", verified: "n/a" }, raw: { processor: proc } });
    let ikey = null;
    if (conn.identity_system_id) {
      ikey = "identity:" + conn.identity_system_id;
      const ok = conn.state === "VERIFIED";
      N.push({ key: ikey, kind: "identity", group: "Authority", label: "Identity", sub: conn.identity_system_id,
        status: { installed: "separate service (not a feature)", running: ok ? "answering (signed)" : esc(conn.state || "UNKNOWN"), authorized: "the authority itself", verified: ok && fresh(conn.last_ok) ? "VERIFIED" : "UNKNOWN" },
        raw: { connection: conn } });
      const st = !ok ? "FAILED" : fresh(conn.last_ok) ? "VERIFIED" : "STALE";
      E.push({ key: "e:" + pkey + "|" + ikey, from: pkey, to: ikey, type: "pairing: Identity's signed status answers", kind: "verified", st,
        last: conn.last_ok, why: ok ? "" : (conn.detail || conn.state), source: "Processor's paired, signed exchange with Identity (identity/2)" });
    }
    const people = Object.fromEntries((s.identities || []).map((e) => [e.id, e.name]));
    for (const v of (s.installations || [])) {
      const r = v.record || {};
      if (r.state !== "INSTALLED") continue;
      const rt = v.runtime || {}, h = rt.health || {}, fst = h.status || {}, iv = v.identity || null;
      const key = r.installation_id;
      const grants = ((iv && iv.grants) || []).filter((g) => g.status === "ACTIVE");
      const reqs = v.requests || [], lastOk = reqs.find((q) => q.outcome === "COMPLETED");
      const running = rt.state === "RUNNING";
      const ctlFresh = running && h.ready && fresh(h.checked_at);
      N.push({ key, kind: "feature", group: "Installed features", label: (r.name || r.feature_id || "feature").split(" (")[0], sub: (r.feature_id || "") + " v" + (r.version || "?"),
        status: { installed: "INSTALLED " + (r.installed_at ? "· " + new Date(r.installed_at).toLocaleDateString() : ""),
          running: esc(rt.state || "UNKNOWN") + (rt.pid ? " · pid " + rt.pid : ""),
          authorized: iv ? `${grants.length} active grant(s) in Identity` : "UNKNOWN (Identity can't confirm right now)",
          verified: ctlFresh ? "VERIFIED (signed status " + ago(h.checked_at) + ")" : running ? "UNKNOWN (no fresh signed status)" : "not running" },
        raw: { record: r, runtime: rt, identity: iv, requests: reqs, people } });
      if (running) {
        const st = h.error ? "FAILED" : ctlFresh ? "VERIFIED" : "STALE";
        E.push({ key: "e:" + pkey + "|" + key, from: pkey, to: key, type: "control: launcher-signed /v1/status", kind: "verified", st, last: h.checked_at,
          why: h.error || (st === "STALE" ? "no signed status in the last " + FRESH_MS / 1000 + " s" : ""), source: "Processor supervisor (status answer signed with this launch's key)",
          extra: lastOk ? `last completed request ${mono(lastOk.request_id)} ${ago(lastOk.at)} · Identity decision ${mono((lastOk.decision_id || "").slice(0, 18))}` : "no completed request recorded in this run" });
      }
      const au = fst.authority;
      if (running && au && ikey) {
        const st = au.state === "CONNECTED" && fresh(au.checked_at) ? "VERIFIED" : au.state === "CONNECTED" ? "STALE" : "FAILED";
        E.push({ key: "e:" + key + "|" + ikey, from: key, to: ikey, type: "live authorization: the feature's own signed check of Identity", kind: "verified", st, last: au.checked_at,
          why: st === "VERIFIED" ? "" : (au.detail || au.state), source: "the feature's status (" + esc(r.feature_id) + "), carried in its launcher-signed answer" });
      }
      const md = fst.model;
      if (running && md && md.url) {
        const mkey = "endpoint:" + md.url;
        if (!N.find((n) => n.key === mkey)) N.push({ key: mkey, kind: "endpoint", group: "Model endpoints", label: md.name || "model endpoint", sub: (md.kind || "") + " · " + md.url,
          status: { installed: "not an installation: reported by " + esc(r.feature_id), running: esc(md.state || "UNKNOWN"), authorized: "n/a", verified: md.state === "REACHABLE" && fresh(md.checked_at) ? "REACHABLE (feature's probe)" : "UNKNOWN" },
          raw: { model: md, reported_by: key } });
        const st = md.state === "REACHABLE" && fresh(md.checked_at) ? "VERIFIED" : md.state === "REACHABLE" ? "STALE" : "FAILED";
        E.push({ key: "e:" + key + "|" + mkey, from: key, to: mkey, type: "model reachability: the feature's loopback probe", kind: "verified", st, last: md.checked_at,
          why: st === "VERIFIED" ? "" : (md.detail || md.state), source: "the feature's own probe, reported in its signed status" });
      }
      if (grants.length && !lastOk) {
        E.push({ key: "e:auth:" + key, from: pkey, to: key, type: "authorized, not observed: " + grants.length + " active grant(s), no completed request in this run", kind: "authorized", st: "AUTHORIZED",
          last: null, why: "", source: "Identity's grant records (a grant is not evidence of traffic)" });
      }
    }
    return { N, E };
  }

  // ---- layout ----
  const ORDER = ["Control", "Authority", "Installed features", "Model endpoints"];
  function autoArrange(all) {
    const p = {};
    const cols = ORDER.map((g) => nodes.filter((n) => n.group === g));
    cols.forEach((list, ci) => list.forEach((n, i) => {
      if (all || !pos[n.key]) p[n.key] = { x: 120 + ci * 290, y: 120 + i * 150 + (ci % 2 ? 70 : 0) };
    }));
    // TASS in the visual centre when it is genuinely installed
    const tass = nodes.find((n) => n.kind === "feature" && /tass/i.test((n.raw.record || {}).feature_id || ""));
    if (tass && (all || !pos[tass.key])) p[tass.key] = { x: 120 + 2 * 290 - 145, y: 180 };
    return p;
  }

  // ---- render ----
  function el(name, attrs, parent) { const e = document.createElementNS(NS, name); for (const k in attrs) e.setAttribute(k, attrs[k]); if (parent) parent.appendChild(e); return e; }
  const W = 210, H = 64;
  function render() {
    const gN = $("#bm-nodes"), gE = $("#bm-edges"), gG = $("#bm-groups");
    gN.textContent = ""; gE.textContent = ""; gG.textContent = "";
    $("#bm-world").setAttribute("transform", `translate(${view.x},${view.y}) scale(${view.k})`);
    const showAuth = $("#bm-show-auth").checked;
    if ($("#bm-group").checked) {
      for (const g of ORDER) {
        const ms = nodes.filter((n) => n.group === g && pos[n.key]);
        if (!ms.length) continue;
        const xs = ms.map((n) => pos[n.key].x), ys = ms.map((n) => pos[n.key].y);
        const x0 = Math.min(...xs) - W / 2 - 18, y0 = Math.min(...ys) - H / 2 - 34, x1 = Math.max(...xs) + W / 2 + 18, y1 = Math.max(...ys) + H / 2 + 18;
        el("rect", { x: x0, y: y0, width: x1 - x0, height: y1 - y0, rx: 16, class: "bm-group" }, gG);
        const t = el("text", { x: x0 + 14, y: y0 + 20, class: "bm-group-t" }, gG); t.textContent = g;
      }
    }
    for (const e of edges) {
      if (e.kind === "authorized" && !showAuth) continue;
      const a = pos[e.from], b = pos[e.to];
      if (!a || !b) continue;
      const g = el("g", { class: "bm-edge " + e.st.toLowerCase() + (sel && sel.key === e.key ? " sel" : "") + (sel && sel.node && (e.from === sel.key || e.to === sel.key) ? " hl" : ""), tabindex: 0, role: "button", "aria-label": e.type + " " + e.st, "data-edge": e.key }, gE);
      const dx = b.x - a.x, dy = b.y - a.y, len = Math.hypot(dx, dy) || 1;
      const sx = a.x + dx / len * Math.min(W / 2, Math.abs(len * W / 2 / (Math.abs(dx) + 1e-9)) || 0, 120), sy = a.y + dy / len * 36;
      const ex = b.x - dx / len * 112, ey = b.y - dy / len * 36;
      el("line", { x1: sx, y1: sy, x2: ex, y2: ey, class: "bm-hit" }, g);
      el("line", { x1: sx, y1: sy, x2: ex, y2: ey, class: "bm-line", "marker-end": "url(#bm-arrow)" }, g);
      const t = el("text", { x: (sx + ex) / 2, y: (sy + ey) / 2 - 6, class: "bm-edge-t", "text-anchor": "middle" }, g);
      t.textContent = e.st === "VERIFIED" ? "verified" : e.st.toLowerCase().replace(/_/g, " ");
    }
    for (const n of nodes) {
      const p = pos[n.key]; if (!p) continue;
      const g = el("g", { class: "bm-node " + n.kind + (sel && sel.key === n.key ? " sel" : ""), transform: `translate(${p.x - W / 2},${p.y - H / 2})`, tabindex: 0, role: "button", "aria-label": n.label + ", " + n.kind, "data-node": n.key }, gN);
      el("rect", { width: W, height: H, rx: n.kind === "endpoint" ? 32 : 12, class: "bm-box" }, g);
      const vs = n.status.verified || "";
      const dotCls = /^VERIFIED|^REACHABLE/.test(vs) ? "ok" : /not running|FAILED/.test(vs + n.status.running) ? "bad" : n.kind === "processor" ? "ok" : "warn";
      el("circle", { cx: 16, cy: 20, r: 5, class: "bm-dot " + dotCls }, g);
      const t1 = el("text", { x: 28, y: 25, class: "bm-node-t" }, g); t1.textContent = n.label.length > 22 ? n.label.slice(0, 21) + "…" : n.label;
      const t2 = el("text", { x: 14, y: 46, class: "bm-node-s" }, g); t2.textContent = n.sub.length > 32 ? n.sub.slice(0, 31) + "…" : n.sub;
      const t3 = el("text", { x: W - 10, y: 25, class: "bm-node-k", "text-anchor": "end" }, g); t3.textContent = n.kind === "endpoint" ? "reported" : n.kind;
    }
    const empty = !nodes.filter((n) => n.kind === "feature").length;
    $("#bm-empty").hidden = !empty;
    if (empty) $("#bm-empty").textContent = "No installed features yet. Install TASS on the Setup & permissions page; it will appear here with its real status.";
  }

  // ---- inspector ----
  function kv(rows) { return `<dl class="bm-kv">${rows.map(([k, v]) => `<dt>${esc(k)}</dt><dd>${v}</dd>`).join("")}</dl>`; }
  function inspectNode(n) {
    const s = n.status, r = n.raw;
    let h = `<h3>${esc(n.label)}</h3><p class="small muted">${esc(n.kind === "endpoint" ? "model endpoint reported by a feature — not an installation" : n.kind)}</p>`;
    h += `<div class="bm-states">${["installed", "running", "authorized", "verified"].map((k) => `<div><b>${k.toUpperCase()}</b><span>${s[k]}</span></div>`).join("")}</div>`;
    if (n.kind === "feature") {
      const rec = r.record, rt = r.runtime || {}, hst = rt.health || {}, fs = hst.status || {};
      h += kv([["Installation ID", mono(rec.installation_id)], ["Feature", esc(rec.feature_id) + " v" + esc(rec.version)], ["Provider (publisher claim)", unk(rec.publisher_claim)],
        ["Source", unk(rec.source_repo)], ["Artifact SHA-256", mono(rec.artifact_sha256)], ["Trusted because", unk(rec.trust_basis)],
        ["Process", esc(rt.state || "UNKNOWN") + (rt.started_at ? " since " + esc(new Date(rt.started_at).toLocaleTimeString()) : "")],
        ["Last signed status", hst.checked_at ? esc(ago(hst.checked_at)) + (hst.ready ? " · ready" : " · not ready") : `<span class="muted">Not reported</span>`],
        ["Model (feature's report)", fs.model ? esc((fs.model.state || "") + " · " + (fs.model.name || fs.model.detail || "")) : `<span class="muted">Not reported</span>`],
        ["Capabilities", (rec.capabilities || []).map((c) => mono(c.id)).join("<br>") || `<span class="muted">none</span>`]]);
      const g = ((r.identity && r.identity.grants) || []).filter((x) => x.status === "ACTIVE");
      h += `<h4>Active grants (from Identity)</h4>` + (r.identity ? (g.length ? `<ul class="small">${g.map((x) => `<li>${mono(x.grant.capability)} for ${esc(r.people[x.grant.subject_identity_id] || x.grant.subject_identity_id)}</li>`).join("")}</ul>` : `<p class="small muted">None.</p>`) : `<p class="small muted">Unknown: Identity can't confirm right now.</p>`);
    } else if (n.kind === "identity") {
      const c = r.connection;
      h += kv([["Identity system ID", mono(c.identity_system_id)], ["Version", unk(c.identity_version)], ["Address", unk(c.base_url)], ["Paired", c.paired_at ? esc(new Date(c.paired_at).toLocaleString()) : `<span class="muted">Not reported</span>`],
        ["Last signed answer", esc(ago(c.last_ok))], ["State", esc(c.state)]]);
    } else if (n.kind === "processor") {
      const p = r.processor;
      h += kv([["Processor ID", mono(p.processor_id)], ["Version", unk(p.version)], ["Port", unk(p.port)], ["Platform", unk(p.platform)], ["PID", unk(p.pid)]]);
    } else {
      const m = r.model;
      h += kv([["Address", mono(m.url)], ["Kind", unk(m.kind)], ["Model", unk(m.name)], ["State", unk(m.state)], ["Detail", unk(m.detail)], ["Checked", esc(ago(m.checked_at))], ["Reported by", mono(r.reported_by)]]);
    }
    const links = edges.filter((e) => e.from === n.key || e.to === n.key);
    h += `<h4>Connections</h4>` + (links.length ? `<ul class="small">${links.map((e) => `<li><b>${esc(e.st)}</b> · ${esc(e.type)}</li>`).join("")}</ul>` : `<p class="small muted">No verified connection observed.</p>`);
    h += `<div class="row"><a class="btn small" href="/workstation" target="_blank" rel="noopener">Open Workstation</a> <button class="btn small" type="button" data-logs="${esc(n.key)}">View logs</button>
      <button class="btn small" type="button" disabled title="Not built yet: proposals need a reviewed capability catalog and separate Identity authorization (D001, D008)">Propose connection</button></div>`;
    return h;
  }
  function inspectEdge(e) {
    const nm = (k) => { const n = nodes.find((x) => x.key === k); return n ? esc(n.label) + " " + mono(k.replace(/^(processor|identity|endpoint):/, "")) : mono(k); };
    return `<h3>Connection</h3><p class="small"><b>${esc(e.st)}</b></p>` + kv([["From", nm(e.from)], ["To", nm(e.to)], ["Type", esc(e.type)], ["Evidence source", esc(e.source)],
      ["Last verified", e.last ? esc(ago(e.last)) : `<span class="muted">Never observed</span>`], ["Reason", e.why ? esc(e.why) : `<span class="muted">—</span>`], ["Recorded traffic", e.extra || `<span class="muted">—</span>`]])
      + `<p class="small muted">Solid only with fresh signed evidence (within ${FRESH_MS / 1000} s). A grant alone is never shown as verified traffic.</p>`;
  }
  function showInspector() {
    const box = $("#bm-insp");
    if (!sel) { box.innerHTML = `<h3>Inspector</h3><p class="muted small">Select a node or a line to see what is actually known about it, where it came from and when it was last checked.</p>`; return; }
    const n = sel.node ? nodes.find((x) => x.key === sel.key) : null, e = !sel.node ? edges.find((x) => x.key === sel.key) : null;
    box.innerHTML = n ? inspectNode(n) : e ? inspectEdge(e) : `<p class="muted small">That item is no longer present.</p>`;
  }

  // ---- logs ----
  function logMatches(e) {
    if (!logScope) return true;
    if (logScope.startsWith("ins_")) return (e.detail || "").includes(logScope);
    if (logScope.startsWith("identity:")) return /identity|connect|pair|admin|grant/i.test(e.component + " " + e.event);
    if (logScope.startsWith("processor:")) return true;
    return false;
  }
  function renderLogs() {
    const body = $("#bm-logs-body");
    const n = logScope ? nodes.find((x) => x.key === logScope) : null;
    $("#bm-log-scope").textContent = n ? n.label : "All";
    $("#bm-log-all").hidden = !logScope;
    if (!diag) { body.innerHTML = `<p class="muted small pad">Loading…</p>`; return; }
    const rows = (diag.entries || []).filter(logMatches).slice().reverse();
    if (!rows.length) { body.innerHTML = `<p class="pad"><b>NO TELEMETRY</b> <span class="muted small">— nothing recorded for this scope in this run. ${esc(diag.note)}</span></p>`; return; }
    body.innerHTML = `<table class="bm-table"><thead><tr><th>Time (local)</th><th>Source</th><th>Event</th><th>Level</th><th>Detail</th></tr></thead><tbody>`
      + rows.map((e) => `<tr><td>${esc(new Date(e.time).toLocaleTimeString())}</td><td>Thought Processor · ${esc(e.component)}</td><td>${mono(e.event)}</td><td>${esc(e.level)}</td><td>${esc(e.detail)}</td></tr>`).join("")
      + `</tbody></table><p class="muted small pad">${esc(diag.source)} · ${esc(diag.window)}. ${esc(diag.note)}</p>`;
  }
  async function copyClean() {
    const rows = ((diag && diag.entries) || []).filter(logMatches);
    const n = logScope ? nodes.find((x) => x.key === logScope) : null;
    const txt = [`TASS Thought Processor — clean diagnostic log`, `copied ${new Date().toISOString()} · scope: ${n ? n.label + " (" + logScope + ")" : "all"} · ${rows.length} event(s)`,
      `source: ${diag ? diag.source : "unavailable"} · ${diag ? diag.window : ""}`, `redaction: scrubbed when written and again here; no keys, tickets, pairing codes or private links`,
      diag ? `not included: ${diag.note}` : "", ""].concat(rows.map((e) => `${e.time} ${e.level} ${e.component} ${e.event} — ${e.detail}`)).join("\n");
    try { await navigator.clipboard.writeText(txt); $("#bm-copy-note").textContent = `Copied ${rows.length} event(s).`; }
    catch (_) { $("#bm-copy-note").textContent = "The browser blocked the clipboard; select the rows and copy instead."; }
  }

  // ---- updates (read-only) ----
  function renderUpdates() {
    const s = state || {}, conn = s.connection || {}, proc = s.processor || {};
    const feats = (s.installations || []).filter((v) => (v.record || {}).state === "INSTALLED");
    $("#bm-updates-body").innerHTML = `<p class="small">Installed versions, from Processor's own records:</p>`
      + `<table class="bm-table"><thead><tr><th>Service</th><th>Version</th><th>Pinned artifact</th></tr></thead><tbody>`
      + `<tr><td>Thought Processor</td><td>${unk(proc.version)}</td><td class="muted">launcher-built from its checkout</td></tr>`
      + `<tr><td>Identity</td><td>${unk(conn.identity_version)}</td><td class="muted">exact SHA checked by the launcher</td></tr>`
      + feats.map((v) => `<tr><td>${esc(v.record.name.split(" (")[0])} ${mono(v.record.installation_id)}</td><td>${esc(v.record.version)}</td><td>${mono(v.record.artifact_sha256.slice(0, 16))}…</td></tr>`).join("")
      + `</tbody></table>`
      + `<h4>Check for updates</h4><p class="small">This page has no trusted update source of its own, so it doesn't guess. The read-only check is in the Workbench menu: <b>6 Check Updates</b> (or <span class="mono">secure_launch.py check</span>). It compares your checkouts with GitHub and changes nothing.</p>`
      + `<p><button class="btn" type="button" disabled>Apply Update</button></p><p class="small muted">Disabled here: applying an update needs an exact pinned artifact, a compatibility check, a tested rollback and your approval. Features are never updated by silently changing their trusted digest.</p>`;
  }

  // ---- interaction: pan, zoom, drag, select ----
  const svg = $("#bm-svg");
  function toWorld(ev) { const r = svg.getBoundingClientRect(); return { x: (ev.clientX - r.left - view.x) / view.k, y: (ev.clientY - r.top - view.y) / view.k }; }
  let drag = null;
  svg.addEventListener("pointerdown", (ev) => {
    const nEl = ev.target.closest("[data-node]"), eEl = ev.target.closest("[data-edge]");
    if (nEl) { const k = nEl.getAttribute("data-node"), w = toWorld(ev); drag = { node: k, dx: w.x - pos[k].x, dy: w.y - pos[k].y, moved: false }; }
    else if (eEl) { sel = { key: eEl.getAttribute("data-edge"), node: false }; render(); showInspector(); return; }
    else drag = { pan: true, sx: ev.clientX - view.x, sy: ev.clientY - view.y, moved: false };
    svg.setPointerCapture(ev.pointerId);
  });
  svg.addEventListener("pointermove", (ev) => {
    if (!drag) return;
    drag.moved = true;
    if (drag.pan) { view.x = ev.clientX - drag.sx; view.y = ev.clientY - drag.sy; $("#bm-world").setAttribute("transform", `translate(${view.x},${view.y}) scale(${view.k})`); return; }
    const w = toWorld(ev); pos[drag.node] = { x: Math.round(w.x - drag.dx), y: Math.round(w.y - drag.dy) }; render();
  });
  svg.addEventListener("pointerup", () => {
    if (drag && drag.node) {
      if (drag.moved) scheduleSave();
      else { sel = { key: drag.node, node: true }; logScope = drag.node; render(); showInspector(); renderLogs(); }
    } else if (drag && drag.pan && !drag.moved) { sel = null; logScope = null; render(); showInspector(); renderLogs(); }
    drag = null;
  });
  svg.addEventListener("wheel", (ev) => {
    ev.preventDefault();
    const r = svg.getBoundingClientRect(), mx = ev.clientX - r.left, my = ev.clientY - r.top;
    const k = Math.min(2.5, Math.max(0.3, view.k * (ev.deltaY < 0 ? 1.12 : 1 / 1.12)));
    view.x = mx - (mx - view.x) * (k / view.k); view.y = my - (my - view.y) * (k / view.k); view.k = k;
    $("#bm-world").setAttribute("transform", `translate(${view.x},${view.y}) scale(${view.k})`);
  }, { passive: false });
  svg.addEventListener("keydown", (ev) => {
    const t = ev.target.closest("[data-node],[data-edge]");
    if (!t || (ev.key !== "Enter" && ev.key !== " ")) return;
    ev.preventDefault();
    const nk = t.getAttribute("data-node");
    sel = nk ? { key: nk, node: true } : { key: t.getAttribute("data-edge"), node: false };
    if (nk) logScope = nk;
    render(); showInspector(); renderLogs();
  });
  function fit() {
    const ks = Object.values(pos); if (!ks.length) return;
    const r = svg.getBoundingClientRect();
    const xs = ks.map((p) => p.x), ys = ks.map((p) => p.y);
    const x0 = Math.min(...xs) - W, x1 = Math.max(...xs) + W, y0 = Math.min(...ys) - 120, y1 = Math.max(...ys) + 120;
    view.k = Math.min(1.4, Math.max(0.3, Math.min(r.width / (x1 - x0), r.height / (y1 - y0))));
    view.x = (r.width - (x0 + x1) * view.k) / 2; view.y = (r.height - (y0 + y1) * view.k) / 2;
    render();
  }
  function scheduleSave() {
    clearTimeout(saveTimer);
    saveTimer = setTimeout(async () => {
      const keep = {}; for (const n of nodes) if (pos[n.key]) keep[n.key] = pos[n.key];
      try { const j = await api("POST", "/api/map/layout", { positions: keep }); $("#bm-saved").textContent = "Positions saved " + new Date(j.saved_at).toLocaleTimeString() + " (positions only)."; }
      catch (e) { $("#bm-saved").textContent = "Couldn't save positions: " + e.message; }
    }, 400);
  }
  document.addEventListener("click", (ev) => {
    const b = ev.target.closest("[data-logs]");
    if (b) { logScope = b.getAttribute("data-logs"); renderLogs(); $("#bm-logs").scrollIntoView({ block: "nearest" }); }
  });
  $("#bm-show-auth").onchange = render;
  $("#bm-group").onchange = render;
  $("#bm-arrange").onclick = () => { pos = autoArrange(true); render(); fit(); scheduleSave(); };
  $("#bm-fit").onclick = fit;
  $("#bm-log-all").onclick = () => { logScope = null; renderLogs(); };
  $("#bm-copy").onclick = copyClean;
  $("#bm-logs-toggle").onclick = () => { const b = $("#bm-logs-body"); b.hidden = !b.hidden; $("#bm-logs-toggle").textContent = (b.hidden ? "▶" : "▼") + " SERVICE LOGS"; $("#bm-logs-toggle").setAttribute("aria-expanded", String(!b.hidden)); };
  $("#bm-updates-btn").onclick = () => { renderUpdates(); $("#bm-updates").hidden = false; $("#bm-updates-btn").setAttribute("aria-expanded", "true"); };
  $("#bm-updates-close").onclick = () => { $("#bm-updates").hidden = true; $("#bm-updates-btn").setAttribute("aria-expanded", "false"); };

  // ---- load ----
  function chip(id, text, cls) { const c = $(id); c.className = "chip " + cls; c.querySelector(".t").textContent = text; }
  async function load(first) {
    try {
      state = await api("GET", "/api/state");
      const conn = state.connection || {};
      chip("#bm-identity", "Identity: " + (conn.state || "unknown").replace(/_/g, " ").toLowerCase(), conn.state === "VERIFIED" ? "ok" : "warn");
      const g = state.github || {};
      chip("#bm-github", "GitHub: " + (g.state ? g.state.replace(/_/g, " ").toLowerCase() + (g.account && g.account.login ? " · " + g.account.login : "") : "unknown"), g.state === "VERIFIED" ? "ok" : "warn");
      const b = build(state); nodes = b.N; edges = b.E;
      if (first) { try { const l = await api("GET", "/api/map/layout"); pos = l.positions || {}; if (l.saved_at) $("#bm-saved").textContent = "Positions from " + new Date(l.saved_at).toLocaleString() + "."; } catch (_) { pos = {}; } }
      Object.assign(pos, autoArrange(false));
      render();
      if (first) fit();
      showInspector();
      $("#bm-refreshed").textContent = "Status read " + new Date().toLocaleTimeString() + " · refreshes every 5 s.";
    } catch (e) {
      $("#bm-refreshed").textContent = "Couldn't read status: " + e.message;
      if (e.status === 401) { $("#bm-empty").hidden = false; $("#bm-empty").textContent = "This window isn't signed in. Use Open in the launcher, then choose Service Map."; }
    }
    try { diag = await api("GET", "/api/map/diagnostics"); } catch (_) { diag = { entries: [], note: "diagnostics unavailable", source: "", window: "" }; }
    renderLogs();
  }
  (async () => { if (!csrf) await refreshCsrf(); await load(true); setInterval(() => { if (!document.hidden && !drag) load(false); }, 5000); })();
})();
