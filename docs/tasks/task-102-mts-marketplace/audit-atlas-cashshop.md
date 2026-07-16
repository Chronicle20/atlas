# Backend Audit — atlas-cashshop

- **Service Path:** services/atlas-cashshop/atlas.com/cashshop
- **Scope:** task-102 modified packages — `wallet`, `kafka/producer/wallet`, `kafka/message/wallet`, `kafka/consumer/wallet`
- **Guidelines Source:** backend-dev-guidelines skill (DOM-01..25, SUB, FILE-01..06, EXT, SEC)
- **Date:** 2026-07-10
- **Build:** PASS (`go build ./...` exit 0)
- **Vet:** PASS (`go vet` on the 4 packages, exit 0)
- **Tests:** PASS (`wallet` ok, `kafka/producer/wallet` ok; `kafka/message/wallet` and `kafka/consumer/wallet` have no test files)
- **Overall:** NEEDS-WORK (build + tests green, but FAIL checks exist in the `wallet` domain package)

## Build & Test Results

```
go build ./...            -> exit 0
go vet ./wallet/... ./kafka/producer/wallet/... ./kafka/message/wallet/... ./kafka/consumer/wallet/...  -> exit 0
go test  (same set) -count=1:
  ok  atlas-cashshop/wallet                 0.011s
  ok  atlas-cashshop/kafka/producer/wallet  0.008s
  ?   atlas-cashshop/kafka/message/wallet   [no test files]
  ?   atlas-cashshop/kafka/consumer/wallet  [no test files]
```

## Task-102 diff (this service)

- `wallet/processor.go` — added `AdjustCurrency`, `AdjustCurrencyWithTransaction`, `Update*WithTransaction`, `EmitAdjustFailure`.
- `kafka/message/wallet/kafka.go` — added `StatusEventTypeError`, `StatusEventErrorBody`, `CommandTypeAdjustCurrency`, `AdjustCurrencyCommand`.
- `kafka/producer/wallet/producer.go` — added `ErrorStatusEventProvider`, transaction-aware update provider.
- `kafka/producer/wallet/producer_test.go` — new (`TestErrorStatusEventProvider`).
- `kafka/consumer/wallet/consumer.go` — added `handleAdjustCurrencyCommand` + registration.

The other `wallet/*.go` files (builder-absence, provider eagerness, REST error mapping) predate task-102; they are reported because the audit is package-scoped and the FAIL-until-proven mindset applies to the whole package.

---

## Package: `wallet` — DOMAIN (has model.go)

