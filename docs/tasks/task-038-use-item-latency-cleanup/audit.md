# Plan Audit — task-038-use-item-latency-cleanup

**Plan Path:** docs/tasks/task-038-use-item-latency-cleanup/plan.md
**Audit Date:** 2026-04-30
**Branch:** task-038-use-item-latency-cleanup
**Base Branch:** main (54d506c16)
**HEAD:** a9cdbd014

## Executive Summary

All 17 plan tasks are implemented. The 12 commits on this branch cleanly map to the 17 plan items (Tasks 2+3, 5+6, 7+8, and 11+12 are paired in single commits per the test-then-impl-in-one-commit convention noted in the task list; Task 17 is verification-only with no commit). All four affected modules build cleanly and every package's tests pass: `libs/atlas-model`, `services/atlas-channel/atlas.com/channel`, `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`, and `services/atlas-consumables/atlas.com/consumables`. Task 17's three guard checks (no stale "Unable to announce character ... health" log strings; `SkipReasonNilTransactionId` wired in three places; `model.NewGroup`/`model.Submit` only inside `ConsumeStandard`/`ConsumeTownScroll`/`ConsumeSummoningSack`) all pass.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Add `golang.org/x/sync` direct dependency to atlas-model | DONE | `libs/atlas-model/go.mod:5` (`require golang.org/x/sync v0.20.0`); commit 407658c3a |
| 2 | Failing `Group`/`Submit`/`Future` tests | DONE | `libs/atlas-model/model/parallel_group_test.go` (107 lines, all 5 test functions present: TestGroup_TwoSuccessfulProviders, TestGroup_OneProviderErrors, TestGroup_BothProvidersError, TestGroup_ThreeProviders_AllSucceed, TestGroup_ConcurrencyProof); paired with Task 3 in commit 314feb7d1 |
| 3 | Implement `Group`/`Submit`/`Future` | DONE | `libs/atlas-model/model/parallel_group.go:13-54` (Group struct, Future[T any], NewGroup, Submit free function, Wait); commit 314feb7d1 |
| 4 | Add minimal `NewBuilder` to `atlas-channel/party` | DONE | `services/atlas-channel/atlas.com/channel/party/model.go:97-126` (modelBuilder, NewBuilder, SetId/SetLeaderId/SetMembers, Build, MustBuild); commit a243a5432 |
| 5 | Failing builder/accessor tests for the `party` field | DONE | `services/atlas-channel/atlas.com/channel/character/builder_test.go:187-233` (TestBuild_PartyDefaultsToZero, TestBuild_SetParty, TestCloneModel_PreservesParty); paired with Task 6 in commit f07062e94 |
| 6 | Add `party` field/builder/accessors to `character.Model` | DONE | `services/atlas-channel/atlas.com/channel/character/model.go:58` (field), `:265-274` (Party + InParty), `:331-333` (SetParty); `services/atlas-channel/atlas.com/channel/character/builder.go:58, 105, 145, 189` (builder field, clone, setter, build wiring); commit f07062e94 |
| 7 | Failing `PartyDecorator` tests | DONE | `services/atlas-channel/atlas.com/channel/character/processor_test.go:231-245` (TestProcessorImpl_PartyDecorator_NotInParty + TestProcessorImpl_PartyDecorator_InterfaceContract); paired with Task 8 in commit cae1b4038 |
| 8 | Implement `PartyDecorator` on interface, impl, and mock | DONE | `services/atlas-channel/atlas.com/channel/character/processor.go:31` (interface entry), `:156-167` (ProcessorImpl.PartyDecorator); `services/atlas-channel/atlas.com/channel/character/mock/processor.go:70` (mock pass-through); commit cae1b4038 |
| 9 | Update `kafka/consumer/character` HP-announce path | DONE | `services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go:98-110` — uses `cp.GetById(cp.PartyDecorator)`, short-circuits on `!cd.InParty()`, uses `model.FixedProvider(cd.Party())` instead of round-tripping `ByMemberIdProvider`, and the previous `Unable to announce…` debug log is gone; commit 2870bc281 |
| 10 | Update `kafka/consumer/map` HP-announce path | DONE | `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:225-238` — same pattern: `cp.GetById(cp.PartyDecorator)`, `!cd.InParty()` short-circuit guards both the announce-to-others and the announce-back-to-joiner loops; commit 9532d75f1 |
| 11 | Failing `TestAcceptEvent_NilTransactionId` | DONE | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/accept_event_test.go:181-197`; paired with Task 12 in commit 000103456 |
| 12 | Add `SkipReasonNilTransactionId` + AcceptEvent guard | DONE | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go:221` (constant); `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go:362-368` (guard at top of AcceptEvent, payload omits transaction_id, retains event_kind); commit 000103456 |
| 13 | Parallelise `ConsumeStandard` reads | DONE | `services/atlas-consumables/atlas.com/consumables/consumable/processor.go:212-237` — `model.NewGroup(ctx)` plus three `model.Submit` calls for character/field/consumable, then sequential `ConsumeItem` and `ApplyItemEffects`; commit cba63f7a6 |
| 14 | Parallelise `ConsumeTownScroll` reads | DONE | `services/atlas-consumables/atlas.com/consumables/consumable/processor.go:239-280` — two parallel reads (field + consumable), dependent `_map3.GetById(m.MapId())` correctly remains sequential after `Wait()`; commit 115ad7504 |
| 15 | Parallelise `ConsumeSummoningSack` reads | DONE | `services/atlas-consumables/atlas.com/consumables/consumable/processor.go:360-401` — two parallel reads (character + consumable), then sequential `position.GetInMap` (depends on c) and the spawn loop; commit 994389aca |
| 16 | Document why other Consume\* variants stay sequential | DONE | Comments at `processor.go:289-292` (ConsumePetFood — could be parallelised in follow-up, intentionally tight scope), `:327-328` (ConsumeCashPetFood — ci.Indexes() feeds the filter, not independent), `:410-411` (RequestScroll — chain depends on c.Inventory()); commit a9cdbd014 |
| 17 | Workspace-wide build + targeted tests + guard greps | DONE | All four module builds clean; all four targeted test suites PASS (see Build & Test Results); `grep "Unable to announce character.*health" services/atlas-channel/` returns nothing; `SkipReasonNilTransactionId` matches in all three saga files; `model.NewGroup`/`model.Submit` only appear in the three intended Consume\* variants (lines 219–222, 246–248, 366–368 of processor.go) and are absent from ConsumePetFood/ConsumeCashPetFood/RequestScroll. Verification-only — no commit, as planned. |

