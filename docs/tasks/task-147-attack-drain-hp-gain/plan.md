# Attack-Side Drain HP Gain Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Attacks made with the four drain-family skills (Assassin Drain 4101005, Marauder Energy Drain 5111004, Thunder Breaker Energy Drain 15111001, Night Walker Vampire 14101006) heal the attacker per damaged monster by `min(monsterMaxHp, floor(totalDamage × X / 100), effectiveMaxHp/2)`.

**Architecture:** All changes live in one file plus tests: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`. The per-monster post-damage hook `onDamageApplied` (MP Eater's hook) is widened to carry the damage total; a pure `drainHealAmount` helper does the cap math; a `drainTryHeal` orchestrator (injected collaborators, testable) fetches the monster snapshot + lazy effective stats and emits `ChangeHP`. The existing lazy venom stats loader is renamed to `loadEffectiveStats` and shared by both consumers.

**Tech Stack:** Go, logrus, table-driven tests, project Builder pattern (`monster.NewModelBuilder`, `field.NewBuilder`, `packetmodel.NewAttackInfo`/`NewDamageInfo`).

## Global Constraints

- Skill ids referenced ONLY via `libs/atlas-constants/skill` constants: `AssassinDrainId`, `MarauderEnergyDrainId`, `ThunderBreakerStage3EnergyDrainId`, `NightWalkerStage2VampireId` — no numeric skill literals in the handler (DOM-21, PRD FR-1).
- Every drain-path failure is logged and swallowed; nothing may abort or delay damage application, the attack broadcast, or projectile consumption (PRD FR-6).
- Heal is defensively clamped to `int16` range before `ChangeHP` (PRD FR-5). No current-HP arithmetic in the handler — downstream clamping is atlas-character's job.
- Effective stats fetched at most once per attack, lazily (PRD FR-4).
- Floor semantics for the percentage: integer division after the multiply, matching Cosmic's `(int)` cast (PRD FR-2).
- Tests use the project Builder pattern; NO `*_testhelpers.go` files.
- NO `MajorVersion` / version gating in this feature. Version applicability is enforced structurally by the skill-ownership check in `processAttack` — see "Version scope" below and PRD §8.1.
- Aran Combo Drain (21100005) is explicitly NOT a drain skill here; its TODO line stays (PRD non-goal).
- Only the `// TODO increase HP from Energy Drain, Vampire, or Drain` line is removed; all adjacent TODOs untouched (PRD FR-7).
- The Go module root is `services/atlas-channel/atlas.com/channel` (module name `atlas-channel`); run all `go` commands from there unless stated otherwise.
- `go test -race ./...`, `go vet ./...`, `go build ./...` clean in atlas-channel; `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh` and `tools/lint.sh --check` clean from repo root. `go.mod` is not touched, so no `docker buildx bake` is expected (if any task somehow touches `go.mod`, bake `atlas-channel`).

## Verified reference facts (do not re-derive)

> **Re-verified 2026-07-25** after rebasing this branch onto main. Line numbers below were refreshed:
> `e15b343b1` (task-116 Gen3 processor convergence), `e0321f319` (task-171 lint/format baseline
> reformat), `be0c94338` (task-158 Shadow Stars) and the legacy-version work all moved lines in
> these files. **Every signature and builder method the plan depends on survived unchanged** — only
> the line numbers shifted.

- `ai.SkillId()` returns `uint32`, pointer receiver `*AttackInfo` (`libs/atlas-packet/model/attack_info.go:397`).
- `effect.Model.X()` returns `int16` (`services/atlas-channel/atlas.com/channel/data/skill/effect/model.go:154`).
- `monster.Model.MaxHp()` returns `uint32` (`services/atlas-channel/atlas.com/channel/monster/model.go:120`); build test monsters with `monster.NewModelBuilder(uniqueId, f, monsterId).SetMaxHp(n).Build()` (`monster/builder.go:30,59` — the builder type is unexported `*modelBuilder`).
- `effective_stats.RestModel.MaxHp` is `uint32` (`services/atlas-channel/atlas.com/channel/effective_stats/rest.go:12`).
- `character.Processor.ChangeHP(f field.Model, characterId uint32, amount int16) error` (`services/atlas-channel/atlas.com/channel/character/processor.go:43` iface, `:276` impl).
- `monster.Processor.GetById(uniqueId uint32) (Model, error)` (`services/atlas-channel/atlas.com/channel/monster/processor.go:18` iface, `:49` impl).
- `packetmodel` DamageInfo builders used by the tests: `NewDamageInfo` (`libs/atlas-packet/model/damage_info.go:13`), `Damages()` (`:90`), `SetMonsterId` (`:104`), `SetDamages` (`:114`).
- `se` in `processAttack` is only populated inside the `ai.SkillId() > 0` block (`character_attack_common.go:281-312`), so a zero-value `se` can never feed the drain branch (which also gates on `ai.SkillId() > 0`). `se` is fetched at `sk.Level()` — the character's **owned** level, not the level claimed in the packet (`character_attack_common.go:293`).
- FR-3 spot X values (verified against local WZ): Drain L1 X=16, L30 X=45; Energy Drain (both) L20 X=20; Vampire L20 X=10.
- No existing test constructs `damageInfoEntryDeps` — the hook-signature change touches only production code plus the new tests written in this plan.
- The TODO block in `processAttack` still holds exactly **24** `TODO` lines (`character_attack_common.go:403-426`), so Task 5's 24→23 grep check is unchanged. The drain TODO is now at **line 410**.
- `"math"` is already imported (`character_attack_common.go:16`) — Task 2 needs no import change.
- `monster.ReflectInfo` field names for Task 3's reflect test: `Kind`, `Percent`, `LtX`, `LtY`, `RbX`, `RbY`, `MaxDamage` (`monster/status_mirror.go:39-47`).

