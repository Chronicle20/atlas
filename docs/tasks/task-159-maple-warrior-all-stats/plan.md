# Maple Warrior All-Stats Bonus Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the MAPLE_WARRIOR buff grant `floor(baseStat × rate / 100)` to each of STR/DEX/INT/LUK in atlas-effective-stats, matching the client's BasicStat computation (base-stat-only basis, per-stat integer truncation), via a one-to-many bonus-constructing mapping API.

**Architecture:** Three-layer change inside one service (atlas-effective-stats): (1) `stat.Bonus` gains an additive `basePercent` dimension plus a dimension-preserving `WithSource` copier; (2) `ComputeEffectiveStats` gains a base-percent accumulator applied as flat with per-bonus integer truncation; (3) `MapBuffStatType` is replaced by `BonusesForBuffChange(source, buffType, amount) []Bonus` and both consumers (buff Kafka consumer, character initializer) migrate to it. No wire changes; JSON gains one additive field.

**Tech Stack:** Go 1.25, miniredis-backed registry tests, standard `testing` package. Module: `services/atlas-effective-stats/atlas.com/effective-stats` (module name `atlas-effective-stats`).

## Global Constraints

- Only atlas-effective-stats is touched. atlas-data, atlas-buffs, `MapStatupType`, HYPER_BODY semantics, and the equipment-snapshot bonus loop are all explicitly out of scope (PRD non-goals; design §5).
- `basePercent` is stored as the raw integer percent from the wire (e.g. `10` for +10%) — never pre-divided into a float (design §2.2).
- Base-percent application is Go integer division `base * pct / 100` per bonus, inside the accumulation loop — truncation per bonus, never on a summed rate (design §2.3).
- The basis for base-percent is `baseValues` (from `m.baseStats`) only — equipment and other flat bonuses must never leak into the basis (FR-7).
- `MapBuffStatType` is deleted, not aliased — the compiler must force every call site through the new API (FR-5).
- Immutable model pattern: private fields + getters + `With*` copiers. No `*_testhelpers.go` files (CLAUDE.md Test Helper Pattern).
- JSON: absent `basePercent` field decodes to `0` — backward compatible, no migration (FR-6).
- Unknown buff types: empty slice return, callers keep debug-log-and-skip (FR-4).
- All commands below run from the worktree root unless a `cd` is shown. The service module directory is `services/atlas-effective-stats/atlas.com/effective-stats`.
- Commit messages end with `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.

---

### Task 1: `Bonus.basePercent` dimension + `WithSource` + dimension-preserving re-sourcing

**Files:**
- Modify: `services/atlas-effective-stats/atlas.com/effective-stats/stat/model.go:46-128` (Bonus struct, constructors, JSON)
- Modify: `services/atlas-effective-stats/atlas.com/effective-stats/character/processor.go:200-229` (`AddBuffBonuses`, `AddPassiveBonuses`)
- Test: `services/atlas-effective-stats/atlas.com/effective-stats/stat/model_test.go`
- Test: `services/atlas-effective-stats/atlas.com/effective-stats/character/processor_test.go`

**Interfaces:**
- Consumes: existing `stat.Bonus`, `NewBonus`, `NewMultiplierBonus`, `NewFullBonus` (unchanged signatures).
- Produces: `func NewBasePercentBonus(source string, statType Type, percent int32) Bonus`; `func (b Bonus) BasePercent() int32`; `func (b Bonus) WithSource(source string) Bonus`. Later tasks (2, 3, 5) rely on exactly these names.

- [ ] **Step 1: Write the failing stat-package tests**

Append to `services/atlas-effective-stats/atlas.com/effective-stats/stat/model_test.go`. Also change the import block at the top of the file from `import ("testing")` to:

```go
import (
	"encoding/json"
	"testing"
)
```

Append:

```go
func TestNewBasePercentBonus(t *testing.T) {
	b := NewBasePercentBonus("buff:2311003", TypeStrength, 10)

	if b.Source() != "buff:2311003" {
		t.Errorf("Source() = %v, want buff:2311003", b.Source())
	}
	if b.StatType() != TypeStrength {
		t.Errorf("StatType() = %v, want %v", b.StatType(), TypeStrength)
	}
	if b.Amount() != 0 {
		t.Errorf("Amount() = %v, want 0", b.Amount())
	}
	if b.Multiplier() != 0.0 {
		t.Errorf("Multiplier() = %v, want 0.0", b.Multiplier())
	}
	if b.BasePercent() != 10 {
		t.Errorf("BasePercent() = %v, want 10", b.BasePercent())
	}
}

func TestBonusWithSource_PreservesDimensions(t *testing.T) {
	bp := NewBasePercentBonus("", TypeLuck, 10).WithSource("buff:2311003")
	if bp.Source() != "buff:2311003" {
		t.Errorf("Source() = %v, want buff:2311003", bp.Source())
	}
	if bp.BasePercent() != 10 {
		t.Errorf("BasePercent() = %v, want 10 (dimension dropped by WithSource)", bp.BasePercent())
	}
	if bp.StatType() != TypeLuck {
		t.Errorf("StatType() = %v, want %v", bp.StatType(), TypeLuck)
	}

	full := NewFullBonus("old", TypeStrength, 7, 0.5).WithSource("new")
	if full.Source() != "new" {
		t.Errorf("Source() = %v, want new", full.Source())
	}
	if full.Amount() != 7 || full.Multiplier() != 0.5 || full.BasePercent() != 0 {
		t.Errorf("WithSource altered dimensions: amount=%v multiplier=%v basePercent=%v, want 7/0.5/0",
			full.Amount(), full.Multiplier(), full.BasePercent())
	}
}

