# Backend Audit — task-130 Gen3 rebase integration

- **Scope:** Rebase-integration changes only (commit `d55b8c9d5` + handler conflict resolution + `tools/packet-audit/cmd/run.go`). NOT a re-audit of the task-130 feature.
- **Services:** atlas-consumables, atlas-channel
- **Date:** 2026-07-13
- **Build:** PASS (both services)
- **Vet:** clean (both services)
- **Tests:** all pass, `-race` clean (both services)
- **Overall:** PASS

## Build & Test Results

- `services/atlas-consumables/atlas.com/consumables`: `go build`/`go vet`/`go test -race ./...` all exit 0. `consumable` package tests pass (`ok atlas-consumables/consumable 0.011s`).
- `services/atlas-channel/atlas.com/channel`: `go build`/`go vet`/`go test -race ./...` all exit 0.

## Questions Answered

### 1. Is the Gen3 migration faithful to the sibling idiom (ConsumeScroll / ConsumeError)?

PASS. Faithful.

- Receivers moved from `*Processor` to `*ProcessorImpl`:
  - `VegaScrollError` — `consumable/vega.go:52`
  - `RequestVegaScroll` — `consumable/vega.go:95`
  - channel `RequestVegaScrollUse` — `channel/consumable/processor.go:46`
- Both new methods joined their `Processor` interface:
  - consumables `processor.go:56` (RequestVegaScroll), `processor.go:57` (VegaScrollError)
  - channel `processor.go:18` (RequestVegaScrollUse)
- Interface satisfaction is compile-enforced: `var _ Processor = (*ProcessorImpl)(nil)` (consumables `processor.go:86`, channel `processor.go:34`) and `var _ consumable.Processor = (*ProcessorMock)(nil)` in both mocks (consumables `mock/processor.go`, channel `mock/processor.go:18`). Build passes, so all four types satisfy their interfaces.
- No interface-where-impl mismatch. Inside the closures (`ReserveVegaScrollStage` `vega.go:168`, `ConsumeVegaScroll` `vega.go:191`) `p := NewProcessor(l, ctx)` is the `Processor` interface; the only methods invoked on it are interface methods — `p.VegaScrollError(...)` (`vega.go:180,206,212,215,219,222,228`) and `p.ValidateScrollUse(...)` (`vega.go:221`). No struct-field access on the interface `p`.
- Field reads on `p.cpp`/`p.cdp` occur only inside `*ProcessorImpl` receiver methods where they are valid: `p.cpp` in `VegaScrollError` (`vega.go:55`) and `p.cdp` in `RequestVegaScroll` (`vega.go:120`). This mirrors `RequestScroll` (`processor.go:534`) and `ConsumeError` (`processor.go:296`).
- The `ConsumeVegaScroll` closure's local `cdp` mirrors `ConsumeScroll`'s closure, which builds the identical local `cdp := consumable3.NewProcessor(l, ctx)` (`processor.go:724`).

### 2. Is exporting `VegaReservation` acceptable, or does it leak an internal type?

PASS. Acceptable, and forced by the idiom.

- The export is required because `VegaScrollError` is a public interface method whose signature names the type (`processor.go:57`), and the `mock` package must reference it as `[]consumable.VegaReservation` (consumables `mock/processor.go`, `VegaScrollError` signature). An unexported type cannot appear in an exported interface method that a foreign package implements.
- The type's fields stay unexported (`inventoryType`, `slot` — `vega.go:43-44`), so no real internal state leaks: external packages can pass a slice through but cannot construct instances. Positional construction (`VegaReservation{inventory2.TypeValueCash, ...}`) only compiles inside package `consumable` (`vega.go:180,198`).
- Directly analogous to `ConsumeError` already being a public interface method (`processor.go:54`). `ConsumeError` did not need a type export only because its parameters are primitives (`inventory2.Type`, `int16`), not a struct. No DOM guideline forbids exporting a type consumed by an exported interface method. Not a finding.

### 3. Does the local-`cdp` change preserve identical behavior to the original `p.cdp` read?

PASS. Identical.

- Pre-Gen3, `p.cdp` was the `ProcessorImpl` struct field initialized as `consumable3.NewProcessor(l, ctx)` in `NewProcessor` (`processor.go:81`). Post-Gen3, `NewProcessor` returns the `Processor` interface, so the field is unreachable through `p` inside the closure; the replacement local `cdp := consumable3.NewProcessor(l, ctx)` (`vega.go:197`) constructs the same processor from the same `l`/`ctx`.
- `cdp.GetById(scrollItem.TemplateId())` (`vega.go:210`) is a stateless data read; same inputs → same result. Behavior preserved.

### 4. Handler conflict resolution (character_cash_item_use.go)

PASS. The vega arm and task-126 point-reset arm coexist correctly as distinct sequential guards keyed on non-overlapping `CashSlotItemType` values:

- Vega arm: `if it == CashSlotItemTypeVegasSpellPre95 || it == CashSlotItemTypeVegasSpell95` (types 68/71) — `character_cash_item_use.go:114-133`, each path ends in `return`.
- Point-reset arm: `if it == CashSlotItemTypePointResetTier1 || it == CashSlotItemTypePointResetShared` (types 24/23) — `character_cash_item_use.go:135-140`, ends in `return`.
- Type constants are disjoint (`character_cash_item_use.go:158-161`); the classifier `GetCashSlotItemType` maps ClassificationVegasSpell → 68/71 (`:518-523`) and ClassificationPointReset → 23/24 (`:182-190`). No fallthrough or shared bucket.

### 5. packet-audit run.go

PASS. `CWvsContext::SendConsumeCashItemUseRequest` (and the `CItemSpeakerDlg` alias) now returns both candidates — `ItemUsePointReset` and `ItemUseVegaScroll` — keyed to the same fname (`tools/packet-audit/cmd/run.go:1994-1998`). Correct merge of task-126 + task-130.

## Non-Blocking Observations

- **Mock default divergence (harmless):** the `VegaScrollError` mock returns `err` on the nil-func default (consumables `mock/processor.go`), while the sibling `ConsumeError` mock returns `nil` (`mock/processor.go` ConsumeError). This is not a defect — production `VegaScrollError` returns `err` (`vega.go:62`), so the vega mock default is arguably the more faithful of the two. No test depends on it.
- **Coverage (pre-existing, out of scope):** `consumable/vega_test.go` covers only the pure helpers `vegaRates` and `resolveVegaEquip`; the migrated `ProcessorImpl` methods and the reservation-chain closures have no direct unit test. This gap predates the rebase (it is task-130 feature coverage, already reviewed) and is not introduced by commit `d55b8c9d5`. Migration correctness is instead guaranteed by the compile-time interface assertions plus the green suite.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- None required. Two harmless observations noted above.
