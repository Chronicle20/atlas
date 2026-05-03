# Reviewer Audit — task-051-status-cure-consumables

**Branch:** task-051-status-cure-consumables
**Base:** main
**Date:** 2026-05-03
**Reviewers dispatched:** plan-adherence-reviewer, backend-guidelines-reviewer
**Combined verdict:** READY_TO_MERGE

---

## 1. Plan Adherence

**Plan Path:** docs/tasks/task-051-status-cure-consumables/plan.md
**Audit Date:** 2026-05-03

### Executive Summary

All 11 plan tasks were faithfully implemented. Code matches the plan-specified bodies in every meaningful way (registry method, processor method, Kafka command shapes, consumer registration, producer, processor wrapper, helper, dispatch ordering, docs). Builds pass and all unit tests pass on both atlas-buffs and atlas-consumables. The single deviation from the plan — the addition of a `noopWriter` Kafka stub plus `kafkaProducer.ConfigWriterFactory` injection in `setupProcessorTest` (`processor_test.go`) — is necessary supporting infrastructure: without it, the plan-specified `assert.NoError(t, err)` on `processor.CancelByStatTypes` would fail because `message.Emit` would attempt a real TCP dial. The atlas-kafka library exposes both `Writer` and `ConfigWriterFactory` as public test seams (used by atlas-kafka's own `manager_test.go`), so the deviation is in-bounds, scoped, and applied only to the Processor test setup.

### Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Registry.CancelByStatTypes + 5 tests | DONE | `services/atlas-buffs/atlas.com/buffs/character/registry.go:192-236` (method body matches plan verbatim). Tests at `registry_test.go:325-413` (5 cases: EmptyTypes, NoMatch, SingleMatch, MultiMatch, UnknownCharacter — all match plan). Commit 4908b92da. |
| 2 | Processor.CancelByStatTypes + 4 tests | DONE (with justified deviation) | Interface: `processor.go:22`. Impl: `processor.go:83-108` (matches plan body verbatim). Tests at `processor_test.go:182-241` (EmptyTypes, NoMatch, MultiMatch, HolyShieldDoesNotBlockRemoval — all four). Commit 8df633d4d. **Deviation:** `processor_test.go:18-50` adds a `noopWriter` and `kafkaProducer.ConfigWriterFactory` injection in `setupProcessorTest` to prevent real Kafka dial; this is supporting infrastructure required to make the plan-specified `assert.NoError(t, err)` pass. The library uses the same pattern in `libs/atlas-kafka/producer/manager_test.go`. |
| 3 | atlas-buffs CANCEL_BY_TYPES constant + body | DONE | `kafka/message/character/kafka.go:17` constant and `:50-52` body type. Commit f0df7fba0. |
| 4 | atlas-buffs handleCancelByTypes consumer | DONE | Registration at `kafka/consumer/character/consumer.go:39-41`; handler `:81-89` matches plan body. Commit 173481c11. |
| 5 | atlas-consumables CANCEL_BY_TYPES constant + body | DONE | `kafka/message/character/buff/kafka.go:14` constant; `:44-46` body. Commit d8833b96e. |
| 6 | cancelByTypesCommandProvider | DONE | `character/buff/producer.go:57-72` with the plan's defensive copy of `types`. Commit 8fe14c03f. |
| 7 | buff.Processor.CancelByTypes | DONE | `character/buff/processor.go:37-39`, single-line wrapper as planned. Commit 20faa0703. |
| 8 | collectCureTypes helper + 5 tests | DONE | Helper at `consumable/processor.go:69-91` (verbatim plan body, fixed POISON/DARKNESS/WEAKEN/SEAL/CURSE order). Tests at `consumable/processor_test.go:261-319` (AntidotePot, HolyWater, AllCure, NonCureConsumable, ZeroFlagsIgnored). Commit 567dd85d5. |
| 9 | ApplyItemEffects cure-first dispatch | DONE | `consumable/processor.go:96-166` matches plan body verbatim. Section 1 cure precedes section 2 HP/MP precedes section 3 buffs; empty cureTypes skips dispatch. Commit 02ac005a6. |
| 10 | Documentation updates | DONE | `services/atlas-buffs/docs/domain.md:68,84,104` (Invariant note + Processor and Registry table rows). `services/atlas-buffs/docs/kafka.md:41,63-69` (Command Types row + CancelByTypesCommandBody section). `services/atlas-consumables/docs/domain.md:169,307` (ApplyItemEffects updated, CancelByTypes added). Commit 7faf9441b. |
| 11 | Final cross-service build + test | DONE | Verified by this audit: see Build & Test Results below. Cross-service grep confined to atlas-buffs and atlas-consumables. |

**Completion Rate:** 11/11 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

### Plan Deviation Evaluation

**`noopWriter` + `ConfigWriterFactory` injection in `processor_test.go`:**

- **Why introduced:** The plan tests at `processor_test.go:189-241` call `processor.Apply(...)` followed by assertions like `assert.Len(t, m.Buffs(), 1)`. `processor.Apply` ends with `message.Emit(...)`, which (in production code) flushes the buffer through the singleton `producer.GetManager()` — defaulting to a `kafka.Writer` that dials a real broker. In a unit-test environment with no Kafka, every `Apply` would error and the registry would never see the buff, breaking the plan's `assert.NoError`/`assert.Len` shape.
- **Why in-bounds:** The atlas-kafka library exposes `Writer` (interface) and `ConfigWriterFactory` (option) precisely for this use case — they are used by the library's own `libs/atlas-kafka/producer/manager_test.go`. No private internals are touched.
- **Scope:** Confined to `setupProcessorTest`; existing registry tests are unaffected (registry tests do not invoke `Emit`).
- **Verdict:** Necessary supporting infrastructure, not scope creep. The plan's TDD shape implicitly requires it; the planner appears to have overlooked the Kafka dial. Mark Task 2 DONE.

### Plan Adherence Verdict

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE
- **Action items:** None.

---

## 2. Backend Guidelines Audit

- **Services:** atlas-buffs, atlas-consumables
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-05-03
- **Build:** PASS
- **Tests:** All passed (atlas-buffs/character 10.966s, atlas-consumables/consumable + map/character)
- **Overall:** PASS

## Build & Test Results

```
cd services/atlas-buffs/atlas.com/buffs && go build ./...    -> ok (no output)
cd services/atlas-buffs/atlas.com/buffs && go test ./...     -> ok (atlas-buffs/character 10.966s; buff, buff/stat, tasks all ok)
cd services/atlas-consumables/atlas.com/consumables && go build ./...  -> ok (no output)
cd services/atlas-consumables/atlas.com/consumables && go test ./...   -> ok (consumable, map/character)
```

## Scope Classification

Files changed (14 total — code + docs):

| File | Package | Classification |
|------|---------|----------------|
| services/atlas-buffs/atlas.com/buffs/character/registry.go | atlas-buffs/character | Domain (has model.go) |
| services/atlas-buffs/atlas.com/buffs/character/processor.go | atlas-buffs/character | Domain |
| services/atlas-buffs/atlas.com/buffs/character/registry_test.go | atlas-buffs/character | Domain (test) |
| services/atlas-buffs/atlas.com/buffs/character/processor_test.go | atlas-buffs/character | Domain (test) |
| services/atlas-buffs/atlas.com/buffs/kafka/consumer/character/consumer.go | atlas-buffs/kafka adapter | Support (Kafka consumer) |
| services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go | atlas-buffs/kafka adapter | Support (Kafka schema) |
| services/atlas-consumables/atlas.com/consumables/character/buff/processor.go | atlas-consumables/character/buff | Support (no model.go, no resource.go — Kafka producer wrapper) |
| services/atlas-consumables/atlas.com/consumables/character/buff/producer.go | atlas-consumables/character/buff | Support |
| services/atlas-consumables/atlas.com/consumables/consumable/processor.go | atlas-consumables/consumable | Support (orchestration; no model.go, no resource.go) |
| services/atlas-consumables/atlas.com/consumables/consumable/processor_test.go | atlas-consumables/consumable | Support (test) |
| services/atlas-consumables/atlas.com/consumables/kafka/message/character/buff/kafka.go | atlas-consumables/kafka schema | Support (Kafka schema) |
| services/atlas-buffs/docs/* (2 files) | docs | N/A |
| services/atlas-consumables/docs/domain.md | docs | N/A |

No `resource.go`, `entity.go`, REST `rest.go`, `model.go`, `builder.go`, `administrator.go` were touched. The change is exclusively a Kafka command + processor/registry primitive plus a consumer orchestration tweak. DOM checklist items that pertain to REST handlers, REST DTOs, GORM entities, or domain-model construction (DOM-01..05, DOM-08..09, DOM-11..19) are not within the scope of this diff and are not regressed.

## DOM Checklist Results

### atlas-buffs/character (domain)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go exists | N/A (preexisting state) | No `builder.go` in `character/` — preexisting; not introduced or regressed by this task. Grep `services/atlas-buffs/atlas.com/buffs/character/` shows no Builder change. |
| DOM-02..05 | ToEntity / Make / Transform / TransformSlice | N/A | No `entity.go` and no `rest.go` for the buff `Model` (this is a Redis-backed registry, not GORM). Not in scope. |
| DOM-06 | Processor accepts `FieldLogger` | PASS | `services/atlas-buffs/atlas.com/buffs/character/processor.go:32` — `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`. Field on impl is `logrus.FieldLogger` (line 28). |
| DOM-07 | Handlers pass `d.Logger()` | N/A | No new HTTP handlers added. Kafka consumer at `services/atlas-buffs/atlas.com/buffs/kafka/consumer/character/consumer.go:81` receives the `logrus.FieldLogger` directly via the Kafka handler adapter — same pattern as preexisting handlers. |
| DOM-08 | POST/PATCH use `RegisterInputHandler` | N/A | No HTTP route changes. |
| DOM-09 | Transform errors handled | N/A | No `Transform()` call sites added. |
| DOM-10 | Test DB has tenant callbacks | N/A | This package is Redis-backed; no GORM. Tests use `setupTestRegistry` + `setupTestTenant` which already provide tenant context (`registry_test.go:323` calls `setupTestTenant(t)`). |
| DOM-11 | Providers use lazy evaluation | PASS | `services/atlas-buffs/atlas.com/buffs/character/producer.go:56-69` (`expiredStatusEventProvider`) returns `model.Provider[[]kafka.Message]`; emission goes through `message.Emit` + `buf.Put` at `processor.go:103-110`. |
| DOM-12 | No `os.Getenv()` in handlers | PASS | Grep of all changed files in atlas-buffs returns zero `os.Getenv` matches. |
| DOM-13 | No cross-domain logic in handlers | PASS | `handleCancelByTypes` (consumer.go:81) only calls `character.NewProcessor(l, ctx).CancelByStatTypes` — single domain hop. |
| DOM-14 | Handlers don't call providers directly | PASS | `handleCancelByTypes` (consumer.go:81) routes through processor. |
| DOM-15 | No direct entity creation in handlers | N/A | No GORM in this package. Registry mutations go through `r.characters.Get` / `Put` which is the established Redis primitive (`registry.go:202`, `:232`). |
| DOM-16 | `administrator.go` exists for writes | N/A | This package is Redis-backed; the `Registry` is the equivalent write boundary and is the only mutator. Processor delegates to `GetRegistry().CancelByStatTypes` (`processor.go:90`). |
| DOM-17 | Domain error → HTTP status mapping | N/A | No HTTP layer touched. |
| DOM-18 | JSON:API interface on REST models | N/A | No REST models touched. |
| DOM-19 | Request models flat | N/A | No request models touched. |
| DOM-20 | Table-driven tests | WARN (non-blocking) | New tests at `processor_test.go:182-241` and `registry_test.go:325-413` are individual `Test...` functions per case, not `tests := []struct{...}` table-driven. They mirror the existing test style in the same files (e.g., the preexisting `TestRegistry_ApplyAndCancel` is also a single function), so this is consistent with the local convention. Coverage of cases (empty, no-match, single match, multi-match, unknown character, Holy Shield bypass) is comprehensive. |
| DOM-21 | No duplication of atlas-constants types | PASS | `Types []string` on the wire is the right shape — `TemporaryStatType` is `type X string` already (`libs/atlas-constants/character/temporary_stat.go:3`), and the registry compares `c.Type()` (returned by `buff.stat.Model.Type()` as `string`) against the set. Producer-side at `services/atlas-consumables/atlas.com/consumables/consumable/processor.go:73-92` correctly uses `ts.TemporaryStatType*` constants and converts via `string(p.stat)`. The atlas-buffs side does not need to import the constant — its only contract is "compare strings to whatever atlas-buffs already stores in `stat.Model.Type()`", which is already a string field set from the original APPLY command bodies. No type duplication. No raw string-literal classification or numeric reinvention introduced. |

### atlas-buffs (Kafka adapter — consumer + schema)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Kafka command type discriminator | PASS | `consumer.go:82-84` early-returns when `c.Type != CommandTypeCancelByTypes`, mirroring `handleApply`/`handleCancel`/`handleCancelAll`. |
| Handler registered | PASS | `consumer.go:39-41` registers `handleCancelByTypes` on `EnvCommandTopic`. |
| Constant naming | PASS | `kafka.go:16` adds `CommandTypeCancelByTypes = "CANCEL_BY_TYPES"` (UPPER_SNAKE matching siblings). |
| Body type | PASS | `kafka.go:50-52` `CancelByTypesCommandBody{Types []string \`json:"types"\`}` — matches consumer-side at `services/atlas-consumables/atlas.com/consumables/kafka/message/character/buff/kafka.go:44-46`. |
| Log-and-continue on error | PASS | `consumer.go:86-88` logs error with character id and types, doesn't fail the consumer goroutine — matches established fire-and-forget pattern in same file. |

### atlas-buffs/character (test infra)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Kafka stub uses public API | PASS | `processor_test.go:26-37` injects via `kafkaProducer.GetManager(kafkaProducer.ConfigWriterFactory(...))` — the official test-injection seam. `ConfigWriterFactory` is the documented mechanism (`libs/atlas-kafka/producer/manager.go:22`). `t.Cleanup(kafkaProducer.ResetInstance)` resets the singleton between tests. |
| `noopWriter` correctness | PASS | `processor_test.go:20-24` implements `Topic()`, `WriteMessages()`, `Close()` — the full `Writer` interface defined at `libs/atlas-kafka/producer/producer.go:21`. |
| No real Kafka dial in tests | PASS | The injected factory is consulted before a real broker connection is attempted (singleton override). |
| Test discipline — Holy Shield bypass via direct registry | PASS | `processor_test.go:226-241` (the Holy-Shield test) deliberately bypasses `processor.Apply`'s immunity gate by calling `GetRegistry().Apply` directly. This is the correct approach: it sets up the impossible-but-must-be-recoverable state described in design D5 ("Holy Shield gates application, not cure"), which cannot be reached via the public API by construction. The comment at `processor_test.go:217-219` documents the intent. |

## SUB Checklist Results

### atlas-consumables/character/buff (Kafka producer wrapper — sub-domain support)

This package has neither `model.go` nor `resource.go`. It is a Kafka emitter wrapper exposing a typed processor over `producer.ProviderImpl`. Per the SUB checklist:

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 | Has processor / parent processor | PASS | `services/atlas-consumables/atlas.com/consumables/character/buff/processor.go:36-38` adds `(p *Processor) CancelByTypes` matching the existing `Apply`/`Cancel` style. |
| SUB-02 | Administrator for writes | N/A | This package emits Kafka commands, not DB writes. |
| SUB-03 | RegisterInputHandler[T] for POST | N/A | No HTTP route in this package. |
| SUB-04 | No manual JSON parsing | PASS | Producer uses `producer.SingleMessageProvider(key, value)` (`producer.go:71`) — typed, no `json.Unmarshal`/`io.ReadAll`. |
| Defensive copy of slice | PASS | `producer.go:59`: `Types: append([]string(nil), types...)`. The slice received by the provider is detached from the caller, so a downstream caller cannot mutate the message body via the original `types` reference. |

### atlas-consumables/consumable (orchestration — support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 | Logic in processor not handler | PASS | `collectCureTypes` and the cure-first dispatch live in `consumable/processor.go:73-92` and `:97-105`; no HTTP handler change. |
| SUB-04 | No manual JSON parsing | PASS | No `json.Unmarshal`/`io.ReadAll` in the diff. |
| Cure-before-HP/MP race avoidance | PASS | `processor.go:99-105` invokes `bp.CancelByTypes` before any `cp.ChangeHP`/`cp.ChangeMP` call (`:111-127`). The single per-character partition key (set by `producer.CreateKey(int(characterId))` at `services/atlas-consumables/atlas.com/consumables/character/buff/producer.go:61`) ensures the CANCEL_BY_TYPES command is consumed by atlas-buffs in send order, so any pending poison tick scheduled for that character is preempted. The inline comment at `processor.go:100-103` documents this. |
| Log-and-continue on cure failure | PASS | `processor.go:101-103` logs the error and proceeds with HP/MP recovery. This is intentionally lenient (matches the surrounding `_ = cp.ChangeHP(...)` style at `:112`, `:117`, `:120`, `:125`). The drinker still gets the heal even if the cure path fails — the alternative (abort heal) would cause worse player UX. |
| `collectCureTypes` deterministic ordering | PASS | `processor.go:78-85` declares the spec→stat pairs in a fixed slice, so output order is stable (POISON, DARKNESS, WEAKEN, SEAL, CURSE). Tests at `processor_test.go:271-289` and `:308-318` lock this order. |
| Zero-value cure spec ignored | PASS | `processor.go:87` requires `val > 0`. Test at `processor_test.go:312-318` covers the `Poison: 0, Curse: 1` case and asserts only `["CURSE"]` is emitted. |

## SEC Checklist Results

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SEC-01..04 | JWT/secret handling | N/A | No auth-service code touched. No new JWT, redirect, or secret-handling code in the diff. Grep of touched files for `jwt`, `Parse`, `os.Getenv` returns zero matches. |

## Summary

### Blocking (must fix)

- None. All build-and-test gates pass; every required check has a pass-or-not-applicable status with cited evidence.

### Non-Blocking (should fix / observations)

- **DOM-20 (table-driven tests)**: The new tests in `processor_test.go` and `registry_test.go` are per-case `TestX` functions rather than the table-driven `tests := []struct{...}` pattern recommended in the testing guide. They mirror the existing local convention in those same files, so accepting them is reasonable, but a future refactor to a single table-driven shape would tighten the suite.
- **Producer API surface (defensive observation, not a guideline violation)**: `Processor.CancelByStatTypes` on the buffs side accepts `[]string`. Callers must pass strings that exactly match `stat.Model.Type()` values written into the registry by `Apply`. There is no compile-time guarantee this matches the `ts.TemporaryStatType` value space — a typo on either end would silently produce a no-op cure rather than a build error. The atlas-consumables producer correctly uses the constants (`processor.go:78-85`); atlas-buffs intentionally stays string-typed because it has no compile-time visibility into the consumable-side enum and accepts arbitrary stat-type strings as a feature. Worth noting for future hardening (e.g. a shared `temporary_stat.IsValid(string)` validator), but not a violation of any current DOM/SUB/SEC rule.

### Overall

PASS. The change set introduces a single new Kafka command end-to-end (CANCEL_BY_TYPES) with mirrored constants, typed body, type-discriminated handler, defensive slice copy in the producer, deterministic cure-type collection, cure-before-heal ordering with documented race rationale, and unit-test coverage of empty/no-match/single/multi/unknown-character/Holy-Shield-bypass paths. Test-only Kafka stubbing uses the official `ConfigWriterFactory` seam with proper singleton reset. No DOM-21 type duplication. No HTTP, REST, or GORM surface modified, so the REST-oriented DOM checks are out of scope.
