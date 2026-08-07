# Backend Audit — task-128 UI-surfacing diff (Go changes only)

- **Scope:** `git diff a4aa4a73b..60880a28f -- 'libs/**/*.go' 'services/**/*.go'` (27 files, +102/-1)
- **Modules touched:** `libs/atlas-saga`, `services/atlas-mts/atlas.com/mts`, `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-07-16
- **Build:** PASS (all 3 modules)
- **Vet:** PASS (all 3 modules)
- **Tests:** PASS, `-race` clean (all 3 modules; 0 failures)
- **Overall:** NEEDS-WORK (one Minor, no Important/Critical)

## Build & Test Results

```
services/atlas-mts/atlas.com/mts:              go build ./... -> exit 0
                                                go vet   ./... -> exit 0
                                                go test -race ./... -count=1 -> ok (all packages)
libs/atlas-saga:                               go build ./... -> exit 0
                                                go vet   ./... -> exit 0
                                                go test -race ./... -count=1 -> ok
services/atlas-saga-orchestrator/.../saga-orchestrator:
                                                go build ./... -> exit 0
                                                go vet   ./... -> exit 0
                                                go test -race ./... -count=1 -> ok (all packages)
```

No compile errors, no vet findings, no race detector findings, zero test failures across all three modules.

## What the diff does (verified against the trace in `mts-owner-flag-notes.md`)

Copies `foundAsset.Owner` (the seller's live equip asset's item-tag owner, `compartment.AssetRestModel.Owner` — `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/rest.go:19`) into the `AcceptToMtsListingPayload` at the saga-expansion capture site (`saga/processor.go:1601`, inside `expandTransferToMts`), then threads it end-to-end through every hop the existing `Flags` field already traverses: payload → saga handler → mts params → mts producer → wire command body (orchestrator side) → wire command body (atlas-mts side) → consumer → `AcceptRequest` → `listing.Builder`/`Model`/`entity`/`administrator`/`provider`/`rest.go`, plus the two secondary capture sites (cancel/expire → seller holding, buy-settle → buyer holding, mirrored into the full `holding.*` chain) and the saga-compensation reconstruction path (`compensator.go:1578`, `assetDataFromMtsListingSnapshot`). All 17 items in the notes doc's "Owner-threading checklist" are present in the diff, including the two "if in scope" items (holding-chain mirror, compensator wiring).

## Domain Checklist Results

### `listing` (domain package, `services/atlas-mts/atlas.com/mts/listing/`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go exists, Owner threaded | PASS | `listing/builder.go:50` (field), `:196-199` (`SetOwner`), `:342` (assembled into `Model{owner: b.owner}`) |
| DOM-04/DOM-18 | `Transform` + JSON:API RestModel | PASS | `listing/rest.go:46` (`Owner string \`json:"owner"\``), `:158` (`Owner: m.Owner()` in `Transform`) |
| DOM-11 | Provider lazy/pure mapping | PASS | `listing/provider.go:277` — `.SetOwner(e.Owner)` inside the existing curried `modelFromEntity`, no eager execution introduced |
| DOM-16 | administrator.go write path | PASS | `listing/administrator.go:126` — `Owner: m.Owner()` in `CreateListing`'s entity literal |
| DOM-21 | No atlas-constants duplication | PASS | `grep -rln Owner libs/atlas-constants` returns nothing; `owner` is a plain player-name `string`, no existing shared type/enum it should wrap. Confirmed no `Owner`/`PlayerName` type exists in `libs/atlas-constants`. |
| DOM-20 | Table-driven tests | **FAIL (Minor)** | `listing/builder_test.go:43-59` (`TestBuilder_SetOwnerRoundTrip`) and `listing/rest_test.go:12-31` (`TestTransformOwner`) are single-assertion functions, not `tests := []struct{...}{}` + `t.Run`. See Findings below — severity Minor because `testing-guide.md:18` phrases this as "Prefer," not "must," and 100% of the surrounding file (`builder_test.go`, `list_flow_test.go`, `processor_test.go`, etc. — 20+ existing tests) already uses this same non-table style; the two new tests are consistent with the file's own established pattern, not a regression. |

