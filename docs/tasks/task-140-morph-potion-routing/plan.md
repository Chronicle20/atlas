# Morph Potion Routing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Route classification-221 (transformation) consumables through `ConsumeStandard` so the existing morph applier fires, and add weighted-random selection for the `morphRandom` spec — all inside `atlas-consumables`.

**Architecture:** Four parts per the approved design: (1) one new case in `usesStandardConsumer`; (2) a pure-helper file `consumable/morph.go` mirroring the task-131 `reward.go` randomness seam (crypto/rand roll isolated from a pure, roll-parameterized selection function); (3) a behavior-preserving extraction of the pure "which effects does this item produce" computation out of `ApplyItemEffects` into `computeEffectPlan` so morph/hp behavior is pinnable by plain unit tests; (4) an additive `Morphs()` getter on the data-side consumable model.

**Tech Stack:** Go, testify (already a dependency), `crypto/rand` for the roll. No new dependencies, no `go.mod` change expected.

## Global Constraints

- **Diff scope:** the code diff must touch ONLY `services/atlas-consumables/` (plus `docs/tasks/task-140-morph-potion-routing/`). No changes in atlas-data, atlas-buffs, atlas-channel, or any lib (PRD acceptance criterion).
- **Named constant:** routing MUST use `item.ClassificationConsumableTransformation` (`libs/atlas-constants/item/constants.go:39`), never a raw `221` literal (FR-1). The existing `Classification(200/201/202/205)` literals stay as-is (design §2 — no named constants exist for those; renaming is explicitly out of scope).
- **Randomness seam:** `crypto/rand` roll isolated in a pure-helper file, NO seeded/injectable PRNG (design §3.1, task-131 precedent). Determinism under test comes from testing the pure selection function exhaustively.
- **Weights are data:** probability of morph `m` = `weight(m) / sum(weights)`; never assume weights sum to 100 (FR-5).
- **Precedence:** fixed `morph` spec > 0 wins over a non-empty `morphRandom` table; no error, no double-apply (FR-7).
- **Behavior preservation:** the `computeEffectPlan` extraction moves computation only; side effects and their exact order (cures → HP/MP changes → single `bp.Apply`) are unchanged for every existing item (design §4.2, §11).
- **Error semantics:** an unusable (zero-total) morph table logs a warning and omits only the morph statup; other specs still apply and consumption stands (design §6).
- **Test fixtures:** build `consumable3.RestModel` literals and run them through the public `Extract` — the same path production data takes. No `*_testhelpers.go`, no test-only constructors (project rule + design §4.4). Morph-table values in tests are synthetic fixtures shaped like the real data, not asserted WZ values.
- **No TODOs / stubs** in any commit.
- **Worktree:** all work happens in the `.worktrees/task-140-morph-potion-routing` worktree on branch `task-140-morph-potion-routing`. Verify `git branch --show-current` after each commit.
- All Go test/vet/build commands run from `services/atlas-consumables/atlas.com/consumables/` (the module root). `tools/redis-key-guard.sh` runs from the worktree root WITHOUT a `GOWORK=off` prefix.

---

### Task 1: `Morphs()` getter on the data-side consumable model (FR-4)

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/data/consumable/model.go` (add getter after `MonsterSummons()`, ~line 188)
- Test: `services/atlas-consumables/atlas.com/consumables/data/consumable/model_test.go` (create)

**Interfaces:**
- Consumes: existing `Model.morphs map[uint32]uint32` field (already populated by `Extract` at `rest.go:163` from `RestModel.Morphs`).
- Produces: `func (m Model) Morphs() map[uint32]uint32` — used by Task 4's morph branch and its tests.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-consumables/atlas.com/consumables/data/consumable/model_test.go`:

