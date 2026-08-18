# Evan Dragon Entity — Design

Version: v1
Status: Approved for planning
Created: 2026-08-13
PRD: [`prd.md`](./prd.md)
---

## 1. Scope of this document

The PRD left seven open questions, five of which gate the wire format and the
lifecycle model. All seven are resolved here from primary evidence — client
decompiles across five IDBs, the version-aware job tables, and the existing
summon paths in `atlas-channel` / `atlas-summons`. Section 2 records that
evidence; sections 3 onward are the design it supports.

Three decisions were escalated to the user because the evidence contradicted
PRD assumptions. Their resolutions are recorded in §4 and are binding.

---

## 2. Client evidence

Every claim in this section is a decompile of a named function in a specific
IDB. Nothing here is inferred from symbol names alone, per
[`IMPLEMENTING_A_PACKET.md`](../../packets/IMPLEMENTING_A_PACKET.md) §0.

### 2.1 The dragon family is dispatched by owner character id

`CUserPool::OnUserCommonPacket` (GMS v95.0 `@0x94cdb0`) decodes a `uint32`
**before** any family dispatch, resolves the `CUser` from it, and only then
routes:

```
v4 = CInPacket::Decode4(iPacket);
User = CUserPool::GetUser(this, v4);
...
if ( (v6 - 206) <= 2 )
    CUser::OnDragonPacket(p, v6, iPacket);
```

**OQ-2 resolved.** The dragon has no wire identity of its own. All three
clientbound dragon ops are addressed by the **owning character id**, consumed by
the pool before the per-op body. A dragon is therefore 1:1 with a `CUser`, which
independently confirms PRD FR-1.1.

### 2.2 `SPAWN_DRAGON` body — `CDragon::OnCreated`

GMS v95.0 `@0x50dc90`, decode sequence verbatim:

```
Decode4 -> m_ptPos.x        (int32)
Decode4 -> m_ptPos.y        (int32)
Decode1 -> m_nMoveAction    (byte; the stance)
Decode2 -> <discarded>      (return value never assigned)
Decode2 -> m_nJobCode       (uint16)
```

JMS185 `@0x52edd3` decodes the identical five fields in the identical order.
The two IDBs sit in the two distinct size classes present across the six
columns (`0x330`: v83 / v87 / JMS185; `0x464`: v92 / v95), so both classes are
covered by a real decompile.

**OQ-1 resolved for this op.** Note two things the PRD did not anticipate:

- `x` and `y` are **4 bytes each**, not the 2-byte coordinates used everywhere
  else in the protocol.
- There is a 2-byte field the client reads and throws away. It must still be
  written or every field after it misaligns. The design writes `0`.

`OnCreated` ends with `CDragon::ChaseTarget(this, 1, &owner->IVecCtrlOwner)` —
the dragon immediately begins chasing its owner. Spawn coordinates therefore
control one frame of rendering and nothing else. This is why a failed position
lookup is degraded-but-correct rather than fatal (§8.2).

### 2.3 `MOVE_DRAGON` clientbound — `CDragon::OnMove`

GMS v95.0 `@0x50ad30`:

```
CMovePath::OnMovePacket(&m_pvc[142], iPacket, 0);
```

The whole body is a `CMovePath` blob. Combined with §2.1 the wire is
`ownerCharacterId` + opaque move path — structurally identical to the summon
move relay already shipped in
`libs/atlas-packet/summon/clientbound/move.go`.

### 2.4 `REMOVE_DRAGON` is a dead opcode in all six clients

`CUser::OnDragonPacket` (GMS v95.0 `@0x8e5c00`):

```
if ( nType == 206 ) { ...release existing...; CDragon::OnCreated(...); }
else { p = this->m_pDragon.p; if ( p && nType == 207 ) CDragon::OnMove(p, iPacket); }
```

There is **no arm for 208**. The pool routes `206..208` into this function
(§2.1) and the third opcode falls through to nothing. The same shape is present
in every column examined:

| Version | IDB | `OnDragonPacket` | spawn | move | third op |
|---|---|---|---|---|---|
| v83 | `MapleStory_dump.exe` `@0x93908f` | 0x8e | `0xB5` | `0xB6` | `0xB7` — no arm |
| v84 | `GMS_v84.1_U_DEVM` | — | `0xB9` | `0xBA` | `0xBB` — no arm (already recorded in `docs/packets/registry/gms_v84.yaml`) |
| v87 | `GMSv87_4GB` `@0x9b3880` | 0x8e | `0xC2` | `0xC3` | `0xC4` — no arm |
| v92 | `GMS_v92_1_DEVM` `@0x8ce880` | 0xdc | `0xD1` (209) | `0xD2` (210) | `0xD3` — no arm |
| v95 | `GMS_v95.0_U_DEVM` `@0x8e5c00` | 0xdc | `0xCE` (206) | `0xCF` (207) | `0xD0` — no arm |
| JMS185 | `MapleStory_dump_SCY` `@0x9f822f` | 0x8e | `0xBB` | `0xBC` | `0xBD` — no arm |

