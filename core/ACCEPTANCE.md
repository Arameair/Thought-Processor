# Owner acceptance test: TASS Go core on your PC (real Nemotron)

About 15 minutes. Write **PASS / FAIL / NOT RUN** for each step, with what you saw. Paste the output of step 8 into Thought-Processor #42.

**Before you start:**
- **Review and merge identity PR #31.** Merging it is Identity's permission. Without it, step 3 shows "no Identity policy" and nothing runs; that is also a valid check.
- The model is running, the usual way: `cd ~/tass-workspace/workspace && ./run.sh` on the branch you normally use. Then switch to the core branch:

```bash
cd ~/tass-workspace && git fetch && git checkout experiment/go-core-20261104 && cd core
```

| # | Do | Expected |
| --- | --- | --- |
| 1 | `./start.sh check` | Every line `[ok]`, or a `[note]` about the Python liaison. Nothing is started or stopped. |
| 2 | `./start.sh`. Answer **N** if asked about the Python liaison. | It ends with "TASS is running." The workspace containers are still running (`docker ps`). |
| 3 | Open http://127.0.0.1:8795 | Thought Engine, Nemotron ready and the engine pulse are green. Identity shows **valid now** and **PROTOTYPE policy**. `tass.md` appears on the right. |
| 4 | Type: **How many sections does your tass.md have?** | One cycle card with a **real Nemotron reply** (no "TEST FIXTURE"), a Spark rating and topic, and a proposed action. **Prediction LOCKED** with a time earlier than **Action result EXECUTED**. Evaluation **CORRECT or INCORRECT** (either is fine: it's honest). Every model request ID is listed. |
| 5 | Type: **hi TASS** | Most likely **NO_ACTION** with his reason. Nothing runs. |
| 6 | Turn off Wi-Fi or unplug the network. Type: **remember that the internet was off** | The GitHub chip turns red, but TASS still answers, and the memory journal gets the line with its cycle ID. `./start.sh status` shows a recent engine pulse. |
| 7 | Still offline: `./start.sh stop`, then `./start.sh`, then ask step 4 again | It starts. The earlier memory is still in the journal. A new cycle runs. |
| 8 | Reconnect and wait 30 s. Then run: <br>`tail -n 3 ~/.thought-engine/tass/cycles.jsonl ~/.tass/sync.jsonl; head -c 300 ~/.thought-engine/tass/model-requests.jsonl` | The sync shows `"ok":true` again. Paste this output; it contains no secrets. |
| 9 | *(GitHub write, optional.)* On GitHub, edit `Arameair/TASS` branch `live` → `tass.md`: change the line under `## Now`. Wait 30 s. | The right panel shows your edit. TASS uses it in his next reply. |
| 10 | *(Conflict, optional.)* `./start.sh stop`. Edit `~/.tass/tass.md` under `## Now`, and edit the same section differently on GitHub. Then `./start.sh` and wait 30 s. | A PR "TASS: which version of Now?" opens into `live`. `~/.tass/conflicts/` holds both versions. Merge or close it, and TASS follows. |
| 11 | *(Revoke.)* In identity `policies/tass.json`, set `"action.read_tass": false` through a PR, and merge it. Ask step 4 again. | The action shows **NOT_AUTHORIZED** and the evaluation **UNSCORABLE**. TASS still talks, because conversation isn't an action. |
| 12 | `./start.sh stop` | Both stop. Your Python stack is untouched. |

**Not covered here (known):**
- No evaluator model is configured; scoring is a deterministic rule.
- The policy is a prototype file, not live Identity grants.
- The engine's older routes are unauthenticated.
- The full five-Spark learning cycle (2026-10-31 design) is not built; this is its first single-Spark step.
