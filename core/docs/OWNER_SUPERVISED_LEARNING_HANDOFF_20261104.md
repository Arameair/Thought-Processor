# Owner handoff — finish the first supervised TASS lesson (2026-11-04)

**To:** Claude (developer/test collaborator)  
**Priority:** P0 owner acceptance, not additional architecture  
**Source:** owner screenshots of local Identity page after successful deployment, plus actual terminal/test evidence  
**Working experimental branches:** Thought-Processor PR #63 `experiment/manual-learning-gpt6-20261104`; Identity PR #35 `experiment/owner-learning-grants-gpt6-20261104`

## What is actually running

Owner machine is running **Core `76ed3b4e85c6`**, **Thought Engine `d8208409457f`**, **Identity `22b6ceb96302`**, one already-running external Nemotron. The owner updater reported `PASS: all healthy`, preserved identity and records, and kept rollback state. Existing ports: **Core 127.0.0.1:8797**, **Identity 127.0.0.1:8796**. GitHub is configured for **Arameair/TASS@live**, `lessons/` and Core has a token; the Core container can reach api.github.com. TASS PR #23 is merged and these Markdown lessons are visible in live: `lessons/claude/01-my-work-pulse.md`, `02-how-i-learn.md`, `03-building-together.md`. The notebook `00-find-and-run-a-lesson.ipynb` is **NEEDS_WORKSPACE** and must NOT be executed in this slice.

## What the screenshots prove—and do not prove

The Identity page shows **UNLOCKED TASS**, the **same entity ID `ent_bd07bc1109ef5c89f255f570`**, and still only:
`conversation`, `model.request`, `thought.process`, `pulse.run`, `action.read_entity`.

The new **Enable learning for this existing TASS identity** card is visible with a passphrase input and **Authorize lesson discovery and study** button. The page also shows the red error **“Confirm your Identity passphrase first.”** The input is empty in the screenshot. In PR #35's current page code, this exact message is **client-side validation for an empty passphrase**; the code clears the field before attempting the POST. Thus this is **not proof of a failed API request** and **not proof the user actually supplied the passphrase**. The learning grants are not shown as applied.

The **Approve one read-only lesson — Thought-Processor Issue #51 / @identity-test** card is a **different old, reference-only approval flow**. Do not direct the owner to click it to enable the Arameair/TASS lessons loop.

## Root cause already found

The persisted entity was created before `github.sync` and `lesson.learn` were present in Identity's `DefaultGrants`. Defaults apply only to new entities. Existing entity grants remain unchanged until an authorized owner action. PR #35 adds an explicit passphrase-gated owner-only `POST /v1/grants/learning` and page UI to add only those two grants, preserving the existing entity, credential and memory. **Do not edit entity.json manually; do not recreate the entity; do not disable authorization checks.**

Autonomous ON is NOT the fix for missing grants, and Autonomous OFF must retain the owner's manual Find → Study → Review → Approve workflow (PR #63). The persisted Autonomous setting was previously ON; ask owner to turn it OFF before testing manual lesson acceptance. When ON, it can auto-approve lessons that pass, and other guarded actions may be allowed. The `github.sync` grant also permits existing document synchronization and may cause repo writes; do not describe it as purely read-only.

## Immediate Claude assignment — show the owner one visible success

1. **Check current Identity owner-card UX.** It should display what permission will change, clearly explain why passphrase confirmation is requested (or support one owner-authenticated consent during unlock if secure), distinguish required input from actual server failure, and after a successful POST show `github.sync` and `lesson.learn` in the grants list. Do NOT remove an authentication step without a secure replacement. Avoid the confusing adjacent, unrelated Issue #51 approval as part of this task.
2. **Verify the running Core UI** has `Find lessons (manual)` and individual `Study (manual)` buttons with **Autonomous OFF**. Its first real call must get a concrete authorization decision. If it fails, surface the exact `403`/decision reason in the Learning section; do not claim a lesson was learned.
3. **End-to-end owner acceptance using the *existing* Nemotron:** grant two permissions via Identity; Find; show 3 Markdown lessons + notebook marked NEEDS_WORKSPACE; click Study for **My work pulse**; observe a real Engine job receipt and answer/check results; **PASSED remains pending**; owner alone clicks Approve; verify `LEARNED` and `## Learned` in local document and, after authorized sync, GitHub. No duplicate jobs or automated approval while Autonomous OFF.
4. If a defect is confirmed, fix only the required Identity/Core surfaces on a dedicated experimental branch or update the relevant PRs, write failing tests first where practical, run Go race, Docker build, launcher fail-closed and targeted UX tests. Preserve current pins/ports/model, records, secrets and rollback; do not merge to main or update owner deployment without acceptance.
5. Report **evidence**: tested commit IDs, CI URLs, actual Identity grant list, lesson status transitions, job ID, model answer excerpt, exact point of failure if any. No “PASS” for screenshot-only or simulated results.

## Non-goals for this slice

Do not launch another TASS or model, implement notebook code execution, turn learning into unrestricted autonomous code execution, weaken Issue #50 locked-prediction/Identity guards, or claim that ChatGPT ↔ TASS live two-way communication already exists. That GitHub-mediated communication bridge is a **subsequent explicit vertical slice** after one real supervised lesson works.

## Owner's success criterion

> “I can open TASS, press Find Lessons, press Study on one real lesson, see what he answered and what he got right, and choose Approve. I should not have to learn terminal details or repeatedly reconfigure things.”

**Status at handoff:** combined repair installed and healthy; new owner permission card visible; screenshot shows `Confirm your Identity passphrase first` and **no new grants yet**. **Owner acceptance of actual learning is PENDING.**
