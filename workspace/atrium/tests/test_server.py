import json
import sys
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.error import HTTPError
from urllib.request import ProxyHandler, Request, build_opener

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from server import create_handler, safe_state, safe_event

OPENER=build_opener(ProxyHandler({}))

class FakeTASS(BaseHTTPRequestHandler):
    def log_message(self, *args): pass
    def respond(self, obj, status=200):
        data=json.dumps(obj).encode()
        self.send_response(status); self.send_header('Content-Type','application/json')
        self.send_header('Content-Length',str(len(data))); self.end_headers(); self.wfile.write(data)
    def do_GET(self):
        if self.path == '/api/state':
            return self.respond({
                'status': {
                    'model_ok':True,'busy':False,'jupyter':'http://127.0.0.1:8888/?token=SECRET',
                    'identity':{'name':'TASS','repo':'Arameair/TASS','branch':'alpha'},
                    'mind':{'library':[{'title':'Look around','path':'library/01.md'}],
                            'projects':[{'title':'Hello TASS','path':'projects/hello-tass'}],
                            'mind':[{'path':'mind/who-i-am.md'}]}
                },
                'timeline':[{'type':'reply','id':4,'text':'Hello from the real test model'}],
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
            n=int(self.headers.get('Content-Length',0))
            body=json.loads(self.rfile.read(n))
            FakeTASS.last_message=body['text']
            return self.respond({'ok': True},202)
        self.respond({'error':'not found'},404)

class ServerTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
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
        self.assertIn(b'TASS / The Atrium',data)
        self.assertIn(b'test-csrf',data)
        self.assertIn("default-src 'none'",headers['Content-Security-Policy'])
        self.assertEqual(headers['Cache-Control'],'no-store')

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
        self.assertEqual(payload['timeline'][0]['text'],'Hello from the real test model')
        self.assertEqual(len(payload['proposals']),1)
        self.assertNotIn('token=',data.decode())
        self.assertNotIn('TOP SECRET',data.decode())

    def test_csrf_host_and_post_methods(self):
        body={'text':'Hello from Atrium'}
        self.assertEqual(self.req('/bridge/message',method='POST',body=body,headers={'Content-Type':'application/json'})[0],403)
        self.assertEqual(self.req('/bridge/message',method='POST',body=body,headers={'Content-Type':'application/json','X-Playground-CSRF':'test-csrf','Origin':'https://evil.example'})[0],403)
        headers={'Content-Type':'application/json','X-Playground-CSRF':'test-csrf','Origin':self.origin}
        status,_,data=self.req('/bridge/message',method='POST',body=body,headers=headers)
        self.assertEqual(status,202)
        self.assertEqual(FakeTASS.last_message,'Hello from Atrium')
        self.assertTrue(json.loads(data)['accepted'])
        self.assertEqual(self.req('/api/proposal',method='POST',body={'decision':'approve'},headers=headers)[0],404)

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