func TestBonusJSONRoundTrip_BasePercent(t *testing.T) {
	b := NewBasePercentBonus("buff:2311003", TypeIntelligence, 15)
	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var out Bonus
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if out != b {
		t.Errorf("round-trip = %+v, want %+v", out, b)
	}
}

func TestBonusUnmarshal_LegacyWithoutBasePercent(t *testing.T) {
	legacy := []byte(`{"source":"equipment:1","statType":"strength","amount":20,"multiplier":0}`)
	var b Bonus
	if err := json.Unmarshal(legacy, &b); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if b.BasePercent() != 0 {
		t.Errorf("BasePercent() = %v, want 0 for legacy JSON without the field", b.BasePercent())
	}
	if b.Source() != "equipment:1" || b.StatType() != TypeStrength || b.Amount() != 20 {
		t.Errorf("legacy fields corrupted: %+v", b)
	}
}
```

- [ ] **Step 2: Write the failing processor re-sourcing test**

Append to `services/atlas-effective-stats/atlas.com/effective-stats/character/processor_test.go`:

```go
func TestProcessor_AddBuffBonuses_PreservesBasePercent(t *testing.T) {
	p, _, ctx, _ := setupProcessorTest(t)

	ch := channel.NewModel(1, 2)
	in := []stat.Bonus{stat.NewBasePercentBonus("", stat.TypeStrength, 10)}
	if err := p.AddBuffBonuses(ch, 12345, 2311003, in); err != nil {
		t.Fatalf("AddBuffBonuses() error = %v", err)
	}

	m, err := GetRegistry().Get(ctx, 12345)
	if err != nil {
		t.Fatalf("Registry.Get() error = %v", err)
	}
	bonuses := m.Bonuses()
	if len(bonuses) != 1 {
		t.Fatalf("Bonuses count = %v, want 1", len(bonuses))
	}
	if bonuses[0].Source() != "buff:2311003" {
		t.Errorf("Source() = %v, want buff:2311003", bonuses[0].Source())
	}
	if bonuses[0].BasePercent() != 10 {
		t.Errorf("BasePercent() = %v, want 10 (re-sourcing dropped the dimension)", bonuses[0].BasePercent())
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
cd services/atlas-effective-stats/atlas.com/effective-stats && go test ./stat/ ./character/ -run 'TestNewBasePercentBonus|TestBonusWithSource|TestBonusJSONRoundTrip_BasePercent|TestBonusUnmarshal_Legacy|TestProcessor_AddBuffBonuses_PreservesBasePercent' -v
```

Expected: FAIL — compile errors `undefined: NewBasePercentBonus`, `b.BasePercent undefined`, `b.WithSource undefined`.

- [ ] **Step 4: Implement the Bonus extension in `stat/model.go`**

Replace the `Bonus` struct (currently lines 46-51) with:

```go
// Bonus represents a single contribution to a stat
type Bonus struct {
	source      string  // e.g., "equipment:12345", "passive:1000001", "buff:2311003"
	statType    Type    // which stat this bonus affects
	amount      int32   // flat bonus value (+20)
	multiplier  float64 // percentage bonus of (base + flat) (0.10 for +10%)
	basePercent int32   // percent of base stat only, applied as floor(base*pct/100) flat
}
```

After the `Multiplier()` getter (line 65-67), add:

```go
func (b Bonus) BasePercent() int32 {
	return b.basePercent
}
```

After `NewFullBonus` (line 90-97), add:

```go
// NewBasePercentBonus creates a stat bonus that grants floor(base*percent/100)
// as a flat addition, where the basis is the character's base stat only.
// The percent is the raw integer rate from the wire (10 = +10%).
func NewBasePercentBonus(source string, statType Type, percent int32) Bonus {
	return Bonus{
		source:      source,
		statType:    statType,
		basePercent: percent,
	}
}

// WithSource returns a copy of the bonus with the source replaced,
// preserving every bonus dimension.
func (b Bonus) WithSource(source string) Bonus {
	b.source = source
	return b
}
```

Replace `MarshalJSON` and `UnmarshalJSON` (lines 99-128) with:

```go
func (b Bonus) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Source      string  `json:"source"`
		StatType    Type    `json:"statType"`
		Amount      int32   `json:"amount"`
		Multiplier  float64 `json:"multiplier"`
		BasePercent int32   `json:"basePercent"`
	}{
		Source:      b.source,
		StatType:    b.statType,
		Amount:      b.amount,
		Multiplier:  b.multiplier,
		BasePercent: b.basePercent,
	})
}

