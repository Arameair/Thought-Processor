import json
import os
import tempfile
import sys
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.error import HTTPError
from urllib.request import ProxyHandler, Request, build_opener

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from server import create_handler, safe_state, safe_event, safe_evidence

OPENER=build_opener(ProxyHandler({}))

class FakeTASS(BaseHTTPRequestHandler):
    last_request_id = 'abcdef1234567890'
    busy = False
    offline = False
    def log_message(self, *args): pass
    def respond(self, obj, status=200):
        data=json.dumps(obj).encode()
        self.send_response(status); self.send_header('Content-Type','application/json')
        self.send_header('Content-Length',str(len(data))); self.end_headers(); self.wfile.write(data)
    def do_GET(self):
        if FakeTASS.offline and self.path in ('/api/status', '/api/state'):
            return self.respond({'error': 'fixture model unavailable'}, 503)
        if self.path == '/api/audit':
            return self.respond([
              {'event':'ran','label':'RUN library/01-look-around.md#1',
               'source':'library/01-look-around.md · cell 1',
               'code_sha256':'a'*64, 'at':'2026-11-03T12:00:00',
               'error':'CREDENTIAL_IS_PRIVATE', 'text':'PRIVATE STDOUT'},
              {'event':'failed','label':'RUN library/01-look-around.md#2',
               'source':'library/01-look-around.md · cell 2', 'at':'2026-11-03T12:10:00',
               'error':'SECRET'},
              {'event':'ran','label':'TASS own code', 'source':'TASS secret code',
               'code_sha256':'b'*64}
            ])
        if self.path == '/api/status':
            return self.respond({'model_ok': True, 'busy': FakeTASS.busy,
                'full_auto': True, 'autonomy': True, 'identity': {'repo': 'Arameair/TASS'},
                'jupyter': 'http://127.0.0.1:8888/?token=SECRET'})
        if self.path == '/api/state':
            return self.respond({
                'status': {
                    'model_ok':True,'busy':FakeTASS.busy,'full_auto':True,'autonomy':True,
                    'jupyter':'http://127.0.0.1:8888/?token=SECRET',
                    'identity':{'name':'TASS','repo':'Arameair/TASS','branch':'alpha'},
                    'mind':{'library':[{'title':'Look around','path':'library/01.md'}],
                            'projects':[{'title':'Hello TASS','path':'projects/hello-tass'}],
                            'mind':[{'path':'mind/who-i-am.md'}]}
                },
                'timeline':[{'type':'reply','id':4,'text':'Hello from the simulated backend'},
                            {'type':'user','id':5,'text':'fixture prompt',
                             'request_id':FakeTASS.last_request_id},
                            {'type':'reply','id':6,'text':'Fixture-only response',
                             'request_id':FakeTASS.last_request_id}],
                'proposals':[{'id':'p12','kind':'run','source':'library/01.md#1','code':'TOP SECRET CODE'}]
            })
        if self.path == '/api/events':
            self.send_response(200);self.send_header('Content-Type','text/event-stream');self.end_headers()
            self.wfile.write(b'data: {"type":"reply","id":8,"text":"live stream response", "unexpected_secret":"NEVER"}\n\n')
            self.wfile.flush()
            return
        self.respond({'error':'not found'},404)
    def do_POST(self):
        if self.path == '/api/message':
            if FakeTASS.offline:
                return self.respond({'error':'fixture model unavailable'},503)
            if FakeTASS.busy:
                return self.respond({'error':'TASS is thinking; message was not accepted'},409)
            n=int(self.headers.get('Content-Length',0))
            body=json.loads(self.rfile.read(n))
            FakeTASS.last_message=body['text']
            FakeTASS.last_chat_only=body.get('chat_only')
            return self.respond({'ok':True, 'request_id':FakeTASS.last_request_id,
                                 'status':'accepted'},202)
        self.respond({'error':'not found'},404)

class ServerTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.shared_temp = tempfile.TemporaryDirectory()
        cls.previous_requests = os.environ.get("TASS_ATRIUM_COLLAB_REQUESTS")
        os.environ["TASS_ATRIUM_COLLAB_REQUESTS"] = cls.shared_temp.name
        cls.backend=ThreadingHTTPServer(('127.0.0.1',0),FakeTASS)
        cls.bport=cls.backend.server_port
        cls.backend_thread=threading.Thread(target=cls.backend.serve_forever,daemon=True);cls.backend_thread.start()
        cls.app=ThreadingHTTPServer(('127.0.0.1',0),create_handler(cls.bport,'test-csrf',0))
        # Rebind handler's explicit Host/Origin validator to actual ephemeral port.
        cls.app.server_close()
        cls.app=ThreadingHTTPServer(('127.0.0.1',0),create_handler(cls.bport,'test-csrf',cls.app.server_port))
        # Port changed on rebind; use a fresh handler with stable assigned-port socket.
        cls.app.RequestHandlerClass=create_handler(cls.bport,'test-csrf',cls.app.server_port)
        cls.port=cls.app.server_port
        cls.thread=threading.Thread(target=cls.app.serve_forever,daemon=True);cls.thread.start()
        cls.origin=f'http://127.0.0.1:{cls.port}'

    @classmethod
    def tearDownClass(cls):
        cls.app.shutdown();cls.app.server_close();cls.backend.shutdown();cls.backend.server_close()
        if cls.previous_requests is None:
            os.environ.pop("TASS_ATRIUM_COLLAB_REQUESTS", None)
        else:
            os.environ["TASS_ATRIUM_COLLAB_REQUESTS"] = cls.previous_requests
        cls.shared_temp.cleanup()

    def req(self,path,method='GET',body=None,headers=None):
        payload=json.dumps(body).encode() if body is not None else None
        req=Request(self.origin+path,method=method,data=payload,headers=headers or {})
        try:
            r=OPENER.open(req,timeout=5)
            with r:return r.status,r.headers,r.read()
        except HTTPError as e:return e.code,e.headers,e.read()

    def test_homepage_served_without_docker(self):
        status,headers,data=self.req('/')
        self.assertEqual(status,200)
        self.assertIn(b'ATRIUM',data)
        self.assertIn(b'DAILY WORKSTATION',data)
        self.assertIn(b'TASS / The Atrium',self.req('/explore')[2])
        self.assertIn(b'test-csrf',data)
        self.assertIn("default-src 'none'",headers['Content-Security-Policy'])
        self.assertEqual(headers['Cache-Control'],'no-store')

    def test_build_room_and_assets_are_served_without_workshop_authority(self):
        status, headers, page = self.req('/build')
        self.assertEqual(status, 200)
        self.assertIn(b'What should we build?', page)
        self.assertIn(b'test-csrf', page)
        self.assertIn(b'https://github.com/Arameair/TASS/issues/20', page)
        for path in ('/assets/build.js','/assets/build.css'):
            self.assertEqual(self.req(path)[0], 200)
        self.assertEqual(self.req('/bridge/share-status')[0], 200)
        self.assertEqual(self.req('/api/settings',method='POST',
                                  headers={'Content-Type':'application/json'})[0], 404)

    def test_share_brief_requires_explicit_consent_and_csrf(self):
        task = 'I want a simple local note viewer'
        allowed = {'Content-Type':'application/json','X-Playground-CSRF':'test-csrf','Origin':self.origin}
        self.assertEqual(self.req('/bridge/share-brief',method='POST',
                                  body={'task':task,'consent':True},
                                  headers={'Content-Type':'application/json'})[0],403)
        self.assertEqual(self.req('/bridge/share-brief',method='POST',
                                  body={'task':task},headers=allowed)[0],400)
        self.assertEqual(self.req('/bridge/share-brief',method='POST',
                                  body={'task':'password=SECRET','consent':True},
                                  headers=allowed)[0],400)
        code, _, data = self.req('/bridge/share-brief',method='POST',
                                 body={'task':task,'consent':True,'code':'DANGEROUS UNREQUESTED'},
                                 headers=allowed)
        self.assertEqual(code, 202)
        response = json.loads(data)
        self.assertTrue(response['queued'])
        queued = list(Path(self.shared_temp.name).glob('brief-*.json'))
        self.assertEqual(len(queued), 1)
        payload = json.loads(queued[0].read_text())
        self.assertEqual(payload['task'], task)
        self.assertEqual(payload['kind'],'owner_task_brief')
        self.assertNotIn('code',payload)
        self.assertNotIn('DANGEROUS',queued[0].read_text())

    def test_static_files_and_unlisted_paths(self):
        for path,mime in [('/assets/style.css','text/css'),('/assets/app.js','text/javascript')]:
            status,headers,data=self.req(path)
            self.assertEqual(status,200)
            self.assertTrue(data)
            self.assertIn(mime,headers['Content-Type'])
        self.assertEqual(self.req('/tools/index.html')[0],404)
        self.assertEqual(self.req('/../../etc/passwd')[0],404)
        self.assertEqual(self.req('/api/proposal')[0],404)

    def test_live_state_and_redaction(self):
        code,_,data=self.req('/bridge/state')
        self.assertEqual(code,200)
        payload=json.loads(data)
        self.assertEqual(payload['status']['identity']['repo'],'Arameair/TASS')
        self.assertEqual(len(payload['status']['mind']['library']),1)
        self.assertEqual(payload['timeline'][0]['text'],'Hello from the simulated backend')
        self.assertTrue(payload['status']['full_auto'])
        self.assertTrue(payload['status']['autonomy'])
        self.assertEqual(payload['timeline'][-1]['request_id'], FakeTASS.last_request_id)
        self.assertEqual(len(payload['proposals']),1)
        self.assertNotIn('token=',data.decode())
        self.assertNotIn('TOP SECRET',data.decode())

    def test_verified_library_evidence_without_code_or_stdout(self):
        code, _, data = self.req('/bridge/evidence')
        self.assertEqual(code, 200)
        payload = json.loads(data)
        records = payload['records']
        self.assertEqual(len(records), 2)
        self.assertEqual(records[0]['book'], 'library/01-look-around.md')
        self.assertEqual(records[0]['cell'], 1)
        self.assertEqual(records[0]['outcome'], 'pass')
        self.assertEqual(records[1]['outcome'], 'fail')
        self.assertNotIn('CREDENTIAL_IS_PRIVATE', data.decode())
        self.assertNotIn('PRIVATE STDOUT', data.decode())
        self.assertNotIn('SECRET', data.decode())
        self.assertEqual(safe_evidence({'fake':'not an array'}), [])

    def test_connection_diagnostics_without_secrets(self):
        code, headers, data = self.req('/bridge/diagnostics')
        self.assertEqual(code, 200)
        self.assertEqual(json.loads(data)['workshop'], 'reachable')
        self.assertEqual(json.loads(data)['model'], 'ready')
        self.assertNotIn('SECRET', data.decode())
        self.assertNotIn('token=', data.decode())

    def test_csrf_host_and_post_methods(self):
        body={'text':'Hello from Atrium'}
        self.assertEqual(self.req('/bridge/message',method='POST',body=body,headers={'Content-Type':'application/json'})[0],403)
        self.assertEqual(self.req('/bridge/message',method='POST',body=body,headers={'Content-Type':'application/json','X-Playground-CSRF':'test-csrf','Origin':'https://evil.example'})[0],403)
        headers={'Content-Type':'application/json','X-Playground-CSRF':'test-csrf','Origin':self.origin}
        status,_,data=self.req('/bridge/message',method='POST',body=body,headers=headers)
        self.assertEqual(status,202)
        self.assertEqual(FakeTASS.last_message,'Hello from Atrium')
        response=json.loads(data)
        self.assertTrue(response['accepted'])
        self.assertEqual(response['status'], 'accepted')
        self.assertEqual(response['request_id'], FakeTASS.last_request_id)
        self.assertTrue(FakeTASS.last_chat_only)
        self.assertEqual(self.req('/api/proposal',method='POST',body={'decision':'approve'},headers=headers)[0],404)

    def test_busy_and_offline_report_failures_without_fake_success(self):
        headers={'Content-Type':'application/json','X-Playground-CSRF':'test-csrf',
                 'Origin':self.origin}
        FakeTASS.busy=True
        try:
            code,_,data=self.req('/bridge/message',method='POST',
                                  body={'text':'try while busy'},headers=headers)
            self.assertEqual(code,409)
            self.assertIn(b'not accepted',data)
            self.assertNotIn(b'accepted": true',data)
        finally:
            FakeTASS.busy=False
        FakeTASS.offline=True
        try:
            code,_,data=self.req('/bridge/message',method='POST',
                                  body={'text':'try offline'},headers=headers)
            self.assertEqual(code,503)
            self.assertNotIn(b'accepted": true',data)
            self.assertEqual(self.req('/bridge/state')[0],503)
        finally:
            FakeTASS.offline=False

    def test_event_stream_filtered(self):
        req=Request(self.origin+'/bridge/events')
        with OPENER.open(req,timeout=5) as res:
            self.assertIn('text/event-stream',res.headers['Content-Type'])
            lines=[res.readline() for _ in range(3)]
        data=b''.join(lines)
        self.assertIn(b'live stream response',data)
        self.assertNotIn(b'NEVER',data)

    def test_invalid_host(self):
        req=Request(self.origin+'/',headers={'Host':'bad.example'})
        with self.assertRaises(HTTPError) as ctx:OPENER.open(req,timeout=5)
        self.assertEqual(ctx.exception.code,403)

    def test_redaction_helper(self):
        self.assertNotIn('secret',safe_event({'type':'reply','text':'ok','secret':'secret'}))
        self.assertNotIn('code',safe_state({'proposals':[{'id':'a','code':'bad'}]})['proposals'][0])

if __name__=='__main__':unittest.main()