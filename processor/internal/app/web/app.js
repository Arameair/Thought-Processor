"use strict";
(() => {
  const $ = (q) => document.querySelector(q);
  const esc = (s) => String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
  let csrf = "", state = null, timer = null, openPlanFor = "", exiting = false;

  async function api(method, path, body) {
    const opt = { method, headers: {} };
    if (method === "POST") {
      opt.headers["Content-Type"] = "application/json";
      opt.headers["X-TP-CSRF"] = csrf;
      opt.body = JSON.stringify(body || {});
    }
    const r = await fetch(path, opt);
    let j = {};
    try { j = await r.json(); } catch (_) {}
    if (r.status === 401) { showGate(j.error); throw new Error(j.error || "not signed in"); }
    if (!r.ok) throw new Error(j.error || ("HTTP " + r.status));
    return j;
  }
  function showGate(msg) {
    $("#shell").hidden = true; $("#gate").hidden = false;
    if (msg) $("#gate-msg").textContent = msg;
  }

  async function signIn() {
    const m = location.hash.match(/code=([0-9a-f]+)/);
    if (m) {
      try {
        const r = await fetch("/api/session", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ code: m[1] }) });
        const j = await r.json();
        if (r.ok) { csrf = j.csrf; sessionStorage.setItem("tp_csrf", csrf); }
        else { showGate(j.error); return false; }
      } catch (_) { showGate(); return false; }
      history.replaceState(null, "", "/"); // the link is private: remove it from the address bar
    } else {
      csrf = sessionStorage.getItem("tp_csrf") || "";
    }
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
    const busy = state && (state.cards || []).some((c) => c.install && c.install.busy);
    timer = setTimeout(async () => { await refresh(); schedule(); }, busy ? 600 : 2000);
  }
  async function act(path, body, errEl) {
    try {
      state = await api("POST", path, body);
      if (errEl) errEl.hidden = true;
      render();
      return true;
    } catch (e) {
      if (errEl) { errEl.textContent = e.message; errEl.hidden = false; }
      else alert(e.message);
      return false;
    }
  }

  // ---------- rendering ----------
  const connWords = {
    CONNECTED: ["ok", "Identity: connected"],
    AUTHORITY_UNAVAILABLE: ["warn", "Identity: can't reach it"],
    CONNECTION_REMOVED: ["bad", "Identity: connection removed"],
    WRONG_SERVICE: ["bad", "Identity: not the one you connected"],
    INCOMPATIBLE: ["bad", "Identity: wrong version"],
    PAIRING_UNREADABLE: ["bad", "Identity: saved connection unreadable"],
    WAITING_FOR_CODE: ["", "Identity: waiting for code"],
    FOUND_NOT_CONNECTED: ["", "Identity: not connected"],
    IDENTITY_NOT_FOUND: ["", "Identity: not running"],
  };

  function render() {
    const s = state, c = s.connection;
    const [cls, words] = connWords[c.state] || ["", c.state];
    const chip = $("#chip-identity");
    chip.className = "chip " + cls;
    chip.querySelector(".t").textContent = words;
    chip.title = c.detail || "";
    $("#devmode").checked = !!s.developer_mode;

    const paired = !!c.identity_system_id && !["CONNECTION_REMOVED", "WRONG_SERVICE", "PAIRING_UNREADABLE", "INCOMPATIBLE"].includes(c.state);
    const showHome = s.verified || (paired && c.state === "AUTHORITY_UNAVAILABLE");
    $("#connect").hidden = showHome;
    $("#home").hidden = !showHome;
    const ban = $("#banner");
    if (c.state === "AUTHORITY_UNAVAILABLE" && paired) {
      ban.className = "banner"; ban.hidden = false;
      ban.textContent = "Can't reach Identity right now. Nothing that needs Identity's permission can be done until it's back — Processor reconnects by itself.";
    } else ban.hidden = true;

    renderGitHub(s);
    if (showHome) renderHome(s); else renderConnect(s);
    renderDev(s);
    renderModals(s);
  }

  function renderConnect(s) {
    const c = s.connection, body = $("#c-body");
    let title = "Connect to Identity", lead = "", h = "";
    switch (c.state) {
      case "IDENTITY_NOT_FOUND":
        title = "Start Identity";
        lead = "Thought Processor works together with Identity, a separate program on this computer. Start Identity (identity.exe) — Processor will find it by itself.";
        h = `<p class="muted small">Looking for Identity every few seconds…</p>` + advancedAddress();
        break;
      case "FOUND_NOT_CONNECTED": {
        lead = "Identity is running. Connect Processor to it once; after that they reconnect by themselves.";
        const cands = c.candidates || [];
        h = cands.map((x) => `<div class="cand"><div><strong>Identity ${esc(x.version)}</strong><div class="muted small">${esc(x.base_url)}${x.profile ? " · " + esc(x.profile) : ""}${x.compatible ? "" : " · <span class='error'>version not supported</span>"}</div></div>
          <button class="btn primary" data-connect="${esc(x.base_url)}" ${x.compatible ? "" : "disabled"}>Connect</button></div>`).join("");
        if (cands.length > 1) h = `<p class="small">More than one Identity is running on this computer. Choose the one you use:</p>` + h;
        h += advancedAddress();
        break;
      }
      case "WAITING_FOR_CODE":
        title = "Type the code from Identity";
        lead = "Look at the Identity window. It's showing an 8-character code in a box marked PAIRING REQUEST. Type it here.";
        h = `<form id="code-form" class="row"><input id="code-in" class="codein" maxlength="9" placeholder="XXXX-XXXX" autocomplete="off" required>
             <button class="btn primary" type="submit">Connect</button></form>
             <p class="muted small">The code works once and expires in 5 minutes. <button class="btn ghost small" type="button" data-newcode="1">Show a new code</button></p>`;
        break;
      case "CONNECTION_REMOVED":
        title = "Identity no longer recognises this Processor";
        lead = (c.detail || "") + " You'll need to connect again.";
        h = `<div class="row"><button class="btn primary" data-reconnect="1">Connect again</button></div>`;
        break;
      case "WRONG_SERVICE":
        title = "A different Identity is answering";
        lead = "The program answering isn't the Identity this Processor connected to (" + esc(c.identity_system_id || "") + "), so nothing it says is trusted. If you changed Identity on purpose, connect again.";
        h = `<div class="row"><button class="btn primary" data-reconnect="1">Connect again</button></div>`;
        break;
      case "INCOMPATIBLE":
      case "PAIRING_UNREADABLE":
        title = c.state === "INCOMPATIBLE" ? "Identity version doesn't match" : "The saved connection can't be read";
        lead = c.detail || "";
        h = `<div class="row"><button class="btn primary" data-reconnect="1">Connect again</button></div>`;
        break;
      default:
        title = "Checking Identity…";
        lead = c.detail || "";
    }
    $("#c-title").textContent = title;
    $("#c-lead").textContent = lead;
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

  function renderGitHub(s) {
    const chip = $("#chip-github"), g = s.github;
    chip.hidden = !s.verified || !g;
    if (!g) return;
    chip.classList.remove("ok", "warn", "bad", "link");
    if (g.state === "VERIFIED") { chip.classList.add("ok"); chip.querySelector(".t").textContent = "GitHub: " + (g.account ? g.account.login : "signed in"); }
    else if (g.state === "WAITING") { chip.classList.add("warn", "link"); chip.querySelector(".t").textContent = "GitHub: waiting for you"; }
    else { chip.classList.add("link"); chip.querySelector(".t").textContent = "Sign in to GitHub"; }
  }

  function renderHome(s) {
    const ids = s.identities || [], sel = s.selected;
    const list = ids.map((e) => `<li><button type="button" class="${sel && sel.id === e.id ? "sel" : ""}" data-select="${esc(e.id)}"><span>${esc(e.name)}</span><span class="kind">${esc(e.kind)}</span></button></li>`).join("")
      || (s.verified ? `<li class="muted small">No identities yet. Add one below.</li>` : `<li class="muted small">Waiting for Identity…</li>`);
    if ($("#id-list").dataset.html !== list) { $("#id-list").innerHTML = list; $("#id-list").dataset.html = list; }
    $("#new-id").querySelectorAll("input,select,button").forEach((x) => (x.disabled = !s.verified));

    const cards = s.cards || [];
    $("#no-identity").hidden = !!sel || (!s.verified && cards.length > 0);
    $("#features").hidden = !sel && !(!s.verified && cards.length > 0);
    $("#f-title").textContent = sel ? "Features for " + sel.name : "Installed features";
    $("#f-sub").textContent = sel ? "Shown from Identity. You only see features your account is approved for." : "";
    const note = $("#catalog-note");
    if (s.catalog_note && s.github && s.github.state !== "VERIFIED") {
      note.hidden = false;
      const h = `<span>${esc(s.catalog_note)}</span><button class="btn primary small" type="button" data-gh="1">Sign in to GitHub</button>`;
      if (note.dataset.html !== h) { note.innerHTML = h; note.dataset.html = h; }
    } else note.hidden = true;

    const h = cards.map((c) => card(c, sel, s.verified)).join("") || (sel ? `<p class="muted">No features are available to this account yet.</p>` : "");
    const box = $("#cards");
    if (box.dataset.html !== h) { box.innerHTML = h; box.dataset.html = h; }
  }

  function fmtSize(n) { return n > 1e6 ? (n / 1e6).toFixed(1) + " MB" : Math.max(1, Math.round(n / 1e3)) + " KB"; }

  function card(c, sel, verified) {
    const iv = c.install || {}, inst = iv.installed;
    const b = [];
    if (c.visibility === "private") b.push(`<span class="badge info" title="Private: shown only to approved accounts">Approved for your account</span>`);
    if (inst) {
      b.push(`<span class="badge ok">Installed ${esc(inst.tag)}</span>`);
      if (iv.verify === "OK") b.push(`<span class="badge ok" title="Checked against the approved SHA-256 ${esc(new Date(iv.verified_at).toLocaleTimeString())}">Files verified</span>`);
      else if (iv.verify) b.push(`<span class="badge bad">Files changed or missing</span>`);
      b.push(`<span class="badge">Not running — starting features comes later</span>`);
    } else b.push(`<span class="badge">Not installed</span>`);
    if (sel && c.from_identity) {
      b.push(`<span class="badge" title="Identity has no grant for this">Not turned on for ${esc(sel.name)}</span>`);
      b.push(`<span class="badge" title="No information source is allowed by default">No sources allowed</span>`);
    }
    let body = "";
    if (iv.busy) {
      const pct = iv.size ? Math.floor((100 * iv.got) / iv.size) : 0;
      body += `<div class="small">${iv.phase === "VERIFYING" ? "Checking the file…" : "Downloading through Identity… " + pct + "%"}</div><progress class="progress" max="100" value="${pct}"></progress>`;
    }
    if (iv.problem) body += `<div class="problem"><strong>${esc(problemTitle(iv.problem.code))}</strong> ${esc(iv.problem.message)}<br><span class="small">${esc(iv.problem.next_step)}</span></div>`;
    if (c.note && c.action === "none") body += `<p class="small muted">${esc(c.note)}</p>`;
    let btn = "";
    const dis = verified ? "" : "disabled";
    switch (c.action) {
      case "install": btn = `<button class="btn primary" data-plan="${esc(c.feature)}" ${dis}>Install…</button>`; break;
      case "update": btn = `<button class="btn" data-plan="${esc(c.feature)}" ${dis}>Check for a newer version</button>`; break;
      case "approve": btn = `<button class="btn primary" data-review="${esc(c.feature)}">Review and approve…</button>`; break;
    }
    if (inst && !iv.busy) btn += ` <button class="btn ghost small" data-remove="${esc(c.feature)}">Remove</button>`;
    return `<div class="card"><h3>${esc(c.name)}</h3><div class="muted small">${esc(c.description || "")}</div><div class="badges">${b.join("")}</div>${body}<div class="row">${btn}</div></div>`;
  }

  function problemTitle(code) {
    return ({ CHECKSUM_MISMATCH: "Download didn't match — nothing installed.", DOWNLOAD_INTERRUPTED: "Download stopped — nothing installed.",
      UNAPPROVED_SOURCE: "Not from the approved source — nothing installed.", NO_RELEASE: "No version published yet.",
      AUTHORITY_UNAVAILABLE: "Identity unavailable.", FILES_CHANGED: "Installed files changed.", FILES_MISSING: "Installed files missing.",
      PLAN_EXPIRED: "Approval out of date.", CONTRACT_INVALID: "Feature description not understood." }[code] || code);
  }

  function renderDev(s) {
    const d = s.dev;
    $("#dev").hidden = !d;
    if (!d) return;
    const logs = (d.log || []).map((e) => `<div class="${e.level === "warn" || e.level === "error" ? "warn" : ""}">${esc(new Date(e.time).toLocaleTimeString())} ${esc(e.component)} ${esc(e.event)} — ${esc(e.detail)}</div>`).join("");
    const info = { processor: s.processor, connection: s.connection, lifecycle: s.lifecycle, build: d.build, run_id: d.run_id, profile: d.profile,
      identity_status: d.identity_status, installs: d.installs, identity_features: d.identity_features };
    const h = `<pre>${esc(JSON.stringify(info, null, 2))}</pre><h3 class="small">Recent activity</h3><div class="log">${logs}</div>`;
    if ($("#dev-body").dataset.html !== h) { $("#dev-body").innerHTML = h; $("#dev-body").dataset.html = h; }
  }

  function renderModals(s) {
    // GitHub sign-in
    const g = s.github, mg = $("#m-github");
    if (s.verified && g && g.state === "WAITING") {
      mg.hidden = false;
      $("#g-code").textContent = g.user_code || "…";
      $("#g-link").href = g.verification_uri || "#";
      $("#g-link").textContent = (g.verification_uri || "").replace(/^https?:\/\//, "");
      $("#g-status").textContent = "Waiting for you to approve on GitHub… " + (g.github_said ? "(GitHub says: " + g.github_said + ")" : "");
    } else mg.hidden = true;
    // Install approval
    const c = (s.cards || []).find((x) => x.feature === openPlanFor && x.install && x.install.plan);
    const ma = $("#m-approve");
    if (!c) { ma.hidden = true; openPlanFor = ""; return; }
    const p = c.install.plan;
    ma.hidden = false;
    $("#a-title").textContent = (p.upgrade ? "Update " : "Install ") + p.name + "?";
    const facts = [["Version", p.tag], ["Publisher", p.publisher], ["From", p.repo], ["File", p.asset + " (" + fmtSize(p.size) + ")"],
      ["For this computer", p.platform], ["SHA-256", p.sha256], ["Built by", p.uploader]];
    $("#a-facts").innerHTML = facts.map(([k, v]) => `<dt>${esc(k)}</dt><dd>${esc(v)}</dd>`).join("");
    $("#a-checks").innerHTML = `<strong>Checked by Identity and Processor</strong><ul>${(p.checks || []).map((x) => `<li>${esc(x)}</li>`).join("")}<li>the downloaded file must match the SHA-256 above, or nothing is installed</li></ul>`;
    $("#a-limit").textContent = "Not checked: " + (p.not_checked || "publisher signature (releases aren't signed yet).");
    $("#a-ok").onclick = async () => { ma.hidden = true; openPlanFor = ""; await act(`/api/features/${encodeURIComponent(c.feature)}/approve`, { plan_id: p.plan_id }); schedule(); };
    $("#a-cancel").onclick = () => { ma.hidden = true; openPlanFor = ""; act(`/api/features/${encodeURIComponent(c.feature)}/cancel`, {}); };
  }

  // ---------- events ----------
  document.addEventListener("click", async (e) => {
    const t = e.target.closest("button, .chip");
    if (!t) return;
    if (t.dataset.connect) { await act("/api/connect/request", { url: t.dataset.connect }, $("#c-error")); }
    else if (t.dataset.newcode) { await act("/api/connect/request", {}, $("#c-error")); }
    else if (t.dataset.reconnect) { await act("/api/connect/disconnect", {}); }
    else if (t.dataset.select) { await act("/api/identities/select", { id: t.dataset.select }); }
    else if (t.dataset.plan) {
      t.disabled = true;
      if (await act(`/api/features/${encodeURIComponent(t.dataset.plan)}/plan`, {})) { openPlanFor = t.dataset.plan; render(); }
    }
    else if (t.dataset.review) { openPlanFor = t.dataset.review; render(); }
    else if (t.dataset.remove) {
      if (confirm("Remove the installed files for this feature from this computer?")) await act(`/api/features/${encodeURIComponent(t.dataset.remove)}/remove`, {});
    }
    else if (t.dataset.gh || (t.id === "chip-github" && state && state.github && state.github.state !== "VERIFIED")) { await act("/api/github/connect", {}); }
  });
  $("#g-cancel").onclick = () => act("/api/github/cancel", {});
  $("#g-copy").onclick = () => navigator.clipboard && navigator.clipboard.writeText($("#g-code").textContent);
  $("#devmode").onchange = (e) => act("/api/devmode", { on: e.target.checked });
  $("#new-id").onsubmit = async (e) => {
    e.preventDefault();
    if (await act("/api/identities", { name: $("#new-name").value, kind: $("#new-kind").value }, $("#id-error"))) $("#new-name").value = "";
  };
  $("#exit-btn").onclick = () => {
    const s = state || {}, lc = s.lifecycle;
    $("#x-note").textContent = (s.exit && s.exit.note) || "";
    const row = $("#x-identity-row");
    row.hidden = !(s.verified && lc);
    if (lc && lc.ownership === "MANAGED") $("#x-identity-why").textContent = "(Processor started it; it stops with the workspace)";
    else $("#x-identity-why").textContent = "(you started it separately; it keeps running unless you tick this)";
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
      $("#d-note").textContent = "Identity: " + r.identity + ".";
    } catch (e) { exiting = false; alert(e.message); }
  };

  (async () => { if (await signIn()) { await refresh(); schedule(); } })();
})();
