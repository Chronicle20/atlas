# Backend Audit — atlas-rps (task-132 RPS NPC minigame)

- **Primary target:** `services/atlas-rps/atlas.com/rps/` (new service)
- **Secondary:** libs/atlas-saga, libs/atlas-packet/rps, atlas-channel rps, atlas-saga-orchestrator rps, atlas-tenants rps-rewards, atlas-npc-conversations rps
- **Guidelines Source:** backend-dev-guidelines skill (DOM-*/SUB-*/SEC-*)
- **Date:** 2026-07-08
- **Build/Tests:** delegated to the background verification-gate agent (not re-run here)
- **Overall:** NEEDS-WORK — one blocking deploy-config defect (DOM-23); all code-level checks pass.

## Service shape note

atlas-rps is a **Redis-backed, in-memory session service**, not a GORM/DB domain.
There is no `entity.go`, `administrator.go`, or DB `provider.go`; session state lives
in a `libs/atlas-redis` TTLRegistry singleton (`game/registry.go`). DOM checks that
presuppose a GORM entity/administrator/provider layer (DOM-02/03/10/11-DB/16) are
therefore N/A-by-design, not failures. State writes are funneled processor → registry,
which satisfies the intent of the layering checks.

## atlas-rps DOM Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go exists | PASS | `game/builder.go:28` NewModelBuilder, fluent setters, `Build()` validates tenant+characterId (builder.go:106-126) |
| DOM-02 | ToEntity() | N/A | No GORM entity; model persists via `MarshalJSON`/`UnmarshalJSON` (model.go:45,71) into Redis |
| DOM-03 | Make(Entity) | N/A | No entity layer |
| DOM-04 | Transform | PASS | `game/rest.go:60` (+ TransformWithPrize rest.go:76) |
| DOM-05 | TransformSlice | N/A | Service has no list endpoint (GET returns a single session; POST returns one). No inline transform loops in resource.go |
| DOM-06 | Processor takes FieldLogger | PASS | `game/processor.go:137,161`; `configuration/processor.go:32`; `saga/processor.go:32` all `logrus.FieldLogger` |
| DOM-07 | Handlers pass d.Logger() | PASS | `game/resource.go:43,71` `newProcessor(d.Logger(), d.Context())`; no `logrus.StandardLogger()` |
| DOM-08 | POST uses RegisterInputHandler | PASS | `game/resource.go:33` `registerInput("create_rps_game", …)`; no PATCH surface |
| DOM-09 | Transform errors handled | PASS | `game/resource.go:50-55,82-87` check err (no `_, _ :=`) |
| DOM-10 | Test DB tenant callbacks | N/A | No GORM; tests use miniredis (`game/registry_test.go:19`) |
| DOM-11 | Providers lazy | PASS | `configuration/processor.go:45` `requests.SliceProvider[...]` (lazy); ladder injected via `LadderProvider` closure |
| DOM-12 | No os.Getenv in handlers | PASS | `resource.go` has none; os.Getenv only in `main.go:71` and `kafka/consumer/consumer.go:23` |
| DOM-13 | No cross-domain logic in handlers | PASS | Handlers call only the game processor |
| DOM-14 | Handlers don't call providers directly | PASS | `resource.go` calls processor methods only |
| DOM-15 | No direct entity writes in handlers | PASS | No `db.Create/Save/Delete`; writes go processor → registry |
| DOM-16 | Write path centralized | PASS (analog) | Registry mutation (`registry.go` Put/Remove) only invoked from processor.go |
| DOM-17 | Domain error → HTTP status | PASS | `resource.go:73-79` ErrSessionNotFound→404, else→500; input parse via `rest.ParseCharacterId`. (No 400/409 surface — invalid-status paths are Kafka-driven, not REST) |
| DOM-18 | JSON:API interface on REST model | PASS | `rest.go:34-56` GetName/GetID/SetID |
| DOM-19 | Flat request model | PASS | `rest.go:23-32` flat, `Id` tagged `json:"-"` |
| DOM-20 | Table-driven tests | PASS (partial) | `game/adjudicate_test.go`, `game/ladder_test.go` table-driven; some registry/processor tests are per-case funcs (acceptable, non-blocking) |
| DOM-21 | No reinvented shared types | PASS (minor note) | `world.Id`/`channel.Id`/`item.Id` used correctly (model.go, rest.go, ladder.go). `game.Throw/Status/Outcome` are legitimate new domain enums. **Minor:** `characterId`/`npcId` are bare `uint32` though `character.Id` exists (`libs/atlas-constants/character/constants.go:3`); peer services (npc-conversations model.go:2223) also use bare uint32, so this is a consistency nit, not a redeclaration violation |
| DOM-22 | Deploy scaffolding (shared-Dockerfile model) | PASS | services.json:424, docker-bake.hcl:87, go.work:75, base/kustomization.yaml:56, image entries in both overlays. No new shared lib introduced |
| DOM-23 | Kafka topic naming / configmap | **FAIL (BLOCKING)** | `COMMAND_TOPIC_RPS` and `EVENT_TOPIC_RPS` (message/rps/kafka.go:10-11) are **absent** from `deploy/k8s/base/env-configmap.yaml`. See detail below |
| DOM-24 | Kafka producer stubbed in emit tests | PASS | `game/testmain_test.go:15` and `kafka/consumer/rps/testmain_test.go:14` call `producertest.InstallNoop()`; capturing managers are layered on top (task_test.go:78, consumer_test.go:79) with NO cleanup reverting to the unstubbed default (explicitly documented consumer_test.go:73-75) |

