# Player NPCs (Imitated NPCs) — Implementation Plan

Task: task-251-player-npcs
Status: Ready for execution
Input: [design.md](design.md) (approved), [prd.md](prd.md) (v1, approved)
Context: [context.md](context.md)

---

## 0. Plan-phase corrections to the design

The design's §7.1 mandated that the implementer re-derive the `IMITATED_NPC_DATA`
field order from the GMS v95.1 IDB and treat a disagreement as authoritative. That
derivation was done during this planning phase. It disagrees. Four corrections
follow; the tasks below are written against the corrected facts.

### P-1 — `IMITATED_NPC_DATA` is a count-prefixed list, not a two-arm dispatcher

The PRD and design §7.1 describe two arms — `0x01` avatar / `0x00` remove-by-objectId.
There are no arms. The leading byte is an **entry count**, and the body is a loop.

Decompiled `CNpcPool::OnNpcImitateData`, identical in all three versions sampled:

| Version | IDB session | Address |
|---|---|---|
| gms_v61 | `GMS_v61.1_U_DEVM.exe.i64` | `0x5efc2e` |
| gms_v83 | `MapleStory_dump.exe.i64` | `0x6d97c6` |
| gms_v95 | `GMS_v95.0_U_DEVM.exe.i64` | `0x679500` |

```
Decode1  count
if count != 0:
  repeat count times:
    Decode4    dwTemplateID      // the imitate script id
    DecodeStr  sName
    AvatarLook::Decode
    // then: for every CNpc in the pool whose m_pTemplate->dwTemplateID == dwTemplateID,
    //       overwrite CNpcTemplate::sName and call CNpc::SetImitatedLook
```

Consequences:

- **There is no object id in this packet.** The client keys the avatar by
  *template id*, i.e. the allocated script id. The design's stored `object_id` is
  still needed — it identifies the map object for `SPAWN_NPC`/`REMOVE_NPC` — but it
  is never encoded into `IMITATED_NPC_DATA`.
- **There is no remove arm.** `count == 0` makes the loop body not execute; it is a
  no-op, not a removal.
- The packet can carry **many entries**, so a map enter is one packet for the whole map.
- The client rewrites the shared `CNpcTemplate::sName` for that template id, which is
  why a script id may be used by at most one Player NPC per (tenant, world) — already
  enforced by the design's `(tenant_id, world_id, script_id)` unique constraint.

### P-2 — `AvatarLook` already exists in the repo; do not re-implement it

`AvatarLook::Decode` (v95 `0x4f2c00`, v83 `0x4e749a`) reads:
gender(1), skin(1), face(4), one discarded byte, hair(4), `{slot byte, itemId int}`
until `0xFF`, the same again for the unseen/masked list, `nWeaponStickerID`(4), then
`DecodeBuffer(12)` — three pet ints.

`libs/atlas-packet/model/avatar.go` (`packetmodel.Avatar`) already encodes exactly
that, including the pre-v61 single-pet and pre-v29 legacy gates. The design's
"a literal `0` … cash weapon id or 0 … three zero ints" is `packetmodel.Avatar`.
The new codec **composes** `packetmodel.Avatar`; it does not re-derive the equip lists.

### P-3 — A `REMOVE_NPC` codec is missing and must be written

Design §7.5 removes a Player NPC with `npcpkt.NpcControllerRevokeBody` followed by an
`IMITATED_NPC_DATA` remove arm. Neither works:

- `NpcControllerRevokeBody` is the flag-0 arm of `CNpcPool::OnNpcChangeController`; it
  demotes the NPC to remote control (`SetRemoteNpc`). It does **not** remove it from
  the pool. See `libs/atlas-packet/npc/clientbound/remove_controller.go:14-21`.
- There is no `IMITATED_NPC_DATA` remove arm (P-1).

The real removal op is `REMOVE_NPC` / `CNpcPool::OnNpcLeaveField`, which reads a single
`Decode4` object id and unlinks the entry (v95 `0x6792c0`, v83 `0x6d9a25`).
`docs/packets/audits/STATUS.md:304` shows it ❌ on every column with **no codec**.
This plan adds it (Task 6). Removal and re-organization use `REMOVE_NPC`.

### P-4 — The ground-snap endpoint mounts under `/data/maps`, and a related endpoint already exists

Design §5.3 specifies `GET /api/maps/{mapId}/ground`. `atlas-data`'s map routes are
mounted at `PathPrefix("/data/maps")` (`services/atlas-data/atlas.com/data/map/resource.go:32`),
and a POST-with-input-model endpoint already exists at
`/data/maps/{mapId}/footholds/below` (`resource.go:44`) which returns the foothold but
**not** the snapped y. The new endpoint follows its neighbours' shape:
`POST /data/maps/{mapId}/ground` taking a list of points and returning a snapped
point + foothold id per input (Task 4).

---

## Execution order

Tasks 1–7 are independent of the new service and can run in any order.
Task 8 must precede Tasks 9–17. Tasks 18–22 depend on Task 17 (the Kafka contract).

---

## Task 1: `job.MaxLevelFor` in `libs/atlas-constants`

- [ ] **Step 1: Write the failing test**

`TestMaxLevelFor` in `libs/atlas-constants/job/max_level_test.go` — table-driven,
setup copied from `libs/atlas-constants/job/constants_test.go` (plain table, no
fixtures).

| case | input `job.Id` | expect |
|---|---|---|
| beginner | `job.BeginnerId` (0) | `200` |
| warrior | `job.WarriorId` (100) | `200` |
| magician | `job.MagicianId` (200) | `200` |
| bowman | `job.BowmanId` (300) | `200` |
| thief | `job.ThiefId` (400) | `200` |
| pirate | `job.PirateId` (500) | `200` |
| noblesse | `job.NoblesseId` (1000) | `200` |
| dawn warrior | `job.Id(1100)` | `200` |
| legend | `job.LegendId` (2000) | `200` |
| evan | `job.EvanId` (2001) | `200` |
| unknown high id | `job.Id(9999)` | `200` |

Every row is 200. `TestMaxLevelForIsExhaustive` additionally asserts
`MaxLevelFor(id) == 200` for every key of `job.Jobs`.

- [ ] **Step 2: Implement**

```go
// MaxLevelFor returns the level cap for the job's line.
func MaxLevelFor(jobId Id) byte
```

Implemented as a `switch` on `GetType(jobId)` (`libs/atlas-constants/job/model.go:64`)
with a per-line arm, **every arm returning 200**. This is not a stub: 200 is the level
cap the server actually implements —
`services/atlas-character/atlas.com/character/character/experience_table.go:4` defines a
single flat `MaxLevel = 200` and `atlas-character` never emits a level above it for any
job. The per-line `switch` exists so the Cygnus figure is a one-line edit if Task 2
confirms it.

- [ ] **Step 3: Verify** — `go build ./... && go test ./...` from `libs/atlas-constants`.

### Files

- `libs/atlas-constants/job/max_level.go` — new file; `MaxLevelFor`
- `libs/atlas-constants/job/max_level_test.go` — new file
- `libs/atlas-constants/job/model.go` — read-only; `GetType`, `IsCygnus`, `Jobs`
- `libs/atlas-constants/job/constants.go` — read-only; `TypeExplorer`/`TypeCygnus` at :182-183, the job ids
- `services/atlas-character/atlas.com/character/character/experience_table.go` — read-only; the `MaxLevel = 200` evidence

Module root for build/test: `libs/atlas-constants`.
Patterns to copy: `libs/atlas-constants/job/advancement.go` (same shape: one small pure function + table test).

---

## Task 2: Verify the Cygnus level cap against WZ/job data

- [ ] **Step 1: Check**

PRD FR-1.2 asserts Cygnus Knights cap at 120; PRD Open Question 5 flags it unverified.
Determine the cap from local evidence only, in this order:

1. `grep -rn 'blockedJobs\|maxLevel\|levelLimit' <wz-root>/String.wz <wz-root>/Character.wz` for a
   per-job cap in the mounted WZ trees named in design §1 C-1 (`ms_1172`, `AtlasMS`).
2. The v95 IDB (session `ecc757f4`) — search for a level-cap comparison in the
   job/level UI path.

- [ ] **Step 2: Act on the result**

- If a 120 cap for `job.TypeCygnus` is confirmed: change the Cygnus arm of
  `MaxLevelFor` to `return 120` and update the Cygnus rows of the Task 1 table test.
- If it is **not** confirmed: leave the table at 200 and record the negative result —
  what was searched, what was found — in `docs/tasks/task-251-player-npcs/progress.md`.
  Do not guess.

- [ ] **Step 3: Verify** — `go test ./...` from `libs/atlas-constants`.

### Files

- `libs/atlas-constants/job/max_level.go` — new file from Task 1; the Cygnus arm, only if confirmed
- `libs/atlas-constants/job/max_level_test.go` — new file from Task 1; Cygnus rows, only if confirmed
- `docs/tasks/task-251-player-npcs/progress.md` — new file if absent; record the finding either way

---

## Task 3: `PlayerNpcObjectIdBase` in `libs/atlas-object-id`

- [ ] **Step 1: Write the failing test**

`TestPlayerNpcObjectIdBandDoesNotCollide` in `libs/atlas-object-id/reserved_test.go`:

| case | assertion |
|---|---|
| above WZ NPC ids | `PlayerNpcObjectIdBase == uint32(100000)` |
| below the shared allocator | `PlayerNpcObjectIdBase < MinId` (`MinId == 1000000`) |
| whole script range fits the band | for `scriptId` in `{9900000, 9901000, 9906599}`, `PlayerNpcObjectIdFor(scriptId)` is `>= PlayerNpcObjectIdBase` and `< MinId` |
| lowest script id maps to the base | `PlayerNpcObjectIdFor(9900000) == 100000` |
| highest maps below MinId | `PlayerNpcObjectIdFor(9906599) == 106599` |

- [ ] **Step 2: Implement**

```go
const PlayerNpcObjectIdBase = uint32(100000)

func PlayerNpcObjectIdFor(scriptId uint32) uint32
```

`PlayerNpcObjectIdFor` returns `PlayerNpcObjectIdBase + (scriptId - 9900000)`. Carry a
doc comment recording the reservation of 100000–999999 and that lowering `MinId` below
1000000 would collide with it, per design D-5. Guard `scriptId < 9900000` by returning
`PlayerNpcObjectIdBase` — the allocator never produces such an id (design §4.2 pool is
9901000–9906599), and the caller must not be handed a wrapped uint32.

- [ ] **Step 3: Verify** — `go build ./... && go test ./...` from `libs/atlas-object-id`.

