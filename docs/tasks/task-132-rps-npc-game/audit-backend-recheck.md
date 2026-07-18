# Backend Audit Recheck — task-132 RPS NPC Minigame

- **Scope:** `git diff b1c50b67d 7ae96ef4b -- '**/*.go'` (main..HEAD, full RPS feature as it lands on main)
- **Services:** atlas-rps, atlas-tenants (configuration/rps-rewards), atlas-channel (rps/, socket/handler/rps_action.go, kafka/consumer/rps), atlas-saga-orchestrator (rps/, saga StartRPSGame), atlas-npc-conversations (rpsAction), libs/atlas-packet/rps, libs/atlas-saga
- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/*` (DOM-01..28, FILE-01..06, SUB-01..04, EXT-01..04), `docs/rest-pagination.md`, `patterns-saga.md`, `patterns-resilience.md`, `patterns-deploy.md`
- **Date:** 2026-07-16
- **Mindset:** FAIL until file:line evidence proves PASS. This document supplements (does not replace) `audit-backend.md`; findings below were not present in that prior pass.

## Build & Test Gate (objective)

| Module | `go build ./...` | `go vet ./...` | `go test -race ./... -count=1` |
|---|---|---|---|
| `services/atlas-rps/atlas.com/rps` | PASS | PASS | PASS (all packages) |
| `services/atlas-tenants/atlas.com/tenants` | PASS | PASS | PASS (`configuration` pkg) |
| `services/atlas-channel/atlas.com/channel` | PASS | — | PASS (`rps`, `kafka/consumer/rps`, `socket/handler`) |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator` | PASS | — | PASS (`rps`, `saga`) |
| `services/atlas-npc-conversations/atlas.com/npc` | PASS | — | PASS (`conversation`, `kafka/consumer/saga`) |
| `libs/atlas-packet` | PASS | — | PASS (`rps/clientbound`, `rps/serverbound`) |
| `libs/atlas-saga` | PASS | — | PASS |

`tools/goroutine-guard.sh`, `tools/redis-key-guard.sh`, `tools/service-registration-guard.sh` all exit 0 from repo root.

## Findings — Important

### IMP-1: `GetAllRpsRewardsHandler` is a bare unpaginated collection endpoint (PS-5 / rest-pagination.md violation)

`services/atlas-tenants/atlas.com/tenants/configuration/resource.go:616-653` — `GetAllRpsRewardsHandler` builds `restModels []RpsRewardRestModel` and marshals it directly:

```go
server.MarshalResponse[[]RpsRewardRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(restModels)
```

No `paginate.ParseParams`, no `paginate.Slice`, no `server.MarshalPaginatedResponse`. Every sibling collection handler added in the **same file, same commit** correctly follows the mandatory pattern:

- `GetAllRoutesHandler` (resource.go:19-70) — `paginate.ParseParams` (24) → `paginate.Slice` (62) → `server.MarshalPaginatedResponse[[]RouteRestModel]` (66)
- `GetAllVesselsHandler` (resource.go:219-270) — same pattern (224, 262, 266)
- `GetAllInstanceRoutesHandler` (resource.go:419-470) — same pattern (424, 462, 466)

`docs/rest-pagination.md` §0: "Every REST route that returns a collection (`GET` on a resource-collection path) must page its result... adopted repo-wide (task-117)." There is no per-endpoint opt-out in the doc's §3 override table for `rps-rewards`. This is a straight regression against the pattern the RPS feature itself uses three times in the same file.

**Compounding risk:** the only current consumer, `services/atlas-rps/atlas.com/rps/configuration/processor.go:47`, reads it via `requests.SliceProvider[RpsRewardRestModel, game.Ladder]` (not `DrainProvider`). Per `docs/rest-pagination.md` §7's "No-envelope compatibility rule," `SliceProvider` only silently degrades to "page 1 only" once the producer starts emitting a `meta` envelope — today the two sides are consistent (neither paginates), but fixing the producer alone (the correct fix) without also converting the consumer to `DrainProvider` will silently truncate ladders with more entries than one page.

**Severity: Important** (structural / mandatory-pattern violation per repo-wide policy; not currently crash-affecting since ladders are small, but violates a documented MUST and creates a latent truncation trap).

### IMP-2: RPS reward CRUD handlers bypass `server.WriteErrorResponse` (DOM-27) and one uses a custom error-response helper (documented anti-pattern)

All five CRUD handlers plus the seed handler for `rps-rewards`, added in `services/atlas-tenants/atlas.com/tenants/configuration/resource.go`, write `w.WriteHeader(http.StatusInternalServerError)` directly on internal-error branches instead of `server.WriteErrorResponse(d.Logger())(w)(err)`:

- `GetAllRpsRewardsHandler` — lines 631, 641
- `GetRpsRewardByIdHandler` — line 673 (transform-error branch; the not-found branch at 666 correctly stays a plain 404, matching siblings)
- `CreateRpsRewardHandler` — lines 702, 716, 723
- `UpdateRpsRewardHandler` — lines 753, 761, 768
- `DeleteRpsRewardHandler` — line 791
- `SeedRpsRewardsHandler` — lines 1084-1085, which additionally hand-rolls a custom error body: `json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})`

Every sibling handler in the **same file** for routes/vessels/instance-routes/mts-configs uses `server.WriteErrorResponse(d.Logger())(w)(err)` on the equivalent branches, e.g. `CreateRouteHandler` (resource.go:119), `CreateVesselHandler` (319), `CreateInstanceRouteHandler` (519), `CreateMtsConfigHandler` (954, 968, 975), `SeedRoutesHandler`/`SeedVesselsHandler`/`SeedMtsConfigsHandler` (811, 853, 1063).

Per `docs/…/patterns-resilience.md` (DOM-27): "Transient DB errors MUST surface as `503 Service Unavailable`... never a generic 500... replace `w.WriteHeader(http.StatusInternalServerError)` with `server.WriteErrorResponse(...)`." atlas-tenants is DB-backed and **already** registers the classifier — `server.RegisterTransientErrorClassifier(...)` at `services/atlas-tenants/atlas.com/tenants/main.go:61`, composing `database.IsTransientConnectionError` (62). The classifier is wired and active; the RPS reward handlers simply don't route through it, so a transient PG connection blip during a rps-reward write surfaces to the caller as a generic uncategorized 500 instead of a 503 + `Retry-After`, while the identical failure on `/configurations/routes` or `/configurations/mts-configs` correctly returns 503.

The `SeedRpsRewardsHandler` custom JSON body is also the documented anti-pattern "Custom error response helpers | Just write status codes directly" (`anti-patterns.md`).

This directly contradicts the specific ask to verify the RPS reward `*AndEmit` methods "exactly match the sibling MtsConfig methods" — the **processor-layer** `*AndEmit` methods do match (see PASS-2 below), but the **handler-layer** error-response wiring built on top of them does not.

**Severity: Important** (DOM-27 is a mandatory, already-adopted-in-this-file pattern; the deviation is isolated to RPS-added code, straightforward to fix by copy-pasting the sibling handler's error branches).

## Findings — Minor

### MIN-1: `atlas-rps/saga` package has no test coverage

`services/atlas-rps/atlas.com/rps/saga/` contains `processor.go`, `producer.go`, and `mock/processor.go` but no `processor_test.go` (confirmed: `find services/atlas-rps/atlas.com/rps/saga -type f` returns exactly those three files). Its sibling `atlas-rps/configuration` package — structurally identical (one `Processor` interface, one method, a thin wrapper over a REST/Kafka call) — has a 143-line `processor_test.go`. `testing-guide.md`'s Focus Areas #2 ("Processors — Test pure and AndEmit forms separately") applies; `saga.ProcessorImpl.Create` (the sole method, used on the payout path in `game.Collect`) is unexercised by any unit test.

**Severity: Minor** (low complexity method, exercised transitively by `game` package tests via the `SagaSubmitter` seam, but a direct unit test is the documented convention and is missing).

### MIN-2: EXT-01 — cross-service REST client models missing relationship interface methods

Two client-side `RestModel`s that call another atlas service via `requests.*Request[T]` lack `SetToOneReferenceID`/`SetToManyReferenceIDs`:

- `services/atlas-rps/atlas.com/rps/configuration/rest.go:22` `RpsRewardRestModel` (consumed via `requests.SliceProvider` at `configuration/processor.go:47`, reading atlas-tenants' `rps-rewards` resource)
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/rps/rest.go:15` `RestModel` (consumed via `requests.PostRequest[RestModel]` at `rps/requests.go:26`, calling atlas-rps's `/rps/games`)

`EXT-01` (backend-guidelines-reviewer checklist): "Both methods present, even if no-op. Without them, api2go errors on any response with a `relationships` block." Neither upstream response (atlas-tenants' rps-rewards, atlas-rps' game session) currently emits a `relationships` block, so this is latent rather than actively broken. It mirrors an existing gap in several sibling client packages in the same services (`gachapon`, `monster`, `rates`, `storage`, `saga`, `transport`, `saved_location` in atlas-saga-orchestrator also lack it), so it is not a regression unique to this branch, but it is a real gap against the documented checklist for a newly-added package.

**Severity: Minor** (no active break; documented gap, consistent with — not worse than — several pre-existing siblings).

## Confirmed PASS (evidence for the five flagged risk areas)

### PASS-1: `atlas-rps` `main.go` — `service.Bootstrap` + `MountReadiness`, local `logger/`/`kafka/producer/` fully removed

- `services/atlas-rps/atlas.com/rps/main.go:41` `rt := service.Bootstrap(serviceName)`; `:66` `AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready))`. Structure is byte-for-byte parallel to the sibling registry-backed service `atlas-chairs/atlas.com/chairs/main.go` (Bootstrap → Redis connect → `InitRegistry` → consumer setup → producer teardown → `MountReadiness`).
- `find services/atlas-rps -iname "*logger*"` and `find services/atlas-rps -path "*kafka/producer*"` both return empty — the local packages are fully deleted, not shadowed.
- All producer call sites (`main.go:53`, `game/processor.go:243,333,375,487,514,545`, `game/task.go:47`, `game/producer.go`, `saga/processor.go:43`, `saga/producer.go`) import `github.com/Chronicle20/atlas/libs/atlas-kafka/producer`; zero references to the deleted local package remain (`grep -rn "atlas-rps/logger\|atlas-rps/kafka/producer"` across the repo returns nothing).

### PASS-2: atlas-tenants RPS `*AndEmit` methods match the MtsConfig sibling exactly; no leaked direct-emit

`services/atlas-tenants/atlas.com/tenants/configuration/processor.go` — `CreateRpsRewardAndEmit` (1341-1353), `UpdateRpsRewardAndEmit` (1430-1442), `DeleteRpsRewardAndEmit` (1463-1469) are structurally identical to `CreateMtsConfigAndEmit` (749-761), `UpdateMtsConfigAndEmit` (838-850), `DeleteMtsConfigAndEmit` (871-877): each wraps its work in `database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {...})`, emits via `message.EmitWithResult[...](outbox.EmitProvider(p.l, p.ctx, tx))` (or `message.Emit` for Delete), and re-enters through `NewProcessor(p.l, p.ctx, tx)` so the read-modify-write inside the closure sees the transaction. No RPS method calls the old direct-emit shape (`message.Emit(producer.ProviderImpl(...))`); `outbox.EmitProvider` (`libs/atlas-outbox/provider.go:20-29`) only enqueues to a DB outbox table within the same `tx`, confirming no live Kafka call happens inside the transaction. `configuration/kafka.go`'s `CreateRpsRewardStatusEventProvider` (85-95) mirrors `CreateMtsConfigStatusEventProvider` (61-71) exactly. Mock (`configuration/mock/processor.go:72-81,508-604`) fully synchronized (interface-change workflow followed).

### PASS-3: Gen3 conformance (`var _ Processor = (*ProcessorImpl)(nil)`)

Present in all three new atlas-rps processors and both new mock packages:
- `game/processor.go:173`
- `configuration/processor.go:39`
- `saga/processor.go:39`
- `configuration/mock/processor.go:17` `var _ configuration.Processor = (*ProcessorMock)(nil)`
- `saga/mock/processor.go:15` `var _ saga.Processor = (*ProcessorMock)(nil)`

(Note: this assertion is not universally present across atlas-channel's actions packages — e.g. `rps/processor.go` and the sibling `mts/processor.go` in atlas-channel both omit it — so it is not flagged as missing there; the task's ask was specifically scoped to the three atlas-rps service processors, which all comply.)

### PASS-4: goroutine spawns via `routine.Go` (DOM-25/26)

`services/atlas-rps/atlas.com/rps/tasks/task.go:19` and `main.go:55` both spawn via `routine.Go(l, ctx, func(_ context.Context) {...})`. `tools/goroutine-guard.sh` exits 0 from repo root with no `//goroutine-guard:allow` markers needed anywhere in the RPS diff; `grep -rnE '^\s*go (func|[A-Za-z_])'` over every RPS-touched package returns zero non-test matches.

### PASS-5: `GET /rps/games/{characterId}` is correctly single-resource; no pagination/DOM violation

`services/atlas-rps/atlas.com/rps/game/resource.go:34,67-95` — `handleGetGame` calls `Processor.Get(characterId)` (one session per character, keyed 1:1) and marshals via `server.MarshalResponse[RestModel]` (91, singular, not `[]RestModel`). This is a single-resource GET, not a collection endpoint, so `docs/rest-pagination.md` does not apply. `handleCreateGame` (POST) likewise marshals a single `RestModel` (59). No DOM-08/DOM-09/DOM-14/DOM-15 violations in `game/resource.go`: POST uses `RegisterInputHandler[RestModel]` (30,33), GET uses `RegisterHandler` (29,34), both delegate exclusively to `newProcessor(...).StartAndEmit`/`.Get` (43,71) — no direct provider or registry calls from the handler.

### PASS-6 (supplemental): config-resolved wire values (DOM-25) for `RPSActionHandleFunc`

`services/atlas-channel/atlas.com/channel/socket/handler/rps_action.go` resolves the RPS_ACTION sub-op byte exclusively through the tenant `operations` table (`isRPSAction`, 114-136) — no hardcoded mode-byte literals. Verified present with matching validator (`LoggedInValidator`) and a full `operations` table (`START/SELECT/UPDATE/CONTINUE/EXIT/RETRY` = 0-5) in **all five** seed templates: `template_gms_83_1.json:1193-1207`, `template_gms_84_1.json:1158-1172`, `template_gms_87_1.json:830-844`, `template_gms_95_1.json:513-527`, `template_jms_185_1.json:942-956`.

### PASS-7 (supplemental): saga wiring

`handleStartRPSGame` (`saga/handler.go:3101-3116`) correctly follows the synchronous-action pattern from `patterns-saga.md`: it POSTs via `h.rpsP.StartGame(...)`, then self-completes with `NewProcessor(h.l, h.ctx).StepCompleted(s.TransactionId(), true)` immediately (3116) rather than waiting on an async event; on error it returns without completing, leaving the step for the framework's failure path. `StartRPSGame`/`handleStartRPSGame` registered in the dispatcher switch (`saga/model.go:169`, `saga/handler.go:904-905`). `libs/atlas-saga`'s `StartRPSGame` action, `StartRPSGamePayload`, and its `unmarshal.go` case (483-488) are all present and covered by `unmarshal_test.go`.

### PASS-8 (supplemental): Kafka topic + k8s registration

`COMMAND_TOPIC_RPS`/`EVENT_TOPIC_RPS` present in `deploy/k8s/base/env-configmap.yaml:68,143` and both `deploy/k8s/overlays/main/kustomization.yaml:107,180` and `deploy/k8s/overlays/pr/kustomization.yaml:163,236` (no per-service literal `env:` override). `tools/service-registration-guard.sh` exits 0. `atlas-rps` is present in `.github/config/services.json`, `docker-bake.hcl:88`, `deploy/k8s/base/atlas-rps.yaml`, and `deploy/shared/routes.conf:5-8` (`/api/rps` → `atlas-rps:8080`).

## Informational (non-blocking)

- `deploy/shared/routes.conf`'s new `/api/rps` block (lines 5-8) is inserted between `merchants` and `messengers`, out of strict alphabetical order per `patterns-ingress-documentation.md`'s placement guidance. However `grep -oP '(?<=\^/api/)[a-z0-9\-]+' deploy/shared/routes.conf` shows the entire file has never been alphabetically maintained (e.g. `characters` recurs a dozen times scattered throughout) — this is consistent with, not a regression from, pre-existing practice. Not flagged as a blocking finding.
- The atlas-rps local `rest` package (`services/atlas-rps/atlas.com/rps/rest/handler.go`) re-exports `atlas-rest/server` symbols via type aliases (`HandlerDependency = server.HandlerDependency`, etc.). This pattern-matches `anti-patterns.md`'s "no type aliases for library migrations" wording, but is byte-for-byte identical to the same pre-existing wrapper in ~34 other services (e.g. `atlas-chairs/atlas.com/chairs/rest/handler.go`, confirmed identical). This is an established fleet idiom RPS correctly copied, not a new violation — not flagged.

## Summary

### Blocking (must fix before merge — Important)

- **IMP-1**: `GetAllRpsRewardsHandler` (`services/atlas-tenants/atlas.com/tenants/configuration/resource.go:616-653`) must paginate — port the `paginate.ParseParams`/`paginate.Slice`/`server.MarshalPaginatedResponse` pattern already used three times in the same file (routes/vessels/instance-routes). Consider also converting `atlas-rps/configuration/processor.go:47`'s `SliceProvider` to `DrainProvider` at the same time so the fix doesn't leave a truncation trap.
- **IMP-2**: RPS reward handlers (`GetAllRpsRewardsHandler`, `GetRpsRewardByIdHandler`, `CreateRpsRewardHandler`, `UpdateRpsRewardHandler`, `DeleteRpsRewardHandler`, `SeedRpsRewardsHandler` in the same `resource.go`) must route error branches through `server.WriteErrorResponse(d.Logger())(w)(err)` instead of bare `w.WriteHeader(http.StatusInternalServerError)`; `SeedRpsRewardsHandler` must drop its custom JSON error body in favor of the same helper.

### Non-Blocking (should fix)

- **MIN-1**: Add a `processor_test.go` for `atlas-rps/saga` (mirror `atlas-rps/configuration/processor_test.go`).
- **MIN-2**: Add no-op `SetToOneReferenceID`/`SetToManyReferenceIDs` to `atlas-rps/configuration/rest.go`'s `RpsRewardRestModel` and `atlas-saga-orchestrator/rps/rest.go`'s `RestModel` (EXT-01), consistent with `cashshop`/`compartment`/`mts`/`validation` siblings in the same services.