func (b *Bonus) UnmarshalJSON(data []byte) error {
	var aux struct {
		Source      string  `json:"source"`
		StatType    Type    `json:"statType"`
		Amount      int32   `json:"amount"`
		Multiplier  float64 `json:"multiplier"`
		BasePercent int32   `json:"basePercent"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	b.source = aux.Source
	b.statType = aux.StatType
	b.amount = aux.Amount
	b.multiplier = aux.Multiplier
	b.basePercent = aux.BasePercent
	return nil
}
```

(`NewBonus`, `NewMultiplierBonus`, `NewFullBonus` are untouched — their struct literals leave `basePercent` at the zero value.)

- [ ] **Step 5: Implement dimension-preserving re-sourcing in `character/processor.go`**

In `AddBuffBonuses` (line 200-210), replace the re-sourcing loop body:

```go
	for _, b := range bonuses {
		sourcedBonuses = append(sourcedBonuses, b.WithSource(source))
	}
```

(was: `sourcedBonuses = append(sourcedBonuses, stat.NewFullBonus(source, b.StatType(), b.Amount(), b.Multiplier()))`)

In `AddPassiveBonuses` (line 218-229), make the identical replacement:

```go
	for _, b := range bonuses {
		sourcedBonuses = append(sourcedBonuses, b.WithSource(source))
	}
```

Do NOT change the equipment re-sourcing loop in `StoreEquipmentBonuses` (line 180-182) — equipment has no base-percent semantics and that loop's behavior guarantee is preserved deliberately (design §2.3/§2.4).

If `stat.NewFullBonus` is now unused in `processor.go`, the `stat` import is still needed (`stat.Bonus` in signatures) — no import churn.

- [ ] **Step 6: Run tests to verify they pass**

```bash
cd services/atlas-effective-stats/atlas.com/effective-stats && go test ./stat/ ./character/ -run 'TestNewBasePercentBonus|TestBonusWithSource|TestBonusJSONRoundTrip_BasePercent|TestBonusUnmarshal_Legacy|TestProcessor_AddBuffBonuses_PreservesBasePercent' -v
```

Expected: PASS (5 tests).

- [ ] **Step 7: Run the full module test suite to catch regressions**

```bash
cd services/atlas-effective-stats/atlas.com/effective-stats && go test ./...
```

Expected: all packages `ok`.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-effective-stats/atlas.com/effective-stats/stat/model.go services/atlas-effective-stats/atlas.com/effective-stats/stat/model_test.go services/atlas-effective-stats/atlas.com/effective-stats/character/processor.go services/atlas-effective-stats/atlas.com/effective-stats/character/processor_test.go
git commit -m "feat(task-159): add basePercent bonus dimension with dimension-preserving WithSource

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 2: Base-percent application in `ComputeEffectiveStats`

**Files:**
- Modify: `services/atlas-effective-stats/atlas.com/effective-stats/character/model.go:269-345` (`ComputeEffectiveStats`)
- Test: `services/atlas-effective-stats/atlas.com/effective-stats/character/model_test.go`

**Interfaces:**
- Consumes: `stat.NewBasePercentBonus(source, statType, percent)`, `Bonus.BasePercent() int32` (Task 1).
- Produces: engine behavior `effective = floor((base + flat + Σᵢ floor(base × pctᵢ / 100)) × (1 + mult))`. Task 5's lifecycle test relies on this.

- [ ] **Step 1: Write the failing engine tests**

Append to `services/atlas-effective-stats/atlas.com/effective-stats/character/model_test.go`:

```go
func TestModelComputeEffectiveStats_BasePercentTruncation(t *testing.T) {
	// FR-13: base STR 13 with MW 10% -> floor(13*10/100) = 1 -> 14.
	// Base DEX 4 with MW 10% -> floor(4*10/100) = 0 -> 4.
	ten := createTestTenant()
	ch := channel.NewModel(1, 2)
	m := NewModel(ten, ch, 12345)

	base := stat.NewBase(13, 4, 30, 25, 5000, 3000)
	bStr := stat.NewBasePercentBonus("buff:2311003", stat.TypeStrength, 10)
	bDex := stat.NewBasePercentBonus("buff:2311003", stat.TypeDexterity, 10)
	m = m.WithBaseStats(base).WithBonus(bStr).WithBonus(bDex)

	computed := m.ComputeEffectiveStats(nil)

	if computed.Strength() != 14 {
		t.Errorf("Strength() = %v, want 14", computed.Strength())
	}
	if computed.Dexterity() != 4 {
		t.Errorf("Dexterity() = %v, want 4", computed.Dexterity())
	}
}

func TestModelComputeEffectiveStats_BasePercentExcludesEquipment(t *testing.T) {
	// FR-13: base 100 + 30 equip STR + MW 10% -> 100 + 30 + floor(100*10/100) = 140, NOT 143.
	ten := createTestTenant()
	ch := channel.NewModel(1, 2)
	m := NewModel(ten, ch, 12345)

	base := stat.NewBase(100, 40, 30, 25, 5000, 3000)
	snap := NewEquippedAsset(1, 1052095, []stat.Bonus{stat.NewBonus("equipment:1", stat.TypeStrength, 30)})
	mw := stat.NewBasePercentBonus("buff:2311003", stat.TypeStrength, 10)
	m = m.WithBaseStats(base).WithEquippedAsset(snap).WithBonus(mw)

	computed := m.ComputeEffectiveStats(map[uint32]bool{1: true})

	if computed.Strength() != 140 {
		t.Errorf("Strength() = %v, want 140 (equipment leaked into base-percent basis if 143)", computed.Strength())
	}
}

func TestModelComputeEffectiveStats_BasePercentIndependentTruncation(t *testing.T) {
	// Two 10% base-percent bonuses on base 15 truncate independently:
	// floor(1.5) + floor(1.5) = 2 -> 17, NOT floor(15*20/100) = 3 -> 18.
	// Distinct sources: WithBonus replaces on same (source, statType).
	ten := createTestTenant()
	ch := channel.NewModel(1, 2)
	m := NewModel(ten, ch, 12345)

	base := stat.NewBase(15, 40, 30, 25, 5000, 3000)
	b1 := stat.NewBasePercentBonus("buff:1", stat.TypeStrength, 10)
	b2 := stat.NewBasePercentBonus("buff:2", stat.TypeStrength, 10)
	m = m.WithBaseStats(base).WithBonus(b1).WithBonus(b2)

	computed := m.ComputeEffectiveStats(nil)

	if computed.Strength() != 17 {
		t.Errorf("Strength() = %v, want 17 (independent truncation)", computed.Strength())
	}
}

func TestModelComputeEffectiveStats_BasePercentWithMultiplier(t *testing.T) {
	// Ordering pinned: floor((base + flat + bp) * (1 + mult)).
	// base 100, bp 10% (-> +10), mult 0.10 -> floor(110 * 1.10) = 121.
	ten := createTestTenant()
	ch := channel.NewModel(1, 2)
	m := NewModel(ten, ch, 12345)

	base := stat.NewBase(100, 40, 30, 25, 5000, 3000)
	bp := stat.NewBasePercentBonus("buff:mw", stat.TypeStrength, 10)
	mult := stat.NewMultiplierBonus("buff:other", stat.TypeStrength, 0.10)
	m = m.WithBaseStats(base).WithBonus(bp).WithBonus(mult)

	computed := m.ComputeEffectiveStats(nil)

	if computed.Strength() != 121 {
		t.Errorf("Strength() = %v, want 121", computed.Strength())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd services/atlas-effective-stats/atlas.com/effective-stats && go test ./character/ -run 'TestModelComputeEffectiveStats_BasePercent' -v
```

Expected: FAIL — 4 tests, wrong computed values (e.g. `Strength() = 13, want 14`) because `ComputeEffectiveStats` ignores `BasePercent()`.

- [ ] **Step 3: Implement the base-percent accumulator**

In `services/atlas-effective-stats/atlas.com/effective-stats/character/model.go`, `ComputeEffectiveStats`:

Replace the accumulator init + non-equipment bonus loop (currently lines 287-298):

```go
	flatBonuses := make(map[stat.Type]int32)
	multipliers := make(map[stat.Type]float64)
	basePercentFlat := make(map[stat.Type]int32)
	for _, statType := range stat.AllTypes() {
		flatBonuses[statType] = 0
		multipliers[statType] = 0.0
		basePercentFlat[statType] = 0
	}

	// Non-equipment bonuses contribute flat, multiplier, and base-percent values.
	// Base-percent truncates per bonus (Go integer division) against the base
	// stat only — equipment and other flat bonuses never enter the basis.
	for _, b := range m.bonuses {
		flatBonuses[b.StatType()] += b.Amount()
		multipliers[b.StatType()] += b.Multiplier()
		if b.BasePercent() != 0 {
			basePercentFlat[b.StatType()] += baseValues[b.StatType()] * b.BasePercent() / 100
		}
	}
```

In `computeEffective` (currently line 311-327), replace the effective line:

```go
		effective := float64(base+flat+basePercentFlat[statType]) * (1.0 + mult)
```

(was `effective := float64(base+flat) * (1.0 + mult)`). The equipment loop and everything else in the function stay untouched.

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd services/atlas-effective-stats/atlas.com/effective-stats && go test ./character/ -run 'TestModelComputeEffectiveStats' -v
```

Expected: PASS — the 4 new tests plus all pre-existing `TestModelComputeEffectiveStats_*` tests (flat/multiplier/mixed/HP-cap behavior unchanged).

- [ ] **Step 5: Run the full module test suite**

```bash
cd services/atlas-effective-stats/atlas.com/effective-stats && go test ./...
```

Expected: all packages `ok`.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-effective-stats/atlas.com/effective-stats/character/model.go services/atlas-effective-stats/atlas.com/effective-stats/character/model_test.go
git commit -m "feat(task-159): apply base-percent bonuses in ComputeEffectiveStats with per-bonus truncation

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 3: `BonusesForBuffChange` mapping API

**Files:**
- Modify: `services/atlas-effective-stats/atlas.com/effective-stats/stat/model.go` (add function; `MapBuffStatType` stays until Task 4)
- Test: `services/atlas-effective-stats/atlas.com/effective-stats/stat/model_test.go`

**Interfaces:**
- Consumes: `NewBonus`, `NewMultiplierBonus`, `NewBasePercentBonus` (Task 1).
- Produces: `func BonusesForBuffChange(source string, buffType string, amount int32) []Bonus`. Tasks 4 and 5 call it with exactly this signature.

- [ ] **Step 1: Write the failing mapping tests**

Append to `services/atlas-effective-stats/atlas.com/effective-stats/stat/model_test.go`:

```go
func TestBonusesForBuffChange_Flat(t *testing.T) {
	// Table ported row-for-row from the old TestMapBuffStatType flat rows.
	tests := []struct {
		input        string
		expectedType Type
	}{
		{"WEAPON_ATTACK", TypeWeaponAttack},
		{"PAD", TypeWeaponAttack},
		{"MAGIC_ATTACK", TypeMagicAttack},
		{"MAD", TypeMagicAttack},
		{"WEAPON_DEFENSE", TypeWeaponDefense},
		{"PDD", TypeWeaponDefense},
		{"MAGIC_DEFENSE", TypeMagicDefense},
		{"MDD", TypeMagicDefense},
		{"ACCURACY", TypeAccuracy},
		{"ACC", TypeAccuracy},
		{"AVOIDABILITY", TypeAvoidability},
		{"AVOID", TypeAvoidability},
		{"EVA", TypeAvoidability},
		{"SPEED", TypeSpeed},
		{"JUMP", TypeJump},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			bs := BonusesForBuffChange("buff:1", tt.input, 20)
			if len(bs) != 1 {
				t.Fatalf("len = %v, want 1", len(bs))
			}
			b := bs[0]
			if b.StatType() != tt.expectedType {
				t.Errorf("StatType() = %v, want %v", b.StatType(), tt.expectedType)
			}
			if b.Amount() != 20 {
				t.Errorf("Amount() = %v, want 20", b.Amount())
			}
			if b.Multiplier() != 0.0 || b.BasePercent() != 0 {
				t.Errorf("kind leaked: multiplier=%v basePercent=%v, want 0/0", b.Multiplier(), b.BasePercent())
			}
			if b.Source() != "buff:1" {
				t.Errorf("Source() = %v, want buff:1", b.Source())
			}
		})
	}
}

