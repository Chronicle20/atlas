# Plan Audit — task-086-mount-system

## Plan Adherence Audit

**Plan Path:** docs/tasks/task-086-mount-system/plan.md
**Audit Date:** 2026-06-12
**Branch:** task-086-mount-system
**Base:** c53d4e691fa8d1dded28073ee1e7419834cd0f3e (main) → HEAD 7834f13d93a2f7b1358f729645d9bc4e2892f2e7

### Executive Summary

All 42 tasks (incl. Task 30b, 41b) are accounted for: 40 IMPLEMENTED, 1 SKIPPED with documented rationale (Task 8), 1 DEFERRED as a pre-deploy gate (Task 41b). Checkbox tally: 110 checked / 7 unchecked — and the 7 unchecked boxes map *exactly* to the two intentional non-implementations (Task 8 = 2 boxes, Task 41b = 5 boxes), so there are **no silent gaps**. Build + test spot-checks on the three acceptance-critical modules (atlas-packet, atlas-mounts, atlas-channel) are green. The Kafka contract chain, the MONSTER_RIDING byte encoding (self + foreign), the 5-case channel toggle, Task 30b's dual-producer cancel-all-buffs, and FR-9 quest-20523 all verified against code.

**Verdict: the implementation faithfully executes the plan.**

### Per-task findings

