# Plan Audit — task-111-resurrection-skill

**Plan Path:** docs/tasks/task-111-resurrection-skill/plan.md
**Audit Date:** 2026-06-25
**Branch:** task-111-resurrection-skill
**Base Branch:** main (334af4a9b)

## Executive Summary

All 10 plan tasks (1, 2, 3, 4, 5a, 5b, 6, 7, 8, 9) were faithfully implemented across the 11 commits `5083d0e6a..e9435e55a`. Every file the plan named to create or modify exists with the specified content; no task was silently skipped, stubbed, or deferred. The implementation actually exceeds the plan with two additive tests (`TestResurrection_WarpFailureIsolation`, `TestSelectDeadInRangeMapPlayers_MissingRectangleReturnsNil`). Both affected modules build clean, vet clean, and pass `go test -race ./...` with no failures or race warnings. No changed file introduces raw go-redis usage, so the documented redis-key-guard GOWORK=off artifact is not a task-111 concern.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | GmResurrectionId const + var + Skills entry + test | DONE | const `libs/atlas-constants/skill/constants.go:3248` `GmResurrectionId = Id(9001005)`; var `:1455`; registry `:2698` `GmResurrectionId: GmResurrection`; test `libs/atlas-constants/skill/resurrection_test.go:5,17`. Pre-existing `BishopResurrectionId` (2321006) and `SuperGmResurrectionId` (9101005) confirmed at `:3085,:3254`. |
| 2 | SET_HP command type | DONE | `kafka/message/character/kafka.go:18` `CommandSetHP = "SET_HP"`; `:66-69` `SetHPCommandBody{ChannelId, Amount uint16}` matching atlas-character's `SetHPBody`. |
| 3 | SetHPCommandProvider | DONE | `character/producer.go:71-84` mirrors `ChangeHPCommandProvider` with `uint16`, `CommandSetHP`, `SetHPCommandBody`; test `character/producer_test.go:14` asserts Type/CharacterId/ChannelId/Amount=0xFFFF. |
| 4 | SetHP processor method + interface + mock | DONE | interface `character/processor.go:42`; impl `:276-278` (ProviderImpl → EnvCommandTopic → SetHPCommandProvider); no-op mock `character/mock/processor.go:123-125`. |
| 5a | SetX/SetY builder setters | DONE | `character/builder.go:138-139` `SetX(int16)`/`SetY(int16)`; model fields/getters pre-existed. |
| 5b | Generalize selector + dead selectors + loadMapPlayerFunc + tests | DONE | `selectPartyMembers` gains `wantDead bool` (`recipients.go:196`), dead/alive branch replaces hard-coded skip (`:240-247`); both existing callers pass `false` (`:110`, `:185`) — behavior preserved; `SelectDeadInRangePartyMembers` (`:114`), `SelectDeadInRangeMapPlayers` (`:130`, caster excluded, death X/Y captured), `loadMapPlayerFunc` seam (`:78`). Tests at `recipients_test.go:299,318,327,337,355,367` incl. living-only regression. |
| 6 | selectByVariant dispatch + tests | DONE | `resurrection/recipients.go`: `selectDeadParty`/`selectDeadMap` seams, Bishop→party / default(GM,SuperGM)→map. Tests `resurrection/recipients_test.go:48,56,64` verify the spy fires the correct selector per variant. |
| 7 | Apply handler: 3-ID registration, setHP-before-warp, per-recipient isolation, caster no-op, broadcast-on-empty + tests | DONE | `resurrection/resurrection.go`: `init()` registers all 3 IDs; `Apply` loads caster (no-op `return nil` on error, before broadcast), `selectByVariant`, per-recipient `setHP(math.MaxUint16)` then `warpToPosition`, each failure `continue`s (isolation), `broadcastEffects` fires unconditionally after loop (empty set still broadcasts). Tests `resurrection_test.go:87,96,114,128,145,166` cover all branches. |
| 8 | Blank import in registrations.go | DONE | `skill/handler/registrations/registrations.go:8` `_ "atlas-channel/skill/handler/resurrection" // ... task-111`; heal/mysticdoor imports intact. |
| 9 | Verification gate | DONE | Re-run during this audit: see Build & Test Results below. Docker bake not re-run in audit (no `go.mod`/Dockerfile/go.work change in the diff; `libs/atlas-constants` and atlas-channel `go.mod` untouched). |

