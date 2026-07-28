# Backend Audit — atlas-channel (task-102 MTS marketplace)

- **Service Path:** services/atlas-channel/atlas.com/channel
- **Guidelines Source:** backend-dev-guidelines skill (DOM/SUB/FILE/EXT/SCAFFOLD/DOM-25)
- **Date:** 2026-07-10
- **Scope:** every package task-102 touched — `mts/` (+ cart, configuration, holding, listing, transaction, wanted, wish), `socket/handler` (MTS/ITC), `socket/writer` (MTS/ITC), `kafka/consumer/{mts,saga,wallet}`, `kafka/message/{mts,saga,wallet}`, `data/item`, `configuration/projection`, `main.go`
- **Build:** PASS (`go build ./...` exit 0)
- **Vet:** PASS (`go vet ./...` exit 0)
- **Tests:** PASS — 22 in-scope packages `ok`, 0 failed (`go test ./... -count=1`); every in-scope package completes < 0.03s (no unstubbed-producer 42s hangs)
- **Overall:** NEEDS-WORK (build/tests green; 4 Important finding classes)

> This file OVERWRITES the earlier MTS-scoped pass — it is the full per-package sweep of every directory task-102 modified.

Note on classification: atlas-channel is a client-facing socket/packet gateway. The MTS packages are REST-read clients of atlas-mts/atlas-tenants/atlas-data plus a Kafka command emitter and status consumers — none own a database, expose REST, or use `resource.go`/`administrator.go`/`entity.go`/`provider.go`. DOM-01..20 that presuppose a DB-owning REST domain are therefore N/A; the load-bearing checklists here are **FILE-\***, **EXT-\***, **DOM-21**, **DOM-23/24/25**, and **SCAFFOLD-07**.

---

## Summary

### Counts by severity
- **Critical:** 0
- **Important:** 4 finding classes (FILE-02, EXT-01, EXT-02, DOM-25) spanning 8 package instances
- **Minor:** 6

### Critical
- None. Build, vet, and the full in-scope test suite are green; no security surface (auth/token) in scope.

### Important (one-liners)
1. **FILE-02 — `mts/configuration`**: `RestModel`, its JSON:API methods, and `Extract` live in `model.go` alongside the domain `Model`; there is no `rest.go`. Failure scenario: the next contributor edits the tenant DTO and the immutable domain model in the same file, and the collapsed layout is exactly the `wallet.go` collapse DOM File-Responsibilities was tightened to reject (task-102).
2. **EXT-01 — `mts/holding`, `mts/transaction`**: neither `RestModel` implements `SetToOneReferenceID`/`SetToManyReferenceIDs`. Failure scenario: if atlas-mts ever adds a `relationships` block to a holdings/transactions response, api2go unmarshal errors and the channel surfaces it as an empty list / misleading load-failure — the exact task-037 "not found" trap the check exists to prevent.
3. **EXT-02 — `mts/holding`, `mts/listing`, `mts/transaction`, `mts/wish`, `mts/configuration`**: these packages call another atlas service over REST but have no `httptest`-backed integration test; their tests only exercise `Extract`/query-string building, which bypasses the api2go unmarshal path. Failure scenario: a `GetName()` / relationship-stub / tag mismatch against the real upstream JSON:API shape is not caught until runtime (only `data/item` has the httptest fixture).
4. **DOM-25 — hardcoded client-interpreted wire codes**: (a) `mtsFailReasonLoadFailed byte = 'N'` (`kafka/consumer/mts/consumer.go:650`, used at `:172`/`:228`) is a `CITC::NoticeFailReason` code written as a Go literal instead of resolved through the same `noticeFailReasons` tenant table the file uses for every other fail reason; (b) `nProcessStatus*` (`mts/transaction/view.go:27-32`) and the auction "Exhibit" code (`mts/listing/view.go:73-76`) are fed through the client `GetContractHistoryCode`/`GetAuctionHistoryCode` lookup switches yet are hardcoded. Failure scenario: the "IDA-verified identical across versions" comments are the exact version-stability defense the task-103 uniformity ruling rejects; a version whose code table differs (or a tenant needing a re-map) silently renders the wrong disposition string.