| Task | Status | Evidence |
|---|---|---|
| 1 Pin game data | IMPLEMENTED | context.md §8.1–8.7 filled with HeavenMS/live-atlas-data citations |
| 2 MONSTER_RIDING base-stat encode (self) | IMPLEMENTED | `libs/atlas-packet/model/character_temporary_stat.go:731-735` (`NewCharacterTemporaryStatBaseWithOptions(false, s.Value(), s.SourceId())`); constructor at `:341`; test `character_temporary_stat_test.go:144` |
| 3 Observer (EncodeForeign) path | IMPLEMENTED | `EncodeForeign` shares `getBaseTemporaryStats()` (`:621` calls `:724`); test `:161 TestCTSMonsterRidingForeignEncodesVehicleAndSkill` |
| 4 atlas-packet module gate | IMPLEMENTED | `go build`/`go test ./model ./character/clientbound ./mount/serverbound` PASS (re-run this audit) |
| 5 SET_TAMING_MOB_INFO writer | IMPLEMENTED | `libs/atlas-packet/character/clientbound/set_taming_mob_info.go:41-51` field order characterId,level,exp,tiredness,levelUp; test `set_taming_mob_info_test.go` |
| 6 atlas-packet gate | IMPLEMENTED | green (above) |
| 7 Skill reader vehicle ids | IMPLEMENTED | `services/atlas-data/atlas.com/data/skill/reader.go:465-472` uses `skill.SkillOnlyMountVehicleId`; test `reader_test.go` |
| 8 Consumable tiredness-heal spec | **SKIPPED (documented)** | plan.md:386-413 + context §8.4: revitalizer 2260000 WZ carries `incFatigue:0/spec.inc:0`; heal is server constant 30. Constant verified used — `consumables/kafka/message/food/kafka.go:61 RevitalizerTirednessHeal = 30`, applied at `consumable/processor.go:269`. Both boxes intentionally unchecked. |
| 8b atlas-data gate | IMPLEMENTED | reader builds/tests green |
| 9 Scaffold atlas-mounts | IMPLEMENTED | `services/atlas-mounts/atlas.com/mounts/go.mod` (`module atlas-mounts`), `logger/init.go`, `go.work` entry |
| 10 character_mounts entity + migration | IMPLEMENTED | `mount/entity.go` (uniqueIndex tenant+character); `entity_test.go` |
| 11 Model + Builder | IMPLEMENTED | `mount/model.go`, `mount/builder.go` (defaults level 1/exp 0/tiredness 0); `builder_test.go` |
| 12 Feed math | IMPLEMENTED | `mount/feed.go` (ExpNeededForLevel table, ApplyFeed, CAP=31, table bound-guard); `feed_test.go` incl. multi-level + table-end boundary |
| 13 Tiredness clamp | IMPLEMENTED | `mount/tiredness.go` `TickTiredness`→min(99,t+1)+TooTired; `tiredness_test.go` |
| 14 Administrator + Processor | IMPLEMENTED | `mount/administrator.go`, `mount/processor.go` (default-on-read GetByCharacterId, upsert, ApplyTick/ApplyFeedAndEmit/EmitSet); `processor_test.go` |
| 15 Mount-status Kafka + producer | IMPLEMENTED | `kafka/message/mount/kafka.go` (EVENT_TOPIC_MOUNT_STATUS, SET/TICK/FEED), `mount/producer.go`; tests `kafka_test.go`, `producer_test.go` |
| 16 Redis active-mount registry | IMPLEMENTED | `mount/registry.go` (MountRideContext via atlas-redis TenantRegistry); `registry_test.go`; redis-key-guard clean per deploy-notes §8 |
| 17 Constants + helpers | IMPLEMENTED | `libs/atlas-constants/skill/constants.go` (+14 lines, exact Noblesse/Legend ids per §8.5, not offset-derived), `item/constants.go` ClassificationRevitalizer, `skill/mount.go` IsTamedMountSkill+SkillOnlyMountVehicleId; `mount_test.go` |
| 18 Buff consumer → registry | IMPLEMENTED | `kafka/consumer/buff/consumer.go:92` IsTamedMountSkill→registryAdd+EmitSet; skill-only→EmitSet only; EXPIRED→remove; `consumer_test.go` |
| 19 Login/logout gating | IMPLEMENTED | `kafka/consumer/character/consumer.go` (online registry + active-mount removal on logout); `consumer_test.go` |
| 20 TamingMobFed consumer | IMPLEMENTED | `kafka/consumer/food/consumer.go`→ApplyFeedAndEmit; `consumer_test.go` |
| 21 60s tiredness ticker | IMPLEMENTED | `mount/task.go` (cadence time.Minute over registry), `tasks/task.go`; `task_test.go` |
| 22 REST resource | IMPLEMENTED | `mount/rest.go`, `mount/resource.go` (GetName "mounts"), `rest/handler.go`; `rest_test.go` |
| 23 main.go wiring | IMPLEMENTED | `main.go` (registry, db migration, buff/character/food consumers, REST, tiredness task) |
| 24 atlas-mounts gate | IMPLEMENTED | full module `go build`+`go test` green (re-run this audit) |
| 25 Mount toggle branch | IMPLEMENTED | `skill/handler/mount.go` HandleMount 5 cases; branch at `common.go:100`; `mount_test.go` 5 cases (CancelsWhenAlreadyMounted, TamedRequiresBothSlots, TamedAppliesVehicleFromSlot18, TamedSlot18EmptyNoOp, SkillOnlyNoSlotCheck) |
| 26 Writer registration | IMPLEMENTED | `socket/writer/set_taming_mob_info.go` + `main.go produceWriters` |
| 27 Mount-status consumer → broadcast | IMPLEMENTED | `kafka/consumer/mount/consumer.go:84` SET/TICK/FEED broadcast; `:96` TooTired rider notice; `kafka/message/mount/kafka.go`; `consumer_test.go` |
| 28 Food opcode 0x4D handler | IMPLEMENTED | `socket/handler/mount_food.go`, packet `libs/atlas-packet/mount/serverbound/food.go` (ts,slot,itemId); `mount_food_test.go`, `food_test.go` |
| 29 Channel food command + producer | IMPLEMENTED | `kafka/message/food/kafka.go` (COMMAND_TOPIC_TAMING_MOB_FOOD), `food/producer.go`, `food/processor.go`; `producer_test.go` |
| 30 Job-change dismount (investigation) | IMPLEMENTED (→30b) | plan.md:1108-1127 documents FR-4.2 was NOT pre-existing; resolved by Task 30b |
| 30b cancel_all_buffs saga step | IMPLEMENTED | **both producers**: `atlas-npc-conversations/.../operation_executor.go:821,932` (gated on `action == saga.ChangeJob`, single + batch paths) and `atlas-messages/.../command/character/commands.go:34` (GM @change job). saga.CancelAllBuffs added to both saga model.go; `operation_executor_test.go`, `commands_test.go` |
| 31 atlas-channel gate | IMPLEMENTED | channel mount packages build+test green (re-run this audit) |
| 32 Consumables food command consumer | IMPLEMENTED | `kafka/consumer/food/consumer.go` (class-226 validate→consume→emit); `consumer_test.go` |
| 33 TamingMobFed event producer | IMPLEMENTED | `kafka/message/food/kafka.go:45 EVENT_TOPIC_TAMING_MOB_FOOD` + `RevitalizerTirednessHeal=30`; emitted `consumable/processor.go:269` |
| 34 atlas-consumables gate | IMPLEMENTED | per deploy-notes §8 |
| 35 Cross-service contract check | IMPLEMENTED | topic constants match across all 3 producer/consumer pairs (verified this audit) |
| 36 Quest definition | IMPLEMENTED (via WZ) | quest 20523 exists in WZ; conversation drives it. context §8.7 corrected |
| 37 NPC conversation | IMPLEMENTED | `deploy/seed/gms/83_1/npc-conversations/quests/quest-20523.json` (questId 20523, npc 1102002, start_quest+complete_quest only — WZ EndActions award saddle 1912005, mob 1902005, skill 10001004) |
| 38 Questline validation | IMPLEMENTED | recorded in plan/deploy-notes §6 |
| 39 services.json + docker-bake | IMPLEMENTED | `.github/config/services.json` +8, `docker-bake.hcl` +1 (atlas-mounts) |
| 40 K8s manifest | IMPLEMENTED | `deploy/k8s/base/atlas-mounts.yaml` (DB_NAME atlas-mounts, no LB ports) + `kustomization.yaml` |
| 41 docker buildx bake | IMPLEMENTED (reported) | deploy-notes §8: bake clean for 6 services. Not re-run this audit (no docker). |
| 41b Cross-version IDA verify | **DEFERRED (documented)** | context §2, deploy-notes §5: pre-deploy gate needing v87/v95/JMS IDBs; 5 boxes intentionally unchecked |
| 42 Live-config notes + final gate | IMPLEMENTED | `deploy-notes.md` (0x4D handler + SetTamingMobInfo writer patch + restart; §8 verification) |

### Acceptance-critical verifications (re-run during this audit)

