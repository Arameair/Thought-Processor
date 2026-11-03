# Atrium Build Room — one real TASS collaboration flow

**Experimental, not merged, not owner-accepted.** Open `http://127.0.0.1:8791/build` after deploying the experimental Atrium container. The old visual map remains at `http://127.0.0.1:8791/`.

## What the page actually does

1. A human writes a task and clicks **Ask TASS**. It sends a user message to the **real** Workshop `/api/message` through Atrium's existing restricted bridge. No simulated TASS text.
2. The page displays actual Workshop conversation, proposals, and executor result events. It polls `/bridge/state` as a fallback if SSE disconnects.
3. If TASS proposes Python, the page links to **Workshop** at port 8790 for the owner to inspect and decide. Atrium cannot approve or execute. The Build Room **disables new build requests when Workshop Full Auto is enabled**; the owner must turn it off in Workshop first.
4. The **Real abilities** panel reads actual mirrored library books plus the sanitized Workshop audit. `Executor reported OK` does **not** prove all side effects without further inspection.
5. The human can explicitly share a **previewed request** in the right-hand panel. This is NOT an automatic full-conversation upload. Sharing queues one bounded task (max 1,200 characters) in a dedicated local `collab_requests/` directory. No GitHub token goes into Atrium; the opt-in `collab-feed` service forwards it through the existing Identity Liaison to private `Arameair/TASS#19`.
6. The owner can then ask ChatGPT in this conversation to inspect the latest private issue comment. ChatGPT cannot continuously watch localhost and will not independently know a new brief appeared until the issue is read. The optional feed can separately queue filtered execution metadata. **GitHub comments are not a live API connection to ChatGPT.**

## First owner test — minimum change, no stack restart

From a clean experimental checkout with the branch from PR #38:

```bash
cd ~/tass-workspace/workspace
mkdir -p collab_requests collab_state
chmod 700 collab_requests collab_state
docker compose up -d --no-deps --build atrium
curl --noproxy '*' -fsS http://127.0.0.1:8791/bridge/diagnostics
```

This rebuilds **only Atrium**; it does not restart the model, Liaison or Workshop. If port 8791 is already used by a standalone Atrium, do not kill anything without identifying the owner. The **Build Room** link is visible in Atrium; direct URL: `http://127.0.0.1:8791/build`.

To test GitHub sharing (explicitly opt in; this does **not** grant extra Identity permission):

```bash
docker compose --profile collaboration up -d --no-deps --build collab-feed
docker compose --profile collaboration logs --tail 20 collab-feed
```

By default this shares bounded event metadata, not conversation content. The only task text published is the exact user-approved brief from the **Share** button. GitHub request delivery is asynchronous via the existing Liaison, and a queued response is **not** proof that a comment reached GitHub. Check private issue [TASS #19](https://github.com/Arameair/TASS/issues/19).

To stop sharing without stopping TASS:

```bash
docker compose --profile collaboration stop collab-feed
```

## Acceptance checks

- [ ] A distinctive message from Build Room appears in the original Workshop timeline, and a real TASS reply appears in both.
- [ ] No fictitious response appears when the model is offline.
- [ ] TASS proposes one small action; Build Room shows the pending state but exposes no execution endpoint.
- [ ] Owner approves in Workshop and Build Room displays the real executor outcome.
- [ ] The library evidence list reflects real audited cells, not placeholder statuses.
- [ ] Full Auto ON disables Build Room's new task submission.
- [ ] With sharing disabled, no task is automatically published. With the collaboration observer enabled and one approved brief shared, a **new** comment appears in the private GitHub issue. There are no secrets, full conversation logs, code bodies, or thoughts.
- [ ] Stopping collab-feed stops publishing, without disrupting TASS.

## Honest limits

- Existing Workshop authentication and authorization gaps remain. Keep experimental ports bound to localhost.
- A per-click consent checkbox and best-effort secret-pattern rejection are **not** a comprehensive privacy DLP system. Check text before sharing even to a private repo.
- The model can misunderstand a request or propose the wrong operation. The owner must review the actual Python in Workshop and inspect the real result.
- No agent automatically opens Windows browser windows, merges PRs, or deploys code.
