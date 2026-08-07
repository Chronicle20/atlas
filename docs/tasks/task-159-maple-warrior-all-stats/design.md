# Maple Warrior All-Stats Bonus — Design

Task: task-159-maple-warrior-all-stats
Status: Approved PRD → design phase
PRD: `docs/tasks/task-159-maple-warrior-all-stats/prd.md`

## 1. Problem Recap

`stat.MapBuffStatType` (`stat/model.go:410`) is a one-to-one mapping
(`buffType → (Type, isMultiplier)`). `MAPLE_WARRIOR` needs a one-to-many
mapping (four stats) with a third bonus semantic the engine doesn't have:
**base-percent** — `floor(baseStat × rate / 100)` added flat, where the basis
is the character's raw base stat only, never `(base + equipment)`
(IDA-verified at v83 and v95, PRD §4.1).

Three things change, all inside atlas-effective-stats:

1. The mapping API (`stat` package) — one-to-many, kind-aware.
2. The `Bonus` model + `ComputeEffectiveStats` — a base-percent dimension.
3. The two consumers of the old API — buff Kafka consumer and character
   initializer.

A fourth change surfaced during design exploration (not in the PRD but
required for correctness): the processor's re-sourcing helpers
(`AddBuffBonuses` / `AddPassiveBonuses`, `character/processor.go:200-229`)
rebuild each bonus via `stat.NewFullBonus(source, statType, amount,
multiplier)`. Any new `Bonus` dimension is silently dropped at that seam
unless re-sourcing becomes dimension-preserving. See §4.4.

## 2. Design Decisions

### 2.1 Mapping API shape — bonus-constructing, not mapping-entry-returning

**Chosen:** replace `MapBuffStatType` with a single function that goes all
the way from a buff `StatChange` to finished bonuses:

```go
// BonusesForBuffChange converts one buff stat change into the stat bonuses
// it grants. Returns an empty slice for unknown buff types.
func BonusesForBuffChange(source string, buffType string, amount int32) []Bonus
```

Behavior table (the internal mapping is a switch, same as today):

| buffType | result |
|---|---|
| `WEAPON_ATTACK`/`PAD`, `MAGIC_ATTACK`/`MAD`, `WEAPON_DEFENSE`/`PDD`, `MAGIC_DEFENSE`/`MDD`, `ACCURACY`/`ACC`, `AVOIDABILITY`/`AVOID`/`EVA`, `SPEED`, `JUMP` | one flat bonus: `NewBonus(source, type, amount)` |
| `HYPER_BODY_HP` | one multiplier bonus: `NewMultiplierBonus(source, TypeMaxHp, float64(amount)/100)` |
| `HYPER_BODY_MP` | same, `TypeMaxMp` |
| `MAPLE_WARRIOR` | four base-percent bonuses: `NewBasePercentBonus(source, T, amount)` for `TypeStrength`, `TypeDexterity`, `TypeIntelligence`, `TypeLuck` |
| anything else | `[]Bonus{}` (callers keep debug-log-and-skip on empty) |

**Why this shape over the two alternatives considered:**

- **Alt A — mapping-entries API**: `MapBuffStatTypes(buffType) []Mapping`
  where `Mapping{StatType Type, Kind BonusKind}` and callers switch on
  `Kind` to build the bonus. This satisfies FR-1 literally, but leaves the
  amount→bonus conversion (`float64(amount)/100` for multipliers, etc.)
  duplicated in both consumers — exactly the duplication that lets the two
  paths drift. FR-11 (live-apply and initializer must produce byte-identical
  bonus sets) is best guaranteed by *construction*, not by test alone: with
  `BonusesForBuffChange` both call sites are one-liners over the same code.
- **Alt B — kind-discriminated Bonus rebuild**: replace
  `amount`/`multiplier` with a `kind` enum + single `value`. Conceptually
  cleanest, but it breaks every existing constructor, the JSON wire shape of
  the REST bonus list, and the equipped-asset snapshot serialization — far
  more churn than the PRD's additive-field intent (FR-6 explicitly allows a
  field *or* discriminator; the additive field is strictly smaller).

`MapBuffStatType` is deleted (FR-5). `MapStatupType` is untouched — it is a
different producer domain (passive statups) and every entry there is flat.

The three-kinds requirement (FR-1) remains expressible: flat →
`Amount() != 0`, percent → `Multiplier() != 0`, base-percent →
`BasePercent() != 0`. Nothing prevents a future buff type from returning a
mix of kinds across several stats.

### 2.2 `Bonus` extension — additive `basePercent` field

