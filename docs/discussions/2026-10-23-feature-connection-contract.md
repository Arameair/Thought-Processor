# Shared connection contract — questions for next design session

**2026-10-23 · OPEN DESIGN OPTIONS, not an implemented API.**  
[Owner-selected six decisions](2026-10-23-six-design-decisions.md).  
Human-directed project notes prepared with AI assistance.

The UI will only be truly dynamic if **each independently runnable application can communicate a small, consistent set of facts** to Thought Processor. We should settle a minimum contract, *not* develop a universal middleware or force all apps through Processor.

## Minimal responsibilities for a compatible application (draft)

| Category | Candidate information | Questions to settle |
| --- | --- | --- |
| Discovery | Stable service **kind**, unique **running instance ID**, advertised protocol/capabilities, endpoint and identification proof | Self-register using configured Processor address vs local discovery broker? Can two instances of same project coexist? |
| Runtime | Health/checked-at, ready/busy/degraded/errors, current run ID | Pull at a bounded interval vs push on state changes? What happens while Processor is offline? |
| Source/version | Repo reference, branch, exact commit/build where available; runtime provenance | How to verify these aren't untrusted self-reports? What if a build wasn't from GitHub? |
| Viewer | Optional original UI URL, supported preview/embedding mode, required authentication | What if X-Frame-Options/CSP, browser sessions or origins prohibit embedding? Open real UI instead, never spoof it. |
| Authorization | Identity and feature/scope requested, grant reference/expiry and service enforcement result | What format lets a service verify permission **locally and quickly**? How do we revoke and renew short offline-limited grants? |
| Activity | Bounded latest event, counts, timestamp, correlation ID | What is safe to show on a thumbnail without private payload disclosure? |
| Diagnostics | Sanitized read-only log subset and metrics endpoint or stream, run IDs, trace IDs | On-demand extraction vs event subscription? How to report missing spans and truncated exports? |
| Connection control | Supported connect/disconnect/refresh/inspect actions with error/status receipts | Which operations are safe without touching lifecycle or spawning hidden dependencies? |

## Three discovery implementation sketches — not yet chosen

- **A: A standard self-registration HTTP request** to a configured Processor address, plus manual Add Service. Easy to prototype but requires a known address and renewal/re-registration rules.
- **B: A small local registration record/manifest** written by independently running apps (with freshness/instance validation) and manually imported for nonlocal apps. Little continuous traffic but platform/file-permission complications.
- **C: A small registration protocol plus optional service discovery adapter**, supporting remote/local hosts as a future extension. Flexible but higher implementation cost.

The **selected product behavior** is *self-registration with manual connection fallback*. The above are implementation alternatives *within that decision*; no one has approved a specific transport or schema.

## Boundaries agreed already

- Project Controller is optional personal tooling and must **not** be needed for discovery, startup, registration or using Identity/Thought Processor.
- Identity decides who and what may access sensitive data; *each receiving service* verifies grant conditions. Discovery never implies permission.
- Thought Processor assists connection and observes status; Engine makes its **own worker API heartbeat requests**, potentially directly, without a Processor proxy in the fast path.
- A feature may remain independently usable even with Processor down.
- Collapsed cards remain lightweight; one selected original app preview at a time.
- Original service retains logs; Processor correlates traces and can collect redacted excerpts when requested.
- Bounded authorization continues briefly on Identity outage; exact expiry/revocation protocol remains open.

## First proposed test

Start **Identity** and **Thought Processor** as independent processes (not via Project Controller) and demonstrate verified Identity registration, one status card, original UI on expansion (or explicit fallback), stopped/disconnected display, and a Copy Context showing honest run/source/branch details. Then add a second basic compatible service to test dynamic card discovery **without modifying Processor code**.

No source code, API or timing benchmark is implemented by this note. Record actual test evidence and exact SHAs before claiming success.