## Version scope (added post-rebase)

GMS legacy versions **48.1 / 61.1 / 72.1 / 79.1** landed on main after this plan was written and are
in scope. **This changes no task in this plan** — see PRD §8.1 for the full matrix and reasoning.
The two facts that matter here:

- The attack pipeline is wired for gms 48/61/72/79/83/84/87/95 + jms185 (not gms_12, not gms_92).
- **No per-version branching is needed.** `processAttack` destroys the session if the character does
  not own the cast skill (`character_attack_common.go:283-291`), so a drain skill absent from a
  version's data can never reach the drain branch. Do **not** add `MajorVersion` gates to this
  feature; if you find yourself reaching for one, re-read PRD §8.1 first.
- `attack_info.go` version gates affect head/trailer framing only — `skillId` and `damageInfo` are
  populated on every version, so `drainHealAmount`'s inputs are version-independent.

---

### Task 1: `isDrainSkill` membership helper

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` (add helper after `attackKindFromAttackType`, around line 76)
- Test (create): `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_drain_test.go`

**Interfaces:**
- Consumes: `skill3` = `github.com/Chronicle20/atlas/libs/atlas-constants/skill` (already imported in the file).
- Produces: `func isDrainSkill(id skill3.Id) bool` — used by Task 5's wiring.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_drain_test.go`:

