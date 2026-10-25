# Design ideas and approaches worth looking at

**Status:** READING LIST + IDEAS. Nothing here is decided or implemented. Each entry says what the idea is, why it matters for *our* system, and what to borrow, plus anything to watch out for.
**Date:** 2026-10-25 · Companion notes: [Authority to install the Thought Engine](INSTALL_AUTHORITY_DISCUSSION_2026-10-25.md) · [The actual thought process](THOUGHT_PROCESS_DISCUSSION_2026-10-25.md)

The names below are real, established projects or patterns, so you can search for them directly. You don't need to adopt any of them whole; most are worth reading for one good idea.

---

## Part A — Processor: authority to install (and later start) the Engine

### A1. The Update Framework (TUF) — *the* reference for safe installs and updates
- **What:** A widely used framework for securing software update systems. Signed metadata lists exactly which files (by hash) are approved, with expiry dates, and separate keys for separate jobs. It's designed so a compromised server or mirror can't push a bad build. There is a Go implementation (`go-tuf`).
- **Why us:** "Install only the approved Engine build" is exactly TUF's problem.
- **Borrow:** the idea of a signed **targets list** (feature → approved file + SHA-256 + expires). Even if we don't use TUF itself, copy its protections against **rollback** (being handed an older approved build) and **freeze** (being shown a stale list forever).
- **Watch out:** full TUF has four roles and key rotation. That's more than v0 needs; read the "threats" section first.

### A2. Simple file signing: minisign / signify, Sigstore (cosign)
- **What:** minisign/signify sign a file with an Ed25519 key and produce a small `.sig`. Sigstore/cosign signs release artifacts and records signatures in a public log.
- **Why us:** the publisher (you) signs each Engine release, and Identity/Processor verify with a public key built into the binary — same pattern as the built-in GitHub App Client ID.
- **Borrow:** Go's standard library already has `crypto/ed25519`, so verifying needs no extra dependencies. The minisign format is a good model for "what a signature file looks like."

### A3. GitHub's own building blocks
- **Releases on a private repo:** assets download through the API with the person's own sign-in. Under "repo access = grant" (option A1 in the install notes), that download *is* the access check.
- **Teams:** grant/revoke the Engine for a group of people at once instead of per collaborator.
- **Artifact attestations / SLSA provenance:** GitHub Actions can attest *which workflow and commit built a binary* (`gh attestation verify`). That answers "was this Engine really built from the approved commit?" Look at **SLSA** levels for the vocabulary.

### A4. Package manager manifests: winget, Scoop, Homebrew
- **What:** each version of each app is a small manifest: download URL + SHA-256 (+ silent install switches).
- **Why us:** a Processor "feature manifest" could be this simple. The manifest is the thing Identity signs.
- **Borrow:** the winget manifest shape (`PackageIdentifier`, `PackageVersion`, `InstallerUrl`, `InstallerSha256`) is a good starting point, and it's Windows-native.

### A5. Narrow, signed permission tokens: PASETO, macaroons, Biscuit
- **What:** formats for "this holder may do *this one thing*" tokens. **PASETO v4.public** is an Ed25519-signed token with standard fields (issuer, audience, expiry) and fewer foot-guns than JWT. **Macaroons** and **Biscuit** let a token be *narrowed* by adding caveats ("only for install", "only this digest", "only until 10:05").
- **Why us:** an Identity install decision is exactly such a token: audience = this Processor, action = install, feature = thought-engine, digest = …, expires in minutes, single use.
- **Borrow:** at minimum the fields and rules (audience, expiry, unique id for replay protection). PASETO v4.public is small enough to implement directly with `crypto/ed25519`.
- **Watch out:** if you use JWT instead, pin the algorithm. The classic JWT bug is accepting `alg: none` or the wrong algorithm.

### A6. Who-may-do-what, written as a policy: Cedar, Open Policy Agent (OPA)
- **What:** policy languages that answer *"may principal P do action A on resource R in context C?"* Cedar (from AWS) is especially readable.
- **Why us:** Identity's decisions are exactly that question, e.g. *person Dustin, action install, resource thought-engine, context: GitHub repo access confirmed 2 min ago.*
- **Borrow:** the **principal / action / resource / context** shape for decision records and logs, even if every rule is hard-coded at first. It keeps install, start, attach, stop and ingest cleanly separate.

