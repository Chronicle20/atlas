# Pick Pocket Meso Spawn (task-149) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the Pick Pocket (Chief Bandit 4211003) proc in atlas-channel's common attack pipeline: while the PICK_POCKET buff is active and the attack skill is whitelisted, each damage line rolls the effect's prop; successes emit an atlas-drops `SPAWN` command spawning a meso-only FFA drop at the struck monster's position.

**Architecture:** All new code lives in atlas-channel. Pure helpers (`pickPocketWhitelisted`, `pickPocketMesoAmount`, generalized `shouldProc`) plus a per-attack state resolver and a per-monster proc function in `socket/handler/character_attack_common.go`, hooked through the `onDamageApplied` callback that fires exactly once per non-reflected damage-applying entry. Task-147 already widened that callback to `func(monsterId uint32, totalDamage uint32)`; this task widens its first parameter to `func(di packetmodel.DamageInfo, totalDamage uint32)` so Pick Pocket can roll per damage line (MP Eater and drain switch to `di.MonsterId()`). A new `SpawnMeso` producer (interface method + `*ProcessorImpl`) in the channel `drop` package emits the existing atlas-drops `SPAWN` contract (`Mesos > 0`, `ItemId == 0`). The proc is version-agnostic — it runs on the already-decoded `AttackInfo`, so it applies to every supported client version (GMS v48–v95, JMS v185) with no version gating. atlas-drops, atlas-buffs, and atlas-data are unchanged.

**Tech Stack:** Go, Kafka (segmentio/kafka-go via libs/atlas-kafka), logrus, libs/atlas-constants, table-driven Go tests.

## Global Constraints

- All paths are relative to the worktree root (`.worktrees/task-149-pickpocket-meso-spawn`). The Go module under change is `services/atlas-channel/atlas.com/channel` (module name `atlas-channel`).
- Verification gate before calling the branch done (PRD §10 + current CLAUDE.md Build & Verification): `go test -race ./...`, `go vet ./...`, `go build ./...` clean in the atlas-channel module; `docker buildx bake atlas-channel` from the worktree root; and from the worktree root — `tools/redis-key-guard.sh` (run WITHOUT a `GOWORK=off` prefix), `tools/goroutine-guard.sh` (this task adds no bare `go` statements, so trivially clean), and `tools/lint.sh --check` (gofumpt/goimports + linters; run `tools/lint.sh` with no flags first to auto-format before committing). The task touches no services.json / deploy / docker-bake / go.work, so the service-registration and template-opcode guards are N/A.
- Test conventions: table-driven tests on pure functions; deps-injection fakes via plain func fields/params; NO `*_testhelpers.go` files; handler tests live in `package handler`, drop tests in `package drop_test` (matching existing files).
- Failure isolation contract (PRD §4.5): every Pick Pocket failure path logs (`Errorf` for real failures, `Debugf` for expected skips) and returns; nothing propagates into the attack pipeline.
- Skill ids come from `libs/atlas-constants/skill` (`skill3` alias in the handler file); the temporary stat type from `libs/atlas-constants/character` (`charconst` alias). Never hardcode numeric ids in production code.
- Per-attack I/O budget (PRD §8): at most ONE buff REST call and ONE effect lookup per attack; ZERO lookups when the skill id is not whitelisted (whitelist check is pure and runs first).
- The `// TODO apply Pick Pocket` line must be gone by the final task; no new TODOs anywhere.
- **Version-agnostic (all supported versions: GMS v48/61/72/79/84/92/95, JMS v185).** The proc runs on the already-decoded `packetmodel.AttackInfo`, downstream of every version gate in `AttackInfo.Decode`; the whitelist keys off version-independent `libs/atlas-constants/skill` ids. No version gating is written anywhere in this task (design §1a). MP Eater and the task-147 drain heal work the same way. The acceptance check should be run on at least one legacy version (e.g. GMS v72) in addition to v83.
- **Rebased on main (task-147 landed first).** The `onDamageApplied` hook already has signature `func(monsterId uint32, totalDamage uint32)` and `character_attack_drain_test.go` already exercises it. Line numbers below are the current-main numbers; the drain arm in the `processAttack` closure must be preserved when the Pick Pocket arm is added.
- Commit after every task. Branch: `task-149-pickpocket-meso-spawn`.

---

### Task 1: Generalize `mpEaterShouldProc` into `shouldProc`

Pick Pocket reuses the exact prop-roll semantics MP Eater already has. One name, one function (design §4.1: duplicating it was rejected).

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:209-217` (the `mpEaterShouldProc` function) and `:299` (its call site inside `mpEaterTryProc`)
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_mp_eater_test.go` (rename `TestMpEaterShouldProc`)

**Interfaces:**
- Consumes: existing `mpEaterShouldProc(prop float64, roll float64) bool`.
- Produces: `shouldProc(prop float64, roll float64) bool` — same body, same semantics (`prop <= 0` never; `prop >= 1.0` always; else `roll < prop`). Tasks 4 and 6 call it.

- [ ] **Step 1: Re-point the existing test at the new name**

In `character_attack_mp_eater_test.go`, rename the test function and every call inside it — the cases stay byte-identical:

```go
func TestShouldProc(t *testing.T) {
	cases := []struct {
		name string
		prop float64
		roll float64
		want bool
	}{
		{"prop 1.0 always true", 1.0, 0.99, true},
		{"prop 1.0 with zero roll", 1.0, 0.0, true},
		{"prop 0.5 roll under", 0.5, 0.49, true},
		{"prop 0.5 roll equal", 0.5, 0.50, false},
		{"prop 0.5 roll over", 0.5, 0.51, false},
		{"prop 0.0 never", 0.0, 0.0, false},
		{"negative prop never", -1.0, 0.0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldProc(tc.prop, tc.roll); got != tc.want {
				t.Fatalf("shouldProc(%v, %v) = %v; want %v", tc.prop, tc.roll, got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `(cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestShouldProc -v)`
Expected: FAIL to build with `undefined: shouldProc`.

- [ ] **Step 3: Rename the function and its call site**

In `character_attack_common.go`, replace the `mpEaterShouldProc` declaration (lines 209–217) with:

```go
// shouldProc returns true when a prop-gated passive (MP Eater, Pick
// Pocket) should fire given the effect's prop and a single uniform roll
// in [0,1). Mirrors Cosmic's `prop == 1.0 || rand() < prop`. Defensive
// against negative props.
func shouldProc(prop float64, roll float64) bool {
	if prop <= 0 {
		return false
	}
	return prop >= 1.0 || roll < prop
}
```

And inside `mpEaterTryProc` change:

```go
	if !shouldProc(eaterEffect.Prop(), rand.Float64()) {
		return
	}
