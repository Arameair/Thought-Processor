package main

// P0's authenticated Identity/1-alpha client. The browser never receives the
// bearer token. A local health response or locator is never a trust decision.
import (
 "bytes"
 "context"
 "crypto/hmac"
 "crypto/rand"
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net"
 "net/http"
 "net/url"
 "os"
 "path/filepath"
 "runtime"
 "strings"
 "time"
)

type P0Failure struct { Code string; HTTP int; Detail string }
func (e *P0Failure) Error() string { return e.Detail }
func p0Fail(code string, status int, detail string) error { return &P0Failure{code,status,detail} }

func loopbackURL(s string) (string,error) {
 if !strings.Contains(s,"://") { s="http://"+s }
 u,e:=url.Parse(s)
 if e!=nil || u.Scheme!="http" || u.User!=nil || u.RawQuery!="" || u.Fragment!="" || u.Path!="" {
  return "",errors.New("only plain local http://127.0.0.1:port addresses are allowed")
 }
 host:=u.Hostname()
 ip:=net.ParseIP(host)
 if !(host=="localhost" || (ip!=nil && ip.IsLoopback())) || u.Port()=="" {
  return "",errors.New("Identity must be at a loopback host and explicit port")
 }
 return strings.TrimRight(u.String(),"/"),nil
}

func p0HTTP(ctx context.Context, method, base, path, bearer, nonce string, in any) (*http.Response,error) {
 base,err:=loopbackURL(base); if err!=nil { return nil,err }
 var body io.Reader
 if in!=nil { b,e:=json.Marshal(in);if e!=nil{return nil,e}; body=bytes.NewReader(b) }
 req,e:=http.NewRequestWithContext(ctx,method,base+path,body);if e!=nil{return nil,e}
 if in!=nil { req.Header.Set("Content-Type","application/json") }
 if bearer!="" { req.Header.Set("Authorization","Bearer "+bearer);req.Header.Set("X-Processor-Nonce",nonce) }
 // identityHTTP from the existing experiment has proxy disabled and redirects off.
 return identityHTTP.Do(req)
}
func p0Nonce() string { b:=make([]byte,12);if _,e:=rand.Read(b);e!=nil{panic(e)};return hex.EncodeToString(b) }

