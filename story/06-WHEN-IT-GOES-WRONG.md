# 06 · The system says no

[← A thought begins](05-A-THOUGHT-BEGINS.md) · [Next: Maya comes back →](07-THE-NEXT-DAY.md)

### A companion she can stop trusting

Maya has seen her first observations move through the system. Now she wants to know if she can stop it.

She clicks **Revoke observation permission**.

> **Identity confirmed: permission revoked**  
> Demo Source → Thought Engine for Compass: **DENIED**  
> The Engine remains installed and running.  
> Existing history has **not** been deleted.
>
> **[Try another observation]**

She presses Try Again. The screen reports an authentic refusal:

> Sender: **authenticated**  
> Identity decision: **DENY — grant revoked**  
> Engine intake: **REJECTED**  
> Worker calls: **0 for this refused event**  
> No new thought created.

The difference is profound: stopping permission did **not** destroy the Engine, cancel GitHub ownership, remove files from the publisher, or switch off Maya's other unrelated features.

### A service goes missing

Next, the demo temporarily makes **Identity unavailable** as a deliberate failure test. This is not the same as Identity issuing a denial.

Processor says:

> **Cannot verify permission right now**  
> Identity is unreachable.  
> Thought Engine is still running, but new protected observations are **paused / refused** while authorization cannot be verified.
>
> **[View diagnostic details] [Retry verification]**

For the first security experiment, Engine **fails closed**: no new privileged event is admitted without an online Identity decision, and queued work is checked again before sending it to a model. Future offline leases may be studied separately, but this story does not quietly assume a five-minute exception.

If Identity is lost after data was **already sent** to a worker, revocation cannot magically retract that transmission. The UI tells Maya this limitation instead of claiming perfect retroactive deletion.

### A crash is not the end of the installation

Processor's browser view accidentally closes.

Maya reopens it. The Engine is still running. The independent session launcher remembers that it started the Engine and Processor reattaches to the same verified instance. It doesn't manufacture a fresh identity or start a duplicate process.

If the Processor backend itself crashes, the separately running launcher retains the session record and reconciles the Engine on restart. If only the Engine crashes, Processor shows the exit reason and offers a **deliberate** reconnect or restart, not infinite automatic restart attempts or duplicate work.

When a port is occupied by something else, Processor reports **Wrong or unknown service on port**, never quietly connects to it because the page returned HTTP 200.

### A model connection fails

The selected model worker can be unavailable while Engine remains usable for diagnostics. An already-authorized event might wait in the queue. Processor shows that actual status and preserves retry safety instead of silently sending it to a different worker or inventing a response.

### Publisher access changes

If Maya loses the right to download an **additional** publisher-controlled feature, the store may say **Unavailable for new install/update**. The system does not pretend it can erase a copy that has already been legitimately delivered into her own repository or machine. Identity's grant system still governs local authenticated operations; it is not a hidden product-licensing switch.

### The test of product quality

Maya can now answer five questions without reading a server log: **What stopped? Why? What is still working? Did any data leave my machine? What can I safely do next?**

**Blueprint seed:** Explicit denial versus outage, queue-revocation behavior, in-flight disclosure, launcher recovery, process ownership, worker failure, publisher access and safe diagnostics.

[Continue: Maya comes back →](07-THE-NEXT-DAY.md)