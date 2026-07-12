# Backend Audit — task-168 (DB Connection Resilience)

- **Branch:** task-168-db-connection-resilience (merge-base ab6794297e)
- **Guidelines Source:** backend-dev-guidelines skill (DOM-*/SUB-*/SEC-* + task-added DOM-26/DOM-27)
- **Date:** 2026-07-12
- **Build:** PASS (libs/atlas-retry, atlas-database, atlas-rest, atlas-model; services atlas-inventory, atlas-login, atlas-skills)
- **Tests:** PASS (all changed modules, `go test ./... -count=1`)
- **Vet:** clean except one PRE-EXISTING warning (`server.go:187-188` WaitGroup.Add-inside-goroutine) NOT introduced by this task
- **Overall:** PASS (no blocking findings; 2 minor non-blocking observations)

## Build & Test Results

All four changed lib modules and the three substantively-changed services build and test green:

```
libs/atlas-retry            ok
libs/atlas-database         ok  (+ databasetest)
libs/atlas-rest/requests    ok  (11.2s — real httptest retry timing)
libs/atlas-rest/server      ok
libs/atlas-model/model      ok
atlas-inventory             ok  (inventory/resource_test 503/500 mapping)
atlas-login                 ok  (world; character read tests)
atlas-skills                ok  (skill, macro)
```

## Checklist Results (scoped to task-168 substantive changes)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | Reuse atlas-constants; no new domain types | PASS | Task adds only lib-level utilities (`IsTransientConnectionError`/`TransientSQLState`/`CountTransient` in `libs/atlas-database/transient.go:28,53` + `metrics.go:32`; `ErrDecorator` in `libs/atlas-model/model/processor.go:107`; `degrade.Observe` in `libs/atlas-rest/degrade/degrade.go:25`). No new `type X` domain/id/classification declaration anywhere in the diff. |
| DOM-22 | Dockerfile mentions per new direct lib require | PASS | Shared root `Dockerfile` (ARG SERVICE). `atlas-retry` present in mod-copy (`Dockerfile:43`), source-copy (`Dockerfile:72`), synthesized in-image `go.work use` (`Dockerfile:93`); `atlas-database` at `:33,:62,:92`. New direct requires in `libs/atlas-database/go.mod:7` (`atlas-retry`, with replace `:48`) and `:10` (`jackc/pgx/v5` — third-party, resolved via go.sum, no Dockerfile row needed). No service go.mod gained a new *direct* `Chronicle20/atlas/libs/*` require (verified across all changed service go.mods). |
| DOM-24 | Kafka producer stubbed in emitting tests | PASS | The two new httptest-backed service tests exercise READ-only paths with no emit: `atlas-inventory/.../inventory/resource_test.go:79,101` drives `handleGetInventory` (GET → `GetByCharacterId`, no producer); `atlas-login/.../character/processor_test.go:55,96` drives `GetById` (REST read). Lib tests (connector/metrics/transient/degrade/error/get_retry/decorator) do not emit. No unstubbed `AndEmit`/`message.Emit`/`producer.Produce` in any changed `_test.go`. |
| DOM-26 | Transient DB errors → 503, never bare 500 | PASS (inventory) | Classifier registered once: `atlas-inventory/.../main.go:64-70` composes `database.IsTransientConnectionError` + `database.CountTransient`. All three changed resource files route error branches through `server.WriteErrorResponse(d.Logger())(w)(err)` (inventory/resource.go:45,53,72,79,98; asset/resource.go:34,41,63; compartment/resource.go:41,48,87,94); 404/400 branches correctly exempt (inventory/resource.go:35,41). Server contract in `libs/atlas-rest/server/error.go:48-66` maps classifier-transient → 503 + `Retry-After: 1`; verified by `error_test.go` and `resource_test.go:79`. |
| DOM-27 | No silent degradation in decorators/enrichment | PASS | Both fleet decorators degrade loudly: `InventoryDecorator` (`atlas-login/.../character/processor.go:109-122`) = `model.ErrDecorator` + `degrade.Observe(p.l,"login.character.inventory",m.Id(),err)`; `CooldownDecorator` (`atlas-skills/.../skill/processor.go:248-265`) = `model.ErrDecorator` + `degrade.Observe(p.l,"skills.skill.cooldown",characterId,err)`, and now also propagates the previously-dropped `CloneModel(...).Build()` error. `ErrDecorator` (`model/processor.go:107-116`) requires non-nil `onErr`. The three char-select `GetRandomInWorld` sites (`character_view_all_selected.go:58-63`, `..._pic.go:92-97`, `..._pic_register.go:66-71`) are loud ABORTS (`Errorf` + `return`), not silent data-drops. |
| Server 503 contract | `WriteErrorResponse` + `TransientRetryAfterSeconds=1` | PASS | `libs/atlas-rest/server/error.go:14,48-66`; classifier injected (no atlas-database import), atomic-pointer registry. |
| Client GET retry-on-503 | GET-only, 3 attempts, MaxDelay 2s, Retry-After honored | PASS | `libs/atlas-rest/requests/get.go:45,83-92,95`; sentinel `ErrServiceUnavailable` (`:22`) via `errors.Is`; POST/PATCH/PUT/DELETE untouched (only get.go modified). `retry.WithDelayHint` (`libs/atlas-retry/retry.go`) honors Retry-After capped at MaxDelay. |
| Acquire-phase retry | `driver.Connector` wrapper, disabled at attempts<=1 | PASS | `libs/atlas-database/connector.go:25-61`; retries only `base.Connect` (acquire phase, structurally cannot double-apply statements); classifier gates retry (`:45`); metrics on each retry (`:50`). |
| /metrics auto-mount | Builder default + explicit mounts removed | PASS | `libs/atlas-rest/server/server.go:107` mounts `/metrics` as default RouteInitializer; explicit mounts removed from atlas-channel/doors/monsters/summons main.go (diff confirms `-AddRouteInitializer(...MountHandler("/metrics"...))` in all four). No double-registration. |

