'use strict';
const $ = id => document.getElementById(id);
const csrf = document.querySelector('meta[name="playground-csrf"]').content;
const state = {
  connected: false, serverReachable: false, busy: false, eventSource: null,
  lastIds: new Set(), ideas: [], scene: 'aurora', selectedNode: null,
  streaming: null, toastTimer: null, lastStatus: {}, started: false,
};
const nodeInfo = {
  identity: {symbol:'◈',title:'Identity · who TASS is',body:'The identity records and permissions are a separate authority. The repo label shown here is from a local mirror, not proof of an authorization grant.'},
  model: {symbol:'✶',title:'Local model · how TASS responds',body:'Nemotron runs locally in the existing Docker environment. This page reads its availability; the model does not control this browser directly.'},
  library: {symbol:'▤',title:'Library · reusable abilities',body:'Small, discoverable notebook books. TASS can inspect them and propose a run, but approval and execution remain in the original Workshop.'},
  projects: {symbol:'⬡',title:'Projects · real builds',body:'Verified files are mirrored by the existing Liaison. A project is more than an idea: it should have entry code, an observed result, and test evidence.'},
  workshop: {symbol:'◎',title:'Workshop · the trusted review surface',body:'The existing Workshop handles code proposals and execution review. The Atrium only sends text messages; it cannot approve or run code.'},
};

