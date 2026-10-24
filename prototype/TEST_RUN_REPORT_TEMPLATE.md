# Claude test report — blank, fill ONLY with observed evidence

**This file is a TEMPLATE, not a result.** The builder should copy it to their own experiment branch as `docs/TEST_REPORT.md` and fill in actual facts after performing each check. Never precheck a PASS or write an example PID, username, grant ID, metric or model output as though it came from a real run.

## Tested build and environment

| Field | Actual measured / observed value |
| --- | --- |
| Test date/time and timezone | NOT RUN |
| Operator / tester | NOT RUN |
| Thought Processor repo/branch/full SHA | NOT RUN |
| Thought Engine repo/branch/full SHA | NOT RUN |
| Identity repo/branch/full SHA, if actually running | NOT RUN |
| Source artifact digest / binary verification | NOT RUN |
| Host OS and version / Go version / browser | NOT RUN |
| Isolated profile/data root (safe relative name) | NOT RUN |
| Processor instance/PID/port | NOT RUN |
| Engine instance/PID/port, ownership type | NOT RUN |
| Real GitHub verification: account + repo (non-sensitive) | NOT RUN |
| Publisher entitlement source and status | NOT RUN |
| Identity grant status and decision provenance | NOT RUN |
| Worker name/status and whether it was genuinely invoked | NOT RUN |
| Evidence type (actual integration / unit fixture / manual observation) | NOT RUN |

## Feature truth table at run time

| Claim displayed in UI | Authority that proved or refused it | Request/trace ID | Check time | Actual UI text | Verdict |
| --- | --- | --- | --- | --- | --- |
| Processor process is running | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN |
| Expected Engine build available | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN |
| Engine service identity authenticated | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN |
| GitHub/repository verified | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN |
| Publisher feature access verified | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN |
| Identity runtime grant verified | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN |
| Worker is actually ready | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN |
| Observation was accepted/processed/saved | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN |

## Automated and manual acceptance results

Use [the complete matrix](ACCEPTANCE_TEST_MATRIX.md) as the authoritative case definitions; list every relevant ID. `BLOCKED` is not a `PASS`.

| Test ID | Outcome: PASS/FAIL/BLOCKED/NOT RUN | Evidence (command, run ID, safe log excerpt, PR link) | Explanation/defect |
| --- | --- | --- | --- |
| A01–A07 (list individually after actual run) | NOT RUN | — | — |
| B01–B15 (list individually after actual run) | NOT RUN | — | — |
| C01–C10 (list individually after actual run) | NOT RUN | — | — |
| D01–D08 (real Identity dependency) | BLOCKED pending actual integration | No fake signed service | Not an implemented live path |
| E01–E07 (starter/license/entitlement decisions) | BLOCKED pending decisions | No unapproved repo provisioner | Normal first-opening not complete |

## Real timing samples

Record individual measurements in milliseconds; do not state a median or reliability rate without showing the sample count and actual source.

| Run ID | Processor start-to-health | Ensure request-to-Engine-start | Engine start-to-verified-health | Inspect latency | Stop-to-exit | Outcome |
| --- | --- | --- | --- | --- | --- | --- |
| NOT RUN | — | — | — | — | — | NOT RUN |
| NOT RUN | — | — | — | — | — | NOT RUN |
| NOT RUN | — | — | — | — | — | NOT RUN |

**Actual process count before/after:** NOT RUN.  
**Duplicate/orphan audit:** NOT RUN.  
**Token/canary Copy Context audit:** NOT RUN.  
**Independent Engine survival after Stop Demo:** NOT RUN.

## Failure reproduction and diagnostics

**First failure:** NOT RUN  
**Exact steps to reproduce:** NOT RUN  
**Actual result versus expected:** NOT RUN  
**Observed Engine/process/error evidence:** NOT RUN  
**Data loss or side effects checked:** NOT RUN  
**Fix attempted (and on which SHA):** NOT RUN  
**Re-test on revised SHA:** NOT RUN

## Owner's experience

**What the owner clicked:** NOT RUN  
**What looked understandable:** NOT RUN  
**What was confusing:** NOT RUN  
**What should be removed:** NOT RUN  
**Owner verdict:** PENDING OWNER TEST — KEEP / REVISE / REJECT  
**Promotion approval:** NOT GIVEN. No automatic merge.

## Final build summary

- **🟢 Verified real-pass behaviors:** NOT RUN
- **🟡 Partial, blocked or unverified behaviors:** NOT RUN
- **🔴 Actual failures and unsafe behavior:** NOT RUN
- **Current first-delivery gate reached:** NOT RUN
- **Next single recommended test:** NOT RUN
- **Exact rollback/restart path:** NOT RUN

**Evidence discipline:** Sanitized raw results or source-linked receipts are required. This report cannot turn authored tests, fixture responses, hardcoded UI, claimed model output or inferred IDs into observed PASS.
