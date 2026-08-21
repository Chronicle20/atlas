# Inner-Portal Registration (`USE_INNER_PORTAL`) — Design

Version: v1
Status: Draft
Created: 2026-08-21
PRD: [prd.md](prd.md)

---

## 0. What this document settles

The PRD left six open questions. Five are answered here from the client
binaries and repository source; the sixth (the ⬜ columns) is answered as a
*procedure* because the IDBs involved are not yet adopted.

| PRD Q | Answer | Evidence |
|---|---|---|
| 9.1 field list | `fieldKey:byte, portalName:string, x:i16, y:i16, targetX:i16, targetY:i16` | §1 |
| 9.2 movement entry point | No discrete position-set entry exists; add `TeleportCharacter` to `movement.Processor` — §4 | §4 |
| 9.3 threshold constant vs config | Package constant, value **derived** from the client's collision rect — §5.4 | §5.4 |
| 9.4 ⬜ columns | v12 is a confirmed absent (`0x000` sentinel in the CSV); v48/61/72/79 have no CSV column *and* no registry entry — absence is inferred, not derived. Re-derivation procedure in §7.2 | §7.2 |
| 9.5 broadcast to other players | Not needed. The client sets `CVecCtrl::SetMovePathAttribute(pvc, 4)` after registering, so its next `MOVE` carries a `TELEPORT` element that the existing `ForCharacter` path already folds and relays | §1, §4.3 |
| 9.6 JMS185 divergence | None. Byte-identical field order on v87, v95 and JMS185 | §1.3 |

---

## 1. Derivation — what `CUserLocal::TryRegisterTeleport` actually writes

### 1.1 The send site

Decompiled from the GMS v95.0 IDB (`GMS_v95.0_U_DEVM.exe.i64`),
`?TryRegisterTeleport@CUserLocal@@IAEHPBUSKILLENTRY@@JPBD1H@Z` at **0x913690**:

```c
int CUserLocal::TryRegisterTeleport(CUserLocal *this, SKILLENTRY *pSkill, int nSLV,
                                    const char *sPortalName,
                                    const char *sTargetPortalName, int bForced)
...
  PortalByName = CPortalList::FindPortalByName(ms_pInstance, sTargetPortalName);
  pfh = PortalByName;
  if ( !PortalByName ) return 0;
  v15 = PortalByName->ptPos.x;
  v16 = PortalByName->ptPos.y - 10;
  if ( !CWvsPhysicalSpace2D::GetFootholdUnderneath(x, v15, v16, 0, 0x7FFFFFFF, 1) ) return 0;
  if ( sPortalName )
  {
    COutPacket::COutPacket(&rc, 113);            // opcode 0x071 (gms_v95)
    COutPacket::Encode1(&rc, get_field()->m_bFieldKey);
    ZXString<char>::ZXString<char>(v74, sPortalName, -1);
    COutPacket::EncodeStr(&rc, v74[0]);
    COutPacket::Encode2(&rc, this->GetPos()->x);
    COutPacket::Encode2(&rc, this->GetPos()->y);
    COutPacket::Encode2(&rc, *(pfh + 12));       // target PORTAL->ptPos.x
    COutPacket::Encode2(&rc, *(v21 + 16));       // target PORTAL->ptPos.y
    CClientSocket::SendPacket(ms_pInstance, &rc);
  }
```

The function has two disjoint branches. When `sTargetPortalName == NULL` it is
the Mage/Evan **Teleport skill** path and sends `SendSkillUseRequest`, *not*
this opcode. Only the portal branch reaches the `COutPacket` above. This
matters for §5: the packet is unambiguously an inner-portal event.

### 1.2 The field list

