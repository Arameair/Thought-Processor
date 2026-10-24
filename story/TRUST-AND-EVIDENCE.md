# Trust and integrity — the rule for every future screen

[← Story home](../README.md) · [Read the story](00-START-HERE.md) · [Blueprint seeds](BLUEPRINT-SEEDS.md)

**Owner direction (2026-10-24):** In a dynamic system, we should **never present invented, simulated, hardcoded or guessed status as real**. If authentic information is available, obtain it from the responsible service. If it is unavailable, say so. Preserve user trust over visual polish.

This is a **design rule**, not a claim of implemented verification.

## A story is allowed to imagine a user journey; the product is not allowed to imagine its telemetry

Maya, Compass, proposed screens, sample words and possible outcomes in these documents are **narrative devices only**. They are not screenshots or test results, and no such account, repository, entity, grant, worker call or thought is known to exist.

**When creating a working prototype, replace all illustrative status and event examples with actual values from the system being exercised.** Never pre-populate a "live" panel with a fictional GitHub username, connected account, permitted feature, service ID, grant, successful model call, Thought, uptime or performance number.

If the product cannot prove a fact, display **Not connected**, **Unknown**, **Not installed**, **Permission not verified**, **Not tested** or **Unavailable** as appropriate—each state is different. Do not use a generic green card that erases the distinction.

## Evidence contract for every operation

For **each value that might affect a user's decision**, record and display on demand:

| Required property | Meaning |
| --- | --- |
| **Claim** | Precisely what is asserted: repository ownership, engine process running, permission granted, worker request sent, result persisted |
| **Authority/source** | GitHub for repo permission; publisher for controlled feature delivery; Identity for local grants; Engine for its own queue, receipts and thoughts; launcher/OS for the process it actually started |
| **Observation method** | Real authenticated API response, verified process handle, signed decision, durable record, or explicit test-only fixture |
| **When checked** | Timestamp and expiration/staleness where applicable; yesterday's connection is not "online now" |
| **Identity and scope** | Exact entity, service, instance, action, repository and build as relevant |
| **Result and provenance** | Requested → accepted/denied/unavailable → observed effect, with decision/event/request IDs and accessible redacted diagnostics |
| **Confidence** | Verified, unverified, stale, or unknown; do not infer "authorized" from "running" or "running" from "configured" |

A disconnected client cannot manufacture "verified" from its local cache. A port that responds HTTP 200 is **reachable**, not automatically authentic. A model answer is not a persisted Thought. A publisher entitlement is not permission to inspect local data.

## Real failure states are part of the experience

The product needs truthful, separate states for **not installed, stopped, starting, running, authenticated, incompatible, not entitled, authorization denied, Identity unreachable, worker unavailable, queued, in progress, processed, saved, and unknown**. Report only states supported by actual evidence; the UI need not display all of them at once.

- When an authorization check cannot reach Identity, do not silently fall back to an Engine-owned ON switch.
- If a model call has already happened when a grant is revoked, do not claim that the transmitted content was withdrawn.
- If the publisher cannot confirm an entitlement, do not say the feature is licensed merely because a local file exists.
- If an upstream update is unverified, do not display a fabricated version or available update.
- If the Engine processes an observation but forms **no** Spark/Trace/Thought, show that actual outcome rather than invent a satisfying conclusion.

## Test fixtures and example observations

Prefer **real, user-approved sample actions executed against actual running services** to synthetic telemetry. A deliberately typed sentence such as "The player returned to Goldshire" is a **user-generated test observation**, not a claim that a game sensor actually detected it. Identify its source explicitly as **Manual demo source**.

Synthetic fixture input may remain inside isolated automated tests when necessary to verify an error, denial or edge case. Fixture results must be unmistakably labeled **TEST FIXTURE / NOT LIVE INTEGRATION**, kept out of ordinary production cards, logs and counts, and must never be used as evidence of production readiness.

If the integration cannot run, show **Blocked: actual service not available**. Do not paint a prewritten story over the missing integration.

## How this affects blueprint work

The blueprint must specify **who can prove each field, what constitutes fresh proof, and what the interface says when proof is absent** before designing visual badges and happy-path animations.

The story will use conditional language: *"When the connection is verified, the panel can show..."* rather than implying that a connection already exists. Screenshot/mockup artwork, if ever created, must be clearly labeled **CONCEPT, NOT LIVE DATA** and should avoid invented metrics or trustworthy-looking customer identifiers.

**Trust and integrity take precedence over an impressive first impression.**

*Recorded as owner direction, documentation only. No authentication, model, telemetry or end-to-end service has been implemented by these notes.*