### Non-Blocking (Minor)
- `mts/cart/cart.go` and `mts/wanted/wanted.go` are single-function view renderers named `<pkg>.go` instead of `view.go` (sibling MTS packages use `view.go`). Single-purpose, so FILE-06 does not fire, but the naming diverges.
- `mts/configuration` has no test file — the `Registry` cache/fallback logic and `Extract` zero-knob folding are untested.
- `kafka/consumer/wallet` has no test file.
- `mts/wish/model.go:19-22` places the `TypeCart`/`TypeWanted` domain enums in `model.go` (arguably `state.go`).
- **EXT-03 — `data/item`**: `GetIdsByName` bubbles the raw error, but the caller (`browseFilterFromSearchItcList`) collapses every error — transport/5xx included — into "no results" (logged). Defensible for a search box, but 404-vs-transport is not distinguished.
- Structural/echoed tab & flag literals (`itcSectionCart=4`, `mtsSectionAuction=3`, `mtsSoldFlag*`, generic-`0` reason fallbacks) are client-round-tripped section indices / booleans, not version-variant switch codes — noted, not flagged.

---

## Per-Package Results

### mts (Kafka command emitter)
| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in processor.go | PASS | mts/processor.go:18,28,33 |
| FILE-06 | no catch-all | PASS | processor.go / producer.go split |
| DOM-24 | producer stubbed in emitting tests | PASS (no emit) | producer_test.go tests `*CommandProvider` builders directly (build message, no `Emit`) |
| DOM-25(c) | semantic keys, not client bytes | PASS | processor.go passes `origin`/`resultKind` strings |

### mts/cart
| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-06 | no catch-all | PASS (single-purpose) | cart/cart.go:31 sole func `Items` (view render) |
| — | naming | MINOR | should be `view.go` for parity with siblings |

### mts/wanted
| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-06 | no catch-all | PASS (single-purpose) | wanted/wanted.go:29 `WorldItems` + `toWantAdItem` (view render only) |
| DOM-21 | atlas-constants reuse | PASS | uses `inventory.TypeFromItemId`, `item.Id`, `world.Id` (wanted.go:41) |
| — | naming | MINOR | should be `view.go` |

### mts/configuration
| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-02 | RestModel + Extract + JSON:API in rest.go | **FAIL (Important)** | `RestModel`, `GetName/GetID/SetID`, `Extract` all in configuration/model.go:68-147; no `rest.go` in package |
| FILE-03 | request funcs in requests.go | PASS | configuration/requests.go:24 |
| EXT-02 | httptest integration test | **FAIL (Important)** | calls atlas-tenants (requests.go:24) but package has no test file |
| DOM-21 | atlas-constants reuse | PASS | no shared type redeclared (economic knobs only) |
| — | cache singleton | PASS | registry.go:31 `sync.Once` singleton |
| — | tests | MINOR | no test file for Registry/Extract |

### mts/holding
| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01/02/03 | file split | PASS | processor.go / rest.go / requests.go / model.go / view.go |
| EXT-01 | relationship stubs present | **FAIL (Important)** | holding/rest.go:19-25 omits `SetToOneReferenceID`/`SetToManyReferenceIDs` |
| EXT-02 | httptest integration test | **FAIL (Important)** | rest_test.go tests only `Extract`/`Resource` — no upstream fixture |
| EXT-04 | RootUrl(domain) | PASS | requests.go:15 `requests.RootUrl("MTS")` |

### mts/listing
| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01/02/03 | file split | PASS | processor.go / rest.go / requests.go (BrowseFilter) / view.go |
| EXT-01 | relationship stubs present | PASS | listing/rest.go:74-75 |
| EXT-02 | httptest integration test | **FAIL (Important)** | requests_test.go tests only `BrowseFilter.query()` — no unmarshal round-trip |
| EXT-04 | RootUrl(domain) | PASS | requests.go:52 |
| DOM-25 | client codes config-resolved | **FAIL (Important)** | listing/view.go:73-76 auction "Exhibit" `processStatus=1` fed to client `GetAuctionHistoryCode` switch, hardcoded |
| DOM-21 | atlas-constants reuse | PASS | view.go:48 `inventory.TypeFromItemId`, `inventory.TypeValueEquip` |

### mts/transaction
| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01/02/03 | file split | PASS | processor.go / rest.go / requests.go / view.go |
| EXT-01 | relationship stubs present | **FAIL (Important)** | transaction/rest.go:21-27 omits both relationship setters |
| EXT-02 | httptest integration test | **FAIL (Important)** | rest_test.go tests only `Extract` |
| DOM-25 | client codes config-resolved | **FAIL (Important)** | transaction/view.go:27-32 `nProcessStatusSold/Purchased/BidLost/Cancelled = 0..3` fed to client `GetContractHistoryCode` switch, hardcoded |
| EXT-04 | RootUrl(domain) | PASS | requests.go:15 |