| # | Field | Width | Source in the client | Meaning |
|---|---|---|---|---|
| 1 | `fieldKey` | `byte` | `get_field()->m_bFieldKey` | Per-field replay key, as in `portal/serverbound/Script` |
| 2 | `portalName` | ASCII string | argument `sPortalName` | The **source** portal's own name (`pn`) |
| 3 | `x` | `int16` | `GetPos().x` | Character position **before** the teleport |
| 4 | `y` | `int16` | `GetPos().y` | Character position **before** the teleport |
| 5 | `targetX` | `int16` | destination `PORTAL->ptPos.x` | Destination portal's coordinates, read from the *client's* WZ |
| 6 | `targetY` | `int16` | destination `PORTAL->ptPos.y` | Destination portal's coordinates, read from the *client's* WZ |

Field 2 is the source, not the destination. The caller supplies both names
separately — see §1.4.

Note fields 3/4: the client does **not** send a claimed destination. It sends
where it was standing plus what its own data says the destination is. That
shape is what makes §5's validation cheap and strong.

### 1.3 Per-version confirmation

| Version | Function | Opcode at the ctor | Encode sequence |
|---|---|---|---|
| gms_v95 | `0x913690` (named) | `push 113` (`0x071`) | `Encode1, EncodeStr, Encode2 ×4` |
| gms_v87 | `0x9da037` (named) | at `0x9da1c0` | `Encode1, EncodeStr, Encode2 ×4` |
| jms_v185 | `0xa2218f` (named) | `push 60h` at `0xa22313` (corrected during Task 12 promotion — this row originally read `0xa2230e`; the opcode, 96 / `0x060`, is unaffected) | `Encode1, EncodeStr, Encode2 ×4` |

All three are byte-identical. **No `MajorAtLeast` gate is required** (PRD
FR-2.3 is satisfied vacuously). If the v83/v84/v92 derivation in §7.1 finds a
delta, the gate is added then — not pre-emptively.

The v84 and v92 IDBs carry **no `TryRegisterTeleport` symbol**
(`func_query "*TryRegisterTeleport*"` returns empty in both). Those two, plus
v83, must be located by the caller-walk in §7.1.

### 1.4 The caller — and the invariant the client itself enforces

`CUserLocal::CheckPortal_Collision` (gms_v95 `0x919a10` — the same address
already pinned as the `portal/serverbound/Script` evidence) calls it at
`0x919d92`:

```asm
0x919d5e   cmp   dword ptr [esi+1Ch], 3B9AC9FFh   ; portal->tm != 999999999
0x919d65   jz    loc_919E34
0x919d6b   call  get_field
0x919d72   lea   ecx, [edi+16Ch]
0x919d78   call  TSecType<unsigned long>::GetData   ; current field id
0x919d7d   cmp   eax, [esi+1Ch]                   ; portal->tm == current map?
0x919d80   jnz   short loc_919DD0
0x919d82   mov   eax, [esi+20h]                   ; portal->tn
0x919d85   mov   esi, [esi+4]                     ; portal->pn
0x919d88   push  1          ; bForced
0x919d8a   push  eax        ; sTargetPortalName = portal->tn
0x919d8b   push  esi        ; sPortalName      = portal->pn
0x919d8c   push  0          ; nSLV
0x919d8e   push  0          ; pSkill
0x919d92   call  TryRegisterTeleport
```

Two things fall out:

1. `sPortalName` is the source portal's `pn`; `sTargetPortalName` is its `tn`.
   The server can therefore resolve the source portal by name and read its own
   `target` (`tn`) to find the destination — **it never needs a coordinate from
   the packet** (PRD FR-4.5).
2. The client's own guard is `tm != 999999999 && tm == currentFieldId`. That is
   exactly the assertion PRD FR-4.3 asks the server to re-make. `999999999` is
   already modelled in the repo as `map.EmptyMapId` with an `Id.IsSentinel()`
   helper (`libs/atlas-constants/map/model.go:39`) — reuse it, do not
   re-declare the literal.

`CUserLocal::HandleUpKeyDown` (`0x919e50`) is the second caller — the up-key
route into the same portal — and reaches the same send site.
`CUserLocal::MoveToPortal` and `DoActiveSkill` pass `sPortalName == NULL` or
`sTargetPortalName == NULL` and send nothing.