An xref sweep on `ZRef<CDragon>::_ReleaseRaw` (v95 `@0x8decb0`) returns exactly
four callers: the `ZRef` destructor, `ZRef::operator=`, `OnDragonPacket`'s
respawn path, and `CUser::~CUser` `@0x8eb030`. **The only client-side dragon
teardown is destroying the `CUser`.**

Consequences, which shape §6.3:

- Sending `REMOVE_DRAGON` is a no-op the client silently discards. It is safe
  but it does not remove anything.
- Map change, channel change, and logout already remove the dragon client-side,
  because the viewer's client destroys the `CUser` when the character leaves the
  field. No dragon packet is involved.
- A dragon **cannot** be removed while its owner remains visible in the field.
  A job change out of the dragon-bearing range leaves the rendered dragon in
  place until the owner next leaves the field. This is a client limitation, not
  an implementation gap; §6.3 states the behaviour explicitly rather than
  pretending otherwise.

`OnDragonPacket`'s spawn arm releases any existing `m_pDragon` before
constructing the new one, so a duplicate `SPAWN_DRAGON` is idempotent
client-side. That is a safety net, not a licence to emit duplicates (§8.3).

### 2.5 `MOVE_DRAGON` serverbound — `CVecCtrlDragon::EndUpdateActive`

GMS v95.0 `@0x996570`:

```
COutPacket::COutPacket(&oPacketMove, 214);
if ( !CMovePath::Flush(p_m_path, &oPacketMove, 0, 0) )
    CClientSocket::SendPacket(..., &oPacketMove);
```

v83 `@0x9b7b9c` is byte-for-byte the same shape with `COutPacket(0xB5)`.

**The body is the `CMovePath` blob and nothing else — there is no leading
identity field.** This differs from the summon send
(`CVecCtrlSummoned::EndUpdateActive`), which writes `Encode4 summonId` before
the blob. The server must therefore resolve the dragon entirely from the
sending session's character id.

That has a pleasant consequence for PRD FR-4.4: "naming a dragon the submitter
does not own" is unrepresentable on the wire. The only validation left is "does
this sender have a dragon at all", which collapses FR-4.1 and FR-4.4 into one
check.

**OQ-5 resolved.** The handler consumes a `CMovePath` fragment list, so it is a
movement handler by the template guard's definition and gets the shared
`options.types` table — see §7.3. The codec itself treats the blob as opaque
and rebroadcasts it byte-faithfully, exactly as `summon/serverbound/move.go`
does today; `startX`/`startY` are lifted from the blob's first four bytes
(`CMovePath::Encode` leads with `Encode2 startX, Encode2 startY`) purely to
seed the persisted position.

### 2.6 Version gating: none required

All four layouts are uniform across v83, v84, v87, v92, v95 and JMS185. No
field is added, removed, or resized in any column.

**PRD FR-5.3 is satisfied vacuously** — there is nothing to gate, so no
`MajorAtLeast` call appears in these codecs and no raw `> N` comparison is
tempting. Per-cell verification (§9.1) still runs per version; if a column
surprises us, the gate goes in then, using `MajorAtLeast`.

### 2.7 Evan does not exist on v83

`libs/atlas-constants/job/version_gms_83_1_gen.go` contains **no** Evan entry —
neither `2001` nor `2200`–`2218`. `version_gms_84_1_gen.go:73,78-87` is the
first table with them. v84 is the earliest version on which an Evan character
can exist at all.

**OQ-4 resolved.** The v83 *client* fully supports dragons (§2.4), so the ops
are not absent from v83 and `n-a` would be a false claim. v83 is in scope for
codec, routing, and matrix verification; it is out of scope behaviourally
because no v83 character can hold a dragon-bearing job.

### 2.8 Growth-stage bound

`libs/atlas-constants/job/constants.go:166-176`:

```
EvanId       = Id(2001)   // beginner — no dragon
EvanStage1Id = Id(2200)
EvanStage2Id = Id(2210)
...
EvanStage10Id = Id(2218)
```

