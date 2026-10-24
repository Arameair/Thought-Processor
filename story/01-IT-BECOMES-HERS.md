# 01 · It becomes hers

[← The door opens](00-START-HERE.md) · [Next: The control room →](02-THE-CONTROL-ROOM.md)

### The account is real; the repository is hers

Maya clicks **Connect GitHub**. An authorized GitHub authentication flow identifies the account **she** chose. Identity displays the actual username returned by GitHub, the permitted repository scopes and whether it can create a repository on her behalf. An unfamiliar account would be a reason to stop.

She sees a second question:

> **Create your starting version**
>
> Destination: **Maya's GitHub account**  
> New repository: **my-companion**  
> Visibility: **Private** (illustrative selection, not a mandated default)  
> Source: **Approved Arameair starter release**  
> Includes: **Actual starter application files, setup materials, source-credit notice, origin and source version**  
> Excludes: **Passwords, GitHub tokens, private user data, downloaded paid feature source**
>
> **[Review contents] [Create my version] [Choose existing repository]**

She reads the source acknowledgement. The program started with Arameair's original project, and that origin stays visible. The specific license and distribution rights are shown honestly once chosen; attribution does not by itself substitute for a license.

She presses **Create my version**. Only now does GitHub create a repository under her account. Identity verifies ownership, permissions, branch and the exact starting version; a green check appears only after GitHub confirms it. If creation fails, the screen reports a retryable failure without pretending a repository exists.

This is **not** a configuration-only repository. Maya can inspect and modify the starter source she received, subject to the actual license.

### The feature shelf has a different owner

Maya is shown optional feature cards:

> Thought Processing — **Eligible for this demo**  
> Memory — **Not installed**  
> Screen observation — **Not available in this test**
>
> **Feature access comes from the publisher**. Local data permissions are chosen separately.

She hasn't allowed the Engine to read anything yet. "Eligible" means the publisher permits delivery/use under a specified policy, **not** that an untrusted locally edited settings field can authorize a premium download.

A protected feature may live in the publisher's private repository or hosted service. The publisher can refuse **new downloads, updates or hosted use** at the boundary they control, but cannot reliably retrieve or erase code already delivered to Maya's editable machine. The customer should never be surprised by an undocumented remote kill switch.

### Behind the screen

- GitHub verifies account and repository access.
- A reviewed bootstrap method reproduces the approved *starter software*, recording its exact source and attribution.
- Identity associates the repository and user with durable installation and entity records, but stores secrets and runtime history outside GitHub.
- Publisher-controlled feature access is verified through an independent authoritative mechanism, **not** issued by Identity itself.
- No Thought Engine grant is implied by publisher access.

### The first hard question

Which starting files are actually redistributed, under what license, and how are upstream improvements later offered without overwriting Maya's changes? This decision is still **open**. A controlled bootstrap from a versioned starter release is our proposed direction, not an implemented provisioner.

**Blueprint seed:** Account consent and repository creation transaction; starter contents/attribution, distinct publisher-access evidence, safe storage separation, update lineage.

[Continue: The control room →](02-THE-CONTROL-ROOM.md)