```go
type Bonus struct {
    source      string
    statType    Type
    amount      int32   // flat
    multiplier  float64 // percent of (base + flat)
    basePercent int32   // percent of base stat only, applied as floor(base*pct/100) flat
}

func NewBasePercentBonus(source string, statType Type, percent int32) Bonus
func (b Bonus) BasePercent() int32
```

- The rate is stored as the raw integer percent from the wire (`X` from the
  skill data, e.g. `10` for +10%) — **not** pre-divided into a float. Keeping
  it integral is what lets the engine reproduce the client's integer
  division exactly; a float rate would invite `base*0.1`-style
  floating-point drift on the exact values the tests pin (FR-13).
- JSON: `basePercent` is added to both `MarshalJSON` and `UnmarshalJSON`
  aux structs. Absent field decodes to `0` (Go zero value) — existing
  serialized bonuses (registry snapshots, equipped-asset JSON) are
  backward compatible with no migration (FR-6, PRD §6).
- `NewBonus`/`NewMultiplierBonus`/`NewFullBonus` keep their signatures and
  set `basePercent: 0`. No call-site churn outside this task's scope.

### 2.3 Engine — base-percent applied per-bonus with integer truncation

In `ComputeEffectiveStats` (`character/model.go:269`), the non-equipment
bonus loop gains a third accumulator:

```go
basePercentFlat := make(map[stat.Type]int32) // zero-init like the others

for _, b := range m.bonuses {
    flatBonuses[b.StatType()] += b.Amount()
    multipliers[b.StatType()] += b.Multiplier()
    if b.BasePercent() != 0 {
        basePercentFlat[b.StatType()] += baseValues[b.StatType()] * b.BasePercent() / 100
    }
}
```

and `computeEffective` becomes:

```go
effective := float64(base+flat+basePercentFlat[statType]) * (1.0 + mult)
```

Full formula per stat (matches PRD FR-7 verbatim):

```
effective = floor((base + flat + Σᵢ floor(base × pctᵢ / 100)) × (1 + mult))
```

Properties, deliberately:

- **Truncation per bonus, inside the loop** — `baseValues[t] * pct / 100`
  is Go integer division, which truncates toward zero; base and rate are
  both non-negative here, so it equals the client's `floor`. Two
  base-percent bonuses on the same stat truncate independently
  (deterministic per FR-7), because the division happens per bonus, not on
  a summed rate.
- **Basis is `baseValues`** — populated straight from `m.baseStats`, before
  any flat or equipment contribution is folded in. Equipment can never leak
  into the basis because equipment bonuses are accumulated into
  `flatBonuses` only (existing code, unchanged).
- **Non-primary stats degrade to zero** — `baseValues` is `0` for
  weapon-attack/speed/etc., so a (never-produced) base-percent bonus on
  those contributes nothing. `TypeMaxHp`/`TypeMaxMp` have a real base, so a
  base-percent bonus there would compute correctly per FR-8 — valid but
  unmapped.
- **Overflow**: base is `uint16` (≤ 65535) and rates are small (≤ ~35);
  `int32` accumulation is comfortably safe.

Equipment snapshot bonuses (`snap.bonuses`) keep contributing `Amount()`
only — equipment has no base-percent semantics, and leaving that loop
untouched preserves the existing behavior guarantee.

### 2.4 Dimension-preserving re-sourcing (`WithSource`)

`AddBuffBonuses` and `AddPassiveBonuses` currently re-stamp the source with
`stat.NewFullBonus(source, b.StatType(), b.Amount(), b.Multiplier())` —
which would zero `basePercent` and silently reduce Maple Warrior to a
no-op on the live-apply path. Fix at the model, not the call sites'
knowledge of fields:

```go
// WithSource returns a copy of the bonus with the source replaced,
// preserving every bonus dimension.
func (b Bonus) WithSource(source string) Bonus
```

Both processor helpers switch to `b.WithSource(source)`. This is the
"future one-to-many buff cannot repeat this bug shape" goal (PRD §2)
applied to the *dimension* axis: the next `Bonus` field added will flow
through re-sourcing automatically instead of needing a parallel edit in
two processors.

### 2.5 Consumer migration

Both call sites become one-liners over the new API; the
`isMultiplier`-branching blocks are deleted.

`kafka/consumer/buff/consumer.go` (`handleBuffApplied`):

```go
for _, change := range e.Body.Changes {
    bs := stat.BonusesForBuffChange("", change.Type, change.Amount)
    if len(bs) == 0 {
        l.Debugf("Unknown buff stat type: %s", change.Type)
        continue
    }
    bonuses = append(bonuses, bs...)
}
```

(Source stays `""` here — `AddBuffBonuses` stamps `buff:<sourceId>` via
`WithSource`, matching current behavior.)

