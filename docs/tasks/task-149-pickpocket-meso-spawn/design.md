# Pick Pocket Meso Spawn (task-149) — Design

Status: Approved for planning
PRD: `docs/tasks/task-149-pickpocket-meso-spawn/prd.md`

## 1. Summary

Implement the Pick Pocket (4211003) proc in atlas-channel's common attack pipeline:
while the PICK_POCKET buff is active and the attack skill is whitelisted, each
damage line against each damaged monster rolls the effect's prop; successes emit
one atlas-drops `SPAWN` command spawning a meso-only drop at the monster's
position. No atlas-drops changes are required — the meso-only SPAWN path is
already exercised in production by atlas-monster-death and
atlas-saga-orchestrator.

## 2. Resolved PRD Open Questions

All three open questions were resolved by reading source (file:line evidence
below); none require guessing.

### 2.1 `Mod` value → send `0`; the field is dead end-to-end today

`Mod` maps to the client packet's animation-*delay* (200 ms per increment per
the saga's `calculateMod` comment,
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/reactor/drop/processor.go:239-245`),
but atlas-drops **discards it**: `handleSpawn`
(`services/atlas-drops/atlas.com/drops/kafka/consumer/drop/consumer.go:53-85`)
never stores it (no `mod` field on `drop.Model`), `StatusEventCreatedBody` on
the drops side never carries it, so the channel's CREATED consumer always
decodes `Mod = 0` and writes `delay = 0`
(`services/atlas-channel/atlas.com/channel/kafka/consumer/drop/consumer.go:104-108`).
Existing producers send `1` (atlas-monster-death, constant) and `0` (saga,
non-spray). Since the PRD already deviates from Cosmic by dropping the 100 ms
stagger, we send `Mod = 0` — the semantically-correct "no delay" value — and do
NOT wire Mod through atlas-drops (that would be an unrelated feature).

### 2.2 `DropType` 2 / owner fields → DropType is cosmetic; FFA pickup comes from `PlayerDrop`

atlas-drops never branches on `DropType` server-side — it is stored, echoed on
the CREATED event, and written into the client packet's display byte only.
Pickup permission is exclusively `Model.CanBeReservedBy`
(`services/atlas-drops/atlas.com/drops/drop/model.go:98-115`), which grants:
`playerDrop == true` → anyone; `ownerId == 0 && ownerPartyId == 0` → anyone;
owner/party match; or ownership timeout.

Cosmic spawns Pick Pocket mesos with `playermeso = true` and droptype 2. The
Atlas equivalent that reproduces "immediately lootable by anyone, displayed as
FFA":

| Field | Value | Why |
|---|---|---|
| `DropType` | `2` | Client-visual FFA styling (server ignores it) |
| `PlayerDrop` | `true` | Grants immediate universal pickup via `CanBeReservedBy` (and matches Cosmic's `playermeso=true`; the packet writer inverts it — `WriteBool(!characterDrop)` — same as Cosmic's byte) |
| `OwnerId` | attacking character id | Informational (CREATED event / REST); irrelevant to pickup because `playerDrop` short-circuits |
| `OwnerPartyId` | `0` | Per PRD; moot for the same reason |

### 2.3 Meso-only drops (`Mesos > 0`, `ItemId == 0`) → fully supported

Spawn (`handleSpawn` sets meso and item unconditionally), packet writing
(`libs/atlas-packet/drop/clientbound/spawn.go:57-76` branches on `meso > 0`),
pickup (RESERVED/PICKED_UP carry `Meso`; atlas-character credits via
`AttemptMesoPickUp`), and expiry (meso-agnostic) all handle it.
atlas-monster-death's `SpawnMeso`
(`services/atlas-monster-death/atlas.com/monster/monster/drop/processor.go:46`)
and the saga's `spawnMesoDrop` emit exactly this shape through the identical
`COMMAND_TOPIC_DROP`/`SPAWN` path today. **atlas-drops needs no changes.**

## 3. Approaches Considered

### 3.1 Where the proc computes (chosen: A)

- **A — atlas-channel attack handler emits SPAWN directly (chosen, PRD-mandated).**
  All inputs are channel-local: the decoded `AttackInfo` (skill id, per-monster
  damage lines), the character's buffs (existing `character/buff` REST
  processor), the skill effect (existing `data/skill` processor), and the
  monster snapshot (existing `monster` processor). Mirrors the MP Eater
  precedent in the same file.
- **B — emit a domain event ("attack landed") and compute the proc in
  atlas-drops or a new consumer.** Rejected: adds a topic + consumer for no
  isolation gain; the whitelist/buff/effect context would have to be shipped
  across the wire or re-fetched; atlas-drops is deliberately dumb about *why*
  drops spawn.
- **C — compute in atlas-monsters' damage consumer.** Rejected: atlas-monsters
  has neither the attack skill id per damage packet nor buff/effect access;
  would smear channel gameplay logic across a service boundary.

### 3.2 How the proc hooks into the pipeline (chosen: B)

- **A — second loop over `ai.DamageInfo()` after the existing per-entry loop.**
  Simple, but re-procs on **reflected** entries (a reflected hit deals no
  damage and applies no status — it must not pick pockets either) unless the
  reflect decision is recomputed or tracked, duplicating state.
- **B — widen the existing `onDamageApplied` hook to carry the damage lines
  (chosen).** `damageInfoEntryDeps.onDamageApplied` currently has signature
  `func(monsterId uint32)`
  (`services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:91`),
  invoked exactly once per non-reflected, damage-applying entry — precisely the
  set of entries Pick Pocket should see. Change it to
  `func(di packetmodel.DamageInfo)` and have the MP Eater call site use
  `di.MonsterId()`. One hook, two passives, correct reflect semantics for free.
- **C — add a parallel `onDamageLines` hook next to `onDamageApplied`.**
  Rejected: two hooks with identical firing conditions is boundary noise.

### 3.3 Per-attack state (buff + effect lookups)

The PRD mandates at most one buff REST call and one effect lookup per attack,
and zero lookups when the skill is not whitelisted. Chosen shape: compute an
immutable per-attack `pickPocketState` (enabled flag, `maxmeso`, `prop`)
**eagerly, before the DamageInfo loop**, guarded first by the whitelist check
(pure, no I/O). Eager beats lazy-inside-the-hook here: the hook then stays a
pure roll-and-emit with no lazy-init locking or error handling mid-loop, and an
attack with the buff but zero damaged monsters costs one wasted REST call only
in a case that barely occurs (whitelisted attack that hits nothing).

## 4. Component Design

All new code lives in atlas-channel. Three units:

### 4.1 `socket/handler/character_attack_common.go` — proc logic

Pure functions (unit-testable, no I/O), mirroring the MP Eater trio:

```go
// pickPocketWhitelisted reports whether skillId can proc Pick Pocket.
// Basic attack (skillId == 0) plus the fixed Cosmic list.
func pickPocketWhitelisted(skillId uint32) bool

