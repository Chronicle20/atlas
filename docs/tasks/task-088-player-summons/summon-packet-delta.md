# Player Summons — Per-Version Packet / Opcode Delta

Source of truth for task-088 Phase 6 (version-conditional summon encode/decode +
per-version opcode seeding). Drives Task 6.2 (encode/decode gating) and Task 6.3
(template opcode seeding).

Every row cites IDA-confirmed (decompiled, named symbols where present), Cosmic-derived,
or derived-unverified. The v83 baseline folds in the earlier `summon-opcodes-v83.md`
harvest (now superseded by this file).

## 0. IDB inventory & dispatch anchors

All target IDBs reachable simultaneously via multi-instance IDA-MCP (`list_instances`,
2026-06-12). The summon recv (server→client / Atlas WRITERS) dispatch is a stable
two-stage shape across GMS v83/v84/v87/JMS185 and a **restructured** shape in v95:

| Version | port | `CUserPool::OnUserCommonPacket` | summon dispatch fn | naming | spawn-reader call |
|---|---|---|---|---|---|
| v83 GMS | 13337 | `0x972401` (named) | `CSummonedPool::OnPacket` @ `0x938dd7` (named) | dense | `(*(*this+0x2C))` vtable+44 |
| v84 GMS | 13341 | `0x9b23a1` (named) | `sub_970201` (summon-band sub) | partial | `(*(*this+0x30))` vtable+48 |
| v87 GMS | 13338 | `0x9f7387` (named) | `CSummonedPool::OnPacket` @ `0x9b35bf` (named) | dense | `(*(*this+0x30))` vtable+48 |
| v95 GMS | 13339 | `0x94cdb0` (named) — **no summon case** | `CSummonedPool::OnPacket` @ `0x75ac70` (named), called from `CField::OnPacket` @ `0x546d50` | dense | n/a — flat switch, all 6 via `Decode4 charId` + leaf |
| JMS v185 | 13340 | `0xa440d7` (named) | `CSummonedPool::OnPacket` @ `0x9f7f6e` (named) | dense | `(*(*this+0x30))` vtable+48 |

### Dispatch shape (v83 / v84 / v87 / JMS185 — "classic")
`CUserPool::OnUserCommonPacket` decodes `Decode4 characterId`, resolves the CUser,
then routes contiguous opcode bands: chat/adboard/balloon/consume/upgrade cases,
then `pet band → CUser::OnPetPacket`, `summon band (6 ops) → CSummonedPool::OnPacket`,
`dragon band (3 ops) → CUser::OnDragonPacket`. `CSummonedPool::OnPacket` then:
`if (op==spawn) (*(*this+vtable))(outBuf,pkt)` (enter-field reader); else
`Decode4 oid`, look up the summon, `if (op==remove) …remove…`, then
`switch: move/attack/hit/skill`. **Byte-identical EH-prolog + branch structure
across all four classic versions** — only the opcode immediates differ.

### v95 restructure (IDA-confirmed)
v95 moved summons OUT of the CUser common band entirely. `CUserPool::OnUserCommonPacket`
@ `0x94cdb0` has **no** `CSummonedPool` case (its tail is pet band 198–205, dragon
206–208). Instead `CField::OnPacket` @ `0x546d50` calls `CSummonedPool::OnPacket`
@ `0x75ac70`, which is a **flat 6-case switch (0x116–0x11B)** that reads `Decode4
characterId` once up front for every op (spawn included) and dispatches to named
`OnCreated/OnRemoved/OnMove/OnAttack/OnSkill/OnHit` leaves. **Note the case order:
… OnAttack(0x119), OnSkill(0x11A), OnHit(0x11B)** — Skill and Hit are swapped vs the
classic Move/Attack/Hit/Skill order.

## 1. Writer (server → client) opcodes — Atlas WRITERS

These are the opcodes the CLIENT receives & dispatches; Atlas must seed them as
`socket.writers[]`. **All five versions IDA-confirmed** (decompiled dispatch).

