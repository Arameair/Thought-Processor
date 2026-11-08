# Independent TASS Stage 2 Revision 3 Design Re-review

**Date:** 2026-11-08

**Disposition:** **FAIL — five specification blockers remain before implementation**

**Target:** Thought-Processor PR #101 commit `d21caf54c6e644bcd713f1ca62e579f52c67ff58`

**Baseline source rechecked:** Core `094f43ba2bc167ace8835bf286f37951fee432c9`, Thought Engine `d69d3917d1601d806d4b2ba2254693e1adb92b6b`, Identity `64fdf8e8cf926df5ea6bb6c23fbab864d0f09e3b`.

This is a design/source review. No Stage 2 implementation exists at the target revision, so no runtime or failure-injection test was executed. No live installation, grant, journal, service, model, or nudge was touched.

## Bottom line

Revision 3 makes G1's core mutual-exclusion idea feasible: a real claim and a void compete to create the same inode with `O_EXCL`, so a successfully durable void excludes a later claim on a supported local POSIX filesystem. That is a design PASS, not an executed-test PASS.

G2 is not yet safe to implement. The proposed launcher publishes restored data first and writes the fence afterward, leaving an unfenced crash state. The proposed physical-line count can also under-cover a valid final JSON record without a trailing newline. A restore must be fenced before the restored data becomes live, or a separately durable pending marker visible to Core must keep all nudge processing locked through the transition.

## PASS / FAIL / OPEN findings

### Guarantees G1 and G2

