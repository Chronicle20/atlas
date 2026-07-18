# Backend Audit — task-173-mts-map-presence

- **Service Path:** services/atlas-channel/atlas.com/channel
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-07-17
- **Scope:** Diff-scoped audit, base `c9490b724` → head `1e6c398125` (design-doc commit + fix commit)
- **Build:** PASS
- **Tests:** all packages PASS (`go test ./...`, `go test -race ./...`, `go vet ./...` all clean)
- **Overall:** NEEDS-WORK (one Important, one Minor finding; no build/test failure)

## Build & Test Results

```
$ go build ./...                     -> clean, no output
$ go vet ./...                       -> clean, no output
$ go test ./... -count=1             -> all packages ok, zero FAIL lines
$ go test -race ./... -count=1       -> all packages ok, zero FAIL lines
$ go test ./session/... -race -v     -> 34 tests, all PASS (includes the 2 new tests)
```
Verified from `services/atlas-channel/atlas.com/channel` inside the worktree.

## Changed Files

- `services/atlas-channel/atlas.com/channel/session/processor.go` (+7/-2)
- `services/atlas-channel/atlas.com/channel/session/processor_test.go` (+47)
- `docs/tasks/task-173-mts-map-presence/design.md` (new)

## Focus Area 1 — Predicate correctness and blast radius

| Check | Status | Evidence |
|---|---|---|
| Predicate change is exactly the described one-line fix | PASS | `session/processor.go:111` — `if s.CharacterId() != 0 && s.CashScene() == CashSceneNone && s.Field().Equals(f) {`. `git diff 634cfed9a..1e6c398125 -- .../processor.go` shows only this predicate + doc-comment change. |
| `CashSceneNone`/`CashSceneCashShop`/`CashSceneMts` values match design.md's citation | PASS | `session/model.go:19-21` — `CashSceneNone byte = 0`, `CashSceneCashShop byte = 1`, `CashSceneMts byte = 2`. Matches design.md lines 55-56 verbatim. |
| MTS entry sets `CashSceneMts` on the session (design.md mts_entry.go:94) | PASS | `socket/handler/mts_entry.go:94` — `_ = session.NewProcessor(l, ctx).SetCashScene(s.SessionId(), session.CashSceneMts)`. |
| MTS/cash-shop-return resets to `CashSceneNone` (design.md map_change.go:39) | PASS | `socket/handler/map_change.go:39` — `_ = session.NewProcessor(l, ctx).SetCashScene(s.SessionId(), session.CashSceneNone)`, inside the `p.CashShopReturn()` branch. |
| Cash-shop session is destroyed on migrate (design.md socket/init.go:54) | PASS | `socket/init.go:49-54` — `socket.SetDestroyer(func(sessionId uuid.UUID) { ... sp.DestroyByIdWithSpan(sessionId) })`. |
| `session.InFieldModelProvider` has exactly one production consumer chain, as design.md claims | PASS | `grep -rn "InFieldModelProvider"` across the module: the only non-test, non-mock production caller is `map/processor.go:44` (`CharacterIdsInMapModelProvider`), which itself is called only from `map/processor.go:53,58` (`ForSessionsInSessionsMap`/`ForSessionsInMap`). Every other hit is a test file, a mock, or an unrelated same-named method on `door`/`merchant` domain types (different `Model`, no `CashScene`). |
| **Sibling predicate not updated — blast-radius gap** | **FAIL (Important)** | `session/processor.go:121-132` `InMapAllInstancesModelProvider` still uses the pre-fix condition `s.CharacterId() != 0 && s.WorldId() == worldId && s.ChannelId() == channelId && s.MapId() == mapId` with **no** `CashScene()` guard. It feeds `map/processor.go:61-62` `CharacterIdsInMapAllInstancesModelProvider` → `map/processor.go:88-90` `ForSessionsInMapAllInstances`, which is consumed in production by `kafka/consumer/route/consumer.go:70` and `:91` (transport-arrival/departure broadcasts, `fieldcb.FieldTransportStateWriter`). Per the design's own stated invariant ("An MTS character is not visually in the map and should neither be seen nor receive field broadcasts", design.md lines 94-95), an MTS-scened (or lingering cash-shop-scened) session sitting in that map still receives the transport-state announcement through this path — the same class of "physically absent but still queried as present" defect the task set out to fix, left unaddressed and never mentioned in design.md's Scope section (which only discusses `InFieldModelProvider`). This does not reproduce the reported ghost-spawn symptom (a different consumer, a broadcast not a snapshot), so it is not blocking on its own, but the design's claim of a "fully understood" blast radius is incomplete: it examined `InFieldModelProvider`'s single consumer chain but never mentions this sibling method's existence or its own separate consumer chain. |

