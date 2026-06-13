# Mount / Monster-Rider System — Implementation Context

Companion to `plan.md`. Captures the key files, verified facts, decisions, and
dependencies an engineer needs before executing. Read this once; refer back per task.

---

## 1. What this builds (one paragraph)

End-to-end tamed-monster and skill-only mounts. A character casts the Monster Rider
skill (`skillId % 10000000 == 1004`) or a skill-only mount skill; the channel toggles a
`MONSTER_RIDING` character-temporary-stat buff carrying the vehicle item id and skill id,
which the existing buff render path draws for self + observers. A new `atlas-mounts`
service owns persistent per-character mount progression (level/exp/tiredness), a 60s
tiredness ticker, and feed (revitalizer) application. A new `SET_TAMING_MOB_INFO` packet
broadcasts mount info. A questline (Riding Mimiana) awards the skill + starter equips.

---

## 2. IDA-confirmed wire facts (v83, port 13337) — do not re-derive

> **MULTI-VERSION SCOPE (decided 2026-06-12).** The feature must support ALL supported
> versions: **GMS 12 / 83 / 87 / 92 / 95 and JMS 185**. The architecture is already
> version-agnostic — opcodes are resolved per-tenant from config, the CTS encoder is
> version-branched (`buildCharacterTemporaryStatRegistry`), and skill/item data is read
> per-tenant from each version's WZ. The wire facts in this table are **IDA-confirmed for
> v83 ONLY**. Three packet *body layouts* must be IDA-verified on the v87 / v95 / JMS IDBs
> and given version branches if they differ, as a **PRE-DEPLOY GATE** (new plan Task 41b),
> before the feature is enabled on any non-v83 tenant:
> 1. the Monster Riding two-state CTS stat (`getBaseTemporaryStats()` block — currently a
>    fixed layout for all versions);
> 2. `SET_TAMING_MOB_INFO` body (Task 5 — single fixed layout, no version branch yet);
> 3. the food-request `0x4D` body (Task 28).
> Also re-confirm the mount skill ids (1004/1013/1017/1018/1019/1031) for JMS (almost
> certainly identical — beginner-band ids are global — but unverified). Decision: continue
> building on the v83 baseline now (most remaining tasks are version-neutral service/Kafka
> logic); run the cross-version IDA pass before deploy.

> **MONSTER_RIDING render bug — root cause was a per-stat double-encode, NOT mask placement
> (IDA-verified 2026-06-12; corrects an earlier wrong table).** The mount not appearing on v83 was
> caused by `Encode` writing a bogus 10-byte per-stat value block for MONSTER_RIDING in addition to
> its base-stat block. RideVehicle is a *base/TwoState* stat: the v83 client reads it in its
> 7-iteration base-stat loop (`SecondaryStat::DecodeForLocal` @0x781D0E, flag `1<<(i+82)` from
> `sub_78D977`), never as a per-stat block. The extra 10 bytes desynced the entire packet tail, so
> the base stats — including the mount — were read as garbage. Fix: skip all `baseStatNames` in the
> per-stat value loop (and its symmetric decode), **version-independent** (`character_temporary_stat.go`).
>
> **The mask placement was always correct — no per-version override needed.** The client's `UINT128`
> is a 4-dword array stored big-endian (`setValue` puts the integer in `dword[3]`), AND'd against the
> decoded mask in wire order (`DecodeBuffer` fills `dword[0]`=wire bytes 0-3 … `dword[3]`=12-15). The
> client flag for RideVehicle is `1<<85` → set in array index 1 → **wire bytes 4-7**. Atlas's encoder
> writes logical bit 85 via `uint32(H&0xFFFFFFFF)` to the *same* wire bytes 4-7. They match because the
> registry's **version gates** already assign MonsterRiding the shift that equals the client's `i+N`
> gate:
>
> | Version | client gate `i+N` (RideVehicle bit) | registry MonsterRiding shift | aligned? |
> |---|---|---|---|
> | GMS v83/v84 | i+82 → 85 | 85 (no post-SoulStone blocks) | ✅ |
> | JMS v185 | i+110 → 113 | 113 (both post-SoulStone blocks, +28) | ✅ |
> | GMS v87 | i+86 → 89 | 89 (Flying block, +4) | ✅ expected |
> | GMS v95 | i+122 → 125 (prior table; re-verify) | 113 (+28) | ⚠️ re-verify in Task 41b |
>
> So mounts render on any version where the registry's CTS enumeration matches the client's bit
> order. v83/v84 verified. **v87/v92/v95 must be re-confirmed in Task 41b** — if a client has CTS
> entries the registry doesn't model (the prior v95 i+122 vs registry-113 gap hints at this), the fix
> is to complete the registry enumeration for that version, *not* to hand-place mask bits.

