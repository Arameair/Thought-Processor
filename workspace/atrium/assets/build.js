'use strict';
const $ = id => document.getElementById(id);
const csrf = document.querySelector('meta[name="playground-csrf"]').content;
const session = {
  ready:false, busy:false, fullAuto:false, lastId:0, seen:new Set(), started:false, eventStream:null,
  request:'', requestStart:Infinity, asked:false, answered:false, proposed:false, executed:false,
  library:[], audit:[], streaming:null, shareAvailable:false,
};
function notify(message) {
  const node=$('toast'); node.textContent=message; node.hidden=false;
  clearTimeout(notify.timeout); notify.timeout=setTimeout(()=>node.hidden=true,4800);
}
function renderStatus() {
  $('modelStatus').textContent=session.ready ? (session.fullAuto?'⚠ FULL AUTO IS ON':session.busy?'◌ TASS is working':'● TASS connected') : '○ TASS unavailable';
  $('modelStatus').classList.toggle('live',session.ready);
  $('modelStatus').classList.toggle('offline',!session.ready);
  $('askTass').disabled=!session.ready || session.busy || session.fullAuto;
  $('streamStatus').textContent=session.fullAuto?'Disable Full Auto in Workshop before building':session.ready ? (session.busy?'TASS is thinking':'Listening for real results') : 'No live model — work is not simulated';
}
function updateProgress() {
  const flags={asked:session.asked,answered:session.answered,proposed:session.proposed,executed:session.executed};
  const order=['asked','answered','proposed','executed'];
  const next=order.find(k=>!flags[k]) || null;
  document.querySelectorAll('#steps li').forEach(el=>{
    el.classList.toggle('complete',!!flags[el.dataset.step]);
    el.classList.toggle('current',el.dataset.step===next);
  });
  $('progressNote').textContent=!session.request?'Start by describing one small thing you want TASS to build.' :
    !session.asked?'Waiting for Workshop to record your request.':
    !session.answered?'TASS has your request. Waiting for his actual reply.':
    !session.proposed?'You have a real TASS reply. Ask him to propose one concrete step if none was offered.':
    !session.executed?'An action was proposed. Examine the code and approve or reject it in Workshop.':
    'Workshop reported an execution result. Check its real output before calling the task complete.';
}
function bubble(type,who,text,fail=false) {
  const log=$('conversation');
  if(log.querySelector('.empty-state')) log.replaceChildren();
  const item=document.createElement('div');
  item.className='bubble '+type+(fail?' failed':'');
  const head=document.createElement('div');head.className='who';head.textContent=who;
  const body=document.createElement('div');body.className='body';body.textContent=String(text||'').slice(0,6000);
  item.append(head,body);log.append(item);
  while(log.children.length>35)log.firstElementChild.remove();
  log.scrollTop=log.scrollHeight;
  return item;
}
function closeStream() {if(session.streaming){session.streaming.remove();session.streaming=null;}}
function handleEvent(ev) {
  if(!ev||!ev.type)return;
  if(ev.type==='reply_delta') {
    if(!session.streaming)session.streaming=bubble('reply','TASS • responding','');
    session.streaming.querySelector('.body').textContent+=String(ev.delta||'');
    return;
  }
  if(ev.type==='busy'){session.busy=!!ev.busy;renderStatus();return;}
  if(!Number.isInteger(ev.id)||session.seen.has(ev.id))return;
  session.seen.add(ev.id);session.lastId=Math.max(session.lastId,ev.id);
  if(session.seen.size>3000)session.seen.clear();
  const inTask=ev.id>session.requestStart;
  if(ev.type==='user'){
    bubble('user','DUSTIN',ev.text||'(no text)');
    if(inTask)session.asked=true;
  }
  else if(ev.type==='reply'){
    closeStream();bubble('reply','TASS • real model',ev.text||'(empty reply)');
    if(inTask)session.answered=true;
  }
  else if(ev.type==='result'){
    bubble('result',ev.ok===true?'WORKSHOP • EXECUTOR REPORTED OK':'WORKSHOP • EXECUTION FAILED',
      (ev.label||'Run')+'\n'+(ev.text||ev.error||'No result detail'),ev.ok!==true);
    if(inTask)session.executed=true;
    fetchEvidence();
  }
  else if(ev.type==='proposal' && inTask) session.proposed=true;
  else if(ev.type==='error')bubble('result','WORKSHOP • ERROR',ev.text||'Unknown error',true);
  if(inTask)updateProgress();
}
function renderProposals(proposals) {
  const pending=(Array.isArray(proposals)?proposals:[]).filter(p=>p.status==='pending' || !p.status);
  $('pending').hidden=!pending.length;
  if(!pending.length)return;
  const p=pending[pending.length-1];
  $('pendingTitle').textContent=p.label||'TASS has proposed an action';
  $('pendingDetail').textContent=(p.source||'Workshop proposal')+' — The exact code is reviewable only in Workshop.';
}
function renderAbilities() {
  const host=$('abilities');host.replaceChildren();
  if(!session.library.length){host.textContent='No live library catalog found.';return;}
  for(const book of session.library){
    const card=document.createElement('div');card.className='ability';
    const left=document.createElement('div');
    const name=document.createElement('b');name.textContent=book.title||book.path;
    const meta=document.createElement('span');meta.textContent=book.path||'';
    left.append(name,meta);
    const indicator=document.createElement('em');
    const results=new Map();
    session.audit.filter(e=>e.book===book.path).forEach(e=>results.set(e.cell,e.outcome));
    const passed=[...results.values()].filter(x=>x==='pass').length;
    const failed=[...results.values()].filter(x=>x==='fail').length;
    const count=Number.isInteger(book.cells)?book.cells:null;
    indicator.textContent=failed?failed+' reported failure'+(failed!==1?'s':''):
      count!==null && count>0 && passed>=count ? 'All cells reported OK' :
      passed?passed+' cell'+(passed!==1?'s':'')+' reported OK':'NOT TESTED';
    card.append(left,indicator);host.append(card);
  }
}
async function fetchEvidence() {
  try{
    const res=await fetch('/bridge/evidence',{cache:'no-store'});
    if(!res.ok)throw Error('Audit unavailable');
    const data=await res.json();
    session.audit=Array.isArray(data.records)?data.records:[];
    renderAbilities();
  }catch(_){$('abilities').textContent='Audit unavailable. Existing capability records cannot be verified.';}
}
async function poll() {
  try {
    const res=await fetch('/bridge/state',{cache:'no-store'});
    if(!res.ok)throw Error('Workshop unreachable');
    const data=await res.json();const stat=data.status||{};
    session.ready=stat.model_ok===true;session.busy=stat.busy===true;session.fullAuto=stat.full_auto===true;
    session.library=stat.mind?.library||[];
    renderStatus();renderProposals(data.proposals);
    if(!session.started) {
      session.started=true;
      const prior=(data.timeline||[]).filter(e=>['user','reply','result','error'].includes(e.type)).slice(-22);
      for(const ev of prior)handleEvent(ev);
      if(!prior.length)bubble('reply','ATRIUM','Connected to TASS. Tell him what you want to build.');
      fetchEvidence();
      startEvents();
    } else {
      // Recover finalized messages if the live event stream disconnected.
      for(const ev of (data.timeline||[]).filter(e=>['user','reply','result','proposal','error'].includes(e.type)))
        handleEvent(ev);
    }
  }catch(_){
    session.ready=false;session.busy=false;renderStatus();
    $('pending').hidden=true;
    $('progressNote').textContent='Workshop unavailable. Open port 8790 to diagnose; no results are being invented.';
  }
}
function startEvents() {
  if(session.eventStream)return;
  const stream=new EventSource('/bridge/events');session.eventStream=stream;
  stream.onmessage=e=>{
    try{handleEvent(JSON.parse(e.data));}catch(_){}
  };
  stream.onerror=()=>{$('streamStatus').textContent='Event stream reconnecting; history polling remains active.';};
}
async function askTass(ev) {
  ev.preventDefault();
  const task=$('requestText').value.trim();
  if(!task)return notify('Describe one thing you would like to build.');
  if(!session.ready)return notify('TASS is offline; open the Workshop before sending.');
  if(session.busy)return notify('TASS is already working on another request.');
  if(session.fullAuto)return notify('Full Auto is enabled. Turn it off in Workshop so you can approve code first.');
  $('askTass').disabled=true;
  const message='TASS, I want to build this with you: '+task+
    '\nPlease tell me what existing library or project ability could help, and propose ONE concrete, safe step that I can review. Do not execute code without my approval. Do not claim work was done until there is a real Workshop result.';
  try {
    const res=await fetch('/bridge/message',{method:'POST',
      headers:{'Content-Type':'application/json','X-Playground-CSRF':csrf},
      body:JSON.stringify({text:message})});
    const data=await res.json();
    if(!res.ok)throw Error(data.error||'Workshop rejected the request');
    session.request=task;
    session.requestStart=session.lastId;
    session.asked=false;session.answered=false;session.proposed=false;session.executed=false;
    session.busy=true;renderStatus();updateProgress();
    notify('Sent to the real TASS. Waiting for Workshop events.');
  }catch(err){notify('Could not send: '+err.message);session.busy=false;renderStatus();}
}
function draftPrompt(value) {
  $('requestText').value=value;
  $('shareConsent').checked=false;
  updatePreview();
  $('requestText').focus();
}
function updatePreview(){
  const value=$('requestText').value.trim();
  $('briefPreview').textContent=value||'Write a task above. Only that text will be shared, not chat history or model internals.';
  $('shareBrief').disabled=!value||value.length>1200||!$('shareConsent').checked;
}
async function checkShare() {
  try{
    const r=await fetch('/bridge/share-status',{cache:'no-store'});
    const data=await r.json();
    session.shareAvailable=data.available===true;
    if(!session.shareAvailable)$('sharingStatus').textContent='Sharing is not configured yet. An owner must create the private request directory and start the optional collaboration service.';
  }catch(_){session.shareAvailable=false;}
}
async function shareBrief() {
  const task=$('requestText').value.trim();
  if(!task||!$('shareConsent').checked)return notify('Review and approve the exact text first.');
  $('shareBrief').disabled=true;
  try{
    const r=await fetch('/bridge/share-brief',{method:'POST',
      headers:{'Content-Type':'application/json','X-Playground-CSRF':csrf},
      body:JSON.stringify({consent:true,task})});
    const data=await r.json();
    if(!r.ok)throw Error(data.error||'Sharing unavailable');
    $('sharingStatus').textContent='Queued locally (ID '+data.id+'). Not yet posted: the optional collaboration feed and Identity Liaison must deliver it to private TASS issue #19. Then ask ChatGPT to review that issue.';
    notify('Brief queued with your explicit permission.');
    $('shareConsent').checked=false;updatePreview();
  }catch(err){$('sharingStatus').textContent=err.message;notify('Not shared: '+err.message);updatePreview();}
}
function init() {
  $('requestForm').addEventListener('submit',askTass);
  $('requestText').addEventListener('input',updatePreview);
  $('shareConsent').addEventListener('change',updatePreview);
  $('shareBrief').addEventListener('click',shareBrief);
  $('refresh').addEventListener('click',()=>{poll();fetchEvidence();checkShare();});
  document.querySelectorAll('[data-prompt]').forEach(b=>b.addEventListener('click',()=>draftPrompt(b.dataset.prompt)));
  updatePreview();updateProgress();renderStatus();poll();checkShare();
  setInterval(poll,3500);setInterval(fetchEvidence,18000);
}
init();