# Whisper `/find` — Accurate Target Location — Design

Version: v1
Status: Draft
Created: 2026-08-18
PRD: [prd.md](prd.md)

---

## 0. Reader's summary — this design contradicts the approved PRD

The PRD's §4.2/§5.1/§6.1/§7 build a **cash-shop presence store inside
`atlas-cashshop`**, and its FR-7 decides "offline" from a 404 on the
`atlas-maps` location endpoint. Investigation during design found that FR-7's
premise is **factually wrong**, and that once it is corrected the
`atlas-cashshop` store is no longer the cheapest — or even a sufficient —
answer.

**The wrong premise.** PRD FR-7 states: *"`atlas-maps` removes that row on
logout (`maps/kafka/consumer/character/consumer.go:182`, `ExitAll`)."*
Line 182 is inside `handleStatusEventDeletedFunc` — the character-**DELETED**
handler — and `mapcharacter.ProcessorImpl.ExitAll`
(`services/atlas-maps/atlas.com/maps/map/character/processor.go:66`) clears the
**in-memory per-map registry**, not the durable `character_locations` row.

The durable row is deleted from exactly one call site,
`services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go:172`,
inside that same DELETED handler. Nothing else in `atlas-maps` ever calls
`location.ProcessorImpl.Delete`.

What LOGOUT actually does is the opposite: `handleStatusEventLogoutFunc`
(`consumer.go:109-138`) resolves the forced-return field and calls
`lp.Set(event.CharacterId, resolved)` — it **persists** the last location so
that the next login can restore it.

Consequences for the PRD as written:

- `location.GetField` **never returns `ErrNotFound` for a logged-off
  character.** It returns their last-known world/channel/map. So FR-7 has no
  mechanism: a logged-off target still yields a channel result, which is the
  exact bug (`/find` says "channel 1") the task exists to remove. Building the
  `atlas-cashshop` store as specified would fix FR-4 and FR-6 and leave FR-7
  broken.
- The PRD's §7 rationale for a separate store — *"[`atlas-maps`] keeps removing
  the map location on cash-shop entry ... which is precisely what makes a
  separate presence store necessary"* — is also wrong for the durable row.
  `atlas-maps`'s cash-shop consumer
  (`kafka/consumer/cashshop/consumer.go:44-54`) calls `p.ExitAndEmit`, which
  reaches only `mapcharacter.Exit` (the in-memory registry) and a Kafka event.
  The `character_locations` row is untouched by cash-shop entry.

**What is missing is not a cash-shop store; it is a liveness signal.** No
service in the repo can currently answer "is character X online, and where".
`atlas-account` exposes no session-state read (`account/resource.go:32-40` is
CRUD plus attempt counters plus a session DELETE). `atlas-maps` holds live
per-map membership in memory but exposes it only map-first
(`map/resource.go:30-31`), never character-first. Whisper **chat** does not
need one because it is broadcast on a topic and each channel filters locally
(`atlas-messages/.../message/processor.go:98-131`).

**This design therefore places the state on the record that already exists**:
it adds a `state` discriminator to `atlas-maps`'s `character_locations` row.
That row is already written on LOGIN, LOGOUT, CHANNEL_CHANGED and CHANGE_MAP,
and `atlas-maps` already consumes the cash-shop status topic — so every
transition the PRD wanted in `atlas-cashshop` already has a handler in
`atlas-maps` that needs one extra line. `atlas-cashshop` is not touched at all.

Everything else in the PRD — the FR-1…FR-7 rule ordering, the cross-world gate,
GM concealment, the MTS-folds-into-cash-shop decision, both find arms, the
observability requirements, the acceptance criteria — is adopted unchanged.
Only the *source* of the cash-shop and offline facts moves.

§3 states the alternatives and why this one wins. If the reader prefers to hold
the PRD's shape, §3.4 records what that costs.

---

## 1. Architecture

### 1.1 Component map

