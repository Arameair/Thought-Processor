# Discord mailbox contract v2

Status: proposed in [#127](https://github.com/Arameair/Thought-Processor/issues/127) (Discord
Milestone A). Implemented by `core/discord-door/door.py`. Nothing here enables a grant, installs a
service or sends a message by itself.

The **Discord door** is the only program that talks to the Communication Bridge. Core (TASS) never
holds the Bridge key or the bot token: it reads and writes plain files in the mailbox. The door
sends a reply only after Identity has redeemed a single-use decision for that exact reply.

## Dependencies

| Component | Required revision | Why |
| --- | --- | --- |
| Communication Bridge | [Communication-Bridge#3](https://github.com/Arameair/Communication-Bridge/pull/3) or later, including its durable source dedup (`sources.idx`) | The Bridge refuses to post a reply to any channel other than the conversation of the message it answers (`failed`, `not sent: destination …`), and it never hands over the same Discord message twice as a fresh message. The door does not rely on the second point: see **One Discord message, one answerable envelope**. |
| Identity | the `discord.reply` contract of [identity#46](https://github.com/Arameair/identity/issues/46), as implemented by [identity#47](https://github.com/Arameair/identity/pull/47) at its accepted head `a67064380cb55d1b55f9e8454cf2f60d12c6b306` | Core obtains the decision; only the door redeems it, once; `check` is refused for `discord.reply`. The contract passed independent review at that head. |

Neither dependency is merged, released or installed. The door's tests use fake Bridge and fake
Identity services only.

## Three different permissions

| | What it is | Who acts |
| --- | --- | --- |
| Owner grant `discord.reply` | TASS may reply in Discord at all. Off by default. Sends nothing by itself. | The owner, on Identity's page. |
| Single-use decision | Identity's answer for ONE exact reply. Expires after 120 s; redeemed at most once. | Core obtains it; the door redeems it just before posting. |
| Manual approval | The owner approving one specific message. | Not used for reactive replies. |

## Directories

| Path | Writer | Readers | Content |
| --- | --- | --- | --- |
| `<mailbox>/inbox/` | door | Core | one file per owner message: `<seq, 6 digits>-<bridge id>.json` |
| `<mailbox>/replies/` | Core | door | one file per reply: `reply-<bridge id>.json` |
| `<mailbox>/receipts/` | door | Core | one file per answered message: `<bridge id>.json` |
| `<mailbox>/rejected/` | door | Core | reply files refused before any Identity call |
| `<mailbox>/status.json` | door | Core | is the door ready to send, and why not |
| `<state>/` | door only | door | trusted inbound records, reply claims, the Discord source index, cursor, lock |

`<state>` must not be inside the mailbox, and the mailbox must not be inside `<state>`. Core must
not be able to write `<state>`: the door builds every binding from its own records there, never
from anything Core wrote.

Every file the door writes is written to a temporary name in the same directory, flushed, fsynced,
then renamed (the directory is fsynced too). Core must write reply files the same way: names
starting with `.` are ignored, and a reply file is read only once it is at least 0.5 s old.

The door refuses to start on a mailbox that has the v1 workspace door's files (`outbox/`, `sent/`,
`failed/`, `.door-state.json`). The v1 door posts anything it finds in `outbox/` without asking
Identity, so the two must never share a mailbox. v2 replies live in `replies/`, which v1 never reads.

## Inbox file (door → Core)

```json
{
  "schema": "tass.discord.inbox/2",
  "id": "m_4d442e5526b608cfe449",
  "seq": 4,
  "source_message_id": "1562034986897375235",
  "channel_id": "1100000000000000001",
  "sender_id": "2100000000000000001",
  "sender_kind": "owner",
  "conversation_id": "discord:1100000000000000001",
  "text": "Hello TASS",
  "source_time": "2026-10-20T09:31:24.451Z",
  "received_at": "2026-10-20T09:31:25.012309258Z",
  "catch_up": false,
  "answerable": true,
  "not_answerable_reason": ""
}
```

The door writes an inbox file only for an inbound Discord message that:

- comes from the configured owner (`DOOR_OWNER_ID`) in the configured channel (`DOOR_CHANNEL_ID`),
  with `sender_kind` `owner`, `kind` `message`, origin trace `["discord"]`, and
  `conversation_id` exactly `discord:<channel_id>`;
- has well-formed identifiers (`m_` + 20 hex for the Bridge id, digits for the Discord ids).

Any other inbound message is never written and never answered; it is counted in `status.json`.

A written message is `answerable: false` (with a reason) when it is a catch-up message (the
Bridge's `catch_up` flag, or it was received before this door process started), when it already
has a reply, when the Bridge was not in manual mode, or when its receive time cannot be read.
Core must not answer those, and the door refuses any reply to them.

The door writes each inbox file once (never rewritten, never with the bot's own messages).

## Reply file (Core → door)

`<mailbox>/replies/reply-<bridge id>.json`, with exactly these keys:

```json
{
  "schema": "tass.discord.reply/2",
  "reply_to": "m_4d442e5526b608cfe449",
  "text": "Hi! Here is my answer.",
  "cycle_id": "cy_0123abcd",
  "decision_id": "dec_9199becdea7d29ec"
}
```

- `reply_to` must equal the id in the file name and name a message from `inbox/`.
- `text` is posted exactly as written. The door never trims, splits, truncates or rewrites it, so
  it refuses text the Bridge would change or reject: empty; leading or trailing whitespace; more
  than 1,900 characters; control characters other than newline and tab; anything that is not
  valid UTF-8.
- `cycle_id` (1–80 of `A-Z a-z 0-9 . _ : -`) is echoed in the receipt.
- `decision_id` is the single-use decision Core obtained from Identity for this binding.

There are no new channel messages in v2: every post is a reply to one owner message.

## Binding

The binding is the lowercase hex SHA-256 of the UTF-8 bytes of these eight lines joined with `\n`
(no trailing newline). It is the same definition as `docs/DISCORD_REPLY_CONTRACT.md` in identity#46.

```
tass.discord.reply/1
discord-door
<bridge id>
<source_message_id>
<sender_id>
<channel_id>
discord:<channel_id>
<lowercase hex SHA-256 of the exact reply text bytes>
```

Core computes it from the inbox file and the text it writes. The door computes it again from its
own trusted record of that message and the text in the reply file. A different text, message,
sender or channel gives a different binding, and Identity refuses the redeem.

Test vector (fixture values): Bridge id `m_4d442e5526b608cfe449`, source message
`1562034986897375235`, sender `2100000000000000001`, channel `1100000000000000001`, text
`TEST FIXTURE reply: hello from TASS` → text hash
`412238e0e55005100cbdeb1137fcec8122648f95aa03fe30bce39fa946ca5a6d` → binding
`f5ab28d15e23a3850274a3604fb999d433bb3980d9e4ff970bec8029d095a25c`.

### Destination

Line 7 is the destination: the reply goes to the conversation of the message it answers. The door
checks that the trusted record's `conversation_id` is exactly `discord:<channel_id>` and that the
channel is the configured one. It sends through `POST /api/replies`, which posts a Discord reply
that references the original, and it checks that the Bridge's receipt names that same
conversation and the same text. With Communication-Bridge#3, the Bridge itself refuses to post if
its configured channel has changed since (`not sent: destination changed`).

## One Discord message, one answerable envelope

The Bridge identifies messages by its own Bridge ID. The same Discord message can come back under a
new Bridge ID (a redelivery, a reset Bridge journal, a Bridge bug), and a new Bridge ID would
otherwise get its own claim, its own decision and its own reply. So the door keeps a private,
durable **source index**:

- `<state>/sources.manifest` is the bound, append-only list of every claim the door has ever made:
  `claim <source key> <bridge id>` for each Discord source, `reply <bridge id>` for each reply claim
  (appended **before** `claims/<bridge id>.json` is created, so before any Identity redemption), and
  operator baseline lines. Each line is appended and fsynced first; nothing is pruned or rewritten.
- `<state>/sources/<source key>.json`, where the key is the sha256 of
  `"discord\n<channel id>\n<Discord message id>"`, names the one Bridge envelope (`bridge_id`) that
  owns that Discord message. File names hold no raw IDs, and the files hold no text.
- When a new envelope from the owner arrives, the door claims its source (manifest line, then the
  file, created exclusively) **before** it writes the trusted record. Only the envelope that
  claims it can be answerable. Any other envelope for the same channel and Discord message is
  recorded as `answerable: false` with `duplicate_of`, is **never written to `inbox/`**, and a
  reply naming it is rejected **before Identity is asked** and before `POST /api/replies`.
- An envelope that already owns its source but has no trusted record, and was not claimed by the
  running process (a crash between claim and record, or lost state), is recorded **held**:
  `answerable: false`, never answerable again. Its history can't be proven.
- The manifest, the claim files, the reply claims and the trusted records must agree. The door
  checks the source **when an envelope is received**, **before it claims a reply**, **immediately
  before the Identity redemption** and **immediately before posting**. Each check re-hashes the
  whole manifest against what the door wrote and proves its path still names the opened file.
  It also proves that the claim file exists and names the manifest's owner, and that a reply line
  has its reply claim.
- **Fail closed, never reconstruct from absence.** A missing, extra, altered or replaced claim file
  or reply claim, or a removed, replaced, truncated or rewritten manifest, is an **integrity
  failure**. The door stops hearing, authorizing and posting (`status.json` `problem` says why and
  what to do), and after a restart it refuses to start. An interrupted claim-file write is
  completed only inside the process that appended its manifest line, before its record exists
  (in-memory proof of that boundary). After a restart it is a refusal, because an interrupted
  write and deleted evidence of an earlier authorized reply look the same on disk.
- At startup every record, claim file, reply claim and manifest line is cross-checked. A state
  directory that holds records of an earlier run (`door.lock`, `cursor.json`, inbound records,
  reply claims or claim files) but no manifest is an upgrade or state loss, not a first start,
  and is refused.
- **Recovery.** Prefer restoring the state directory from a copy taken while the door was
  stopped; startup verifies it, but can't tell that copy is the latest. If evidence is lost,
  duplicate protection for it **can't be restored**. The operator may accept that explicitly with
  `door.py --accept-source-baseline --acknowledge-dedup-loss --discord-floor <ID>`, where ID is a
  Discord message posted in the bridged channel after the loss. Without both options it refuses
  and changes nothing. It deletes nothing and resumes nothing silently:
  - a lost reply claim is recorded as a final `not_sent` claim (permanent refusal);
  - a lost claim file is restored only for the owner the manifest names;
  - every record without a reply claim is **held**;
  - records with no source claim are claimed or demoted as duplicates;
  - it appends `baseline v1 <decision ms> <reason> discord-ms <floor>`. From then on every Discord
    message at or before the floor is refused as not provably new. The floor is the latest of the
    operator's message time (Discord's clock), this host's clock and any message the door still
    has a record of.

  Damaged or conflicting records are still refused. On an intact state the baseline changes
  nothing.
- If a claim file can't be written, the message isn't recorded and the feed is read again next
  tick (`status.json` shows the problem).

## How the door handles one reply file

1. **Check the file** (no network). Name, keys, schema, ids, text rules, the trusted record exists,
   is answerable and owns its Discord message in the source index, and no claim exists for that
   message. A failure moves the file to
   `rejected/` (no receipt, nothing sent).
2. **Wait until the Bridge is ready**: Discord connected, not paused, mode `manual`, the door not
   blocked, and the rate limit (3 s between posts, 20 per 10 minutes) allows it. The file waits in
   `replies/`. If it waits longer than 110 s (the decision would be close to expiring), it is
   rejected.
3. **Claim the message.** `<state>/claims/<bridge id>.json` is created exclusively. If it already
   exists the reply is refused: **one reply per owner message, ever.** The reply file is removed.
4. **Redeem with Identity**: `POST /v1/decisions/redeem` with the door's own key and
   `{"decision_id", "action": "discord.reply", "binding"}`. The door never sends `check`. It goes
   on only if the answer is HTTP 200 with `allow: true`, `mode: "redeem"`,
   `for_decision` equal to the decision id, `service: "discord-door"` and a non-empty
   `entity_id`. Anything else, including Identity being unreachable, is **not sent**.
5. **Record "redeemed"** durably, then post once: `POST /api/replies` with
   `Idempotency-Key: tass-door-<bridge id>`.
6. **Follow delivery** through the Bridge's message feed until it is final.

Nothing is ever retried: not after a refusal, not after a failure, not after an uncertain result,
not after a crash.

## Receipt file (door → Core)

`<mailbox>/receipts/<bridge id>.json`, rewritten (atomically) as the state changes:

```json
{
  "schema": "tass.discord.receipt/2",
  "reply_to": "m_4d442e5526b608cfe449",
  "cycle_id": "cy_0123abcd",
  "decision_id": "dec_9199becdea7d29ec",
  "reply_file": "reply-m_4d442e5526b608cfe449.json",
  "text_sha256": "…",
  "binding": "…",
  "state": "delivered",
  "detail": "",
  "bridge_message_id": "m_8c64ab6d17deffb03855",
  "discord_message_id": "1562035000000000001",
  "claimed_at": "…", "updated_at": "…"
}
```

| `state` | Meaning | Final? |
| --- | --- | --- |
| `claimed` | the message is claimed; Identity not asked yet | no |
| `redeemed` | Identity allowed it; posting now | no |
| `queued`, `sending` | the Bridge has it | no |
| `delivered` | posted; `discord_message_id` is set | yes |
| `not_sent` | definitely not posted (Identity refused, Bridge refused, `failed`, destination check) | yes |
| `uncertain` | it may have been posted; never retried | yes* |
| `answered_elsewhere` | someone else replied to that message first; the door will never reply | yes |

\* An `uncertain` that the door recorded itself (no answer or an unexpected answer from the Bridge
after posting, no final Bridge state within 120 s, or a crash after redeeming) is updated if the Bridge later reports that same reply as `delivered`, `failed`
or `uncertain`. That is a correction of the record, never a second post.

Receipts and `rejected/` files never contain message text.

After a crash: a claim still at `claimed` becomes `not_sent` (the door posts only after recording
`redeemed`); a claim at `redeemed` becomes `uncertain` and is matched to the Bridge's record of the
reply if one appears.

## Competing responders

- A lock in `<state>` stops a second door from using the same state.
- The door does not send while the Bridge is in echo mode, paused or disconnected.
- If the Bridge shows an API reply to a tracked owner message that this door did not create,
  another program is answering through the Bridge: the door stops sending (`blocked` in
  `status.json`) until it is restarted.
- A reply the owner types on the Bridge page marks that message `answered_elsewhere`; it does not
  block the door.

## Configuration

| Variable | Meaning |
| --- | --- |
| `DOOR_MAILBOX` | the mailbox directory |
| `DOOR_STATE` | the door's private state directory |
| `DOOR_BRIDGE`, `DOOR_KEY_FILE` | the Bridge URL and its API key file |
| `DOOR_IDENTITY`, `DOOR_IDENTITY_KEY_FILE` | Identity's URL and the door's own service key file |
| `DOOR_CHANNEL_ID`, `DOOR_OWNER_ID` | the one channel and the one owner the door accepts |
| `DOOR_FORWARD_PORT` | optional: forward the Bridge's own page (off when `0`, the default) |

The door refuses to start without valid channel and owner ids, with overlapping mailbox and state
directories, or on a v1 mailbox. Keys are read from files at use; they never appear in a file the
door writes, in `status.json` or in a log line. Log lines carry no message text.

## Out of scope

Core's adapter, the Compose overlay and launcher switch, proactive messages, and any live Discord
test. They need their own slices and owner approval.
