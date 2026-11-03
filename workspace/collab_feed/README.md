# Optional private Workshop collaboration feed

The user wants ChatGPT and TASS to collaborate without repeatedly pasting terminal output. This small observer publishes **new, bounded Workshop activity metadata** to an existing issue in the owner's **private** `Arameair/TASS` repository using Identity Liaison.

**Not a hidden background sync:** it is disabled unless Dustin starts the `collaboration` Compose profile. It does not run under the normal `./run.sh` workflow. It reads the existing `data/timeline.json` through a read-only mount, writes a request to the existing `outbox/` (which the Identity Liaison consumes), and stores a cursor in `collab_state/`. It has **no GitHub token, network, or Docker socket**, and it cannot execute notebook code or approve a proposal.

The configured coordination thread is **[private TASS issue #19](https://github.com/Arameair/TASS/issues/19)**. The Identity Liaison is already configured with the TASS repository in its direct-comment allowlist; this does not grant new repo permissions.

## Explicit owner activation (WSL; only after reviewing the experimental branch)

```bash
cd ~/tass-workspace/workspace
mkdir -p collab_state
chmod 700 collab_state
docker compose --profile collaboration up -d --build --no-deps collab-feed
docker compose logs --tail 20 collab-feed
```

The observer's **first** successful pulse establishes the starting event ID. Existing conversations are deliberately not exported. Send a **new** harmless request to TASS after activation, and look for a new comment in issue #19. The observer submits to the outbox; a separate Identity Liaison must post the GitHub comment. An empty or delayed comment does not prove the observer failed; inspect its logs and the Liaison status.

**To stop only the sharing service:**
```bash
docker compose --profile collaboration stop collab-feed
```

By default, only event IDs, event types, tool/ability labels, and reported execution outcomes are shared. **User and TASS message text is withheld.** Internal model thoughts, proposed code, notebook stdout/images, memories, file contents, Jupyter tokens, and credentials are never serialized by the observer. A brief result receipt can tell ChatGPT a cell ran and whether the executor reported OK, but not its full output.

**Optional broader sharing:** To let ChatGPT see brief user requests and TASS replies, explicitly set `TASS_COLLAB_INCLUDE_TEXT=1` before creating the Compose service. This is riskier even in a private repository; redaction is best-effort and cannot recognize every private datum. Never use it for secret-bearing conversations. Stop/recreate the service to change the option. Keep it off unless necessary.

## Verification

1. Ensure the Workshop at port 8790 and the existing Identity Liaison are running. Do not restart the whole stack solely to test this observer.
2. Opt in with the above command; verify the observer logged its startup.
3. Run a new harmless library test in the existing Workshop.
4. Verify the private GitHub thread shows the observed book and result event. It should **not** contain a thought, code body, token, memory, or raw stdout.
5. Stop the sharing service and run another test. There must be **no new collaboration receipt** while stopped.

Only the owner decides whether to deploy and use this optional feed. It is not a substitute for Identity authorization: the Liaison's existing outbox trust model deserves a separate security review. It writes comments, never code or direct commits.