### 1.5 Step 0 — this is not a wrapper over an existing decoder

PRD FR-1.3 requires the "already implemented?" check before writing a codec.

`portal/serverbound/Script` decodes **four** fields
(`fieldKey, portalName, x, y` — `libs/atlas-packet/portal/serverbound/script.go`).
`USE_INNER_PORTAL` carries **six**. They share a prefix and a source function
family, but they are distinct opcodes with distinct layouts, and `Script` is
already verified on five versions — reshaping it would violate FR-6.3. A new
struct is required.

---

## 2. Codec

**Location:** `libs/atlas-packet/portal/serverbound/inner_portal.go`
**Type:** `InnerPortal` — immutable, unexported fields, value-receiver
accessors, `Operation() string`, `String()`, both `Encode` and `Decode`.
**Handle:** `const InnerPortalHandle = "InnerPortalHandle"` in the same package.
**Marker:** `// packet-audit:fname CUserLocal::TryRegisterTeleport`.

```go
type InnerPortal struct {
    fieldKey   byte
    portalName string
    x          int16
    y          int16
    targetX    int16
    targetY    int16
}
```

`Decode` reads exactly those six in order and nothing else (FR-2.4). `Encode`
mirrors it textually — the same discipline the movement codec comments call
out (`libs/atlas-packet/model/movement.go`), because Atlas encodes its own
fixtures.

**Naming rationale.** `InnerPortal` over `Teleport`: the file lives beside
`Script` in `portal/serverbound`, `Teleport` collides conceptually with the
Mage skill and with `model.TeleportElement`, and the matrix row is
`USE_INNER_PORTAL`. The matrix codec identifier becomes
`portal/serverbound/InnerPortal`.

**Accessors.** `FieldKey()`, `PortalName()`, `X()`, `Y()`, `TargetX()`,
`TargetY()`. Named `TargetX`/`TargetY` rather than `DestX`/`DestY` so the
"client-asserted, not adopted" nature stays legible at every call site.

---

## 3. Where the validation lives

Three placements were considered.

**(a) All in the handler.** Precedent exists — `teleport_rock_use.go` does its
ownership and range guards inline with package-var test seams. Cheapest to
write. Rejected: this handler needs two portal lookups, a sentinel check, a
same-map assertion, two distance comparisons and a position publish. That is a
processor's worth of logic sitting in a socket callback, and testing it means
constructing a `session.Model`.

**(b) A new `innerportal` package.** Rejected as a package for one function
that would immediately need `data/portal` and `movement` anyway.

**(c) [chosen] Extend `atlas-channel/portal.Processor`.** The package already
owns "a character interacted with a portal" — `Enter`, `Warp`,
`WarpToPosition`, `WarpToPortal` — and already holds a `data/portal.Processor`.
Add:

```go
EnterInner(f field.Model, characterId uint32, sourcePortalName string,
           claimedX int16, claimedY int16,
           claimedTargetX int16, claimedTargetY int16) error
```

The handler stays three lines plus the debug log, exactly like
`PortalScriptHandleFunc`. The validation is unit-testable against the portal
and movement mocks that already exist (`data/portal/mock`, `movement/mock`).

New dependency edge: `portal` → `movement`. `movement` does not import
`portal`, so no cycle. `movement` is already imported by `socket/handler`, so
nothing new enters the channel binary.

---

## 4. Position registration

### 4.1 The existing path

`movement.Processor.ForCharacter(f, characterId, model.Movement)`
(`services/atlas-channel/atlas.com/channel/movement/processor.go:59`) does two
independent things in two goroutines:

1. Broadcasts `charpkt.CharacterMovement` to **other** sessions in the map.
2. Folds the movement path to `(x, y, fh, stance)` and publishes
   `COMMAND_TOPIC_CHARACTER_MOVEMENT` via `CommandProducer`.

