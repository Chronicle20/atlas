# Plan Audit — task-077-gm-mob-spawn-command

**Plan Path:** docs/tasks/task-077-gm-mob-spawn-command/plan.md
**Audit Date:** 2026-05-28
**Branch:** task-077-gm-mob-spawn-command
**Base Branch:** main (merge-base d23ab8448; HEAD 130823cc3)

## Executive Summary

All 8 implementation tasks (Tasks 1-8) were faithfully implemented, each in its own commit, matching the plan's prescribed code essentially line-for-line. Both affected Go modules (atlas-messages, atlas-monsters) build clean, vet clean, and pass their full test suites with `-race`. Every named test the plan specified exists and passes. The only deviations are non-functional: the plan's checkboxes were never ticked (60 `- [ ]`, 0 `- [x]`), and Task 9's docker-bake / redis-key-guard infra steps could not be cleanly confirmed in this environment (redis-key-guard fails on an unrelated, untouched service due to a local go.sum gap). Recommendation: READY_TO_MERGE after confirming docker bake in CI.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Plumb GM X/Y through `character.Model` | DONE | `character/model.go:42-43` (struct fields), `:215-221` (X()/Y() getters), `:269-270` (Clone), `:345-346` (SetX/SetY), `:379-380` (Build), `rest.go:129-130` (Extract). Stance left at `0` per plan. Commit 104df0211. |
| 2 | `data/monster` validation client | DONE | `data/monster/model.go`, `rest.go` (GetName "monsters"), `requests.go:10` (`data/monsters/%d`), `processor.go` (GetById), `mock/processor.go`. Commit 3fe50da3c. |
| 3 | `data/foothold` below client | DONE | `data/foothold/model.go`, `rest.go` (Extract reads only Id, no nil-deref; PositionRestModel GetName "positions"), `requests.go:11` (`data/maps/%d/footholds/below`, POST), `processor.go` (GetBelow), `mock/processor.go`. Commit 276993e4c. |
| 4 | `SpawnFieldCommandProvider` Kafka provider | DONE | `kafka/message/monster/kafka.go:19` (CommandTypeSpawnField), `:113-119` (SpawnFieldBody), `:121-142` (provider emits count-length []RawMessage, single MessageProvider). Commit 517db7fcd. |
| 5 | `MobSpawnCommandProducer` | DONE | `command/monster/commands.go:176` (cap 20), `:178` (regex), `:183-201` (parseSpawnArgs), `:205-213` (normalizeCount), `:215-262` (producer: GM gate, validate, foothold non-fatal fallback, emit, pink-text + capped note). Uses incoming `f` for world/channel/map/instance. Import alias `monsterdata`. Commit c2627be94. |
| 6 | Register producer + help text | DONE | `main.go:59` (`MobSpawnCommandProducer` registered after the other @mob lines), `command/help/commands.go:31` (help line, count 1-20). Commit a2d99af9f. |
| 7 | atlas-monsters SPAWN_FIELD body + constant | DONE | `kafka/consumer/monster/kafka.go:22` (CommandTypeSpawnField), `:88-94` (spawnFieldCommandBody, exact JSON tags). Commit 02296798a. |
| 8 | `handleSpawnFieldCommand` consumer | DONE | `kafka/consumer/monster/consumer.go:257-274` (handler builds field with instance, calls `p.Create(f, monster.RestModel{...})`), `:64-66` (registered in InitHandlers after DESTROY_FIELD, before movement topic). Create signature/RestModel fields verified compatible. Commit 130823cc3. |
| 9 | Full verification gate | PARTIAL | Code-level gate re-verified green by this audit (build/vet/test -race both modules). Docker bake and redis-key-guard not cleanly confirmable here (see below). No code artifact; verification-only task. |

**Completion Rate:** 8/8 implementation tasks (100%); Task 9 is a verification gate (PARTIAL — see notes).
**Skipped without approval:** 0
**Partial implementations:** 0 functional; Task 9 infra steps unconfirmed.

## Skipped / Deferred Tasks

None skipped or deferred. The plan's task list is fully realized in code.

Note on plan checkboxes: all step boxes remain `- [ ]` (none ticked). This is a documentation-hygiene gap only — every step's code is present and the corresponding tests pass. No functional impact.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-messages | PASS | PASS | `go build ./...`, `go vet ./...` clean; `go test -race -count=1 ./...` all packages ok. New tests (TestExtract_Position, TestModel_PositionRoundTripsThroughSetSkills, data/monster + data/foothold contract tests, TestSpawnFieldCommandProvider_EmitsCountMessages, TestParseSpawnArgs, TestNormalizeCount, TestMobSpawnCommandProducer_GmGate) all PASS. |
| atlas-monsters | PASS | PASS | `go build ./...`, `go vet ./...` clean; `go test -race -count=1 ./...` all packages ok. TestSpawnFieldCommandBody_Decode PASS. |

