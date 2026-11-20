# workstream — Stage 2 / S2-A contract and reviewer provenance

**Status:** S2-A candidate on the draft PR. It needs independent audit and is **not adopted**. It is a pure Go library plus a read-only observer. It has no command, no daemon and no GitHub writes, and nothing in `tass-core` calls it. Plan: [PR #133 §4](https://github.com/Arameair/Thought-Processor/pull/133).

## Verified simplification: no separate ledger in S2-A

The handoff asked whether the owning PR's comments, plus verified actor and owner records, make a separate ledger unnecessary. **For S2-A they do.** The GitHub API already supplies each comment's ID, author login, `created_at` / `updated_at` and ordering, and the PR's exact head SHA. One workstream maps to its owning PR. Events are typed blocks inside ordinary PR comments. State is **derived on every read** and never stored. So there is no `team/ws/*.md` ledger, no `REGISTER.md`, and no second team index beside `docs/TEAM_ACTIVE.md`.

S2-A does **not** cover comment deletion between reads, durable receipts, or crash recovery. Those belong to S2-B (transport and durability), which is separately gated. In S2-A, an owner attestation pins a comment ID and body digest, so deleting or editing an attested comment fails closed. Unattested history has no such protection.

## Event block

One block per comment. Any other mention of `tass-workstream` makes the comment malformed, and the workstream goes to BLOCKED.

````
```tass-workstream-v1
{"v":1,"event_id":"ws-s2a-e1","workstream":"WS-S2A","repo":"Arameair/Thought-Processor","pr":134,
 "scope":"S2-A","kind":"CANDIDATE","actor":"Claude","commit":"<exact 40-hex>","evidence":"go test -race ./... PASS"}
```
````

The parser is strict:
- unknown fields, trailing data and versions other than `v: 1` are refused
- `commit` must be an exact lowercase 40-hex revision
- `CANDIDATE` and `AUDIT` must carry evidence (4 KiB max)
- `verdict` appears only on `AUDIT` and must be `PASS`, `NEEDS_REPAIR` or `BLOCKED`
- a block is 16 KiB max

## Verified evidence vs. untrusted claims

| Source | Examples | Trusted? |
| --- | --- | --- |
| GitHub API | comment ID, login, `created_at`/`updated_at`, PR head SHA | Yes, as GitHub reports them (`Record.Verified`) |
| Comment body | actor, kind, commit, verdict, evidence, anything that looks like an approval | **No.** It is a claim (`Record.Claim`) |
| Owner-held `Trust` (local, never read from GitHub) | role binding, task scope, dedicated-login principals, shared logins, per-comment attestations | Yes, by assumption. How it is produced and protected is decision D2 below |

Every event gets a **provenance** class:
- `OWNER_ATTESTED`: the owner attested this exact comment (ID + SHA-256 of its body) as this actor, kind and commit.
- `PRINCIPAL`: the comment came from a dedicated login that maps to exactly one actor and is not listed as shared.
- `UNVERIFIED`: only the text names the actor. This is the default today, because Claude, Codex and GPT-6 all post as `Arameair`.

**Only `OWNER_ATTESTED` or `PRINCIPAL` events can change state.** An `UNVERIFIED` event is kept in `Report.Rejected` as a visible claim. Plain text such as `Codex: PASS` is not an event at all.

## Transitions

| From phase | Event | Who | Result |
| --- | --- | --- | --- |
| `ASSIGNED` | `CANDIDATE c` | implementer | `AWAITING_AUDIT` (BUILD) |
| `AWAITING_AUDIT` / `AUDIT_BLOCKED` | `AUDIT c PASS` / `NEEDS_REPAIR` / `BLOCKED` | assigned auditor ≠ implementer, `c` = current candidate | `AUDIT_PASSED` / `NEEDS_REPAIR` / `AUDIT_BLOCKED` (AUDIT or RE-AUDIT) |
| `NEEDS_REPAIR` / `AUDIT_BLOCKED` | `CANDIDATE c'` | implementer; `c'` never failed and differs from `c` | `AWAITING_AUDIT` (REPAIR) |
| `AUDIT_PASSED` | `OWNER_ACCEPT c` | owner, attested or dedicated principal; `c` = passed candidate | `OWNER_ACCEPTED` |

Two derived phases sit on top of these:
- **`STALE_CANDIDATE`:** the PR head is not the latest candidate. No verdict applies to the current head.
- **`BLOCKED`:** an integrity failure needs owner reconciliation. The causes are a malformed block, an edited event comment, a changed or missing attested comment, a reused `event_id` or comment ID, an event naming another PR or workstream, an incomplete observation, or an invalid `Trust`.

`OWNER_ACCEPTED` is **not** a merge, release, installation or grant.

## Negative tests (`workstream_test.go`, `observe_test.go`)

| Case | Test |
| --- | --- |
| forged actor (attestation, principal, or commit mismatch) | `TestForgedActor` |
| modified / edited / deleted attested comment | `TestModifiedEvents`, `TestObserveDetectsEditsFromUpdatedAt` |
| replayed: repost, re-render, reused id, other PR, other workstream, duplicate comment | `TestReplayedEvents` |
| unknown actor | `TestUnknownActor` |
| self-review, including an implementer = auditor binding | `TestSelfReview` |
| stale source SHA: superseded audit, moved head, owner accepts the wrong revision, resubmitted failed revision | `TestStaleSourceSHA` |
| stale task scope | `TestStaleTaskScope` |
| fabricated owner approval: shared login, pasted "attestation", assistant as owner, accept before PASS | `TestFabricatedOwnerApproval` |
| plain-text `Codex: PASS` / approvals ignored | `TestPlainTextVerdictsAndApprovalsAreIgnored` |
| shared login never authenticates | `TestSharedLoginWithoutAttestationIsUnverified`, `TestSharedLoginCannotBeConfiguredAsPrincipal` |
| invalid `Trust` logins: case-insensitive duplicate principals (conflicting or same actor), malformed or padded principal/shared logins → always `BLOCKED` / `TRUST_CONFIG`, identical report across 1,000 runs | `TestPrincipalCaseCollisionIsTrustConfig`, `TestInvalidLoginsInTrustAreTrustConfig` |
| strict parsing (18 malformed forms) | `TestParseIsStrict` |
| partial or moving reads; read-only HTTP | `TestObserveRefusesPartialOrMovingReads`, fake server fails any non-GET |

## Reused vs. not reused

- **From #132 (Claude), reused after redesign:** a pure transition function (renamed `apply`), an exact-commit audit binding, the rule that a failed revision can't be resubmitted, and the separation of implementer and auditor.
- **From #132, not reused:**
  - the hash chain. A hash doesn't authenticate anyone, as the plan notes.
  - the canonical-rendering ledger, `REGISTER.md`, and the in-memory seen-history
  - the local `Policy` and the write path
- **From #131 (GPT-6), reused:** the single-owner-per-transition idea and receipts that cite exact evidence. Here, the receipt is the GitHub comment itself.
- **From #131, not reused:** the Python runtime, its in-memory CAS, and its per-actor handoff files.

## Decisions that need independent review or owner approval before adoption

- **D1:** use PR comments plus derived state instead of a separate ledger for S2-A, as above.
- **D2:** how the owner produces and protects `Trust`, especially attestations, which are the *externally grounded reviewer attribution*. One option is a local owner-held file the owner signs off on. Another is dedicated machine accounts for Codex and Claude, which turns `PRINCIPAL` into a real option. This is a hard stop before S2-B.
- **D3:** fail-closed policy. Any edit to an event comment blocks the workstream until the owner reconciles it. This is strict by design, and the owner may prefer a narrower rule.
- **D4:** event-block format, contract name, size limits and the `event_id` convention.
- **D5:** `CANDIDATE` events also need verified provenance. This is stricter than verifying only audits, but it keeps implementer identity grounded for the self-review check.

## Reproduce

```sh
cd core
gofmt -l ./workstream
go vet ./...
go test -race -count=1 ./...
go test -race -count=1 -v ./workstream
```
