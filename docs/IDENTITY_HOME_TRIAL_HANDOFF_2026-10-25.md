# Processor home trial — companion handoff to the consolidated Identity prototype

**Date:** 2026-10-25 (America/Chicago). **Status:** INSTRUCTIONS ONLY; no Processor code changed or new runtime tests asserted by this document.

**Primary assignment and Claude prompt live in Identity** (single source of truth):
- [Start here](https://github.com/Arameair/identity/blob/experiment/identity-consolidated-home-trial-20261025/docs/START_HERE_CLAUDE_2026-10-25.md)
- [Owner acceptance](https://github.com/Arameair/identity/blob/experiment/identity-consolidated-home-trial-20261025/docs/HOME_TRIAL_ACCEPTANCE_2026-10-25.md)
- [Claude prompt](https://github.com/Arameair/identity/blob/experiment/identity-consolidated-home-trial-20261025/docs/CLAUDE_HANDOFF_PROMPT_2026-10-25.md)

This handoff branch `experiment/processor-home-trial-handoff-20261025` is documentation-only, created directly from Thought-Processor `alpha` at `9fac9fa44cb020bb6f8aa6bcb1e9faeda6e0b47a`. **Do not run this branch expecting a program.** Alpha/Stable/Main must not change automatically.

## Priority zero: run the EXISTING candidate, do not rebuild Processor

Reference runnable branch: `experiment/processor-controller-launch-fix-20261024`, commit `0ba8bd52facedda1405897e9136d68ab184a2e32`.

Existing source:
- `marquee.json`: Project Controller Go build/run manifest, including Windows and Linux paths; receives `PORT` and `PC_DATA`.
- `processor/`: real Go diagnostic shell (independent if built manually); saved isolated profile, local operator login, real health and read-only probes.
- `processor/authority.go`: intentionally returns BLOCKED for Engine start/attach/stop/observation; it has no secure Identity authorization producer or enforcement runner. **This is correct honesty, not a reason to fake these controls.**

## Owner's shortest home test

**Preferred developer test:** In Project Controller, add `Arameair/Thought-Processor` branch `experiment/processor-controller-launch-fix-20261024`, select Play and accept exact-code review using Controller's native UI. Controller must report a real build/run/port. If a saved run override shadows `marquee.json`, explicitly inspect/reset it before claiming a packaging defect. Open the live Processor view; use its local operator code **privately**; Copy Context **only after redaction test**; stop and restart.

**Fallback standalone developer build (Go required)** from a checked-out reference branch, on Windows PowerShell:
```powershell
git clone --branch experiment/processor-controller-launch-fix-20261024 https://github.com/Arameair/Thought-Processor.git
cd Thought-Processor/processor
go test ./...
go build -o thought-processor.exe .
.\thought-processor.exe -port 7860 -data "$env:USERPROFILE\tp-home-trial"
```
Run commands are *instructions*, not verified owner Windows evidence. The app normally prints a local one-time operator URL; treat it as a secret and **never paste it in chat, issues, telemetry or diagnostics**. It may alternatively be stored under the profile runtime directory (see actual source).
From a separate terminal you may also use `http://127.0.0.1:7860/api/health` only for **public self-reported health**. This does NOT prove authentication, permission or Engine readiness.

**Important:** the current reference branch was based on Alpha, not this documentation-only handoff. If Claude must modify Processor source, create a **new implementation experiment from latest Processor Alpha**, bring in only reviewed relevant source from the reference, and clearly link its new SHA in the owner test report. Never rewrite the reference branch or auto-promote.

## Integration milestone: one truthful read-only handshake

1. Start consolidated Identity independently in an isolated local profile. It must not demand Project Controller, Thought Engine or GitHub login merely to show a healthy Identity authority dashboard. GitHub login **is required before requesting/using a publisher-gated feature** per owner product model; the emergency test of an already checked-out diagnostic shell is not a customer entitlement release.
2. Start actual Processor independently (Controller is allowed as owner's development harness). Record both real PID, exact repository refs, ports and run IDs.
3. In Processor choose Identity's `127.0.0.1:PORT` candidate and run real read-only health probe. Its own UI must say `CLAIMS_IDENTITY_UNVERIFIED` until authentication exists. Do not wire a trusted Identity session using only `app: identity` over HTTP.
4. In Identity choose Processor's actual local address and read ONLY a public health route if safe. Show `DISCOVERED / SELF-REPORTED / UNVERIFIED` and a link to the original Processor UI, not an inferred access grant or replay of the one-time operator URL.
5. Verify one process stops without killing the other; stale state transitions to offline. Stop/restart both with no duplicate identities or orphan processes. Copy Diagnostics on Identity has bounded persisted real log entries, redacted operator/session codes and both services' verified/unknown statuses.

## The next gate is separate, and not a blocker for the demo

Future secure flow: Identity authenticates principal + services and owns durable scoped grants; Engine verifies those grants at intake and before worker calls; Processor requests controls and shows real decision + Engine receipts. See [Identity security gate #15](https://github.com/Arameair/identity/issues/15) and [Processor integration #8](https://github.com/Arameair/Thought-Processor/issues/8). The engine-side experiment exists at `Arameair/Thought-engine:experiment/identity-grant-enforcement-20261024` but is **not** tested end-to-end against a working Identity permission endpoint. Never use Engine-local grant to mark it authorized.

## Evidence and acceptance

Report **PASS/FAIL/NOT RUN**, commands, actual host/OS, SHAs, ports, build and startup timings, log excerpts, error names, old-profile isolation, how to stop/rollback. Distinguish developer fixture, developer Linux machine and owner Windows machine. A quick real Processor-only run is already useful. No claim of complete Identity–Engine authorization, publisher-granted feature entitlement or finished onboarding without evidence.

**Operational rule:** if the next proposed task would delay the first successful real Processor startup, defer that task unless it fixes an observed blocker.