`atlas-character` consumes that topic
(`kafka/consumer/character/consumer.go:409`) and calls
`character.Processor.Move(objectId, x, y, fh, stance)`, which writes the
Redis-backed temporal registry (`character/temporal_data.go:78`). **That
registry is the server's position authority.** So "register through the same
processor `character_move.go` uses" (FR-5.1) concretely means "publish the same
Kafka command".

### 4.2 Options

**(A) Synthesise a `model.Movement` with one `TeleportElement` and call
`ForCharacter`.** Rejected on two counts. It re-broadcasts a clientbound
movement packet, which FR-5.3 forbids. And it requires fabricating `fh` and
`bMoveAction` values we do not have, i.e. inventing wire content.

**(B) [chosen] Add `TeleportCharacter` to `movement.Processor`.**

```go
// TeleportCharacter publishes an authoritative position for a character that
// relocated without a movement path — an inner portal. It emits the SAME
// COMMAND_TOPIC_CHARACTER_MOVEMENT command ForCharacter emits, so
// atlas-character remains the single position authority, and it emits NO
// clientbound broadcast: the client performed the teleport locally and its
// next MOVE carries the TELEPORT element that relays it to the field.
TeleportCharacter(f field.Model, characterId uint32, x int16, y int16) error
```

Body: one `routine.Go` publishing
`CommandProducer(f, uint64(characterId), characterId, x, y, 0, 0)`. No new
event type is introduced, satisfying FR-5.2 as written.

### 4.3 The foothold problem — and the one cross-service change this needs

`CommandProducer` requires an `fh`. Portal data carries none
(`data/portal/model.go` is `id, name, target, portalType, x, y, targetMapId,
scriptName`), and inventing one is out of the question. `fh = 0` is the honest
value.

But `temporalRegistry.Update` writes `fh` **unconditionally**
(`character/temporal_data.go:78-80`), so a `0` would clobber a real foothold.

The channel's own fold already has the correct rule and states it in a comment:

> `Fh is preserved across mid-air frames … we copy v.Fh from those, but only
> when non-zero so we don't trample the spawn-time fh`
> — `movement/processor.go:288-294`

**Decision:** apply the same rule at the consumer. In
`handleMovementEvent`, when `c.Fh == 0`, preserve the stored foothold — the
read-modify-write shape already used by `temporalRegistry.UpdateStance`
(`temporal_data.go:82-85`). Concretely: add
`temporalRegistry.UpdatePosition(ctx, t, characterId, x, y, stance)` and have
`character.Processor.Move` route to it when `fh == 0`.

**This adds `services/atlas-character` to the PRD's Service Impact table.** The
PRD did not anticipate it because the foothold coupling is only visible from
the consumer side. It is one small, semantics-preserving change that makes
`Fh == 0` mean "no foothold information" on both sides of the topic instead of
only one — and it is a prerequisite this task can produce itself, so it is done
here rather than deferred. A test asserting that a zero-`fh` movement command
leaves the stored foothold intact ships with it.

### 4.4 Why no clientbound broadcast (PRD 9.5)

The last thing the accepted branch of `TryRegisterTeleport` does is
`CVecCtrl::SetMovePathAttribute(pvc, 4)`. The next movement path the client
flushes therefore carries a `TELEPORT` fragment. `libs/atlas-packet/model`
already decodes `TeleportElement` (`movement.go:175, 217`), and the channel's
folder already applies it to `X`, `Y`, `Stance` and non-zero `Fh`
(`movement/processor.go:308-318`). Other clients in the field learn the new
position through that ordinary `MOVE`, exactly as they do today.

So `USE_INNER_PORTAL` is not what makes remote clients correct — it is what
makes the *server* correct immediately, and what gives the server a chance to
object. The design does not duplicate the relay.

---

## 5. Validation

`EnterInner` runs these in order. Every refusal is `return`-with-log and **no
position change**; none disconnects, kicks or repositions (FR-4.6).

### 5.1 Resolve the source portal

`data/portal.GetInMapByName(f.MapId(), sourcePortalName)`.
Unresolvable → `Warnf` with character id, field, and the unresolved name;
refuse (FR-4.2).

