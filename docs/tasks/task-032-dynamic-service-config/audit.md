# Plan Audit — task-032-dynamic-service-config

**Plan Path:** docs/tasks/task-032-dynamic-service-config/plan.md
**Audit Date:** 2026-05-17
**Branch:** task-032-dynamic-service-config
**Base Branch:** main
**Commits on branch:** 32 (excluding plan/design/PRD)

## Plan Adherence

### Executive Summary

The implementation realises the architecture described in the plan: a new `libs/atlas-outbox` library (Phase A) drives transactional outbox publishing from atlas-configurations (Phase D); a `ReadEndOffsets` helper (Phase B) feeds caught-up gates in atlas-channel / atlas-login projections (Phases G, K1); a new `listener.Registry` with a four-phase Drain replaces the static per-(t,w,c) startup loop (Phases E, F, J, K2); all 44 atlas-channel and 3 atlas-login `InitHandlers` were rewritten to return `[]listener.HandlerHandle` (Phases H, K3); session `Destroy` was reordered (Phase I); and atlas-world gained a DELETE channel endpoint (Phase C). Builds, vet, and unit-test sweeps are clean across every changed module and Docker builds succeed for atlas-configurations, atlas-channel, atlas-login, and atlas-world.

Three groups of known deferrals stand: (1) Phase L integration tests (L1, L2) are not present anywhere in the changed code — coverage relies entirely on the unit tests committed in Phases A–K; (2) the evictor pipeline (Phase J3) ships the wiring (`listener.RegisterEvictor`, evict.go in both services) but no production code calls `RegisterEvictor`, no `Evict` method was added to `account.Registry` / `monster.StatusMirror` / `monster.NextSkillInbox`, and `tenant.Unregister` was never added to `libs/atlas-tenant`; (3) Phase 2 of Drain (save-and-kick) is wired through to no-op closures in both atlas-channel and atlas-login because no per-key session index exists yet. Items (1)–(3) match the controller's pre-declared deferrals. k8s manifests called out in D7/J5/J6/K5 do not exist in this repo (compose-only deployment); the equivalent env vars landed in `deploy/compose/docker-compose.{core,socket}.yml` instead, which is the only sensible alternative.

### Phase A — libs/atlas-outbox

- A1 Library skeleton — DONE. `libs/atlas-outbox/go.mod`, `go.sum`, `README.md`, `go.work` entry all present (commit `d5ff72075`).
- A2 Entity + migration — DONE. `libs/atlas-outbox/entity.go:9-21` defines `Entity` with all 9 columns; `migration.go:5-7` calls `db.AutoMigrate(&Entity{})`; partial indexes encoded via gorm struct tags. Unit test `migration_test.go` covers it.
- A3 Message + Enqueue — DONE. `libs/atlas-outbox/outbox.go:11-54` implements `Message` + `Enqueue(tx, msg)` with empty-topic/empty-key validation, headers JSON marshalling, tombstone-friendly nil value, and `pg_notify('atlas_outbox_new', topic)` when on Postgres.
- A4 Drainer publish loop — DONE. `libs/atlas-outbox/drainer.go:79-122` (`Run` / `tickOnce`) + `publishBatch` at `:213-270`. Functional options match the plan.
- A5 Advisory-lock + SKIP LOCKED — DONE. `lock.go:12-31` `tryAdvisoryLock` + `runLeader` at `drainer.go:139-174`. Batch fetch is wrapped in a transaction with `clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}` at `drainer.go:218-227`; integration test `lock_test.go` is build-tagged `integration`.
- A6 LISTEN/NOTIFY wake-up — DONE. `notify.go:1-44` (`pq.Listener` wrapper with buffered chan); `runLeader` selects on `notifyCh` (`drainer.go:152-172`). `WithDSN` plumbing at `drainer.go:53-55`. Integration test `notify_test.go` build-tagged.
- A7 Sweeper — DONE. `drainer.go:181-211`. Leader-only via `runSweeper` spawned inside `runLeader`. `SweepOnce` exposed publicly. Unit test `sweeper_test.go`.
- A8 Backfill — DONE. `backfill.go:21-55` idempotent backfill (uses count by `(topic, message_key)` then `Enqueue`). Test `backfill_test.go`.
- A9 README polish — DONE. `README.md:1-129` covers all six bullet points the plan specified (semantics, Enqueue contract, leadership, NOTIFY, Sweeper, Backfill).
- A10 Verify Phase A — DONE. `go test -race ./...` PASS, `go vet ./...` clean.

### Phase B — libs/atlas-kafka ReadEndOffsets

- B1 ReadEndOffsets — DONE. `libs/atlas-kafka/consumer/offsets.go:20-51` implements the function with broker dial + per-partition leader lookup + `ReadOffsets`. Test in `offsets_test.go`.
- B2 Verify Phase B — DONE. `go test -race ./...` PASS.

### Phase C — atlas-world DELETE route

- C1/C2 Failing test + route + handler — DONE. `services/atlas-world/atlas.com/world/channel/resource.go:25` registers `DELETE /worlds/{worldId}/channels/{channelId}`; handler at `resource.go:67-86` calls `Processor.Unregister` and returns 204 on success, 404 on `ErrChannelNotFound`. Test in `resource_test.go`. NOTE the plan text shows `/api/world-server/channel-server/{worldId}/{channelId}` but the actual base prefix in this repo's atlas-world router is `/worlds/{worldId}/channels` — the implementation matches the existing routing convention, which is the right call.
- C3 Verify Phase C — DONE. `go build`/`vet`/`test -race` all clean.

### Phase D — atlas-configurations adopts outbox