### Files

- `libs/atlas-object-id/reserved.go` — new file
- `libs/atlas-object-id/reserved_test.go` — new file
- `libs/atlas-object-id/allocator.go` — read-only; `MinId`/`MaxId` at :28-38 and the namespace doc comment

Module root: `libs/atlas-object-id`.

---

## Task 4: `atlas-data` — `imitate` flag and the batched ground endpoint

- [ ] **Step 1: Write the failing tests**

`TestNpcReadImitateFlag` in `services/atlas-data/atlas.com/data/npc/reader_test.go`
(new file — no reader test exists for npc today; copy the XML-node fixture shape from
`services/atlas-data/atlas.com/data/map/reader_test.go`):

| case | `info` node | expect `RestModel.Imitate` |
|---|---|---|
| imitate 1 | `<int name="imitate" value="1"/>` | `true` |
| imitate 0 | `<int name="imitate" value="0"/>` | `false` |
| absent | no `imitate` child | `false` |
| no info section | node has no `info` child | `false` |

`TestHandleGetMapGroundRequest` in
`services/atlas-data/atlas.com/data/map/resource_ground_test.go` (new file; HTTP-handler
setup copied from `services/atlas-data/atlas.com/data/map/resource_test.go`):

| case | request body points | expect |
|---|---|---|
| single point over a flat foothold | `[{x: 0, y: -100}]` | one result, `y` equal to that foothold's `First.Y`, `fh` its id |
| two points, one over empty space | `[{x: 0, y: -100}, {x: 30000, y: -100}]` | two results in input order; the second has `found: false` and zeroed `y`/`fh` |
| empty list | `[]` | `400 Bad Request` |

- [ ] **Step 2: Implement**

1. Add `Imitate bool \`json:"imitate"\`` to `npc.RestModel` and parse it in `Read` with
   `m.Imitate = node.GetIntegerWithDefault("imitate", 0) == 1`, in the same block that
   already reads `trunkPut`/`storebank`/`hideName`
   (`services/atlas-data/atlas.com/data/npc/reader.go:66-77`).
2. Add `GroundPointRestModel` (`X, Y int16`), `GroundRequestRestModel`
   (`Points []GroundPointRestModel`) and `GroundResultRestModel`
   (`X, Y int16; Fh uint32; Found bool`) to `services/atlas-data/atlas.com/data/map/rest.go`.
3. Add `handleGetMapGroundRequest` to `resource.go` and register it as
   `r.HandleFunc("/{mapId}/ground", rest.RegisterInputHandler[GroundRequestRestModel](l)(si)("get_map_ground", handleGetMapGroundRequest(db))).Methods(http.MethodPost)`,
   immediately after the `/{mapId}/footholds/below` line. Per input point, call the
   existing `calcPointBelow(m.FootholdTree, point.RestModel{X, Y})`
   (`processor.go:132`) and `m.FootholdTree.findBelow` (`model.go:35`) for the foothold
   id; results are returned **in input order**, one per input, with `Found: false` where
   `calcPointBelow` reports no ground. Do not drop unmatched points — the caller's
   lattice walk indexes by position.

Do not reimplement the geometry; both helpers already exist in this package.

- [ ] **Step 3: Verify** — `go build ./... && go test ./...` from
  `services/atlas-data/atlas.com/data`.

### Files

- `services/atlas-data/atlas.com/data/npc/rest.go` — add `Imitate`
- `services/atlas-data/atlas.com/data/npc/reader.go` — parse `info/imitate` (:66 onward)
- `services/atlas-data/atlas.com/data/npc/reader_test.go` — new file
- `services/atlas-data/atlas.com/data/map/rest.go` — the three new rest models
- `services/atlas-data/atlas.com/data/map/resource.go` — handler + route (route goes after :44)
- `services/atlas-data/atlas.com/data/map/resource_ground_test.go` — new file
- `services/atlas-data/atlas.com/data/map/processor.go` — read-only; `calcPointBelow` at :132
- `services/atlas-data/atlas.com/data/map/model.go` — read-only; `findBelow` at :35, `findById` at :78

Module root: `services/atlas-data/atlas.com/data`.

---

## Task 5: `IMITATED_NPC_DATA` codec

Read [docs/packets/IMPLEMENTING_A_PACKET.md](../../packets/IMPLEMENTING_A_PACKET.md)
§2–§4 before starting. §0 and §1 are already done — P-1 and P-2 above are the
derivation, and they are authoritative for this task.

- [ ] **Step 1: Write the failing test**

`TestImitatedNpcData` in `libs/atlas-packet/npc/clientbound/imitated_npc_data_test.go`.
Scaffolding (imports, `test.Variants` loop, `test.CreateContext`, `test.RoundTrip`)
copied verbatim from `libs/atlas-packet/npc/clientbound/spawn_test.go:1-23`.

Golden-byte case — one entry, GMS v83 context, exactly one equip and no masked equip
(a single map entry keeps the encoding deterministic; `packetmodel.Avatar` ranges over
Go maps):

```
input := NewImitatedNpcData([]ImitatedNpc{
    NewImitatedNpc(9901000, "Hero", packetmodel.NewAvatar(
        0,                                     // gender
        3,                                     // skin
        20000,                                 // face
        false,                                 // mega
        30030,                                 // hair
        map[slot.Position]uint32{5: 1040010},  // equipment
        map[slot.Position]uint32{},            // masked
        map[int8]uint32{},                     // pets
    )),
})
got := input.Encode(nil, test.CreateContext("GMS", 83, 1))(nil)
```

expected bytes, field by field:

| bytes | field | source |
|---|---|---|
| `01` | entry count | `Decode1` @v83 `0x6d97e1` |
| `C8 1B 97 00` | templateId 9901000 | `Decode4` @`0x6d97fd` |
| `04 00 48 65 72 6F` | name "Hero" (uint16 len + ShiftJIS bytes) | `DecodeStr` @`0x6d9803` |
| `00` | gender | `AvatarLook::Decode` @v83 `0x4e74ad` |
| `03` | skin | `0x4e74ba` |
| `20 4E 00 00` | face 20000 | `0x4e74ce` |
| `01` | `!mega` (the byte the client discards) | `0x4e74ea` |
| `4E 75 00 00` | hair 30030 | `0x4e74f6` |
| `05 8A DE 0F 00` | equip slot 5 → item 1040010 | loop @`0x4e74ff` |
| `FF` | equip terminator | `0x4e7506` |
| `FF` | masked terminator (empty list) | `0x4e753d` |
| `00 00 00 00` | `nWeaponStickerID` | `0x4e7572` |
| `00 00 00 00 00 00 00 00 00 00 00 00` | three pet ints (`DecodeBuffer(12)`) | `0x4e7585` |

Second case, `TestImitatedNpcDataEmpty`: `NewImitatedNpcData(nil)` encodes to exactly
`[]byte{0x00}` — the count-0 no-op, which is **not** a removal.

Then the standard `test.Variants` round-trip loop. `packet-audit:verify` markers, one
per applicable version, above `TestImitatedNpcData`:

```
// packet-audit:verify packet=npc/clientbound/ImitatedNpcData version=gms_v61 ida=<resolved>
// ... same for gms_v72, gms_v79, gms_v83, gms_v84, gms_v87, gms_v92, gms_v95, jms_v185
```

`gms_v48` gets **no** marker — `docs/packets/audits/STATUS.md:134` records it ⬜ with no
opcode. Resolve each `ida=` address from that version's fname in
`docs/packets/registry/<version>.yaml` (they differ per version:
`sub_830B0B` on v61, `sub_902E83` on v72, `CWvsContext::OnImitatedNPCData` on
v79/v83/v87/v92/v95, `CNpcPool::OnNpcImitateData` on v84/jms_v185) — never hand-write it.

- [ ] **Step 2: Implement**

`libs/atlas-packet/npc/clientbound/imitated_npc_data.go`, following the immutable-struct
+ `Operation()` convention of `spawn_request_controller.go`:

```go
const NpcImitatedDataWriter = "ImitatedNPCData"

// packet-audit:fname CNpcPool::OnNpcImitateData
type ImitatedNpc struct { /* templateId uint32; name string; look packetmodel.Avatar */ }
func NewImitatedNpc(templateId uint32, name string, look packetmodel.Avatar) ImitatedNpc

type ImitatedNpcData struct { /* entries []ImitatedNpc */ }
func NewImitatedNpcData(entries []ImitatedNpc) ImitatedNpcData
func (m ImitatedNpcData) Operation() string
func (m ImitatedNpcData) String() string
func (m ImitatedNpcData) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte
func (m *ImitatedNpcData) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{})
```

`Encode` writes `WriteByte(byte(len(entries)))` then per entry `WriteInt(templateId)`,
`WriteAsciiString(name)`, and appends `look.Encode(l, ctx)(options)`. `Decode` mirrors
it via `packetmodel.Avatar.Decode`. There is **no** version gate in this struct: the only
version-dependent bytes live inside `packetmodel.Avatar`, which already carries them
(`libs/atlas-packet/model/avatar.go:52,79,89`). Do not add a `MajorAtLeast` branch here —
adding one would double-gate the avatar and change the wire on already-verified versions.

- [ ] **Step 3: Verify**

```
go test ./npc/... ./model/...          # from libs/atlas-packet
go run ./tools/packet-audit evidence pin --packet npc/clientbound/ImitatedNpcData \
  --version <key> --ida "<fname from docs/packets/registry/<key>.yaml>" --category TIER1-FIXTURE
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
```

Repeat the pin for each of the nine applicable versions, then add the `verifies:` list to
each generated `docs/packets/evidence/<version>/npc.clientbound.ImitatedNpcData.yaml`
pointing at `libs/atlas-packet/npc/clientbound/imitated_npc_data_test.go#TestImitatedNpcData`.
The `IMITATED_NPC_DATA` row must flip to ✅ on all nine columns and stay ⬜ on `gms_v48`,
with no new orphan/dangling/stale/drift lines. A cell that does not promote is a failure.

### Files

- `libs/atlas-packet/npc/clientbound/imitated_npc_data.go` — new file
- `libs/atlas-packet/npc/clientbound/imitated_npc_data_test.go` — new file
- `libs/atlas-packet/model/avatar.go` — read-only; `packetmodel.Avatar`, `NewAvatar` at :24
- `libs/atlas-packet/npc/clientbound/spawn_request_controller.go` — read-only; struct/Operation/Encode shape to copy
- `libs/atlas-packet/npc/clientbound/spawn_test.go` — read-only; test scaffolding to copy
- `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json` — regenerated, not hand-edited
- `docs/packets/evidence/<version>/npc.clientbound.ImitatedNpcData.yaml` — new files, one per version

