# Backend Audit — task-093 Mystic Door party-state reconciliation

- **Services:** atlas-doors (primary), atlas-parties, atlas-channel
- **Change range:** `7abb70e42..6cefdc004`
- **Guidelines Source:** backend-dev-guidelines skill (DOM-*/SUB-*/SEC-*)
- **Date:** 2026-06-18
- **Build:** PASS (`go build ./...` clean in atlas-doors; user-reported clean for all three modules + `docker buildx bake`)
- **Tests:** reported clean (`go test -race ./...`) across all three modules
- **Overall:** PASS

## Scope note

This change is **engine/consumer/producer logic only**. No new REST endpoints, no
new HTTP handlers, no DB writes, no GORM entities, no new Kafka topics. The
`door` package is a **registry-backed (Redis/in-memory) domain**, not a
GORM-backed one — it has `model.go`/`builder.go` but intentionally no
`entity.go`/`administrator.go`/`provider.go` DB layer. Therefore the DB- and
REST-write-oriented DOM checks (ToEntity/Make, TransformSlice, RegisterInputHandler,
administrator.go, DB tenant callbacks, error→HTTP-status mapping, JSON:API request
models) are **N/A for the changed code** and are recorded as such, not as fails.

## DOM / SUB checklist — applicable items

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go + fluent setters + Clone | PASS | `door/builder.go:32` `NewBuilder()`, `:34` `Clone(Model)`, fluent setters `:44-59`, `Build()` `:61`. (No `error` return / validation — consistent with pre-existing in-memory model; not regressed by this change.) |
| Immutable model | private fields + getters; mutation via builder | PASS | `door/model.go:13-30` all-private fields; getters `:32-48`; `Reslot` returns new instance via `Clone(...).Build()` `:50-52`. `reconcile.go` mutates only through `Clone(d).Set...().Build()` (`:94`, `:125`). |
| Processor Interface+Impl | `NewProcessor(l, ctx)` returns Impl | PASS | `door/processor.go:23` `type Processor interface`, `:60` `ProcessorImpl`, `:69` `NewProcessor(l logrus.FieldLogger, ctx context.Context)`. |
| DOM-06 | Processor accepts `FieldLogger` | PASS | `door/processor.go:69` param is `logrus.FieldLogger`; consumer `kafka/consumer/party/consumer.go:85` passes the consumer's `l`. atlas-parties `party/processor.go:57` likewise. |
| Multi-tenancy via context | tenant from ctx, not headers/params | PASS | `door/processor.go:71` `tenant.MustFromContext(ctx)`; consumer registers `consumer.TenantHeaderParser` `kafka/consumer/party/consumer.go:23`; all registry calls thread `p.ctx, p.t` (`reconcile.go:32,96,126`). No manual tenant field plumbing. |
| Kafka producer pattern | `producer.ProviderImpl(l)(ctx)` curried, context-aware | PASS | `door/processor.go:72-74` emit seam wraps `doorproducer.ProviderImpl(l)(ctx)(topic)(p)`; producers `door/producer.go:24,38,53` use `producer.SingleMessageProvider`. |
| DOM-12 | No `os.Getenv()` in handlers/engine | PASS | grep of `+` lines in range: zero `os.Getenv`. |
| DOM-13 | No cross-domain logic in handler; orchestration in engine layer | PASS | Consumer handlers (`consumer.go:75-157`) call only `party.NewProcessor(...).GetById` (read of sibling party state) + `enginedoor.ReconcileParty`; all door projection logic lives in `door/reconcile.go`. |
| DOM-21 | Reuse libs/atlas-constants types | PASS (strong) | `character.Id`, `_map.Id`, `point.X/Y`, `skill.Id`, `world.Id`, `channel.Id` used throughout `kafka.go:31-83`, `reconcile.go`, `slot.go`, `producer.go`. No re-declared id/classification types. `noMap = _map.EmptyMapId` (`town.go:9`) reuses the shared sentinel. |
| DOM-23 | Kafka topic naming/config | PASS (N/A-new) | No new topic constants introduced; all emits reuse pre-existing `EnvEventTopicDoorStatus` = `"EVENT_TOPIC_DOOR_STATUS"` (`door/kafka.go:13`). |
| DOM-24 | Kafka producer stubbed in emitting tests | PASS (new code) | New door tests inject a fake emit recorder via the processor's `emit` seam — no real producer (`reconcile_test.go:29` `&fakeEmit{}`, asserts on `em.values`). New parties test `TestLeaderLeaveDisbandEventIncludesLeader` uses `message.NewBuffer()` + buffer decode, `p: nil` (`processor_test.go` new block) — never reaches `producer.Produce`. New channel test stubs the package-level `broadcastDoorToEligible`/`announceTownPortalToParty` vars (`consumer_test.go:57,67`). See "Pre-existing" note below. |
| DOM-20 | Table-driven / named tests | PASS | New tests use named `t.Run`/dedicated `Test*` funcs with explicit fixtures; channel test is table-style (`consumer_test.go`). |
| Dead-code cleanup after refactor | five delta methods removed cleanly | PASS | `processor.go` diff deletes `JoinPartyDoor`/`ShowPartyDoorsToCharacter`/`HidePartyDoorsFromCharacter`/`LeavePartyDoor`/`DisbandPartyDoors` and `reslot.go`'s `ReslotParty`; grep across atlas-doors finds zero remaining references to any deleted symbol. |

