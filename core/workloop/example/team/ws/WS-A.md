# Workstream WS-A

> Append-only ledger. Every event is re-validated on each read (hash chain + state machine); a hand edit anywhere in this file makes the workstream fail closed. Markdown here never grants permission.

**Task:** calc.Add handles signs  
**Phase:** `AUDIT_PASSED` · **candidate:** `e0800c1f9dcf799d020a607efce01452790ae0be`  
**Roles:** coordinator=gpt6 · implementer=claude · auditor=codex · owner=dustin

## Events

### e1 · ASSIGN · gpt6 (coordinator) · 2026-11-19T12:00:00Z

```json
{
  "seq": 1,
  "at": "2026-11-19T12:00:00Z",
  "actor": "gpt6",
  "role": "coordinator",
  "kind": "ASSIGN",
  "task": "calc.Add handles signs",
  "assignees": [
    "implementer=claude",
    "auditor=codex",
    "owner=dustin"
  ],
  "read_rules": "c991c877f40bac2aa3a8cf95233e665a46810c3f",
  "prev": ""
}
```

### e2 · CANDIDATE · claude (implementer) · 2026-11-19T12:00:40Z

```json
{
  "seq": 2,
  "at": "2026-11-19T12:00:40Z",
  "actor": "claude",
  "role": "implementer",
  "kind": "CANDIDATE",
  "commit": "553c3655cb10d9f330bd04afb4c8285b95e9bdcf",
  "evidence": "go test -count=1 ./calc/ at 553c3655cb10: ok",
  "read_rules": "c991c877f40bac2aa3a8cf95233e665a46810c3f",
  "prev": "3552ab1db7ea2a84622bc7f76dbcde3cf634dffef9a4da9a7bf2880528007105"
}
```

### e3 · AUDIT FAIL · codex (auditor) · 2026-11-19T12:01:20Z

```json
{
  "seq": 3,
  "at": "2026-11-19T12:01:20Z",
  "actor": "codex",
  "role": "auditor",
  "kind": "AUDIT",
  "commit": "553c3655cb10d9f330bd04afb4c8285b95e9bdcf",
  "verdict": "FAIL",
  "evidence": "fresh clone at 553c3655cb10; go test -count=1 ./calc/ (+ auditor acceptance test): FAIL",
  "repair": [
    "fix: --- FAIL: TestAuditAddSigns (0.00s)",
    "fix: audit_acceptance_test.go:8: Add(-2, 3) = 5, want 1",
    "fix: audit_acceptance_test.go:8: Add(-2, -3) = -1, want -5",
    "retest: go test ./calc/ must pass the auditor's acceptance test; submit a NEW commit"
  ],
  "read_rules": "c991c877f40bac2aa3a8cf95233e665a46810c3f",
  "prev": "29aa279a10acf765889945a0be886b0fd4c32e5a5c1f7871d72d02d4ef7246d6"
}
```

### e4 · CANDIDATE · claude (implementer) · 2026-11-19T12:02:00Z

```json
{
  "seq": 4,
  "at": "2026-11-19T12:02:00Z",
  "actor": "claude",
  "role": "implementer",
  "kind": "CANDIDATE",
  "commit": "e0800c1f9dcf799d020a607efce01452790ae0be",
  "evidence": "go test -count=1 ./calc/ at e0800c1f9dcf: ok",
  "read_rules": "c991c877f40bac2aa3a8cf95233e665a46810c3f",
  "prev": "e69040f7d8538610d60df1ad41ae0160e0e06205304d55dd4080f0697a65e035"
}
```

### e5 · AUDIT PASS · codex (auditor) · 2026-11-19T12:02:20Z

```json
{
  "seq": 5,
  "at": "2026-11-19T12:02:20Z",
  "actor": "codex",
  "role": "auditor",
  "kind": "AUDIT",
  "commit": "e0800c1f9dcf799d020a607efce01452790ae0be",
  "verdict": "PASS",
  "evidence": "fresh clone at e0800c1f9dcf; go test -count=1 ./calc/ (+ auditor acceptance test): ok",
  "read_rules": "c991c877f40bac2aa3a8cf95233e665a46810c3f",
  "prev": "80dcb49fdc39dc0d2ab1966cb4cbdde0d65f1aaef33ae0f2a6dc706c1f3698d4"
}
```