Module root: `libs/atlas-packet`. `packet-audit` runs from the repo root.

---

## Task 6: `REMOVE_NPC` codec

- [ ] **Step 1: Write the failing test**

`TestNpcRemove` in `libs/atlas-packet/npc/clientbound/remove_test.go`. Scaffolding copied
from `spawn_test.go:1-23`.

Golden-byte case, GMS v83:

```
input := NewNpcRemove(100123)
got := input.Encode(nil, test.CreateContext("GMS", 83, 1))(nil)
want := []byte{0x9B, 0x87, 0x01, 0x00}   // objectId 100123 uint32 LE (Decode4 @v83 0x6d9a25)
```

Then the `test.Variants` round-trip loop. `packet-audit:verify` markers for **all ten**
versions — `docs/packets/audits/STATUS.md:304` shows `REMOVE_NPC` carrying an opcode on
every column including `gms_v48` (0x0B2). Resolve each `ida=` from the per-version fname
`CNpcPool::OnNpcLeaveField` in `docs/packets/registry/<version>.yaml`.

- [ ] **Step 2: Implement**

```go
const NpcRemoveWriter = "RemoveNPC"

// packet-audit:fname CNpcPool::OnNpcLeaveField
type Remove struct { /* id uint32 */ }
func NewNpcRemove(id uint32) Remove
```

`Encode` is a single `WriteInt(m.id)`; `Decode` a single `ReadUint32`. No version gate:
`CNpcPool::OnNpcLeaveField` reads exactly one `Decode4` on both v95 (`0x6792c0`) and
v83 (`0x6d9a25`), and the function body is `0x5e` bytes on v61 (`0x5efe4b`), v83 and
v95 alike. Confirm the remaining seven versions read one `Decode4` before pinning them.

- [ ] **Step 3: Verify** — same command sequence as Task 5, with
  `--packet npc/clientbound/NpcRemove`. The `REMOVE_NPC` row must flip to ✅ on all ten
  columns.

### Files

- `libs/atlas-packet/npc/clientbound/remove.go` — new file
- `libs/atlas-packet/npc/clientbound/remove_test.go` — new file
- `libs/atlas-packet/npc/clientbound/remove_controller.go` — read-only; the *different* op it must not be confused with
- `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json` — regenerated
- `docs/packets/evidence/<version>/npc.clientbound.NpcRemove.yaml` — new files

Module root: `libs/atlas-packet`.

---

## Task 7: Route both writers in the seed templates

- [ ] **Step 1: Establish the expected state**

`TestTemplatesRouteImitatedNpcData` in
`services/atlas-configurations/atlas.com/configurations/seed_template_writers_test.go`
(new file) — parse every `services/atlas-configurations/seed-data/templates/*.json`
and assert, per template, the presence/absence and opcode of both writers:

| template | `ImitatedNPCData` opCode | `RemoveNPC` opCode |
|---|---|---|
| `template_gms_12_1.json` | absent | absent |
| `template_gms_48_1.json` | absent | `0xB2` |
| `template_gms_61_1.json` | `0x4E` | `0xC3` |
| `template_gms_72_1.json` | `0x4E` | `0xE4` |
| `template_gms_79_1.json` | `0x4E` | `0xEC` |
| `template_gms_83_1.json` | `0x51` | `0x102` |
| `template_gms_84_1.json` | `0x53` | `0x109` |
| `template_gms_87_1.json` | `0x53` | `0x113` |
| `template_gms_92_1.json` | `0x56` | `0x130` |
| `template_gms_95_1.json` | `0x54` | `0x138` |
| `template_jms_185_1.json` | `0x55` | `0x117` |

Every opcode above is the decimal `opcode:` field of the matching
`docs/packets/registry/<version>.yaml` entry converted to hex — re-read the registry per
file rather than copying a neighbouring row; the v84 clientbound table is shifted
relative to v83.

`template_gms_12_1.json` gets neither writer: there is no `docs/packets/registry/gms_v12.yaml`
and no v12 IDB, so no opcode is evidenced. Registering a writer with an unevidenced or
absent opcode is what produces a silent mis-encode.
`template_gms_48_1.json` gets `RemoveNPC` only — `IMITATED_NPC_DATA` is ⬜ on `gms_v48`.

- [ ] **Step 2: Implement**

Insert each entry into that template's `socket.writers[]` array in **sorted-opcode
position**, in the plain form `{ "opCode": "0x51", "writer": "ImitatedNPCData" }`. The
writer names must match the `Operation()` strings from Tasks 5 and 6 exactly. Neither
writer carries an `options`/`operations` block — neither has a config-resolved mode byte.

- [ ] **Step 3: Verify** — `go test ./...` from
  `services/atlas-configurations/atlas.com/configurations`.

### Files

- `services/atlas-configurations/seed-data/templates/template_gms_48_1.json` — `RemoveNPC` only
- `services/atlas-configurations/seed-data/templates/template_gms_61_1.json` — both
- `services/atlas-configurations/seed-data/templates/template_gms_72_1.json` — both
- `services/atlas-configurations/seed-data/templates/template_gms_79_1.json` — both
- `services/atlas-configurations/seed-data/templates/template_gms_83_1.json` — both
- `services/atlas-configurations/seed-data/templates/template_gms_84_1.json` — both
- `services/atlas-configurations/seed-data/templates/template_gms_87_1.json` — both
- `services/atlas-configurations/seed-data/templates/template_gms_92_1.json` — both
- `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` — both
- `services/atlas-configurations/seed-data/templates/template_jms_185_1.json` — both
- `services/atlas-configurations/atlas.com/configurations/seed_template_writers_test.go` — new file
- `docs/packets/registry/*.yaml` — read-only; the per-version opcode source

This task is ten files but a single mechanical edit repeated per file; see context.md.

---

## Task 8: `atlas-player-npcs` service scaffold and registration

- [ ] **Step 1: Establish the expected state**

`tools/service-registration-guard.sh` must exit 0 with the new service present. That is
the acceptance check for this task; there is no unit test to write first.

- [ ] **Step 2: Implement**

Code scaffold, copied structurally from `atlas-notes` (the nearest DB-backed
REST + Kafka + outbox service):

- `services/atlas-player-npcs/atlas.com/player-npcs/go.mod` — `module atlas-player-npcs`, `go 1.25.5`
- `services/atlas-player-npcs/atlas.com/player-npcs/main.go` — copy
  `services/atlas-notes/atlas.com/notes/main.go` wholesale, substituting
  `serviceName = "atlas-player-npcs"`, `consumergroup.Resolve("Player NPC Service")`,
  and the migration/route/consumer wiring added by Tasks 12 and 17. Until then it
  compiles with the `playernpc.Migration` from Task 12 — **do this task's `main.go` last
  within the task**, or leave the consumer/route lines to Tasks 16/17 as noted there.
- `services/atlas-player-npcs/.bruno/{bruno.json,collection.bru,environments/*.bru}` —
  per `.claude/skills/backend-dev-guidelines/resources/scaffolding-checklist.md` §3.

Registration, per [docs/adding-a-new-service.md](../../adding-a-new-service.md):

| § | File | Entry |
|---|---|---|
| 1.1 | `.github/config/services.json` | `name: atlas-player-npcs`, `path: services/atlas-player-npcs`, `module_path: services/atlas-player-npcs/atlas.com/player-npcs`, `docker_image: ghcr.io/chronicle20/atlas-player-npcs/atlas-player-npcs`, `docker_context: "."` |
| 1.2 | `docker-bake.hcl` | `"atlas-player-npcs"` in `go_services` |
| 1.3 | `go.work` | `./services/atlas-player-npcs/atlas.com/player-npcs` |
| 2.1 | `deploy/k8s/base/atlas-player-npcs.yaml` | new; copy `deploy/k8s/base/atlas-notes.yaml`; container name `player-npcs`; `DB_NAME: atlas-player-npcs` |
| 2.2 | `deploy/k8s/base/kustomization.yaml` | add to `resources:` |
| 2.3 | `deploy/k8s/base/env-configmap.yaml` | `COMMAND_TOPIC_PLAYER_NPC` and `EVENT_TOPIC_PLAYER_NPC_STATUS` as `KEY: "KEY"` |
| 3.1 | `deploy/k8s/overlays/main/patches/db-name-suffix.yaml` | `DB_NAME: "atlas-player-npcs-main"` |
| 3.2 | `deploy/k8s/overlays/main/patches/atlas-env-env.yaml` | `ATLAS_ENV: "main"` |
| 3.3–3.4 | `deploy/k8s/overlays/main/kustomization.yaml` | `images:` pin at the current fleet tag; both topic literals as `KEY=KEY-main` |
| 4.1–4.2 | `deploy/k8s/overlays/pr/kustomization.yaml` | `atlas-player-npcs` in `ATLAS_DB_NAMES`; `images:` entry |
| 4.3–4.6 | generator-owned | re-run `deploy/k8s/overlays/pr/scripts/gen-topic-config.sh`, `gen-db-name-suffix.sh`, `gen-consumer-group-patch.sh`, `gen-cleanup-env.sh` and commit their output — do not hand-edit |
| 5.1–5.2 | `deploy/shared/routes.conf` + `tools/gen-routes.sh` | `location ~ ^/api/player-npcs(/.*)?$` → `atlas-player-npcs:8080`, alphabetically placed; then regenerate and commit `deploy/k8s/base/routes.conf.template.generated` |
| 6.2 | `tools/db-bootstrap.sh` | `atlas-player-npcs` in the `DBS` list |

- [ ] **Step 3: Verify**

```
tools/service-registration-guard.sh                     # must exit 0
kubectl kustomize deploy/k8s/overlays/pr > /dev/null    # renders clean
go build ./...                                          # from the new module root
```

Two steps this task cannot complete and must hand back to the operator, recorded in
`docs/tasks/task-251-player-npcs/progress.md`: creating `atlas-player-npcs-main` on
postgres.home (§6.1) and flipping the new GHCR package to public after the first image
push (§6b).

### Files

