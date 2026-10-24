# Architectural decision: Identity authority and feature self-registration

**Date:** 2026-10-24  
**Status:** Owner-approved *direction*; proposed connection contract only. **No implementation, working protocol or owner-tested integration.**  
**Repository / branch:** `Arameair/Thought-Processor` / `experiment/identity-authority-registration-20261024`  
**Baseline:** `alpha` @ `9fac9fa44cb020bb6f8aa6bcb1e9faeda6e0b47a`  
**Tracking issue:** [#6 — Identity authority and feature registration](https://github.com/Arameair/Thought-Processor/issues/6)  
**Attribution:** Human-directed decisions, documented with assistance from AI.

## Principle 1 — the architectural foundation

> **Identity owns the organization and authority of the system. Each feature owns its actual work.**

This principle applies beyond the Thought Engine. Every independent feature has an identity; the Engine is an **optional feature**, not part of the definition of Identity. Identity does not think, interpret raw logs, request model heartbeats, or require Thought Engine/Memory to be installed.

### What is centrally organized

**Identity is the registry and authority:**
- Persistent **entity IDs**, **feature/service IDs**, and relationships to owner and installation.
- Authorized accounts, access policy, narrowly scoped feature-to-feature grants and revocation.
- Configuration references, repository associations, directory allocation/policies and data ownership boundaries.
- Registered feature capabilities and instance locators; verified account/installation ownership and installation status when available.
- The fact that a feature exists or is installed **does not imply** it is running, connected, enabled, or authorized to read other features' data.

**Each feature remains responsible for its own work:**
- Runtime, readiness, lifecycle and internal implementation; it can start/stop independently.
- Its own isolated databases, event histories, queue and source-owned logs in Identity-managed/referenced storage areas.
- Only its own scoped access and data-processing responsibility.
- Direct authorized service-to-service traffic after connection setup, avoiding mandatory hops through Processor or Identity for every event.

**Thought Processor is the connection and observation point:**
- Discovers the authorized feature catalog from Identity; distinguishes a **feature definition** from a **live service instance**.
- Helps establish and inspect approved connections, showing the exact side/service that failed.
- Shows Option C lightweight application cards, one selected original UI, expandable Logs / Flow / Metrics inspector, Copy Context and Copy Logs when available.
- Can cache bounded runtime/status metadata for resilience, but **must not create a competing canonical identity/permission registry**.
- May route *control-plane requests* to the rightful service and show receipts, but is not a universal data-plane relay, a credential vault, a deployment system, or a Thought database.

**Thought Engine is a feature like others.** When enabled for an entity, it can process authorized source observations into working thoughts and directed evidence waves; Memory is another optional separately authorized feature. A memory can trigger a thought, but a thought is not a memory. Thought Processor does not own cognition or scheduling.

## Proposed operating topology

```text
                      owner / administrator
                              |
                              v
                      Identity authority
    IDs | storage ownership | accounts | grants | service registry
             ^                       ^                  |
             | self-registration     | grants           | approved catalog
             |                       |                  v
  +----------+----------+            |           Thought Processor
  |          |          |            |         discover / connect / inspect
 TASS     ScreenBridge   Thought Engine <------------ status/control ----+
  |          |          |                                      |
  |          |          +--> direct permitted worker request   |
  |          +---------------- direct approved observations ---+
  +--------------------------- independent feature lifecycle

  Memory (optional independent feature) -> approved recall -> optional thought input
```

The diagram is conceptual. Do not interpret the arrows as required proxies or implemented endpoints.

## Proposed startup and discovery behavior

1. **Independent startup.** A feature launches without requiring Processor. If already enrolled, it restores the same **stable service identity**, with a new **runtime instance ID** for this execution.
2. **Self-registration.** It announces a compact description to Identity: stable service ID, instance ID, service kind, supported protocol/capabilities, endpoint, last-seen time, version/build provenance if available, and viewer/diagnostic affordances. Runtime content and credentials are not advertised.
3. **Identity validates, records and governs.** Authentication/ownership are verified before treating the entry as trusted. First-time enrollment may require admin approval. A service claiming another feature's identity or a port responding HTTP 200 is *not proof*.
4. **Processor discovers.** It queries the Identity-authorized registry and makes a lightweight card. If discovery fails, manual endpoint entry can offer an explicitly **UNVERIFIED / READ-ONLY** diagnostic route rather than an authorization bypass.
5. **Owner approves a connection when needed.** Identity grants a specific source→destination capability, resource limits and duration. Destination **enforces** that authorization itself and returns a receipt; Processor reports this fact rather than inventing a green connection.
6. **Direct steady-state traffic.** Once authorized, features may communicate with one another, including Engine→worker and source→Engine; Processor reads bounded authentic status and trace reports.
7. **Stop/restart and revocation.** Closing Processor never stops the feature. A stopped feature becomes offline/stale without deleting its Identity record. Revocation stops protected operations at the receiving service, not simply in the UI.

### Persistent ID versus live instance

One installation can have a durable Thought Engine **service ID** across upgrades while each Play generates a fresh **instance/run ID**. Multiple concurrent instances must be distinguishable; their identities and authorized data roots cannot be silently conflated. An entity such as TASS is **not** the same thing as the runtime process running the TASS feature.

### Proposed feature registration payload (NOT AN IMPLEMENTED API)

```json
{
  "schema": "identity.feature-registration.v0-DRAFT",
  "service_id": "<stable Identity-issued ID>",
  "instance_id": "<run-scoped ID>",
  "kind": "thought-engine",
  "protocol": "feature-observation.v0-DRAFT",
  "capabilities": ["health", "diagnostics"],
  "endpoint": "http://127.0.0.1:<assigned-port>",
  "build_sha": "<reported exact SHA or unknown>",
  "last_seen": "<UTC timestamp>"
}
```

The content alone carries **no authentication**. The final contract needs an authenticated enrollment mechanism, an authenticated transport and replay/freshness checks; do not implement the draft JSON as a trusted registration simply because it looks correct. Specific host, port, schema and capabilities require future review.

## Storage boundaries: clean machines and repos

Identity **allocates/references** storage locations and access policy. A feature writes its own authorized data directly to its isolated storage; bulk game logs, audio/video and thought records should not cross through Identity's database or into git checkouts. Repositories hold code and approved **non-secret** configuration/provenance. Credentials stay in an approved protected credential store. Migration, backup, quota and retention need separately tested policies.

**Avoid two copies of truth:** Thought Processor may hold a bounded, access-controlled index of recent status/trace metadata, **not authoritative records or copied credentials**. Stale cache never confers permission.

## Failure and security behavior

| Situation | Correct interpretation |
| --- | --- |
| Feature absent | Identity can still work; installed/known does not mean running |
| Identity offline | Features continue nonprivileged independent work; previously granted protected operations only under an explicitly valid short-lived lease; **no new grants** |
| Processor offline | Existing services and authorized direct connections continue; no forced stop |
| Feature registers itself | Discovery/identity claim pending verification, **not automatically authorized** |
| Registered but wrong version | INCOMPATIBLE; show protocol/build evidence and owner action |
| Feature asks for new log source | Require identity-scoped source authorization, then source/destination enforcement |
| Thought or Memory disabled | Other apps remain operational; no automatic long-term Memory from Thought |
| Revoked grant | Receiving feature rejects new protected activity even if Processor/UI is closed |
| Source logs unavailable | Display unknown/unavailable; don't synthesize events or success |
| Service restarts on another port | Match stable service + fresh instance authentication, not port number alone |

Previously discussed bounded offline grants were **direction only**, not a selected TTL or tested revocation scheme.

## First small acceptance experiment

**Goal:** Show Identity + Processor + one other independent feature working together, not simulate a marketplace.

1. Start Identity by itself. Create/enroll one test service and preserve the ID after restart.
2. Start Processor independently. Verify Identity appears as a real service card only after truthful discovery/verification; original UI opens or a clearly labeled fallback.
3. Start a second feature independently and self-register without a new hard-coded Processor tile.
4. Confirm actual service IDs, different runtime IDs, ports, source SHAs if available, and separate data roots.
5. Stop Processor. The features continue. Restart Processor and restore visible status without duplicate identities.
6. Test two authorization outcomes against the **receiving service**: one denied before grant and one allowed after valid grant. If Identity auth is unavailable, label this **BLOCKED / NOT YET BUILT** rather than silently using an insecure local toggle.
7. Revoke grant and verify a subsequent protected operation is denied. Capture full run IDs, request/correlation IDs, latency, errors, safe logs and owner decision.

**All checks NOT RUN.** The current Alpha is documentation-only. Do not pretend this note proves working cross-service authentication.

## Decisions still to settle

- Where does a newly installed, unknown feature get its first Identity-issued service ID? Does enrollment require a manual owner action once?
- How does Identity authenticate local and later remote services, including across Windows/WSL/VM boundaries?
- Does Processor use live Identity lookup, subscription, or a bounded local status cache? How does it reconnect after an outage?
- What is the exact grant, expiration, renewal and revocation contract for direct services?
- What are the storage-root conventions and feature data-migration/backup policies?
- What status may be shown publicly versus only to an authorized developer?
- Which controls remain in Processor vs the original feature UI? Avoid duplicating the owner's configuration.

## Related decisions and theory

- [Previously selected self-registration/connection directions](https://github.com/Arameair/Thought-Processor/blob/experiment/six-design-decisions-20261023/docs/discussions/2026-10-23-six-design-decisions.md)
- [Open connection-contract questions](https://github.com/Arameair/Thought-Processor/blob/experiment/six-design-decisions-20261023/docs/discussions/2026-10-23-feature-connection-contract.md)
- [Identity architecture](https://github.com/Arameair/identity/blob/main/docs/ARCHITECTURE.md)
- [Thought-directed waves theory](https://github.com/Arameair/Thought-engine/blob/experiment/thought-engine-v1-20261023/docs/THOUGHT_DIRECTED_WAVES_THEORY_2026-10-24.md)

**Governance:** This note is on an experiment branch started from owner-approved Alpha. **No Alpha, Stable, or Main changes.** Keep the original six-decision branch and newer direction proposals available for owner comparison before implementation.
