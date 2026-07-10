# Combo Drain (Aran) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the `COMBO_DRAIN` buff (Aran skill 21100005) actually heal the attacker `x`% of each accepted attack's total damage, replacing the `// TODO Combo Drain` in atlas-channel's attack pipeline.

**Architecture:** Design Approach B — `processAttack` fetches the attacker's buffs exactly once (hoisted above the projectile planner, which loses its internal fetch and instead receives the slice), and a new `comboDrainTryProc` evaluates the same slice post-damage, emitting at most one `ChangeHP` Kafka command through the existing `character.Processor`. Pure helpers (`buffStatAmount`, `attackTotalDamage`, `comboDrainHealAmount`) are pinned by table tests, mirroring the MP Eater / computeReflect style already in the handler package.

**Tech Stack:** Go, atlas-channel service only. Existing `buff.Processor` REST read, existing `ChangeHP` Kafka command. No new endpoints, topics, packets, or schema.

## Global Constraints

- Single service touched: `services/atlas-channel/atlas.com/channel` (design §1).
- Exactly one buff lookup per attack, ever (PRD NFR / design §2 Approach B).
- Buff-service failure degrades to "no heal" with a logged warning; the attack pipeline is never aborted (FR-1, design §3.1).
- Heal is once per accepted attack, from the plain total over all monsters and hit lines — no Cosmic per-monster running-total (PRD §1, FR-2).
- `heal = totalDamage * percent / 100`, integer arithmetic, clamped to `math.MaxInt16` before narrowing to `int16`; no emission when `heal <= 0` (FR-2).
- Buff-only gate: no job, skill-ownership, or attack-type check (FR-3, design Approach D rejection).
- Gate stat: `ts.TemporaryStatTypeComboDrain` from `libs/atlas-constants/character` (`"COMBO_DRAIN"`); never a string literal.
- Tests use production constructors (`buff.NewBuff`, `stat.NewStat`, `packetmodel.NewAttackInfo`/`NewDamageInfo` builders) — no `*_testhelpers.go` (project rule).
- Keep the diff to the TODO block minimal: replace exactly the one `// TODO Combo Drain` line in place with the call (design §7 — sibling tasks edit neighboring lines in their own worktrees).
- Verification gates (design §6): `go test -race ./...`, `go vet ./...`, `go build ./...` clean in the module; `tools/redis-key-guard.sh` clean from repo root; `docker buildx bake atlas-channel` from the worktree root (mandatory even though `go.mod` is untouched).
- All repo paths below are relative to the worktree root (`.worktrees/task-166-combo-drain`). The Go module root is `services/atlas-channel/atlas.com/channel` — run go commands from there.

---

### Task 1: Pure helpers — `buffStatAmount`, `attackTotalDamage`, `comboDrainHealAmount`

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo_drain.go`
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo_drain_test.go`

**Interfaces:**
- Consumes: `buff.Model` (`atlas-channel/character/buff`: `Expired() bool`, `Changes() []stat.Model`), `stat.Model` (`Type() string`, `Amount() int32`), `packetmodel.AttackInfo` (`DamageInfo() []DamageInfo`), `packetmodel.DamageInfo` (`Damages() []uint32`), `ts.TemporaryStatType` (`libs/atlas-constants/character`).
- Produces (Task 2 and 3 rely on these exact signatures):
  - `func buffStatAmount(buffs []buff.Model, statType ts.TemporaryStatType) (int32, bool)`
  - `func attackTotalDamage(ai packetmodel.AttackInfo) uint64`
  - `func comboDrainHealAmount(totalDamage uint64, percent int32) int16`

Note on names already taken in package `handler` tests: `buffWithStat`/`expiredBuffWithStat` (fixed amount 1) live in `character_attack_projectile_test.go`, and `testField` lives in `mystic_door_enter_test.go`. The new test file defines amount-parameterized buff helpers under fresh names and reuses `testField` as-is.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo_drain_test.go`:

```go
package handler