From `design.md` §1.1. These are authoritative; build against them:

| Fact | Value |
|---|---|
| MONSTER_RIDING temp-stat encoding | `nOption` = vehicle/taming-mob item id (1st int), `rOption` = source skill id (2nd int), then expire time |
| Tamed-mount prerequisites | BOTH slot -18 (taming-mob) AND slot -19 (saddle) required; either empty → silent no-op |
| Re-cast behavior | Server-driven toggle. Same packet `CP_UserSkillUseRequest` (0x5B) mounts or dismounts; server owns the decision |
| Mount food | Dedicated client→server opcode **0x4D** (`SendTamingMobFoodItemUseRequest`); body `ts(4), slot(2), itemId(4)`; gated on item classification **226** |
| `SET_TAMING_MOB_INFO` (s→c) field order | `characterId(4), level(4), exp(4), tiredness(4), levelUp(1 byte)` |

---

## 3. Game-data values that MUST be pinned (verify-over-memory)

These are **not** in the repo and were **not** carried into this plan from memory. Plan
**Task 1 (Pin game data)** resolves them before any consuming task. Sources, in priority order:

- **Mount skill-id / vehicle-id set (OQ 9.6):** the beginner-band ids in the design
  (SpaceShip 1013, YetiMount1 1017, YetiMount2 1018, Broomstick 1019, Balrog 1031) and
  their vehicle ids (Yeti1→1932003, Yeti2→1932004, Broomstick→1932005, Balrog→1932010,
  SpaceShip→`1932000 + skillLevel`). Confirm the Noblesse/Legend band skill ids by
  reading `libs/atlas-constants/skill/constants.go` patterns + the reference-server skill
  table; confirm vehicle ids against live atlas-data skill effects.
- **Exp-to-level table + level cap (OQ 9.4):** reference-server `getMountExpNeededForLevel`.
  Cap is believed to be 31 — confirm. Record the exact table in this file under §8 once pinned.
- **Revitalizer tiredness-heal value (OQ 9.4):** the per-item heal (design assumes 30).
  Check the consumable WZ spec via live atlas-data (`GET /api/data/consumables/{id}` with
  TENANT_ID/REGION/MAJOR_VERSION/MINOR_VERSION headers — see the `reference_atlas_data_wz_inspection`
  memory) or MinIO. If absent in WZ, use reference parity (30) and document the decision here.

**Do not start Tasks 7 (atlas-data reader), 12 (feed math), or 17 (constants band) until §8 below is filled in by Task 1.**

---

## 4. Architectural decisions (from design.md)

- **New `atlas-mounts` service** owns persistence + active-mount registry + tiredness ticker,
  modeled verbatim on `atlas-pets`. Rendering stays on `atlas-buffs` + `atlas-channel`.
- **Buff duration sentinel:** atlas-buffs `NewBuff` rejects `duration <= 0` and there is **no
  never-expires path** (verified: `services/atlas-buffs/.../buff/model.go` `ErrInvalidDuration`).
  Mounts must not auto-expire, so apply with a large int32 sentinel. **Decision: use
  `math.MaxInt32` (2147483647 ms ≈ 24.8 days).** A channel re-cast (toggle) or job change
  cancels well before then; logout deregisters the ticker. Document this constant in code.