// p0Signed checks the exact returned bytes before a caller may interpret JSON,
// including signed 4xx refusals. 401 is a pre-authentication refusal.
func p0Signed(ctx context.Context, pr Pairing, base, method, path string, input any) (map[string]any,error) {
 nonce:=p0Nonce()
 resp,e:=p0HTTP(ctx,method,base,path,pr.Token,nonce,input)
 if e!=nil { return nil,p0Fail("AUTHORITY_UNAVAILABLE",0,"Identity is not responding; protected operations remain blocked") }
 defer resp.Body.Close()
 if resp.StatusCode==http.StatusUnauthorized { return nil,p0Fail("CONNECTION_REMOVED",401,"Identity refused this pairing; connect again with approval") }
 raw,e:=io.ReadAll(io.LimitReader(resp.Body,256<<10+1))
 if e!=nil || len(raw)>256<<10 { return nil,p0Fail("INVALID_RESPONSE",resp.StatusCode,"Identity returned a large or unreadable response") }
 mac:=hmac.New(sha256.New,[]byte(pr.Token))
 mac.Write([]byte("identity-v1\n"+nonce+"\n"));mac.Write(raw)
 got:=resp.Header.Get("X-Identity-Proof")
 want:=hex.EncodeToString(mac.Sum(nil))
 if len(got)!=len(want) || !hmac.Equal([]byte(got),[]byte(want)) ||
   resp.Header.Get("X-Identity-System-ID")!=pr.IdentitySystemID {
  return nil,p0Fail("WRONG_SERVICE",resp.StatusCode,"Identity's authenticated reply or installation ID did not match; refusing to continue")
 }
 var out map[string]any
 if e=json.Unmarshal(raw,&out);e!=nil { return nil,p0Fail("INVALID_RESPONSE",resp.StatusCode,"Signed Identity reply was not valid JSON") }
 if resp.StatusCode<200 || resp.StatusCode>=300 {
  msg,_:=out["error"].(string)
  if len(msg)>180 {msg=msg[:180]}
  if msg=="" {msg=fmt.Sprintf("Identity refused the operation (HTTP %d)",resp.StatusCode)}
  return nil,p0Fail("IDENTITY_REFUSED",resp.StatusCode,msg)
 }
 return out,nil
}
func p0Health(ctx context.Context, base string) (map[string]any,error) {
 resp,e:=p0HTTP(ctx,"GET",base,"/api/health","","",nil)
 if e!=nil {return nil,e}
 defer resp.Body.Close()
 if resp.StatusCode!=200 {return nil,errors.New("not an Identity service")}
 var m map[string]any
 if e:=json.NewDecoder(io.LimitReader(resp.Body,4096)).Decode(&m);e!=nil{return nil,e}
 if m["app"]!="identity" {return nil,errors.New("service is not Identity")}
 return m,nil // UNTRUSTED self-report
}
func p0LocatorPath(systemID string) string {
 if systemID=="" || !strings.HasPrefix(systemID,"idn_") || strings.ContainsAny(systemID,"/\\.") {return ""}
 if runtime.GOOS=="windows" {
  local:=os.Getenv("LOCALAPPDATA");if local=="" {return ""}
  return filepath.Join(local,"Arameair","Identity","instances",systemID+".json")
 }
 conf,e:=os.UserConfigDir();if e!=nil{return ""}
 return filepath.Join(conf,"arameair-identity","instances",systemID+".json")
}
func p0Candidates(pr *Pairing, configured string) []string {
 candidates:=[]string{}
 if pr!=nil {
  if loc:=p0LocatorPath(pr.IdentitySystemID);loc!="" {
   if b,e:=os.ReadFile(loc);e==nil && len(b)<8192 {
    var m struct {BaseURL string `json:"base_url"`}
    if json.Unmarshal(b,&m)==nil {candidates=append(candidates,m.BaseURL)}
   }
  }
  candidates=append(candidates,pr.IdentityAddress)
 }
 candidates=append(candidates,configured,"http://127.0.0.1:7790")
 seen:=map[string]bool{};out:=[]string{}
 for _,c:=range candidates {if c==""{continue};s,e:=loopbackURL(c);if e==nil&&!seen[s]{out=append(out,s);seen[s]=true}}
 return out
}
func p0Discover(ctx context.Context, pr *Pairing, configured string) (string,map[string]any,string) {
 for _,base:=range p0Candidates(pr,configured) {
  if pr!=nil {
   v,e:=p0Signed(ctx,*pr,base,"GET","/api/v1/status",nil)
   if e==nil {
    if v["api"]!="identity/1-alpha" {return base,nil,"INCOMPATIBLE"}
    return base,v,"VERIFIED"
   }
   var f *P0Failure
   if errors.As(e,&f)&&f.Code=="WRONG_SERVICE" {return base,nil,"WRONG_SERVICE"}
   if errors.As(e,&f)&&f.Code=="CONNECTION_REMOVED" {return base,nil,"CONNECTION_REMOVED"}
  } else if h,e:=p0Health(ctx,base);e==nil {
   if api,_:=h["api"].(string);!strings.HasPrefix(api,"identity/1-") {return base,nil,"INCOMPATIBLE"}
   return base,nil,"DISCOVERED_UNVERIFIED"
  }
 }
 return "",nil,"AUTHORITY_UNAVAILABLE"
}
func p0Read(ctx context.Context, pr Pairing, base, path string) (map[string]any,error) {
 return p0Signed(ctx,pr,base,"GET",path,nil)
}
func p0Write(ctx context.Context, pr Pairing, base, path string, input any) (map[string]any,error) {
 return p0Signed(ctx,pr,base,"POST",path,input)
}