import (
	"math"
	"testing"
	"time"

	"atlas-channel/character/buff"
	"atlas-channel/character/buff/stat"

	ts "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// comboDrainBuffWithAmount builds a live buff carrying a single COMBO_DRAIN
// stat change of the given amount.
func comboDrainBuffWithAmount(amount int32) buff.Model {
	return buff.NewBuff(0, 1, 0,
		[]stat.Model{stat.NewStat(string(ts.TemporaryStatTypeComboDrain), amount)},
		time.Now(), time.Now().Add(time.Hour))
}

// expiredComboDrainBuffWithAmount builds an already-expired COMBO_DRAIN buff.
func expiredComboDrainBuffWithAmount(amount int32) buff.Model {
	past := time.Now().Add(-time.Hour)
	return buff.NewBuff(0, 1, 0,
		[]stat.Model{stat.NewStat(string(ts.TemporaryStatTypeComboDrain), amount)},
		past, past)
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
		[]stat.Model{stat.NewStat(string(ts.TemporaryStatTypeSoulArrow), 1)},
		time.Now(), time.Now().Add(time.Hour))
	mixedStats := buff.NewBuff(0, 1, 0,
		[]stat.Model{
			stat.NewStat(string(ts.TemporaryStatTypeSoulArrow), 1),
			stat.NewStat(string(ts.TemporaryStatTypeComboDrain), 4),
		},
		time.Now(), time.Now().Add(time.Hour))

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
			amount, ok := buffStatAmount(tc.buffs, ts.TemporaryStatTypeComboDrain)
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

- [ ] **Step 2: Run the tests to verify they fail**

Run (from `services/atlas-channel/atlas.com/channel`):

```bash
go test ./socket/handler/ -run 'TestBuffStatAmount|TestAttackTotalDamage|TestComboDrainHealAmount' -v
```

Expected: compilation FAILURE — `undefined: buffStatAmount`, `undefined: attackTotalDamage`, `undefined: comboDrainHealAmount`.

- [ ] **Step 3: Write the implementation**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo_drain.go`:

```go
package handler

import (
	"atlas-channel/character/buff"
	"math"

	ts "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// buffStatAmount returns the Amount of the first stat change of statType
// carried by a non-expired buff, mirroring hasBuff's matching rules.
func buffStatAmount(buffs []buff.Model, statType ts.TemporaryStatType) (int32, bool) {
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
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo_drain.go` (append function + imports)
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo_drain_test.go` (append tests)

**Interfaces:**
- Consumes: Task 1's three helpers (exact signatures above); `field.Model` (`libs/atlas-constants/field`); `logrus.FieldLogger`.
- Produces (Task 3's call site relies on this exact signature — `changeHP` matches `character.Processor.ChangeHP(f field.Model, characterId uint32, amount int16) error`):
  - `func comboDrainTryProc(l logrus.FieldLogger, buffs []buff.Model, changeHP func(f field.Model, characterId uint32, amount int16) error, f field.Model, characterId uint32, ai packetmodel.AttackInfo)`

- [ ] **Step 1: Write the failing tests**

Append to `character_attack_combo_drain_test.go`. New imports needed in the test file's import block: `"atlas-channel/character/buff/stat"` and `ts` are already there; add `"github.com/Chronicle20/atlas/libs/atlas-constants/field"`, `"github.com/sirupsen/logrus"`, `"errors"`:

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

func TestComboDrainTryProc(t *testing.T) {
	l := logrus.New()
	f := testField(100000000)

	tests := []struct {
		name      string
		buffs     []buff.Model
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
			// The handler collapses a buff-fetch error to nil buffs, so this
			// case is also the buff-fetch-error AC at proc level.
			name:      "nil buffs (fetch error posture)",
			buffs:     nil,
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
			name:      "energy attack heals",
			buffs:     []buff.Model{comboDrainBuffWithAmount(5)},
			ai:        attackWithDamages(packetmodel.AttackTypeEnergy, []uint32{1000}),
			wantCalls: []int16{50},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := &recordingChangeHP{err: tc.changeErr}
			comboDrainTryProc(l, tc.buffs, rec.fn, f, 42, tc.ai)
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

Append to `character_attack_combo_drain.go`. Extend its import block with `"github.com/Chronicle20/atlas/libs/atlas-constants/field"` and `"github.com/sirupsen/logrus"`:

```go
// comboDrainTryProc evaluates Combo Drain for one accepted attack and emits
// at most one ChangeHP via the injected changeHP: once per attack, computed
// from the plain damage total across all monsters and hit lines (no
// per-monster running total). The gate is the COMBO_DRAIN stat alone — no
// job, skill-ownership, or attack-type check. Failures are logged and
// swallowed — never abort the attack pipeline. Downstream max-HP clamping
// is owned by atlas-character.
func comboDrainTryProc(
	l logrus.FieldLogger,
	buffs []buff.Model,
	changeHP func(f field.Model, characterId uint32, amount int16) error,
	f field.Model,
	characterId uint32,
	ai packetmodel.AttackInfo,
) {
	percent, ok := buffStatAmount(buffs, ts.TemporaryStatTypeComboDrain)
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

### Task 3: Single shared buff fetch + wiring into `processAttack`

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile.go` (`Plan` signature gains `buffs`, internal fetch removed, `bp` field dropped)
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` (hoisted buff fetch; `Plan` call site; TODO line replaced with the proc call; `buff` import added)
- Modify: `docs/TODO.md` (check off the Combo Drain line item)

**Interfaces:**
- Consumes: `comboDrainTryProc` (Task 2 signature); `buff.NewProcessor(l, ctx).GetByCharacterId(characterId uint32) ([]buff.Model, error)`; `character.Processor.ChangeHP` (`cp` is already in scope in `processAttack`); `session.Model.Field()`/`.CharacterId()`.
- Produces: `ProjectileProcessor.Plan(c character.Model, ai packetmodel.AttackInfo, se effect.Model, buffs []buff.Model) (*ProjectilePlan, bool)` — new interface shape. `Plan` has exactly one production call site (`character_attack_common.go:316`) and no test constructs a `ProjectileProcessorImpl` or calls `Plan` directly (the projectile tests exercise the pure helpers `computeCount`/`resolvePlan`/`hasBuff`, which already take `[]buff.Model` and are untouched), so no test-file edits are required in this task.

No new test file in this task: the behavior it wires is pinned by Task 2's proc tests (including the nil-buffs fetch-error posture) and by the existing projectile helper tests, which this task must keep green. The refactor is behavior-neutral for the planner: the same slice reaches the same decision points, fetched by the caller instead.

- [ ] **Step 1: Change `ProjectileProcessor.Plan` to accept the buff slice**

In `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile.go`:

(a) Interface (currently at line 43):

```go
type ProjectileProcessor interface {
	Plan(c character.Model, ai packetmodel.AttackInfo, se effect.Model, buffs []buff.Model) (*ProjectilePlan, bool)
	Emit(characterId uint32, plan *ProjectilePlan) error
}
```

(b) Drop the `bp` field from the struct and constructor (the `buff` import stays — `hasBuff`/`computeCount` still use `buff.Model`):

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

(c) Update the method signature and delete the internal fetch. The method opens:

```go
func (p *ProjectileProcessorImpl) Plan(c character.Model, ai packetmodel.AttackInfo, se effect.Model, buffs []buff.Model) (*ProjectilePlan, bool) {
```

and this entire block (currently right after the `requiredClassification` gate) is deleted — the `buffs` parameter feeds the Soul Arrow check and `computeCount` directly:

```go
	buffs, err := p.bp.GetByCharacterId(c.Id())
	if err != nil {
		// Treat a buff-lookup failure as "no buffs" so consumption still fires.
		// Soul Arrow is a gameplay-critical skip but we'd rather over-consume than
		// nil-ref the attack hot path.
		p.l.WithError(err).WithField("characterId", c.Id()).
			Warnf("Unable to load buffs for projectile gate; assuming none active.")
		buffs = nil
	}
```

Everything below (`hasBuff(buffs, ts.TemporaryStatTypeSoulArrow)` check, `computeCount(weaponType, se, buffs)`) is unchanged — it already consumes the local name `buffs`, which is now the parameter.

- [ ] **Step 2: Hoist the single buff fetch in `processAttack` and pass it to `Plan`**

In `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`, add `"atlas-channel/character/buff"` to the import block (first group, alongside `"atlas-channel/character"`), then replace (currently lines 313–316):

```go
					// Compute projectile consumption plan before broadcasting so planner
					// errors surface before visible side effects. Emission happens post-broadcast.
					pp := NewProjectileProcessor(l, ctx)
					projectilePlan, hasProjectilePlan := pp.Plan(c, ai, se)
```

with:

```go
					// One buff snapshot per attack, taken at attack-accept time and
					// shared by the projectile consumption gate and post-damage
					// buff-driven effects (Combo Drain). A lookup failure degrades to
					// "no buffs active" for every consumer — never aborts the attack.
					buffs, buffErr := buff.NewProcessor(l, ctx).GetByCharacterId(s.CharacterId())
					if buffErr != nil {
						l.WithError(buffErr).Warnf("Unable to load buffs for character [%d] attack; assuming none active.", s.CharacterId())
						buffs = nil
					}

					// Compute projectile consumption plan before broadcasting so planner
					// errors surface before visible side effects. Emission happens post-broadcast.
					pp := NewProjectileProcessor(l, ctx)
					projectilePlan, hasProjectilePlan := pp.Plan(c, ai, se, buffs)
```

- [ ] **Step 3: Replace the TODO line with the proc call**

In the same file, in the post-broadcast TODO block (currently line 420), replace exactly this one line:

```go
					// TODO Combo Drain
```

with:

```go
					comboDrainTryProc(l, buffs, cp.ChangeHP, s.Field(), s.CharacterId(), ai)
```

Do not touch any neighboring TODO line — sibling tasks own them in their own worktrees, and a one-line in-place replacement is the merge-friendliest diff. Ordering satisfies FR-3: this sits after the per-monster damage loop (`ai.DamageInfo()` is final) and after broadcast/projectile `Emit`, independent of broadcast success.

- [ ] **Step 4: Run the full package tests**

```bash
go test ./socket/handler/ -v
```

Expected: PASS — including all pre-existing projectile tests (`TestComputeCount`, `TestResolvePlan_*`, `TestRequiredClassification`), the cost-gate test, MP Eater tests, and Tasks 1–2's Combo Drain tests. If anything referencing `Plan` or `bp` fails to compile, a call site was missed — `grep -rn "\.Plan(c" services/atlas-channel` must show only the one updated site.

- [ ] **Step 5: Check off the TODO.md line item**

In `docs/TODO.md` (currently line 108), change:

```markdown
- [ ] Combo Drain
```

to:

```markdown
- [x] Combo Drain
```

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go docs/TODO.md
git commit -m "feat(channel): wire combo drain heal into attack pipeline (task-166)"
```

---

### Task 4: Verification gates

**Files:**
- None created or modified — this task runs the design §6 gates and must show clean output before the branch can be called done. No commit unless a gate forces a fix.

**Interfaces:**
- Consumes: the completed Tasks 1–3.
- Produces: evidence for `superpowers:verification-before-completion`.

- [ ] **Step 1: Module gates (from `services/atlas-channel/atlas.com/channel`)**

```bash
go build ./... && go vet ./... && go test -race ./...
```

Expected: all three exit 0; `go test -race` shows `ok` for every package including `atlas-channel/socket/handler`, no `DATA RACE` output.

- [ ] **Step 2: Redis key guard (from the worktree root)**

```bash
tools/redis-key-guard.sh
```

Expected: exit 0, no violations (this change adds no Redis usage). Per project memory, run from repo root without a global `GOWORK=off` prefix.

- [ ] **Step 3: Docker bake (from the worktree root — mandatory)**

```bash
docker buildx bake atlas-channel
```

Expected: image builds successfully. `go.mod` was not touched, but the project rule requires the bake for every changed service regardless.

- [ ] **Step 4: Confirm the TODO marker is gone**

```bash
grep -rn "TODO Combo Drain" services/ docs/TODO.md ; echo "exit=$?"
```

Expected: no matches, `exit=1`.

If any gate fails: fix, re-run every gate from Step 1, and amend/commit the fix on the task branch. After all gates pass, run `superpowers:requesting-code-review` before opening a PR (project rule — not part of this plan's tasks, but the required next step).
