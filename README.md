# The First Spark
## A story-driven prototype of the Arameair companion ecosystem

> **This branch is a readable experience prototype, not a software prototype.**  
> No executable, mock application, UI implementation or live integration is created here. Every screen and event described is a **proposed future experience**, not a current product capability. **Fictional storytelling is not live telemetry:** no account, connection, permission, metric, thought or AI output in this narrative has been verified as an operating result.

**The question this project explores:** What does it actually *feel like* to own a companion, open Identity, launch Thought Processor, and watch the first authentic observation become part of a developing thought?

Rather than start with API specifications and wiring diagrams, we're following **Maya**, a fictional first-time user, through one ordinary afternoon. She wants a companion that can accompany her through a game, notice recurring experiences, and eventually become more useful without taking control away from her. She is not a demonstration of a running app; she is a way to test whether our design makes sense to a human.

This prototype intentionally shows the awkward moments too: installing a feature, failing an authorization check, losing a model connection, coming back tomorrow, and receiving an update. We will turn those events into blueprints **after** the story makes sense.

### Begin here

**[▶ Start the story — The door opens](story/00-START-HERE.md)**

| Chapter | What Maya experiences | The design question |
| --- | --- | --- |
| [00 · The door opens](story/00-START-HERE.md) | A companion she can own, not a login to someone else's machine | What does this product promise? |
| [01 · It becomes hers](story/01-IT-BECOMES-HERS.md) | GitHub account, genuine user-owned starter repository, credit to its creator | What does a customer actually own? |
| [02 · The control room](story/02-THE-CONTROL-ROOM.md) | Identity opens the exact Thought Processor chosen for this installation | How does one app hand control to another? |
| [03 · The engine wakes](story/03-THE-ENGINE-WAKES.md) | Processor finds or starts the correct Engine without duplicates | Who ensures a service is available? |
| [04 · A real permission](story/04-A-REAL-PERMISSION.md) | A small feature asks permission, and the Engine proves it can check | How do we distinguish running from allowed? |
| [05 · Something becomes a thought](story/05-A-THOUGHT-BEGINS.md) | One small event, real evidence, perhaps a Spark/Trace/Thought | What does intelligence look like when it's *not* faked? |
| [06 · The system says no](story/06-WHEN-IT-GOES-WRONG.md) | Revocation, missing worker, broken connections and recovery | Is the failure experience understandable? |
| [07 · Maya comes back](story/07-THE-NEXT-DAY.md) | Reopen without rebuilding identity; own data, updates, optional features | What remains stable as the system evolves? |

### When you've finished reading

- **[The cast and promises](story/CAST-AND-PROMISES.md)** explains each component in plain language, including **publisher feature access** versus **Identity runtime grants**.
- **[Storyboard of the actual user-facing screens](story/SCREEN-STORYBOARD.md)** shows what the next UI designer could draw, step by step.
- **[From story to blueprints](story/BLUEPRINT-SEEDS.md)** identifies concrete behavior, contracts, unknowns and testable acceptance criteria *without trying to solve them all in code*.
- **[Trust and evidence: no invented live data](story/TRUST-AND-EVIDENCE.md)** establishes the rule every later working UI must follow.
- **[Owner and independent reviewer questions](story/REVIEW-QUESTIONS.md)** is where feedback, including Grok's, should challenge the story.

### Four permanent boundaries

1. **GitHub makes the customer's source their own.** Normal first-opening setup requires connecting the customer's GitHub account and creating or selecting their own **real starter software repository**, not just a settings repository. Credit and origin must be retained. The exact starter contents, licensing, copying method and future-update policy are **not decided yet**; do not imply a technical method is already selected. Before GitHub is connected, setup/repair is available but the product isn't presented as fully operational.
2. **Publisher access controls delivery of protected features**, not the customer's local identity authority or arbitrary ability to erase already-delivered code.
3. **Identity decides** who/what exists and which local operation is authorized. **Thought Engine enforces** the operation grant. **Thought Processor connects, tests and explains** the real system.
4. **Processes are independent.** Processor asks an approved local launcher to *ensure the Engine is available*. Closing a browser tab does not destroy an Engine or its thoughts.

The future companion can eventually make observations from games, screen activity, Discord and other feature sources. **In this story we deliberately use only a tiny authenticated demo observation source.** No real game data, Memory, dynamic coding, autonomous coding or publisher billing platform is needed to understand the first experience.

### Status and provenance

| Item | Status |
| --- | --- |
| This story | **Design fiction / reviewable narrative** |
| Actual Thought Processor application | **Not built on this branch** |
| Thought Engine Identity grant experiment | **Engine-side experiment, no verified live Identity integration yet** |
| Automatic process management | **Proposed, not implemented here** |
| Alpha / Stable / Main | **Unchanged by this story** |

**Repository arrangement:** This is a self-contained *story edition* on a separate branch of `Arameair/Thought-Processor` because the connected GitHub tool can write to an existing repository but cannot create a new one. The `story/` directory and this branch README are designed to be moved into a dedicated `First-Spark-Storybook` repository later without needing the application source.

**Human-directed project built with AI collaboration.** This story is design work, not a claim that any model vendor endorses or verifies the system. Actual software licensing, the publisher access contract and exact blueprint designs remain owner decisions.
