# Thought Processor — active design backlog

**Verified October 26, 2026:** **7 open issues (#2, #4, #6, #7, #8, #13, #15), 7 historical issues closed as superseded (#1, #3, #9, #10, #11, #12, #14), and 0 open pull requests**. PR #5 was closed without merging. Older branches, evidence and conversation remain in GitHub history.

## First stop: one design hub

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
