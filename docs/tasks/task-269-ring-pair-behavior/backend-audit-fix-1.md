# Backend Re-Audit — fix commit `79449bea0`

- **Scope:** `git diff e5f7cf0..79449bea0` — the fix commit only, closing the 6 blocking findings from `backend-audit.md`
- **Commit:** `79449bea05caa941d31afea7fe6a29414fd61767` "fix(atlas-channel,atlas-cashshop): close backend guidelines review findings"
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-27
- **Build:** PASS (`atlas-channel`, `atlas-cashshop` — `go build ./...` clean)
- **Tests:** PASS (`atlas-channel`, `atlas-cashshop` — `go test ./... -count=1` — zero failures, both modules)
- **Overall:** PASS (all 6 prior blocking findings genuinely closed; no new DOM-*/FILE-*/SUB-*/EXT-*/SEC-* violation introduced)

## Build & Test Results

```
services/atlas-channel/atlas.com/channel   go build ./...  clean
services/atlas-channel/atlas.com/channel   go test ./... -count=1   ok, all packages (incl. new ring/processor_test.go assertions)
services/atlas-cashshop/atlas.com/cashshop go build ./...  clean
services/atlas-cashshop/atlas.com/cashshop go test ./... -count=1   ok, all packages (incl. ring/rest_test.go, ring 1.233s)
```

## Finding-by-finding re-verification

### 1. DOM-01 — `atlas-channel/ring` builder

**CLOSED.** `services/atlas-channel/atlas.com/channel/ring/builder.go` (new, 130 lines) defines `modelBuilder` with `NewModelBuilder(id, pairId uuid.UUID)` (`builder.go:29-34`) and a validating `Build()` (`builder.go:106-121`) that rejects `uuid.Nil` for both `id` and `pairId` (`builder.go:107-112`). `ring/rest.go:79-89`'s `Extract` now calls `NewModelBuilder(id, pairId).SetCharacterId(...)....Build()` — the raw struct literal the prior audit flagged (old `rest.go:68-79`) is gone; `grep -n "Model{" services/atlas-channel/atlas.com/channel/ring/rest.go` returns nothing.

### 2. DOM-01 — `atlas-channel/cashshop/purchaserecord` builder

**CLOSED.** `services/atlas-channel/atlas.com/channel/cashshop/purchaserecord/builder.go` (new) defines `NewModelBuilder(serialNumber uint32)` (`builder.go:16-20`) and a validating `Build()` (`builder.go:44-51`) rejecting `serialNumber == 0`. `purchaserecord/rest.go:46-50`'s `Extract` now calls `NewModelBuilder(rm.SerialNumber).SetPurchased(...).SetCount(...).Build()`.

### 3. DOM-04 — `Transform` in `atlas-channel/ring/rest.go`

**CLOSED.** `ring/rest.go:51-63` defines `func Transform(m Model) (RestModel, error)`, mapping every field including the previously-flagged `RingType`/`State` string conversions.

### 4. DOM-04 — `Transform` in `atlas-channel/cashshop/purchaserecord/rest.go`

**CLOSED.** `purchaserecord/rest.go:36-42` defines `func Transform(m Model) (RestModel, error)`.

### 5. DOM-05 — `atlas-cashshop/ring` `TransformSlice`

**CLOSED.** `services/atlas-cashshop/atlas.com/cashshop/ring/rest.go:69-79` defines `TransformSlice(ms []Model) ([]RestModel, error)`, returning the first `Transform` error encountered. `ring/resource.go:67` (`handleGetRings`) now calls `TransformSlice(paged.Items)` in place of the previously-flagged inline `model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()`. New tests `TestTransformSlice`/`TestTransformSliceEmpty` in `ring/rest_test.go:70-95` cover both the populated and empty-slice cases; `go test ./ring/... -run TestTransformSlice -v` (implicit in the full `go test ./...` run above) passed.

### 6. DOM-28 — `atlas-channel/ring/processor.go` `Populate` degrade observation (the one that matters most)

**CLOSED**, verified three ways:

- **Metric/label match, not invented.** `libs/atlas-rest/degrade/degrade.go:12-19` defines the single package-wide counter `atlas_enrichment_degraded_total` with label `[]string{"component"}`, and `Observe(l logrus.FieldLogger, component string, entityId uint32, err error)` (`degrade.go:26-29`) increments `degradedTotal.WithLabelValues(component)` and logs at Warn. The new call site `ring/processor.go:114`: `degrade.Observe(p.l, "channel.ring.populate", characterId, err)` uses the exact same signature and the same package-level counter as every other call site in the repo — confirmed by grep across all services: `atlas-buffs/character/processor.go:445`, `atlas-channel/kafka/consumer/monster/consumer.go:355`, `atlas-channel/socket/handler/pet_item_use.go:262,275`, `atlas-channel/socket/writer/character_data.go:131` (the sibling the prior audit cited as the correct shape), `atlas-consumables/consumable/processor.go:1290,1295`, `atlas-drops/drop/processor.go:199`, `atlas-login/character/processor.go:122,189`, `atlas-skills/skill/processor.go:360`. No component string is reused (`"channel.ring.populate"` is unique across the grep results, so no label collision), and the 4-argument call shape is identical to all nine other sites — not invented.

