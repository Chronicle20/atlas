# Backend Guidelines Audit — task-155 (Time Leap Cooldown Reset)

- **Scope:** commits `9391be6f2..477c5b8e9`
  - `services/atlas-skills/atlas.com/skills`: `skill/cooldown_registry.go`, `skill/processor.go`, `skill/mock/processor.go`, `kafka/message/skill/kafka.go`, `kafka/consumer/skill/consumer.go` (+ tests)
  - `services/atlas-channel/atlas.com/channel`: `character/skill/processor.go`, `data/skill/producer.go`, `kafka/message/skill/kafka.go`, `skill/handler/timeleap/` (new), `skill/handler/rectangle.go` (new), `skill/handler/heal/*` (refactor), `skill/handler/registrations/registrations.go`
- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/resources/*`
- **Date:** 2026-07-28
- **Build/Test Gate:** Reported clean at HEAD per task context (not re-run in full here); one specific test timed below to verify a checklist item empirically.
- **Overall:** NEEDS-WORK (1 Important finding; 2 Minor)

## Mindset note

Per-diff review of newly added code. Pre-existing structural gaps in files this diff only lightly touched (e.g. `atlas-channel/character/skill` has no `builder.go`/`entity.go`/`administrator.go` because it's a REST-client reader domain, not GORM-backed) are noted only where the diff's own additions interact with them; they are not re-litigated wholesale since they predate this branch.

---

## Findings

### [Important] DOM-24 — New test drives an unstubbed Kafka producer emit path; empirically 13.3s per run

**File:** `services/atlas-skills/atlas.com/skills/kafka/consumer/skill/consumer_internal_test.go:79`

```go
79:	handleCommandResetCooldowns(db)(logger, ctx, c)
```

`handleCommandResetCooldowns` (consumer.go) calls `skill.NewProcessor(l, ctx, db).ResetCooldownsAndEmit(...)`, which wraps `message.Emit(producer.ProviderImpl(p.l)(p.ctx))(...)` (`skill/processor.go:1285`). No test in this package (or anywhere in `atlas-skills`) installs `producertest.InstallNoop()` or an equivalent stub — confirmed by:

```
$ grep -rn "TestMain\|producertest\|InstallNoop" services/atlas-skills/atlas.com/skills/
(no matches)
```

Empirical proof this is not free:
```
$ go test ./kafka/consumer/skill/... -run TestHandleCommandResetCooldowns -v -count=1
=== RUN   TestHandleCommandResetCooldowns_TypeGuard
--- PASS: TestHandleCommandResetCooldowns_TypeGuard (0.01s)
=== RUN   TestHandleCommandResetCooldowns_ClearsRegistry
--- PASS: TestHandleCommandResetCooldowns_ClearsRegistry (13.28s)
```

The test's own comment (`consumer_internal_test.go:51-54`) rationalizes this instead of fixing it:
```go
51:	// Happy path: cooldowns cleared except the exclusion list. The Kafka emit
52:	// step fails in the test environment (no broker/env topic) AFTER the
53:	// registry mutation, which is the documented partial-success behavior —
54:	// assert on registry state only.
```

This is precisely the anti-pattern `testing-guide.md` ("Stubbing the Kafka Producer in Tests", pitfall #7) calls out: an unstubbed `*AndEmit()`/`message.Emit(...)` path burns ~10 retries × exponential backoff before the error surfaces. It is also a **novel** regression, not an existing convention being repeated: every pre-existing `atlas-skills` command-handler test (`consumer_test.go` — `TestHandleCommandRequestCreate_Success`, `TestHandleCommandRequestUpdate_Success`, etc.) deliberately calls the pure, buffer-only processor method (e.g. `processor.Create(mb)(...)`) instead of the handler function, specifically to avoid invoking the real producer. `consumer_internal_test.go` breaks that established avoidance idiom by calling the handler directly, which is the one path that reaches `*AndEmit`.

**Why it matters:** every future `go test ./...` run for this service now pays a fixed ~13s tax (worse under load/CI, up to the documented ~42s worst case), and the test asserts around a failure it never verifies — if the emit path silently changed to succeed or hang differently, this test would not catch it.

**Fix:** add a `TestMain` in `kafka/consumer/skill` (or the whole `atlas-skills` module, package by package) that calls `producertest.InstallNoop()` per the Pattern A recipe in `testing-guide.md`, and drop the "documented partial-success" comment in favor of actually asserting the emitted event once the stub makes emission deterministic (mirrors what `processor_test.go`'s `decodeCooldownExpiredEvents` helper already does for the buffered-only path).

---

### [Minor] No direct unit test for `character/skill.ProcessorImpl.ResetCooldowns`

**File:** `services/atlas-channel/atlas.com/channel/character/skill/processor.go:87-91`

```go
87: func (p *ProcessorImpl) ResetCooldowns(transactionId uuid.UUID, f field.Model, exceptSkillIds []uint32, sourceSkillId uint32) model.Operator[uint32] {
88: 	return func(characterId uint32) error {
89: 		return producer.ProviderImpl(p.l)(p.ctx)(skill2.EnvCommandTopic)(skill3.ResetCooldownsCommandProvider(transactionId, f.WorldId(), characterId, exceptSkillIds, sourceSkillId))
90: 	}
91: }
```

There is no `*_test.go` in `character/skill` exercising this method directly — it is only reached indirectly: (a) `data/skill/producer_test.go` tests the `ResetCooldownsCommandProvider` envelope in isolation, and (b) `timeleap/timeleap_test.go` substitutes a package-level `emitReset` seam that bypasses `ProcessorImpl.ResetCooldowns` entirely. Nothing in the diff verifies that `ProcessorImpl.ResetCooldowns` actually wires `transactionId`/`f.WorldId()`/`exceptSkillIds`/`sourceSkillId` into the provider call as written. This matches the sibling `ApplyCooldown` method's convention (also untested directly, `character/skill/processor.go:78-82`), so it is not a new pattern, but per the audit mandate ("prevalence is not compliance") it is still a coverage gap on a newly-added public interface method — `testing-guide.md`'s "Processors — test pure and AndEmit forms separately" focus area is not met for this method.

**Fix:** add a small test (table-driven or discrete) in `character/skill` that calls `NewProcessor(...).ResetCooldowns(...)(characterId)` against a captured/faked producer and asserts the emitted `Command[ResetCooldownsBody]` fields, the same way `data/skill/producer_test.go` already does one layer down.

---

### [Minor] DOM-20 — New tests are not table-driven

**Files:**
- `services/atlas-skills/atlas.com/skills/skill/cooldown_registry_test.go:1095-1156` (4 new discrete `Test*` functions: `TestGetAllForCharacter_ReturnsOnlyCharacterEntries`, `TestGetAllForCharacter_PrefixSafety`, `TestGetAllForCharacter_UnknownCharacter_Empty`, `TestGetAllForCharacter_SkipsMalformedSuffixes`)
- `services/atlas-skills/atlas.com/skills/skill/processor_test.go:1371-1502` (4 new discrete `Test*` functions covering `ResetCooldowns`)
- `services/atlas-channel/atlas.com/channel/skill/handler/timeleap/timeleap_test.go:716-796` (6 new discrete `Test*` functions)

None use the `tests := []struct{...}{...}` + `t.Run` table-driven shape DOM-20 calls for. All of the pre-existing tests in these same files also use discrete functions rather than tables, so this is a file-wide/service-wide idiom, not something task-155 invented — but per the audit's stated rule, prevalence does not convert a documented-checklist deviation into a pass. Flagged as Minor because coverage itself is thorough (happy path, no-op, prefix-safety, malformed-suffix, all-excepted, party-fan-out, and error-continuation cases are all present) — the deviation is presentational, not a coverage gap.

**Fix (optional, non-blocking):** convert to `tests := []struct{name string; ...}{...}` + `t.Run(tt.name, ...)` if/when this file is next touched.

---

## Checklist Results

### `services/atlas-skills/atlas.com/skills/skill` (domain package, has `model.go`) — only new surface audited

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor accepts `FieldLogger` | PASS | `skill/processor.go` — `NewProcessor(l logrus.FieldLogger, ctx, db)` unchanged by diff, still `logrus.FieldLogger` |
| DOM-09 | Transform errors handled | N/A | No `resource.go`/`Transform` touched by this diff |
| DOM-10 | Test DB has tenant callbacks | PASS | `test.SetupTestDB` (used by all new tests) registers callbacks — `services/atlas-skills/atlas.com/skills/test/database.go:31` |
| DOM-15 | No direct entity creation in handlers | PASS | `handleCommandResetCooldowns` (`kafka/consumer/skill/consumer.go:839-852`) calls only `skill.NewProcessor(...).ResetCooldownsAndEmit(...)`, no `db.Create`/`db.Save` |
| DOM-16 | administrator.go for writes | N/A (correctly) | `ResetCooldowns` is documented registry-only, never touches DB (`skill/processor.go:1252-1255` comment) — no `administrator.go` change needed |
| DOM-21 | No duplication of atlas-constants types | PASS | `skill2.BuccaneerTimeLeapId` used (`atlas-channel/.../timeleap.go:516,584-585`), defined once at `libs/atlas-constants/skill/constants.go:3217`; `world.Id` used from `libs/atlas-constants/world`, not redefined |
| DOM-24 | Kafka producer stubbed in tests that emit | **FAIL** | `kafka/consumer/skill/consumer_internal_test.go:79` — see finding above; measured 13.28s |
| FILE-01 | Processor logic in processor.go | PASS | `ResetCooldowns`/`ResetCooldownsAndEmit` both added to `skill/processor.go` (interface at lines ~1221-1229, impl at 1256-1300) |
| FILE-06 | No package-named catch-all file | PASS | No new `skill.go`; new logic lives in `processor.go`/`cooldown_registry.go` |

### `services/atlas-skills/atlas.com/skills/kafka/message/skill` and `kafka/consumer/skill` (support packages)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Handler logic stays out of message/wire package | PASS | `kafka/message/skill/kafka.go` only adds `CommandTypeResetCooldowns` const + `ResetCooldownsBody` struct — no logic |
| DOM-08/SUB-03 | N/A (Kafka handler, not REST) | N/A | This package has no `resource.go` |

### `services/atlas-channel/atlas.com/channel/character/skill` (domain package, has `model.go`, REST-client reader — pre-existing, diff only touches `processor.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor logic in processor.go | PASS | `ResetCooldowns` added to `processor.go` interface (line 62) and impl (lines 87-91) |
| DOM-21 | No duplication of atlas-constants | PASS | Uses `field.Model`, `skill.Id`, `world.Id` (via `f.WorldId()`) from atlas-constants, no local redefinition |
| Testing-guide | Processor method tested | Minor FAIL | See finding above — no direct test of `ResetCooldowns` |

### `services/atlas-channel/atlas.com/channel/data/skill` (support/producer package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01/producer.go | Kafka message creation in producer.go | PASS | `ResetCooldownsCommandProvider` added to `producer.go`, not a catch-all file |
| Test | Provider encode/decode test present | PASS | `producer_test.go` new, round-trips the envelope via `json.Unmarshal` |
| DOM-24 | No unstubbed emit | PASS (N/A) | Test calls the pure `model.Provider[[]kafka.Message]` function directly — no network I/O, no stub needed |

### `services/atlas-channel/atlas.com/channel/skill/handler` (support package — shared helpers)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-06 | No catch-all file | PASS | `rectangle.go` is single-purpose (dedupe-warn helper), hoisted verbatim out of `heal/recipients.go` |
| Refactor correctness | `heal.go` uses the hoisted helper | PASS | `skill/handler/heal/heal.go:284` calls `channelhandler.WarnIfMissingRectangle(...)` |

### `services/atlas-channel/atlas.com/channel/skill/handler/timeleap` (new support package — per-skill handler, no `model.go`/`resource.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01/02/03 | No Processor/RestModel/requests collapsed into this file | PASS | `timeleap.go` defines only `Apply` + three test-seam vars (`loadCaster`, `selectParty`, `emitReset`); all actual Processor/RestModel/requests logic lives in `character`, `character/skill`, and `data/skill` respectively — correctly delegated, not duplicated |
| DOM-14 | Handler doesn't call providers directly | PASS | `loadCaster` calls `character.NewProcessor(l, ctx).GetById()`, not a provider |
| DOM-21 | No duplication of atlas-constants | PASS | `skill2.BuccaneerTimeLeapId` (`timeleap.go:516`), `skill2.Id(info.SkillId())` (`timeleap.go:577`) |
| DOM-25 | Client wire values config-resolved | N/A (correctly) | Feature adds no new opcode/dispatcher byte; reuses existing `USE_SKILL` decode path and `COOLDOWN_EXPIRED` writer, per design doc — confirmed no new byte literals in `timeleap.go` |
| DOM-26 | No bare `go` statements | PASS | `grep -n '^\s*go ' timeleap.go` — no matches |
| Registration | Blank-import registers handler | PASS | `registrations.go:476` adds `_ "atlas-channel/skill/handler/timeleap"`; `timeleap.go:515-517` `init()` calls `channelhandler.Register(...)`; verified by `TestTimeLeapRegistered` (`timeleap_test.go:792-796`) |
| Test coverage | Solo/party/failure/degrade paths covered | PASS | 6 tests cover solo cast, party fan-out, caster-load failure, missing-rectangle fallback, per-recipient emission-failure continuation, and registration |

---

## Summary

### Blocking (must fix)
- **DOM-24**: `services/atlas-skills/atlas.com/skills/kafka/consumer/skill/consumer_internal_test.go:79` drives an unstubbed Kafka producer emit path (measured 13.28s for one test); no `producertest.InstallNoop()` anywhere in the service. Add a `TestMain` stub per `testing-guide.md`.

### Non-Blocking (should fix)
- No direct unit test of `character/skill.ProcessorImpl.ResetCooldowns` (`services/atlas-channel/atlas.com/channel/character/skill/processor.go:87-91`) — only exercised via test seams/lower-layer tests.
- New tests in `cooldown_registry_test.go`, `processor_test.go` (atlas-skills), and `timeleap_test.go` (atlas-channel) use discrete `Test*` functions rather than the table-driven `tests := []struct{...}` + `t.Run` shape called for by DOM-20 (matches surrounding file convention, coverage itself is thorough).