- `services/atlas-player-npcs/atlas.com/player-npcs/go.mod` — new file
- `services/atlas-player-npcs/atlas.com/player-npcs/main.go` — new file
- `services/atlas-player-npcs/.bruno/bruno.json`, `collection.bru`, `environments/Local.bru`, `environments/Local Debug.bru`, `environments/Atlas - K3S.bru` — new files
- `deploy/k8s/base/atlas-player-npcs.yaml` — new file
- `.github/config/services.json`, `docker-bake.hcl`, `go.work`, `tools/db-bootstrap.sh`, `deploy/shared/routes.conf` — one entry each
- `deploy/k8s/base/kustomization.yaml`, `deploy/k8s/base/env-configmap.yaml`
- `deploy/k8s/overlays/main/kustomization.yaml`, `deploy/k8s/overlays/main/patches/db-name-suffix.yaml`, `deploy/k8s/overlays/main/patches/atlas-env-env.yaml`
- `deploy/k8s/overlays/pr/kustomization.yaml`
- `services/atlas-notes/atlas.com/notes/main.go` — read-only; the scaffold to copy
- `deploy/k8s/base/atlas-notes.yaml` — read-only; the manifest to copy

This task is deliberately large — it is one registration checklist, not six changes; see context.md.

---

## Task 9: `routing/` — job category → Hall of Fame map

- [ ] **Step 1: Write the failing test**

`TestHallOfFameMapFor` in `services/atlas-player-npcs/atlas.com/player-npcs/routing/routing_test.go`:

| case | input `job.Id` | expect `_map.Id` |
|---|---|---|
| warrior | `job.WarriorId` (100) | `_map.VictoriaRoadHallOfWarriors1Id` |
| fighter (sub-job) | `job.Id(110)` | `_map.VictoriaRoadHallOfWarriors1Id` |
| magician | `job.MagicianId` (200) | `_map.VictoriaRoadHallOfMagicians1Id` |
| bowman | `job.BowmanId` (300) | `_map.VictoriaRoadHallOfBowmen1Id` |
| thief | `job.ThiefId` (400) | `_map.VictoriaRoadHallOfThieves1Id` |
| pirate | `job.PirateId` (500) | `_map.TheNautilusTrainingRoom2Id` |
| dawn warrior | `job.Id(1100)` | `_map.EmpressRoadKnightsChamber1Id` |
| thunder breaker | `job.Id(1500)` | `_map.EmpressRoadKnightsChamber1Id` |
| aran | `job.Id(2100)` | `_map.SnowIslandPalaceOfTheMaster1Id` |
| beginner | `job.BeginnerId` (0) | `_map.EmpressRoadKnightsChamber2ndFloorId` |
| noblesse | `job.NoblesseId` (1000) | `_map.EmpressRoadKnightsChamber2ndFloorId` |
| evan | `job.EvanId` (2001) | `_map.EmpressRoadKnightsChamber2ndFloorId` |

`TestIsPodiumMap`:

| case | input | expect |
|---|---|---|
| hall of warriors | `_map.VictoriaRoadHallOfWarriors1Id` | `true` |
| hall of magicians | `_map.VictoriaRoadHallOfMagicians1Id` | `true` |
| hall of bowmen | `_map.VictoriaRoadHallOfBowmen1Id` | `true` |
| hall of thieves | `_map.VictoriaRoadHallOfThieves1Id` | `true` |
| nautilus training room | `_map.TheNautilusTrainingRoom2Id` | `true` |
| knights' chamber | `_map.EmpressRoadKnightsChamber1Id` | `false` |
| knights' chamber 2nd floor | `_map.EmpressRoadKnightsChamber2ndFloorId` | `false` |
| arbitrary map | `_map.Id(100000000)` | `false` |

`TestIsHallOfFameMap` asserts `true` for all ten constants named in design §1 C-2 and
`false` for `_map.Id(100000000)`.

- [ ] **Step 2: Implement**

```go
func HallOfFameMapFor(jobId job.Id) _map.Id
func IsPodiumMap(mapId _map.Id) bool
func IsHallOfFameMap(mapId _map.Id) bool
func JobCategory(jobId job.Id) uint16   // (jobId / 100) * 100
```

`HallOfFameMapFor` switches on `JobCategory` for Explorer lines and on
`job.GetType(jobId) == job.TypeCygnus` for the Cygnus branches, defaulting to
`EmpressRoadKnightsChamber2ndFloorId`. Reference the existing constants from
`libs/atlas-constants/map/constants.go` — **do not declare parallel map ids**; all ten
already exist (design §1 C-2, verified at lines 69, 103, 162, 191, 527, 535, 536, 537,
538, 566). No literal map id appears in this package.

- [ ] **Step 3: Verify** — `go build ./... && go test ./...` from
  `services/atlas-player-npcs/atlas.com/player-npcs`.

### Files

- `services/atlas-player-npcs/atlas.com/player-npcs/routing/routing.go` — new file
- `services/atlas-player-npcs/atlas.com/player-npcs/routing/routing_test.go` — new file
- `libs/atlas-constants/map/constants.go` — read-only; the ten map constants
- `libs/atlas-constants/job/model.go` — read-only; `GetType`, `IsBeginner`
- `libs/atlas-constants/job/constants.go` — read-only; `TypeCygnus` at :183

---

## Task 10: `allocation/` — the script-id pool

- [ ] **Step 1: Write the failing test**

`TestBranchFor` in `services/atlas-player-npcs/atlas.com/player-npcs/allocation/allocation_test.go`:

| case | jobId / mapId | expect branch |
|---|---|---|
| warrior, hall of warriors | 100 / 102000004 | 10 |
| magician, hall of magicians | 200 / 101000004 | 11 |
| bowman | 300 / 100000204 | 12 |
| thief | 400 / 103000008 | 13 |
| pirate | 500 / 120000105 | 14 |
| dawn warrior | 1100 / 130000100 | 15 |
| blaze wizard | 1200 / 130000100 | 16 |
| wind archer | 1300 / 130000100 | 17 |
| night walker | 1400 / 130000100 | 18 |
| thunder breaker | 1500 / 130000100 | 19 |
| aran | 2100 / 140010110 | 20 |
| evan | 2001 / 130000110 | 21 |
| beginner | 0 / 130000110 | 22 |
| noblesse | 1000 / 130000110 | 23 |
| legend | 2000 / 130000110 | 24 |
| GM deploy, non-HoF map, continent 1 | 100 / 100000000 | 27 (`26 + 4*1`) |
| GM deploy, non-HoF map, continent 2 | 100 / 200000000 | 30 (`26 + 4*2`) |

`TestAllocate` — the allocator is pure: it takes `usable map[uint32]bool`,
`inUse map[uint32]bool`, `branch uint32` and returns `(uint32, error)`.

| case | usable | inUse | branch | expect |
|---|---|---|---|---|
| in-branch hit, lowest first | `{9901000, 9901001}` | `{}` | 10 | `9901000` |
| in-branch, lowest free | `{9901000, 9901001}` | `{9901000}` | 10 | `9901001` |
| branch empty → global fallback | `{9901500}` | `{}` | 14 (pirate) | `9901500` |
| branch exhausted → global fallback | `{9901000, 9901500}` | `{9901000}` | 10 | `9901500` |
| whole pool exhausted | `{9901000}` | `{9901000}` | 10 | `ErrPoolExhausted` |
| empty usable set | `{}` | `{}` | 10 | `ErrPoolExhausted` |
| GM branch, nothing in branch, fallback wins | `{9901000}` | `{}` | 27 | `9901000` |

`TestUsablePool` — the usable-set builder is given a lookup
`func(uint32) (exists bool, imitate bool, err error)`:

| case | lookup result for the id | expect in set |
|---|---|---|
| exists and imitate | `(true, true, nil)` | yes |
| exists, imitate 0 | `(true, false, nil)` | **no** |
| template missing | `(false, false, nil)` | **no** |
| lookup error | `(false, false, err)` | **no**, and the error is surfaced |

The two rejection rows are explicit PRD acceptance criteria (PRD §10) — a script id that
fails either condition must be unallocatable by construction.

- [ ] **Step 2: Implement**

```go
const PoolMin = uint32(9901000)
const PoolMax = uint32(9906599)
const BranchBase = uint32(9900000)
const BranchSize = uint32(100)

var ErrPoolExhausted = errors.New("pool_exhausted")

func BranchFor(jobId job.Id, mapId _map.Id) uint32
func BranchRange(branch uint32) (uint32, uint32)
func Allocate(usable map[uint32]bool, inUse map[uint32]bool, branch uint32) (uint32, error)
```

`BranchFor` implements the PRD FR-3.2 table for Hall of Fame maps (via
`routing.IsHallOfFameMap`) and the FR-3.3 GM formula `26 + 4*(mapId/100000000)`
otherwise. `Allocate` scans the branch range ascending, then the whole `usable` set
ascending as the global fallback (design D-1). The fallback scans the *same* validated
set, so a fallback-allocated id is exactly as safe as a branch-allocated one.

Pure package: no database, no HTTP client, no context. The impure usable-set builder and
its per-tenant cache live in `allocation/pool.go` and take the lookup function as a
parameter, so the tests above need no infrastructure. The cache is built once, lazily,
per tenant, and held for the process lifetime — `atlas-data` projects WZ read-only, so
the set cannot go stale without a restart (design §4.2).

- [ ] **Step 3: Verify** — `go build ./... && go test ./...` from the module root.

### Files

- `services/atlas-player-npcs/atlas.com/player-npcs/allocation/allocation.go` — new file; `BranchFor`, `BranchRange`, `Allocate`
- `services/atlas-player-npcs/atlas.com/player-npcs/allocation/pool.go` — new file; usable-set builder + per-tenant cache
- `services/atlas-player-npcs/atlas.com/player-npcs/allocation/allocation_test.go` — new file
- `services/atlas-player-npcs/atlas.com/player-npcs/routing/routing.go` — from Task 9; `IsHallOfFameMap`

---

## Task 11: `position/` — grid and podium positioners

- [ ] **Step 1: Write the failing test**

`TestGridPositions` in `services/atlas-player-npcs/atlas.com/player-npcs/position/position_test.go`.
Tuning fixed at the FR-4.7 defaults (`initialX` 262, `initialY` 262, `areaX` 320,
`areaY` 160, `areaSteps` 4) unless a row says otherwise.

| case | step | expect |
|---|---|---|
| pitch at step 0 | 0 | `dx == 320` (`areaX/(0+1)`), `dy == 160` (`160/2 + 160/2^1`) |
| pitch at step 1 | 1 | `dx == 160`, `dy == 120` (`80 + 40`) |
| pitch at step 2 | 2 | `dx == 106` (integer division), `dy == 100` (`80 + 20`) |
| pitch at step 3 | 3 | `dx == 80`, `dy == 90` (`80 + 10`) |
| first placement on an empty map | 0 | the first lattice point inset by `initialX`/`initialY` that snaps to ground |
| second placement avoids the first | 0 | a point whose `dx × dy` rectangle does not intersect the first |
| no free slot at `areaSteps` | 4, lattice fully occupied | `ErrMapFull` |

