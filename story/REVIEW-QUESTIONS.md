# Owner review — challenge the experience before building it

[← Story home](../README.md) · [Story chapters](00-START-HERE.md) · [Blueprint seeds](BLUEPRINT-SEEDS.md)

**This document is an invitation to disagree**, not an approved implementation checklist. The story and screens are imagined; no external AI reviewer has been automatically contacted or consulted by this commit.

## Questions for the owner

### Ownership and first use
- When you open Identity the first time, does the required GitHub connection feel justified and well explained?
- Does a **customer-owned copy of real starter source** meet your vision? Which parts belong in that copy versus protected publisher feature distribution? **The exact contents, license, repository-creation method and future update mechanism remain OPEN; do not demand a premature answer.**
- Should creating a companion/entity be part of the initial repository creation or a separate step once the repo exists?

### Startup and product boundaries
- When you press **Start Experience**, should it start Processor and automatically request Engine under one approved launch profile, or require a second Play button for Engine?
- Should Processor remain an independently launchable app outside Identity?
- When the user explicitly presses **Stop Demo**, should a session-started Engine stop, or should it remain running unless the user also asks to stop Engine?
- What is the *smallest* status view that still makes troubleshooting easy?

### Trust and thought experience
- Is the one-source, one-entity grant request understandable? Are we asking the owner to approve too many things?
- Would the first demonstration feel meaningful with **one real observation and an authentic receipt**, even before any AI worker is actually available, or must it include an actual Spark/Trace result?
- Does **no Thought formed yet** feel informative enough, or should the interface explain thresholds and how Traces accumulate?

### Faults and coming back
- Is it acceptable that **Engine remains running while Processor's UI is closed**?
- Should loss of GitHub after initial setup block **new privileged actions**, just updates/downloads, or only operations requiring current GitHub access? The policy is not decided.
- What should happen to queued but unprocessed observations on grant revocation? Deny/discard, pause for review, or preserve content with strict no-worker policy?
- What should an account or user be able to recover without Identity's original keys?

## Questions for Grok and other independent reviewers

Please identify **the first scene where a real user would be confused or misled**, then propose a simpler alternative. Critique the power split among Identity, Processor, launcher and Engine. Call out any hidden security assumption or dependency that makes the story impossible on Windows, WSL or Debian.

Review should distinguish:
- **A fictional scene that helps people discuss the design** versus **a fabricated data value falsely displayed by a live application**. The latter is prohibited. Challenge any storyboard element without a verifiable underlying source.
- Story flow/UX that could be improved.
- Technical contract that cannot safely work as proposed.
- Publisher-access claims that are unenforceable after source delivery.
- Identity runtime permissions that a local customer can actually control.
- Proof we should demand from the first real test.

**Reviewer's format:** What scene? What breaks? Why? What smaller alternative would be better? How do we test it? **Decision state remains PROPOSED** until the human owner accepts it.

## Review status

| Decision | Current status |
| --- | --- |
| GitHub required for normal first-open product readiness | **Owner direction** |
| Customer-owned initial starter source repository with origin credit | **Owner direction** |
| Publisher feature access separate from local Identity grants | **Owner direction** |
| Identity → Processor → Engine as first demonstration path | **Owner direction** |
| "Ensure Engine Available" instead of unconditional Start | **Proposed design** |
| One session launcher shared by Identity and Processor | **Proposed design** |
| Cross-service grant protocol, starter contents/license, repository bootstrap/update mechanism | **Open; do not assume a preferred method** |
| No fabricated live data, status, or model outcomes; unknown means unknown | **Owner direction** |
| Specific screens, wording and UI hierarchy | **Story concepts only** |
| Executable implementation | **Not created** |

No automatic merge, publisher access modification, GitHub repository creation, feature download or model test occurs through this narrative branch.
