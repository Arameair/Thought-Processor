package main

// A feature-ID-based installation transaction. This does not start, register,
// enable or authorize any feature. Release source is the paired Identity's
// approved GitHub adapter; signed publisher provenance is NOT provided in P0.
import (
 "context"
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net/http"
 "os"
 "path/filepath"
 "regexp"
 "runtime"
 "strconv"
 "strings"
 "sync"
 "time"
)

var p0Key = regexp.MustCompile(`^[a-z][a-z0-9-]{1,63}$`)
var p0File = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)
var p0Digest = regexp.MustCompile(`^[0-9a-f]{64}$`)
const p0MaxBytes int64 = 200 << 20

type P0Receipt struct {
 Feature string `json:"feature"`
 Tag string `json:"tag"`
 Asset string `json:"asset"`
 Size int64 `json:"size"`
 SHA256 string `json:"sha256"`
 DecisionID string `json:"decision_id"`
 IdentitySystemID string `json:"identity_system_id"`
 InstalledAt time.Time `json:"installed_at"`
 PublisherAuthenticity string `json:"publisher_authenticity"`
}
type P0InstallManager struct {
 root string
 mu sync.Mutex
 busy bool
 phase string
 failure string
 done int64
 total int64
 feature string
}
func NewP0InstallManager(profileDir string) *P0InstallManager {
 return &P0InstallManager{root:filepath.Join(profileDir,"features"),phase:"IDLE"}
}
func (i *P0InstallManager) rootFor(feature string) (string,error) {
 if !p0Key.MatchString(feature){return "",errors.New("unrecognized feature ID")}
 return filepath.Join(i.root,feature),nil
}
func (i *P0InstallManager) installed(feature string) (any,string) {
 root,e:=i.rootFor(feature);if e!=nil{return nil,"INVALID_FEATURE"}
 b,e:=os.ReadFile(filepath.Join(root,"current.json"));if errors.Is(e,os.ErrNotExist){return nil,"NOT_INSTALLED"};if e!=nil{return nil,"INSTALL_RECORD_UNREADABLE"}
 var r P0Receipt
 if json.Unmarshal(b,&r)!=nil || r.Feature!=feature || !p0File.MatchString(r.Tag) || !p0File.MatchString(r.Asset) || !p0Digest.MatchString(r.SHA256) || r.Size<=0||r.Size>p0MaxBytes {
  return nil,"INSTALL_RECORD_INVALID"
 }
 file:=filepath.Join(root,r.Tag,r.Asset)
 st,e:=os.Lstat(file);if e!=nil || !st.Mode().IsRegular() || st.Size()!=r.Size{return &r,"INSTALLED_FILE_MISSING_OR_CHANGED"}
 return &r,"INSTALLED_UNVERIFIED" // verify digest explicitly; no fabricated verified-on-restart claim
}
func (i *P0InstallManager) View(feature string) map[string]any {
 r,condition:=i.installed(feature)
 i.mu.Lock();defer i.mu.Unlock()
 return map[string]any{"installed":r,"condition":condition,"busy":i.busy,"phase":i.phase,"error":i.failure,"bytes_downloaded":i.done,"total_bytes":i.total,"working_feature":i.feature,"publisher_provenance":"GitHub release via trusted paired Identity; independently signed publisher provenance NOT IMPLEMENTED"}
}
func (i *P0InstallManager) Verify(feature string) (string,error) {
 r,condition:=i.installed(feature);if condition!="INSTALLED_UNVERIFIED" {return condition,fmt.Errorf("%s",condition)}
 rec:=r.(*P0Receipt)
 root,_:=i.rootFor(feature)
 f,e:=os.Open(filepath.Join(root,rec.Tag,rec.Asset));if e!=nil{return "UNREADABLE",e}
 defer f.Close()
 h:=sha256.New(); n,e:=io.Copy(h,io.LimitReader(f,p0MaxBytes+1))
 if e!=nil || n!=rec.Size || hex.EncodeToString(h.Sum(nil))!=rec.SHA256 {
  return "CHECKSUM_MISMATCH",errors.New("installed bytes failed checksum verification")
 }
 return "VERIFIED",nil
}
func (i *P0InstallManager) release(ctx context.Context, pr Pairing, base, feature string) (map[string]any,error) {
 if !p0Key.MatchString(feature){return nil,p0Fail("INVALID_FEATURE",400,"invalid feature ID")}
 ent,e:=p0Read(ctx,pr,base,"/api/v1/features/"+feature+"/eligibility")
 if e!=nil{return nil,e}
 meta,_:=ent["eligible_to_install"].(map[string]any)
 if meta["state"]!="ALLOWED" {reason,_:=meta["reason"].(string);return nil,p0Fail("INSTALL_DENIED",403,"Identity has not authorized this installation: "+reason)}
 if ent["install_adapter"]=="NONE" {return nil,p0Fail("NO_INSTALL_ADAPTER",404,"Identity does not offer an installer for this feature")}
 platform:=runtime.GOOS+"-"+runtime.GOARCH
 return p0Read(ctx,pr,base,"/api/v1/features/"+feature+"/release?platform="+platform)
}
func (i *P0InstallManager) Preview(ctx context.Context,pr Pairing,base,feature string)(map[string]any,error){
 m,e:=i.release(ctx,pr,base,feature);if e!=nil{return nil,e}
 for _,k:=range []string{"tag","asset","sha256","size"}{if _,ok:=m[k];!ok{return nil,p0Fail("BAD_RELEASE",502,"release metadata missing "+k)}}
 return map[string]any{"feature":feature,"release":m,"meaning":"Approval permits download/install only. No feature run, registration or source access. Publisher signing is not available in this candidate."},nil
}
func (i *P0InstallManager) Start(ctx context.Context,pr Pairing,base,feature,approvedTag string) error {
 if !p0Key.MatchString(feature)||!p0File.MatchString(approvedTag){return p0Fail("BAD_REQUEST",400,"invalid feature or tag")}
 i.mu.Lock()
 if i.busy {i.mu.Unlock();return p0Fail("BUSY",409,"another installation is in progress")}
 i.busy=true;i.phase="CHECKING";i.failure="";i.done=0;i.total=0;i.feature=feature
 i.mu.Unlock()
 go func(){
  work,cancel:=context.WithTimeout(context.Background(),15*time.Minute);defer cancel()
  e:=i.install(work,pr,base,feature,approvedTag)
  i.mu.Lock();if e!=nil {i.phase="FAILED";i.failure=e.Error()}else{i.phase="INSTALLED"};i.busy=false;i.mu.Unlock()
 }()
 return nil
}
func (i *P0InstallManager) install(ctx context.Context,pr Pairing,base,feature,approvedTag string) error {
 ri,e:=i.release(ctx,pr,base,feature);if e!=nil{return e}
 tag,_:=ri["tag"].(string);asset,_:=ri["asset"].(string);sum,_:=ri["sha256"].(string)
 sizeF,_:=ri["size"].(float64);size:=int64(sizeF)
 if tag!=approvedTag {return errors.New("release changed since approval; review the new version before installing")}
 if !p0File.MatchString(tag)||!p0File.MatchString(asset)||!p0Digest.MatchString(sum)||size<=0||size>p0MaxBytes {return errors.New("release metadata invalid or too large")}
 if (runtime.GOOS=="windows"&&!strings.HasSuffix(asset,".exe")) || !strings.Contains(asset,runtime.GOOS+"-"+runtime.GOARCH) {return errors.New("release asset does not match this platform")}
 root,_:=i.rootFor(feature)
 if e:=os.MkdirAll(root,0700);e!=nil{return e}
 i.mu.Lock();i.phase="DOWNLOADING";i.total=size;i.mu.Unlock()
 platform:=runtime.GOOS+"-"+runtime.GOARCH
 path:="/api/v1/features/"+feature+"/release/download?platform="+platform+"&tag="+urlQueryEscape(tag)
 resp,e:=p0HTTP(ctx,"GET",base,path,pr.Token,p0Nonce(),nil)
 if e!=nil{return fmt.Errorf("Identity download unavailable: %w",e)}
 defer resp.Body.Close()
 if resp.StatusCode!=200 {return fmt.Errorf("Identity refused download (HTTP %d); no installation changed",resp.StatusCode)}
 if resp.Header.Get("X-Release-Tag")!=tag || resp.Header.Get("X-Asset-Name")!=asset{return errors.New("download did not match the selected release metadata")}
 if ct:=resp.Header.Get("Content-Length");ct!="" {if n,_:=strconv.ParseInt(ct,10,64);n!=size{return errors.New("download length differs from approved release")}}
 f,e:=os.CreateTemp(root,".partial-*");if e!=nil{return e}
 stage:=f.Name();defer os.Remove(stage)
 h:=sha256.New()
 writer:=io.MultiWriter(f,h,&p0Progress{i})
 n,e:=io.Copy(writer,io.LimitReader(resp.Body,size+1))
 if e!=nil {f.Close();return fmt.Errorf("download interrupted: %w",e)}
 if n!=size||hex.EncodeToString(h.Sum(nil))!=sum {f.Close();return errors.New("download failed length or SHA-256 verification; previous installation unchanged")}
 if e=f.Sync();e!=nil{f.Close();return e}
 if e=f.Close();e!=nil{return e}
 if e=os.Chmod(stage,0700);e!=nil{return e}
 i.mu.Lock();i.phase="VERIFYING";i.mu.Unlock()
 versionDir:=filepath.Join(root,tag)
 if e=os.MkdirAll(versionDir,0700);e!=nil{return e}
 dest:=filepath.Join(versionDir,asset)
 // Refuse to overwrite a version unexpectedly. Existing bytes must match.
 if _,e=os.Lstat(dest);e==nil {
  old,ee:=os.Open(dest);if ee!=nil{return ee}
  hh:=sha256.New();_,ee=io.Copy(hh,io.LimitReader(old,size+1));old.Close()
  if ee!=nil||hex.EncodeToString(hh.Sum(nil))!=sum{return errors.New("version already exists with different bytes; refusing overwrite")}
 } else if errors.Is(e,os.ErrNotExist) {
  if e=os.Rename(stage,dest);e!=nil{return e}
 } else{return e}
 decID,_:=ri["decision_id"].(string)
 rec:=P0Receipt{Feature:feature,Tag:tag,Asset:asset,Size:size,SHA256:sum,DecisionID:decID,IdentitySystemID:pr.IdentitySystemID,InstalledAt:time.Now().UTC(),PublisherAuthenticity:"NOT_INDEPENDENTLY_SIGNED"}
 if e=writeJSONAtomic(filepath.Join(versionDir,"install.json"),rec);e!=nil{return e}
 if e=writeJSONAtomic(filepath.Join(root,"current.json"),rec);e!=nil{return e}
 return nil
}
type p0Progress struct{i *P0InstallManager}
func (p *p0Progress) Write(b []byte)(int,error){
 p.i.mu.Lock();p.i.done+=int64(len(b));p.i.mu.Unlock();return len(b),nil
}
func urlQueryEscape(s string)string{return strings.ReplaceAll(s,"+","%2B")}
