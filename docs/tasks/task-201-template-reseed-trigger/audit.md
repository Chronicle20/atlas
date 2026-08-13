# Backend Audit — atlas-configurations (task-201-template-reseed-trigger)

- **Service Path:** services/atlas-configurations/atlas.com/configurations
- **Scope:** `templates/revision.go`, `templates/shipped.go`, `templates/rest.go`, `templates/processor.go`, `templates/resource.go`, `templates/mock/processor.go`, `seeder/seeder.go`, `main.go`, plus their tests
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-07
- **Build:** PASS (`go build ./services/atlas-configurations/atlas.com/configurations/...`)
- **Tests:** all packages `ok` (`go test ./services/atlas-configurations/atlas.com/configurations/... -count=1`); no failures
- **Goroutine guard:** no bare `go` statements in the changed non-test files (`revision.go`, `shipped.go`, `rest.go`, `processor.go`, `resource.go`, `seeder.go`, `main.go`)
- **Overall:** PASS (no blocking findings against this diff)

## Domain Checklist Results — `templates` package

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor accepts `logrus.FieldLogger` | PASS | `templates/processor.go:57` `func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor` |
| DOM-08 | POST/PATCH use `RegisterInputHandler` | PASS (justified exception) | `templates/resource.go:32` registers `POST /{templateId}/reseed` via `rest.RegisterHandler`, not `RegisterInputHandler[T]`. The endpoint takes no request body (only a path param), matching the codebase's established body-less action-trigger pattern (`RegisterHandler` used for e.g. `services/atlas-ban/.../ban/resource.go:31` `expire_ban`, `services/atlas-npc-conversations/.../npc/resource.go:37` `reindex_recipes`, `services/atlas-tenants/.../configuration/resource.go:1236` `seed_rps_rewards`). `RegisterInputHandler[T]` is documented for "POST/PATCH handlers (with typed request model)" — there is no request model here, so the exception is guideline-consistent, not a deviation. |
| DOM-12 | No `os.Getenv()` in handlers | PASS | `grep -n "os.Getenv" templates/resource.go` → no matches |
| DOM-13/14/15 | No cross-domain logic, no direct provider calls, no direct DB writes in handlers | PASS | All new/changed handlers in `templates/resource.go` (`handleCreateConfigurationTemplate:44`, `handleReseedConfigurationTemplate:204`) call only `NewProcessor(...)`/`viewProcessor(...)` methods; no `db.Create`/`db.Save`/`db.Delete` or provider functions appear in `resource.go` |
| DOM-17 | Domain error → HTTP status mapping | PASS | `templates/resource.go:214-231` `handleReseedConfigurationTemplate` switches `errors.Is(err, ErrTemplateNotFound)` → 404, `errors.Is(err, ErrNoShippedTemplate)` → 409, `errors.As(err, &ve)` → 400, default → `server.WriteErrorResponse` (500/503 per the classifier). Sentinel errors defined at `templates/processor.go:26,29` |
| DOM-11 | Providers use lazy evaluation | PASS | `templates/processor.go:82-126` — `ByRegionAndVersionProvider`, `ByIdProvider`, `AllProvider`, and the new `ViewByRegionAndVersionProvider`/`ViewByIdProvider`/`AllViewProvider` all compose via `model.Map`/`model.MapPaged`, no eager `FixedProvider` wrapping of computed work |
| DOM-27 | Transient DB errors → 503, not bare 500 | PASS | `main.go:48-54` registers `server.RegisterTransientErrorClassifier` composing `database.IsTransientConnectionError` + `database.CountTransient`; every non-400/404/409 branch in the changed handlers calls `server.WriteErrorResponse(d.Logger())(w)(err)` (`templates/resource.go:57,67,91,117,135,163,179,231`) — no direct `w.WriteHeader(http.StatusInternalServerError)` in the diff |
| DOM-21 | No duplication of atlas-constants types | PASS | New types (`CatalogEntry`, `catalogKey`, `Catalog`, `ViewRestModel`) are template-catalog/view concerns local to this service; none overlap `libs/atlas-constants/` (item/inventory/weapon/world/job/skill/monster ids) |
| DOM-10 | Test DB has tenant callbacks | N/A (documented exception) | `templates/entity.go:16-18` — `Entity` has no tenant column; the design explicitly states "the reseed endpoint is intentionally NOT tenant-scoped: templates are global" (confirmed — `Entity` keyed only on `Region`/`MajorVersion`/`MinorVersion`, `templates/entity.go:16-18`). `setupTestDB` (`templates/processor_test.go:34`) correctly omits `database.RegisterTenantCallbacks` because there is no tenant column to filter |