// pickPocketMesoAmount computes the meso for one damage line:
// min(max(d/20000 * maxmeso, 1), maxmeso), float math then truncation,
// matching Cosmic. Returns 0 when maxmeso <= 0. A 0-damage line still
// yields 1 on a successful roll.
func pickPocketMesoAmount(damage uint32, maxmeso int32) uint32
```

The prop roll reuses the existing semantics by generalizing the current
`mpEaterShouldProc` into `shouldProc(prop, roll float64) bool` (rename +
update the MP Eater call site and its tests; behavior byte-identical:
`prop <= 0` never, `prop >= 1.0` always, else `roll < prop`). Duplicating the
three-line function under a second name was considered and rejected — same
semantics, one name.

Whitelist contents (from `libs/atlas-constants/skill/constants.go`):
`0` (basic attack), `RogueDoubleStabId`, `BanditSavageBlowId`,
`ChiefBanditAssaulterId`, `ChiefBanditBandOfThievesId`,
`ShadowerAssassinateId`, `ShadowerTauntId`, `ShadowerBoomerangStepId` — as a
package-level `map[uint32]struct{}`.

Per-attack state resolution (side-effecting, swallows all failures):

```go
type pickPocketState struct {
	enabled bool
	maxmeso int32   // PICK_POCKET stat Amount() captured at buff time
	prop    float64 // effect prop at the buff's captured Level()
}