- **Toggle ownership:** the channel decides mount-vs-dismount from session-local buff state
  (the buff give/cancel consumer tracks active buffs per session).
- **Feed math** (FR-8.2): `heal = min(tirednessHeal, tiredness)`;
  `exp += ceil((heal / tirednessHeal) * (2*level + 6))`; level up while
  `exp >= expNeededForLevel(level) && level < CAP`.

---

## 5. Key files & verified signatures (by area)

### libs/atlas-packet (packet fix + new writer)
- `model/character_temporary_stat.go:715-726` — `getBaseTemporaryStats()`; Monster Riding
  base stat at index 3 is `NewCharacterTemporaryStatBase(false)` with the zeroed-value TODO.
- `CharacterTemporaryStatBase` struct (l.326-332): fields `bDynamicTermSet, nOption int32,
  rOption int32, tLastUpdated int64, usExpireItem int16`. `Encode` writes `nOption` then
  `rOption` then time.
- Stored stat `CharacterTemporaryStatValue` (l.298-304): `statType, sourceId int32, level byte,
  value int32, expiresAt`. Accessors `Value()`, `SourceId()`, `Level()`. The active stats live
  in `m.stats[type]`.
- Both `Encode` (self) and `EncodeForeign` (observer) iterate `getBaseTemporaryStats()` and
  append each block — fix once, both paths benefit.
- Tests: `model/character_temporary_stat_test.go`; byte-level style `TestCTSEncodeSlowDiseasePerStatLayout`
  uses `pt.CreateContext("GMS",83,1)`, `tenant.Create`, `AddStat`, then asserts wire bytes with `bytes.Equal`.
- `AddStat` signature: `AddStat(l)(t)(name string, sourceId int32, amount int32, level byte, expiresAt time.Time)`.

### libs/atlas-constants
- `skill/constants.go`: `BeginnerMonsterRidingId = Id(1004)`, `NoblesseMonsterRidingId =
  Id(10001004)`, `LegendMonsterRidingId = Id(20001004)`, `EvanMonsterRidingId = Id(20011004)`,
  `CorsairBattleshipId` (do NOT touch Battleship — out of scope).
- `item/constants.go`: `ClassificationTamedMob = Classification(190)`, `ClassificationSaddle =
  Classification(191)`; `GetClassification(itemId) = floor(itemId/10000)`.
- `inventory/slot/constants.go:37-40`: `{tamingMob,-18}`, `{saddle,-19}`, `{mobEquip,-20}` exist.
- `character/temporary_stat.go:119`: `TemporaryStatTypeMonsterRiding = "MONSTER_RIDING"` exists.

### atlas-data
- `skill/reader.go:226-228` — mount branch with TODO. Existing line:
  `} else if skill.Is(skillId, skill.BeginnerMonsterRidingId, …, skill.CorsairBattleshipId) {`
  `statups = produceBuffStatAmount(statups, character.TemporaryStatTypeMonsterRiding, int32(skillId))`.
  For 1004-band the amount = skillId placeholder (channel overrides with equipped taming-mob id);
  for skill-only mounts the amount must be the **vehicle id** (per-level for SpaceShip).
- `skill/effect/statup/rest.go`: `RestModel{ Type string; Amount int32 }`.
- `consumable/rest.go`: `RestModel.Spec map[SpecType]int32`; `SpecType` enum (hp, mp, …).
  **No tiredness/mount-food spec type exists** — add one if WZ carries it; else atlas-consumables
  passes the pinned heal constant.

### atlas-buffs
- `buff/stat/model.go`: `Model{ statType string; amount int32 }`, `NewStat(statType, amount)`.
- `buff/model.go`: `Model.SourceId() int32`; `NewBuff(sourceId, level, duration, changes)` →
  `ErrInvalidDuration` if `duration <= 0` (the sentinel constraint).