### Domain Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go exists | **FAIL** | No `builder.go` in package; no `NewBuilder`/`type Builder` anywhere (grep NONE). `Model` is assembled with raw struct literals: model.go:55, rest.go:43 (`Extract`), entity.go:33 (`Make`). |
| DOM-02 | `ToEntity()` method | **FAIL** | entity.go defines `Make(Entity)` (entity.go:32) but no `func (m Model) ToEntity()`; grep for `ToEntity` = NONE. |
| DOM-03 | `Make(Entity)` function | PASS | entity.go:32 `func Make(e Entity) (Model, error)`. |
| DOM-04 | `Transform` function | PASS | rest.go:32 `func Transform(m Model) (RestModel, error)`. |
| DOM-05 | `TransformSlice` function | WARN (N/A) | No list endpoint (only GET-by-account, POST, PATCH); no inline transform loops in resource.go. `TransformSlice` genuinely not required, so no violation, but the function is absent. |
| DOM-06 | Processor accepts `FieldLogger` | PASS | processor.go:43 `NewProcessor(l logrus.FieldLogger, ...)`. |
| DOM-07 | Handlers pass `d.Logger()` | PASS | resource.go:32, :61, :86 all `NewProcessor(d.Logger(), d.Context(), db)`; no `StandardLogger()`. |
| DOM-08 | POST/PATCH use `RegisterInputHandler` | PASS | resource.go:22 (POST) & :23 (PATCH) use `rest.RegisterInputHandler[RestModel]`. |
| DOM-09 | Transform errors handled | PASS | resource.go:42-47, :67-72, :92-97 each check the `Transform` error. |
| DOM-10 | Test DB has tenant callbacks | PASS | provider_test.go:19 `databasetest.NewInMemoryTenantDB(t, Migration)` (registers tenant callbacks); tenant isolation proven by `TestWalletProvider_ByAccountId_FiltersByTenant` (provider_test.go:26). |
| DOM-11 | Providers use lazy evaluation | **FAIL** | provider.go:10-19 executes `db.Where("account_id = ?").First(&result)` eagerly and wraps the already-fetched row in `model.FixedProvider`/`model.ErrorProvider` — the exact anti-pattern DOM-11 bans. Should use `database.Query[Entity]`. |
| DOM-12 | No `os.Getenv()` in handlers | PASS | resource.go — zero `os.Getenv`. |
| DOM-13 | No cross-domain logic in handlers | PASS | Handlers call only the wallet processor. |
| DOM-14 | Handlers don't call providers directly | PASS | resource.go calls `NewProcessor(...).GetByAccountId/CreateAndEmit/UpdateAndEmit` only. |
| DOM-15 | No direct entity creation in handlers | PASS | resource.go — no `db.Create/Save/Delete`. |
| DOM-16 | `administrator.go` exists for writes | PASS | administrator.go:8/24/46 `createEntity`/`updateEntity`/`deleteEntity`; called from processor.go:78/102/196. |
| DOM-17 | Domain error → HTTP status mapping | **FAIL** | GET maps `ErrRecordNotFound`→404 (resource.go:33-36) but POST (resource.go:62-65) and PATCH (resource.go:87-90) collapse **every** error to `500`. An update against a missing wallet (`updateEntity` `First` → `ErrRecordNotFound`) returns 500, not 404; no 400 for validation. |
| DOM-18 | JSON:API interface on REST models | PASS | rest.go:15 `GetName()`, :19 `GetID()`, :23 `SetID()`. |
| DOM-19 | Request models use flat structure | PASS | rest.go `RestModel` is flat (no nested Data/Type/Attributes); reused as input model. |
| DOM-20 | Table-driven tests | WARN | model_test.go / rest_test.go / provider_test.go use one-off `Test*` funcs, not `tests := []struct{...}` + `t.Run`. Tests exist and pass; style-only. |
| DOM-21 | No duplication of atlas-constants types | PASS | New types (`StatusEvent*`, `AdjustCurrencyCommand`) are wallet-domain DTOs. `currencyType uint32` with 1/2/3 is a wallet-local currency selector with no atlas-constants equivalent. (Note: 1/2/3 are undocumented magic numbers — see non-blocking notes.) |
| DOM-22 | Dockerfile 4-block per direct lib | N/A | Shared root `Dockerfile` (ARG SERVICE); no own service Dockerfile. task-102 introduced no new `Chronicle20/atlas/libs/*` require (`go.mod` unchanged). |
| DOM-23 | Kafka topic naming | PASS | `COMMAND_TOPIC_WALLET` (kafka.go:7) and `EVENT_TOPIC_WALLET_STATUS` (kafka.go:6) both appear as `KEY: "KEY"` in deploy/k8s/base/env-configmap.yaml:77 & :152. deploy/k8s/base/atlas-cashshop.yaml consumes via `envFrom: configMapRef: atlas-env` (line 21-23) with **no** literal `- name: COMMAND_TOPIC_WALLET / value:` override. |
| DOM-24 | Kafka producer stubbed in emitting tests | PASS | No `wallet` test invokes an emit path — grep for `AndEmit`/`message.Emit` in `wallet/*_test.go` = NONE. Tests exercise pure logic (`Transform`, providers, `Model` accessors) only, so no unstubbed producer hang. |
| DOM-25 | Client wire values config-resolved | PASS (N/A) | wallet is a domain service; it emits semantic string keys (`ERROR`, `UPDATED`) and numeric currency selectors, not client packet bytes. No client wire code as a Go literal. |

### File Responsibilities Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in processor.go | PASS | `type Processor`/`ProcessorImpl`/`NewProcessor` all in processor.go:18/35/43. |
| FILE-02 | RestModel + Transform/Extract + JSON:API in rest.go | PASS | rest.go holds `RestModel`, `Transform`, `Extract`, `GetName/GetID/SetID`. |
| FILE-03 | Cross-service requests in requests.go | N/A | No cross-service HTTP client in this package. |
| FILE-04 | Entity + Migration + TableName in entity.go | PASS | entity.go:12 `Entity`, :8 `Migration`, :21 `TableName`. |
| FILE-05 | Builder/Model/admin/provider/state placement | **FAIL** | Model in model.go (ok), writes in administrator.go (ok), providers in provider.go (ok), **but Builder is entirely absent** — no `builder.go` and no `NewBuilder`. Same root cause as DOM-01. |
| FILE-06 | No package-named catch-all file | PASS | Files are cleanly split (model/entity/builder-missing/administrator/provider/processor/rest/resource); no `wallet.go` bundling responsibilities. |

---

## Package: `kafka/message/wallet` — SUPPORT (message DTOs)

Single non-test file `kafka.go`: topic env constants + `StatusEvent[E]`/body types + `Command`/`AdjustCurrencyCommand`.