## SUB / cross-cutting

| Area | Status | Evidence |
|------|--------|----------|
| Command consumer (action-event) | PASS | `kafka/consumer/rps/consumer.go` typed `message.AdaptHandler` per Command Type; curried `InitConsumers`/`InitHandlers`; header parsers Span+Tenant (consumer.go:28) |
| No manual JSON parsing | PASS | No `json.NewDecoder`/`io.ReadAll` in consumers or resource; envelope handled by framework |
| Processor Interface+Impl + buffered/AndEmit split | PASS | `processor.go:78-119` interface; each `Method(mb,…)` pure + `MethodAndEmit` wraps `message.EmitWithResult` (e.g. Start/StartAndEmit processor.go:196,239) |
| tenant.MustFromContext | PASS | processor.go:138,162; registry.go:44,51,61; task.go:46 |
| Redis discipline (lib types only) | PASS | registry.go uses `atlas.NewTTLRegistry`/`atlas.NewSet` (registry.go:30,33); no raw keyed go-redis calls |
| Money path — Collect payout | PASS | processor.go:398-437: on `sagaSubmitter` failure returns err WITHOUT removing session or buffering GameEnded (retry-safe); no swallowed error |
| Saga submission path | PASS | `saga/processor.go:40` returns producer error; caller (processor.Collect) propagates |
| Concurrency (concurrent Kafka goroutines) | PASS | `DefaultThrowSource` uses auto-seeded, concurrent-safe global `math/rand` (adjudicate.go:45); `beats` map read-only; `newProcessor` package var read-only; per-character state in Redis, no shared in-proc mutable state |
| No stubs/TODO/501 in landed code | PASS (1 note) | grep clean. Channel RETRY sub-op is a documented no-op (rps_action.go:102-105), a genuinely separable "restart-with-fee" follow-up, logged at debug — not a silent stub |

## Security (SEC-*)

Not an auth/token service — SEC-01..03 N/A. SEC-04: no hardcoded secrets in atlas-rps.
Money path reviewed under "Collect payout" above — payout only on `StatusAwaitingDecision`,
saga failure is retry-safe, sweep path never pays out (task.go:41-51).

## Peer-service spot checks (all PASS)