**OQ-3 resolved.** The dragon-bearing set is exactly the identities
`EvanStage1`…`EvanStage10`. The Evan *beginner* (2001) is excluded. The design
does **not** express this as a numeric range — see §5.2.

### 2.9 Character lifecycle events all exist

**OQ-6 resolved.** `EVENT_TOPIC_CHARACTER_STATUS` already carries every event
FR-1 needs, and `atlas-summons` already consumes three of them for its own
despawn cascade
(`services/atlas-summons/atlas.com/summons/kafka/consumer/character/consumer.go`):

| Event | Body type | Carries |
|---|---|---|
| `LOGIN` | `StatusEventLoginBody` | channelId, mapId, instance |
| `LOGOUT` | `StatusEventLogoutBody` | channelId, mapId, instance |
| `MAP_CHANGED` | `MapChangedBody` | channelId, oldMapId, oldInstance, targetMapId, targetInstance |
| `CHANNEL_CHANGED` | `ChannelChangedBody` | channelId, oldChannelId, mapId, instance |
| `JOB_CHANGED` | `JobChangedStatusEventBody` | channelId, **jobId** |

No producer change is required in `atlas-character`.

None of the five bodies carries `x`/`y`, and only `JOB_CHANGED` carries the job
id. `atlas-dragons` therefore fetches the character on spawn (§5.4). Per §2.2
that fetch is not on any critical correctness path.

---

## 3. Resolved open questions — summary

| OQ | Resolution | Evidence |
|---|---|---|
| OQ-1 field layouts | All four derived; see §2.2–§2.5 | v95 + JMS185 + v83 decompiles |
| OQ-2 dragon wire identity | Owner character id, consumed by `CUserPool` | §2.1 |
| OQ-3 growth-stage bound | `EvanStage1`…`EvanStage10` identities; beginner 2001 excluded | §2.8 |
| OQ-4 v83 applicability | Packets in scope, behaviour not — Evan absent from the v83 job table | §2.7 |
| OQ-5 movement-table handler | Yes — `CMovePath` fragment list; gets `options.types` | §2.5, §7.3 |
| OQ-6 character events | All five exist; no producer change needed | §2.9 |
| OQ-7 cross-service seam | Covered by §9.3's contract + seam tests | §9.3 |

Two PRD statements are corrected by this evidence and the design follows the
corrected version:

- **PRD FR-2.3** describes an "in-memory registry" and calls it a mirror of
  `atlas-summons`. `atlas-summons` is in fact **Redis-backed** via
  `libs/atlas-redis` (`summon/registry.go:158-178`), with no database. §5.3
  follows the real `atlas-summons` shape, not the PRD's description of it.
- **PRD FR-5.3** mandates `MajorAtLeast` gating for divergent fields. Nothing
  diverges (§2.6), so no gate is written.

---

## 4. Decisions taken

These three were escalated because the evidence cut against the PRD. All are
binding on the plan.

**D-1 — `atlas-dragons` is built as the PRD specifies.** The alternative
considered was implementing the whole feature inside `atlas-channel` with no
state at all: §2.1–§2.5 show a dragon is a pure function of (owner, job,
position), and `atlas-channel` already holds all three at its character-spawn
choke point. That alternative was **rejected**; the service is built. The design
below is therefore state-owning, and §5.3's registry is the authority for
"which dragons are in this field", rather than deriving it from character models
the channel happens to have loaded.

**D-2 — `REMOVE_DRAGON` is implemented, verified, and sent on destroy.**
Despite §2.4 showing the client discards it. PRD FR-3.3 stands as written. §6.3
records what the packet does and does not accomplish so nobody later mistakes it
for the mechanism that removes the dragon.

**D-3 — v83 is routed and verified with no behaviour.** All 24 matrix cells,
all six templates. Per §2.7, no v83 dragon will ever spawn, and that falls out
for free from §5.2's resolver-based job check rather than needing a version
special-case.

---

## 5. `atlas-dragons`

### 5.1 Module shape

Modelled directly on `atlas-summons`, which is the closest existing analogue: a
field-scoped, owner-keyed, database-free entity service.

```
services/atlas-dragons/atlas.com/dragons/
  main.go
  dragon/
    model.go        immutable Model + getters
    builder.go      Builder
    registry.go     Redis-backed registry (atlasredis)
    processor.go    Interface + Impl, NewProcessor(l, ctx)
    producer.go     EVENT_TOPIC_DRAGON_STATUS emitters
    rest.go         JSON:API transport model
    resource.go     GetName() == "dragons"
    kafka.go        topic/type constants, command + event bodies
  character/
    model.go        the slice of character we need
    requests.go     atlas-character REST client
    processor.go
  kafka/
    consumer/
      character/    LOGIN, LOGOUT, MAP_CHANGED, CHANNEL_CHANGED, JOB_CHANGED
      dragon/       COMMAND_TOPIC_DRAGON
  rest/handler.go
```

