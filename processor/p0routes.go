package main

// P0 HTTP facade. Existing local operator-session, CSRF and Host guards still
// protect this page. All Identity bearer credentials stay in the Go backend.
import(
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "net/http"
 "strings"
 "time"
)

func p0Error(w http.ResponseWriter,e error){
 var f *P0Failure
 if errors.As(e,&f){code:=f.HTTP;if code<400||code>599{code=503};writeJSON(w,code,map[string]any{"error":f.Code,"message":f.Detail});return}
 writeJSON(w,503,map[string]any{"error":"OPERATION_FAILED","message":e.Error()})
}
func (a *App) p0Pairing() *Pairing {
 a.mu.Lock();defer a.mu.Unlock()
 if a.pairing==nil{return nil}
 p:=*a.pairing;return &p
}
func (a *App) p0Configured() string {
 a.mu.Lock();defer a.mu.Unlock();return a.cands.Identity
}
// p0FirstPeer uses actual automatic discovery; manual address is only a fallback.
// Pairing is still a deliberate human-approved separate action.
func (a *App) p0FirstPeer(ctx context.Context)(string,error){
 if a.p0Pairing()!=nil{return "",p0Fail("ALREADY_PAIRED",409,"A pairing already exists")}
 base,_,state:=p0Discover(ctx,nil,a.p0Configured())
 if state!="DISCOVERED_UNVERIFIED"{return "",p0Fail(state,503,"Identity could not be discovered. Start Identity or choose its local address.")}
 return base,nil
}
func (a *App) p0Verified(ctx context.Context)(Pairing,string,error){
 pr:=a.p0Pairing()
 if pr==nil{return Pairing{},"",p0Fail("NEEDS_PAIRING",409,"Connect to Identity first")}
 base,_,state:=p0Discover(ctx,pr,a.p0Configured())
 if state!="VERIFIED" {return Pairing{},"",p0Fail(state,503,"Identity connection is not verified: "+state)}
 return *pr,base,nil
}
func (a *App) p0State(ctx context.Context, selected string)map[string]any {
 pr:=a.p0Pairing()
 base,status,conn:=p0Discover(ctx,pr,a.p0Configured())
 out:=map[string]any{"connection":conn,"address":base,"paired":pr!=nil,
 "processor_id":a.procID,"identity_status":status,"identities":[]any{},"features":[]any{},
 "selected_identity":"","github":map[string]any{"state":"UNKNOWN"},
 "installations":map[string]any{},
 "limitations":[]string{"No Identity grants or feature registration in this candidate","No runtime actions or source access","Publisher-signed release provenance unavailable","Processor owns no external processes"}}
 if conn!="VERIFIED" {return out}
 vals,e:=p0Read(ctx,*pr,base,"/api/v1/identities")
 if e!=nil {out["connection"]="AUTHORITY_UNAVAILABLE";out["message"]=e.Error();return out}
 ids,_:=vals["identities"].([]any)
 out["identities"]=ids
 exists:=false
 if selected!="" {for _,v:=range ids{m,_:=v.(map[string]any);if m["id"]==selected{exists=true;break}}}
 if !exists&&len(ids)>0 {m,_:=ids[0].(map[string]any);selected,_=m["id"].(string)}
 out["selected_identity"]=selected
 var path string
 if selected!="" {path="/api/v1/identities/"+selected+"/features"}else{path="/api/v1/features"}
 fs,e:=p0Read(ctx,*pr,base,path)
 if e!=nil {out["message"]="Could not retrieve features: "+e.Error();return out}
 features,_:=fs["features"].([]any)
 cards:=[]any{}
 installs:=map[string]any{}
 // Current Identity candidate exposes internal catalog names to the paired
 // operator. Hide unavailable private card names from the P0 UI; full
 // server-side private catalog filtering requires a future Identity API.
 needGitHub:=false
 for _,v:=range features {
  m,ok:=v.(map[string]any);if !ok{continue}
  id,_:=m["feature"].(string);if !p0Key.MatchString(id){continue}
  eligible,_:=m["eligible_to_install"].(map[string]any)
  es,_:=eligible["state"].(string)
  if id!="thought-processor" && es!="ALLOWED" {if es=="NOT_SIGNED_IN"||es=="SIGN_IN_EXPIRED"||es=="CHECK_FAILED"{needGitHub=true};continue}
  cards=append(cards,m)
  installs[id]=a.p0inst.View(id)
 }
 out["features"]=cards
 out["installations"]=installs
 out["additional_features_require_github"]=needGitHub
 gh,e:=p0Read(ctx,*pr,base,"/api/v1/github")
 if e==nil {
  // device code only belongs in explicit github sign-in action, not status
  delete(gh,"user_code")
  out["github"]=gh
 }
 return out
}
func (a *App) p0Routes(m *http.ServeMux){
 m.HandleFunc("GET /api/p0/state",a.wrap(private,func(w http.ResponseWriter,r *http.Request){
  selected:=r.URL.Query().Get("identity_id")
  if selected!=""&&!strings.HasPrefix(selected,"ent_"){writeJSON(w,400,errBody("bad_id","invalid identity ID"));return}
  ctx,cancel:=context.WithTimeout(r.Context(),8*time.Second);defer cancel()
  writeJSON(w,200,a.p0State(ctx,selected))
 }))
 m.HandleFunc("POST /api/p0/address",a.wrap(mutate,func(w http.ResponseWriter,r *http.Request){
  var in struct{Address string `json:"address"`}
  if readStrict(r,&in)!=nil {writeJSON(w,400,errBody("bad_request","expected Identity address"));return}
  base,e:=loopbackURL(in.Address);if e!=nil{p0Error(w,e);return}
  a.mu.Lock();c:=a.cands;c.Identity=base;a.mu.Unlock()
  if e=a.prof.SaveCandidates(c);e!=nil{p0Error(w,e);return}
  a.mu.Lock();a.cands=c;a.mu.Unlock()
  writeJSON(w,200,map[string]string{"address":base,"state":"DISCOVERY_ONLY"})
 }))
 m.HandleFunc("POST /api/p0/pair/request",a.wrap(mutate,func(w http.ResponseWriter,r *http.Request){
  if a.p0Pairing()!=nil {writeJSON(w,409,errBody("already_paired","Disconnect the existing Identity pairing before changing services"));return}
  if !noParams(w,r){return}
  ctx,cancel:=context.WithTimeout(r.Context(),8*time.Second);defer cancel()
  base,discoveryErr:=a.p0FirstPeer(ctx)
  if discoveryErr!=nil{p0Error(w,discoveryErr);return}
  resp,e:=p0HTTP(ctx,"POST",base,"/api/v1/pair/request","","",map[string]string{"processor_id":a.procID,"processor_label":"Thought Processor P0"})
  if e!=nil{p0Error(w,p0Fail("AUTHORITY_UNAVAILABLE",503,"Identity is not responding"));return}
  defer resp.Body.Close()
  var out map[string]any
  _=json.NewDecoder(resp.Body).Decode(&out)
  if resp.StatusCode!=202 {code:=resp.StatusCode;if code<400{code=400};writeJSON(w,code,map[string]any{"error":"PAIR_REQUEST_REFUSED","retry_after":resp.Header.Get("Retry-After"),"message":"Identity refused the code request; check Identity's window and wait if rate limited"});return}
  writeJSON(w,202,map[string]any{"status":"CODE_SHOWN_IN_IDENTITY","message":"Look in the running Identity console/troubleshooting window. Type the one-time code here; it was not returned to Processor.","expires_at":out["expires_at"]})
 }))
 m.HandleFunc("POST /api/p0/pair",a.wrap(mutate,func(w http.ResponseWriter,r *http.Request){
  if a.p0Pairing()!=nil {writeJSON(w,409,errBody("already_paired","Unpair before pairing a different Identity"));return}
  var in struct{Code string `json:"code"`}
  if readStrict(r,&in)!=nil || !pairCodeRE.MatchString(strings.TrimSpace(in.Code)) {writeJSON(w,400,errBody("bad_code","Type the eight-character code shown in Identity"));return}
  ctx,cancel:=context.WithTimeout(r.Context(),15*time.Second);defer cancel()
  base,discoveryErr:=a.p0FirstPeer(ctx)
  if discoveryErr!=nil{p0Error(w,discoveryErr);return}
  p,e:=IdentityPair(ctx,base,strings.TrimSpace(in.Code),a.procID,"Thought Processor P0");if e!=nil{p0Error(w,e);return}
  st,e:=p0Signed(ctx,p,base,"GET","/api/v1/status",nil)
  if e!=nil {p0Error(w,e);return}
  if st["api"]!="identity/1-alpha" {writeJSON(w,409,errBody("incompatible","Identity API major version is not supported"));return}
  if e=a.prof.SavePairing(p);e!=nil{p0Error(w,e);return}
  a.mu.Lock();a.pairing=&p;a.pairErr="";a.mu.Unlock()
  a.log.Add("pairing","p0_paired","","Identity pairing saved and verified against its installation ID")
  writeJSON(w,200,map[string]any{"connection":"VERIFIED","identity_system_id":p.IdentitySystemID})
 }))
 m.HandleFunc("POST /api/p0/unpair",a.wrap(mutate,func(w http.ResponseWriter,r *http.Request){
  if !noParams(w,r){return}
  p,base,e:=a.p0Verified(r.Context());if e!=nil{p0Error(w,e);return}
  _,e=p0Write(r.Context(),p,base,"/api/v1/unpair",map[string]any{});if e!=nil{p0Error(w,e);return}
  if e=a.prof.RemovePairing();e!=nil{p0Error(w,e);return}
  a.mu.Lock();a.pairing=nil;a.mu.Unlock()
  writeJSON(w,200,map[string]string{"connection":"NEEDS_PAIRING"})
 }))
 // Explicit local reset is needed after a remote revocation or wrong service.
 // It does NOT revoke the old token on an unreachable Identity; explain this.
 m.HandleFunc("POST /api/p0/pair/forget-local",a.wrap(mutate,func(w http.ResponseWriter,r *http.Request){
  var in struct{Confirm string `json:"confirm"`}
  if readStrict(r,&in)!=nil||in.Confirm!="forget-local-pairing" {writeJSON(w,400,errBody("confirm_required","Confirm local pairing reset"));return}
  if e:=a.prof.RemovePairing();e!=nil{p0Error(w,e);return}
  a.mu.Lock();a.pairing=nil;a.mu.Unlock()
  a.log.Add("pairing","p0_forget_local","","Operator discarded local pairing; remote pairing may still need revocation inside the old Identity")
  writeJSON(w,200,map[string]any{"state":"LOCAL_PAIRING_FORGOTTEN","notice":"Old Identity may still trust its previous pairing. Revoke it from Identity's troubleshooting page if available."})
 }))
 m.HandleFunc("POST /api/p0/identity/create",a.wrap(mutate,func(w http.ResponseWriter,r *http.Request){
  var in struct{Name,Kind string}
  if readStrict(r,&in)!=nil || len(in.Name)==0 || len(in.Name)>100 {writeJSON(w,400,errBody("bad_identity","enter a short identity name and type"));return}
  p,base,e:=a.p0Verified(r.Context());if e!=nil{p0Error(w,e);return}
  result,e:=p0Write(r.Context(),p,base,"/api/v1/identities",map[string]string{"name":in.Name,"kind":in.Kind})
  if e!=nil{p0Error(w,e);return}
  writeJSON(w,201,result)
 }))
 m.HandleFunc("POST /api/p0/github/connect",a.wrap(mutate,func(w http.ResponseWriter,r *http.Request){
  if !noParams(w,r){return}
  p,base,e:=a.p0Verified(r.Context());if e!=nil{p0Error(w,e);return}
  result,e:=p0Write(r.Context(),p,base,"/api/v1/github/connect",map[string]any{});if e!=nil{p0Error(w,e);return}
  // The device user_code is shown only in this deliberate sign-in response.
  writeJSON(w,200,result)
 }))
 m.HandleFunc("POST /api/p0/github/cancel",a.wrap(mutate,func(w http.ResponseWriter,r *http.Request){
  if !noParams(w,r){return}
  p,base,e:=a.p0Verified(r.Context());if e!=nil{p0Error(w,e);return}
  result,e:=p0Write(r.Context(),p,base,"/api/v1/github/cancel",map[string]any{});if e!=nil{p0Error(w,e);return}
  writeJSON(w,200,result)
 }))
 m.HandleFunc("GET /api/p0/install/{feature}/preview",a.wrap(private,func(w http.ResponseWriter,r *http.Request){
  feature:=r.PathValue("feature")
  p,base,e:=a.p0Verified(r.Context());if e!=nil{p0Error(w,e);return}
  result,e:=a.p0inst.Preview(r.Context(),p,base,feature);if e!=nil{p0Error(w,e);return}
  writeJSON(w,200,result)
 }))
 m.HandleFunc("POST /api/p0/install/{feature}",a.wrap(mutate,func(w http.ResponseWriter,r *http.Request){
  var in struct{Tag string `json:"tag"`}
  if readStrict(r,&in)!=nil {writeJSON(w,400,errBody("bad_request","missing approved release tag"));return}
  p,base,e:=a.p0Verified(r.Context());if e!=nil{p0Error(w,e);return}
  e=a.p0inst.Start(r.Context(),p,base,r.PathValue("feature"),in.Tag);if e!=nil{p0Error(w,e);return}
  a.log.Add("install","p0_started","","P0 feature installation was explicitly requested")
  writeJSON(w,202,map[string]string{"status":"STARTED","note":"Installation runs independently and does not enable or start the feature"})
 }))
 m.HandleFunc("POST /api/p0/install/{feature}/verify",a.wrap(mutate,func(w http.ResponseWriter,r *http.Request){
  if !noParams(w,r){return}
  status,e:=a.p0inst.Verify(r.PathValue("feature"));if e!=nil{p0Error(w,p0Fail(status,409,e.Error()));return}
  writeJSON(w,200,map[string]string{"verification":status})
 }))
}
// Ensure Go vet sees a real use of fmt in release-independent errors.
var _ = fmt.Sprintf