### 5.2 Assert the inner-portal invariant

Refuse unless `pm.TargetMapId().IsSentinel() == false && pm.TargetMapId() ==
f.MapId()` — the client's own guard from §1.4, re-made server-side (FR-4.3).
Use `map.Id.IsSentinel()`; do not introduce a `999999999` literal.

Also refuse when `pm.Target() == ""` — a portal with no `tn` has no
destination and cannot be an inner portal.

### 5.3 Resolve the destination portal

`data/portal.GetInMapByName(f.MapId(), pm.Target())`. Unresolvable → refuse and
log. Its `(x, y)` is the **only** coordinate the server will adopt (FR-4.5).

### 5.4 Plausibility

Two independent comparisons, both against server data:

**(i) Entry proximity.** `dist(packet(x,y), sourcePortal(x,y)) <=
maxPortalEntryDistance`. The client only fires this on a collision between the
avatar's rect and the portal's rect, so a legitimate value is bounded by that
rect plus one frame of movement latency.

**(ii) Destination agreement.** `packet(targetX,targetY) ==
destinationPortal(x,y)`, exact. Both sides read the same WZ, so a legitimate
client always matches. A mismatch is either a crafted client or a
server/client data divergence — both worth refusing and logging loudly.

Check (ii) is the stronger one and costs nothing, which is why it is here even
though the PRD only asked for (i).

**Deriving the threshold (PRD 9.3).** A package constant
`maxPortalEntryDistance` in `atlas-channel/portal`, unit = map coordinate
units, documented. Its **value is derived, not chosen**: read the portal
collision rect half-extents out of `CUserLocal::CheckPortal_Collision`
(gms_v95 `0x919a10`) and set the constant to that plus a stated
movement-latency margin, with the derivation recorded in
`structures/gms_v95.md`. No tenant variance is known, so it stays a constant;
promoting it to configuration later is a one-line change at one call site.

Squared-distance comparison against a squared constant — no `math.Sqrt` on the
socket path.

### 5.5 Last-known position (PRD FR-4.4)

FR-4.4 asks for the character's **last known position**. `atlas-channel` does
not have one: `session.Model` carries `id, accountId, characterId, field, gm,
storageNpcId, cashScene, …` and no coordinates, and
`character.Processor.GetById` is a REST call into `atlas-character` — a
per-packet remote fetch on the socket read path, which NFR-Performance
forbids.

It is cheap to have one, though, because the channel already computes it.
`ForCharacter` folds every `MOVE` to an authoritative `(x, y)` before
publishing it. **Decision:** add a process-local last-position registry in the
`movement` package — tenant + character id → `(x, y)`, written by
`ForCharacter` and by `TeleportCharacter`, read by `EnterInner`. Precedent:
`monster.GetLiveMirror()` (`movement/processor.go:145-166`),
`chakra.GetRegistry()`, `session` registry.

Then FR-4.4 is satisfied literally: check (i) in §5.4 runs against the
**server's** last known position rather than the packet's claim, and the
packet's `(x, y)` is logged and compared but never trusted.

**On a registry miss** — fresh login, map change, channel restart — skip the
last-known-position leg and fall through to the packet-claim comparison. A
miss must never refuse: a legitimate player's first action after entering a map
can be an inner portal, and refusing there would break the "always accepted"
user story for no security gain.

### 5.6 Accept

`movement.TeleportCharacter(f, characterId, destPortal.X(), destPortal.Y())`,
plus a `Debugf`. Accepted moves stay at debug so a player pacing through a
portal does not flood the log (NFR-Observability).

---

## 6. Portal data caching

`data/portal` has **no cache**: every `GetInMapByName` is a REST GET to
`atlas-data` (`data/portal/requests.go`). `EnterInner` needs two lookups per
packet, and inner portals fire in bursts. Shipping as-is would violate
NFR-Performance outright.

`data/map` already solves this for the same class of data — static WZ content —
with a tenant-scoped `sync.Map` plus a per-key load mutex, cached for the
process lifetime, and says why in a comment
(`data/map/processor.go:34-73`).

