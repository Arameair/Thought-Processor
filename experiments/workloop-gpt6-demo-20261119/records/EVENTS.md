# Synthetic event evidence

Not GitHub events. Source digests are SHA-256 of local fixture files, not Git commit SHAs.

1. **WS-A / GPT-6 / COORDINATE** — Synthetic task assigned to Claude; Codex reviews independent WS-B candidate
2. **WS-A / TASS-sim / ACK_PROPOSED** — Read synthetic assignment WS-A; response: ready to observe, NOT AUTHORIZED TO ACT; NOT SENT
3. **WS-A / Claude / BUILD** — candidate fixture normalize_v1.py; sha256:6a8815f0189dceb51ed0d0702396adbffd870e35b4108059799fb8e2b228840d
4. **WS-B / Codex / AUDIT** — PASS: 3 checked cases; candidate sha256:f0402d446a58f7e577303b55d8315ad337548b15a3a668bfd44b96b7d458c50b
5. **WS-A / Claude / TEST** — smoke PASS: 1 checked cases; candidate sha256:6a8815f0189dceb51ed0d0702396adbffd870e35b4108059799fb8e2b228840d
6. **WS-A / Codex / AUDIT** — FAIL: ' hello  world ': got 'hello  world', expected 'hello world'; candidate sha256:6a8815f0189dceb51ed0d0702396adbffd870e35b4108059799fb8e2b228840d
7. **WS-A / Claude / REPAIR** — new fixture normalize_v2.py; sha256:89df20e7db340b5141f251f2a4545133a071b7eb0df2c749f7d8af31e0cb05ad; prior audit invalidated
8. **WS-A / Claude / RETEST** — PASS: smoke 1 checked cases; regression 1 checked cases; sha256:89df20e7db340b5141f251f2a4545133a071b7eb0df2c749f7d8af31e0cb05ad
9. **WS-A / Codex / RE-AUDIT** — PASS: 2 checked cases; candidate sha256:89df20e7db340b5141f251f2a4545133a071b7eb0df2c749f7d8af31e0cb05ad
10. **WS-A / GPT-6 / COORDINATE** — Both synthetic audits PASS; fixture stops before owner merge/deploy/grant gate
