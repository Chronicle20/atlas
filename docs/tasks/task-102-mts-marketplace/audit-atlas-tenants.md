# Backend Audit — atlas-tenants

- **Service Path:** services/atlas-tenants/atlas.com/tenants
- **Scope:** task-102 modified packages — `configuration`, `configuration/mock`, `rest` (added the `mts-configs` configuration resource)
- **Guidelines Source:** backend-dev-guidelines skill (File Responsibilities FILE-01..06, DOM-*, SUB-*, EXT-*, SEC-*)
- **Date:** 2026-07-10
- **Build:** PASS (`go build ./...` exit 0)
- **Vet:** PASS (`go vet ./...` exit 0)
- **Tests:** PASS — `atlas-tenants/configuration` ok, `atlas-tenants/tenant` ok; all other packages have no test files (`go test ./... -count=1` exit 0)
- **Overall:** NEEDS-WORK

The task-102 MTS additions themselves are clean: the FILE Responsibilities checklist — the review focus, and the exact place a misplaced `RestModel`/`Transform` would hide — **PASSES on all three packages**. `MtsConfigRestModel`, `TransformMtsConfig`, `ExtractMtsConfig`, and the JSON:API methods all live in `rest.go` where they belong; `ParseMtsConfigId` is in `rest/handler.go`; the mock has full interface parity with a compile-time assertion. The NEEDS-WORK verdict comes from **pre-existing** package-wide DOM deviations that the MTS code inherited by copy-paste from the Route/Vessel resources (no canonical `Transform`/`TransformSlice`, no `ToEntity()`, non-table-driven tests, test DB without tenant callbacks). None are introduced or worsened by task-102, but per "prevalence is not compliance" they are recorded against the package as it stands.

## Build & Test Results

```
go build ./...   -> exit 0
go vet ./...      -> exit 0
go test ./... -count=1:
  ?   atlas-tenants                       [no test files]
  ok  atlas-tenants/configuration         0.021s
  ?   atlas-tenants/configuration/mock    [no test files]
  ?   atlas-tenants/rest                  [no test files]
  ok  atlas-tenants/tenant                0.016s
  (kafka/*, logger, test, tenant/mock — no test files)
```

## Package Classification

| Package | Classification | Notes |
|---------|----------------|-------|
| `configuration` | Domain (`model.go` present) | Generic per-tenant JSONB config store. Domain `Model` = `(id, tenantId, resourceName, resourceData)`; sub-resources (routes/vessels/instance-routes/mts-configs) are marshaled JSON:API objects inside `resourceData`. |
| `configuration/mock` | Support | Test double `ProcessorMock` for `configuration.Processor`. |
| `rest` | Support | REST infra aliases + `Parse*`/`Register*` helpers shared by handlers. |

---

## Package: `configuration`

### File Responsibilities Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | `Processor`/`ProcessorImpl`/`NewProcessor` in `processor.go` | PASS | `type Processor interface` processor.go:18; `type ProcessorImpl` processor.go:119; `func NewProcessor` processor.go:127. No Processor symbols in any other file. |
| FILE-02 | `RestModel` + `Transform`/`Extract` + JSON:API methods in `rest.go` | PASS | `MtsConfigRestModel` rest.go:230; `GetID/SetID/GetName` rest.go:244/249/255; `TransformMtsConfig` rest.go:260; `ExtractMtsConfig` rest.go:328. All in `rest.go` — **not** misplaced into `model.go`/`configuration.go`. |
| FILE-03 | Cross-service request funcs in `requests.go` | N/A (PASS) | Package is the config source-of-truth; no `requests.RootUrl`/`GetRequest`/`PostRequest` anywhere. |
| FILE-04 | Entity + `TableName` + migration in `entity.go` | PASS | `type Entity` entity.go:11; `TableName()` entity.go:20; `MigrateEntities` entity.go:25. |
| FILE-05 | Builder/Model/writes/providers placed per table | PASS (1 nit) | Builder `modelBuilder` builder.go:15; `Model` model.go:11; writes `CreateConfiguration`/`UpdateConfiguration`/`DeleteConfiguration` administrator.go:13/20/27; readers `database.Query`/`model.Map` provider.go. Nit: `Make(Entity)` is in model.go:44 rather than entity.go (guideline places `Make` in entity.go); entity-construction helpers `NewEntityBuilder`/`FromModel` in entity_builder.go. |
| FILE-06 | No package-named catch-all file | PASS | No `configuration.go`. Files are split by responsibility (commit 6f2ac59ae6 explicitly split collapsed files). No single file carries ≥2 responsibilities. |