- `kafka/message/character/kafka.go`: `EnvEventStatusTopic = "EVENT_TOPIC_CHARACTER_BUFF_STATUS"`,
  `EventStatusTypeBuffApplied="APPLIED"`, `EventStatusTypeBuffExpired="EXPIRED"`;
  `StatusEvent[E]{WorldId, CharacterId, Type, Body}`; `AppliedStatusEventBody{FromId, SourceId,
  Level, Duration, Changes []StatChange, CreatedAt, ExpiresAt}`; `ExpiredStatusEventBody{…}`.
  `StatChange{Type string, Amount int32}`.
- `character/processor.go`: `CancelByStatTypes(worldId, characterId, types []string)` already exists
  (job-change dismount routes through here).

### atlas-channel
- `socket/handler/character_skill_use.go` — `CharacterUseSkillHandleFunc`; decodes
  `packetmodel.SkillUsageInfo`, loads effect via `skill3.NewProcessor(l,ctx).GetEffect(skillId,level)`,
  dispatches to `skill/handler.UseSkill(l)(ctx)(wp, field, characterId, sui, se)`.
- `skill/handler/common.go:97` — the existing skill→buff apply:
  `applyBuffFunc := buff.NewProcessor(l, ctx).Apply(f, characterId, int32(info.SkillId()),
  info.SkillLevel(), e.Duration(), e.StatUps()); applyBuffFunc(characterId)`. The **mount branch
  slots in here**, before the generic `e.Duration() > 0 && len(e.StatUps()) > 0` block.
- `character/buff/processor.go:19,45` — `Apply(f, fromId, sourceId, level, duration int32, statups
  []statup.Model) model.Operator[uint32]`; `Cancel(f, characterId, sourceId int32) error`.
- Buff render consumer `kafka/consumer/buff/consumer.go`: `handleStatusEventApplied` →
  `CharacterBuffGiveWriter` (self) + `ForOtherSessionsInMap` → `CharacterBuffGiveForeignWriter`;
  `handleStatusEventExpired` → cancel writers. This is the session buff-state source of truth.
- Equip read: `cp.GetById(cp.InventoryDecorator)(characterId)` → `c.Inventory()` /
  `c.Equipment().Get(slotType)`; compartment `FindBySlot(int16)`.
- `socket/handler/pet_food.go` — template for the new mount-food handler: decodes a serverbound
  packet, calls `consumable.NewProcessor(...).RequestItemConsume(...)`.
- Handler registration: `main.go produceHandlers()` map keyed by string handle const; opcode
  resolved from tenant `Socket.Handlers` config. Writer registration: `main.go produceWriters()`
  string list; opcode from tenant `Socket.Writers`.
- Map broadcast: `_map.NewProcessor(l,ctx).ForSessionsInMap(field, op)` /
  `ForOtherSessionsInMap`; `session.NewProcessor(l,ctx).IfPresentByCharacterId(channel)(charId, op)`;
  `session.Announce(l)(ctx)(wp)(writerName)(encoder)(s)`.

### atlas-consumables
- `consumable/processor.go`: `RequestItemConsume(c/field, characterId, slot, itemId, quantity)`;
  routing by `item.GetClassification(itemId)`; `usesStandardConsumer` switch (200/201/202/205);
  `ConsumePetFood`, `ConsumeTownScroll`, etc.; `ConsumeItem` decrements via compartment command.
- `kafka/message/consumable/kafka.go`: `Command[E]{TransactionId, WorldId, ChannelId, MapId,
  Instance, CharacterId, Type, Body}`; `Event[E]{CharacterId, Type, Body}`; producer `ProviderImpl`.

### atlas-pets (clone source for atlas-mounts)
- Service root `services/atlas-pets/atlas.com/pets/`. Packages: `pet/` (entity/model/builder/
  processor/administrator/rest/resource + registries), `character/` (Redis logged-in registry),
  `tasks/` (Task interface + `Register`), `kafka/{message,producer,consumer/{character,asset,pet}}`,
  `main.go`.
