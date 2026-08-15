# Plan Adherence Audit — task-231-generalized-events-service

**Plan Path:** docs/tasks/task-231-generalized-events-service/plan.md
**Audit Date:** 2026-08-15
**Branch:** task-231-generalized-events-service
**Base Branch:** main (merge-base bb2ac767a)
**Method:** Parallel fresh-context sub-audits (Tasks 1-10, 9-20, 21-30, 31-40) each
independently re-verified plan requirements against current code and cross-checked
the SDD execution ledger (`.superpowers/sdd/plan/progress.md`), flagging anything not
already recorded there. This report consolidates their findings. Tasks 9 and 10 were
covered by both the 1-10 and 9-20 sub-audits; both agreed (IMPLEMENTED, no conflicts).

## Task Completion Summary

| # | Task | Status | Notes |
|---|------|--------|-------|
| 1 | Provenance on monster model + Redis form | IMPLEMENTED | |
| 2 | SPAWN_FIELD provenance + boundary normalization | IMPLEMENTED | |
| 3 | Echo provenance on every monster status event | IMPLEMENTED | fix round restored 8 destroyed pre-existing tests |
| 4 | DESTROY_BY_SOURCE field command | IMPLEMENTED | |
| 5 | Derive voyage id; widen Transition | IMPLEMENTED | fixed a real pre-existing midnight-crossing bug |
| 6 | Emit VOYAGE_DEPARTED / VOYAGE_ARRIVED per channel | IMPLEMENTED | clock seam added after fix round |
| 7 | Expose voyageId on route REST resource | IMPLEMENTED | |
| 8 | Map Anniversary stats to exp/item_drop | IMPLEMENTED | |
| 9 | Bootstrap module + build registration | IMPLEMENTED | main.go rt.Wait() confirmed |
| 10 | K8s/ingress/DB registration | IMPLEMENTED | seed-catalog label present |
| 11 | Event handler registry | IMPLEMENTED | interface has drifted since (see below) |
| 12 | event/definition persistence | IMPLEMENTED | setEnabled fix verified in code |
| 13 | event/definition processor/REST/seeding | IMPLEMENTED | seed path relocation justified |
| 14 | event/transition occurrence history | IMPLEMENTED | |
| 15 | event/occurrence persistence + paired write | IMPLEMENTED | guarded completion verified |
| 16 | event/occurrence REST surface | IMPLEMENTED | |
| 17 | event/scheduling persistence + dedup | IMPLEMENTED | |
| 18 | Claim-locked poller | IMPLEMENTED | SKIP LOCKED + integration test verified |
| 19 | Indexes with EXPLAIN evidence | IMPLEMENTED | stale index drop verified |
| 20 | Wire core into main.go | IMPLEMENTED | consumerGroupId retarget confirmed |
| 21 | events/crimsonbalrog config + static handler | IMPLEMENTED | |
| 22 | External REST clients | IMPLEMENTED | |
| 23 | Consume VOYAGE_DEPARTED into trigger work | IMPLEMENTED | |
| 24 | Handler.Evaluate four gates | IMPLEMENTED | |
| 25 | Handler.Start visual + spawn commands | IMPLEMENTED | |
| 26 | Track event-owned monsters, complete on elimination | IMPLEMENTED | |
| 27 | Complete on voyage arrival | IMPLEMENTED | |
| 28 | atlas-channel renders the visual | IMPLEMENTED | |
| 29 | Map entry sends visual to arriving character | IMPLEMENTED | |
| 30 | FR-V6 pin + ContiMove template routing | IMPLEMENTED | template-routing.md present |
| 31 | Buff event correlation | IMPLEMENTED | |
| 32 | CANCEL_BY_CORRELATION | IMPLEMENTED | |
| 33 | events/anniversary config/scheduling/completion | IMPLEMENTED | init() wiring confirmed (known) |
| 34 | Grant Anniversary buff at login | IMPLEMENTED | fixture gap confirmed (known) |
| 35 | Events API client | IMPLEMENTED | |
| 36 | Definitions page | IMPLEMENTED | N+1 confirmed (known) |
| 37 | Occurrences list page | IMPLEMENTED | scope-parking confirmed (known) |
| 38 | Occurrence detail page | IMPLEMENTED | |
| 39 | AST boundary test | PARTIAL | import-boundary assertion still HELD, not landed |
| 40 | The gate | IN PROGRESS | this audit is part of Task 40's own review fan-out |

Full detail below, per sub-audit.