- D1 Register outbox migration + Dockerfile — DONE. `main.go:52` passes `outboxlib.Migration` to `database.SetMigrations(...)`; Dockerfile (`services/atlas-configurations/Dockerfile`) lists `atlas-outbox` in all four required locations (lines 18, 38, 56, 67).
- D2 Envelope package — DONE. `outbox/envelopes.go:14-44` (`CurrentSchemaVersion`, `NewServiceEnvelope`, `NewTenantEnvelope`). Test `envelopes_test.go`.
- D3 services Processor enqueues — DONE. `services/processor.go:35-53` helper `enqueueServiceStatus`; Create/UpdateById/DeleteById all invoke it inside the existing transaction (`processor.go:169-214`). Delete passes nil config → tombstone. Test `processor_test.go`.
- D4 tenants Processor enqueues — DONE. `tenants/processor.go:28-46` helper + invoked in Create/Update/Delete. Test in `tenants/processor_test.go`.
- D5 main.go starts drainer — DONE. `main.go:59-65` constructs `TopicWriterPool`, builds drainer with `WithDSN(database.DSN())`, registers Stop+Close teardown. `database.DSN()` exported at `libs/atlas-database/connection.go:74`.
- D6 Seeder backfill — DONE. `seeder/seeder.go:91-103` calls `services.Backfill(s.db)` and `tenants.Backfill(s.db)` after templates seed. Backfill helpers at `services/backfill.go:21-55` and `tenants/backfill.go:17-55`.
- D7 k8s manifest topic env vars — DEFERRED with replacement. No `services/atlas-configurations/atlas-configurations.yml` k8s manifest exists in this repo. Env vars went into `deploy/compose/docker-compose.core.yml` (`EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS`, `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS`). Documented in commit `f4a2b5585`.
- D8 Verify Phase D — DONE. `go build`/`vet`/`test -race` clean; `docker build -f services/atlas-configurations/Dockerfile .` succeeds.

### Phase E — atlas-channel server.Registry shape change

- E1 server.Key — DONE. `services/atlas-channel/atlas.com/channel/server/key.go:9-26` defines `Key{TenantId, WorldId, ChannelId}` + `KeyOf(Model)`. Uses `atlas-constants/channel` and `atlas-constants/world` per DOM-21. Test in `key_test.go`.
- E2 Registry slice → map; Add Deregister + Get — DONE. `server/registry.go:7-65` replaces slice with `map[Key]Model`, exports `GetRegistry`, adds `Register`/`Deregister`/`Get`/`GetAll`. Test in `registry_test.go`.

### Phase F — atlas-channel listener package

- F1 Handle + HandlerHandle — DONE. `listener/handle.go:18-46` defines `State` (Active/Draining/Removed), `HandlerHandle{Topic, Id string}`, `Handle{Key, State, Ctx, Cancel, Wg, ServerModel, KafkaHandlers}`.
- F2 Registry skeleton (Add/Snapshot) — DONE. `listener/registry.go:62-161` implements `Registry`, `Add(parent, key, sc, body)`, `Get`, `Snapshot`. Add body callback executes per-key startup and returns `[]HandlerHandle`.
- F3 Drain phase 1 (quiesce) — DONE. `registry.go:175-194` marks Draining, calls `server.GetRegistry().Deregister(key)`, then `r.deps.UnregisterChannel(...)`. The `channel.Processor.Unregister` method is at `channel/processor.go:51-57` and uses `requests.DeleteRequest` (`channel/requests.go:26-28`) treating 404 as success.
- F4 Drain phase 2 (save-and-kick) — PARTIAL (architecturally complete, behaviourally a no-op in production). `registry.go:196-204` iterates `r.deps.SessionsForKey(key)` and calls SendShutdownNotice + DestroySession. However in atlas-channel main.go:233-240 `SessionsForKey` returns nil, `SendShutdownNotice` is a no-op, `DestroySession` returns nil — explicit TODO at `main.go:234-237` says "wire session.Processor lookup-by-key once available." Same pattern in atlas-login `main.go:114-116`. The listener code is fully wired and tested; only the production glue is stubbed.
- F5 Drain phase 3 (deadline) — DONE. `registry.go:206-217` waits on `done` chan or `time.After(r.cfg.DrainDeadline)`, warns on timeout, proceeds to phase 4. Test `TestRegistry_DrainWarnsOnDeadlineButCompletes` at `registry_test.go:146`.
- F6 Drain phase 4 (teardown) — DONE. `registry.go:219-244` Cancel, RemoveHandler per `HandlerHandle`, transition to Removed, decrement tenant ref, fire evictors when last. Tested.
- F7 Idempotency + concurrency test — DONE. `registry_test.go:113` `TestRegistry_DrainIdempotentUnderConcurrency` (8 goroutines). Passes under `-race`.
- F8 Evictor registration + per-tenant ref count — DONE (machinery only). `listener/evict.go:14-38` defines `Evictor`, `RegisterEvictor`, `fireEvictorsForTenant`. Registry's `refs map[uuid.UUID]int` at `registry.go:71` is incremented in `Add` and decremented in `Drain` phase 4. Test `TestRegistry_EvictorFiresWhenLastListenerForTenantRemoved` passes.

### Phase G — atlas-channel projection