- Entity pattern, immutable Model+Builder, `NewProcessor(l, ctx, db)`, `With(WithTransaction(tx))`,
  `database.ExecuteTransaction`, `message.Emit`/`Buffer.Put`, producer `Provider`.
- Registry: `atlas.NewTenantRegistry[uint32, field.Model](client, "pet-character", keyFn)` +
  `atlas.NewSet`; `InitRegistry(rc)` in main; populated from `character_status_event`
  LOGIN/LOGOUT. **Route all Redis through `libs/atlas-redis` (repo invariant).**
- Hunger task: `pet/task.go` `Timeout{Run, SleepTime}`; `tasks.Register(l, ctx)(NewHungerTask(...,
  time.Minute*3))`. Run() iterates `character.GetLoggedIn(ctx)` and mutates+emits.

### Build / deploy wiring (new service)
- `.github/config/services.json` — add atlas-mounts entry (clone atlas-pets, alphabetical).
- `go.work` — add `./services/atlas-mounts/atlas.com/mounts`.
- `docker-bake.hcl` — add `"atlas-mounts"` to `go_services` (alphabetical).
- `deploy/k8s/base/atlas-mounts.yaml` — clone `atlas-pets.yaml`; `DB_NAME: atlas-mounts`;
  **no LB socket ports** (REST+Kafka only).
- Repo-root `Dockerfile` — **no edit** (no new shared lib; `ARG SERVICE` parameterized).

---

## 6. Kafka topic catalog (design §12)

| Topic (env var) | Producer | Consumer(s) | Shape |
|---|---|---|---|
| `COMMAND_TOPIC_CHARACTER_BUFF` (existing) | channel | buffs | Apply/Cancel MONSTER_RIDING |
| `EVENT_TOPIC_CHARACTER_BUFF_STATUS` (existing) | buffs | channel (render), **mounts (registry)** | APPLIED/EXPIRED |
| `EVENT_TOPIC_CHARACTER_STATUS` (existing) | character | **mounts** | LOGIN/LOGOUT gating |
| `COMMAND_TOPIC_TAMING_MOB_FOOD` (new) | channel | consumables | `{characterId, slot, itemId}` |
| `EVENT_TOPIC_TAMING_MOB_FOOD` (new) | consumables | mounts | `{characterId, itemId, tirednessHeal}` |
| `EVENT_TOPIC_MOUNT_STATUS` (new) | mounts | channel | `{Type∈SET/TICK/FEED, Level, Exp, Tiredness, LevelUp, TooTired}` |

---

## 7. Verification gates (every changed module)

From CLAUDE.md §Build & Verification:
1. `go test -race ./...` clean
2. `go vet ./...` clean
3. `go build ./...` clean
4. `docker buildx bake atlas-<svc>` from worktree root for every service whose `go.mod` changed
   (mounts, channel, buffs, data, consumables — and any constants consumer that needs a rebuild).
5. `tools/redis-key-guard.sh` clean from repo root.

Acceptance-critical test: the MONSTER_RIDING byte-level encode test (Task 4) — must pass for
both `Encode` and `EncodeForeign`.

---

## 8. Pinned game data (FILLED BY PLAN TASK 1)

> Every value below cites the source actually read (verify-over-memory).
> Primary reference server: **HeavenMS** (ronancpl/HeavenMS @ master — v83 lineage).
> Live data: **atlas-data** in k8s namespace `atlas-main`, GMS 83.1 tenant
> `ec876921-c363-4cc6-9c51-5bb8d57f9553` (headers `TENANT_ID/REGION=GMS/MAJOR_VERSION=83/MINOR_VERSION=1`).

### 8.1 Level cap

`CAP = 31` — a mount levels up only while `level < 31`.
Source: HeavenMS `src/net/server/channel/handlers/UseMountFoodHandler.java`
(`boolean levelup = mount.getExp() >= ExpTable.getMountExpNeededForLevel(level) && level < 31;`).