**Completion Rate:** 10/10 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. The OQ-1..OQ-5 items in the plan's "Live verification gates" section are explicitly post-implementation, on-environment checks (not code-completion blockers), and the plan correctly instructs not to build the OQ-2 revive-byte fallback up front. No code was deferred under cover of those gates.

## Build & Test Results

| Module | Build | Tests (-race) | Vet | Notes |
|--------|-------|---------------|-----|-------|
| libs/atlas-constants | PASS | PASS | PASS | `go test -race ./skill/` ok; vet clean. |
| atlas-channel | PASS | PASS | PASS | `go build ./...` clean; `go test -race ./...` — all packages ok, no FAIL/panic, no race warnings; vet clean. |

Redis key guard: no changed file references `go-redis`/`redis.` (verified by grep over the diff name-list). The documented GOWORK=off dep-resolution artifact on main is therefore not a task-111 regression.

## Notes on Deviations (all benign, grounded)

- The handler comments for `warpToPosition`/`Apply` were softened (commit 38de560e8) to describe the in-map chase-warp as the OQ-1 live-verification gate rather than asserting OnRevive as a verified property — accurate and consistent with the plan's grounding rules.
- Two extra tests beyond the plan: `TestResurrection_WarpFailureIsolation` (resurrection_test.go:166) and `TestSelectDeadInRangeMapPlayers_MissingRectangleReturnsNil` (recipients_test.go:327). Additive coverage.
- The two "not implemented in mock" strings in `character/mock/processor.go:80,84` are pre-existing (asset providers), outside the task-111 diff region (line 123+), and unrelated to this task.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None required for code completion. Before merge, per CLAUDE.md the author should still run `docker buildx bake atlas-channel` from the worktree root (Task 9 Step 5) — not re-run in this audit — and the OQ-1..OQ-5 live gates remain for the PR description / on-environment verification.

---

# Backend Audit — atlas-channel (task-111)

- **Auditor:** backend-guidelines-reviewer (DOM-*/SUB-*/SEC-*)
- **Service Path:** services/atlas-channel/atlas.com/channel + libs/atlas-constants/skill
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-06-25
- **Scope:** git diff 334af4a9b..HEAD (commits 5083d0e6a..e9435e55a)
- **Build:** PASS — `go build ./...` clean (atlas-channel + libs/atlas-constants)
- **Tests:** PASS — `go test -race -count=1 ./skill/... ./character/...` all `ok`, zero FAIL, zero DATA RACE; `go vet` clean
- **Overall:** PASS (no blocking findings)

## Scope Classification

This change is **not** a classic REST/DB domain package. It adds a per-skill
*active-skill handler* (atlas-channel plugin/registry pattern) plus a new
absolute `SET_HP` channel→character Kafka **command type** (not a new topic) and
two reusable dead-target recipient selectors. There is no `model.go`/`entity.go`/
`rest.go`/`resource.go`/`administrator.go`/`provider.go` in the changed set, so the
REST/GORM-oriented DOM checks (DOM-01..05, DOM-11, DOM-16..19, EXT-*, SCAFFOLD-*,
SUB-01..04) are **N/A by design**. The applicable checks are listed below with
file:line evidence.

