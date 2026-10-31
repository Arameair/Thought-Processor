"use strict";
(() => {
  const $ = (q) => document.querySelector(q);
  const esc = (s) => String(s == null ? "" : s).replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
  let csrf = "", repo = "", dir = "", state = null, timer = 0, curFile = null, tassBusy = false, tassLog = [];

  async function signIn() {
    const m = location.hash.match(/code=([0-9a-f]+)/);
    if (m) {
      history.replaceState(null, "", "/workstation"); // the ticket is single-use; drop it from the address bar now
      const r = await fetch("/api/session", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ code: m[1] }) });
      const j = await r.json().catch(() => ({}));
      if (!r.ok) { lock(j.error || "This opening link was already used or expired. Use Open in the launcher again."); return false; }
      csrf = j.csrf; sessionStorage.setItem("tp_csrf", csrf);
    } else {
      csrf = sessionStorage.getItem("tp_csrf") || "";
      if (!csrf) await refreshCsrf();
    }
    return true;
  }
  // #33: see app.js refreshCsrf — the token of the session this window's cookie names.
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
    if (!r.ok) { const e = new Error(j.error || "HTTP " + r.status); e.status = r.status; e.gate = j.gate; throw e; }
    return j;
  }
  function chip(el, text, cls) { el.className = "chip " + cls; el.querySelector(".t").textContent = text; }
  function lock(msg) { $("#ws-main").hidden = true; $("#ws-locked").hidden = false; $("#ws-locked-msg").textContent = msg; }

  async function refresh() {
    let s;
    try { s = await api("GET", "/api/ws/state"); }
    catch (e) {
      if (e.status === 401) { lock("This window isn't signed in. Use Open in the launcher (it hands the browser a one-time ticket)."); return; }
      return;
    }
    state = s;
    const g = s.gate || {};
    chip($("#ws-identity"), g.reason === "IDENTITY_NOT_CONNECTED" ? "Identity: not connected" : "Identity: connected", g.reason === "IDENTITY_NOT_CONNECTED" ? "bad" : "ok");
    chip($("#ws-github"), g.github_state === "VERIFIED" ? "GitHub: " + (g.github_login || "verified") + (g.github_mode && g.github_mode !== "real" ? " (TEST FIXTURE)" : "")
      : "GitHub: " + String(g.github_state || "unknown").toLowerCase().replace(/_/g, " "), g.ok ? "ok" : g.github_state === "WAITING" ? "warn" : "bad");
    if (!g.ok) { lock(g.detail || g.reason || "Locked"); return; }
    $("#ws-locked").hidden = true; $("#ws-main").hidden = false;
    renderRepos(s.repos || []); renderTests(s.tests || []); renderRuns(s.runs || []);
    refreshTass();
    if (!repo && s.repos && s.repos.length) open(s.repos[0].name, "");
  }

  function renderRepos(rs) {
    $("#ws-repos").innerHTML = rs.map((r) => `<button class="ws-repo${r.name === repo ? " on" : ""}" data-repo="${esc(r.name)}" type="button">
      <b>${esc(r.name)}</b> <span class="muted small">${esc(r.branch || "(detached)")}</span><br>
      <span class="mono small">${esc((r.head_sha || "").slice(0, 12))}</span> <span class="small ${r.changed_files ? "ws-warn" : "muted"}">${r.changed_files ? r.changed_files + " changed" : "clean"}</span><br>
      <span class="muted small">${esc(r.last_commit || "")}</span><br><span class="muted small mono">${esc(r.origin || "")}</span></button>`).join("");
    document.querySelectorAll("[data-repo]").forEach((b) => b.onclick = () => open(b.dataset.repo, ""));
  }
  function renderTests(ts) {
    const busy = (state.runs || []).some((r) => r.state === "RUNNING");
    $("#ws-tests").innerHTML = ts.length ? ts.map((t) => `<div class="ws-test"><div>${esc(t.label)}</div>
      <div class="mono small muted">${esc(t.repo)}/${esc(t.dir)}: ${esc(t.argv.join(" "))}</div>
      <button class="btn small" data-test="${esc(t.id)}" type="button"${busy ? " disabled" : ""}>Run</button></div>`).join("") : `<p class="muted small">No tests for these repositories.</p>`;
    document.querySelectorAll("[data-test]").forEach((b) => b.onclick = async () => {
      b.disabled = true;
      try { await api("POST", "/api/ws/test", { id: b.dataset.test }); } catch (e) { alertRow(e.message); }
      refresh();
    });
  }
  function renderRuns(rs) {
    $("#ws-runs").innerHTML = rs.length ? rs.map((r) => `<details class="ws-run"${r === rs[0] ? " open" : ""}><summary>
      <span class="ws-badge ${r.state === "PASS" ? "ok" : r.state === "RUNNING" ? "warn" : "bad"}">${esc(r.state)}</span>
      ${esc(r.label)} <span class="muted small">@ ${esc((r.head_sha || "").slice(0, 12))} · exit ${r.exit_code} · ${esc(r.started)} · by ${esc(r.requested_by)}</span></summary>
      <pre class="ws-pre">${esc(r.output || (r.state === "RUNNING" ? "running…" : ""))}</pre></details>`).join("") : `<p class="muted small">No test run yet.</p>`;
  }
  function alertRow(msg) { $("#ws-runs").insertAdjacentHTML("afterbegin", `<p class="ws-warn small">${esc(msg)}</p>`); }

  async function open(r, d) {
    repo = r; dir = d;
    try {
      const j = await api("GET", "/api/ws/tree?repo=" + encodeURIComponent(r) + "&path=" + encodeURIComponent(d));
      const parts = j.path ? j.path.split("/") : [];
      let acc = "";
      $("#ws-crumbs").innerHTML = `<a href="#" data-dir="">${esc(r)}</a>` + parts.map((p) => { acc = acc ? acc + "/" + p : p; return ` / <a href="#" data-dir="${esc(acc)}">${esc(p)}</a>`; }).join("");
      $("#ws-tree").innerHTML = j.entries.map((e) => `<a href="#" class="ws-ent ${e.dir ? "dir" : "file"}" data-${e.dir ? "dir" : "file"}="${esc(j.path ? j.path + "/" + e.name : e.name)}">${e.dir ? "▸ " : ""}${esc(e.name)}</a>`).join("") || `<p class="muted small">Empty folder.</p>`;
      document.querySelectorAll("#ws-crumbs [data-dir], #ws-tree [data-dir]").forEach((a) => a.onclick = (ev) => { ev.preventDefault(); open(repo, a.dataset.dir); });
      document.querySelectorAll("#ws-tree [data-file]").forEach((a) => a.onclick = (ev) => { ev.preventDefault(); view(a.dataset.file); });
      if (state) renderRepos(state.repos || []);
    } catch (e) { $("#ws-tree").innerHTML = `<p class="ws-warn small">${esc(e.message)}</p>`; }
  }
  async function view(p) {
    curFile = { repo, path: p }; $("#ws-tass-cur").textContent = repo + "/" + p;
    $("#ws-file-name").textContent = repo + "/" + p; $("#ws-file-meta").textContent = "";
    try {
      const j = await api("GET", "/api/ws/file?repo=" + encodeURIComponent(repo) + "&path=" + encodeURIComponent(p));
      $("#ws-file-meta").textContent = " · " + j.size + " bytes";
      $("#ws-file").textContent = j.shown ? j.text : j.note;
      if (!tassLog.length) renderTassLog();
    } catch (e) { $("#ws-file").textContent = e.message; }
  }

  // ---- TASS ----
  const STEP = {
    INSTALL: ["TASS isn't installed for the workstation yet.", "Install TASS", "/api/ws/tass/install"],
    START: ["TASS is installed but not running.", "Start TASS", "/api/ws/tass/start"],
    OWNER: ["Setting who TASS talks to (your one person identity)…", "", "/api/ws/tass/owner"],
    ALLOW_CHAT: ["You need permission to talk to TASS (12 hours, approved by the Admin in Identity).", "Ask the Admin for permission", "/api/ws/tass/allow", { what: "chat" }],
    ALLOW_MODEL: ["TASS has no local model yet. You need permission to choose one (approved by the Admin).", "Ask the Admin for permission", "/api/ws/tass/allow", { what: "model" }],
    SET_MODEL: ["Choose TASS's local model server in WSL.", "Use this model", "/api/ws/tass/model"],
  };
  let ownerAsked = false;
  async function refreshTass() {
    let t;
    try { t = await api("GET", "/api/ws/tass"); } catch (_) { return; }
    const m = t.model || {};
    $("#ws-tass-chips").innerHTML = t.installation_id ? `<span class="ws-badge ${t.runtime === "RUNNING" ? "ok" : "warn"}">${esc(t.version)} ${esc(t.runtime)}</span> <span class="ws-badge ${m.state === "REACHABLE" ? "ok" : "warn"}">MODEL ${esc(String(m.state || "?").replace(/_/g, " "))}${m.name ? " · " + esc(m.name) : ""}</span>` + (t.owner ? ` <span class="muted">for ${esc(t.owner.name)}</span>` : "") : "";
    const ready = t.step === "READY";
    $("#ws-tass-chat").hidden = !ready;
    let h = "";
    if (t.older_tass) h += `<p class="muted small">${esc(t.older_tass.note)}</p>`;
    if (!ready) {
      const [msg, label, path, body] = STEP[t.step] || ["", "", ""];
      h += `<p class="small">${esc(msg)}</p>`;
      const pend = t.step === "INSTALL" ? t.install_pending : t.step === "ALLOW_CHAT" ? t.chat_grant : t.step === "ALLOW_MODEL" ? t.model_grant : null;
      if (pend && pend.approval_url) h += `<p><a class="btn small" href="${esc(pend.approval_url)}" target="_blank" rel="noopener">Approve in Identity</a> <span class="muted small">then come back; this updates by itself</span></p>`;
      else if (t.step === "SET_MODEL") h += `<label>Server <select id="ws-tass-kind"><option value="ollama">Ollama</option><option value="openai-compatible">llama.cpp / OpenAI-compatible</option></select></label> <label>Address <input id="ws-tass-url" value="http://127.0.0.1:11434"></label> <label>Model <input id="ws-tass-model" value="${esc(t.default_model)}"></label> <button class="btn small" type="button" id="ws-tass-step">${esc(label)}</button>`;
      else if (label) h += `<button class="btn small" type="button" id="ws-tass-step">${esc(label)}</button>`;
      if (t.step === "OWNER" && !ownerAsked) { ownerAsked = true; tassStep(path, {}); }
      if (t.step === "SET_MODEL" && m.detail) h += `<p class="muted small">${esc(m.detail)}</p>`;
    }
    const editing = $("#ws-tass-kind") ? [$("#ws-tass-kind").value, $("#ws-tass-url").value, $("#ws-tass-model").value] : null;
    $("#ws-tass-setup").innerHTML = h + `<div id="ws-tass-msg" class="small ws-warn"></div>`;
    const kind = $("#ws-tass-kind");
    if (kind && editing) {
      kind.value = editing[0]; $("#ws-tass-url").value = editing[1]; $("#ws-tass-model").value = editing[2];
    }
    if (kind) kind.onchange = () => {
      const granite = kind.value === "openai-compatible";
      $("#ws-tass-url").value = granite ? "http://127.0.0.1:8080" : "http://127.0.0.1:11434";
      $("#ws-tass-model").value = granite ? "ibm-research/granite-vision-3.2-2b-GGUF:Q4_K_M" : t.default_model;
    };
    const b = $("#ws-tass-step");
    if (b) b.onclick = () => {
      const [, , path, body] = STEP[t.step];
      tassStep(path, t.step === "SET_MODEL" ? { model: $("#ws-tass-model").value.trim(), kind: $("#ws-tass-kind").value, url: $("#ws-tass-url").value.trim() } : (body || {}));
    };
  }
  async function tassStep(path, body) {
    try {
      const j = await api("POST", path, body);
      if (j.approval_url) window.open(j.approval_url, "_blank", "noopener");
      if (j.grant && j.grant.approval_url) window.open(j.grant.approval_url, "_blank", "noopener");
      if (j.result && j.result.outcome !== "COMPLETED") $("#ws-tass-msg").textContent = j.result.outcome + ": " + j.result.detail;
    } catch (e) {
      const el = $("#ws-tass-msg");
      if (el) el.textContent = e.message;
    }
    refreshTass();
  }
  function renderTassLog() {
    $("#ws-tass-log").innerHTML = tassLog.map((m) => `<div class="ws-msg ${m.role}"><div class="small muted">${m.role === "you" ? "You" : "TASS"}${m.about ? " · about " + esc(m.about.repo + "/" + m.about.path) + (m.about.truncated ? " (first part only)" : "") : ""}${m.by ? " · " + esc(m.by) : ""}</div><div class="ws-msgtext">${esc(m.text)}</div></div>`).join("") || `<p class="muted small">Open a file, then ask TASS about it.</p>`;
    const l = $("#ws-tass-log"); l.scrollTop = l.scrollHeight;
  }
  async function askTass() {
    const q = $("#ws-tass-q").value.trim();
    if (!q || tassBusy) return;
    const useFile = $("#ws-tass-usefile").checked && curFile;
    tassBusy = true; $("#ws-tass-send").disabled = true; $("#ws-tass-busy").textContent = "TASS is thinking with your local model (a file question can take up to 2 minutes)…";
    tassLog.push({ role: "you", text: q, about: useFile ? curFile : null }); renderTassLog();
    try {
      const j = await api("POST", "/api/ws/tass/ask", { question: q, repo: useFile ? curFile.repo : "", path: useFile ? curFile.path : "" });
      if (j.outcome === "COMPLETED" && j.tass) {
        tassLog.push({ role: "tass", text: j.tass.reply || j.tass.headline, by: j.tass.answered_by });
        $("#ws-tass-q").value = "";
      } else tassLog.push({ role: "tass", text: "Not answered — " + j.outcome + ": " + j.detail, by: "nothing was saved" });
    } catch (e) { tassLog.push({ role: "tass", text: "Not answered — " + e.message, by: "" }); }
    tassBusy = false; $("#ws-tass-send").disabled = false; $("#ws-tass-busy").textContent = "";
    renderTassLog();
  }
  $("#ws-tass-send").onclick = askTass;
  $("#ws-tass-q").onkeydown = (e) => { if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) askTass(); };

  async function loop() { clearTimeout(timer); await refresh(); timer = setTimeout(loop, (state && (state.runs || []).some((r) => r.state === "RUNNING")) ? 1500 : 4000); }
  signIn().then((ok) => { if (ok) loop(); });

  // ---- Thoughts: what each feature has thought, and which model wrote it ----
  // Read-only. Every read is a normal Processor request: Identity ticket + live
  // check, using a permission the Admin already approved. Nothing is changed.
  const thMono = (s) => `<span class="mono">${esc(s)}</span>`;
  const thWhen = (t) => { if (!t) return ""; const d = new Date(t); return isNaN(d) ? esc(t) : esc(d.toLocaleString()); };
  const thModel = (m) => `<span class="badge info" title="which model wrote this">${esc(m || "model not recorded")}</span>`;
  const thFold = (label, text) => text ? `<details class="ws-th-fold"><summary>${esc(label)}</summary><pre class="ws-pre">${esc(text)}</pre></details>` : "";
  const TH_SOURCES = [
    { rx: /\.self\.read$/, kind: "pulse", title: "its own pulse" },
    { rx: /\.team\.read$/, kind: "team", title: "Core Team thought cycles" },
    { rx: /\.thoughts\.read$/, kind: "thoughts", title: "Thoughts" },
  ];
  function thPulse(o) {
    const refl = (o.reflections || []).slice().reverse();
    let h = `<p class="small">${esc(o.headline || "")}${o.pulse && o.pulse.stage ? ` · stage <b>${esc(o.pulse.stage)}</b>` : ""}${o.pulse && o.pulse.next_thought_in_s != null ? ` · next chance to think in ${esc(o.pulse.next_thought_in_s)} s` : ""}</p>`;
    if (o.seeds && o.seeds.length) h += `<div class="ws-th-item"><div class="ws-th-meta"><span class="badge">seed — written by Dustin, not a model</span></div><p>${o.seeds.map((x) => esc(x.text)).join(" · ")}</p></div>`;
    if (!refl.length) h += `<p class="muted small">No reflections yet. It thinks only when something about it changed, at most once a minute, and only with a local model set up.</p>`;
    h += refl.map((r) => `<div class="ws-th-item"><div class="ws-th-meta">${thModel(r.model)} <span class="muted small">${thWhen(r.at)}${r.took_s != null ? " · took " + esc(r.took_s) + " s" : ""}</span></div>
      <p><b>${esc(r.question || "")}</b></p><p>${esc(r.answer || "")}</p>${thFold("the model's reasoning — not a fact", r.thinking)}</div>`).join("");
    return h + (o.honesty ? `<p class="muted small">${esc(o.honesty)}</p>` : "");
  }
  function thTeam(o) {
    const c = o.latest;
    if (!c) return `<p class="muted small">No thought cycles yet.</p>`;
    let h = `<p class="small">Latest cycle ${thMono(c.id)} · <b>${esc(c.state)}</b> · ${thWhen(c.started_at)}</p><p><b>Topic:</b> ${esc(c.topic)}</p>`;
    h += (c.members || []).map((m) => `<div class="ws-th-item"><div class="ws-th-meta"><b>${esc(m.label)}</b> ${thModel(m.by)} <span class="badge ${m.state === "DONE" ? "ok" : "warn"}">${esc(m.state)}</span></div><p>${esc(m.text || m.detail || "")}</p></div>`).join("");
    if (c.thinker) h += `<div class="ws-th-item"><div class="ws-th-meta"><b>Thinker</b> ${thModel(c.thinker.by)} <span class="badge ${c.thinker.state === "DONE" ? "ok" : "warn"}">${esc(c.thinker.state)}</span></div><p>${esc(c.thinker.answer || c.thinker.detail || "")}</p>${thFold("the model's reasoning — not a fact", c.thinker.thinking)}</div>`;
    if (c.evaluator) h += `<div class="ws-th-item"><div class="ws-th-meta"><b>Evaluator</b> ${thModel(c.evaluator.by)} <span class="badge ${c.evaluator.state === "DONE" ? "ok" : "warn"}">${esc(c.evaluator.state)}</span></div><p>${esc(c.evaluator.text || c.evaluator.detail || "")}</p></div>`;
    if (c.thought) h += `<div class="ws-th-item ws-th-final"><div class="ws-th-meta"><b>Thought</b> ${thModel(c.thought.produced_by)} <span class="badge warn">${esc(c.thought.status || "UNVERIFIED")}</span></div><p>${esc(c.thought.text)}</p></div>`;
    return h + ((c.problems || []).length ? `<p class="problem small">${c.problems.map(esc).join("<br>")}</p>` : "");
  }
  function thThoughts(o) {
    const list = o.thoughts || [];
    let h = `<p class="small muted">${o.counts ? `active ${esc(o.counts.active)} · sleeping ${esc(o.counts.sleeping)} · resolved ${esc(o.counts.resolved)}` : ""}</p>`;
    if (!list.length) h += `<p class="muted small">No Thoughts yet.</p>`;
    h += list.map((t) => `<div class="ws-th-item"><div class="ws-th-meta"><span class="badge">rules — no AI model</span> <span class="badge ${t.state === "active" ? "ok" : ""}">${esc(t.state)}</span> <span class="muted small">${thWhen(t.updated_at || t.created_at)}</span></div>
      <p><b>${esc(t.question || t.topic)}</b></p><p class="small">${esc(t.why_it_formed || "")} · ${esc(t.evidence_count)} piece(s) of evidence</p></div>`).join("");
    return h;
  }
  let thBusy = false, thTimer = 0;
  async function loadThoughts() {
    if (thBusy || $("#ws-main").hidden) return;
    thBusy = true; $("#ws-th-status").textContent = "Reading…";
    try {
      const s = await api("GET", "/api/state");
      const out = [];
      for (const v of (s.installations || [])) {
        const r = v.record || {}, caps = r.capabilities || [];
        if (r.state !== "INSTALLED") continue;
        const name = (r.name || r.feature_id || "feature").split(" (")[0];
        for (const src of TH_SOURCES) {
          const cap = caps.find((c) => src.rx.test(c.id));
          if (!cap) continue;
          const head = `<h4>${esc(name)} · ${esc(src.title)} <span class="muted small">${thMono(cap.id)}</span></h4>`;
          if (!v.runtime || v.runtime.state !== "RUNNING") { out.push(head + `<p class="muted small">${esc(name)} isn't running. Start it on the Thought Processor page.</p>`); continue; }
          const g = ((v.identity && v.identity.grants) || []).find((x) => x.status === "ACTIVE" && x.grant.capability === cap.id);
          if (!g) { out.push(head + `<p class="muted small">Needs the ${thMono(cap.id)} permission. Ask the Admin on the Thought Processor page (Get TASS ready for this person).</p>`); continue; }
          let res = null;
          try { const j = await api("POST", `/api/installations/${encodeURIComponent(r.installation_id)}/action`, { subject: g.grant.subject_identity_id, capability: cap.id, input: "" }); res = j && j.result; }
          catch (e) { out.push(head + `<p class="problem small">${esc(e.message)}</p>`); continue; }
          const o = res && res.observation && res.observation.output;
          if (!o) { out.push(head + `<p class="problem small">${esc(String((res && res.outcome) || "no answer").replace(/_/g, " "))}${res && res.detail ? " — " + esc(res.detail) : ""}</p>`); continue; }
          out.push(head + (src.kind === "pulse" ? thPulse(o) : src.kind === "team" ? thTeam(o) : thThoughts(o)));
        }
      }
      $("#ws-th-list").innerHTML = out.join("") || `<p class="muted small">No installed feature shares its thoughts yet.</p>`;
      $("#ws-th-status").textContent = "Updated " + new Date().toLocaleTimeString();
    } catch (e) { $("#ws-th-status").textContent = "Couldn't read: " + e.message; }
    finally { thBusy = false; }
  }
  function thSchedule() {
    clearInterval(thTimer);
    if ($("#ws-th-auto").checked) thTimer = setInterval(() => { if (!document.hidden) loadThoughts(); }, 30000);
  }
  $("#ws-th-refresh").onclick = () => loadThoughts();
  $("#ws-th-auto").checked = sessionStorage.getItem("ws_th_auto") === "1";
  $("#ws-th-auto").onchange = () => { sessionStorage.setItem("ws_th_auto", $("#ws-th-auto").checked ? "1" : "0"); thSchedule(); };
  thSchedule();
  setTimeout(loadThoughts, 2000); // once, after the workstation has unlocked
})();
