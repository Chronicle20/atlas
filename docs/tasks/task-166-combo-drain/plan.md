# Combo Drain (Aran) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

Revised 2026-08-07: rebased onto `main` (254 commits); every line reference
re-derived; architecture changed from design Approach B to Approach C; Task 4
(version-scope verification) added.

Revised 2026-08-13: merged `main` into the task branch; every line reference
re-derived again (see the table below). No task, step, code block, or test
case changed — the merge added post-damage effects around the TODO block
(task-216 Energy Charge, task-217 Aran Combo Counter) but none of them reads
buffs, so Approach C and the one-read ceiling stand exactly as written
(design §2, PRD §1.2). Re-confirmed against the merged tree: `buff.NewBuff` is
still 7-arg, `stat.NewStat` unchanged, `buffWithStat`/`expiredBuffWithStat`/
`testField` still the taken names, `ProjectileProcessor.Plan` still has exactly
one call site and no test caller, and the per-version applicability tables
(Task 4 Step 3) still reproduce byte-for-byte.

**Goal:** Make the `COMBO_DRAIN` buff (Aran skill 21100005) actually heal the
attacker `x`% of each accepted attack's total damage, replacing the
`// TODO Combo Drain` in atlas-channel's attack pipeline — correct on every
supported client version that ships the skill, with no version branch.

**Architecture:** Design Approach C — `processAttack` builds a per-attack
memoized `loadBuffs` closure (mirroring the neighbouring `loadEffectiveStats`),
hands it to the two existing buff consumers (projectile gate, Pick Pocket) in
place of their own processor, and a new `comboDrainTryProc` calls it
post-damage, emitting at most one `ChangeHP` Kafka command through the existing
`character.Processor`. Pure helpers (`buffStatAmount`, `attackTotalDamage`,
`comboDrainHealAmount`) are pinned by table tests, mirroring the
`drainHealAmount` / MP Eater / `computeReflect` style already in the package.

**Tech Stack:** Go, atlas-channel service only. Existing `buff.Processor` REST
read, existing `ChangeHP` Kafka command. No new endpoints, topics, packets,
templates, or schema — on any of the eleven supported versions.

## Global Constraints

- Single service touched: `services/atlas-channel/atlas.com/channel` (design §1).
- **No version branch.** The diff must contain no `MajorVersion`,
  `MajorAtLeast`, `IsRegion`, or version-keyed literal, and must modify no
  socket-config template (PRD FR-5, design §6.1). Availability is data-driven:
  versions without Aran cannot produce the statup.
- At most one buff REST read per attack, and only when a consumer needs one —
  the memoized loader is the mechanism; the two existing consumers keep their
  gate-before-fetch behavior (PRD NFR, design §3 Approach C).
- Buff-service failure degrades to "no heal" with a logged warning; the attack
  pipeline is never aborted, and the existing consumers' degraded postures are
  unchanged (FR-1, FR-4, design §4.1).
- Heal is once per accepted attack, from the plain total over all monsters and
  hit lines — no Cosmic per-monster running-total (PRD §1, FR-2).
- `heal = totalDamage * percent / 100`, integer arithmetic, clamped to
  `math.MaxInt16` before narrowing to `int16`; no emission when `heal <= 0`
  (FR-2).
- Buff-only gate: no job, skill-ownership, version, or attack-type check
  (FR-3, FR-5, design Approach D rejection).
- Gate stat: `charconst.TemporaryStatTypeComboDrain` from
  `libs/atlas-constants/character` (`"COMBO_DRAIN"`); never a string literal.
- Tests use production constructors (`buff.NewBuff` — **seven** args now,
  `stat.NewStat`, `packetmodel.NewAttackInfo`/`NewDamageInfo` builders) — no
  `*_testhelpers.go` (project rule).
- Keep the diff to the TODO block minimal: replace exactly the one
  `// TODO Combo Drain` line in place with the call (design §9 — sibling tasks
  edit neighbouring lines in their own worktrees).
- Verification gates (design §8) are Task 5; do not claim done before they run.
- All repo paths below are relative to the worktree root
  (`.worktrees/task-166-combo-drain`). The Go module root is
  `services/atlas-channel/atlas.com/channel` — run go commands from there.

### Line references (re-derived against the `main` merge `f1b4e4046`)