| Atlas writer | v83 | v84 | v87 | v95 | JMS185 | confirmation |
|---|---|---|---|---|---|---|
| SummonSpawn  (Created)  | 0xAF (175) | 0xB3 (179) | 0xBC (188) | **0x116 (278)** | 0xB5 (181) | IDA all |
| SummonRemove (Removed)  | 0xB0 (176) | 0xB4 (180) | 0xBD (189) | **0x117 (279)** | 0xB6 (182) | IDA all |
| SummonMove              | 0xB1 (177) | 0xB5 (181) | 0xBE (190) | **0x118 (280)** | 0xB7 (183) | IDA all |
| SummonAttack            | 0xB2 (178) | 0xB6 (182) | 0xBF (191) | **0x119 (281)** | 0xB8 (184) | IDA all |
| SummonDamage (OnHit)    | 0xB3 (179) | 0xB7 (183) | 0xC0 (192) | **0x11B (283)** | 0xB9 (185) | IDA all |
| SummonSkill             | 0xB4 (180) | 0xB8 (184) | 0xC1 (193) | **0x11A (282)** | 0xBA (186) | IDA all |

Per-version anchors:
- **v83** `CSummonedPool::OnPacket@0x938dd7`: spawn `if(a2==0xAF)`, remove `0xB0`,
  `case 0xB1` OnMove, `0xB2` OnAttack, `0xB3` OnHit, `0xB4` OnSkill. Summon band routed
  from `OnUserCommonPacket@0x972401`: `if (v6>=0xAF && v6<=0xB4) CSummonedPool::OnPacket`.
- **v84** `sub_970201` (reached from `OnUserCommonPacket@0x9b23a1`, `if (v6>=179 && v6<=184)`):
  spawn `if(a2==179)` vtable+48, remove `180`, `case 181/182/183/184` =
  Move/Attack/Hit/Skill → 0xB3–0xB8. **+4 vs v83** (pet band widened to 171–178).
- **v87** `CSummonedPool::OnPacket@0x9b35bf` (band `188–193` from `OnUserCommonPacket@0x9f7387`):
  spawn `0xBC` vtable+48, remove `0xBD`, `case 0xBE/0xBF/0xC0/0xC1` =
  Move/Attack/Hit/Skill → 0xBC–0xC1. **+13 vs v83** (extra common case `OnHitByUser@0xB3`,
  pet band 180–187).
- **v95** `CSummonedPool::OnPacket@0x75ac70` (called from `CField::OnPacket@0x546d50`):
  flat switch `case 0x116` OnCreated, `0x117` OnRemoved, `0x118` OnMove, `0x119` OnAttack,
  `0x11A` OnSkill, `0x11B` OnHit. **Skill/Hit order swapped.**
- **JMS185** `CSummonedPool::OnPacket@0x9f7f6e` (band `181–186` from `OnUserCommonPacket@0xa440d7`):
  spawn `0xB5` vtable+48, remove `0xB6`, `case 0xB7/0xB8/0xB9/0xBA` =
  Move/Attack/Hit/Skill → 0xB5–0xBA. **+6 vs v83.**

### Derived, no IDB (mark unverified — confirm against capture)
| version | Spawn | Remove | Move | Attack | Damage | Skill | basis |
|---|---|---|---|---|---|---|---|
| v12 (gms_12) | 0xAF? | 0xB0? | 0xB1? | 0xB2? | 0xB3? | 0xB4? | **derived, unverified** — v12 brackets below v83; oldest GMS summon band historically tracks v83's 0xAF–0xB4. No IDB. Confirm against capture. |
| v92 (gms_92) | 0xC2? | 0xC3? | 0xC4? | 0xC5? | 0xC6? | 0xC7? | **derived, unverified** — v92 sits between v87 (0xBC base) and v95 (0x116 base, restructured). Linear-interpolating the classic band gives ≈0xC2–0xC7, BUT v92 may already be on the v95 restructured `CField`-routed high-band (0x11x). No IDB — **must capture-confirm both the band AND whether it uses the classic-vtable or v95-flat dispatch** before seeding. |