```
atlas-channel  socket/handler/character_chat_whisper.go
                 └── findDecision(...)          pure, table-driven, unit-tested
                       ├── characterByNameFunc  → atlas-character (name → id, world, gm)
                       ├── localSessionFunc     → in-process session registry (+ CashScene)
                       └── characterLocationFunc→ atlas-maps GET /characters/{id}/location
                                                   now carrying `state`

atlas-maps     character/location/              +state column, +state transitions
                 kafka/consumer/character/       LOGIN / LOGOUT / CHANNEL_CHANGED
                 kafka/consumer/cashshop/        CHARACTER_ENTER / CHARACTER_EXIT

libs/atlas-constants/character/presence.go       shared PresenceState enum

libs/atlas-packet field/clientbound/whisper_test.go   fixtures only
```

`atlas-cashshop`, `atlas-mts`, `atlas-character`: **no change**.

### 1.2 The presence state

A three-valued discriminator on the location row, expressed as a string enum in
`libs/atlas-constants/character` (checked first per repo convention — no
existing equivalent; `libs/atlas-constants/character/` holds only
`temporary_stat.go` and `energy_charge.go`):

```go
package character

type PresenceState string

const (
    PresenceStateOffline    PresenceState = "OFFLINE"
    PresenceStateInField    PresenceState = "IN_FIELD"
    PresenceStateInCashShop PresenceState = "IN_CASH_SHOP"
)
```

It lives in `atlas-constants` rather than in either service because it crosses
the `atlas-maps` → `atlas-channel` REST boundary as a wire value; duplicating
the literals in two services is the drift this library exists to prevent.

`IN_CASH_SHOP` covers the MTS as well, per PRD FR-11 — the ITC renders inside
the cash-shop `CStage` and emits the identical `CHARACTER_ENTER` event
(`atlas-channel/.../socket/handler/mts_entry.go:103`), so `atlas-maps` cannot
distinguish them and does not need to.

### 1.3 State machine

| Event (topic) | Handler in `atlas-maps` today | New effect |
|---|---|---|
| `LOGIN` (`EVENT_TOPIC_CHARACTER_STATUS`) | `handleStatusEventLoginFunc`, `consumer.go:94` | also set `state = IN_FIELD` |
| `CHANNEL_CHANGED` (same) | `handleStatusEventChannelChangedFunc`, `consumer.go:141` | also set `state = IN_FIELD` |
| `CHANGE_MAP` (`COMMAND_TOPIC_...`) | `handleChangeMapFunc`, registered `consumer.go:69` | leaves `state` alone |
| `LOGOUT` (same status topic) | `handleStatusEventLogoutFunc`, `consumer.go:109` | also set `state = OFFLINE` (world/channel/map still persisted, unchanged) |
| `DELETED` (same status topic) | `handleStatusEventDeletedFunc`, `consumer.go:160` | row deleted — already correct |
| `CHARACTER_ENTER` (`EVENT_TOPIC_CASH_SHOP_STATUS`) | `cashshop/consumer.go:44` | also set `state = IN_CASH_SHOP` |
| `CHARACTER_EXIT` (same) | `cashshop/consumer.go:56` | also set `state = IN_FIELD` |

**`OFFLINE` is terminal except via LOGIN / CHANNEL_CHANGED.** The cash-shop
status topic and the character status topic are separate Kafka topics with no
mutual ordering guarantee, so a late-delivered `CHARACTER_EXIT` could otherwise
resurrect a logged-off character as `IN_FIELD`. The cash-shop transitions are
therefore conditional: they are applied only when the current state is not
`OFFLINE`. LOGIN and CHANNEL_CHANGED are unconditional, because they are the
only events that legitimately mean "this character is live right now".

This single rule replaces the PRD's whole OQ-1 staleness discussion (§3.5).

### 1.4 Migration and cold start

Additive: one nullable-then-defaulted column on `character_locations`, no new
table, no backfill, no changes to existing columns.

Existing rows adopt the zero value. **The zero value must be `OFFLINE`**, which
is a deliberate choice: at deploy time every row reads offline, so `/find`
answers "not findable" for anyone mid-session until their next LOGIN or channel
change. That is the honest-error direction the PRD asks for, it self-heals
within one login cycle, and it is strictly better than defaulting to `IN_FIELD`
— which would assert liveness for the (large) majority of rows belonging to
characters who are genuinely logged off.