| What | Where |
|---|---|
| `// TODO Combo Drain` | `character_attack_common.go:1117` |
| `cp := character.NewProcessor(l, ctx)` | `character_attack_common.go:777` |
| `pp := NewProjectileProcessor(l, ctx)` | `character_attack_common.go:888` |
| `pp.Plan(c, ai, se)` call site | `character_attack_common.go:889` |
| `loadEffectiveStats` memoization idiom to mirror | `character_attack_common.go:898-912` |
| `pickPocketResolveState(...)` call site | `character_attack_common.go:919-925` |
| `pickPocketResolveState` definition | `character_attack_common.go:276-282` |
| `ProjectileProcessor` interface | `character_attack_projectile.go:45-48` |
| `ProjectileProcessorImpl` struct + ctor | `character_attack_projectile.go:50-64` |
| `Plan` method signature | `character_attack_projectile.go:66` |
| internal buff fetch to delete | `character_attack_projectile.go:99-107` |
| `hasBuff` (matching rules to mirror) | `character_attack_projectile.go:199` |
| `buffWithStat` / `expiredBuffWithStat` (names taken) | `character_attack_projectile_test.go:58,63` |
| `testField` (reuse as-is) | `mystic_door_enter_test.go:30` |
| `buff.NewBuff` (7 args) | `character/buff/model.go:63` |
| `stat.NewStat` | `character/buff/stat/model.go:16` |
| `docs/TODO.md` Combo Drain item | `docs/TODO.md:151` |

Re-confirm each with `grep` before editing — sibling worktrees land in this
file frequently and these numbers drift.

---

### Task 1: Pure helpers — `buffStatAmount`, `attackTotalDamage`, `comboDrainHealAmount`

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo_drain.go`
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo_drain_test.go`

**Interfaces:**
- Consumes: `buff.Model` (`atlas-channel/character/buff`: `Expired() bool`,
  `Changes() []stat.Model`), `stat.Model` (`Type() string`, `Amount() int32`),
  `packetmodel.AttackInfo` (`DamageInfo() []DamageInfo`), `packetmodel.DamageInfo`
  (`Damages() []uint32`), `charconst.TemporaryStatType`.
- Produces (Tasks 2 and 3 rely on these exact signatures):
  - `func buffStatAmount(buffs []buff.Model, statType charconst.TemporaryStatType) (int32, bool)`
  - `func attackTotalDamage(ai packetmodel.AttackInfo) uint64`
  - `func comboDrainHealAmount(totalDamage uint64, percent int32) int16`

Names already taken in package `handler` tests: `buffWithStat`/`expiredBuffWithStat`
(fixed amount 1) and `testField`. The new test file defines amount-parameterized
buff helpers under fresh names and reuses `testField` as-is.

- [ ] **Step 1: Write the failing tests**

Create `character_attack_combo_drain_test.go`:

```go
package handler

import (
	"math"
	"testing"
	"time"

	"atlas-channel/character/buff"
	"atlas-channel/character/buff/stat"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// comboDrainBuffWithAmount builds a live buff carrying a single COMBO_DRAIN
// stat change of the given amount. buff.NewBuff takes seven arguments — the
// trailing bool is noExpiry.
func comboDrainBuffWithAmount(amount int32) buff.Model {
	return buff.NewBuff(0, 1, 0,
		[]stat.Model{stat.NewStat(string(charconst.TemporaryStatTypeComboDrain), amount)},
		time.Now(), time.Now().Add(time.Hour), false)
}

// expiredComboDrainBuffWithAmount builds an already-expired COMBO_DRAIN buff.
func expiredComboDrainBuffWithAmount(amount int32) buff.Model {
	past := time.Now().Add(-time.Hour)
	return buff.NewBuff(0, 1, 0,
		[]stat.Model{stat.NewStat(string(charconst.TemporaryStatTypeComboDrain), amount)},
		past, past, false)
}

// attackWithDamages builds an AttackInfo of the given type with one
// DamageInfo per damages slice (monster ids 100, 101, ...).
func attackWithDamages(at packetmodel.AttackType, monsterDamages ...[]uint32) packetmodel.AttackInfo {
	ai := packetmodel.NewAttackInfo(at)
	for i, damages := range monsterDamages {
		di := packetmodel.NewDamageInfo(byte(len(damages))).
			SetMonsterId(uint32(100 + i)).
			SetDamages(damages)
		ai = ai.AddDamageInfo(*di)
	}
	return *ai
}

func TestBuffStatAmount(t *testing.T) {
	otherStat := buff.NewBuff(0, 1, 0,
		[]stat.Model{stat.NewStat(string(charconst.TemporaryStatTypeSoulArrow), 1)},
		time.Now(), time.Now().Add(time.Hour), false)
	mixedStats := buff.NewBuff(0, 1, 0,
		[]stat.Model{
			stat.NewStat(string(charconst.TemporaryStatTypeSoulArrow), 1),
			stat.NewStat(string(charconst.TemporaryStatTypeComboDrain), 4),
		},
		time.Now(), time.Now().Add(time.Hour), false)

	tests := []struct {
		name       string
		buffs      []buff.Model
		wantAmount int32
		wantOk     bool
	}{
		{"present", []buff.Model{comboDrainBuffWithAmount(5)}, 5, true},
		{"absent - other stat only", []buff.Model{otherStat}, 0, false},
		{"nil slice", nil, 0, false},
		{"expired buff skipped", []buff.Model{expiredComboDrainBuffWithAmount(5)}, 0, false},
		{"expired then live - live wins", []buff.Model{expiredComboDrainBuffWithAmount(9), comboDrainBuffWithAmount(3)}, 3, true},
		{"first live match wins", []buff.Model{comboDrainBuffWithAmount(2), comboDrainBuffWithAmount(7)}, 2, true},
		{"matching stat alongside other stats", []buff.Model{mixedStats}, 4, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			amount, ok := buffStatAmount(tc.buffs, charconst.TemporaryStatTypeComboDrain)
			if ok != tc.wantOk || amount != tc.wantAmount {
				t.Fatalf("buffStatAmount = (%d, %v), want (%d, %v)", amount, ok, tc.wantAmount, tc.wantOk)
			}
		})
	}
}

func TestAttackTotalDamage(t *testing.T) {
	tests := []struct {
		name string
		ai   packetmodel.AttackInfo
		want uint64
	}{
		{"single monster single line", attackWithDamages(packetmodel.AttackTypeMelee, []uint32{1000}), 1000},
		{"multi monster multi line", attackWithDamages(packetmodel.AttackTypeMelee, []uint32{1000, 2000}, []uint32{3000}), 6000},
		{"no damage entries", attackWithDamages(packetmodel.AttackTypeMelee), 0},
		{"large lines sum in uint64", attackWithDamages(packetmodel.AttackTypeMelee, []uint32{math.MaxUint32, math.MaxUint32}), 2 * uint64(math.MaxUint32)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := attackTotalDamage(tc.ai); got != tc.want {
				t.Fatalf("attackTotalDamage = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestComboDrainHealAmount(t *testing.T) {
	tests := []struct {
		name        string
		totalDamage uint64
		percent     int32
		want        int16
	}{
		{"nominal", 1000, 5, 50},
		{"integer truncation", 999, 3, 29},
		{"zero damage", 0, 5, 0},
		{"zero percent", 1000, 0, 0},
		{"negative percent", 1000, -5, 0},
		{"sub-1 result truncates to zero", 99, 1, 0},
		{"exact MaxInt16 unclamped", 3276700, 1, math.MaxInt16},
		{"one over boundary clamps", 3276800, 1, math.MaxInt16},
		{"huge total saturates without overflow", 15 * 15 * uint64(math.MaxUint32), 2147483647, math.MaxInt16},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := comboDrainHealAmount(tc.totalDamage, tc.percent); got != tc.want {
				t.Fatalf("comboDrainHealAmount(%d, %d) = %d, want %d", tc.totalDamage, tc.percent, got, tc.want)
			}
		})
	}
}
```

Before running: confirm `buff.NewBuff`'s arity and `packetmodel.NewDamageInfo`'s
builder methods against source — both changed since this plan was first written.

- [ ] **Step 2: Run the tests to verify they fail**

From `services/atlas-channel/atlas.com/channel`:

```bash
go test ./socket/handler/ -run 'TestBuffStatAmount|TestAttackTotalDamage|TestComboDrainHealAmount' -v
```

Expected: compilation FAILURE — `undefined: buffStatAmount`,
`undefined: attackTotalDamage`, `undefined: comboDrainHealAmount`.

- [ ] **Step 3: Write the implementation**

Create `character_attack_combo_drain.go`:

