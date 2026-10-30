"use strict";
(() => {
  const $ = (q) => document.querySelector(q);
  const esc = (s) => String(s == null ? "" : s).replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
  let csrf = "", repo = "", dir = "", state = null, timer = 0;

  async function signIn() {
    const m = location.hash.match(/code=([0-9a-f]+)/);
    if (m) {
      history.replaceState(null, "", "/workstation"); // the ticket is single-use; drop it from the address bar now
      const r = await fetch("/api/session", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ code: m[1] }) });
      const j = await r.json().catch(() => ({}));
      if (!r.ok) { lock(j.error || "This opening link was already used or expired. Use Open in the launcher again."); return false; }
      csrf = j.csrf; sessionStorage.setItem("tp_csrf", csrf);
    } else csrf = sessionStorage.getItem("tp_csrf") || "";
    return true;
  }
  async function api(method, path, body) {
    const h = {};
    if (body !== undefined) { h["Content-Type"] = "application/json"; h["X-TP-CSRF"] = csrf; }
    const r = await fetch(path, { method, headers: h, body: body === undefined ? undefined : JSON.stringify(body) });
    const j = await r.json().catch(() => ({}));
    if (!r.ok) { const e = new Error(j.error || "HTTP " + r.status); e.status = r.status; e.gate = j.gate; throw e; }
    return j;
  }
  function chip(el, text, cls) { el.className = "chip " + cls; el.querySelector(".t").textContent = text; }
  function lock(msg) { $("#ws-main").hidden = true; $("#ws-updates").hidden = true; $("#ws-locked").hidden = false; $("#ws-locked-msg").textContent = msg; }

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
    $("#ws-locked").hidden = true; $("#ws-main").hidden = tab !== "work";
    renderRepos(s.repos || []); renderTests(s.tests || []); renderRuns(s.runs || []);
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
    $("#ws-file-name").textContent = repo + "/" + p; $("#ws-file-meta").textContent = "";
    try {
      const j = await api("GET", "/api/ws/file?repo=" + encodeURIComponent(repo) + "&path=" + encodeURIComponent(p));
      $("#ws-file-meta").textContent = " · " + j.size + " bytes";
      $("#ws-file").textContent = j.shown ? j.text : j.note;
    } catch (e) { $("#ws-file").textContent = e.message; }
  }

  // ---- Updates ----
  let tab = "work", updTimer = 0;
  function showTab(t) {
    tab = t;
    $("#ws-updates").hidden = t !== "upd";
    if (state && state.gate && state.gate.ok) $("#ws-main").hidden = t !== "work";
    if (t === "upd") loadUpdates();
  }
  const stateClass = (s) => ({ PASS: "ok", CURRENT: "ok", UPDATE_AVAILABLE: "warn", APPROVAL_REQUIRED: "warn", UPDATING: "warn", RESTARTING: "warn",
    RECOVERED_PREVIOUS_VERSION: "warn" }[s] || "bad");
  async function loadUpdates() {
    clearTimeout(updTimer);
    let j;
    try { j = await api("GET", "/api/ws/updates"); } catch (e) { $("#ws-upd-list").innerHTML = `<p class="ws-warn small">${esc(e.message)}</p>`; return; }
    $("#ws-checked").textContent = " checked " + new Date(j.checked_at).toLocaleTimeString() + " · Identity upgrade API: " + (j.identity_upgrade_api || "NONE");
    let active = false;
    $("#ws-upd-list").innerHTML = j.components.map((c) => {
      const tx = c.tx, live = tx && !["PASS", "FAIL", "RECOVERED_PREVIOUS_VERSION", "NEEDS_ATTENTION", "DENIED_BY_ADMIN", "APPROVAL_EXPIRED", "CANCELLED", "BLOCKED"].includes(tx.state);
      if (live) active = true;
      const shown = live ? tx.state : (c.update_state || "");
      let h = `<div class="ws-test"><div><b>${esc(c.name)}</b> <span class="muted small">${esc(c.kind)}</span> <span class="ws-badge ${stateClass(shown)}">${esc(shown.replace(/_/g, " "))}</span></div>
        <div class="small">Installed <b>${esc(c.version)}</b> <span class="mono muted">${esc((c.revision || "").slice(0, 12))}</span> · status ${esc(c.status || "?")} · updates: ${esc(c.update_support)}</div>`;
      if (c.candidate) h += `<div class="small">Candidate <b>${esc(c.candidate.version)}</b> <span class="mono muted">${esc(String(c.candidate.artifact).slice(0, 12))}</span> · ${c.candidate.reviewed ? "reviewed (pinned)" : "NOT reviewed"} · data migration ${esc(c.candidate.data_migration || "?")}${c.candidate.security_fix ? " · security fix" : ""}<br><span class="muted">${esc(c.candidate.notes || "")} — ${esc(c.candidate.source)}</span></div>`;
      if (c.reason) h += `<div class="small muted">${esc(c.reason)}</div>`;
      if (c.requires && c.requires.length && c.can_update) h += `<div class="small muted">Requires: ${esc(c.requires.join("; "))}</div>`;
      if (c.can_update) h += `<button class="btn small" type="button" data-upd="${esc(c.id)}">Update to ${esc(c.candidate.version)}</button> `;
      if (tx && tx.state === "APPROVAL_REQUIRED") h += `<a class="btn small" href="${esc(tx.approval_url)}" target="_blank" rel="noopener">Approve in Identity</a> <button class="btn ghost small" type="button" data-updcancel="${esc(c.id)}">Cancel</button>`;
      if (tx) h += `<details${live ? " open" : ""}><summary class="small">Update ${esc(tx.from_version)} → ${esc(tx.to_version)}: <b>${esc(tx.state)}</b> ${esc(tx.result || "")}</summary><pre class="ws-pre">${esc(tx.steps.map((s) => s.at.slice(11, 19) + "  " + s.result.padEnd(5) + " " + s.step + (s.detail ? "  — " + s.detail : "")).join("\n"))}</pre></details>`;
      return h + "</div>";
    }).join("");
    document.querySelectorAll("[data-upd]").forEach((b) => b.onclick = async () => {
      b.disabled = true;
      try { await api("POST", "/api/ws/updates/" + encodeURIComponent(b.dataset.upd) + "/start", {}); } catch (e) { alert(e.message); }
      loadUpdates();
    });
    document.querySelectorAll("[data-updcancel]").forEach((b) => b.onclick = async () => {
      try { await api("POST", "/api/ws/updates/" + encodeURIComponent(b.dataset.updcancel) + "/cancel", {}); } catch (e) { alert(e.message); }
      loadUpdates();
    });
    if (tab === "upd") updTimer = setTimeout(loadUpdates, active ? 1500 : 10000);
  }

  async function loop() { clearTimeout(timer); await refresh(); timer = setTimeout(loop, (state && (state.runs || []).some((r) => r.state === "RUNNING")) ? 1500 : 4000); }
  $("#ws-tab-work").onclick = () => showTab("work");
  $("#ws-tab-upd").onclick = () => showTab("upd");
  $("#ws-check").onclick = () => loadUpdates();
  signIn().then((ok) => { if (ok) loop(); });
})();
