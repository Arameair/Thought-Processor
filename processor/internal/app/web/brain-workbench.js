"use strict";
(() => {
  const $ = (s) => document.querySelector(s);
  const map = $("#bw-map"), layer = $("#bw-nodes"), wires = $("#bw-wires");
  const root = $("#ws-main");
  let ws = null, engine = null, selected = "processor", mode = "move", pending = null, drag = null, wireDrag = null;
  let nodes = [], proposals = [], layout = {}, lastClean = "", loading = false, actionBusy = false;
  const storageLayout = "tp_brain_positions_v1", storageProposals = "tp_brain_proposals_v1";
  try { layout = JSON.parse(localStorage.getItem(storageLayout) || "{}"); } catch (_) { layout = {}; }
  try { proposals = JSON.parse(localStorage.getItem(storageProposals) || "[]"); } catch (_) { proposals = []; }
  if (!layout || typeof layout !== "object" || Array.isArray(layout)) layout = {};
  if (!Array.isArray(proposals)) proposals = [];
  proposals = proposals.filter(p => p && typeof p.from === "string" && typeof p.to === "string").slice(0,50);
  const saveLayout = () => { try { localStorage.setItem(storageLayout, JSON.stringify(layout)); } catch (_) {} };
  const saveProposals = () => { try { localStorage.setItem(storageProposals, JSON.stringify(proposals)); } catch (_) {} };
  const put = (id, value) => { const e = $("#" + id); if (e) e.textContent = String(value == null ? "Not reported" : value); };
  const safe = (v) => v === null || v === undefined || v === "" ? "Not reported" : String(v);
  const element = (tag, cls, text) => { const e = document.createElement(tag); if (cls) e.className = cls; if (text != null) e.textContent = String(text); return e; };
  const byId = id => nodes.find(n => n.id === id);
  const status = msg => put("bw-action-status", msg);

  async function api(path, options) {
    const r = await fetch(path, Object.assign({ cache: "no-store" }, options || {}));
    const j = await r.json().catch(() => ({}));
    if (!r.ok) throw Error(j.error || "HTTP " + r.status);
    return j;
  }
  async function post(path) {
    let csrf = sessionStorage.getItem("tp_csrf") || "";
    if (!csrf) {
      const k = await api("/api/session/csrf", { headers: { "X-TP-Want-CSRF": "1" } });
      csrf = k.csrf || ""; if (csrf) sessionStorage.setItem("tp_csrf", csrf);
    }
    const send = (token) => api(path, { method: "POST", headers: { "Content-Type": "application/json", "X-TP-CSRF": token }, body: "{}" });
    try { return await send(csrf); }
    catch (e) {
      if (!/anti-forgery/i.test(e.message)) throw e;
      const k = await api("/api/session/csrf", { headers: { "X-TP-Want-CSRF": "1" } });
      if (!k.csrf) throw e;
      sessionStorage.setItem("tp_csrf", k.csrf); return send(k.csrf);
    }
  }
  function buildNodes(s) {
    const connected = s && s.verified === true;
    const conn = s && s.connection || {};
    const entries = Array.isArray(s && s.installations) ? s.installations : [];
    nodes = [
      { id:"processor", name:"Thought Processor", subtitle:"Human-facing control", x:42, y:42, icon:"🧠", info:s && s.processor || {}, kind:"hub", state:"RUNNING" },
      { id:"identity", name:"Identity", subtitle:"Authentication and permissions", x:7, y:14, icon:"🛡", kind:"authority", state:connected ? "VERIFIED" : safe(conn.state) }
    ];
    entries.forEach((view, i) => {
      const r = view.record || {}, rt = view.runtime || {};
      if (r.state !== "INSTALLED" || !r.installation_id) return;
      const angle = i * 2.39996, x = 42 + Math.cos(angle) * 34, y = 41 + Math.sin(angle) * 32;
      nodes.push({ id:r.installation_id, name:r.name || r.feature_id || "Installed feature",
        subtitle: r.feature_id || "Installed", icon:"◈", kind:"feature",
        x:Math.max(0,Math.min(77,x)), y:Math.max(0,Math.min(78,y)),
        state:rt.state || "NOT_REPORTED", view });
    });
    nodes.forEach(n => {
      const l = layout[n.id];
      if (l && Number.isFinite(l.x) && Number.isFinite(l.y)) {
        n.x = Math.max(0, Math.min(80, l.x)); n.y = Math.max(0, Math.min(81, l.y));
      }
    });
    if (!byId(selected)) selected = "processor";
  }
  function currentStatus(n) {
    if(n.kind === "hub") return "RUNNING";
    if(n.kind === "authority") return n.state;
    return n.view && n.view.runtime ? n.view.runtime.state || "NOT_REPORTED" : "NOT_REPORTED";
  }
  function lamp(n) {
    const st = currentStatus(n);
    if(st === "RUNNING" || st === "VERIFIED") return "ok";
    if(st === "STOPPED" || st === "STARTING" || st === "NOT_REPORTED") return "warn";
    return "bad";
  }
  function nodeElement(n) {
    const b = element("button", "bw-node" + (n.kind === "hub" ? " hub" : "") + (selected === n.id ? " selected" : ""));
    b.type = "button"; b.dataset.nodeId = n.id;
    b.style.left = n.x + "%"; b.style.top = n.y + "%";
    b.append(element("span","bw-node-icon",n.icon), element("strong", "", n.name),
      element("small", "", n.subtitle), element("small", "", currentStatus(n)));
    b.append(element("span","bw-lamp " + lamp(n)));
    const port=element("span","bw-port"); port.title="Drag from here to propose a connection"; b.append(port);
    b.addEventListener("click", () => {
      if(b.dataset.suppressClick === "true") { b.dataset.suppressClick = ""; return; }
      pick(n.id);
    });
    b.addEventListener("pointerdown", e => {
      if(mode === "connect" && e.target === port && e.button === 0){
        e.preventDefault();e.stopPropagation();pending=null;
        wireDrag={from:n.id,to:null};
        b.setPointerCapture(e.pointerId);
        $("#bw-help").textContent="Drag the connector to another feature to propose an inactive link.";
        return;
      }
      if(mode !== "move" || e.button !== 0) return;
      const r = map.getBoundingClientRect();
      drag = {id:n.id, startX:e.clientX, startY:e.clientY, x:n.x*r.width/100, y:n.y*r.height/100, moved:false};
      b.setPointerCapture(e.pointerId);
    });
    b.addEventListener("pointermove", e => {
      if(wireDrag && wireDrag.from===n.id){
        const r=map.getBoundingClientRect();
        wireDrag.to={x:e.clientX-r.left,y:e.clientY-r.top};
        drawWires();return;
      }
      if(!drag || drag.id !== n.id || mode !== "move") return;
      const r = map.getBoundingClientRect(), dx = e.clientX - drag.startX, dy = e.clientY - drag.startY;
      if(Math.abs(dx)+Math.abs(dy) > 5) drag.moved = true;
      const x = Math.max(0,Math.min(r.width-b.offsetWidth,drag.x+dx));
      const y = Math.max(0,Math.min(r.height-b.offsetHeight,drag.y+dy));
      n.x = x / r.width * 100; n.y = y / r.height * 100;
      b.style.left = n.x+"%"; b.style.top = n.y+"%";
      drawWires();
    });
    b.addEventListener("pointerup", e => {
      if(wireDrag && wireDrag.from===n.id){
        const target=document.elementFromPoint(e.clientX,e.clientY);
        const node=target && target.closest("[data-node-id]");
        const to=node && node.dataset.nodeId;
        wireDrag=null;b.dataset.suppressClick="true";
        if(to && to!==n.id) propose(n.id,to);
        else status("Connection drag cancelled. Drop on another node.");
        drawWires();return;
      }
      if(!drag || drag.id !== n.id) return;
      if(drag.moved) { b.dataset.suppressClick = "true"; layout[n.id] = {x:n.x,y:n.y}; saveLayout(); status("Layout saved locally. No service configuration changed."); }
      drag = null;
    });
    b.addEventListener("pointercancel", () => { drag=null;wireDrag=null;drawWires(); });
    return b;
  }
  function position(id) {
    const e = layer.querySelectorAll("[data-node-id]");
    const b = Array.from(e).find(x => x.dataset.nodeId === id);
    if(!b) return null;
    return {x:b.offsetLeft + b.offsetWidth/2, y:b.offsetTop + b.offsetHeight/2};
  }
  function path(a,b) {
    const dx = (b.x-a.x)*0.42;
    return "M "+a.x+" "+a.y+" C "+(a.x+dx)+" "+a.y+" "+(b.x-dx)+" "+b.y+" "+b.x+" "+b.y;
  }
  function drawWires() {
    wires.replaceChildren();
    const edges=[];
    nodes.filter(n => n.kind === "feature").forEach(n => {
      const rt=n.view.runtime || {}, grants=(n.view.identity && n.view.identity.grants) || [];
      if(rt.state === "RUNNING" && rt.health && rt.health.ready) edges.push({from:"processor",to:n.id,kind:"verified"});
      if($("#bw-authorized").checked && grants.some(g => g.status === "ACTIVE")) edges.push({from:"identity",to:n.id,kind:"authorized"});
    });
    proposals.forEach(p=>{if(byId(p.from)&&byId(p.to))edges.push({from:p.from,to:p.to,kind:"proposed"});});
    if(wireDrag && wireDrag.to){
      const a=position(wireDrag.from),b=wireDrag.to;
      if(a && b)edges.push({from:wireDrag.from,to:null,kind:"proposed", preview:b});
    }
    edges.forEach(link=>{
      const a=position(link.from),b=link.preview || position(link.to);if(!a||!b)return;
      const line=document.createElementNS("http://www.w3.org/2000/svg","path");
      line.setAttribute("class","bw-wire "+link.kind);line.setAttribute("d",path(a,b));wires.append(line);
    });
  }
  function renderGraph() {
    layer.replaceChildren(); nodes.forEach(n=>layer.append(nodeElement(n)));
    $("#bw-empty").hidden = nodes.length > 2;
    if(nodes.length === 2) put("bw-empty","No installed feature instances reported. Identity and Processor remain visible.");
    drawWires(); renderInspector(); renderProposals(); updateSources();
  }
  function kv(k,v) {
    const dl=$("#bw-fields");dl.append(element("dt","",k),element("dd","",safe(v)));
  }
  function renderInspector() {
    const n=byId(selected);if(!n)return;
    put("bw-name",n.name);
    const rt=n.view && n.view.runtime || {},r=n.view && n.view.record || {},iv=n.view && n.view.identity;
    put("bw-description",n.kind==="feature" ? "Installed feature instance. Information reported by Processor and Identity." : n.subtitle);
    $("#bw-fields").replaceChildren();
    if(n.kind==="feature"){
      kv("Installation",r.installation_id); kv("Feature",r.feature_id);
      kv("Version",r.version); kv("Runtime",currentStatus(n));kv("PID",rt.pid);
      kv("Health",rt.health ? (rt.health.ready ? "READY" : "NOT READY") : "Not reported");
      kv("File integrity",n.view.files);
      kv("Grants",iv ? ((iv.grants||[]).filter(g=>g.status==="ACTIVE").length+" active") : "Not reported");
      kv("Last exit",rt.last_exit && (rt.last_exit.reason||rt.last_exit.code));
    }else if(n.kind==="authority"){
      kv("Identity status",n.state);kv("Verified",engine && engine.verified ? "Yes" : "No");
      kv("Connection",engine && engine.connection && engine.connection.state);
    }else{
      kv("Version",n.info.version);kv("PID",n.info.pid);kv("Platform",n.info.platform);
      kv("Identity",engine && engine.connection && engine.connection.state);
    }
    const isFeature=n.kind==="feature",busy=["STARTING","STOPPING","RESTARTING"].includes(currentStatus(n));
    const running=currentStatus(n)==="RUNNING",allowed=isFeature && !busy && engine && engine.verified && !!iv;
    $("#bw-start").disabled=!allowed || running;$("#bw-stop").disabled=!allowed || !running;
    $("#bw-selftest").disabled=!allowed || !running;$("#bw-activity").disabled=!isFeature;
    layer.querySelectorAll(".bw-node").forEach(e=>e.classList.toggle("selected",e.dataset.nodeId===selected));
  }
  function renderProposals() {
    const box=$("#bw-proposals");box.replaceChildren();
    const valid=proposals.filter(p=>byId(p.from)&&byId(p.to));
    if(!valid.length){box.textContent="No inactive proposals.";return;}
    valid.forEach(p=>{
      const row=element("div","bw-proposal-row");
      row.append(element("span","",byId(p.from).name+" → "+byId(p.to).name+" (inactive)"));
      const btn=element("button","btn small","Remove");btn.type="button";
      btn.addEventListener("click",()=>{proposals=proposals.filter(x=>x!==p);saveProposals();renderProposals();drawWires();status("Visual proposal removed. No service settings changed.");});
      row.append(btn);box.append(row);
    });
  }
  function propose(from,to){
    if(proposals.some(p=>p.from===from&&p.to===to || p.from===to&&p.to===from)) status("That connection proposal already exists.");
    else{proposals.push({from,to,state:"PROPOSED_INACTIVE"});saveProposals();status("Connection saved as INACTIVE proposal. Approval and activation are not implemented.");}
    renderProposals();drawWires();
  }
  function pick(id) {
    if(mode==="remove" || mode==="connect"){
      if(!pending){pending=id;status("Select the destination node.");$("#bw-help").textContent="Choose the other node to "+(mode==="remove"?"remove the proposal.":"create an inactive connection proposal.");return;}
      if(pending===id){pending=null;status("Connection selection cancelled.");return;}
      const from=pending;pending=null;
      if(mode==="remove"){
        const before=proposals.length;
        proposals=proposals.filter(p=>!(p.from===from&&p.to===id || p.from===id&&p.to===from));
        status(before===proposals.length?"No proposal exists between those nodes.":"Proposal removed; live wiring unchanged.");
      }else{ propose(from,id); }
      saveProposals();renderProposals();drawWires();return;
    }
    selected=id;renderInspector();
    if(byId(id).kind==="feature") $("#bw-log-source").value=id;
  }
  function updateSources(){
    const select=$("#bw-log-source"),old=select.value;select.replaceChildren();
    const option=(id,label)=>{const o=element("option","",label);o.value=id;select.append(o);};
    option("processor","Processor (clean events)");
    nodes.filter(n=>n.kind==="feature").forEach(n=>option(n.id,n.name+" (activity)"));
    if(Array.from(select.options).some(o=>o.value===old))select.value=old;
    else if(byId(selected) && byId(selected).kind==="feature") select.value=selected;
  }
  async function refreshGraph() {
    if(loading || !ws || !ws.gate || !ws.gate.ok)return;
    loading=true;
    try {
      const s=await api("/api/state");
      if(!s.verified){engine=null;put("bw-identity","NOT VERIFIED");status("Identity not verified; graph details unavailable.");return;}
      engine=s;buildNodes(s);renderGraph();updateStats();
      put("bw-empty",nodes.length===2?"No installed feature instances reported.":"");
    } catch(e){status("Unable to read service state: "+e.message);put("bw-identity","UNKNOWN");}
    finally{loading=false;}
  }
  function updateStats(){
    const v=(engine&&engine.installations)||[];
    const installed=v.filter(x=>x.record && x.record.state==="INSTALLED");
    put("bw-installed",installed.length);put("bw-running",installed.filter(x=>x.runtime && x.runtime.state==="RUNNING").length);
    put("bw-grants",installed.reduce((n,x)=>n+((x.identity&&x.identity.grants)||[]).filter(g=>g.status==="ACTIVE").length,0));
    put("bw-identity",engine&&engine.verified?"VERIFIED":"NOT VERIFIED");
    const runs=ws&&ws.runs||[];put("bw-tests",runs.length?(runs[0].state+" · "+runs.length+" recent"):"No runs");
  }
  function readView(view){
    root.dataset.view=view;
    root.querySelectorAll("[data-bw-view]").forEach(b=>{const on=b.dataset.bwView===view;b.classList.toggle("on",on);b.setAttribute("aria-pressed",String(on));});
    if(view==="brain") requestAnimationFrame(drawWires);
  }
  async function runAction(action){
    const n=byId(selected);if(!n || n.kind!=="feature" || actionBusy)return;
    const r=n.view.record,st=currentStatus(n);
    if((action==="stop" && st!=="RUNNING") || (action==="start" && st==="RUNNING"))return;
    if(action==="stop" && !confirm("Stop "+n.name+"? Any in-progress work may be interrupted."))return;
    actionBusy=true;["bw-start","bw-stop","bw-selftest"].forEach(k=>$("#"+k).disabled=true);
    status(action+" requested for "+n.name+"…");
    try {await post("/api/installations/"+encodeURIComponent(r.installation_id)+"/"+(action==="selftest"?"self-test":action));status("Request accepted by Processor. Refreshing actual runtime status…");await refreshGraph();}
    catch(e){status(action+" failed: "+e.message);}
    finally{actionBusy=false;renderInspector();}
  }
  function cleanLine(e){
    return [e.time || "", (e.level || "info").toUpperCase(),e.component || "",e.event || "",e.detail || ""].filter(Boolean).join(" | ");
  }
  async function loadLog(){
    const source=$("#bw-log-source").value;
    put("bw-log-status","Reading redacted events…");lastClean="";
    try{
      let txt;
      if(source==="processor"){
        const j=await api("/api/ws/diagnostics");
        txt=(j.events || []).map(cleanLine).join("\n");
      }else{
        if(!byId(source))throw Error("Selected installation is not present.");
        const j=await api("/api/installations/"+encodeURIComponent(source)+"/activity");
        txt=j.text || "";
      }
      lastClean=txt || "";
      $("#bw-log").value=lastClean || "No telemetry reported for this source.";
      put("bw-log-status",lastClean?(lastClean.split("\n").length+" redacted line(s)"):"No telemetry reported");
    }catch(e){lastClean="";$("#bw-log").value="Could not load diagnostics: "+e.message;put("bw-log-status","Unavailable — not a clean log");}
  }
  async function copyClean(){
    await loadLog();
    if(!lastClean){status("No clean diagnostic text is available to copy.");return;}
    try{await navigator.clipboard.writeText(lastClean);status("Clean log copied. Review it for project names or sensitive details before sending.");}
    catch(e){const box=$("#bw-log");box.focus();box.select();status("Clipboard unavailable. Diagnostic text selected; press Ctrl+C to copy.");}
  }
  async function copySupportReport(){
    await loadLog();
    const g=ws && ws.gate || {}, installed=nodes.filter(n=>n.kind==="feature");
    const lines=[
      "TASS Brain Workstation — review before sharing",
      "Captured: "+new Date().toISOString(),
      "Processor version: "+safe(engine && engine.processor && engine.processor.version),
      "Identity: "+(engine && engine.verified ? "VERIFIED" : "NOT VERIFIED"),
      "GitHub status: "+safe(g.github_state),
      "Installed features: "+installed.length,
      "Recent test runs: "+safe(ws && ws.runs && ws.runs.length),
      "",
      "SERVICE STATUS"
    ];
    installed.forEach(n=>{
      const rt=n.view.runtime || {},h=rt.health || {};
      lines.push(n.name+" | "+currentStatus(n)+" | health "+(h.ready?"READY":"NOT VERIFIED")+" | version "+safe(n.view.record.version));
    });
    lines.push("","CLEAN DIAGNOSTICS ("+($("#bw-log-source").selectedOptions[0]?.textContent || "unknown")+")");
    lines.push(lastClean || "No sanitized diagnostic events available.");
    const report=lines.join("\n");
    try{await navigator.clipboard.writeText(report);status("Support report copied; review before sending. No raw file text or test stdout included.");}
    catch(e){$("#bw-log").value=report;$("#bw-log").focus();$("#bw-log").select();status("Clipboard unavailable; support report selected. Press Ctrl+C to copy.");}
  }
  window.addEventListener("bw:workstation-state",e=>{
    ws=e.detail;updateStats();
    if(!ws.gate || !ws.gate.ok){engine=null;nodes=[];layer.replaceChildren();wires.replaceChildren();status("Workstation locked — verify Identity and GitHub.");return;}
    if(!engine) refreshGraph();
    else if(Date.now() - (window._bwLastRefresh||0) > 8000)refreshGraph();
  });
  const originalRefresh=refreshGraph;
  refreshGraph=async function(){window._bwLastRefresh=Date.now();return originalRefresh();};
  $("#bw-refresh").onclick=()=>{refreshGraph();loadLog();};
  $("#bw-reset").onclick=()=>{layout={};saveLayout();if(engine){buildNodes(engine);renderGraph();}status("Default layout restored. No service settings changed.");};
  $("#bw-updates").onclick=()=>status("To check for Workbench updates use launcher menu 6; to apply the approved update use menu 8. This page does not silently update code.");
  root.querySelectorAll("[data-bw-view]").forEach(b=>b.onclick=()=>readView(b.dataset.bwView));
  root.querySelectorAll("[data-bw-mode]").forEach(b=>b.onclick=()=>{
    mode=b.dataset.bwMode;pending=null;
    root.querySelectorAll("[data-bw-mode]").forEach(x=>x.classList.toggle("on",x===b));
    $("#bw-help").textContent=mode==="move"?"Drag nodes to arrange. Click to inspect.":mode==="connect"?"Click a source node and then a destination. The link is a saved inactive proposal.":"Click both ends of an inactive proposal to remove it.";
  });
  $("#bw-authorized").onchange=drawWires;
  $("#bw-start").onclick=()=>runAction("start");
  $("#bw-stop").onclick=()=>runAction("stop");
  $("#bw-selftest").onclick=()=>runAction("selftest");
  $("#bw-activity").onclick=()=>{$("#bw-log-source").value=selected;copyClean();};
  $("#bw-reload-log").onclick=loadLog;
  $("#bw-copy-log").onclick=copyClean;
  $("#bw-copy-report").onclick=copySupportReport;
  $("#bw-log-toggle").onclick=()=>{
    const section=$(".bw-diag"),collapsed=section.classList.toggle("collapsed");
    $("#bw-log-toggle").textContent=collapsed?"Expand":"Collapse";
    $("#bw-log-toggle").setAttribute("aria-expanded",String(!collapsed));
  };
  $("#bw-log-source").onchange=loadLog;
  readView("brain");
  window.addEventListener("resize",()=>{if(root.dataset.view==="brain")drawWires();});
})();