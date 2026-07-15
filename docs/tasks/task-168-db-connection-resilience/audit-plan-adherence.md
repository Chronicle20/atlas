# Plan Audit — task-168-db-connection-resilience

**Plan Path:** docs/tasks/task-168-db-connection-resilience/plan.md
**Audit Date:** 2026-07-12
**Branch:** task-168-db-connection-resilience
**Base Branch:** main (merge-base ab6794297e)
**Scope:** READ-ONLY plan-adherence audit. Docker bake not re-run (parent confirms all 59 images built, exit 0).

## Executive Summary

All 14 plan tasks are faithfully implemented with file:line evidence for every stated deliverable and interface. No task is missing, stubbed, a TODO, or deferred. Each planned test file exists (11/11); the two acceptance-critical reference suites (atlas-login incident replay, atlas-inventory 503 contract) pass on focused re-run. The documented design.md deviation (atlas-login has no DB → no classifier) is correctly honored, and the plan's own T12 example-test compile bug was correctly avoided in the landed code. Verdict: FULL adherence, READY_TO_MERGE.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `retry.WithDelayHint` hook | DONE | `libs/atlas-retry/retry.go:95-100` (WithDelayHint + `delayHintError` with `Unwrap`); honored in `Try` at `:64-71` (cap at MaxDelay). Commit b2852e5842. |
| 2 | Transient classifier | DONE | `libs/atlas-database/transient.go:28` `IsTransientConnectionError`, `:53` `TransientSQLState`; SQLSTATE allow-list 53300/57P03/08001/08006 at `:14-19`; PgError checked before ConnectError (`:32-39`). Commit 07e4691919. |
| 3 | DB metrics counters | DONE | `libs/atlas-database/metrics.go:13` `acquireRetriesTotal`, `:21` `transientErrorsTotal`, `:32` `CountTransient`, `:39` `registerDBStats` (Register+Warn, not MustRegister). Commits 0d525ba025 + 11c2b22c81. |
| 4 | Retry connector + Connect() swap | DONE | `libs/atlas-database/connector.go:25` `newRetryConnector` (≤1 returns base), `:37` Connect retry loop; `connection.go:110-139` swap to `sql.OpenDB(newRetryConnector(...))` + `postgres.New{Conn}` + `registerDBStats`. `TestMidStatementErrorIsNotRetried` present in connector_test.go. Commit 3364a107a1. |
| 5 | Server-side 503 contract | DONE | `libs/atlas-rest/server/error.go:14` `TransientRetryAfterSeconds=1`, `:31` `RegisterTransientErrorClassifier` (atomic.Pointer), `:48` `WriteErrorResponse` (503+Retry-After+JSON:API body). No atlas-database import. Commit 68cc3fdc2e. |
| 6 | Client GET retry on 503 | DONE | `requests/get.go:22` `ErrServiceUnavailable`, `:45` default retries 3, `:83-92` 503 handling + WithDelayHint, `:95` MaxDelay 2s; `requests/metrics.go:9` `atlas_rest_client_retries_total`. post/patch/put/delete.go untouched (diff empty). Commit a72071ed24. |
| 7 | Auto-mount `/metrics` + remove 4 mounts | DONE | `server/server.go:107` seeds `routeInitializers` with `MountHandler("/metrics", promhttp.Handler())`; grep confirms 0 explicit `/metrics` mounts remain in channel/summons/doors/monsters; each main.go −2 lines in diffstat. Commit f954ec7640. |
| 8 | `model.ErrDecorator` | DONE | `libs/atlas-model/model/processor.go:107` `ErrDecorator[M]`; module stays dependency-free (onErr is injection point). Commit f765780120. |
| 9 | `degrade.Observe` | DONE | `libs/atlas-rest/degrade/degrade.go` `degradedTotal{component}` + `Observe(l, component, entityId, err)` (Warn + counter; id in log only). Commit 7e9b578ed0. |
| 10 | login InventoryDecorator loud-degrade | DONE | `login/character/processor.go:109-122` `model.ErrDecorator` + `degrade.Observe(p.l, "login.character.inventory", m.Id(), err)`; processor_test.go present with incident replay test. No classifier in login (correct — no DB). Commit 8505129bb2. |
| 11 | atlas-inventory 503 adoption | DONE | `inventory/main.go:64-67` classifier registration; 0 bare `StatusInternalServerError` writes remain; `WriteErrorResponse` at 3(asset)+5(inventory)+4(compartment)=12 sites; resource_test.go present. Commit 60e36aaa51. |
| 12 | Decorator audit + fixes | DONE | `decorator-audit.md` (2 impls: login InventoryDecorator, skills CooldownDecorator); `skills/skill/processor.go:248` CooldownDecorator now `ErrDecorator`+`degrade.Observe("skills.skill.cooldown", characterId, err)` and now propagates the build error the original dropped via `_`; 3 FR-5.4 char-select `GetRandomInWorld` silent drops fixed with loud abort. Commits 07e4691919-range: 07e..c4efa. |
| 13 | Docs + reviewer checklist | DONE | `patterns-resilience.md` + `libs/atlas-database/README.md` present; SKILL.md:162 references patterns-resilience; DOM-26/DOM-27 both present in backend-guidelines-reviewer.md. Commit beb75089d2. |
| 14 | Tidy sweep + verification battery | DONE | go.mod/go.sum sweep across ~all services (diffstat); commit 3f3e3b07b6. Bake verified by parent (59 images, exit 0). |

**Completion Rate:** 14/14 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None.

## Observations (non-findings)

- **`// TODO issue error` in the FR-5.4 char-select fix** (`character_view_all_selected.go:60` and the two sibling `*_pic*` handlers). This is NOT a newly introduced stub: 3 identical `// TODO issue error` comments already exist on the base branch (ab6794297e:27,46,52) as this codebase's convention for "send a client-facing error packet." The landed change is a real behavioral fix — a loud `Errorf` + `return` abort that prevents routing the player to dead channel 0 / empty IP — mirroring the pre-existing sibling `world.GetById`/`character.GetById` error handling in the same files. The TODO is aspirational client-error-packet emission consistent with the 3 pre-existing ones, not a stubbed handler; the functional fix is complete.
- **login has NO classifier** — correct per context.md's documented design.md deviation (login has no `database.Connect`). Verified absent in `login/main.go`.
- **T12 example-test compile bug** — the plan's T12 canonical example was illustrative; landed code uses the correct shapes; both reference suites compile and pass.

## Build & Test Results

Focused re-runs only (full battery + bake already passed per parent; not re-run to respect READ-ONLY / time).

| Service / Module | Build | Tests | Notes |
|---|---|---|---|
| atlas-login (character) | PASS | PASS | `go test ./character -run TestInventoryDecorator` → ok (degrade + incident-replay). |
| atlas-inventory (inventory) | PASS | PASS | `go test ./inventory -run TestGetInventory` → ok (503 + Retry-After, non-transient 500). |
| All 11 planned test files | — | present | retry, transient, metrics, connector, error, server, get_retry, degrade, decorator, login processor, inventory resource — all exist. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None. All 14 tasks implemented with evidence; acceptance criteria (§10 mapping) each map to a real test or artifact:
classifier+table → transient_test.go; acquire retry + mid-statement-not-retried + knobs → connector_test.go; inventory 503 → resource_test.go; client 503 matrix → get_retry_test.go; InventoryDecorator loud + incident replay → login processor_test.go; decorator-audit zero-silent → decorator-audit.md; gauges + 4 counters + /metrics mount → metrics/degrade/server tests; docs + DOM-26/27 → Task 13 artifacts; verification battery + bake → parent-confirmed.
