# Teams

**Shared file, v0.1 — October 31, 2026.** Same text in all five repos' `teams/` folders; change all copies together.

Each repository in Dustin's Brain system has a team. This folder holds **that repository's team identity** (who the team is, what it owns, what it refuses, and how it works with the others) plus the rules every team shares.

**Status:** Proposal on the `teams` branch. It appoints no one, starts no agent and changes no code. Dustin approves anything that moves to `alpha`, `stable` or `main`.

## Files

| File | What |
| --- | --- |
| `README.md` | This page: the directory of all teams |
| `TEAM_OPERATING_MODEL.md` | Shared rules: five roles, `WORKING` / `DONE` / `BLOCKED`, alerts, handoffs, Ground Truth First, pulse |
| `TEAM_LIAISON.md` | Shared charter for the Liaison role every team has |
| `<NAME>_TEAM.md` | **This repository's own team identity** |

## The five teams

| Code | Team | Repository | In one line | Team identity |
| --- | --- | --- | --- | --- |
| `IDN` | **Identity** | [Arameair/identity](https://github.com/Arameair/identity) | The authority: who exists, who may do what for whom | [IDENTITY_TEAM.md](https://github.com/Arameair/identity/blob/teams/teams/IDENTITY_TEAM.md) |
| `TP` | **Thought Processor** | [Arameair/Thought-Processor](https://github.com/Arameair/Thought-Processor) | The control surface: the one place people set up, install, approve and watch | [THOUGHT_PROCESSOR_TEAM.md](https://github.com/Arameair/Thought-Processor/blob/teams/teams/THOUGHT_PROCESSOR_TEAM.md) |
| `TE` | **Thought Engine** | [Arameair/Thought-engine](https://github.com/Arameair/Thought-engine) | AI-driven reasoning: observations → Sparks → Traces → Thoughts, per identity | [THOUGHT_ENGINE_TEAM.md](https://github.com/Arameair/Thought-engine/blob/teams/teams/THOUGHT_ENGINE_TEAM.md) |
| `THT` | **Thought** | [Arameair/Thought](https://github.com/Arameair/Thought) | Deterministic Spark → Trace → Thought rules as a Processor feature | [THOUGHT_TEAM.md](https://github.com/Arameair/Thought/blob/teams/teams/THOUGHT_TEAM.md) |
| `TASS` | **TASS** | [Arameair/TASS](https://github.com/Arameair/TASS) | Dustin's private companion, running on a local model, as a Processor feature | [TASS_TEAM.md](https://github.com/Arameair/TASS/blob/teams/teams/TASS_TEAM.md) |

## How the teams fit together

```text
                          Dustin (owner, final approval)
                                     |
           +-------------------------+--------------------------+
           |                                                    |
   IDN  Identity  <-------- asks for approvals / tickets -------  TP  Thought Processor
   (authority: IDs, grants,                                     (control surface: install,
    revocation, live checks)                                     start, stop, show status)
           ^                                                    |
           |  every feature checks grants live                  |  installs and runs
           |                                                    v
           +-------------------  TASS · THT · TE  <-------------+
                                (features: own their work and data)
```

- **Identity decides.** Nobody else grants access.
- **Thought Processor coordinates and shows.** It doesn't grant and doesn't do the features' work.
- **Features do their own work** (TASS, Thought, Thought Engine) and enforce Identity's grants themselves.

## Who to go to for what

| You need… | Go to |
| --- | --- |
| A new capability name, grant type, ticket field or installation rule | `IDN` |
| A screen, install flow, launcher, or a change to the feature API contract | `TP` |
| AI reasoning over observations, Chat-Jimmy or worker integration | `TE` |
| Deterministic Spark/Trace/Thought rules, or that feature's capabilities | `THT` |
| Companion behavior, the pulse/genesis, local model setup | `TASS` |
| Cross-repo priorities or a conflict between teams | Dustin, via [Developmnt issue #1](https://github.com/Arameair/Developmnt/issues/1) |