### mts/wish
| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01/02/03 | file split | PASS | processor.go / rest.go / requests.go / view.go / model.go |
| EXT-01 | relationship stubs present | PASS | wish/rest.go:33-34 |
| EXT-02 | httptest integration test | **FAIL (Important)** | processor_test.go tests `Extract`/`findBy*`/`ToMtsItem` — no upstream fixture |
| EXT-04 | RootUrl(domain) | PASS | requests.go:20 |
| DOM-25(c) | semantic keys | PASS | uses `WishOrigin*` strings |
| — | enums in model.go | MINOR | model.go:19-22 `TypeCart`/`TypeWanted` (→ state.go) |

### kafka/consumer/mts
| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 (fail reasons) | config-resolved | PASS (mostly) | `failNoticeOr`/`noticeFailReasonCode` resolve reason codes from tenant `noticeFailReasons` (consumer.go:526-576) |
| DOM-25 (load-failed) | config-resolved | **FAIL (Important)** | `mtsFailReasonLoadFailed byte = 'N'` (consumer.go:650) hardcoded `CITC::NoticeFailReason`; used `:172`,`:228` |
| DOM-24 | producer stubbed | PASS (no emit) | consumer_test.go exercises only `failNoticeOr` encoding + reason-tag discipline; 0.008s, no emit path |
| DOM-25(c) | semantic keys inbound | PASS | consumes `ReasonKey`/`Origin`/`ResultKind` strings |

### kafka/consumer/saga
| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | atlas-constants reuse | PASS | tab from `inventory.TypeFromItemId`/`TypeValueEquip` (consumer.go:110-114) |
| DOM-25 | generic reason literals | MINOR | `mtsSagaFailureReason/SaleLimit = 0` are documented generic-default fallbacks |
| DOM-24 | producer stubbed | PASS (no emit) | consumer_test.go 0.007s |

### kafka/consumer/wallet
| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| structure | scene-gated writer selection | PASS | consumer.go:76-83 branches on `session.CashScene()` |
| — | tests | MINOR | no test file |

### kafka/message/{mts,saga,wallet}
| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-23 | topic naming | PASS | `COMMAND_TOPIC_MTS`/`EVENT_TOPIC_MTS_STATUS`/`EVENT_TOPIC_WALLET_STATUS` in deploy/k8s/base/env-configmap.yaml as `KEY:"KEY"`; no literal override in service manifest |
| DOM-25(c) | domain emits semantic keys | PASS | mts/kafka.go uses `ReasonKey`/`Origin`/`ResultKind` (string); saga/wallet use string `ErrorCode`/`kind` — no `byte` client code on any domain-produced event |

### data/item
| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01/02/03 | file split | PASS | processor.go / rest.go / requests.go / model.go |
| EXT-01 | relationship stubs present | PASS | rest.go:33-39 |
| EXT-02 | httptest integration test | PASS | processor_test.go:30 `httptest.NewServer` serves a JSON:API fixture and asserts populated result |
| EXT-03 | 404 distinguished | MINOR | `GetIdsByName` bubbles raw error, but caller collapses all errors → empty results |
| EXT-04 | RootUrl(domain) | PASS | requests.go:14 `requests.RootUrl("DATA")` |

### socket/handler (ITC/MTS)
| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 (dispatcher modes) | config-resolved | PASS | `resolveItcOperationKey` reverse-resolves mode bytes from tenant `options["operations"]` (itc_operation.go:102-125); no mode byte hardcoded |
| SUB-04 | no manual JSON parsing | PASS | all arms use `fieldsb.*.Decode` codecs |
| DOM-21 | atlas-constants reuse | PASS | `inventory.Type*`, `item.Id`, `world.Id` throughout |
| DOM-25 (section/tab) | echoed indices | MINOR | `itcSectionWanted=2`/`itcSectionCart=4` are round-tripped client tab indices, not writer-table codes |

### socket/writer (ITC/MTS)
| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 | config-resolved knobs | PASS | set_itc.go:40-48 reads listing fee / commission / auction hours from tenant `mts-configs` via `configuration.GetRegistry()` |
| FILE | thin body wrappers | PASS | mts_operation2.go / set_itc.go delegate to `libs/atlas-packet` codecs |

### configuration/projection
| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| structure | pure diff/reconcile | PASS | apply.go `ComputeOps`/`flatten` are pure; state/subscriber/loop split cleanly |
| tests | present | PASS | projection_test.go |

