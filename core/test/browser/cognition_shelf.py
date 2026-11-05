#!/usr/bin/env python3
"""Isolated Cognition UI checks; never contacts the owner's TASS deployment.
Requires Playwright Python + an installed Chromium. CHROMIUM_PATH overrides executable.
"""
from pathlib import Path
import json
import os
from playwright.sync_api import sync_playwright

HTML=(Path(__file__).resolve().parents[2]/"cmd/tass-core/cognition.html").read_text()
BROWSER=os.environ.get("CHROMIUM_PATH","/usr/bin/chromium")

def live_html(fetch_js):
    return HTML.replace('const isLive=location.pathname==="/cognition" && /^https?:$/.test(location.protocol);','const isLive=true;').replace('<script>\n"use strict";','<script>'+fetch_js+'</script><script>\n"use strict";')

with sync_playwright() as p:
    b=p.chromium.launch(headless=True,executable_path=BROWSER,args=["--no-sandbox"])
    try:
        pg=b.new_page()
        errors=[]
        pg.on("pageerror",lambda err: errors.append(str(err)))
        pg.set_content(HTML)
        pg.wait_for_selector(".tile")
        assert "PREVIEW ONLY" in pg.locator("#banner").inner_text()
        assert pg.locator(".tile").count()>=6
        pg.get_by_role("button",name="Cells",exact=True).click()
        assert pg.locator(".tile").count()==1
        pg.get_by_role("button",name="Inspect List every lesson").click()
        assert "NOT EXECUTABLE" in pg.locator("#fields").inner_text()
        pg.get_by_role("button",name="Close details").click()
        pg.get_by_role("button",name="All",exact=True).click()
        pg.locator("#search").fill("DEMO-CYCLE-001")
        assert pg.locator(".tile").count()==2
        pg.locator("#search").fill("")
        pg.locator(".tile").first.click()
        pg.get_by_role("button",name="Inspect Thought Engine").click()
        assert "Nemotron model health: OK" in pg.locator("#thoughtDetail").inner_text()
        assert not errors,errors
        pg.close()
        print("PASS demo shelf filters/search/detail/nested Engine")

        st={"identity":{"state":"UNLOCKED"},"engine":{"reachable":True,"last_pulse":{"model_ok":True}},
            "cycles":[{"cycle_id":"LIVE_TEST_001","request":"Can I run a cell?","proposal":{"action":"NONE","prediction":{"value":3}},"outcome":{"status":"NO_ACTION"}}],
            "skills":{"receipts":[{"receipt_id":"R_01","skill":"lesson-workspace.list-lessons","state":"AWAITING_APPROVAL","executed":False}]}}
        brain={"graph":{"revision":9,"notes":[]},"blocks":[{"id":"cell:live","kind":"code-cell","title":"List every lesson LIVE","status":"DISCOVERED","runnable":"NOT EXECUTABLE"}]}
        mock="window.fetch=async(path)=>({ok:true,status:200,json:async()=>path==='/api/status'?"+json.dumps(st)+":"+json.dumps(brain)+"})"
        pg=b.new_page()
        pg.set_content(live_html(mock));pg.wait_for_selector(".tile")
        assert "LIVE READ ONLY" in pg.locator("#mode").inner_text()
        assert "9" in pg.locator("#summary").inner_text()
        assert pg.locator(".tile").count()==4
        pg.get_by_role("button",name="Cells",exact=True).click()
        assert "List every lesson LIVE" in pg.locator(".tile").first.inner_text()
        pg.close()
        print("PASS mocked live Core brain/status/receipts")

        pg=b.new_page()
        pg.set_content(live_html("window.fetch=async()=>{throw new Error('offline')}"))
        assert "LIVE UNAVAILABLE" in pg.locator("#mode").inner_text()
        assert "no sample data substituted" in pg.locator("#banner").inner_text()
        assert pg.locator(".tile").count()==0
        pg.close()
        print("PASS unavailable Core never substitutes demo data")
    finally:
        b.close()
print("3/3 PASS (fixtures, not real Core HTTP or Nemotron)")