## SUB checklist (action-event party consumer)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 | Business logic not in handler | PASS | Handlers `kafka/consumer/party/consumer.go:75-157` are thin: type-guard, fetch authoritative members, delegate to `ReconcileParty`. |
| SUB-04 | No manual JSON parsing | PASS | Consumer uses `message.AdaptHandler(message.PersistentConfig(...))` typed handlers (`consumer.go:32-46`); no `json.Unmarshal`/`io.ReadAll` in production consumer. (Test-only decode in `reconcile_test.go:20` and `processor_test.go` `findDisbandMembers` is acceptable.) |

## SEC checklist

Not auth/authz/token service — **N/A**. No secrets, JWT, or redirect handling in scope.

## Observations (non-blocking, not guideline violations)

1. **`ReconcileParty` is a free function taking `*ProcessorImpl`, not a `Processor`
   interface method.** `door/reconcile.go:17` + callers `consumer.go:85,105,122,136,154`
   reach into the concrete impl's private seams (`p.ctx`, `p.t`, `p.l`, `p.emit`).
   This is legal same-package Go and consistent with the package's deliberate
   field-injected test seam, but it means the party-projection orchestration is not
   expressible through the `Processor` interface and cannot be mocked at that boundary.
   Acceptable; flagged for awareness only.

2. **`builder.Build()` returns no error / performs no validation** (`builder.go:61`).
   DOM-01's strict reading wants `Build()` with validation. This is a pre-existing
   property of the door model (the model has no rejectable invariants) and is **not
   introduced or worsened** by this change, so it is not scored as a regression.

3. **Pre-existing DOM-24 smell, untouched by this change:** atlas-parties
   `processor_test.go:72-73` `setupTestWithProducer` builds a real
   `producer.ProviderImpl(...)` "pointing to a non-existent broker (will fail
   gracefully)" rather than the shared `producertest.InstallNoop()`. This is the exact
   ~42s-per-emit retry-backoff hazard DOM-24 calls out. **It is NOT in this change's
   diff** and the new parties test correctly avoids it (buffer-only). Recommend a
   follow-up to migrate `setupTestWithProducer` to `producertest`, but it does not
   block task-093.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix / awareness)
- Pre-existing: migrate atlas-parties `setupTestWithProducer` (`processor_test.go:72`) to shared `producertest.InstallNoop()` (DOM-24 hygiene; out of this change's scope).
- `ReconcileParty` bypasses the `Processor` interface (same-package private-field access) — acceptable, noted.

**Overall: PASS** — build/tests clean, refactor removed dead code without dangling
references, strong atlas-constants type reuse, correct context-based multi-tenancy,
and new tests stub the producer correctly.
