# Morph Potion Routing — Design

Task: task-140-morph-potion-routing
Status: Approved design (Phase 2)
PRD: `docs/tasks/task-140-morph-potion-routing/prd.md`

## 1. Summary

Route classification-221 (transformation) consumables through `ConsumeStandard` so the existing morph applier fires, and add weighted-random selection for the `morphRandom` spec. All changes live in `atlas-consumables`. The design has four parts:

1. **Routing** — one new case in `usesStandardConsumer`.
2. **Selection seam** — a pure-helper file `consumable/morph.go` mirroring the task-131 `reward.go` precedent (crypto/rand roll + deterministic pure selection function).
3. **Testability refactor** — extract the pure "which effects does this item produce" computation out of `ApplyItemEffects` into `computeEffectPlan`, so FR-3/FR-7 and the hp-alongside-morph acceptance criterion are pinnable by plain unit tests. Side-effect execution order is preserved exactly.
4. **Model getter** — additive `Morphs()` on the data-side consumable model.

## 2. Routing (FR-1, FR-2)

Add `item.ClassificationConsumableTransformation` to the switch in `usesStandardConsumer` (`services/atlas-consumables/atlas.com/consumables/consumable/processor.go:77`) and extend the function's doc comment to name transformation potions.

The existing cases stay as raw `item2.Classification(200/201/202/205)` literals: `libs/atlas-constants/item/constants.go` has no named constants for those four (the named consumable classifications start at 203). Adding names for them is unrelated churn and is explicitly not done here (noted as a possible cleanup, not a follow-up obligation).

FR-2 (coexisting specs like `hp` still apply) requires no code: routing through `ConsumeStandard` runs the untouched generic spec handling. It is pinned by test T5 (§7).

## 3. Random-morph selection seam (FR-5, FR-6)

### 3.1 Precedent check (FR-6)

task-131 (random-reward items, branch `task-131-random-reward-items`, `consumable/reward.go`) established the service's randomness seam:

- a dedicated pure-helper file in package `consumable`;
- `crypto/rand` for the roll (deliberate deviation from Cosmic's order-biased algorithm — that design's §2.4);
- **no seeded PRNG**: determinism under test comes from isolating the roll (one random draw) from the selection logic (pure function of the draw), and testing the pure part.

This task reuses that seam verbatim rather than inventing an injectable-`rand.Rand` variant.

### 3.2 New file `consumable/morph.go`

```go
// selectMorph is the pure selection function: given the weighted morph table
// and a roll in [0, sum(weights)), return the selected morph id. Morph ids are
// walked in ascending order — Go map iteration order is randomized, so sorting
// is what makes selection a deterministic function of the roll. Returns false
// when the table is empty or all weights are zero.
func selectMorph(morphs map[uint32]uint32, roll uint32) (uint32, bool)

// rollMorph sums the weights, errors on a zero total (defense in depth),
// draws one crypto/rand integer in [0, total), and delegates to selectMorph.
func rollMorph(morphs map[uint32]uint32) (uint32, error)
```

Probability of morph `m` = `weight(m) / sum(weights)` with no assumption that weights sum to 100 (FR-5).

**FR-6 resolution**: the deterministic surface is `selectMorph`. The exhaustive seam test (§7 T2) enumerates *every* roll value in `[0, total)` and asserts the selection count for each morph equals its weight exactly — stronger than any seeded statistical test, and consistent with how task-131 tests its pure sub-functions.

### 3.3 Alternatives rejected

- **Injectable `*rand.Rand` on the Processor** — invents a new seam, contra FR-6's "reuse the task-131 precedent"; also `Processor` is rebuilt inside consumer closures (`NewProcessor(l, ctx)`), so threading a seed through is invasive.
- **Inline `math/rand` in `ApplyItemEffects`** (the `ConsumeSummoningSack` style at `processor.go:494`) — weighting untestable; task-131 deliberately moved away from this style.

