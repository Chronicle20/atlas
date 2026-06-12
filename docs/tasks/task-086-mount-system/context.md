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

## 8. Pinned game data (FILLED BY PLAN TASK 1 — empty until then)

> Task 1 replaces the bracketed placeholders below with verified values + the source it read.
> No downstream task may consume a value still in brackets.

- **Level cap:** `[CAP = ? — confirm 31]`
- **expNeededForLevel(level):** `[table/formula — source: ?]`
- **Revitalizer tiredness-heal:** `[? — confirm 30; source: ?]`
- **Skill-only mount skill ids (beginner):** SpaceShip 1013, Yeti1 1017, Yeti2 1018, Broomstick 1019, Balrog 1031 `[confirm]`
- **Skill-only mount vehicle ids:** Yeti1 1932003, Yeti2 1932004, Broomstick 1932005, Balrog 1932010, SpaceShip 1932000+lvl `[confirm]`
- **Noblesse/Legend band skill ids for the above:** `[derive from constants.go patterns + confirm]`
- **Revitalizer item id(s) / classification:** classification 226 (2260000–2269999) `[confirm exact starter-grant item id for the questline]`
- **Riding Mimiana quest id(s), NPC id(s), starter saddle + taming-mob item ids:** `[from script comments / local data]`