### Domain Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` with validating `Build()` | PASS | builder.go:64 `Build()` validates tenantId + resourceName (`ErrTenantIdRequired`/`ErrResourceNameRequired`). Constructor is `NewModelBuilder()` (naming variant, still fluent). |
| DOM-02 | `ToEntity()` on Model | FAIL (pre-existing) | No `ToEntity()` method exists on `Model`. Model→Entity is done via `FromModel(m Model) Entity` (entity_builder.go:56). Pre-existing; task-102 did not touch model.go/entity.go. |
| DOM-03 | `Make(Entity) (Model, error)` | PASS | model.go:44 (location nit: in model.go, not entity.go). |
| DOM-04 | `Transform` in `rest.go` | PASS (adapted) | No canonical `func Transform(Model)`. Per-resource `TransformMtsConfig(map)` (rest.go:260) fills the role — appropriate for a generic JSONB store whose `Model` is a config wrapper, not the sub-resource. task-102 followed the established Route/Vessel pattern. |
| DOM-05 | `TransformSlice` + no inline loops in resource.go | FAIL (pre-existing) | No `TransformSlice`; list handlers use inline `for` loops (resource.go:37-46 routes, 224-232 vessels, 408-417 instance-routes). **MTS adds no new slice loop** — `GetMtsConfigHandler` returns a single object (`configs[0]`, resource.go:664). |
| DOM-06 | Processor takes `logrus.FieldLogger` | PASS | processor.go:120 `l logrus.FieldLogger`; processor.go:127 `NewProcessor(l logrus.FieldLogger, ...)`. |
| DOM-07 | Handlers pass `d.Logger()` | PASS | All MTS `NewProcessor(d.Logger(), d.Context(), db)` — resource.go:644,685,722,773,811,831. No `logrus.StandardLogger()`. |
| DOM-08 | POST/PATCH use `RegisterInputHandler[T]` | PASS | `registerMtsConfigInputHandler` for POST (resource.go:886) and PATCH (resource.go:887); declared resource.go:856. |
| DOM-09 | Transform errors handled | PASS | Every `TransformMtsConfig(...)` call checks `err` (resource.go:664-668, 694-698, 744-748, 789-793). No `_, _ :=`. |
| DOM-10 | Test DB registers tenant callbacks | FAIL/deviation | `test.SetupTestDB` (test/database.go:14) `AutoMigrate`s but never calls `database.RegisterTenantCallbacks`. Mitigated: providers filter tenant explicitly (`GetByTenantIdAndResourceNameProvider` maps `tenant_id` in the query, provider.go:13-19) rather than relying on GORM callbacks, so automatic filtering is not the mechanism under test. Pre-existing test infra. |
| DOM-11 | Providers lazy | PASS | `database.Query[Entity]` + `model.Map` returning `model.Provider`, curried on `*gorm.DB` — provider.go:152-215 (mts). No eager exec wrapped in FixedProvider. |
| DOM-12 | No `os.Getenv` in handlers | PASS | 0 matches in resource.go. (`os.Getenv` in seed.go:26/34/54/68 is seed-path config, not a handler.) |
| DOM-13 | No cross-domain logic in handlers | PASS | MTS handlers call only `configuration` processor methods. |
| DOM-14 | Handlers don't call providers directly | PASS | Handlers call `processor.GetAllMtsConfigs`/`GetMtsConfigById`/`CreateMtsConfigAndEmit`/etc., never `Get*Provider` directly. |
| DOM-15 | No direct entity writes in handlers | PASS | No `db.Create`/`db.Save`/`db.Delete` in resource.go. |
| DOM-16 | `administrator.go` for writes | PASS | Writes route through `CreateConfiguration`/`UpdateConfiguration` (administrator.go). Nit: `CreateMtsConfig` builds the `Entity{}` literal inline in processor.go:661-666 before calling `CreateConfiguration`. |
| DOM-17 | Domain error → HTTP status mapping | PARTIAL | `GetMtsConfigHandler` correctly distinguishes not-found→404 vs other→500 (resource.go:646-656) — better than sibling handlers. But `GetMtsConfigByIdHandler` maps **every** error to 404 (resource.go:688-691), so an unmarshal/internal failure is surfaced as 404. Create/Update/Delete map all processor errors to 500 (no 400/409 for validation/conflict). Consistent with Route/Vessel. |
| DOM-18 | JSON:API interface on REST models | PASS | `MtsConfigRestModel` `GetName()`="mts-configs" (rest.go:255), `GetID` (244), `SetID` (249). |
| DOM-19 | Request models flat | PASS | `MtsConfigRestModel` (rest.go:230) is flat; no nested Data/Type/Attributes. Used as both request and response model. |
| DOM-20 | Table-driven tests | FAIL (pre-existing) | MTS tests (`TestCreateMtsConfig_Success`, `TestGetMtsConfigById_Found/NotFound`, `TestMtsConfigRoundTrip`, etc.) are individual funcs; zero `tests := []struct{...}` / `t.Run` in processor_test.go. Matches the whole file's pre-existing style. |
| DOM-21 | No atlas-constants duplication | PASS | MTS config fields are economic primitives (`listingFee uint32`, `commissionRate float64`, `minLevel int`, ...). No item/world/inventory/job id types redeclared. |
| DOM-22 | Dockerfile 4-block per direct lib | N/A (PASS) | `go.mod` and `services/atlas-tenants/Dockerfile` unchanged vs `main`; no new `libs/atlas-*` direct require introduced. |
| DOM-23 | Kafka topic naming | N/A (PASS) | MTS reuses the existing produced topic `EVENT_TOPIC_CONFIGURATION_STATUS` (kafka.go:11), adding only new event **types** (`MTS_CONFIG_CREATED/UPDATED/DELETED`, kafka.go:21-23). No new topic constant. |
| DOM-24 | Kafka producer stubbed in emitting tests | PASS | Tests use a local `testProcessor` (processor_test.go:15) that writes via `CreateConfiguration` + providers and never calls `*AndEmit`/`message.Emit`/`producer.Produce`. No unstubbed emit path is exercised, so no ~42s producer hang. (See observation on coverage below.) |
| DOM-25 | Client wire values config-resolved | N/A (PASS) | atlas-tenants is not channel/socket code; MTS config carries semantic economic values, not client opcodes. |