func TestBonusesForBuffChange_HyperBody(t *testing.T) {
	tests := []struct {
		input        string
		expectedType Type
	}{
		{"HYPER_BODY_HP", TypeMaxHp},
		{"HYPER_BODY_MP", TypeMaxMp},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			bs := BonusesForBuffChange("buff:1", tt.input, 60)
			if len(bs) != 1 {
				t.Fatalf("len = %v, want 1", len(bs))
			}
			b := bs[0]
			if b.StatType() != tt.expectedType {
				t.Errorf("StatType() = %v, want %v", b.StatType(), tt.expectedType)
			}
			if b.Multiplier() != 0.60 {
				t.Errorf("Multiplier() = %v, want 0.60", b.Multiplier())
			}
			if b.Amount() != 0 || b.BasePercent() != 0 {
				t.Errorf("kind leaked: amount=%v basePercent=%v, want 0/0", b.Amount(), b.BasePercent())
			}
		})
	}
}

func TestBonusesForBuffChange_MapleWarrior(t *testing.T) {
	bs := BonusesForBuffChange("buff:2311003", "MAPLE_WARRIOR", 10)
	if len(bs) != 4 {
		t.Fatalf("len = %v, want 4", len(bs))
	}

	got := make(map[Type]Bonus, 4)
	for _, b := range bs {
		got[b.StatType()] = b
	}
	for _, want := range []Type{TypeStrength, TypeDexterity, TypeIntelligence, TypeLuck} {
		b, ok := got[want]
		if !ok {
			t.Errorf("missing base-percent bonus for %v", want)
			continue
		}
		if b.BasePercent() != 10 {
			t.Errorf("%v BasePercent() = %v, want 10", want, b.BasePercent())
		}
		if b.Amount() != 0 || b.Multiplier() != 0.0 {
			t.Errorf("%v kind leaked: amount=%v multiplier=%v, want 0/0", want, b.Amount(), b.Multiplier())
		}
		if b.Source() != "buff:2311003" {
			t.Errorf("%v Source() = %v, want buff:2311003", want, b.Source())
		}
	}
}