## Focus Area 2 — Consequence for other field-scoped broadcasts (chat/weather/field-effects)

| Check | Status | Evidence |
|---|---|---|
| Map chat routes through the now-filtered path | PASS | `kafka/consumer/message/consumer.go:98` — `_map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(e.MapId, e.Instance), showGeneralChatForSession(...))`. `ForSessionsInMap` (`map/processor.go:57-58`) calls `CharacterIdsInMapModelProvider` → `session.InFieldModelProvider` (fixed). |
| Weather start/end and BGM route through the now-filtered path | PASS | `kafka/consumer/map/consumer.go:719,738,807` — all three call `_map.NewProcessor(l, ctx).ForSessionsInMap(f, ...)`. |
| Door/merchant/monster/drop/reactor/pet/mount/summon/consumable/party_quest/asset/chalkboard/skill-recipients broadcasts route through the now-filtered path | PASS | All identified call sites (`kafka/consumer/door/consumer.go:84`, `merchant/consumer.go:151,181,242,247,643`, `monster/consumer.go:143,185,228,253,272,386,394`, `drop/consumer.go:105,126,145,202`, `reactor/consumer.go:84,103,130`, `pet/consumer.go:224,296,334`, `mount/consumer.go:64`, `summon/consumer.go:85,125,198`, `consumable/consumer.go:150,230`, `party_quest/consumer.go:78`, `asset/consumer.go:360`, `chalkboard/consumer.go:66,85`, `skill/handler/recipients.go:63`, `mist/consumer.go:62,70`) all call `_map.NewProcessor(l, ctx).ForSessionsInMap(...)`, which resolves through the fixed `InFieldModelProvider`. |
| This is the correct, intended consequence, not a regression | PASS | design.md lines 89-95 explicitly states excluding cash-scened sessions from these broadcasts is deliberate and symmetric with the cash-shop precedent (a cash-shop session already misses these broadcasts today because it is absent from the registry). Verified this is the *only* other class of consumer on the fixed path (Focus Area 1's single-consumer-chain check), so no broadcast type was missed on the "fixed" side. |
| `ForSessionsInMapAllInstances` (route/transport broadcasts) does **not** get the same treatment | See Focus Area 1 Important finding — `session/processor.go:121-132`, `map/processor.go:88-90`, `kafka/consumer/route/consumer.go:70,91`. |

## Focus Area 3 — Test quality

| ID | Check | Status | Evidence |
|---|---|---|---|
| — | Tests exercise the actual predicate (not incidental) | PASS | `session/processor_test.go:870-891` (`TestInFieldModelProvider_ExcludesMtsScenedSessions`) and `:894-915` (`TestInFieldModelProvider_ExcludesCashShopScenedSessions`) each register two same-field sessions, scene one of them, and assert `len(got) == 1`. |
| — | Red-before/green-after | PASS (verified by diff inspection, not by reverting source — instructed not to modify source) | The only predicate difference the two new tests exercise is `s.CashScene() == CashSceneNone` (`session/processor.go:111`, added by `git diff 634cfed9a..1e6c398125`). Without that clause, both sessions in each new test satisfy `CharacterId() != 0 && Field().Equals(f)`, so `InFieldModelProvider` would return `len(got) == 2`, failing the `len(got) != 1` assertion in both tests. No other change in the diff could make these tests pass; the fix is necessary and sufficient for them. |
| — | Builder pattern / no test-only constructors | PASS | Both new tests use the pre-existing `addFieldSession` helper (`session/processor_test.go:727-738`, unchanged by this diff — confirmed via `git diff`, present before the task-173 commits) built on `field.NewBuilder(...)` (the project Builder) plus already-public `session.NewSession`/`session.AddSessionToRegistry`/`p.SetCharacterId`/`p.SetField`/`p.SetCashScene`. No new `*_testhelpers.go` file or test-only constructor was introduced. |
| DOM-20 | Table-driven tests (`tests := []struct{...}` + `t.Run`) | **FAIL (Minor)** | `session/processor_test.go:870-915` — both new tests are individually-named functions with hard-coded scene/expectation values, not a `tests := []struct{...}` table with `t.Run` subtests. This is a real deviation from the DOM-20 pass criteria. Graded on its own terms per the "prevalence is not compliance" rule (not excused because the rest of the file already uses one-function-per-case). Rated Minor rather than Important because this is a test-organization preference, not a File-Responsibilities/structural violation — the two cases are simple enough (2 assertions each) that the deviation carries low risk, and no guideline text elevates DOM-20 to a default-Important severity the way file-placement violations are. |
| — | Regression tests still pass | PASS | Pre-existing `TestInFieldModelProvider_ExactMatchIncludingInstance`, `_WorldChannelDiscrimination`, `_ExcludesCharacterlessSessions`, `_ExcludesOtherTenant`, `_EmptyFieldNoError` and `map/processor_test.go`'s `TestCharacterIdsInMapModelProvider_DedupsCharacterIds`/`TestOtherCharacterIdsInMapModelProvider_ExcludesReference` (default `CashSceneNone` sessions) all pass — confirmed via `go test -race ./... -count=1`, zero FAIL lines. |

## Focus Area 4 — DOM-* guideline violations in the changed Go

| ID | Check | Status | Evidence |
|---|---|---|---|
| FILE-01 | Processor method lives in `processor.go` | PASS | Change is confined to an existing method body inside `session/processor.go`; no relocation. |
| DOM-06 | Processor constructor accepts `logrus.FieldLogger` | PASS (pre-existing, unaffected by diff) | `session/processor.go:67` — `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`. |
| DOM-11 | Providers use lazy evaluation | PASS | `session/processor.go:106-117` — `InFieldModelProvider` still returns a `func() ([]Model, error)` closure over the registry read; no eager evaluation was introduced. |
| DOM-12 | No `os.Getenv()` in handlers | N/A | Diff touches no `resource.go`/handler file. |
| DOM-24 | Kafka producer stubbed in tests that emit | N/A | Neither new test calls an emit path (`AndEmit`, `message.Emit`, `producer.Produce`); `InFieldModelProvider` is a pure in-memory registry read. |
| DOM-21 | No duplication of atlas-constants types | PASS | No new type/const introduced; `CashSceneNone`/`CashSceneMts` are pre-existing session-local scene-state bytes, not an atlas-constants concept. |
| DOM-26 | Goroutines via `routine.Go` | N/A | No goroutine introduced by this diff. |

## Summary

### Blocking (must fix)
None — build and tests are clean, and no Critical-severity finding exists.

### Important (should fix before merge)
- **Blast-radius gap:** `session/processor.go:121-132` (`InMapAllInstancesModelProvider`) was not given the same `CashScene() == CashSceneNone` guard as `InFieldModelProvider`. It feeds `map/processor.go:88-90` (`ForSessionsInMapAllInstances`), consumed by `kafka/consumer/route/consumer.go:70,91` for transport arrival/departure broadcasts — an MTS-scened session in that map still receives `fieldcb.FieldTransportStateWriter` announcements, contradicting the design's own stated invariant that cash-scened sessions should receive no field broadcasts. Not the reported symptom and not build/test-breaking, but an unaddressed and undocumented instance of the same defect class; design.md's Scope section should either fix it or explicitly record it as an accepted, deliberate exception.

### Non-Blocking (should fix)
- **DOM-20:** `session/processor_test.go:870-915` — the two new tests are not table-driven (`tests := []struct{...}` + `t.Run`), per the Domain Package Checklist's testing pass criteria.

### Notable positives (not required, cited for completeness)
- Every design.md source citation checked against this diff and the surrounding codebase (mts_entry.go:94, map_change.go:39, socket/init.go:54, model.go:19-21, the "single production consumer chain" claim for `InFieldModelProvider`) was verified accurate.
- All other field-scoped broadcast consumers (chat, weather, BGM, door, merchant, monster, drop, reactor, pet, mount, summon, consumable, party_quest, asset, chalkboard, skill recipients, mist) correctly inherit the fix through the shared `ForSessionsInMap` → `CharacterIdsInMapModelProvider` → `InFieldModelProvider` chain — this is a deliberate and correct consequence per design.md, not a regression.