### main.go
| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SCAFFOLD-07 | writers/handlers seeded | PASS | `SetItc`/`MtsOperation`/`MtsOperation2`/`MtsChargeParamResult` writers + `EnterMtsHandle`/`ItcOperationHandle`/`ItcQueryCashRequestHandle`/`ItcStatusChargeHandle` handlers present in template_gms_{83,84,87,95}_1.json |
| registration | consumers + handlers wired | PASS | main.go:211-212 (consumers), 531-534 (status handlers), 745-747 (writers), 883-886 (recv handlers) |

---

## Blocking (must fix)
- **FILE-02**: `mts/configuration` — move `RestModel` + JSON:API methods + `Extract` from `model.go` into a new `rest.go`.
- **EXT-01**: add no-op `SetToOneReferenceID`/`SetToManyReferenceIDs` to `mts/holding/rest.go` and `mts/transaction/rest.go`.
- **EXT-02**: add `httptest`-backed integration tests (upstream JSON:API fixture → populated model) for `mts/holding`, `mts/listing`, `mts/transaction`, `mts/wish`, `mts/configuration`.
- **DOM-25**: config-resolve `mtsFailReasonLoadFailed` ('N') through the `noticeFailReasons` table, and resolve `nProcessStatus`/auction-Exhibit codes from a tenant writer-options table (or document an explicit guideline exception — version-stability alone does not exempt per task-103).

## Non-Blocking (should fix)
- Rename `mts/cart/cart.go` → `view.go` and `mts/wanted/wanted.go` → `view.go` for parity.
- Add tests for `mts/configuration` (Registry fallback + Extract folding) and `kafka/consumer/wallet`.
- Move `mts/wish` `TypeCart`/`TypeWanted` enums to `state.go`.
- `data/item` EXT-03: distinguish `requests.ErrNotFound` from transport/5xx if the search box should differentiate.

## Final resolution (post-audit fixes)

Fixed (task-102 code, contained):
- **FILE-02 (mts/configuration) — FIXED.** `RestModel` + `Extract` moved out of `model.go` into a dedicated `rest.go` (mirrors the atlas-mts configuration split).
- **EXT-01 (holding/transaction rest models) — FIXED.** Added the `SetToOneReferenceID`/`SetToManyReferenceIDs` stubs to `mts/holding/rest.go` and `mts/transaction/rest.go` (the api2go/task-037 unmarshal trap), matching the listing/wish siblings.
- **DOM-25(a) (hardcoded load-failed byte) — FIXED, and corrected a real notice bug.** `mtsFailReasonLoadFailed byte='N'` (78) is gone; the GetUserSale/PurchaseItemFailed panel-load arms now config-resolve the semantic key `LIST_LOAD_FAILED` through the tenant `noticeFailReasons` table (already seeded = 73 in all versions) via a new `userListFailedBody` helper. IDA (v83 `CITC::NoticeFailReason` 0x5a4752) proves 73 → SP_4785 "failed to load the list", whereas the old hardcoded 78 → SP_4768 "you have at least 1 bid on the item" — so the load-failure arms had been showing the WRONG notice.

Deferred (larger — see the consolidated summary for the decision):
- **DOM-25(b)** — `nProcessStatus 0..3` (mts/transaction/view.go) and auction `Exhibit=1` (mts/listing/view.go) feed the client GetContractHistoryCode/GetAuctionHistoryCode switches and are still hardcoded. Config-resolving these needs NEW tenant tables (contract-history / auction-category), seeded per version, plus a rollout note — a design decision, not a contained fix.
- **EXT-02** — httptest-backed unmarshal tests for the 5 MTS REST read clients (holding/listing/transaction/wish/configuration). Real test-coverage gap; producible but 5 new fixtures.

## Update — the two "deferred" Important items are now FIXED

- **DOM-25(b) — FIXED.** MtsItem `nProcessStatus` (History disposition + Auction
  category) is now a SEMANTIC key config-resolved from the tenant `processStatusCodes`
  writer table at Encode (soft resolver, 0 on miss). Added `fieldcb.MtsProcessStatus*`
  keys; transaction/listing/holding/wish views pass keys; seeded gms_83/84/87/95;
  rollout-checklist §3c documents the live patch.
- **EXT-02 — FIXED.** Added httptest-backed unmarshal tests for all 5 MTS REST
  read-clients (holding/listing/transaction/wish/configuration) — each serves a real
  JSON:API body and asserts the domain model is populated, exercising the api2go
  unmarshal path + the EXT-01 relationship stubs (not a mock).

All atlas-channel Important findings from this audit are now resolved. The Minor
items (cart.go/wanted.go file naming, a couple untested consumers, data/item EXT-03)
remain as low-risk notes.