### 8.2 expNeededForLevel(level)

`getMountExpNeededForLevel(level)` returns `mount[level]` (0-indexed) from this table.
Source: HeavenMS `src/constants/game/ExpTable.java`, field `private static final int[] mount`.

```go
// expNeededForLevel(level) == mountExp[level]; valid indices 0..28.
var mountExp = []int32{
    1, 24, 50, 105, 134, 196, 254, 263, 315, 367,
    430, 543, 587, 679, 725, 897, 1146, 1394, 1701, 2247,
    2543, 2898, 3156, 3313, 3584, 3923, 4150, 4305, 4550,
}
```

| level | exp needed | level | exp needed | level | exp needed |
|---|---|---|---|---|---|
| 0 | 1 | 10 | 430 | 20 | 2543 |
| 1 | 24 | 11 | 543 | 21 | 2898 |
| 2 | 50 | 12 | 587 | 22 | 3156 |
| 3 | 105 | 13 | 679 | 23 | 3313 |
| 4 | 134 | 14 | 725 | 24 | 3584 |
| 5 | 196 | 15 | 897 | 25 | 3923 |
| 6 | 254 | 16 | 1146 | 26 | 4150 |
| 7 | 263 | 17 | 1394 | 27 | 4305 |
| 8 | 315 | 18 | 1701 | 28 | 4550 |
| 9 | 367 | 19 | 2247 | | | | |

**Array-bounds caveat (downstream Task 12 must guard):** the HeavenMS `mount[]` array has
only **29 entries (indices 0–28)**, but the cap check allows `level < 31`. HeavenMS calls
`getMountExpNeededForLevel(level)` with the *current* level; once a mount reaches level 29 it
would read `mount[29]` (out of bounds) on the next feed. In practice mounts rarely reach that
range, but the Go port must bound-check (e.g. treat any `level >= len(mountExp)` as
"effectively max / no further level-up", or extend the table). Do **not** blindly index.

### 8.3 Feed math (re-confirms design §4 from the same source)

Source: HeavenMS `UseMountFoodHandler.java`:
```
healedTiredness = min(curTiredness, 30)
healedFactor    = healedTiredness / 30        // float
exp += ceil(healedFactor * (2*level + 6))
levelup = exp >= getMountExpNeededForLevel(level) && level < 31
```

### 8.4 Revitalizer tiredness-heal

`tirednessHeal = 30` — **server-side constant, NOT data-driven.** Hardcoded in
`UseMountFoodHandler.java` as `Math.min(curTiredness, 30)`.

Live WZ cross-check (atlas-data, tenant above): `GET /api/data/consumables/2260000`
returns `incFatigue: 0` and `spec.inc: 0` — i.e. the v83 WZ spec for the only class-226 item
present (2260000) carries **no** non-zero tiredness/fatigue value. There is therefore no
data-driven heal to read; the pinned value is the reference-server constant **30**.

**Decision for downstream tasks:** `atlas-consumables` passes the constant **30** as
`tirednessHeal` on the `EVENT_TOPIC_TAMING_MOB_FOOD` event (design §4 / §6). The
`EVENT_TOPIC_TAMING_MOB_FOOD` shape already carries `tirednessHeal` per-event, so if a future
WZ item populates `incFatigue`/`spec.inc`, atlas-data's consumable reader could surface it and
atlas-consumables could forward it — but for now it is a constant. (Verified: only item
2260000 exists in class 226 for this tenant; 2260001–2260004 404.)

### 8.5 Skill-only mount skill ids

All five beginner ids confirmed present as real skill-only mount skills in **live atlas-data**
(`GET /api/data/skills/{id}`, tenant above): each returns `skill: true`, `duration:
2100000000`, `MPConsume: 10`. SpaceShip (1013) has `maxLevel: 2` (consistent with per-level
vehicle id); the others `maxLevel: 1`.