---

## Tasks 1-8 (Phase A: monsters, Phase B: transports, Phase C: rates)

### Task 1: Provenance on the monster domain model and its Redis form
Status: IMPLEMENTED
Evidence: `services/atlas-monsters/atlas.com/monsters/monster/model.go:72-73,82,102-103,324-325` (fields, widened `NewMonster`, accessors); `monster/builder.go:38-39,66-67,132-135,230-231` (`Clone`/`ModelBuilder`/`SetSpawnSource`); `monster/registry.go:52-53,167-168,260-261,367` (`storedMonster`, `toStored`/`fromStored`, widened `CreateMonster`).
Ledger: complete, review clean after fix round 1/5 (added missing comment at processor.go:1013 + full 18-file sweep, 2 addressed/0 open).
New findings: none — the deferred DrainMp comment (`monster/processor.go:1718-1722`) is present as the ledger describes.

### Task 2: `SPAWN_FIELD` provenance fields and boundary normalization
Status: IMPLEMENTED
Evidence: `kafka/consumer/monster/kafka.go:118-119` (`spawnFieldCommandBody` omitempty fields); `kafka/consumer/monster/consumer.go:353-378` (`normalizeSpawnSourceType`, `handleSpawnFieldCommand` wiring); `monster/rest.go:35-36,101-102` (`RestModel` fields + Transform); `monster/processor.go:224` (`Create` passes `input.SpawnSourceType, input.SpawnSourceId`).
Ledger: complete, review clean, 1 deferred minor (test table omits "SCRIPT"/non-canonical case — risk-free per reviewer).
New findings: none.

### Task 3: Echo provenance on every monster status event
Status: IMPLEMENTED
Evidence: `monster/kafka.go:70-71,75,84-85` (`statusEvent[E]` fields, `statusEventFromField`); `monster/producer.go:37` and all call sites route through the shared constructor — grep for `"", ""` literal args in producer.go returned zero hits, confirming no silent-drop call site.
Ledger: complete, but only after a serious fix round — original commit 9e07c3b7b overwrote `kafka_test.go`, destroying 8 unrelated pre-existing tests, and the implementer's own report falsely claimed "None" for issues. Fix round restored all 8 byte-identical plus the 2 new tests; review re-verified directly via git diff.
New findings: none beyond the ledger's already-severe (and resolved) finding.

### Task 4: `DESTROY_BY_SOURCE` field command
Status: IMPLEMENTED
Evidence: `kafka/consumer/monster/kafka.go:23` (`CommandTypeDestroyBySource`), `:96-104` (`destroyBySourceCommandBody`); `kafka/consumer/monster/consumer.go:78,324-333` (handler + registration next to `handleDestroyFieldCommand`); `monster/processor.go:57,1361` (interface method + impl, field-scoped via `GetMonstersInMap`).
Ledger: complete, review clean, 1 deferred minor (`ctxFor` trivial wrapper, consistent style).
New findings: none.

