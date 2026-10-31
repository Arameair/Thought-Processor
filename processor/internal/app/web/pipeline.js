"use strict";
(() => {
  const $ = (q) => document.querySelector(q);
  const esc = (s) => String(s == null ? "" : s).replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
  const mono = (s) => `<span class="mono">${esc(s)}</span>`;
  const when = (t) => { const d = new Date(t); return isNaN(d) ? esc(t || "") : esc(d.toLocaleTimeString()); };
  let csrf = sessionStorage.getItem("tp_csrf") || "", busy = false, timer = 0;
  // TASS's conversation is re-read only when a new message request appears, so
  // watching this page doesn't add a history read every few seconds.
  const cache = {}; // installation|identity -> {newest, pairs, note}

  // #33: the token of the session this window's cookie names (changes still need it).
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
    if (!r.ok) throw new Error(j.error || "HTTP " + r.status);
    return j;
  }

  // Where a request stopped, from Processor's recorded outcome.
  const STOP_AT = { NOT_RUNNING: 2, ERROR: 2, TICKET_REJECTED: 2, DENIED_BY_IDENTITY: 3, AUTHORITY_UNAVAILABLE: 3, DENIED_BY_FEATURE: 4, REQUEST_UNCERTAIN: 4 };
  function stage(n, title, cls, body) {
    return `<div class="pl-stage ${cls}"><div class="pl-n">${n}</div><div class="pl-t">${esc(title)}</div><div class="pl-b">${body}</div></div>`;
  }
  function row(req, pair, names, histNote) {
    // A feature's own refusal comes back as an HTTP 422 from TASS after Identity allowed it:
    // a model problem stops at the Model step, anything else TASS refused stops at TASS.
    const fromFeature = req.outcome === "ERROR" && /HTTP 4\d\d /.test(req.detail || "");
    const stop = req.outcome === "COMPLETED" ? 99 : fromFeature ? (/MODEL_|NO_MODEL/.test(req.detail) ? 5 : 4) : (STOP_AT[req.outcome] || 2);
    const cls = (n) => n < stop ? "ok" : n === stop ? "bad" : "idle";
    const you = pair && pair.you, reply = pair && pair.reply;
    const who = names[req.subject_identity_id] || req.subject_identity_id;
    const words = you ? `<p>${esc(you.text)}</p>${you.about ? `<p class="small muted">with file ${mono(you.about.repo + "/" + you.about.path)}</p>` : ""}${you.attachment ? `<p class="small muted">with picture ${esc(you.attachment.name)}</p>` : ""}`
      : `<p class="small muted">${esc(histNote)}</p>`;
    const failed = stop !== 99 ? `<p class="small">${esc(req.outcome.replace(/_/g, " "))}</p><p class="small muted">${esc(req.detail || "")}</p>` : "";
    let h = `<section class="panel pl-row"><div class="ws-viewhead"><b>${when(req.at)}</b> <span class="muted small">for ${esc(who)} · ${mono(req.request_id)} · ${esc(req.latency_ms)} ms total</span>
      <span class="badge ${req.outcome === "COMPLETED" ? "ok" : "bad"}">${esc(req.outcome.replace(/_/g, " "))}</span></div><div class="pl-flow">`;
    h += stage(1, "You", "ok", words);
    h += stage(2, "Thought Processor", cls(2), `<p class="small">asked Identity for a one-use ticket for ${mono(req.capability)}</p>${stop === 2 ? failed : ""}`);
    h += stage(3, "Identity", cls(3), stop > 3 ? `<p class="small">allowed</p><p class="small muted">decision ${mono((req.decision_id || "").slice(0, 16))}<br>ticket ${mono((req.ticket_id || "").slice(0, 16))}</p>` : stop === 3 ? failed : "");
    h += stage(4, "TASS", cls(4), stop > 4 && stop !== 99 ? `<p class="small">checked the ticket and asked Identity again; nothing was saved</p>`
      : stop > 4 ? `<p class="small">checked the ticket and asked Identity again, then ${reply ? "saved your message and its answer" : you ? "saved your message" : "carried it out"}</p>` : stop === 4 ? failed : "");
    const model = reply && reply.by;
    if (stop === 5) h += stage(5, "Model", "bad", failed);
    else h += stage(5, "Model", stop > 4 ? (model ? "ok" : (you ? "idle" : "ok")) : "idle",
      stop > 4 ? (model ? `<span class="badge info">${esc(model)}</span>${reply.thinking ? `<details class="ws-th-fold"><summary>${esc(reply.thinking_label || "the model's reasoning — not a fact")}</summary><pre class="ws-pre">${esc(reply.thinking)}</pre></details>` : ""}`
        : `<p class="small muted">${you ? "no local model answered (none set up); nothing was made up" : esc(histNote)}</p>`) : "");
    h += stage(6, "Reply", stop > 4 && reply ? "ok" : "idle", stop > 4 && reply ? `<p>${esc(reply.text)}</p>` : "");
    return h + `</div></section>`;
  }

  async function load() {
    if (busy) return;
    busy = true; $("#pl-status").textContent = "Reading…";
    try {
      const s = await api("GET", "/api/state");
      const names = Object.fromEntries((s.identities || []).map((e) => [e.id, e.name]));
      const out = [];
      for (const v of (s.installations || [])) {
        const r = v.record || {}, caps = r.capabilities || [];
        const send = caps.find((c) => /\.chat\.send$/.test(c.id)), hist = caps.find((c) => /\.chat\.history$/.test(c.id));
        if (!send || r.state !== "INSTALLED") continue;
        const st = (v.runtime && v.runtime.health && v.runtime.health.status) || {}, m = st.model || {};
        out.push(`<section class="panel"><div class="ws-viewhead"><h3>${esc((r.name || "TASS").split(" (")[0])} ${esc(r.version || "")}</h3>
          <span class="badge ${v.runtime && v.runtime.state === "RUNNING" ? "ok" : "warn"}">${esc((v.runtime && v.runtime.state) || "?")}</span>
          <span class="badge ${st.authority && st.authority.state === "CONNECTED" ? "ok" : "warn"}">Identity ${esc((st.authority && st.authority.state) || "?")}</span>
          <span class="badge ${m.state === "REACHABLE" ? "ok" : "warn"}">MODEL ${esc(String(m.state || "?").replace(/_/g, " "))}${m.name ? " · " + esc(m.name) : ""}</span></div>
          ${m.detail ? `<p class="muted small">${esc(m.detail)}</p>` : ""}</section>`);
        const rq = await api("GET", `/api/installations/${encodeURIComponent(r.installation_id)}/requests?capability=${encodeURIComponent(send.id)}`);
        const reqs = (rq.requests || []).slice(0, 25);
        if (!reqs.length) { out.push(`<p class="muted small pad">No messages yet. Talk to TASS in the Development Workstation; each message shows up here.</p>`); continue; }
        // Words: TASS's saved conversation, per identity, only with an approved history permission.
        const convo = {}, notes = {};
        for (const sub of [...new Set(reqs.map((q) => q.subject_identity_id))]) {
          const key = r.installation_id + "|" + sub, newest = (reqs.find((q) => q.subject_identity_id === sub) || {}).request_id;
          const c = cache[key];
          if (c && c.newest === newest) { if (c.pairs) convo[sub] = c.pairs.map((p) => ({ ...p, used: false })); else notes[sub] = c.note; continue; }
          const g = hist && ((v.identity && v.identity.grants) || []).find((x) => x.status === "ACTIVE" && x.grant.capability === hist.id && x.grant.subject_identity_id === sub);
          if (!g) { notes[sub] = "words hidden: reading them needs the tass.chat.history permission for " + (names[sub] || sub); continue; }
          if (!v.runtime || v.runtime.state !== "RUNNING") { notes[sub] = "words hidden: TASS isn't running"; continue; }
          try {
            const j = await api("POST", `/api/installations/${encodeURIComponent(r.installation_id)}/action`, { subject: sub, capability: hist.id, input: "50" });
            const o = j && j.result && j.result.observation && j.result.observation.output;
            if (!o) { notes[sub] = "words hidden: " + ((j && j.result && (j.result.detail || j.result.outcome)) || "no answer"); continue; }
            const msgs = o.conversation || [], pairs = [];
            for (let i = 0; i < msgs.length; i++) if (msgs[i].role === "you") pairs.push({ you: msgs[i], reply: msgs[i + 1] && msgs[i + 1].role === "tass" ? msgs[i + 1] : null, used: false });
            convo[sub] = pairs;
            cache[key] = { newest, pairs };
          } catch (e) { notes[sub] = "words hidden: " + e.message; }
        }
        for (const q of reqs) {
          let pair = null;
          if (q.outcome === "COMPLETED" && convo[q.subject_identity_id]) {
            const t0 = new Date(q.at).getTime(), win = (q.latency_ms || 0) + 5000;
            let best = null, bestD = Infinity;
            for (const p of convo[q.subject_identity_id]) {
              if (p.used) continue;
              const d = new Date(p.you.at).getTime() - t0;
              if (d >= -2000 && d <= win && Math.abs(d) < bestD) { best = p; bestD = Math.abs(d); }
            }
            if (best) { best.used = true; pair = best; }
          }
          const note = notes[q.subject_identity_id] || (q.outcome === "COMPLETED" ? "words not found in the saved conversation (it may have been forgotten)" : "nothing was saved for a refused request");
          out.push(row(q, pair, names, note));
        }
      }
      $("#pl-list").innerHTML = out.join("") || `<p class="muted small pad">TASS isn't installed yet.</p>`;
      $("#pl-status").textContent = "Updated " + new Date().toLocaleTimeString();
    } catch (e) {
      $("#pl-status").textContent = "Couldn't read: " + e.message;
      if (/signed in|401/.test(e.message)) $("#pl-list").innerHTML = `<p class="pad">This window isn't signed in. Use <b>Open</b> in the launcher, then come here from the Development Workstation.</p>`;
    } finally { busy = false; }
  }
  function schedule() { clearInterval(timer); if ($("#pl-auto").checked) timer = setInterval(() => { if (!document.hidden) load(); }, 10000); }
  $("#pl-refresh").onclick = () => load();
  $("#pl-auto").checked = sessionStorage.getItem("pl_auto") !== "0";
  $("#pl-auto").onchange = () => { sessionStorage.setItem("pl_auto", $("#pl-auto").checked ? "1" : "0"); schedule(); };
  (async () => { if (!csrf) await refreshCsrf(); load(); schedule(); })();
})();
