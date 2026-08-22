# Backend Audit — atlas-parcel

- **Service Path:** services/atlas-parcel (new service, task-241)
- **Also audited (DOM-21 constants-reuse lens only):** `libs/atlas-constants`, `libs/atlas-saga`, `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/parcel`
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-19
- **Build:** PASS (`atlas-parcel`: `go build ./...` clean; corroborating: `libs/atlas-saga` and `atlas-saga-orchestrator` both build clean)
- **Tests:** atlas-parcel: all packages `ok` (`atlas-parcel/parcel`, `atlas-parcel/kafka/consumer/custody`); `libs/atlas-saga`: `ok`. No failures.
- **Overall:** NEEDS-WORK

## Build & Test Results

```
$ cd services/atlas-parcel/atlas.com/parcel && go build ./...
(clean, no output)

$ go test ./... -count=1
?   	atlas-parcel	[no test files]
?   	atlas-parcel/kafka/consumer	[no test files]
ok  	atlas-parcel/kafka/consumer/custody	0.024s
?   	atlas-parcel/kafka/message/custody	[no test files]
?   	atlas-parcel/kafka/message/parcel	[no test files]
?   	atlas-parcel/kafka/producer/custody	[no test files]
?   	atlas-parcel/kafka/producer/parcel	[no test files]
ok  	atlas-parcel/parcel	0.082s
?   	atlas-parcel/rest	[no test files]
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | Yes | `parcel` package has `model.go`, `entity.go`, `rest.go`, `provider.go` |
| FILE placement (FILE-01..06) | Yes | Every changed Go package audited: `parcel`, `rest`, `kafka/consumer`, `kafka/consumer/custody`, `kafka/message/custody`, `kafka/message/parcel`, `kafka/producer/custody`, `kafka/producer/parcel`, `main` |
| SUB sub-domain (SUB-01..04) | No | No changed package has `resource.go` without `model.go` — the only `resource.go` (`parcel/resource.go`) sits beside `parcel/model.go` |
| REST (DOM-06..09,12..15,17..19,32) | Yes | `parcel` has `resource.go`, `rest.go`, `processor.go`; registers HTTP routes |
| Constants reuse (DOM-21) | Yes | New types/consts declared: `parcel.Status*`, `parcel.AssetData`, saga `Action`/`Type` consts, `AcceptToParcelPayload` et al. in `libs/atlas-saga` |
| Testing (DOM-10,20,24,33) | Yes | Diff adds/changes `_test.go` files throughout; `saga.Handler` interface gains methods |
| Cache (DOM-29) | No | No `cache.go`; no processor/struct holds cached state |
| Messaging (DOM-30) | Yes | `parcel/notification_task.go` and `kafka/consumer/custody/consumer.go` call `producer.ProviderImpl` |
| Multi-tenancy (DOM-31) | Yes | `parcel` has `rest.go`; multiple readers/writers use `tenant.MustFromContext` / `db.WithContext(ctx)` |
| Migration hygiene (DOM-34,35) | Partial | `AssetData` is explicitly documented as "copied verbatim" (a move) from `services/atlas-merchant/.../asset/kafka.go`, not an extraction between a service and a `libs/atlas-*` module — trigger (service ⇄ `libs/atlas-*`) does not fire; N/A |
| Deploy & topics (DOM-22,23) | Yes | New service adds no `libs/atlas-*` module; adds 3 Kafka topic env vars |
| Runtime safety (DOM-26) | Yes | Non-test Go files changed throughout |
| Channel wire values (DOM-25) | No | Diff (in this scope) does not touch `services/atlas-channel` or `libs/atlas-packet`; parcel service emits semantic Kafka event types (`PARCEL_ARRIVED`), not client-interpreted bytes |
| Resilience (DOM-27,28) | Yes (27) / No (28) | Service calls `database.Connect`; no `model.Decorator` / enrichment path changed |
| External clients (EXT-01..04) | No | `atlas-parcel` makes zero `requests.RootUrl` / `requests.*Request[T]` calls |
| Scaffolding (SCAFFOLD-01..09) | Yes | Diff adds `services/atlas-parcel/` |
| Security (SEC-01..04) | No | Service handles no auth/tokens/redirects/secrets |

## Checklist Results

### parcel (domain package — has `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` with `NewBuilder()`, fluent setters, validating `Build()` | PASS | `parcel/builder.go:41-214` — `NewBuilder`, `Set*` chain, `validate()`+`Build()` |
| DOM-02 | `Model.ToEntity()` in `entity.go` | FAIL | No `ToEntity()` method exists anywhere. The Model→Entity mapping is a free function `entityFromModel(m Model) Entity` in `parcel/builder.go:253-279`, not a method on `Model`, and not in `entity.go`. |
| DOM-03 | `Make(Entity) (Model, error)` in `entity.go` | FAIL | `Make` exists (`parcel/builder.go:219-246`) but is defined in `builder.go`, not `entity.go` as the rule requires. |
| DOM-04 | `Transform(Model) (RestModel, error)` in `rest.go` | PASS | `parcel/rest.go:61-87` |
| DOM-05 | `TransformSlice` in `rest.go`, used by list handlers | FAIL | `rest.go` defines no `TransformSlice`. The list handler `writeParcels` (`parcel/resource.go:140-151`) calls `model.SliceMap(Transform)(model.FixedProvider(ms))(model.ParallelMap())()` directly instead. |
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `parcel/processor.go:42` `NewProcessor(l logrus.FieldLogger, ...)` |
| DOM-07 | Handlers pass `d.Logger()` into `NewProcessor` | PASS | `parcel/resource.go:81,165,207,251,262,279` all use `NewProcessor(d.Logger(), d.Context(), d.DB())` |
| DOM-08 | POST/PATCH routes via `RegisterInputHandler[T]` | FAIL | `PATCH /parcels/{parcelId}/notify` is registered via `registerGet` (`rest.RegisterHandler`), not `RegisterInputHandler[T]`: `parcel/resource.go:52`. (The sibling `PATCH /parcels/{parcelId}` discard route correctly uses `registerPatch := rest.RegisterInputHandler[DiscardRestModel]`, `resource.go:46,51`.) |
| DOM-09 | Every `Transform(` call site checks its error | PASS | `parcel/resource.go:141-146,176-181,222-227` all check `err != nil` before using the result |
| DOM-11 | Providers lazy via `database.Query`/`SliceQuery`, not eager+`FixedProvider` | FAIL | Every function in `parcel/provider.go` (`ById` L17-26, `ByRecipient` L33-46, `BySender` L50-62, `ReceivableByRecipient` L66-79, `ReceivableByRecipientAnyWorld` L88-100) executes `db.Where(...).Find(&results).Error` / `.First(&e).Error` eagerly inside the returned closure and only then wraps the already-fetched result in `model.FixedProvider(...)` — exactly the anti-pattern the rule prohibits. No use of `database.Query`/`database.SliceQuery` anywhere in the file. |
| DOM-12 | No `os.Getenv()` in handlers | PASS | No `os.Getenv` in `parcel/resource.go` |
| DOM-13 | No cross-domain orchestration in handlers | PASS | Every handler in `resource.go` calls only this package's own `Processor` |
| DOM-14 | Handlers call processor methods, never providers | PASS | No direct provider call sites in `resource.go` |
| DOM-15 | No `db.Create`/`db.Save`/`db.Delete` in handlers | PASS | None present in `resource.go` |
| DOM-16 | `administrator.go` holds write functions | PASS | `parcel/administrator.go:17` `Create`, `:33` `UpdateStatus`, `:53` `UpdateStatusIfPending`, `:77` `ClaimExpired`, `:103` `StampNotified`, `:125` `ClaimNotifiable` |
| DOM-17 | Domain errors → correct HTTP status | PASS | `parcel/resource.go:166-169` 404 on `ErrNotFound`; `:208-215` 404/409 on `ErrNotFound`/`ErrNotRecipient`+`ErrNotPending` for discard |
| DOM-18 | RestModels implement JSON:API interface | PASS | `RestModel` (`rest.go:47-58`), `DiscardRestModel` (`rest.go:98-109`), `parcelStatusRestModel` (`rest.go:124-135`) all implement `GetName/GetID/SetID` |
| DOM-19 | Request models flat, no nested Data/Type/Attributes | PASS | `DiscardRestModel` (`rest.go:93-96`) is flat (`Id`, `RecipientId`) |
| DOM-27 | DB-backed handlers use `server.WriteErrorResponse` for 500s, not bare `StatusInternalServerError` | PASS | Zero bare `http.StatusInternalServerError` writes in `resource.go`; every non-4xx error path routes through `server.WriteErrorResponse(d.Logger())(w)(err)` (e.g. `resource.go:109,126,144,172,179,217,224,258,264,281,289`), and the service calls `database.Connect` (`main.go:48`) |
| DOM-30 | DB write's events emitted via `AndEmit`+`message.Buffer`, not a direct `producer.ProviderImpl` call on the success path | FAIL | `parcel/notification_task.go:165-166`: after `ClaimNotifiable` (an administrator DB write, `administrator.go:125-144`) claims/stamps a row, `Run()` calls `kp := producer.ProviderImpl(t.l)(tenantCtx)` and invokes it directly per claimed row on the success path — never wrapped in `message.Emit`/`message.Buffer`. Does not fit either documented exception (not a post-failure branch; the operation does write to the database). |
| DOM-31 | Tenant/trace travel in context only | PASS | `RestModel`/`DiscardRestModel`/`parcelStatusRestModel` (`rest.go`) carry no tenant/trace field; `AcceptToParcelCommandBody`/`AcceptParams` (`kafka/message/custody/kafka.go`, `parcel/processor_custody.go`) carry no `TenantId` field either — tenant travels via `db.WithContext(ctx)` / Kafka header only |
| DOM-32 | Routes resolve to `server.RegisterHandler`/`RegisterInputHandler[T]`, no raw handler literal, no manual tenant parsing, no custom error helper | PASS | `parcel/resource.go` routes all go through `rest.RegisterHandler`/`rest.RegisterInputHandler[T]` aliases (`resource.go:45-46`), which compose `server.RetrieveSpan`+`server.ParseTenant` (`rest/handler.go:72-77,87-93`) — the same repo-wide idiom used by `atlas-mts`'s `rest/handler.go` for DB-threaded services (verified: `services/atlas-mts/atlas.com/mts/rest/handler.go:69-95` is structurally identical). No manual header parsing, no custom error writer. |

### entity.go / model.go / builder.go (FILE placement, `parcel` package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in `processor.go`/`processor_<group>.go` | PASS | `parcel/processor.go` (interface + core methods) + `parcel/processor_custody.go` (additive custody methods) — a legitimate group split |
| FILE-02 | RestModel/Transform/JSON:API methods in `rest.go` | PASS | `parcel/rest.go` |
| FILE-03 | Cross-service request functions in `requests.go` | N/A | No `requests.RootUrl`/`requests.*Request[T]` calls anywhere in the service — no `requests.go` needed |
| FILE-04 | Entity struct, `Migration`, `TableName` in `entity.go` | PASS | `parcel/entity.go:39,81,86` |
| FILE-05 | Builder/Model/writes/readers placement | PASS (with the DOM-02/03 caveat above) | Builder in `builder.go`, `Model` in `model.go`, writes in `administrator.go`, readers in `provider.go` |
| FILE-06 | No package-named catch-all bundling ≥2 responsibilities | PASS | No `parcel.go`; each file (`model.go`, `entity.go`, `builder.go`, `rest.go`, `administrator.go`, `provider.go`, `processor.go`, `processor_custody.go`, `asset_data.go`, `errors.go`, `task.go`, `notification_task.go`) is single-purpose |

### rest (support package — no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File placement | PASS | `rest/handler.go` is pure REST-registration plumbing (`HandlerDependency`, `RegisterHandler`, `RegisterInputHandler[M]`, `ParseCharacterId`, `ParseParcelId`) — not a package-named catch-all, does not bundle ≥2 of Processor/RestModel/requests/entity responsibilities |

### kafka/consumer/custody (support package — no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File placement | PASS | `consumer.go` is a single-purpose Kafka handler-registration file |
| DOM-30 | DB write's events via `AndEmit`+`message.Buffer`, not direct `producer.ProviderImpl` on success path | FAIL | `handleAcceptToParcel` (`kafka/consumer/custody/consumer.go:74-133`): `AcceptCustody` (`processor_custody.go:77-151`) writes the row inside `database.ExecuteTransaction`, then, on success, line 130 calls `p(custody.EnvStatusTopic)(custodyproducer.AcceptedStatusEventProvider(...))` directly — not `message.Emit`/`Buffer`-wrapped. Same shape in `handleReleaseFromParcel` (`consumer.go:139-161`): `ReleaseCustody` writes inside a transaction (`processor_custody.go:160-196`), then line 158 emits `ReleasedStatusEventProvider` via a direct `p(...)` call on the success path. Neither documented DOM-30 exception (post-failure branch; non-DB state) applies — both are DB writes acked directly. |
| DOM-24 | Test package reaching an emit path installs `producertest` or injects a no-op producer | PASS | `kafka/consumer/custody/consumer_test.go:33-59` injects a custom `recordingProducer.provider()` as the handler's `pf providerFn` parameter — the handlers under test never call the real `producer.ProviderImpl`, satisfying the "per-test injection of a no-op producer" branch of the pass criteria. |

### kafka/message/custody, kafka/message/parcel, kafka/producer/custody, kafka/producer/parcel, kafka/consumer, main (support packages)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File placement | PASS | Each file is single-purpose (message envelopes, producer providers, consumer config, `main.go` wiring) |
| DOM-23 | Kafka topic env vars in base configmap + both overlays, `KEY: "KEY"`, no literal manifest `env:` value | PASS | `COMMAND_TOPIC_PARCEL_CUSTODY`, `EVENT_TOPIC_PARCEL_CUSTODY_STATUS`, `EVENT_TOPIC_PARCEL_STATUS` all present in `deploy/k8s/base/env-configmap.yaml:62,155,156` and both `deploy/k8s/overlays/{pr,main}/kustomization.yaml`; `deploy/k8s/base/atlas-parcel.yaml:28` `env:` block carries no literal topic values (only `envFrom`-style configmap refs) |
| DOM-26 | Every goroutine via `routine.Go`, bare `go` needs `//goroutine-guard:allow` | PASS | `tools/goroutine-guard.sh` exit 0 (repo-wide, includes `atlas-parcel`); `parcel/task.go:98` and `notification_task.go:94` both use `routine.Go(t.l, t.ctx, ...)` |

### Testing (DOM-10, 20, 24, 33) — service-wide

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-10 | Test DB setup calls `database.RegisterTenantCallbacks` | PASS | Via the shared `databasetest.NewInMemoryTenantDB(t, Migration)` helper (used throughout `administrator_test.go`, `builder_test.go`, `notification_task_test.go`, `processor_test.go`, `provider_tenant_test.go`, `resource_test.go`, `task_test.go`), which itself calls `database.RegisterTenantCallbacks(l, db)` — `libs/atlas-database/databasetest/testdb.go:39` |
| DOM-20 | Tests use `tests := []struct{...}` + `t.Run` table-driven pattern | FAIL | Zero occurrences of a `[]struct{` table anywhere in the diff's test files (`grep -c "\[\]struct{"` on all 9 `_test.go` files → 0 matches). Every test file uses bare named `t.Run("case name", func(t *testing.T){...})` subtests instead: `administrator_test.go` (2 `t.Run`), `processor_test.go` (16), `provider_tenant_test.go` (3), `resource_test.go` (13), `task_test.go` (7), `kafka/consumer/custody/consumer_test.go` (10), `builder_test.go` (4), `notification_task_test.go` (multiple). None is a packet-fixture file, so the DOM-20 playbook exception does not apply. |
| DOM-24 | Emit-reaching test packages stub the producer | PASS | `parcel/notification_task_test.go:19,31,34` installs `producertest.InstallCapturing()` in `TestMain`; `kafka/consumer/custody/consumer_test.go` uses per-test producer injection (see above). `parcel/task_test.go` (`ExpiryTask`) reaches no Kafka emit path — N/A for that file. |
| DOM-33 | Interface-changing diffs update every mock | PASS / N/A | `parcel.Processor` is a brand-new interface with no pre-existing mocks to break (grep for `Mock struct` in the service: none). In the adjacent-scope `saga-orchestrator`, `saga.Handler` gains `WithParcelProcessor`/`handleAcceptToParcel`/`handleReleaseFromParcel`/`handleShowParcel` (`saga/handler.go`); its mocks are updated in the same diff (`saga/parcel_compensation_test.go`), corroborated by a clean `go build ./...` in `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`. |

### Deploy & Scaffolding (new service)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SCAFFOLD-01 | `.github/config/services.json` has a `go-service` entry | PASS | `.github/config/services.json:361-365` (`"name": "atlas-parcel"`, `"path": "services/atlas-parcel"`, ...) |
| SCAFFOLD-02 | K8s base manifest exists, listed in base `kustomization.yaml` | PASS | `deploy/k8s/base/atlas-parcel.yaml` exists; `deploy/k8s/base/kustomization.yaml:53` lists `atlas-parcel.yaml` |
| SCAFFOLD-03 | `docker-bake.hcl` entry + `go.work` `use()` entry | PASS | `docker-bake.hcl:80` `"atlas-parcel"`; `go.work:70` `./services/atlas-parcel/atlas.com/parcel` |
| SCAFFOLD-04 | Ingress block in `deploy/shared/routes.conf` | PASS | `deploy/shared/routes.conf:16-17` (`/api/parcels`) and `:151-152` (`/api/characters/{id}/parcel-status`) |
| SCAFFOLD-05 | Generated routes template regenerated and committed | PASS | `tools/gen-routes.sh --check` → `gen-routes: up to date` |
| SCAFFOLD-06 | docker-compose entry alongside peers | FAIL | No `atlas-parcel` entry in any of `deploy/compose/docker-compose.yml`, `docker-compose.core.yml`, `docker-compose.socket.yml` (`grep -n parcel` on all three → no match) |
| SCAFFOLD-08 | Bruno collection present (REST service) | FAIL | No `.bruno`/`bruno.json` directory anywhere under `services/atlas-parcel/` (every peer REST service — e.g. `atlas-storage`, `atlas-notes`, `atlas-ban` — has one) |
| SCAFFOLD-09 | Overlay enumerations, `ATLAS_DB_NAMES`, DB bootstrap — machine-checked | PASS | `tools/service-registration-guard.sh` → `service-registration-guard: clean` |

## Constants Reuse (DOM-21) — libs/atlas-constants, libs/atlas-saga, saga-orchestrator/parcel

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | `world.Id` / `channel.Id` reused, not redeclared | PASS | `parcel.Model.worldId world.Id` (`parcel/model.go:17`), `ShowParcelPayload.ChannelId channel.Id` (`libs/atlas-saga/payloads.go` new block) both use the existing `libs/atlas-constants` types |
| DOM-21 | `item.ClassificationDueyCoupon` reused for the ticket gate | PASS | `libs/atlas-constants/item/duey.go:19` `QuickDeliveryTicketId`, doc comment cites the existing `ClassificationDueyCoupon` and states it was checked against the existing constants first |
| DOM-21 | `TransferToParcelPayload.SourceInventoryType` / `WithdrawFromParcelPayload.InventoryType` | FAIL | Both new fields are declared as raw `byte` (`libs/atlas-saga/payloads.go:992`, `:1082`), even though `libs/atlas-constants/inventory/constants.go:9` already defines `type Type int8` for exactly this domain concept, and that shared type is in active, documented use elsewhere in the very same file (`payloads.go:708-712`, whose comment reads "InventoryType is the shared inventory.Type ... the item CAME from"). The new Duey fields invent a second, untyped representation instead of reusing the constant already present two hundred lines away in the file they were added to. |
| DOM-21 | `AssetData` (parcel) vs `asset.AssetData` (merchant) | N/A | Documented as "copied verbatim ... a straightforward move, not a cross-service import" (`parcel/asset_data.go:10-13`) — this is DOM-34/35 migration-hygiene territory, not DOM-21 (it is not a duplicate declaration alongside an existing `libs/atlas-constants` type; `AssetData` does not live in `libs/atlas-constants` on either side). See Migration hygiene note below. |

### Migration hygiene (DOM-34/35) note

`parcel/asset_data.go`'s doc comment describes the type as "copied verbatim" from `services/atlas-merchant/.../asset/kafka.go` and cites CLAUDE.md's "prefer straightforward moves ... don't call another layer's internals across a service boundary" guidance directly. DOM-34/35's own trigger ("diff moves or extracts symbols between a service and a `libs/atlas-*` module") does not fire here — this is a service-to-service copy, not a service⇄`libs/atlas-*` extraction — so DOM-34/35 itself is N/A. No numbered rule in the checklist governs a service-to-service struct copy of this kind; flagged here for visibility only, not as a FAIL.

## Security Review

Trigger did not fire — `atlas-parcel` handles no authentication, authorization, tokens, redirects, or secrets. SEC-01..04: N/A.

## Not evaluable from the diff

- SCAFFOLD-07 (channel writer/handler seeding in every targeted tenant opcode template) — this task's design (§9.5) adds `handleDueyCouponUse`/`PARCEL[OPEN_QUICK]` announcement in `atlas-channel`, but that service is outside this audit's assigned scope (`services/atlas-parcel` + the constants/saga/orchestrator lens); a sibling reviewer covers `atlas-channel`.
- DOM-25 (client-interpreted wire values) — the `PARCEL`/`DUEY_ACTION` dispatcher codecs in `libs/atlas-packet` were flagged as landed by prior commits on this branch but are outside this audit's assigned scope; not evaluated here.
- Full DOM/FILE/REST sweep of `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/parcel` and `services/atlas-saga-orchestrator/.../saga/*` beyond the DOM-21/DOM-30/DOM-33 spot checks recorded above — the task scoped that service to "insofar as they define or consume the parcel domain," not a full audit; a complete review would need the orchestrator's own `resource.go`/`processor.go` read against every REST/DOM rule, which was not done here.

## Summary

### Blocking (must fix)
- DOM-02: `parcel/builder.go:253` — `entityFromModel` is a free function, not `Model.ToEntity()`, and it is not in `entity.go`.
- DOM-03: `parcel/builder.go:219` — `Make(Entity)` is defined in `builder.go`, not `entity.go`.
- DOM-05: `parcel/rest.go` has no `TransformSlice`; `parcel/resource.go:141` inlines `model.SliceMap(Transform)` in the list handler instead.
- DOM-08: `parcel/resource.go:52` — `PATCH /parcels/{parcelId}/notify` registers via `RegisterHandler`, not `RegisterInputHandler[T]`.
- DOM-11: `parcel/provider.go:17-100` — every provider function eagerly executes its query and wraps the result in `model.FixedProvider`, defeating lazy composition.
- DOM-30: `parcel/notification_task.go:165-166` — direct `producer.ProviderImpl` call on the success path of a DB-writing operation, not `AndEmit`/`message.Buffer`-wrapped.
- DOM-30: `kafka/consumer/custody/consumer.go:130,158` — `handleAcceptToParcel`/`handleReleaseFromParcel` ack a DB write via a direct producer call on the success path.
- DOM-20: no `_test.go` file added by this diff uses the `tests := []struct{...}` + `t.Run` table-driven pattern.
- SCAFFOLD-06: no `atlas-parcel` entry in any `deploy/compose/docker-compose*.yml`.
- SCAFFOLD-08: no Bruno collection under `services/atlas-parcel/`.
- DOM-21: `libs/atlas-saga/payloads.go:992,1082` — `SourceInventoryType`/`InventoryType` declared as raw `byte` instead of the existing `libs/atlas-constants/inventory.Type`, which the same file already uses elsewhere.

### Non-Blocking (should fix)
- None recorded as WARN — every deviation found rises to FAIL against its rule's own pass criteria; there is no softer disposition available for a structural/File-Responsibilities or messaging-atomicity finding per the Mindset severity rule.

### Not evaluable
- 3 items (SCAFFOLD-07, DOM-25, full orchestrator sweep) — see the section above.