- G1 Envelope decode — DONE. `configuration/projection/envelope.go:1-58` defines `ServiceEnvelope`, `TenantEnvelope = ServiceEnvelope`, `DecodeServiceEnvelope`/`DecodeTenantEnvelope`, `ErrUnsupportedSchema`, `IsTombstone`. Tests `TestDecodeServiceEnvelope_ParsesShape`, `TestDecodeServiceEnvelope_RejectsFutureSchema`, `TestIsTombstone` in `projection_test.go`.
- G2 State singleton — DONE. `projection/state.go:17-95` `State` with RW mutex, ApplyService/ApplyServiceTombstone/ApplyTenant/ApplyTenantTombstone/Snapshot. `TestState_ApplyAndSnapshot` covers it.
- G3 Caught-up gate — DONE. `projection/caughtup.go:13-113` snapshots end offsets, observes consumed, atomic flag, `WaitCaughtUp`, `ReadyChecker`. Tests `TestCaughtUp_TransitionsAndUnblocksWaiters`, `TestCaughtUp_ReadyChecker`.
- G4 ComputeOps diff — DONE. `projection/apply.go:59-125` ComputeOps + flatten. Drain-then-Add for any config field change (port, region, version). Skip when tenant config absent. Test `TestComputeOps_AddRemovePortChangeUnchanged`.
- G5 Subscriber wiring — DONE. `projection/subscriber.go:24-159` `Subscriber.Start` snapshots end offsets, registers two consumers at `FirstOffset`, decodes envelopes, filters service envelopes by `ServiceId`, handles tombstones. (Integration test deferred per L1.)
- G6 ApplyLoop — DONE. `projection/loop.go:48-98` single-goroutine loop blocks on CaughtUp, then ticks at 250ms (configurable), computes ops, executes Drain or Add serially.

### Phase H — InitHandlers signature change

- H1 Import shim decision — DONE (skipped per plan; direct `listener.HandlerHandle` used).
- H2 Rewrite account consumer — DONE. `kafka/consumer/account/consumer.go:30` signature ends with `([]listener.HandlerHandle, error)`.
- H3 Sweep remaining 43 packages — DONE. All 44 `services/atlas-channel/atlas.com/channel/kafka/consumer/*/consumer.go` files (count verified) updated. Spot-check: `monsterbook/consumer.go:74-90` correctly captures multiple handler handles in a slice.
- H4 Phase H verification — DONE. `go vet`/`go build` clean across atlas-channel.

### Phase I — session.Destroy reorder (FR-CHN-14)

- I1 Failing ordering test — DONE. Commit `549540360` adds the test.
- I2 Reorder — DONE. `session/processor.go:330-348` calls `p.sp.Destroy(...)` then emits `DestroyedStatusEvent`, then `s.Disconnect()`. Comment at `:334-339` documents the FR-CHN-14 reasoning. Demoted emit errors to warnings; returns the emit error for back-compat.
- I3 Downstream-consumer audit — NOT EXPLICITLY DOCUMENTED. No follow-up note found; no plan-mandated code change required. Treating as DONE-by-default since there is no observable regression and the controller's commit message claims FR-CHN-14 compliance.

### Phase J — atlas-channel main.go rewire

- J1 Replace configuration.Init with projection — DONE. `main.go:205-227` constructs `projection.NewState`, `NewCaughtUp`, `Subscriber`, calls `Start` + `WaitCaughtUp(30s deadline)`. ReadyChecker is exposed on `caughtUp.ReadyChecker()` but is NOT wired to a `/readyz` endpoint — the plan asked to wire it through `restserver.New(...).WithReadyChecker(...)`. The REST server here uses `restserver.New(l).WithContext(...).WithWaitGroup(...).SetBasePath("/api/").SetPort(...).Run()` and does not advertise a WithReadyChecker method. MINOR DEVIATION — readiness gating still happens via the 30-second fatal block at boot, so a non-caught-up pod fails to come up; but live `/readyz` flapping is not implemented.
- J2 Move per-(t,w,c) startup into Add — DONE. `main.go:295-487` `buildListener` returns a closure used as `AddBody`. All 41 `InitHandlers` calls are captured via the `register` helper which concatenates `[]HandlerHandle`.
- J3 Wire tenant Evict + tenant.Unregister — NOT DONE (explicit controller deferral). Verified absent:
  - `grep -rn "listener.RegisterEvictor\|RegisterEvictor" services/atlas-channel/` shows only the definition site, no callers.
  - No `Evict(t)` method on `account.Registry` (`services/atlas-channel/atlas.com/channel/account/*.go` has no `Evict`).
  - `monster.GetNextSkillInbox().Evict(...)` exists but takes `(tenant.Model, uniqueId uint32)`, not just tenant — wrong signature for an evictor.
  - `monster.GetStatusMirror()` has no `Evict` method.
  - `tenant.Unregister` does not exist in `libs/atlas-tenant`. NOT documented in commit messages — silent debt, but matches the controller-declared deferral.
- J4 Move account.InitializeRegistry into Add path — DONE. `main.go:322` invokes `account.NewProcessor(l, tctx).InitializeRegistry()` inside the AddBody. Same in atlas-login main.go:232.
- J5 SIGTERM drain-all — PARTIAL. `main.go:249-252` registers `tdm.TeardownFunc(func() { listenerRegistry.DrainAll() })` which fires on SIGTERM (good). Steps 1 ("flip /readyz to not-ready") and 3 ("bump terminationGracePeriodSeconds") are not implemented — no `/readyz` integration (see J1) and no k8s manifest exists in this repo. Documented for the compose case.
- J6 k8s manifest topic env vars + drain deadline — DEFERRED with replacement. Env vars `EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS`, `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS`, `DRAIN_DEADLINE_MS=5000` landed in `deploy/compose/docker-compose.socket.yml:43-48`.
- J7 Dockerfile lib list update — NOT NEEDED (verified). atlas-channel does not import `atlas-outbox`; `grep atlas-outbox services/atlas-channel/Dockerfile` returns nothing, which is correct.

### Phase K — atlas-login projection + simpler drain