## Non-Blocking Observations

- **atlas-skills DB-backed but retains bare-500 REST handlers.** `atlas-skills` calls `database.Connect` (main.go:61) and its resource handlers write `http.StatusInternalServerError` directly (`skill/resource.go:37,55,73,93`; `macro/resource.go:32`) with no `RegisterTransientErrorClassifier` in main.go — so a transient pool-exhaustion error on a skills DB read surfaces as bare 500, not 503. This is **out of DOM-26's literal scope** (DOM-26 fires on *changed* resource handlers; skills' resource.go was not changed — only `skill/processor.go`). The task deliberately scoped 503 adoption to atlas-inventory as the reference implementation. Flagged as a fleet-completeness inconsistency, not a rule violation.

- **`// TODO issue error` in three login char-select handlers** (`character_view_all_selected.go:61`, `..._pic.go:95`, `..._pic_register.go:69`). CLAUDE.md bans TODOs in landed commits, but these are a **pre-existing idiom** already present on the sibling world/capacity/character branches in the same files, and the accompanying behavior (loud abort: `Errorf` + `return`) is complete — not a silent stub or 501. "issue error" refers to an optional client-facing error packet no sibling path sends either. Minor.

- **Pre-existing vet warning** at `libs/atlas-rest/server/server.go:187-188` (`sb.wg.Add(1)` inside the spawned goroutine). The diff of server.go touches only the `/metrics` default mount + import; this WaitGroup block predates task-168 and is out of scope.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should consider)
- atlas-skills: extend the 503 contract (classifier + `WriteErrorResponse`) to its DB-backed REST handlers for fleet consistency (`skill/resource.go:37,55,73,93`; `macro/resource.go:32`).
- Replace the three `// TODO issue error` markers with real client error-packet emission (or drop the comment) on the login char-select abort paths.
