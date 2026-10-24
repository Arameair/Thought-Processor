"use strict";
// Thought Processor page. Everything shown comes from /api/state. Text from
// other programs is inserted with textContent only and labeled UNVERIFIED.
(() => {
  const $ = (id) => document.getElementById(id);
  let csrf = "";
  let state = null;
  let timer = null;
  let lastLog = [];
  let tab = "flow";

  function el(tag, attrs, ...kids) {
    const n = document.createElement(tag);
    for (const [k, v] of Object.entries(attrs || {})) {
      if (k === "class") n.className = v; else n.setAttribute(k, v);
    }
    for (const k of kids) if (k != null) n.append(k.nodeType ? k : String(k));
    return n;
  }
  const t = (iso) => { if (!iso) return "—"; const d = new Date(iso); return isNaN(d) ? "—" : d.toLocaleTimeString(); };
  const ago = (iso) => { const s = Math.round((Date.now() - new Date(iso)) / 1000); return s < 2 ? "just now" : s + " s ago"; };

  async function api(method, path, body) {
    const opt = { method, headers: {}, credentials: "same-origin" };
    if (method !== "GET") {
      opt.headers["Content-Type"] = "application/json";
      opt.headers["X-TP-CSRF"] = csrf;
      opt.body = JSON.stringify(body || {});
    }
    let r;
    try { r = await fetch(path, opt); } catch (e) { return { ok: false, status: 0, data: null }; }
    const ct = r.headers.get("Content-Type") || "";
    const data = ct.includes("json") ? await r.json().catch(() => null) : await r.text();
    return { ok: r.ok, status: r.status, data };
  }

  function show(which) {
    $("signin").hidden = which !== "signin";
    $("app").hidden = which !== "app";
    $("gone").hidden = which !== "gone";
  }

  // ---------- sign-in ----------
  async function signIn(code) {
    const r = await fetch("/api/session", { method: "POST", credentials: "same-origin",
      headers: { "Content-Type": "application/json" }, body: JSON.stringify({ code }) }).catch(() => null);
    if (!r) { $("signin-msg").textContent = "Processor is not answering."; return false; }
    const d = await r.json().catch(() => ({}));
    if (!r.ok) { $("signin-msg").textContent = d.message || "Sign-in refused."; $("signin-msg").className = "msg err"; return false; }
    csrf = d.csrf; return true;
  }

  async function boot() {
    const m = location.hash.match(/code=([0-9a-f]+)/);
    if (m) history.replaceState(null, "", location.pathname); // keep the code out of history
    let r = await api("GET", "/api/session");
    if (r.status === 0) { show("gone"); return; }
    if (r.ok) csrf = r.data.csrf;
    else if (m && await signIn(m[1])) { /* signed in */ }
    else { show("signin"); $("code").focus(); return; }
    show("app");
    await refresh(true);
    timer = setInterval(() => refresh(true), 10000);
  }

  $("signin-form").addEventListener("submit", async (e) => {
    e.preventDefault();
    if (await signIn($("code").value.trim())) { $("code").value = ""; boot(); }
  });

  // ---------- rendering ----------
  const pillClass = {
    NOT_CONFIGURED: "", NOT_CHECKED: "", STALE: "warn", UNREACHABLE: "bad",
    PORT_OPEN_UNIDENTIFIED: "warn", CLAIMS_IDENTITY_UNVERIFIED: "info", ANSWERED_NOT_IDENTITY: "bad", NO_PUBLIC_METADATA: "warn",
  };
  const pillText = {
    NOT_CONFIGURED: "Not set", NOT_CHECKED: "Not checked", STALE: "Stale", UNREACHABLE: "Unreachable",
    PORT_OPEN_UNIDENTIFIED: "Unidentified", CLAIMS_IDENTITY_UNVERIFIED: "Found · unverified", ANSWERED_NOT_IDENTITY: "Wrong program",
  };

  function facts(dl, rows) {
    dl.replaceChildren();
    for (const [k, v] of rows) { if (v == null) continue; dl.append(el("dt", null, k), el("dd", null, v)); }
  }

  function renderDep(prefix, d) {
    const pill = $(prefix + "-pill");
    pill.className = "pill " + (pillClass[d.state] || "");
    pill.textContent = pillText[d.state] || d.state;
    $(prefix + "-label").textContent = d.label;
    const rows = [["Address", d.configured || "—"]];
    if (d.probe) {
      const p = d.probe;
      rows.push(["Checked", t(p.checked_at) + " (" + ago(p.checked_at) + ", " + p.elapsed_ms + " ms)"]);
      rows.push(["Port", p.port]);
      if (prefix === "id") {
        rows.push(["Metadata", p.metadata + (p.http_status ? " · HTTP " + p.http_status : "")]);
        if (p.claimed_app || p.claimed_version) {
          const dd = el("span", null, el("span", { class: "claim" }, (p.claimed_app || "?") + " " + (p.claimed_version || "")), el("span", { class: "unv" }, "UNVERIFIED CLAIM"));
          rows.push(["It says", dd]);
        }
      }
      rows.push(["Note", p.note]);
    }
    rows.push([prefix === "id" ? "Approval" : "Attachment", d.approval]);
    facts($(prefix + "-facts"), rows);
    const input = $(prefix === "id" ? "identity-addr" : "engine-addr");
    if (document.activeElement !== input) input.value = (d.configured || "").replace("http://", "");
  }

  function render(s) {
    state = s;
    const me = s.self;
    $("runline").textContent = `run ${me.run_id.slice(4, 12)} · pid ${me.pid} · 127.0.0.1:${me.port}`;
    const b = me.build;
    facts($("self"), [
      ["Address", me.address], ["Process", "pid " + me.pid], ["Run", me.run_id],
      ["Started", t(me.started_at) + " (" + me.startup_ms + " ms to start)"],
      ["Build", (b.revision === "unknown" ? "unknown commit" : b.revision.slice(0, 12)) + (b.modified === "true" ? " + local changes" : "")],
      ["Version", b.version + " · " + b.go_version + " · " + b.os + "/" + b.arch],
      ["Profile", me.profile],
      ["Owns", "no processes (it cannot start any)"],
      me.previous_unclean_run ? ["Last run", "ended without a clean stop; nothing was done about it"] : null,
      me.profile_problem ? ["Problem", me.profile_problem] : null,
      me.log_error ? ["Log", me.log_error] : null,
    ].filter(Boolean));
    renderDep("id", s.identity);
    renderDep("en", s.engine);

    const n = s.next_action;
    $("next-title").textContent = n.title;
    $("next-detail").textContent = n.detail;
    const nb = $("next-btn");
    nb.textContent = { set_identity: "Enter Identity address", fix_profile: "Enter Identity address", wrong_identity: "Change Identity address",
      recheck: "Check now", start_identity: "Re-check after starting Identity", blocked_contract: "Read what Identity must add" }[n.code] || "Re-check now";
    nb.dataset.code = n.code;

    const ops = $("ops"); ops.replaceChildren();
    const probeOps = $("probe-ops"); probeOps.replaceChildren();
    for (const op of s.operations) {
      const btn = el("button", { type: "button", disabled: "", "aria-describedby": "op-" + op.action }, op.label);
      ops.append(el("li", null, btn, el("p", { id: "op-" + op.action }, op.reason)));
      const tryBtn = el("button", { type: "button", class: "quiet" }, "Ask server: " + op.action);
      tryBtn.addEventListener("click", () => askServer(op.action));
      probeOps.append(tryBtn);
    }

    const caps = $("caps"); caps.replaceChildren();
    for (const c of s.capabilities) {
      const cls = c.status === "WORKING" ? "ok" : c.status === "BLOCKED" ? "warn" : "";
      caps.append(el("tr", null, el("td", null, c.label), el("td", null, el("span", { class: "pill " + cls }, c.status)), el("td", null, c.why)));
    }
    lastLog = s.log;
    renderLogs();
    const sel = s.selection;
    for (const k of ["entity", "feature", "instance"]) { $("sel-" + k).textContent = sel[k].label; $("sel-" + k + "-why").textContent = sel[k].why; }
    $("checked").textContent = "Page updated " + t(s.self.server_time) + ". Checks repeat every 10 s while this page is open.";
  }

  function renderLogs() {
    const kind = $("log-kind").value;
    const log = $("log"); log.replaceChildren();
    for (const e of lastLog.slice().reverse()) {
      if (kind && e.kind !== kind) continue;
      log.append(el("li", null, t(e.at) + " ", el("span", { class: "k" }, e.kind + "/" + e.code + " "), e.detail, e.op_id ? " · " + e.op_id : ""));
    }
    if (!log.children.length) log.append(el("li", { class: "muted" }, "No events of this kind yet."));
  }
  $("log-kind").addEventListener("change", renderLogs);

  const statusPill = { DONE: "ok", REFUSED: "bad", BLOCKED: "warn", "NOT AVAILABLE": "warn", "NOT ATTEMPTED": "", NONE: "" };
  async function renderFlow() {
    const r = await api("GET", "/api/flow");
    const ol = $("flow"); ol.replaceChildren();
    if (!r.ok) return;
    if (!r.data.traces.length) {
      ol.append(el("li", { class: "muted" }, "No protected requests yet. Use “Prove it: ask the server anyway” under Engine controls to create one."));
      return;
    }
    for (const tr of r.data.traces) {
      const stages = el("ol", { class: "stages" });
      for (const st of tr.stages) {
        stages.append(el("li", null, el("b", null, st.name), el("span", { class: "pill " + (statusPill[st.status] || "") }, st.status), " ", st.owner, el("br"), el("span", { class: "muted" }, st.detail)));
      }
      ol.append(el("li", null, el("div", { class: "head" }, el("b", null, tr.action), el("span", { class: "pill " + (statusPill[tr.outcome] || "") }, tr.outcome), el("span", { class: "muted" }, t(tr.at) + " · " + tr.op_id)), stages));
    }
  }

  async function renderMetrics() {
    const r = await api("GET", "/api/metrics");
    if (!r.ok) return;
    const m = r.data;
    $("metrics-source").textContent = m.source + ".";
    const rows = [["Uptime", m.uptime_s + " s (since " + t(m.since) + ")"],
      ["Memory", Math.round(m.process.heap_alloc_bytes / 1024) + " KiB heap · " + m.process.goroutines + " goroutines · " + m.process.gc_cycles + " GC"],
      ["Sign-ins", m.operator_sign_ins + " ok · " + m.operator_code_errors + " wrong code · " + m.active_sessions + " active"]];
    for (const [k, p] of Object.entries(m.probes)) {
      const l = p.latency_last_50;
      rows.push([k + " checks", p.checks + " · " + Object.entries(p.outcomes).map(([o, n]) => o + " ×" + n).join(", ")]);
      if (l.samples) rows.push([k + " check time", "min " + l.min_ms + " / median " + l.median_ms + " / max " + l.max_ms + " ms (last " + l.samples + ")"]);
    }
    const fmt = (o) => Object.keys(o).length ? Object.entries(o).map(([k, v]) => k + " ×" + v).join(", ") : "none";
    rows.push(["Refused operations", fmt(m.refused_operations)]);
    rows.push(["Rejected requests", fmt(m.rejected_requests)]);
    facts($("metrics"), rows);
    $("metrics-na").textContent = m.not_available.join("; ") + ".";
  }

  function showTab(name) {
    tab = name;
    for (const b of document.querySelectorAll(".tabs button")) b.setAttribute("aria-selected", String(b.dataset.tab === name));
    for (const p of document.querySelectorAll(".tab")) p.hidden = p.id !== "tab-" + name;
    if (name === "flow") renderFlow();
    if (name === "metrics") renderMetrics();
  }
  for (const b of document.querySelectorAll(".tabs button")) b.addEventListener("click", () => showTab(b.dataset.tab));
  $("diag").addEventListener("toggle", () => { if ($("diag").open) showTab(tab); });

  $("copy-logs").addEventListener("click", async () => {
    const r = await api("GET", "/api/logs" + ($("log-kind").value ? "?kind=" + encodeURIComponent($("log-kind").value) : ""));
    if (!r.ok) { $("logs-msg").textContent = "Could not read logs."; return; }
    try { await navigator.clipboard.writeText(r.data); $("logs-msg").textContent = "Copied " + r.data.split("\n").length + " lines."; }
    catch { $("logs-msg").textContent = "Clipboard not available in this browser."; }
  });

  async function refresh(probe) {
    const r = probe ? await api("POST", "/api/probe") : await api("GET", "/api/state");
    if (r.status === 0) { clearInterval(timer); show("gone"); return; }
    if (r.status === 401) { clearInterval(timer); show("signin"); return; }
    if (r.ok) {
      render(r.data);
      if ($("diag").open && tab === "flow") renderFlow();
      if ($("diag").open && tab === "metrics") renderMetrics();
    }
  }

  async function saveAddr(kind, value, msgId) {
    const body = {}; body[kind] = value;
    const r = await api("POST", "/api/candidates", body);
    const m = $(msgId);
    if (r.ok) { m.className = "msg"; m.textContent = value ? "Saved and checked. Saving an address grants nothing." : "Cleared."; render(r.data); }
    else { m.className = "msg err"; m.textContent = (r.data && r.data.message) || "Not saved."; }
  }
  $("id-form").addEventListener("submit", (e) => { e.preventDefault(); saveAddr("identity", $("identity-addr").value, "id-msg"); });
  $("en-form").addEventListener("submit", (e) => { e.preventDefault(); saveAddr("engine", $("engine-addr").value, "en-msg"); });
  $("recheck").addEventListener("click", () => refresh(true));
  $("next-btn").addEventListener("click", () => {
    const code = $("next-btn").dataset.code;
    if (code === "blocked_contract") { window.open($("deps-link").href, "_blank", "noopener,noreferrer"); return; }
    if (code === "recheck" || code === "start_identity") { refresh(true); return; }
    $("identity-addr").focus(); $("identity-addr").select();
  });

  async function askServer(action) {
    const r = await api("POST", "/api/engine/" + action);
    const out = $("probe-out");
    out.hidden = false;
    out.textContent = "HTTP " + r.status + "\n" + JSON.stringify(r.data, null, 2);
    await refresh(false);
    if ($("diag").open && tab === "flow") renderFlow();
  }

  $("copy").addEventListener("click", async () => {
    const r = await api("GET", "/api/context");
    if (!r.ok) { $("copy-msg").textContent = "Could not build context."; return; }
    $("context").hidden = false; $("context").textContent = r.data;
    try { await navigator.clipboard.writeText(r.data); $("copy-msg").textContent = "Copied. Codes, cookies and other programs' text are left out."; }
    catch { $("copy-msg").textContent = "Shown below — copy it by hand (clipboard not available)."; }
  });

  $("stop").addEventListener("click", async () => {
    if (!confirm("Stop this Thought Processor? Identity and any Engine keep running.")) return;
    const r = await api("POST", "/api/shutdown");
    clearInterval(timer);
    $("gone-title").textContent = r.status === 202 ? "Processor stopped" : "Stop request failed";
    $("gone-detail").textContent = r.status === 202 ? "Only this Processor stopped. Start it again from its terminal; it will print a new sign-in link." : "HTTP " + r.status;
    show("gone");
  });

  // Opening the sign-in link in a tab that already shows this page only
  // changes the #fragment, which does not reload — so react to it.
  window.addEventListener("hashchange", () => { if (/code=/.test(location.hash)) { clearInterval(timer); boot(); } });

  boot();
})();