```go
package handler

import (
	"testing"

	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

func TestIsDrainSkill(t *testing.T) {
	tests := []struct {
		name string
		id   skill3.Id
		want bool
	}{
		{"assassin drain", skill3.AssassinDrainId, true},
		{"marauder energy drain", skill3.MarauderEnergyDrainId, true},
		{"thunder breaker energy drain", skill3.ThunderBreakerStage3EnergyDrainId, true},
		{"night walker vampire", skill3.NightWalkerStage2VampireId, true},
		{"aran combo drain is NOT attack-side drain", skill3.AranStage2ComboDrainId, false},
		{"zero id", skill3.Id(0), false},
		{"adjacent id", skill3.AssassinDrainId + 1, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isDrainSkill(tc.id); got != tc.want {
				t.Errorf("isDrainSkill(%d) = %v, want %v", tc.id, got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run (from `services/atlas-channel/atlas.com/channel`):
```bash
go test ./socket/handler/ -run TestIsDrainSkill -v
```
Expected: FAIL to build with `undefined: isDrainSkill`.

- [ ] **Step 3: Write minimal implementation**

In `character_attack_common.go`, after `attackKindFromAttackType` (line 76):

```go
// isDrainSkill reports whether id is one of the four attack-side
// drain-family skills that heal the attacker from damage dealt
// (Assassin Drain, Marauder/Thunder Breaker Energy Drain, Night
// Walker Vampire). Aran Combo Drain is buff-driven and excluded.
func isDrainSkill(id skill3.Id) bool {
	switch id {
	case skill3.AssassinDrainId,
		skill3.MarauderEnergyDrainId,
		skill3.ThunderBreakerStage3EnergyDrainId,
		skill3.NightWalkerStage2VampireId:
		return true
	}
	return false
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./socket/handler/ -run TestIsDrainSkill -v
```
Expected: PASS (7 subtests).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_drain_test.go
git commit -m "feat(channel): add isDrainSkill membership helper for drain-family heal"
```

---

### Task 2: `drainHealAmount` pure cap math

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` (add helper after `mpEaterAbsorbAmount`, around line 200)
- Test (modify): `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_drain_test.go`

**Interfaces:**
- Consumes: `math` (already imported in `character_attack_common.go`).
- Produces: `func drainHealAmount(totalDamage uint32, x int16, monsterMaxHp uint32, effectiveMaxHp uint32) int16` — used by Task 4's `drainTryHeal`.

- [ ] **Step 1: Write the failing test**

Append to `character_attack_drain_test.go`:

```go
func TestDrainHealAmount(t *testing.T) {
	const big = uint32(2_000_000_000) // caps well above every raw heal below

	tests := []struct {
		name           string
		totalDamage    uint32
		x              int16
		monsterMaxHp   uint32
		effectiveMaxHp uint32
		want           int16
	}{
		// FR-3 spot values: percentage math with no cap engaged.
		{"drain L30 x=45: 1000 dmg heals 450", 1000, 45, big, big, 450},
		{"drain L1 x=16 floor: 333*16/100 = 53", 333, 16, big, big, 53},
		{"vampire L20 x=10: 5000 dmg heals 500", 5000, 10, big, big, 500},
		{"energy drain L20 x=20: 12345*20/100 = 2469", 12345, 20, big, big, 2469},
		// Caps.
		{"monster max HP caps the heal", 10000, 45, 100, big, 100},
		{"half effective max HP caps the heal", 10000, 45, big, 2000, 1000},
		{"half-cap floors odd effectiveMaxHp: 2001/2 = 1000", 10000, 45, big, 2001, 1000},
		{"tighter of the two caps wins", 10000, 45, 300, 2000, 300},
		// Zero guards.
		{"zero damage heals nothing", 0, 45, big, big, 0},
		{"x=0 heals nothing", 1000, 0, big, big, 0},
		{"negative x heals nothing", 1000, -5, big, big, 0},
		{"effectiveMaxHp=0 (stats fetch failed) heals nothing", 1000, 45, big, 0, 0},
		{"monsterMaxHp=0 heals nothing", 1000, 45, 0, big, 0},
		// Defensive int16 clamp with pathological inputs.
		{"int16 clamp on pathological damage", 4_000_000_000, 100, 4_294_967_295, 4_294_967_295, 32767},
		{"raw heal exactly at MaxInt16 passes through", 32767, 100, big, big, 32767},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := drainHealAmount(tc.totalDamage, tc.x, tc.monsterMaxHp, tc.effectiveMaxHp)
			if got != tc.want {
				t.Errorf("drainHealAmount(%d, %d, %d, %d) = %d, want %d",
					tc.totalDamage, tc.x, tc.monsterMaxHp, tc.effectiveMaxHp, got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./socket/handler/ -run TestDrainHealAmount -v
```
Expected: FAIL to build with `undefined: drainHealAmount`.

- [ ] **Step 3: Write minimal implementation**

In `character_attack_common.go`, after `mpEaterAbsorbAmount`:

```go
// drainHealAmount computes the drain-family HP gain for one damaged
// monster: floor(totalDamage * x / 100), capped by the monster's max HP
// and by half the attacker's effective (buff-inclusive) max HP, then
// defensively clamped to int16 range for ChangeHP. Returns 0 for
// non-positive x, zero damage, or zero effectiveMaxHp (fail-safe when
// the effective-stats fetch failed).
func drainHealAmount(totalDamage uint32, x int16, monsterMaxHp uint32, effectiveMaxHp uint32) int16 {
	if totalDamage == 0 || x <= 0 || effectiveMaxHp == 0 {
		return 0
	}
	heal := uint64(totalDamage) * uint64(x) / 100
	if m := uint64(monsterMaxHp); heal > m {
		heal = m
	}
	if h := uint64(effectiveMaxHp) / 2; heal > h {
		heal = h
	}
	if heal > math.MaxInt16 {
		return math.MaxInt16
	}
	return int16(heal)
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./socket/handler/ -run TestDrainHealAmount -v
```
Expected: PASS (15 subtests).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_drain_test.go
git commit -m "feat(channel): add drainHealAmount pure cap math for drain-family heal"
```

---

### Task 3: Widen `onDamageApplied` hook and rename `loadVenomStats` → `loadEffectiveStats`

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` (line numbers re-verified 2026-07-25 post-rebase):
  - `damageInfoEntryDeps` struct (lines 82-93): rename field `loadVenomStats` → `loadEffectiveStats`; widen `onDamageApplied` to `func(monsterId uint32, totalDamage uint32)`
  - `processDamageInfoEntry` (lines 100-181): two `deps.loadVenomStats()` call sites (lines 124, 171); hook invocation (lines 178-180) gains the damage sum
  - `processAttack`: the lazy `loadVenomStats` closure (lines 324-340) is renamed `loadEffectiveStats` along with its cache vars and its error log generalized; the `deps` literal (lines 342-357) is updated together with the MP Eater closure signature
- Test (modify): `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_drain_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces (relied on by Tasks 4-5):
  - `damageInfoEntryDeps.loadEffectiveStats func() effective_stats.RestModel`
  - `damageInfoEntryDeps.onDamageApplied func(monsterId uint32, totalDamage uint32)` — invoked once per non-reflected, damage-carrying `DamageInfo`, with `totalDamage` = sum of that entry's damage lines (uint64-summed, clamped to `math.MaxUint32`)
  - `loadEffectiveStats` closure in `processAttack` — lazy, once-per-attack, returns zero `effective_stats.RestModel` on fetch failure

- [ ] **Step 1: Write the failing hook-behavior tests**

Append to `character_attack_drain_test.go` (new imports on the existing import block: `io`, `"atlas-channel/monster"`, `"atlas-channel/data/skill/effect"`, `"github.com/Chronicle20/atlas/libs/atlas-constants/channel"`, `"github.com/Chronicle20/atlas/libs/atlas-constants/field"`, `_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"`, `monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"`, `"github.com/Chronicle20/atlas/libs/atlas-constants/world"`, `packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"`, `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"`, `"github.com/google/uuid"`, `"github.com/sirupsen/logrus"`; `monster.ReflectInfo` field names verified at `monster/status_mirror.go:38-47`):

```go
func discardLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

func testField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
}

// TestOnDamageApplied_ReceivesSummedDamageTotal pins the widened hook
// contract: one invocation per damage-carrying entry, carrying the sum
// of that entry's damage lines.
func TestOnDamageApplied_ReceivesSummedDamageTotal(t *testing.T) {
	ai := *packetmodel.NewAttackInfo(packetmodel.AttackTypeMelee)
	di := *packetmodel.NewDamageInfo(2).SetMonsterId(4001).SetDamages([]uint32{100, 250})

	var gotMonsterId, gotTotal uint32
	calls := 0
	deps := damageInfoEntryDeps{
		applyDamage: func(_ field.Model, _, _ uint32, _ []uint32, _ byte) error { return nil },
		onDamageApplied: func(monsterId uint32, totalDamage uint32) {
			calls++
			gotMonsterId = monsterId
			gotTotal = totalDamage
		},
	}

	processDamageInfoEntry(discardLogger(), di, ai, effect.Model{}, 1, 999, 0, 0, testField(), testTenant(t), "", deps)

	if calls != 1 {
		t.Fatalf("onDamageApplied calls = %d, want 1", calls)
	}
	if gotMonsterId != 4001 {
		t.Errorf("monsterId = %d, want 4001", gotMonsterId)
	}
	if gotTotal != 350 {
		t.Errorf("totalDamage = %d, want 350", gotTotal)
	}
}

// TestOnDamageApplied_NotCalledForZeroDamageEntry: a status-only entry
// (no damage lines) never reaches the hook.
func TestOnDamageApplied_NotCalledForZeroDamageEntry(t *testing.T) {
	ai := *packetmodel.NewAttackInfo(packetmodel.AttackTypeMelee)
	di := *packetmodel.NewDamageInfo(0).SetMonsterId(4002).SetDamages(nil)

	called := false
	deps := damageInfoEntryDeps{
		applyDamage:     func(_ field.Model, _, _ uint32, _ []uint32, _ byte) error { return nil },
		onDamageApplied: func(_ uint32, _ uint32) { called = true },
	}

	processDamageInfoEntry(discardLogger(), di, ai, effect.Model{}, 1, 999, 0, 0, testField(), testTenant(t), "", deps)

	if called {
		t.Fatalf("onDamageApplied fired for a zero-damage entry")
	}
}

// TestOnDamageApplied_NotCalledForReflectedEntry: a reflected entry deals
// no damage, so the hook must not fire (drain inherits this for free).
func TestOnDamageApplied_NotCalledForReflectedEntry(t *testing.T) {
	ai := *packetmodel.NewAttackInfo(packetmodel.AttackTypeMelee)
	di := *packetmodel.NewDamageInfo(1).SetMonsterId(4003).SetDamages([]uint32{500})
	f := testField()

	called := false
	damaged := false
	deps := damageInfoEntryDeps{
		getReflect: func(_ tenant.Model, _ uint32, _ string) (monster.ReflectInfo, bool) {
			return monster.ReflectInfo{
				Kind:      monster2.ReflectKindPhysical,
				Percent:   30,
				LtX:       -100,
				LtY:       -100,
				RbX:       100,
				RbY:       100,
				MaxDamage: 9999,
			}, true
		},
		getMonster: func(monsterId uint32) (monster.Model, error) {
			return monster.NewModelBuilder(monsterId, f, 100100).Build(), nil
		},
		applyDamage: func(_ field.Model, _, _ uint32, _ []uint32, _ byte) error {
			damaged = true
			return nil
		},
		emitReflectDamage: func(_ field.Model, _, _, _ uint32, _ uint32, _ string) error { return nil },
		onDamageApplied:   func(_ uint32, _ uint32) { called = true },
	}

	processDamageInfoEntry(discardLogger(), di, ai, effect.Model{}, 1, 999, 0, 0, f, testTenant(t), monster2.ReflectKindPhysical, deps)

	if damaged {
		t.Fatalf("applyDamage fired for a reflected entry")
	}
	if called {
		t.Fatalf("onDamageApplied fired for a reflected entry")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./socket/handler/ -run 'TestOnDamageApplied' -v
```
Expected: FAIL to build — `onDamageApplied` in the deps literal has the wrong signature (`func(uint32, uint32)` vs. current `func(uint32)`).

- [ ] **Step 3: Widen the hook and rename the loader in production code**

In `character_attack_common.go`:

3a. `damageInfoEntryDeps` (lines 81-92) — rename the field and widen the hook:

```go
type damageInfoEntryDeps struct {
	getReflect        func(t tenant.Model, monsterId uint32, kind string) (monster.ReflectInfo, bool)
	getMonster        func(monsterId uint32) (monster.Model, error)
	applyDamage       func(f field.Model, monsterId, characterId uint32, damages []uint32, attackType byte) error
	emitReflectDamage func(f field.Model, uniqueId, templateId, characterId uint32, reflectDamage uint32, reflectType string) error
	applyStatus       func(f field.Model, monsterId, characterId, skillId, skillLevel uint32, statuses map[string]int32, duration uint32) error
	// loadEffectiveStats lazily fetches the caster's buff-inclusive
	// effective stats, at most once per attack. Consumed by the venom
	// DPT snapshot and by the drain-family heal cap.
	loadEffectiveStats func() effective_stats.RestModel
	// onDamageApplied is invoked once per non-reflected DamageInfo after
	// damage and status apply, with the summed damage of that entry.
	// Optional; nil-safe. Used by passives that fire per damaged
	// monster (e.g., MP Eater, drain-family heals).
	onDamageApplied func(monsterId uint32, totalDamage uint32)
}
```

3b. In `processDamageInfoEntry`, replace both `deps.loadVenomStats()` calls (lines 123 and 170) with `deps.loadEffectiveStats()`.

3c. Replace the hook invocation (lines 177-179) with:

```go
	if deps.onDamageApplied != nil {
		var total uint64
		for _, d := range damages {
			total += uint64(d)
		}
		if total > math.MaxUint32 {
			total = math.MaxUint32
		}
		deps.onDamageApplied(di.MonsterId(), uint32(total))
	}
```

3d. In `processAttack` (lines 324-357), rename the lazy loader and its cache, generalize the log line, and update the deps literal (MP Eater ignores the new argument for now — drain wiring is Task 5):

```go
					// Lazy effective-stats fetch: needed when a damage entry
					// produces a VENOM apply and by drain-family heals.
					// Cached for the duration of one attack.
					var effectiveStats effective_stats.RestModel
					effectiveStatsLoaded := false
					loadEffectiveStats := func() effective_stats.RestModel {
						if effectiveStatsLoaded {
							return effectiveStats
						}
						effectiveStatsLoaded = true
						stats, sErr := effective_stats.NewProcessor(l, ctx).GetByCharacterId(s.WorldId(), s.ChannelId(), s.CharacterId())
						if sErr != nil {
							l.WithError(sErr).Errorf("Unable to fetch effective stats for character [%d]; venom DPT and drain heal will fall back to zero.", s.CharacterId())
							return effective_stats.RestModel{}
						}
						effectiveStats = stats
						return effectiveStats
					}

					deps := damageInfoEntryDeps{
						getReflect:         mirror.GetReflect,
						getMonster:         mp.GetById,
						applyDamage:        mp.Damage,
						emitReflectDamage:  mp.EmitDamageReflected,
						applyStatus:        mp.ApplyStatus,
						loadEffectiveStats: loadEffectiveStats,
						// MP Eater proc: per-monster, after status apply,
						// magic attacks only. Failures are swallowed so the
						// rest of the attack pipeline is unaffected.
						onDamageApplied: func(monsterId uint32, totalDamage uint32) {
							if ai.AttackType() == packetmodel.AttackTypeMagic && ai.SkillId() > 0 {
								mpEaterTryProc(l, ctx, mp, c, monsterId, s.Field(), s.CharacterId())
							}
						},
					}
```

- [ ] **Step 4: Run the package tests to verify everything passes**

```bash
go build ./... && go test ./socket/handler/ -v -run 'TestOnDamageApplied|TestReflectFlow|TestMpEater'
```
Expected: build clean; the 3 new tests plus all existing reflect/MP Eater tests PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_drain_test.go
git commit -m "refactor(channel): widen onDamageApplied hook with damage total; generalize effective-stats loader"
```

---

### Task 4: `drainTryHeal` orchestrator

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` (add after `mpEaterTryProc`, around line 264)
- Test (modify): `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_drain_test.go`

**Interfaces:**
- Consumes: `drainHealAmount` (Task 2), `monster.Model`, `effective_stats.RestModel`, `field.Model`.
- Produces (relied on by Task 5):

```go
func drainTryHeal(
	l logrus.FieldLogger,
	getMonster func(monsterId uint32) (monster.Model, error),
	changeHP func(f field.Model, characterId uint32, amount int16) error,
	loadEffectiveStats func() effective_stats.RestModel,
	x int16,
	skillId uint32,
	monsterId uint32,
	totalDamage uint32,
	f field.Model,
	characterId uint32,
)
```

Production wiring passes `mp.GetById` and `cp.ChangeHP` (both signatures verified above).

- [ ] **Step 1: Write the failing flow tests**

Append to `character_attack_drain_test.go` (new imports: `"errors"`, `"atlas-channel/effective_stats"`):

```go
type changeHPCall struct {
	characterId uint32
	amount      int16
}

// TestDrainTryHeal_EmitsCappedHeal: happy path — heal computed from the
// damage total and X, capped, emitted with a positive amount.
func TestDrainTryHeal_EmitsCappedHeal(t *testing.T) {
	f := testField()
	var calls []changeHPCall

	drainTryHeal(
		discardLogger(),
		func(monsterId uint32) (monster.Model, error) {
			return monster.NewModelBuilder(monsterId, f, 100100).SetMaxHp(6000).Build(), nil
		},
		func(_ field.Model, characterId uint32, amount int16) error {
			calls = append(calls, changeHPCall{characterId, amount})
			return nil
		},
		func() effective_stats.RestModel { return effective_stats.RestModel{MaxHp: 3000} },
		45,   // x (Drain L30)
		4101005,
		7001, // monsterId
		1000, // totalDamage -> raw heal 450, under both caps (6000, 1500)
		f,
		999,
	)

	if len(calls) != 1 {
		t.Fatalf("ChangeHP calls = %d, want 1", len(calls))
	}
	if calls[0].characterId != 999 || calls[0].amount != 450 {
		t.Errorf("ChangeHP(%d, %d), want (999, 450)", calls[0].characterId, calls[0].amount)
	}
}

// TestDrainTryHeal_MonsterFetchError_SkipsHeal: FR-4 — snapshot failure
// skips the heal for that monster, never errors out.
func TestDrainTryHeal_MonsterFetchError_SkipsHeal(t *testing.T) {
	called := false
	drainTryHeal(
		discardLogger(),
		func(_ uint32) (monster.Model, error) { return monster.Model{}, errors.New("gone") },
		func(_ field.Model, _ uint32, _ int16) error { called = true; return nil },
		func() effective_stats.RestModel { return effective_stats.RestModel{MaxHp: 3000} },
		45, 4101005, 7002, 1000, testField(), 999,
	)
	if called {
		t.Fatalf("ChangeHP fired despite monster fetch error")
	}
}

// TestDrainTryHeal_ZeroEffectiveStats_SkipsHeal: FR-4 fail-safe — a failed
// effective-stats fetch (zero RestModel) yields no heal, not an uncapped one.
func TestDrainTryHeal_ZeroEffectiveStats_SkipsHeal(t *testing.T) {
	f := testField()
	called := false
	drainTryHeal(
		discardLogger(),
		func(monsterId uint32) (monster.Model, error) {
			return monster.NewModelBuilder(monsterId, f, 100100).SetMaxHp(6000).Build(), nil
		},
		func(_ field.Model, _ uint32, _ int16) error { called = true; return nil },
		func() effective_stats.RestModel { return effective_stats.RestModel{} },
		45, 4101005, 7003, 1000, f, 999,
	)
	if called {
		t.Fatalf("ChangeHP fired despite zero effective stats")
	}
}

// TestDrainTryHeal_EmitErrorSwallowed: FR-6 — a ChangeHP emit failure is
// logged and swallowed (no panic, no propagation).
func TestDrainTryHeal_EmitErrorSwallowed(t *testing.T) {
	f := testField()
	drainTryHeal(
		discardLogger(),
		func(monsterId uint32) (monster.Model, error) {
			return monster.NewModelBuilder(monsterId, f, 100100).SetMaxHp(6000).Build(), nil
		},
		func(_ field.Model, _ uint32, _ int16) error { return errors.New("kafka down") },
		func() effective_stats.RestModel { return effective_stats.RestModel{MaxHp: 3000} },
		45, 4101005, 7004, 1000, f, 999,
	)
	// Reaching here without panic is the assertion.
}

// TestDrainTryHeal_PerMonsterCaps: two monsters, individually capped —
// multi-target Vampire semantics (one call per damaged monster).
func TestDrainTryHeal_PerMonsterCaps(t *testing.T) {
	f := testField()
	maxHpByMonster := map[uint32]uint32{8001: 6000, 8002: 200}
	var calls []changeHPCall

	for _, monsterId := range []uint32{8001, 8002} {
		drainTryHeal(
			discardLogger(),
			func(id uint32) (monster.Model, error) {
				return monster.NewModelBuilder(id, f, 100100).SetMaxHp(maxHpByMonster[id]).Build(), nil
			},
			func(_ field.Model, characterId uint32, amount int16) error {
				calls = append(calls, changeHPCall{characterId, amount})
				return nil
			},
			func() effective_stats.RestModel { return effective_stats.RestModel{MaxHp: 3000} },
			10, // x (Vampire L20)
			14101006,
			monsterId,
			5000, // raw heal 500; monster 8002 caps it at 200
			f,
			999,
		)
	}

	if len(calls) != 2 {
		t.Fatalf("ChangeHP calls = %d, want 2", len(calls))
	}
	if calls[0].amount != 500 {
		t.Errorf("monster 8001 heal = %d, want 500 (under caps)", calls[0].amount)
	}
	if calls[1].amount != 200 {
		t.Errorf("monster 8002 heal = %d, want 200 (monster max HP cap)", calls[1].amount)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./socket/handler/ -run TestDrainTryHeal -v
```
Expected: FAIL to build with `undefined: drainTryHeal`.

- [ ] **Step 3: Write the implementation**

In `character_attack_common.go`, after `mpEaterTryProc`:

```go
// drainTryHeal computes and emits the drain-family heal for one damaged
// monster: floor(totalDamage * x / 100), capped by the monster's max HP
// and half the caster's effective max HP. Called once per damaged
// monster after damage apply. All collaborators are injected so flow
// tests can drive every branch; production passes mp.GetById and
// cp.ChangeHP. Errors are logged and swallowed — never abort the
// surrounding attack pipeline.
func drainTryHeal(
	l logrus.FieldLogger,
	getMonster func(monsterId uint32) (monster.Model, error),
	changeHP func(f field.Model, characterId uint32, amount int16) error,
	loadEffectiveStats func() effective_stats.RestModel,
	x int16,
	skillId uint32,
	monsterId uint32,
	totalDamage uint32,
	f field.Model,
	characterId uint32,
) {
	mon, err := getMonster(monsterId)
	if err != nil {
		l.WithError(err).Debugf("Drain heal: monster [%d] snapshot fetch failed; skipping heal for caster [%d].", monsterId, characterId)
		return
	}

	stats := loadEffectiveStats()
	heal := drainHealAmount(totalDamage, x, mon.MaxHp(), stats.MaxHp)
	if heal <= 0 {
		return
	}

	l.Debugf("Drain heal: caster=[%d] skill=[%d] monster=[%d] damage=[%d] x=[%d] heal=[%d].",
		characterId, skillId, monsterId, totalDamage, x, heal)

	if err := changeHP(f, characterId, heal); err != nil {
		l.WithError(err).Errorf("Drain heal: CHANGE_HP emit failed for character [%d] (skill [%d], monster [%d]).", characterId, skillId, monsterId)
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./socket/handler/ -run TestDrainTryHeal -v
```
Expected: PASS (5 tests).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_drain_test.go
git commit -m "feat(channel): add drainTryHeal orchestrator for drain-family HP gain"
```

---

### Task 5: Wire drain heal into `processAttack`, remove the TODO, full verification

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`:
  - the `onDamageApplied` closure in the `deps` literal (as rewritten in Task 3, Step 3d)
  - delete the line `// TODO increase HP from Energy Drain, Vampire, or Drain` (line 410 pre-change; renumbered by earlier tasks — locate it by content)

**Interfaces:**
- Consumes: `isDrainSkill` (Task 1), `drainTryHeal` (Task 4), `loadEffectiveStats` closure (Task 3), plus in-scope `processAttack` locals: `ai`, `se`, `mp`, `cp`, `s`, `l`.
- Produces: the complete feature; nothing downstream.

- [ ] **Step 1: Wire the drain branch into the hook closure**

In `processAttack`'s `deps` literal, replace the `onDamageApplied` closure with:

```go
						// MP Eater proc: per-monster, after status apply,
						// magic attacks only. Drain-family heal: per-monster,
						// skill-id gated (no attack-type gate — the four
						// skills span melee/ranged/energy). Failures are
						// swallowed so the rest of the attack pipeline is
						// unaffected.
						onDamageApplied: func(monsterId uint32, totalDamage uint32) {
							if ai.AttackType() == packetmodel.AttackTypeMagic && ai.SkillId() > 0 {
								mpEaterTryProc(l, ctx, mp, c, monsterId, s.Field(), s.CharacterId())
							}
							if ai.SkillId() > 0 && isDrainSkill(skill3.Id(ai.SkillId())) {
								drainTryHeal(l, mp.GetById, cp.ChangeHP, loadEffectiveStats, se.X(), ai.SkillId(), monsterId, totalDamage, s.Field(), s.CharacterId())
							}
						},
```

- [ ] **Step 2: Remove the drain TODO line**

Delete exactly this line from the TODO block near the end of `processAttack` (leave every other TODO, including `// TODO Combo Drain`):

```go
					// TODO increase HP from Energy Drain, Vampire, or Drain
```

Verify with:
```bash
grep -n "TODO increase HP from Energy Drain" services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go
```
Expected: no output.
```bash
grep -c "TODO" services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go
```
Expected: exactly one fewer than before the edit (the block had 24 TODO lines; 23 remain).

- [ ] **Step 3: Full module verification**

From `services/atlas-channel/atlas.com/channel`:
```bash
go build ./... && go vet ./... && go test -race ./...
```
Expected: all clean, all tests PASS.

From the repo root (worktree root) — the guard set grew after this plan was first written
(`tools/goroutine-guard.sh` from task-115, `tools/lint.sh` from task-171):
```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/lint.sh --check
```
Expected: all clean (exit 0). Do NOT prefix with a global `GOWORK=off`.

`tools/lint.sh --check` is the one most likely to fail on first run: it enforces gofumpt +
goimports tree-wide. Run `tools/lint.sh` (no flags) to auto-fix in place before committing.
This branch was created before task-171's baseline reformat landed, so run the fix mode once
after the first code edit rather than debugging individual format diffs.

Confirm `go.mod` untouched (no bake needed):
```bash
git diff --stat -- services/atlas-channel/atlas.com/channel/go.mod
```
Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go
git commit -m "feat(channel): heal attacker from drain-family skills per damaged monster

Wires Assassin Drain, Marauder/Thunder Breaker Energy Drain, and Night
Walker Vampire into the per-monster post-damage hook. Heal per monster:
min(monsterMaxHp, floor(totalDamage * X / 100), effectiveMaxHp/2),
emitted via the existing ChangeHP command path. Removes the drain TODO."
```

---

## Acceptance criteria traceability (PRD §10)

| Criterion | Covered by |
|---|---|
| Four skill ids heal; others don't | Task 1 (membership + tests), Task 5 (gate wiring) |
| Heal formula `min(monsterMaxHp, floor(dmg×X/100), effMaxHp/2)`, X per FR-3 | Task 2 (math + FR-3 spot-value tests) |
| Multi-target: per-monster heal, individually capped | Task 4 (`TestDrainTryHeal_PerMonsterCaps`), Task 3 (hook fires per entry) |
| Killed monster still heals | Design D8: registry snapshot survives the async damage emit; Task 4 fetch-then-heal order |
| Failures produce no heal, never abort attack | Task 4 tests (fetch error, zero stats, emit error swallowed), Task 3 (hook after damage apply, before broadcast) |
| int16 defensive clamp | Task 2 (`int16 clamp on pathological damage`) |
| Table-driven unit tests for math/caps/zero/floor | Task 2 |
| TODO removed, adjacent TODOs untouched | Task 5 Step 2 (grep-verified) |
| No per-version branching introduced (PRD §8.1) | Global Constraints; enforced by the ownership gate, not by code in this plan |
| `go test -race` / `go vet` / `go build` / redis-key-guard / goroutine-guard / lint.sh clean | Task 5 Step 3 |
