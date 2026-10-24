# 05 · Something becomes a thought

[← A real permission](04-A-REAL-PERMISSION.md) · [Next: The system says no →](06-WHEN-IT-GOES-WRONG.md)

### The first experiment is tiny on purpose

Maya wants to see something happen. Processor offers one test she can actually understand:

> **Try an observation**  
> Source: **Authenticated Demo Game Observation**  
> Working identity: **Compass**
>
> "The player entered Goldshire."
>
> **[Send one observation]**

She clicks. The source sends a real HTTP observation to the Engine, which checks the actual Identity grant. Processor watches the receipts rather than claiming to be the sender or intercepting every model token.

The **What happened?** panel fills in one line at a time:

> **1. Demo source authenticated** — the sender was the approved source  
> **2. Identity checked** — current grant allowed this exact event scope  
> **3. Engine accepted** — event queued; not yet processed  
> **4. Worker called** — *only if an actual configured worker is ready*  
> **5. Engine recorded result** — actual Spark / Trace / Thought or none

A good control room distinguishes **accepted**, **queued**, **processed**, and **persisted**. An HTTP 202 is not a thought, and a model response alone does not mean that history was saved.

### A thought is an earned result, not a scripted animation

In this hypothetical successful branch of the story, a real fast worker notices the topic relates to a location. Engine records a lightweight **Spark** and, if the existing processing rules consider the evidence valuable enough, a **Trace**.

Maya tries another two **deliberate** examples:

> "The player returned to Goldshire later."  
> "The player came back a third time and paused near the inn."

The Engine can group related Traces into a candidate **Thought** according to its real thresholds and rules. If they do not meet those thresholds, the UI says **No Thought formed yet** and explains what it actually recorded. We should never prefill a fake thought to make the demo look successful.

What excites Maya isn't simply seeing a bright "Thought created" tile. It is opening the evidence and discovering what fed the idea: *which observations, which source, which real worker response, and which rule connected them*.

> **Compass / What I noticed**
>
> Recent observations: **3 submitted in this illustrative story**  
> Related theme: **Goldshire** (*possible, not guaranteed*)  
> Thought: **May form if the actual Engine decides the evidence supports it**
>
> **[View evidence] [Open Engine's own UI]**

The short narrative here is a **desired experience**, not a replay of actual Chat-Jimmy results. Worker text, ratings, number of Traces and outcomes in a real build must come from the actual service, or be labeled *Test fixture* when synthesized for automated tests.

### No Chat-Jimmy? An honest quiet screen

If the AI worker is unavailable, the Engine may still accept an authorized event for a bounded queue (subject to its actual tested persistence and limits). Processor says **Authorized and queued — waiting for a worker**. It does not produce a made-up Spark, and it doesn't call another paid API without consent.

That is still a successful observation **delivery** demonstration, but not a successful thinking demonstration.

### Why this matters for the bigger picture

Later, TASS, a screen reader, Discord and game add-ons might independently publish authorized observations. They should all use the same Engine intake and evidence contract, with **separate source approvals** and source-owned logs. The Processor can help people connect and inspect them, but it should not become the only route that allows thoughts to occur.

**Blueprint seed:** One-source demo contract, event and trace receipts, queue versus processing state, worker adapter, evidence inspector, no simulated thought creation, data ownership.

[Continue: The system says no →](06-WHEN-IT-GOES-WRONG.md)