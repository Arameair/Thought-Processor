# Start Claude here — copy/paste build assignment

This message is deliberately short. It delegates the **implementation** to Claude while keeping the detailed specification and tests in GitHub. This branch contains **notes and story only**, not an implementation.

---

Please build the **first runnable Thought Processor prototype** from the owner-reviewed story **The First Spark**.

**Repository:** https://github.com/Arameair/Thought-Processor

**Read first, in order:**
1. Revised narrative: https://github.com/Arameair/Thought-Processor/blob/experiment/experience-storybook-20261024/README.md
2. No-fake-data rule: https://github.com/Arameair/Thought-Processor/blob/experiment/experience-storybook-20261024/story/TRUST-AND-EVIDENCE.md
3. Your full implementation contract: https://github.com/Arameair/Thought-Processor/blob/experiment/experience-storybook-20261024/prototype/CLAUDE_BUILD_BRIEF.md
4. Acceptance test matrix: https://github.com/Arameair/Thought-Processor/blob/experiment/experience-storybook-20261024/prototype/ACCEPTANCE_TEST_MATRIX.md
5. Milestones, expectations and stop conditions: https://github.com/Arameair/Thought-Processor/blob/experiment/experience-storybook-20261024/prototype/EXPECTATIONS_AND_GATES.md
6. Reporting template: https://github.com/Arameair/Thought-Processor/blob/experiment/experience-storybook-20261024/prototype/TEST_RUN_REPORT_TEMPLATE.md

**Required first actions:** Inspect the actual current `alpha` branch and selected real Thought Engine experiment; create **your own new experimental implementation branch from `alpha`**. Do not use the story branch as an implementation base and never change or auto-merge Alpha/Stable/Main.

**Build Gates 0–3 ONLY for the first owner trial:**
- A real, independently runnable Processor with a compact, truthful local workspace.
- A bounded **Ensure Engine Available** operation that safely attaches to the correct already-running Engine or starts the selected approved Engine **once**.
- Actual process/build/port/instance/trust state; correct distinction between started, attached, compatible, authenticated, authorized and worker-ready.
- Real inspect/logs/metrics/Copy Context/Stop controls with accurate outcomes and no secrets.
- Error handling and negative tests: duplicate Ensure, wrong build, occupied port, Engine crash, UI close/reopen, Processor restart, owned versus external process stop.
- Tests run against a **real selected Engine**, not just mocked adapters. Include exact SHA, commands, results, timings and safe diagnostic evidence.

**Do NOT fabricate any GitHub account, repo, feature entitlement, Identity grant, AI output, thought, process status, metrics or successful test.** No named model dependency, no paid fallback. Current user-owned GitHub starter contents/license/creation mechanism and publisher feature-access protocol are **undecided**; normal full-product first-opening is **not yet complete**. The real Identity grant API is missing. Show `SETUP REQUIRED` / `AUTHORIZATION INTEGRATION NOT READY` accurately instead of bypassing either requirement. The limited standalone Processor–Engine developer trial must be clearly labeled as such.

**Do NOT modify Identity/Thought-engine approved branches** or introduce a fake allow endpoint. If another project needs work, specify the exact missing contract. Do not implement Gate 4 (live Identity authorization) or Gate 5 (customer GitHub bootstrap) without owner approval and real dependencies.

**Return after Gate 3** with a working branch URL, exact SHA, one-command build/run instructions, a completed real `docs/TEST_REPORT.md`, 🟢/🟡/🔴 verdicts for every relevant test, true remaining blockers and a one-minute owner test script. **No Alpha promotion**. If a milestone can't work, return the real partial implementation and its precise failure rather than claiming success.

**Your objective is not to reproduce the fictional story verbatim. It is to deliver the smallest real system whose behavior supports the story.**

---

**Note:** The user has already approved creating a prototype and testing it on an experimental branch, not merging or distributing any protected software. No additional product or licensing decisions are implied.
