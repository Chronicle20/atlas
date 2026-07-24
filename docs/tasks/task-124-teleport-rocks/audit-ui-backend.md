# Backend Audit — task-124 Teleport Rocks (list-editor delta)

- **Scope:** Go changes only, commit range `1f084b302..35030b125` (the synchronous Add/Remove + REST + UI-wiring slice; the earlier async/packet slice was already audited in `audit-backend.md`).
- **Files in scope:**
  - `services/atlas-character/atlas.com/character/teleport_rock/{processor.go, rest.go, resource.go, mock/processor.go, processor_test.go, rest_test.go}`
  - `services/atlas-character/atlas.com/character/main.go` (teleport_rock `InitResource` wiring + injected `WorldIdOf` resolver)
  - `services/atlas-character/docs/rest.md`
- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/resources/*`
- **Date:** 2026-07-18
- **Build/Tests:** Reported green by the caller (`go build ./...`, `go test -race ./...`, `go vet ./...`, `tools/lint.sh --check --go` all clean) — not re-run; spot-read only.
- **Overall:** NEEDS-WORK

## Domain Checklist Results — `atlas-character/teleport_rock` (delta only)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor accepts `FieldLogger` | PASS | `processor.go:62` — `NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor` unchanged; `Add`/`Remove` added to the same impl, no new constructor. |
| DOM-07 | Handlers pass `d.Logger()` | **PARTIAL FAIL** | `resource.go:44,78,113` all correctly call `NewProcessor(d.Logger(), d.Context(), d.DB())`. But `resource.go:73,108` call the injected `worldIdOf(d.Context(), characterId)`, whose *implementation* — `main.go:127-133` — constructs `character.NewProcessor(l, ctx, db)` using the **bootstrap-time root logger `l`** (`main.go:64`, `rt.Logger()`, a bare `*logrus.Logger`), not a request-scoped `FieldLogger`. A grep of `resource.go` alone gives a false PASS; the actual per-request character lookup on every POST/DELETE loses the `originator`/`type`/tenant/span fields that `d.Logger()` carries (see `rest/handler.go:72-76,87-91`: `RetrieveSpan` + `ParseTenant` build `tl` from `l`, and `d.Logger()` returns that `tl`, not `l`). Also a functional deviation from the task's own plan — `docs/tasks/task-124-teleport-rocks/plan-ui-list-editor.md:422-428` specified `characterWorldId(d *rest.HandlerDependency, characterId uint32)` using `d.Logger()`/`d.Context()`/`d.DB()`; the import-cycle constraint (real — `character` already imports `teleport_rock`) justifies moving the resolver out of `resource.go`, but not silently swapping the request-scoped logger for the boot-time one. **Severity: Important.** Fix: change `WorldIdOf`'s signature to accept a `logrus.FieldLogger` (`func(ctx context.Context, l logrus.FieldLogger, characterId uint32) (world.Id, error)`) and have both call sites in `resource.go` pass `d.Logger()` through. |
| DOM-08 | POST/PATCH use `RegisterInputHandler` | PASS | `resource.go:34` — `rest.RegisterInputHandler[AddMapInputRestModel](l)(db)(si)(...)`.Methods(http.MethodPost)`. DELETE correctly uses `RegisterHandler` (`resource.go:35`) per the table (DELETE has no body). |
| DOM-09 | Transform errors handled | PASS | `resource.go:141-146` (`writeModel`) checks the `model.Map(Transform)(...)()` error explicitly before marshaling. |
| DOM-12 | No `os.Getenv` in resource.go | PASS | None found in `resource.go`. |
| DOM-13/14 | No cross-domain logic / no direct provider calls in handlers | PASS | Handlers call only `NewProcessor(...).Add/Remove/GetByCharacterId` and the injected `worldIdOf`; no provider functions called directly. |
| DOM-15 | No direct entity creation in handlers | PASS | No `db.Create/Save/Delete` in `resource.go`. |
| DOM-17 | Domain error → HTTP status mapping | PASS | `rest.go:69-80` (`statusForError`) maps `ErrMapNotAllowed`→400, `ErrListFull`/`ErrDuplicate`→409, `ErrNotFound`→404; `resource.go:80-87,115-122` use it and fall back to `server.WriteErrorResponse(d.Logger())(w)(err)` (503-capable via the classifier registered at `main.go:84-90`) for unrecognized errors — matches `patterns-resilience.md`'s 503 contract, not a bare 500. |
| DOM-18 | JSON:API interface on REST models | PASS | `rest.go:21-32` (`RestModel`) and `rest.go:52-63` (`AddMapInputRestModel`) both implement `GetName`/`GetID`/`SetID`. |
| DOM-19 | Request models use flat structure | PASS | `rest.go:46-50` — `AddMapInputRestModel{Id, List, MapId}`, no nested Data/Type/Attributes. |
| DOM-20 | Table-driven tests | **FAIL (Minor)** | `processor_test.go:79-233` and `rest_test.go:10-54` are all discrete `Test*` functions, not `tests := []struct{...}{}` + `t.Run(...)`. `TestStatusForError` (`rest_test.go:42-54`) comes closest but iterates a bare `map[error]int` with no `t.Run` subtests (map order is nondeterministic, though harmless here since it uses `t.Errorf` so every case still runs). `testing-guide.md:18` softens this to "Prefer table-driven tests" (not MUST), so graded Minor rather than blocking — but it is a real, citable deviation from the DOM-20 pass criteria, not a pass. |
| DOM-24 | Kafka producer stubbed in tests that emit | PASS | `processor_test.go:26-29` — `TestMain` calls `producertest.InstallNoop()` before `Add`/`Remove` (which drive `message.Emit(producer.ProviderImpl(...))` per `processor.go:151,230`) are exercised anywhere in the package. No `t.Cleanup(producer.ResetInstance)` found (`grep -rn "ResetInstance\|t.Cleanup" teleport_rock/*_test.go` → no matches) — the stub stays installed for the whole package run. |
| DOM-26 | Goroutines via `routine.Go` | PASS | No new `go` statements in the diff (`main.go`'s existing `routine.Go` calls unchanged). |
| DOM-27 | Transient DB errors → 503, never bare 500 | PASS | `resource.go:75,85,110,120` all route the fallback branch through `server.WriteErrorResponse(d.Logger())(w)(err)`, not a bare `w.WriteHeader(http.StatusInternalServerError)`; the classifier is registered once at `main.go:84-90`. (The earlier async-slice audit had flagged and fixed the GET handler's version of this bug; this new POST/DELETE code was written correctly from the start.) |

## File Responsibilities Checklist (delta)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor logic in `processor.go` | PASS | All new/changed `Processor` interface methods and `ProcessorImpl` methods (`Add`, `Remove`, `addMap`, `removeMap`, `reasonForError`, the `Err*` sentinels) live in `processor.go`. No leakage into `resource.go`/`rest.go`. |
| FILE-02 | `RestModel`/`Transform`/JSON:API methods in `rest.go` | PASS | `AddMapInputRestModel`, `statusForError`, capacity fields on `RestModel` all in `rest.go`. |
| FILE-06 | No package-named catch-all file | PASS | No `teleport_rock.go`; each touched file stays single-purpose. |

## Multitenancy (patterns-multitenancy-context.md)

| Check | Status | Evidence |
|-------|--------|----------|
| Request context (with tenant) flows through the injected `WorldIdOf` | PASS | `resource.go:73,108` pass `d.Context()` (the `tctx` built by `server.ParseTenant`, per `rest/handler.go:74,89`) into `worldIdOf`; `main.go:127-133`'s closure forwards that same `ctx` unmodified into `character.NewProcessor(l, ctx, db)`. Tenant isolation for the cross-domain `character` lookup is preserved even though (per the DOM-07 finding above) the *logger* used for that lookup is not request-scoped. |
| `Add`/`Remove` DB writes use `p.db.WithContext(p.ctx)` | PASS | `processor.go:111,193` (`addMap`/`removeMap`, unchanged from the pre-existing `AddMap`/`RemoveMap` transaction wrapping) — both call `database.ExecuteTransaction(p.db.WithContext(p.ctx), ...)`. |

## Kafka / `message.Emit` Atomicity

| Check | Status | Evidence |
|-------|--------|----------|
| `Add`/`Remove` buffer-then-flush is atomic and validation failures buffer nothing | PASS | `processor.go:149-160,228-239` wrap `p.addMap`/`p.removeMap` in `message.Emit(producer.ProviderImpl(p.l)(p.ctx))(...)`; `message.go:46-61` confirms `Emit` returns immediately on a non-nil inner error without flushing any buffered messages, and `addMap`/`removeMap` (`processor.go:109-143,191-222`) only call `mb.Put(...)` on the success path inside the same DB transaction that persists the mutation — so a validation failure (`ErrMapNotAllowed`/`ErrListFull`/`ErrDuplicate`/`ErrNotFound`) rolls back the transaction and buffers no event, and the typed sentinel propagates unmodified through `Emit` to the REST caller (verified directly by `processor_test.go:195-233`, `TestAddReturnsTypedErrors`/`TestAddReturnsListFull`). |

## Test Coverage Gap — Missing `resource_test.go`

| Check | Status | Evidence |
|-------|--------|----------|
| Handler-level (HTTP) tests for the new POST/DELETE wiring | **FAIL (Important)** | No `teleport_rock/resource_test.go` exists (`find ... -iname resource_test.go` → no match) despite `resource.go` gaining 91 new lines: route registration for two new methods, the `listVip` switch, `mapId` path-param parsing, the `worldIdOf` error branch, and the `statusForError` → `w.WriteHeader` wiring. `processor_test.go` exercises `Processor.Add/Remove` directly (bypassing HTTP entirely) and `rest_test.go` exercises only the pure `Transform`/`statusForError` functions — nothing drives an actual `http.Request` through `InitResource(...)(db)(worldIdOf)` via `httptest`/`mux`, so nothing verifies: (a) the JSON:API request-envelope round-trip for `AddMapInputRestModel` (the exact class of bug in project memory's `bug_ui_jsonapi_envelope_required_for_input_handlers.md`), (b) that an unknown `list` value actually yields HTTP 400 from the live handler, (c) that a `worldIdOf` failure is actually surfaced via `WriteErrorResponse`, or (d) that `statusForError`'s mapped codes are actually written to `w` in the live handler path. This is not a hypothetical gap — the established convention for exactly this kind of test exists in the **same service**: `session/history/resource_test.go:1-37` and `character/resource_test.go` both drive `InitResource(...)(db)` through a real `mux.Router` + `httptest` with tenant headers. `testing-guide.md`'s Focus Area 4 ("REST — Verify status mapping and JSON:API output") is unmet for this delta. |

## Documentation (`docs/rest.md`)

| Check | Status | Evidence |
|-------|--------|----------|
| New endpoints documented, paths/params/status codes match implementation | PASS | `docs/rest.md` (diff) — GET/POST/DELETE sections for `/characters/{characterId}/teleport-rock-maps[/{list}/{mapId}]` match `resource.go`'s actual routes and `statusForError`'s status codes (400/409 for POST, 400/404 for DELETE); response body shape matches `rest.go`'s `RestModel` fields including the new `regularCapacity`/`vipCapacity`. |
| Ingress | PASS (no change needed) | `deploy/shared/routes.conf:372` — the existing catch-all `location ~ ^/api/characters(/.*)?$` (proxying to `atlas-character:8080`) already covers the new sub-path; no new service/base-path was added, so per `patterns-ingress-documentation.md` no ingress edit was required. |

## Mock (`teleport_rock/mock/processor.go`)

| Check | Status | Evidence |
|-------|--------|----------|
| Mock kept in sync with interface | PASS | `mock/processor.go:19,22,46-51,67-72` — `AddFunc`/`RemoveFunc` fields and `Add`/`Remove` methods added, matching the interface exactly (`testing-guide.md`'s Interface Change Workflow); `var _ teleport_rock.Processor = (*ProcessorMock)(nil)` (`mock/processor.go:74`) statically enforces the contract. |

## Summary

### Blocking (must fix) — Important

- **DOM-07 (adjacent)**: `services/atlas-character/atlas.com/character/main.go:127-133` — the `WorldIdOf` closure builds `character.NewProcessor(l, ctx, db)` with the bootstrap root logger instead of a request-scoped `FieldLogger`, on every new POST/DELETE `teleport-rock-maps` call. Thread `d.Logger()` through `WorldIdOf`'s signature instead of closing over `l`.
- **Test coverage**: `services/atlas-character/atlas.com/character/teleport_rock/` has no `resource_test.go` exercising the new POST/DELETE HTTP wiring end-to-end (route registration, `listVip`, mapId parsing, `worldIdOf` error path, status-code writing, JSON:API envelope round-trip for `AddMapInputRestModel`). Add one following the established `session/history/resource_test.go` pattern in the same service.

### Non-Blocking (should fix) — Minor

- **DOM-20**: `processor_test.go` and `rest_test.go`'s new tests are scenario-per-function rather than `[]struct{...}` + `t.Run` table-driven (softened to "Prefer" in `testing-guide.md`, so non-blocking). `TestStatusForError` (`rest_test.go:42-54`) in particular would benefit from `t.Run` subtests for per-case failure attribution.