**Completion Rate:** 17/17 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. Every plan task has both code and verification evidence.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| libs/atlas-model | PASS | PASS | 4 packages tested (model, async, testutil, root); concurrency proof test exercises Group with real goroutines and passes within tolerance. |
| services/atlas-channel/atlas.com/channel | PASS | PASS | character, monster, movement, note, npc/shops, npc/shops/commodities, pet, reactor, session, socket/handler, socket/model, storage, transport/route, world all `ok`. No FAIL anywhere in the suite. |
| services/atlas-saga-orchestrator/atlas.com/saga-orchestrator | PASS | PASS | All 23 testable packages `ok`, including `saga` (208s — exercises `accept_event_test.go` and the new TestAcceptEvent_NilTransactionId) and the full set of consumer suites. |
| services/atlas-consumables/atlas.com/consumables | PASS | PASS | `consumable` and `map/character` both `ok`. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None — the plan landed as written. Worth flagging (informational, not blocking):

1. PRD criterion 7 ("p50 ≤ 200ms on root span") is explicitly verified out-of-band on a Tempo trace per the plan's own Acceptance Criteria Mapping table; the audit cannot validate that from the code alone. Confirm a fresh trace shows the expected p50 once this branch is merged and deployed.
2. The Task 16 comment on `ConsumePetFood` openly notes the two reads are independent and "could be parallelised in a follow-up." Consider filing this as a small follow-up task if the latency budget benefits, since the Group primitive is now available at zero marginal cost.