function notice(message) {
  const node = $('toast'); node.textContent = message; node.hidden = false;
  if (state.toastTimer) clearTimeout(state.toastTimer);
  state.toastTimer = setTimeout(() => { node.hidden = true; }, 4500);
}
function formatTime() {
  return new Date().toLocaleTimeString([], {hour:'2-digit', minute:'2-digit'});
}
function addActivity(title, detail='', tone='') {
  const holder = $('activityStream');
  if (holder.querySelector('.quiet-activity')) holder.textContent = '';
  const item = document.createElement('div'); item.className = 'activity-item ' + tone;
  const strong = document.createElement('strong'); strong.textContent = `${title} · ${formatTime()}`;
  const span = document.createElement('span'); span.textContent = detail;
  item.append(strong, span); holder.prepend(item);
  while (holder.children.length > 12) holder.lastChild.remove();
}
function addBubble(kind, text, title='') {
  const el = document.createElement('div'); el.className = 'bubble ' + kind;
  if (title) { const head = document.createElement('span'); head.className = 'who'; head.textContent = title; el.append(head); }
  const body = document.createElement('span'); body.className = 'bubble-body'; body.textContent = text;
  el.append(body); const chat = $('chatStream');
  if (chat.querySelector('.welcome-message') && kind !== 'note') chat.querySelector('.welcome-message').remove();
  chat.appendChild(el); chat.scrollTop = chat.scrollHeight;
  while (chat.children.length > 42) chat.firstElementChild.remove();
  return el;
}
function updateStream(text) {
  if (!state.streaming) state.streaming = addBubble('tass streaming','','TASS · answering');
  state.streaming.querySelector('.bubble-body').textContent += String(text||'');
  $('chatStream').scrollTop = $('chatStream').scrollHeight;
}
function closeStream() {
  if (state.streaming) { state.streaming.remove(); state.streaming = null; }
}
function drawTimelineEvent(ev, initial=false) {
  if (!ev || !ev.type) return;
  if (Number.isInteger(ev.id)) {
    if (state.lastIds.has(ev.id)) return;
    state.lastIds.add(ev.id);
    if (state.lastIds.size > 2500) state.lastIds.clear();
  }
  switch (ev.type) {
    case 'user':
      closeStream(); addBubble('me', ev.text || '(empty)', 'YOU'); break;
    case 'reply':
      closeStream(); addBubble('tass', ev.text || '(no reply text)', 'TASS · REAL MODEL RESPONSE');
      if (!initial) addActivity('TASS replied', ev.action ? 'Proposed: ' + ev.action : 'Model answer received', 'model');
      break;
    case 'reply_delta':
      if (!initial) updateStream(ev.delta); break;
    case 'thought_start':
      if (!initial) addActivity('Thinking started','Model-generated text is not independent evidence','model'); break;
    case 'thought':
      if (!initial) addActivity('Thinking finished', `${ev.seconds ?? '—'} seconds`, 'model'); break;
    case 'spark':
      if (!initial) addActivity('New spark', ev.kind || 'Turn started'); break;
    case 'book':
      if (!initial) addActivity('Tool discovered', ev.title || ev.path || 'Book opened','model'); break;
    case 'proposal':
      if (!initial) {
        addActivity('Approval required','Review the exact code in the original Workshop','warn');
        addBubble('review','TASS proposed an action. Use the original Workshop to inspect and approve or reject the code.','REVIEW REQUIRED');
        pollState();
      }
      break;
    case 'result':
      addBubble('result', (ev.ok ? '✓ Run reported OK' : '✗ Run reported an error') + '\n' +
        (ev.text || ev.error || 'No output text'), 'WORKSHOP EXECUTION RESULT');
      if (!initial) addActivity('Run result', ev.ok ? 'Reported successful; verify actual output' : (ev.error || 'Failed'), ev.ok ? 'model' : 'warn');
      break;
    case 'notice':
      if (!initial) addActivity('Workshop notice', ev.text || 'New notice'); break;
    case 'memory':
      if (!initial) addActivity('Memory event', 'A memory event occurred in the existing Workshop'); break;
    case 'mind_pr':
      if (!initial) addActivity('Mind change', ev.ok ? 'GitHub PR reported' : 'PR creation not confirmed'); break;
    case 'error':
      if (!initial) { addBubble('error', ev.text || 'An error occurred', 'WORKSHOP ERROR'); addActivity('Error', ev.text || '', 'warn'); }
      break;
    case 'busy': state.busy = !!ev.busy; updateComposer(); break;
  }
}
function setMode(reachable, s) {
  const wasConnected = state.connected;
  state.serverReachable = reachable;
  state.connected = !!(reachable && s && s.model_ok);
  state.busy = !!(s && s.busy);
  state.lastStatus = s || {};
  const box = $('connection');
  box.classList.toggle('live', state.connected);
  box.classList.toggle('offline', !state.connected);
  $('connectionLabel').textContent = state.connected ? 'LIVE TASS CONNECTED' : reachable ? 'WORKSHOP ONLINE · MODEL OFFLINE' : 'SANDBOX · TASS OFFLINE';
  $('chatMode').textContent = state.connected ? 'LIVE' : 'SANDBOX';
  $('chatMode').classList.toggle('live', state.connected);
  $('signalLabel').textContent = state.connected ? 'REAL WORKSHOP EVENTS' : 'NO LIVE MODEL';
  $('identityName').textContent = s?.identity?.name || 'TASS';
  $('identitySubtitle').textContent = s?.identity?.repo || 'Local companion';
  const books = s?.mind?.library?.length;
  const projects = s?.mind?.projects?.length;
  $('bookCount').textContent = reachable ? String(books ?? '0') : '—';
  $('projectCount').textContent = reachable ? String(projects ?? '0') : '—';
  updateComposer();
  if (!wasConnected && state.connected) addActivity('Connected','Same TASS session as the existing Workshop','model');
  if (wasConnected && !state.connected) addActivity('Disconnected','Your ideas still work locally','warn');
  if (state.selectedNode) showNode(state.selectedNode);
}
function updateComposer() {
  $('sendButton').disabled = !state.connected || state.busy;
  $('composerHint').textContent = !state.serverReachable
    ? 'Offline sandbox · Explore and draft ideas while TASS is unavailable.'
    : !state.connected ? 'Workshop reachable; local model not ready. Messages are disabled.'
    : state.busy ? 'TASS is working on a turn. Messages resume when it finishes.'
    : 'Connected to your real TASS session · You control every send.';
}
function setPending(proposals) {
  const n = Array.isArray(proposals) ? proposals.length : 0;
  $('pendingReview').hidden = !n;
  $('pendingCount').textContent = `${n} action${n === 1 ? '' : 's'} waiting for review`;
}
async function pollState() {
  try {
    const response = await fetch('/bridge/state', {cache:'no-store'});
    if (!response.ok) throw new Error('Workshop not available');
    const payload = await response.json();
    setMode(true, payload.status || {});
    setPending(payload.proposals || []);
    if (!state.started) {
      state.started = true;
      const important = (payload.timeline || []).filter(e => ['user','reply','result'].includes(e.type)).slice(-22);
      important.forEach(ev => drawTimelineEvent(ev, true));
      if (!important.length) addBubble('note','Connected. The next message goes to the real local TASS.','ATRIUM');
      startEvents();
    } else {
      // SSE may disconnect without the state endpoint failing. Backfill missed activity.
      (payload.timeline || []).filter(e => ['user','reply','result','proposal','error'].includes(e.type))
        .forEach(ev => drawTimelineEvent(ev, false));
    }
  } catch (err) {
    setMode(false, {});
    $('pendingReview').hidden = true;
    if (state.eventSource) { state.eventSource.close(); state.eventSource = null; }
    state.started = false;
  }
}
function startEvents() {
  if (state.eventSource) state.eventSource.close();
  const es = new EventSource('/bridge/events'); state.eventSource = es;
  es.addEventListener('message', event => {
    try { const data = JSON.parse(event.data); drawTimelineEvent(data); }
    catch (_) { addActivity('Unreadable event','The Workshop sent an unrecognized event','warn'); }
  });
  es.addEventListener('error', () => {
    $('signalLabel').textContent = 'RECONNECTING EVENT STREAM';
    // Browser EventSource reconnects by itself. Polling also checks the backend.
  });
  es.addEventListener('open', () => { $('signalLabel').textContent = 'REAL WORKSHOP EVENTS'; });
}
async function sendText(text) {
  text = (text || '').trim();
  if (!text) return false;
  if (!state.connected) { notice('The Workshop model is offline. Save the idea locally or start TASS first.'); return false; }
  if (state.busy) { notice('TASS is still processing a turn.'); return false; }
  $('sendButton').disabled = true;
  try {
    const response = await fetch('/bridge/message', {
      method:'POST', headers:{'Content-Type':'application/json','X-Playground-CSRF':csrf},
      body:JSON.stringify({text}),
    });
    const data = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(data.error || `HTTP ${response.status}`);
    $('messageInput').value = '';
    state.busy = true;
    updateComposer();
    addActivity('Message delivered', 'Waiting for the real TASS response', 'model');
    return true;
  } catch (err) {
    notice(`Cannot send: ${err.message}`);
    updateComposer();
    return false;
  }
}
let lastDiagnostic = 'Not checked';
async function diagnose() {
  $('doctorSummary').textContent = 'Checking the running services...';
  try {
    const response = await fetch('/bridge/diagnostics', {cache:'no-store'});
    if (!response.ok) throw new Error('Atrium diagnostic request failed');
    const check = await response.json();
    const msg = `Atrium: ${check.atrium} · Workshop: ${check.workshop} · Model: ${check.model} · Identity: ${check.identity}. ${check.next_step}`;
    lastDiagnostic = msg;
    $('doctorSummary').textContent = msg;
    addActivity('Connection checked', `${check.workshop} / model ${check.model}`, check.model === 'ready' ? 'model' : 'warn');
    return check;
  } catch (err) {
    lastDiagnostic = 'Atrium diagnostics could not run. Check whether the Atrium server on port 8791 is still running.';
    $('doctorSummary').textContent = lastDiagnostic;
    return null;
  }
}
function draftPrompt(text) {
  $('messageInput').value = text;
  $('messageInput').focus();
  $('messageInput').scrollIntoView({block:'nearest',behavior:'smooth'});
  notice('Draft ready. Nothing has been sent or executed.');
}
function showNode(key) {
  state.selectedNode = key;
  document.querySelectorAll('.node').forEach(el => el.classList.toggle('selected', el.dataset.node === key));
  const n = nodeInfo[key]; if (!n) return;
  $('inspectorIcon').textContent = n.symbol;
  $('inspectorTitle').textContent = n.title;
  $('inspectorBody').textContent = n.body;
  const s = state.lastStatus; let badge='CONCEPT';
  if (key === 'model') badge = s.model_ok ? 'MODEL ONLINE' : 'NOT VERIFIED';
  if (key === 'identity') badge = s.identity?.repo ? 'REPO MIRRORED' : 'NOT VERIFIED';
  if (key === 'library') badge = state.serverReachable ? `${s.mind?.library?.length || 0} BOOKS` : 'NOT VERIFIED';
  if (key === 'projects') badge = state.serverReachable ? `${s.mind?.projects?.length || 0} PROJECTS` : 'NOT VERIFIED';
  if (key === 'workshop') badge = state.serverReachable ? 'API REACHABLE' : 'OFFLINE';
  $('inspectorState').textContent = badge;
}
function showView(view) {
  for (const name of ['cosmos','ideas','lab']) {
    $(`view-${name}`).classList.toggle('is-visible', name === view);
  }
  document.querySelectorAll('.nav-item').forEach(b => b.classList.toggle('is-active', b.dataset.view === view));
}
function renderIdeas() {
  const holder = $('ideaCards'); holder.replaceChildren();
  $('gardenCount').textContent = `${state.ideas.length} idea${state.ideas.length===1?'':'s'}`;
  if (!state.ideas.length) {
    const wrap = document.createElement('div'); wrap.className='empty-garden';
    const inner=document.createElement('div');
    const symbol=document.createElement('div');symbol.className='garden-flower';symbol.textContent='✺';
    const title=document.createElement('b');title.textContent='Every world begins with an idea.';
    const subtitle=document.createElement('span');subtitle.textContent='Try planting one. This garden does not run code.';
    inner.append(symbol,title,subtitle);wrap.append(inner);holder.append(wrap);return;
  }
  state.ideas.forEach((idea,index) => {
    const card=document.createElement('div'); card.className='idea';
    const tag=document.createElement('small'); tag.textContent=`IDEA ${String(index+1).padStart(2,'0')} / LOCAL`;
    const desc=document.createElement('div');desc.className='idea-text';desc.textContent=idea;
    const actions=document.createElement('div');actions.className='idea-bottom';
    const draft=document.createElement('button');draft.type='button';draft.textContent='Ask TASS ↗';
    draft.onclick=()=>draftPrompt(`TASS, let's explore this idea together: ${idea}\nWhat would you do first? Propose a single safe step and wait for my approval.`);
    const remove=document.createElement('button');remove.type='button';remove.textContent='Remove';
    remove.onclick=()=>{state.ideas.splice(index,1);renderIdeas();};
    actions.append(draft,remove);card.append(tag,desc,actions);holder.append(card);
  });
}
function exportIdeas() {
  const body=JSON.stringify({title:'TASS Atrium ideas',exported_at:new Date().toISOString(),ideas:state.ideas},null,2);
  const url=URL.createObjectURL(new Blob([body],{type:'application/json'}));
  const a=document.createElement('a');a.href=url;a.download='tass-atrium-ideas.json';a.click();
  setTimeout(()=>URL.revokeObjectURL(url),5000);
  notice('Your idea garden was exported locally.');
}
function setScene(scene) {
  state.scene=scene;document.body.dataset.scene=scene;
  document.querySelectorAll('[data-scene]').forEach(btn=>btn.classList.toggle('chosen',btn.dataset.scene===scene));
  notice(`${scene[0].toUpperCase()+scene.slice(1)} atmosphere selected. Your model was not changed.`);
}
function initStars() {
  const canvas=$('stars'); const host=$('constellation');if(!canvas||!host)return;
  const ctx=canvas.getContext('2d');if(!ctx)return;
  const reduced=window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  let stars=[]; let lastSize='';
  function resize() {
    const rect=host.getBoundingClientRect();const dpr=Math.min(window.devicePixelRatio||1,2);
    const size=`${Math.floor(rect.width)}x${Math.floor(rect.height)}x${dpr}`;
    if(size===lastSize)return;lastSize=size;
    canvas.width=Math.max(1,Math.floor(rect.width*dpr));canvas.height=Math.max(1,Math.floor(rect.height*dpr));
    ctx.setTransform(dpr,0,0,dpr,0,0);
    stars=Array.from({length:Math.floor(rect.width*rect.height/2600)},(_,i)=>({
      x:Math.random()*rect.width,y:Math.random()*rect.height,r:.5+Math.random()*1.1,
      phase:Math.random()*6.3,speed:.12+Math.random()*.6,
    }));
  }
  function frame(time) {
    resize();const w=host.clientWidth,h=host.clientHeight;ctx.clearRect(0,0,w,h);
    for(const st of stars){const p=reduced?0.75:(.4+.35*Math.sin(st.phase+time*.0007*st.speed));
      ctx.globalAlpha=Math.max(.2,p);ctx.fillStyle=st.r>1.2?'#8ceddf':'#d9d3ff';
      ctx.beginPath();ctx.arc(st.x,st.y,st.r,0,Math.PI*2);ctx.fill();}
    ctx.globalAlpha=1;if(!reduced)requestAnimationFrame(frame);
  }
  requestAnimationFrame(frame);
  if(reduced) window.addEventListener('resize',()=>{lastSize='';frame(0)});
}
function boot() {
  document.querySelectorAll('.nav-item').forEach(btn=>btn.addEventListener('click',()=>showView(btn.dataset.view)));
  document.querySelectorAll('.node').forEach(btn=>btn.addEventListener('click',()=>showNode(btn.dataset.node)));
  document.querySelectorAll('.mission').forEach(btn=>btn.addEventListener('click',()=>draftPrompt(btn.dataset.prompt)));
  document.querySelectorAll('.scene-card').forEach(btn=>btn.addEventListener('click',()=>setScene(btn.dataset.scene)));
  $('chatForm').addEventListener('submit',async ev=>{ev.preventDefault();await sendText($('messageInput').value);});
  $('messageInput').addEventListener('keydown',ev=>{
    if(ev.key==='Enter'&&!ev.shiftKey){ev.preventDefault();$('chatForm').requestSubmit();}
  });
  $('ideaForm').addEventListener('submit',ev=>{ev.preventDefault();const v=$('ideaInput').value.trim();if(!v)return;
    state.ideas.push(v);$('ideaInput').value='';renderIdeas();notice('Idea planted. No code ran.');});
  $('exportIdeas').addEventListener('click',exportIdeas);
  $('clearIdeas').addEventListener('click',()=>{
    if(!state.ideas.length)return;if(confirm('Clear your local ideas? This cannot be undone unless exported.')){state.ideas=[];renderIdeas();}
  });
  $('diagnoseButton').addEventListener('click', diagnose);
  $('firstTaskButton').addEventListener('click', () => draftPrompt('TASS, I want to do one real task together. Please check whether you can see projects/hello-tass. If it exists, OPEN that project, explain what it changes, then PROPOSE running it and wait for my review. If it does not exist, tell me what project you can actually see. Do not claim success until you receive a real execution result.'));
  $('copyReportButton').addEventListener('click', () => {
    const report = 'TASS Atrium connection check\n' + lastDiagnostic + '\nWorkshop: http://127.0.0.1:8790/\nAtrium: http://127.0.0.1:8791/';
    if (navigator.clipboard?.writeText) navigator.clipboard.writeText(report).then(()=>notice('Connection report copied.')).catch(()=>{prompt('Copy the report:',report);});
    else prompt('Copy the report:',report);
  });
  $('sendScene').addEventListener('click',()=>draftPrompt(`TASS, I tried an interactive ${state.scene} color scene in our Atrium playground. Can we think about a small real interface inspired by this look? Please propose one safe first step.`));
  showNode('model'); renderIdeas();initStars(); pollState();
  setInterval(pollState,5000);
  diagnose();
}
boot();