- **atlas-packet:** `go build ./...` OK; `go test ./model ./character/clientbound ./mount/serverbound` → all PASS (incl. the two MONSTER_RIDING self+foreign byte tests).
- **atlas-mounts:** `go build ./...` OK; `go test ./...` → mount, kafka/consumer/{buff,character,food}, kafka/message/mount all PASS.
- **atlas-channel:** `go build ./...` OK; `go test ./skill/handler ./kafka/consumer/mount ./socket/handler ./food` → all PASS.

### Findings (silent-gap scan)

- **No silent gaps.** Unchecked boxes (7) reconcile exactly to Task 8 (2) + Task 41b (5), both with explicit in-plan rationale. No task is checked-but-absent-in-code.
- Task 8's skip is load-bearing-correct: the substitute constant (30) is genuinely present and wired (`RevitalizerTirednessHeal`, consumables `processor.go:269`).
- Task 30 originally assumed CancelByStatTypes; plan honestly records FR-4.2 was unimplemented in the codebase and resolved server-wide via Task 30b. The `cancel_all_buffs` step is confirmed in **both** job-change producers and gated on the ChangeJob action in npc-conversations (single + batch saga paths).
- FR-9 correctly relies on quest 20523's WZ EndActions (saddle 1912005 / mob 1902005 / skill 10001004) with the conversation issuing only `start_quest`/`complete_quest` — matching the documented `suppressAwardAssetByCompleteQuest` dedup behavior (avoids double-grant).
- Kafka chain is contract-consistent end to end: `COMMAND_TOPIC_TAMING_MOB_FOOD` (channel→consumables), `EVENT_TOPIC_TAMING_MOB_FOOD` (consumables→mounts), `EVENT_TOPIC_MOUNT_STATUS` (mounts→channel).

### Recommendation

READY (subject to the documented Task 41b pre-deploy cross-version IDA gate before enabling on non-v83 tenants, and the live-config opcode patch from deploy-notes §2 for existing tenants). Docker bake (Task 41) was reported clean but not re-run in this audit (no docker available here) — trust-but-verify in CI.

---

# Backend Guidelines Audit (DOM-*)

- **Primary subject:** new service `services/atlas-mounts/atlas.com/mounts/`
- **Secondary subjects:** atlas-consumables (food), atlas-channel (food/mount), atlas-data (skill reader), atlas-constants, atlas-packet, saga `cancel_all_buffs` additions
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-06-12
- **Mindset:** FAIL-until-proven, file:line evidence required
- **Build:** PASS — atlas-mounts `go build ./...` clean; atlas-channel / atlas-consumables / atlas-data / atlas-messages / atlas-npc-conversations `go build ./...` clean
- **Tests:** PASS — atlas-mounts `go test ./...` all packages PASS; changed packages in secondary modules PASS (`atlas-consumables/{consumable,kafka/consumer/food}`, `atlas-channel/{food,kafka/consumer/mount,skill/handler,socket/handler}`)
- **Overall:** NEEDS-WORK (build+tests green; 3 FAIL checks below)

## Phase 1 — Build & Test Gate

```
atlas-mounts: go build ./...  -> exit 0
atlas-mounts: go test ./...   -> all PASS (mount, kafka/consumer/{buff,character,food}, kafka/message/mount)
atlas-channel/atlas-consumables/atlas-data/atlas-messages/atlas-npc-conversations: go build ./... -> exit 0
```

## Phase 2 — Domain Discovery (atlas-mounts)

- `mount/` — **Domain package** (`model.go` present) → full DOM checklist.
- `kafka/consumer/{buff,character,food}`, `kafka/message/*`, `kafka/producer`, `rest`, `tasks`, `logger` — **Support packages** (Kafka/REST/infra) → idiom checks only.