## 2. Handler (client → server) opcodes — Atlas HANDLERS

Client SEND opcodes (Atlas `socket.handlers[]`). Unlike writers these are NOT a
dispatch switch — they are distributed `COutPacket::COutPacket(long)` call sites
(see task-083 §1). Not isolated in this harvest. v83 values are Cosmic-derived:

| Atlas handler | v83 (Cosmic) | v84 | v87 | v95 | JMS185 | confirmation |
|---|---|---|---|---|---|---|
| SummonMoveHandle   | 0xAF (175) | unconfirmed | unconfirmed | unconfirmed | unconfirmed | Cosmic-derived (v83); others **derived-unconfirmed** |
| SummonAttackHandle | 0xB0 (176) | unconfirmed | unconfirmed | unconfirmed | unconfirmed | Cosmic-derived (v83); others **derived-unconfirmed** |
| SummonDamageHandle | 0xB1 (177) | unconfirmed | unconfirmed | unconfirmed | unconfirmed | Cosmic-derived (v83); others **derived-unconfirmed** |

The v83 send table is independent of the recv table (recv MOVE_SUMMON 0xAF collides
numerically with the send-side SummonSpawn 0xAF but they are distinct directions).
For v84+/v95/JMS the send opcodes almost certainly shift with the same per-version
band pressure as the writers, but they were not byte-read in this harvest. Treat as
**derived-unconfirmed** and resolve from `COutPacket(long)` send-site xrefs or a live
capture when wiring per-version handler seeding (Task 6.3).

## 3. Packet layout deltas

### 3.1 SummonSpawn (enter-field)

v83 atlas layout (`libs/atlas-packet/summon/clientbound/spawn.go`, Cosmic-derived,
tested against live v83):

```
int   ownerId
int   oid
int   skillId
byte  0x0A            // "marker" — see note
byte  level
short x
short y
byte  stance
short 0               // reserved
byte  movementType
bool  !puppet         // attack flag
bool  !animated
```

**IDA finding (v95 `CSummoned::Init(packet)` @ `0x755740`, named) — definitive field semantics:**
the v95 reader decodes, after `Decode4 charId / Decode4 oid / Decode4 skillId /
Decode1 / Decode1`:
```
short nX            (Decode2)   = atlas x
short nY            (Decode2)   = atlas y
byte  nMoveAction   (Decode1)   = atlas stance
short nCurFoothold  (Decode2)   = atlas "reserved short 0"  (it is the FOOTHOLD id, not padding)
byte  nMoveAbility  (Decode1)   = atlas movementType
byte  nAssistType   (Decode1)   = atlas !puppet flag
byte  nEnterType    (Decode1)   = atlas !animated flag
byte  bAvatarLook   (Decode1)   = NEW in v95 (avatar-look-present flag)
  if bAvatarLook: AvatarLook::Decode(pkt)               // NEW in v95
  if skillId==35111002 (Tesla Coil): byte state + 3×(short,short) triangle   // NEW in v95
```
and the two leading `Decode1`s before position are **charLevel** then **SLV
(skill level)** — i.e. the v83 "0x0A marker" byte position is **charLevel** and the
next is the skill level. The classic readers (`sub_7A379B` v83 / `sub_7F504A` v87,
both byte-identical) read the SAME position tail (`Decode2 x, Decode2 y, Decode1
moveAction, Decode2 foothold, Decode1 moveAbility, Decode1 assistType, Decode1
enterType, Decode1`) and pass the trailing byte as the `AvatarLook*` slot — i.e.
**no avatar-look blob, no Tesla special** in the classic versions.

Per-version layout:

