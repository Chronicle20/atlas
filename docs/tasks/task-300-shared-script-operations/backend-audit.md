# Backend Audit — task-300-shared-script-operations

- **Service Path:** libs/atlas-script-core (ops), services/atlas-map-actions, services/atlas-reactor-actions, services/atlas-portal-actions, services/atlas-npc-conversations, services/atlas-saga-orchestrator (test-only)
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-09-04
- **Build:** PASS
- **Tests:** all packages `ok` (no failures observed across the six affected modules)
- **Overall:** NEEDS-WORK

## Build & Test Results

```
cd libs/atlas-script-core && go build ./... && go test ./... -count=1
  ok  github.com/Chronicle20/atlas/libs/atlas-script-core/ops   0.006s
  (condition, context, operation, outcome: no test files, pre-existing)

cd services/atlas-map-actions/atlas.com/map-actions && go build ./... && go test ./... -count=1
  ok  atlas-map-actions            0.018s
  ok  atlas-map-actions/script     0.039s

cd services/atlas-reactor-actions/atlas.com/reactor && go build ./... && go test ./... -count=1
  ok  atlas-reactor-actions        0.017s
  ok  atlas-reactor-actions/script 0.044s

cd services/atlas-portal-actions/atlas.com/portal && go build ./... && go test ./... -count=1
  ok  atlas-portal-actions         0.020s
  ok  atlas-portal-actions/action  6.815s
  ok  atlas-portal-actions/dedupe  1.751s
  ok  atlas-portal-actions/kafka/consumer/saga 0.009s
  ok  atlas-portal-actions/script  0.045s

cd services/atlas-npc-conversations/atlas.com/npc && go build ./... && go test ./... -count=1
  ok  atlas-npc-conversations (all sub-packages) — no failures

cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./... -count=1
  ok  atlas-saga-orchestrator/saga  0.510s
  (all other sub-packages ok or no test files)
```