## Applicable Checklist Results

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor/handler accepts `logrus.FieldLogger`, not `*logrus.Logger` | PASS | resurrection/resurrection.go:30,40,49,55,72; recipients.go seams take `logrus.FieldLogger` |
| DOM-07 | No `logrus.StandardLogger()` injected | PASS | Handler is invoked with the dispatch-supplied `l`; grep of changed files shows zero `StandardLogger` |
| DOM-09 | Errors handled, not discarded with `_` | PASS | resurrection.go:84-88 (caster), 93-96 (setHP), 97-100 (warp) all check err; the two `_ =` at resurrection.go:56,60 are intentional best-effort effect broadcasts (mirrors existing skill handlers) |
| DOM-12 | No `os.Getenv()` in handler | PASS | No `os.Getenv` in any changed file; topic via `character.EnvCommandTopic` (processor.go:276) |
| DOM-13 | No cross-domain logic inline | PASS | resurrection.go delegates to `character.NewProcessor`, `portal.NewProcessor`, `session.NewProcessor`, `channelmap.NewProcessor` — no inline orchestration of foreign internals |
| DOM-14 | Handler calls processor methods, not providers/DB | PASS | resurrection.go:41,50 call `*.NewProcessor(...).SetHP/.WarpToPosition`; no provider/`db.*` calls anywhere in the diff |
| DOM-15 | No direct entity writes (`db.Create/Save/Delete`) | PASS | Zero matches in changed files; state change goes channel→character via `SET_HP` command |
| DOM-20 | Table-driven / case tests | PASS | recipients_test.go (6 cases incl. regression guard), resurrection_test.go (6 scenario tests), resurrection/recipients_test.go (3 variant cases), producer_test.go, resurrection_test.go (constants) |
| DOM-21 | Reuse libs/atlas-constants types (no redeclaration) | PASS | Skill IDs from `libs/atlas-constants/skill` (resurrection.go:23-25; constants.go adds only `GmResurrectionId=9001005` + registry wiring at constants.go:2698,3248). Bishop(2321006)/SuperGm(9101005) pre-existed (constants.go:3080,3248). Field/channel/world/point types all from atlas-constants. No service-local redeclaration. |
| DOM-23 | Kafka topic naming (no new dotted/literal topics) | PASS (N/A new topic) | `SET_HP` is a command-body `Type` string (kafka.go:18), reusing existing `character.EnvCommandTopic`; no new `COMMAND_TOPIC_*`/`EVENT_TOPIC_*` constant, no configmap/manifest change required |
| DOM-24 | Emit paths in tests are stubbed | PASS | resurrection_test.go installs no-op seams for `setHP`/`warpToPosition`/`broadcastEffects` (lines 58-74), so no real `producer.ProviderImpl` is hit; producer_test.go calls `SetHPCommandProvider(...)()` which only **builds** messages (no `Emit`), so no producer stub needed. Full race suite ran in ~1s/pkg — no 42s emit-retry stall. |
| (Immutability/Builder) | New model fields via builder, validated `Build()` | PASS | character/builder.go:138-139 `SetX/SetY`; wired into `Build()` at builder.go:184-185; `PartyRecipient` built via `NewPartyRecipientBuilder()` (recipients.go:257-263) |
| (Mock sync) | Interface change → mock updated | PASS | `Processor.SetHP` added (processor.go:42) + impl (processor.go:276) + mock (character/mock/processor.go:123) in the same change; full suite compiles & passes |
| (Symbol existence) | All referenced symbols verified to exist | PASS | `portal.WarpToPosition` (portal/processor.go:50), `AnnounceSkillUse`/`AnnounceForeignSkillUse` (socket/handler/effects.go:19,31), `inMapCharacterIdsFunc` (recipients.go:57), character `X/Y/Hp/MaxHp/Level/Id` (model.go:99-243), `field.Channel()` (atlas-constants field/model.go:32) |
| (Production wiring) | Handler reaches the binary | PASS | resurrection registered via `init()` (resurrection.go:22-26); package blank-imported in registrations.go which main.go:58 blank-imports |

## Verified Non-Issues (adversarial checks that held up)

- **Range-rectangle cast safety:** `dx < int16(lt.X())` etc. (recipients.go:251-253)
  — `point.X`/`point.Y` are `int16` (libs/atlas-constants/point/constants.go:3,5),
  so the casts are lossless and identical to the pre-existing party selector's
  filter; not a truncation bug.
- **wantDead refactor regression:** the living-only `SelectInRangePartyMembers`
  path still excludes dead members after the `wantDead` flag was threaded through
  `selectPartyMembers` — guarded by recipients_test.go
  `TestSelectInRangePartyMembers_StillExcludesDead`.
- **Seam-mutation tests + parallelism:** package-level seam vars are mutated in
  tests but restored via `t.Cleanup`; `-race` run is clean (tests are not `t.Parallel`).
- **Full-HP semantics:** `math.MaxUint16` SetHP relies on atlas-character clamping
  to effective MaxHp (resurrection.go:38-39, 93). This is a documented cross-service
  contract assumption; verifying the clamp lives in atlas-character is out of this
  diff's scope and is correctly called out as an OQ live gate, not asserted here.

## Blocking (must fix)

- None.

## Non-Blocking (observations only)

- The revive-animation behavior of warping a dead client to its own death
  coordinates is an unverified live-gate (OQ-1), documented honestly in
  resurrection.go:44-48 rather than asserted — acceptable per guidelines.
