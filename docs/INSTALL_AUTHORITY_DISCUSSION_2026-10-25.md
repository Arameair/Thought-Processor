# Authority to install the Thought Engine — discussion notes

**Status:** DISCUSSION NOTES ONLY. No code changed; nothing here is decided.
**Date:** 2026-10-25 · **Branch:** `experiment/install-authority-discussion-20261025` (from `alpha` @ `9fac9fa`)
**Owner direction (2026-10-25):** build Thought Processor first; its job is to make sure it has **authority to install the Thought Engine** before it does so.
**See also:** [Design ideas and approaches to look at](DESIGN_IDEAS_TO_LOOK_AT_2026-10-25.md) (Part A covers the patterns behind each option here).
**Builds on:** the detailed requirements in [Identity → Engine lifecycle approval](https://github.com/Arameair/Thought-Processor/blob/experiment/processor-controller-launch-fix-20261024/docs/contracts/IDENTITY_ENGINE_LIFECYCLE_APPROVAL_REQUEST.md) (start / attach / stop / ingest) and the running Identity Prototype B (`Arameair/identity`, `experiment/identity-home-claude-20261025`).

---

## 1. The goal in one sentence

> Thought Processor installs a specific, approved Thought Engine build for a specific person **only** after it can show — with evidence it didn't produce itself — that this person is allowed to have it, and it records exactly what it installed and why.

**New scope to note:** the existing contract covers *start, attach, stop, ingest*. **Install** is a fifth, separate scope. Being allowed to install doesn't mean being allowed to start, and the reverse.

## 2. "Authority" is really a chain of five questions

| # | Question | Who should answer it | What exists today |
| --- | --- | --- | --- |
| 1 | **Who is asking?** | Identity (GitHub sign-in) | Identity B's GitHub device sign-in — being tested on your machine now |
| 2 | **Is this person allowed to have the Thought Engine?** | The **publisher** (Arameair), recorded somewhere Identity can check | Nothing — Identity B only shows "repository visible", deliberately not a grant |
| 3 | **Is it really Identity telling Processor this?** | A trust link between the two local programs | Nothing — both only see each other's public `/api/health` ("self-reported, unverified") |
| 4 | **Exactly what is being installed?** | The publisher's approved build (commit + file digest) | Nothing — Thought-engine has no `alpha`, no releases, no "approved" build |
| 5 | **What was installed, when, on whose authority?** | Processor's install record | Nothing |

Each row needs a decision. The rest of this note is options for each.

---

## 3. Q2 — What proves a person is allowed to have the Thought Engine?

This is the core question.

**A1. GitHub repository access *is* the grant (for now).**
Arameair grants someone the Thought Engine by giving their GitHub account read access to `Arameair/Thought-engine` (collaborator or team). Identity checks it with the person's own sign-in.
- ✅ Real and enforced by GitHub itself: without access, nobody can even download the code. Revocable in one click. No new servers or keys.
- ⚠️ Coarse: "can read the code" = "may install the feature". The home-trial rules said visibility is *not* an entitlement — so if you choose A1, it should be a **conscious, written decision** ("for v0, repo read access is the install grant"), and the UI should say so plainly.
- ⚠️ Works only while repos stay private.

**A2. A publisher-signed grant.**
You (the publisher) sign a small record — `{github user id, feature: thought-engine, allowed build, expires}` — with a private key that only you hold. Identity ships the matching **public** key built in (like the GitHub App Client ID). Anyone can verify it; only you can create one.
- ✅ Precise (per feature, per build, with expiry), works offline, doesn't depend on repo privacy.
- ⚠️ Needs a small signing tool for you and a way to deliver the grant (e.g. a private "grants" repo Identity reads, or a file you send).
- ⚠️ Revocation = short expiry or a published revocation list.

**A3. A hosted request → approve service.**
The full "request access → publisher approves" flow from the Identity vision.
- ✅ The eventual product.
- ⚠️ A server to build, host and secure. Too much for the first real prototype.

*Lean:* **A1 for the first working prototype**, written down as an explicit temporary rule, with **A2 as the next step**. A1 is the only option that is real today with zero new infrastructure, and it's enforced by GitHub rather than by our own code.

---

## 4. Q3 — How does Processor know the answer really came from Identity?

The owner decision is that Identity is the authority and Processor is not a second one. So Processor shouldn't just do its own GitHub check and decide for itself — it should get a **decision from Identity** that it can verify.

**B1. Pair once, then signed decisions.**
First time only: Identity shows a short pairing code; you type it into Processor (like pairing a Bluetooth device). From then on Identity **signs** each decision with its own private key (Ed25519), and Processor checks the signature with Identity's public key it saved during pairing.
- ✅ Only Identity can create a decision — this avoids the weakness the contract doc already flags with the Engine's shared-HMAC experiment, where anyone holding the shared key could forge one.
- ✅ Processor can keep the signed decision as proof in its install record.

**B2. Processor signs in to GitHub itself and checks repo access.**
- ✅ Simplest to build.
- ⚠️ Makes Processor its own authority (contradicts the design), and the person signs in twice.

**B3. Shared folder / file drop.** Identity writes a decision file Processor reads.
- ⚠️ Any local program could write that file. Weak; not recommended.

*Lean:* **B1.** It's a small amount of code on each side and it's the honest version of "Identity approves".

**What a signed install decision would carry** (sketch, not a format):
person (GitHub id + login, verified when) · Identity installation id · Processor instance id · action = `install` · feature = `thought-engine` · exact build (repo, commit, file digest) · grant evidence (A1: "repo access confirmed at time T") · decision id · issued / expires (minutes, single use) · Identity's signature.

---

## 5. Q4 — Exactly what gets installed, and how does it arrive?

Today there is nothing "approved" to install: Thought-engine has two experiment branches and no `alpha` or releases.

**What gets installed:**
- **C1. A pinned commit** the publisher marks as approved (e.g. an `alpha` tag), built from source on the user's machine. ⚠️ Requires Go on every user's computer — not realistic for your brother.
- **C2. A prebuilt binary** attached to a GitHub Release on the private Thought-engine repo, plus its SHA-256. Processor downloads it **with the person's own GitHub sign-in** (which under A1 *is* the access check) and refuses it if the digest doesn't match the decision.

*Lean:* **C2.** It's what makes "your brother clicks install" possible, and the digest check means Processor installs exactly what was approved.

**How it arrives:** Processor needs the person's GitHub sign-in to download. Options: Identity hands Processor a short-lived download link, or Identity downloads and hands over the file plus the signed decision. *Lean:* **Identity downloads, Processor verifies digest + signature, Processor installs** — the GitHub token never leaves Identity.

---

## 6. Q5 — What does "installed" mean afterwards?

To decide:
- **Where:** a Processor-owned folder (e.g. `<processor profile>/features/thought-engine/<commit>/`), separate from the Engine's own data folder.
- **Install record:** decision id, signed decision kept as proof, build commit + digest, installed at, by whom. Shown in Processor's UI and Copy Context.
- **Uninstall:** remove the files; never the Engine's data without a separate confirmation.
- **Updates:** a new build needs a **new** decision (the old one names the old digest).
- **Revocation:** if the grant is later withdrawn, does Processor uninstall, refuse to start, or only warn? (The contract doc's rule: loss of permission is not automatically a kill instruction.)

---

## 7. Q6 — Install, then what?

Today Processor deliberately has **no process runner** — it cannot start anything. Installing a program you then can't start is only half useful.

- **D1. Install only.** Processor installs and shows "installed, not started"; you start the Engine yourself (or via Project Controller). Smallest step.
- **D2. Install, then a separate `start` decision.** Processor gains a runner, and starting needs its own signed decision (the existing contract's start scope).

*Lean:* **D1 first**, D2 immediately after — same pairing and signing, a different action name.

---

## 8. Smallest real prototype (strawman to argue with)

Using the leans above (A1 + B1 + C2 + D1):

1. **Publisher, once:** make one Thought-engine build "approved": tag a commit and attach a prebuilt Windows binary to a GitHub Release on the private repo. Give your brother read access to `Arameair/Thought-engine` (A1 grant).
2. **Identity:** person signs in with GitHub (working now).
3. **Pair:** Processor shows "Not paired with Identity" → you type Identity's pairing code → Processor saves Identity's public key. Both sides log it.
4. **Request:** in Processor, *Install Thought Engine* → Processor asks Identity for an install decision.
5. **Identity decides:** checks (a) the person is signed in now, (b) GitHub confirms read access to `Arameair/Thought-engine`, (c) the approved release exists, then (d) signs a single-use decision naming that exact build, and downloads the release asset.
6. **Processor verifies:** signature, freshness, single use, that it's addressed to this Processor, and that the file's SHA-256 matches. Any failure → refuse, with the reason shown.
7. **Install + record:** copy to the features folder, write the install record, show "Installed — authorized by Identity (decision …)".
8. **Negative tests that must fail closed:** not signed in · signed in but no repo access (your brother before you add him) · Identity not paired · Identity offline · tampered binary · replayed decision · decision for a different Processor · expired decision.

What this proves for real: a person's install is authorized by a check GitHub enforces, decided by Identity, verifiably delivered to Processor, and recorded. What it does **not** prove (honest list): publisher approval beyond repo access (A2), per-person build choice, hosted request/approve (A3), starting the Engine (D2).

---

## 9. Decision sheet

| # | Question | Options | Decision |
| --- | --- | --- | --- |
| Q2 | What proves a person may have the Engine (v0)? | A1 repo access · A2 signed grant · A3 hosted service | |
| Q2b | If A1: write it down as an explicit temporary rule? | yes · no | |
| Q3 | How Processor trusts Identity | B1 pair + signatures · B2 Processor checks GitHub itself · B3 file drop | |
| Q4 | What gets installed | C1 build from source · C2 signed-off release binary | |
| Q4b | Who downloads | Identity (token stays there) · Processor | |
| Q4c | Which Engine build is "approved" first | branch/commit to tag | |
| Q5 | Revocation behavior | uninstall · refuse to start · warn | |
| Q6 | After install | D1 install only · D2 install + start decision | |
| — | Does the Engine need an `alpha` before this? | yes · tag on the experiment branch is enough | |
