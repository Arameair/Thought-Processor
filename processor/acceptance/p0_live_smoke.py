#!/usr/bin/env python3
"""P0 real-process smoke against Identity alpha-candidate. No GitHub or Engine fixture.

Runs both *real* Go binaries on loopback with throwaway profiles. Codes never
leave process memory and are never printed to CI logs. No releases are fetched.
"""
import json, os, pathlib, re, socket, subprocess, sys, tempfile, threading, time
import urllib.error, urllib.request, http.cookiejar

ROOT = pathlib.Path(__file__).resolve().parents[2]
PROC = ROOT / "processor"
IDENTITY = ROOT / "identity-ref"

def free_port():
    with socket.socket() as s:
        s.bind(("127.0.0.1",0))
        return s.getsockname()[1]

class Child:
    def __init__(self, executable, argv, cwd):
        self.proc = subprocess.Popen([str(executable), *argv], cwd=cwd,
            stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
            text=True, encoding="utf-8", errors="replace", bufsize=1)
        self.lines=[]
        self.lock=threading.Lock()
        def pump():
            for line in self.proc.stdout:
                with self.lock:
                    self.lines.append(line.strip())
        threading.Thread(target=pump,daemon=True).start()
    def capture(self, pattern):
        with self.lock:
            for line in self.lines:
                m=re.search(pattern,line)
                if m:return m.group(1)
        return None
    def capture_last(self, pattern):
        with self.lock:
            for line in reversed(self.lines):
                m=re.search(pattern,line)
                if m:return m.group(1)
        return None
    def stop(self):
        if self.proc.poll() is not None:return
        self.proc.terminate()
        try:self.proc.wait(timeout=8)
        except subprocess.TimeoutExpired:self.proc.kill();self.proc.wait(timeout=5)

def request(base, path, data=None, csrf="", opener=None):
    method="POST" if data is not None else "GET"
    raw=json.dumps(data).encode() if data is not None else None
    headers={}
    if data is not None:
        headers["Origin"]=base
        headers["Content-Type"]="application/json"
        if csrf:headers["X-TP-CSRF"]=csrf
    rq=urllib.request.Request(base+path,data=raw,headers=headers,method=method)
    try:
        response=opener.open(rq,timeout=8) if opener else urllib.request.urlopen(rq,timeout=8)
    except urllib.error.HTTPError as e:response=e
    with response:
        raw=response.read()
        try:result=json.loads(raw)
        except json.JSONDecodeError:result={}
        return response.status,result

def await_condition(what, fn, timeout=15):
    deadline=time.monotonic()+timeout
    while time.monotonic()<deadline:
        try:
            v=fn()
            if v:return v
        except (OSError,urllib.error.URLError):
            pass
        time.sleep(.15)
    raise RuntimeError("timed out: "+what)

def build(src, out):
    subprocess.run(["go","build","-o",str(out),"."],cwd=src,check=True,
        stdout=subprocess.DEVNULL)