### A7. Pairing two local programs: Bluetooth numeric comparison, magic-wormhole (SPAKE2)
- **What:** ways for two programs to trust each other using a short code a human sees. **magic-wormhole** uses a password-authenticated key exchange (SPAKE2), so a short code safely becomes a strong shared secret. Bluetooth/Signal show both sides the same number and ask "do these match?"
- **Why us:** Identity ↔ Processor pairing (option B1).
- **Borrow:** simplest honest version: Processor sends its public key plus the pairing code shown by Identity; both windows then show the same short **fingerprint** for you to confirm. After that, every decision is signed.

### A8. A stronger local channel than "a port on 127.0.0.1"
- **What:** Windows **named pipes** and Unix domain sockets can restrict access to the same OS user, which a loopback TCP port cannot.
- **Why us:** today any program on the machine can call the loopback APIs. Worth knowing for later hardening, not needed for v0.

### A9. Running and supervising the Engine (for the later *start* step)
- **What:** Windows **Job Objects** (child processes die with their parent and can be resource-limited), the Windows Service Control Manager, and the Go library `kardianos/service`.
- **Why us:** once Processor may start the Engine (option D2), it must own exactly the process it started. The existing contract doc warns that bare PIDs are not proof.
- **Borrow:** Job Objects on Windows for "the process I started"; record executable path, digest and start time alongside the PID.

### A10. Tamper-evident records: hash-chained audit log
- **What:** each log entry includes the hash of the previous one (the idea behind Certificate Transparency and git itself). Editing or removing an old entry breaks the chain.
- **Why us:** the install record ("installed build X under decision Y") should be hard to quietly alter.
- **Borrow:** add a `prev_hash` field to install-record lines; Copy Context can show "chain intact".

### A11. Offline license files (how desktop software does entitlements)
- **What:** commercial apps ship a signed license file `{who, product, features, expires}` that the app checks offline with a built-in public key.
- **Why us:** this is option A2 ("publisher-signed grant") in the install notes. It's a well-trodden path, so it's lower-risk than inventing something.

### Two ways the Processor pieces could fit together

| | **Path 1 — GitHub-native v0** | **Path 2 — signed-everything** |
| --- | --- | --- |
| Who may have it | Read access to `Arameair/Thought-engine` (A3) | Publisher-signed grant / license file (A11) |
| What's approved | A GitHub Release + its SHA-256 | A signed targets list (A1/A2/A4) |
| Identity → Processor | Pairing (A7) + Ed25519-signed decision (A5) | same |
| Build proof | Release on the repo | + artifact attestation (A3) |
| Effort | Small | Medium |
| Weakness | "Can read the code" = "may install" | You need a signing step for each grant and release |

*Natural order:* Path 1 now, with the decision and manifest shapes chosen so Path 2 slots in without redesign.

---

## Part B — the thought process itself (for after the Processor step)

### B1. Generative Agents (Park et al., 2023 — "Smallville")
- **What:** a well-known research paper where simulated characters keep a **memory stream** of observations. Each observation gets an **importance score 1–10** from the model. Retrieval ranks by **recency × importance × relevance**, and **reflections** (higher-level thoughts) are triggered when recent importance adds up past a threshold.
- **Why us:** it's strikingly close to Spark (importance), Trace (kept evidence) and Thought (reflection). Thought-engine v1 already rates 0–10.
- **Borrow:** the recency/importance/relevance retrieval formula, and "reflect when accumulated importance crosses a threshold" as an alternative to "3 traces on one topic."
- **Watch out:** they merged memory and thinking; our design deliberately keeps them separate.

### B2. Blackboard architecture (classic AI, e.g. Hearsay-II)
- **What:** a shared workspace where independent specialists post and revise hypotheses at different levels, and a controller decides which specialist acts next.
- **Why us:** it maps cleanly onto Sparks/Traces/Thoughts as levels of a board, and onto *directed waves* as the controller choosing what evidence to look at next.
- **Borrow:** the levels idea and an explicit, logged "what do we work on next" controller instead of a timer heartbeat.

