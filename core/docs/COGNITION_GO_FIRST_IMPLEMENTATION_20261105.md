# Cognition — Go-first engineering handoff (2026-11-05)

**LATEST OWNER ALIGNMENT — 2026-11-05:** Read [cross-loop, 20-second Go junction alignment](https://github.com/Arameair/Thought-Processor/blob/docs/loop-junction-alignment-20261105/core/docs/CROSS_LOOP_ALIGNMENT_20261105.md) and [draft coordination PR #85](https://github.com/Arameair/Thought-Processor/pull/85). **No complete circles are implemented yet; observability is intentionally being built while the loops are assembled.** The priority remains accurate Go-based Cognition records/playback/logs, but a single supervised Go information-junction vertical slice may be built/tested alongside visibility rather than waiting for every Cognition UI feature. Do not invent a second scheduler: **20-second controlling pulse only**, not 20 minutes. Markdown junction cells are bounded information descriptions, not executable authority. The old phrasing that might suggest 'complete Cognition before any junction work' is superseded by this incremental approach. No Talk/Discord or autonomous implementation authorized.

**Authoritative implementation contract for the next Cognition slice.** Read alongside [Issue #82](https://github.com/Arameair/Thought-Processor/issues/82), [PR #84](https://github.com/Arameair/Thought-Processor/pull/84), [Issue #80](https://github.com/Arameair/Thought-Processor/issues/80) and [Issue #79](https://github.com/Arameair/Thought-Processor/issues/79).

**Owner scope:** Cognition first. A Netflix-like library of **real persistent thoughts** and recorded thought-cycle playback; simple log retrieval and a safe link to supervised Development experiments. **Do not implement proactive speaking, voice/TTS, notifications, scheduled speech, arbitrary notebook execution, new model/service, or unattended updates.** No change to the owner's running deployment.

## Why Go-first

The current PR #84 page does most grouping and cross-record interpretation in `core/cmd/tass-core/cognition.html` JavaScript. Keep its shelves and visual style as a usable prototype, but move factual record selection, joining, state derivation, timeline construction, security and export formatting to **Go**. The browser should mostly render server-prepared view models, filter/scroll cards, open details, and play already-recorded events. A little vanilla JavaScript is appropriate for interaction, and the page can remain embedded with `//go:embed`; do not add a Node runtime, heavy SPA framework or separate frontend build service.

Suggested package boundaries, subject to code review:
- **Thought Engine (Go):** owns source records and any new *authenticated, bounded, read-only* retrieval of actual chronological cycle journal events (never POST for playback). Its existing `cycles.jsonl` has `cycleEntry{at, cycle_id, step, data}`, steps `received | spark | proposal | outcome | evaluation | problem`. Do not expose raw `data` as an unrestricted log endpoint. Reuse Engine's existing auth/middleware and established Core-to-Engine credentials.
- **TASS Core (Go):** owns the Cognition view/projection and same-origin authenticated owner endpoints. Read existing `Core.Status()`, `Core.BrainBlocks()`/saved graph, Engine records via the existing client, and Identity/skill/lesson receipts. Produce bounded typed `ThoughtCard`, `CycleCard`, `TimelineStep`, `EvidenceRef`, `ExportReport` structs with explicit `source`, `observed_at`, `recorded_at`, `status`, `verified` and `not_recorded` fields. Go performs allowlist-only export projection and size caps.
- **UI (embedded HTML + minimal vanilla JS):** shelves, search, filters, card inspector, Play/Pause/Step of *recorded* steps, copy/download on explicit click, Source/UNKNOWN labels. No business logic determining that a prediction was executed, no speculative joins, no action POSTs. No simulation fallback in integrated live mode. An isolated preview may use distinctly marked demo fixtures.

## Confirmed Thought Engine record model (pinned Engine 21b3d502775b3ed065ab445b88f51909500770b3)

Source: `Arameair/Thought-engine/engine.go`, `cycles.go`, `server.go` and `model.go` at this exact SHA.

**Persistent Thought EXISTS:** `Thought{thought_id, entity_id, topic, question, state, trace_ids, origin_trace_ids, notes, history, created_at, updated_at}`. `ThoughtStep{at, kind, event_id, trace_id, request_id}` and `Trace{trace_id, source_event_id, thought_id}`. The Engine already has owner-sensitive `GET /api/entities/{eid}` and `GET /api/entities/{eid}/thoughts/{tid}` for entity thought records and supporting evidence.

**Cycle is DIFFERENT:** `Cycle{cycle_id, entity_id, spark, proposal, outcome, evaluation}`. `SparkStep{event_id, thought_id, trace_id, model_request_ids}` enables a *verified* cycle → persistent thought link when `thought_id` is nonempty and that target exists for the same entity. Otherwise label the cycle **UNLINKED** and leave it on a distinct **Recent cycles** shelf; do not group by topic or identical wording. Some events produce only Sparks, some Traces, some lasting Thoughts. Do not claim all cycles result in a persistent Thought.

**Timestamp limitation:** The aggregate `Cycle` contains some event timestamps, but the complete stage timestamps exist in the append-only `cycles.jsonl` journal. A genuine step-by-step playback should read those chronological event records through an authorized Go endpoint, not invent step timestamps or timings in JS. Explicitly mark missing/duplicate/out-of-order entries and log truncation. Preserve recorded order and source line/event ordinal as needed; keep timestamps truthful when clocks differ.

**Model limitation:** `Proposal.Thought` and `ModelResult.Thought` may contain text named `model_reasoning`/internal thought output. Do not expose it as hidden private chain-of-thought or claim full introspection; show only permitted summaries, observable proposal fields, safe model response metadata and recorded evaluation. A model's claim is not a tool receipt.

**Authorization distinction:** The Engine's local `entity_id` is not itself the canonical Identity permission; the Core's service/Identity authorization remains mandatory. Do not bypass Identity by calling Engine directly from browser. Engine read endpoints can be sensitive: go through authenticated Core facade and check entity ownership/scope. `@lesson-workspace.list-lessons` remains owner-approval-gated even if read-only. `NO_ACTION` is never successful execution.

## User-facing Cognition behavior

**Main screen:** Netflix-style horizontal shelves of persistent Thoughts (active/sleeping/resolved as supported), recent cycles, real lessons, notebook cells, skills/receipts and observed outcomes. No top-level repository/service dashboard. TASS, Identity, Development and Cognition are owner-facing areas; TASS Core stays inside TASS. Nemotron appears only after drilling into Thought Engine from a relevant record.

**Thought detail:** canonical question/topic/state and history from the Engine, actual linked Traces and events, verified associated cycles; missing links labeled. Click an individual cycle to see its recorded playback.

**Playback:** Play, Pause, Step and jump-to-event operate on an immutable list of previously recorded events. Show event/request → Spark/Trace/Thought (if any) → proposal → locked prediction → Identity authorization evidence (only if correlated) → action result → evaluation. Each step must show when/where it was recorded, ID(s), status, and missing evidence. Playback NEVER invokes model, skill, lesson, notebook, Pulse or approval endpoints.

**Logs:** View selected cycle/receipt logs on the owner's machine; Copy evidence; Download sanitized Markdown/JSON. Export content is an allowlist of selected fields, bounded in records, bytes and time, carrying deployed versions and verified source IDs. Do not provide direct filesystem path parameters or raw JSONL download to browser. Full raw log is local owner-only with an explicit privacy warning and must not auto-publish to GitHub or remote ChatGPT. Use exact status labels: `OBSERVED`, `APP-REPORTED`, `PROPOSED`, `NOT RECORDED`; do not imply all Core/Engine/Identity records can always be joined by ID.

**Development bridge:** A Cognition button `Try adjustment in Development` transfers only a reference (thought/cycle ID, pinned brain revision, selected evidence IDs) to an *unexecuted proposal*, never directly modifies a live graph or authorization. Development controls candidate changes, supervised tests, A/B review and deployment through Issue #79's hardened updater. Avoid implementing general experiment execution in this first Cognition slice; a read-only context link is a valid initial bridge, clearly labeled.

## API recommendation (names illustrative, use existing route conventions)

- `GET /api/cognition/library?limit=...`: Core-generated compact cards, counts and source freshness.
- `GET /api/cognition/thoughts/{id}`: canonical Thought details and verified links; bounded to current authorized entity.
- `GET /api/cognition/cycles/{id}/timeline`: ordered Go-projected stage records, with `recorded_at` and source/ID; no private raw model context.
- `GET /api/cognition/cycles/{id}/export?format=md|json`: safe explicit user download, Content-Disposition with safe filename, no-store and size caps. A separate lower-privilege copy endpoint may be unnecessary because browser can copy the same sanitized response.

If the Engine's authenticated API cannot supply exact journal entries, implement a scoped read-only Engine endpoint using its existing key/auth, with correct streaming/rotation/size limit and tests. Do **not** mount all Engine/Identity private data volumes into Core just to bypass service boundaries, or invent log correlations without evidence. Avoid expensive full log rescans every page refresh: bounded selection, paging, lightweight indexes or known IDs if justified.

## Implementation order and tests

1. **First:** small Go testable `ThoughtCard` projection from actual Thought/Trace/receipt data plus separate cycle cards. Include stable identity, deterministic sorting, empty and unlinked states.
2. **Second:** Go timeline projection from actual journal entries for **one** real cycle, with missing/out-of-order/duplicate/rotation handling and no execution side effects.
3. **Third:** sanitized, bounded Go evidence export and browser Copy; test fake secrets, injected HTML, unauthorized 401/403, path traversal IDs, huge log/partial JSONL, Identity LOCKED/denied, no raw private prompts, no cache.
4. **Fourth:** simple JS playback controls, filter/shelf/detail, keyboard/accessibility and no action requests. Preserve current PR #84 visual prototype and prior /dev brain revisions.
5. **Fifth:** Cognition → Development read-only experiment proposal link, then a **controlled, owner-approved real-host acceptance**. Test cycle source IDs and actual Nemotron/Identity receipts. Prototype fixture PASS ≠ real deployment PASS.

From clean checked-out candidate on WSL (not on running production stack): `cd core && gofmt -w <changed-go-files> && go vet ./... && go test ./... && go test -race ./...`; use existing isolated Go/Playwright harness for browser checks. Cite exact commits, OS, commands, output; record `PASS / PARTIAL / FAIL / NOT RUN` honestly. An old successful CI run for PR #84 does not validate new backend work.

**No deployment:** keep `experiment/tass-unified-integration-20261105` as integration target. Build on a focused branch/PR; merge only with review, then install only through a reviewed owner-approved `tass.sh` update with verified backups and Restore Previous. No live restarts, no Alpha/main overwrite, no new model, no speech work.

## Owner acceptance definition

The owner opens Cognition, sees **persistent Thoughts** rather than disguised cycle cards, selects one actual thought with verified linked evidence, plays a real recorded cycle step-by-step without performing actions, copies/downloads a safe report, and opens its corresponding proposed Development experiment without modifying live TASS. Every factual link is supported by an ID/receipt; unknowns are visible.

**Current state:** PR #84 is a working visual/fixture prototype with limited live read-only source, not this completed Go-first feature. Prior `core-tests` CI was green for that branch; owner-machine acceptance and Go-first timeline integration remain **NOT DONE**.