`tools/script-ops-guard.sh` → exit 0, `OK — no shared script-operation payload constructed under the script-operation-table services.`
`tools/script-ops-guard_test.sh` → exit 0, all 8 assertions pass.
`tools/goroutine-guard.sh` (repo-wide) → exit 0.

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| FILE placement (FILE-01..06) | Yes | Every changed Go package (ops, four script/executor packages, npc/saga, npc/conversation) is in scope unconditionally. |
| DOM structure (DOM-01..05,11,16) | No | No changed package has `model.go`, `entity.go`, `rest.go`, or `provider.go`. |
| SUB (SUB-01..04) | No | No changed package has `resource.go` without `model.go`. |
| REST (DOM-06..09,12..15,17..19,32) | Partially — DOM-06 only | Two changed packages have `processor.go` (`npc/conversation/processor.go`, `npc/saga/processor.go`); no `resource.go` and no route registration anywhere in scope, so only DOM-06 (processor.go's own trigger) evaluates. |
| Constants reuse (DOM-21) | Yes | Diff declares new types (`Step`, `Target`, `TargetBuilder`, `ParamError`, `Resolver`, `DirectResolver`, `QuestDefaults`, `skillParams`) in `libs/atlas-script-core/ops`. |
| Testing (DOM-10,20,24,33) | Yes | Diff adds/changes `_test.go` files across all six modules. |
| Cache (DOM-29) | No | No `cache.go`; no processor/struct holds cached state in the diff. |
| Messaging (DOM-30) | No | No changed file calls `AndEmit`/`message.Emit`/`producer.ProviderImpl` directly; saga creation goes through the pre-existing, unchanged `sagaP.Create` seam. |
| Multi-tenancy (DOM-31) | No | No changed file has `rest.go`; no changed code reads/passes tenant or trace state. |
| Migration hygiene (DOM-34,35) | Yes | Diff extracts/re-homes the script-operation payload-construction logic into `libs/atlas-script-core/ops` and removes the `atlas-npc-conversations/saga` re-export shim (`model.go`, `builder.go`). |
| Deploy & topics (DOM-22,23) | No | `libs/atlas-script-core` already existed pre-diff (not a new `libs/atlas-*` module); no Kafka topic env var added or renamed. |
| Runtime safety (DOM-26) | Yes | Non-test Go files changed across six modules; `tools/goroutine-guard.sh` exit 0. |
| Channel wire values (DOM-25) | No | Diff does not touch `services/atlas-channel` or `libs/atlas-packet`; message-type string mapping (`"5"`→`PINK_TEXT`, `"6"`→`BLUE_TEXT`) is carried forward unchanged logic, not a new client-interpreted byte. |
| Resilience (DOM-27,28) | No | No DB-backed handler error branch changed; no `model.Decorator`/enrichment path changed. |
| External clients (EXT-01..04) | Yes (pre-existing, unmodified logic) | `services/atlas-reactor-actions/.../script/executor.go` and `evaluator.go` call `requests.GetRequest[pqInstanceRestModel]` for the PARTY_QUESTS service; the call bodies are unmodified by this diff (see WARN below). |
| Scaffolding (SCAFFOLD-01..09) | No | No new `services/atlas-<svc>/` directory, no new channel writer/handler, no `routes.conf` change. |
| Security (SEC-01..04) | No | None of the affected services handle auth tokens, redirects, or secrets. |
| Foundational: patterns-provider.md | No | Diff does not define/compose providers. |
| Foundational: patterns-functional.md | Partial | `ops` package uses curried-adjacent constructors (`NewTargetBuilder`, builder pattern) — reviewed inline under DOM structure discussion; no violation found. |

## Checklist Results

### libs/atlas-script-core/ops (support — no model.go/resource.go)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File responsibilities | N/A | Package holds none of the FILE-01..05 categories (no Processor, RestModel, requests.go, entity.go, builder+model+admin+provider). It is a new "shared step builder" file category the checklist does not classify; no catch-all combining ≥2 of the listed responsibilities exists — `ops.go`, `message.go`, `monster.go`, `movement.go`, `skill.go`, `quest.go`, `environment.go`, `effect.go` each own one saga-payload family. |
| DOM-21 | No redeclaration of shared domain types | PASS | `libs/atlas-script-core/ops/ops.go:60-116` (`Step`, `Target`, `TargetBuilder`, `ParamError`, `Resolver`) and `ops/quest.go:18-22` (`QuestDefaults`) are new operation-building types, not domain classifications; `ops/monster.go:5-8` and `ops/movement.go:1-8` reuse `libs/atlas-constants/map.Id`, `ops/environment.go:29-56` reuses `field.ParseObjectKind` from `libs/atlas-constants/field` rather than redeclaring. |
| DOM-20 | Table-driven tests | FAIL | `libs/atlas-script-core/ops/ops_test.go:14` (`TestDirectResolver`), `:81` (`TestParamErrorMessage`), `:110` (`TestTargetBuilder`), `:153` (`TestStepAppendTo`), `:173` (`TestPayloadOf`), `:190` (`TestRangeHelpers`), `:317` (`TestOptionalIntUsesResolver`) — all use `t.Run` subtests with no `tests := []struct{...}` table. Also `effect_test.go:177` (`TestPlayPortalSound`), `environment_test.go:165` (`TestResetEnvironment`), `message_test.go:157` (`TestSendMessageResolvesThroughResolver`). |
| DOM-24 | producertest stub for emit-reaching tests | N/A | `ops` package performs no I/O by design (`ops.go:1-5` doc comment: "no network, Redis, Kafka or REST I/O"); tests never reach `AndEmit`/`message.Emit`/`producer.Produce`. |
| DOM-26 | Goroutines via `routine.Go` | PASS | No bare `go` statement in the package; `tools/goroutine-guard.sh` exit 0 repo-wide. |
| DOM-34 | No re-export shims left after migration | PASS | `ops` is the *destination* of the migration, not a re-export; `grep -rnE '=\s*[a-z]\w*\.[A-Z]'` over the package found no delegating aliases (only `var now = time.Now`, an internal test seam, `ops.go:21`). |
| DOM-35 | No dead symbols left after extraction | PASS | N/A to `ops` itself (destination); see `atlas-npc-conversations/saga` row below for the source-side check. |

### services/atlas-map-actions/atlas.com/map-actions/script (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File responsibilities | PASS | `executor.go` holds only `OperationExecutor`/`NewOperationExecutor`/`Execute*` methods — a script-executor role, not a Processor/RestModel/requests/entity/builder-model-admin-provider catch-all. No FILE-06 violation. |
| DOM-20 | Table-driven tests | PASS | `executor_test.go:283` `TestExecuteSpawnMonsterCarriesInstance` uses `tests := []struct{...}` + `t.Run` (verified via grep of the 6 lines following the func signature). |
| DOM-26 | Goroutines | PASS | No bare `go`; `tools/goroutine-guard.sh` exit 0. |
| DOM-34 | Direct delegation, no wrapper | PASS | `executor.go:95-96,114-115,158-159,178-179,197-198` call `ops.NewTargetBuilder(f).Build()` / `ops.<Op>(...)` directly — no local wrapper duplicating the shared builder. |

### services/atlas-reactor-actions/atlas.com/reactor/script (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File responsibilities | PASS | `executor.go` holds only executor methods; no catch-all. |
| FILE-03 | Cross-service request functions live in `requests.go` | WARN | `executor.go:472-479` (`getPqInstanceByCharacter`) and `evaluator.go:177-183` both call `requests.GetRequest[pqInstanceRestModel]` for the PARTY_QUESTS service from outside `requests.go`; both function bodies are byte-for-byte unchanged by this diff (`git diff` shows no hunk touching those lines) — pre-existing debt this migration did not introduce or touch. Non-blocking. |
| EXT-01 | Target REST model implements `SetToOneReferenceID`/`SetToManyReferenceIDs` | WARN | `evaluator.go:155-168` defines `pqInstanceRestModel` with `GetName`/`GetID`/`SetID` but no `SetToOneReferenceID`/`SetToManyReferenceIDs`; pre-existing, unmodified by this diff. Non-blocking, same rationale as FILE-03 row. |
| DOM-20 | Table-driven tests | Mixed | `executor_test.go:313` `TestExecuteSpawnMonsterRejectsBadNumerics` — PASS (table present). `TestExecuteDropMessageAcceptsTypeAlias` — FAIL, no `tests := []struct{...}` table (single-scenario `t.Run`-free assertion block). |
| DOM-26 | Goroutines | PASS | No bare `go`; guard exit 0. |
| DOM-34 | Direct delegation | PASS | `executor.go:200-201,224-225,251-252,281-282,320-321,438-439` call `ops.*` directly. |

### services/atlas-portal-actions/atlas.com/portal/script (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File responsibilities | PASS | `executor.go` holds only executor/opTable-dispatch methods; no catch-all. |
| DOM-20 | Table-driven tests | FAIL | `executor_test.go:164` `TestExecuteDropMessageAcceptsTypeAlias`, `:183` `TestExecuteCreateSkillWidensLevel`, `:203` `TestExecuteWarpKeepsTransactionWiring` — all single-scenario, no `tests := []struct{...}` table. |
| DOM-26 | Goroutines | PASS | No bare `go`; guard exit 0. |
| DOM-34 | Direct delegation | PASS | `executor.go` calls `ops.PlayPortalSound`, `ops.WarpToPortal`, `ops.SendMessage`, `ops.ShowHint`, `ops.CreateSkill`, `ops.UpdateSkill`, `ops.StartInstanceTransport`, `ops.ApplyConsumableEffect`, `ops.SaveLocation`, `ops.WarpToSavedLocation`, `ops.StartQuest` directly — no local re-implementation. |

### services/atlas-npc-conversations/atlas.com/npc/conversation (support — no model.go)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `processor.go:116` `func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor`. |
| FILE-01 | Processor lives in `processor.go` | PASS | `type Processor interface` at `processor.go:33`. |
| DOM-20 | Table-driven tests | Mixed | `operation_executor_test.go` new funcs: `TestCreateStepSendMessageDefaultsMessageType:1159`, `TestCreateStepSpawnMonsterOptionalPosition:1229`, `TestCreateStepCreateSkillHonoursExpiration:1302`, `TestCreateStepWarpToMapRequiresMapId:1348`, `TestCreateStepStartQuestUsesContextDefaults:1426`, `TestCreateStepStageClearAttemptPq:1514`, and `TestExecuteDropMessageAcceptsMessageTypeAlias` — all FAIL, no `tests := []struct{...}` table (each is a single-scenario body). |
| DOM-26 | Goroutines | PASS | No bare `go`; guard exit 0. |
| DOM-34 | Direct delegation, no shim | PASS | `operation_executor.go` imports `saga "github.com/Chronicle20/atlas/libs/atlas-saga"` directly (line 27) and calls `ops.WarpToPortal`, `ops.CreateSkill`, `ops.UpdateSkill`, `ops.SpawnMonster`, `ops.StartQuest`, `ops.ApplyConsumableEffect`, `ops.SendMessage`, `ops.PlayPortalSound`, `ops.ShowHint`, `ops.ShowIntro`, `ops.StartInstanceTransport`, `ops.SaveLocation`, `ops.WarpToSavedLocation`, `ops.StageClearAttemptPq` — no local re-implementation of the extracted payload construction. `processor.go:6,80-91,873,917,979,1047,1108,1173` import and call `npcsaga` (the local, still-legitimate `atlas-npc-conversations/saga` package for `Processor`/`ValidateCharacterStatePayload`) directly, not through the removed re-export. |

### services/atlas-npc-conversations/atlas.com/npc/saga (support — shim removal)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-34 | No re-export shim survives the move | PASS | `saga/model.go` (164 lines of `type X = sharedsaga.X` / `const X = sharedsaga.X` re-exports) and `saga/builder.go` (`type Builder = sharedsaga.Builder`, `var NewBuilder = sharedsaga.NewBuilder`) are deleted in full (`git diff --stat`: `model.go 164 ---`, `builder.go 12 --`). Remaining `saga/model.go` after the diff holds only the genuinely NPC-specific `ValidateCharacterStatePayload` wrapper (not a shim — it converts a local `validation.ConditionInput` slice to the shared type). |
| DOM-35 | No dead symbols left behind | PASS | `grep -n '\bSymbolName\b'` for a sample of the removed re-exported names (`WarpToPortalPayload`, `SpawnMonsterPayload`, `CreateSkillPayload`, etc.) across `services/atlas-npc-conversations/atlas.com/npc` finds only qualified references through the shared `saga` import alias (`saga.WarpToPortalPayload`, etc.) at call sites already updated in this diff — no remaining reference to the deleted local aliases. |

### services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga (test-only)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-20 | Table-driven tests | PASS | `handler_test.go` new `TestHandleSpawnMonsterCarriesInstanceToField` uses `tests := []struct{...}` + `t.Run` (verified: table literal directly precedes the `for _, tt := range tests` loop). |
| DOM-33 | Interface change updates every mock | N/A | `handler.go` is untouched by this diff (`git diff` empty for that file); `WithFootholdProcessor`/`WithMonsterProcessor` already existed pre-diff. The new test adds in-file func-field test doubles (`footholdProcessorMock`, `monsterProcessorMock`) that satisfy the pre-existing `foothold.Processor`/`monster.Processor` interfaces (`var _ foothold.Processor = (*footholdProcessorMock)(nil)` at handler_test.go, `var _ monster.Processor = (*monsterProcessorMock)(nil)`) — no interface signature changed. |
| DOM-24 | producertest stub for emit-reaching tests | N/A | `handleSpawnMonster` under test does not itself emit; it calls the injected `monsterP.SpawnMonster`/`footholdP.GetFootholdBelow` mocks, which return without touching Kafka. |

## Security Review

Not applicable — SEC-* trigger did not fire. None of the six affected services/modules handle authentication, authorization, tokens, redirects, or secrets in the changed code.

## Not evaluable from the diff

- DOM-30 (messaging AndEmit/Buffer pattern) for the `sagaP.Create` seam each executor calls into: the `saga.Processor.Create` implementations (`.../saga/processor.go` in each of the four consumer services) are unchanged by this diff, so whether they follow the `AndEmit` + `message.Buffer` pattern was not re-verified here; it was out of the changed-file surface.
- DOM-27 (503 on transient DB errors) — none of the changed packages are DB-backed HTTP handlers, so no handler error branch exists to evaluate; noted here rather than silently passed since the family's other members did fire.

## Summary

### Blocking (must fix)

- DOM-20: `libs/atlas-script-core/ops/ops_test.go:14,81,110,153,173,190,317` — seven test functions use bare `t.Run` subtests, not the required `tests := []struct{...}` + `t.Run` table-driven shape.
- DOM-20: `libs/atlas-script-core/ops/effect_test.go:177` (`TestPlayPortalSound`), `environment_test.go:165` (`TestResetEnvironment`), `message_test.go:157` (`TestSendMessageResolvesThroughResolver`) — same non-table-driven shape.
- DOM-20: `services/atlas-portal-actions/atlas.com/portal/script/executor_test.go:164,183,203` (`TestExecuteDropMessageAcceptsTypeAlias`, `TestExecuteCreateSkillWidensLevel`, `TestExecuteWarpKeepsTransactionWiring`) — same.
- DOM-20: `services/atlas-reactor-actions/atlas.com/reactor/script/executor_test.go` (`TestExecuteDropMessageAcceptsTypeAlias`) — same.
- DOM-20: `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor_test.go:1159,1229,1302,1348,1426,1514` (`TestCreateStepSendMessageDefaultsMessageType`, `TestCreateStepSpawnMonsterOptionalPosition`, `TestCreateStepCreateSkillHonoursExpiration`, `TestCreateStepWarpToMapRequiresMapId`, `TestCreateStepStartQuestUsesContextDefaults`, `TestCreateStepStageClearAttemptPq`) and `TestExecuteDropMessageAcceptsMessageTypeAlias` — same.

### Non-Blocking (should fix)

- FILE-03 / EXT-01: `services/atlas-reactor-actions/atlas.com/reactor/script/executor.go:472-479` and `evaluator.go:155-183` — cross-service request function outside `requests.go`, and `pqInstanceRestModel` missing `SetToOneReferenceID`/`SetToManyReferenceIDs`. Pre-existing, unmodified by this diff; flagged for a future pass.

---

## Controller disposition of the DOM-20 blocking findings (post-audit)

**Recorded by the `/execute-task` controller after the audit was written. The audit's
findings above are unmodified; this section records how they were dispositioned and why.**

The five blocking findings are all DOM-20 (table-driven test shape). They were verified
before being acted on, per `superpowers:receiving-code-review`, and the verification
split two ways.

**The findings are factually accurate.** `libs/atlas-script-core/ops/ops_test.go:14` and
every other cited site really are bare `t.Run` subtests with inline literals rather than
`tests := []struct{...}` + `t.Run`. Nothing in the audit's evidence is disputed.

**The blocking severity is contradicted by the repository's own convention:**

- Only **4 of 3,119** `*_test.go` files repo-wide contain a `:= []struct{` table literal
  (`git grep -l ":= \[\]struct{" -- '*_test.go' | wc -l` against `git ls-files '*_test.go' | wc -l`).
- Every test file this branch changed contains zero such literals — **and each already
  contained zero at the merge base `31a791e3a`**, before task-300 touched it:
  `saga/handler_test.go` 33 funcs / 0 tables; `conversation/operation_executor_test.go`
  16 / 0; `portal/script/executor_test.go` 7 / 0.

Task-300's new tests therefore *match* the local convention of the files they live in;
they are not a regression, and no defect is masked by their shape. Converting only this
branch's ~15 test functions would leave it inconsistent with ~3,115 sibling files.

This is the precedent set at `docs/tasks/task-051-status-cure-consumables/audit.md:119`,
where DOM-20 was dispositioned WARN/non-blocking on exactly this reasoning: the new tests
mirrored the existing style in the same files.

**Disposition: the five DOM-20 findings are accepted as non-blocking WARNs. No code
change.** Effective verdict for this branch: APPROVED_WITH_FINDINGS — 0 blocking,
6 WARN (the 5 DOM-20 plus the pre-existing FILE-03/EXT-01 WARN in
`atlas-reactor-actions/script`, which this diff does not touch).

Ruled by the user during the pre-PR review step. If the repo later adopts DOM-20's literal
shape as a real convention, that is a repo-wide sweep, not a task-300 change.

## Post-audit commit

One commit landed after this audit was written, closing the plan-adherence audit's single
non-blocking finding:

- `701ad8b9b` — test(npc-conversations): pin update_skill expiration threading.
  Test-only, +47 lines in `conversation/operation_executor_test.go`, adding
  `TestCreateStepUpdateSkillHonoursExpiration` beside its `create_skill` sibling. Confirmed
  RED against a temporarily broken `ops/skill.go` sentinel, then GREEN with that file
  restored. No production code changed. It deliberately follows the file's existing
  non-table shape, consistent with the DOM-20 disposition above.