---

## Backend Guidelines Audit

- **Branch:** task-038-use-item-latency-cleanup
- **Base:** main (54d506c16)
- **HEAD:** a9cdbd014
- **Date:** 2026-04-30
- **Scope:** Changed Go packages only (DOM-* / SUB-* / SEC-* applied where applicable)
- **Build:** PASS for all four affected modules
- **Tests:** PASS for all four affected modules
- **Overall:** PASS

### Phase 1 — Build & Test (Objective Gate)

| Module | Build | Tests |
|--------|-------|-------|
| `libs/atlas-model` | PASS | PASS — `model`, `model/async`, `model/testutil` all green |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS — every package with tests green; `character/builder_test.go` covers party-field changes |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator` | PASS | PASS — `saga` (222.1s), `saga/mock`, `validation/mock` green |
| `services/atlas-consumables/atlas.com/consumables` | PASS | PASS — `consumable`, `map/character` green |

### Phase 2 — Domain Discovery (Changed Packages Only)

The diff touches a mix of layer types. Classification per Atlas conventions:

| Package | Type | Notes |
|---------|------|-------|
| `libs/atlas-model/model` | Support library | New primitive `Group`/`Submit[T]`/`Future[T]`. Not a service domain — DOM-* checklist N/A; bespoke concurrency review applied below. |
| `atlas-channel/character` | Domain (model only) | Has `model.go` + `builder.go`; no `entity.go`, no DB persistence (REST-backed). DOM rules around immutability/builders apply; persistence-related rules (entity, administrator, provider) N/A. |
| `atlas-channel/party` | Sub-domain (REST DTO) | Pre-existing remote-resource package. Adds `NewBuilder`. No `model.go` validation invariants exist in the package today. |
| `atlas-channel/character/mock` | Test support | Mock processor only. |
| `atlas-channel/kafka/consumer/character`, `.../map` | Kafka consumer | No DOM-* checklist; reviewed for layer compliance and concurrency safety. |
| `atlas-saga-orchestrator/saga` | Application service | Reviewed for behavioural change only (uuid.Nil guard). |
| `atlas-consumables/consumable` | Service processor | Reviewed for the parallel-read changes. |

### Phase 3 — Mechanical Checks on Changed Code

#### `atlas-channel/character` (DOM-* applicable subset)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go exists with fluent setters + Build() with validation | PASS | `services/atlas-channel/atlas.com/channel/character/builder.go:62` (`NewModelBuilder`), `:148` (`Build` with `ErrInvalidId` validation), `:194` (`MustBuild`). New `SetParty` setter at `:145`. |
| DOM-02 | Model immutability preserved with new field | PASS | `services/atlas-channel/atlas.com/channel/character/model.go:58` — `party party.Model` is a private field. `:265` getter `Party()`, `:273` `InParty()`, `:331` `SetParty(p)` returns a new Model via `CloneModel(m).SetParty(p).MustBuild()`. No public field mutation. |
| DOM-19 | Models flat / no JSON:API envelope | N/A | No `rest.go` change here. |
| DOM-20 | Table-driven / sufficient unit tests | PARTIAL/PASS | New tests in `services/atlas-channel/atlas.com/channel/character/builder_test.go` cover the party field: `:187` `TestBuild_PartyDefaultsToZero`, `:202` `TestBuild_SetParty`, `:219` `TestCloneModel_PreservesParty`. Not table-driven, but each is a focused assertion and matches the shape of the rest of that file. |
| DOM-builder-validation | `Build()` enforces invariants | PASS | `builder.go:148-150` — `b.id == 0` → `ErrInvalidId`. New party field has no invariant; defaulting to zero is documented behaviour (`model.go:269-272`). |
| Mock-interface-sync | Mock implements new method | PASS | `services/atlas-channel/atlas.com/channel/character/mock/processor.go:70` — `PartyDecorator` pass-through; compile-time assertions at `processor_test.go:228` (`var _ character.Processor = (*mock.MockProcessor)(nil)`) and `:243` (`var _ func(character.Model) character.Model = (mock.NewMockProcessor()).PartyDecorator`). |

#### `atlas-channel/character` Processor — Decorator pattern compliance

| Check | Status | Evidence |
|-------|--------|----------|
| `PartyDecorator` shape matches `InventoryDecorator` | PASS | `services/atlas-channel/atlas.com/channel/character/processor.go:68-74` (`InventoryDecorator`) vs. `:161-167` (`PartyDecorator`). Both: take `Model`, fetch via dedicated processor, on error return undecorated `m`, on success return `m.SetXxx(...)`. Doc-comment at `:156-160` explicitly notes this mirroring and warns callers about the undecorated case. |
| Method on `Processor` interface | PASS | `processor.go:31` — `PartyDecorator(m Model) Model` declared on interface. |
| Logger source — `FieldLogger`, not `*logrus.Logger` | PASS | `processor.go:51` — `NewProcessor(l logrus.FieldLogger, ctx context.Context)` unchanged. |

#### `atlas-channel/party` (sub-domain / REST DTO)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Builder added | PASS | `services/atlas-channel/atlas.com/channel/party/model.go:106` — `NewBuilder()`. Setters at `:110-112`. `Build()` at `:114-120`, `MustBuild` at `:124-126`. |
| Build() validation | WARN (non-blocking) | `model.go:114-120` — `Build()` is unconditional (no validation, returns no error). Deviates from `patterns-functional.md` ("Validation occurs in `Build()`"). However: `party.Model` is a JSON:API DTO produced by `Extract` over the REST response; the package has no DB write path that would need validated invariants, and the doc comment at `model.go:103-105` explicitly scopes the builder to test/in-process construction. Not a blocker, but worth noting if `party.Model` ever grows production-side construction. |

#### `atlas-channel/kafka/consumer/character` and `.../map`

| Check | Status | Evidence |
|-------|--------|----------|
| Solo short-circuit uses interface contract correctly | PASS | `kafka/consumer/character/consumer.go:99-103` — `cp.GetById(cp.PartyDecorator)(c.Id())`; on error or `!cd.InParty()` early-returns. Same shape at `kafka/consumer/map/consumer.go:226-230` (inside the `go func() {}` for HP announce). `Model.InParty()` semantics (`character/model.go:269-275`) are correctly used: a decorated solo character has `party.Id() == 0` → `InParty() == false`, so the cheap path is taken and no broadcast goroutine is launched. |
| `model.FixedProvider(cd.Party())` correct conversion | PASS | `libs/atlas-model/model/processor.go:291` — `FixedProvider[M](m M) Provider[M]` returns a closure yielding `m`. Used at `kafka/consumer/character/consumer.go:107` and `kafka/consumer/map/consumer.go:231` to feed `party.FilteredMemberProvider` (which expects `model.Provider[party.Model]`). Correct lazy-eval conversion of an already-loaded value. |
| `cp.PartyDecorator` as a method value | PASS | Go binds the receiver into the method value; the resulting `func(Model) Model` matches `model.Decorator[Model]` (`libs/atlas-model/model/processor.go:101`). Same idiom as `cp.InventoryDecorator` used elsewhere (e.g. `kafka/consumer/map/consumer.go:111`). |
| Error swallowing on broadcast | WARN (non-blocking) | `kafka/consumer/character/consumer.go:109` — `_ = session.NewProcessor(...).ForEachByCharacterId(...)`. Pattern matches the surrounding code in the same file (`kafka/consumer/map/consumer.go:234-237` has identical `_ =` patterns for the per-member announce). The `ai-guidance.md` testing-rules section is silent on consumer-side announce failures, and `anti-patterns.md` does not cover this. Consistent with established intra-service convention. Not blocking; if upgraded later, both call sites should be updated together. |
| Layering — handler/consumer calls processor, not provider | PASS | Both consumers use `character.NewProcessor(...).GetById(decorator)` — no direct calls to provider/DB. |

#### `atlas-saga-orchestrator/saga` (uuid.Nil guard)

| Check | Status | Evidence |
|-------|--------|----------|
| Guard placement before saga lookup | PASS | `saga/processor.go:362-368` — guard runs FIRST, before `p.GetById`. Avoids the `saga_not_found` red-herring log for uuid.Nil events. |
| New skip-reason constant defined | PASS | `saga/event_acceptance.go:221` — `SkipReasonNilTransactionId = "nil_transaction_id"` added to the centralized constant block alongside the existing five. |
| Behavioural test coverage | PASS | `saga/accept_event_test.go:181-197` (`TestAcceptEvent_NilTransactionId`) asserts: returns `false`; emits exactly one debug log; reason is `nil_transaction_id`; `transaction_id` field is *omitted* from log payload (which `processor.go:364-366` does correctly — only `event_kind` is set). |
| LogSkip pattern preserved | PASS | `saga/processor.go:364-366` uses `LogSkip(p.l, ..., SkipReasonNilTransactionId)` — same helper as the other four skip paths in `AcceptEvent`. |

#### `atlas-consumables/consumable` parallel reads

| Check | Status | Evidence |
|-------|--------|----------|
| `ConsumeStandard` parallelizes 3 truly-independent reads | PASS | `consumable/processor.go:212-237`. Three reads (character, map, consumable-data) all keyed only on `characterId`/`itemId`, no inter-dependency. Submitted via `model.Submit`, awaited at `:223` `pg.Wait()` with error checked. Successful path reads `.Get()` only after Wait returns nil. |
| `ConsumeTownScroll` parallelizes 2 reads, dependent third stays sequential | PASS | `consumable/processor.go:239-280`. Map + consumable-data run via `model.Group` (`:246-252`); the dependent `mapData.GetById(m.MapId())` correctly stays after `Wait()` (`:259-265`) with explanatory inline comment at `:258`. |
| `ConsumeSummoningSack` parallelizes 2 reads, dependent position read stays sequential | PASS | `consumable/processor.go:360-401`. Character + consumable-data run in parallel (`:366-372`); position read at `:375` is correctly noted as dependent on `c.MapId()/c.X()/c.Y()` in the comment at `:374`. |
| `ConsumePetFood` / `ConsumeCashPetFood` / `RequestScroll` documented as intentionally sequential | PASS | `consumable/processor.go:289-292` (`ConsumePetFood` — explicit out-of-scope note), `:327-328` (`ConsumeCashPetFood` — `ci.Indexes()` feeds the next filter, genuinely dependent), `:410-411` (`RequestScroll` — chain depends on `c.Inventory()`). Comments are informative, not aspirational — they correctly state why the reads cannot be parallelised. |
| Error from any submitted provider short-circuits with `ConsumeError` (correct cancellation path) | PASS | `consumable/processor.go:223-225`, `:249-251`, `:369-371` — every `pg.Wait()` error path routes through `p.ConsumeError(...)` which calls `cpp.CancelItemReservation` (`:193-198`), preserving the existing reservation-cancellation invariant. |
| `Future.Get()` only called after `Wait()` returns nil | PASS | All three call sites read `.Get()` (`:226`, `:252`, `:372`) only after the preceding `if err := pg.Wait(); err != nil { return ... }`. Matches the documented contract at `libs/atlas-model/model/parallel_group.go:18-22`. |

#### `libs/atlas-model/model/parallel_group.go` — concurrency review

The Group/Submit/Future primitive is the load-bearing piece of the change. The DOM checklist does not have a row for "thin errgroup wrapper", but the user explicitly asked for race-safety review:

| Check | Status | Evidence |
|-------|--------|----------|
| `f.value` write happens-before its read | PASS | `parallel_group.go:39-50` writes `f.value = v` inside the goroutine submitted to `errgroup.Group.Go`. `parallel_group.go:54` returns `g.g.Wait()`. `errgroup.Wait` (golang.org/x/sync) blocks until all goroutines submitted via `Go` complete; the goroutine return synchronizes-with the `Wait` return per Go memory model (it is the documented errgroup contract). Reads of `f.value` via `Future.Get()` (`:26`) by the calling goroutine happen-after `Wait()` returns. No race. |
| Behaviour on error documented | PASS | `parallel_group.go:18-22` — comment explicitly states `Get`'s behaviour is undefined when `Wait` returned an error. Test `TestGroup_OneProviderErrors` (`parallel_group_test.go:28-38`) verifies the error-propagation path; tests do *not* call `Get()` after errored `Wait()`, consistent with the contract. |
| Parallel execution actually achieved | PASS | `parallel_group_test.go:71-107` (`TestGroup_ConcurrencyProof`) — uses atomic counter + wall-clock assertion. Asserts `maxConcurrent >= 2` and `elapsed < 2*sleep - tolerance`. Empirically proves goroutines run in parallel, not serialized. |
| Context cancellation propagated | PASS | `parallel_group.go:31-34` — `errgroup.WithContext(ctx)` returns a child context cancelled when any submitted func errors. The child is returned to the caller and is the documented mechanism for "first error cancels all." Standard errgroup usage. |
| Type parameters on `Submit` (free function not method) — correctly noted | PASS | `parallel_group.go:36-38` — explanatory comment that Go forbids type-parameterised methods. Correct — generic methods are not supported, free function is the only option. |
| No goroutine leak on partial errors | PASS | `errgroup.Wait` waits for *all* submitted goroutines, even after one errors. Test `TestGroup_BothProvidersError` (`parallel_group_test.go:40-54`) covers this — both errors observed, no leak. Confirmed by inherited errgroup semantics. |
| Direct dependency declared | PASS | `libs/atlas-model/go.mod` and `libs/atlas-model/go.sum` updated to add `golang.org/x/sync` (commit 407658c3a). |

### Phase 4 — Security Review

Not applicable. None of the changed packages handle authentication, authorization, token issuance, redirects, or secret material. The saga-orchestrator change is a defensive null-UUID guard, not an auth change.

### Summary

#### Blocking (must fix)

None.

#### Non-Blocking (worth noting)

- `atlas-channel/party` `modelBuilder.Build()` (`services/atlas-channel/atlas.com/channel/party/model.go:114`) performs no validation. This is consistent with the package's role as a REST-DTO container and the doc comment at `:103-105` correctly scopes the builder to tests / in-process construction. If `party.Model` ever gains a production-side construction path (e.g. local cache materialization), `Build()` should grow id/leaderId invariants like `character/builder.go` does.
- HP-announce broadcast errors in both consumers (`kafka/consumer/character/consumer.go:109` and `kafka/consumer/map/consumer.go:234`) are silently swallowed via `_ =`. This matches the pre-existing pattern in the same files and the surrounding `go func()` blocks. Consider a service-wide cleanup pass if observability of broadcast failures is later required; not introduced by this branch.
- `ConsumePetFood` (`atlas-consumables/consumable/processor.go:282-319`) has two genuinely independent reads (`HungriestByOwnerProvider` + `cdp.GetById`) that are explicitly documented as a deferred parallelisation candidate. Acceptable scope discipline given the PRD §4.3 explicit target list, but it is now a known unparallelised hot path.

#### Notable Compliance Wins

- `Group`/`Submit`/`Future` primitive is documented, race-safe per errgroup semantics, has a concurrency-proving test, and the call sites correctly check `Wait()` errors before reading `.Get()`. Future-task cleanup of remaining `Consume*` variants will land trivially on this primitive.
- `PartyDecorator` is a faithful structural mirror of `InventoryDecorator` — same return-on-error semantics, interface placement, mock pass-through, and compile-time interface assertion in tests.
- `AcceptEvent` uuid.Nil guard is correctly placed (before `GetById`), uses the centralized `SkipReason*` constant block, and has a regression test that pins down the exact log shape (no `transaction_id` field) so a future drift would be caught.