## 4. `ApplyItemEffects` integration and testability (FR-3, FR-7)

### 4.1 Problem

`ApplyItemEffects` interleaves the *decision* (which effects the item produces) with the *side effects* (`ChangeHP`/`ChangeMP` calls, `bp.Apply` → Kafka). FR-3 demands a unit test pinning the fixed-morph path end-to-end, and the acceptance criteria require asserting HP recovery alongside morph — neither is possible today without mocking two processors' transports.

### 4.2 Decision: extract `computeEffectPlan`

Add to package `consumable` (alongside `ApplyItemEffects`):

```go
// effectPlan is the pure result of interpreting a consumable's specs against a
// character: everything ApplyItemEffects will do, decided before any side effect.
type effectPlan struct {
    cureTypes []string       // ordered; from collectCureTypes
    hpChanges []int16        // ordered ChangeHP calls (hp, then hpR-derived)
    mpChanges []int16        // ordered ChangeMP calls (mp, then mpR-derived)
    statups   []stat.Model   // includes the resolved morph statup, if any
    duration  int32          // time spec / 1000
}

func computeEffectPlan(l logrus.FieldLogger, c character.Model, ci consumable3.Model) effectPlan
```

- `hpChanges`/`mpChanges` are ordered slices, not sums, so the exact per-call `ChangeHP`/`ChangeMP` sequence (and any per-call clamping in atlas-character) is byte-for-byte preserved.
- The morph branch inside statup collection implements FR-7 precedence structurally:

```go
if val, ok := ci.GetSpec(consumable3.SpecTypeMorph); ok && val > 0 {
    statups = append(statups, stat.Model{Type: ts.TemporaryStatTypeMorph, Amount: val})
} else if len(ci.Morphs()) > 0 {
    if morphId, err := rollMorph(ci.Morphs()); err == nil {
        statups = append(statups, stat.Model{Type: ts.TemporaryStatTypeMorph, Amount: int32(morphId)})
    } else {
        l.WithError(err).Warnf("Skipping morph for item [%d]: unusable morphRandom table.", ci.Id())
    }
}
```

`ApplyItemEffects` becomes: `plan := computeEffectPlan(l, c, ci)` followed by execution in the existing order — cures first (the task-051 D3 ordering comment moves with the code), then HP/MP changes, then `bp.Apply(f, characterId, -int32(itemId), 0, plan.duration, plan.statups)`. No behavior change for any existing item: the refactor moves computation, not effects.

`computeEffectPlan` takes the logger only for the roll-failure warn path; it stays unit-testable with a plain logrus logger.

### 4.3 Alternatives rejected

- **Minimal extraction (statups+duration only)** — leaves `hp` application inside the side-effecting function, so the "HP recovery alongside morph" acceptance criterion would be verifiable only by code inspection, not by test.
- **Interface-mock the buff/character processors** — much larger surface, no mocking precedent in this service's tests, and contra the PRD's "no special-casing beyond routing" spirit.

### 4.4 Fixture strategy (no test-only constructors)

- `character.Model`: public `character.NewModelBuilder()` (`character/model.go:365`) provides `MaxHp`/`MaxMp` for the `hpR`/`mpR` paths.
- `consumable3.Model`: build a `RestModel` literal (exported fields, including `Spec` and `Morphs`) and run it through the public `Extract` (`data/consumable/rest.go:92`) — the same path production data takes. No `*_testhelpers.go`, per project rules.

## 5. `Morphs()` getter (FR-4)

`data/consumable/model.go`:

```go
func (m Model) Morphs() map[uint32]uint32 {
    return m.morphs
}
```

Returns the internal map reference, matching the existing accessor convention (`MonsterSummons()` returns the internal slice); all callers are read-only. `rest.go:163` already populates the field from the wire.

## 6. Error handling