```go
package handler

import (
	"atlas-channel/character/buff"
	"math"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// buffStatAmount returns the Amount of the first stat change of statType
// carried by a non-expired buff, mirroring hasBuff's matching rules
// (character_attack_projectile.go).
func buffStatAmount(buffs []buff.Model, statType charconst.TemporaryStatType) (int32, bool) {
	for _, b := range buffs {
		if b.Expired() {
			continue
		}
		for _, c := range b.Changes() {
			if c.Type() == string(statType) {
				return c.Amount(), true
			}
		}
	}
	return 0, false
}

// attackTotalDamage sums every damage line across every DamageInfo entry.
// uint64 so a full multi-target attack (15 targets x 15 lines x MaxUint32)
// cannot overflow the sum.
func attackTotalDamage(ai packetmodel.AttackInfo) uint64 {
	total := uint64(0)
	for _, di := range ai.DamageInfo() {
		for _, d := range di.Damages() {
			total += uint64(d)
		}
	}
	return total
}

// comboDrainHealAmount computes totalDamage * percent / 100 in integer
// arithmetic, returning 0 when percent <= 0 or totalDamage == 0, and
// clamping to math.MaxInt16 before narrowing. For any percent >= 1,
// totalDamage >= MaxInt16*100 already guarantees the clamp, so saturate
// early — below that bound totalDamage*percent fits uint64 for any int32
// percent.
func comboDrainHealAmount(totalDamage uint64, percent int32) int16 {
	if percent <= 0 || totalDamage == 0 {
		return 0
	}
	if totalDamage >= uint64(math.MaxInt16)*100 {
		return math.MaxInt16
	}
	heal := totalDamage * uint64(percent) / 100
	if heal > uint64(math.MaxInt16) {
		return math.MaxInt16
	}
	return int16(heal)
}
```

- [ ] **Step 4: Run the tests to verify they pass**

```bash
go test ./socket/handler/ -run 'TestBuffStatAmount|TestAttackTotalDamage|TestComboDrainHealAmount' -v
```

Expected: PASS (all subtests).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo_drain.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo_drain_test.go
git commit -m "feat(channel): combo drain pure helpers (task-166)"
```

---

### Task 2: `comboDrainTryProc` orchestrator

**Files:**
- Modify: `character_attack_combo_drain.go` (append function + imports)
- Test: `character_attack_combo_drain_test.go` (append tests)

**Interfaces:**
- Consumes: Task 1's three helpers; `field.Model`
  (`libs/atlas-constants/field`); `logrus.FieldLogger`.
- Produces (Task 3's call site relies on this exact signature — `getBuffs`
  matches `buff.Processor.GetByCharacterId` and therefore the Task-3 loader;
  `changeHP` matches `character.Processor.ChangeHP`):
  - `func comboDrainTryProc(l logrus.FieldLogger, getBuffs func(characterId uint32) ([]buff.Model, error), changeHP func(f field.Model, characterId uint32, amount int16) error, f field.Model, characterId uint32, ai packetmodel.AttackInfo)`

The proc takes the **loader**, not a slice, so the "at most one read" property
lives in one place and is directly testable with a counting fake (PRD §10).

- [ ] **Step 1: Write the failing tests**

Append to `character_attack_combo_drain_test.go`; add `"errors"`,
`"github.com/Chronicle20/atlas/libs/atlas-constants/field"` and
`"github.com/sirupsen/logrus"` to its import block:

```go
// recordingChangeHP captures every emission comboDrainTryProc makes and can
// simulate a downstream failure.
type recordingChangeHP struct {
	calls []int16
	err   error
}

func (r *recordingChangeHP) fn(_ field.Model, _ uint32, amount int16) error {
	r.calls = append(r.calls, amount)
	return r.err
}

// countingBuffs serves a fixed buff slice (or error) and counts invocations.
type countingBuffs struct {
	buffs []buff.Model
	err   error
	calls int
}

func (c *countingBuffs) fn(_ uint32) ([]buff.Model, error) {
	c.calls++
	return c.buffs, c.err
}