### 5.2 The dragon-bearing job predicate

A single exported predicate, used by every lifecycle path, expressed through the
**version-aware resolver** rather than a numeric range:

```go
// HasDragon reports whether the wire job id resolves, on this tenant's client
// version, to an Evan growth stage (EvanStage1..EvanStage10). The Evan beginner
// (2001) is excluded: CDragon is created at the first growth stage.
func HasDragon(t tenant.Model, wireJobId job.Id) bool {
    id, ok := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion()).Job.Resolve(wireJobId)
    if !ok {
        return false
    }
    return id >= job.EvanStage1 && id <= job.EvanStage10
}
```

Three properties this buys, none of which a `2200 <= id <= 2218` range check
would:

1. **v83 falls out for free.** `Resolve` fails on a v83 tenant because the v83
   table has no Evan entry (§2.7), so `HasDragon` returns false without a
   version special-case anywhere in the lifecycle code.
2. **`tools/skill-job-id-guard.sh` compliance.** `job.EvanId` is on the guard's
   banned-comparison list. The predicate never touches it, and it compares
   `Identity` values rather than wire ids.
3. The comparison operates on resolved identities, so a future version that
   remaps 22xx cannot silently break it.

### 5.3 Registry

Redis-backed via `libs/atlas-redis`, mirroring `summon/registry.go`. Not a
`sync.Map`: `atlas-dragons` may run more than one replica, and a plain in-process
map loses every dragon on restart while giving each replica a different answer
to `GET /dragons`.

```go
type storedDragon struct {
    TenantId, TenantRegion               string
    TenantMajorVersion, TenantMinorVersion uint16
    OwnerCharacterId uint32
    WorldId, ChannelId byte
    MapId    uint32
    Instance string
    X, Y     int32   // int32 — SPAWN_DRAGON encodes 4-byte coords (§2.2)
    Stance   byte
    JobId    uint16
}

type Registry struct {
    reg      *atlasredis.Registry[string, storedDragon]  // key: <tenant>:<characterId>
    fieldIdx *atlasredis.KeyedSet[string]                // key: <tenant>:<w>:<c>:<map>:<instance>
}
```