`TestPodiumPositions`:

| case | rank / step | expect x / y |
|---|---|---|
| platform 0, first slot | r=0, s=1 | `x == -50 + 100*1/2 == 0`, `y == -47` |
| platform 0, second slot | r=1, s=2 | platform `1/2 == 0`, relative `(1%2)+1 == 2`, `x == -50 + 100*2/3 == 16`, `y == -47` |
| platform 1 | r=2, s=1 | platform `2/1 == 2` → `platformX == 70`, `platformY == 40` |
| platform 1 boundary | r=1, s=1 | platform 1 → `x == -170 + 100*1/2 == -120`, `y == 40` |
| step 0 is an error, not a panic | r=0, s=0 | `ErrInvalidStep` — **must not panic** (`r / s` divides by zero) |

`TestPodiumStepRaise`:

| case | encoded `(step, count)` | expect |
|---|---|---|
| decode | `count*32 + step` with count=3, step=1 → 97 | `(step 1, count 3)` |
| raise when `count >= 3*step` | step 1, count 3 | step becomes 2, re-organization required |
| no raise below the threshold | step 2, count 5 | step stays 2 |
| bounded by `areaSteps` | step 4, count 12, `areaSteps` 4 | `ErrMapFull` — no raise past the bound |

`TestReorganize`:

| case | input | expect |
|---|---|---|
| ordering | three NPCs with script ids 9901002, 9901000, 9901001 | recomputed in ascending script-id order — 9901000, 9901001, 9901002 |
| all repositioned | same | three results, one per input, all at `step + 1` |

- [ ] **Step 2: Implement**

```go
type Tuning struct { /* InitialX, InitialY, AreaX, AreaY int16; AreaSteps byte; OrganizeArea bool */ }
type Rect struct { /* X, Y, W, H int16 */ }
type Point struct { /* X, CY int16; Fh uint32 */ }
type SnapFunc func(points []Point) ([]Point, error)

var ErrMapFull = errors.New("map_full")
var ErrInvalidStep = errors.New("invalid_step")

func GridPitch(t Tuning, step byte) (dx int16, dy int16)
func NextGridPosition(t Tuning, bounds Rect, step byte, placed []Rect, snap SnapFunc) (Point, error)
func PodiumPosition(rank uint32, step byte) (Point, error)
func EncodePodiumState(step byte, count uint32) uint32
func DecodePodiumState(state uint32) (step byte, count uint32)
func Reorganize(t Tuning, bounds Rect, step byte, existing []Placement, snap SnapFunc) ([]Placement, error)
```

Both positioners are pure: `snap` is injected, so the tests use a stub and touch no HTTP
client. `NextGridPosition` issues **one** `snap` call for the whole step's lattice (the
`SnapFunc` takes and returns a slice) so the walk costs one round trip per step, per
design §5.3.

`PodiumPosition` returns `ErrInvalidStep` when `step == 0` rather than dividing by zero.
`platformX` is −50 / −170 / 70 for platform 0 / 1 / other; `platformY` is −47 for
platform 0 and 40 otherwise.

- [ ] **Step 3: Verify** — `go build ./... && go test ./...` from the module root.

### Files

- `services/atlas-player-npcs/atlas.com/player-npcs/position/grid.go` — new file
- `services/atlas-player-npcs/atlas.com/player-npcs/position/podium.go` — new file
- `services/atlas-player-npcs/atlas.com/player-npcs/position/types.go` — new file; `Tuning`, `Rect`, `Point`, `Placement`, `SnapFunc`, the errors
- `services/atlas-player-npcs/atlas.com/player-npcs/position/position_test.go` — new file

---

## Task 12: Persistence — entity, model, builder, administrator, provider

- [ ] **Step 1: Write the failing test**

`TestPlayerNpcEntityRoundTrip` in
`services/atlas-player-npcs/atlas.com/player-npcs/playernpc/model_test.go`, built through
the Builder (no `*_testhelpers.go`):

| case | assertion |
|---|---|
| model → entity → model | every field survives `MakeEntity` then `Make` unchanged |
| `dir` default | a Builder with no `SetDir` produces `Dir() == 1` |
| computed rx | `SetX(100)` produces `RX0() == 150`, `RX1() == 50` |
| job category stored | `SetJobId(job.Id(112))` stores `JobId() == 100` |
| equipment child collection | three equipment rows survive the round trip in slot order |

`TestPlayerNpcAdministrator` in `.../playernpc/administrator_test.go` (test-DB setup
copied from `services/atlas-notes/atlas.com/notes/note/processor_test.go`):

| case | assertion |
|---|---|
| create | row persists; equipment rows persist with the parent's id |
| duplicate name on the same map | second create violates `(tenant_id, world_id, map_id, name)` and returns an error |
| duplicate script id in the same world | violates `(tenant_id, world_id, script_id)` |
| duplicate object id on the same map | violates `(tenant_id, world_id, map_id, object_id)` |
| cascade delete | deleting the root removes its `player_npc_equipment` rows |
| cross-tenant isolation | two tenants sharing world 0 and map 102000004 see disjoint result sets |

- [ ] **Step 2: Implement**

Two GORM entities per PRD §6, with the design §3.1 addition:

`player_npcs` — `id uuid` PK, `tenant_id`, `character_id`, `name`, `world_id`, `map_id`,
`script_id`, `object_id`, `gender`, `skin`, `face`, `hair`, `job_id`, `x`, `cy`, `fh`,
`rx0`, `rx1`, `dir` (default 1), `world_rank`, `overall_rank`, `world_job_rank`,
**`overall_job_rank`**, `created_at`, `updated_at`.

`player_npc_equipment` — `id uuid` PK, `tenant_id`, `player_npc_id` FK → `player_npcs.id`
with cascade delete, `slot int16`, `item_id`.

Constraints exactly as PRD §6: unique `(tenant_id, world_id, script_id)`,
`(tenant_id, world_id, map_id, name)`, `(tenant_id, world_id, map_id, object_id)`,
`(tenant_id, player_npc_id, slot)`; indexes on `(tenant_id, world_id, map_id)` and
`(tenant_id, world_id, character_id)`.

`overall_job_rank` is a stored column, not derived: design §4.3 — under D-1 an id can be
allocated from another branch, so FR-3.7's derivation from the script id is no longer
meaningful.

Immutable model with unexported fields, accessors, a Builder, and `Make`/`MakeEntity`,
following `services/atlas-notes/atlas.com/notes/note/{model,builder,entity}.go`.
`rx0 = x + 50` and `rx1 = x - 50` are computed once in the Builder and stored, never
recomputed on read (design §3.1).

- [ ] **Step 3: Verify** — `go build ./... && go test ./...` from the module root.

### Files

- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/entity.go` — new file; both entities + `Migration`
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/model.go` — new file
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/builder.go` — new file
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/administrator.go` — new file
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/provider.go` — new file
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/model_test.go`, `administrator_test.go` — new files
- `services/atlas-notes/atlas.com/notes/note/entity.go`, `model.go`, `builder.go`, `administrator.go`, `provider.go` — read-only; the shape to copy

---

## Task 13: Read clients

- [ ] **Step 1: Write the failing test**

One `rest_test.go` per client, asserting the JSON:API decode of a captured payload —
setup copied from `services/atlas-messages/atlas.com/messages/character/rest_test.go`.

| client | case | assertion |
|---|---|---|
| `character` | decode | `name`, `gender`, `skinColor`, `face`, `hair`, `jobId`, `level`, `gm` populate |
| `character` | equipment | the `include=inventory` block yields the equipped compartment |
| `data/npc` | decode | `imitate: true` populates `Imitate()`; absent field → `false` |
| `data/map` | decode | `mapArea` bounds and the ground response decode |
| `ranking` | decode | `rank` → `Rank()`, `jobRank` → `JobRank()` |
| `ranking` | absent ranking | a 404 yields a zero-rank result, not an error (a character with no computed ranking must still deploy) |
| `configuration` | defaults | a 404 from `atlas-tenants` yields the FR-4.7 defaults |

- [ ] **Step 2: Implement**

Five read clients, each `model.go` / `rest.go` / `requests.go` / `processor.go`:

| package | upstream | request path |
|---|---|---|
| `character/` | `CHARACTERS` | `characters/%d?include=inventory`, `characters?name=%s&include=inventory` |
| `data/npc/` | `DATA` | `data/npcs/%d` |
| `data/map/` | `DATA` | `data/maps/%d` and `POST data/maps/%d/ground` (Task 4) |
| `ranking/` | `RANKINGS` | `rankings/characters/%d?filter[worldId]=%d` |
| `configuration/` | `TENANTS` | `tenants/%s/configurations/player-npcs` |

`configuration/` follows `services/atlas-rankings/atlas.com/rankings/configuration/` line
for line, including its 404-is-the-unconfigured-state handling: a missing config yields
the FR-4.7 defaults (`initialX` 262, `initialY` 262, `areaX` 320, `areaY` 160,
`areaSteps` 4, `organizeArea` true, `autoDeployEnabled` true), and any other error is
logged at warn and also falls back — one tenant's config problem must never stall
deployment.

- [ ] **Step 3: Verify** — `go build ./... && go test ./...` from the module root.

### Files

- `services/atlas-player-npcs/atlas.com/player-npcs/character/{model,rest,requests,processor}.go` — new files
- `services/atlas-player-npcs/atlas.com/player-npcs/data/npc/{model,rest,requests,processor}.go` — new files
- `services/atlas-player-npcs/atlas.com/player-npcs/data/map/{model,rest,requests,processor}.go` — new files
- `services/atlas-player-npcs/atlas.com/player-npcs/ranking/{model,rest,requests,processor}.go` — new files
- `services/atlas-player-npcs/atlas.com/player-npcs/configuration/{model,rest,requests,processor}.go` — new files
- `services/atlas-messages/atlas.com/messages/character/{rest,requests,processor}.go` — read-only; the client shape to copy
- `services/atlas-fame/atlas.com/fame/character/requests.go` — read-only; the `?include=inventory` form at :13-14
- `services/atlas-rankings/atlas.com/rankings/configuration/{rest,requests}.go` — read-only; the tenant-config client to copy
- `services/atlas-rankings/atlas.com/rankings/ranking/rest.go` — read-only; the ranking payload

This task is five clients across fifteen files, but each is the same four-file template
against a different upstream; see context.md.

---