// pickPocketResolveState gates and resolves the per-attack state:
//  1. whitelist check (pure — MUST run before any I/O)
//  2. buff lookup via character/buff Processor.GetByCharacterId; find a
//     non-Expired() buff whose Changes() include
//     charconst.TemporaryStatTypePickPocket; maxmeso = that change's Amount()
//  3. effect lookup: data/skill GetEffect(ChiefBanditPickpocketId, buff.Level())
// Any error or maxmeso <= 0 or prop <= 0 → {enabled: false}, logged, never
// propagated.
func pickPocketResolveState(l, ctx, buffProcessor, effectGetter, skillId, characterId) pickPocketState
```

Per-monster proc (called from the widened `onDamageApplied` hook, after the
MP Eater call):

```go
// pickPocketTryProc rolls each damage line of one non-reflected DamageInfo
// and emits one SPAWN per success. Monster snapshot via mp.GetById (fetch
// failure skips this monster's procs, logged Debugf). Emit errors logged
// Errorf and swallowed. Roll: shouldProc(state.prop, rand.Float64());
// X offset: mon.X() + int16(rand.Intn(100)-50); Y = mon.Y().
func pickPocketTryProc(l, ctx, dp, mp, state, di, f, characterId)
```

`dp` is the `drop.Processor` (new `SpawnMeso` method, §4.2), `di.MonsterId()`
is the monster's field-unique id and is passed as `DropperId`.

Wiring in `processAttack`: resolve `pickPocketState` once before the
`ai.DamageInfo()` loop; inside the `onDamageApplied` closure (signature now
`func(di packetmodel.DamageInfo)`):

```go
onDamageApplied: func(di packetmodel.DamageInfo) {
	if ai.AttackType() == packetmodel.AttackTypeMagic && ai.SkillId() > 0 {
		mpEaterTryProc(l, ctx, mp, c, di.MonsterId(), s.Field(), s.CharacterId())
	}
	if ppState.enabled {
		pickPocketTryProc(l, ctx, dp, mp, ppState, di, s.Field(), s.CharacterId())
	}
},
```

The `// TODO apply Pick Pocket` line (character_attack_common.go:408) is
removed.

Note Pick Pocket has no attack-type gate: the whitelist alone constrains it
(all whitelisted skills are melee; basic attack `skillId == 0` procs from any
basic swing, matching Cosmic which keys purely off the skill list).

### 4.2 `drop` package — SPAWN producer

Message contract (`kafka/message/drop/kafka.go`): add

```go
CommandTypeSpawn = "SPAWN"

type SpawnCommandBody struct {
	ItemId       uint32 `json:"itemId"`
	Quantity     uint32 `json:"quantity"`
	Mesos        uint32 `json:"mesos"`
	DropType     byte   `json:"dropType"`
	X            int16  `json:"x"`
	Y            int16  `json:"y"`
	OwnerId      uint32 `json:"ownerId"`
	OwnerPartyId uint32 `json:"ownerPartyId"`
	DropperId    uint32 `json:"dropperId"`
	DropperX     int16  `json:"dropperX"`
	DropperY     int16  `json:"dropperY"`
	PlayerDrop   bool   `json:"playerDrop"`
	Mod          byte   `json:"mod"`
}
```

Field names/JSON tags mirror atlas-drops' `CommandSpawnBody`
(`services/atlas-drops/atlas.com/drops/kafka/message/drop/kafka.go:123-137`)
exactly, minus the embedded `EquipmentData` — those fields are meaningless for
a meso drop and JSON decoding on the consumer zero-fills absent keys (the
consumer only reads EquipmentData when `ItemId` is an equip).

Producer + processor (`drop/producer.go`, `drop/processor.go`), following the
existing `RequestReservationCommandProvider` / `Processor.RequestReservation`
pattern verbatim:

```go
// SpawnMesoCommandProvider emits a meso-only FFA drop: ItemId=0, Quantity=0,
// DropType=2 (client-visual FFA), PlayerDrop=true (server-side universal
// pickup), OwnerPartyId=0, Mod=0.
func SpawnMesoCommandProvider(f field.Model, mesos uint32, x, y int16, ownerId uint32, dropperId uint32, dropperX, dropperY int16) model.Provider[[]kafka.Message]

func (p *Processor) SpawnMeso(f field.Model, mesos uint32, x, y int16, ownerId uint32, dropperId uint32, dropperX, dropperY int16) error
```

Kafka key: `producer.CreateKey(int(dropperId))` — partitions per dropper,
consistent with the reservation provider keying on the entity id.

### 4.3 atlas-drops, atlas-buffs, atlas-data — no changes

Confirmed by §2. If integration testing surfaces an edge in atlas-drops it is
handled there, per PRD §7, but none is expected.

## 5. Data Flow

