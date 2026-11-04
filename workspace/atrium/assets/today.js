'use strict';
const $ = id => document.getElementById(id);
const csrf = document.querySelector('meta[name="playground-csrf"]').content;
const fields = ['title','why','next_step','expected','status','result','notes','evidence'];
const formIds = {
  title:'taskTitle', why:'taskWhy', next_step:'taskNext', expected:'taskExpected',
  status:'taskStatus', result:'taskResult', notes:'taskNotes', evidence:'taskEvidence'
};
const ui = { board:null, dirty:false, inFlight:false, live:null, sharing:false, lastTimeline:[],
  sending:false, pendingRequest:null };

function notice(message) {
  const node = $('toast');
  node.textContent = message;
  node.hidden = false;
  clearTimeout(notice.timer);
  notice.timer = setTimeout(() => node.hidden = true, 5000);
}
function text(el, value) { $(el).textContent = String(value); }
function current() {
  return ui.board?.tasks?.find(t => t.id === ui.board.selected) || null;
}
function fieldValue(key) { return $(formIds[key]).value.trim(); }
function setDirty(value) {
  ui.dirty = value;
  if (value) text('boardSaveState','Unsaved local edits — click Save task notes.');
}
function fillForm() {
  const task = current();
  if (!task) return;
  for (const field of fields) {
    const value = task[field];
    $(formIds[field]).value = value === undefined || value === null
      ? (field === 'status' ? 'planned' : field === 'result' ? 'unverified' : '') : value;
  }
  setDirty(false);
  showSaved();
  prepareAsk();
  prepareBrief();
}
function showSaved() {
  text('boardSaveState', ui.board
    ? 'Local operator notes · revision ' + ui.board.revision + ' · not TASS memory'
    : 'Local operator task notes unavailable.');
}
function statusLabel(status) {
  return ({planned:'NOT STARTED',working:'IN PROGRESS',blocked:'BLOCKED',review:'READY FOR REVIEW'})[status] || 'UNKNOWN';
}
function resultLabel(result) {
  return ({unverified:'NOT VERIFIED',reported_pass:'OWNER PASS',failed:'FAILED',blocked:'BLOCKED'})[result] || 'UNKNOWN';
}
function renderTasks() {
  const list=$('taskList'); list.replaceChildren();
  if(!ui.board){list.textContent='Task ledger unavailable. Check the Atrium volume and reload.';return;}
  text('taskCount',ui.board.tasks.length + ' TASKS');
  for(const task of ui.board.tasks){
    const item=document.createElement('button');item.type='button';
    item.className='task-item'+(task.id===ui.board.selected?' selected':'');
    item.setAttribute('role','listitem');
    item.setAttribute('aria-current',task.id===ui.board.selected?'true':'false');
    const title=document.createElement('strong');title.textContent=task.title;
    const meta=document.createElement('div');meta.className='task-meta';
    const st=document.createElement('span');st.textContent=statusLabel(task.status);
    const result=document.createElement('span');result.textContent=resultLabel(task.result);
    if(task.result==='reported_pass')result.className='pass';
    if(task.result==='failed'||task.result==='blocked')result.className='bad';
    meta.append(st,result);item.append(title,meta);
    item.addEventListener('click',()=>selectTask(task.id));
    list.append(item);
  }
}
async function loadBoard() {
  try {
    const r=await fetch('/desk/state',{cache:'no-store'});
    const data=await r.json();
    if(!r.ok)throw Error(data.error||'Task board not readable');
    ui.board=data;ui.dirty=false;
    renderTasks();fillForm();
    return true;
  }catch(e){
    text('boardSaveState','Local task ledger unavailable: '+e.message);
    renderTasks();
    notice('Task notes could not be loaded; do not overwrite existing local data.');
    return false;
  }
}
async function saveBoard(board) {
  if(ui.inFlight)throw Error('Another save is in progress');
  ui.inFlight=true;
  try{
    const r=await fetch('/desk/state',{method:'POST',headers:{
      'Content-Type':'application/json','X-Playground-CSRF':csrf
    },body:JSON.stringify(board)});
    const data=await r.json();
    if(!r.ok)throw Error(data.error||'Could not save task notes');
    ui.board=data;ui.dirty=false;
    renderTasks();showSaved();
    return data;
  }finally{ui.inFlight=false;}
}
async function selectTask(id) {
  if(!ui.board||id===ui.board.selected)return;
  if(ui.dirty&&!window.confirm('Discard unsaved task edits and select another task?'))return;
  const proposed=structuredClone(ui.board);proposed.selected=id;
  try{await saveBoard(proposed);fillForm();}
  catch(e){notice('Selection not saved: '+e.message);}
}
async function saveTask(e) {
  e.preventDefault();
  if(!ui.board){notice('Task ledger unavailable; no changes saved.');return;}
  const task=current();if(!task)return;
  const proposed=structuredClone(ui.board);
  const target=proposed.tasks.find(t=>t.id===task.id);
  for(const key of fields)target[key]=fieldValue(key);
  if(target.result==='reported_pass'&&!target.evidence){
    notice('A reported PASS requires observed evidence and source revision.');$('taskEvidence').focus();return;
  }
  try{await saveBoard(proposed);prepareAsk();prepareBrief();notice('Task notes saved locally. No code or GitHub files changed.');}
  catch(err){notice('Not saved: '+err.message);}
}
async function addTask(e) {
  e.preventDefault();
  if(!ui.board){notice('Load the local task ledger first.');return;}
  if(ui.dirty){notice('Save or discard the current task edits before adding an idea.');return;}
  const title=$('newTitle').value.trim();if(!title)return;
  if(ui.board.tasks.length>=40){notice('Task board is at its 40-item limit.');return;}
  const slug=title.toLowerCase().normalize('NFKD').replace(/[^a-z0-9]+/g,'-').replace(/^-|-$/g,'').slice(0,38)||'idea';
  const id=slug+'-'+Date.now().toString(36);
  const proposed=structuredClone(ui.board);
  proposed.tasks.push({id,title,why:'',next_step:'',expected:'',notes:'',evidence:'',status:'planned',result:'unverified'});
  proposed.selected=id;
  try{
    await saveBoard(proposed);
    $('newTitle').value='';
    fillForm();$('taskWhy').focus();
    notice('Idea added. Add a concrete next step and save your notes.');
  }catch(err){notice('Idea not saved: '+err.message);}
}
function prepareAsk() {
  if(ui.dirty){notice('Save your task changes before preparing a TASS request.');return;}
  const task=current();if(!task)return;
  const instruction=[
    'TASS, I want to collaborate on one existing project task. Please investigate, do not execute.',
    'TASK: '+task.title,
    'WHY: '+(task.why||'Not yet defined'),
    'NEXT SAFE STEP: '+(task.next_step||'Help identify the next small test'),
    'EXPECTED EVIDENCE: '+(task.expected||'Define observable evidence first'),
    'Use only tools and information you really have. Distinguish live state from GitHub plans or placeholders.',
    'Propose ONE harmless next action with expected result, and wait for explicit owner approval in Workshop.',
    'Do not alter repos, execute code, modify identity or permissions, or claim a PASS without a real result.'
  ].join('\n');
  $('askText').value=instruction.slice(0,3000);
}
function prepareBrief() {
  if(ui.dirty){notice('Save task changes before preparing a GitHub brief.');return;}
  const task=current();if(!task)return;
  const brief=[
    'Atrium 2026-11-04 — owner-reviewed task brief',
    'Task: '+task.title,
    'Next step: '+(task.next_step||'To be decided'),
    'Expected: '+(task.expected||'To be defined'),
    'Status: '+statusLabel(task.status)+'; test: '+resultLabel(task.result),
    'Evidence: '+(task.evidence||'Not verified'),
    'Please help plan the smallest next safe test. No changes are authorized by this brief.'
  ].join('\n').slice(0,1200);
  $('briefPreview').value=brief;$('shareConsent').checked=false;updateShareButton();
}
async function askTass() {
  const message=$('askText').value.trim();
  if(!message){notice('Write a message before sending.');return;}
  if(!ui.live?.model_ok){notice('Workshop has not reported a ready model; no message sent.');return;}
  if(ui.live.busy){notice('TASS is busy; wait for the current turn.');return;}
  if(ui.sending||ui.pendingRequest){notice('Wait for the current request outcome.');return;}
  ui.sending=true;updateAskButton();
  text('askState','Sending a chat-only request; no execution authority.');
  try{
    const r=await fetch('/bridge/message',{method:'POST',headers:{
      'Content-Type':'application/json','X-Playground-CSRF':csrf
    },body:JSON.stringify({text:message})});
    const data=await r.json();
    if(!r.ok)throw Error(data.error||'TASS did not accept the request');
    if(data.accepted!==true||!(/^[0-9a-f]{16}$/).test(data.request_id||''))
      throw Error('Workshop did not confirm a request ID');
    ui.pendingRequest=data.request_id;
    text('askState','Request '+data.request_id+' ACCEPTED. Waiting for a real model reply.');
    notice('Chat-only request accepted. Full Auto and Autonomy unchanged.');
    await pollLive();
  }catch(e){text('askState','Message not sent: '+e.message);notice('TASS request failed.');}
  finally{ui.sending=false;updateAskButton();}
}
function setSignal(prefix,status,detail,kind='') {
  text(prefix+'Value',status);text(prefix+'Detail',detail);
  const el=$(prefix+'Signal');el.classList.remove('ready','failed');
  if(kind)el.classList.add(kind);
}
function updateAskButton() {
  const s=ui.live;
  $('askTass').disabled=!(s&&s.model_ok===true&&!s.busy&&!ui.sending&&!ui.pendingRequest);
}
function renderEvents(timeline) {
  const host=$('eventStream');host.replaceChildren();
  const relevant=(Array.isArray(timeline)?timeline:[])
    .filter(e=>['user','reply','result','error','spark','trace','thought','proposal','notice'].includes(e.type))
    .slice(-20);
  text('eventCount',relevant.length+' EVENTS');
  if(!relevant.length){const empty=document.createElement('div');empty.className='empty';empty.textContent='No Workshop events observed. IDLE is not verified.';host.append(empty);return;}
  for(const ev of relevant){
    const item=document.createElement('div');item.className='event '+(ev.type==='error'?'error':ev.type==='result'?'result':'');
    const title=document.createElement('strong');title.textContent=ev.type==='result'
      ?'Workshop executor reported '+(ev.ok===true?'OK':'FAIL')
      :ev.type.toUpperCase().replace('_',' ');
    const id=document.createElement('span');id.className='event-id';
    id.textContent=(Number.isInteger(ev.id)?'#'+ev.id:'')+(ev.request_id?' · req '+ev.request_id:'');
    const body=document.createElement('p');
    const detail=ev.text||ev.error||ev.kind||ev.label||(
      ev.type==='trace'?'Workshop trace; authoritative Engine integration unverified':
      ev.type==='proposal'?'Review the exact proposal in Workshop':ev.type==='thought'?'Model reported thought completion':'No additional detail');
    body.textContent=String(detail).slice(0,500);
    item.append(id,title,body);host.append(item);
  }
  host.scrollTop=host.scrollHeight;
}
async function pollLive() {
  text('lastChecked','Checking live Workshop…');
  try{
    const r=await fetch('/bridge/state',{cache:'no-store'});
    if(!r.ok)throw Error('Workshop state unavailable');
    const data=await r.json();
    const s=data.status||{};
    ui.live=s;ui.lastTimeline=data.timeline||[];
    setSignal('workshop','API REACHABLE','Real API response; code revision not exposed','ready');
    setSignal('model',s.model_ok?'REPORTED READY':'NOT READY',
      s.model_ok?'Status only; verify a real reply':'Workshop did not report a ready model',s.model_ok?'ready':'failed');
    const identity=s.identity||{};
    setSignal('identity',identity.repo?'REPO MIRRORED':'UNCONFIRMED',
      identity.repo?(identity.repo+' @ '+(identity.branch||'?')):'Source not reported; no grant proven');
    text('runtimeFoot','Kernel: '+(s.kernel||'unknown')+
      ' · Mirror synced: '+(s.mind?.synced_at||'unknown')+
      ' · These signals do not prove Docker, Engine, or Identity grants.');
    const proposals=Array.isArray(data.proposals)?data.proposals:[];
    $('pendingActions').hidden=!proposals.some(p=>!p.status||p.status==='pending');
    renderEvents(ui.lastTimeline);
    if(ui.pendingRequest){
      const rid=ui.pendingRequest;
      const outcome=ui.lastTimeline.filter(e=>e.request_id===rid&&['reply','error'].includes(e.type)).at(-1);
      if(outcome){
        text('askState','Request '+rid+(outcome.type==='reply'&&outcome.text?.trim()
          ?' REPLIED — real model response visible in the event stream.'
          :' FAILED — '+(outcome.text||'No model reply was received.')));
        ui.pendingRequest=null;
      }else text('askState','Request '+rid+' ACCEPTED — awaiting a real reply; not verified.');
    }
    text('lastChecked','Checked '+new Date().toLocaleTimeString());
  }catch(_){
    ui.live=null;ui.lastTimeline=[];
    if(ui.pendingRequest)text('askState','Request '+ui.pendingRequest+' status UNKNOWN — Workshop unreachable; no reply verified.');
    setSignal('workshop','UNREACHABLE','Workshop bridge did not respond','failed');
    setSignal('model','UNKNOWN','No model status while Workshop unavailable');
    setSignal('identity','UNKNOWN','No mirror source available');
    text('lastChecked','Last check failed · '+new Date().toLocaleTimeString());
    $('pendingActions').hidden=true;
    renderEvents([]);
  }
  updateAskButton();
}
async function checkShare() {
  try{
    const r=await fetch('/bridge/share-status',{cache:'no-store'});
    if(!r.ok)throw Error();
    const data=await r.json();
    ui.sharing=data.available===true;
    text('shareState',ui.sharing
      ?'Local queue available. Delivery still requires the optional feed and Identity Liaison.'
      :'Local sharing queue unavailable. Use Copy clean handoff instead.');
  }catch(_){ui.sharing=false;text('shareState','Sharing not configured; use Copy clean handoff.');}
  updateShareButton();
}
function updateShareButton(){
  const msg=$('briefPreview').value.trim();
  $('shareBrief').disabled=!(ui.sharing&&$('shareConsent').checked&&msg.length>0&&msg.length<=1200);
}
async function shareBrief(){
  const msg=$('briefPreview').value.trim();
  if(!ui.sharing||!$('shareConsent').checked||!msg)return;
  $('shareBrief').disabled=true;
  try{
    const r=await fetch('/bridge/share-brief',{method:'POST',headers:{
      'Content-Type':'application/json','X-Playground-CSRF':csrf
    },body:JSON.stringify({consent:true,task:msg})});
    const data=await r.json();
    if(!r.ok)throw Error(data.error||'Sharing failed');
    text('shareState','Queued locally: '+data.id+'. NOT confirmed on GitHub; check private TASS issue #19.');
    $('shareConsent').checked=false;updateShareButton();
    notice('Approved brief queued locally. Check GitHub for delivery.');
  }catch(e){text('shareState','Not shared: '+e.message);updateShareButton();}
}
function cleanHandoff() {
  const task=current();
  const lines=[
    'ATRIUM / 2026-11-04 / OWNER HANDOFF',
    'Repository discussion: https://github.com/Arameair/TASS/issues/21',
    'Task: '+(task?.title||'Unknown'),
    'Work status: '+statusLabel(task?.status),
    'Result: '+resultLabel(task?.result),
    'Expected: '+(task?.expected||'Not defined'),
    'Observed evidence: '+(task?.evidence||'Not verified'),
    'Next safe step: '+(task?.next_step||'Not defined'),
    'Workshop: '+(ui.live?'API reachable':'UNVERIFIED'),
    'Model: '+(ui.live?.model_ok?'Status reported ready; real reply must be checked':'NOT VERIFIED'),
    'Mind source: '+(ui.live?.identity?.repo||'UNKNOWN'),
    'Thought Engine: NOT VERIFIED via Atrium',
    'Permissions: NOT VERIFIED via Atrium',
    'Real memory: NOT IMPLEMENTED',
    'No execution, merge, or permission change is authorized by this report.'
  ];
  return lines.join('\n').slice(0,7000);
}
async function copyHandoff(){
  const report=cleanHandoff();
  try{
    if(!navigator.clipboard?.writeText)throw Error('Clipboard not available');
    await navigator.clipboard.writeText(report);
    notice('Clean handoff copied. It excludes model transcripts and private task notes.');
  }catch(_){window.prompt('Select and copy this handoff:',report);}
}
function bind() {
  $('taskForm').addEventListener('submit',saveTask);
  for(const key of fields)$(formIds[key]).addEventListener('input',()=>setDirty(true));
  $('resetTask').addEventListener('click',fillForm);
  $('addTaskForm').addEventListener('submit',addTask);
  $('prepareAsk').addEventListener('click',prepareAsk);
  $('askTass').addEventListener('click',askTass);
  $('refreshLive').addEventListener('click',pollLive);
  $('copyHandoff').addEventListener('click',copyHandoff);
  $('prepareBrief').addEventListener('click',prepareBrief);
  $('shareConsent').addEventListener('change',updateShareButton);
  $('briefPreview').addEventListener('input',()=>{$('shareConsent').checked=false;updateShareButton();});
  $('shareBrief').addEventListener('click',shareBrief);
  window.addEventListener('beforeunload',e=>{if(ui.dirty){e.preventDefault();e.returnValue='';}});
  $('askTass').disabled=true;
  loadBoard();
  pollLive();
  checkShare();
  setInterval(pollLive,5000);
}
bind();
