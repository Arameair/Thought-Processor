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

In one **possible future outcome**, a genuinely connected model worker might identify the location as relevant, and the Engine might record a Spark or Trace. **Neither result is promised by the story.** Actual worker inputs, outputs and Engine decisions must be read from real event records when we build it.

Maya tries another two **deliberate** examples:

> "The player returned to Goldshire later."  
> "The player came back a third time and paused near the inn."

The Engine can group related Traces into a candidate **Thought** according to its real thresholds and rules. If they do not meet those thresholds, the UI says **No Thought formed yet** and explains what it actually recorded. We should never prefill a fake thought to make the demo look successful.

What excites Maya isn't simply seeing a bright "Thought created" tile. It is opening the evidence and discovering what fed the idea: *which observations, which source, which real worker response, and which rule connected them*.

> **Compass / What I noticed**
>
> Recent observations: **[actual number from Engine receipts, otherwise Unknown]**  
> Related theme: **[actual detected topic, or None]**  
> Thought: **[actual result, or No Thought formed]**
>
> **[View evidence] [Open Engine's own UI]**

The narrative above is **not** a replay of an operating system. Worker responses, ratings, counts and persistence claims must come from the actual service. **Do not inject fixture outputs into the user-facing experience.** If fixtures are required for isolated automated tests, keep their results separate and unmistakably identified as not live.

### No verified worker? An honest quiet screen

If no AI worker is verified as available, the Engine may be able to accept an authorized event for a bounded queue (subject to its actual tested persistence and limits). **Only if an event really was accepted and persisted** should Processor say **Authorized and queued — waiting for a worker**; otherwise it shows the truthful refusal or unknown result. It does not produce a made-up Spark, and it doesn't call another paid API without consent.

That is still a successful observation **delivery** demonstration, but not a successful thinking demonstration.

### Why this matters for the bigger picture

Later, TASS, a screen reader, Discord and game add-ons might independently publish authorized observations. They should all use the same Engine intake and evidence contract, with **separate source approvals** and source-owned logs. The Processor can help people connect and inspect them, but it should not become the only route that allows thoughts to occur.

**Blueprint question:** What exact authenticated, timestamped evidence proves *each transition* from source submission to persisted Engine result, including **no Thought formed** and **worker unavailable**? See [Trust and evidence](TRUST-AND-EVIDENCE.md).

[Continue: The system says no →](06-WHEN-IT-GOES-WRONG.md)