```

- [ ] **Step 4: Run the handler tests to verify they pass**

Run: `(cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -v)`
Expected: PASS (all existing handler tests, including the renamed `TestShouldProc`).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_mp_eater_test.go
git commit -m "refactor(channel): generalize mpEaterShouldProc into shouldProc"
```

---

### Task 2: Pure helpers — `pickPocketWhitelisted` and `pickPocketMesoAmount`

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` (append after `shouldProc`)
- Create (Test): `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_pick_pocket_test.go`

**Interfaces:**
- Consumes: `skill3` = `github.com/Chronicle20/atlas/libs/atlas-constants/skill` (already imported in the handler file); constants `RogueDoubleStabId` (4001334), `BanditSavageBlowId` (4201005), `ChiefBanditAssaulterId` (4211002), `ChiefBanditBandOfThievesId` (4211004), `ShadowerAssassinateId` (4221001), `ShadowerTauntId` (4221003), `ShadowerBoomerangStepId` (4221007) — all verified present in `libs/atlas-constants/skill/constants.go`.
- Produces: `pickPocketWhitelisted(skillId uint32) bool` and `pickPocketMesoAmount(damage uint32, maxmeso int32) uint32`. Tasks 4 and 6 call them.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_pick_pocket_test.go`:

```go
package handler

import (
	"testing"

	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

func TestPickPocketWhitelisted(t *testing.T) {
	cases := []struct {
		name    string
		skillId uint32
		want    bool
	}{
		{"basic attack", 0, true},
		{"Double Stab", uint32(skill3.RogueDoubleStabId), true},
		{"Savage Blow", uint32(skill3.BanditSavageBlowId), true},
		{"Assaulter", uint32(skill3.ChiefBanditAssaulterId), true},
		{"Band of Thieves", uint32(skill3.ChiefBanditBandOfThievesId), true},
		{"Assassinate", uint32(skill3.ShadowerAssassinateId), true},
		{"Taunt", uint32(skill3.ShadowerTauntId), true},
		{"Boomerang Step", uint32(skill3.ShadowerBoomerangStepId), true},
		{"Meso Explosion not whitelisted", uint32(skill3.ChiefBanditMesoExplosionId), false},
		{"Pick Pocket itself not whitelisted", uint32(skill3.ChiefBanditPickpocketId), false},
		{"ranged skill not whitelisted", uint32(skill3.BowmasterHurricaneId), false},
		{"magic skill not whitelisted", uint32(skill3.MagicianMagicClawId), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pickPocketWhitelisted(tc.skillId); got != tc.want {
				t.Fatalf("pickPocketWhitelisted(%d) = %v; want %v", tc.skillId, got, tc.want)
			}
		})
	}
}

func TestPickPocketMesoAmount(t *testing.T) {
	cases := []struct {
		name    string
		damage  uint32
		maxmeso int32
		want    uint32
	}{
		{"zero damage yields floor of 1", 0, 60, 1},
		{"exact maxmeso at 20000 damage", 20000, 60, 60},
		{"half maxmeso at 10000 damage", 10000, 60, 30},
		{"huge damage clamps to maxmeso", 2000000, 60, 60},
		{"zero maxmeso yields nothing", 10000, 0, 0},
		{"negative maxmeso yields nothing", 10000, -5, 0},
		{"product below 1 raised to floor (0.999)", 333, 60, 1},
		{"non-integer product truncates (25.5 -> 25)", 8500, 60, 25},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pickPocketMesoAmount(tc.damage, tc.maxmeso); got != tc.want {
				t.Fatalf("pickPocketMesoAmount(%d, %d) = %d; want %d", tc.damage, tc.maxmeso, got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `(cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run 'TestPickPocketWhitelisted|TestPickPocketMesoAmount' -v)`
Expected: FAIL to build with `undefined: pickPocketWhitelisted` (and `pickPocketMesoAmount`).

- [ ] **Step 3: Implement the helpers**

Append to `character_attack_common.go` (after `shouldProc`; `math`, `skill3` already imported):

```go
// pickPocketWhitelist is the fixed set of skills that can proc Pick
// Pocket (Cosmic AbstractDealDamageHandler parity). Basic attack
// (skillId == 0) is handled in pickPocketWhitelisted.
var pickPocketWhitelist = map[uint32]struct{}{
	uint32(skill3.RogueDoubleStabId):          {},
	uint32(skill3.BanditSavageBlowId):         {},
	uint32(skill3.ChiefBanditAssaulterId):     {},
	uint32(skill3.ChiefBanditBandOfThievesId): {},
	uint32(skill3.ShadowerAssassinateId):      {},
	uint32(skill3.ShadowerTauntId):            {},
	uint32(skill3.ShadowerBoomerangStepId):    {},
}

// pickPocketWhitelisted reports whether skillId can proc Pick Pocket.
func pickPocketWhitelisted(skillId uint32) bool {
	if skillId == 0 {
		return true
	}
	_, ok := pickPocketWhitelist[skillId]
	return ok
}