## Phase 3 — Domain Checklist: `mount`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go exists (NewBuilder, fluent setters, Build()) | PASS | `mount/builder.go:22` `NewModelBuilder`, fluent `SetLevel/SetExp/...`, `Build()` @ `:61`. (Build() returns `(Model, nil)` with no validation — acceptable here; all fields have sane defaults/no invariants, but see Minor M-3) |
| DOM-02 | `ToEntity()` on Model | N/A→PASS | No `ToEntity()`; the create path builds `Entity` directly in `administrator.go:13`. Convention varies by service (pets does the same). Not a violation. |
| DOM-03 | `Make(Entity)` returns `(Model,error)` | PASS | `mount/entity.go:28` |
| DOM-04 | `Transform` in rest.go | PASS | `mount/rest.go:37` |
| DOM-05 | `TransformSlice` / no inline loops in resource | N/A | Single-resource GET only; resource uses `model.Map(Transform)` (`resource.go:35`), no slice handler. No inline loop. |
| DOM-06 | Processor accepts `FieldLogger` | PASS | `mount/processor.go:39` `NewProcessor(l logrus.FieldLogger, ...)` |
| DOM-07 | Handlers pass `d.Logger()` | PASS | `mount/resource.go:28` `NewProcessor(d.Logger(), d.Context(), d.DB())`; no `StandardLogger()` |
| DOM-08 | POST/PATCH use `RegisterInputHandler` | N/A | Read-only service; only GET registered (`resource.go:20`). No POST/PATCH. |
| DOM-09 | Transform errors handled | PASS | `resource.go:35-40` checks `err` from `model.Map(Transform)` |
| DOM-10 | Test DB registers tenant callbacks | PASS | `mount/processor_test.go:44` `database.RegisterTenantCallbacks(l, db)` |
| DOM-11 | Providers lazy / context-scoped queries | PASS | `administrator.go:33,87` query via passed `db`; processor uses `p.db.WithContext(p.ctx)` (`processor.go:71`). No eager-in-FixedProvider. |
| DOM-12 | No `os.Getenv()` in handlers | PASS | grep `os.Getenv` in `resource.go` → 0 |
| DOM-13 | No cross-domain logic in handlers | PASS | Handler calls only `mount` processor (`resource.go:29`) |
| DOM-14 | Handlers don't call providers directly | PASS | `resource.go` calls `p.GetByCharacterId` only; no `getByCharacterId`/`create` from handler |
| DOM-15 | No direct entity writes in handlers | PASS | grep `db.Create/Save/Delete` in `resource.go` → 0; writes go processor→administrator |
| DOM-16 | `administrator.go` for writes | PASS | `mount/administrator.go` (`create`/`update`); processor calls them in a tx |
| DOM-17 | Domain error → HTTP status mapping | PARTIAL | `resource.go` maps every error to 500 (`:32`,`:38`). Acceptable for a read/default-create GET (no 4xx-class domain errors reachable), but not differentiated. Minor M-4. |
| DOM-18 | JSON:API interface on REST model | PASS | `rest.go:21-32` `GetName/GetID/SetID` |
| DOM-19 | Request models flat (no nested Data/Type/Attributes) | PASS / N/A | No request models (read-only). `RestModel` is flat. |
| DOM-20 | Table-driven tests | PASS | `mount/feed_test.go`, `mount/producer_test.go` use `[]struct{...}`+`t.Run`. Other tests are scenario-style (acceptable). |
| DOM-21 | Reuse atlas-constants types (no reinvention) | PASS | `world.Id` (`processor.go:10`), `skill.Id`/`skill.IsTamedMountSkill` (`consumer/buff/consumer.go:92`), `skill.SkillOnlyMountVehicleId` (atlas-data `reader.go`), `item.GetClassification`/`ClassificationRevitalizer` + `inventory.TypeFromItemId` (consumables `processor.go`), `character.TemporaryStatTypeMonsterRiding` (`consumer/buff/consumer.go:70`). New `Classification(226)` added to the shared lib (`item/constants.go:40`), not redeclared locally. No reinvented id/classification types. |
| DOM-22 | Dockerfile lib wiring | PASS (shared-Dockerfile variant) | Repo uses a single parameterized root `Dockerfile`; service registered in `.github/config/services.json:289`, `docker-bake.hcl:71`, `go.work:59`. All direct-require libs (`atlas-redis`,`atlas-service`,etc.) present in root `Dockerfile` mod-only + source COPY blocks. |
| DOM-23 | Kafka topic in env-configmap (`KEY: "KEY"`), no literal override | **FAIL** | The two NEW topics `EVENT_TOPIC_MOUNT_STATUS` and `EVENT_TOPIC_TAMING_MOB_FOOD` are absent from `deploy/k8s/base/env-configmap.yaml` and `deploy/compose/.env.example` (grep → 0 hits across `deploy/`). All three services that use them (atlas-mounts producer, atlas-channel consumer, atlas-consumables producer/consumer) rely on `topic.EnvProvider`, which (provider.go:18) falls back to the literal token name on a missing env var — so producer/consumer stay aligned and the chain still works, but every pod logs a warning and the central topic registry is bypassed. The deployment manifest correctly uses `envFrom: configMapRef: atlas-env` and does NOT add forbidden literal `- name:/value:` overrides. Fix = add `EVENT_TOPIC_MOUNT_STATUS: "EVENT_TOPIC_MOUNT_STATUS"` and `EVENT_TOPIC_TAMING_MOB_FOOD: "EVENT_TOPIC_TAMING_MOB_FOOD"` to `env-configmap.yaml` (+ compose `.env.example`, + the `pr` overlay placeholder block). |
| DOM-24 | Kafka producer stubbed via shared `producertest` in tests that emit | **FAIL** | `atlas-mounts`: PASS — processor tests only buffer (`mb *message.Buffer`, assert `mb.GetAll()`; `processor_test.go:139,192,224`) and never call `Emit`; consumer tests override the emit seams (`applyFeed`/`applyTick`/`emitSet`/`registryAdd`) with fakes (`consumer/*/consumer_test.go`). No unstubbed emit path. **`atlas-consumables` food consumer test FAILS the rule**: `kafka/consumer/food/consumer_test.go:19-74` rolls a **service-local** `capturingWriter`/`writerRegistry` + `kafkaProducer.ConfigWriterFactory`, which DOM-24(d) forbids (shared `producertest` is the single source of truth), and uses `t.Cleanup(kafkaProducer.ResetInstance)` (DOM-24(e) forbids — un-stubs the singleton for later tests; there is also no `TestMain` noop in the package). Mitigating context: the test must *capture & assert* the emitted `TamingMobFed` shape, and the shared `producertest` only offers discard-only `InstallNoop()` — it has no capturing variant — so a literal swap to `producertest` would lose the assertion. The genuinely fixable parts are (a) drop `t.Cleanup(ResetInstance)` / add a `TestMain` noop floor, and (b) ideally upstream a capturing helper into `producertest`. |