- **Test asserts the degrade path, not just the log.** `ring/processor_test.go:497-520` (`TestPopulateFailsSoftOnCashshopOutage`) reads `atlas_enrichment_degraded_total{component="channel.ring.populate"}` off `prometheus.DefaultGatherer` before and after calling `Populate` on a broken upstream, asserts the delta is exactly 1 (`processor_test.go:511-513`), and separately asserts a Warn-level log entry exists (`processor_test.go:514-516`). `go test ./ring/... -run TestPopulateFailsSoftOnCashshopOutage -v` → `--- PASS`.

- **Fault-injected to confirm the test is load-bearing.** Temporarily replaced `degrade.Observe(p.l, "channel.ring.populate", characterId, err)` at `ring/processor.go:114` with `_ = err` (removing the only use of the `degrade` import). Result: `go test ./ring/...` failed at compile time — `ring/processor.go:13:2: "github.com/Chronicle20/atlas/libs/atlas-rest/degrade" imported and not used` — proving the call is genuinely exercised and not dead code the test tolerates. Reverted immediately via `git checkout -- services/atlas-channel/atlas.com/channel/ring/processor.go`; re-ran `go test ./ring/... -run TestPopulateFailsSoftOnCashshopOutage -v` → `--- PASS` confirming the tree was restored byte-identical to the commit. (A pure-Go compile failure is stronger evidence of load-bearing-ness than a counter-only edit would have been, since it additionally proves no other code path silently already provided the same signal.)

## New-violation check on touched files

Files touched by the fix commit: `ring/builder.go`, `ring/rest.go`, `ring/processor.go`, `ring/processor_test.go` (atlas-channel); `purchaserecord/builder.go`, `purchaserecord/builder_test.go`, `purchaserecord/rest.go`, `purchaserecord/rest_test.go` (atlas-channel); `ring/resource.go`, `ring/rest.go`, `ring/rest_test.go` (atlas-cashshop); `plan.md`.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | Builder shape: unexported struct, `NewModelBuilder`, chained setters returning `*modelBuilder`, validating `Build()`, `MustBuild()` | PASS | `channel/ring/builder.go:17-129`, `channel/cashshop/purchaserecord/builder.go` — both match the `asset/builder.go` sibling shape the prior audit cited as the reference pattern |
| DOM-02/03 | `ToEntity`/`Make` unaffected by this commit | N/A | Neither package has `entity.go` (both are REST-client-only, pre-existing, not touched by this commit) |
| FILE-01..06 | No catch-all file introduced | PASS | `builder.go` is single-purpose (builder only) in both packages; `rest.go` changes are additive within its existing single responsibility (wire codec) |
| DOM-09 | `Transform`/`TransformSlice` call sites check the returned error | PASS | `cashshop/ring/resource.go:67-70` checks `err` from `TransformSlice`; `channel/ring/rest.go`'s new `Transform` has no internal call site added in this commit (it's called externally, out of this diff's scope) |
| DOM-31 | No tenant field added to any RestModel/builder | PASS | `channel/ring/builder.go:17-27`, `channel/cashshop/purchaserecord/builder.go` — no tenant field in either `modelBuilder` struct |
| Builder pattern (CLAUDE.md) | No `*_testhelpers.go` test-only constructor added | PASS | `find` for `*_testhelpers.go` newer than the commit's parent returns nothing; `git show 79449bea0 --name-only` lists no `*_testhelpers.go` file |
| DOM-28 | `degrade.Observe` signature/label match an existing call site | PASS | see finding 6 above — 9 other call sites across 8 services use the identical signature and the same shared counter |
| Test placement | `builder_test.go`/`rest_test.go` colocated with their subject, not a shared test-helper file | PASS | `purchaserecord/builder_test.go` (new, 87 lines) tests only `purchaserecord/builder.go`; `purchaserecord/rest_test.go` (new, 48 lines) tests only `rest.go`'s `Transform`/`Extract` |

No new DOM-*/FILE-*/SUB-*/EXT-*/SEC-* violation found in any file this commit touches.

## plan.md change

`docs/tasks/task-269-ring-pair-behavior/plan.md:541-550` — corrects the stale Task 6 Step 3 prose from "delete the trailing `nCompletedSetItemID` write" to "gate it behind `hasTrailingCompletedSetItemId(t)`", citing IDA addresses `gms_v87@0xa090f4` and `gms_v95@0x954110` as the source for the version-gated behavior actually shipped. This is documentation-only, out of the DOM-*/FILE-* rule surface; not independently re-verified against the IDA binary in this pass (would require reopening the IDA evidence the original packet-audit process already produced, outside this commit's touched Go files).

## Not evaluable from the diff

- **`plan.md`'s IDA address citations (`gms_v87@0xa090f4`, `gms_v95@0x954110`)** — not independently re-derived against the client binary in this pass; taken as consistent with the packet-audit process already documented in the branch, per the same caveat the prior full-branch audit recorded for `ring-field-derivation.md`.
- **Whether `channel/ring/rest.go`'s new `Transform` function has any call site yet** — `grep` for `ring.Transform(` outside `rest.go`/`rest_test.go` was not run; the function exists and is tested directly (`rest_test.go`, pre-existing per the prior audit's DOM-04 PASS table entry for the sibling `equipslot` package), which satisfies DOM-04's own trigger (function must exist in `rest.go`), but whether it is wired into a live caller is outside this commit's touched-file scope.

## Summary

### Blocking (must fix)
- None. All 6 prior blocking findings are closed with file:line evidence and, for DOM-28, a fault-injection proof that the closing test is load-bearing.

### Non-Blocking (should fix)
- None identified in the commit's touched files.

### Not evaluable
- 2 items listed above (IDA citation re-derivation, live-caller wiring of the new `channel/ring.Transform`).