def main():
    if not IDENTITY.is_dir():
        raise RuntimeError("Identity checkout absent: "+str(IDENTITY))
    with tempfile.TemporaryDirectory(prefix="p0-live-") as tmp:
        tmp=pathlib.Path(tmp)
        suffix=".exe" if sys.platform=="win32" else ""
        identity_bin=tmp/("identity"+suffix)
        processor_bin=tmp/("thought-processor"+suffix)
        build(IDENTITY,identity_bin);build(PROC,processor_bin)
        pport, iport=free_port(),free_port()
        pb="http://127.0.0.1:"+str(pport)
        ib="http://127.0.0.1:"+str(iport)
        profile=tmp/"processor-profile"
        identity=None;processor=None
        try:
            # Processor intentionally launches FIRST.
            processor=Child(processor_bin,["-port",str(pport),"-data",str(profile)],PROC)
            await_condition("Processor health",lambda:request(pb,"/api/health")[0]==200)
            code=await_condition("local operator code",
                lambda:processor.capture(r"#code=([0-9a-f]{16,})"))
            opener=urllib.request.build_opener(
                urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
            st,data=request(pb,"/api/session",{"code":code},opener=opener)
            assert st==200,(st,data)
            csrf=data["csrf"]
            st,data=request(pb,"/api/p0/state",opener=opener)
            assert st==200 and data["connection"]=="AUTHORITY_UNAVAILABLE",data
            print("PASS P0 starts independently and fails closed with Identity off")
            identity=Child(identity_bin,["-port",str(iport),"-data",str(tmp/"identity-profile")],IDENTITY)
            await_condition("Identity health",lambda:request(ib,"/api/health")[0]==200)
            st,data=request(pb,"/api/p0/state",opener=opener)
            assert st==200 and data["connection"]=="DISCOVERED_UNVERIFIED",data
            print("PASS automatic locator discovery stays unverified before pairing")
            st,data=request(pb,"/api/p0/pair/request",{},csrf,opener)
            assert st==202,(st,data)
            one_time=await_condition("Identity console pairing code",
                lambda:identity.capture(r"\b([A-HJ-NP-Z2-9]{4}-[A-HJ-NP-Z2-9]{4})\b"))
            st,bad=request(pb,"/api/p0/pair",{"code":"AAAA-AAAA"},csrf,opener)
            assert st in (400,403,409),(st,bad)
            print("PASS invalid code refused before valid pairing")
            st,data=request(pb,"/api/p0/pair",{"code":one_time},csrf,opener)
            assert st==200 and data["connection"]=="VERIFIED",(st,data)
            st,data=request(pb,"/api/p0/state",opener=opener)
            assert st==200 and data["connection"]=="VERIFIED",data
            system_id=data["identity_status"]["identity_system_id"]
            print("PASS real-process pairing, HMAC-verified Identity API")
            st,created=request(pb,"/api/p0/identity/create",{"Name":"P0 Trial","Kind":"companion"},csrf,opener)
            assert st==201 and created["id"].startswith("ent_"),(st,created)
            ent_id=created["id"]
            st,data=request(pb,"/api/p0/state?identity_id="+ent_id,opener=opener)
            assert data["selected_identity"]==ent_id and any(i["id"]==ent_id for i in data["identities"]),data
            assert not next(i for i in data["identities"] if i["id"]==ent_id)["authorized_sources"]
            print("PASS Identity-issued durable entity and zero implicit source grants")
            processor.stop()
            # New local session on restart; paired ID and token survive.
            processor=Child(processor_bin,["-port",str(pport),"-data",str(profile)],PROC)
            await_condition("Processor restart health",lambda:request(pb,"/api/health")[0]==200)
            code2=await_condition("new local operator code",
                lambda:processor.capture(r"#code=([0-9a-f]{16,})"))
            opener=urllib.request.build_opener(
                urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
            st,data=request(pb,"/api/session",{"code":code2},opener=opener)
            assert st==200,(st,data)
            st,data=request(pb,"/api/p0/state?identity_id="+ent_id,opener=opener)
            assert data["connection"]=="VERIFIED" and data["identity_status"]["identity_system_id"]==system_id,data
            assert data["selected_identity"]==ent_id,data
            print("PASS Processor restart reconnects to same verified Identity and profile")
            # Identity-first order with a second clean Processor profile. Its
            # discovery and approval must be independent of the first Processor.
            second=None
            try:
                second_port=free_port()
                second_base="http://127.0.0.1:"+str(second_port)
                second=Child(processor_bin,["-port",str(second_port),"-data",str(tmp/"processor-profile-2")],PROC)
                await_condition("Identity-first Processor health",lambda:request(second_base,"/api/health")[0]==200)
                second_code=await_condition("second operator code",lambda:second.capture(r"#code=([0-9a-f]{16,})"))
                opener2=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
                st,data=request(second_base,"/api/session",{"code":second_code},opener=opener2)
                assert st==200,(st,data)
                csrf2=data["csrf"]
                st,data=request(second_base,"/api/p0/state",opener=opener2)
                assert st==200 and data["connection"]=="DISCOVERED_UNVERIFIED",data
                time.sleep(3.2) # Identity pairs at most once every 3 seconds.
                st,data=request(second_base,"/api/p0/pair/request",{},csrf2,opener2)
                assert st==202,(st,data)
                second_pair_code=await_condition("second pairing code",lambda:identity.capture_last(r"\b([A-HJ-NP-Z2-9]{4}-[A-HJ-NP-Z2-9]{4})\b"))
                st,data=request(second_base,"/api/p0/pair",{"code":second_pair_code},csrf2,opener2)
                assert st==200 and data["connection"]=="VERIFIED",(st,data)
                st,data=request(second_base,"/api/p0/state?identity_id="+ent_id,opener=opener2)
                assert data["connection"]=="VERIFIED" and data["selected_identity"]==ent_id,data
                assert data["identity_status"]["identity_system_id"]==system_id
                print("PASS Identity-first startup: second clean Processor discovers and securely pairs")
                st,data=request(second_base,"/api/p0/unpair",{},csrf2,opener2)
                assert st==200,(st,data)
                st,data=request(second_base,"/api/p0/state",opener=opener2)
                assert data["connection"]=="DISCOVERED_UNVERIFIED",data
                print("PASS independent pairing revocation and new approval requirement")
            finally:
                if second:second.stop()
            assert identity.proc.poll() is None,"stopping Processor unexpectedly stopped external Identity"
            print("PASS external Identity remains running after Processor stop")
            identity.stop()
            # A killed authority may leave stale locators, but cannot authorize.
            st,data=request(pb,"/api/p0/state",opener=opener)
            assert data["connection"]=="AUTHORITY_UNAVAILABLE",data
            print("PASS Identity outage fails closed after prior verification")
            print("P0 REAL-PROCESS SMOKE: PASS (provider GitHub and actual releases NOT TESTED)")
        finally:
            if processor:processor.stop()
            if identity:identity.stop()

if __name__=="__main__":
    try:main()
    except Exception as exc:
        print("P0 REAL-PROCESS SMOKE: FAIL ("+type(exc).__name__+")",file=sys.stderr)
        # Do not echo raw exceptions, paths, codes or HTTP response bodies.
        sys.exit(1)