| ID | Result | Finding and exact evidence | Smallest necessary correction |
|---|---|---|---|
| R3-01 | **PASS (design feasibility)** | G1's claim/void exclusion is mechanically sound if both paths create the exact same `claim.json` with `O_CREAT|O_EXCL`, sync the file and parent directory, and treat every existing malformed file as `RECOVERY_REQUIRED`. The design states the supported local-filesystem assumption and excludes NFS/SMB/drvfs ([revision 3 §4.1–4.2](https://github.com/Arameair/Thought-Processor/blob/d21caf54c6e644bcd713f1ca62e579f52c67ff58/core/docs/STAGE2_NUDGE_PROCESSING_DESIGN_20261108.md#L123-L140)); the paused-handler ordering is specified in §4.5 ([lines 169–180](https://github.com/Arameair/Thought-Processor/blob/d21caf54c6e644bcd713f1ca62e579f52c67ff58/core/docs/STAGE2_NUDGE_PROCESSING_DESIGN_20261108.md#L169-L180)). PID namespaces do not defeat filesystem `O_EXCL`. | Keep this primitive. Make the state reader fail closed on empty, short, unparsable, wrong-schema, or impossible claim content. Verify the actual owner data filesystem before enabling the route. |
| R3-02 | **FAIL** | G1's void route is authorized only by the shared Engine key and explicitly needs no Identity decision ([§4.5 line 173](https://github.com/Arameair/Thought-Processor/blob/d21caf54c6e644bcd713f1ca62e579f52c67ff58/core/docs/STAGE2_NUDGE_PROCESSING_DESIGN_20261108.md#L169-L176), [API contract lines 228–237](https://github.com/Arameair/Thought-Processor/blob/d21caf54c6e644bcd713f1ca62e579f52c67ff58/core/docs/STAGE2_NUDGE_PROCESSING_DESIGN_20261108.md#L228-L237)). A key holder can create arbitrary terminal tombstones for any entity/nudge and consume the 1,000-folder limit. Preventing work is an availability mutation; “it can only prevent” is not an authority proof. | Require an owner-bound, single-use void capability tied to entity, nudge ID, and existing Core submission. The narrowest consistent option is a fresh Core-minted/Engine-redeemed Identity decision for a dedicated void action that remains available even if the process grant was revoked. |
| R3-03 | **FAIL** | G2 publishes restored `data/` before writing its fence. Current `restore_data()` moves live data aside and extracts directly into `$APP_HOME`, then returns ([pinned `tass.sh` lines 332–345](https://github.com/Arameair/Thought-Processor/blob/094f43ba2bc167ace8835bf286f37951fee432c9/core/tass.sh#L332-L345)). Revision 3 adds the fence only after unpack ([§4.6 lines 182–194](https://github.com/Arameair/Thought-Processor/blob/d21caf54c6e644bcd713f1ca62e579f52c67ff58/core/docs/STAGE2_NUDGE_PROCESSING_DESIGN_20261108.md#L182-L194)). A kill after extraction but before fence durability leaves live-looking restored drafts with no PRE_RESTORE proof. | Extract to a private staging directory, strictly validate the restored nudge journal, create and sync the fence inside staged `data/core`, sync directories, and only then publish the staged data. Also durably record restore-pending before touching live data; Core must refuse all nudge processing if pending/fence/operation metadata is absent, torn, or inconsistent. Clear pending only after the fenced data and operation phase are durable. |
| R3-04 | **FAIL** | `draft_lines = wc -l` is not a safe boundary. The pinned parser accepts valid JSON from a final record even without a newline, while skipping malformed lines ([`readNudges` lines 207–232](https://github.com/Arameair/Thought-Processor/blob/094f43ba2bc167ace8835bf286f37951fee432c9/core/tass/nudge.go#L207-L232)). `wc -l` undercounts a newline-free final record, so that restored draft could appear post-fence and processable. Invalid lines also destroy a simple valid-record-index mapping. | Replace line count with a strictly validated physical byte boundary plus SHA-256 of that exact prefix, or an explicit immutable set of restored nudge IDs. Core must verify the fence against the journal and lock processing on truncation, invalid JSON, missing newline if required, prefix mismatch, or ambiguous record-to-boundary mapping. |
| R3-05 | **FAIL** | The updater operation record is not specified as durable evidence. Current `op_begin`/`op_set` use rename without file or directory `fsync`, and `op_end` simply removes the record ([pinned `tass.sh` lines 355–370](https://github.com/Arameair/Thought-Processor/blob/094f43ba2bc167ace8835bf286f37951fee432c9/core/tass.sh#L355-L370)). Core sees only `/data/core`, not `$DEPLOY/operation.env` ([compose mount](https://github.com/Arameair/Thought-Processor/blob/094f43ba2bc167ace8835bf286f37951fee432c9/core/compose.yaml#L112-L126)). Therefore Core cannot currently correlate a missing/stale fence to the interrupted restore record. | Make the restore transition protocol explicit and durable. Prefer staged fenced publication; additionally expose a minimal read-only restore-generation/pending record to Core or place it in a persistent location Core can verify. File+parent-directory sync and ordering must be specified for begin, fence, phase, publish, and clear. |
| R3-06 | **PASS (source coverage)** | Every supported data-changing restore currently calls `restore_data()`: failed-update auto-restore ([lines 532–539](https://github.com/Arameair/Thought-Processor/blob/094f43ba2bc167ace8835bf286f37951fee432c9/core/tass.sh#L532-L539)), `rollback --restore-data` and its guard restore ([560–583](https://github.com/Arameair/Thought-Processor/blob/094f43ba2bc167ace8835bf286f37951fee432c9/core/tass.sh#L560-L583)), `restore-previous` and its guard restore ([600–640](https://github.com/Arameair/Thought-Processor/blob/094f43ba2bc167ace8835bf286f37951fee432c9/core/tass.sh#L600-L640)), and `recover` ([647–665](https://github.com/Arameair/Thought-Processor/blob/094f43ba2bc167ace8835bf286f37951fee432c9/core/tass.sh#L647-L665)). Code-only rollback correctly does not restore data. | Preserve the shared function, but test every caller and nested guard/recovery restore after implementing the atomic protocol. External/manual restores remain an explicitly documented non-guarantee. |
| R3-07 | **OPEN (execution)** | X9b, X11b/c, F19–F22, two PID namespaces, disk-full, short-write, and kill-at-boundary tests have not run because the design is not implemented. | Do not convert R3-01 or R3-06 into runtime PASS until the isolated implementation tests return server-side POST counts and durable-file evidence. |

### Remaining authorization, concurrency, recovery, privacy, and evidence

| ID | Result | Finding and exact evidence | Smallest necessary correction |
|---|---|---|---|
| R3-08 | **FAIL** | The authorization flow contradicts itself. Identity is specified to require a binding for every `cognition.nudge.process` authorization ([§2 line 62](https://github.com/Arameair/Thought-Processor/blob/d21caf54c6e644bcd713f1ca62e579f52c67ff58/core/docs/STAGE2_NUDGE_PROCESSING_DESIGN_20261108.md#L58-L65)), but C2 performs an unbound authorization and expects it to pass before replay ([§3 lines 83–89](https://github.com/Arameair/Thought-Processor/blob/d21caf54c6e644bcd713f1ca62e579f52c67ff58/core/docs/STAGE2_NUDGE_PROCESSING_DESIGN_20261108.md#L83-L89)). As written, C2 either always fails or Identity weakens the binding requirement. | Make one fresh **bound** authorization before C3. Use it to authorize an exact-tuple replay; if there is no existing run, carry the same decision into C5/Engine redemption. Remove the second authorization. |
| R3-09 | **FAIL** | “One active run” is a check-then-create race across different IDs: C4 reads all run states, then C5 creates only this nudge's file ([§3 lines 86–93](https://github.com/Arameair/Thought-Processor/blob/d21caf54c6e644bcd713f1ca62e579f52c67ff58/core/docs/STAGE2_NUDGE_PROCESSING_DESIGN_20261108.md#L86-L93)). Two simultaneous POSTs for different drafts can both pass C4 and each win its unrelated `O_EXCL`. | Serialize C1 through successful C5 under a dedicated Core run mutex and state that the supported topology has one Core process. If multiple Core processes sharing data are supported, add a filesystem-global active-run claim instead. Add a barrier test for different nudge IDs. |
| R3-10 | **PASS (design feasibility)** | The bound digest covers action, owner, Engine entity, Thought, nudge, attention digest, recipient, and expiry ([§2.1 lines 67–74](https://github.com/Arameair/Thought-Processor/blob/d21caf54c6e644bcd713f1ca62e579f52c67ff58/core/docs/STAGE2_NUDGE_PROCESSING_DESIGN_20261108.md#L67-L74)). P1 is after the model slot and before the durable HTTP-intent marker; P2 is immediately before save ([§3 lines 95–115](https://github.com/Arameair/Thought-Processor/blob/d21caf54c6e644bcd713f1ca62e579f52c67ff58/core/docs/STAGE2_NUDGE_PROCESSING_DESIGN_20261108.md#L95-L115)). This is feasible and closes the prior substitution/revocation gaps once R3-08 is corrected. | Preserve the exact tuple and point-of-use ordering. Runtime result remains OPEN. |
| R3-11 | **PASS (design feasibility)** | Writing and syncing `http.json` before the model POST can prove “no HTTP was started” only when the marker is absent and the run files are otherwise valid. Once the marker exists, the only truthful client conclusion is UNCERTAIN until a response is durable. The design's failure matrix uses fixture-server POST receipt as its test oracle ([§7 lines 245–258](https://github.com/Arameair/Thought-Processor/blob/d21caf54c6e644bcd713f1ca62e579f52c67ff58/core/docs/STAGE2_NUDGE_PROCESSING_DESIGN_20261108.md#L245-L258)). | Rename user-facing `attempted` to `attempt_uncertain` or define it explicitly as “intent marker durable; server receipt unknown.” Tests must report fixture POST counts, not infer them from `http.json`. |
| R3-12 | **OPEN, not a blocker** | The privacy design is feasible but internally stale: §5 still lists removed `Core nudge-runs.jsonl` ([lines 210–224](https://github.com/Arameair/Thought-Processor/blob/d21caf54c6e644bcd713f1ca62e579f52c67ff58/core/docs/STAGE2_NUDGE_PROCESSING_DESIGN_20261108.md#L210-L224)), and S15 says only that attention appears where §5 says ([line 313](https://github.com/Arameair/Thought-Processor/blob/d21caf54c6e644bcd713f1ca62e579f52c67ff58/core/docs/STAGE2_NUDGE_PROCESSING_DESIGN_20261108.md#L295-L315)). The model reply may echo the input, which is distinct from Core duplicating the original attention. | Replace the stale path with per-nudge folders. Make S15/X12 separately inventory original input bytes and output echoes across logs, run files, backups, API responses, GitHub, and `tass.md`. |
| R3-13 | **OPEN, spec correction** | F16 says an Identity restart before redeem is recovered by “owner clicks again,” but C3 permanently returns the durable SUBMITTED record and never resends ([F16 line 267](https://github.com/Arameair/Thought-Processor/blob/d21caf54c6e644bcd713f1ca62e579f52c67ff58/core/docs/STAGE2_NUDGE_PROCESSING_DESIGN_20261108.md#L260-L270)). | Correct recovery to owner Stop → successful VOID → Copy as new/new approval, or specify another bounded recovery mechanism. |
| R3-14 | **OPEN, owner scope decision** | Copy-as-new after PRE_RESTORE or UNCERTAIN intentionally permits a second semantic model call under a new nudge ID. G1/G2 are per-ID guarantees, not semantic exactly-once execution. Plain `think` also remains outside this system ([residual risks R1/R3–R5](https://github.com/Arameair/Thought-Processor/blob/d21caf54c6e644bcd713f1ca62e579f52c67ff58/core/docs/STAGE2_NUDGE_PROCESSING_DESIGN_20261108.md#L285-L291)). | State this explicitly in the owner dialog and D3/D13/D14. Require a specific warning and new approval; do not describe Copy as exactly-once recovery. |

## Blocker-only correction set

Implementation should not begin until these five items are incorporated into the design:

1. **Authorize void:** owner-bound, exact-nudge, single-use stop capability; shared Engine key alone is insufficient.
2. **Make restore publication atomic/fail-closed:** stage, validate, fence, sync, then publish; durable pending state blocks processing throughout interruption/recovery.
3. **Replace `draft_lines`:** use a verified byte-prefix boundary plus digest, or an explicit restored-ID set; reject any journal ambiguity.
4. **Fix authorization order:** one bound authorization before existing-run replay; reuse it for a new submission.
5. **Serialize Core admission:** hold one mutex across the active-run check and durable `submitted.json` creation for all nudge IDs.

The stale privacy filename, F16 recovery wording, and `attempted` label should be corrected in the same document but do not require a broader architecture.

## Focused tests required after implementation

**Test name:** G1 claim-versus-authorized-void linearization

**What it checks:** G1 at revision 3 plus R3-02: a scoped owner void and a real claim compete for the same durable object; a successful void permanently excludes model work.

**Setup:** Isolated Engine, Identity and Core fixtures on the supported local POSIX filesystem; fixture model with a server-side POST counter; two Engine processes in distinct PID namespaces where applicable; fault hooks around create/write/file-sync/directory-sync.

**Steps:** 1. Pause a valid handler after redeem and before claim. 2. Submit the exact owner-bound void. 3. Exercise both orderings and 100 barrier-controlled interleavings. 4. Kill each participant after every persistence boundary and restart. 5. Repeat with disk-full and short-write injection. 6. Attempt arbitrary-ID, wrong-entity, reused, expired and Engine-key-only voids. Return phase files and fixture counts without secrets.

**Expected result:** Exactly one durable claim object exists. If valid void wins and is reported successful, the fixture receives zero POSTs forever for that ID. If claim wins, void returns the existing run and never reports NOT_DELIVERED. Every malformed existing claim is RECOVERY_REQUIRED. Unauthorized voids create no folder/file.

**Failure case:** Any successful void followed by a fixture POST, any second claim inode, or any unbound arbitrary tombstone is a release blocker.

**Test name:** G2 interrupted restore publication matrix

**What it checks:** Every supported `restore_data()` caller keeps all restored drafts non-processable across crashes before, during and after fence publication.

**Setup:** Isolated launcher installation with a verified backup containing multiple valid drafts plus a valid final record; fixture model counter; fault hooks for staging extraction, journal validation, pending marker, fence write/sync, data publish, operation-phase update and pending clear.

**Steps:** 1. Run failed-update auto-restore, rollback `--restore-data`, rollback guard restore, restore-previous, restore-previous guard restore, and recover. 2. Kill after each boundary. 3. Run recover repeatedly, including interruption of recover itself. 4. Attempt supported startup and process every restored ID before and after recovery. 5. Verify code-only rollback creates no restore fence.

**Expected result:** Until a matching durable fence is published, all nudge processing is locked. After recovery, every ID present in the restored journal is PRE_RESTORE and returns 409; fixture POST count is zero. New post-restore IDs remain eligible. Code-only rollback does not alter fence generation.

**Failure case:** A missing/torn/stale fence, lost operation record, or interrupted recovery must never expose a restored draft as DRAFT/processable.

**Test name:** G2 journal-boundary falsification

**What it checks:** The restore fence covers the exact validated restored journal, not a newline count.

**Setup:** Isolated backups containing: normal JSONL; valid final JSON without newline; truncated JSON; invalid middle record followed by valid record; duplicate ID; oversized scanner record; and a post-fence prefix mutation.

**Steps:** 1. Restore each backup through `tass.sh`. 2. Inspect the recorded boundary/digest without exposing content. 3. Start Core and list/process restored drafts. 4. Mutate/truncate the prefix in the fixture and restart.

**Expected result:** Valid accepted journals have every restored ID covered. Every ambiguous or mutated journal locks processing and produces no Identity bound decision and zero fixture POSTs.

**Failure case:** Any valid restored draft outside the fence, or acceptance of an invalid/mismatched prefix, fails G2.

**Test name:** Bound replay authorization order

**What it checks:** R3-08: replay and new submission both require a fresh bound decision for the exact tuple without an unbound authorization exception.

**Setup:** Isolated Identity/Core/Engine fixtures with existing terminal run and no-run cases; revocable process grant.

**Steps:** 1. POST the existing run with grant on. 2. Revoke and POST again. 3. Restore grant and POST a new draft. 4. Substitute each tuple field and repeat after expiry/reuse/restart.

**Expected result:** Exact authorized replay returns existing state; revoked or substituted requests are 403 before claim; a new run redeems the same bound decision minted before C3; fixture count does not change for replay.

**Failure case:** Any unbound allow, replay after revocation, or substitution reaching claim is a blocker.

**Test name:** Different-ID Core admission race

**What it checks:** R3-09: the single-active-run rule is atomic across different nudge IDs.

**Setup:** One isolated supported Core process, two valid drafts, barrier immediately before active-run check and before submitted-file creation, fixture Engine/model counters.

**Steps:** 1. Release two process POSTs for different IDs simultaneously. 2. Repeat 100 times and with a Core crash immediately after the winning submitted file is durable. 3. Restart and repeat status reads.

**Expected result:** Exactly one `submitted.json` is created and at most one Engine/model request occurs; the loser receives 409. Restart and GET never dispatch.

**Failure case:** Two submitted folders or two Engine POSTs from one admission window violates the design.

## Readiness recommendation

Revision 3 is closer and G1's exclusion primitive should be retained. It is still **not ready for implementation** because R3-02 through R3-05, R3-08 and R3-09 include five distinct correction themes listed above (R3-03/R3-05 are one restore-publication theme). After the document is corrected, the design can proceed to implementation in isolated fixtures; all runtime tests remain OPEN until then.
