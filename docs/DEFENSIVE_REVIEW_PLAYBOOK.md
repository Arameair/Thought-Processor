# Defensive review playbook — authorized TASS engineering

**Status:** proposed documentation guidance (2026-11-19). No change to CI jobs, enforcement, permissions, grants, live systems, or required test coverage. This guidance supplements [WORKFLOW.md](WORKFLOW.md) and the repository [AGENTS.md](../AGENTS.md). It is not an access waiver or a substitute for independent review.

## Purpose and approved scope

TASS reviewers inspect owner-controlled source and isolated fake-service environments for defects in authorization, durability, replay prevention, recovery and delivery. Begin a review by stating its authorized repository/PR, exact refs, defensive objective, allowed local fixtures, and prohibited production actions. Use Code/Engine/Identity/Bridge owners' existing acceptance contracts; do not invent new requirements to justify repeated reviews.

A model access restriction is **not** a security finding, and a test PASS is **not** proof that a model-specific access restriction has been lifted. Do not use wording changes, fragmented requests, alternate accounts or tools for the purpose of circumventing safety controls. Follow the published product guidance when authorized work repeatedly requires broader security access.

Official references:
- [Additional safety checks in ChatGPT, Codex and API](https://help.openai.com/en/articles/20001326-additional-safety-checks-for-biological-and-cybersecurity-requests-in-chatgpt-codex-and-the-api)
- [OpenAI Daybreak authorized access overview](https://help.openai.com/en/articles/20001258-openai-daybreak-trusted-access-for-cyber-overview)
- [Contact support](https://help.openai.com/en/articles/6614161-how-can-i-contact-support)

## Required acceptance matrix

Track each required invariant as a row. Neither removing a test nor softening a negative assertion is a valid response to an AI access interruption.

| Evidence category | Required artifact / acceptance condition |
| --- | --- |
| Revision identity | Exact repository, base/head SHA, tested tree and synthetic-merge ref, and changed-path list |
| Functional baseline | Normal owner message produces the intended one result under permitted fake-service tests |
| Authorization | Identity grant checks, single-use decision, revocation/lock handling, least privilege, no unauthorized calls |
| Source deduplication | Duplicate events never create a second answerable request; cover rollover, concurrency and restart |
| Persistence and recovery | Durable claims/receipts and conservative behavior on incomplete, corrupt or missing state |
| Delivery/idempotency | No second send after claimed, consumed, failed or uncertain delivery state |
| Isolation/privacy | No production tokens or user data in fixtures, logs or public evidence; component secrets separated |
| Compatibility | Schema, exact dependency pins, and representative composed fake-service tests |
| Test execution | Unit/integration/race/crash-or-restart/negative cases as applicable; exact commands, environment, PASS/FAIL/NOT RUN |
| Independent review | Auditor identifies observed facts versus implementer claims, unresolved gaps and explicit PASS/NEEDS REPAIR/BLOCKED |

Test contracts and expected outcomes must be independently inspectable. Detailed code-level traces and test artifacts may remain in scoped private review attachments rather than being reproduced in conversational summaries. This does **not** excuse missing evidence.

## Short review handoff template

> **Authorized scope:** repository + PR + owner-controlled offline fixtures; exact head/base.  
> **Goal:** verify specified defensive acceptance invariants, without production access.  
> **Evidence:** changed paths, existing test names and reports, exact revision/environment.  
> **Required checks:** list the missing acceptance rows; preserve all existing tests.  
> **Output:** PASS / NEEDS REPAIR / BLOCKED, separately for each component; residual risks and owner decisions.  
> **Restrictions:** no deployment, merge, production credentials or live Discord action without separate authorization.

## When a cybersecurity response is not displayed

1. Stop the interrupted model-assisted operation. Record the visible warning, date/time, client/product, exact task category and the PR/ref being reviewed. Preserve the request privately if permitted, but do not copy tokens, private records or sensitive payloads into support tickets.
2. Record completed local and CI results as **OBSERVED** and unavailable model-audit checks as **NOT VERIFIED / BLOCKED**. Do not call the source defective solely because of the product warning.
3. Continue ordinary nonblocked work (local automated tests, CI logs, independent reviewer) within authorized access. Do not disguise, split or iterate restricted requests to evade controls.
4. Use the product feedback control or OpenAI Support with the error/screenshot, authorization context and non-sensitive request summary. Daybreak access is optional and separate; it does not replace project authorization or guarantee every request.
5. Resume the same bounded review when legitimate access is available, preserving the test matrix and exact SHA. Never claim security PASS while required evidence is missing.

## Residual risks and owner acceptance

- Local durable indexes can still lose trust after wholesale state restoration, rollback or malicious modification by a writer with equivalent filesystem privileges.
- Operator-acknowledged recovery may deliberately hold unanswered messages. Any recovery cutoff must be explicitly validated and documented, not inferred from an untrusted host clock.
- CI does not establish behavior in the owner's installed environment, with real Discord or with current credentials and permissions.
- A PASS on an isolated component does not authorize a composed deployment or live reply grant.
- The owner must explicitly accept known limitations and separately authorize installation, grant enablement and live tests.

## Change-control guardrails

This document does not change GitHub Actions YAML, test commands, branch protections, authorization code, or release policy. If a future reviewer proposes changing a test, require a separate issue describing why the test is invalid, a replacement assertion, CI evidence and owner authorization. Prefer substantive engineering fixes over wording-only audit cycles.
