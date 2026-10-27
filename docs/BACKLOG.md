# Thought Processor — active design backlog

**Historical triage snapshot — October 26, 2026:** The previous open/closed issue counts reflected that day's design stage, **not today's issue list**. Later P0/P1 build, review and owner voice handoffs now exist. Use the [live GitHub issues](https://github.com/Arameair/Thought-Processor/issues) for current counts, [Issue #20](https://github.com/Arameair/Thought-Processor/issues/20) for confirmed cross-repo decisions, and [STATUS](STATUS.md) for implementation evidence.

## Current P0/P1 engineering and owner review

- [#19 — P1 feature plan](https://github.com/Arameair/Thought-Processor/issues/19): owner-promoted P0 and next integration goals.
- [#20 — Voice architecture decisions](https://github.com/Arameair/Thought-Processor/issues/20): Identity, Processor, Engine, Worker responsibilities and settled security boundaries.
- [#21 — Initial P1 build](https://github.com/Arameair/Thought-Processor/issues/21), [#22 — adversarial QA](https://github.com/Arameair/Thought-Processor/issues/22), [#23 — P1 cleanup](https://github.com/Arameair/Thought-Processor/issues/23): developer evidence and acceptance caveats. P1 is **not** promoted.

## Historical first stop: one design hub

**[#15 — First usable Thought Processor workspace](https://github.com/Arameair/Thought-Processor/issues/15)** is the focus of the current collaboration: a small, real **system-wide** Processor experience (proposed Profiles → Catalog → Collections). The [October 26 discussion document](https://github.com/Arameair/Thought-Processor/blob/experiment/processor-experience-design-20261026/docs/PROCESSOR_FOUNDATION_DESIGN_2026-10-26.md) lists prototype boundaries and owner acceptance criteria. **It is design only. No build is authorized.**

## Open issues by dependency

| Priority for discussion | Issue | Why it stays open |
| --- | --- | --- |
| **Now** | [#15 — Processor experience and first user action](https://github.com/Arameair/Thought-Processor/issues/15) | Decide navigation, first useful interaction, minimum authorized Identity API, and measurable first owner test. |
| **Now / architecture** | [#6 — Identity registration and authority](https://github.com/Arameair/Thought-Processor/issues/6) | Define real durable profile/feature authority and how services register; Processor cannot make its own grants. |
| **Next dependency** | [#8 — Identity ↔ Engine authorization test](https://github.com/Arameair/Thought-Processor/issues/8) | Receiver must enforce Identity permission, with deny/revoke/outage results, before Processor claims working permission controls. |
| **Later feature integration** | [#13 — Engine installation authorization](https://github.com/Arameair/Thought-Processor/issues/13) | Temporary repo-visibility fixture experiment exists; actual publisher entitlement, signed distribution, and runtime grants remain separate/unproved. |
| **Later / research** | [#2 — Canonical Thought Engine decision](https://github.com/Arameair/Thought-Processor/issues/2) | Avoid two competing Thought stores or silent data migrations. Not a prerequisite to discussing Processor's interface. |
| **Later / interface contracts** | [#4 — Source feeds and workers](https://github.com/Arameair/Thought-Processor/issues/4) | Agree scopes, observation provenance, batching and worker abstraction only when implementing relevant features. |
| **Independent critique** | [#7 — Architecture and Identity/experience review](https://github.com/Arameair/Thought-Processor/issues/7) | Historical critiques and unresolved hypotheses remain available; they do not override owner decisions or establish working code. |

### Closed as superseded, not accepted

- [#1](https://github.com/Arameair/Thought-Processor/issues/1) and [#3](https://github.com/Arameair/Thought-Processor/issues/3): older documentation foundation and Engine-centric control-point implementation gate.
- [#9](https://github.com/Arameair/Thought-Processor/issues/9): earlier **Identity-first** launch experience.
- [#10](https://github.com/Arameair/Thought-Processor/issues/10), [#11](https://github.com/Arameair/Thought-Processor/issues/11): historical implementation and Project Controller diagnostic-shell handoffs; useful technical/security notes preserved.
- [#12](https://github.com/Arameair/Thought-Processor/issues/12): October 25 prototype trial **record**; its actual evidence remains cited from [status](STATUS.md).
- [#14](https://github.com/Arameair/Thought-Processor/issues/14): reviewer suggestions, preserved for evaluating future experiments.
- [PR #5](https://github.com/Arameair/Thought-Processor/pull/5): unmerged October 23 documentation rewrite, closed without promoting any code.

**A closed issue or PR does not mean its previous acceptance tests passed.** Remnant valid questions were retained on open issues or the current design workshop.

## Owner-approved boundaries that apply to every issue

- **Thought Processor** is the *normal human/admin system-wide workspace*; **Identity** is the authority for IDs, security and permissions; **each receiving feature enforces** Identity decisions.
- Feature installations and data stores are independently owned. **No information-source access by default** and no cross-identity content sharing without explicit approval.
- Observed, authenticated, authorized, installed, enabled and actually working are different facts. A mocked worker or GitHub fixture cannot prove a live release.
- Alpha is manually promoted by the owner; functional experiments start from its approved baseline. No task, document or PR authorizes a branch promotion by itself.

[Public overview](../README.md) · [Readiness and conflicts](DESIGN_READINESS_2026-10-26.md) · [Source evidence](STATUS.md) · [Architecture decisions](DECISIONS.md)