## Support-Package Idiom Checks (atlas-mounts)

| Area | Status | Evidence |
|------|--------|----------|
| Immutable Model + private fields + getters | PASS | `mount/model.go:13-49` all-private fields, value-receiver getters |
| Builder + Clone(immutability) | PASS | `builder.go:33` `Clone`; processor mutates only via `Clone(m).SetX().Build()` (`processor.go:119,161`) |
| Processor interface+impl, `NewProcessor(l,ctx,db)`, `With(WithTransaction)` | PASS | `processor.go:23-64`; tx+emit atomic via `database.ExecuteTransaction` wrapping both `update()` and `mb.Put()` (`processor.go:112-134`, `:150-180`) |
| GORM entity + migration | PASS | `entity.go:10-26`; unique index `(tenant_id, character_id)`; registered `database.SetMigrations(mount.Migration)` (`main.go:61`) |
| Multi-tenancy scoping | PASS | `tenant.MustFromContext` (`processor.go:44`, `registry.go:52,65`); queries use `db.WithContext(ctx)` + tenant callback, no manual `Where("tenant_id=?")` (`administrator.go:35,47,88`); ticker re-scopes each entry's tenant (`task.go:66`) |
| Redis via `libs/atlas-redis` only | PASS | `registry.go` uses `atlas.NewTenantRegistry`/`atlas.NewSet`; `goredis.Client` used solely as the constructor arg type. grep for raw keyed go-redis calls (`.Set/.Get/.Del/.SAdd/...`) outside lib → 0. redis-key-guard clean. |
| message.Buffer / Emit idiom | PASS | `kafka/message/message.go`; emit seams call `mountmessage.Emit(producer.ProviderImpl(l)(ctx))` (`consumer/food/consumer.go:26`, `consumer/buff/consumer.go:35`, `task.go:31`) |
| Curried `InitConsumers`/`InitHandlers` | PASS | `consumer/buff/consumer.go:41,49`; `consumer/food`, `consumer/character` same shape |
| `main.go` does NOT call RegisterTenantCallbacks | PASS | `main.go` uses `database.Connect(...)` (auto-registers); no manual call |

## Findings — Blocking (must fix)

- **DOM-23 (Important): new Kafka topics missing from `env-configmap.yaml`.** `EVENT_TOPIC_MOUNT_STATUS` + `EVENT_TOPIC_TAMING_MOB_FOOD` are not declared in `deploy/k8s/base/env-configmap.yaml`, `deploy/compose/.env.example`, or the `deploy/k8s/overlays/pr` placeholder block. Runtime is saved only by `topic.EnvProvider`'s literal-token fallback (provider.go:18), so the chain functions but logs warnings on every pod and bypasses centralized topic management. Add both keys (`KEY: "KEY"` shape) to the three locations.
- **DOM-24 (Important): consumables food consumer test stubs the producer with a service-local capturing writer + `t.Cleanup(ResetInstance)`.** `services/atlas-consumables/atlas.com/consumables/kafka/consumer/food/consumer_test.go:19-74`. Drop `t.Cleanup(kafkaProducer.ResetInstance)` (it un-stubs the singleton for subsequent tests) and add a package `TestMain` that installs a noop floor; prefer extending the shared `producertest` with a capturing variant rather than a bespoke `capturingWriter`.

## Findings — Non-Blocking (should fix)

- **M-1 (Minor): dead code in `mount/administrator.go`.** `upsert` (`:60`) and `deleteByCharacterId` (`:87`) have zero references anywhere in the service (cloned-from-pets carryover). Anti-patterns guide forbids leaving dead code after refactoring. Delete both.
- **M-2 (Minor): unused `Extract` (`mount/rest.go:50`) and `EmitWithResult` (`kafka/message/message.go:61`).** Both unreferenced. `Extract`/`EmitWithResult` are conventional scaffolding present in most services, so lower priority than M-1, but still dead in this read-only service.
- **M-3 (Minor): `Build()` performs no validation** (`builder.go:61` always returns nil error). DOM-01 ideally validates invariants in Build(); none are enforced here (e.g., level≥1, tiredness 0..99). Low risk because all writers clamp via pure helpers, but the builder contract is nominally weaker than the guideline.
- **M-4 (Minor): `mount/resource.go` maps all errors to HTTP 500** (`:32`,`:38`). DOM-17 wants differentiated status. Acceptable here (the only reachable error is an infra/DB failure on a default-create GET), but not strictly differentiated.
- **M-5 (Minor / informational): wire-contract structs use raw `uint32` character ids** (`kafka/message/{buff,character,food}/kafka.go`, consumables `TamingMobFedEventProvider`). This is NOT a DOM-21 violation — the upstream producers (atlas-character, atlas-buffs) emit `CharacterId uint32`, and these structs must mirror them byte-for-byte. Flagged only so a future reviewer doesn't "fix" it into a contract drift.
- **M-6 (Minor / informational): `feed.go` CAP=31 vs 29-entry `mountExp` table.** The code's own comment (`feed.go:11-15`) acknowledges levels ≥ len(table) return `MaxInt32` so the loop terminates safely; functionally bounded, not a guideline violation, but the CAP/table mismatch is worth a confirming glance against HeavenMS source.

## Overall Verdict

