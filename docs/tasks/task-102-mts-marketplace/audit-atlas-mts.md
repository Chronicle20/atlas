# Backend Audit — atlas-mts

- **Service Path:** services/atlas-mts/atlas.com/mts
- **Guidelines Source:** backend-dev-guidelines skill (DOM-*/SUB-*/EXT-*/SEC-*)
- **Date:** 2026-07-10
- **Scope focus:** consumer→processor refactor, wish `listing_serial` column, multi-tenancy, money path (listing/bid/holding/wallet/transaction/saga)
- **Build:** PASS (`go build ./...` exit 0)
- **Vet:** PASS (`go vet ./...` exit 0)
- **Tests:** PASS (`go test ./... -count=1` exit 0 — every package `ok`, no hangs; mts consumer pkg 0.075s)
- **Overall:** NEEDS-WORK (build+tests green; 1 Important guideline violation)

## Build & Test Results

`go build ./...`, `go vet ./...`, and `go test ./... -count=1` all exit 0. No package took longer than 0.178s (listing) — no evidence of an unstubbed Kafka producer hang (a ~42s/emit stall would be visible). Confirmed emit-heavy packages (`kafka/consumer/mts`, `kafka/consumer/custody`, `holding`, `listing`, `testsupport`) all complete sub-second.

## Refactor Verification (consumer→processor extraction)

The extraction is faithful. Both `kafka/consumer/mts/consumer.go` and `kafka/consumer/custody/consumer.go` are now thin: decode command → single Processor method → Kafka emit. Business logic / DB-transaction orchestration moved to `listing/custody.go` (`Accept`, `SettleMove`, `RemoveSpuriousActive`, `RestoreFromHolding`), `holding/custody.go` (`Release`, `RestoreHolding`), and `wish/register.go` (`RegisterWish`, `RemoveWish`). Kafka emission and post-commit side-effects correctly stayed in the consumers. New methods follow the `Processor` interface + `ProcessorImpl` pattern (`listing/processor.go:156`, `wish/processor.go:15`, `holding/processor.go`). Idempotency semantics preserved (deterministic ids: `listing/custody.go:80-91` accept id-existence check, `MoveHoldingId` at `:185`, holding-exists guard `:276`; race arbiters via conditional `UpdateState`/`AdvanceAuctionBid`).

## Domain Checklist Results

### wish (domain; new `listing_serial` column)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go exists | PASS | wish/builder.go:28 `NewBuilder`, fluent setters, `Build()` validates tenantId (:91) |
| DOM-02 | ToEntity() | WARN | No `ToEntity()`; entity built inline in `CreateWish` (wish/administrator.go:94). Service-wide convention. |
| DOM-03 | Make(Entity) | WARN | Uses `modelFromEntity` (wish/provider.go:116) instead of `Make`; functionally equivalent. |
| DOM-04 | Transform | PASS | wish/rest.go:37 |
| DOM-05 | TransformSlice | WARN | No `TransformSlice`; handlers use `model.SliceMap(Transform)` inline (wish/resource.go:62,89). Composition, not a raw loop. |
| DOM-06 | Processor takes FieldLogger | PASS | wish/processor.go:42 `NewProcessor(l logrus.FieldLogger, ...)` |
| DOM-07 | Handlers pass d.Logger() | PASS | wish/resource.go:46,82,116,154 |
| DOM-08 | POST uses RegisterInputHandler | PASS | wish/resource.go:34 `registerInput("create_wish", ...)` via `rest.RegisterInputHandler[RestModel]` |
| DOM-09 | Transform errors handled | PASS | wish/resource.go:63,90,123 all check err |
| DOM-10 | Test DB tenant callbacks | PASS | test/database.go:28 `database.RegisterTenantCallbacks` |
| DOM-11 | Providers lazy | PASS | wish/provider.go:10-13 `database.SliceQuery`; map-keyed `Where` elsewhere |
| DOM-12 | No os.Getenv in handlers | PASS | grep: 0 matches in resource.go |
| DOM-15 | No direct entity writes in resource.go | PASS | grep: 0 `db.Create/Save/Delete` in resource.go |
| DOM-16 | administrator.go for writes | PASS | wish/administrator.go `CreateWish`/`DeleteWish`/`DeleteExpiredWanted` |
| DOM-18 | JSON:API interface on REST model | PASS | wish/rest.go:24-35 `GetName`/`GetID`/`SetID` |
| DOM-19 | Flat request model | PASS | wish/rest.go:10-22 flat, `Id json:"-"`, no nested Data/Type/Attributes |
| DOM-20 | Table-driven tests | PASS | wish/processor_test.go, resource_test.go present, pkg `ok` |
| DOM-21 | No atlas-constants duplication | PASS | uses `world.Id` (model.go:31); `listing/custody.go:109` uses `inventory.TypeFromItemId(item.Id(...))` |