### `holding` (domain package, `services/atlas-mts/atlas.com/mts/holding/`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go, Owner threaded | PASS | `holding/builder.go:47` (field), `:196-199` (`SetOwner`), `:241` (assembled into `Model{owner: b.owner}`) |
| DOM-04/DOM-18 | RestModel + Transform | PASS | `holding/rest.go:42` (`Owner string \`json:"owner"\``), `:115` (`Owner: m.Owner()`) |
| DOM-11 | Provider mapping | PASS | `holding/provider.go:139` — `.SetOwner(e.Owner)` |
| DOM-16 | administrator.go | PASS | `holding/administrator.go:108` — `Owner: m.Owner()` |
| — | GORM entity + migration correctness | PASS | `holding/entity.go:75` — `Owner string \`gorm:"column:owner;not null;default:''"\``. `Migration()` (`holding/entity.go:18-23`) is a bare `db.AutoMigrate(&entity{})` — for a table that already exists in deployed tenants, Postgres `ALTER TABLE ADD COLUMN ... NOT NULL` requires a `DEFAULT`; the `default:''` matches the file's own precedent for post-creation columns (`listing/entity.go:89-90`, `OfferWishSerial`/`OfferWishOwnerId`, both `default:0` with the comment "AutoMigrate adds them (no index changes)"). Sibling `not null` string columns created in the *original* `CREATE TABLE` (`SellerName`, `SaleType`, `Category`, `SubCategory`) correctly omit the default since they were never added post-hoc. |
| — | No `holding` test added for Owner round-trip | Note (not a FAIL — no DOM-* ID covers this) | The diff does not add a `holding`-package unit test analogous to `listing/builder_test.go:43` or `listing/rest_test.go:12`; `holding.Owner()` is only exercised indirectly through the `listing` package's `SettleMove`/`transitionToSellerHolding` call sites and validated end-to-end by `go test -race ./...` passing, not by a dedicated `holding` unit test. Not scored against a checklist item since no DOM-* ID requires 1:1 test coverage per touched package. |