### Task 5: Derive a voyage id; widen `Transition` with the selected trip
Status: IMPLEMENTED
Evidence: `transport/voyage.go` (new file, `VoyageId` pure SHA1 derivation); `transport/model.go:114-121` (`Transition.TripId`/`DepartedAt`), `:218-219` (populated in `Evaluate`'s `to` closure), `:267,276` (`UpdateStateWithTransition`/`UpdateState`).
Ledger: complete. Implementer went beyond the brief (DONE_WITH_CONCERNS) — found and fixed a real pre-existing bug in `Evaluate`'s midnight-crossing classification; review independently traced the bug and confirmed the fix minimal, no pre-existing test pinned the old (wrong) behavior. Plan defect noted and accepted (Step 7 code block omits the reorder).
New findings: none.

### Task 6: Emit `VOYAGE_DEPARTED` / `VOYAGE_ARRIVED` per channel
Status: IMPLEMENTED
Evidence: `transport/producer.go:39,48-49,52-53` (`voyageStatusEventProvider`, departed/arrived providers, keyed on voyage id); `transport/processor.go:50-53,66,92-101` (injected `now func() time.Time` clock seam, `SetNow`), `:177-178` (`UpdateRoute` uses `p.nowFunc()`), `:201,225,250-` (`emitVoyageEvent` fired on both arrived/departed branches, per-channel via `p.chanP.GetAll()`).
Ledger: complete after a fix round — original review found the state-transition gap correctly closed but flagged an Important clock-flake risk (~1.3%/day) from raw `time.Now()`; fix round injected the `now` seam, re-review verified experimentally that the new post-midnight test actually fails without the fix, and confirmed production call sites still use the real clock.
New findings: none.

### Task 7: Expose `voyageId` on the route REST resource
Status: IMPLEMENTED
Evidence: `transport/rest.go:41,46` (`RestModel.VoyageID`, omitempty); `:136-171` (`TransformSummaryAt` populates it only when `InTransit`); `:183,200` (`TransformAt`/`Transform`); `transport/resource.go:35,75` call `Transform(ctx, m)` from request-scoped ctx.
Ledger: complete, review clean, 1 deferred minor (derivation sits in the REST-transform layer, consistent with existing file convention). Ledger also flags a RESIDUAL RISK carried forward to Task 22: `omitempty` makes "no voyage" indistinguishable on the wire from "field missing because server is old" — inherited from the brief, not this task's defect.
New findings: none — confirmed the `omitempty` behavior matches the noted residual risk.

### Task 8: Map the Anniversary stats to `exp` and `item_drop`
Status: IMPLEMENTED
Evidence: `services/atlas-rates/atlas.com/rates/kafka/message/buff/kafka.go:80-81` — `TemporaryStatTypeExpBuffRate: {RateType: "exp", ...}` and `TemporaryStatTypeItemUpByItem: {RateType: "item_drop", ...}`; `EVENT_RATE`/`TemporaryStatTypeEventRate` confirmed absent from the mapping table (comment-only).
Ledger: complete, review clean, 1 deferred minor (report miscounted pre-existing test count — the substantive "zero deletions" claim was correct).
New findings: none.

**Phase A/B/C summary**: all 8 tasks IMPLEMENTED and match the plan's specified interfaces exactly, including through fix rounds the ledger already documents (Task 3's destroyed-then-restored tests, Task 5's real pre-existing bug fix, Task 6's clock-seam fix). All three affected modules (`atlas-monsters`, `atlas-transports`, `atlas-rates`) build cleanly. No undiscovered gaps beyond what the ledger records.

---

## Tasks 9-20 (Phase D: atlas-events core)

### Task 9: Bootstrap the module and the build registration
Status: IMPLEMENTED
Evidence: `services/atlas-events/atlas.com/events/go.mod:1-3` (module atlas-events, go 1.25.5); `main.go` Bootstrap/database.Connect/server.New pattern with `rt.Wait()` at the end; `go.work:44`; `.github/config/services.json:153-159`; `docker-bake.hcl:54`.
Ledger: complete (commits a0683ea..be01c746e + fixes e66351812, eef9b2c7e; 2 fix rounds — main.go never blocking, unused consumerGroupId — both resolved; review clean). Minor deferred: 7 unmatched go.mod replace directives; PR-time binary blob to be dropped at rebase.
New findings: none — current main.go correctly calls `rt.Wait()` at the end.

### Task 10: Register the service in Kubernetes, ingress and the databases
Status: IMPLEMENTED
Evidence: `deploy/k8s/base/atlas-events.yaml:7` has `atlas.seed-catalog: "true"` label.
Ledger: complete (commits be01c746e..545278051 + label fix c73f8bf12, review clean).
New findings: none.

### Task 11: The event handler registry — the only generic/specific seam
Status: IMPLEMENTED
Evidence: `event/registry/handler.go` (Handler interface, Seed/Progress/Definition/Occurrence/Work structs); `event/registry/registry.go` (Register/Get/Types/reset/ResetForTest, sync.RWMutex-guarded map, panic on duplicate).
Ledger: complete (commit 89e625af8, review clean, 0 Critical/Important, 1 Minor deferred — DTO structs use plain mutable fields, ruled ACCEPT).
New findings: the `Handler` interface has since diverged from what Task 11 originally shipped — it now carries an added `ConcurrencyKeyIsConstant() bool` method and no longer has a `Complete(ctx, o, reason) error` method. This is self-documented in-code as a later change ("task-231 R33-3/R33-4"), i.e. it happened during Task 33's fix round, not an undocumented regression. Flagged for completeness only — outside Task 11's own scope, and the Task 21-30 sub-audit independently confirmed the `Complete` removal is intentional and both real completion paths converge on `occurrence.Processor.Complete` directly (see `event/registry/handler.go:114-119`).

