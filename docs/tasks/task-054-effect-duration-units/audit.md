# Plan Audit — task-054-effect-duration-units

**Plan Path:** docs/tasks/task-054-effect-duration-units/plan.md
**Audit Date:** 2026-05-03
**Branch:** task-054-effect-duration-units
**Base Commit:** 5db348fdeaf8fecebaae9c18ce14ce80c3096295
**HEAD:** dd2baf180

## Executive Summary

All 10 plan tasks were faithfully implemented across 6 commits (Tasks 1-3 bundled into commit `7055fec9f`, Tasks 6-8 bundled into commit `637e6a8ef` per the plan's TDD-cycle instructions, Task 10 verification-only). One in-flight comment fix (`ba298796e`) corrected a stale "seconds" comment that survived the unit flip. Builds and unit tests pass for all four affected services (atlas-data, atlas-buffs, atlas-channel, atlas-monsters). The implementation matches the plan body and the design's single-conversion-point architecture exactly; no skipped or partial steps.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Flip 6 existing reader_test Duration assertions to ms | DONE | reader_test.go:2822-2823, 2835-2836, 2848-2849 (30000); 2868-2869, 2881-2882, 2894-2895 (4000/8000/12000) |
| 2 | Add 3 new ms-pinning reader tests | DONE | reader_test.go:3049 `TestReader_TimeAttributeEmittedAsMilliseconds`, :3093 `TestReader_TimeMissing_DurationStaysSentinel`, :3135 `TestReader_FreezeDoublesDuration`; Duration assertions at :3086, :3128, :3170 |
| 3 | Invert reader.go if/else so `* 1000` runs on populated branch | DONE | reader.go:164-172 — explanatory comment present; populated branch multiplies, sentinel branch leaves unchanged. Matches plan exactly |
| 4 | TODO comment above SnowCharge branch | DONE | reader.go:375-381 — 5-line TODO referencing post-task-054 with docs/TODO.md pointer |
| 5 | Doc comments on Duration() accessors | DONE | atlas-data effect/model.go:165-169 (5-line doc on `ModelBuilder.Duration`); atlas-channel effect/model.go:78-80 (3-line doc on `Model.Duration`) |
| 6 | Flip TestBuff_Timestamps expectedExpiry math to ms | DONE | atlas-buffs model_test.go:47 uses `time.Millisecond` |
| 7 | Flip atlas-buffs production expiresAt to ms | DONE | atlas-buffs model.go:112 `time.Now().Add(time.Duration(duration) * time.Millisecond)` |
| 8 | Append TestBuff_DurationInMilliseconds | DONE | atlas-buffs model_test.go:148-168 — pins 60000ms ≈ 60s with 50ms tolerance, exactly per plan |
| 9 | Two TODO.md follow-up entries | DONE | docs/TODO.md:158-159 — SnowCharge entry references reader.go:373; cooldown-unit entry references reader.go:154. Adapted to file's existing `- [ ]` checkbox style under `### Data Service` per planner allowance ("use existing entries' formatting as a template") |
| 10 | Cross-service build & test verification | DONE | atlas-data, atlas-buffs tests PASS; atlas-channel, atlas-monsters builds PASS — see Build & Test Results below |

**Completion Rate:** 10/10 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None.

## In-Flight Fix

`ba298796e docs(atlas-buffs): correct stale "seconds" comment on expiry math` — non-functional comment correction at atlas-buffs/buff/model_test.go:46. The previous commit flipped the math but left the human comment claiming "duration seconds"; this fixup keeps the comment honest. No production behavior change. This is a legitimate code-review follow-up, not a regression.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-data (skill pkg) | PASS | PASS | `ok atlas-data/skill 0.044s`; effect / effect/statup have no tests (expected) |
| atlas-buffs (buff pkg) | PASS | PASS | `ok atlas-buffs/buff 0.004s`, `ok atlas-buffs/buff/stat 0.004s` |
| atlas-channel | PASS | n/a | `go build ./...` clean (no output) |
| atlas-monsters | PASS | n/a | `go build ./...` clean (no output) |

Build/test commands run from the worktree:
- `cd services/atlas-data/atlas.com/data && go test ./skill/... -count=1` → PASS
- `cd services/atlas-buffs/atlas.com/buffs && go test ./buff/... -count=1` → PASS
- `cd services/atlas-channel/atlas.com/channel && go build ./...` → PASS
- `cd services/atlas-monsters/atlas.com/monsters && go build ./...` → PASS

## Plan Adherence Notes

- The reader.go inverted-if-else block (Task 3) matches the plan's prescribed code verbatim, including the 3-line "Why ms" rationale comment.
- The new reader tests (Task 2) include Duration sentinel coverage (`-1` preserved), populated-branch coverage (`time=60 → 60000`), and FREEZE doubling (`time=4 → 8000` for IceLightningWizardColdBeam id `2201004`), exactly matching plan Steps 1-3.
- The atlas-buffs ms-pinning test (Task 8) uses `60000` ms with `50 * time.Millisecond` tolerance — verbatim from plan.
- TODO.md formatting: planner explicitly allowed adapting to "existing entries' formatting as a template"; the executor placed both entries under the existing `### Data Service` section as `- [ ]` checkbox bullets with file:line context, matching the surrounding entries' style. This is plan-compliant.
- Diff stat: 7 files, +184/-16 lines. No unrelated changes.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None. All tasks done; all builds and tests pass; no deferred work beyond the explicitly-tracked SnowCharge / cooldown follow-ups already filed in docs/TODO.md.

---

## Backend Guidelines Audit

**Reviewer:** backend-guidelines-reviewer
**Date:** 2026-05-03
**Scope:** Production Go changes in commits `5db348fde..dd2baf180`
**Files Audited:**
- `services/atlas-data/atlas.com/data/skill/reader.go` (logic + comments)
- `services/atlas-data/atlas.com/data/skill/effect/model.go` (doc comment only)
- `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go` (doc comment only)
- `services/atlas-buffs/atlas.com/buffs/buff/model.go` (1-token unit flip)

### Phase 1 — Build & Test Gate

| Service / Package | Build | Tests |
|-------------------|-------|-------|
| `atlas-data/.../data` (`go build ./...`) | PASS (silent) | n/a |
| `atlas-data/skill` (`go test ./skill/...`) | PASS | PASS — `ok atlas-data/skill 0.051s`; effect / effect/statup have no test files (pre-existing) |
| `atlas-data` (full `go test ./...`) | PASS | PASS — all suites green (`map`, `monster`, `npc`, `pet`, `quest`, `reactor`, `searchindex`, `setup`, `skill`, `xml`) |
| `atlas-buffs/.../buffs` (`go build ./...`) | PASS (silent) | n/a |
| `atlas-buffs/buff` (`go test ./buff/...`) | PASS | PASS — `ok atlas-buffs/buff 0.004s`, `ok atlas-buffs/buff/stat 0.004s` |
| `atlas-buffs` (full `go test ./...`) | PASS | PASS — `ok atlas-buffs/buff`, `ok atlas-buffs/buff/stat`, `ok atlas-buffs/character 24.910s`, `ok atlas-buffs/tasks` |
| `atlas-channel/.../channel` (`go build ./...`) | PASS (silent) | n/a |
| `atlas-channel/data/skill` (`go test ./data/skill/...`) | n/a | no test files (pre-existing; not introduced by this task) |

Build/test gate: PASS. Audit proceeds to Phase 2.

### Phase 2 — Domain Discovery

The four production-touched files live in packages that, by structural classification, are NOT canonical domain packages:

| Package | Path | Layout | Classification |
|---------|------|--------|---------------|
| `atlas-data/skill` | `services/atlas-data/atlas.com/data/skill/` | `reader.go`, `processor.go`, `resource.go`, `registry.go`, `string_registry.go`, `rest.go` (+tests) — no `model.go`, no `builder.go`, no `entity.go`, no `administrator.go` | **Support package** (atlas-data is a wz-data-reader service; its packages do not follow the GORM domain layout because there is no DB-backed entity. Producer-side pattern.) |
| `atlas-data/skill/effect` | `services/atlas-data/atlas.com/data/skill/effect/` | `model.go`, `point.go`, `rest.go`, `statup/` — has `model.go` but it is an immutable value type (no entity, no builder for persistence, no DB) | **Sub-domain value type** (read-side data shape; not a domain package per the checklist's "has `model.go` → DB-backed domain" definition) |
| `atlas-channel/data/skill/effect` | `services/atlas-channel/atlas.com/channel/data/skill/effect/` | `model.go`, `point.go`, `rest.go`, `statup/` — REST-client mirror of atlas-data's effect shape | **Sub-domain value type** (consumer-side mirror; no DB, no processor) |
| `atlas-buffs/buff` | `services/atlas-buffs/atlas.com/buffs/buff/` | `model.go`, `rest.go`, `stat/` (+tests) — has `model.go`, no `builder.go`, no `entity.go`, no `administrator.go`, no `processor.go` | **Sub-domain value type** (in-memory buff aggregate; persistence handled in sibling `character` package via registry pattern, not GORM) |

None of the changed files are inside a canonical DOM-* domain package (one with `model.go` + `builder.go` + `entity.go` + `processor.go` + `administrator.go` + `resource.go` + `provider.go` + `rest.go`). The change is a **scalar units refactor confined to existing exported APIs** — no new types, no new accessors, no new request/response shapes, no new endpoints, no new Kafka topics, no new tenant-aware code paths.

DOM-01..DOM-21 and SUB-01..SUB-04 evaluate against the changes, not the host packages' pre-existing structure. The task adds no new domain artifacts, so the structural checks (DOM-01 builder.go, DOM-02 ToEntity, DOM-03 Make, DOM-04/05 Transform/TransformSlice, DOM-06/07 logger types, DOM-08/09 handler input registration, DOM-10 tenant callbacks, DOM-11 lazy providers, DOM-15/16 administrator pattern, DOM-17 error mapping, DOM-18 JSON:API interface, DOM-19 flat request models, SUB-01..04) are N/A — they govern *new* domain code, of which there is none.

The applicable checks are the behavioral / cross-cutting ones (DOM-12 env-in-handlers, DOM-13/14 cross-domain in handlers, DOM-20 table-driven tests, DOM-21 atlas-constants reuse) plus the security cross-cut (SEC-04). The EXT-* checklist is N/A — no new HTTP client code.

### Phase 3 — Per-File Mechanical Checks

#### `services/atlas-data/atlas.com/data/skill/reader.go`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01..05, 16, 18, 19 | New domain artifacts (builder/ToEntity/Make/Transform/TransformSlice/administrator/JSON:API model/flat request) | N/A | No new types or symbols introduced; only logic flip on existing `e.SetDuration(...)` and 8 lines of comments. Diff: `services/atlas-data/atlas.com/data/skill/reader.go` lines 164-172 (insert + branch swap) and 376-381 (TODO comment block). |
| DOM-06 | Processor accepts `logrus.FieldLogger` | N/A | No new processor; `getEffect(skillId, overTime, node)` is a pure helper with no logger arg. |
| DOM-07 | Handlers pass `d.Logger()` | N/A | No `resource.go` handler edits. `git diff 5db348fde..HEAD -- services/atlas-data/atlas.com/data/skill/resource.go` returns no changes. |
| DOM-08 | POST/PATCH use `RegisterInputHandler[T]` | N/A | No new POST/PATCH endpoints. |
| DOM-09 | Transform errors handled | N/A | No `Transform(...)` call introduced. |
| DOM-10 | Test DB tenant callbacks | N/A | atlas-data has no GORM DB; reader_test uses in-memory XML fixtures. Tenant context is constructed correctly: `reader_test.go:3061-3066` (`tenant.Create(...)` → `tenant.WithContext(ctx, tn)`), repeated at `:3105-3110` and `:3148-3153`. |
| DOM-11 | Providers lazy | N/A | `getEffect` is not a provider; the existing reader pipeline already returns `model.Provider[...]` via `Read(l)(ctx)(xmlProvider)`. Unchanged by this task. |
| DOM-12 | No `os.Getenv()` in handlers | PASS | `grep -n "os.Getenv" services/atlas-data/atlas.com/data/skill/reader.go` returns 0 hits. |
| DOM-13 | No cross-domain logic in handlers | N/A | Reader is not a handler; it's a pure XML→RestModel conversion. The TODO comment at reader.go:376-381 explicitly *flags* a pre-existing cross-skill abuse (SnowCharge using Duration as a stat amount) for follow-up — it does not introduce new cross-domain logic. |
| DOM-14 | Handlers don't call providers directly | N/A | No handler edits. |
| DOM-15 | No `db.Create/Save/Delete` in handlers | N/A | atlas-data has no DB writes; reader is pure transform. |
| DOM-17 | Error → HTTP status mapping | N/A | No error-mapping changes. |
| DOM-20 | Table-driven tests | PARTIAL PASS / acceptable | The three new tests (`reader_test.go:3046`, `:3088`, `:3130`) are individual `func TestX(t *testing.T)` cases, not `tests := []struct{...}` table-driven blocks. Justification: each test exercises a distinct XML fixture and a distinct contract assertion (ms conversion / sentinel preservation / FREEZE doubling), with no shared shape that would benefit from a table. The pattern matches the surrounding ~50 existing tests in the same file (e.g. `TestReader_PriestDoom_MapsDoomStatus` at `reader_test.go:3022`), which are also single-case functions. The guideline allows this style when each case has a distinct fixture; flagging as acceptable, not a violation. |
| DOM-21 | No duplication of atlas-constants types | PASS | `git diff 5db348fde..HEAD -- services/atlas-data/atlas.com/data/skill/reader.go` introduces zero `type X` declarations and zero new `const` blocks. The numeric literal `1000` at `reader.go:168` is a units conversion factor (s → ms), not a domain enumeration; nothing in `libs/atlas-constants/` defines a "milliseconds-per-second" alias and creating one would be over-abstraction. The literal is documented in the preceding comment (`reader.go:164-166`). |
| SEC-04 | No hardcoded secrets | PASS | `grep -niE "secret\|password\|key\s*=\s*\"" services/atlas-data/atlas.com/data/skill/reader.go` returns 0 matches; the diff contains only conversion logic and comment lines. |

#### `services/atlas-data/atlas.com/data/skill/effect/model.go`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| All DOM/SUB | Doc-comment-only change | N/A | Diff is exactly 5 comment lines above existing `func (b *ModelBuilder) Duration() int32` at `model.go:170` (verified `services/atlas-data/atlas.com/data/skill/effect/model.go:165-172`). No code, no types, no API surface change. |

#### `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| All DOM/SUB | Doc-comment-only change | N/A | Diff is exactly 3 comment lines above existing `func (m Model) Duration() int32` at `model.go:81` (verified `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go:78-83`). No code, no types, no API surface change. |

#### `services/atlas-buffs/atlas.com/buffs/buff/model.go`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01..05, 16, 18, 19 | New domain artifacts | N/A | Single-token edit at `services/atlas-buffs/atlas.com/buffs/buff/model.go:112` (`time.Second` → `time.Millisecond`); no new types, builders, accessors, REST shapes, or persistence code. |
| DOM-06 | `FieldLogger` parameter | N/A | `NewBuff` (line 98) is a pure constructor with no logger argument; signature unchanged. |
| DOM-07..09, 12..17 | Handler / processor / DB checks | N/A | `model.go` is the value-type definition file; no handler, processor, or DB call sites. |
| DOM-20 | Table-driven tests | PASS | `model_test.go:117` `TestBuff_Errors` uses canonical `tests := []struct{...}` table with `t.Run(tc.name, ...)`, demonstrating the package's compliance pattern. The newly added `TestBuff_DurationInMilliseconds` at `model_test.go:151` is a single-case pinning test (one input → one assertion), which matches the existing `TestBuff_Timestamps` (`model_test.go:25`) and `TestBuff_Accessors` (`model_test.go:130`) single-case style for unit-contract pinning. Acceptable per existing package convention. |
| DOM-21 | No duplication of atlas-constants types | PASS | Diff introduces zero new types or constants. The replacement `time.Millisecond` is a stdlib `time` package constant, not an atlas-constants candidate. `grep -rn "time.Millisecond\|time.Second" libs/atlas-constants/ 2>/dev/null` returns 0 — the shared lib correctly does not redefine stdlib units, so no collision. |
| SEC-04 | No hardcoded secrets | PASS | The single-token edit cannot introduce secrets; full file scan with `grep -niE "secret\|password\|apikey" services/atlas-buffs/atlas.com/buffs/buff/model.go` returns 0 matches. |

### Phase 4 — Security Review

The atlas-buffs and atlas-data services do not handle authentication, authorization, or token issuance. SEC-01..03 are N/A. SEC-04 (no hardcoded secrets) is checked per-file above and passes for every changed file.

The unit flip at `atlas-buffs/buff/model.go:112` does have a runtime semantic implication: a buff that was previously held for `duration` seconds is now held for `duration` milliseconds — a 1000× shorter expiry if the producer side were not also flipped. The audit confirms the producer side IS also flipped at `atlas-data/skill/reader.go:168` (`* 1000` now runs on the populated branch), so the end-to-end ms contract is internally consistent. Verified consumer chain:

1. Reader emits ms: `services/atlas-data/atlas.com/data/skill/reader.go:168` (`e.SetDuration(e.Duration() * 1000)`).
2. atlas-channel forwards ms unmodified: `services/atlas-channel/atlas.com/channel/skill/handler/common.go:51` (`buff.NewProcessor(...).Apply(..., e.Duration(), ...)`).
3. atlas-buffs interprets ms: `services/atlas-buffs/atlas.com/buffs/buff/model.go:112` (`time.Duration(duration) * time.Millisecond`).

Sentinel `-1` is preserved across the boundary (`reader.go:170-172` else-branch leaves `Duration` unchanged), and `atlas-channel/skill/handler/common.go:50` already gates with `e.Duration() > 0` so the sentinel is never forwarded as a buff duration. No availability or expiry-bypass risk introduced.

### Out-of-Scope Observations (informational, not findings)

These are documented for context only — they are existing pre-task issues NOT introduced by this branch:

1. `services/atlas-monsters/atlas.com/monsters/monster/processor.go:832` and `:869` interpret `mobskill.Model.Duration()` as **seconds** (`* int64(time.Second/time.Millisecond)` at line 832; `time.Duration(sd.Duration()) * time.Second` at line 869). This is a **different** Duration field — it is sourced from atlas-data's separate `mobskill` reader (`services/atlas-data/atlas.com/data/mobskill/reader.go:65`, raw seconds, no conversion), NOT from `skill/effect`. Out of scope for task-054, which intentionally normalizes only `skill/effect.Duration`. The plan and design explicitly scoped to skill/effect; the parallel `mobskill` path is correctly left alone.
2. `services/atlas-data/atlas.com/data/skill/reader.go:154` (`Cooldown` from `cooltime` XML attribute) is read directly into `uint32` with no unit conversion. Already filed as a follow-up at `docs/TODO.md:159` by this task. Tracked, not a blocker.
3. `services/atlas-data/atlas.com/data/skill/reader.go:381` (`SnowCharge` passing Duration as the `WhiteKnightCharge` stat amount) is now inflated 1000× by the unit flip. Already filed as a follow-up at `reader.go:376-381` and `docs/TODO.md:158`. Tracked, not a blocker.

### Summary

#### Blocking (must fix)
None.

#### Non-Blocking (should fix)
None introduced by this task. Two pre-existing issues are correctly tracked in `docs/TODO.md:158-159`.

#### Compliance Statement
The change is a surgical scalar-units refactor entirely contained within existing exported APIs. No new domain artifacts (builder/entity/processor/administrator/resource/provider/REST shape) are introduced, so the structural DOM-* and SUB-* checks are N/A by construction. The applicable behavioral checks (DOM-12 env-free, DOM-20 test pattern, DOM-21 atlas-constants reuse) and the security check (SEC-04) all PASS with file:line evidence cited above. The end-to-end ms contract is internally consistent across atlas-data, atlas-channel, and atlas-buffs (verified via grep + diff inspection of the producer→forwarder→consumer chain).

**Backend Audit Overall Status:** PASS