## Task 14: `snapshot/` and `eligibility/`

- [ ] **Step 1: Write the failing test**

`TestCaptureSnapshot` in
`services/atlas-player-npcs/atlas.com/player-npcs/snapshot/snapshot_test.go`, driven by
stub read-client interfaces (no HTTP):

| case | input | expect |
|---|---|---|
| appearance | character with gender 0, skin 3, face 20000, hair 30030 | those four values on the snapshot |
| job category | `jobId` 112 | `JobId() == 100` |
| visible equip | equipped slot −5 holding a non-cash equip | equipment row `slot 5`, that item id |
| cash equip masks | equipped slot −5 with both a cash and a real equip | visible row `slot 5` = the **cash** item; masked row `slot 105` = the **real** item |
| out-of-range slot dropped | equipped slot −60 | no equipment row |
| ranks | rankings `rank` 42, `jobRank` 7 | `WorldRank() == 42`, `OverallRank() == 42` |
| no ranking | ranking lookup 404 | `WorldRank() == 0`, `OverallRank() == 0`, no error |

`TestEligible` in `.../eligibility/eligibility_test.go`:

| case | autoDeploy / level / maxLevel / gm / existing | expect |
|---|---|---|
| eligible for auto-deploy | true / 200 / 200 / false / none | `(true, "")` |
| below max level | true / 199 / 200 / false / none | `(false, "ineligible")` |
| is a GM | true / 200 / 200 / true / none | `(false, "ineligible")` |
| already deployed on the map | true / 200 / 200 / false / one | `(false, "duplicate")` |
| conversation predicate needs auto-deploy off | true / 200 / 200 / false / none, `conversationPath: true` | `(false, "ineligible")` |
| conversation predicate satisfied | false / 200 / 200 / false / none, `conversationPath: true` | `(true, "")` |

The last two rows are FR-6.1: the conversation condition additionally requires that
automatic deployment is **disabled** for the tenant. One predicate serves both callers so
FR-1.1 and FR-6.1 can never disagree (design §9.1).

- [ ] **Step 2: Implement**

`snapshot.Capture` fans out to the Task 13 clients and assembles appearance, equipment
and ranks. Equipment capture follows the repo's existing wire-slot convention from
`services/atlas-channel/atlas.com/channel/socket/model/avatar.go:10-31`: the visible list
takes `Position * -1` and prefers the cash equipable when one is present; the masked list
takes `Position * -1 + 100` and holds the real equipable only when a cash equipable masks
it. Slots outside 1–11 (visible) and 101–111 (masked) are dropped at this boundary
(FR-5.2) so no out-of-range slot can reach the codec.

`overall_rank` is set equal to `world_rank` — `atlas-rankings` exposes no cross-world
ranking (design D-3), and inventing one would be fabricating data.

`eligibility.Evaluate` takes the tenant config, the character, and the count of existing
Player NPCs for `(character name, map)` and returns `(bool, reason)` where reason is one
of the design §8.3 codes.

- [ ] **Step 3: Verify** — `go build ./... && go test ./...` from the module root.

### Files

- `services/atlas-player-npcs/atlas.com/player-npcs/snapshot/snapshot.go` — new file
- `services/atlas-player-npcs/atlas.com/player-npcs/snapshot/snapshot_test.go` — new file
- `services/atlas-player-npcs/atlas.com/player-npcs/eligibility/eligibility.go` — new file
- `services/atlas-player-npcs/atlas.com/player-npcs/eligibility/eligibility_test.go` — new file
- `services/atlas-channel/atlas.com/channel/socket/model/avatar.go` — read-only; the slot convention at :10-31
- `libs/atlas-constants/inventory/slot` — read-only; `slot.Position`, `slot.Slots`

---

## Task 15: The deploy transaction

- [ ] **Step 1: Write the failing test**

