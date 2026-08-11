# Backend Audit — task-211-kite-cash-item

- **Scope:** whole-branch diff, merge-base `eca47150f` -> head `e6e8f7a7f`
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-10
- **Build:** PASS (all 4 touched modules)
- **Tests:** PASS (all 4 touched modules, `-count=1`)
- **Overall:** NEEDS-WORK

## Build & Test Results

| Module | Build | Test |
|---|---|---|
| `libs/atlas-packet` | PASS | PASS (all packages) |
| `services/atlas-kites/atlas.com/kites` | PASS | PASS (`character`, `configuration`, `kafka/consumer/character`, `kite`) |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS (`kite`, `kafka/message/*`, `socket/handler`) |
| `services/atlas-tenants/atlas.com/tenants` | PASS | PASS (`configuration`, `configuration/seed`) |

## Domain Checklist Results

### `services/atlas-kites/atlas.com/kites/kite` (domain package, Redis-backed — no `entity.go`/GORM, so DOM-02/03/16 N/A)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go exists | PASS | `kite/builder.go` — `NewBuilder`, fluent setters, `Build()` |
| DOM-04/05 | Transform/TransformSlice | PASS | `kite/rest.go:51` `Transform`; list handler uses `model.SliceMap(Transform)` (`kite/resource.go:91`) |
| DOM-06 | Processor accepts FieldLogger | PASS | `kite/processor.go:56` `l logrus.FieldLogger` |
| DOM-07 | Handlers pass d.Logger() | PASS | `kite/resource.go:38,76` |
| DOM-09 | Transform errors handled | PASS | `kite/resource.go:49-54,91-96` check `err` |
| DOM-12 | No os.Getenv in handlers | PASS | none in `resource.go` |
| DOM-14/15 | No provider/db calls in handlers | PASS | handlers call `NewProcessor(...).GetByCharacterId/GetInMap` only |
| DOM-17 | Error->HTTP mapping | PASS | `kite/resource.go:40-46` `ErrNotFound`->404, else 500 |
| DOM-18 | JSON:API interface on RestModel | PASS | `kite/rest.go:34-49` |
| DOM-20 | Table-driven tests | PASS | `kite/processor_test.go`, `kite/registry_test.go` |
| DOM-24 | Kafka producer stubbed in tests | PASS | per-test `producer.Provider` injection via `NewProcessorWithProvider` (documented escape hatch); `kafka/consumer/character/consumer_test.go` uses shared `producertest.InstallCapturing()` |
| **error-swallow (repeat of already-fixed class)** | Redis error not conflated with "not found"/"empty" | **FAIL (Important)** | `character/registry.go:40-44` `GetInMap` returns `nil` on **any** error from `r.sets.Members`, indistinguishable from "no characters present." Consumed by `character/processor.go:38-41` `InMapProvider` (no error path at all) and then by `kite/processor.go:300-322` `InMapModelProvider`, whose own doc comment (lines 292-299) claims "every error here is returned to the caller... a Redis failure fails GetInMap/Create loudly rather than being swallowed into a wrong count" — that guarantee is false because the character-index side of the same call chain already swallowed it. A Redis blip on the character-index read silently reports zero characters in the map, letting `Create`'s `MaxPerMap` cap (`kite/processor.go:156`) under-count and admit placements past the tenant's configured limit — exactly the class of bug this branch already found and fixed once in `kite.Registry.Get` (commit `cf81971d0`). |
| **error-discard** | Registry writes discard errors | FAIL (Minor) | `character/registry.go:32-38` `AddCharacter`/`RemoveCharacter` use `_ = r.sets.Add(...)` / `_ = r.sets.Remove(...)` with no log; a failed index update is invisible and silently stales the same index `GetInMap` reads. |

### `services/atlas-kites/atlas.com/kites/configuration` (support/REST-client package — External HTTP Client Checklist)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..04 | Processor/RestModel/requests/entity in correct files | PASS | `registry.go` (cache singleton, not a Processor per se but correctly separated), `rest.go` (RestModel+Extract), `requests.go` (`requestForTenant`) — no catch-all file |
| EXT-01 | Relationship interface stubs | **FAIL (Important)** | `configuration/rest.go:8-26` `RestModel` has no `SetToOneReferenceID`/`SetToManyReferenceIDs`. Per `libs/atlas-rest/CLAUDE.md` these are required "even when you don't care about relationships" — api2go errors on any response carrying a `relationships` block and the error surfaces as a misleading generic failure (the exact task-037 pattern the guideline names). |
| EXT-02 | httptest-backed integration test | **FAIL (Important)** | `configuration/rest_test.go` only unit-tests `Extract`/`IsMapBlocked`; no `httptest.NewServer` fixture round-trips `requestForTenant`/`defaultFetcher`. The guideline explicitly disqualifies unit-tests-on-Extract-alone. |
| EXT-03 | 404 vs other-error distinction | FAIL (Minor-Important) | `configuration/registry.go:76-80` `GetTenantConfig` funnels every `fetch` error (genuine 404-no-config, connection refused, 5xx, decode failure) into the same "log Warn, use DefaultConfig" path. No `errors.Is(err, requests.ErrNotFound)` anywhere in the package. A live atlas-tenants outage is indistinguishable from "tenant hasn't configured kites yet." |
| EXT-04 | RootUrl(domain), no hardcode | PASS | `configuration/requests.go:16-18` `requests.RootUrl("TENANTS")` |

