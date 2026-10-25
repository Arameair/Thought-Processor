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
    loadLink();
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
    CLAIMS_ENGINE: "info", ANSWERED_NOT_ENGINE: "bad",
  };
  const pillText = {
    NOT_CONFIGURED: "Not set", NOT_CHECKED: "Not checked", STALE: "Stale", UNREACHABLE: "Unreachable",
    PORT_OPEN_UNIDENTIFIED: "Unidentified", CLAIMS_IDENTITY_UNVERIFIED: "Found · unverified", ANSWERED_NOT_IDENTITY: "Wrong program",
    CLAIMS_ENGINE: "Running", ANSWERED_NOT_ENGINE: "Wrong program", NO_PUBLIC_METADATA: "No contract",
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
      {
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
    nb.textContent = { set_engine: "Enter Engine address", fix_profile: "Enter Engine address", wrong_engine: "Change Engine address",
      recheck: "Check now", start_engine: "Re-check after starting the Engine", attach_engine: "Type the attach code",
      create_identity: "Create an identity", turn_on: "Switch Thought Processing on", start_worker: "Re-check after starting Chat-Jimmy",
      observe: "Type an observation" }[n.code] || "Re-check now";
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
    if (code === "recheck" || code === "start_engine" || code === "start_worker") { refresh(true); return; }
    if (code === "turn_on") { $("proc-toggle").click(); return; }
    const f = $(state && state.next_action.focus) || $("engine-addr");
    if (code === "set_engine" && !f.value) f.value = "127.0.0.1:7798";
    f.focus(); if (f.select) f.select();
  });

  // ---------- Thought Engine link ----------
  let link = null, linkTimer = null;
  const obsPill = { waiting_for_ai: "warn", queued: "info", processing: "info", done: "ok", failed: "bad", denied: "bad" };
  const obsWord = { waiting_for_ai: "Waiting for AI", queued: "Queued", processing: "Processing", done: "Done", failed: "Failed", denied: "Not processed" };
  const linkPill = { ATTACHED: ["ok", "Attached"], NOT_ATTACHED: ["info", "Not attached"], ENGINE_NOT_RUNNING: ["bad", "Not running"],
    NOT_THOUGHT_ENGINE: ["bad", "Wrong program"], ATTACHMENT_REFUSED: ["bad", "Key refused"], ATTACHED_OTHER_ADDRESS: ["warn", "Other address"], NO_ENGINE_ADDRESS: ["", "Not set"] };

  async function loadLink() {
    const r = await api("GET", "/api/link");
    if (r.status === 401) { show("signin"); return; }
    if (r.ok) renderLink(r.data);
  }

  function renderLink(v) {
    link = v;
    const [cls, word] = linkPill[v.state] || ["", v.state];
    $("link-pill").className = "pill " + cls; $("link-pill").textContent = word;
    $("link-label").textContent = v.label;
    $("link-detail").textContent = v.detail;
    $("link-problem").hidden = !v.problem; $("link-problem").textContent = v.problem || "";
    const canAttach = ["NOT_ATTACHED", "ATTACHMENT_REFUSED", "ATTACHED_OTHER_ADDRESS"].includes(v.state);
    $("attach-form").hidden = !canAttach;
    const att = v.state === "ATTACHED";
    $("attached").hidden = !att;
    const st = v.engine_status || {};
    const fixture = !!(st.engine && st.engine.test_fixture_worker) || !!(st.worker && st.worker.test_fixture);
    $("fixture-banner").hidden = !(att && fixture);
    if (!att) { schedule(v); return; }
    const w = st.worker || {}, eng = st.engine || {};
    facts($("link-facts"), [
      ["Engine", (eng.version || "?") + " · " + (eng.branch || "?") + " @ " + (eng.commit || "?") + " · pid " + (eng.pid || "?")],
      ["AI worker", el("span", null, el("span", { class: "pill " + (w.state === "READY" ? (w.test_fixture ? "warn" : "ok") : "warn") }, w.label || w.state || "?"), " ", w.detail || "")],
      ["Waiting", String(st.waiting ?? 0) + " observation(s) saved, waiting for the AI worker"],
      ["Engine loop", st.loop || "—"],
      ["Attachment", v.attachment_id + " since " + t(v.attached_at)],
    ]);
    $("link-open").href = v.open_url || "#";
    const sel = $("id-select"); const prev = v.entity_id || "";
    sel.replaceChildren(el("option", { value: "" }, (st.identities || []).length ? "— choose —" : "— none yet —"));
    for (const id of st.identities || []) {
      const o = el("option", { value: id.entity_id }, id.display_name + (id.thought_processing ? " · on" : " · off"));
      if (id.entity_id === prev) o.selected = true;
      sel.append(o);
    }
    const act = v.activity;
    const on = !!(act && act.identity && act.identity.thought_processing);
    const tg = $("proc-toggle");
    tg.hidden = !act; tg.textContent = on ? "Thought Processing: ON — switch off" : "Thought Processing: OFF — switch on";
    tg.className = on ? "" : "primary"; tg.dataset.on = on ? "1" : "";
    $("obs-send").disabled = !(act && on);
    $("obs-text").disabled = !(act && on);
    $("obs-text").title = act ? (on ? "" : "Switch Thought Processing on first") : "Choose or create an identity first";
    const tl = $("thoughts"); tl.replaceChildren();
    for (const th of (act && act.thoughts) || []) {
      tl.append(el("li", null, el("span", { class: "q" }, th.question),
        th.test_fixture ? el("span", { class: "pill warn" }, "TEST FIXTURE") : null, " ",
        el("span", { class: "muted small" }, "topic " + th.topic + " · " + th.state + " · " + th.trace_count + " trace(s) · question by " + th.question_by + " · " + t(th.created_at))));
    }
    if (!tl.children.length) tl.append(el("li", { class: "muted" }, act ? "No Thought yet. " + (st.rule ? "Rule: " + st.rule : "") : "Choose an identity to see its Thoughts."));
    const ol = $("obs-list"); ol.replaceChildren();
    for (const o of (act && act.observations) || []) {
      ol.append(el("li", null, el("span", { class: "pill " + (obsPill[o.state] || "") }, obsWord[o.state] || o.state), " ",
        el("span", { class: "txt" }, o.text), el("span", { class: "st" }, t(o.received_at) + " · " + o.state_text)));
    }
    if (!ol.children.length) ol.append(el("li", { class: "muted" }, "Nothing sent yet."));
    $("link-rule").textContent = st.rule ? "Engine rule: " + st.rule : "";
    schedule(v);
  }

  // Poll quickly while something is in flight, slowly otherwise.
  function schedule(v) {
    clearTimeout(linkTimer);
    const busy = v.state === "ATTACHED" && v.activity && (v.activity.observations || []).some((o) => ["waiting_for_ai", "queued", "processing"].includes(o.state));
    const a = document.activeElement;
    linkTimer = setTimeout(() => { if (!document.hidden && !(a && a.id === "attach-code" && a.value)) loadLink(); else schedule(v); }, busy ? 1000 : 3000);
  }

  async function linkAction(path, body, msgId) {
    const r = await api("POST", path, body);
    const m = $(msgId);
    if (!r.ok) { m.className = "msg err"; m.textContent = (r.data && r.data.message) || ("HTTP " + r.status); return null; }
    m.className = "msg"; m.textContent = "";
    renderLink(r.data.link || r.data);
    refresh(false);
    return r.data;
  }
  $("attach-form").addEventListener("submit", async (e) => {
    e.preventDefault();
    const code = $("attach-code").value.trim();
    if (!code) return;
    $("attach-code").value = ""; // never keep the code on screen
    if (await linkAction("/api/link/attach", { code }, "attach-msg")) $("attach-msg").textContent = "Attached. The Engine now knows this Processor's key; the code is used up.";
  });
  $("link-detach").addEventListener("click", async () => {
    if (!confirm("Detach from this Thought Engine? Its data stays; you'd need a new attach code.")) return;
    linkAction("/api/link/detach", {}, "attach-msg");
  });
  $("id-select").addEventListener("change", () => { if ($("id-select").value) linkAction("/api/link/select", { entity_id: $("id-select").value }, "obs-msg"); });
  $("id-create").addEventListener("click", () => linkAction("/api/link/identity", { display_name: $("id-name").value }, "obs-msg"));
  $("proc-toggle").addEventListener("click", () => linkAction("/api/link/processing", { enabled: !$("proc-toggle").dataset.on }, "obs-msg"));
  $("obs-form").addEventListener("submit", async (e) => {
    e.preventDefault();
    const text = $("obs-text").value.trim();
    if (!text) return;
    const d = await linkAction("/api/link/observe", { text }, "obs-msg");
    if (d) { $("obs-text").value = ""; $("obs-msg").textContent = "Sent — the Engine saved it. " + ((d.observation && d.observation.state_text) || ""); $("obs-text").focus(); }
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
