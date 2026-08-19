# Backend Audit — task-240-cash-shop-stub-operations (atlas-cashshop / atlas-channel)

- **Service Path:** `services/atlas-cashshop`, `services/atlas-channel`
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-19
- **Range:** `d9ec287b8..3bc7ebd21`, changed Go files under `services/atlas-cashshop` and `services/atlas-channel` only (88 files)
- **Build:** PASS
- **Tests:** all packages `ok` (0 failures) in both modules
- **Overall:** NEEDS-WORK

## Build & Test Results

```
$ cd services/atlas-cashshop/atlas.com/cashshop && go build ./...        -> exit 0, no output
$ cd services/atlas-cashshop/atlas.com/cashshop && go test ./... -count=1 -> all `ok` or `[no test files]`, 0 FAIL
$ cd services/atlas-channel/atlas.com/channel && go build ./...          -> exit 0, no output
$ cd services/atlas-channel/atlas.com/channel && go test ./... -count=1  -> all `ok` or `[no test files]`, 0 FAIL
$ tools/goroutine-guard.sh                                                -> exit 0 (88 modules)
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| FILE-01..06 | Yes | every changed package audited |
| DOM structure (DOM-01,02,03,04,05,11,16) | Yes | `ring/` has `model.go`+`entity.go`+`rest.go`; `purchaserecord/` has `entity.go` (no `model.go`, see SUB) |
| SUB-01..04 | Yes | `purchaserecord/` has `resource.go`, no `model.go` |
| REST (DOM-06..09,12..15,17..19,32) | Yes | `ring/resource.go`, `purchaserecord/resource.go`, `data/cashpackage/rest.go` etc. |
| Constants reuse (DOM-21) | Yes | new `ring.Type`/`ring.State`, `walletCurrency*` consts, new Kafka command/event type strings |
| Testing (DOM-10,20,24,33) | Yes | diff adds/changes many `_test.go` files; several reach `AndEmit`/`message.Emit`/`producer.Produce` |
| Cache (DOM-29) | N/A | no `cache.go` added or changed; no processor holds cached state |
| Messaging (DOM-30) | Yes | `gift.go`/`package.go`/`rebate.go`/`ring.go`/`equipslot.go` all emit via `message.Emit`+`outbox.EmitProvider` |
| Multi-tenancy (DOM-31) | Yes | many new `resource.go`/`rest.go`, tenant reads throughout |
| Migration hygiene (DOM-34,35) | N/A | diff does not move/extract symbols to/from `libs/atlas-*` |
| Deploy & topics (DOM-22,23) | N/A | no `libs/atlas-*` module added; no new Kafka topic **env var** (new command/event *type strings* reuse existing `EnvCommandTopic`/`EnvEventTopicStatus`) |
| Runtime safety (DOM-26) | Yes | many non-test files changed; `tools/goroutine-guard.sh` exit 0 |
| Channel wire values (DOM-25) | Yes | diff touches `services/atlas-channel`; verified mode-byte routing and the `slotIndex=0` value (IDB-derived, see below) |
| Resilience (DOM-27,28) | Yes | new resource handlers (`ring`, `purchaserecord`) call `restserver.WriteErrorResponse`; `character_data.go` gained a new remote-fetch enrichment fallback |
| External clients (EXT-01..04) | Yes | 4 new packages call another atlas service via `requests.*Request[T]` |
| Scaffolding (SCAFFOLD-01..09) | N/A | no new `services/atlas-<svc>/`, no new channel `Writer`/`Handler` registration, no `routes.conf` change in this diff's file set |
| Security (SEC-01..04) | Partially | new secondary-credential (PIC/birthday) gate; not JWT/redirect — SEC-01/02/03 N/A, SEC-04 checked |

## Checklist Results

### `cashshop/cashshop` (domain — has `model.go` via `processor.go`'s Model type is elsewhere; package itself is the core `Processor`/`ProcessorImpl` home)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor iface/ctor/`ProcessorImpl` methods in `processor.go` or `processor_<group>.go` | **FAIL (Important)** | `ProcessorImpl` methods sit in bare topic-named files, not `processor.go` nor a `processor_`-prefixed split: `GiftAndEmit` in `cashshop/gift.go:51`, `PurchasePackageAndEmit` in `cashshop/package.go:59`, `RebateAndEmit` in `cashshop/rebate.go:57`, `PurchaseRingAndEmit` in `cashshop/ring.go:74`, `PurchaseEquipSlotAndEmit`/`CompleteEquipSlotExtension` in `cashshop/equipslot.go:46,163`. The `Processor` interface itself is correctly in `processor.go:93-104`. `file-responsibilities.md`'s own FAIL example is "a bare topic-named file like `custody.go` / `register.go`" — these five files are exactly that shape (missing the `processor_` prefix the idiomatic split requires). |
| DOM-30 | Writes emit via `AndEmit`+`message.Buffer`, not a direct producer call on the success path | PASS | `cashshop/gift.go:67` `message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(...)`; direct `producer.ProviderImpl` only used on the rejection branch (`gift.go:70`), matching the existing `errPurchaseRejected` pattern in `processor.go`. Same shape in `package.go`, `rebate.go`, `ring.go`, `equipslot.go`. |
| DOM-21 | No redeclaration of an existing `libs/atlas-constants` type/const | PASS | `processor.go:~2247` documents the check explicitly: "No entry in libs/atlas-constants covers wallet currency buckets, so these are defined here"; `equipslot.go:39` reuses `libs/atlas-constants/inventory/slot` (`slot.GetSlotByType("pendant2")`) rather than inventing a slot constant. |
| DOM-25 | Client wire values not hardcoded outside a resolved table | PASS (with derivation evidence) | `handleStatusEventEquipSlotIncreased` passes a literal `0` for the packet-level `slotIndex` argument (`services/atlas-channel/.../kafka/consumer/cashshop/consumer.go`, `cashpkt.CashShopEnableEquipSlotExtSuccessBody(0, e.Body.Days)`), but this is IDB-derived — not invented: `docs/tasks/task-240-cash-shop-stub-operations/derivation-equip-slot.md:61-168,508` documents three independent cross-checks (an equality-test bound in the client, and `aEquipExtExpire` being a 1-element array) that pin the wire value at exactly 0 for every version. Failure routing itself is resolved through the existing `errors`-table key mechanism (`cashcb.CodeConfigured`), not a literal. |
| Stub-vs-unimplemented (task framing) | The ring couple/friendship same-template-only path | Judged compliant, not a CLAUDE.md violation | `cashshop/ring.go:56-68` explicitly documents that a distinct-template couple ring has "nothing to detect and branch on" today and reserves a typed `COUPLE_FAILED`/`FRIENDSHIP_FAILED` for when that data exists — this matches `design.md §4.3/§9`'s explicitly sanctioned "land the verifiable half plus a typed failure" outcome for an unresolved derivation (OQ-R1), not a stubbed/unimplemented status response. |

### `cashshop/character` (REST-client, support package — no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-02 | `RestModel`/`Transform`/`GetName`/`GetID`/`SetID` live in `rest.go` | **FAIL (Important)** | New `EquipSlotExtensionRestModel` and `ExtendEquipSlotInputRestModel`, with their `GetName()`/`GetID()`/`SetID()` methods, are defined in `character/equipslot.go:25-71` instead of the package's existing `rest.go`. |
| FILE-03 | Cross-service request functions live in `requests.go` | **FAIL (Important)** | `requestExtendEquipSlot` and the `EquipSlotExtensions` path const are defined in `character/equipslot.go:17,73-80` instead of the package's existing `requests.go`. |
| FILE-06 | No single file carrying ≥2 of the responsibilities above | **FAIL (Important)** | `character/equipslot.go` alone carries both the REST-model responsibility (FILE-02) and the requests responsibility (FILE-03) — a bare topic-named file collapsing two roles, the exact shape FILE-06 forbids. |
| EXT-01 | Target RestModel implements `SetToOneReferenceID`/`SetToManyReferenceIDs` | **FAIL (Important)** | `EquipSlotExtensionRestModel` (`character/equipslot.go:25-31`) has none. Grep for both methods in the file returns no match. |
| EXT-02 | httptest-backed integration test with a representative fixture | **FAIL (Important)** | No `_test.go` exists for this package at all (`ls services/atlas-cashshop/atlas.com/cashshop/character/` shows only `equipslot.go`, `model.go`, `processor.go`, `requests.go`, `rest.go`). |
| EXT-03 | Only genuine 404s map to "not found"; other failures bubble with original error | PASS (vacuous) | `ExtendEquipSlot` (`character/processor.go:59-61`) returns the raw `requests.Provider` error unmodified — no domain "not found" concept exists to misapply. |
| EXT-04 | URL composed via `requests.RootUrl`/`RootUrlFor`, not hardcoded DNS | PASS | `character/requests.go:16` `requests.RootUrlFor(ctx, "CHARACTERS")`, reused by `requestExtendEquipSlot` in `equipslot.go:75`. |

### `ring/` (new domain — has `model.go`+`entity.go`+`rest.go`+`resource.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` with `NewBuilder()`/`Build()` | **FAIL (Important)** | No `builder.go` in the package (`ls services/atlas-cashshop/atlas.com/cashshop/ring/`); no `NewBuilder` anywhere (`grep -rn NewBuilder ring/` → no match). `administrator.go:CreatePair` constructs `Entity{}` literals directly instead of going through a validating builder. |
| DOM-02 | `Model.ToEntity()` in `entity.go` | **FAIL (Important)** | `grep -n ToEntity ring/` → no match. `entity.go` defines only `Make(Entity) (Model, error)` (DOM-03); the reverse direction does not exist, and `administrator.go:27-49` hand-builds `Entity{}` literals instead. |
| DOM-03 | `Make(Entity) (Model, error)` in `entity.go` | PASS | `ring/entity.go:38-49`. |
| DOM-04 | `Transform(Model) (RestModel, error)` in `rest.go` | PASS | `ring/rest.go:52-62`. |
| DOM-05 | `TransformSlice` defined in `rest.go`, used by list handlers | **FAIL (Minor)** | No `TransformSlice` function exists in `ring/rest.go` (or anywhere in the service — `grep -rn "func TransformSlice" cashshop/` returns nothing). The list handler `handleGetRings` (`ring/resource.go:76`) uses `model.SliceMap(Transform)(...)` instead — not an inline `for` loop, but not the required named function either. Downgraded to Minor because the composition is equivalent in effect and mirrors the service's pre-existing `coupon/redemption/resource.go:88` idiom, but the rule as written requires the named function. |
| DOM-11 | Providers use `database.Query`/`SliceQuery`, not eager `FixedProvider` | PASS | `ring/provider.go:11-30` — curried functions returning `database.EntityProvider[T]`, lazily evaluated. |
| DOM-16 | Writes live in `administrator.go` | PASS | `ring/administrator.go:CreatePair`, called only from `ring/processor.go:41`. |
| FILE-05 | Readers in `provider.go`, writes in `administrator.go` | **FAIL (Important)** | `byCharacterIdPagedProvider` — a `database.EntityProvider[model.Paged[Entity]]` reader — is defined in `ring/resource.go:27-31` instead of `ring/provider.go`, where `byCharacterIdProvider`/`byIdProvider` already live. The file's own comment (`resource.go:22-26`) states this is deliberate ("stays out of that file while its own review is in flight"), which is a review-process note, not a documented guideline exception. |
| DOM-09 | Every `Transform(` call site checks its error | PASS | `ring/resource.go:75-79` (`res, err := model.SliceMap(Transform)(...)(); if err != nil`) and `:112-116`. |
| DOM-17 | Domain errors map to specific HTTP status | PASS | `ring/resource.go:104-107` maps `gorm.ErrRecordNotFound` to 404 explicitly, everything else to `WriteErrorResponse`. |
| DOM-18 | RestModel implements `GetName`/`GetID`/`SetID` | PASS | `ring/rest.go:29-49`. |
| DOM-27 | DB-backed handler error branches use `WriteErrorResponse` | PASS | `ring/resource.go:69,83,107,111` — no direct `w.WriteHeader(http.StatusInternalServerError)`. |
| DOM-31 | Tenant/trace never in REST model or query/path param | PASS | `RestModel` (`ring/rest.go:20-28`) carries no `TenantId`; `characterId` is a legitimate domain filter, not a tenant identifier; tenant is read only from context (`ring/resource.go:66,101`). |
| DOM-10 | Test DB setup calls `RegisterTenantCallbacks` | PASS | Via shared helper: `ring/administrator_test.go:17` and `ring/resource_test.go:44` call `databasetest.NewInMemoryTenantDB`, which itself calls `database.RegisterTenantCallbacks(l, db)` (`libs/atlas-database/databasetest/testdb.go:39`). |

### `purchaserecord/` (sub-domain — has `resource.go`, no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 | Business logic in processor, not handler | PASS | `resource.go:35` handler calls `NewProcessor(...).Get(...)`; no inline logic. |
| SUB-02 | No `db.Create`/`db.Save` in `resource.go` | PASS | `grep -n "db.Create\|db.Save" purchaserecord/resource.go` → no match. |
| SUB-03 | POST routes via `RegisterInputHandler[T]` | N/A | Package registers only `GET /accounts/{accountId}/purchaseRecords/{serialNumber}` — no POST route. |
| SUB-04 | No manual JSON parsing in `resource.go` | PASS | `grep -n "json.NewDecoder\|json.Unmarshal\|io.ReadAll" purchaserecord/resource.go` → no match. |
| FILE-05 | Readers in `provider.go`, writes in `administrator.go` | **FAIL (Important)** | `Get(db, tenantId, accountId, serialNumber)` — a plain read — is defined in `purchaserecord/administrator.go:44-53`, alongside the write function `Record`. No `provider.go` exists in the package. |
| DOM-31 | Tenant never in REST model or query/path param | PASS | `RestModel` (`purchaserecord/rest.go:8-13`) carries no `TenantId`; `accountId`/`serialNumber` are legitimate path params, not tenant identifiers. |
| DOM-10 | Test DB setup calls `RegisterTenantCallbacks` | PASS | `purchaserecord/administrator_test.go:16` via `databasetest.NewInMemoryTenantDB` (same shared helper as above). |

### `data/cashpackage/` (new REST-client, support package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| EXT-01 | `SetToOneReferenceID`/`SetToManyReferenceIDs` on `RestModel` | **FAIL (Important)** | `data/cashpackage/rest.go:5-11` has no such methods; `grep -n "SetToOneReferenceID\|SetToManyReferenceIDs"` → no match. |
| EXT-02 | httptest-backed integration test | **FAIL (Important)** | No `_test.go` exists in the package at all. |
| EXT-03 | Only genuine 404s map to "not found" | PASS (vacuous) | `data/cashpackage/processor.go:29` returns the raw error unmodified; no domain "not found" mapping exists to misapply. |
| EXT-04 | URL via `requests.RootUrlFor` | PASS | `data/cashpackage/requests.go:16` `requests.RootUrlFor(ctx, "DATA")`. |
| FILE-01/02/03/04/05 | Each responsibility in its own file | PASS | `model.go`, `processor.go`, `requests.go`, `rest.go` each hold exactly their documented responsibility; no `data/cashpackage/cashpackage.go` catch-all exists. |

### `services/atlas-channel/.../character/equipslot/` (new REST-client, support package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| EXT-01 | `SetToOneReferenceID`/`SetToManyReferenceIDs` on `RestModel` | **FAIL (Important)** | `channel/character/equipslot/rest.go:10-16` has no such methods. |
| EXT-02 | httptest-backed integration test | **FAIL (Important)** | No `_test.go` in the package. |
| EXT-03 | Only genuine 404s map to "not found" | PASS (vacuous) | `equipslot/processor.go:32` returns the raw error from `requests.SliceProvider` unmodified. |
| EXT-04 | URL via `requests.RootUrlFor` | PASS | `equipslot/requests.go:14` `requests.RootUrlFor(ctx, "CHARACTERS")`. |
| FILE-01/02/03 | Each responsibility in its own file | PASS | `processor.go`, `requests.go`, `rest.go` each hold exactly their documented responsibility. |

### `services/atlas-channel/.../cashshop/purchaserecord/` (new REST-client, support package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| EXT-01 | `SetToOneReferenceID`/`SetToManyReferenceIDs` on `RestModel` | **FAIL (Important)** | `channel/cashshop/purchaserecord/rest.go:8-14` has no such methods. |
| EXT-02 | httptest-backed integration test | **FAIL (Important)** | No `_test.go` in the package. |
| EXT-03 | Only genuine 404s map to "not found" | PASS (vacuous) | `channel/cashshop/purchaserecord/processor.go` returns the raw error unmodified. |
| EXT-04 | URL via `requests.RootUrlFor` | PASS | `channel/cashshop/purchaserecord/requests.go` composes against `requests.RootUrlFor(ctx, "CASHSHOP")` (verified by reading the file). |
| FILE-01/02/03/05 | Each responsibility in its own file | PASS | `model.go`, `processor.go`, `requests.go`, `rest.go` each hold exactly their documented responsibility. |

### `services/atlas-channel/.../socket/writer` (`character_data.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-28 | Fallible enrichment fetching remote data degrades loudly (`model.ErrDecorator`+`degrade.Observe`) | **FAIL (Important)** | `buildInventoryData` (`socket/writer/character_data.go:120-126`) fetches equip-slot extensions over REST (`equipslot.NewProcessor(l, ctx).GetActive(c.Id())`); on error it logs `l.WithError(err).Warnf(...)` and silently falls back to `exts = nil` (`:125-126`) — it neither propagates the error nor calls `degrade.Observe(...)`/`model.ErrDecorator`. `grep -n "degrade\.\|ErrDecorator" character_data.go` → no match. |
| DOM-25 | Client wire values resolved from a table, not hardcoded | PASS | The new `EquipSlotExtExpire` field is a computed `FILETIME` timestamp (`packetmodel.MsTime`), not a dispatcher/mode/reason code; no literal client code introduced here. |

### `services/atlas-channel/.../kafka/consumer/cashshop/consumer.go`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 | Client wire values resolved from a table / IDB-verified, not invented | PASS | `handleStatusEventEquipSlotIncreased`'s literal `0` for wire `slotIndex` is IDB-derived (see `cashshop/cashshop` row above); `failureBodyForOperation` (consumer.go:298-318) routes purely on `ErrorEventBody.Operation`, matching `design.md §7` exactly, `default:` reproduces prior behavior byte-for-byte. |
| DOM-30 / event routing | Result bodies picked via existing bound writer-options tables | PASS | Every `session.Announce(...)(cashpkt.CashShopOperationWriter)(...)` call passes a body constructor already present in `libs/atlas-packet` per `design.md §0`; no new codec introduced in this diff. |

### `services/atlas-channel/.../socket/handler/cash_shop_credential.go`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SEC-01 (JWT parsing verified) | N/A | trigger did not fire — this gate compares a stored PIC/birthday, not a JWT. |
| SEC-02 (revocation reads validated claims) | N/A | no revocation/logout path touched. |
| SEC-03 (redirect validation) | N/A | no redirect/callback handler touched. |
| SEC-04 (no hardcoded secrets) | PASS | `cash_shop_credential.go` contains no literal secret; PIC/birthday are read from the account service per request, matching the cited `character_selected_pic.go:49` precedent. |

### Testing family — cross-cutting

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-24 | Test packages reaching an emit path install the shared `producertest` stub | **FAIL (Important)** for one file; PASS elsewhere | `atlas-cashshop/cashshop` package: PASS — `processor_test.go:52` `TestMain` calls `producertest.InstallCapturing()` once for the whole package, covering `gift_test.go`, `ring_test.go`, `rebate_test.go`, `package_test.go`, `equipslot_test.go` (same `package cashshop`). `atlas-channel/socket/handler` package: `cash_shop_buy_normal_test.go:66,111` (new in this diff) reaches the emit path via `RequestPurchase`/`CashShopOperationHandleFunc`, using the pre-existing helper `installCapturingProducer()` (`cash_item_gachapon_test.go:50-58`). That helper installs a hand-rolled `capturingWriter` via `kafkaproducer.ConfigWriterFactory` directly and only calls `producertest.InstallNoop()` on cleanup — `testing-guide.md`'s own pass criteria states explicitly: "A service-local `noopWriter`/`testkafka` helper does NOT satisfy it, even one that calls `ConfigWriterFactory` correctly — the shared `producertest` package is the single source of truth." The helper predates this diff (used by 6 other pre-existing test files too), but the new test file's reliance on it is an in-scope instance of the non-compliant pattern, not exempted by its prevalence. |
| DOM-20 | Table-driven tests (`tests := []struct{...}` + `t.Run`) | WARN | Several new test files (`gift_test.go`, `ring_test.go`, `rebate_test.go`, `package_test.go`, `ring/administrator_test.go`) use named `t.Run(...)` subtests but not a `tests := []struct{...}` data table — each scenario needs materially different DB fixtures, which the table shape does not accommodate cleanly. `TestBuyNormalPurchaseCurrency` (`cash_shop_buy_normal_test.go:52-91`) IS a proper `tests := []struct{...}` table. Not escalated to FAIL: DOM-20's own text allows a more specific per-file playbook to govern, and no such playbook forbids per-scenario subtests for stateful integration tests; flagged WARN because the letter of the generic rule is not met. |
| DOM-33 | Interface changes updated in every mock in the same diff | N/A | `cashshop.Processor` (`channel/cashshop/processor.go`) gained 5 methods; no mock of this interface exists anywhere in the service (`grep -rln "cashshop.Processor" channel/` → only a comment reference in a test file). Handler tests fake the Kafka producer, not the processor interface. `ring.Processor`/`purchaserecord.Processor` are new interfaces, not changed ones — DOM-33 does not apply to a brand-new interface. |

## Security Review

Trigger: the secondary-credential (PIC/birthday) gate is new authentication-adjacent logic.

- `verifySecondaryCredential` (`services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_credential.go:37-59`) resolves the account, compares PIC/birthday in plaintext, matching the cited pre-existing precedent `character_selected_pic.go:49`. Gate-passes-when-unset behavior matches `design.md §2` step 4 exactly, with a Debug-level log (`cash_shop_credential.go:50`).
- Failed attempts are recorded via the existing `account.Processor.RecordPicAttempt` (`cash_shop_credential.go:55`), matching `design.md §2`'s stated free rate-limiting integration.
- No hardcoded secret literals found in the changed files (SEC-04 PASS).
- SEC-01/02/03 do not apply — no JWT parsing, no revocation path, no redirect handler in the diff's file set.

## Not evaluable from the diff

- **SCAFFOLD-07 (tenant opcode template seeding for `gms_95` CashShopOpen / new channel-consumed event types):** `design.md §10` describes deriving and adding a v95 opcode registration to `template_gms_95_1.json` under `services/atlas-configurations`. That directory is out of this review's scope per the dispatch instructions (a separate reviewer covers atlas-configurations); would need to read the seed templates directly to confirm the registration landed and to check `errors`/`operations` table keys this diff's new failure paths (`INVALID_BIRTHDAY`, `CANNOT_GIFT_RECIPIENT_INVENTORY_FULL`, etc.) are actually bound in every targeted version's template.
- **atlas-character write route for `ExtendEquipSlot` (`services/atlas-character/.../equipslot/`):** `design.md §9`/`derivation-equip-slot.md` describe a write-side domain in atlas-character that this diff's `cashshop/character/processor.go:ExtendEquipSlot` and `channel/character/equipslot` read client call into. Out of scope per the dispatch instructions (a separate reviewer covers atlas-character); would need to read that service's `equipslot/resource.go` to confirm the POST route, dedupe-on-`transactionId` behavior, and canonical slot-index handling this diff's clients assume.
- **`libs/atlas-constants` sweep for `RingType`/ring-pairing concepts beyond `ClassificationRing`:** the code comment (`ring/model.go:11-13`) asserts no existing equivalent, but a full independent sweep of `libs/atlas-constants/` was not performed — out of this review's file-scope; treated as adequately evidenced by the in-code rationale rather than independently re-derived.
- **IDB-derivation correctness itself** (the `slotIndex = 0` / body-part `K=59` claims in `derivation-equip-slot.md`): taken as given from the committed derivation document per this review's evidence bar; re-deriving from the IDB directly was not performed.

## Summary

### Blocking (must fix)

- FILE-01: `ProcessorImpl` methods for gift/package/rebate/ring/equip-slot arms sit in bare topic-named files instead of `processor.go`/`processor_<group>.go` — `services/atlas-cashshop/atlas.com/cashshop/cashshop/{gift,package,rebate,ring,equipslot}.go`
- FILE-02/FILE-03/FILE-06: `services/atlas-cashshop/atlas.com/cashshop/character/equipslot.go` collapses RestModel + request-function responsibilities into one bare file
- FILE-05: `services/atlas-cashshop/atlas.com/cashshop/ring/resource.go:27-31` defines a DB reader outside `provider.go`
- FILE-05: `services/atlas-cashshop/atlas.com/cashshop/purchaserecord/administrator.go:44-53` mixes a reader into the write file, no `provider.go` exists
- DOM-01: `services/atlas-cashshop/atlas.com/cashshop/ring/` has no `builder.go`/`NewBuilder()`
- DOM-02: `services/atlas-cashshop/atlas.com/cashshop/ring/entity.go` has no `Model.ToEntity()`
- EXT-01: 4 new REST-client `RestModel`s missing `SetToOneReferenceID`/`SetToManyReferenceIDs` — `data/cashpackage/rest.go`, `cashshop/character/equipslot.go`, `services/atlas-channel/.../character/equipslot/rest.go`, `services/atlas-channel/.../cashshop/purchaserecord/rest.go`
- EXT-02: same 4 packages have zero tests, no httptest fixture coverage
- DOM-28: `services/atlas-channel/atlas.com/channel/socket/writer/character_data.go:120-126` — remote-fetch enrichment fallback logs but never calls `degrade.Observe`/`model.ErrDecorator`
- DOM-24: `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_buy_normal_test.go` reaches the Kafka emit path via a service-local capturing writer instead of the shared `producertest` package

### Non-Blocking (should fix)

- DOM-05: `services/atlas-cashshop/atlas.com/cashshop/ring/resource.go:76` uses `model.SliceMap(Transform)` in the list handler instead of a defined `TransformSlice` in `rest.go`
- DOM-20: several new `_test.go` files use named `t.Run` subtests rather than the `tests := []struct{...}` table shape