| Mount | Beginner | Noblesse | Legend | Evan |
|---|---|---|---|---|
| MonsterRider (1004-band) | 1004 | 10001004 | 20001004 | 20011004 |
| SpaceShip | 1013 | **1001014** | — (none) | — |
| Yeti1 | 1017 | 10001019 | 20001019 | — |
| Yeti2 | 1018 | 10001022 | 20001022 | — |
| Broomstick | 1019 | 10001023 | 20001023 | — |
| Balrog | 1031 | 10001031 | 20001031 | — |
| Corsair Battleship | — | — | — | 5221006 (own band) |

Source for the Noblesse/Legend/Evan ids: HeavenMS `src/constants/skills/{Beginner,Noblesse,Legend,Evan,Corsair}.java`.

**⚠ The Noblesse/Legend skill-only mount ids do NOT follow the `+10000000 / +20000000`
band-offset pattern** that the 1004-band MonsterRider uses (and that the scout note in this
task assumed). They are an independent layout, e.g. Noblesse SpaceShip = **1001014** (not
10001013), Noblesse Yeti1 = 10001019 (not 10001017). **Task 17 must transcribe the exact ids
from this table; do not derive Noblesse/Legend ids by offsetting the beginner ids.** Only the
MonsterRider (1004) band and the Balrog mount (…1031) happen to match a clean offset; the
others do not. Legend has no SpaceShip; Evan has only MonsterRider (no skill-only mounts).

### 8.6 Skill-only mount vehicle ids (the `MONSTER_RIDING` nOption)

Source: HeavenMS `src/server/MapleStatEffect.java` (mount-apply block, `isMonsterRiding()`):

| Mount skill (any band) | Vehicle item id |
|---|---|
| SpaceShip (Beginner 1013 / Noblesse 1001014) | `1932000 + skillLevel` |
| Yeti1 (1017 / 10001019 / 20001019) | 1932003 |
| Yeti2 (1018 / 10001022 / 20001022) | 1932004 |
| Broomstick (1019 / 10001023 / 20001023) | 1932005 |
| Balrog (1031 / 10001031 / 20001031) | 1932010 |
| Corsair Battleship (5221006) | 1932000 (out of mount scope) |

Cross-check: HeavenMS `MapleCharacter.java:4435` re-applies Battleship as
`giveBuff(1932000, 5221006, …)`, confirming the 1932xxx vehicle band and that the buff carries
`(vehicleId, skillId)` — matching the §2 IDA encoding (`nOption=vehicleId`, `rOption=skillId`).
Note: the live skill-effect JSON (8.5) does **not** encode the vehicle id — it is hardcoded
server-side, which is exactly why Task 7 must add it to atlas-data's `skill/reader.go`.

For **tamed** mounts (the 1004-band MonsterRider), the vehicle id is the equipped taming-mob
item id (slot -18), per HeavenMS `MapleStatEffect.java` (`ridingMountId = mount.getItemId()`)
and design §2. Tamed taming-mob items live in class 190 (`1902xxx`); see HeavenMS
`MapleMount.java` comment (1902000 Hog, 1902001 Silver Mane, 1902002 Red Draco, 1902005
Mimiana, 1902012 Yeti, …) and `getId() => itemid - 1901999`.

### 8.7 Riding-Mimiana questline ids — ✅ CORRECTED 2026-06-12 (was wrong below)

