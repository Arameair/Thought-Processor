# 03 · The engine wakes

[← The control room](02-THE-CONTROL-ROOM.md) · [Next: A real permission →](04-A-REAL-PERMISSION.md)

### One question instead of a hundred startup buttons

Maya is looking at Processor. The Engine card says **Checking**, not green.

Processor quietly asks its authorized session launcher one question:

> **Ensure the selected Thought Engine is available.**

It is not saying *start a new copy*. It is asking for a verified working Engine instance, preferably an existing one.

The launcher examines the approved session profile and the running processes it knows about. One of three sensible stories could unfold.

**If the right Engine is already running:** it verifies the installed executable, the trusted service identity and its compatible protocol. It returns **Attached — independently running** with an instance ID, build provenance and address. It does not spawn a twin Engine or take ownership of the existing process.

**If the right Engine is installed but stopped:** it starts the approved executable in its own feature data folder, waits for the actual service handshake and reports **Started by this session**. Starting is not the same thing as ready; Maya sees **Starting → Connected → Engine available** only as each check succeeds.

**If something is wrong:** perhaps the binary is missing, the port is occupied, the service responds with the wrong protocol, or the build hash doesn't match. Processor shows **Needs attention** and the real reason. It never kills an unknown process to free the port.

In Maya's story, the correct Engine starts. The UI changes:

> **Thought Engine**  
> Process: **Started by this experience**  
> Version: **Approved candidate (exact SHA available)**  
> Connection: **Authenticated**  
> Engine: **Available**  
> Thought permission: **None yet**  
> AI worker: **Unverified**
>
> **[Inspect Engine] [View logs] [Show timings]**

Maya feels a small but important difference: the control room is not telling her that the companion is "alive" because a server started. It is honestly telling her which machine is ready and what it still cannot do.

### Behind the screen

This is the **Processor–Engine gap** the prototype is meant to clarify.

The **Processor backend** owns the operator intent, connection state and diagnostics. It asks one bounded **session launcher** to ensure the selected service is available. The launcher does process management. Thought Engine owns its executable, actual API, queue, data and worker calls. Identity owns the registration and authority. None of them becomes a general proxy for every observation.

A first prototype may use same-host HTTP/JSON for a versioned Engine health/operation interface. That is a design choice to test, not an implemented authenticated protocol. The launcher should compare the approved executable digest and service registration; a claimed Git SHA in `GET /api/status` alone does not establish trust.

The launcher remembers **owned vs attached**, so Stop Demo cannot accidentally terminate something Maya started independently. If Processor's browser tab closes, Engine stays running. If the Processor backend crashes, a separately living session launcher should let the next Processor reconnect without duplication.

### The question we'd ask the owner here

Should the Engine continue running after **Stop Experience**, or only after *closing Processor*? Our proposed first-demo answer is: it survives closing the UI or temporarily losing Processor, but an explicit **Stop Demo** ends only the processes this session itself started. Independent engines keep running.

**Blueprint seed:** `ensure_running` intent; lifecycle adapter, actual process ownership, identity/build verification, startup state machine, reconnection and safe stop.

[Continue: A real permission →](04-A-REAL-PERMISSION.md)