**Decision:** mirror that in `data/portal`, keyed by `{tenantId, mapId}`,
caching the **whole portal list** for a map. `atlas-data` already serves the
unfiltered list — `GET /maps/{mapId}/portals`
(`services/atlas-data/atlas.com/data/map/resource.go:36`) — so one fetch per
map per tenant replaces every by-name round trip, and name filtering happens
locally.

The `Processor` interface is unchanged; `GetInMapByName` filters the cached
slice. `portal.Enter` (`CHANGE_MAP_SPECIAL`) picks up the same benefit.

**Risk accepted:** portals become process-lifetime cached, identical to maps.
If WZ data changes, the pod restarts — the same contract `data/map` already
lives under.

---

## 7. Version coverage

### 7.1 In scope

| Template | Version | Opcode | Registry |
|---|---|---|---|
| `template_gms_83_1.json` | gms_v83 | `0x065` | `registry/gms_v83.yaml:2454` |
| `template_gms_84_1.json` | gms_v84 | `0x065` | `registry/gms_v84.yaml:3142` |
| `template_gms_87_1.json` | gms_v87 | `0x068` | `registry/gms_v87.yaml:2570` |
| `template_gms_92_1.json` | gms_v92 | `0x070` | `registry/gms_v92.yaml:2790` |
| `template_gms_95_1.json` | gms_v95 | `0x071` | `registry/gms_v95.yaml:2825` |
| `template_jms_185_1.json` | jms_v185 | `0x060` | `registry/jms_v185.yaml:2547` |

Template entry shape (from `template_gms_83_1.json:884-891`):

```json
{
  "opCode": "0x65",
  "validator": "LoggedInValidator",
  "handler": "InnerPortalHandle",
  "fname": "CUserLocal::TryRegisterTeleport",
  "services": ["channel"]
}
```

`LoggedInValidator`, matching the neighbouring `PortalScriptHandle` entry.

**v83, v84 and v92 need the caller-walk.** Their IDBs have no
`TryRegisterTeleport` symbol. Procedure: locate `FindPortalByName` (or
`CheckPortal_Collision`), follow the call that takes five arguments ending in
`bForced = 1`, and read the constant pushed into the `COutPacket` constructor.
Confirm it equals the registry value above; a disagreement is a finding, not
something to reconcile silently.

### 7.2 Re-deriving the ⬜ columns (PRD FR-6.2, Q9.4)

The current ⬜ is *inferred from absence*, and the two kinds of absence differ:

- **gms_v12** — the CSV has a `GMS v12` column and it holds `0x000`
  (`docs/packets/MapleStory Ops - ServerBound.csv:237`), the absent sentinel.
  That is a positive record of absence. `template_gms_12_1.json` gets no route.
- **gms_v48, v61, v72, v79** — no CSV column *at all*, and no registry entry.
  Nothing has been derived. Their support docs say `n-a` with an empty
  evidence cell (`audits/support/gms_v48.md:798` and siblings).

Procedure for each of the four: adopt the IDB via `idb_open` (all four are
running unadopted GUI instances — `GMS_v48_1_DEVM.exe.i64`,
`GMS_v61.1_U_DEVM.exe.i64`, `GMS_v72.1_U_DEVM.exe.i64`,
`GMS_v79_1_DEVM.exe.i64`), `func_query "*TryRegisterTeleport*"`, and on a miss
run the §7.1 caller-walk. Record the outcome per version in
`docs/tasks/task-250-inner-portal-registration/version-coverage.md`. If an
opcode exists, that version joins §7.1 in full — registry entry, template
route, codec coverage, fixture.

The CSV also carries a `GMS v111` column (`0x0B1`) with no matrix column. Out
of scope; noted so the next reader does not re-discover it as a gap.

---

## 8. Files