**listing_serial column:** entity `ListingSerial uint32 gorm:"...;not null;default:0"` (wish/entity.go:63, additive AutoMigrate, no index churn); private field + getter (model.go:36,50); builder setter (builder.go:59); provider round-trip (provider.go:121); administrator persist (administrator.go:101); rest field + Transform (rest.go:16,45). Immutable-model compliance is clean.

### listing / holding / bid / transaction (money-path domains)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | FieldLogger | PASS | listing/processor.go:287, holding/processor.go, bid, transaction |
| DOM-08 | POST→RegisterInputHandler | PASS | listing/resource.go:164 list POST returns 202; DELETE cancel maps errors |
| DOM-11 | Providers lazy + world-0-safe | PASS | listing/provider.go map-keyed `Where` (:183); serial/serial.go:83 explicit map WHERE (documents world-0 struct-elision hazard) |
| DOM-16 | administrator.go | PASS | listing/administrator.go, holding/administrator.go, bid/administrator.go, transaction/administrator.go |
| DOM-17 | Error→HTTP status mapping | PASS | listing/resource.go: 400/403/404/409/500/202/204 (lines 69,86,97,109,289) |
| DOM-20 | Table-driven tests | PASS | administrator_test, processor_test, *_flow_test present; all `ok` |
| DOM-24 | Kafka producer stubbed in emitting tests | PASS | Per-test DI of a `producer.Provider` (`recordingProducer`, kafka/consumer/mts/consumer_test.go:35) injected via the handler's `pf` param; processors stub emission via `WithSagaEmitter`/`WithBalanceReader` (listing/processor.go:271,281). No global-singleton emit path → no ~42s stall (runtimes confirm). |

### Multi-tenancy

PASS. `tenant.MustFromContext`/`tenant.FromContext` used at every write site (listing/custody.go:93,298; wish/register.go:41; consumer history rows). Every query runs through `db.WithContext(ctx)` for callback tenant-scoping. World-0 struct-condition elision is explicitly defended with map-keyed `Where` clauses (wish/provider.go:27-39,86-98; serial/serial.go:83-99). Cross-tenant sweep paths take `tenantId` explicitly + `WithoutTenantFilter` (listing/provider.go:198-203; serial.Next passes tenantId explicitly). Holdings are tenant-self-describing from the listing row (`lm.TenantId()`, listing/processor.go:547) so the cross-tenant ticker needs no tenant model.

### External HTTP client — wallet

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| EXT-01 | JSON:API relationship stubs | WARN | wallet/wallet.go RestModel omits `SetToOneReferenceID`/`SetToManyReferenceIDs`; documented as relationship-free (:27-29). Low risk (cashshop wallet payload has no relationships block) but stubs are the checklist default. |
| EXT-02 | httptest integration test | PASS | wallet/wallet_test.go:16,40 `httptest.NewServer` serving wallet JSON; exercises real unmarshal via `requestByAccountId`. |
| EXT-03 | 404 distinguished | PASS | wallet/wallet.go:116 `errors.Is(err, requests.ErrNotFound)`; `PrepaidBalance`/`Balance` bubble transport errors unchanged (no map-all-to-not-found). |
| EXT-04 | RootUrl, not hardcoded | PASS | wallet/wallet.go:52 `requests.RootUrl("CASHSHOP")` |

## Findings (ranked)

### Critical
None introduced by this work.

Context (out of scope, do not re-litigate): the platform `database.ExecuteTransaction` no-op bug (bug_execute_transaction_noop / task-119) means the "one local DB transaction" atomicity claims throughout `listing/custody.go` (`Accept`, `SettleMove`), `holding/custody.go`, `wish/register.go`, and `PlaceBid`/`Cancel` do not actually hold — a mid-tx failure will not roll back prior mutations. This is a pre-existing platform defect, not caused by the refactor, and the refactor faithfully carried the (correct-once-the-platform-is-fixed) transaction structure forward.

### Important