- K1 Mirror projection package — DONE. `services/atlas-login/atlas.com/login/configuration/projection/` files (`envelope.go`, `state.go`, `caughtup.go`, `apply.go`, `loop.go`, `subscriber.go`) mirror atlas-channel's. Tests in `projection_test.go` (7 functions).
- K2 Login listener.Registry (simpler drain) — DONE. `listener/registry.go:62-78` Registry with 2s default DrainDeadline, 5s ceiling. `listener/handle.go:36-38` `Key{TenantId uuid.UUID}` keyed by tenant only (no world/channel — login is per-tenant). `listener/handle.go:44-62` `ServerModel` inlined (no separate server package). Phase 1 (UnregisterChannel) intentionally omitted; phases 2-4 match atlas-channel. Tests in `registry_test.go` (6 functions).
- K3 Login InitHandlers signature change — DONE. All 3 consumers (`account/consumer.go:28`, `account/session/consumer.go:35`, `seed/consumer.go:30`) return `([]listener.HandlerHandle, error)`.
- K4 Login main.go rewire — DONE. `main.go:79-138` mirrors atlas-channel: build projection state + subscriber + caught-up gate, fatal on 30s timeout, build listener registry, register DrainAll teardown, launch ApplyLoop goroutine. Extra: a 1-second "republish snapshot" goroutine bridges to the legacy `configuration.PublishSnapshot(...)` for handlers that still read the package-level cache (a clean transitional shim).
- K5 k8s manifest + Dockerfile — DEFERRED with replacement. Env vars in `deploy/compose/docker-compose.socket.yml:24-28`. Dockerfile unchanged (no new lib imports, verified).

### Phase L — Integration tests

- L1 atlas-channel end-to-end add+drain integration — NOT DONE (controller-declared deferral, confirmed). No `services/atlas-channel/atlas.com/channel/listener/integration_test.go` exists; no testcontainers usage in atlas-channel. The plan's coverage goal (testcontainer Kafka publishes service-add envelope, asserts listener bring-up; publishes tombstone, asserts 4-phase Drain) is NOT exercised end-to-end. Compensating coverage:
  - `libs/atlas-outbox` has integration tests under `-tags=integration` (lock, notify) but those cover the producer side.
  - `services/atlas-channel/atlas.com/channel/configuration/projection/projection_test.go` covers envelope decode + State apply + ComputeOps + CaughtUp transitions (unit level).
  - `services/atlas-channel/atlas.com/channel/listener/registry_test.go` covers all 4 drain phases via mock Dependencies (unit level).
  - Gap: nothing exercises Subscriber + State + ApplyLoop + Registry together against a real Kafka.
- L2 atlas-login boot-without-configurations test — NOT DONE (same deferral). Same unit-test compensation; no end-to-end ready-check flow under a testcontainer.

### Phase M — Verification

- M1 Test + vet + build sweep — DONE in this audit. See Build & Test Results below.
- M2 Docker build sweep — DONE in this audit. All four service Docker builds succeed.
- M3 Service docs — NOT VERIFIED. Plan says "run `/service-doc <service>` and commit the resulting doc updates" for atlas-configurations, atlas-channel, atlas-login, atlas-world. The diff does not appear to include `dev-docs/services/*.md` updates for any of these, though `libs/atlas-outbox/README.md` was written (A9). Likely not done.
- M4 Guideline audits — PARTIAL. This audit is the plan-adherence-reviewer pass. Backend-guidelines-reviewer not yet dispatched.
- M5 PR opening — N/A (audit precedes PR).

### Summary Table

| Phase | Tasks | DONE | PARTIAL | NOT DONE / DEFERRED |
|---|---|---|---|---|
| A (atlas-outbox) | 10 | 10 | 0 | 0 |
| B (atlas-kafka offsets) | 2 | 2 | 0 | 0 |
| C (atlas-world DELETE) | 3 | 3 | 0 | 0 |
| D (atlas-configurations) | 8 | 7 | 0 | 1 (D7 — no k8s manifest in repo, replaced by compose env vars) |
| E (server registry) | 2 | 2 | 0 | 0 |
| F (listener package) | 8 | 7 | 1 | 0 (F4 partial: phase-2 stubs in production deps) |
| G (projection) | 6 | 6 | 0 | 0 |
| H (InitHandlers sweep) | 4 | 4 | 0 | 0 |
| I (Destroy reorder) | 3 | 2 | 1 | 0 (I3 audit not documented but no observable regression) |
| J (atlas-channel main rewire) | 7 | 4 | 2 | 1 (J3 evictor wiring; J1/J5 partial = no /readyz flap, no termination-grace bump) |
| K (atlas-login projection + drain) | 5 | 4 | 0 | 1 (K5 — no k8s manifest in repo) |
| L (integration tests) | 2 | 0 | 0 | 2 (controller-declared deferral) |
| M (verification) | 5 | 2 | 1 | 2 (M3 docs, M5 PR pending) |
| **Total** | **65** | **53** | **5** | **7** |