### `services/atlas-channel/atlas.com/channel/kite` (support/REST-client package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | One responsibility per file, no catch-all | PASS | `model.go`, `builder.go`, `processor.go`, `producer.go`, `requests.go`, `rest.go` each hold exactly their designated symbols |
| EXT-01 | Relationship interface stubs | **FAIL (Important)** | `kite/rest.go:16-29` `RestModel` has no `SetToOneReferenceID`/`SetToManyReferenceIDs` — same missing boilerplate as above, second occurrence in this branch. |
| EXT-02 | httptest-backed integration test | PASS | `kite/processor_drain_test.go` — `httptest.NewServer` serves a real JSON:API kites document across 3 pages and asserts the decoded `Model` slice is fully populated and complete (`TestInMapModelProviderDrainsBeyondOnePage`, `TestInMapModelProviderRequestsInstanceScopedPath`). |
| EXT-04 | RootUrl(domain) | PASS | `kite/requests.go:15-17` `requests.RootUrl("KITES")` |

### `services/atlas-kites/atlas.com/kites/kite` — DOM-25 / client wire values

| Check | Status | Evidence |
|---|---|---|
| Category-508 detection reuses shared constant | PASS | `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:841` dispatches on `item.ClassificationMessageBanner` (the existing `libs/atlas-constants/item/constants.go:78` constant) — no reinvented 508 literal. |
| `mapPrefixDivisor`/`IsMapBlocked` (kite placement policy) vs. `map.IsFreeMarketRoom` | Reviewed, not a finding | `configuration/model.go:12,35-43` implements the CLIENT's own coarse `id/10,000,000==91` gate (IDA-verified, `CWvsContext::SendConsumeCashItemUseRequest`), which is a **different, wider** range than `libs/atlas-constants/map/model.go:47` `IsFreeMarketRoom` (the 23 real Free Market room ids). These are legitimately different semantics (mirroring the client's own coarse bucket vs. the WZ-verified room list), not a duplicate of existing atlas-constants logic — no DOM-21 finding. |
| `KiteDestroyAnimationType` (`libs/atlas-packet/field/clientbound/kite_destroy.go:21-28`) | Reviewed, not a finding | A fixed protocol invariant (server always sends `KiteDestroyAnimated`, doc-commented as intentional at the call site, `kafka/consumer/kite/consumer.go:87-92`), not a per-version/per-tenant classification the client varies — does not need a tenant config table the way NoticeFailReason/msgType do. |
| Provisional `DestroyReasonOwnerLeft` for the DESTROY command (no producer exists yet) | Confirmed present, not a new finding (pre-adjudicated) | `kafka/consumer/kite/consumer.go:55-59` carries the required comment explaining the provisional choice. |

### `services/atlas-tenants/atlas.com/tenants/configuration` (kite-configs addition to existing shared multi-resource file)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01/02 (catch-all) | Kite-configs symbols land in the pre-existing shared `processor.go`/`resource.go`/`rest.go` alongside routes/vessels/mts-configs/rankings/rps-rewards | PASS, not a finding | Each symbol is still in its per-responsibility file (Processor methods in `processor.go`, RestModel+Transform/Extract in `rest.go`, handlers in `resource.go`) — this is the pre-existing repo-wide shape for this package (not introduced by this branch) and satisfies the File Responsibilities table on its own terms; it is not the `wallet.go`-style single-file collapse the checklist warns about. |
| Multi-tenancy | tenant.MustFromContext / GORM auto-filter | PASS | consistent with sibling resources already in the file |

### Deployment/registration files (noted per instructions, not scored against DOM checklist)

- `deploy/k8s/base/env-configmap.yaml:213,235` — `COMMAND_TOPIC_KITE`/`EVENT_TOPIC_KITE_STATUS` added in `KEY: "KEY"` form, alphabetically placed — matches DOM-23 convention.
- `services.json`, `docker-bake.hcl`, `deploy/k8s/base/kustomization.yaml`, `deploy/shared/routes.conf`, `deploy/k8s/base/routes.conf.template.generated`, both kustomize overlays — all carry an `atlas-kites` entry consistent with the sibling services (SCAFFOLD-01/02/04 satisfied at a glance).
- DOM-22 (Dockerfile-per-lib-mention count): **N/A for this repo state** — the repo-root `Dockerfile` was refactored (pre-dating this branch) into one shared, generic multi-stage build that unconditionally `COPY`s every `libs/atlas-*` tree for every service (`Dockerfile:1-80`); there is no longer a per-service Dockerfile or a per-lib enumeration to drop. `patterns-deploy.md`'s four-block/per-service-Dockerfile model is stale relative to the current architecture. Not exercised via `docker buildx bake` in this audit (not run — time-boxed); flagged only as an observation, not a finding.
- Bruno collection for `atlas-kites` (SCAFFOLD-08): **not checked** — `services/atlas-kites/.bruno/` presence not verified in this pass; call out if a full scaffolding audit is wanted.

## Summary

### Blocking (must fix)
- **[Important]** `character/registry.go:40-44` `GetInMap` swallows real Redis errors into an empty result, contradicting the loud-failure guarantee `kite/processor.go:292-299` claims for the same call chain — a Redis blip can silently admit kite placements past `MaxPerMap`.
- **[Important]** `configuration/rest.go` (both `atlas-kites` and `atlas-channel/kite`) — `RestModel` missing `SetToOneReferenceID`/`SetToManyReferenceIDs` (EXT-01), the exact task-037 class of bug per `libs/atlas-rest/CLAUDE.md`.
- **[Important]** `atlas-kites/configuration` — no httptest-backed integration test for the atlas-tenants kite-configs fetch (EXT-02); only `Extract`-level unit tests exist.

### Non-Blocking (should fix)
- **[Minor]** `character/registry.go:32-38` `AddCharacter`/`RemoveCharacter` discard Redis errors silently (`_ = ...`), no log.
- **[Minor-Important]** `atlas-kites/configuration.GetTenantConfig` does not distinguish 404 from transport/5xx failures (EXT-03) — masks atlas-tenants outages as normal tenant-defaulting.