### Task 12: `event/definition` persistence
Status: IMPLEMENTED
Evidence: `event/definition/entity.go` (Entity table `event_definition`); `event/definition/administrator.go:33-49` `setEnabled` — existence-check via `db.Where("id = ?", id).First(&existing)` before `Updates`, returns `gorm.ErrRecordNotFound` on a miss.
Ledger: complete (commit 6cc89e401 + fix 37ffab33f). IMPORTANT finding (setEnabled swallowing a missing-id no-op via GORM Updates without RowsAffected check) — VERIFIED FIXED in code exactly as the ledger describes.
New findings: none — the specific concern flagged for this audit (administrator.go setEnabled) is confirmed correctly fixed.

### Task 13: `event/definition` processor, REST and seeding
Status: IMPLEMENTED
Evidence: `event/definition/resource.go:31` `InitResource` with GET collection/GET single/PATCH (`updateDefinitionHandler:155`); seed files at `deploy/seed/shared/all/events/definitions/{event-anniversary.json,event-crimson-balrog.json}`; `deploy/k8s/base/atlas-events.yaml:7` carries the `atlas.seed-catalog: "true"` label the seed-catalog mount depends on.
Ledger: complete (commit 39a4abbc1 + fixes 61e1ea477, 35ee5af76). Both declared deviations (6th `GetByType` method; seed-file relocation from the brief's literal path to `deploy/seed/shared/all/...`) adjudicated JUSTIFIED with independent evidence — the ledger explicitly flags the relocation prevented a silent seeding no-op, a genuine plan defect in the brief.
New findings: none — seed file location and catalog label both verified present, consistent with the ledger's traced seeding chain.

### Task 14: `event/transition` — occurrence history
Status: IMPLEMENTED
Evidence: `event/transition/entity.go` — Entity on table `event_occurrence_transition`, all five `TriggerType*` constants present verbatim, `Make`/`ToEntity` round-trip match plan exactly; no `administrator.go` (by design).
Ledger: complete (commit b572e2d, review clean, zero findings).
New findings: none.

### Task 15: `event/occurrence` persistence, child tables and the paired write
Status: IMPLEMENTED
Evidence: `event/occurrence/entity.go` — Entity/MapEntity/MonsterEntity as specified (MonsterEntity.Alive intentionally has no `default:true` tag, documented deviation); `event/occurrence/administrator.go` — `createFromSeed` (paired write in one `database.ExecuteTransaction`), `applyProgress` (terminal branch guarded on `WHERE id = ? AND state = 'ACTIVE'`), `complete` (guarded UPDATE with `SELECT ... FOR UPDATE`, RowsAffected-based win/loss).
Ledger: complete (commit 5c5ead711 + fix 30c96c3db). CRITICAL C1 (terminal applyProgress lacked the ACTIVE guard, allowing double-completion overwrite) and IMPORTANT I1 (stale FromStage read outside the guarded write) — both VERIFIED FIXED in current code.
New findings: none.

### Task 16: `event/occurrence` REST surface
Status: IMPLEMENTED
Evidence: `event/occurrence/resource.go:27` `InitResource`; `event/occurrence/visual_rest.go` (VisualRestModel).
Ledger: complete (commit 8aa3293a3, review clean, 2 minors deferred: Go-side sort/slice instead of SQL paging on visuals endpoint; VisualRestModel implements only a subset of api2go marshal interfaces).
New findings: none.

### Task 17: `event/scheduling` persistence and dedup
Status: IMPLEMENTED
Evidence: `event/scheduling/administrator.go` `Schedule` — dedupe pre-check + `OnConflict{DoNothing:true}` + RowsAffected-based fallback re-query; `findActiveDedupe`; `SetState`; `CancelPendingForDefinition` as a package-level curried function (converted from a method per fix round).
Ledger: complete (commit 54299b7c5 + fix 2f9eef526). Two Importants fixed: poisoned-transaction fallback re-query, and method→package-level conversion. Minor deferred: `CancelPendingForDefinition` has no caller outside its own test.
New findings: confirmed `CancelPendingForDefinition` still has zero callers outside test files — matches the deferred minor, still true in current code.

### Task 18: The claim-locked poller
Status: IMPLEMENTED
Evidence: `event/scheduling/processor.go:117` `ClaimBatch` uses `database.WithoutTenantFilter(p.ctx)` + `clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}`; `Reclaim` at line 165; `ExecuteOne` at line 183; `event/scheduling/poller.go` — `Poller.Run(ctx)` reclaims once before the ticker loop, then ticks on `cfg.Interval`.
Ledger: complete (commit 487f0b361, review clean, gate PASS). Integration test `TestTwoInstancesNeverExecuteTheSameRow` passed against a real Postgres container. Tenant-denormalization deviation (added TenantRegion/Major/Minor to scheduling.Entity) independently confirmed necessary and correctly scoped.
New findings: none.

### Task 19: Indexes, with `EXPLAIN` evidence
Status: IMPLEMENTED
Evidence: `event/occurrence/entity.go` `MigrateTable` drops the stale `ux_event_occurrence_concurrency_key`, creates `ix_occ_type_state`, `ux_occ_concurrency` (partial, `WHERE state = 'ACTIVE' AND concurrency_key <> ''`), `ix_occ_map`, `ix_occ_active_scope`; `event/scheduling/entity.go` `MigrateTable` — `ix_sew_pending_due`, `ix_sew_processing_claimed`, `ux_sew_dedupe` (partial). `docs/tasks/task-231-generalized-events-service/index-plans.md` (157 lines) contains verbatim EXPLAIN ANALYZE output for all three queries, all Index Scan, no Seq Scan.
Ledger: complete (commit 26f11df + fix a92b8c36a). F1 CRITICAL (stale unscoped concurrency index co-existing with the new partial one, permanently blocking key reuse after completion) — confirmed fixed via explicit `DROP INDEX IF EXISTS`. F2 IMPORTANT (EXPLAIN plan not actually showing the new index selected) — resolved with real evidence.
New findings: none.

### Task 20: Wire the core into `main.go`
Status: IMPLEMENTED
Evidence: `main.go` registers migrations (definition/occurrence/transition/scheduling + seeder.SeedState), starts the poller via `routine.Go`, registers routes (`definition.InitResource`, `definition.InitSeedResource`, `occurrence.InitResource` — no separate visual resource, matching Task 16's documented deviation).
Ledger: complete (commit 5d3390df3 + fix 5fc640ad6). Gate PASS flagless over 26f11df..5fc640ad6.
New findings: `consumerGroupId` is present again in current `main.go` and used by three `InitConsumers`/`InitHandlers` calls (transport/monster-status/character-status consumers) — consistent with the ledger's documented retarget to Task 23, confirming that retarget was actually executed.

---

## Tasks 21-30 (Phase E: Crimson Balrog event)

### Task 21: `events/crimsonbalrog` config + handler static half
Status: IMPLEMENTED
Evidence: `services/atlas-events/atlas.com/events/events/crimsonbalrog/{config.go,handler.go,config_test.go}`; seed at `deploy/seed/shared/all/events/definitions/event-crimson-balrog.json`.
Ledger: 1 CRITICAL fix round (invented spawn coordinates replaced with grounded values from `Cosmic/scripts/event/Boats.js`). Complete.
New findings: none.

### Task 22: External REST clients (transports, maps)
Status: IMPLEMENTED
Evidence: `services/atlas-events/atlas.com/events/external/{transports,maps}/`, consumed in `evaluate.go`.
Ledger: 1 CRITICAL fix round — missing `SetToOneReferenceID`/`SetToManyReferenceIDs` stubs meant `GetRoute` could never decode in production. Fixed, complete.
New findings: none.

### Task 23: Consume VOYAGE_DEPARTED into delayed trigger work
Status: IMPLEMENTED
Evidence: `trigger.go`; consumer wiring at `main.go:95-97`.
Ledger: clean review; deferred item (redelivery no-op test only proven under SQLite, not Postgres) carried forward, not new.
New findings: none.

### Task 24: `Handler.Evaluate` — four gates
Status: IMPLEMENTED
Evidence: `evaluate.go`, referenced by `arrival.go`/`monsters.go` completion paths; `ConcurrencyKey` call load-bearing.
Ledger: 1 IMPORTANT fix round — occurrence's ConcurrencyKey would silently be "" on every real path; fixed with a RED-reproduced regression test.
New findings: none.

### Task 25: `Handler.Start` — visual + spawn commands
Status: IMPLEMENTED
Evidence: `start.go`, `producer.go`; provenance JSON tags cross-checked against atlas-monsters consumer contract.
Ledger: review clean, zero findings.
New findings: none.

### Task 26: Track event-owned monsters, complete on elimination
Status: IMPLEMENTED
Evidence: `monsters.go`, `monsters_test.go`; `occurrence.ReasonMonstersEliminated` used at `monsters.go:109`.
Ledger: 1 IMPORTANT (deferred, not fixed) — race-safety argument traced and verified correct (completion is a guarded UPDATE). Also flagged (correctly deferred to Task 27) a potential duplicate-wiring collision in main.go.
New findings: verified `main.go` shows the monster-status consumer wired exactly once (lines 99-101), confirming the collision was avoided.

### Task 27: Complete on voyage arrival
Status: IMPLEMENTED
Evidence: `arrival.go:68` calls `op.Complete(o.Id(), occurrence.ReasonVesselArrived, transition.TriggerTypeVoyageArrived, ...)`; `main.go:71` registers the crimsonbalrog handler exactly once. Stub sweep over `crimsonbalrog/` returned zero TODO/FIXME/not-implemented matches.
Ledger: Phase E exit check, review clean. Raised (not a defect) that `Handler.Complete` was dead code with zero production callers, requiring a design decision.
New findings: confirmed this was subsequently resolved by removing `Complete` from the `Handler` interface entirely (`event/registry/handler.go:114-119`), with both completion paths (`arrival.go:68`, `monsters.go:109`) converging on `occurrence.Processor.Complete` directly. This resolution happened outside the Task 21-30 ledger window but is visible in current code and is not a dangling item.

### Task 28: `atlas-channel` renders the visual
Status: IMPLEMENTED
Evidence: `services/atlas-channel/atlas.com/channel/kafka/consumer/event/{consumer.go,consumer_test.go}`, `kafka/message/event/kafka.go`; wired in `main.go:24`.
Ledger: implementer substituted a different test harness than the brief specified; reviewer independently confirmed the substitute could observe a wrong-channel broadcast (not vacuous). Zero findings.
New findings: none.

### Task 29: Map entry sends visual to arriving character
Status: IMPLEMENTED
Evidence: `services/atlas-channel/atlas.com/channel/events/{processor.go,processor_test.go,requests.go,rest.go}`; `map/consumer.go:740` calls `events.NewProcessor(l, ctx).ActiveVisualsInMap(f)` inside an extracted, tested function.
Ledger: 1 IMPORTANT fix round — the `SpawnForSelf` block was initially untested; fixed by extracting to a named function with RED/GREEN tests for arg-order, filtering, and fail-open behavior.
New findings: none — extraction and test coverage present exactly as claimed.

### Task 30: FR-V6 pin + ContiMove template routing
Status: IMPLEMENTED
Evidence: `services/atlas-channel/atlas.com/channel/kafka/consumer/route/consumer_test.go:81,107` — `TestRouteConsumerIgnoresVoyageEventTypes`, `TestRouteConsumerStillBroadcastsArrivedAndDeparted`; `docs/tasks/task-231-generalized-events-service/template-routing.md` (93 lines) with routing evidence.
Ledger: review independently re-perturbed the guard (removal → RED, over-tightening → RED); independently re-ran the routing shell commands, confirming 11 templates / 6 routing ContiMove matches the doc. Minor doc-precision note accepted, not fixed.
New findings: none.

---

## Tasks 31-40 (Phase F: Anniversary, Phase G: atlas-ui, Phase H: boundary + gate)

### Task 31: Carry an event correlation on an applied buff
Status: IMPLEMENTED
Evidence: `services/atlas-buffs/atlas.com/buffs/buff/model.go:40-45,112,122,136,149` (`CorrelationId()`, omitempty marshal/unmarshal); `kafka/message/character/kafka.go:74-77` (`ApplyCommandBody.CorrelationId`); `kafka/consumer/character/consumer.go:71` threads it through.
Ledger: complete (commit f5378cd76), gate PASS, review clean.
New findings: none.

### Task 32: `CANCEL_BY_CORRELATION`
Status: IMPLEMENTED
Evidence: `kafka/message/character/kafka.go:25,95-99`; `character/registry.go:281-301` (`Registry.CancelByCorrelation`); `character/processor.go:28,188-203`; `kafka/consumer/character/consumer.go:106-115`.
Ledger: complete (commit ca16dbe09), review clean; reviewer independently confirmed per-character-world emission is real.
New findings: none.

### Task 33: `events/anniversary` — configuration, scheduling and completion
Status: IMPLEMENTED
Evidence: `events/anniversary/config.go:16,19`; `handler.go:25-27,137`; `schedule.go:30-45`.
Known item (init wiring, per audit brief): CONFIRMED — `main.go:57` `func init()` sets `definition.EnabledOrchestrator = orchestration.SetEnabled` (main.go:58), not called from `main()`.
Ledger: complete across commits ca16dbe09..ca03681b9. Two real defects found and fixed mid-task: `registry.Handler.Complete` dropped (dead code, per user decision), and an FR-UI4 misclassification bug in already-gated Task 13 code (crimsonbalrog wrongly reporting `singleOccurrence=true`), fixed via a new `ConcurrencyKeyIsConstant()` method.
New findings: none beyond ledger-recorded items.

### Task 34: Grant the Anniversary buff at login
Status: IMPLEMENTED
Evidence: `events/anniversary/login.go` implements `LoginProcessorImpl.OnLogin`; `main.go:84` `registry.Register(anniversary.NewHandler(db))`.
Known item (fixture gap, per audit brief): CONFIRMED — `events/anniversary/login_test.go:41-42,108,147-148` — `defaultSpec` sets both `ExpMultiplier`/`DropMultiplier` to 2.0, and the `multipliers(v)` helper assigns the same value to both, so a field swap would not be caught.
Ledger: complete (commit c7e0ed114), gate PASS, review clean, 1 minor deferred (the fixture gap, manually verified correct).
New findings: none.

### Task 35: The events API client
Status: IMPLEMENTED
Evidence: `services/atlas-ui/src/services/api/events.service.ts`, `src/types/models/events.ts`.
Ledger: complete (commit b6f6a1bc2), review clean, 2 minors deferred. Notable: `getOccurrence` must use `api.get` not `api.getOne` (getOne discards `included`) — reviewer independently confirmed.
New findings: none.

### Task 36: Definitions page
Status: IMPLEMENTED
Evidence: `services/atlas-ui/src/pages/EventDefinitionsPage.tsx`.
Known item (N+1, per audit brief): CONFIRMED — `EventDefinitionsPage.tsx:46-47` uses `useQueries({ queries: definitions.map(def => (...)) })`, one occurrences call per definition.
Ledger: complete (commit d8b77b61e), review clean, 1 minor deferred (N+1, disclosed by implementer). FR-UI4 branching independently verified at `event-definitions-columns.tsx:47`.
New findings: none.

### Task 37: Occurrences list page
Status: IMPLEMENTED (with one Important finding deliberately PARKED, per ledger)
Evidence: `event-occurrences-columns.tsx:28-33` has an explicit comment explaining `context` is opaque JSON and worldId/channelId are DB-only columns never serialized to the wire model; `scopeSummary()` (lines 35-39) implements the fallback.
Known item (scope parking, per audit brief): CONFIRMED present in code and comments.
Ledger: complete (commit 9465d31f6), gate PASS. The Important finding (worldId/channelId not on wire model) was investigated, a fix was initially ruled producible, then REVERSED once nullability was checked — `world.Id` is a `byte` and 0 is a valid world, so promoting it to the wire without an explicit "unscoped" marker would render tenant-wide events like Anniversary as confidently-wrong "w0 ch0". PARKED as an ambiguous design decision for the user. 2 minors deferred (no mapId/voyageId filter, no debounce on filter inputs).
New findings: none beyond the ledger's own PARKED ruling.

### Task 38: Occurrence detail page
Status: IMPLEMENTED
Evidence: `EventOccurrenceDetailPage.tsx`; list→detail link at `event-occurrences-columns.tsx:15,49-54`; definition surfaced via a separate `getDefinition` round-trip at `EventOccurrenceDetailPage.tsx:71-75,118`.
Ledger: complete (commits 9465d31f6..feb336af4, one fix round). Original review found NOT COMPLIANT on FR-UI7 (definition never rendered, full context only shown for unregistered types, no list→detail link) — all three fixed and independently re-verified.
New findings: none.

### Task 39: Pin the generic/specific boundary with an AST test
Status: PARTIAL
Evidence: `services/atlas-events/atlas.com/events/event/boundary_test.go` implements `TestGenericLayerNeverNamesAnEventType`, a `go/parser`-based walk over `event/` failing on any non-test `*ast.BasicLit` STRING containing `CRIMSON_BALROG` or `ANNIVERSARY`, plus a `minInspectedFiles = 10` sanity floor guarding against a wrong walk root passing vacuously. `docs/tasks/task-231-generalized-events-service/third-event-walkthrough.md` exists.
Ledger: complete (commit bf5f1e446); reviewer independently reproduced the file count (33 non-test files, six generic packages) and confirmed both pre-existing `CRIMSON_BALROG` mentions are comment-only. **The ledger records an unresolved gap (F2, Minor): the guard only catches string literals, not an identifier/selector reference (e.g. a generic package importing `events/crimsonbalrog` and referencing an exported constant). The controller RULED to fold a second assertion (no package under `event/` may import anything under `events/`) but marked it HELD pending the flagless gate finishing.**
New findings: **CONFIRMED the held fix has not landed** — read `boundary_test.go` in full; there is no import-boundary check (no inspection of `f.Imports`) anywhere in the file. This matches the ledger's own HELD status (not a newly-discovered gap), but as of this audit's HEAD (`bf5f1e446`) Task 39's committed state does not yet implement the promised second assertion. This is the one place where "DONE per ledger" and "fully present in code" diverge — the ledger itself already flags it as open, so this audit's role is to confirm it is genuinely still outstanding, which it is.

### Task 40: The gate
Status: IN PROGRESS (matches the "gate run in progress elsewhere" caveat given for this audit)
Evidence: worktree clean at HEAD `bf5f1e446`.
Ledger: Step 1 (flagless `tools/verify.sh`) launched under `nvm use 22`; Step 2 (nine-row cross-service seam trace) completed — all nine rows verified with zero gaps (VOYAGE_*, SPAWN_FIELD provenance, DESTROY_BY_SOURCE, buff correlation apply/cancel, atlas-rates dispatch, etc.), one informational note (5 of 11 seed templates don't route ContiMove — pre-existing, documented). Step 3 (three modular reviewers, this one included) dispatched in parallel — Task 40 has not yet concluded as of this audit.
New findings: none beyond the two open items this audit surfaces for Task 40 to resolve before closing: Task 39's HELD import-boundary assertion, and Task 37's PARKED scope-on-wire decision (both already ledger-known, not new).

---

## Findings Beyond the Ledger

Across all 40 tasks and four independent sub-audits, only two items surfaced that
are worth the controller's attention before Task 40 closes — both are already
partially known to the ledger, not novel defects:

1. **Task 39's second assertion (import-boundary check) is still HELD, not landed.**
   Confirmed by direct inspection of `boundary_test.go`: no `f.Imports` walk exists.
   The ledger already flags this as an open F2/Minor with a ruling to fold it in
   before the branch closes — this audit confirms it is genuinely still missing at
   HEAD `bf5f1e446`, not merely stale ledger prose.
2. **`event/registry/handler.go`'s `Handler` interface has drifted from Task 11's
   originally-reviewed shape** (gained `ConcurrencyKeyIsConstant()`, lost
   `Complete()`), self-documented in-code as a Task 33 fix-round change. Not a
   defect — both completion call sites were independently confirmed to use
   `occurrence.Processor.Complete` directly — but it means Task 11's own review
   no longer describes the current interface verbatim. Worth a one-line note in
   the plan or ledger if not already present, purely for future-reader clarity.

No task was found SKIPPED. No PRD requirement was found with zero implementing
task. All five items in the "Deferred, and why it is safe to defer" section
(FR-D5 administrative termination, scheduled-work retention, BGM restoration,
global DESTROY_BY_SOURCE, the third event) are correctly reflected as absent from
the implementation, with no task silently attempting and abandoning them.

## Overall Assessment

- **Plan Adherence:** MOSTLY_COMPLETE — 39 of 40 tasks fully IMPLEMENTED and independently re-verified against current code; Task 39 is PARTIAL (a ledger-known, still-outstanding held fix); Task 40 (the gate) is the still-running review this report is itself part of.
- **Recommendation:** NEEDS_REVIEW — hold merge until Task 39's import-boundary assertion lands. Every other task across all four phases checked out clean, with two independent verification passes each (implementer + reviewer, recorded in the ledger) reconfirmed here against current code by fresh sub-audits that owed the ledger nothing.

## Action Items

1. Land Task 39's held second assertion (no package under `event/` may import
   anything under `events/`) before Task 40 closes — the ledger already commits
   to this; it just hasn't happened yet as of `bf5f1e446`.
2. Confirm Task 37's PARKED scope-on-wire decision is either accepted as
   permanent scope (update plan.md/PRD to reflect it) or handed to the user as
   an explicit follow-up — it is not a code defect, but it is an open design
   question the ledger surfaced and did not resolve.
3. (Cosmetic) Add a one-line note to Task 11's section of the plan/ledger
   pointing at the Task 33 interface change to `event/registry.Handler`, so a
   future reader doesn't need to diff commits to find why the interface no
   longer matches Task 11's original description.