func TestBonusesForBuffChange_Unknown(t *testing.T) {
	bs := BonusesForBuffChange("buff:1", "UNKNOWN", 20)
	if len(bs) != 0 {
		t.Errorf("len = %v, want 0", len(bs))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd services/atlas-effective-stats/atlas.com/effective-stats && go test ./stat/ -run 'TestBonusesForBuffChange' -v
```

Expected: FAIL — compile error `undefined: BonusesForBuffChange`.

- [ ] **Step 3: Implement `BonusesForBuffChange`**

In `services/atlas-effective-stats/atlas.com/effective-stats/stat/model.go`, insert directly above `MapBuffStatType` (line 408):

```go
// BonusesForBuffChange converts one buff stat change into the stat bonuses
// it grants. Returns an empty slice for unknown buff types.
//
// One buff change can affect several stats (MAPLE_WARRIOR grants a
// base-percent bonus to all four primary stats); the amount-to-bonus
// conversion lives here so the live-apply and initializer paths cannot
// drift.
func BonusesForBuffChange(source string, buffType string, amount int32) []Bonus {
	switch buffType {
	case "WEAPON_ATTACK", "PAD":
		return []Bonus{NewBonus(source, TypeWeaponAttack, amount)}
	case "MAGIC_ATTACK", "MAD":
		return []Bonus{NewBonus(source, TypeMagicAttack, amount)}
	case "WEAPON_DEFENSE", "PDD":
		return []Bonus{NewBonus(source, TypeWeaponDefense, amount)}
	case "MAGIC_DEFENSE", "MDD":
		return []Bonus{NewBonus(source, TypeMagicDefense, amount)}
	case "ACCURACY", "ACC":
		return []Bonus{NewBonus(source, TypeAccuracy, amount)}
	case "AVOIDABILITY", "AVOID", "EVA":
		return []Bonus{NewBonus(source, TypeAvoidability, amount)}
	case "SPEED":
		return []Bonus{NewBonus(source, TypeSpeed, amount)}
	case "JUMP":
		return []Bonus{NewBonus(source, TypeJump, amount)}
	case "HYPER_BODY_HP":
		return []Bonus{NewMultiplierBonus(source, TypeMaxHp, float64(amount)/100.0)}
	case "HYPER_BODY_MP":
		return []Bonus{NewMultiplierBonus(source, TypeMaxMp, float64(amount)/100.0)}
	case "MAPLE_WARRIOR":
		// Client applies rate% of the RAW base stat (never base+equip) to each
		// primary stat, truncating per stat (IDA-verified v83 BasicStat::SetFrom
		// @0x77ec9f, v95 @0x732ba0 — see PRD §4.1).
		return []Bonus{
			NewBasePercentBonus(source, TypeStrength, amount),
			NewBasePercentBonus(source, TypeDexterity, amount),
			NewBasePercentBonus(source, TypeIntelligence, amount),
			NewBasePercentBonus(source, TypeLuck, amount),
		}
	default:
		return []Bonus{}
	}
}
```

`MapBuffStatType` stays in place for this task — it still has two call sites; deleting it here would break the build. Task 4 removes it.

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd services/atlas-effective-stats/atlas.com/effective-stats && go test ./stat/ -v
```

Expected: PASS — all stat-package tests including the 4 new `TestBonusesForBuffChange_*` and the still-present `TestMapBuffStatType`.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-effective-stats/atlas.com/effective-stats/stat/model.go services/atlas-effective-stats/atlas.com/effective-stats/stat/model_test.go
git commit -m "feat(task-159): add one-to-many BonusesForBuffChange mapping API

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 4: Migrate both consumers, delete `MapBuffStatType`, update docs

**Files:**
- Modify: `services/atlas-effective-stats/atlas.com/effective-stats/kafka/consumer/buff/consumer.go:50-67`
- Modify: `services/atlas-effective-stats/atlas.com/effective-stats/character/initializer.go:182-198`
- Modify: `services/atlas-effective-stats/atlas.com/effective-stats/stat/model.go` (delete `MapBuffStatType`, lines 408-440)
- Modify: `services/atlas-effective-stats/atlas.com/effective-stats/stat/model_test.go` (delete `TestMapBuffStatType`)
- Modify: `services/atlas-effective-stats/atlas.com/effective-stats/docs/domain.md:85`

**Interfaces:**
- Consumes: `stat.BonusesForBuffChange(source, buffType, amount) []Bonus` (Task 3); `AddBuffBonuses` re-stamps source via `WithSource` (Task 1).
- Produces: nothing new — after this task `MapBuffStatType` no longer exists anywhere (FR-5).

- [ ] **Step 1: Migrate the buff Kafka consumer**

In `services/atlas-effective-stats/atlas.com/effective-stats/kafka/consumer/buff/consumer.go`, `handleBuffApplied`, replace the conversion loop (lines 50-67):

```go
	// Convert buff stat changes to stat bonuses. Source stays "" here —
	// AddBuffBonuses stamps "buff:<sourceId>" via WithSource.
	bonuses := make([]stat.Bonus, 0)
	for _, change := range e.Body.Changes {
		bs := stat.BonusesForBuffChange("", change.Type, change.Amount)
		if len(bs) == 0 {
			l.Debugf("Unknown buff stat type: %s", change.Type)
			continue
		}
		bonuses = append(bonuses, bs...)
	}
```

The `if len(bonuses) > 0 { ... AddBuffBonuses ... }` block below stays unchanged.

- [ ] **Step 2: Migrate the initializer**

In `services/atlas-effective-stats/atlas.com/effective-stats/character/initializer.go`, `fetchBuffBonuses`, replace the inner loop (lines 182-198):

```go
	bonuses := make([]stat.Bonus, 0)
	for _, buff := range buffList {
		source := fmt.Sprintf("buff:%d", buff.SourceId)
		for _, change := range buff.Changes {
			bs := stat.BonusesForBuffChange(source, change.Type, change.Amount)
			if len(bs) == 0 {
				l.Debugf("Unknown buff stat type: %s", change.Type)
				continue
			}
			bonuses = append(bonuses, bs...)
		}
	}
```

- [ ] **Step 3: Delete `MapBuffStatType` and its test**

- In `services/atlas-effective-stats/atlas.com/effective-stats/stat/model.go`, delete the whole `MapBuffStatType` function including its doc comment (lines 408-440 pre-Task-3; sits directly below the new `BonusesForBuffChange`). Do NOT touch `MapStatupType`.
- In `services/atlas-effective-stats/atlas.com/effective-stats/stat/model_test.go`, delete the whole `TestMapBuffStatType` function (its table now lives row-for-row in `TestBonusesForBuffChange_Flat`/`_HyperBody`; the `MAPLE_WARRIOR` row is superseded by `TestBonusesForBuffChange_MapleWarrior`, and the unknown row by `_Unknown`). Do NOT touch `TestMapStatupType`.

- [ ] **Step 4: Verify zero remaining references**

```bash
cd services/atlas-effective-stats/atlas.com/effective-stats && grep -rn "MapBuffStatType" . ; echo "exit: $?"
```

Expected: one hit in `docs/domain.md` only (fixed next step), then after Step 5 re-run expects `exit: 1` (no matches).

- [ ] **Step 5: Update the service domain doc**

In `services/atlas-effective-stats/atlas.com/effective-stats/docs/domain.md`, replace line 85:

> The `MapBuffStatType` function maps buff stat type strings to stat types. Most buff types map to flat bonuses. `HYPER_BODY_HP`, `HYPER_BODY_MP`, and `MAPLE_WARRIOR` map to multiplier bonuses.

with:

> The `BonusesForBuffChange` function converts one buff stat change into the stat bonuses it grants (one-to-many). Most buff types yield a single flat bonus. `HYPER_BODY_HP` and `HYPER_BODY_MP` yield multiplier bonuses on `(base + flat)`. `MAPLE_WARRIOR` yields four base-percent bonuses (strength, dexterity, intelligence, luck), each applied as `floor(base × percent / 100)` added flat — the basis is the raw base stat only, never equipment. Unknown buff types yield no bonuses.

Then re-run the Step 4 grep; expected: no matches, `exit: 1`.

- [ ] **Step 6: Build and run the full module test suite**

```bash
cd services/atlas-effective-stats/atlas.com/effective-stats && go build ./... && go test ./...
```

Expected: build clean (proves both call sites migrated — FR-5), all packages `ok`.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-effective-stats/atlas.com/effective-stats/kafka/consumer/buff/consumer.go services/atlas-effective-stats/atlas.com/effective-stats/character/initializer.go services/atlas-effective-stats/atlas.com/effective-stats/stat/model.go services/atlas-effective-stats/atlas.com/effective-stats/stat/model_test.go services/atlas-effective-stats/atlas.com/effective-stats/docs/domain.md
git commit -m "feat(task-159): migrate buff consumer and initializer to BonusesForBuffChange, drop MapBuffStatType

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 5: Lifecycle and path-parity tests

**Files:**
- Test: `services/atlas-effective-stats/atlas.com/effective-stats/character/processor_test.go`

**Interfaces:**
- Consumes: `stat.BonusesForBuffChange` (Task 3), `AddBuffBonuses`/`RemoveBuffBonuses`/`SetBaseStats` (existing processor), engine behavior (Task 2). No new production code — these tests must pass against Tasks 1-4 as landed; a failure here is a bug in an earlier task, not a reason to change these tests.

- [ ] **Step 1: Write the lifecycle test (FR-14)**

Append to `services/atlas-effective-stats/atlas.com/effective-stats/character/processor_test.go`:

```go
func TestProcessor_MapleWarriorLifecycle(t *testing.T) {
	p, _, ctx, _ := setupProcessorTest(t)

	ch := channel.NewModel(1, 2)
	// NewBase(strength, dexterity, luck, intelligence, maxHp, maxMp)
	base := stat.NewBase(100, 80, 60, 40, 5000, 3000)
	if err := p.SetBaseStats(ch, 12345, base); err != nil {
		t.Fatalf("SetBaseStats() error = %v", err)
	}

	// Apply: one MAPLE_WARRIOR change -> four base-percent bonuses.
	bs := stat.BonusesForBuffChange("", "MAPLE_WARRIOR", 10)
	if err := p.AddBuffBonuses(ch, 12345, 2311003, bs); err != nil {
		t.Fatalf("AddBuffBonuses() error = %v", err)
	}

	m, err := GetRegistry().Get(ctx, 12345)
	if err != nil {
		t.Fatalf("Registry.Get() error = %v", err)
	}
	if m.Computed().Strength() != 110 {
		t.Errorf("Strength = %v, want 110", m.Computed().Strength())
	}
	if m.Computed().Dexterity() != 88 {
		t.Errorf("Dexterity = %v, want 88", m.Computed().Dexterity())
	}
	if m.Computed().Luck() != 66 {
		t.Errorf("Luck = %v, want 66", m.Computed().Luck())
	}
	if m.Computed().Intelligence() != 44 {
		t.Errorf("Intelligence = %v, want 44", m.Computed().Intelligence())
	}
	if m.Computed().MaxHp() != 5000 {
		t.Errorf("MaxHp = %v, want 5000 (MW must not touch HP)", m.Computed().MaxHp())
	}

	// Expire: removal by source drops all four together.
	if err := p.RemoveBuffBonuses(12345, 2311003); err != nil {
		t.Fatalf("RemoveBuffBonuses() error = %v", err)
	}
	m, err = GetRegistry().Get(ctx, 12345)
	if err != nil {
		t.Fatalf("Registry.Get() error = %v", err)
	}
	if m.Computed().Strength() != 100 || m.Computed().Dexterity() != 80 ||
		m.Computed().Luck() != 60 || m.Computed().Intelligence() != 40 {
		t.Errorf("post-expiry stats = STR %v/DEX %v/LUK %v/INT %v, want 100/80/60/40",
			m.Computed().Strength(), m.Computed().Dexterity(), m.Computed().Luck(), m.Computed().Intelligence())
	}
	if len(m.Bonuses()) != 0 {
		t.Errorf("Bonuses count = %v, want 0 after expiry", len(m.Bonuses()))
	}
}
```

- [ ] **Step 2: Write the path-parity test (FR-11)**

Append to `services/atlas-effective-stats/atlas.com/effective-stats/character/processor_test.go`:

```go
func TestProcessor_MapleWarrior_PathParity(t *testing.T) {
	p, _, ctx, ten := setupProcessorTest(t)
	ch := channel.NewModel(1, 2)

	// Live-apply path: consumer builds with empty source, AddBuffBonuses
	// re-stamps "buff:2311003" via WithSource.
	consumerBonuses := stat.BonusesForBuffChange("", "MAPLE_WARRIOR", 10)
	if err := p.AddBuffBonuses(ch, 12345, 2311003, consumerBonuses); err != nil {
		t.Fatalf("AddBuffBonuses() error = %v", err)
	}
	live, err := GetRegistry().Get(ctx, 12345)
	if err != nil {
		t.Fatalf("Registry.Get() error = %v", err)
	}

	// Initializer path: source pre-stamped, applied via WithBonuses.
	initBonuses := stat.BonusesForBuffChange("buff:2311003", "MAPLE_WARRIOR", 10)
	initModel := NewModel(ten, ch, 67890).WithBonuses(initBonuses)

	key := func(b stat.Bonus) string { return b.Source() + "|" + string(b.StatType()) }
	liveSet := make(map[string]stat.Bonus)
	for _, b := range live.Bonuses() {
		liveSet[key(b)] = b
	}
	initSet := make(map[string]stat.Bonus)
	for _, b := range initModel.Bonuses() {
		initSet[key(b)] = b
	}

	if len(liveSet) != 4 {
		t.Fatalf("live path bonuses = %v, want 4", len(liveSet))
	}
	if len(initSet) != 4 {
		t.Fatalf("initializer path bonuses = %v, want 4", len(initSet))
	}
	for k, lb := range liveSet {
		ib, ok := initSet[k]
		if !ok {
			t.Errorf("initializer path missing bonus %s", k)
			continue
		}
		if lb != ib {
			t.Errorf("bonus %s differs: live=%+v init=%+v", k, lb, ib)
		}
	}
}
```

(`stat.Bonus` has only comparable fields, so `lb != ib` compares every dimension including `basePercent`.)

- [ ] **Step 3: Run the new tests**

```bash
cd services/atlas-effective-stats/atlas.com/effective-stats && go test ./character/ -run 'TestProcessor_MapleWarrior' -v
```

Expected: PASS (2 tests). If either fails, the defect is in Tasks 1-4 — fix there, don't bend the assertions.

- [ ] **Step 4: Run the full module test suite with race detector**

```bash
cd services/atlas-effective-stats/atlas.com/effective-stats && go test -race ./...
```

Expected: all packages `ok`.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-effective-stats/atlas.com/effective-stats/character/processor_test.go
git commit -m "test(task-159): Maple Warrior lifecycle and live-apply/initializer parity coverage

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 6: Full verification sweep

**Files:** none created/modified — verification only (CLAUDE.md Build & Verification + PRD acceptance criteria).

**Interfaces:**
- Consumes: everything landed in Tasks 1-5.
- Produces: the evidence trail required before code review / PR.

- [ ] **Step 1: Module-level checks**

```bash
cd services/atlas-effective-stats/atlas.com/effective-stats && go test -race ./... && go vet ./... && go build ./...
```

Expected: every package `ok`, vet silent, build silent.

- [ ] **Step 2: Docker bake (mandatory — catches Dockerfile COPY gaps `go build` cannot)**

From the worktree root:

```bash
docker buildx bake atlas-effective-stats
```

Expected: image builds successfully, exit 0.

- [ ] **Step 3: Redis key guard**

From the worktree root:

```bash
tools/redis-key-guard.sh
```

Expected: clean (no keyed Redis commands on the raw client outside libs/atlas-redis), exit 0.

- [ ] **Step 4: Acceptance-criteria spot audit**

Confirm each PRD §10 criterion against the landed code (no commit; report findings):

- `MAPLE_WARRIOR` → exactly 4 base-percent bonuses; no other mapping changed; unknown → nothing (`TestBonusesForBuffChange_*`).
- `effective = base + equip + floor(base × X / 100)` per primary stat, equipment excluded from basis (`TestModelComputeEffectiveStats_BasePercent*`).
- Expiry removes all four; live-apply ≡ initializer bonus sets (`TestProcessor_MapleWarrior*`).
- `grep -rn "MapBuffStatType" services/atlas-effective-stats/` → no matches.

- [ ] **Step 5: Code review**

Run `superpowers:requesting-code-review` (dispatches backend-guidelines-reviewer + plan-adherence-reviewer) before opening a PR — required by CLAUDE.md; findings go to `docs/tasks/task-159-maple-warrior-all-stats/audit.md`.
