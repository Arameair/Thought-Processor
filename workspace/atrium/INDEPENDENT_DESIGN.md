# Atrium — ChatGPT's independent Thought-Processor interface experiment

**Status: experimental, NOT deployed or owner-verified.** This is a *user-facing Thought-Processor design*, not a second model, second Identity authority, or a replacement for the existing Workshop executor.

## Working agreement with Claude's independent interface

The owner requested **two independent UI approaches** to the same TASS system, to compare the best useful ideas. ChatGPT owns this Atrium experiment; Claude may implement its own interface on a separate branch. Neither assistant's assertions count as test evidence.

**Shared interfaces, not shared appearance:**
- Same running TASS Workshop session and model; do not create a second chatbot.
- Same Identity authorization boundary and owner approval requirements.
- GitHub branches/PRs and the same traceable test results; never merge an experimental branch without Dustin.
- Reuse existing API concepts: status, timeline, message, proposal, decision, execution result, and audit. If an API needs changing, document the contract and compatibility.
- Preserve Workshop execution approval; neither experimental UI may bypass it.
- No automatic permissions, untrusted local HTML from TASS on the execution origin, or public exposure of local services.

**Atrium's own design hypothesis:** a spatial service map becomes useful only when tied to real state. Display *discovered* tools separately from *tested* tools. Provide visible communication, action proposals, diagnostics and evidence. The owner should be able to work through one task without switching to a terminal repeatedly.

## Evidence and comparison contract

Test both interfaces against the same local model and identical tasks, with timestamps and logs:

| Test | Observable acceptance | Evidence source |
|---|---|---|
| Open | Both separate pages load without breaking Workshop | Browser + HTTP response |
| Real chat | Distinctive message gets a genuine TASS reply, reflected in Workshop | Workshop timeline |
| Library discovery | Show actual book catalog; never count decorative map nodes | Workshop mirror |
| Ability execution | TASS opens Look Around, proposes one cell, owner approves, audited run succeeds | Workshop audit/result, code hash |
| GitHub visibility | Optional feed posts **new** bounded event receipts to owner's private issue; no secrets or model thoughts | Identity Liaison + GitHub comment |
| Safety | Neither interface grants permissions or executes code without a separate owner decision | Negative API tests |
| Offline | Clear error state; no pretend responses | Browser + API |
| Usability | Count owner steps, manual commands, broken transitions and recovery actions | Owner observations |

A CI PASS using a fake model is **only test harness evidence**. It is not a real TASS acceptance pass.

## Current differences

Atrium provides a visual map, an idea garden, design lab, real TASS chat bridge, a connection diagnostic, a first-task prompt, and an audit-backed per-book ability list with test prompts. Its collaboration publisher is an optional separate service with **no GitHub token or network access**. It queues bounded comments through the already-configured Identity Liaison to a private GitHub issue only when Dustin opts in.

A Claude version may choose an entirely different layout or workflow. Compare them against the shared acceptance tests rather than forcing design convergence. Useful features may later be combined via small reviewed PRs.

## Known limitations

- The existing experimental Workshop has authentication/authorization gaps. Loopback is not authentication.
- A reported execution PASS comes from the Workshop/audit, not independent validation of all file effects.
- TASS cannot yet launch host windows. Host browser control needs a separately authorized host-side capability.
- The GitHub feed requires the existing Identity Liaison to be active and authorized for the target private repo.
- Owner acceptance has not been performed on this experimental branch.
