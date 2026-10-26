# Thought Processor — diagnostic shell (experiment)

A small, independently runnable Processor. It tells you the truth about itself, looks (read-only) at the Identity and Thought Engine addresses you give it, and **refuses** to start, attach to, stop or send observations to an Engine, because Identity cannot approve those yet. See the [dependency request](../docs/contracts/IDENTITY_ENGINE_LIFECYCLE_APPROVAL_REQUEST.md) and the [experiment report](../docs/experiments/PROCESSOR_FIRST_PROTOTYPE_2026-10-24.md).

It has **no process runner** — it cannot start or kill anything except itself.

## Run (tested on Linux x86-64, Go 1.24)

```bash
git clone -b experiment/processor-first-prototype-20261024 https://github.com/Arameair/Thought-Processor
cd Thought-Processor/processor
go build -o thought-processor .          # build from the checkout so the commit is embedded
./thought-processor -data ~/tp-trial -port 7860
```

It prints a one-time link like `http://127.0.0.1:7860/#code=…` — open it in any browser. The code changes every run and is also saved in `<profile>/runtime/operator-code`.

Stop: **Ctrl+C**, or **Stop Processor** at the bottom of the page. Only Processor stops.

| Flag | Default | Meaning |
| --- | --- | --- |
| `-data` | `$THOUGHT_PROCESSOR_DATA`, then `~/.thought-processor/default` | Isolated profile folder (lock, run record, saved addresses, log) |
| `-port` | `7860` | Loopback port; `0` picks a free one |

Delete the profile folder to reset. Nothing is written anywhere else.

## Tests

```bash
go test ./...                                    # L1 unit + L2 labeled fixtures
python3 -I acceptance/live_acceptance.py --processor ./thought-processor \
  [--identity <identity binary> --engine <thought-engine binary>] --out /tmp/tp-evidence   # L3 live, needs Playwright Chromium
```

The live script starts Identity/Engine itself as **external** programs (playing the owner) — Processor never starts them.
