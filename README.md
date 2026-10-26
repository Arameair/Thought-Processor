# Thought Processor — GPT-6 Frictionless First-Run Experiment

**Independent candidate · 2026-10-26 · Not owner-approved Alpha**

A runnable Windows/Linux Thought Processor candidate focused on a **guided first-run experience** instead of a diagnostic dashboard.

The path for a new user is deliberately simple: **Connect Identity → Choose/Create Identity → Discover and review features.** Identity pairing and GitHub approval remain explicit. Processor automatically discovers local Identity locators, remembers its previously verified pairing, and renders truthful feature eligibility, installation state and source permissions. Developer Mode supplies status, logs, metrics and flow separately.

**[Windows trial — start here](docs/GPT6_WINDOWS_FIRST_RUN_2026-10-26.md)** · **[Development timeline and test evidence](docs/GPT6_DEVELOPMENT_EVIDENCE_2026-10-26.md)** · **[CI builds and Windows download](https://github.com/Arameair/Thought-Processor/actions/workflows/p0.yml)**

## Run directly

On Windows, download the `GPT6-Thought-Processor-Windows` artifact from a successful branch-specific Actions run, extract `thought-processor.exe` and double-click it. It opens its own local browser workspace. **Identity must already be independently installed/running, or separately started.** A manual Go build is a fallback, not a requirement for the Processor executable.

At source checkout: `cd processor && go build -o thought-processor.exe .`, then execute the resulting binary. Default port is `7860`. The older diagnostic UI is still at `/diagnostics`, and the prior GPT-6 onboarding comparison lives at `/previous-onboarding`.

## Architecture and security

- Processor is the human/admin workspace, not a source of identity or grants.
- Identity API `identity/1-alpha` owns canonical `ent_…` identities and entitlement checks.
- Pairing requires a person typing Identity's own one-time code. The Processor Go backend checks a nonce-bound HMAC and exact `identity_system_id` on every protected response.
- The generic install contract `feature-install/1-alpha` uses feature-ID endpoints. It checks exact release metadata, per-user approval, byte count and SHA-256 before a versioned local receipt. Installation is **not** an Engine start/grant.
- New identities have no source permissions. Runtime grants are outside P0; Identity currently returns DENIED.
- Default UX does not expose sensitive tokens or codes in diagnostics. Full Windows DPAPI/ACL audit remains open.

**Not yet working/validated:** a genuinely published approved Thought Engine release (none found at review); server-side privacy of private feature catalog names; independent publisher signing; a clean owner's Windows runtime with screenshots; full managed Identity installation; runtime grants/cognition.

## Attribution and branches

Created directly from owner-controlled Processor `alpha` at `9fac9fa44cb020bb6f8aa6bcb1e9faeda6e0b47a`. Initial experiment commit `a86efcd98c01387cca7cfa9de6be3c42b249fa69` explicitly reused the author's earlier **GPT-6 P0** backend `260c90556d3eeea5c4a19733cc0d1813c803c863`, **not Claude's parallel branch or UI**. See the detailed evidence ledger for fair development-efficiency comparison.

No merges, promotions, tags or releases. Owner must separately approve any Alpha changes. Recommended **REVISE** before promotion.