| ID | Status | Evidence |
|----|--------|----------|
| FILE-01 Processor | PASS | No processor symbols. |
| FILE-02 RestModel | PASS | No `RestModel`/`Transform`. |
| FILE-03 requests | PASS | No cross-service client. |
| FILE-04 entity | PASS | No GORM entity/Migration/TableName. |
| FILE-05 placement | PASS | No builder/model/admin/provider responsibilities present. |
| FILE-06 catch-all | PASS | `kafka.go` carries only message-package responsibilities (event/command DTOs + topic env keys) — none of the FILE-table responsibilities; package is `wallet`, so `kafka.go` is not a `<pkgname>.go` catch-all. |

No DOM/SUB/EXT/SEC triggers.

---

## Package: `kafka/producer/wallet` — SUPPORT (Kafka message providers)

Files: `producer.go` (`*StatusEventProvider` functions), `producer_test.go`.

| ID | Status | Evidence |
|----|--------|----------|
| FILE-01..06 | PASS | `producer.go` holds only Kafka message-creation providers (the `producer.go` file responsibility); correctly named; no Processor/RestModel/requests/entity/builder collapse. |
| DOM-24 (emit tests) | PASS | producer_test.go:18 invokes `ErrorStatusEventProvider(...)()` which only materializes `[]kafka.Message` — it does **not** call `message.Emit`/`AndEmit`/`producer.Produce`. No unstubbed producer path. |

No other DOM/SUB/EXT/SEC triggers.

---

## Package: `kafka/consumer/wallet` — SUB-DOMAIN (no model.go, action-event consumer)

Single non-test file `consumer.go`: `InitConsumers`, `InitHandlers`, `handleAdjustCurrencyCommand`.

### Sub-Domain Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 | Uses processor for business logic | PASS | consumer.go:48 `wallet2.NewProcessor(l, ctx, db)`; the handler is thin and delegates to `AdjustCurrencyWithTransaction` (processor.go:150). |
| SUB-02 | Administrator handles writes | PASS | Writes flow domain processor → administrator (`updateEntity`); no `db.Create`/`db.Save`/`db.Delete` in consumer.go. |
| SUB-03 | RegisterInputHandler for POST | N/A | Kafka consumer; no REST POST endpoint. |
| SUB-04 | No manual JSON parsing | PASS | Uses `message.AdaptHandler`/typed `message.Handler[AdjustCurrencyCommand]` (consumer.go:32,40); no `json.Unmarshal`/`json.NewDecoder`/`io.ReadAll`. |

### File Responsibilities

| ID | Status | Evidence |
|----|--------|----------|
| FILE-01..06 | PASS | `consumer.go` holds only consumer registration + one handler; no FILE-table responsibility collapse; not a `<pkgname>.go` catch-all. |

**Non-blocking note:** `handleAdjustCurrencyCommand` transitively emits (`AdjustCurrencyWithTransaction` → `UpdateAndEmitWithTransaction` → `message.Emit`, and `EmitAdjustFailure` → `message.Emit`), but the package has **no test file**, so DOM-24 is not triggered. The failure-fast emit path (the core of the task-102 change) is therefore untested at the consumer boundary.

---

## Security Review

Not an auth/token/authorization service. SEC-01..03 N/A. SEC-04 (no hardcoded secrets): PASS — no keys/passwords in the audited packages.

## External HTTP Client

EXT-01..04 N/A — none of the four packages call another atlas service (no `requests.RootUrl`/`GetRequest`/`PostRequest`).

---

## Summary

### Blocking (must fix) — Important
- **DOM-01 / FILE-05:** `wallet` package has no `builder.go`; `Model` is built with raw struct literals (model.go:55, rest.go:43, entity.go:33). No validated `Build()`.
- **DOM-02:** No `Model.ToEntity()` in wallet/entity.go — only the reverse `Make(Entity)`.
- **DOM-11:** wallet/provider.go:10-19 runs an eager `First()` wrapped in `FixedProvider`/`ErrorProvider` instead of `database.Query[Entity]` (lazy).
- **DOM-17:** wallet POST/PATCH handlers (resource.go:62-65, :87-90) map all errors to 500; not-found on update should be 404, validation 400.

### Non-Blocking (should fix) — Minor / WARN
- **DOM-20:** wallet tests are one-off funcs, not table-driven (`tests := []struct{}` + `t.Run`).
- **DOM-05:** no `TransformSlice` (acceptable — no list endpoint).
- **DOM-21 note:** `currencyType` 1/2/3 (processor.go:163-185, model.go:33-53) are undocumented magic numbers; consider a named `type`/consts (no atlas-constants equivalent exists).
- **Consumer emit path untested:** `kafka/consumer/wallet` has no test covering the task-102 adjust/failure-fast emit flow.

None of the FAILs were introduced by task-102 (they predate it in the untouched wallet files), but all are live package-level violations under the FAIL-until-proven mindset.