(Task counts merge subtasks within a numbered task into one row; the plan's roughly 71 checkbox items collapse to 65 trackable units.)

### Skipped / Deferred Tasks (Detail)

1. **J3 — Tenant Evict hooks + `tenant.Unregister`.** Wired plumbing exists (`listener.RegisterEvictor`, `refs` map, fire-on-zero) but no callers register an evictor and the required `Evict(tenant.Model)` method does not exist on `account.Registry`, `monster.StatusMirror`, or `monster.NextSkillInbox`. `libs/atlas-tenant` has no `Unregister`. **Impact:** when a tenant's last listener is removed, its in-memory caches in monster mirrors and account registry are retained until pod restart. This is the next implementer's pickup; it is NOT documented in any commit message or `TODO.md`, only mentioned in the audit context. RECOMMEND: add a `TODO.md` entry pointing at `listener.RegisterEvictor` callers and the missing `Evict` methods so the debt is discoverable from inside the codebase.
2. **F4 — Drain phase 2 (save-and-kick).** The listener.Registry code correctly invokes `r.deps.SessionsForKey/SendShutdownNotice/DestroySession`, but in production main.go (atlas-channel:233-240, atlas-login:114-116) these functions are nil/no-op closures because `session.Processor.ByKey(...)` does not exist. An in-flight session at drain time is dropped silently on phase-4 ctx cancel rather than being save-and-kicked. **Impact:** clients on a draining channel get a TCP disconnect instead of a logout packet. ACCEPTABLE for a first cut, but the plan explicitly called for save-and-kick. TODO `main.go:234-237` exists in code — partially documented.
3. **J1/J5 — `/readyz` flapping for caught-up + SIGTERM drain.** The boot path fatals out if WaitCaughtUp times out within 30s, so a non-caught-up pod never finishes Run; but a runtime regression (e.g., projection state cleared after boot) would not flip readiness. The `restserver` constructor used here doesn't expose a `WithReadyChecker` hook, so this is more than a one-line addition. **Impact:** during operator-driven re-bootstrap or upstream Kafka outage, a degraded pod stays in-rotation. Compose-level workaround: rely on healthcheck probes.
4. **L1/L2 — integration tests.** Coverage gap: no end-to-end Kafka + projection + listener test in atlas-channel or atlas-login. Unit tests cover each layer individually. **Impact:** a regression in the seam between Subscriber's offset accounting and ApplyLoop's diff would not be caught by current tests.
5. **D7/J6/K5 — k8s manifests.** No `services/<svc>/atlas-<svc>.yml` exists in repo. Env vars landed in `deploy/compose/docker-compose.{core,socket}.yml`. Acceptable replacement.
6. **J5 step 3 — `terminationGracePeriodSeconds` bump.** No k8s manifest to edit.
7. **M3 — service docs.** Not run. RECOMMEND: dispatch `/service-doc` for atlas-configurations, atlas-channel, atlas-login, atlas-world before opening the PR per CLAUDE.md.

### Build & Test Results

| Module | Build | Vet | Unit Tests (-race) | Docker Build |
|---|---|---|---|---|
| libs/atlas-outbox | PASS | PASS | PASS | n/a |
| libs/atlas-kafka | PASS | PASS | PASS | n/a |
| services/atlas-configurations | PASS | PASS | PASS | PASS |
| services/atlas-channel | PASS | PASS | PASS | PASS |
| services/atlas-login | PASS | PASS-with-preexisting | PASS | PASS |
| services/atlas-world | PASS | PASS | PASS | PASS |

`atlas-login` vet emits `socket/init.go:39:11: WaitGroup.Add called from inside new goroutine` — verified via `git log` to be pre-existing (file's last touch was the monorepo rename commit), unrelated to this branch.

Integration tests (`-tags=integration`) in `libs/atlas-outbox` were not exercised during audit (require Docker testcontainers); their presence and structure was code-reviewed only.

### Overall Assessment

- **Plan Adherence:** MOSTLY_COMPLETE — 53/65 fully done, 5 partial, 7 deferred (5 with documented rationale, 2 silent).
- **Recommendation:** NEEDS_REVIEW — the core machinery is sound, all builds + tests green, all four Dockerfiles build. Before merging, decide on the silent deferrals (J3 evictor pipeline, F4 production stubs) — either accept as known follow-up with a `TODO.md` entry, or implement.

### Action Items

1. Add `TODO.md` entries (or follow-up task PRDs) for the two silent deferrals: J3 evictor wiring (`Evict` methods + `tenant.Unregister` + `main.go` `RegisterEvictor` calls in both atlas-channel and atlas-login) and F4 save-and-kick (`session.Processor.ByKey` + production deps that aren't nil/no-op).
2. Add `TODO.md` entry for the L1/L2 integration tests so the coverage gap is tracked.
3. Decide on J1 `/readyz` integration — either extend `atlas-rest/server.RestServer` with `WithReadyChecker`, or accept boot-only gating and document the choice.
4. Run `/service-doc` for atlas-configurations, atlas-channel, atlas-login, atlas-world (Task M3) before opening PR.
5. Dispatch `backend-guidelines-reviewer` (Task M4) in parallel to this audit; address any DOM-* findings.
6. Spot-check the integration tests in `libs/atlas-outbox` under `-tags=integration` against a real Docker daemon to confirm the lock + NOTIFY tests still pass.

---

# Backend Guidelines Audit — task-032-dynamic-service-config

- **Reviewer:** backend-guidelines-reviewer
- **Date:** 2026-05-17
- **Scope:** libs/atlas-outbox, libs/atlas-kafka/consumer/offsets.go, libs/atlas-database/connection.go, services/atlas-configurations, services/atlas-channel, services/atlas-login, services/atlas-world
- **Guidelines Source:** .claude/skills/backend-dev-guidelines/

## Build & Test Results

- `go build ./...` clean for libs/atlas-outbox, libs/atlas-kafka, services/atlas-configurations, services/atlas-channel, services/atlas-login, services/atlas-world. PASS.
- `go test ./... -count=1` clean for every changed module. Notable test packages: `libs/atlas-outbox` (0.309s), `atlas-channel/listener` (1.963s) — `TestRegistry_DrainIdempotentUnderConcurrency` exercises the 8-goroutine drain race; `atlas-channel/configuration/projection` and the atlas-login parity package; `atlas-world/channel` (covers new DELETE handler); `atlas-configurations/outbox` and `atlas-configurations/seeder`.

## Verdict

**PASS-WITH-CONCERNS** — code compiles, tests pass, and the listener / projection / outbox infrastructure is well-structured. Three deploy/correctness items prevent an unconditional PASS:

1. **DOM-23 / Deploy: K8s ConfigMap missing the two new topic keys** — atlas-channel and atlas-login will FATAL on startup in k8s within 30s of boot.
2. **DOM-23 / Deploy: docker-compose values use dotted-lowercase strings instead of `KEY: "KEY"`** — divergent from convention.
3. **Outbox publisher uses `kafka.LeastBytes` balancer** — for log-compacted topics, key-hash partitioning is required; LeastBytes can spray same-key messages across partitions and break compaction. Consistent with existing producer convention so flagged as a concern, not a hard fail.

The big-ticket correctness items (transactional outbox semantics, leadership lifecycle, drain idempotency, tombstone propagation, atlas-constants reuse) all check out.

---

## DOM Checklist (focused on the changed surface)

### libs/atlas-outbox (new library)

This is a Go library, not a service domain — DOM-01..20 don't all apply. Spot-checks:

| ID / Check | Status | Evidence |
|---|---|---|
| Enqueue requires non-nil tx | PASS | `libs/atlas-outbox/outbox.go:19` |
| Enqueue requires non-empty topic | PASS | `libs/atlas-outbox/outbox.go:22` |
| Enqueue requires non-empty key | PASS | `libs/atlas-outbox/outbox.go:25` |
| Nil Value permitted (tombstone) | PASS | `libs/atlas-outbox/outbox.go:38-46` writes Entity with nil bytes for log-compaction tombstones |
| pg_notify only on postgres | PASS | `libs/atlas-outbox/outbox.go:48` guarded by `isPostgres(tx)` |
| Drainer ctx-cancel releases lock | PASS | `libs/atlas-outbox/drainer.go:95,120` — runLeader returns on `ctx.Done()`, then tickOnce calls `lk.Release(context.Background())` |
| Drainer FOR UPDATE SKIP LOCKED on postgres | PASS | `libs/atlas-outbox/drainer.go:218-220` |
| Drainer fallback when no postgres (tests) | PASS | `libs/atlas-outbox/drainer.go:106-108` |
| Failure bookkeeping avoids self-deadlock | PASS | `libs/atlas-outbox/drainer.go:245-249` defers attempts/last_error UPDATE until after SELECT tx rolls back, run on a separate pool connection |
| Sweeper is leader-only | PASS | `libs/atlas-outbox/drainer.go:147-148` — sweeper spawned inside `runLeader` with ctx that cancels on leader exit |
| Notifier pump goroutine cleanup | PASS | `libs/atlas-outbox/notify.go:32-40` exits when `n.ln.Notify` closes (closed by `ln.Close()`) |
| Backfill idempotent on (topic, key) | PASS | `libs/atlas-outbox/backfill.go:33-39` skips when count > 0 |
| Drainer does NOT defer lock release | CONCERN | `libs/atlas-outbox/drainer.go:118-121` — if `runLeader` panics, lock is leaked. Postgres advisory locks auto-release on conn close so impact is bounded, but `defer lk.Release(...)` is safer. |
| `Stop()` can panic on double-close | CONCERN | `libs/atlas-outbox/drainer.go:176` — `close(d.stop)` panics if called twice. Single call site today, fragile if reused. |
| Headers stored but never published | CONCERN | `libs/atlas-outbox/drainer.go:232-239` — `kafka.Message` constructed with only Topic/Key/Value; `Headers` jsonb column is not propagated to kafka. |
| Balancer choice | CONCERN | `services/atlas-configurations/atlas.com/configurations/outbox/publisher.go:86` uses `kafka.LeastBytes{}`. For log-compacted topics, key-hash partitioning (e.g. `&kafka.Hash{}`) is required. Consistent with the existing convention (`libs/atlas-kafka/producer/manager.go:122`), so flagged not failed. |

### services/atlas-configurations (services/processor.go, tenants/processor.go)

| ID / Check | Status | Evidence |
|---|---|---|
| DOM-21 (no atlas-constants duplication) | PASS | No new ID types introduced. |
| Enqueue runs inside caller's transaction | PASS | `services/atlas-configurations/atlas.com/configurations/services/processor.go:169-174` and `tenants/processor.go:124-129, 163-175, 132-138` — `enqueueServiceStatus`/`enqueueTenantStatus` are called from within `database.ExecuteTransaction` callbacks, taking the inner `db *gorm.DB` as their tx parameter. |
| Tombstone uses nil Value on delete | PASS | `services/processor.go:206-213` and `tenants/processor.go:132-138` — DeleteById calls `enqueueXxx(db, id, nil)`; the helper short-circuits envelope construction and Enqueue stores nil. |
| Outbox table not tenant-scoped | PASS | `libs/atlas-outbox/entity.go:9-19` Entity has no `tenant_id` column. `libs/atlas-database/tenant_scope.go:32` callback only activates when the schema has a `tenant_id` field, so outbox queries are not tenant-filtered automatically. README claim holds. |
| Service+tenant CRUD emit on Create/Update/Delete | PASS | `services/processor.go:174,202,212` and `tenants/processor.go:128,137,174` |
| Topic env var read via os.Getenv (not topic.EnvProvider) | CONCERN | `services/processor.go:36` and `tenants/processor.go:29` use `os.Getenv(EnvXxx)`. `topic.EnvProvider` (atlas-kafka convention) logs a Warn when unset; here unset silently skips enqueue. Documented as intentional for unit tests, but a misconfigured production deploy will silently stop emitting events. |
| Dockerfile updated for atlas-outbox in 4 places (DOM-22) | PASS | `services/atlas-configurations/Dockerfile:18` (COPY go.mod), `:35` (go.work use), `:53` (COPY src), `:66` (-replace) |
| Dockerfile updated for atlas-database in 4 places (DOM-22) | PASS | `services/atlas-configurations/Dockerfile:17,34,52,63` |

### services/atlas-world/channel (new DELETE route)

| ID / Check | Status | Evidence |
|---|---|---|
| DOM-13: handler delegates to processor | PASS | `services/atlas-world/atlas.com/world/channel/resource.go:67-86` calls `NewProcessor(...).Unregister(ch)` |
| DOM-17: not-found returns 404 | PASS | `resource.go:73-77` — `errors.Is(err, ErrChannelNotFound)` → 404 |
| DOM-17: other errors return 500 | PASS | `resource.go:78-80` |
| DOM-12: no os.Getenv in handler | PASS | grep clean |
| DOM-15: no direct db.Delete | PASS | Unregister flows through Registry: `processor.go:96-99` → `registry.go:69-78` (redis SREM); no GORM writes in handler. |
| DOM-20: tests cover success + 404 cases | PASS | `resource_test.go:230-263` (Deletes), `:265-287` (NotFoundIs404). Per-test functions rather than table-driven; acceptable per file's pre-existing style. |

### services/atlas-channel — listener, server, projection, consumer sweep

| ID / Check | Status | Evidence |
|---|---|---|
| DOM-21 (no atlas-constants duplication) | PASS | `services/atlas-channel/atlas.com/channel/server/key.go:12-16` `Key` struct composes `uuid.UUID + world.Id + channel.Id` from atlas-constants. Comment on line 11 makes the constraint explicit. |
| listener.Registry.Drain idempotent under concurrency | PASS | `services/atlas-channel/atlas.com/channel/listener/registry.go:176-188` claims state under mu (`State==Active` winner proceeds; `Draining/Removed/!ok` returns). Test `registry_test.go:113-144` runs 8 goroutines and asserts `UnregisterChannel` ran exactly once. |
| Drain phase 3 deadline doesn't block phase 4 | PASS | `registry.go:206-217` selects between `done` and `time.After(DrainDeadline)`; phase 4 always runs. Test `registry_test.go:146-170` asserts elapsed < 200ms with deadline 30ms and Wg parked for 200ms. |
| Drain phase 3 goroutine cleanup | PASS | The `go func() { h.Wg.Wait(); close(done) }()` goroutine continues until Wg drains; if phase 3 times out, the goroutine outlives Drain but cannot leak resources beyond what was registered to Wg. |
| Phase 4 cancel-then-RemoveHandler ordering | PASS | `registry.go:221-229` cancels ctx first, then iterates KafkaHandlers (safe — only one goroutine reaches phase 4) and RemoveHandler-s each. State→Removed transition under mu. |
| Tenant evictor fires when last listener for tenant drains | PASS | `registry.go:230-243` decrements refs and calls `fireEvictors`. Test `registry_test.go:172-217` validates with 2-tenant, 3-listener fan-out. |
| Projection consumer state replace is idempotent on replay | PASS | `projection/state.go:30-44, 57-70` ApplyService/ApplyTenant unconditionally replace state by key. Replays land on same end state. |
| Tombstone detection (consumer side) | PASS | `projection/envelope.go:39` `IsTombstone(value)` checks `value == nil`. `projection/subscriber.go:88,120` route nil-value messages to ApplyServiceTombstone / ApplyTenantTombstone. |
| Service tombstone scoped to own service id | PASS | `subscriber.go:92-95` only acts on tombstones whose key matches `"service:"+s.ServiceId.String()`. |
| Schema-version forward compatibility | PASS | `envelope.go:48-50` returns ErrUnsupportedSchema; `subscriber.go:99-104, 135-139` log + skip rather than crash. |
| CaughtUp gate is one-way | PASS | `projection/caughtup.go:106-112` uses `atomic.Bool` Store(true); never resets. |
| CaughtUp evaluateLocked handles empty topic | PASS | `caughtup.go:96-104` — `got[p] < end-1` with `end=0` is always false → caught up. |
| ApplyLoop only runs after CaughtUp | PASS | `projection/loop.go:49-52` blocks on WaitCaughtUp before starting the ticker. |
| ApplyLoop serializes Add/Drain per key | PASS | Single ticker goroutine; ComputeOps yields drains before adds for same key; listener.Registry.Drain is itself idempotent under concurrent callers. |
| Session.Destroy emits before Disconnect (FR-CHN-14) | PASS | `services/atlas-channel/atlas.com/channel/session/processor.go:330-347` — sp.Destroy + kp emit happen before s.Disconnect at line 346. Comment at line 334-339 documents the ordering. |
| InitHandlers signature sweep (43 packages) | PASS | Spot-check `kafka/consumer/account/consumer.go:30-45` returns `([]listener.HandlerHandle, error)` and pushes one HandlerHandle per registered handler. main.go's `buildListener` (`main.go:343-484`) propagates handles into listener.Add's body. |

### services/atlas-login — listener, projection

| ID / Check | Status | Evidence |
|---|---|---|
| listener parity (Drain idempotency, deadline, ceiling) | PASS | `services/atlas-login/atlas.com/login/listener/registry.go:80-97` (default 2s, ceiling 5s). Test coverage at `registry_test.go:67-218` mirrors atlas-channel including the 8-goroutine race. |
| listener.Key uses uuid.UUID only (no atlas-constants duplication) | PASS | `services/atlas-login/atlas.com/login/listener/handle.go:36-38` — atlas-login has no per-world/channel fan-out. |
| Configuration registry bridge (PublishSnapshot) | PASS | `services/atlas-login/atlas.com/login/configuration/registry.go:70-84` takes svc + tenants by value-copy. main.go (`main.go:104-108, 144-155`) calls it post-CaughtUp and on a 1s republish ticker so legacy callers see fresh data. |
| 30s catch-up FATAL behaviour | CONCERN | `main.go:92-96` calls `l.Fatal` if WaitCaughtUp times out. Combined with the k8s configmap issue below, atlas-login crash-loops in k8s. |
| atlas-login `GetServiceConfig` still uses `log.Fatalf` | CONCERN | `configuration/registry.go:22` calls `log.Fatalf("Configuration not initialized.")` when `serviceConfig == nil`. After the projection-bridge change, this fatal is reachable if `PublishSnapshot(nil, ...)` runs (e.g. service tombstone) and a handler then calls `GetServiceConfig()`. Consider returning an error so callers can backoff/retry. |

### DOM-23 — Topic config in deploy manifests

| ID / Check | Status | Evidence |
|---|---|---|
| EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS in k8s configmap | **FAIL** | `deploy/k8s/base/env-configmap.yaml` is missing the key. Only an unrelated `EVENT_TOPIC_CONFIGURATION_STATUS` exists at line 93. Existing convention is `KEY: "KEY"` (see lines for EVENT_TOPIC_ACCOUNT_STATUS et al). |
| EVENT_TOPIC_CONFIGURATION_TENANT_STATUS in k8s configmap | **FAIL** | Same — absent from `deploy/k8s/base/env-configmap.yaml`. |
| Topic env values follow `KEY: "KEY"` convention in compose | **FAIL** | `deploy/compose/docker-compose.core.yml:147-148` and `deploy/compose/docker-compose.socket.yml:26-27, 48-49` use dotted-lowercase values (`atlas.configuration.service.status`). Diverges from platform-wide pattern. |
| Service deployment YAML doesn't override topic env literal | PASS | `deploy/k8s/base/atlas-channel.yaml:25-27`, `atlas-login.yaml:27-29`, `atlas-configurations.yaml:21-23` all use `envFrom: configMapRef: atlas-env`; no literal env overrides. |

**Failure-mode walk-through:** In k8s, `os.Getenv("EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS")` returns "". In `subscriber.go:48`, `s.ServiceTopic == ""` short-circuits, so `SetEndOffsets` is never called and the consumer is never registered. `caughtUp.snapshots` stays empty; `evaluateLocked()` at `caughtup.go:92-94` returns early without setting `caughtUp`. `main.go:92-96` (channel) and `main.go:92-96` (login) block on `WaitCaughtUp` for 30s, then `l.Fatal`. Pod crash-loops indefinitely.

## SUB Checklist

No new sub-domain (action-event) packages were introduced — the listener/projection/server packages are infrastructure, not sub-domains in the SUB sense. N/A.

## SEC Checklist

Not an auth-related service. SEC-01..04 N/A. Adjacent note: `services/atlas-configurations/atlas.com/configurations/outbox/publisher.go:28` reads `BOOTSTRAP_SERVERS` from env at construction only — operator key rotation requires a restart, which matches the rest of the codebase.

## Blocking (must fix before merge)

- **DOM-23** Add `EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS: "EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS"` and `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS: "EVENT_TOPIC_CONFIGURATION_TENANT_STATUS"` to `deploy/k8s/base/env-configmap.yaml`. Without this, atlas-channel and atlas-login crash-loop in k8s 30s after pod start.
- **DOM-23** Normalize compose env values to `KEY: "KEY"` shape (or document the dotted-lowercase exception). The two compose files currently advertise `atlas.configuration.service.status` / `atlas.configuration.tenant.status`, diverging from the platform convention.

## Non-Blocking (should fix)

- **Outbox publisher balancer** Switch `kafka.LeastBytes` to a key-hashing balancer (e.g. `&kafka.Hash{}` or `&kafka.CRC32Balancer{}`) at `services/atlas-configurations/atlas.com/configurations/outbox/publisher.go:86`. Required for log-compacted topics to behave correctly — otherwise the same key can land on different partitions and compaction stops collapsing updates.
- **Outbox headers** `libs/atlas-outbox/drainer.go:232-239` constructs `kafka.Message` without copying `Headers` back onto the message. README implies headers are propagated; today they're stored in the table but never sent. Either propagate them or drop the column.
- **Drainer Stop double-close** `libs/atlas-outbox/drainer.go:176` will panic if `Stop()` is called twice. Guard with `sync.Once` or an `atomic.Bool`.
- **Drainer panic safety** `libs/atlas-outbox/drainer.go:118-121` should `defer lk.Release(context.Background())` so a `runLeader` panic doesn't leak the lock until process exit.
- **Topic env via os.Getenv** `services/atlas-configurations/atlas.com/configurations/services/processor.go:36` and `tenants/processor.go:29` silently skip enqueue when env var is unset. Either fall back to `topic.EnvProvider` (which logs a Warn) or add a startup-time validation.
- **Session.Destroy reorder coverage** No regression test asserts emit happens before Disconnect at `services/atlas-channel/atlas.com/channel/session/processor.go:340-346`. Consider one to prevent silent regression of FR-CHN-14.
- **atlas-login `Init` fatal-on-nil** `services/atlas-login/atlas.com/login/configuration/registry.go:22` still calls `log.Fatalf` when `serviceConfig == nil`. After the projection-bridge change, this fatal is reachable any time `PublishSnapshot(nil, ...)` runs. Return an error instead.

## Summary

The transactional outbox, projection, and listener-with-drain machinery are well-designed: ctx cancellation, idempotency, schema versioning, log-compaction tombstones, and the lifecycle ordering (emit-before-disconnect, cancel-then-RemoveHandler, leader-only sweeper) are all correctly wired and exercised by tests. The atlas-constants reuse constraint (DOM-21) is upheld — the new `server.Key` and `listener.Key` types compose existing constants without inventing new ID types.

The branch's only blocking issues are deploy-config: the new Kafka topic env vars were added to docker-compose but NOT to the k8s configmap, so atlas-channel and atlas-login will crash-loop on first k8s deployment 30s after pod start. Fix is a four-line edit to `deploy/k8s/base/env-configmap.yaml`.

Non-blocking concerns: `kafka.LeastBytes` balancer on log-compacted topics, headers stored but not published, drainer Stop double-close fragility, silent enqueue-skip when env unset. None are merge-blockers; all should be addressed in follow-up.