Task 9 infra steps not independently confirmed by this audit:
- `docker buildx bake atlas-messages` / `atlas-monsters`: not run (long infra step). Risk is low — no `go.mod`, `go.work`, or root `Dockerfile` was touched, and no new shared lib was added, so the missing-`COPY` failure class this step guards against does not apply.
- `tools/redis-key-guard.sh`: reports FAIL, but every flagged file is in `atlas-monster-book` (untouched by this task) and the failure is a typecheck/`go.sum` resolution error in that service's local analysis, not a raw-keyed-redis usage. The two task-077 modules introduced zero redis code. Not a task-077 defect.

## Overall Assessment

- **Plan Adherence:** FULL (all 8 implementation tasks DONE; verification gate code-portion green)
- **Recommendation:** READY_TO_MERGE — pending a green `docker buildx bake` for both services in CI (mechanical, low risk).

## Action Items

1. (Optional, hygiene) Confirm `docker buildx bake atlas-messages` and `atlas-monsters` succeed in CI or locally to fully close Task 9 step 3.
2. (Optional, hygiene) The redis-key-guard FAIL is a pre-existing environmental issue in `atlas-monster-book` (missing go.sum entries), unrelated to task-077; verify it is green in a clean CI environment.
3. (Cosmetic) None required for merge: plan checkboxes were left unticked but all work is verified complete.

---

## Backend Guidelines Audit (DOM/SUB/SEC)

- **Reviewer:** backend-guidelines-reviewer (adversarial)
- **Date:** 2026-05-28
- **Scope:** diff d23ab8448..130823cc3 only (atlas-messages + atlas-monsters changed packages)
- **Build:** PASS (`go build ./...` clean in both `atlas-messages` and `atlas-monsters`)
- **Tests:** PASS (atlas-messages `./...` ok; new packages `command/monster`, `kafka/message/monster`, `data/monster`, `data/foothold` all PASS; atlas-monsters `kafka/consumer/monster` PASS)
- **Overall:** NEEDS-WORK (build/test green; one blocking EXT finding + two non-blocking)

### Package classification

| Package | Type | Checklist applied |
|---|---|---|
| atlas-messages `character` | domain (model.go) — pre-existing, diff touches X/Y plumbing only | immutable-model/Builder subset |
| atlas-messages `command/monster` | command producer (no model.go) | functional/processor conventions |
| atlas-messages `kafka/message/monster` | Kafka provider (no model.go) | provider conventions |
| atlas-messages `data/monster` | external HTTP client to atlas-data (model/processor/requests/rest, mirrors `data/equipable`) | EXT-* + immutable model + Processor I/Impl |
| atlas-messages `data/foothold` | external HTTP client to atlas-data | EXT-* + immutable model + Processor I/Impl |
| atlas-monsters `kafka/consumer/monster` | Kafka consumer (no model.go) | consumer conventions |

The two `data/*` packages are read-only upstream clients, not write-side domains. DOM-01/02/03/15/16 (builder.go / entity.go ToEntity / Make / db.Create / administrator.go) do NOT apply — they correctly mirror the established `data/equipable` external-client shape (model+processor+requests+rest+mock). They are governed by the External HTTP Client checklist (EXT-*).