```go
package consumable

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMorphsGetter(t *testing.T) {
	rm := RestModel{Morphs: map[uint32]uint32{100: 30, 101: 70}}
	m, err := Extract(rm)
	assert.NoError(t, err)
	assert.Equal(t, map[uint32]uint32{100: 30, 101: 70}, m.Morphs())
}

func TestMorphsGetter_Empty(t *testing.T) {
	m, err := Extract(RestModel{})
	assert.NoError(t, err)
	assert.Empty(t, m.Morphs())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run (from `services/atlas-consumables/atlas.com/consumables/`):
```bash
go test ./data/consumable/ -run TestMorphsGetter -v
```
Expected: FAIL to compile with `m.Morphs undefined (type Model has no field or method Morphs)`.

- [ ] **Step 3: Write minimal implementation**

In `data/consumable/model.go`, directly after the `MonsterSummons()` method (line 186-188):

```go
// Morphs returns the item's morphRandom table (morph id -> weight). The
// returned map is the internal reference, matching the MonsterSummons()
// accessor convention; callers are read-only.
func (m Model) Morphs() map[uint32]uint32 {
	return m.morphs
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./data/consumable/ -run TestMorphsGetter -v
```
Expected: PASS (both subtests).

- [ ] **Step 5: Commit**

```bash
git add data/consumable/model.go data/consumable/model_test.go
git commit -m "feat(consumables): expose Morphs() getter on consumable data model"
git branch --show-current   # must print task-140-morph-potion-routing
```

---

### Task 2: Weighted-random morph selection seam `morph.go` (FR-5, FR-6)

**Files:**
- Create: `services/atlas-consumables/atlas.com/consumables/consumable/morph.go`
- Test: `services/atlas-consumables/atlas.com/consumables/consumable/morph_test.go` (create)

**Interfaces:**
- Consumes: nothing from other tasks (plain `map[uint32]uint32`).
- Produces:
  - `func selectMorph(morphs map[uint32]uint32, roll uint32) (uint32, bool)` — pure; deterministic function of the roll.
  - `func rollMorph(morphs map[uint32]uint32) (uint32, error)` — crypto/rand draw + delegate; used by Task 4.

Note: `morph.go` is a separate file so `crypto/rand` does not collide with `processor.go`'s `math/rand` import — same layout as task-131's `reward.go` precedent.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-consumables/atlas.com/consumables/consumable/morph_test.go`:

```go
package consumable

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// T2: exhaustive weighting — enumerate every roll in [0, total) and assert the
// per-morph selection count equals its weight exactly. Stronger than any
// seeded statistical test (design §3.2). Table is shaped like the real
// 2211000/2212000 data (entries summing to 100); values are synthetic fixtures.
func TestSelectMorph_ExhaustiveWeighting(t *testing.T) {
	morphs := map[uint32]uint32{10: 20, 11: 30, 12: 50}
	counts := make(map[uint32]int)
	for roll := uint32(0); roll < 100; roll++ {
		id, ok := selectMorph(morphs, roll)
		assert.True(t, ok, "roll %d", roll)
		counts[id]++
	}
	assert.Equal(t, map[uint32]int{10: 20, 11: 30, 12: 50}, counts)
}

// T2b: FR-5 — no assumption that weights sum to 100.
func TestSelectMorph_WeightsNotSummingTo100(t *testing.T) {
	morphs := map[uint32]uint32{1: 3, 2: 1}
	counts := make(map[uint32]int)
	for roll := uint32(0); roll < 4; roll++ {
		id, ok := selectMorph(morphs, roll)
		assert.True(t, ok, "roll %d", roll)
		counts[id]++
	}
	assert.Equal(t, map[uint32]int{1: 3, 2: 1}, counts)
}

// T3: degenerate tables.
func TestSelectMorph_EmptyTable(t *testing.T) {
	_, ok := selectMorph(map[uint32]uint32{}, 0)
	assert.False(t, ok)
}

func TestSelectMorph_AllZeroWeights(t *testing.T) {
	_, ok := selectMorph(map[uint32]uint32{5: 0, 6: 0}, 0)
	assert.False(t, ok)
}

func TestSelectMorph_ZeroWeightEntrySkipped(t *testing.T) {
	morphs := map[uint32]uint32{1: 0, 2: 5}
	for roll := uint32(0); roll < 5; roll++ {
		id, ok := selectMorph(morphs, roll)
		assert.True(t, ok, "roll %d", roll)
		assert.Equal(t, uint32(2), id, "roll %d", roll)
	}
}

// T4: roll seam — zero-total errors; valid results are always table keys.
// No distribution assertion here; TestSelectMorph_ExhaustiveWeighting owns weighting.
func TestRollMorph_ZeroTotalErrors(t *testing.T) {
	_, err := rollMorph(map[uint32]uint32{7: 0})
	assert.Error(t, err)
	_, err = rollMorph(map[uint32]uint32{})
	assert.Error(t, err)
}

func TestRollMorph_ResultAlwaysTableKey(t *testing.T) {
	morphs := map[uint32]uint32{10: 20, 11: 30, 12: 50}
	for i := 0; i < 200; i++ {
		id, err := rollMorph(morphs)
		assert.NoError(t, err)
		_, present := morphs[id]
		assert.True(t, present, "rolled id %d not in table", id)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./consumable/ -run 'TestSelectMorph|TestRollMorph' -v
```
Expected: FAIL to compile with `undefined: selectMorph` / `undefined: rollMorph`.

- [ ] **Step 3: Write the implementation**

Create `services/atlas-consumables/atlas.com/consumables/consumable/morph.go`:

```go
package consumable

import (
	"crypto/rand"
	"errors"
	"math/big"
	"sort"
)

// selectMorph is the pure selection function: given the weighted morph table
// and a roll in [0, sum(weights)), return the selected morph id. Morph ids are
// walked in ascending order — Go map iteration order is randomized, so sorting
// is what makes selection a deterministic function of the roll. Returns false
// when the table is empty or all weights are zero.
func selectMorph(morphs map[uint32]uint32, roll uint32) (uint32, bool) {
	ids := make([]uint32, 0, len(morphs))
	for id := range morphs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	var cumulative uint32
	for _, id := range ids {
		cumulative += morphs[id]
		if roll < cumulative {
			return id, true
		}
	}
	return 0, false
}

// rollMorph performs one clean weight-based pick over the morph table using a
// CSPRNG (the task-131 reward.go seam). It sums the weights, errors on a zero
// total (defense in depth; the caller treats this as "skip the morph statup"),
// draws one integer in [0, total), and delegates to selectMorph.
func rollMorph(morphs map[uint32]uint32) (uint32, error) {
	var total uint32
	for _, w := range morphs {
		total += w
	}
	if total == 0 {
		return 0, errors.New("morph table has zero total weight")
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(total)))
	if err != nil {
		return 0, err
	}

	id, ok := selectMorph(morphs, uint32(n.Int64()))
	if !ok {
		// Unreachable given total > 0, but never return a fabricated morph id.
		return 0, errors.New("morph selection failed")
	}
	return id, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./consumable/ -run 'TestSelectMorph|TestRollMorph' -v
```
Expected: PASS (all seven tests).

- [ ] **Step 5: Commit**

```bash
git add consumable/morph.go consumable/morph_test.go
git commit -m "feat(consumables): weighted-random morph selection seam (crypto/rand roll + pure selection)"
git branch --show-current   # must print task-140-morph-potion-routing
```

---

### Task 3: Extract `computeEffectPlan` from `ApplyItemEffects` (behavior-preserving refactor)

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/processor.go:109-182` (`ApplyItemEffects`; add `effectPlan` + `computeEffectPlan` directly above it)
- Test: `services/atlas-consumables/atlas.com/consumables/consumable/processor_test.go` (append)

**Interfaces:**
- Consumes: `collectCureTypes(ci)` (existing, `processor.go:89`), `character.Model` getters `MaxHp()`/`MaxMp()`, `consumable3.Model.GetSpec`, `stat.Model{Type character.TemporaryStatType; Amount int32}`.
- Produces (used by Task 4):

```go
type effectPlan struct {
	cureTypes []string     // ordered; from collectCureTypes
	hpChanges []int16      // ordered ChangeHP calls (hp, then hpR-derived)
	mpChanges []int16      // ordered ChangeMP calls (mp, then mpR-derived)
	statups   []stat.Model // includes the resolved morph statup, if any
	duration  int32        // time spec / 1000
}

func computeEffectPlan(l logrus.FieldLogger, c character.Model, ci consumable3.Model) effectPlan
```

This task does NOT add the morph-random branch — it is a pure move of the existing computation (the fixed-`morph` branch moves verbatim). The logger parameter is unused until Task 4 adds the roll-failure warn path; it is part of the signature from the start so Task 4 does not change the signature (Go permits unused function parameters).

- [ ] **Step 1: Write the failing regression tests (T8 + hpR math pin)**

Append to `services/atlas-consumables/atlas.com/consumables/consumable/processor_test.go`. Also extend the import block: add `"io"`, `"atlas-consumables/character"`, `"atlas-consumables/character/buff/stat"`, `ts "github.com/Chronicle20/atlas/libs/atlas-constants/character"`, and `"github.com/sirupsen/logrus"` to the existing imports.

```go
// discardLogger returns a logger for computeEffectPlan tests; the function
// only logs on the morphRandom roll-failure path.
func discardLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

// extractConsumable builds a consumable model the same way production data
// arrives: a RestModel literal run through the public Extract (design §4.4).
func extractConsumable(t *testing.T, rm consumable3.RestModel) consumable3.Model {
	t.Helper()
	m, err := consumable3.Extract(rm)
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	return m
}

// T8: refactor regression — representative pre-existing items produce the same
// decisions ApplyItemEffects made before the extraction.
func TestComputeEffectPlan_CurePotWithHp(t *testing.T) {
	c := character.NewModelBuilder().SetMaxHp(500).SetMaxMp(500).Build()
	ci := extractConsumable(t, consumable3.RestModel{
		Spec: map[consumable3.SpecType]int32{
			consumable3.SpecTypePoison: 1,
			consumable3.SpecTypeHP:     300,
		},
	})
	plan := computeEffectPlan(discardLogger(), c, ci)
	assert.Equal(t, []string{"POISON"}, plan.cureTypes)
	assert.Equal(t, []int16{300}, plan.hpChanges)
	assert.Empty(t, plan.mpChanges)
	assert.Empty(t, plan.statups)
	assert.Equal(t, int32(0), plan.duration)
}

func TestComputeEffectPlan_StatPotWithTime(t *testing.T) {
	c := character.NewModelBuilder().SetMaxHp(500).SetMaxMp(500).Build()
	ci := extractConsumable(t, consumable3.RestModel{
		Spec: map[consumable3.SpecType]int32{
			consumable3.SpecTypeWeaponAttack: 12,
			consumable3.SpecTypeTime:         300000,
		},
	})
	plan := computeEffectPlan(discardLogger(), c, ci)
	assert.Empty(t, plan.cureTypes)
	assert.Empty(t, plan.hpChanges)
	assert.Empty(t, plan.mpChanges)
	assert.Equal(t, []stat.Model{{Type: ts.TemporaryStatTypeWeaponAttack, Amount: 12}}, plan.statups)
	assert.Equal(t, int32(300), plan.duration)
}

func TestComputeEffectPlan_HpRecoveryPercent(t *testing.T) {
	// Pins the MaxHp * pct floor math: floor(1547 * 0.60) = 928.
	c := character.NewModelBuilder().SetMaxHp(1547).Build()
	ci := extractConsumable(t, consumable3.RestModel{
		Spec: map[consumable3.SpecType]int32{
			consumable3.SpecTypeHPRecovery: 60,
		},
	})
	plan := computeEffectPlan(discardLogger(), c, ci)
	assert.Equal(t, []int16{928}, plan.hpChanges)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./consumable/ -run TestComputeEffectPlan -v
```
Expected: FAIL to compile with `undefined: computeEffectPlan`.

- [ ] **Step 3: Implement the extraction**

In `consumable/processor.go`, insert directly above `ApplyItemEffects` (after `collectCureTypes`):

```go
// effectPlan is the pure result of interpreting a consumable's specs against a
// character: everything ApplyItemEffects will do, decided before any side effect.
type effectPlan struct {
	cureTypes []string     // ordered; from collectCureTypes
	hpChanges []int16      // ordered ChangeHP calls (hp, then hpR-derived)
	mpChanges []int16      // ordered ChangeMP calls (mp, then mpR-derived)
	statups   []stat.Model // includes the resolved morph statup, if any
	duration  int32        // time spec / 1000
}

// computeEffectPlan interprets a consumable's specs against a character with
// no side effects. ApplyItemEffects executes the plan; keeping the decision
// pure is what makes the morph/hp paths pinnable by plain unit tests.
func computeEffectPlan(l logrus.FieldLogger, c character.Model, ci consumable3.Model) effectPlan {
	plan := effectPlan{
		cureTypes: collectCureTypes(ci),
		hpChanges: make([]int16, 0, 2),
		mpChanges: make([]int16, 0, 2),
		statups:   make([]stat.Model, 0),
	}

	if val, ok := ci.GetSpec(consumable3.SpecTypeHP); ok && val > 0 {
		plan.hpChanges = append(plan.hpChanges, int16(val))
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeHPRecovery); ok && val > 0 {
		pct := float64(val) / float64(100)
		plan.hpChanges = append(plan.hpChanges, int16(math.Floor(float64(c.MaxHp())*pct)))
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeMP); ok && val > 0 {
		plan.mpChanges = append(plan.mpChanges, int16(val))
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeMPRecovery); ok && val > 0 {
		pct := float64(val) / float64(100)
		plan.mpChanges = append(plan.mpChanges, int16(math.Floor(float64(c.MaxMp())*pct)))
	}

	if val, ok := ci.GetSpec(consumable3.SpecTypeAccuracy); ok && val > 0 {
		plan.statups = append(plan.statups, stat.Model{Type: ts.TemporaryStatTypeAccuracy, Amount: val})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeEvasion); ok && val > 0 {
		plan.statups = append(plan.statups, stat.Model{Type: ts.TemporaryStatTypeAvoidability, Amount: val})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeJump); ok && val > 0 {
		plan.statups = append(plan.statups, stat.Model{Type: ts.TemporaryStatTypeJump, Amount: val})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeMagicAttack); ok && val > 0 {
		plan.statups = append(plan.statups, stat.Model{Type: ts.TemporaryStatTypeMagicAttack, Amount: val})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeMagicDefense); ok && val > 0 {
		plan.statups = append(plan.statups, stat.Model{Type: ts.TemporaryStatTypeMagicDefense, Amount: val})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeWeaponAttack); ok && val > 0 {
		plan.statups = append(plan.statups, stat.Model{Type: ts.TemporaryStatTypeWeaponAttack, Amount: val})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeWeaponDefense); ok && val > 0 {
		plan.statups = append(plan.statups, stat.Model{Type: ts.TemporaryStatTypeWeaponDefense, Amount: val})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeSpeed); ok && val > 0 {
		plan.statups = append(plan.statups, stat.Model{Type: ts.TemporaryStatTypeSpeed, Amount: val})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeMorph); ok && val > 0 {
		plan.statups = append(plan.statups, stat.Model{Type: ts.TemporaryStatTypeMorph, Amount: val})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeTime); ok && val > 0 {
		plan.duration = val / 1000
	}

	return plan
}
```

Then replace the body of `ApplyItemEffects` (keep the function signature and its doc comment; the task-051 D3 cure-ordering comment moves with the code):

```go
// ApplyItemEffects applies the effects of a consumable item to a character.
// This is the shared logic used by both regular item consumption and NPC-initiated item use.
// It handles stat buffs (accuracy, evasion, attack, defense, speed, jump) and HP/MP recovery.
func ApplyItemEffects(l logrus.FieldLogger, ctx context.Context, c character.Model, f field.Model, ci consumable3.Model, characterId uint32, itemId item2.Id) {
	bp := buff.NewProcessor(l, ctx)
	cp := character.NewProcessor(l, ctx)

	plan := computeEffectPlan(l, c, ci)

	// 1. Cure first. Cure runs before HP/MP recovery so a queued poison tick
	// (also routed through atlas-buffs's per-character partition) lands behind
	// the cancel and cannot eat part of the heal between drink-time and
	// cancel-commit-time. See task-051 D3.
	if len(plan.cureTypes) > 0 {
		if err := bp.CancelByTypes(f, characterId, plan.cureTypes); err != nil {
			l.WithError(err).Errorf("Unable to dispatch cure-by-types for character [%d] item [%d].", characterId, itemId)
		}
	}

	// 2. HP/MP recovery.
	for _, amount := range plan.hpChanges {
		_ = cp.ChangeHP(f, characterId, amount)
	}
	for _, amount := range plan.mpChanges {
		_ = cp.ChangeMP(f, characterId, amount)
	}

	// 3. Status-up buffs.
	if len(plan.statups) > 0 {
		_ = bp.Apply(f, characterId, -int32(itemId), byte(0), plan.duration, plan.statups)(characterId)
	}
}
```

- [ ] **Step 4: Run the full package tests to verify pass (including all pre-existing tests)**

```bash
go test -race ./consumable/ -v
```
Expected: PASS — the three new `TestComputeEffectPlan_*` tests AND every pre-existing test (cure-type tests, scroll tests, `TestUsesStandardConsumer`) unchanged.

- [ ] **Step 5: Commit**

```bash
git add consumable/processor.go consumable/processor_test.go
git commit -m "refactor(consumables): extract pure computeEffectPlan from ApplyItemEffects"
git branch --show-current   # must print task-140-morph-potion-routing
```

---

### Task 4: Morph branch in `computeEffectPlan` (FR-3, FR-5, FR-7)

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/processor.go` (the fixed-`morph` branch inside `computeEffectPlan` added in Task 3)
- Test: `services/atlas-consumables/atlas.com/consumables/consumable/processor_test.go` (append)

**Interfaces:**
- Consumes: `rollMorph(morphs map[uint32]uint32) (uint32, error)` from Task 2; `ci.Morphs() map[uint32]uint32` from Task 1; `effectPlan`/`computeEffectPlan` from Task 3.
- Produces: final morph semantics — fixed `morph` spec wins; otherwise one weighted-random pick from `Morphs()`; unusable table warns and skips only the morph statup.

- [ ] **Step 1: Write the failing tests (T5, T6, T7 + unusable-table skip)**

Append to `consumable/processor_test.go`:

```go
// T5 (FR-3 + hp-alongside): fixed-morph 221 item applies MORPH statup with the
// morph id, duration = time/1000, and the coexisting hp spec still heals.
func TestComputeEffectPlan_FixedMorphWithHp(t *testing.T) {
	c := character.NewModelBuilder().SetMaxHp(100).Build()
	ci := extractConsumable(t, consumable3.RestModel{
		Spec: map[consumable3.SpecType]int32{
			consumable3.SpecTypeMorph: 2,
			consumable3.SpecTypeTime:  600000,
			consumable3.SpecTypeHP:    50,
		},
	})
	plan := computeEffectPlan(discardLogger(), c, ci)
	assert.Equal(t, []stat.Model{{Type: ts.TemporaryStatTypeMorph, Amount: 2}}, plan.statups)
	assert.Equal(t, int32(600), plan.duration)
	assert.Equal(t, []int16{50}, plan.hpChanges)
}

// T6: 2211000-shaped item — no fixed morph spec, non-empty morphRandom table.
// Exactly one MORPH statup whose amount is a table key; hp still applies.
func TestComputeEffectPlan_RandomMorphOnly(t *testing.T) {
	c := character.NewModelBuilder().SetMaxHp(100).Build()
	morphs := map[uint32]uint32{20: 50, 21: 30, 22: 20}
	ci := extractConsumable(t, consumable3.RestModel{
		Spec: map[consumable3.SpecType]int32{
			consumable3.SpecTypeTime: 600000,
			consumable3.SpecTypeHP:   50,
		},
		Morphs: morphs,
	})
	plan := computeEffectPlan(discardLogger(), c, ci)
	if assert.Len(t, plan.statups, 1) {
		s := plan.statups[0]
		assert.Equal(t, ts.TemporaryStatTypeMorph, s.Type)
		_, present := morphs[uint32(s.Amount)]
		assert.True(t, present, "morph amount %d is not a table key", s.Amount)
	}
	assert.Equal(t, []int16{50}, plan.hpChanges)
	assert.Equal(t, int32(600), plan.duration)
}

// T7 (FR-7): fixed morph wins over a table that deliberately does not contain it.
func TestComputeEffectPlan_FixedMorphPrecedence(t *testing.T) {
	ci := extractConsumable(t, consumable3.RestModel{
		Spec:   map[consumable3.SpecType]int32{consumable3.SpecTypeMorph: 2},
		Morphs: map[uint32]uint32{20: 100},
	})
	plan := computeEffectPlan(discardLogger(), character.NewModelBuilder().Build(), ci)
	assert.Equal(t, []stat.Model{{Type: ts.TemporaryStatTypeMorph, Amount: 2}}, plan.statups)
}

// Design §6: an unusable (zero-total) table skips only the morph statup;
// other specs still apply.
func TestComputeEffectPlan_ZeroWeightMorphTableSkipsMorphOnly(t *testing.T) {
	ci := extractConsumable(t, consumable3.RestModel{
		Spec:   map[consumable3.SpecType]int32{consumable3.SpecTypeHP: 50},
		Morphs: map[uint32]uint32{20: 0},
	})
	plan := computeEffectPlan(discardLogger(), character.NewModelBuilder().Build(), ci)
	assert.Empty(t, plan.statups)
	assert.Equal(t, []int16{50}, plan.hpChanges)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./consumable/ -run TestComputeEffectPlan -v
```
Expected: `TestComputeEffectPlan_FixedMorphWithHp`, `TestComputeEffectPlan_FixedMorphPrecedence` PASS already (fixed-morph branch moved in Task 3); `TestComputeEffectPlan_RandomMorphOnly` FAILs (`plan.statups` has length 0, not 1). `TestComputeEffectPlan_ZeroWeightMorphTableSkipsMorphOnly` passes vacuously. The red test that drives this task is T6.

- [ ] **Step 3: Implement the morph-random branch**

In `computeEffectPlan` (`consumable/processor.go`), replace the fixed-morph branch:

```go
	if val, ok := ci.GetSpec(consumable3.SpecTypeMorph); ok && val > 0 {
		plan.statups = append(plan.statups, stat.Model{Type: ts.TemporaryStatTypeMorph, Amount: val})
	}
```

with the precedence-structured branch (FR-7: fixed `morph` wins; the `else if` makes double-apply impossible):

```go
	if val, ok := ci.GetSpec(consumable3.SpecTypeMorph); ok && val > 0 {
		plan.statups = append(plan.statups, stat.Model{Type: ts.TemporaryStatTypeMorph, Amount: val})
	} else if len(ci.Morphs()) > 0 {
		if morphId, err := rollMorph(ci.Morphs()); err == nil {
			plan.statups = append(plan.statups, stat.Model{Type: ts.TemporaryStatTypeMorph, Amount: int32(morphId)})
		} else {
			l.WithError(err).Warnf("Skipping morph for item [%d]: unusable morphRandom table.", ci.Id())
		}
	}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test -race ./consumable/ -v
```
Expected: PASS — all `TestComputeEffectPlan_*` tests plus every pre-existing test.

- [ ] **Step 5: Commit**

```bash
git add consumable/processor.go consumable/processor_test.go
git commit -m "feat(consumables): apply morphRandom weighted table with fixed-morph precedence"
git branch --show-current   # must print task-140-morph-potion-routing
```

---

### Task 5: Route classification 221 through `ConsumeStandard` (FR-1, FR-2)

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/processor.go:72-83` (`usesStandardConsumer` + its doc comment)
- Test: `services/atlas-consumables/atlas.com/consumables/consumable/processor_test.go` (extend the existing `TestUsesStandardConsumer` table at ~line 321)

Routing lands LAST deliberately: every earlier commit leaves 221 items on the old `ConsumeBare` path, so no intermediate commit ships a half-wired standard consume for them.

**Interfaces:**
- Consumes: `item.ClassificationConsumableTransformation` (`libs/atlas-constants/item/constants.go:39`, already imported as `item2` in processor.go / `item` in the test file).
- Produces: 221 items flow `RequestItemConsume` → `ConsumeStandard` → `ApplyItemEffects` → morph statup via Tasks 3-4.

- [ ] **Step 1: Extend the routing test (failing)**

In `TestUsesStandardConsumer` (`consumable/processor_test.go`), add these rows to the `cases` slice, after the `"all cure potion (205)"` row:

```go
		{"morph potion (221)", item.Id(2210000), true},
		{"cliff's special potion — morphRandom (221)", item.Id(2211000), true},
		{"maplemas party potion (221; client intercepts 2212xxx before use-item, but classification routing is uniform)", item.Id(2212000), true},
```

The existing `{"equip scroll (204)", item.Id(2040727), false}` row already pins the "a non-standard classification still returns false" side.

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./consumable/ -run TestUsesStandardConsumer -v
```
Expected: FAIL — the three new 221 subtests report `want true, got false`.

- [ ] **Step 3: Implement the routing case**

In `consumable/processor.go`, update `usesStandardConsumer`:

```go
// usesStandardConsumer reports whether an item routes through ConsumeStandard
// (which invokes ApplyItemEffects for HP/MP recovery, status buffs, status
// cure, and morph). Anything not matched here falls through to ConsumeBare and
// silently skips effect application. Cure pots (classification 205) belong
// here because their disease cure flags are read inside ApplyItemEffects;
// transformation potions (classification 221) because their morph/morphRandom
// specs are applied there.
func usesStandardConsumer(itemId item2.Id) bool {
	switch item2.GetClassification(itemId) {
	case item2.Classification(200), item2.Classification(201), item2.Classification(202), item2.Classification(205),
		item2.ClassificationConsumableTransformation:
		return true
	}
	return false
}
```

(The 200/201/202/205 raw literals stay: no named constants exist for them and renaming is out of scope per design §2.)

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test -race ./consumable/ -v
```
Expected: PASS — all of `TestUsesStandardConsumer` including the three new rows, and every other test.

- [ ] **Step 5: Commit**

```bash
git add consumable/processor.go consumable/processor_test.go
git commit -m "feat(consumables): route transformation potions (221) through standard consumer"
git branch --show-current   # must print task-140-morph-potion-routing
```

---

### Task 6: Full verification, follow-up backlog filing, acceptance sweep

**Files:**
- Modify (main repo checkout, untracked research doc — NOT part of this branch's diff): `docs/research/missing-features/items-and-consumables.md`, reachable from the worktree root at `../../docs/research/missing-features/items-and-consumables.md`
- No code changes.

- [ ] **Step 1: Full module verification**

From `services/atlas-consumables/atlas.com/consumables/`:

```bash
go test -race ./...
go vet ./...
go build ./...
```
Expected: all three clean (exit 0, no output from vet/build, all tests PASS).

- [ ] **Step 2: redis-key-guard**

From the worktree root (no `GOWORK=off` prefix):

```bash
tools/redis-key-guard.sh
```
Expected: clean / PASS, exit 0.

- [ ] **Step 3: Diff-scope check (acceptance criterion)**

From the worktree root:

```bash
git diff --name-only $(git merge-base HEAD origin/main)..HEAD
```
Expected: every path starts with `services/atlas-consumables/` or `docs/tasks/task-140-morph-potion-routing/`. If `services/atlas-consumables/atlas.com/consumables/go.mod` appears (not expected — no new deps), `docker buildx bake atlas-consumables` from the worktree root becomes mandatory before proceeding.

- [ ] **Step 3b: Legacy-version support verification (read-only; all live versions on main)**

The legacy templates gms_48/61/72/79 landed on main 2026-07-13, after this plan froze; they are in scope. The atlas-consumables change is version-neutral (routing keys off classification; `computeEffectPlan`/`morph.go` carry no version literals), and the wire path already spans legacy — verified against main, NOT owned by this task:

- `CharacterItemUseHandle` is wired in every seed template incl. `template_gms_48_1.json` (the consume request reaches atlas-consumables on all versions).
- `TemporaryStatTypeMorph` is registered unconditionally at a version-stable bit in `libs/atlas-packet/model/character_temporary_stat.go`, with `legacyGmsMask` covering the pre-v61 8-byte mask.

So no code delta is required for the new versions. Two things are still unverified — check them read-only (no code/template/WZ change):

1. **Legacy WZ 221 data.** Query live atlas-data for a legacy tenant (per `reference_atlas_data_wz_inspection`): run a throwaway curl pod and hit `/api/data/consumables/{itemId}` for a candidate 221 id (e.g. `2210000`) with `REGION: GMS` / `MAJOR_VERSION: 72` (repeat for 48/61/79). Record for each version whether a classification-221 item with a `morph` or `morphRandom` spec exists. If present → feature applies automatically; if absent → record a documented no-op (feature intact, nothing to route on that version). Do NOT invent WZ values — report exactly what the REST call returns, or "unavailable" if no legacy tenant is reachable, and flag it as a blocker rather than assuming.
2. **Legacy MORPH round-trip.** For at least one legacy version whose data has a 221 item (prefer v72/79, plus one pre-v61 = v48/61 to exercise `legacyGmsMask`), confirm a consume emits a MORPH temporary stat the legacy client reads — via an integration/manual check or by pinning a legacy-tenant `CharacterTemporaryStat` MORPH encode fixture. The bit position is currently asserted only by a code comment; this is what backs the PRD §8 "no client-interpreted wire values" claim for legacy.

Record the per-version result (present / no-op / blocked) in the task folder. This step adds NO code to the branch diff and does not affect the "diff touches only atlas-consumables" acceptance criterion.

- [ ] **Step 4: File the 2212000 follow-up backlog item (acceptance criterion)**

The item-effects backlog research doc lives only in the main repo checkout (untracked), at `../../docs/research/missing-features/items-and-consumables.md` relative to the worktree root. Edit it as follows; do not commit it on this branch.

4a. In the §"3. Morph potions: effect supported, classification routing makes it dead code" entry (under Broken/partial, ~line 160), append this sentence to the end of the paragraph:

```
**Addressed by task-140** (221 routed through ConsumeStandard, morphRandom applied via weighted crypto/rand pick) — EXCEPT the 2212000 morph-other arm, see the follow-up note below.
```

4b. Directly below that paragraph, add:

```markdown
> **Follow-up (filed at task-140 close): 2212000 "Maplemas Party Potion" morph-other packet flow.**
> Not covered by task-140. IDA-verified (v83_Me, `MapleStory_dump.exe`, evidence in
> `docs/tasks/task-140-morph-potion-routing/prd.md` §1): `CDraggableItem::OnDoubleClicked` gates
> `id/10000 == 221 && (id%10000)/1000 == 2` (2212xxx) into a dedicated target-picker dialog
> (`CUIRandomMorphDlg` via `CWvsContext::SendRandomMorphOtherRequest`) with its own serverbound
> request and clientbound response (`CWvsContext::OnRandomMorphRes`, "failed to find user" /
> "only in town" failure arms). The client never sends the normal use-item packet for 2212xxx, so
> the standard consume path is unreachable. Needs its own packet-audit + implementation task:
> serverbound request handler, `OnRandomMorphRes` writer, town-only validation, target resolution,
> apply-morph-to-target (reuse task-140's `rollMorph` seam). Scope: feature-sized, per-version opcodes.
```

If the doc is missing or restructured, locate it with `grep -rln "Wholly-missing" ../../docs/` and place the note in the morph-potions entry found there; if it cannot be found at all, STOP and report blocked rather than inventing a new location.

- [ ] **Step 5: Acceptance-criteria sweep**

Walk PRD §10 and confirm each line, citing the evidence produced above:

1. Fixed-morph 221 item applies MORPH + time/1000 duration, item decrements — `TestComputeEffectPlan_FixedMorphWithHp` + routing via `TestUsesStandardConsumer` (decrement is the untouched `ConsumeStandard` commit path).
2. morphRandom applies exactly one table morph; weighting pinned — `TestComputeEffectPlan_RandomMorphOnly` + `TestSelectMorph_ExhaustiveWeighting` / `TestSelectMorph_WeightsNotSummingTo100`.
3. `hp` coexisting spec still applies — `TestComputeEffectPlan_FixedMorphWithHp` (hpChanges=[50]).
4. Fixed morph precedence — `TestComputeEffectPlan_FixedMorphPrecedence`.
5. 221 no longer routes to `ConsumeBare` — `TestUsesStandardConsumer` new rows.
6. Diff touches only atlas-consumables — Step 3 output.
7. test/vet/build/redis-key-guard clean — Steps 1-2 output.
8. Legacy-version support (gms_48/61/72/79) confirmed present-or-documented-no-op — Step 3b per-version record.
9. Follow-up backlog item filed — Step 4.

Record the sweep results (pass/fail per criterion with the evidence) in the task folder if any criterion needed manual judgment; otherwise the test names above are the record.

- [ ] **Step 6: Code review, then finish**

Per project rules, run `superpowers:requesting-code-review` (dispatches `plan-adherence-reviewer` + `backend-guidelines-reviewer`; findings land in `docs/tasks/task-140-morph-potion-routing/audit.md`) BEFORE opening a PR. Then proceed to `superpowers:finishing-a-development-branch`.

---

## Self-Review Record

- **Spec coverage:** FR-1/FR-2 → Task 5; FR-3 → Tasks 3-4 (T5); FR-4 → Task 1; FR-5/FR-6 → Task 2 (T2/T2b, exhaustive-roll determinism per design §3.2); FR-7 → Task 4 (T7); FR-8 is a verified non-requirement (no task; death cancellation pre-exists via respawn `CancelAllBuffs`). Design test plan T1→Task 5, T2/T3/T4→Task 2, T5/T6/T7→Task 4, T8→Task 3. All PRD §10 acceptance criteria mapped in Task 6 Step 5.
- **Version coverage:** the change is version-neutral by construction (no version literals in routing/`computeEffectPlan`/`morph.go`), so it needs no code delta for the legacy versions (gms_48/61/72/79) that landed on main 2026-07-13 after this plan froze. The pre-existing wire path (`CharacterItemUseHandle` in every template; unconditional `TemporaryStatTypeMorph` bit with `legacyGmsMask`) covers them. The only added obligation is the read-only legacy verification in Task 6 Step 3b (live atlas-data REST for legacy 221 data + a MORPH round-trip incl. the pre-v61 mask path); it adds no code to the diff and preserves the atlas-consumables-only acceptance criterion.
- **Placeholder scan:** no TBD/TODO markers; every code step shows complete code; the only conditional instruction (bake on go.mod change, backlog-doc relocation) states its exact trigger and action.
- **Type consistency:** `selectMorph(map[uint32]uint32, uint32) (uint32, bool)` and `rollMorph(map[uint32]uint32) (uint32, error)` match between Task 2 (definition) and Task 4 (use); `Morphs() map[uint32]uint32` matches between Task 1 and Task 4; `effectPlan` fields and `computeEffectPlan(l logrus.FieldLogger, c character.Model, ci consumable3.Model) effectPlan` match between Task 3 (definition) and Task 4-5 (tests); `stat.Model{Type character.TemporaryStatType; Amount int32}` matches the real `character/buff/stat/model.go`.