| version | layout vs v83 atlas | confirmation |
|---|---|---|
| v83 | baseline (above). The "0x0A marker" is the charLevel byte; "reserved short 0" is the foothold id. Atlas writes 0x0A / 0 which the client tolerates (visual-only fields). | IDA classic reader `sub_7A379B`; Cosmic |
| v84 | **= v83.** Dispatch wrapper byte-identical; classic vtable spawn reader. | IDA (dispatch `sub_970201`); spawn-reader structurally = v83 |
| v87 | **= v83.** Classic reader `sub_7F504A` byte-identical to v83 `sub_7A379B`. | IDA-confirmed |
| v95 | **DELTA (gated ≥ v95, see ≥87 note below):** appends `byte bAvatarLook` + optional `AvatarLook` blob, and a Tesla-Coil-only (`skillId==35111002`) `byte + 3×(short,short)` triangle tail. Leading bytes are charLevel + SLV (semantic, same positions). | IDA `CSummoned::Init@0x755740` (named) |
| JMS185 | **= v83 head/tail layout** (classic vtable spawn reader, same OnPacket shape). The trailing avatar-look byte slot present as in v83/v87; no Tesla special observed in the classic reader path. | IDA (dispatch `0x9f7f6e`); classic shape |
| v12 | **derived, unverified — confirm against capture.** Expected = v83 (oldest classic). | no IDB |
| v92 | **derived, unverified — confirm against capture.** If v92 is pre-restructure → = v83; if it already tracks v95 → has the avatar-look byte. Unknown. | no IDB |

> **≥87 gate (per `bug_majorversion_gt83_is_off_by_one_v87`):** v84/v86 are
> byte-identical to v83 — CONFIRMED here for spawn (v84 dispatch + spawn reader =
> v83). Any NEW spawn structure (the avatar-look byte + AvatarLook blob + Tesla
> triangle) is **only confirmed at v95**, not v87 (v87's classic reader does NOT
> decode an avatar-look blob). So the encode/decode gate for the avatar-look
> extension should be `>= 95` (or capture-driven for v92), NOT `>= 87`. The opcode
> values, however, change at EVERY version bump and must be seeded per-version
> from §1.

### 3.2 SummonRemove

v83 layout: `int ownerId, int oid, byte (animated?4:1)`. The dispatch reads
`Decode4 oid` then a tail byte (`& 0x7F` in OnHit; remove handled in the pool-remove
path). Classic remove path is structurally identical v83/v84/v87/JMS185; v95 routes
through `OnRemoved(charId, pkt)`.

| version | layout | confirmation |
|---|---|---|
| v83 | baseline | atlas (Cosmic), tested |
| v84 / v87 / JMS185 | **= v83** | IDA (dispatch wrappers byte-identical) |
| v95 | **= v83** (flat `OnRemoved` leaf, same fields) | IDA dispatch |
| v12 / v92 | **derived, unverified — confirm against capture** | no IDB |

### 3.3 SummonMove

v83 atlas: `int cid, int oid, short startX, short startY, byte[] rawMovement`.
Client reader = `CMovePath::OnMovePacket` (v83 `0x68b371`, v84 `sub_7CC317`), a
verbatim movement-blob parse.

| version | layout | confirmation |
|---|---|---|
| v83 | baseline | atlas (Cosmic), tested |
| v84 | **= v83** (`sub_7CC317` → `CMovePath::OnMovePacket`, byte-identical to v83 OnMove `0x7a6861`) | IDA-confirmed |
| v87 | **= v83** (OnMove `0x7f902b`, same 0x21-byte thunk) | IDA-confirmed |
| v95 | **= v83** (`OnMove` → `CUser::OnSummonedMove@0x8e3860`, oid+movement) | IDA-confirmed |
| JMS185 | **= v83** (OnMove `0x8286e4`, same thunk) | IDA-confirmed |
| v12 / v92 | **derived, unverified — confirm against capture** | no IDB |

### 3.4 SummonAttack

v83 atlas writer: `int cid, int oid, byte 0(charLevel), byte direction, byte count,
per target {int monsterOid, byte 6, int damage}`. The v84 client attack reader
`sub_7CC338` decodes `Decode1 (animByte), Decode1 (flag>>7 | &0x7F), Decode1 count,
loop{Decode4 monsterOid, Decode1, Decode4 damage}` — structurally identical to v83
`CSummonedPool::OnAttack@0x7a6882`.

