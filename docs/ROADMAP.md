# Roadmap — smallest credible control point first

**Planning only.** Phases are **test gates**, not dates, release promises, or automatic implementation authorization.

| Stage | Objective | Evidence that proves it | State |
|---|---|---|---|
| **0 — Direction** | Clear roles, owner-selected Option C cards, branch/provenance policy | Owner reviews consolidated docs; open choices recorded | **Under review** |
| **1 — One real card** | Processor starts independently and discovers **one actual Identity runtime instance** | Correct service ID, protocol, branch/SHA, port and status; Select opens original UI or honest fallback; Stop changes card state without stopping Processor | Not built |
| **2 — Dynamic cards + diagnostics** | Discover a second compatible running application without hard-coded tile; one selected real UI plus inspector | Accurate readiness, switch performance, real logs/metrics, Copy Context; no hidden full previews | Not built |
| **3 — Connected flow** | Verify a real Engine + current fast worker; Processor-assisted discovery, **direct Engine API heartbeat** and trace view | Engine request ID, model call, measured response, persisted Thought/evidence, truthful failure ownership | Not built |
| **4 — Feature/feed authorization** | Identity grants and individually permitted log feeds, without changing source-app lifecycle | Denied/allowed identities, source pause with source still running, logs/event IDs and queue receipts | Not built |
| **5 — Replaceability and resilience** | Model/provider swap, worker downtime, queue/retry policy, multi-identity isolation and restart | No data loss, bounded load, no unsafe duplicate requests, provenance survives Stop/Play | Not built |
| **6 — Owner release** | Reproducible approved candidate and rollback | Project Controller manual tests, owner-selected exact commit, explicit Alpha then Stable then Main decisions | Not built |

### Recommended next small experiment

**Start with one real Identity card.** This isolates the most distinctive new idea—the Processor's live application catalog—without first needing to solve the whole Identity-to-Engine grant protocol or rewrite either Thought implementation.

The first experiment should prove: **discovery → verified status → one original UI → stop/restart detection → redacted diagnostics**, with measured overhead. A second service is the next test, not eight imaginary cards.

Separately, keep discussing the canonical Thought Engine choice. Avoid silently wiring both independent Thought stores into one identity.

### Quality gates for every phase

- **Independence:** Processor never silently starts or stops other applications.
- **Proof:** statuses refer to actual running processes, not repository URLs or demo JSON.
- **Traceability:** request/correlation ID, build/source, latency, error owner and durable record when applicable.
- **Resource cost:** run health checks efficiently; no live browsers hidden in collapsed cards.
- **Permissions:** discovery never implies authorization, and feature disconnect never disables independent sources.
- **Model neutrality:** current Chat-Jimmy work proves only one adapter, not a permanent dependency.
- **Rollback:** exact branch/SHA, data preservation and owner-approved promotions.

### Decisions that may change the sequence

See [DECISIONS](DECISIONS.md): canonical Engine; registration/discovery mechanism; authentication and permission flow; per-source subscriptions; Processor-offline behavior; retention; embedding; worker adapter capabilities. Record actual owner decisions before treating a proposed protocol as required.

**Not in the initial scope:** full feature marketplace/billing; automated code changes; remote cross-device control; every app rendered simultaneously; Thought Processor authoring its own thought history.