`TestDeploy` in `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/processor_test.go`
(test-DB setup copied from Task 12's `administrator_test.go`):

| case | assertion |
|---|---|
| happy path | one row created; script id from the branch; position from the positioner; `DEPLOYED` emitted after commit |
| rank ordinals | first deploy for `(world 0, job 100)` gets `world_job_rank == 1`; second gets `2` |
| overall job ordinal | independent counter over `(tenant, job 100)`, `1` then `2` |
| duplicate | a second deploy for the same `(name, map)` returns `duplicate` and creates no row |
| pool exhausted | every usable id in use → `pool_exhausted`, no row, no event |
| map full | positioner returns `ErrMapFull` → `map_full`, no row, no event |
| ineligible on the checked path | `enforceEligibility: true` with a level below max → `ineligible` |
| GM path bypasses level | `enforceEligibility: false` with a level below max → succeeds |
| allocation does not leak on failure | a positioner failure after allocation rolls back; the script id is free on the next deploy |
| re-deploy in place | `Redeploy` refreshes appearance and ranks; script id, object id and position are unchanged; `UPDATED` emitted |
| remove | `Remove` deletes the row and its equipment rows; `REMOVED` emitted |
| reorganize | a map with no free slot at the current step repositions every NPC at `step+1` in one transaction, then emits one `REPOSITIONED` |

- [ ] **Step 2: Implement**

`Deploy` runs exactly the design §8.1 sequence inside **one** transaction:

```
BEGIN
  pg_advisory_xact_lock(hash(tenant, world, map))
  read in-use script ids and placed rectangles for (tenant, world[, map])
  allocate script id                        (allocation.Allocate)
  choose a position                         (position.NextGridPosition / PodiumPosition)
  objectId = objectid.PlayerNpcObjectIdFor(scriptId)
  compute rank ordinals                     (MAX(rank)+1 over the matching rows)
  INSERT player_npcs + player_npc_equipment
COMMIT
then emit DEPLOYED
```

The advisory lock is transaction-scoped (`pg_advisory_xact_lock`), so it releases on
commit or rollback with no cleanup path and no possibility of a leak. `libs/atlas-lock`
is a leader-election lease and is the wrong tool here (design D-6).

The event is emitted **after** commit, never inside the transaction: a client must never
be told about an NPC that then rolls back. The reverse failure — commit succeeds, emit
fails — leaves a persisted NPC that appears on the next map enter, which is the
acceptable direction (design §8.1).

Re-organization takes the same lock, persists all new positions in one transaction, and
emits a single `REPOSITIONED` carrying the full list — persist-then-broadcast, never the
reverse (design §5.4).

Failures map to the design §8.3 codes: `pool_exhausted`, `map_full`, `duplicate`,
`ineligible`.

- [ ] **Step 3: Verify** — `go build ./... && go test ./...` from the module root.

### Files

- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/processor.go` — new file
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/errors.go` — new file; the four failure codes
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/processor_test.go` — new file
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/administrator.go` — new file from Task 12; add the advisory-lock helper and the ordinal queries
- `services/atlas-player-npcs/atlas.com/player-npcs/allocation/allocation.go`, `position/grid.go`, `position/podium.go`, `snapshot/snapshot.go`, `eligibility/eligibility.go` — from Tasks 10, 11, 14
- `libs/atlas-object-id/reserved.go` — new file from Task 3; read-only here
- `libs/atlas-database` — read-only; `ExecuteTransaction`

---

## Task 16: REST resource

- [ ] **Step 1: Write the failing test**

`TestPlayerNpcResource` in
`services/atlas-player-npcs/atlas.com/player-npcs/playernpc/resource_test.go`
(HTTP-handler setup copied from `services/atlas-notes/atlas.com/notes/note/resource.go`'s
neighbouring tests):

| case | request | expect |
|---|---|---|
| list by map | `GET /api/player-npcs?filter[mapId]=102000004&filter[worldId]=0` | 200, only that map's rows |
| list is tenant-scoped | same request under a second tenant sharing world 0 | 200, disjoint set |
| pagination | `GET /api/player-npcs?page[size]=1` | 200, one item, standard envelope |
| get one | `GET /api/player-npcs/{id}` | 200 |
| get missing | `GET /api/player-npcs/{unknown}` | 404 |
| deploy | `POST /api/player-npcs` with `characterId`, `worldId`, `mapId` | 201 with the created resource |
| deploy with explicit position | same plus `position: {x, y}` | 201; that position is used, not the positioner |
| duplicate | second `POST` for the same `(name, map)` | 409 with `code: "duplicate"` |
| pool exhausted | `POST` with the pool full | 409 with `code: "pool_exhausted"` |
| map full | `POST` with the map full | 409 with `code: "map_full"` |
| ineligible | eligibility-checked `POST` below max level | 409 with `code: "ineligible"` |
| unresolvable character | `POST` with an unknown `characterId` | 422 |
| re-deploy | `PATCH /api/player-npcs/{id}` | 200; script id, object id and position unchanged |
| delete one | `DELETE /api/player-npcs/{id}` | 204 |
| delete by character | `DELETE /api/player-npcs?filter[characterId]=1` | 204; all of that character's rows gone |
| delete by character and map | same plus `&filter[mapId]=102000004` | 204; only that map's row gone |
| eligibility endpoint | `GET /api/player-npcs/eligibility?characterId=1&mapId=102000004` | 200 with `{eligible, reason}` |

- [ ] **Step 2: Implement**

Routes registered under `PathPrefix("/player-npcs")`, matching the PRD §5 surface. The
resource shape is PRD §5 exactly, plus `overallJobRank` (which PRD §5 already lists and
PRD §6 omitted — design §3.1).

Register `/eligibility` **before** the `/{id}` pattern so a route is not shadowed, the
same ordering hazard the tenant configuration routes document at
`services/atlas-tenants/atlas.com/tenants/configuration/resource.go:1552-1554`.

Wire `playernpc.InitializeRoutes(GetServer())(db)` into the Task 8 `main.go`.

- [ ] **Step 3: Verify** — `go build ./... && go test ./...` from the module root.

### Files

- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/rest.go` — new file
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/resource.go` — new file
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/requests.go` — new file
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/resource_test.go` — new file
- `services/atlas-player-npcs/atlas.com/player-npcs/main.go` — new file from Task 8; add the route initializer
- `services/atlas-notes/atlas.com/notes/note/{rest,resource}.go` — read-only; the resource shape to copy
- `services/atlas-tenants/atlas.com/tenants/configuration/resource.go` — read-only; the route-ordering hazard at :1552

---

## Task 17: Kafka — messages, producer, consumers

- [ ] **Step 1: Write the failing test**

`TestPlayerNpcCommandConsumer` in
`services/atlas-player-npcs/atlas.com/player-npcs/kafka/consumer/playernpc/consumer_test.go`:

| case | command | expect |
|---|---|---|
| deploy | `DEPLOY` with `characterId`, `worldId`, `mapId`, `enforceEligibility: true` | processor `Deploy` called with those values |
| deploy with position | `DEPLOY` with `position` | that position passed through |
| redeploy | `REDEPLOY` with `characterId`, `mapId` | `Redeploy` called |
| remove | `REMOVE` with `characterId` and no `mapId` | `Remove` called with a nil map scope |
| remove map-scoped | `REMOVE` with `mapId` | `Remove` called with that map |

`TestLevelChangedConsumer` in `.../kafka/consumer/character/consumer_test.go`:

| case | event body | tenant config | expect |
|---|---|---|---|
| at max level, eligible | `current: 200` | `autoDeployEnabled: true` | character fetched; `DEPLOY` command emitted for `HallOfFameMapFor(job)` |
| below max level | `current: 199` | any | **no character fetch at all**, no command |
| auto-deploy disabled | `current: 200` | `autoDeployEnabled: false` | no command |
| character is a GM | `current: 200` | enabled | no command |
| already deployed | `current: 200` | enabled | no command |
| fetch fails | `current: 200` | enabled | logged at warn with character id, target map and reason; **no error propagated** |
| duplicate redelivery | the same event twice | enabled | one command; the storage-layer unique constraint is the backstop |

The "below max level" row asserts the cheap path: the level check happens **before** any
fetch, because that is the overwhelmingly common case (design §8.2).

- [ ] **Step 2: Implement**

Message definitions in `kafka/message/playernpc/kafka.go`:

- Commands on `COMMAND_TOPIC_PLAYER_NPC`: `DEPLOY` (characterId, worldId, mapId, optional
  position, `enforceEligibility`), `REDEPLOY` (characterId, mapId), `REMOVE` (characterId,
  optional mapId).
- Events on `EVENT_TOPIC_PLAYER_NPC_STATUS`: `DEPLOYED` (full resource payload),
  `UPDATED` (same), `REMOVED` (id, objectId, mapId, worldId), `REPOSITIONED`
  (mapId, worldId, and a list of `{id, objectId, x, cy, fh, rx0, rx1}`).

The `LEVEL_CHANGED` consumer reads
`services/atlas-character/atlas.com/character/kafka/message/character/kafka.go:307-311`
(`{channelId, amount, current}`) — no job, no gm flag, no map — so it fetches the
character to evaluate the remaining conditions. That fetch is on the deployment path
anyway, so **no change to `atlas-character` is required**, contrary to PRD §7's
conditional. Every failure is logged at warn and swallowed: a failed deployment must never
block or roll back a level-up (FR-1.5).

Wire both consumers and the producer into the Task 8 `main.go`, following
`services/atlas-notes/atlas.com/notes/main.go:71-80`.

- [ ] **Step 3: Verify** — `go build ./... && go test ./...` from the module root.

### Files

- `services/atlas-player-npcs/atlas.com/player-npcs/kafka/message/playernpc/kafka.go` — new file
- `services/atlas-player-npcs/atlas.com/player-npcs/kafka/message/character/kafka.go` — new file; the `LEVEL_CHANGED` envelope
- `services/atlas-player-npcs/atlas.com/player-npcs/kafka/consumer/playernpc/consumer.go` — new file
- `services/atlas-player-npcs/atlas.com/player-npcs/kafka/consumer/character/consumer.go` — new file
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/producer.go` — new file; the four status events
- `services/atlas-player-npcs/atlas.com/player-npcs/main.go` — new file from Task 8; consumer + producer wiring
- `services/atlas-character/atlas.com/character/kafka/message/character/kafka.go` — read-only; `LevelChangedStatusEventBody` at :307
- `services/atlas-notes/atlas.com/notes/kafka/consumer/character/consumer.go` — read-only; the consumer shape to copy

---

## Task 18: `atlas-channel` — `playernpc/` read client

- [ ] **Step 1: Write the failing test**

`TestPlayerNpcRest` in `services/atlas-channel/atlas.com/channel/playernpc/rest_test.go`,
copied from `services/atlas-channel/atlas.com/channel/kite/rest.go`'s neighbouring test:

| case | assertion |
|---|---|
| decode | every PRD §5 attribute populates, including `overallJobRank` |
| equipment | the `equipment` array yields `(slot, itemId)` pairs in order |
| by map | `ForEachInMap` requests `player-npcs?filter[mapId]=%d&filter[worldId]=%d` |

- [ ] **Step 2: Implement**

A `playernpc/` package mirroring `services/atlas-channel/atlas.com/channel/kite/`:
`model.go`, `builder.go`, `rest.go`, `requests.go`, `processor.go`. `processor.go` exposes
`ForEachInMap(f field.Model, op model.Operator[Model]) error` against
`GET /api/player-npcs?filter[mapId]&filter[worldId]`, using `requests.RootUrlFor(ctx, "PLAYER_NPCS")`.

- [ ] **Step 3: Verify** — `go build ./... && go test ./...` from
  `services/atlas-channel/atlas.com/channel`.

### Files

- `services/atlas-channel/atlas.com/channel/playernpc/{model,builder,rest,requests,processor}.go` — new files
- `services/atlas-channel/atlas.com/channel/playernpc/rest_test.go` — new file
- `services/atlas-channel/atlas.com/channel/kite/{model,builder,rest,requests,processor}.go` — read-only; the client shape to copy

---

## Task 19: `atlas-channel` — spawn, broadcast, controller exclusion

- [ ] **Step 1: Write the failing test**

`TestSpawnPlayerNpcForSession` in
`services/atlas-channel/atlas.com/channel/kafka/consumer/map/player_npc_test.go`:

| case | assertion |
|---|---|
| ordering | for one Player NPC, `SpawnNPC` is written **strictly before** `ImitatedNPCData` — the client needs the object in its pool before the avatar data can attach (FR-7.1) |
| no controller grant | no `SpawnNPCRequestController` packet is written for a Player NPC (FR-7.4) |
| batched data | three Player NPCs on the map produce three `SpawnNPC` packets and **one** `ImitatedNPCData` carrying three entries |
| no controller hand-off on exit | after a Player NPC is spawned, a `CharacterExit` produces no controller hand-off for its object id (FR-7.4) |

`TestPlayerNpcStatusConsumer` in `.../kafka/consumer/playernpc/consumer_test.go`:

| event | expect |
|---|---|
| `DEPLOYED` | `SpawnNPC` then `ImitatedNPCData` to every character on that map, on every channel of the world |
| `UPDATED` | `ImitatedNPCData` only — no despawn/respawn; the object has not moved |
| `REMOVED` | `RemoveNPC` (Task 6) for the object id, to everyone on the map on every channel |
| `REPOSITIONED` | per listed object, `RemoveNPC` then `SpawnNPC` then one `ImitatedNPCData` for the whole list |

- [ ] **Step 2: Implement**

1. A new `routine.Go` block in `SpawnForSelf`
   (`services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:229-232` is the
   NPC block to sit alongside), iterating `playernpc.NewProcessor(l, ctx).ForEachInMap`
   and issuing `npcpkt.NewNpcSpawn(...)` per NPC, then one
   `npccb.NewImitatedNpcData(entries)` for the map.
2. Player NPCs are materialized with **plain `SPAWN_NPC` and no controller grant**
   (design D-4). Ordinary NPCs are spawned by `NpcSpawnWriter` and only *then* optionally
   granted control (`consumer.go:656-681`, `spawnNPCForSession`) — the object exists after
   the spawn packet and the grant is a separate, optional step. Player NPCs take the first
   half and skip the second.
3. Player NPC object ids are never passed to `npc/controller`'s `TryClaim`, `ElectFor` or
   `ReleaseFor`. No special-casing is needed in the exit path
   (`consumer.go:620-640`): `ReleaseFor` only returns ids the registry holds, and a Player
   NPC id is never entered into it. The exit test above asserts this rather than assuming it.
4. A `kafka/consumer/playernpc/` consumer for the four status events, wired into
   `main.go` alongside `kiteconsumer` (`main.go:250`, `main.go:555`).
5. Register `npccb.NpcImitatedDataWriter` and `npccb.NpcRemoveWriter` in the writer list
   at `services/atlas-channel/atlas.com/channel/main.go:787-795`.

- [ ] **Step 3: Verify** — `go build ./... && go test ./...` from
  `services/atlas-channel/atlas.com/channel`.

### Files

- `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go` — the `SpawnForSelf` block (near :229)
- `services/atlas-channel/atlas.com/channel/kafka/consumer/map/player_npc.go` — new file; the spawn helper
- `services/atlas-channel/atlas.com/channel/kafka/consumer/map/player_npc_test.go` — new file
- `services/atlas-channel/atlas.com/channel/kafka/consumer/playernpc/consumer.go` — new file
- `services/atlas-channel/atlas.com/channel/kafka/consumer/playernpc/kafka.go` — new file
- `services/atlas-channel/atlas.com/channel/kafka/consumer/playernpc/consumer_test.go` — new file
- `services/atlas-channel/atlas.com/channel/main.go` — consumer registration (:250, :555) and writer list (:787)
- `services/atlas-channel/atlas.com/channel/npc/controller/processor.go` — read-only; `TryClaim`/`ElectFor`/`ReleaseFor`

This task touches one service across seven files, six of them new; see context.md.

---

## Task 20: `atlas-tenants` — the `player-npcs` configuration resource

- [ ] **Step 1: Write the failing test**

`TestPlayerNpcConfigHandlerWireRoundTrip` in
`services/atlas-tenants/atlas.com/tenants/configuration/player_npc_config_handler_test.go`,
copied from `configuration/rankings_handler_test.go:84-217`:

| case | assertion |
|---|---|
| create then get | `POST` the seven values, `GET` returns them unchanged through the real JSON:API codec |
| update | `PATCH` changes `areaSteps` only; the other six are unchanged |
| delete | `DELETE` then `GET` returns 404 |
| unconfigured | `GET` before any create returns 404 |
| type fidelity | `organizeArea`/`autoDeployEnabled` survive as booleans, not strings |

- [ ] **Step 2: Implement**

Follow the **kite-config** shape, which the repo already documents as the
"rankings shape: one config per tenant, no `/seed` endpoint, no id-addressed
sub-resource" (`configuration/resource.go:1613-1614`). Add, mirroring `KiteConfig`:

| file | additions |
|---|---|
| `configuration/rest.go` | `PlayerNpcConfigRestModel` (`Id string; InitialX, InitialY, AreaX, AreaY int16; AreaSteps byte; OrganizeArea, AutoDeployEnabled bool`), `TransformPlayerNpcConfig`, `ExtractPlayerNpcConfig`, `CreateSinglePlayerNpcConfigJsonData` |
| `configuration/processor.go` | `Create/Update/Delete/Get` + the `AndEmit` variants and the provider, on the `Processor` interface and `ProcessorImpl` |
| `configuration/mock/processor.go` | matching mock funcs |
| `configuration/provider.go` | `GetByTenantIdAndResourceNameProvider(tenantID, "player-npcs")` |
| `configuration/kafka.go` | `CreatePlayerNpcConfigStatusEventProvider` with `ResourceType: "player-npcs"` |
| `configuration/resource.go` | the four handlers and their four routes at `/tenants/{tenantId}/configurations/player-npcs`, registered in `RegisterRoutes` after the kite-config block |

The attribute names must match exactly what the Task 13 `configuration/` client decodes —
the rankings resource documents this coupling at `configuration/rest.go:817-820`.

- [ ] **Step 3: Verify** — `go build ./... && go test ./...` from
  `services/atlas-tenants/atlas.com/tenants`.

### Files

- `services/atlas-tenants/atlas.com/tenants/configuration/rest.go`
- `services/atlas-tenants/atlas.com/tenants/configuration/processor.go`
- `services/atlas-tenants/atlas.com/tenants/configuration/provider.go`
- `services/atlas-tenants/atlas.com/tenants/configuration/kafka.go`
- `services/atlas-tenants/atlas.com/tenants/configuration/resource.go`
- `services/atlas-tenants/atlas.com/tenants/configuration/mock/processor.go`
- `services/atlas-tenants/atlas.com/tenants/configuration/player_npc_config_handler_test.go` — new file
- `services/atlas-tenants/atlas.com/tenants/configuration/rankings_handler_test.go` — read-only; the test to copy

Six files in one service, all the same resource added to one aggregate; see context.md.

---

## Task 21: `atlas-messages` — GM commands

- [ ] **Step 1: Write the failing test**

`TestPlayerNpcCommands` in
`services/atlas-messages/atlas.com/messages/command/playernpc/commands_test.go`, copied
from `command/monster/commands_test.go`:

| case | chat message | gm | expect |
|---|---|---|---|
| deploy matches | `@playernpc add Hero` | true | producer returns an executor |
| deploy, non-GM | `@playernpc add Hero` | false | `(nil, false)` — the command does not match |
| deploy emits | `@playernpc add Hero` | true | a `DEPLOY` command on `COMMAND_TOPIC_PLAYER_NPC` with the GM's current map and position and `enforceEligibility: false` |
| remove all | `@playernpc remove Hero` | true | a `REMOVE` command with no map scope |
| remove map-scoped | `@playernpc remove Hero here` | true | a `REMOVE` command scoped to the GM's current map |
| unknown character | `@playernpc add Nobody` | true | pink text to the invoking GM: character not found; no command emitted |
| non-matching text | `@playernpcs add Hero` | true | `(nil, false)` |
| failure reported back | the service replies `pool_exhausted` | true | pink text naming `pool_exhausted` to the invoking GM |

- [ ] **Step 2: Implement**

A `command/playernpc/commands.go` exporting `DeployCommandProducer` and
`RemoveCommandProducer` in the `Producer`/`Executor` shape of
`services/atlas-messages/atlas.com/messages/command/types.go`, registered in
`services/atlas-messages/atlas.com/messages/main.go` alongside the existing
`command.Registry().Add(...)` calls (:69-88).

The named character is resolved with the existing
`character.NewProcessor(l, ctx).GetByName(...)`
(`services/atlas-messages/atlas.com/messages/character/processor.go:52`). GM gating uses
the existing `c.Gm()` check every other command already uses
(`command/monster/commands.go:47`) — no new authorization mechanism.

Deploy bypasses the level and auto-deploy checks (`enforceEligibility: false`) but still
honours script-id availability and the per-map duplicate rule (FR-8.1). Both commands
report success or the specific design §8.3 reason back to the invoking GM via
`message.NewProcessor(l, ctx).IssuePinkText(...)` (FR-8.3).

These commands live in `atlas-messages`, not `atlas-channel` — `atlas-channel` has no
command infrastructure at all (design §1 C-3).

- [ ] **Step 3: Verify** — `go build ./... && go test ./...` from
  `services/atlas-messages/atlas.com/messages`.

### Files

- `services/atlas-messages/atlas.com/messages/command/playernpc/commands.go` — new file
- `services/atlas-messages/atlas.com/messages/command/playernpc/commands_test.go` — new file
- `services/atlas-messages/atlas.com/messages/kafka/message/playernpc/kafka.go` — new file; the command envelope
- `services/atlas-messages/atlas.com/messages/main.go` — two `Registry().Add` lines near :88
- `services/atlas-messages/atlas.com/messages/command/monster/commands.go` — read-only; the producer shape at :36-90
- `services/atlas-messages/atlas.com/messages/command/types.go` — read-only; `Producer`/`Executor`
- `services/atlas-messages/atlas.com/messages/character/processor.go` — read-only; `GetByName` at :52

---

## Task 22: Conversation-engine condition and operation

- [ ] **Step 1: Write the failing test**

`TestCanSpawnPlayerNpcCondition` in
`services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model_test.go`
(append to the existing table there):

| case | eligibility endpoint reply | expect |
|---|---|---|
| eligible | `{eligible: true}` | condition passes, `ActualValue == 1` |
| not eligible | `{eligible: false, reason: "ineligible"}` | condition fails, `ActualValue == 0`, the description names the reason |
| endpoint unreachable | error | condition fails; graceful degradation, no panic |
| missing referenceId | input with `referenceId == 0` | builder error: `referenceId (mapId) is required for canSpawnPlayerNpc conditions` |

`TestDeployPlayerNpcAction` in
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler_test.go`:

| case | payload | expect |
|---|---|---|
| explicit map | `{characterId, mapId}` | a `DEPLOY` command with that map |
| default map | `{characterId}` with no `mapId` | a `DEPLOY` command with the character's current map |
| failure code surfaces | the service replies `pool_exhausted` | the step's failure status carries `pool_exhausted` so a script can branch (FR-6.3) |

- [ ] **Step 2: Implement**

**Condition (FR-6.1).**

1. `CanSpawnPlayerNpcCondition = "canSpawnPlayerNpc"` in `libs/atlas-saga/validation.go`
   (the constant block at :10-46).
2. Re-export it in
   `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model.go:50`,
   add it to the valid-type `case` list (:130), the two `referenceId`-required
   validations (:255 and :333), the `rest.go` input validation (:271), the evaluation
   `switch` (:809), and `requiresContextPath` (`processor.go:171-178`) — it reads a lazy
   processor, so it must take the context route.
3. A `GetPlayerNpcEligibility(mapId)` lazy accessor on `ValidationContext`
   (`validation/context.go`), modelled on `GetTransportState` at :333-354, calling
   `GET /api/player-npcs/eligibility?characterId&mapId` and degrading gracefully.

Putting the predicate behind the single endpoint rather than reassembling it from four
separate conditions keeps FR-1.1's automatic check and FR-6.1's conversation check on one
code path, so they cannot disagree (design §9.1).

**Operation (FR-6.2).**

4. `DeployPlayerNpc Action = "deploy_player_npc"` in `libs/atlas-saga/model.go` (the
   action block at :90-167), a `DeployPlayerNpcPayload` (`CharacterId uint32; MapId *uint32`)
   in `libs/atlas-saga/payloads.go`, and its `case` in `libs/atlas-saga/unmarshal.go`
   (alongside `SpawnMonster` at :318).
5. `handleDeployPlayerNpc` in
   `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go`, its
   entry in the action dispatch (:881), the re-export in `saga/model.go` (:149, :314),
   the `event_acceptance.go` entry (:319), and the `character_extractor.go` case (:47).
   It emits the `DEPLOY` command; it is **not** a local operation
   (`atlas-npc-conversations`' `isLocalOperationType`,
   `conversation/operation_executor.go:250`) because it mutates cross-service state.

**Documentation.** Add the condition to
`services/atlas-query-aggregator/docs/rest.md` (the table at :123) and
`services/atlas-query-aggregator/docs/domain.md` (:54 and the context-dependent list at
:150), and the operation to `docs/npc_conversation_conversion_spec.md` (the operation list
at :463). No JSON-schema change is needed — neither
`services/atlas-npc-conversations/docs/npc_conversation_schema.json` nor
`services/atlas-ui/docs/npc_conversation_schema.json` enumerates condition or operation
types. Authoring the instructor conversation JSON is out of scope (FR-6.4).

- [ ] **Step 3: Verify**

```
go build ./... && go test ./...    # from libs/atlas-saga
go build ./... && go test ./...    # from services/atlas-query-aggregator/atlas.com/query-aggregator
go build ./... && go test ./...    # from services/atlas-saga-orchestrator/atlas.com/saga-orchestrator
```

### Files

- `libs/atlas-saga/validation.go` — the condition constant (:10-46)
- `libs/atlas-saga/model.go` — the action constant (:90-167)
- `libs/atlas-saga/payloads.go` — `DeployPlayerNpcPayload`
- `libs/atlas-saga/unmarshal.go` — the payload case (near :318)
- `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model.go` — :50, :130, :255, :333, :809
- `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/rest.go` — :271
- `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/processor.go` — :171
- `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/context.go` — the lazy accessor, near :333
- `services/atlas-query-aggregator/atlas.com/query-aggregator/playernpc/{rest,requests,processor}.go` — new files; the eligibility client
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go` — :881 and the new handler
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go` — :149, :314
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go` — :319
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/character_extractor.go` — :47
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/playernpc/{processor,requests}.go` — new files
- `services/atlas-query-aggregator/docs/rest.md`, `services/atlas-query-aggregator/docs/domain.md`, `docs/npc_conversation_conversion_spec.md` — documentation
- `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model_test.go`, `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler_test.go` — new cases

This task spans three modules because one condition and one action must be declared in the
shared library and consumed in both services; splitting it would leave a module that does
not compile. See context.md.

---

## Final gate

Before requesting code review:

```
tools/verify.sh                          # flagless; must exit 0
go run ./tools/packet-audit matrix --check
tools/service-registration-guard.sh
tools/plan-lint.sh docs/tasks/task-251-player-npcs/plan.md
```

Then run the code-review step: `atlas-reviewer` per task commit range,
`backend-guidelines-reviewer` over the changed Go packages,
`plan-adherence-reviewer` over this plan, and `packet-completeness-critic` for
Tasks 5–7.