| Path | Change |
|---|---|
| `libs/atlas-packet/portal/serverbound/inner_portal.go` | New `InnerPortal` codec + `InnerPortalHandle` |
| `libs/atlas-packet/portal/serverbound/inner_portal_test.go` | Round-trip + golden-byte tests, `packet-audit:verify` markers |
| `services/atlas-channel/.../socket/handler/portal_inner.go` | `InnerPortalHandleFunc` — decode, log, delegate |
| `services/atlas-channel/.../main.go` | `handlerMap[portal2.InnerPortalHandle] = handler.InnerPortalHandleFunc`, beside line 901 |
| `services/atlas-channel/.../portal/processor.go` | `EnterInner` + `maxPortalEntryDistance` |
| `services/atlas-channel/.../data/portal/processor.go` | Per-map tenant-scoped cache |
| `services/atlas-channel/.../movement/processor.go` | `TeleportCharacter` + last-position registry write |
| `services/atlas-channel/.../movement/position.go` | Last-position registry |
| `services/atlas-character/.../character/temporal_data.go` | `UpdatePosition` preserving `fh` |
| `services/atlas-character/.../character/processor.go` | `Move` routes to it when `fh == 0` |
| `services/atlas-configurations/seed-data/templates/*.json` | Six routes (§7.1) |
| `docs/packets/audits/` | Regenerated `STATUS.md` / `status.json`, evidence records |
| `docs/tasks/task-250-.../structures/*.md`, `version-coverage.md`, `coverage-manifest.yaml` | Derivation, ⬜ findings, claimed cells |

Mocks under `data/portal/mock` and `movement/mock` extend with the new methods.

---

## 9. Testing

**Codec.** Round-trip across `pt.Variants`; golden bytes per version asserting
the exact six-field wire form; a short-buffer case proving `Decode` does not
tolerate a truncated read.

**`EnterInner`** — table-driven against the portal and movement mocks, one case
per branch, each asserting **no** `TeleportCharacter` call on refusal:

- source portal unresolvable → refused
- source `targetMapId` is the sentinel → refused
- source `targetMapId` is a different map → refused
- source `target` empty → refused
- destination portal unresolvable → refused
- last known position beyond threshold → refused
- packet `targetX/targetY` disagreeing with the destination portal → refused
- last-position registry **miss** → accepted (falls back to the packet claim)
- happy path → `TeleportCharacter` called **with the destination portal's own
  coordinates**, fed deliberately mismatched packet coordinates to prove the
  packet is not the source of truth (PRD acceptance criterion)

**Movement.** `TeleportCharacter` publishes the movement command and emits no
clientbound broadcast. The last-position registry is written by both
`ForCharacter` and `TeleportCharacter`.

**atlas-character.** A movement command with `Fh == 0` leaves the stored
foothold intact; a non-zero `Fh` overwrites it.

**Caching.** Two `GetInMapByName` calls for the same tenant+map perform one
REST fetch.

---

## 10. Risks

| Risk | Handling |
|---|---|
| v83/v84/v92 layout diverges from the three confirmed versions | §7.1 derives each before coding; a delta gets a `MajorAtLeast` gate, never a `> N` |
| Portal-list caching makes portal data process-lifetime static | Accepted — identical to `data/map`, and stated in the cache comment |
| `fh = 0` clobbers a foothold | §4.3 — consumer-side preservation plus a test |
| A tight threshold refuses legitimate use | Derived from the client's own collision rect with a stated margin; a refusal is a no-op, so a false positive degrades to today's behaviour, not to a broken portal |
| `atlas-character` was not in the PRD's Service Impact table | §4.3 states the addition and why it belongs to this task rather than a follow-up |
| An inner portal whose `tn` names a portal in another map | Refused by §5.2/§5.3 and logged — the same case the client refuses |

---

## 11. Out of scope

Unchanged from PRD §2: no `CHANGE_MAP`, portal scripts, mystic doors or
teleport rocks; no anti-cheat framework, violation store or punishment; no new
portal authoring or WZ extraction; no client-side behaviour change and no
server-initiated re-teleport on the accepted path.