## File Responsibilities Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor logic in `processor.go` | PASS | `Processor` interface (`templates/processor.go:32-47`), `ProcessorImpl` struct and all methods including the new `WithCatalog`, `makeView`, `ViewByRegionAndVersionProvider`, `ViewByIdProvider`, `AllViewProvider`, `ReseedById` (`templates/processor.go:49-291`) all live in `processor.go` |
| FILE-02 | `RestModel`/JSON:API methods in `rest.go` | PASS | `RestModel` + `GetName/GetID/SetID` (`templates/rest.go:11-35`, pre-existing) and the new `ViewRestModel` (`templates/rest.go:52-57`) both in `rest.go` |
| FILE-06 | No package-named catch-all file | PASS | New files `revision.go` (single `Revision` function) and `shipped.go` (`CatalogEntry`/`Catalog`/`LoadCatalog`/`InitShippedCatalog`/`ShippedCatalog` — one cohesive "shipped catalog" concern) are each single-purpose; neither bundles Processor+RestModel+requests. No new `templates.go`-style catch-all was introduced |

## Design Decisions Verified

- **D1 — Catalog in `templates`, not `seeder`:** confirmed by import direction: `seeder/seeder.go:5` imports `"atlas-configurations/templates"`; no `templates/*.go` file imports `seeder`. Reversing the dependency would indeed create a cycle. Compliant.
- **D2/D3 — Drift attributes on `ViewRestModel`, not `RestModel`:** `templates/rest.go:52-57`; `Create` marshals the bare `RestModel` (`templates/processor.go:162,169-191` `canonicalBytes`/`Create`), so no drift-attribute field is ever written into the persisted document. Verified further by `TestComputedAttributesAreNotPersisted` (`templates/processor_test.go:615`).
- **D4 — Re-seed writes via `canonicalBytes`, not `UpdateById`:** `templates/processor.go:256` (`ReseedById`) calls `canonicalBytes(entry.Model)` directly and reuses the `update(...)` transaction function, bypassing `UpdateById`'s preset-validator reassignment (`templates/processor.go:193-227`). Verified by `TestReseedProducesSameBytesAsFreshCreate` (`templates/processor_test.go:829`).
- **D5 — Reseed not tenant-scoped:** confirmed under DOM-10 above.
- **D6 — Handler self-writes 404/409:** `templates/resource.go:192-202` (`writeJSONAPIError`) and `:214-231`. This mirrors the pre-existing local pattern already used for validation failures at `resource.go:49-55` and `:155-161` (both predate this diff), so it is consistent with the file's established convention for statuses `server.WriteErrorResponse` cannot express, not a new "custom error response helper" anti-pattern introduced by this change.

## Pre-existing gaps observed (NOT introduced by this diff — informational only)

- `templates/rest.go` has no `Transform(Model) (RestModel, error)` / `TransformSlice` functions (DOM-04/DOM-05), and `templates/entity.go` has no `Model.ToEntity()` (DOM-02) — this package uses `RestModel` directly as both the domain and wire model (`Make` lives in `processor.go:128`, not `entity.go`), a pattern that predates this branch (confirmed via `git diff` on `rest.go` showing only the `ViewRestModel` addition, base file otherwise unchanged). Flagged for visibility; not counted against this PR since none of the touched code introduced or worsened it — `ViewRestModel` follows the same existing convention rather than a new one.

## Summary

### Blocking (must fix)
None found in the scoped diff.

### Non-Blocking (should fix)
- Pre-existing absence of `Transform`/`TransformSlice`/`ToEntity` in the `templates` package (see above) — out of scope for this PR, but worth a follow-up if the package is ever brought fully into the standard domain-package shape.
