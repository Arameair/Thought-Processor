# 02 · The control room

[← It becomes hers](01-IT-BECOMES-HERS.md) · [Next: The engine wakes →](03-THE-ENGINE-WAKES.md)

### One deliberate beginning

Identity now knows which user account and repository were verified and which **working entity** Maya selected. In the story she chooses an example companion called **Compass**. That name is a fictional example, not a predefined record created by this branch.

Identity looks intentionally calm:

> **Your installation**  
> GitHub: **Connected as Maya's account**  
> My repository: **Verified**  
> Working identity: **Compass**  
> Thought Processing feature: **Access available; not yet authorized**
>
> **[Start Experience]**

Maya clicks. The next screen says *exactly what will run*:

> **Start the experience**
>
> Thought Processor — **selected approved build**  
> Thought Engine — **selected approved build**  
> Demo Observation Source — **one local test-only sender**  
> Chat-Jimmy — **optional; availability must be verified**
>
> No automatic startup of other features.  
> Existing applications will not be stopped or replaced.
>
> **[Start selected experience] [Cancel]**

Identity is **not** asking for permission to read all game or desktop activity. She is approving the launch of a specific, bounded test session. The protected data permissions come later.

When she approves, a small **session launcher** starts or attaches the exact approved Thought Processor executable. It verifies the expected product, version, actual artifact and compatible interface rather than assuming any web page on localhost is Processor.

Identity opens a new workspace. The welcome card reads:

> **THOUGHT PROCESSOR — First Experience**  
> Selected identity: **Compass**  
> Source repository: **Maya / my-companion**  
> Identity: **Verified**  
> Engine: **Checking**  
> Worker: **Not yet verified**
>
> The center of the workspace is mostly empty. That is normal.

Maya has opened a **control room**, not moved her thoughts into a second identity database.

### Behind the screen

The startup request is handed to one bounded **session launcher**. It has an approved list of executables, specific version fingerprints, isolated data locations, request/session IDs and ownership records. This launcher is the same mechanism Processor will use to ask for the Engine, avoiding duplicate startup systems in Identity and Processor.

The launcher is a small component in the proposed architecture; we have not decided its final packaging or operating-system service strategy. A browser window closing will not itself kill Engine or Processor backend.

Processor does not own the canonical GitHub identity, publisher access or local authorization record. It reads verified context and presents the system's real running state.

### A small moment of uncertainty

Suppose Thought Processor is not installed. Maya sees **Processor build missing** and a specific approved candidate to select or install. Identity does not silently launch old Thought Controller just because it exists on the machine.

**Blueprint seed:** Start Experience consent, pinned manifest, session launcher ownership, Processor handshake, one workspace with evidence rather than fake sample data.

[Continue: The engine wakes →](03-THE-ENGINE-WAKES.md)