- **libs/atlas-saga**: `StartRPSGame` action + `StartRPSGamePayload` + unmarshal case + test added consistently (model.go, payloads.go:680, unmarshal.go, unmarshal_test.go).
- **atlas-saga-orchestrator/rps**: `handleStartRPSGame` mirrors existing synchronous-REST handlers (handler.go:2896+); acceptance table entry added; `requestStartGame` uses `requests.RootUrl("RPS_URL")` + `PostRequest[RestModel]` (requests.go); REST model flat + JSON:API iface (rest.go).
- **atlas-channel/rps**: consumer translates events→frames idiomatically; `sc.Is(tenant,...)` gating; int8 `straightVictoryCount` clamp guards overflow (consumer.go:138); handler uses `operations`-table sub-op resolution (rps_action.go:114, mirrors storage).
- **atlas-tenants rps-rewards**: POST/PATCH via `RegisterInputHandler[RpsRewardRestModel]` (resource.go:853,883-884), GET/DELETE via `RegisterHandler`; Create/Update `…AndEmit`.
- **atlas-npc-conversations**: RPSActionType wired into validator (validator.go:101,302), reachability (l.527) and cycle detection (l.681) — consistent with sibling action types.

---

## Blocking (must fix)

### DOM-23 — RPS Kafka topics missing from `deploy/k8s/base/env-configmap.yaml`

**Evidence:**
- Topics used: `COMMAND_TOPIC_RPS`, `EVENT_TOPIC_RPS`
  (`services/atlas-rps/atlas.com/rps/kafka/message/rps/kafka.go:10-11`; same constants
  in `services/atlas-channel/atlas.com/channel/kafka/message/rps/kafka.go:16-17`).
- `deploy/k8s/base/env-configmap.yaml` contains 141 `COMMAND_TOPIC_*`/`EVENT_TOPIC_*`
  entries (shape `KEY: "KEY"`) but **neither RPS key** (grep for `RPS` → 0 hits).
  The file was NOT modified on this branch (`git diff` empty for it).

**Why it breaks the deploy (not caught by `go build`/`go test`):**
`deploy/k8s/base/env-configmap.yaml` is the single source of truth for:
1. **Topic precreation** — `atlas-kafka-precreate.yaml:42` iterates env vars matching
   `^(COMMAND|EVENT)_TOPIC_` and precreates each. RPS topics won't be precreated.
2. **Per-environment topic isolation** — `deploy/k8s/overlays/pr/scripts/gen-topic-config.sh`
   emits a `-PLACEHOLDER_ATLAS_ENV` suffixed literal for every topic key in that file.
   Because the RPS keys are absent, `topic.EnvProvider` (libs/atlas-kafka/topic/provider.go:16-19)
   falls back to the **bare token string** `"COMMAND_TOPIC_RPS"`/`"EVENT_TOPIC_RPS"` at
   runtime — unsuffixed. Every ephemeral/PR environment and main would then share one
   un-isolated RPS topic on the shared broker (cross-environment message bleed), while
   every other topic is per-env suffixed. This is exactly the "hardcoded topics break
   environment portability" anti-pattern.

**Fix (trivial):** add to `deploy/k8s/base/env-configmap.yaml`:
```yaml
  COMMAND_TOPIC_RPS: "COMMAND_TOPIC_RPS"
  EVENT_TOPIC_RPS: "EVENT_TOPIC_RPS"
```

## Non-Blocking (should fix / note)

- **DOM-21 (minor):** `characterId`/`npcId` fields use bare `uint32` where
  `character.Id` (`libs/atlas-constants/character/constants.go:3`) exists. Consistent
  with peer services that also use `uint32`, so not a redeclaration violation — optional
  consistency improvement.
- **Channel RETRY sub-op** (`rps_action.go:102-105`) is a documented no-op (restart-with-fee
  parked as a follow-up). Acceptable as a design decision, but confirm it is tracked so the
  client's RETRY button isn't silently dead in the deployed feature.
- **DOM-20 (minor):** several `game` tests are per-case funcs rather than table-driven
  (e.g. registry_test.go). Core rule logic (adjudicate/ladder) is table-driven.
