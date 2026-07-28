# Backend Guidelines Audit — task-152 (Mortal Blow)

- **Service Paths:** `services/atlas-channel/atlas.com/channel`, `services/atlas-monsters/atlas.com/monsters`
- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/resources/*`
- **Date:** 2026-07-27
- **Scope:** Changed Go packages only (range `c09233e3f..HEAD`), per parent task brief — not a full-service sweep.
- **Build:** PASS (both modules, `go build ./...`, re-verified)
- **Vet:** PASS (both modules, `go vet ./...`, re-verified)
- **Tests:** PASS (`atlas-channel`: socket/handler, monster, data/skill/effect, kafka/message/monster all `ok`; `atlas-monsters`: monster, kafka/consumer/monster all `ok`)
- **Overall:** NEEDS-WORK (one FAIL: DOM-20 on atlas-monsters `kill_test.go`; see Summary)

## Build & Test Results

```
$ cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./...
(clean)
$ go test ./socket/handler/... ./monster/... ./data/skill/effect/... ./kafka/message/monster/... -count=1
ok  	atlas-channel/socket/handler	0.764s
ok  	atlas-channel/monster	0.074s
ok  	atlas-channel/data/skill/effect	0.005s
ok  	atlas-channel/kafka/message/monster	0.005s

$ cd services/atlas-monsters/atlas.com/monsters && go build ./... && go vet ./...
(clean)
$ go test ./monster/... ./kafka/consumer/monster/... -count=1
ok  	atlas-monsters/monster	13.955s
ok  	atlas-monsters/kafka/consumer/monster	0.013s
```

`tools/goroutine-guard.sh` re-run: no bare `go` statements in the diff (script output confirms scan of all `libs/*`, no hits in the changed service packages).

## Domain Package Checklist — `atlas-channel` / `monster` (has `model.go`, `builder.go`, `entity`-free registry-mirror domain)

Scope limited to the `Kill` addition; pre-existing symbols in this package were not re-audited.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor accepts `FieldLogger` | PASS | `services/atlas-channel/atlas.com/channel/monster/processor.go:40` — `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` (pre-existing, unchanged; `Kill` reuses it). |
| FILE-01 | `Processor` interface + impl in `processor.go` | PASS | `services/atlas-channel/atlas.com/channel/monster/processor.go:145` — `func (p *ProcessorImpl) Kill(f field.Model, monsterId uint32, characterId uint32, skillId uint32) error` in `processor.go`, interface entry co-located. |
| FILE-05 | Kafka message creation in `producer.go` | PASS | `services/atlas-channel/atlas.com/channel/monster/producer.go:175` — `func KillCommandProvider(...)`. |
| — | Mock kept in sync (testing-guide Interface Change Workflow) | PASS | `services/atlas-channel/atlas.com/channel/monster/mock/processor.go:25` (`KillFunc` field) and `:128-134` (`Kill` method, nil-check pattern matching sibling `DrainMp` mock). |
| — | `KillCommandBody` / `CommandTypeKill` placed in message-contract package | PASS | `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go:20` (const) and `:94` (struct) — same file as sibling `DrainMpCommandBody`. |

## File Responsibilities Checklist — `atlas-channel` / `socket/handler` (support package, packet-handler architecture — no `model.go`)

This package does not follow the DDD `model.go`/`entity.go`/`processor.go` layering; it is the existing socket packet-handler architecture where per-passive proc logic (pickpocket, drain, sacrifice, and now Mortal Blow) is co-located in `character_attack_common.go` alongside `processAttack`. This is the file's established, single-purpose role (attack-pipeline helper functions), not a `<pkg>.go`-style catch-all bundling unrelated responsibilities (Processor+RestModel+requests) — no FILE-06 violation.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | No duplication of atlas-constants types | PASS | `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:406-407` — `skill3.Id(skillId) == skill3.RangerMortalBlowId \|\| skill3.Id(skillId) == skill3.SniperMortalBlowId`, both defined in `libs/atlas-constants/skill/constants.go:3102,3124`. No new numeric literal skill-id classification introduced. |
| DOM-12 | No `os.Getenv()` in handler code | PASS | `grep -c os.Getenv character_attack_common.go` → 0 in the diff. |
| — | Errors logged + swallowed, pipeline not aborted (FR-5) | PASS | `character_attack_common.go:472-476` (`mortalBlowTryProc`) — snapshot-fetch error: `l.WithError(err).Debugf(...); return`; emit error: `l.WithError(err).Errorf(...)` with no propagation, matching sibling `pickPocketTryProc`/`drainTryHeal` swallow pattern. |
| — | Gating: proc only fires after damage/status apply, never for reflected/status-only entries | PASS (test-verified) | `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_mortal_blow_test.go:294-324` (`TestProcessDamageInfoEntry_ReflectedEntrySkipsOnDamageApplied`) and `:274-289` (status-only entry) both assert `onDamageApplied` — and therefore Mortal Blow — never fires. |
| — | Wiring uses caller-scoped `l`, no `logrus.StandardLogger()` | PASS | `character_attack_common.go:773` — `mortalBlowTryProc(l, mortalBlowDeps{...})` inside `processAttack(l)`, same `l` threaded to every sibling proc call at `:759-771`. |

Test file `character_attack_mortal_blow_test.go` is table-driven for the three pure predicate functions (`TestMortalBlowEligible:25-53`, `TestMortalBlowKillRoll:55-76`, `TestIsMortalBlowAttack:78-100`, all using `cases := []struct{...}` + `t.Run`). This package has no `model.go`, so the DOM-20 domain-checklist row does not formally apply here (Domain Package Checklist header: "every domain with `model.go`") — noted for completeness, not scored.

## Domain Package Checklist — `atlas-monsters` / `monster` (has `model.go`)

Scope limited to the `Damage`→`checkReflect`+`damageCore` split and the new `Kill` method; pre-existing symbols not re-audited.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor accepts `FieldLogger` | PASS | `services/atlas-monsters/atlas.com/monsters/monster/processor.go:102` — `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` (pre-existing, unchanged). |
| FILE-01 | Processor methods in `processor.go` | PASS | `services/atlas-monsters/atlas.com/monsters/monster/processor.go:1723` (`Kill`) and `:573` (`damageCore`, the extracted shared core) both in `processor.go`; interface entry `Kill(uniqueId uint32, characterId uint32, skillId uint32)` co-located. |
| DOM-21 | No duplication of atlas-constants types | PASS | No new domain type/enum introduced; `Kill` reuses the existing `information.Model.Boss()` accessor (`services/atlas-monsters/atlas.com/monsters/monster/information/model.go:46`) rather than re-deriving a boss classification. |
| — | Fail-closed boss guard (explicit design requirement, FR-4) | PASS (test-verified) | `processor.go:1735-1738` — `if infoErr != nil { p.l.WithError(infoErr).Errorf(...); return }` before the `info.Boss()` check; verified by `services/atlas-monsters/atlas.com/monsters/monster/kill_test.go:111-140` (`TestKill_InfoLookupError_DroppedFailClosed` — 0 events, monster untouched, HP unchanged on lookup error). Contrast documented against `DrainMp`'s fail-open lookup in the same comment block (`processor.go:1706-1710`). |
| — | Alive/presence guards precede the boss check | PASS (test-verified) | `processor.go:1724-1732`; `kill_test.go:145-157` (missing monster, 0 events) and `:161-187` (HP-0 monster still in registry, 0 events). |
| — | Errors distinguishable, no error silently returned as success | PASS | `Kill` has no error return (`processor.go` interface line 67, matching the pre-existing void `UseBasicAttack(uniqueId uint32, attackPos uint8)` at `processor.go:60/981` — an established void-method convention in this interface, not a new deviation). |
| DOM-20 | Table-driven tests | **FAIL** | `services/atlas-monsters/atlas.com/monsters/monster/kill_test.go:23,76,111,145,161` — five discrete `func TestKill_*(t *testing.T)` functions, none using the `tests := []struct{...}` + `t.Run` pattern required by the DOM-20 pass criteria. |

## Sub-Domain-Equivalent Checklist — `atlas-monsters` / `kafka/consumer/monster` (Kafka command consumer, no `resource.go` — SUB checklist analog)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 analog | Business logic not in handler | PASS | `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go:177-183` (`handleKillCommand`) — type-gate then a single `p.Kill(...)` call; all business logic in `monster.ProcessorImpl.Kill`. |
| SUB-04 analog | No manual JSON parsing in handler | PASS | Body decoding delegated to the generic `command[killCommandBody]` envelope machinery (`message.AdaptHandler(message.PersistentConfig(handleKillCommand))`, `consumer.go:54-56`); no `json.Unmarshal`/`io.ReadAll` in the handler. |
| — | New command wired into `InitHandlers` | PASS | `consumer.go:54-56`. |
| — | `killCommandBody` placed in `kafka.go` (message-contract file for this consumer package) | PASS | `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go:27` (const), `:110` (struct) — same file as sibling `drainMpCommandBody`. |

## `atlas-channel` / `data/skill/effect` and `kafka/message/monster` — diff-only checks

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| — | Test addition uses existing `Extract()`/Builder-style construction, no ad-hoc struct literal bypass | PASS | `services/atlas-channel/atlas.com/channel/data/skill/effect/model_test.go:45-60` (`TestExtractThreadsXAndY`) calls `Extract(RestModel{X: 20, Y: 5})`, the documented `rest.go` transform (`effect/rest.go:68`). |
| — | No new `*_testhelpers.go` file | PASS | `git diff --name-status c09233e3f..HEAD` shows no file matching `*_testhelpers.go` anywhere in scope. |

## Kafka Producer Test-Stubbing (DOM-24)

| Test file | Emits real Kafka? | Status | Evidence |
|-----------|--------------------|--------|----------|
| `services/atlas-channel/atlas.com/channel/monster/producer_test.go:74-106` (`TestKillCommandProvider`) | No — calls `KillCommandProvider(...)` directly and inspects the returned `model.Provider[[]kafka.Message]`, never touches `producer.ProviderImpl` | PASS | No stub needed; message-construction test only. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_mortal_blow_test.go` (all `mortalBlowTryProc` tests) | No — `emitKill` is a func-literal in `mortalBlowDeps`, never the real `monster.Processor.Kill` | PASS | e.g. `:135-138`, `:155-158`, `:169-172`. |
| `services/atlas-monsters/atlas.com/monsters/monster/kill_test.go` (all `TestKill_*`) | No — `ProcessorImpl.emit` is injected via `newRecordingProcessorWithBodies` (`processor_test.go:236-260`, pre-existing helper), a fake function field, not the real Kafka producer | PASS | `kill_test.go:39-40` etc. |

## Non-Guideline Observation (not scored)

`atlas-monsters/monster` test run took ~14s; `TestKill_NonBoss_KilledAndRemoved` specifically logs a real outbound HTTP attempt and retry/backoff (`Failed calling [GET] on [data/monsters/1000000]... unsupported protocol scheme`). Root cause: `damageCore` (`processor.go:576-579`) calls `information.NewProcessor(p.l, p.ctx).GetById(m.MonsterId())` directly for the `isBoss`/`revives` fields used in the damaged-event body — this call is **not** gated by the `testInformationLookup` test seam (only `Kill`'s own boss guard at `processor.go:1736-1737` is). This code was moved verbatim out of the pre-existing `Damage` method (confirmed via `git show c09233e3f:.../processor.go`, same unconditional call already present before this diff) — it is not a new defect introduced by this branch, and it does not fail the test (the `infoErr == nil` guard degrades to `isBoss=false` on failure). Flagged for awareness only; no DOM-* item covers un-stubbed HTTP calls in tests (DOM-24 is Kafka-specific), so this is not scored as a finding.

## Summary

### Blocking (must fix)
- **DOM-20** — `services/atlas-monsters/atlas.com/monsters/monster/kill_test.go:23,76,111,145,161`: five `TestKill_*` functions are not table-driven (no `tests := []struct{...}` + `t.Run`), failing the literal DOM-20 pass criteria for a package with `model.go`.

### Non-Blocking (should fix / context)
- The DOM-20 finding above mirrors the **pre-existing, service-wide convention** in this exact package — every sibling scenario-test file (`drain_mp_test.go`, and the majority of `processor_test.go`) uses the same discrete-function style, and `testing-guide.md` itself uses "Prefer table-driven tests" (soft language), not a hard mandate. Per audit policy this is still recorded as a finding rather than excused by prevalence, but it is the only FAIL found across ~820 lines of diff, all other structural/DOM/FILE/EXT items pass with citation, and the underlying source guideline does not use mandatory language — recommend treating as low-severity/non-blocking for merge purposes, distinct from a File-Responsibilities-table violation.
- Pre-existing un-gated `information.NewProcessor(...).GetById` call inside `damageCore` (see Non-Guideline Observation above) causes real HTTP retry noise in `atlas-monsters/monster` tests, including the new `TestKill_NonBoss_KilledAndRemoved`. Not a new regression, not scored, but worth a follow-up to route it through `testInformationLookup` like the other three boss-lookup call sites in this file.