There is no owner index and no id allocator — unlike summons, the owner
character id **is** the primary key (§2.1), which makes FR-1.1 ("at most one
dragon per character") a property of the key space rather than something to
enforce.

`x`/`y` are `int32` to match the wire (§2.2), not the `int16` used by
`atlas-summons`.

All keyed Redis access goes through `libs/atlas-redis`; nothing touches the raw
`go-redis` client (`tools/redis-key-guard.sh`).

### 5.4 Processor

```go
type Processor interface {
    GetByCharacterId(characterId uint32) (Model, error)
    GetInField(f field.Model) ([]Model, error)

    Create(f field.Model, characterId uint32) error          // + AndEmit variant
    Destroy(characterId uint32) error
    Move(characterId uint32, startX, startY int16, stance byte, rawMovement []byte) error
}
```

`Create` fetches the character from `atlas-character` for `jobId`, `x`, `y`:

- **Job gate.** If `HasDragon` is false, `Create` is a no-op returning nil. This
  is the one place the predicate is enforced, so no caller has to remember it.
- **Fetch failure.** A `404` means the character is gone; no-op, `Warn`, no
  error. Any other error is returned and the Kafka handler lets the message
  retry.
- Position is used verbatim for the dragon's spawn coordinates. Per §2.2 the
  client re-chases the owner immediately, so a stale position costs one frame.

`Destroy` on a character with no dragon is a no-op returning nil, not an error
(FR-1.6).

`Move` writes position/stance and emits `MOVED` carrying the raw blob. It does
not create a dragon as a side effect (FR-4.4): if the registry has no entry for
the sender, the command is dropped with a `Warn` and no event.

### 5.5 REST

JSON:API via api2go, resource type `dragons`, registered with
`RegisterHandler(l)(si)`.

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/dragons?filter[worldId]=&filter[channelId]=&filter[mapId]=&filter[instance]=` | dragons in a field (FR-3.2) |
| `GET` | `/dragons/{characterId}` | one character's dragon |

Tenant-scoped through `tenant.MustFromContext(ctx)`. No `POST`/`DELETE` — every
mutation is a consequence of a lifecycle event or a Kafka command.

`GET /dragons/{characterId}` returns **404 for a character with no dragon**.
This is the normal, expected answer for the overwhelming majority of characters,
not an error. Every consumer of this endpoint must treat `ErrNotFound` as "no
dragon" and continue; a consumer that logs it as a fetch failure will emit
one error line per non-Evan character in the game. The channel-side client
(§6.2) does this explicitly, and it is a test case, not a comment.

### 5.6 Kafka contracts

Both topics carry tenant headers. `TenantHeaderDecorator` has silently dropped
headers before, so header propagation is asserted in a test (§9.3), not assumed.

**`COMMAND_TOPIC_DRAGON`**

| Type | Body |
|---|---|
| `CREATE` | `{ worldId, channelId, mapId, instance, characterId }` |
| `DESTROY` | `{ characterId }` |
| `MOVE` | `{ characterId, startX, startY, stance, rawMovement []byte }` |

**`EVENT_TOPIC_DRAGON_STATUS`**

| Type | Body |
|---|---|
| `CREATED` | `{ ownerCharacterId, x int32, y int32, stance, jobId }` + envelope field |
| `DESTROYED` | `{ ownerCharacterId }` + envelope field |
| `MOVED` | `{ ownerCharacterId, rawMovement []byte }` + envelope field |

The envelope follows the project shape used by summons —
`{ transactionId, worldId, channelId, mapId, instance, type, body }` — so
`atlas-channel`'s handlers can run the standard
`sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId)` guard before doing
any work.

`MOVED` carries the raw blob and no coordinates: the blob is what other clients
render, and the stored position exists only so a late-entering viewer gets a
sane first frame.

---

## 6. `atlas-channel`

### 6.1 New surface

```
socket/writer/dragon.go              DragonSpawnBody / DragonMoveBody / DragonRemoveBody
socket/handler/dragon_move.go        DragonMoveHandleFunc
dragon/{model,requests,processor,producer}.go   REST client + COMMAND_TOPIC_DRAGON producer
kafka/consumer/dragon/consumer.go    EVENT_TOPIC_DRAGON_STATUS -> broadcasts
kafka/message/dragon/kafka.go        contract mirror
```

This is a one-for-one shadow of the existing `summon` surface in the same
service, so the shapes are already established.

### 6.2 Field entry (FR-3.2)

`SpawnForSelf` in `kafka/consumer/map/consumer.go` is the path that renders an
entering session's view of the field, and it already fans out other characters'
pets from the same place. Dragons are added as a sibling fan-out inside the same
`routine.Go` block that handles pets:

```
GET /dragons?filter[...]=<the field>   -> []dragon.Model
for each dragon whose owner != s.CharacterId():
    Announce DragonSpawnWriter(DragonSpawnBody(...))(s)