### `saga` / `mts` (sub-domain / support packages, `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 | Business logic not in handler | PASS | `saga/handler.go:2059` (`handleAcceptToMtsListing`) does a flat field copy `Owner: payload.Owner,` into `mts.AcceptToMtsListingParams{}` — no new branching/logic added, matches the file's existing thin-mapping style |
| — | Params/producer split (`processor.go`/`producer.go`) | PASS | `mts/processor.go:54` (`Owner string` on `AcceptToMtsListingParams`, a DTO — correctly co-located with the other DTO fields, not in `producer.go`); `mts/producer.go:51` (`Owner: params.Owner,` — Kafka message assembly, correctly in `producer.go` per file-responsibilities' `producer.go` = "Kafka message creation") |
| — | Compensator wiring | PASS | `saga/compensator.go:1579` — `Owner: p.Owner,` added to `assetDataFromMtsListingSnapshot`'s `asset2.AssetData{}` literal, alongside the pre-existing `Flag: p.Flags,`. Target field `asset2.AssetData.Owner` already existed unused (`kafka/message/asset/kafka.go:34`) before this diff — the diff activates it, doesn't invent it. |
| DOM-24 | Kafka producer stubbed in emitting tests | PASS (pre-existing infra, not modified by diff) | `saga/mts_expansion_test.go` (edited by the diff) runs inside a package whose `TestMain` calls `producertest.InstallNoop()` — `saga/testmain_test.go:7,11`. The diff's edits to `mts_expansion_test.go` (`:53` add `"owner": "Chronicle"` to the JSON fixture, `:123` add `require.Equal(t, "Chronicle", acc.Owner)`) are pure data/assertion additions inside an already-stubbed package; no new emit path introduced. |

### Wire-contract parity (the two hand-mirrored command-body structs)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| — | `AcceptToMtsListingCommandBody` identical on both sides (no shared type between services) | PASS | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/mts/custody/kafka.go:75` and `services/atlas-mts/atlas.com/mts/kafka/message/custody/kafka.go:96` both add `Owner string \`json:"owner"\`` at the identical position (immediately after `Flags uint16 \`json:"flags"\``, immediately before the `// sale params` / `// offer link` blocks). Diffed the two full struct bodies field-by-field (`sed -n` on both files) — every field name, type, and json tag matches, including the new one. |
| — | `AcceptToMtsListingPayload` (`libs/atlas-saga/payloads.go`) round-trips through generic decode | PASS | `libs/atlas-saga/unmarshal.go:378-379` decodes via `var payload AcceptToMtsListingPayload; json.Unmarshal(...)` — a plain struct-tag-driven decode, no explicit per-field list to update, so no `unmarshal.go` change was required for the new `Owner` field (correctly not touched by the diff). |

## Findings

### Blocking (must fix)

None. No Critical or Important findings.

### Non-Blocking (should fix)

- **DOM-20 (Minor):** `services/atlas-mts/atlas.com/mts/listing/builder_test.go:43` (`TestBuilder_SetOwnerRoundTrip`) and `services/atlas-mts/atlas.com/mts/listing/rest_test.go:12` (`TestTransformOwner`) are single-assertion test functions rather than table-driven (`tests := []struct{...}{}` + `t.Run`), per `testing-guide.md:18` ("Prefer table-driven tests"). Graded against the guideline text, not against prevalence — but noted as Minor because the guideline's own wording is a soft preference ("Prefer"), not one of the file's MUST-level rules, and the two new tests are single-scenario round-trip checks (the same shape as the pre-existing `TestBuilder_BuildsFixedListing` and `TestBuilder_RequiresTenantAndWorld` in the same file) where a table adds no coverage a second scenario wouldn't already provide.

### Observations (not scored — no matching DOM-* ID / pre-existing / out of diff-scope)

- **Formatting drift, not introduced by this diff:** `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/mts/producer.go:20-53` — the `AcceptToMtsListingCommandBody{}` struct literal is gofmt-misaligned (colons not vertically aligned to the longest key). Verified via `git show a4aa4a73b:.../mts/producer.go | gofmt -l` that this exact block was **already** misaligned at the diff's base commit — the diff's `Owner: params.Owner,` addition (`:51`) sits inside the pre-existing dirty block without correcting it. `go build`/`go vet` are unaffected (gofmt is not part of either). Not a DOM-* item and not introduced by this diff; flagged only so a future formatting pass catches it.
- `libs/atlas-saga/payloads.go` and `services/atlas-mts/atlas.com/mts/listing/processor.go` also show up under `gofmt -l`, but in struct literals the diff never touches (`TransferToMtsPayload`'s comment alignment, `CancelResult`'s field alignment respectively) — confirmed pre-existing via the same base-commit gofmt check, unrelated to this diff.
- Holding-package Owner round-trip has no dedicated unit test (see table above) — covered only transitively via `-race` passing on `listing`'s `SettleMove`/cancel-flow tests. No DOM-* ID mandates 1:1 per-package test coverage, so not scored, but worth a follow-up test if the holding package gets its own `builder_test.go`/`rest_test.go` treatment later.
- `RingId`/`ViciousCount`/`ItemLevel` are still not copied in `expandTransferToMts`'s snapshot literal (documented as a pre-existing, out-of-scope gap in `mts-owner-flag-notes.md:86`) — confirmed still true after this diff (`saga/processor.go:1570-1614` — no `RingId:`/`ViciousCount:`/`ItemLevel:` lines in the literal). Unrelated to `owner`; not this diff's responsibility.

## Security Review

N/A — atlas-mts and atlas-saga-orchestrator are not auth/token services; this diff touches only item-snapshot field plumbing.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- DOM-20: `listing/builder_test.go:43`, `listing/rest_test.go:12` — new tests are not table-driven (Minor, guideline is a soft "prefer").
