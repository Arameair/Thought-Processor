# Supervised lesson — what was tested, what failed, what was fixed (2026-11-04, Claude)

Follows `OWNER_SUPERVISED_LEARNING_HANDOFF_20261104.md`. Everything below was observed, not assumed.

## Test setup (sandbox — be clear about what is real)

| Piece | What ran |
| --- | --- |
| Identity | real `tass-identity` binary, PR #35 head `22b6ceb`, then the fix `91e1521` |
| Thought Engine | real binary at the owner's pin `d820840`, `-worker nemotron` |
| Core | real binary, PR #63 head `65074e8`, then this branch |
| Lessons | the real files from `Arameair/TASS@live` `d9ec62a` (`lessons/claude/00..03`) |
| GitHub | **test fixture** `core/test/fake_github.py` seeded with those files (the owner's live repo was never written) |
| Model | **test fixture** `core/test/fixture-model/fake_llama.py` — this sandbox can't download Nemotron. **The real-Nemotron study is the owner's acceptance run.** |
| Browser | headless Google Chrome via Playwright, clicking the real pages (`core/test/browser/*.py`) |

The entity was created, then its grants were cut back to the original five to reproduce the owner's pre-learning TASS.

## 1. "Confirm your Identity passphrase first." — verified

Real browser against PR #35 `22b6ceb`:

| Action | Message | Network |
| --- | --- | --- |
| click with empty field | `Confirm your Identity passphrase first.` | **no request sent** |
| wrong passphrase | `wrong passphrase` | `POST /v1/grants/learning -> 403` |
| right passphrase | (none — card silently disappears) | `POST /v1/grants/learning -> 200`, grants now include `github.sync`, `lesson.learn` |

Audit line written: `grants.enable_learning … owner passphrase confirmed`. **The server works; the screenshot was the empty-field check.** The card gave no success feedback and showed errors at the bottom of the page.

Fixed in Identity `91e1521` (branch `experiment/claude-learning-card-ux-20261104`): result shown on the card ("Nothing was sent to Identity" / "Identity refused it: … Nothing was changed." / green success line with the granted names), Enter submits, the card says `github.sync` **can write his document to the repository**, and the Issue #51 `@identity-test` card is labelled *Separate developer test (not TASS's lessons)*. Locked now returns "Identity is locked: unlock it…". Re-tested: empty → no request; wrong → 403 shown on card; right → 200 + success line.

## 2. Core defect: Find lessons never asked for lessons — fixed

Real browser against PR #63 `65074e8`, Autonomous OFF: clicking **Find lessons (manual)** sent `POST /api/pulse -> 409`.
Cause: the button is inside `<div class="ctl">`, and the page later bound *every* `.ctl button` to the pulse handler, replacing Find's click handler. The Go tests passed because they call the API, not the page.

Fix: pulse binding is `.ctl button[data-a]`; `cmd/tass-core/page_test.go` fails on the old selector. Also: `refresh()` no longer overwrites the result of the owner's last click every second; errors show the HTTP status and Identity's reason; the lesson shows "Studying now (job …)", "Every check was right. Waiting for you", what he *claims* he learned, and his full answer.

## 3. End-to-end in the browser after the fixes (Autonomous OFF)

1. Find before grants → `POST /api/lessons/discover -> 409`: *Identity denied github.sync: Identity dec_b107eae88b60ee78: the entity has no grant for github.sync*.
2. Enable learning on Identity (passphrase) → 200, 7 grants.
3. Find → `200`, *Found 4 lesson(s)*: `00-find-and-run-a-lesson.ipynb` **NEEDS_WORKSPACE** (not run), `01-my-work-pulse` NEW, `02-how-i-learn` NEW, `03-building-together` NEW.
4. Study *My work pulse* → `200`, job `lesson_dc183473b7_1`. Engine: `pulse_000001_204840 WORKING START: lesson_dc183473b7_1`, authorization `Identity dec_fa353a9e2734e0cd: pulse.run allowed`; job QUEUED → RUNNING → DONE; **1** model request for it.
5. ~18 s later: **PASSED 3/3**, held — *Waiting for your approval*. Answers shown (fixture): `20 seconds`, `I say CONTINUE`, `I am BLOCKED`.
6. Approve → `200` → **LEARNED**; local `tass.md` gained `## Learned` / `- My work pulse: … (lesson lessons/claude/01-my-work-pulse.md, 3/3, approved 2026-11-04)`; next sync: `PUT …/contents/tass.md` and the GitHub fixture has the same line.
7. `settings.json` stayed `autonomous:false` throughout; no auto-approval.

## 4. Autonomous ON

Toggled on in the page: `02` attempt 1 wrong on purpose (fixture) → RETRY → attempt 2 3/3 → auto-approved; `03` 3/3 → auto-approved. Event text now says *auto-approved* instead of *you approved it*. Toggled back off.

## 5. Locked Identity

Lock → Find: `409 Identity denied github.sync: Identity dec_efebd78fd3efdb56: LOCKED…` — now followed by "unlock TASS on Identity's page" instead of "enable learning".

## Tests

`go vet` + `go test -race ./...` pass in both repos; `test/test_launcher_regression.py` 6 passed. Docker images could **not** be built in this sandbox (Docker Hub unreachable); CI builds the core image.

## Not done / not claimed

- No real Nemotron answer yet — that is the owner's run.
- No notebook execution; Issue #50 guards and Identity checks unchanged.
- GitHub-mediated ChatGPT ↔ Claude ↔ TASS communication does not exist yet (next slice).
