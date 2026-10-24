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
> Source: **Arameair starting project; approved source/version not yet selected**  
> Includes: **Actual starter software, attribution and origin — exact contents and legal terms to be established first**  
> Excludes: **Passwords, GitHub tokens, private user data, downloaded paid feature source**
>
> **[Review contents] [Create my version] [Choose existing repository]**

She reads the source acknowledgement. The program started with Arameair's original project, and that origin stays visible. **The exact files, redistribution license, and update method are not chosen today.** In the product we eventually build, a user must see the real approved contents and terms *before* any repository is created. Attribution does not substitute for a license.

**In the future version of the product, once the source contents and legal terms have been approved**, Maya presses **Create my version**. Only then would GitHub create a repository under her account. Identity would verify ownership, permissions, branch and the actual starting source. A success indicator should appear only after GitHub confirms the result. If creation fails, the screen reports a retryable failure without pretending a repository exists.

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
- A **not-yet-selected** lawful repository-creation method reproduces the approved *starter software*, recording its exact source and attribution; until that decision is made, creation must remain a design proposal.
- Identity associates the repository and user with durable installation and entity records, but stores secrets and runtime history outside GitHub.
- Publisher-controlled feature access is verified through an independent authoritative mechanism, **not** issued by Identity itself.
- No Thought Engine grant is implied by publisher access.

### The first hard question

**We do not have an answer yet**, and that is acceptable at the story stage. We still need to choose which real software files may be distributed, the license/permissions governing them, whether creation uses a fork, GitHub template or another approved method, and how future updates can be proposed without overwriting customer changes. **No method is selected or preferred by this chapter.** When those questions are answered, the story should be updated to reflect the actual policy and tested workflow—not guess at it.

**Blueprint questions:** What precisely belongs in the customer's software repository? What license permits that? Which creation and update method is verifiably safe? What evidence proves account/repository ownership? Until resolved, **do not fabricate a starter version or claim an existing release**.

[Continue: The control room →](02-THE-CONTROL-ROOM.md)