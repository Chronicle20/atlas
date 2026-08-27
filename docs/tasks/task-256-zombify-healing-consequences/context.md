# task-256 — Implementation Context

Companion to `plan.md`. Everything here was established by reading the tree in
this worktree during the plan phase; it exists so an implementer does not
re-derive it.

## Key files

### `atlas-consumables` (module root `services/atlas-consumables/atlas.com/consumables`)

| File | Role |
|---|---|
| `character/buff/processor.go` | Write-only today: `Apply` / `Cancel` / `CancelByTypes`, all Kafka producers. Task 1 adds the REST read. |
| `character/buff/stat/model.go` | `Model{Type character.TemporaryStatType; Amount int32}` — **exported fields, already typed**. No cast needed when comparing to `TemporaryStatTypeUndead`. |
| `character/buff/mock/processor.go` | The package mock; `morph_coupon_test.go`'s harness already wires it. |
| `consumable/processor.go:162` | `computeEffectPlan` — pure, gains `zombified bool`. |
| `consumable/processor.go:232` | `ApplyItemEffects` — already constructs `bp := buff.NewProcessor(l, ctx)` at line 233, so the read seam costs no new construction. |
| `consumable/processor.go:238-254` | The task-051 D3 cure-then-HP ordering. Must not move. |
| `consumable/morph_coupon.go:43` | `computeMorphCouponPlan` — pure, gains `zombified bool`. |
| `consumable/morph_coupon.go:100-133` | `consumeMorphCoupon` — the pre-commit fallible-read group, the commit, then the effects. |
| `consumable/morph_coupon_test.go:208-255` | `morphCouponHarness` — five mocks + call recorders. The harness for every new coupon test. |
| `consumable/processor_test.go:474-483` | `extractConsumable` — builds a `consumable3.Model` through the public `Extract`, per the no-test-helper-constructor rule. |

### `atlas-channel` (module root `services/atlas-channel/atlas.com/channel`)

| File | Role |
|---|---|
| `character/buff/model.go:10-20` | `IsMount` — the predicate `IsZombified` copies. |
| `character/buff/stat/model.go:8-10` | `Type()` returns a **`string`** — the cast `string(charconst.TemporaryStatTypeUndead)` is required here. |
| `character/buff/processor.go:46-76` | `ByCharacterIdProvider` + its `ErrNotFound` → empty-slice normalization. Already exists; Task 4 just calls `GetByCharacterId`. |
| `skill/handler/heal/heal.go` | The cast body. Eight seams added in Task 4. |
| `skill/handler/heal/formula.go:46-59` | `appliedPerRecipient` — **not modified**. `healDelta` branches before ever calling it with a negative. |
| `skill/handler/dispel/dispel.go:43-65` | The seam idiom being copied. |
| `skill/handler/dispel/dispel_test.go:54-100` | The save/restore/recorder test shape. |
| `data/skill/effect/` | `effect.Model`'s fields are unexported and there is **no builder**. `effect.Extract(effect.RestModel{Hp: 300})` is the only way to construct one carrying an HP value in a test. |

## Decisions made at plan time

**Heal gets eight seams, not one.** design D5 specified a single
`casterZombifiedFunc` and explicitly deferred "can the existing handler test
scaffolding drive `Apply` end-to-end offline?" to this phase. It cannot: `Apply`
loads the caster over REST at `heal.go:83` and returns early on failure, so with
one seam the zombify read is never even reached in a test. That would leave PRD
acceptance criteria 2–7 and 9 unassertable at the `ChangeHP` boundary the PRD
names. The resolution stays inside the design's own stated preference — the
`dispel` / `monstermagnet` package-var idiom, one line each, no `healDeps`
struct, no change to `Apply`'s flow. The seams also make the caster and
effective-stats fetches hermetic, so the new tests issue no HTTP at all.

**`resolveZombified` takes a `buff.Processor` rather than seaming
`ApplyItemEffects`.** design §3 asks for a fail-open test "exercising
`ApplyItemEffects` with a `buff/mock.ProcessorMock`", but `ApplyItemEffects` is a
package-level function that constructs its own processors and has no injection
point. Rather than seam it, the fail-open branch is extracted into
`resolveZombified(l, bp, characterId)`, which takes the interface and is directly
testable with the existing mock. FR-3's acceptance criterion ("a failed
`atlas-buffs` read logs at `Warn` and yields full-value healing") is then
asserted as two facts: `resolveZombified` returns `false` and logs one `Warn` on
a read error, and `computeEffectPlan(..., false)` produces full-value
`hpChanges`. The same helper is reused by `consumeMorphCoupon`, where the
harness *does* allow an end-to-end assertion.

**`computeMorphCouponPlan` moves after the commit.** It is currently called at
`morph_coupon.go:109`, before `ConsumeItem`. design D4 puts the zombify read
after the commit (so a buff-read fault can never bounce a paid cash item), and
the plan is a pure function of that read, so it moves with it. No existing
coupon test depends on the pre-commit position — `TestConsumeMorphCouponConsumeFailureAppliesNoEffects`
asserts only that no effects fire when the commit fails, which still holds.

**The "neither a morph nor an hp" warning is gated on `!zombified`.** That
message blames the tenant's cash WZ ingest. A coupon whose `hp: 1` halved to 0
under zombify would trip it and produce a false diagnosis.

## Deliberately-large task

**Task 1 touches 9 files** (6 source + 2 new tests + 1 read-only reference),
above the ~6-file splitting guideline. It is kept whole because it is a single
cohesive unit — one package's read half, copied file-for-file from
`atlas-channel/character/buff` — and every file is 15–50 lines of
copy-and-adapt with no cross-file design decisions. Splitting it would produce a
non-compiling intermediate state (the `Processor` interface and its mock must
change in the same commit as the `Model` they return).

Tasks 2, 3, and 4 are each within the guideline (4, 2, and 5 files, one service
each).

## Dependencies and ordering

```
Task 1 (consumables buff client) ──> Task 2 (consumables halving)
Task 3 (channel IsZombified)     ──> Task 4 (channel heal negation)
```

The two chains are independent and may run in either order. Nothing in this task
touches `atlas-buffs`, `atlas-character`, `libs/atlas-constants`, `atlas-ui`, or
any deployment manifest — design OQ-2 confirmed `atlas-consumables` already
resolves `BUFFS` through the shared `BASE_SERVICE_URL` it uses for its six
existing domains, so no ConfigMap change is needed.

## Not implemented, on purpose

- **FR-17 (`hpCon` suppression).** design OQ-1 queried skill 2301002 across all
  ten provisioned main-environment tenants: 300 effect rows, `HPConsume: 0` in
  every one. `skill/handler/common.go:137` already gates the cast-cost `ChangeHP`
  on `e.HPConsume() > 0`, so the reference's forcing is a structural no-op. No
  code, no test.
- **Undead mob damage from Heal** (design D6), **Chakra, Recovery Aura, MP
  Recovery, SuperGM Heal+Dispel** (PRD §4.4). Each deferred with a stated reason;
  none is touched.
- **`healDelta`'s `math.MinInt16` saturation branch.** Unreachable from an
  `int16` `perTarget`. Kept as a defensive guard, deliberately untested rather
  than covered by a test that cannot construct the input.