**NEEDS-WORK.** The new `atlas-mounts` service is a clean, idiomatic clone of the pets pattern: immutable model+builder, processor interface/impl with `With(WithTransaction)` and atomic tx-then-emit, tenant-scoped GORM access with no manual tenant predicates, Redis confined to the `atlas-redis` lib wrappers, and curried Kafka consumer/handler registration. Build and tests are green. Two Important issues must be fixed before merge — the missing env-configmap topic declarations (DOM-23) and the consumables food-consumer producer-stub discipline (DOM-24) — plus the dead-code cleanup (M-1). DOM-21 constant reuse is fully satisfied across all touched modules.

---

## Plan adherence audit (post-mount-fix) — 2026-06-13

**Plan Path:** docs/tasks/task-086-mount-system/plan.md
**Branch:** task-086-mount-system (HEAD cfb91e71b)
**Auditor focus:** post-IDA mount-render fix + SET_TAMING_MOB_INFO → char-info substitution; food opcodes per version; Task 41b cross-version status.

### Executive summary

The plan is faithfully implemented. All 42 numbered tasks are DONE except: Task 8 (correctly conditional-SKIPPED, WZ carries no heal field), Task 41b (PARTIAL — v83/v84/v87/v95/jms verified, v12/v92 explicitly deferred with no IDB), and Task 30 (resolved via the user-approved Task 30b cancel_all_buffs substitution). The post-plan IDA discovery — that no supported client has a standalone SET_TAMING_MOB_INFO opcode — was handled by a complete, consistent substitution: the standalone broadcast is guarded OFF and mount stats are injected into the character-info packet. Builds and targeted tests are green across all 6 affected modules; redis-key-guard (workspace mode) exits 0.

### Build & test results (this audit, worktree)

| Module | Build | Tests (targeted) |
|---|---|---|
| libs/atlas-packet | PASS | PASS (model, character/clientbound, mount/serverbound) |
| libs/atlas-constants | PASS | PASS (skill) |
| services/atlas-mounts | PASS | PASS (all pkgs); `go vet` clean |
| services/atlas-channel | PASS | PASS (skill/handler, socket/handler, kafka/consumer/mount, food) |
| services/atlas-consumables | PASS | PASS (kafka/consumer/food, consumable) |
| services/atlas-data | PASS | PASS (skill) |

redis-key-guard.sh (workspace mode): exit 0.

### SET_TAMING_MOB_INFO substitution — VERIFIED COMPLETE & CONSISTENT

The plan's standalone-broadcast design (Tasks 5/26/27) was superseded by char-info injection. Every leg of the substitution is present:

1. **Packet model** — `libs/atlas-packet/character/clientbound/info.go:42-47` `MountInfo{Active,Level,Exp,Tiredness}`; encode at `info.go:111-120` writes a 1-byte present flag then 3×int32 (or a single 0 byte), with a matching decode at `:189`. Documented as IDA-uniform across v83/v87/v95/JMS.
2. **Standalone writer still exists but is GUARDED OFF** — `services/atlas-channel/.../kafka/consumer/mount/consumer.go:63` `clientHasStandaloneTamingMobInfo = func(_ tenant.Model) bool { return false }`; gated at `:106`. The Task-5 writer (`libs/atlas-packet/.../set_taming_mob_info.go`) and Task-26 body wrapper (`socket/writer/set_taming_mob_info.go`) remain as dead-but-ready code behind the flag. The TooTired notice (`:111-113`) still fires unconditionally — correct.
3. **Char-info handler fetches the mount** — `socket/handler/character_info_request.go:65-75` queries `mount.NewProcessor(...).GetByCharacterId(...)` and passes a populated `MountInfo` into `writer.CharacterInfoBody`.
4. **atlas-channel/mount REST client** — `services/atlas-channel/atlas.com/channel/mount/{model,processor,requests,rest}.go` present and compiles.
5. **nginx route** — `deploy/shared/routes.conf:91` and `deploy/k8s/base/routes.conf.template.generated:91` route `^/api/characters/[^/]+/mount(/.*)?$` → `atlas-mounts:8080`.

No half-done edges found: the broadcast path, the guard, the char-info injection, the REST client, and the nginx route are mutually consistent.

### Food opcodes (Task 28 / 41b)

Per-version `MountFoodHandle` opcode bindings confirmed in seed templates (services/atlas-configurations/seed-data/templates):
- gms_83_1 = 0x4D, gms_84_1 = 0x4D, gms_87_1 = 0x50, gms_95_1 = 0x53, jms_185_1 = 0x45.
These match deploy-notes.md §"Multi-version mount food opcodes". v12/v92 intentionally absent (no IDB) — documented. Serverbound body (`libs/atlas-packet/mount/serverbound/food.go` ts/slot/itemId) version-uniform per IDA.

### Kafka contract cross-check (Task 35) — PASS

- `COMMAND_TOPIC_TAMING_MOB_FOOD` (channel→consumables), `EVENT_TOPIC_TAMING_MOB_FOOD` (consumables→mounts), `EVENT_TOPIC_MOUNT_STATUS` (mounts→channel) — topic constants identical on both ends; all three declared in `deploy/k8s/base/env-configmap.yaml` + `deploy/compose/.env.example` (DOM-23 fix landed).
- Revitalizer heal pinned: `RevitalizerTirednessHeal = 30` (consumables `kafka/message/food/kafka.go:61`), emitted at `consumable/processor.go:270` gated on `item.ClassificationRevitalizer` (226).

