# TASS Notebook Laboratory — Lesson 01: Identity verification

**Branch:** experiment/atrium-notebook-lab-20261104, stacked on the still-unmerged Atrium PR #52. The lesson uses one fixed notebook cell. Nothing here restarts or modifies the running owner stack.

## Existing components inventoried from actual source

| Service | Inspected code and location | What exists; relevant limit |
| --- | --- | --- |
| Thought-Processor Python Workshop | workspace/tass_workspace/server.py, kernel.py, thinker.py, mind.py on the PR #52 parent da40d1087b1d3710d03168645ded8a7ac4d70856 | The Workshop starts JupyterLab on internal port 8888, attaches SharedKernel to TASS-live.ipynb and executes owner-approved code using jupyter_client. A journal at journal/tass-journal.ipynb records code and output. The existing model can OPEN notebooks and propose RUN of numbered cells, but Workshop runs in a shared, mutable kernel; it is NOT an isolated security authority. |
| Workspace Docker | workspace/compose.yaml and Dockerfile | Host workspace/shared is bind-mounted as /workspace/shared with live notebook and journal; workspace/data is the controls/chat/Jupyter log volume; workspace/mirror contains the read-only synced library. Atrium is a separate service on port 8791 with no Go toolchain or approved source checkouts. Do not mount more secrets or runtime folders into it. |
| Claude Nemotron playground | Arameair/nemotronTASS, experiment/tass-alpha-20261102 at 4df21110ee7f8c1edd07216d1c3cbac0c7018330 | notebooks/lab03_chat_and_cells.ipynb and chat_kit.py present a human-controlled Insert & Run/Insert Only/No thanks flow; tass_alpha.ipynb and tass_core.py provide a local browser companion/heartbeat experiment; run.sh starts its own Jupyter and model. Human-approved generated code is not yet a version-locked, limited TASS executor. |
| Discord prototype | Arameair/TASS PR #1 head f64edb71c411fc86225605a62bcccaf3d0a99e57 | Experimental discord.go accepts filtered owner/channel !tass messages and returns replies; the PR explicitly reports live Discord NOT RUN. No notebook authorization, no executed-cell receipt, and no connection to this lab. Discord remains conversation transport only. |
| Existing Atrium test cards | Thought-Processor PR #52 at da40d1087b1d3710d03168645ded8a7ac4d70856 | 20-second timer, explicit owner approval, durable single-job claim, no retry across restart, PASS/FAIL receipts and bounded/redacted evidence. The standard Atrium container defaults to execution disabled. |

## Smallest integration implemented

- One versioned lesson: workspace/atrium/lessons/identity_test.ipynb, Git commit 25fe25bb57b9306e1e7b312e1d28e75152f5d28e. This has a markdown teacher instruction and exactly one code cell with ID identity-go-packages.
- Notebook Git blob hash 72d2a86db4209b18afe34b26a130ddce00676f07. Cell source SHA-256 822f3b33168b95887c4b0b66fdbb783e8659850b67f8e3f1213a4113fe840d65. These exact values, the clean Git checkout HEAD and expected remote origin are verified before any kernel process starts.
- The cell executes only the already approved Identity Go authority/pairing/entities package checks on checkout b0c7267b47f6111c5024c9bba8a4135e2357b432, printing the real exit code and bounded package output.
- One additional fixed card: NOTEBOOK-IDENTITY-LAB. The existing PR #52 controller schedules and records it; legacy two-card ledgers gain the new PROPOSED card through a strictly additive migration, preserving old receipts and statuses.
- notebook_lab.py maps this card to the fixed notebook_cell_worker.py. The worker imports the pinned existing Workshop SharedKernel implementation and creates a PRIVATE Jupyter kernel in a temporary directory; it never attaches to the owner's live Jupyter session, never exposes a general Jupyter HTTP endpoint and launches no second long-running service.
- Parent subprocess cancellation, one-job dispatch, pause/resume/step, approval, persistence, bounded/redacted receipt, digest and restart are reused from PR #52. This is not a second scheduler.
- The normal Atrium Docker runtime remains disabled and has no Jupyter toolchain or cloned test targets. Go/Python execution needs an independently isolated, non-root disposable runner: fixed commands alone do not isolate filesystem or network, so do not enable these flags inside the current Atrium container.

## Teaching contract and actual evidence