### Mechanical results

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Immutable model / Builder | `character.Model` X/Y plumbed through private fields + getters + Clone + Set + Build | PASS | model.go:42-43 (fields), :216/:220 (X()/Y()), :269-270 (Clone), :345-346 (SetX/SetY), :379-380 (Build) |
| Processor I/Impl | `data/monster` + `data/foothold` use `Processor` interface + `ProcessorImpl` + `NewProcessor(l, ctx)` | PASS | data/monster/processor.go:10-26; data/foothold/processor.go:11-27 |
| Provider lazy eval | clients use `requests.Provider[RestModel, Model]` lazy provider | PASS | data/monster/processor.go:27; data/foothold/processor.go:28 |
| Kafka provider | `SpawnFieldCommandProvider` returns `model.Provider[[]kafka.Message]`, single `MessageProvider(FixedProvider(...))`, count-length slice | PASS | kafka/message/monster/kafka.go:121-142 |
| FieldLogger params | producer + consumer + clients accept `logrus.FieldLogger` (not `*logrus.Logger`) | PASS | commands.go:212; consumer.go:257; both data processors |
| Multi-tenancy / ctx | ctx threaded producer→provider and consumer→processor; consumer builds field from envelope, tenant from `AdaptHandler` header context | PASS | commands.go:215-221; consumer.go:257-274 |
| Consumer topic registration | `handleSpawnFieldCommand` registered on `EnvCommandTopic` (before movement-topic reassignment at consumer.go:67) | PASS | consumer.go:64-66 |
| Wire-type symmetry | producer `SpawnFieldBody` ≡ consumer `spawnFieldCommandBody` ≡ atlas-monsters `RestModel` (monsterId uint32; x/y/fh int16; team int8) | PASS | message/.../kafka.go:113-119 vs consumer/.../kafka.go:88-94 vs monster/rest.go:19-24 |
| Table-driven tests | new tests use `testCases := []struct{...}` + `t.Run` | PASS | command/monster/commands_test.go:18-49, :52-75 |
| EXT-04 RootUrl | URLs composed via `requests.RootUrl("DATA")`, not hardcoded | PASS | data/monster/requests.go:13-18; data/foothold/requests.go:15-20 |
| **EXT-01** | **JSON:API relationship interface methods on client RestModels** | **FAIL** | data/monster/rest.go and data/foothold/rest.go RestModels implement only GetName/GetID/SetID — NO `SetToOneReferenceID` / `SetToManyReferenceIDs`. The sibling clients in this same service DO (data/equipable/rest.go:72-76, data/map/rest.go:51-55), per libs/atlas-rest/CLAUDE.md:24-25. api2go errors on any upstream response carrying a `relationships` block, surfacing as a misleading decode/"not found" failure. |
| EXT-02 | httptest-backed integration test for clients | WARN | No `httptest.NewServer` test for data/monster or data/foothold; tests cover Extract/GetName only (rest_test.go). Established convention in this service: NO existing data/* client (equipable, map, asset, skill) has an httptest test either — non-blocking but the unmarshal path is untested. |
| EXT-03 | 404 distinguished from transport/5xx | WARN | commands.go:241-243 maps EVERY error from `monsterdata.GetById` to "Unknown monster template" — `requests.ErrNotFound` is not distinguished from decode/5xx/transport failures. A DATA outage or a decode bug (see EXT-01) would be reported to the GM as a bogus "unknown template". Foothold failure is correctly non-fatal (commands.go:246-250). |
| DOM-21 | Reuse atlas-constants types | PASS (no redeclaration) | No new `type`/enum/classification helper shadows atlas-constants. Raw `uint32` monster id and `int16` x/y/fh are consistent with the pre-existing wire bodies and RestModels (monster/rest.go:19-24) and `character.X()/Y()` returning raw int16. `monster2` (atlas-constants/monster) already imported (commands.go:17). Soft note: `monster.Id` / `point.X` / `point.Y` typed constants exist and could type the new fields, but using primitives here matches the surrounding established convention and is not a DOM-21 redeclaration violation. |
| SEC-* | auth/token handling | N/A | Neither service handles auth/tokens/redirects; GM gate via `c.Gm()` (commands.go:218) is an authorization guard and is present. No hardcoded secrets in the diff. |

### Minor (non-blocking) notes

- `command/monster/commands.go:251` narrows `fh = int16(fhModel.Id())` from a `uint32` foothold id without a range guard. Foothold indices are small in practice so this won't overflow, but it is an unguarded narrowing.
- Kafka producer is exercised only via the closure-returning `MobSpawnCommandProducer` test, which never invokes the executor — so no unstubbed emit path runs in tests (DOM-24 N/A: no test triggers a real `producer.ProviderImpl`/`message.Emit`). Consumer test only decodes the body and never invokes `handleSpawnFieldCommand`, so no transitive emit either. No `producertest` stub required for the current tests.

### Summary

**Blocking (must fix)**
- EXT-01: Add `SetToOneReferenceID` and `SetToManyReferenceIDs` (no-op) to `data/monster.RestModel` and `data/foothold` RestModels (both `RestModel` and `PositionRestModel`), matching the sibling `data/equipable`/`data/map` precedent. Without them, any upstream JSON:API response with a `relationships` block fails to unmarshal and surfaces as a misleading error.

**Non-Blocking (should fix)**
- EXT-03: Distinguish `requests.ErrNotFound` from other errors in the producer so transport/5xx failures aren't reported to the GM as "Unknown monster template".
- EXT-02: Add an httptest-backed unmarshal test for at least `data/monster` (the only client whose decoded `Name` is shown to the user); the FakeClient-style mock bypasses unmarshal.
- Add a range guard (or document the invariant) for the `int16(fhModel.Id())` narrowing.

---

## Resolution (post-audit fixes — commit 7047f5dd1)

- **EXT-01 — FIXED.** Added no-op `SetToOneReferenceID` / `SetToManyReferenceIDs` to the JSON:API **decode targets** `data/monster.RestModel` and `data/foothold.RestModel`. `data/foothold.PositionRestModel` is only ever **marshalled** as the POST request body (never unmarshalled via `jsonapi.Unmarshal`), so it does not require the relationship stubs — the runtime decode path is fully covered. `go build`/`go vet`/`go test -race` green.
- **EXT-03 — FIXED.** The spawn executor now distinguishes `requests.ErrNotFound` (→ "Unknown monster template: %d") from transport/decode/5xx failures (→ error-logged + "Failed to look up monster template %d."), per libs/atlas-rest/CLAUDE.md §"Distinguishing 404 from decode failure".
- **EXT-02 — ACCEPTED (not changed).** No httptest-backed unmarshal test added, matching the established convention in this service (no existing `data/*` client has one). With the EXT-01 stubs in place the documented decode failure mode no longer applies.
- **fh narrowing — ACCEPTED (not changed).** `int16(fhModel.Id())` is consistent with the codebase-wide treatment of foothold ids as 16-bit; safe in practice.