`character/initializer.go` (`fetchBuffBonuses`):

```go
source := fmt.Sprintf("buff:%d", buff.SourceId)
for _, change := range buff.Changes {
    bs := stat.BonusesForBuffChange(source, change.Type, change.Amount)
    if len(bs) == 0 {
        l.Debugf("Unknown buff stat type: %s", change.Type)
        continue
    }
    bonuses = append(bonuses, bs...)
}
```

Because both paths construct through `BonusesForBuffChange` and end up
source-stamped `buff:<sourceId>` (one via `WithSource`, one directly),
the FR-11 parity property holds by construction; the test in §3 asserts it
anyway.

Expiry needs no change: `RemoveBuffBonuses` → `WithoutBonusesBySource`
removes by source string, so all four MW bonuses (same `buff:<sourceId>`,
distinct stat types) are removed together (FR-9). Registry upsert semantics
(`WithBonus` keyed on `(source, statType)`) are also safe: the four MW
entries have distinct stat types and coexist; a re-applied MW buff
overwrites its own four entries idempotently.

## 3. Testing

All tests use the existing Builder-pattern setup (`NewModel().WithBaseStats(...)
.WithBonuses(...)`); no `*_testhelpers.go`.

1. **Mapping table** (`stat/model_test.go`) — migrate `TestMapBuffStatType`
   to `TestBonusesForBuffChange`:
   - `MAPLE_WARRIOR` → exactly 4 bonuses; assert stat-type set
     {STR, DEX, INT, LUK}, `BasePercent() == amount`, `Amount() == 0`,
     `Multiplier() == 0`, source propagated (FR-12).
   - Every legacy type → exactly 1 bonus with the same kind/values the old
     function produced (table ported row-for-row).
   - Unknown type → empty slice.
2. **Bonus JSON round-trip** (`stat/model_test.go`) — marshal/unmarshal a
   base-percent bonus; decode a legacy JSON blob *without* `basePercent`
   and assert it decodes to 0 (FR-6).
3. **Engine truncation** (`character/model_test.go`) — FR-13 cases:
   - base STR 13, MW 10% → +1 (13×10/100 = 1.3 → 1).
   - base 4, 10% → +0.
   - base 100 + 30 equip STR + MW 10% → 140, not 143 (equipment excluded
     from basis).
   - two base-percent bonuses on one stat truncate independently
     (e.g. base 15, 10% + 10% → +2, not floor(15×0.20)=3).
   - base-percent coexists with a multiplier on the same stat:
     `floor((base + flat + bp) × (1+mult))` ordering pinned.
4. **Lifecycle** (`character/processor_test.go`) — `AddBuffBonuses` with the
   4 MW bonuses raises all four computed primaries; `RemoveBuffBonuses`
   restores all four (FR-14). This path also proves `WithSource` preserves
   `basePercent` end-to-end.
5. **Path parity** (FR-11) — build the bonus set for one MW buff via the
   consumer path (`BonusesForBuffChange("", …)` + `AddBuffBonuses`
   re-sourcing) and via the initializer path
   (`BonusesForBuffChange("buff:<id>", …)`), assert the resulting model
   bonus sets are identical.

## 4. Error Handling

- Unknown buff type: unchanged — debug log, skip (FR-4). Empty-slice return
  is the sentinel; no error value (matches the old `("", false)` contract).
- No new failure modes: no I/O, no config, no wire changes. The registry,
  tenancy, and Kafka layers are untouched.

## 5. Files Touched

| File | Change |
|---|---|
| `stat/model.go` | `Bonus.basePercent` + accessor + `NewBasePercentBonus` + `WithSource`; JSON round-trip; delete `MapBuffStatType`; add `BonusesForBuffChange` |
| `character/model.go` | base-percent accumulator + formula in `ComputeEffectiveStats` |
| `character/processor.go` | `AddBuffBonuses`/`AddPassiveBonuses` re-source via `WithSource` |
| `kafka/consumer/buff/consumer.go` | migrate to `BonusesForBuffChange` |
| `character/initializer.go` | migrate `fetchBuffBonuses` to `BonusesForBuffChange` |
| `stat/model_test.go`, `character/model_test.go`, `character/processor_test.go` | tests per §3 |

Out of scope (per PRD non-goals): HYPER_BODY semantics, atlas-data/atlas-buffs
emitters, `MapStatupType`, client display.

## 6. Verification

Per CLAUDE.md: `go test -race ./...`, `go vet ./...`, `go build ./...` in
atlas-effective-stats; `docker buildx bake atlas-effective-stats` from the
worktree root; `tools/redis-key-guard.sh`; code review
(backend-guidelines-reviewer + plan-adherence-reviewer) before PR.