### Task-by-task status

| # | Task | Status | Evidence |
|---|---|---|---|
| 1 | Pin game data | DONE | context.md §8 (CAP=31, mountExp 29-entry table, heal=30, quest 20523) |
| 2-3 | MONSTER_RIDING base-stat encode (self+foreign) | DONE | character_temporary_stat.go:849-856 (nOption=Value/vehicle, rOption=SourceId/skill); tests pass |
| 4,6 | atlas-packet module gates | DONE | build+test green this audit |
| 5 | SET_TAMING_MOB_INFO writer | DONE (superseded/guarded) | set_taming_mob_info.go present; guarded off — see substitution above |
| 7 | atlas-data skill reader vehicle ids | DONE | skill/reader.go (+reader_test.go); tests pass |
| 8 | Consumable tiredness-heal spec | SKIPPED (justified) | WZ carries no heal field (context §8.4); heal is server constant 30 — task's own conditional |
| 8b | atlas-data gate | DONE | build+test green |
| 9-24 | atlas-mounts service (scaffold→entity→model→feed→tiredness→processor→kafka→registry→consumers→ticker→rest→main→gate) | DONE | full service builds, vet clean, all pkg tests pass; feed.go CAP=31 + table match §8.2; registry via atlas-redis (key-guard clean) |
| 17 | atlas-constants mount ids/class226/helpers | DONE | skill/constants.go, skill/mount.go, item/constants.go; tests pass |
| 25 | Channel mount toggle | DONE | skill/handler/mount.go (both-slots -18/-19, MaxInt32, isMounted, skill-only); mount_test.go pass |
| 26-27 | Writer reg + mount-status consumer | DONE (guarded) | broadcast guarded off; TooTired notice live; char-info injection is the active path |
| 28-29 | Food handler 0x4D + channel command | DONE | socket/handler/mount_food.go; food/producer.go; per-version opcodes in templates |
| 30 | Job-change dismount | DONE (via 30b) | resolved by cancel_all_buffs substitution |
| 30b | cancel_all_buffs on job change | DONE | npc operation_executor.go:821 + messages commands.go:34 both append the step |
| 31 | Channel gate | DONE | build+targeted tests green |
| 32-34 | Consumables food consumer + TamingMobFed + gate | DONE | consumable/processor.go RequestFeed (class-226 gate, heal 30); consumer tests pass |
| 35 | Kafka contract check | DONE | topic constants + heal verified identical across pairs (above) |
| 36-38 | Riding Mimiana questline | DONE | deploy/seed/gms/83_1/npc-conversations/quests/quest-20523.json (start/complete; WZ EndActions award skill 10001004 + saddle 1912005 + taming-mob 1902005) |
| 39-40 | services.json + docker-bake + go.work + k8s manifest | DONE | all reference atlas-mounts; routes + env-configmap + PR overlay onboarded |
| 41 | docker buildx bake | DONE (per deploy-notes §8) | not re-run in this read-only audit; recorded clean on branch |
| 41b | Cross-version packet verification | PARTIAL (deferral documented/justified) | v83/v84/v87/v95/jms food opcodes + v95 SecondaryStat verified; v12/v92 deferred (no IDB); JMS skill-ids + main-tenant live patches remain post-merge. Plan checkboxes for 41b left unchecked — honest. |
| 42 | Live-config deploy note + final gate | DONE | deploy-notes.md present and detailed |

### Silently-skipped work

None found. The only unchecked boxes are Task 8 (conditional skip, documented with rationale) and Task 41b steps (pre-deploy gate, explicitly partial with v12/v92 and JMS-skill-id remainder called out in plan + deploy-notes). No task claimed DONE without supporting code.

### Verdict

**Plan adherence: FULL** (with the two documented, justified non-DONE items above). The post-plan architectural pivot (char-info injection replacing the standalone opcode) is complete and internally consistent — not half-done. **Recommendation: READY_TO_MERGE** for v83/v84/v87/v95/jms tenants; v12/v92 enablement and live main-tenant opcode patches are correctly gated behind Task 41b as post-merge / pre-deploy work.

---

# Backend guidelines audit (post-mount-fix) — 2026-06-13

Re-audit of the full task-086 branch (`git diff origin/main..HEAD`) after the fixes that
followed the 2026-06-12 audit. Scope: atlas-mounts (new service), atlas-channel mount/food
client + consumer + socket handlers, atlas-consumables food consumer, libs/atlas-packet
(character_temporary_stat, character info, mount food), libs/atlas-constants (temporary_stat,
skill/mount). Default-FAIL mindset; every PASS carries a file:line.

## Build & Test Results (objective gate)

| Module | build | vet | test -race |
|--------|-------|-----|------------|
| services/atlas-mounts/atlas.com/mounts | PASS | PASS | PASS (mount, kafka/consumer/{buff,character,food}, kafka/message/mount) |
| libs/atlas-packet | PASS | — | PASS (all packages) |
| libs/atlas-constants | PASS | — | PASS (incl. skill/mount_test) |
| services/atlas-channel/atlas.com/channel | PASS | PASS | PASS (mount, kafka/consumer/mount, socket/handler, socket/writer, skill/handler, food) |
| services/atlas-consumables/.../kafka/consumer/food | — | — | PASS (0.006s — confirms producer stub floor; no 42s emit hang) |