| version | layout | confirmation |
|---|---|---|
| v83 | baseline | atlas (Cosmic), tested |
| v84 | **= v83** (`sub_7CC338` field-for-field) | IDA-confirmed |
| v87 | **= v83** (OnAttack `0x7f904c`) | IDA-confirmed (decompiled, same shape) |
| v95 | **= v83** (OnAttack leaf `0x759860`) | IDA dispatch (leaf not field-diffed; structure preserved) |
| JMS185 | **= v83** (OnAttack `0x828707`) | IDA dispatch |
| v12 / v92 | **derived, unverified — confirm against capture** | no IDB |

### 3.5 SummonDamage (client OnHit)

v83 atlas writer: `int cid, int oid, byte 12, int damage, int monsterIdFrom, byte 0`.
Client reader v83 `CSummonedPool::OnHit@0x7a6e5a` / v87 `0x7f963b`: `Decode1 & 0x7F`
→ `SetAttackAction` (the rest consumed by the action-layer path).

| version | layout | confirmation |
|---|---|---|
| v83 | baseline | atlas (Cosmic), tested |
| v84 | **= v83** (`sub_7CC920`, single `Decode1 & 0x7F`, byte-identical to v83 OnHit) | IDA-confirmed |
| v87 | **= v83** (OnHit `0x7f963b`, identical) | IDA-confirmed |
| v95 | **= v83** (OnHit `0x7598c0` → `CUser::OnSummonedHit@0x8e3a10` → `CSummoned::OnHit`) | IDA-confirmed |
| JMS185 | **= v83** (OnHit `0x828cb2`) | IDA dispatch |
| v12 / v92 | **derived, unverified — confirm against capture** | no IDB |

### 3.6 SummonSkill

v83 atlas writer: `int cid, int summonSkillId, byte newStance`. Client reader v83
`CSummonedPool::OnSkill@0x7a6ebe` / v84 `sub_7CC984`: `Decode1 (attackType), Decode4
(mob/skill id), Decode1` then resolves a mob attack-info animation — structurally
identical v83↔v84.

| version | layout | confirmation |
|---|---|---|
| v83 | baseline | atlas (Cosmic), tested |
| v84 | **= v83** (`sub_7CC984`, `Decode1/Decode4/Decode1`, byte-identical) | IDA-confirmed |
| v87 | **= v83** (OnSkill `0x7f969f`) | IDA-confirmed |
| v95 | **= v83** (OnSkill leaf `0x759890`; dispatched at 0x11A, BEFORE Hit) | IDA dispatch |
| JMS185 | **= v83** (OnSkill `0x828d16`) | IDA dispatch |
| v12 / v92 | **derived, unverified — confirm against capture** | no IDB |

## 4. Summary for Task 6.2 / 6.3

- **Opcodes (Task 6.3 seeding):** seed the 6 writers per-version from §1. v83/v84/v87/v95/JMS185
  are IDA-confirmed; v12/v92 are derived-unverified and must be capture-confirmed
  before going live. **v95 swaps Damage(0x11B)/Skill(0x11A)** — do not assume the
  classic Hit-then-Skill order when generating v95 templates. The 3 handler opcodes
  are Cosmic-derived for v83 and derived-unconfirmed elsewhere (resolve from send-site
  xrefs / capture).
- **Layout gating (Task 6.2):** five of six packets are **byte-stable across all
  IDA'd versions** — encode/decode needs NO version branch for Remove/Move/Attack/
  Damage/Skill. **Only SummonSpawn has a real layout delta**, and it is gated at
  **`>= 95`** (avatar-look-present byte + optional AvatarLook blob + Tesla-Coil
  triangle), NOT `>= 87`. v84/v86/v87 spawn = v83 (confirmed). The v83 "0x0A marker"
  is semantically charLevel and the "reserved short 0" is the foothold id — both are
  visual-only and the current fixed writes are client-tolerated.