// pickPocketMesoAmount computes the meso payout for one damage line:
// min(max(damage/20000 * maxmeso, 1), maxmeso), float math then
// truncation, matching Cosmic. Returns 0 when maxmeso <= 0. A 0-damage
// line still yields 1 on a successful roll.
func pickPocketMesoAmount(damage uint32, maxmeso int32) uint32 {
	if maxmeso <= 0 {
		return 0
	}
	v := math.Max(float64(damage)/20000.0*float64(maxmeso), 1)
	if v > float64(maxmeso) {
		return uint32(maxmeso)
	}
	return uint32(v)
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `(cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run 'TestPickPocketWhitelisted|TestPickPocketMesoAmount' -v)`
Expected: PASS, all 20 subtests.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_pick_pocket_test.go
git commit -m "feat(channel): add Pick Pocket whitelist and meso amount helpers"
```

---

### Task 3: `SPAWN` producer in the channel `drop` package

atlas-channel's drop message contract today only declares `REQUEST_RESERVATION`. Add the `SPAWN` command (field names/JSON tags mirror atlas-drops' `CommandSpawnBody` at `services/atlas-drops/atlas.com/drops/kafka/message/drop/kafka.go:123-137` exactly, minus `EquipmentData` — the consumer zero-fills absent JSON keys and only reads equipment data for equip item ids).

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/drop/kafka.go:12-15` (const block)  and append the body type
- Modify: `services/atlas-channel/atlas.com/channel/drop/producer.go` (append provider)
- Modify: `services/atlas-channel/atlas.com/channel/drop/processor.go` — add `SpawnMeso` to the `Processor` **interface** (`:16-20`) AND implement it on `*ProcessorImpl` (mirroring `RequestReservation` at `:19`/`:51`)
- Create (Test): `services/atlas-channel/atlas.com/channel/drop/producer_test.go`

**Interfaces:**
- Consumes: `drop2.Command[E]` envelope, `producer.CreateKey`, `producer.SingleMessageProvider` (libs/atlas-kafka), `producer.ProviderImpl` (service-local `atlas-channel/kafka/producer`), `field.Model`.
- Produces:
  - `drop2.CommandTypeSpawn = "SPAWN"` and `drop2.SpawnCommandBody` (kafka message package)
  - `SpawnMesoCommandProvider(f field.Model, mesos uint32, x int16, y int16, ownerId uint32, dropperId uint32, dropperX int16, dropperY int16) model.Provider[[]kafka.Message]`
  - `SpawnMeso(...) error` on the `Processor` interface, implemented on `*ProcessorImpl` — Task 6/7 pass `dp.SpawnMeso` as the emit func.

- [ ] **Step 1: Write the failing provider test**

Create `services/atlas-channel/atlas.com/channel/drop/producer_test.go`:

```go
package drop_test

import (
	"atlas-channel/drop"
	drop2 "atlas-channel/kafka/message/drop"
	"encoding/json"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestSpawnMesoCommandProvider(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()

	msgs, err := drop.SpawnMesoCommandProvider(f, 123, 45, -67, 999, 700001, 40, -67)()
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d; want 1", len(msgs))
	}

	var cmd drop2.Command[drop2.SpawnCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cmd.Type != drop2.CommandTypeSpawn {
		t.Fatalf("Type = %q; want %q", cmd.Type, drop2.CommandTypeSpawn)
	}
	if cmd.WorldId != f.WorldId() || cmd.ChannelId != f.ChannelId() || cmd.MapId != f.MapId() {
		t.Fatalf("field routing = %d/%d/%d; want %d/%d/%d",
			cmd.WorldId, cmd.ChannelId, cmd.MapId, f.WorldId(), f.ChannelId(), f.MapId())
	}

	b := cmd.Body
	if b.Mesos != 123 || b.X != 45 || b.Y != -67 {
		t.Fatalf("mesos/x/y = %d/%d/%d; want 123/45/-67", b.Mesos, b.X, b.Y)
	}
	if b.OwnerId != 999 || b.OwnerPartyId != 0 {
		t.Fatalf("owner = %d/%d; want 999/0", b.OwnerId, b.OwnerPartyId)
	}
	if b.DropperId != 700001 || b.DropperX != 40 || b.DropperY != -67 {
		t.Fatalf("dropper = %d@(%d,%d); want 700001@(40,-67)", b.DropperId, b.DropperX, b.DropperY)
	}
	if b.ItemId != 0 || b.Quantity != 0 {
		t.Fatalf("itemId/quantity = %d/%d; want 0/0 (meso-only drop)", b.ItemId, b.Quantity)
	}
	if b.DropType != 2 {
		t.Fatalf("DropType = %d; want 2 (FFA)", b.DropType)
	}
	if !b.PlayerDrop {
		t.Fatal("PlayerDrop = false; want true (universal pickup)")
	}
	if b.Mod != 0 {
		t.Fatalf("Mod = %d; want 0", b.Mod)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `(cd services/atlas-channel/atlas.com/channel && go test ./drop/ -run TestSpawnMesoCommandProvider -v)`
Expected: FAIL to build with `undefined: drop.SpawnMesoCommandProvider` (and `drop2.SpawnCommandBody`).

- [ ] **Step 3: Add the message contract**

In `services/atlas-channel/atlas.com/channel/kafka/message/drop/kafka.go`, extend the command const block:

```go
const (
	EnvCommandTopic               = "COMMAND_TOPIC_DROP"
	CommandTypeRequestReservation = "REQUEST_RESERVATION"
	CommandTypeSpawn              = "SPAWN"
)
```

and append after `RequestReservationCommandBody`:

```go
// SpawnCommandBody mirrors atlas-drops' CommandSpawnBody field-for-field
// (minus EquipmentData, which is only read for equip item ids and
// zero-fills on decode when absent).
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

- [ ] **Step 4: Add the provider and processor method**

Append to `services/atlas-channel/atlas.com/channel/drop/producer.go`:

```go
// SpawnMesoCommandProvider emits a meso-only FFA drop: ItemId=0,
// Quantity=0, DropType=2 (client-visual FFA styling; the server ignores
// it), PlayerDrop=true (universal pickup via CanBeReservedBy),
// OwnerPartyId=0, Mod=0 (no client animation delay; atlas-drops discards
// the field today).
func SpawnMesoCommandProvider(f field.Model, mesos uint32, x int16, y int16, ownerId uint32, dropperId uint32, dropperX int16, dropperY int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(dropperId))
	value := &drop2.Command[drop2.SpawnCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		Type:      drop2.CommandTypeSpawn,
		Body: drop2.SpawnCommandBody{
			ItemId:       0,
			Quantity:     0,
			Mesos:        mesos,
			DropType:     2,
			X:            x,
			Y:            y,
			OwnerId:      ownerId,
			OwnerPartyId: 0,
			DropperId:    dropperId,
			DropperX:     dropperX,
			DropperY:     dropperY,
			PlayerDrop:   true,
			Mod:          0,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

In `services/atlas-channel/atlas.com/channel/drop/processor.go`, add the method to the `Processor` interface (alongside `RequestReservation`):

```go
type Processor interface {
	// ... existing methods (InMapModelProvider, ForEachInMap, RequestReservation) ...
	SpawnMeso(f field.Model, mesos uint32, x int16, y int16, ownerId uint32, dropperId uint32, dropperX int16, dropperY int16) error
}
```

and implement it on `*ProcessorImpl` (mirroring `RequestReservation`):

```go
func (p *ProcessorImpl) SpawnMeso(f field.Model, mesos uint32, x int16, y int16, ownerId uint32, dropperId uint32, dropperX int16, dropperY int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(drop2.EnvCommandTopic)(SpawnMesoCommandProvider(f, mesos, x, y, ownerId, dropperId, dropperX, dropperY))
}
```

(Both files already import everything needed: `drop2`, `field`, `model`, kafka `producer`, service `producer`.)

- [ ] **Step 5: Run the test to verify it passes**

Run: `(cd services/atlas-channel/atlas.com/channel && go test ./drop/ ./kafka/... -v)`
Expected: PASS, including `TestSpawnMesoCommandProvider`.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/drop/kafka.go services/atlas-channel/atlas.com/channel/drop/producer.go services/atlas-channel/atlas.com/channel/drop/processor.go services/atlas-channel/atlas.com/channel/drop/producer_test.go
git commit -m "feat(channel): add meso SPAWN producer to drop package"
```

---

### Task 4: Per-attack state — `pickPocketState` and `pickPocketResolveState`

One whitelist check (pure, first), at most one buff REST call, at most one effect lookup. Every failure disables the proc and is swallowed (design §3.3, §6).

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` (append after `pickPocketMesoAmount`; add imports `"atlas-channel/character/buff"` and `charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"`)
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_pick_pocket_test.go` (append)

**Interfaces:**
- Consumes: `pickPocketWhitelisted` (Task 2); `buff.Model` (`Level() byte`, `Expired() bool`, `Changes() []stat.Model`), `stat.Model` (`Type() string`, `Amount() int32`) from `atlas-channel/character/buff`; `effect.Model.Prop() float64`; `charconst.TemporaryStatTypePickPocket` (`"PICK_POCKET"`); `skill3.ChiefBanditPickpocketId` (4211003).
- Produces:

```go
type pickPocketState struct {
	enabled bool
	maxmeso int32
	prop    float64
}

func pickPocketResolveState(
	l logrus.FieldLogger,
	getBuffs func(characterId uint32) ([]buff.Model, error),
	getEffect func(uniqueId uint32, level byte) (effect.Model, error),
	skillId uint32,
	characterId uint32,
) pickPocketState
```

Task 6 reads `pickPocketState`; Task 7 calls `pickPocketResolveState` with `buff.NewProcessor(l, ctx).GetByCharacterId` and `skill2.NewProcessor(l, ctx).GetEffect` as the two getters.

- [ ] **Step 1: Write the failing tests**

Append to `character_attack_pick_pocket_test.go` (extend the import block with `"atlas-channel/character/buff"`, `"atlas-channel/character/buff/stat"`, `"atlas-channel/data/skill/effect"`, `"errors"`, `"time"`, `charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"`, `"github.com/sirupsen/logrus"`):

```go
// ppTestBuff builds an active (or expired) buff carrying a PICK_POCKET
// stat change of the given amount at the given level.
func ppTestBuff(level byte, amount int32, expired bool) buff.Model {
	expiresAt := time.Now().Add(time.Minute)
	if expired {
		expiresAt = time.Now().Add(-time.Minute)
	}
	return buff.NewBuff(
		int32(skill3.ChiefBanditPickpocketId),
		level,
		60000,
		[]stat.Model{stat.NewStat(string(charconst.TemporaryStatTypePickPocket), amount)},
		time.Now().Add(-time.Second),
		expiresAt,
	)
}

func ppTestEffect(t *testing.T, prop float64) effect.Model {
	t.Helper()
	se, err := effect.Extract(effect.RestModel{Prop: prop})
	if err != nil {
		t.Fatalf("effect.Extract: %v", err)
	}
	return se
}

func TestPickPocketResolveState_NonWhitelistedSkillMakesNoLookups(t *testing.T) {
	buffCalls := 0
	getBuffs := func(characterId uint32) ([]buff.Model, error) {
		buffCalls++
		return []buff.Model{ppTestBuff(15, 40, false)}, nil
	}
	getEffect := func(uniqueId uint32, level byte) (effect.Model, error) {
		t.Fatal("effect lookup must not run for a non-whitelisted skill")
		return effect.Model{}, nil
	}

	st := pickPocketResolveState(logrus.New(), getBuffs, getEffect, uint32(skill3.ChiefBanditMesoExplosionId), 1)

	if st.enabled {
		t.Fatal("state.enabled = true; want false for non-whitelisted skill")
	}
	if buffCalls != 0 {
		t.Fatalf("buff lookups = %d; want 0 for non-whitelisted skill", buffCalls)
	}
}

func TestPickPocketResolveState_BuffLookupErrorDisables(t *testing.T) {
	getBuffs := func(characterId uint32) ([]buff.Model, error) {
		return nil, errors.New("atlas-buffs unavailable")
	}
	getEffect := func(uniqueId uint32, level byte) (effect.Model, error) {
		t.Fatal("effect lookup must not run when the buff lookup fails")
		return effect.Model{}, nil
	}

	st := pickPocketResolveState(logrus.New(), getBuffs, getEffect, 0, 1)

	if st.enabled {
		t.Fatal("state.enabled = true; want false on buff lookup error")
	}
}

func TestPickPocketResolveState_NoPickPocketBuffDisables(t *testing.T) {
	other := buff.NewBuff(2311003, 10, 60000,
		[]stat.Model{stat.NewStat("HOLY_SYMBOL", 150)},
		time.Now(), time.Now().Add(time.Minute))
	getBuffs := func(characterId uint32) ([]buff.Model, error) {
		return []buff.Model{other}, nil
	}
	getEffect := func(uniqueId uint32, level byte) (effect.Model, error) {
		t.Fatal("effect lookup must not run without a PICK_POCKET buff")
		return effect.Model{}, nil
	}

	st := pickPocketResolveState(logrus.New(), getBuffs, getEffect, 0, 1)

	if st.enabled {
		t.Fatal("state.enabled = true; want false without a PICK_POCKET buff")
	}
}

func TestPickPocketResolveState_ExpiredBuffDisables(t *testing.T) {
	getBuffs := func(characterId uint32) ([]buff.Model, error) {
		return []buff.Model{ppTestBuff(15, 40, true)}, nil
	}
	getEffect := func(uniqueId uint32, level byte) (effect.Model, error) {
		t.Fatal("effect lookup must not run for an expired buff")
		return effect.Model{}, nil
	}

	st := pickPocketResolveState(logrus.New(), getBuffs, getEffect, 0, 1)

	if st.enabled {
		t.Fatal("state.enabled = true; want false for an expired buff")
	}
}

func TestPickPocketResolveState_NonPositiveMaxMesoDisables(t *testing.T) {
	getBuffs := func(characterId uint32) ([]buff.Model, error) {
		return []buff.Model{ppTestBuff(15, 0, false)}, nil
	}
	getEffect := func(uniqueId uint32, level byte) (effect.Model, error) {
		t.Fatal("effect lookup must not run when maxmeso <= 0")
		return effect.Model{}, nil
	}

	st := pickPocketResolveState(logrus.New(), getBuffs, getEffect, 0, 1)

	if st.enabled {
		t.Fatal("state.enabled = true; want false when maxmeso <= 0")
	}
}

func TestPickPocketResolveState_EffectLookupErrorDisables(t *testing.T) {
	getBuffs := func(characterId uint32) ([]buff.Model, error) {
		return []buff.Model{ppTestBuff(15, 40, false)}, nil
	}
	getEffect := func(uniqueId uint32, level byte) (effect.Model, error) {
		return effect.Model{}, errors.New("atlas-data unavailable")
	}

	st := pickPocketResolveState(logrus.New(), getBuffs, getEffect, 0, 1)

	if st.enabled {
		t.Fatal("state.enabled = true; want false on effect lookup error")
	}
}

func TestPickPocketResolveState_NonPositivePropDisables(t *testing.T) {
	getBuffs := func(characterId uint32) ([]buff.Model, error) {
		return []buff.Model{ppTestBuff(15, 40, false)}, nil
	}
	getEffect := func(uniqueId uint32, level byte) (effect.Model, error) {
		return ppTestEffect(t, 0), nil
	}

	st := pickPocketResolveState(logrus.New(), getBuffs, getEffect, 0, 1)

	if st.enabled {
		t.Fatal("state.enabled = true; want false when prop <= 0")
	}
}

func TestPickPocketResolveState_HappyPath(t *testing.T) {
	var gotUniqueId uint32
	var gotLevel byte
	getBuffs := func(characterId uint32) ([]buff.Model, error) {
		return []buff.Model{ppTestBuff(15, 40, false)}, nil
	}
	getEffect := func(uniqueId uint32, level byte) (effect.Model, error) {
		gotUniqueId = uniqueId
		gotLevel = level
		return ppTestEffect(t, 0.6), nil
	}

	st := pickPocketResolveState(logrus.New(), getBuffs, getEffect, uint32(skill3.ShadowerBoomerangStepId), 1)

	if !st.enabled {
		t.Fatal("state.enabled = false; want true on happy path")
	}
	if st.maxmeso != 40 {
		t.Fatalf("maxmeso = %d; want 40 (buff-captured stat amount)", st.maxmeso)
	}
	if st.prop != 0.6 {
		t.Fatalf("prop = %v; want 0.6", st.prop)
	}
	if gotUniqueId != uint32(skill3.ChiefBanditPickpocketId) {
		t.Fatalf("effect looked up skill %d; want %d (Pick Pocket)", gotUniqueId, uint32(skill3.ChiefBanditPickpocketId))
	}
	if gotLevel != 15 {
		t.Fatalf("effect looked up level %d; want 15 (buff-captured level)", gotLevel)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `(cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestPickPocketResolveState -v)`
Expected: FAIL to build with `undefined: pickPocketResolveState` (and `pickPocketState`).

- [ ] **Step 3: Implement `pickPocketState` and `pickPocketResolveState`**

Append to `character_attack_common.go`. Add `"atlas-channel/character/buff"` to the first import group and `charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"` to the second:

```go
// pickPocketState is the per-attack Pick Pocket context, resolved once
// before the DamageInfo loop (design §3.3): whitelist gate first (pure,
// no I/O), then at most one buff REST call and one effect lookup.
type pickPocketState struct {
	enabled bool
	maxmeso int32   // PICK_POCKET stat Amount() captured at buff time
	prop    float64 // effect prop at the buff's captured Level()
}

// pickPocketResolveState gates and resolves Pick Pocket for one attack.
// Any failure (buff REST error, effect lookup error) or non-positive
// maxmeso/prop yields a disabled state; errors are logged and swallowed,
// never propagated into the attack pipeline.
func pickPocketResolveState(
	l logrus.FieldLogger,
	getBuffs func(characterId uint32) ([]buff.Model, error),
	getEffect func(uniqueId uint32, level byte) (effect.Model, error),
	skillId uint32,
	characterId uint32,
) pickPocketState {
	if !pickPocketWhitelisted(skillId) {
		return pickPocketState{}
	}

	buffs, err := getBuffs(characterId)
	if err != nil {
		l.WithError(err).Errorf("Pick Pocket: buff lookup failed for character [%d].", characterId)
		return pickPocketState{}
	}

	var maxmeso int32
	var level byte
	found := false
	for _, b := range buffs {
		if b.Expired() {
			continue
		}
		for _, ch := range b.Changes() {
			if ch.Type() == string(charconst.TemporaryStatTypePickPocket) {
				maxmeso = ch.Amount()
				level = b.Level()
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		return pickPocketState{}
	}
	if maxmeso <= 0 {
		l.Debugf("Pick Pocket: non-positive maxmeso [%d] for character [%d]; proc disabled.", maxmeso, characterId)
		return pickPocketState{}
	}

	se, err := getEffect(uint32(skill3.ChiefBanditPickpocketId), level)
	if err != nil {
		l.WithError(err).Errorf("Pick Pocket: effect lookup failed at level [%d] for character [%d].", level, characterId)
		return pickPocketState{}
	}
	if se.Prop() <= 0 {
		l.Debugf("Pick Pocket: non-positive prop at level [%d] for character [%d]; proc disabled.", level, characterId)
		return pickPocketState{}
	}

	return pickPocketState{enabled: true, maxmeso: maxmeso, prop: se.Prop()}
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `(cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestPickPocketResolveState -v)`
Expected: PASS, all 8 tests.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_pick_pocket_test.go
git commit -m "feat(channel): resolve Pick Pocket per-attack state"
```

---

### Task 5: Widen the `onDamageApplied` hook to carry the `DamageInfo`

The hook fires exactly once per non-reflected, damage-applying entry — precisely the set Pick Pocket must see (design §3.2, option B). **Task-147 already widened it to `func(monsterId uint32, totalDamage uint32)`**, and `character_attack_drain_test.go` already pins its firing + reflect semantics. Pick Pocket needs the *per-line* damages (the summed `totalDamage` discards them), so widen the FIRST parameter from `monsterId uint32` to `di packetmodel.DamageInfo`, yielding `func(di packetmodel.DamageInfo, totalDamage uint32)`. MP Eater and drain switch their call sites to `di.MonsterId()`; drain keeps `totalDamage` verbatim.

This signature change **breaks the three existing drain hook tests** (they declare the old `func(monsterId uint32, totalDamage uint32)` shape) — update them mechanically to the new shape, reading `di.MonsterId()`. Do **not** re-add a reflected-entry test: drain's `TestOnDamageApplied_NotCalledForReflectedEntry` already covers it (design §7). Add exactly ONE new test asserting the widened hook carries the per-line `di.Damages()` — the capability Pick Pocket depends on.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:107-111` (deps field), `:197-206` (invocation), `:444-451` (MP Eater + drain closure in `processAttack`)
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_drain_test.go` (update the three `TestOnDamageApplied_*` tests to the new signature)
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_pick_pocket_test.go` (append one per-line-damages test)

**Interfaces:**
- Consumes: `damageInfoEntryDeps`, `processDamageInfoEntry` (both existing), `packetmodel.NewDamageInfo(hits).SetMonsterId().SetDamages()`, `packetmodel.NewAttackInfo(attackType)`, the `testDrainField()` / `testTenant(t)` helpers already in `character_attack_drain_test.go` (same package).
- Produces: `damageInfoEntryDeps.onDamageApplied` with signature `func(di packetmodel.DamageInfo, totalDamage uint32)`. Task 7's closure relies on this.

- [ ] **Step 1: Update the three existing drain hook tests to the new signature**

In `character_attack_drain_test.go`, the `onDamageApplied` closures currently take `(monsterId uint32, totalDamage uint32)`. Change each to `(di packetmodel.DamageInfo, totalDamage uint32)` and read `di.MonsterId()` where the old `monsterId` param was used. Three edits:

`TestOnDamageApplied_ReceivesSummedDamageTotal`:

```go
		onDamageApplied: func(di packetmodel.DamageInfo, totalDamage uint32) {
			calls++
			gotMonsterId = di.MonsterId()
			gotTotal = totalDamage
		},
```

`TestOnDamageApplied_NotCalledForZeroDamageEntry`:

```go
		onDamageApplied: func(_ packetmodel.DamageInfo, _ uint32) { called = true },
```

`TestOnDamageApplied_NotCalledForReflectedEntry`:

```go
		onDamageApplied:   func(_ packetmodel.DamageInfo, _ uint32) { called = true },
```

(`packetmodel` is already imported in this file.)

- [ ] **Step 2: Add one new test pinning the per-line damages the widened hook must carry**

Append to `character_attack_pick_pocket_test.go` (extend imports with `packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"`; the `testDrainField()` and `testTenant(t)` helpers live in `character_attack_drain_test.go`, same package):

```go
// TestOnDamageApplied_CarriesPerLineDamages pins the reason the hook was
// widened to carry the DamageInfo: Pick Pocket rolls once per damage line,
// so the per-line breakdown (not just the summed total) must reach the hook.
func TestOnDamageApplied_CarriesPerLineDamages(t *testing.T) {
	ai := *packetmodel.NewAttackInfo(packetmodel.AttackTypeMelee)
	di := *packetmodel.NewDamageInfo(3).SetMonsterId(4101).SetDamages([]uint32{100, 250, 400})

	var gotMonsterId uint32
	var gotLines []uint32
	calls := 0
	deps := damageInfoEntryDeps{
		applyDamage: func(_ field.Model, _, _ uint32, _ []uint32, _ byte) error { return nil },
		onDamageApplied: func(di packetmodel.DamageInfo, _ uint32) {
			calls++
			gotMonsterId = di.MonsterId()
			gotLines = di.Damages()
		},
	}

	processDamageInfoEntry(discardLogger(), di, ai, effect.Model{}, 1, 999, 0, 0, testDrainField(), testTenant(t), "", deps)

	if calls != 1 {
		t.Fatalf("onDamageApplied calls = %d; want 1", calls)
	}
	if gotMonsterId != 4101 {
		t.Errorf("monsterId = %d; want 4101", gotMonsterId)
	}
	if len(gotLines) != 3 || gotLines[0] != 100 || gotLines[1] != 250 || gotLines[2] != 400 {
		t.Fatalf("per-line damages = %v; want [100 250 400]", gotLines)
	}
}
```

- [ ] **Step 3: Run the tests to verify they fail**

Run: `(cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestOnDamageApplied -v)`
Expected: FAIL to build — `cannot use func literal (type func(packetmodel.DamageInfo, uint32)) as type func(uint32, uint32)` on the `onDamageApplied` assignments (the hook still takes `func(monsterId uint32, totalDamage uint32)`).

- [ ] **Step 4: Widen the hook**

In `character_attack_common.go` make three edits.

The deps field (lines 107–111):

```go
	// onDamageApplied is invoked once per non-reflected DamageInfo after
	// damage and status apply, with the entry's summed damage (clamped to
	// MaxUint32). Optional; nil-safe. Used by passives that fire per
	// damaged monster (MP Eater, drain-family heals, Pick Pocket).
	onDamageApplied func(di packetmodel.DamageInfo, totalDamage uint32)
```

The invocation at the end of `processDamageInfoEntry` (lines 197–206) — pass `di` alongside the existing `total`:

```go
	if deps.onDamageApplied != nil {
		var total uint64
		for _, d := range damages {
			total += uint64(d)
		}
		if total > math.MaxUint32 {
			total = math.MaxUint32
		}
		deps.onDamageApplied(di, uint32(total))
	}
```

The closure inside `processAttack` (lines 444–451) — **preserve the drain arm added by task-147**; MP Eater and drain now read `di.MonsterId()`:

```go
			onDamageApplied: func(di packetmodel.DamageInfo, totalDamage uint32) {
				if ai.AttackType() == packetmodel.AttackTypeMagic && ai.SkillId() > 0 {
					mpEaterTryProc(l, ctx, mp, c, di.MonsterId(), s.Field(), s.CharacterId())
				}
				if ai.SkillId() > 0 && isDrainSkill(skill3.Id(ai.SkillId())) {
					drainTryHeal(l, mp.GetById, cp.ChangeHP, loadEffectiveStats, se.X(), ai.SkillId(), di.MonsterId(), totalDamage, s.Field(), s.CharacterId())
				}
			},
```

(The Pick Pocket arm is added to this same closure in Task 7.)

- [ ] **Step 5: Run the handler tests to verify they pass**

Run: `(cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -v)`
Expected: PASS — the new `TestOnDamageApplied_CarriesPerLineDamages`, the three updated drain hook tests, and every pre-existing handler test.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_drain_test.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_pick_pocket_test.go
git commit -m "refactor(channel): widen onDamageApplied hook to carry DamageInfo"
```

---

### Task 6: Per-monster proc — `pickPocketTryProc`

Rolls each damage line of one non-reflected entry and emits one SPAWN per success. Monster snapshot fetch failure skips the monster; emit failure continues with remaining lines (PRD §4.3–4.5).

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` (append after `pickPocketResolveState`)
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_pick_pocket_test.go` (append)

**Interfaces:**
- Consumes: `pickPocketState` (Task 4), `shouldProc` (Task 1), `pickPocketMesoAmount` (Task 2), `monster.Model` (`X() int16`, `Y() int16`), `packetmodel.DamageInfo` (`MonsterId() uint32`, `Damages() []uint32`), `rand` (already imported).
- Produces:

```go
func pickPocketTryProc(
	l logrus.FieldLogger,
	getMonster func(monsterId uint32) (monster.Model, error),
	spawnMeso func(f field.Model, mesos uint32, x int16, y int16, ownerId uint32, dropperId uint32, dropperX int16, dropperY int16) error,
	state pickPocketState,
	di packetmodel.DamageInfo,
	f field.Model,
	characterId uint32,
)
```

Task 7 calls it with `mp.GetById` and `dp.SpawnMeso` (Task 3) as the two funcs.

- [ ] **Step 1: Write the failing tests**

Append to `character_attack_pick_pocket_test.go`:

```go
type ppSpawnCall struct {
	mesos              uint32
	x, y               int16
	ownerId, dropperId uint32
	dropperX, dropperY int16
}

func TestPickPocketTryProc_PropOneEmitsPerDamageLine(t *testing.T) {
	f := testField(_map.Id(100000000))
	di := packetmodel.NewDamageInfo(3).SetMonsterId(700020).SetDamages([]uint32{0, 10000, 2000000})
	mon := monster.NewModelBuilder(700020, f, 100100).SetX(100).SetY(-30).MustBuild()

	var calls []ppSpawnCall
	getMonster := func(monsterId uint32) (monster.Model, error) {
		if monsterId != 700020 {
			t.Fatalf("getMonster(%d); want 700020", monsterId)
		}
		return mon, nil
	}
	spawnMeso := func(_ field.Model, mesos uint32, x int16, y int16, ownerId uint32, dropperId uint32, dropperX int16, dropperY int16) error {
		calls = append(calls, ppSpawnCall{mesos, x, y, ownerId, dropperId, dropperX, dropperY})
		return nil
	}

	state := pickPocketState{enabled: true, maxmeso: 60, prop: 1.0}
	pickPocketTryProc(logrus.New(), getMonster, spawnMeso, state, *di, f, 42)

	if len(calls) != 3 {
		t.Fatalf("spawn calls = %d; want 3 (one per damage line at prop 1.0)", len(calls))
	}
	// Damage-line order: 0 -> floor 1; 10000 -> 30; 2000000 -> clamp 60.
	wantMesos := []uint32{1, 30, 60}
	for i, c := range calls {
		if c.mesos != wantMesos[i] {
			t.Fatalf("call %d mesos = %d; want %d", i, c.mesos, wantMesos[i])
		}
		if c.x < mon.X()-50 || c.x > mon.X()+49 {
			t.Fatalf("call %d x = %d; want within [%d, %d]", i, c.x, mon.X()-50, mon.X()+49)
		}
		if c.y != mon.Y() {
			t.Fatalf("call %d y = %d; want monster y %d", i, c.y, mon.Y())
		}
		if c.ownerId != 42 {
			t.Fatalf("call %d ownerId = %d; want 42", i, c.ownerId)
		}
		if c.dropperId != 700020 || c.dropperX != mon.X() || c.dropperY != mon.Y() {
			t.Fatalf("call %d dropper = %d@(%d,%d); want 700020@(%d,%d)", i, c.dropperId, c.dropperX, c.dropperY, mon.X(), mon.Y())
		}
	}
}

func TestPickPocketTryProc_PropZeroEmitsNothing(t *testing.T) {
	f := testField(_map.Id(100000000))
	di := packetmodel.NewDamageInfo(2).SetMonsterId(700021).SetDamages([]uint32{100, 200})
	mon := monster.NewModelBuilder(700021, f, 100100).SetX(0).SetY(0).MustBuild()

	emitted := 0
	getMonster := func(_ uint32) (monster.Model, error) { return mon, nil }
	spawnMeso := func(_ field.Model, _ uint32, _ int16, _ int16, _ uint32, _ uint32, _ int16, _ int16) error {
		emitted++
		return nil
	}

	state := pickPocketState{enabled: true, maxmeso: 60, prop: 0}
	pickPocketTryProc(logrus.New(), getMonster, spawnMeso, state, *di, f, 42)

	if emitted != 0 {
		t.Fatalf("spawn calls = %d; want 0 at prop 0", emitted)
	}
}

func TestPickPocketTryProc_DisabledStateMakesNoLookups(t *testing.T) {
	f := testField(_map.Id(100000000))
	di := packetmodel.NewDamageInfo(1).SetMonsterId(700022).SetDamages([]uint32{100})

	getMonster := func(_ uint32) (monster.Model, error) {
		t.Fatal("getMonster must not run for a disabled state")
		return monster.Model{}, nil
	}
	spawnMeso := func(_ field.Model, _ uint32, _ int16, _ int16, _ uint32, _ uint32, _ int16, _ int16) error {
		t.Fatal("spawnMeso must not run for a disabled state")
		return nil
	}

	pickPocketTryProc(logrus.New(), getMonster, spawnMeso, pickPocketState{}, *di, f, 42)
}

func TestPickPocketTryProc_MonsterFetchErrorSkipsMonster(t *testing.T) {
	f := testField(_map.Id(100000000))
	di := packetmodel.NewDamageInfo(2).SetMonsterId(700023).SetDamages([]uint32{100, 200})

	getMonster := func(_ uint32) (monster.Model, error) {
		return monster.Model{}, errors.New("snapshot gone")
	}
	spawnMeso := func(_ field.Model, _ uint32, _ int16, _ int16, _ uint32, _ uint32, _ int16, _ int16) error {
		t.Fatal("spawnMeso must not run when the monster fetch fails")
		return nil
	}

	state := pickPocketState{enabled: true, maxmeso: 60, prop: 1.0}
	pickPocketTryProc(logrus.New(), getMonster, spawnMeso, state, *di, f, 42)
}

func TestPickPocketTryProc_EmitErrorContinuesRemainingLines(t *testing.T) {
	f := testField(_map.Id(100000000))
	di := packetmodel.NewDamageInfo(3).SetMonsterId(700024).SetDamages([]uint32{100, 200, 300})
	mon := monster.NewModelBuilder(700024, f, 100100).SetX(0).SetY(0).MustBuild()

	attempts := 0
	getMonster := func(_ uint32) (monster.Model, error) { return mon, nil }
	spawnMeso := func(_ field.Model, _ uint32, _ int16, _ int16, _ uint32, _ uint32, _ int16, _ int16) error {
		attempts++
		return errors.New("kafka down")
	}

	state := pickPocketState{enabled: true, maxmeso: 60, prop: 1.0}
	pickPocketTryProc(logrus.New(), getMonster, spawnMeso, state, *di, f, 42)

	if attempts != 3 {
		t.Fatalf("spawn attempts = %d; want 3 (emit errors must not stop remaining lines)", attempts)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `(cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestPickPocketTryProc -v)`
Expected: FAIL to build with `undefined: pickPocketTryProc`.

- [ ] **Step 3: Implement `pickPocketTryProc`**

Append to `character_attack_common.go`:

```go
// pickPocketTryProc rolls each damage line of one non-reflected
// DamageInfo and emits one meso SPAWN per success. Monster snapshot
// fetch failure skips this monster's procs (Debugf); emit failures are
// logged (Errorf) and swallowed, continuing with the remaining lines.
func pickPocketTryProc(
	l logrus.FieldLogger,
	getMonster func(monsterId uint32) (monster.Model, error),
	spawnMeso func(f field.Model, mesos uint32, x int16, y int16, ownerId uint32, dropperId uint32, dropperX int16, dropperY int16) error,
	state pickPocketState,
	di packetmodel.DamageInfo,
	f field.Model,
	characterId uint32,
) {
	if !state.enabled {
		return
	}

	mon, err := getMonster(di.MonsterId())
	if err != nil {
		l.WithError(err).Debugf("Pick Pocket: monster [%d] snapshot fetch failed; skipping its procs.", di.MonsterId())
		return
	}

	for _, d := range di.Damages() {
		if !shouldProc(state.prop, rand.Float64()) {
			continue
		}
		mesos := pickPocketMesoAmount(d, state.maxmeso)
		l.Debugf("Pick Pocket proc: character=[%d] monster=[%d] mesos=[%d].", characterId, di.MonsterId(), mesos)
		x := mon.X() + int16(rand.Intn(100)-50)
		if sErr := spawnMeso(f, mesos, x, mon.Y(), characterId, di.MonsterId(), mon.X(), mon.Y()); sErr != nil {
			l.WithError(sErr).Errorf("Pick Pocket: SPAWN emit failed for monster [%d] character [%d].", di.MonsterId(), characterId)
		}
	}
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `(cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestPickPocketTryProc -v)`
Expected: PASS, all 5 tests.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_pick_pocket_test.go
git commit -m "feat(channel): add Pick Pocket per-monster proc"
```

---

### Task 7: Wire the proc into `processAttack`, remove the TODO, run the full gate

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` — `processAttack` body (state resolution + closure) and the TODO block (`// TODO apply Pick Pocket` at `:504` on current main; shifted by earlier tasks' edits)

**Interfaces:**
- Consumes: everything produced by Tasks 1–6; `buff.NewProcessor(l, ctx)` (`atlas-channel/character/buff`), `drop.NewProcessor(l, ctx)` (`atlas-channel/drop`), `skill2.NewProcessor(l, ctx).GetEffect`, `mp.GetById`.
- Produces: the complete feature; no interface for later tasks.

- [ ] **Step 1: Wire the state resolution and the closure**

In `character_attack_common.go`, add `"atlas-channel/drop"` to the first import group (`"atlas-channel/character/buff"` was added in Task 4).

In `processAttack`, immediately before the `deps := damageInfoEntryDeps{` construction (i.e. after the `attackKind := ...` line and the task-147 `loadEffectiveStats` closure block), insert:

```go
					// Pick Pocket per-attack state: whitelist gate first
					// (pure, no I/O), then at most one buff REST call and
					// one effect lookup. Failures disable the proc and
					// never abort the attack.
					dp := drop.NewProcessor(l, ctx)
					ppState := pickPocketResolveState(
						l,
						buff.NewProcessor(l, ctx).GetByCharacterId,
						skill2.NewProcessor(l, ctx).GetEffect,
						ai.SkillId(),
						s.CharacterId(),
					)
```

Replace the `onDamageApplied` closure (the Task 5 version — MP Eater + drain arms) with the version that adds the Pick Pocket arm, **keeping the existing MP Eater and drain arms**:

```go
						// Per-monster passives, fired once per non-reflected
						// entry after damage and status apply. Failures are
						// swallowed so the rest of the attack pipeline is
						// unaffected.
						onDamageApplied: func(di packetmodel.DamageInfo, totalDamage uint32) {
							if ai.AttackType() == packetmodel.AttackTypeMagic && ai.SkillId() > 0 {
								mpEaterTryProc(l, ctx, mp, c, di.MonsterId(), s.Field(), s.CharacterId())
							}
							if ai.SkillId() > 0 && isDrainSkill(skill3.Id(ai.SkillId())) {
								drainTryHeal(l, mp.GetById, cp.ChangeHP, loadEffectiveStats, se.X(), ai.SkillId(), di.MonsterId(), totalDamage, s.Field(), s.CharacterId())
							}
							if ppState.enabled {
								pickPocketTryProc(l, mp.GetById, dp.SpawnMeso, ppState, di, s.Field(), s.CharacterId())
							}
						},
```

Delete the line `// TODO apply Pick Pocket` (`:504` on current main) from the TODO block near the end of `processAttack`. Leave every other TODO line untouched.

- [ ] **Step 2: Verify the TODO is gone and the module compiles**

Run: `grep -n "TODO apply Pick Pocket" services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go || echo "TODO removed"`
Expected: `TODO removed`.

Run: `(cd services/atlas-channel/atlas.com/channel && go build ./...)`
Expected: clean exit, no output.

- [ ] **Step 3: Run the full atlas-channel test suite with race detector**

Run: `(cd services/atlas-channel/atlas.com/channel && go test -race ./...)`
Expected: `ok` for every package, no failures.

- [ ] **Step 4: Run vet**

Run: `(cd services/atlas-channel/atlas.com/channel && go vet ./...)`
Expected: clean exit, no output.

- [ ] **Step 5: Run the repo-root guards**

Run (from the worktree root):
- `tools/redis-key-guard.sh` — clean (no keyed Redis commands outside libs/atlas-redis; this change adds none).
- `tools/goroutine-guard.sh` — clean (no bare `go` statements added).
- `tools/lint.sh` (no flags) to auto-format, then `tools/lint.sh --check` — clean.

- [ ] **Step 6: Bake the atlas-channel image**

Run: `docker buildx bake atlas-channel` (from the worktree root)
Expected: build completes successfully. No `go.mod` was touched, but the gate runs regardless (CLAUDE.md Build & Verification rule 4).

- [ ] **Step 7: Commit (include any lint auto-format changes)**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go
git commit -m "feat(channel): wire Pick Pocket meso spawn into attack pipeline"
```

- [ ] **Step 8: Version acceptance check (design §1a)**

The proc is version-agnostic (runs on the decoded `AttackInfo`), but confirm it actually fires end-to-end on more than the v83 baseline: exercise a whitelisted melee attack (or basic attack, `skillId == 0`) with Pick Pocket active on a **legacy-version tenant (e.g. GMS v72)** and verify meso drops spawn and are lootable. Whitelisted skills that don't exist in that client are simply never cast — no gating required.

---

## Acceptance Criteria Traceability (PRD §10)

| Criterion | Covered by |
|---|---|
| Whitelisted damage lines roll prop; successes spawn mesos at monster (x±50, monster y) | Tasks 6 (roll/position logic + tests), 7 (wiring), 3 (SPAWN emit) |
| Meso amount `min(max(d/20000 × X, 1), X)` incl. d=0, d huge, X≤0 | Task 2 (`TestPickPocketMesoAmount`) |
| Non-whitelisted skills never proc | Task 2 (`TestPickPocketWhitelisted`), Task 4 (state gate test) |
| No buff REST call for non-whitelisted skill | Task 4 (`TestPickPocketResolveState_NonWhitelistedSkillMakesNoLookups`) |
| Without the buff, no emissions | Task 4 (no-buff/expired tests), Task 6 (disabled-state test) |
| Failures logged and swallowed; attack unaffected | Task 4 (buff/effect error tests), Task 6 (fetch/emit error tests); reflect semantics already pinned by task-147's `character_attack_drain_test.go` |
| `// TODO apply Pick Pocket` removed | Task 7 Step 2 |
| test -race / vet / build / bake / redis-key-guard / goroutine-guard / lint clean | Task 7 Steps 2–6 |
| Version-agnostic; fires on legacy versions too | design §1a; Task 5 (`TestOnDamageApplied_CarriesPerLineDamages`), Task 7 Step 8 |