### B3. ReAct and Reflexion (LLM reasoning loops)
- **What:** **ReAct** interleaves *reason → act (e.g. search) → observe → reason*. **Reflexion** adds the model critiquing its own past attempt.
- **Why us:** a directed wave is a ReAct loop whose only allowed action is "fetch this kind of evidence."
- **Borrow:** the strict loop shape with a step limit. **Watch out:** never feed the model's own reasoning back as if it were an observation (already a rule in the waves theory).

### B4. Event sourcing
- **What:** store every input event immutably; everything else (traces, thoughts) is derived and can be **rebuilt by replaying** the events.
- **Why us:** it makes the replay experiment in the thought-process notes trivial, and lets two loop designs run on exactly the same input.
- **Borrow:** treat observations as the only source of truth, and thoughts as rebuildable views.

### B5. Complex event processing (CEP) — rules before the model
- **What:** streaming systems (Esper, Flink CEP) detect patterns in time windows ("3 of these within 5 minutes," "A then B without C").
- **Why us:** a cheap rules layer that drops noise and spots obvious patterns before spending a model call.
- **Borrow:** windowed rules for deduplication, bursts and "A followed by B."

### B6. Surprise as the trigger (prediction-error / predictive-processing idea)
- **What:** attention goes to what *differs from what was expected*, not to everything.
- **Why us:** a Spark could fire on **surprise** (an event unlike recent ones, or one that contradicts an open thought) rather than on a generic importance score.
- **Borrow:** compare each observation to a short running summary/expectation; spark on deviation.

### B7. Belief–Desire–Intention (BDI) agents
- **What:** a classic agent vocabulary: **beliefs** (what it thinks is true), **desires** (goals), **intentions** (what it's committed to pursuing).
- **Why us:** useful for Q1 in the thought-process notes ("what is a thought *for*?"). A thought might be a belief being tested, or a question tied to a goal.

### B8. Grouping by meaning: embeddings + vector search (Qdrant), links in a graph (Neo4j)
- **What:** embeddings turn text into vectors so "near in meaning" can be searched; a graph stores which evidence supports which thought about which entity.
- **Why us:** a sturdier alternative to free-text topic labels (Q4). You've built Neo4j/Qdrant pipelines before, so this is familiar ground.
- **Borrow:** later, for grouping and for "find evidence related to this thought." Not needed for the first loop if "match against open thoughts" works.

### B9. Making small models answer in a fixed shape: structured outputs
- **What:** hosted APIs offer JSON-schema / structured-output modes. Local runners can constrain output to a grammar (llama.cpp GBNF grammars; Ollama's `format` option).
- **Why us:** Thought-engine v1 has a lot of code for sloppy free-text replies (`IMPORTANCE:` / `TOPIC:` lines). Constrained output removes most of that problem.

### B10. Evaluating thoughts ("evals")
- **What:** fixed test sets, rubrics, and pairwise "which is better, A or B?" comparisons, standard practice for judging AI output.
- **Why us:** Q11. Rate thoughts with five buttons, keep a replay set, and compare loop versions **pairwise** on the same replay.

### Two ways the thought pieces could fit together

| | **Lean loop** | **Directed loop** |
| --- | --- | --- |
| Input | Event-sourced observations (B4) + rules filter (B5) | same |
| Attention | Batched importance rating (B1) | Surprise trigger (B6) |
| Grouping | Match against open thoughts | Embeddings (B8) |
| Thinking | Revise only on new evidence | Blackboard controller (B2) + ReAct-style evidence fetching (B3) |
| Output shape | Structured outputs (B9) | same |
| Judged by | Ratings + baseline (B10) | same, compared pairwise with the lean loop |

---

## Suggested reading order (about 2–3 hours total)

1. **TUF overview + its threat list** (A1): the mindset for "install only what's approved."
2. **PASETO v4.public** or the **macaroons** idea (A5): what a narrow, signed decision looks like.
3. **Cedar's principal/action/resource/context** examples (A6): a clean way to name every decision.
4. **winget manifest** example (A4): what an approved-feature manifest could look like.
5. **Generative Agents** paper, sections on memory stream, importance and reflection (B1).
6. **ReAct** paper abstract and figure 1 (B3).