func TestComboDrainTryProc(t *testing.T) {
	l := logrus.New()
	f := testField(100000000)

	tests := []struct {
		name      string
		buffs     []buff.Model
		buffErr   error
		ai        packetmodel.AttackInfo
		changeErr error
		wantCalls []int16
	}{
		{
			name:      "buff present single monster",
			buffs:     []buff.Model{comboDrainBuffWithAmount(5)},
			ai:        attackWithDamages(packetmodel.AttackTypeMelee, []uint32{1000}),
			wantCalls: []int16{50},
		},
		{
			// Pins the anti-Cosmic-quirk AC: one heal from the plain total
			// (6000 * 10 / 100 = 600), never per-monster running totals.
			name:      "buff present multi monster multi line - one call on plain total",
			buffs:     []buff.Model{comboDrainBuffWithAmount(10)},
			ai:        attackWithDamages(packetmodel.AttackTypeMelee, []uint32{1000, 2000}, []uint32{3000}),
			wantCalls: []int16{600},
		},
		{
			name:      "buff absent",
			buffs:     []buff.Model{},
			ai:        attackWithDamages(packetmodel.AttackTypeMelee, []uint32{1000}),
			wantCalls: nil,
		},
		{
			name:      "buff fetch error",
			buffErr:   errors.New("buffs down"),
			ai:        attackWithDamages(packetmodel.AttackTypeMelee, []uint32{1000}),
			wantCalls: nil,
		},
		{
			name:      "expired buff only",
			buffs:     []buff.Model{expiredComboDrainBuffWithAmount(5)},
			ai:        attackWithDamages(packetmodel.AttackTypeMelee, []uint32{1000}),
			wantCalls: nil,
		},
		{
			name:      "zero total damage",
			buffs:     []buff.Model{comboDrainBuffWithAmount(5)},
			ai:        attackWithDamages(packetmodel.AttackTypeMelee, []uint32{0}),
			wantCalls: nil,
		},
		{
			name:      "heal truncates to zero",
			buffs:     []buff.Model{comboDrainBuffWithAmount(1)},
			ai:        attackWithDamages(packetmodel.AttackTypeMelee, []uint32{99}),
			wantCalls: nil,
		},
		{
			name:      "changeHP error swallowed - no panic no retry",
			buffs:     []buff.Model{comboDrainBuffWithAmount(5)},
			ai:        attackWithDamages(packetmodel.AttackTypeMelee, []uint32{1000}),
			changeErr: errors.New("kafka down"),
			wantCalls: []int16{50},
		},
		// Attack-type blindness (melee/ranged/magic/energy AC): the proc has
		// no type filter by construction; these pin that none creeps in.
		// Energy is the touch handler's type (character_attack_touch.go).
		{
			name:      "ranged attack heals",
			buffs:     []buff.Model{comboDrainBuffWithAmount(5)},
			ai:        attackWithDamages(packetmodel.AttackTypeRanged, []uint32{1000}),
			wantCalls: []int16{50},
		},
		{
			name:      "magic attack heals",
			buffs:     []buff.Model{comboDrainBuffWithAmount(5)},
			ai:        attackWithDamages(packetmodel.AttackTypeMagic, []uint32{1000}),
			wantCalls: []int16{50},
		},
		{
			name:      "energy (touch) attack heals",
			buffs:     []buff.Model{comboDrainBuffWithAmount(5)},
			ai:        attackWithDamages(packetmodel.AttackTypeEnergy, []uint32{1000}),
			wantCalls: []int16{50},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := &recordingChangeHP{err: tc.changeErr}
			cb := &countingBuffs{buffs: tc.buffs, err: tc.buffErr}
			comboDrainTryProc(l, cb.fn, rec.fn, f, 42, tc.ai)
			if cb.calls > 1 {
				t.Fatalf("getBuffs called %d times, want at most 1", cb.calls)
			}
			if len(rec.calls) != len(tc.wantCalls) {
				t.Fatalf("changeHP called %d times (%v), want %d (%v)", len(rec.calls), rec.calls, len(tc.wantCalls), tc.wantCalls)
			}
			for i := range tc.wantCalls {
				if rec.calls[i] != tc.wantCalls[i] {
					t.Fatalf("changeHP call %d amount = %d, want %d", i, rec.calls[i], tc.wantCalls[i])
				}
			}
		})
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
go test ./socket/handler/ -run TestComboDrainTryProc -v
```

Expected: compilation FAILURE — `undefined: comboDrainTryProc`.

- [ ] **Step 3: Write the implementation**

Append to `character_attack_combo_drain.go`; extend its imports with
`"github.com/Chronicle20/atlas/libs/atlas-constants/field"` and
`"github.com/sirupsen/logrus"`:

```go
// comboDrainTryProc evaluates Combo Drain for one accepted attack and emits
// at most one ChangeHP via the injected changeHP: once per attack, computed
// from the plain damage total across all monsters and hit lines (no
// per-monster running total). The gate is the COMBO_DRAIN stat alone — no
// job, skill-ownership, attack-type, or version check. A version whose
// client has no Aran simply never carries the stat, so this is correct on
// every supported version without a branch (design §6.1). Failures are
// logged and swallowed — never abort the attack pipeline. Downstream
// max-HP clamping is owned by atlas-character.
func comboDrainTryProc(
	l logrus.FieldLogger,
	getBuffs func(characterId uint32) ([]buff.Model, error),
	changeHP func(f field.Model, characterId uint32, amount int16) error,
	f field.Model,
	characterId uint32,
	ai packetmodel.AttackInfo,
) {
	buffs, err := getBuffs(characterId)
	if err != nil {
		// The loader already logged the failure at Warn level once per
		// attack; skipping the heal is the FR-1 degraded posture.
		l.WithError(err).Debugf("Combo Drain: buff snapshot unavailable for character [%d]; skipping heal.", characterId)
		return
	}
	percent, ok := buffStatAmount(buffs, charconst.TemporaryStatTypeComboDrain)
	if !ok {
		return
	}
	total := attackTotalDamage(ai)
	heal := comboDrainHealAmount(total, percent)
	if heal <= 0 {
		return
	}
	l.Debugf("Combo Drain heal: caster=[%d] totalDamage=[%d] percent=[%d] heal=[%d].", characterId, total, percent, heal)
	if err := changeHP(f, characterId, heal); err != nil {
		l.WithError(err).Errorf("Combo Drain: CHANGE_HP emit failed for character [%d].", characterId)
	}
}
```

- [ ] **Step 4: Run the tests to verify they pass**

```bash
go test ./socket/handler/ -run TestComboDrainTryProc -v
```

Expected: PASS (all subtests, including the error-swallow case with no panic).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo_drain.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo_drain_test.go
git commit -m "feat(channel): combo drain proc orchestrator (task-166)"
```

---

### Task 3: Shared memoized buff loader + wiring into `processAttack`

**Files:**
- Modify: `character_attack_projectile.go` (`Plan` gains a `getBuffs` param;
  internal fetch replaced; `bp` field dropped)
- Modify: `character_attack_common.go` (`loadBuffs` closure; `Plan` call site;
  Pick Pocket call site; TODO line replaced with the proc call)
- Test: `character_attack_combo_drain_test.go` (append the loader test)
- Modify: `docs/TODO.md` (check off the Combo Drain line item)

**Interfaces:**
- Consumes: `comboDrainTryProc` (Task 2 signature);
  `buff.NewProcessor(l, ctx).GetByCharacterId(characterId uint32) ([]buff.Model, error)`;
  `character.Processor.ChangeHP` (`cp` already in scope);
  `session.Model.Field()`/`.CharacterId()`.
- Produces: `ProjectileProcessor.Plan(c character.Model, ai packetmodel.AttackInfo, se effect.Model, getBuffs func(characterId uint32) ([]buff.Model, error)) (*ProjectilePlan, bool)`.

Before editing, confirm no test constructs `ProjectileProcessorImpl` or calls
`Plan`:

```bash
grep -rn "ProjectileProcessorImpl\|\.Plan(" services/atlas-channel/atlas.com/channel
```

Expected: the interface/impl definitions plus exactly one production call site.
If a test appears, update it mechanically in this task.

- [ ] **Step 1: Change `ProjectileProcessor.Plan` to accept the loader**

In `character_attack_projectile.go`:

(a) Interface (currently line 45):

```go
type ProjectileProcessor interface {
	Plan(c character.Model, ai packetmodel.AttackInfo, se effect.Model, getBuffs func(characterId uint32) ([]buff.Model, error)) (*ProjectilePlan, bool)
	Emit(characterId uint32, plan *ProjectilePlan) error
}
```

(b) Drop the `bp` field from the struct and constructor (the `buff` import
stays — `hasBuff`/`computeCount` still use `buff.Model`):

```go
type ProjectileProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	cpp compartment.Processor
}

func NewProjectileProcessor(l logrus.FieldLogger, ctx context.Context) ProjectileProcessor {
	return &ProjectileProcessorImpl{
		l:   l,
		ctx: ctx,
		cpp: compartment.NewProcessor(l, ctx),
	}
}
```

(c) Update the method signature (line 66) and swap the fetch — the fetch stays
where it is, behind the same gates; only its source changes:

```go
func (p *ProjectileProcessorImpl) Plan(c character.Model, ai packetmodel.AttackInfo, se effect.Model, getBuffs func(characterId uint32) ([]buff.Model, error)) (*ProjectilePlan, bool) {
```

and at line 99:

```go
	buffs, err := getBuffs(c.Id())
```

Leave the surrounding warn block, `projectileConsumptionSkipped` and
`computeCount` untouched — the planner keeps its own domain warning and its
"assume none active" posture.

- [ ] **Step 2: Add the memoized loader in `processAttack`**

In `character_attack_common.go`, immediately before the `pp := NewProjectileProcessor(l, ctx)`
block (currently line 888-889), insert:

```go
					// One buff snapshot per attack, shared by the projectile
					// consumption gate, Pick Pocket, and post-damage
					// buff-driven effects (Combo Drain). Fetched at most once
					// and only when a consumer actually needs it; a lookup
					// failure is cached as "no buffs active" for every
					// consumer and never aborts the attack. Mirrors
					// loadEffectiveStats below.
					var attackBuffs []buff.Model
					attackBuffsLoaded := false
					loadBuffs := func(characterId uint32) ([]buff.Model, error) {
						if attackBuffsLoaded {
							return attackBuffs, nil
						}
						attackBuffsLoaded = true
						bs, bErr := buff.NewProcessor(l, ctx).GetByCharacterId(characterId)
						if bErr != nil {
							l.WithError(bErr).Warnf("Unable to load buffs for character [%d] attack; assuming none active.", characterId)
							attackBuffs = nil
							return nil, bErr
						}
						attackBuffs = bs
						return attackBuffs, nil
					}
```

`buff` is already imported in this file (`atlas-channel/character/buff`).

- [ ] **Step 3: Point both existing consumers at the loader**

`Plan` call site (currently line 889):

```go
					projectilePlan, hasProjectilePlan := pp.Plan(c, ai, se, loadBuffs)
```

Pick Pocket call site (currently line 919-925) — replace
`buff.NewProcessor(l, ctx).GetByCharacterId` with `loadBuffs`:

```go
					ppState := pickPocketResolveState(
						l,
						loadBuffs,
						skill2.NewProcessor(l, ctx).GetEffect,
						ai.SkillId(),
						s.CharacterId(),
					)
```

No signature change for `pickPocketResolveState` — it already takes a
`getBuffs` function.

- [ ] **Step 4: Replace the TODO line with the proc call**

In the post-broadcast TODO block (currently line 1117 — after Sacrifice, the
attack-cast dispatcher, Homing Beacon, combo orbs, Aran combo eligibility and
Energy Charge), replace exactly this one line:

```go
					// TODO Combo Drain
```

with:

```go
					comboDrainTryProc(l, loadBuffs, cp.ChangeHP, s.Field(), s.CharacterId(), ai)
```

Do not touch any neighbouring TODO line — sibling tasks own them in their own
worktrees, and a one-line in-place replacement is the merge-friendliest diff.
Ordering satisfies FR-3: this sits after the per-monster damage loop
(`ai.DamageInfo()` is final) and after broadcast/projectile `Emit`, independent
of broadcast success.

- [ ] **Step 5: Add the loader test**

Append to `character_attack_combo_drain_test.go` a test that pins the
one-read-per-attack AC on the loader shape itself — a closure built with the
same memoize-and-cache-the-failure logic, asserting that (a) three calls
produce one underlying fetch and return the same slice, and (b) when the
underlying fetch fails, the error surfaces once and subsequent calls return
`nil, nil` without re-fetching. If the loader is inlined in `processAttack`
and therefore untestable directly, extract it into a small package-level
constructor in `character_attack_combo_drain.go`
(`func newAttackBuffLoader(l logrus.FieldLogger, fetch func(uint32) ([]buff.Model, error)) func(uint32) ([]buff.Model, error)`)
and have `processAttack` call that — prefer this, it keeps the AC honest.

- [ ] **Step 6: Run the full package tests**

```bash
go test ./socket/handler/ -v
```

Expected: PASS — including all pre-existing projectile tests
(`TestComputeCount`, `TestResolvePlan_*`, `TestRequiredClassification`), the
cost-gate test, MP Eater, drain, Pick Pocket, Mortal Blow, Sacrifice and combo
tests, plus Tasks 1–2's Combo Drain tests. If anything referencing `Plan` or
`bp` fails to compile, a call site was missed.

- [ ] **Step 7: Check off the TODO.md line item**

In `docs/TODO.md` (currently line 151), change `- [ ] Combo Drain` to
`- [x] Combo Drain`.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile.go \
        services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go \
        services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo_drain.go \
        services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo_drain_test.go \
        docs/TODO.md
git commit -m "feat(channel): wire combo drain heal into attack pipeline (task-166)"
```

---

### Task 4: Version-scope verification

**Files:** none created or modified — this task proves the version claims in
PRD §4A / design §6 hold for the produced diff. No commit unless a check fails.

**Interfaces:** consumes the completed Tasks 1–3; produces evidence for the
PRD's "Version coverage" acceptance criteria.

- [ ] **Step 1: No version or region branch in the diff**

```bash
git diff main -- services/ | grep -n '^+' | grep -E 'MajorVersion|MajorAtLeast|MinorVersion|IsRegion|Region\(\)'
echo "exit=$?"
```

Expected: no matches (`exit=1`). A hit means the implementation grew a version
gate it must not have (PRD FR-5).

- [ ] **Step 2: No template, packet, or matrix change**

```bash
git diff --stat main -- services/atlas-configurations/seed-data/templates/
git diff --stat main -- docs/packets/ libs/atlas-packet/
```

Expected: both empty. This feature routes no new handler on any of the eleven
templates and promotes no coverage-matrix cell.

- [ ] **Step 3: Re-confirm the per-version applicability table**

Re-derive rather than trust the doc — these are the exact commands the table
was built from. From the worktree root:

```bash
# Aran Combo Drain presence per version (expect: absent in 12/48/61/72,
# present in 79/83/84/87/92/95/jms185)
grep -l "21100005" libs/atlas-constants/gen/wzsnapshot/*.json | sort

# Attack handler routing per template (expect: gms_12_1 and gms_92_1 have none)
for f in services/atlas-configurations/seed-data/templates/template_*.json; do
  printf "%-28s" "$(basename "$f")"
  for h in CharacterMeleeAttackHandle CharacterRangedAttackHandle \
           CharacterMagicAttackHandle CharacterTouchAttackHandle; do
    printf " %s=%s" "${h#Character}" "$(grep -c "\"$h\"" "$f")"
  done
  echo
done
```

If either output disagrees with PRD §4A / design §6.2, update the docs in the
same commit — the table is a claim about the repo and must stay true.

- [ ] **Step 4: Confirm no divergent-id comparison was introduced**

```bash
tools/skill-job-id-guard.sh
grep -rn "21100005\|AranStage2ComboDrainId" services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo_drain.go
```

Expected: guard exit 0; the grep finds nothing — the gate is the temporary
stat, never the skill id.

---

### Task 5: Verification gates

**Files:** none created or modified — runs the design §8 gates. No commit
unless a gate forces a fix.

- [ ] **Step 1: Module gates (from `services/atlas-channel/atlas.com/channel`)**

```bash
go build ./... && go vet ./... && go test -race ./...
```

Expected: all three exit 0; `go test -race` shows `ok` for every package
including `atlas-channel/socket/handler`, no `DATA RACE` output.

- [ ] **Step 2: Repo-root guards**

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/skill-job-id-guard.sh
tools/buff-duration-guard.sh
tools/lint.sh --check
```

Expected: each exits 0. `tools/lint.sh` (no flags) rewrites formatting in place
if `--check` fails. Per project memory, `lint.sh --check` false-fails without
nvm on PATH — load nvm 22 first.

- [ ] **Step 3: Docker bake (from the worktree root — mandatory)**

```bash
docker buildx bake atlas-channel
```

Expected: image builds successfully. `go.mod` is untouched, but the project
rule requires the bake for every changed service regardless.

- [ ] **Step 4: Confirm the TODO marker is gone**

```bash
grep -rn "TODO Combo Drain" services/ docs/TODO.md ; echo "exit=$?"
```

Expected: no matches, `exit=1`.

If any gate fails: fix, re-run every gate from Step 1, and commit the fix on the
task branch. After all gates pass, run `superpowers:requesting-code-review`
before opening a PR (project rule — not part of this plan's tasks, but the
required next step).
