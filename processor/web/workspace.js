"use strict";
(() => {
 const $=id=>document.getElementById(id);
 let csrf="",selected="",state=null,busy=false,dev=false,signInCode=null;
 const plain=(v)=>v==null?"—":String(v);
 function say(m){$("message").textContent=m||""}
 function node(tag,cls,text){const e=document.createElement(tag);if(cls)e.className=cls;if(text!=null)e.textContent=String(text);return e}
 async function api(method,path,payload) {
  const headers={};if(method!=="GET"){headers["Content-Type"]="application/json";if(csrf)headers["X-TP-CSRF"]=csrf;}
  const res=await fetch(path,{method,headers,credentials:"same-origin",body:method==="GET"?undefined:JSON.stringify(payload||{})}).catch(()=>null);
  if(!res)return{ok:false,status:0,data:{message:"Processor is not responding"}};
  let data;try{data=await res.json()}catch{data={message:"Unreadable server response"}}
  return {ok:res.ok,status:res.status,data};
 }
 async function signIn(code) {
  const r=await api("POST","/api/session",{code});
  if(r.ok){csrf=r.data.csrf;return true}
  say(r.data.message||"Sign-in refused");return false;
 }
 async function boot() {
  const hash=location.hash.match(/code=([0-9a-f]+)/i);
  if(hash){signInCode=hash[1];history.replaceState(null,"",location.pathname);}
  let s=await api("GET","/api/session");
  if(s.ok){csrf=s.data.csrf;signInCode=null}
  else if(signInCode&&await signIn(signInCode)){signInCode=null}
  else{$("signin").hidden=false;$("app").hidden=true;return}
  $("signin").hidden=true;$("app").hidden=false;
  await refresh();setInterval(()=>{if(!busy)refresh()},8000);
 }
 async function refresh(){
  busy=true;
  const r=await api("GET","/api/p0/state"+(selected?"?identity_id="+encodeURIComponent(selected):""));
  busy=false;
  if(!r.ok){if(r.status===401){$("signin").hidden=false;$("app").hidden=true;}else say(r.data.message||"Could not refresh");return}
  state=r.data;selected=state.selected_identity||"";
  render();
 }
 const action=(id,cb)=>$(id).addEventListener("click",cb);
 function render(){
  const s=state,c=s.connection,ok=c==="VERIFIED";
  $("conn-head").textContent=c.replaceAll("_"," ");
  $("home-title").textContent=ok?"Your workspace":c==="DISCOVERED_UNVERIFIED"?"Connect to Identity":c==="AUTHORITY_UNAVAILABLE"?"Identity isn't running":"Identity connection needs attention";
  $("conn-detail").textContent=ok?("Verified Identity installation "+plain(s.identity_status.identity_system_id)+". This connection grants no information-source access."):
   ({DISCOVERED_UNVERIFIED:"Identity was found, but it is not yet paired with Processor.",
      AUTHORITY_UNAVAILABLE:"Start your independent Identity application, then check again. Protected actions remain blocked.",
      WRONG_SERVICE:"A service replied with the wrong authentication proof or installation identity. No protected action is allowed.",
      CONNECTION_REMOVED:"Your saved pairing is no longer accepted. Review the Identity installation before reconnecting.",
      INCOMPATIBLE:"This Identity API version is not compatible with Processor."}[c]||"Investigate this connection before proceeding.");
  $("connect").hidden=ok||s.paired||c!=="DISCOVERED_UNVERIFIED";
  $("pair-box").hidden=ok||s.paired||c==="AUTHORITY_UNAVAILABLE"||c==="INCOMPATIBLE";
  $("workspace").hidden=!ok;
  $("unpair").disabled=!ok;
  if(!ok){$("developer").hidden=!dev;renderDev();return}
  const picker=$("identities");picker.replaceChildren();
  const ids=Array.isArray(s.identities)?s.identities:[];
  for(const row of ids){const o=node("option","",row.name+" ("+row.kind+")");o.value=row.id;picker.append(o)}
  if(ids.length===0){const o=node("option","","No identities yet");o.value="";picker.append(o)}
  picker.value=s.selected_identity||"";
  $("github-state").textContent="GitHub: "+plain(s.github.state)+(s.github.account?.login?" as "+s.github.account.login:"")+
   " · GitHub sign-in is only required to check eligible private installations.";
  $("github-connect").textContent=s.github.state==="VERIFIED"?"GitHub verified":"Connect GitHub";
  $("more-private").hidden=!s.additional_features_require_github;
  const fl=$("feature-list");fl.replaceChildren();
  const fs=Array.isArray(s.features)?s.features:[];
  if(!fs.length)fl.append(node("p","muted","No eligible features are available. Connect GitHub to check any private testing entitlements."));
  for(const feature of fs){
   const id=feature.feature||"",el=node("article","feature");
   el.append(node("h3","",feature.name||id));
   const eligibility=feature.eligible_to_install?.state||"UNKNOWN";
   const ins=s.installations?.[id],receipt=ins?.installed;
   const facts=node("dl","facts");
   const rows=[
    ["Install eligibility",eligibility],["Installed locally",receipt?receipt.tag+" · "+plain(ins.condition):"Not installed"],
    ["Identity service registration",feature.service_registered||"NOT_REGISTERED"],
    ["Enabled for identity",feature.enabled_for_identity||"NOT_GRANTED"],
    ["Authorized sources",(feature.authorized_sources||[]).length],
    ["Running",feature.running||"UNKNOWN_TO_IDENTITY"]
   ];
   for(const [k,v] of rows){const div=node("div");div.append(node("dt","",k),node("dd","",v));facts.append(div)}
   el.append(facts);
   if(ins?.busy)el.append(node("p","muted","Installing "+ins.working_feature+": "+plain(ins.phase)+" "+plain(ins.bytes_downloaded)+"/"+plain(ins.total_bytes)+" bytes"));
   if(ins?.error)el.append(node("p","muted","Install result: "+ins.error));
   if(id!=="thought-processor"&&eligibility==="ALLOWED"&&!ins?.busy){
    const b=node("button","primary",receipt?"Review reinstall/update":"Review installation");
    b.addEventListener("click",()=>installPreview(id));el.append(b);
   }
   if(receipt&&!ins?.busy){const b=node("button","","Verify installed files");b.addEventListener("click",()=>verify(id));el.append(b)}
   fl.append(el);
  }
  renderDev();
 }
 function renderDev(){
  $("developer").hidden=!dev; if(!dev)return;
  const s=state||{};
  $("dev-output").textContent=JSON.stringify({connection:s.connection,processor_id:s.processor_id,
   identity_system_id:s.identity_status?.identity_system_id,
   build:s.identity_status?.build,lifecycle:s.identity_status?.lifecycle,
   identities:s.identities?.map(x=>({id:x.id,kind:x.kind})),
   features:s.features?.map(f=>({feature:f.feature,eligible_to_install:f.eligible_to_install?.state,service_registered:f.service_registered})),
   limitations:s.limitations},null,2);
 }
 async function post(path,data){
  busy=true;const r=await api("POST",path,data);busy=false;
  if(!r.ok){say(plain(r.data.message||r.data.error||"Operation refused"));return null}
  return r.data;
 }
 async function installPreview(id){
  const r=await api("GET","/api/p0/install/"+encodeURIComponent(id)+"/preview");
  if(!r.ok){say(r.data.message||"No usable release was available.");return}
  const v=r.data.release;
  if(!window.confirm("Install "+id+" ("+v.tag+")?\n\nAsset: "+v.asset+"\nSize: "+v.size+
    " bytes\nSHA-256: "+v.sha256+"\n\nThis downloads and installs software, but DOES NOT start it or authorize any data source.\n"+
    "Publisher signature/provenance is NOT independently verified in this prototype.\n\nApprove this exact release?"))return;
  const result=await post("/api/p0/install/"+encodeURIComponent(id),{tag:v.tag});
  if(result){say("Install started; progress is shown on the feature card. No data permissions are granted.");await refresh()}
 }
 async function verify(id){const v=await post("/api/p0/install/"+encodeURIComponent(id)+"/verify",{});if(v){say("Installed files: "+v.verification);await refresh()}}
 $("signin-form").addEventListener("submit",async(e)=>{e.preventDefault();if(await signIn($("code").value.trim())){ $("code").value="";await boot()}});
 action("refresh",refresh);
 action("connect",()=>{$("pair-box").hidden=false;$("request-code").focus()});
 action("request-code",async()=>{const r=await post("/api/p0/pair/request",{});if(r)say("A pairing code was shown in Identity's own window. Type it here.")});
 action("pair-submit",async()=>{const r=await post("/api/p0/pair",{code:$("pair-code").value.trim()});if(r){$("pair-code").value="";say("Identity connection verified.");await refresh()}});
 $("addr-form").addEventListener("submit",async(e)=>{e.preventDefault();const r=await post("/api/p0/address",{address:$("addr").value.trim()});if(r)await refresh()});
 $("identities").addEventListener("change",async(e)=>{selected=e.target.value;$("feature-list").replaceChildren();await refresh()});
 action("create-open",()=>$("create-dialog").showModal());
 for(const b of document.querySelectorAll("[data-close]"))b.addEventListener("click",()=>$(b.dataset.close).close());
 $("create-form").addEventListener("submit",async(e)=>{e.preventDefault();
  const v=await post("/api/p0/identity/create",{name:$("new-name").value.trim(),kind:$("new-kind").value});
  if(v){selected=v.id;$("create-dialog").close();$("new-name").value="";say("Identity created with zero source access.");await refresh()}
 });
 action("github-connect",async()=>{
  const v=await post("/api/p0/github/connect",{});if(!v)return;
  if(v.state==="VERIFIED"){say("Already signed in to GitHub.");await refresh();return}
  if(v.user_code&&v.verification_uri) {
   const n=$("github-code");n.hidden=false;n.replaceChildren();
   n.append(node("p","","Open this GitHub URL and enter the code:"));
   const a=node("a","",v.verification_uri);a.href=v.verification_uri;a.target="_blank";a.rel="noopener noreferrer";a.style.color="#a8beff";
   n.append(a,node("p","",v.user_code));
   const cancel=node("button","","Cancel sign-in");cancel.addEventListener("click",async()=>{await post("/api/p0/github/cancel",{});n.hidden=true;await refresh()});n.append(cancel);
   say("Approve GitHub's device authorization yourself; Processor never stores GitHub's token.");
  } else say(plain(v.message||v.next_step||v.state));
 });
 action("unpair",async()=>{if(!window.confirm("Revoke this Processor's Identity pairing? You will need a new human-approved code."))return;
  const r=await post("/api/p0/unpair",{});if(r){selected="";await refresh()}});
 action("dev-toggle",()=>{dev=!dev;$("dev-toggle").textContent=dev?"Hide Developer Mode":"Developer Mode";renderDev()});
 action("copy-diagnostics",async()=>{if(!dev)return;await navigator.clipboard.writeText($("dev-output").textContent).catch(()=>{});say("Copied sanitized state only; no codes, credentials or raw content.")});
 action("exit",()=>$("exit-dialog").showModal());
 action("confirm-exit",async()=>{const choice=document.querySelector('input[name="exit-choice"]:checked').value;
  const r=await post("/api/shutdown",{});if(r){$("exit-dialog").close();say(choice==="keep"?"Processor exiting; external services stay running.":"Processor exiting; it owns no other processes to stop. External services stay running.")}});
 boot();
})();