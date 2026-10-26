package main

import (
 "context"
 "crypto/hmac"
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "os"
 "path/filepath"
 "runtime"
 "strings"
 "testing"
 "time"
)

const testPairToken="0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
func fixtureSigned(w http.ResponseWriter,r *http.Request,sysID string,status int,v any) {
 b,_:=json.Marshal(v)
 m:=hmac.New(sha256.New,[]byte(testPairToken))
 m.Write([]byte("identity-v1\n"+r.Header.Get("X-Processor-Nonce")+"\n"));m.Write(b)
 w.Header().Set("X-Identity-Proof",hex.EncodeToString(m.Sum(nil)))
 w.Header().Set("X-Identity-System-ID",sysID)
 w.Header().Set("Content-Type","application/json")
 w.WriteHeader(status);w.Write(b)
}
func fixturePair(base string)Pairing{return Pairing{Token:testPairToken,IdentityAddress:base,IdentitySystemID:"idn_fixture"}}

func TestP0AuthenticatedReplyFixture(t *testing.T){
 good:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  if r.Header.Get("Authorization")!="Bearer "+testPairToken||r.Header.Get("X-Processor-Nonce")==""{t.Error("missing bearer or nonce")}
  fixtureSigned(w,r,"idn_fixture",200,map[string]any{"state":"READY","api":"identity/1-alpha"})
 }))
 defer good.Close()
 p:=fixturePair(good.URL)
 m,e:=p0Read(context.Background(),p,good.URL,"/api/v1/status")
 if e!=nil||m["state"]!="READY"{t.Fatalf("signed state: %v %v",m,e)}
 forged:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  fixtureSigned(w,r,"idn_other",200,map[string]string{"state":"READY"})
 }))
 defer forged.Close()
 _,e=p0Read(context.Background(),p,forged.URL,"/api/v1/status")
 if e==nil||!strings.Contains(e.Error(),"did not match"){t.Fatalf("wrong service accepted: %v",e)}
 unsigned:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  w.Header().Set("Content-Type","application/json");w.Write([]byte(`{"state":"READY"}`))
 }))
 defer unsigned.Close()
 _,e=p0Read(context.Background(),p,unsigned.URL,"/api/v1/status")
 if e==nil{t.Fatal("unsigned reply accepted")}
}
func TestP0RejectsNonLocalPeer(t *testing.T){
 for _,a:=range []string{"https://127.0.0.1:7790","http://example.com:7790","http://127.0.0.1:7790/evil","http://127.0.0.1:7790@evil.com:7790"} {
  if v,e:=loopbackURL(a);e==nil{t.Fatalf("accepted %s as %s",a,v)}
 }
}
func TestP0GenericInstallerFixture(t *testing.T){
 asset:=[]byte("fixture bytes, NOT real thought-engine")
 digest:=sha256.Sum256(asset)
 sum:=hex.EncodeToString(digest[:])
 feature:="sample-feature"
 arch:=runtime.GOOS+"-"+runtime.GOARCH
 filename:=feature+"-"+arch
 if runtime.GOOS=="windows"{filename+=".exe"}
 digestShouldMatch:=true
 srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  switch r.URL.Path {
  case "/api/v1/status":
   fixtureSigned(w,r,"idn_fixture",200,map[string]any{"state":"READY","api":"identity/1-alpha"})
  case "/api/v1/features/"+feature+"/eligibility":
   fixtureSigned(w,r,"idn_fixture",200,map[string]any{"feature":feature,"eligible_to_install":map[string]string{"state":"ALLOWED"},"install_adapter":"fixture"})
  case "/api/v1/features/"+feature+"/release":
   h:=sum
   if !digestShouldMatch{h=strings.Repeat("0",64)}
   fixtureSigned(w,r,"idn_fixture",200,map[string]any{"tag":"v1","asset":filename,"sha256":h,"size":len(asset),"decision_id":"d1"})
  case "/api/v1/features/"+feature+"/release/download":
   w.Header().Set("X-Release-Tag","v1");w.Header().Set("X-Asset-Name",filename);w.Write(asset)
  default: http.NotFound(w,r)
  }
 }))
 defer srv.Close()
 tmp:=t.TempDir()
 ins:=NewP0InstallManager(tmp)
 p:=fixturePair(srv.URL)
 prev,e:=ins.Preview(context.Background(),p,srv.URL,feature)
 if e!=nil||prev["feature"]!=feature{t.Fatalf("preview: %v %v",prev,e)}
 if e=ins.Start(context.Background(),p,srv.URL,feature,"v1");e!=nil{t.Fatal(e)}
 deadline:=time.Now().Add(5*time.Second)
 for ins.View(feature)["busy"]==true&&time.Now().Before(deadline){time.Sleep(10*time.Millisecond)}
 view:=ins.View(feature)
 if view["phase"]!="INSTALLED"{t.Fatalf("install not completed: %+v",view)}
 if status,e:=ins.Verify(feature);e!=nil||status!="VERIFIED"{t.Fatalf("verify: %s %v",status,e)}
 restarted:=NewP0InstallManager(tmp)
 if _,e:=restarted.Verify(feature);e!=nil{t.Fatalf("restart lost installation: %v",e)}
 b,e:=os.ReadFile(filepath.Join(tmp,"features",feature,"v1",filename))
 if e!=nil||string(b)!=string(asset){t.Fatalf("installed wrong bytes: %v",e)}

 // Pinned tag protects against a release change.
 if e:=ins.Start(context.Background(),p,srv.URL,feature,"v2");e!=nil{t.Fatal(e)}
 for ins.View(feature)["busy"]==true&&time.Now().Before(time.Now().Add(3*time.Second)){time.Sleep(10*time.Millisecond)}
 if ins.View(feature)["phase"]!="FAILED"{t.Fatal("unexpected version accepted")}
 if status,e:=ins.Verify(feature);e!=nil||status!="VERIFIED"{t.Fatalf("previous install was lost: %s %v",status,e)}
}
