# 04 · A real permission

[← The engine wakes](03-THE-ENGINE-WAKES.md) · [Next: A thought begins →](05-A-THOUGHT-BEGINS.md)

### A working engine is not a listening engine

Maya sees that Thought Engine is connected. She wants to try it. Processor presents **a single, understandable permission request**:

> **Allow this one test?**
>
> Working identity: **Compass**  
> Test source: **Demo Game Observation**  
> Destination: **Thought Engine**  
> Permission: **Send observations for Thought Processing**  
> Scope: **This identity and this source only**
>
> The request does **not** allow screen recording, Discord access, long-term Memory, access to GitHub repositories, or other sources.
>
> **[Ask Identity for approval] [Not now]**

She clicks approval. Processor does not flip an `Engine enabled` flag behind the scenes. Instead it asks **Identity's actual authority** for a scoped grant. Identity authenticates who is approving, records the grant and returns evidence of the decision. If Identity cannot confirm the request, the operation stays unapproved.

The card now says:

> **Observation permission: Approved by Identity**  
> Grant: **scoped to Compass / Demo Source / Thought Engine**  
> Last verified: **current check**  
> **[Revoke] [Inspect permission]**

There is another important identity hidden behind the card. "Compass" names the **entity** that owns the thought history; it is *not* the service sending events. The sender has a separate authenticated service identity. A random local application cannot impersonate the demo source by typing `source: "demo"` in a message.

### What happens on the next event

The Demo Source sends the observation **directly to Thought Engine**, not through Processor. Engine authenticates the sender, validates the event and asks Identity:

> Is service S permitted to submit an observation for entity Compass to Engine E, right now?

Identity responds **ALLOW, DENY or UNAVAILABLE** with a verifiable decision and request identity. Only an authenticated, matching **ALLOW** can queue the event. The Engine checks again before a model request, because the owner could revoke a grant while an observation waits.

The Engine's original experimental local ON/OFF flag is **not** a substitute for this external authority.

### Two kinds of access Maya should never confuse

Earlier the publisher said she could **obtain** the Thought Processing feature. That was feature distribution/entitlement. Now Identity decides whether that installed feature may **process a particular source's data**. The publisher's approval can't authorize reading Maya's data; Maya's editable local settings can't manufacture paid publisher access.

Maya doesn't need to understand cryptography to appreciate the distinction. The screen should express it in ordinary language and provide technical evidence on demand.

### Current reality check

The Engine has an **experimental Engine-side grant-check adapter**, including a proposed signed decision format. As of this story, a matching real Identity endpoint and end-to-end verification are still missing. This chapter is **the experience we want to prove**, not something the current apps can already do. Until the real endpoint exists, the real screen must say **Authorization not available**, never invent a grant or show a permission card as verified.

**Blueprint seed:** Two independent kinds of authority, separate entity/source/service identities, owner grant approval, decision verification, revocation, action-time enforcement and evidence ID correlation.

[Continue: A thought begins →](05-A-THOUGHT-BEGINS.md)