- **Zero-total / unusable morph table**: `rollMorph` errors; `computeEffectPlan` logs a warning and omits only the morph statup. Other specs on the item still apply and the consumption stands — same "errors logged, consumption not rolled back" semantics as every existing spec (PRD §8).
- **No morph and no table**: nothing appended; identical to today.
- No new failure modes on the reserve/commit path; `ConsumeStandard` error semantics are untouched.
- Fixed morph with no `time` spec would get duration 0 — data-driven, unchanged from the existing fixed-morph code path; all 28 fixed-morph v83 items carry `time`.

## 7. Test plan

All in `atlas-consumables` (package `consumable` unless noted):

- **T1 routing** (`usesStandardConsumer`): a 221 item id (e.g. 2210000) returns true; a still-bare classification (e.g. 204 scroll id) returns false. Pins the "no longer routes to ConsumeBare" acceptance criterion.
- **T2 exhaustive weighting** (`selectMorph`): table shaped like the real data (three entries, weights summing to 100) — enumerate rolls 0..99, assert per-morph selection counts equal the weights exactly. Second table whose weights do **not** sum to 100 (e.g. {1:3, 2:1}, total 4) to pin FR-5's no-sum-assumption.
- **T3 degenerate tables** (`selectMorph`): empty map and all-zero weights return `ok=false`.
- **T4 roll seam** (`rollMorph`): zero-total errors; over repeated calls on a valid table, every result is a table key (no distribution assertion here — T2 owns weighting).
- **T5 fixed morph end-to-end plan** (FR-3 + hp-alongside): 221 fixture with `morph=2`, `time=600000`, `hp=50` → plan has exactly one `MORPH` statup with amount 2, `duration=600`, and `hpChanges=[50]`.
- **T6 random-only plan**: 2211000-shaped fixture (no `morph` spec, non-empty `Morphs`, `hp=50`) → exactly one `MORPH` statup whose amount is a table key; `hpChanges=[50]`.
- **T7 precedence** (FR-7): fixture with both `morph=2` and a table not containing 2 → exactly one `MORPH` statup, amount 2.
- **T8 refactor regression**: representative pre-existing items (a cure pot with `poison`+`hp`, a stat pot with `pad`+`time`) → plans match the pre-refactor behavior (cure ordering, statup set, duration), pinning that `computeEffectPlan` is a pure move.

## 8. Data flow (unchanged pipeline)

client use-item → atlas-channel handler → `REQUEST_ITEM_CONSUME` → `RequestItemConsume` (now selects `ConsumeStandard` for 221) → reserve → commit → `ApplyItemEffects` → `bp.Apply` → atlas-buffs → temporary-stat change → client `MORPH`. Expiry via the buff duration; death cancellation via the existing respawn saga `CancelAllBuffs` (no new work, per PRD FR-8).

## 9. Out of scope (per PRD)

- 2212000 morph-other packet flow (`SendRandomMorphOtherRequest`/`OnRandomMorphRes`) — client-intercepted before any use-item packet; file the follow-up backlog item at task close, referencing the PRD's IDA evidence.
- Cash morph coupons (classification 530).
- Server-side anti-cheat for attacks while morphed.
- Named constants for classifications 200/201/202/205.

## 10. Verification

`go test -race ./...`, `go vet ./...`, `go build ./...` clean in `services/atlas-consumables/atlas.com/consumables`; `tools/redis-key-guard.sh` clean from repo root. No `go.mod` change is expected, so no bake is required; if one sneaks in, `docker buildx bake atlas-consumables` becomes mandatory. Diff must touch only `atlas-consumables` (acceptance criterion).

## 11. Risk

The single real risk is the `ApplyItemEffects` refactor: it is shared by the standard consume path and NPC-initiated effects (`ApplyConsumableEffect`). Mitigation: the extraction moves computation only (side effects and their order are untouched), and T8 pins representative pre-existing plans. DOM-25 is not implicated — the morph id is WZ game data, not a client-interpreted wire byte.