| Phase | What GPT-6 implemented | What TASS/Nemotron actually performed |
| --- | --- | --- |
| Teach | Lesson markdown and lesson_prompt() present one objective, exact cell name and expected observable Go result. | NOT RUN: real Nemotron has not read the lesson. |
| Select/predict | Strict JSON parser accepts only @identity-test, cell identity-go-packages and a bounded textual expectation. record_model_selection() stores a separate non-authorizing, attributed proposal before owner approval. Invalid names, cells, command fields and mismatched expectations fail closed. | CI used MODEL_FIXTURE, not actual TASS. |
| Request/approve | A notebook card requires a recorded model selection, then a separate owner approval and enabled isolated runner. All three conditions are rechecked. The model cannot grant access to itself. | CI simulates the model actor and explicit owner attestation. No live Identity grant. |
| Execute | Disposable CI invokes exact cell via actual Jupyter client and the existing SharedKernel code, executing real pinned Go tests. | Real Jupyter and Go executed in CI, but TASS did not operate them. |
| Verify/explain | Durable receipt has actual source SHA, cell hash, notebook hash, timestamps, exit status, output digest and bounded excerpt. A separate fixture reflection explains that only selected packages were verified. | Model interpretation NOT RUN; fixture text explicitly labeled. |
| Propose skill | propose_skill_candidate() creates JSON for owner review only after a matching evidence-backed PASS and valid selection/reflection. Candidate includes @identity-test reference, exact notebook revision/hash/cell, required capability test.identity.packages, immutable receipt ID and output digest. | No live model learning, no approved skill installed, no memory/fine-tuning. |

The @identity-test lookup is exactly that: a name-to-versioned-candidate lookup. resolve_skill() rejects @TASS, @ThoughtEngine, casing variations and all unknown skill names. Lookup returns execution_enabled_by_lookup=false. It never grants test.identity.packages or skips owner/Identity approval.

A future request such as "TASS, use @identity-test to check Identity and explain the results" must resolve name, pin lesson+cell, get a real TASS proposal and prediction, enforce live Identity authorization, run through the restricted executor, return real evidence, and ask TASS to interpret it. The first four operations are only partly implemented; no actual Nemotron call or live authorization path exists in this experiment.

## Automated tests and evidence

The workflow .github/workflows/atrium-notebook-teaching.yml checks out the two exact repositories/revisions in a disposable GitHub Actions runner with persisted credentials disabled. It executes a real private Jupyter kernel and real Go test. It simulates selection and reflection (MODEL_FIXTURE), verifies one job, one real receipt and no repeat after coordinator restart. It uploads JSON artifact atrium-identity-notebook-real-kernel-evidence with execution receipt and proposed skill, explicitly tagged nemotron_model_selection=NOT_RUN, nemotron_reflection=NOT_RUN.

The existing workspace-tests and PR #52 fixed Go-card CI remain intact. New unit tests cover strict skill names, untrusted command rejection, pinned notebook/cell SHA mismatch, default-disabled execution, missing model selection, immutable model-proposal record, additive ledger migration, corrupt/unapproved evidence denial and skill candidate refusing non-PASS or mismatched receipts.

**State matrix:**

| Check | Status |
| --- | --- |
| Existing Jupyter/Notebook/Discord source inventory | DONE (code inspected; no owner-runtime verification) |
| PR #52 scheduler/approval/receipt reuse | IMPLEMENTED; automated tests |
| One real pinned Jupyter cell running approved Identity packages | PASS on isolated CI runner |
| Two-card migration / idempotent restart | PASS automated |
| Unknown skill, wrong hash, command and permission rejection | PASS automated |
| Full teacher-to-model-to-cell-to-model loop | FIXTURE ONLY, not live Nemotron |
| TASS learns, repeats and independently interprets | NOT RUN |
| Live Identity permission authority, approved reusable skill | NOT IMPLEMENTED / BLOCKED |
| Owner Jupyter/Discord/Nemotron integration or deployment | NOT RUN |

## Owner acceptance: one safe first action

Review the stacked experimental PR and its GitHub Actions execution artifact. Verify the normal Atrium container remains execution DISABLED and the new card says AWAITING MODEL CELL SELECTION. **Do not merge, deploy, launch the Workshop, run Docker commands, mount checkouts or set test-execution flags** for this review. A later, separately approved action will be needed to connect a real model and externally isolated authorization boundary.
