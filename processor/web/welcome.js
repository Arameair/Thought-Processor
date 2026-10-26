"use strict";
// GPT-6 first-run experience; independent of the older diagnostics UI.
// Signed Identity answers and all sensitive credentials are handled by Go.
(() => {
  const $ = id => document.getElementById(id);
  let csrf = "", state = null, identityId = "", currentStep = null, busy = false;
  let waitingPairCode = false, ghCode = null, release = null, dev = false, devTab = "overview";
  let report = "", firstStateRead = false, lastIdentitySystem = "";
  let booted = false;
  const read = (o,k,d="") => o && o[k] != null ? o[k] : d;
  const safeText = x => String(x == null ? "" : x);
  const el = (tag, cls, content) => {
    const n = document.createElement(tag);
    if(cls)n.className=cls;
    if(content!==undefined)n.textContent=safeText(content);
    return n;
  };
  function notice(msg) {
    const n=$("notice");n.textContent=msg;n.hidden=!msg;
    if(msg){const message=msg;setTimeout(()=>{if(n.textContent===message)n.hidden=true},8500)}
  }
  async function api(method,path,data) {
    const h={};
    if(method!=="GET"){h["Content-Type"]="application/json";if(csrf)h["X-TP-CSRF"]=csrf}
    let r;
    try {
      r=await fetch(path,{method,credentials:"same-origin",headers:h,
        body:method==="GET"?undefined:JSON.stringify(data||{}),cache:"no-store"});
    } catch {return {ok:false,status:0,data:{message:"Processor is not responding."}}}
    let value;
    const ct=r.headers.get("Content-Type")||"";
    try{value=ct.includes("json")?await r.json():await r.text()}catch{value={message:"The response was unreadable."}}
    return {ok:r.ok,status:r.status,data:value};
  }
  async function command(path,data={}) {
    busy=true;
    const r=await api("POST",path,data);
    busy=false;
    if(!r.ok){notice(r.data?.message||r.data?.error||"That operation could not complete.");return null}
    return r.data;
  }
  async function login(code) {
    const res=await api("POST","/api/session",{code});
    if(res.ok){csrf=res.data.csrf;return true}
    notice(res.data?.message||"The local Processor code wasn't accepted.");
    return false;
  }
  function showApp(ok) {
    $("signin").hidden=ok;$("application").hidden=!ok;
    if(!ok)$("operator-code").focus();
  }
  async function boot() {
    if(booted)return;booted=true;
    const match=window.location.hash.match(/(?:^|[&#])code=([a-f0-9]+)/i);
    if(match)history.replaceState(null,"",location.pathname);
    const session=await api("GET","/api/session");
    if(session.ok){csrf=session.data.csrf;showApp(true)}
    else if(match&&await login(match[1]))showApp(true);
    else showApp(false);
    if(!$("application").hidden){
      await refresh();
      setInterval(()=>{if(!busy&&!document.hidden)refresh()},8000);
      document.addEventListener("visibilitychange",()=>{if(!document.hidden&&!busy)refresh()});
    }
  }
  const storedKey=sid => "thought-processor:selected:"+sid;
  async function refresh() {
    if(busy)return;
    busy=true;
    const r=await api("GET","/api/p0/state"+(identityId?"?identity_id="+encodeURIComponent(identityId):""));
    busy=false;
    if(!r.ok) {
      if(r.status===401){showApp(false);csrf=""}
      else notice(r.data?.message||"Could not check the system right now.");
      return;
    }
    state=r.data;
    const verified=state.connection==="VERIFIED";
    if(verified){
      const system=state.identity_status?.identity_system_id||"";
      const ids=state.identities||[];
      if(!firstStateRead||lastIdentitySystem!==system){
        firstStateRead=true;lastIdentitySystem=system;
        const preferred=localStorage.getItem(storedKey(system));
        if(preferred&&ids.some(x=>x.id===preferred) && preferred!==identityId) {
          identityId=preferred;currentStep="feature";render();
          queueMicrotask(refresh);return;
        }
      }
      if(state.selected_identity && state.selected_identity!==identityId) {
        identityId=state.selected_identity;
      }
      if(!currentStep) currentStep=ids.length?"profile":"profile";
      if(ids.length===0)currentStep="profile";
      if(currentStep==="connect")currentStep="profile";
    } else {
      currentStep="connect";
      firstStateRead=false;
      ghCode=null;
    }
    render();
    if(dev)loadDev();
  }
  function progress(step) {
    for(const [i,name] of ["connect","profile","feature"].entries()) {
      const n=$("step-"+name);n.classList.toggle("current",name===step);
      n.classList.toggle("done",["connect","profile","feature"].indexOf(name)<["connect","profile","feature"].indexOf(step));
      n.querySelector(".number").textContent=n.classList.contains("done")?"✓":String(i+1);
    }
  }
  function render() {
    if(!state)return;
    const s=state,c=s.connection,verified=c==="VERIFIED";
    let name=verified?"Identity verified":c==="DISCOVERED_UNVERIFIED"?"Identity found":c==="AUTHORITY_UNAVAILABLE"?"Identity offline":c==="CONNECTION_REMOVED"?"Connection removed":c==="WRONG_SERVICE"?"Wrong Identity":"Needs attention";
    const badge=$("connection-badge");
    badge.textContent=name;
    badge.className="status "+(verified?"success":c==="DISCOVERED_UNVERIFIED"?"":"alert");
    const active=currentStep||"connect";
    progress(active);
    $("stage-connect").hidden=active!=="connect";
    $("stage-profile").hidden=active!=="profile";
    $("stage-feature").hidden=active!=="feature";
    $("journey-title").textContent=active==="connect"?"Let's connect your workspace.":active==="profile"?"Make it yours.":"Your workspace is ready.";
    $("journey-subtitle").textContent=active==="connect"?
       "Processor will find Identity automatically. You only need to approve your first connection.":
       active==="profile"?"Choose an actual identity. Each one stays separate.":
       "Choose which approved features to add. Technical setup stays in the background.";
    if(active==="connect")renderConnect();
    if(active==="profile")renderProfiles();
    if(active==="feature")renderFeatures();
  }
  function renderConnect() {
    const c=state.connection,paired=state.paired,verified=c==="VERIFIED";
    const found=c==="DISCOVERED_UNVERIFIED",missing=c==="AUTHORITY_UNAVAILABLE";
    const explanations={
      AUTHORITY_UNAVAILABLE:["Identity is not running","Start Identity if it isn't already open, then let Processor check again."],
      DISCOVERED_UNVERIFIED:["Identity is ready to connect","This appears to be a running Identity installation. Approve a one-time pairing so Processor can verify it."],
      WRONG_SERVICE:["This doesn't match your saved Identity","Processor refused the response. It may be a different installation or a missing authentication proof."],
      CONNECTION_REMOVED:["Your previous connection was removed","Identity no longer accepts your saved pairing. Approve a new connection before continuing."],
      INCOMPATIBLE:["This Identity version isn't compatible","Update the Identity application to a supported identity/1 version. Processor will not bypass this check."]
    };
    const [title,detail]=explanations[c]||["Find your Identity service","Processor is looking for the local Identity authority."];
    $("connect-title").textContent=title;
    $("connect-explain").textContent=detail;
    $("connection-steps").hidden=missing;
    $("check-find").textContent=found?"✓":"1";
    $("check-find-text").textContent=found?
      "A local service responded. It is not trusted until pairing succeeds.":
      paired?"Your previous Identity installation is known, but cannot currently be verified.":
      "Checking local service locators and the default port.";
    $("check-trust-text").textContent=paired?
      "The previous approval is saved. A new one is required only if that approval is revoked.":
      "The one-time code is shown inside Identity. You'll type it here.";
    const button=$("btn-connection");
    button.textContent=found&&!paired?"Request connection code":"Find Identity again";
    button.hidden=paired&&!found;
    $("btn-forget").hidden=!paired;
    $("pairing-entry").hidden=!found||!waitingPairCode||paired;
    $("missing-help").hidden=!missing;
    $("manual-address").hidden=found&&!paired;
  }
  function renderProfiles() {
    const ids=state.identities||[],picker=$("profile-picker");
    picker.replaceChildren();
    for(const row of ids){
      const opt=el("option","",row.name+" · "+row.kind);
      opt.value=row.id;picker.appendChild(opt);
    }
    if(identityId&&ids.some(x=>x.id===identityId))picker.value=identityId;
    else if(state.selected_identity)picker.value=state.selected_identity;
    $("existing-profiles").hidden=ids.length===0;
    const formVisible=ids.length===0||$("profile-form").dataset.open==="1";
    $("profile-form").hidden=!formVisible;
    $("btn-profile-cancel").hidden=ids.length===0;
  }
  function featureCard(f) {
    const id=f.feature||"";
    const ins=state.installations?.[id],receipt=ins?.installed;
    const card=el("article","feature-card");
    const head=el("div","feature-top"),left=el("div"),pill=el("span","tag");
    left.append(el("h3","",f.name||id),el("p","",f.description||"Installed and managed through Identity."));
    const eligible=f.eligible_to_install?.state||"UNKNOWN";
    pill.textContent=id==="thought-processor"?"System workspace":receipt?"Installed":eligible==="ALLOWED"?"Available":"Unavailable";
    head.append(left,pill);card.append(head);
    const facts=el("div","feature-status");
    const makeFact=(label,value)=>{
      const d=el("div","datum");d.append(el("span","",label),el("strong","",value));
      return d;
    };
    facts.append(
      makeFact("Installation",receipt?receipt.tag+" · "+(ins.condition==="INSTALLED_UNVERIFIED"?"Recheck available":ins.condition):"Not installed"),
      makeFact("Install eligibility",eligible==="NOT_APPLICABLE"?"Built-in":eligible==="ALLOWED"?"Approved":eligible),
      makeFact("Access for this identity",f.enabled_for_identity||"NOT_GRANTED"),
      makeFact("Information sources",(f.authorized_sources||[]).length+" approved")
    );
    card.append(facts);
    if(id==="thought-processor"){card.append(el("p","feature-message","This is the interface you're using."));return card}
    if(f.service_registered==="NOT_REGISTERED")card.append(el("p","feature-message","Not registered to run for an identity yet. This is expected in P0."));
    if(ins?.busy) {
      card.append(el("p","feature-message","Installing: "+ins.phase+" · "+(ins.bytes_downloaded||0)+" of "+(ins.total_bytes||0)+" bytes"));
      if(ins.total_bytes>0){
        const bar=el("div","progressbar"),fill=el("div");fill.style.width=(100*ins.bytes_downloaded/ins.total_bytes).toFixed(1)+"%";bar.append(fill);card.append(bar);
      }
    }
    if(ins?.error)card.append(el("p","feature-message","Installation was not completed: "+ins.error));
    const actions=el("div","feature-actions");
    if(eligible==="ALLOWED"&&!ins?.busy){
      const b=el("button","primary",receipt?"Review update or reinstall":"Review installation");
      b.addEventListener("click",()=>reviewInstall(id,f.name||id));actions.append(b);
    }
    if(receipt&&!ins?.busy){
      const b=el("button","quiet","Verify installed files");b.addEventListener("click",()=>verifyFeature(id));actions.append(b);
    }
    card.append(actions);
    return card;
  }
  function renderFeatures(){
    const ids=state.identities||[],chosen=ids.find(x=>x.id===identityId)||ids[0];
    $("selected-name").textContent=chosen?chosen.name:"No identity selected";
    const gh=state.github||{},ghState=gh.state||"UNKNOWN",isGH=ghState==="VERIFIED";
    $("github-title").textContent=isGH?"GitHub connected":"Check access through GitHub";
    const who=gh.account?.login||gh.login||"your account";
    const captions={
      VERIFIED:"Signed in as "+who+". Identity can check installation eligibility, not data-source permissions.",
      WAITING:"Waiting for you to approve the GitHub code.",
      EXPIRED:"Your GitHub code expired. Request a new one when you're ready.",
      DENIED:"GitHub sign-in was declined. No install eligibility is assumed.",
      SIGNED_OUT:"Connect only if you want to check private feature eligibility.",
      NOT_CONFIGURED:"Identity has no GitHub sign-in configured.",
      ERROR:"Could not check GitHub right now. You can retry."
    };
    $("github-caption").textContent=captions[ghState]||gh.next_step||"GitHub is used only for installation eligibility.";
    $("github-button").textContent=isGH?"Connected":"Connect GitHub";
    $("github-button").disabled=isGH||ghState==="NOT_CONFIGURED";
    if(ghCode&&ghState!=="VERIFIED"&&ghState!=="EXPIRED"&&ghState!=="DENIED") {
      $("github-device").hidden=false;$("device-code").textContent=ghCode.code;
      $("open-github").href=ghCode.uri;
    } else {$("github-device").hidden=true;if(ghState!=="WAITING")ghCode=null}
    const list=$("feature-list");list.replaceChildren();
    const features=state.features||[];
    const real=features.filter(x=>x.feature!=="thought-processor");
    $("feature-summary").textContent=real.length?
      "These features are visible based on current Identity eligibility. Private-catalog confidentiality still requires backend enforcement.":
      state.additional_features_require_github?"Connect GitHub to check which private features you can install.":
      "No additional feature is available to install right now.";
    if(real.length===0) {
      const card=el("div","feature-card");card.append(el("h3","","No additional features yet"),
        el("p","muted",state.additional_features_require_github?
          "Sign in to check access. No feature will be installed without your approval.":
          "This may mean no approved feature is available for your GitHub account."));
      list.append(card);
    }
    for(const f of real)list.append(featureCard(f));
  }
  async function reviewInstall(id,name) {
    busy=true;
    const r=await api("GET","/api/p0/install/"+encodeURIComponent(id)+"/preview");
    busy=false;
    if(!r.ok){notice(r.data?.message||r.data?.error||"No approved release is available.");return}
    const meta=r.data.release;
    if(!meta||!meta.tag||!meta.asset||!meta.sha256){notice("Release metadata is incomplete. Nothing will be installed.");return}
    release={id,tag:meta.tag};
    $("install-name").textContent="Install "+name+"?";
    $("install-tag").textContent=meta.tag;
    $("install-asset").textContent=meta.asset;
    $("install-size").textContent=(Number(meta.size)||0).toLocaleString()+" bytes";
    $("install-hash").textContent=meta.sha256;
    $("install-dialog").showModal();
  }
  async function verifyFeature(id) {
    const r=await command("/api/p0/install/"+encodeURIComponent(id)+"/verify",{});
    if(r) {notice("Installed file verification: "+r.verification);await refresh()}
  }
  async function loadDev(){
    if(!dev)return;
    let content="";
    if(devTab==="overview"){
      const s=state||{};
      content=JSON.stringify({
        connection:s.connection,address:s.address,
        identity_system_id:s.identity_status?.identity_system_id||null,
        identity_version:s.identity_status?.build||null,
        identity_lifecycle:s.identity_status?.lifecycle||null,
        profile_ids:(s.identities||[]).map(p=>({id:p.id,kind:p.kind})),
        features:(s.features||[]).map(f=>({id:f.feature,eligible:f.eligible_to_install?.state,registered:f.service_registered})),
        install_states:Object.fromEntries(Object.entries(s.installations||{}).map(([k,v])=>[k,{phase:v.phase,condition:v.condition,installed_tag:v.installed?.tag,verify:v.verify}])),
        limitations:s.limitations
      },null,2);
      report=content;
    }else if(devTab==="metrics"){
      const r=await api("GET","/api/metrics");content=typeof r.data==="string"?r.data:JSON.stringify(r.data,null,2);
    }else if(devTab==="logs"){
      const r=await api("GET","/api/logs");content=typeof r.data==="string"?r.data:"Logs unavailable";
    }else{
      const r=await api("GET","/api/flow");content=JSON.stringify(r.data,null,2);
    }
    $("diagnostics-content").textContent=content;
  }
  async function navigate(next){
    currentStep=next;
    render();window.scrollTo({top:0,behavior:"smooth"});
  }
  async function doPairingRequest(){
    const r=await command("/api/p0/pair/request",{});
    if(r){waitingPairCode=true;renderConnect();$("pair-code").focus();notice("Read the one-time code from Identity's own window.")}
  }
  // ---- interactions ----
  $("signin-form").addEventListener("submit",async e=>{
    e.preventDefault();
    if(await login($("operator-code").value.trim())){ $("operator-code").value="";showApp(true);await refresh();setInterval(()=>{if(!busy&&!document.hidden)refresh()},8000)}
  });
  $("btn-connection").addEventListener("click",()=>{
    if(state.connection==="DISCOVERED_UNVERIFIED")doPairingRequest();else refresh();
  });
  $("pair-form").addEventListener("submit",async e=>{
    e.preventDefault();
    const v=await command("/api/p0/pair",{code:$("pair-code").value.trim()});
    if(v){$("pair-code").value="";waitingPairCode=false;currentStep="profile";notice("Identity verified. Your original approval is saved for next time.");await refresh()}
  });
  $("address-form").addEventListener("submit",async e=>{
    e.preventDefault();
    const v=await command("/api/p0/address",{address:$("identity-address").value.trim()});
    if(v){waitingPairCode=false;await refresh()}
  });
  $("btn-forget").addEventListener("click",async()=>{
    if(!confirm("Forget the pairing saved on this computer? This does not revoke your old token inside Identity. Revoke it from Identity's troubleshooting page if possible, then pair again."))return;
    const v=await command("/api/p0/pair/forget-local",{confirm:"forget-local-pairing"});
    if(v){notice(v.notice);waitingPairCode=false;await refresh()}
  });
  $("btn-profile-next").addEventListener("click",()=>{
    const chosen=$("profile-picker").value;
    if(!chosen)return;
    identityId=chosen;
    const sid=state.identity_status?.identity_system_id||"";
    if(sid)localStorage.setItem(storedKey(sid),identityId);
    navigate("feature");refresh();
  });
  $("btn-profile-create").addEventListener("click",()=>{
    $("profile-form").dataset.open="1";renderProfiles();$("profile-name").focus();
  });
  $("btn-profile-cancel").addEventListener("click",()=>{
    $("profile-form").dataset.open="0";renderProfiles();
  });
  $("profile-form").addEventListener("submit",async e=>{
    e.preventDefault();
    const name=$("profile-name").value.trim(),kind=$("profile-kind").value;
    if(!name)return;
    const created=await command("/api/p0/identity/create",{Name:name,Kind:kind});
    if(!created)return;
    const id=created.id||created.identity?.id;
    if(id){identityId=id;const sid=state.identity_status?.identity_system_id||"";
      if(sid)localStorage.setItem(storedKey(sid),id)}
    $("profile-form").dataset.open="0";$("profile-name").value="";
    currentStep="feature";notice("Identity created. No information sources are authorized.");await refresh();
  });
  $("btn-change-profile").addEventListener("click",()=>navigate("profile"));
  $("btn-choose-profile").addEventListener("click",()=>navigate("profile"));
  $("github-button").addEventListener("click",async()=>{
    const result=await command("/api/p0/github/connect",{});
    if(!result)return;
    const uri=result.verification_uri||"";
    let allowed=false;
    try{const u=new URL(uri);allowed=u.protocol==="https:"&&u.hostname==="github.com"}
    catch{}
    if(result.user_code&&allowed){
      ghCode={code:result.user_code,uri};
      renderFeatures();notice("Authorize GitHub in your browser. Processor does not receive the GitHub token.");
    }else if(result.state==="VERIFIED"){notice("GitHub already connected.");await refresh()}
    else notice(result.next_step||"GitHub authorization is not configured.");
  });
  $("copy-device").addEventListener("click",async()=>{
    if(!ghCode)return;
    try{await navigator.clipboard.writeText(ghCode.code);notice("Temporary GitHub code copied.")}catch{notice("Select and copy the code shown above.")}
  });
  $("cancel-github").addEventListener("click",async()=>{
    if(await command("/api/p0/github/cancel",{})){ghCode=null;await refresh()}
  });
  $("cancel-install").addEventListener("click",()=>{release=null;$("install-dialog").close()});
  $("confirm-install").addEventListener("click",async()=>{
    if(!release)return;const selected={...release};
    const result=await command("/api/p0/install/"+encodeURIComponent(selected.id),{tag:selected.tag});
    if(result){release=null;$("install-dialog").close();notice("Download started. The feature remains disabled for all identities.");await refresh()}
  });
  $("developer-toggle").addEventListener("click",async()=>{
    dev=!dev;$("developer").hidden=!dev;
    $("developer-toggle").setAttribute("aria-expanded",String(dev));
    $("developer-toggle").textContent=dev?"Hide Developer Mode":"Developer Mode ⌄";
    if(dev){await loadDev();$("developer").scrollIntoView({behavior:"smooth",block:"start"})}
  });
  $("diagnostics-refresh").addEventListener("click",loadDev);
  for(const b of document.querySelectorAll(".diag-tabs [data-panel]")){
    b.addEventListener("click",()=>{devTab=b.dataset.panel;
      for(const c of document.querySelectorAll(".diag-tabs [data-panel]")){
        const selected=c===b;c.classList.toggle("selected",selected);
        c.setAttribute("aria-selected",String(selected));
      }loadDev();
    });
  }
  $("copy-report").addEventListener("click",async()=>{
    // Only the explicitly selected safe overview is copied, never tokens,
    // device codes or entire raw external service logs.
    if(!report)await loadDev();
    try{await navigator.clipboard.writeText(report);notice("Redacted report copied.")}
    catch{notice("Clipboard unavailable. Select the status report manually.")}
  });
  $("btn-exit").addEventListener("click",()=>$("exit-dialog").showModal());
  $("cancel-exit").addEventListener("click",()=>$("exit-dialog").close());
  $("confirm-exit").addEventListener("click",async()=>{
    const result=await command("/api/shutdown",{});
    if(result){$("exit-dialog").close();notice("Processor is closing. Other processes will remain running.")}
  });
  boot();
})();