### External HTTP Client (EXT-01..04)
N/A — `configuration` exposes REST and does not call another atlas service (`atlas-mts` is the consumer of this endpoint). No `requests.RootUrl`/`GetRequest[T]`/`PostRequest[T]`.

### Security (SEC-01..04)
N/A — not an auth/authorization/token service. MTS config is non-secret tenant economic configuration; no hardcoded secrets, no JWT/redirect handling.

---

## Package: `configuration/mock`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01/06 | Single-purpose mock file | PASS | `mock/processor.go` holds only `ProcessorMock` and its methods — no RestModel/requests/entity collapse. Not a `<pkg>.go` catch-all. |
| Mock parity | Mock ⇔ Processor interface | PASS | Compile-time assertion `var _ configuration.Processor = (*ProcessorMock)(nil)` (mock/processor.go:12). 44 interface methods = 44 mock methods; set-diff empty in both directions. All 10 MTS methods present with matching signatures (mock/processor.go:41-56, 365-...). Build passing confirms satisfaction. |

---

## Package: `rest`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-06 | No catch-all collapse | PASS | `rest/handler.go` holds REST infra only: type aliases + `RegisterHandler`/`RegisterInputHandler` wrappers + `Parse*` helpers. Genuine single-purpose infra, not a Processor+RestModel+requests bundle. |
| Placement | `ParseMtsConfigId` correctly located | PASS | task-102 added `ParseMtsConfigId` (rest/handler.go:48) alongside the sibling `Parse*` helpers — correct file. |
| SUB-03/04 | (n/a — no handlers here) | N/A | This package defines helpers, not endpoints. |

---

## Observations (non-checklist, low severity)

- **`CreateMtsConfig` single-resource merge drops the existing object.** processor.go:626-632: when an `mts-configs` row already exists as a *single* JSON:API object (not an array), a second create builds a fresh array containing only the new config, discarding the prior one. Harmless in practice — MTS is one-config-per-tenant and `GetMtsConfigHandler` reads `configs[0]` — but it is a latent data-loss branch copy-pasted from the Route resource.
- **AndEmit + merge paths are untested.** The table-harness `testProcessor` deliberately bypasses `CreateMtsConfigAndEmit`/`UpdateMtsConfigAndEmit` and the array-merge logic in the real `ProcessorImpl`. Real emission + merge behavior has no coverage (this is why DOM-24 is clean, but it is a coverage gap).

---

## Summary

### Critical
- None. Build, vet, and tests all pass; no security exposure; no data corruption on the supported single-config MTS path.

### Important
- **DOM-05** — `configuration` has no `TransformSlice`; list handlers use inline `for` loops (resource.go:37-46, 224-232, 408-417). Pre-existing; MTS added no new slice loop. Structural, so not down-graded to Minor.
- **DOM-02** — no `ToEntity()` on `Model` (conversion done via `FromModel`, entity_builder.go:56). Pre-existing; structural.

### Minor / Deviations
- **DOM-17** — `GetMtsConfigByIdHandler` conflates all errors to 404 (resource.go:688-691); create/update/delete map validation errors to 500 rather than 400/409. Consistent with siblings; `GetMtsConfigHandler` itself is correct.
- **DOM-10** — `test.SetupTestDB` omits `database.RegisterTenantCallbacks` (test/database.go:23); mitigated by explicit `tenant_id` filtering in providers.
- **DOM-20** — MTS tests are not table-driven (no `t.Run`); matches the file's pre-existing style.
- Location nits: `Make` in model.go rather than entity.go; inline `Entity{}` literal in `CreateMtsConfig` (processor.go:661).

### Verdict
**NEEDS-WORK** — build/vet/tests green and the task-102 MTS additions are correctly placed (FILE-01..06 PASS on all three packages, full mock parity). Remaining findings are pre-existing package conventions the MTS resource inherited, none introduced by this task.
