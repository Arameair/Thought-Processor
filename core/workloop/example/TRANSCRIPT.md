# Demo transcript (simulated; fixture GitHub, fixture clock)

```text
12:00:00  WS-A    gpt6      ASSIGN implementer=claude auditor=codex
12:00:00  WS-B    gpt6      ASSIGN implementer=claude auditor=codex observer=tass
12:00:40  WS-A    claude    CANDIDATE 553c3655cb10  -> AWAITING_AUDIT
12:00:40  WS-B    claude    CANDIDATE a4aa194f9975  -> AWAITING_AUDIT
12:01:20  WS-A    codex     AUDIT FAIL 553c3655cb10  -> AUDIT_FAILED
12:01:40  WS-B    codex     AUDIT PASS a4aa194f9975  -> AUDIT_PASSED
12:02:00  WS-A    claude    CANDIDATE e0800c1f9dcf  -> AWAITING_AUDIT
12:02:20  WS-A    codex     AUDIT PASS e0800c1f9dcf  -> AUDIT_PASSED
12:02:40  WS-B    tass      ACK PREPARED (input revision 85e2ad17ecb1, not published: no grant)
```

Commits to the fixture mailbox: 16 (per actor: claude=6 codex=6 gpt6=4).