```

One field-scoped query per entry, not one per character — this keeps the
entering path off an N+1. A query failure is logged and skipped; it degrades the
entering player's view of other Evans, and must not abort the rest of
`SpawnForSelf`.

Dragons belonging to characters the entering session cannot see are still
skipped for free, because the fan-out iterates the same `cms` character map used
for pets, intersected with the dragon query result. That inherits the GM-hide
suppression already enforced at `spawnCharacterForSession`
(`kafka/consumer/map/consumer.go:482`) without duplicating the check.

The owner's own dragon is **not** spawned here — it arrives via the `CREATED`
event broadcast (§6.3), which is map-wide and includes the owner.

### 6.3 Broadcast handlers

`kafka/consumer/dragon/consumer.go`, mirroring
`kafka/consumer/summon/consumer.go`:

| Event | Fan-out | Writer |
|---|---|---|
| `CREATED` | `ForSessionsInMap` — **including** the owner (FR-3.1) | `DragonSpawnWriter` |
| `MOVED` | `ForOtherSessionsInMap(f, ownerCharacterId)` — **excluding** the owner (FR-4.3) | `DragonMoveWriter` |
| `DESTROYED` | `ForSessionsInMap` (FR-3.3, per D-2) | `DragonRemoveWriter` |

The owner is excluded from `MOVED` because their client already rendered the
motion locally; re-sending double-applies it. This is the same reasoning as the
summon move relay's comment at `consumer/summon/consumer.go:105-106`.

**What `DESTROYED` actually accomplishes.** Per §2.4 the client discards
`REMOVE_DRAGON`. The dragon disappears from other players' screens because the
owner's `CUser` is destroyed when they leave the field — which map change,
channel change, and logout all do through paths that already exist. The
`DESTROYED` broadcast is emitted because D-2 says so, and it is correct to send,
but it is not the mechanism. Two behaviours follow and are **expected, not
bugs**:

- A job change out of the dragon-bearing range clears the dragon server-side
  and stops all dragon traffic, but the rendered dragon persists on every
  client in the field until the owner next leaves it.
- The PRD acceptance criterion "logging out produces `REMOVE_DRAGON` for
  remaining players in the map" is verifiable as *packet emitted*, not as
  *dragon visually removed by that packet* — the removal is already handled by
  the character-removal path and happens either way.

### 6.4 Movement handler

`DragonMoveHandleFunc` decodes `dragon/serverbound.Move` and emits a
`COMMAND_TOPIC_DRAGON` `MOVE` keyed on `s.CharacterId()`. Per §2.5 the packet
carries no identity, so the session **is** the identity; there is no id to
reconcile and no cross-character spoofing surface.

The relay round-trips channel → Kafka → `atlas-dragons` → Kafka → channel,
identical to the summon move path shipped today. PRD §8's performance
requirement is "no per-move REST call to another service", which this satisfies;
it is not a requirement for a channel-local relay.

---

## 7. Packet codecs

### 7.1 Package layout

```
libs/atlas-packet/dragon/clientbound/spawn.go   + spawn_test.go
libs/atlas-packet/dragon/clientbound/move.go    + move_test.go
libs/atlas-packet/dragon/clientbound/remove.go  + remove_test.go
libs/atlas-packet/dragon/serverbound/move.go    + move_test.go
```

Immutable structs, private fields, getters, `New…` constructors, **both**
`Encode` and `Decode` on each, and a `packet-audit:fname` marker naming the
client function the layout came from.

### 7.2 Wire formats

**`DragonSpawn`** — `packet-audit:fname CDragon::OnCreated`

| Field | Type | Note |
|---|---|---|
| ownerCharacterId | `uint32` | consumed by `CUserPool` (§2.1) |
| x | `int32` | 4 bytes, not 2 |
| y | `int32` | 4 bytes, not 2 |
| stance | `byte` | `m_nMoveAction` |
| — | `uint16` | client decodes and discards; write `0` |
| jobId | `uint16` | `m_nJobCode` |

**`DragonMove`** (clientbound) — `packet-audit:fname CDragon::OnMove`

| Field | Type |
|---|---|
| ownerCharacterId | `uint32` |
| rawMovement | opaque `[]byte`, byte-faithful |

**`DragonRemove`** — `packet-audit:fname CUser::OnDragonPacket`

| Field | Type |
|---|---|
| ownerCharacterId | `uint32` |

No body. The struct's doc comment records §2.4 — the opcode is routed but has no
handler arm in any of the six clients — so a future reader does not "fix" the
apparently-missing body.

**`Move`** (serverbound) — `packet-audit:fname CVecCtrlDragon::EndUpdateActive`

| Field | Type |
|---|---|
| rawMovement | opaque `[]byte` (the whole body) |

`startX`/`startY` are derived accessors reading the blob's first four bytes, as
`summon/serverbound/move.go` does. **No leading identity field** — the doc
comment must say so explicitly, because every sibling move packet in the codebase
has one and its absence looks like a bug.

### 7.3 Opcodes and template routing

Writers (clientbound):

| Op | v83 | v84 | v87 | v92 | v95 | JMS185 |
|---|---|---|---|---|---|---|
| `SPAWN_DRAGON` | `0x0B5` | `0x0B9` | `0x0C2` | `0x0D1` | `0x0CE` | `0x0BB` |
| `MOVE_DRAGON` | `0x0B6` | `0x0BA` | `0x0C3` | `0x0D2` | `0x0CF` | `0x0BC` |
| `REMOVE_DRAGON` | `0x0B7` | `0x0BB` | `0x0C4` | `0x0D3` | `0x0D0` | `0x0BD` |

Handler (serverbound):

| Op | v83 | v84 | v87 | v92 | v95 | JMS185 |
|---|---|---|---|---|---|---|
| `MOVE_DRAGON` | `0x0B5` | `0x0BA` | `0x0C1` | `0x0D3` | `0x0D6` | `0x0B9` |

Six templates change: `template_gms_83_1.json`, `_84_`, `_87_`, `_92_`, `_95_`,
`template_jms_185_1.json`. `template_gms_12_1.json`, `_48_`, `_61_`, `_72_`,
`_79_` are untouched — those columns are `⬜` and have no opcode assignment.

Per-entry requirements:

- Every entry inserted at its **sorted `opCode` position**, never appended next
  to a semantically-related neighbour (`template-opcode-order-guard.sh`).
- Every writer carries an `fname` (§7.2's four names).
- The handler carries `"validator": "LoggedInValidator"` — a handler with a
  missing validator is silently dropped at load.
- No leading-zero-padded duplicate bindings
  (`template-duplicate-binding-guard.sh`).

**Movement types table (FR-6.7).** `DragonMoveHandle` carries an
`options.types` array byte-identical to the `SummonMoveHandle` entry in the same
template, and `DragonMoveHandle` is added to `MOVE_HANDLERS` in
`tools/template-movement-types-guard.sh`. The codec treats the blob as opaque so
the table is currently unread — exactly as it is for `SummonMoveHandle` today.
It is carried anyway, and the guard extended, so that the six templates stay
uniform and any future drift is caught mechanically rather than discovered as a
"Code [N] not configured for use in movement" spew.

**Live tenant reconciliation (FR-6.8).** Updating a seed template does not
update a provisioned tenant. Every live tenant socket configuration must be
reconciled to the updated templates before behavioural verification; an opcode
present only in the template is silently dropped at runtime and the feature
simply does not work with a clean server log.

---

## 8. Cross-cutting concerns

### 8.1 Multi-tenancy

Every registry key is prefixed with the tenant id (§5.3), every REST call
resolves through `tenant.MustFromContext(ctx)`, and every broadcast passes the
`sc.Is(tenant, worldId, channelId)` guard before touching a session. The field
index key includes the instance uuid, so instanced fields never leak into the
base map's dragon list.

### 8.2 Concurrency

`ForEachInMap` / `ForSessionsInMap` in `atlas-channel` run their handlers
**in parallel**. No dragon broadcast path may accumulate into shared mutable
state across sessions. All three handlers in §6.3 build their `packet.Encode`
once and hand the same immutable closure to each session, which is safe. The
registry is Redis-backed and needs no in-process locking. All goroutines go
through `routine.Go` (`tools/goroutine-guard.sh`).

### 8.3 Idempotency

Kafka is at-least-once, and this feature has already been bitten elsewhere by
redelivery through non-idempotent handlers.

- **`CREATE`** — the registry key is `(tenant, characterId)`, so a redelivered
  create overwrites rather than duplicating. But a naive implementation would
  still emit a second `CREATED` event and a second map-wide `SPAWN_DRAGON`.
  `Create` therefore emits `CREATED` **only on the absent→present transition**,
  read back from the registry write. The client's own release-then-recreate
  (§2.4) is a second line of defence, not the first.
- **`DESTROY`** — a destroy for a character with no dragon is a silent no-op and
  emits no `DESTROYED`.
- **`MOVE`** — position writes are last-write-wins and carry no accumulation, so
  redelivery is naturally idempotent. The relayed blob may be re-broadcast; the
  client re-applies the same path and lands in the same place.

### 8.4 Constants and wire values

`world.Id`, `channel.Id`, `_map.Id`, `field.Model`, `job.Id`, `job.Identity`
come from `libs/atlas-constants`; no parallel local types (DOM-21). Every opcode
is resolved from the tenant socket configuration, never hard-coded (DOM-25) —
§7.3's table is documentation of what the templates must contain, not a
constant in Go.

### 8.5 Observability

Create, destroy, and move log at their decision points with tenant, character
id, and field. Loki queries use `service_name`, not `app` — an `app=` selector
returns zero rows silently.

---

## 9. Testing

### 9.1 Packet cells

24 cells (4 ops × 6 versions), each promoted per
[`VERIFYING_A_PACKET.md`](../../packets/audits/VERIFYING_A_PACKET.md): read
order derived from that version's client, a byte-fixture test with a
`packet-audit:verify` marker, and a pinned evidence record — the three artifacts
committed together.

The §2 decompiles cover v83, v87, v92, v95 and JMS185 for the dispatcher and two
of the six columns for `OnCreated`'s body. That is enough to design against; it
is **not** a substitute for per-cell verification. A round-trip fixture through
our own encoder proves nothing about the client and does not promote a cell.

`REMOVE_DRAGON`'s cells verify a 4-byte body and the routed-but-unhandled
dispatch. The evidence record must state the absence of the handler arm, so the
cell is not later re-opened as an incomplete verification.

Declare a `dragon` family in `docs/packets/feature-families.yaml` covering the
four ops. All four are `⬜` uniformly across v48/v61/v72/v79, so no
n-a-versus-verified-sibling inconsistency arises and no
`feature-na-evidence.yaml` entry is needed.

### 9.2 Service unit tests

`atlas-dragons`: the `HasDragon` predicate across every Evan stage plus the
beginner plus a non-Evan job plus a **v83 tenant** (which must return false);
registry round-trip; create/destroy idempotency; the `Move`-without-a-dragon
rejection. Models are built through the Builder — no `*_testhelpers.go`.

### 9.3 Cross-service seam (OQ-7)

Green unit tests in `atlas-dragons` prove nothing about `atlas-channel`'s
consumer, and a stubbed seam is zero coverage. Three specific tests close it:

1. **Contract mirror.** `atlas-channel`'s `kafka/message/dragon/kafka.go` is a
   mirror of the `atlas-dragons` contract across a module boundary — a renamed
   json tag in one and not the other fails no build and decodes into a
   zero-valued body at runtime. A test unmarshals a producer-serialised event
   into the consumer-side struct and asserts every field survives, for all three
   event types.
2. **Tenant headers.** A test asserts the produced message carries the tenant
   headers the consumer's `TenantHeaderParser` requires. This has failed
   silently before.
3. **Broadcast fan-out.** Table-driven tests over the three handlers in §6.3
   asserting who receives what: `CREATED` reaches the owner, `MOVED` does not,
   `DESTROYED` reaches everyone, and a mismatched world/channel reaches nobody.

### 9.4 Live behaviour

The PRD's behavioural criteria are verified on a live tenant at **v84 or above**
(§2.7 — a v83 tenant cannot produce an Evan), with the tenant socket
configuration reconciled to the updated templates first (§7.3). The map-change
criterion is checked as "exactly one `SPAWN_DRAGON` in the new map, no orphan in
the old" — the old map's cleanup is the character-removal path, per §6.3.

---

## 10. Service registration

`atlas-dragons` is a REST service with no database. Per
[`adding-a-new-service.md`](../../adding-a-new-service.md), in full:

- `.github/config/services.json` — the single source of truth driving CI and
  `docker-bake.hcl`
- `go.work`
- repo-root `Dockerfile` — only if a new shared lib is introduced (none is)
- `deploy/k8s/base/atlas-dragons.yaml`, with `REDIS_URL` from the shared
  `atlas-env` configMap
- `deploy/k8s/overlays/main/` **and** `deploy/k8s/overlays/pr/` — both, with the
  image pinned to a commit sha, never `:latest`
- Ingress (it serves REST)
- **Not** `tools/db-bootstrap.sh` — no database, no migration
- GHCR package visibility flipped to public after the first publish. A private
  package produces an anonymous-pull `401` and an `ImagePullBackOff` while CI
  reports green; publishing visibility is data-driven, so this is not a CI bug.
- `tools/service-registration-guard.sh` clean

---

## 11. Risks

| Risk | Mitigation |
|---|---|
| `OnCreated`'s discarded `uint16` is written wrong or omitted | Omitting it misaligns `jobId` and every subsequent read. Byte-fixture tests per version catch it; §7.2 records it explicitly. |
| `x`/`y` written as `int16` by habit | Same misalignment class. The stored model uses `int32` end to end (§5.3) so the narrow type never enters the pipeline. |
| Serverbound codec grows a leading id by analogy with summons | §2.5 and the codec's doc comment call the absence out; a byte fixture from the real send site fails immediately if one is added. |
| Live tenant configs not reconciled → silent no-op | §7.3; verification happens only after reconciliation. |
| `GET /dragons/{characterId}` 404 treated as an error | §5.5 makes it the normal answer and a test case. |
| Job change leaves a rendered dragon behind | §6.3 — a client limitation, documented as expected behaviour, not chased as a bug. |
| A second `CREATED` on Kafka redelivery double-broadcasts | §8.3 — emit only on the absent→present transition. |

---

## 12. Verification gates

Beyond the standard build gates in `CLAUDE.md`:

- `go test -race ./...`, `go vet ./...`, `go build ./...` in every changed module
- `docker buildx bake atlas-dragons` from the worktree root — mandatory, since a
  new service means a new `go.mod`
- `tools/service-registration-guard.sh`
- `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`,
  `tools/skill-job-id-guard.sh`
- `tools/template-opcode-order-guard.sh`,
  `tools/template-duplicate-binding-guard.sh`,
  `tools/template-movement-types-guard.sh` — the last one after
  `DragonMoveHandle` is added to its `MOVE_HANDLERS` set
- `tools/lint.sh --check`
- `packet-audit matrix --check` with 24 cells promoted and no previously-`✅`
  cell regressed
- `superpowers:requesting-code-review` before the PR
