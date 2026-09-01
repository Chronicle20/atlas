# Task 18 review — `atlas-maker` crystal-band table and seed groups

Range reviewed: `450879b45..fd7a3c6c1` (5 commits: gachapon/reagent-adjacent docs,
v72/v79 reagent seeds, the derivation doc, the ledger row, and the crystalband
feature commit `fd7a3c6c1`). Module: `services/atlas-maker/atlas.com/maker`.

## 1. Seed fidelity against the derivation

Extracted all 27 seed files (`gms/{72_1,79_1,83_1}/crystal-bands/crystal-band-*.json`)
and diffed the `(minLevel, maxLevel, crystalItemId, count)` tuples across all three
version directories against derivation §5.4's 9-row table.

Result: **exact match, all three versions identical**, no invented/dropped/altered row:

```
31/50/4260000, 51/60/4260001, 61/70/4260002, 71/80/4260003, 81/90/4260004,
91/100/4260005, 101/110/4260006, 111/120/4260007, 121/200/4260008
```
— all `count=1`. PASS.

`crystal-band-31.json` and `crystal-band-121.json` (72_1 vs 83_1, 79_1 vs 83_1)
byte-diffed identical. PASS.

## 2. Band-boundary correctness

`Model.Contains` (`crystalband/model.go:53`) is `reqLevel >= min && reqLevel <= max` —
inclusive both ends, matching derivation §5.4.

`TestCrystalForLevelAtBandBoundaries` (`crystalband/processor_test.go:84`) checks, for
every one of the 9 rows, `minLevel-1/minLevel/minLevel+1`-equivalent behaviour via
`b.minLevel`, `b.minLevel-1` and `b.maxLevel`, `b.maxLevel+1`, asserting adjacency (a
level one below a band's min lands in the previous band, one above a band's max lands
in the next) and the two outer edges (30/201) land in `ErrNotFound`. This is a genuine
boundary pin, not an interior-only check — it would fail under an off-by-one at any of
the 8 internal seams or either outer edge. `TestModelContainsIsInclusiveAtBothEnds`
(`builder_test.go:69`) additionally unit-pins 31/40/50 in and 30/51 out at the unit
level. PASS.

## 3. The two controller rulings

**Count column.** `entity.go:23-29` and `model.go:44-47` both carry a comment stating
`Count` is "an Atlas product decision, NOT a derived value" with no derivation-doc
citation as evidence — worded exactly as ruled. Seeded `1` in all 27 files. PASS.

**Out-of-band behaviour.** `processor.go:20-27` (the `ErrNotFound` doc comment) and the
`TestCrystalForLevelBelowLowestBand` comment (`processor_test.go:128-133`) both state
the client never reads the table back and that rejecting the craft is "Atlas's own
ruling," not client-mirrored behaviour. The test asserts `ErrNotFound` at level `1`,
`30`, and symmetrically `201` — no clamping at either end. PASS.

## 4. `CrystalForLevel` signature and sentinel

`crystalband/processor.go:39`: `CrystalForLevel(reqLevel uint32) (item.Id, uint32, error)`
— exact signature match to the brief/controller text. `ErrNotFound` is a package-level
`var` (`errors.New`, `processor.go:20`), wrapped with `%w` at both the not-found sites
(`processor.go:69`, `:89`), so `errors.Is(err, crystalband.ErrNotFound)` works for a
caller — same shape as `reagent.ErrNotFound`. PASS.

## 5. Tenant scoping

`TestCrystalForLevelIsTenantScoped` (`processor_test.go:159`) seeds differing bands
(different `crystalItemId`/`count`) for two tenants at the same `(minLevel, maxLevel)`
and asserts each tenant's `CrystalForLevel` call reads back its own values — this would
fail if the query weren't tenant-scoped. Unique index shape matches the brief:
`uniqueIndex:idx_crystal_bands_tenant_min` on `(TenantId priority 1, MinLevel priority
2)` (`entity.go:19-20`), the same two-column shape as `reagent`'s
`idx_reagents_tenant_item`.

**Gap (non-blocking):** unlike `reagent.processor_test.go`'s
`TestGetByItemIdNotFoundAcrossTenants` (a band seeded only under tenant A, then read
under tenant B's context and asserted `ErrNotFound`), there is no negative-scoping test
for crystalband — only the positive "each tenant sees its own values" case. The positive
test already proves the query is tenant-filtered (it would collide/produce ambiguous
results otherwise), so this is not a correctness gap, but it does not fully mirror the
stated bar (reagent's scoped/not-found *pair*).

## 6. Item id width

`item.Id` is `uint32` (confirmed:
`.../atlas-constants@.../item/constants.go:5: type Id uint32`). Entity column
`CrystalItemId uint32` (`entity.go:22`), no narrowing anywhere in
`administrator.go`/`provider.go`/`subdomain.go`. `4260008` (the widest id in this
table, well under uint32 range, but the pattern that bit an earlier task was narrower
storage, not overflow) is round-tripped through `TestCrystalForLevelAtEachBand`,
`TestCrystalForLevelAtBandBoundaries`, and `TestBuilderRoundTrip`. PASS.

## 7. Load-once-per-tenant

`CrystalForLevel` (`processor.go:81-92`) calls `p.GetAll()` once, then iterates the
in-memory slice with `Contains` — no per-lookup query. `processor.go:79-80` comment
states this explicitly. PASS.

## 8. Reference-shape conformance (`reagent`)

File-for-file identical set to `reagent/` (`entity.go`, `model.go`, `builder.go`,
`processor.go`, `provider.go`, `administrator.go`, `rest.go`, `resource.go`,
`subdomain.go`, `mock/processor.go`, plus the four `_test.go` files). `mock/processor.go`
diffed cleanly against `reagent`'s except for the crystalband-specific
`GetByMinLevel`/`CrystalForLevel` swap-ins. No unexplained divergence found.

## 9. `CATALOG_REVISION`

`grep -rn CATALOG_REVISION services/atlas-maker deploy/seed` returns nothing in this
range — not touched, matching commits `062373736`/`2bf4370ec`. PASS.

## Repo conventions

- No `*_testhelpers.go` files (`find ... -iname '*testhelper*'` empty).
- No absolute/home paths in any committed file under this range.
- Builder pattern used throughout tests (`crystalband.NewBuilder(...)`), no bespoke
  constructors.
- `libs/atlas-constants` checked and reused (`item.Id`) rather than reinventing a type.

## Build/test verification

```
cd services/atlas-maker/atlas.com/maker
go build ./...                              # ok
go test ./crystalband/... ./seed/... -count=1   # ok
go test ./... -count=1                      # ok (atlas-maker, crystalband, reagent, seed)
```

## Findings

Non-blocking:
- `services/atlas-maker/atlas.com/maker/crystalband/processor_test.go` — no
  cross-tenant-not-found test analogous to `reagent`'s
  `TestGetByItemIdNotFoundAcrossTenants` (band seeded only for tenant A, read under
  tenant B's context, assert `ErrNotFound`). The existing
  `TestCrystalForLevelIsTenantScoped` proves scoping via differing values per tenant,
  which is adequate evidence, but does not fully match the stated reagent bar (the
  scoped/not-found *pair*).

Not evaluable: none — the full review surface (crystalband package, seed groups,
derivation doc §5, seed JSON across all three versions, main.go wiring) was read and
verified against repo state.

## Verdict

verdict: APPROVED_WITH_FINDINGS