All four named modules build, vet, and test clean. Objective gate: **PASS** — proceed.

## Status changes vs the 2026-06-12 audit

- **DOM-23 — now PASS (was FAIL/blocking).** All three topics are declared in
  `deploy/k8s/base/env-configmap.yaml` with the required `KEY: "KEY"` shape:
  `EVENT_TOPIC_CHARACTER_STATUS` (:92), `EVENT_TOPIC_MOUNT_STATUS` (:125),
  `EVENT_TOPIC_TAMING_MOB_FOOD` (:142). The atlas-mounts Deployment consumes them via
  `envFrom: configMapRef` (`deploy/k8s/base/atlas-mounts.yaml:21-22`) with no literal
  per-service overrides. The earlier blocking miss is resolved.
- **DOM-24 — now PASS (was FAIL/blocking).** `services/atlas-consumables/.../kafka/consumer/food/consumer_test.go`
  now has a `TestMain` installing the shared `producertest.InstallNoop()` floor (:27-29), and
  the forbidden `t.Cleanup(ResetInstance)` revert is gone — the helper explicitly documents NOT
  resetting the singleton (:82-91). The service-local `capturingWriter`/`ConfigWriterFactory`
  layered on top (:89-91) is the accepted message-capture path (producertest has no capturing
  variant); the two negative tests assert *no* emission and the shape test calls the provider
  directly, so no real broker path is reachable. Test runtime 0.006s confirms it.

## Domain Checklist — atlas-mounts `mount` (unchanged PASS, spot-reverified)

DOM-01..DOM-22 all PASS as previously recorded. Reverified high-signal items:
- DOM-06 `logrus.FieldLogger` processor.go:39; DOM-10 `RegisterTenantCallbacks` processor_test.go:44;
  DOM-11 `db.WithContext(ctx)` processor.go:71 + string-based WHERE administrator.go:35,47 (no manual
  tenant predicate, no struct-WHERE); DOM-15 no `db.Create/Save/Delete` in resource.go;
  DOM-21 reuse confirmed — buff/consumer.go:70 uses `characterconst.TemporaryStatTypeMonsterRiding`,
  :92 `skill.IsTamedMountSkill`; no service-local redeclaration of shared types.
- DOM-17 remains **WARN** (non-blocking): mount/resource.go:32,38 map every error to HTTP 500.
  Acceptable because the processor default-creates on first read (processor.go:70-103), so a genuine
  404 is unreachable on this GET; only infra errors arrive, for which 500 is correct.

## New findings this pass (not in prior audit) — atlas-channel mount REST client

The `services/atlas-channel/atlas.com/channel/mount/` client was not assessed against the External
HTTP Client checklist before. Triggered by `requests.GetRequest[RestModel]` in requests.go:18.

- **EXT-01 — PASS.** Upstream `mounts` resource is a flat JSON:API model with no `relationships`
  block (`services/atlas-mounts/atlas.com/mounts/mount/rest.go` has no relationship methods), so the
  missing `SetToOneReferenceID`/`SetToManyReferenceIDs` on the channel-side `RestModel` (rest.go:4-19)
  cannot cause an api2go unmarshal error. Low risk; not blocking.
- **EXT-02 — FAIL (non-blocking).** No httptest-backed integration test exists for the client. The
  `mount/` package has only model.go/processor.go/requests.go/rest.go — no `_test.go` at all. The
  client's `GetByCharacterId` unmarshal path (processor.go:26-28 via `requests.Provider`) is exercised
  by nothing. A `httptest.NewServer` fixture returning the upstream `{"data":{"type":"mounts",...}}`
  shape and asserting a populated Model is required by the checklist.
- **EXT-03 — FAIL (Important).** `socket/handler/character_info_request.go:66` treats **any**
  `GetByCharacterId` error as "no mount block" (`if ... mErr == nil { populate }` else leaves
  `mountInfo` zero/inactive). The client does not use `requests.ErrNotFound` anywhere
  (grep -> 0 in mount/). Because atlas-mounts default-creates a row on first read, a real 404 is not
  expected — which means every error that reaches this branch is actually a transport/decode/5xx
  failure being silently swallowed as "character has no mount." This is exactly the masking the
  checklist warns against. At minimum the non-404 error should be logged at the call site (it currently
  is not) and ideally surfaced distinctly from the genuine no-mount case.
- **EXT-04 — PASS.** URL composed via `requests.RootUrl("MOUNTS")` (requests.go:14), not hardcoded DNS.

## Overall Verdict (post-fix)

**NEEDS-WORK.** The two prior blocking items (DOM-23 env-configmap, DOM-24 producer stub) are
**resolved** — build/vet/test are green across all four modules and the consumables food test no
longer risks the 42s emit hang. The atlas-mounts service itself remains a clean, idiomatic
implementation with full DOM-21 constant reuse. The remaining issues are confined to the
atlas-channel mount REST client: EXT-03 (error masking at character_info_request.go:66) is the one
item worth fixing before merge; EXT-02 (no httptest client test) is a should-fix. Neither breaks the
build. No new blocking defects in the mount domain itself.