```
attack packet → processAttack
  ├─ pickPocketWhitelisted(skillId)?  ── no → state.enabled=false (no I/O)
  ├─ buff GetByCharacterId (1 REST)   ── no active PICK_POCKET → disabled
  ├─ GetEffect(4211003, buffLevel) (1 lookup) → prop
  ├─ per DamageInfo entry: damage/reflect/status (unchanged)
  │    └─ onDamageApplied(di)  [non-reflected only]
  │         ├─ mpEaterTryProc (unchanged behavior)
  │         └─ pickPocketTryProc
  │              ├─ mp.GetById(di.MonsterId())  → position (fail → skip monster)
  │              └─ per damage line: shouldProc(prop, roll)
  │                   └─ dp.SpawnMeso(...) → Kafka COMMAND_TOPIC_DROP SPAWN
  └─ broadcast, projectiles, ... (unchanged)

atlas-drops handleSpawn → drop registry → CREATED event → channel writes
DropEnterTypeFresh spawn packet → player loots → RESERVED/PICKED_UP →
atlas-character credits mesos          (all existing, unchanged)
```

Tenancy: the buff/effect/monster processors and `producer.ProviderImpl(p.l)(p.ctx)`
all derive tenant from `ctx` exactly as the surrounding handler does — no new
tenant plumbing.

## 6. Error Handling

Identical contract to MP Eater — every failure logs and returns, never
propagates into the attack pipeline:

| Failure | Behavior |
|---|---|
| Buff REST error | `Errorf`, state disabled, attack unaffected |
| No PICK_POCKET buff / expired | silent (normal case), disabled |
| Effect lookup error | `Errorf`, disabled |
| `maxmeso <= 0` or `prop <= 0` | `Debugf`, disabled |
| Monster snapshot fetch error | `Debugf`, skip that monster's lines |
| Kafka emit error | `Errorf`, continue with remaining lines |

Successful procs log `Debugf` (character, monster, meso amount) per PRD
observability.

## 7. Testing

Unit (table-driven, pure functions — no test helpers, per project convention):

- `pickPocketWhitelisted`: all 8 whitelisted ids pass; Meso Explosion
  (4211006), Pick Pocket itself (4211003), a ranged skill, and a magic skill
  fail.
- `pickPocketMesoAmount`: d=0 → 1; d=20000, maxmeso=60 → 60 (clamp);
  d=10000, maxmeso=60 → 30; d huge → clamps to maxmeso; maxmeso=0 and
  maxmeso<0 → 0; d=333, maxmeso=60 → 1 (product 0.999 raised by the
  max(…, 1) floor); a mid-range row with a non-integer product (e.g.
  d=8500, maxmeso=60 → 25.5 → 25) asserting truncation, not rounding.
- `shouldProc`: existing MP Eater cases re-pointed at the renamed function
  (prop 0, negative, ≥1.0, roll below/above).

Handler-level (following `character_attack_mp_eater_test.go` /
`character_attack_common_test.go` deps-injection style):

- `pickPocketResolveState` with fake buff/effect getters: whitelist miss makes
  zero buff calls (assert via counting fake); buff error → disabled; no
  PICK_POCKET change → disabled; expired buff → disabled; happy path captures
  Amount() and Level()-resolved prop.
- `pickPocketTryProc` with fake monster getter + spawn emitter: N damage lines
  with prop 1.0 → N emissions in line order with correct meso amounts and
  positions; prop 0 → none; monster fetch error → none; emit error → remaining
  lines still attempted.
- Reflect interaction: existing `processDamageInfoEntry` tests extended to
  assert the widened `onDamageApplied(di)` hook still fires only on
  non-reflected entries (signature change is mechanical for the MP Eater
  tests).

Verification gate (PRD §10): `go test -race ./...`, `go vet ./...`,
`go build ./...` in atlas-channel; `docker buildx bake atlas-channel`;
`tools/redis-key-guard.sh` from repo root. No `go.mod` changes are expected
(all imports already in atlas-channel's module graph), but the bake runs
regardless.

## 8. Deviations from Cosmic (carried from PRD, confirmed viable)

1. No Java int-overflow `d − 2` artifact — use the actual unsigned damage.
2. No 100 ms per-drop stagger — all SPAWNs emitted immediately; `Mod = 0`
   (and Mod is dead in atlas-drops anyway, §2.1).
3. No GM max-level special-casing.
4. `maxmeso` from the buff-captured stat amount, not a re-read of current
   skill level.

## 9. Out of Scope

Meso Explosion consumption (TODO line 407), Bandit Steal, wiring `Mod`
through atlas-drops, buff-side changes, drop staggering, and every other TODO
in the block.