### 1.5 REST surface

`GET /characters/{characterId}/location` gains one attribute. No new endpoint,
no new resource type, no new client package.

```json
{
  "data": {
    "type": "character-locations",
    "id": "12345",
    "attributes": {
      "worldId": 0,
      "channelId": 3,
      "mapId": 100000000,
      "instance": "00000000-0000-0000-0000-000000000000",
      "state": "IN_CASH_SHOP"
    }
  }
}
```

Adding a JSON key is backward compatible for the endpoint's existing consumers
(`atlas-channel` session bootstrap at `kafka/consumer/session/consumer.go:214`,
`atlas-character`'s logout path at `character/processor.go:558`,
`atlas-login`'s character-list writer) — they decode into structs that ignore
unknown fields and will simply not read `state`.

A response with an empty/absent `state` is treated by the channel as `OFFLINE`,
so a `atlas-maps` that has not yet been redeployed degrades `/find` to
"not findable" rather than to a fabricated channel.

**`ErrNotFound` keeps its current meaning** — no row at all, i.e. a character
who has never logged in. It is a distinct branch from `state == OFFLINE`, and
both map to the same wire shape (FR-7) with different log lines.

---

## 2. The `atlas-channel` handler

### 2.1 Decomposition

`produceFindResultBody` today is a single closure that constructs its
collaborators inline (`character.NewProcessor`, `session.NewProcessor`, the
package-level `location.ResolveMapId`). That is why none of FR-1…FR-7 is
testable. The rewrite splits it in two:

- **`findDecision`** — a pure function over three injected lookups, returning a
  `findOutcome` value (a discriminated result: error / cash-shop / map / channel,
  plus the branch name for logging). No writer, no session mutation, no
  packet types. This is where every FR-1…FR-7 test lands.
- **`produceFindResultBody`** — unchanged shape, now a thin adapter: pick the
  result mode byte, call `findDecision`, project the outcome onto the packet
  constructor, announce.

The three lookups are package-level `var …Func` seams, matching the pattern
already established in this exact package by `checkNameChangeValidityFunc`
(`socket/handler/cash_shop_check_name_change.go:26`, swapped in tests at
`cash_shop_check_name_change_test.go:108-117`). No new mocking framework, no
interface churn, no `*_testhelpers.go` file.

```go
var findCharacterByNameFunc = func(l, ctx, name string) (character.Model, error) { … }
var findLocalSessionFunc   = func(l, ctx, ch channel.Model, id uint32) (session.Model, error) { … }
var findCharacterLocationFunc = func(l, ctx, id uint32) (location.Model, error) { … }
```

`findCharacterLocationFunc` returns a `location.Model` (world, channel, map,
instance, state) rather than a `field.Model`, because `field.Model` has nowhere
to carry the state. `location.GetField` is kept as-is for its existing callers;
a sibling `location.Get` returning the richer model is added alongside it.

### 2.2 Decision table

Evaluated in order; first match wins. This is PRD §4.1 with the FR-4/FR-6/FR-7
sources corrected.

| # | Condition | Source | Outcome |
|---|---|---|---|
| FR-1 | name does not resolve | `findCharacterByNameFunc` error | error shape, requested name echoed |
| FR-2 | `target.WorldId() != s.Field().WorldId()` | resolved character | error shape |
| FR-3 | `target.GmLevel() > 0 && requester not GM` | resolved character + session | error shape |
| FR-4a | local session exists **and** `CashScene() != CashSceneNone` | in-process registry | cash-shop shape |
| FR-5 | local session exists **and** `CashScene() == CashSceneNone` | in-process registry + location row | map shape (`WithXY` on the `0x09` arm) |
| FR-4b | `location.State() == IN_CASH_SHOP` | location row | cash-shop shape |
| FR-6 | `location.State() == IN_FIELD` | location row | channel shape, `location.ChannelId()` |
| FR-7a | `location.State() == OFFLINE` (or empty) | location row | error shape |
| FR-7b | `ErrNotFound` — never logged in | location row | error shape |
| FR-7c | any other lookup error | infrastructure | error shape, logged at error level |

Notes on the ordering, which differs from the PRD's:

- **The local session is consulted before the location row**, so the
  requester's own channel is answered without trusting any remote state at all.
  A live local session is by construction authoritative about both liveness and
  cash-scene, and costs nothing.
- **FR-4b now sits *after* FR-5 rather than before it.** In the PRD, cash-shop
  presence was checked first because it lived in a different store that could
  disagree with the location row. Here they are the same row, so there is
  nothing to disagree and the natural read order applies.
- The `IN_FIELD` case at FR-6 is reached only when there is no local session,
  which — given `ByCharacterIdModelProvider` filters on world **and** channel
  (`session/processor.go:176-180`) — means the target is on some other channel.
  A target whose location row says `IN_FIELD` on the requester's *own* channel
  but who has no local session is a genuinely inconsistent state; it falls to
  FR-6 and reports the requester's own channel, which is harmless and honest.

### 2.3 Channel id conversion

PRD FR-6 verified against the codec: `WhisperFindResultChannel.Encode` writes
`w.WriteInt(m.channelId)` with no adjustment
(`libs/atlas-packet/field/clientbound/whisper.go:227`), and the client adds one
for display. `channel.Id` is already the 0-based internal value, so the
conversion is `uint32(loc.ChannelId())` and nothing else. The existing
hard-coded `0` is precisely why every off-channel target reads as "channel 1".

### 2.4 GM concealment (resolves OQ-3)

`atlas-channel`'s `character.Model` stores `gm int` and exposes only
`Gm() bool { return m.gm == 1 }` (`character/model.go:49,64`). A GM at level 2
therefore reads as **not** a GM — the concealment gate would leak exactly the
accounts it most needs to hide. A `GmLevel` concept does exist elsewhere in the
repo as an int (`atlas-query-aggregator/.../character/model.go:60`,
`libs/atlas-saga/validation.go:20`), confirming levels above 1 are meaningful.

This design therefore:

- adds `GmLevel() int` to `atlas-channel`'s `character.Model`, and
- changes `Gm()` to `m.gm > 0`.

The `Gm()` change is safe: it has three non-test callers
(`kafka/consumer/session/consumer.go:212`, `kafka/consumer/message/consumer.go:99`
and `:178`), all of which want "is this player a GM" for session flagging and
chat colouring — `> 0` is strictly more correct for each.

**OQ-3 is answered: GM visibility is a boolean predicate on `level > 0`, not a
tier comparison.** Any GM sees any GM; no GM is hidden from another GM. A tier
ladder would need a policy the repo does not have and no client shape expresses.

The requester's GM status is read from the session, as PRD FR-3 requires.
`session.Model` carries `gm bool`, set at login bootstrap from `c.Gm()`
(`kafka/consumer/session/consumer.go:212`), but has **no exported getter** —
only `setGm` (`session/model.go:103`). A `Gm() bool` accessor is added. Once
`character.Model.Gm()` is `> 0`, the session flag inherits the corrected
semantics for free.

### 2.5 Observability (FR-13)

`findOutcome` carries the branch name (`unresolved`, `cross-world`,
`gm-concealed`, `cash-shop-local`, `cash-shop-remote`, `map-local`,
`channel-remote`, `offline`, `never-logged-in`, `lookup-failed`). The adapter
emits one debug line per `/find` with requester id, target name, arm
(`0x09`/`0x48`) and branch — so the four wire-identical error branches are
separable in logs. FR-7c additionally logs at error level with the underlying
error attached.

### 2.6 Both arms (FR-12) and the IDA evidence (resolves OQ-2)

`findDecision` is arm-agnostic; the arm affects only the echoed mode byte and
whether the map shape carries x/y. That is not asserted by symmetry — it is
confirmed against the checked-in client decompilation
(`docs/packets/ida-exports/gms_v83.json`, `CField::OnWhisper`, `0x53228e`):

```
DecodeStr  guard: v120 != 0x22
Decode1    guard: v120 != 0x22            <- findMode
Decode4    guard: v120 != 0x22            <- the shape's payload
Decode4    guard: v120 != 0x22 && (v120 & 1) != 0 && v34 == 1   <- x
Decode4    guard: v120 != 0x22 && (v120 & 1) != 0 && v34 == 1   <- y
```

Reading this: the mode byte `v120` selects the arm and `v34` is the findMode
byte. The name / findMode / payload triple is read for **every** find arm — the
only excluded mode is `0x22`. The x/y pair is read only when the mode is odd
**and** findMode is 1.

- `0x09` is odd → x/y read on findMode 1. Matches `WhisperFindResultMapWithXY`.
- `0x48` is even → x/y never read. Matches `WhisperFindResultMap`.
- findMode `2` (cash shop) and `3` (channel) read the shared `Decode4` on
  **both** arms.

**OQ-2 is answered: yes.** The buddy-window arm accepts the cash-shop body
identically; FR-12 gains no arm-specific exception. `gms_v72` (`v93`) and
`gms_v79` (`v93`) carry the same guard structure.

**WITHDRAWN — flagged divergence on `gms_v92`/`gms_v95` (v1 of this design).**
v1 read `gms_v95.json`'s raw `CField::OnWhisper` guard `Decode4 | v3 == 72 &&
v28 == 1` as evidence that those two clients read x/y on the `0x48` arm, and
recorded it as a pre-existing packet gap. **That reading was wrong, and the
finding is withdrawn.** Re-derived during `/plan-task` against the live IDBs:

`CField::OnWhisper` @`0x5448a0` (v95, `GMS_v95.0_U_DEVM.exe`) and @`0x53e2a0`
(v92, `GMS_v92_1_DEVM.exe`) both have the shape:

```c
case 9:
case 72:
  DecodeStr(name); v28 = Decode1();  // findMode
  v29 = Decode4();                   // payload
  if ( (v3 & 1) == 0 )               // EVEN mode -> 0x48
  {
    if ( (v3 & 0x40) != 0 ) { switch (v28) { case 2: case 3: case 1: } }
    goto LABEL_125;                  // no further decode -- no x/y
  }
  switch ( v28 )                     // ODD mode -> 0x09
  {
    case 1:
      v44 = Decode4();               // x
      nTargetPosition_Y = Decode4(); // y
```

x/y is read **only** when the mode is odd (`0x09`) **and** findMode is 1 — the
same rule as v83/v84/v87, and exactly what Atlas already encodes. The raw guard
quoted in v1 is a harvesting artifact: the exporter collapsed the shared
`case 9: case 72:` label onto its last value and dropped the `(v3 & 1)` branch
that separates the two arms. The curated per-arm record in the same export
(`CField::OnWhisper#FindResultMap`: `x (mode 0x09 only)`, *"Wire
version-invariant"*) was correct all along.

**There is no version gap, no version gate to add, and no wire change to any
version.** `libs/atlas-packet` stays test-only, as §7 says.

The derivation did retire a separate blocker: `gms_v92.json`'s eight
`CField::OnWhisper#*` arm records were `"unresolved": true` (*"requires a
per-arm decompile pass against the v92 IDB"*), which is why the `gms_v92`
clientbound WHISPER matrix cell sits at `incomplete` — *"tier-1 without
fixture; verdict ❌"*. The read order is now derived, so the plan promotes that
cell (plan Task 9). The serverbound `chat/serverbound/ChatWhisper` × `gms_v92`
cell (*"no audit report"*) is a separate op and stays out of scope.

### 2.7 Instanced maps (resolves OQ-4)

`location.Model` carries an `Instance` uuid; `WhisperFindResultMap` carries only
a map id. A target inside an instance the requester cannot enter is reported by
map id regardless. **Decision: leave unchanged.** It is pre-existing behaviour,
the client has no shape to express "in an instance", and suppressing the answer
would trade a mild over-disclosure for a new class of "my friend is standing
right there and `/find` says they are not findable" report. Recorded as an
explicit decision rather than an accident, as the PRD asks.

---

## 3. Alternatives considered

### 3.1 Chosen — state column on `atlas-maps`'s `character_locations`

- **Cost:** one column, one shared enum, five one-line handler edits, one REST
  attribute, one field on the channel's `location` client. `atlas-cashshop`
  untouched.
- **Answers all of FR-4, FR-6, FR-7** from a single call the handler already
  makes.
- **Fewer moving parts on the `/find` path than today's broken version**: worst
  case two service calls (name, location) versus the PRD's three.
- **The events are already consumed.** `atlas-maps` subscribes to both the
  character status topic and the cash-shop status topic today; no new consumer,
  no new consumer group, no new topic subscription anywhere.
- **Domain fit:** "where is this character" is `atlas-maps`'s domain, and
  "in the cash shop" is a location answer. `atlas-maps` already imports the
  cash-shop status message type (`kafka/message/cashshop`), so no new
  cross-service concept is introduced.
- **Ordering is bounded** by the single `OFFLINE`-is-terminal rule (§1.3).
- **Risk:** `character_locations` becomes load-bearing for liveness, not only
  for last-known position. Mitigated by making `OFFLINE` the zero value, so any
  unwritten or stale row fails toward "not findable".

### 3.2 Rejected — the PRD's `atlas-cashshop` presence store

- Solves FR-4 only. **FR-7 remains unimplementable**, because nothing else
  supplies liveness — the branch would still have to guess, and guessing is the
  bug.
- Adds a table, a consumer on a topic `atlas-cashshop` does not currently
  subscribe to, a handler on its character consumer, a processor, a REST
  resource, and a client package in `atlas-channel` — to answer one bit that
  the location row can carry as one column.
- Adds a third service to the `/find` hot path and with it the PRD §8
  "`atlas-cashshop` unreachable" resilience requirement, which this design
  simply does not have to satisfy.
- Introduces a **second** authority on where a character is, which can disagree
  with the first. The PRD's own rule ordering (cash-shop before location) is a
  symptom: it exists to resolve a disagreement that a single row cannot have.

### 3.3 Rejected — a general presence service

Correct in the abstract and explicitly non-goaled by the PRD (§2). It is also
premature: exactly one consumer exists today. If buddy-list status, party
channel display, and cross-channel messaging later want the same fact, the
`state` column is the natural thing to promote — extracting it then will be a
better-informed decision than inventing the service now.

### 3.4 If the PRD's shape must be held

Should the reviewer prefer to keep the `atlas-cashshop` store as specified, the
FR-7 gap does not go away: `atlas-maps` still needs a liveness bit, so the state
column (or an equivalent) gets built anyway — and then the cash-shop half is
duplicated in two stores that can disagree. The honest version of "hold the PRD"
is therefore "build both", at roughly twice the surface for strictly worse
consistency. This design recommends against it.

### 3.5 OQ-1 is dissolved, not answered

The PRD's OQ-1 asked: GORM table or in-memory registry for presence, and what
TTL sweeps stale rows.

**There is no separate presence store, so the question does not arise.** For
the record, had one been built, in-memory would have been **incorrect**, not
merely lossy: `atlas-cashshop` consumes its topics in a Kafka consumer group, so
under more than one replica each `CHARACTER_ENTER` reaches exactly one replica
while the REST read lands on an arbitrary one. Presence would be visible only
by coincidence. A table was the only viable form, and its staleness question is
answered here by the `OFFLINE`-is-terminal rule plus the `OFFLINE` zero value —
no sweeper, no TTL, and therefore no invented timeout constant.

---

## 4. Data flow

**`/find` on a target in the cash shop on channel 5, requester on channel 2:**

```
client 0x09 ──▶ CharacterChatWhisperHandleFunc
                └─▶ findDecision
                     ├─ characterByName("Bob")        → id 42, world 0, gm 0
                     ├─ world match                    ✓
                     ├─ gm gate                        ✓ (target not GM)
                     ├─ localSession(ch=2, 42)         → miss
                     └─ characterLocation(42)          → {world 0, ch 5, map …, IN_CASH_SHOP}
                                                        → outcome{cashShop, "cash-shop-remote"}
                └─▶ NewWhisperFindResultCashShop(0x09, "Bob") ──▶ findMode 2, int32 -1
```

**How that row got there:**

```
player clicks cash shop
  └─ CashShopEntryHandleFunc  (atlas-channel, cash_shop_entry.go:127)
       ├─ cashshop.Processor.Enter → CHARACTER_ENTER on EVENT_TOPIC_CASH_SHOP_STATUS
       └─ session.SetCashScene(sid, CashSceneCashShop)   [local only]
                        │
                        ▼
  atlas-maps kafka/consumer/cashshop handleStatusEventEnter
       ├─ mapcharacter Exit  (in-memory registry)        [today]
       └─ location.SetState(42, IN_CASH_SHOP)            [new]
```

**Disconnect from inside the cash shop** — the path PRD FR-9 correctly
identifies as emitting no `CHARACTER_EXIT`:

```
socket closes → session timeout task → account session LOGOUT command
  → atlas-character emits LOGOUT (outbox, character/processor.go:579)
      → atlas-maps handleStatusEventLogoutFunc
           ├─ location.Set(resolved field)     [today — position preserved]
           └─ location.SetState(42, OFFLINE)   [new]
```

`/find` on that character now answers with the error shape. Under the PRD's
design this same scenario is what the mandatory-logout-transition FR-9 was
guarding against; here it is the same one transition, in a handler that already
exists.

---

## 5. Error handling

| Failure | Behaviour |
|---|---|
| name lookup fails (any cause) | error shape, debug log, branch `unresolved` — indistinguishable from FR-2/FR-3 on the wire, by design (PRD §8) |
| location lookup returns `ErrNotFound` | error shape, debug log, branch `never-logged-in` |
| location lookup returns 5xx / network / decode error | error shape, **error**-level log with the error attached, branch `lookup-failed` |
| location row present but `state` empty or unrecognised | treated as `OFFLINE` → error shape, warn log naming the value |
| `atlas-maps` entirely unreachable | every `/find` for an off-channel target answers "not findable"; same-channel `/find` still answers correctly from the local session, except that the map id falls back (see below) |

`location.ResolveMapId` — which collapses *every* failure to map id 0 — is
**not** used by the find path any more. Its collapse is what lets a transport
failure render as a real location today. It remains in the tree for its other
callers and is not modified.

For FR-5 (same channel, on a map) the map id still comes from the location row.
If that lookup fails while a live local session exists, the outcome is the error
shape rather than a map result with a fabricated map id — reporting "not
findable" for a player who is demonstrably online is wrong but recoverable;
reporting map 0 is a confidently wrong answer of exactly the kind this task
removes.

---

## 6. Testing

### 6.1 `atlas-channel` — decision table

`socket/handler/character_chat_whisper_test.go`, following the env-struct
pattern of `cash_shop_check_name_change_test.go`: a real session in the registry
(`session.AddSessionToRegistry` + `t.Cleanup(ClearRegistryForTenant)`), a
capturing `writer.Producer`, and the three `…Func` seams swapped and restored
via `t.Cleanup`.

Table-driven over `{FR-1…FR-7} × {0x09, 0x48}`, asserting the decoded body, not
just the constructor:

- FR-1 unresolvable → error shape, echoes the **requested** name.
- FR-2 cross-world → error shape **and** the `cross-world` log line, proving it
  is a distinct branch and not FR-1 by accident.
- FR-3 GM target / non-GM requester → error shape; GM requester → normal result;
  **plus a case at `gm == 2`**, which fails against today's `gm == 1` predicate.
- FR-4a local `CashSceneCashShop` and `CashSceneMts` → cash-shop shape, with the
  location seam asserted **not called**.
- FR-4b remote `IN_CASH_SHOP` → cash-shop shape.
- FR-5 local, `CashSceneNone` → `WithXY` on `0x09`, without on `0x48`, correct
  map id and coordinates.
- FR-6 `IN_FIELD` on channel **7** → channel shape carrying 7. Deliberately not
  0 or 1, so the old hard-coded `0` cannot pass (PRD acceptance criterion).
- FR-7a `OFFLINE`, FR-7b `ErrNotFound`, FR-7c 5xx → error shape; FR-7c asserts
  an error-level log entry.
- An arm-symmetry test asserting the `0x09` and `0x48` bodies are byte-identical
  for every outcome except the map shape's trailing x/y.

### 6.2 `atlas-maps` — transitions

Consumer-handler tests against a sqlite-backed `*gorm.DB`, per the service's
existing consumer tests:

- LOGIN → `IN_FIELD`; LOGOUT → `OFFLINE` with world/channel/map preserved.
- CHANNEL_CHANGED → `IN_FIELD` with the new channel.
- `CHARACTER_ENTER` → `IN_CASH_SHOP`; replayed duplicate enter is idempotent
  (at-least-once delivery, PRD §6.1).
- `CHARACTER_EXIT` → `IN_FIELD`.
- **`CHARACTER_EXIT` arriving after LOGOUT leaves the state `OFFLINE`** — the
  ordering rule of §1.3.
- DELETED → row gone.
- Tenant scoping: a row in tenant A is invisible to tenant B.
- A `GET /characters/{id}/location` round trip carrying `state`, and a decode of
  a `state`-less payload yielding `OFFLINE`.

### 6.3 `libs/atlas-packet` — fixtures

`field/clientbound/whisper_test.go` already round-trips all five find shapes and
pins byte output for v48/v61/v72/v79
(`TestWhisperVariantsByteOutputV79`, `…V72`, `TestWhisperByteOutputV61`,
`…V48`). The gap against the PRD's coverage criterion is the *cash-shop* and
*channel* shapes in the byte-output tests. Those are added; no struct or
encoding changes (PRD §2 non-goal, §7).

### 6.4 Gate

Flagless `tools/verify.sh` must exit 0, and code review runs before the PR
(CLAUDE.md).

---

## 7. Service impact

| Service / lib | Change |
|---|---|
| `atlas-maps` | `character/location`: `state` column + migration, model/builder/rest field, `SetState`; `kafka/consumer/character`: LOGIN / LOGOUT / CHANNEL_CHANGED set state; `kafka/consumer/cashshop`: ENTER / EXIT set state |
| `atlas-channel` | `socket/handler/character_chat_whisper.go` rewritten to §2.2 with three seams, both TODOs removed; `maps/location`: `Get` returning the state-bearing model + `state` on `RestModel`; `session/model.go`: `Gm()` accessor; `character/model.go`: `GmLevel()` and `Gm()` as `> 0` |
| `libs/atlas-constants` | new `character/presence.go` |
| `libs/atlas-packet` | tests only |
| `atlas-cashshop` | **none** (PRD §7 superseded) |
| `atlas-mts` | none |
| `atlas-character` | none |

---

## 8. Open items for the user

1. **The PRD reversal itself** (§0, §3). This design does not build the
   `atlas-cashshop` presence store the approved PRD specifies. The reasoning is
   the FR-7 premise error, evidenced at
   `atlas-maps/.../kafka/consumer/character/consumer.go:109-138` and `:172`.
   Confirm the reversal before `/plan-task`.
2. ~~**The v92/v95 buddy-window x/y divergence** (§2.6).~~ **RESOLVED during
   `/plan-task` — the divergence does not exist.** Re-derived against the live
   v92/v95 IDBs; both gate x/y on the odd-mode test exactly as v83/v84/v87 do,
   so Atlas's current encoding is correct. See §2.6 for the decompile. The
   derivation retired the `unresolved` v92 arm records, and the plan uses them
   to promote the `gms_v92` clientbound WHISPER matrix cell.
3. **The `Gm()` semantics change** (§2.4). Correct and small, but it touches two
   callers outside `/find` (GM chat colouring, session flagging). Confirm that
   widening `gm == 1` to `gm > 0` is intended repo-wide.