**I-1 — DOM-25: hardcoded clientbound NoticeFailReason byte on cancel + take-home Failed events.**
`mtsFailReasonGeneric byte = 0` (kafka/consumer/mts/consumer.go:84) is passed as a literal wire byte to `ListingCancelFailedStatusEventProvider` (consumer.go:173) and `TakeHomeFailedStatusEventProvider` (consumer.go:253). The event bodies carry it as `Reason byte json:"reason"` (kafka/message/mts/kafka.go:391 `StatusEventListingCancelFailedBody`, :430 `StatusEventTakeHomeFailedBody`), and per the code's own comment (consumer.go:80-84) the channel "passes it straight into the *Failed clientbound codec's Decode1 reason field" — i.e. the domain service emits a client-interpreted wire byte instead of a semantic key resolved channel-side against the tenant `noticeFailReasons` writer table. The sibling failure events were already migrated to the config-driven `ReasonKey string` form: `ListingCreateFailedStatusEventProvider(...reasonKey string)` (producer.go:163), `BuyFailedStatusEventProvider(...reason string...)` (:210), `BidFailedStatusEventProvider(...reason string)` (:227), backed by the `FailReason*` semantic keys (kafka/message/mts/kafka.go:16-24). This is the exact anti-pattern enumerated in `anti-patterns.md` §"Hardcoding client-interpreted wire values" (reviewed as DOM-25), which even names "task-102 NoticeFailReason" as a prior instance; "the value is version-stable" is explicitly non-exempting (task-103 uniformity ruling).
*Failure scenario:* a tenant on a client version whose `CITC::NoticeFailReason` table does not map byte 0 to the intended generic notice shows the wrong cancel/take-home failure text, and it cannot be re-tuned per-version from the seed template the way buy/bid/create can — it requires a code change. Runtime impact is limited today because both paths only ever produce a generic failure, but the two remaining `reason byte` events are the last hardcoded-wire-value holdouts in an otherwise-migrated event surface.
*Fix:* give both events a `ReasonKey string json:"reasonKey,omitempty"` (mirroring create/buy/bid), emit a semantic key (e.g. `FailReasonGeneric`), and resolve it channel-side; delete `mtsFailReasonGeneric`.

### Minor

**M-1 — Consumer layer writes through administrator/provider directly, bypassing the processor.** `transaction.CreateTransaction(...)` is called straight from the mts consumer for best-effort history rows (kafka/consumer/mts/consumer.go:228 cancel, :412 bid-lost), and the custody consumer resolves + consumes a fulfilled want-ad via `wish.GetBySerial(...)` + `wish.DeleteWish(...)` directly (kafka/consumer/custody/consumer.go:290,292). The rest of the refactor pushed logic into processor methods; these post-commit side-effects reach into the administrator/provider layer from the emission layer. `file-responsibilities.md` puts writes behind processor → administrator. Consider a `transaction` processor method and a `wish.Processor.ConsumeFulfilled(...)` for symmetry.

**M-2 — DOM-05: no `TransformSlice` in any package.** `grep 'func TransformSlice'` → 0 matches (listing/wish/holding/transaction rest.go). List handlers use `model.SliceMap(Transform)(...)` inline (wish/resource.go:62,89) — functional composition, not a raw for-loop, so no correctness issue, but it deviates from the documented DOM-05 `TransformSlice` helper.

**M-3 — EXT-01: wallet RestModel omits the no-op relationship stubs.** `SetToOneReferenceID`/`SetToManyReferenceIDs` absent (wallet/wallet.go:30-49). Documented as relationship-free; safe as long as the upstream cashshop wallet payload never grows a `relationships` block, but the checklist wants the stubs present as defense.

**M-4 — DOM-02/03 naming convention.** Domain packages use `modelFromEntity` (provider.go) + inline `entity{...}` assembly in the administrator instead of the documented `Model.ToEntity()` / `Make(Entity)` in entity.go. Service-wide and functionally equivalent; noted for convention consistency only.

## Summary

### Blocking (must fix)
- **I-1 (DOM-25):** cancel + take-home Failed events emit a hardcoded NoticeFailReason wire byte instead of a config-resolved semantic key.

### Non-Blocking (should fix)
- **M-1:** consumer-layer direct administrator/provider writes bypass the processor (history rows, want-ad consume).
- **M-2:** no `TransformSlice` helper (inline `model.SliceMap` used instead).
- **M-3:** wallet RestModel missing no-op JSON:API relationship stubs (EXT-01).
- **M-4:** `modelFromEntity`/inline-entity vs documented `Make`/`ToEntity` naming.

## Final resolution (post-audit fixes)

- **I-1 (Important, DOM-25) — FIXED.** `LISTING_CANCEL_FAILED` and `TAKE_HOME_FAILED` now carry a semantic `ReasonKey` (string, JSON tag `reasonKey`) instead of a raw `Reason` byte; atlas-mts emits `FailReasonGeneric` and the channel resolves it through the tenant `noticeFailReasons` table via `failNoticeOr` (uniform with buy/bid). Removed the `mtsFailReasonGeneric byte` const. `kafka/message/mts/kafka.go`, `kafka/producer/mts/producer.go`, `kafka/consumer/mts/consumer.go`; channel side `kafka/message/mts/kafka.go` + `kafka/consumer/mts/consumer.go`. The topic now carries no numeric `reason` tag at all (collision test rewritten).
- **M-1..M-4 (Minor) — DEFERRED (convention-consistent / low-risk).** Direct administrator/provider calls for post-commit best-effort writes; no `TransformSlice` helper; wallet RestModel relationship stubs; `modelFromEntity` vs `Make`/`ToEntity` naming. All service-wide idioms, not new to this work.
- **ExecuteTransaction no-op (out of scope)** — pre-existing platform bug `bug_execute_transaction_noop` (task-119); not introduced here.
