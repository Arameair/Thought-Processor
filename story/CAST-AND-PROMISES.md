# The cast — who everyone is and what each one promises

[← Story home](../README.md) · [Start reading the story](00-START-HERE.md)

The names are fictional and the applications are not implemented as described here. **A future product must never use these examples as live account, permission, process or model data.** Think of these as **actors with responsibilities**, not a requirement to combine them into one executable.

| Actor | What Maya understands | What it is responsible for | What it must not pretend to do |
| --- | --- | --- | --- |
| **Maya** (fictional customer) | "I own my starting version." | Approves GitHub repo, installation, demo and data permissions | Her local UI cannot create publisher entitlements out of thin air |
| **GitHub** | "My account and my repository." | Authenticates the user, stores customer-owned starter source and history | Decides whether a local Engine can read a game event |
| **Arameair publisher distribution** | "Which optional features can I obtain?" | Enforces delivery/hosted access at trusted publisher boundaries and publishes approved updates | Reliably deletes copied software or unilaterally grants access to Maya's private local data |
| **Identity** | "Who am I? What exists? What is allowed?" | Durable installation, entity/service identities; registry; approved locations; **local operation grants** | Processes thoughts, mints publisher entitlements, owns every log |
| **Thought Processor** | "What is connected and what happened?" | Small operator workspace, connection intent, health, granted actions and evidence timeline | Becomes the single point every observation must pass through or invents grants |
| **Session launcher** | "Make the approved parts available." | Starts/attaches pinned processes, tracks ownership and safe stop/reconnection | Starts every installed feature or kills independent processes by port/name |
| **Thought Engine** | "Notice and process what I'm allowed to notice." | Authentic observation intake, queue, Sparks/Traces/Thoughts, worker calls, its own persistence | Trusts a payload label as caller identity, overrides Identity permission |
| **Demo source** | "Here's one sample event." | Sends a genuinely authenticated observation for one selected entity | Claims to be TASS, a game, or ScreenBridge based on string text |
| **Replaceable AI worker** | "Help analyze context when a model is truly available." | Answer actual scoped worker requests when verified and authorized | Issues permissions, owns thought history, or pretends its answer is the whole companion |
| **Memory** (future) | "Help retain and recall long-term experience." | Separate, optional durable retrieval/persistence under permission | Automatically exist because Thought Engine has temporary thought records |

## Four identities that must not be confused

**Customer GitHub account** is the user's account with GitHub; **installation identity** is the running deployment; **companion/entity identity** is Compass; **service identity** is a durable credential-bearing feature such as the Engine or Demo Source. A **runtime instance** is just the specific process currently running and should change on restarts.

## Four different kinds of evidence

- **GitHub verified:** the user proved access to a particular repository; not a local observation grant.
- **Publisher entitlement verified:** source delivery/hosted access approved by a publisher-controlled decision; not proof of local permission.
- **Identity grant verified:** an authenticated source may perform one scoped operation for an entity/target; not proof a service is running.
- **Observed process or event receipt:** a specific service responded or performed an action; not an enduring permission.

### One sentence per boundary

> **GitHub establishes customer-owned source. Publisher controls delivery of protected features. Identity grants local authority. Thought Processor operates and explains. Thought Engine performs and enforces.**

## Who owns the data?

Maya's GitHub repository keeps her permitted starter **source files, provenance, and changes**, not raw credentials or automatically every runtime observation. Identity keeps canonical ID/policy metadata in its selected protected root. Engine retains its own events, Traces and Thoughts. Other source features retain their own source logs. Memory, when enabled later, retains its own historical data.

**Blueprint seed:** Authority matrix, entity/service/instance schema, runtime storage allocation and independent deployment model.