> **CORRECTION (2026-06-12).** The "classic quest does not exist" conclusion below was WRONG —
> it only grepped `deploy/seed/.../quests` JSON. **Quest 20523 "Riding Mimiana" EXISTS as WZ data**
> for the v83 tenant (`GET /api/data/quests/20523`) and its WZ `endActions` ALREADY award:
> saddle **1912005** (class 191, slot -19), taming-mob **1902005** (class 190, slot -18), skill
> **10001004** (MonsterRider), and consume quest item **4032117**. Start: npc **1102002**, level 50+,
> Cygnus Knight job, prereq **quest 20522** completed. atlas-quest `processEndActions`
> (`services/atlas-quest/.../quest/processor.go`) builds AddAwardItem/AddCreateSkill from the WZ
> EndActions when `complete_quest` fires — so an NPC conversation needs ONLY `start_quest` /
> `complete_quest` (NO manual award; the `suppressAwardAssetByCompleteQuest` dedup suppresses manual
> awards that duplicate the WZ ones). FR-9 was implemented as a single NPC conversation file
> `deploy/seed/gms/83_1/npc-conversations/quests/quest-20523.json` (v83). Other versions' quest-20523
> conversation is per-version data, deferred to the multi-version pre-deploy pass. The legacy notes
> below (Empress's Knights 20522/20526 line) are still factually present but were NOT the right quest.

### 8.7-legacy Riding-Mimiana questline ids — ⚠ PARTIAL / divergent from design (SUPERSEDED — see 8.7 above)

**The classic v83 "Riding Mimiana" Monster-Rider acquisition quest (the one that grants the
1004-band MonsterRider skill + a class-191 saddle + class-190 taming-mob) is NOT present in
this repo's seed data.** Searched `deploy/seed/{gms,jms}/**/npc-conversations/quests` and
`docs/`: no quest grants skill 1004 / 10001004 / 20001004, and there is **no class-191 (191xxxx)
saddle item anywhere in the seed data** (grep `19[01][0-9]{4}` → only 1902005/1902016/1902017/
1902018, all class 190).

What the repo **does** contain (verified) is the later **Empress's Knights "Mimiana"
questline** (item-based mount, different mechanic — no separate saddle, no 1004-band skill):

| Item | Value | Source |
|---|---|---|
| "Raising Mimiana" quest | **questId 20522** (questName `"Raising Mimiana"`) | `deploy/seed/gms/83_1/npc-conversations/quests/quest-20522.json` |
| "Mimiana Recovery" quest | **questId 20526** (questName `"Mimiana Recovery"`) | `deploy/seed/gms/83_1/npc-conversations/quests/quest-20526.json` |
| Quest-giver NPC | **2060005** (Mimiana keeper, named in quest-20522/20526 dialog `#p2060005#`); the broader Knights NPC `1202009` also references the line | quest-20522/20526 dialog + `npc-1202009.json` |
| Mimiana egg (ETC item raised into the mount) | **4220137** (`referenceId`/`award_item`/`destroy_item` in both quests) | quest-20522.json, quest-20526.json |
| Mimiana taming-mob item awarded (class 190) | **1902005** (`award_item` on quest-20526 completion) | quest-20526.json:105 |
| Skill granted | **NONE** — the Knights Mimiana mount is item-driven (equip 1902005 in slot -18 + cast the 1004-band MonsterRider the character already has); no `award_skill`/`skillId` operation in either quest | quest-20522/20526.json (no skill op) |

**Action for Task 36 (questline):** the design's FR-9 assumes the *classic* Riding-Mimiana
quest (skill grant + class-191 saddle + class-190 mob). That exact data **does not exist in the
repo** and must be authored from scratch (or sourced) rather than reused. Best-known classic
reference values (HeavenMS / GMS v83), **UNVERIFIED against repo data — confirm before
Task 36**:
- Classic "Riding Mimiana" / Monster-Rider acquisition quest ≈ quest **3000** band on Mushroom
  Castle; grants skill **1004** (Beginner MonsterRider) + saddle (class 191, `1910000`-band) +
  a starter taming-mob (class 190). These specific ids are **not** confirmed from local data and
  must be pinned (WZ/Quest.wz) before implementing FR-9.
- If FR-9 is satisfied by the **Empress's Knights** Mimiana line instead, use the verified
  20522/20526 + NPC 2060005 + items 4220137/1902005 above (no class-191 saddle, no skill grant).

This divergence should be surfaced to the task owner: **FR-9's "Riding Mimiana grants the
MonsterRider skill + class-191 saddle" premise is not backed by repo data; the only Mimiana
data present is the item-based Knights questline.**
