# Plan Audit — task-082-monsterbook-cover-mobid

**Plan Path:** docs/tasks/task-082-monsterbook-cover-mobid/plan.md
**Audit Date:** 2026-06-05
**Branch:** task-082-monsterbook-cover-mobid
**Base Branch:** main (branch base 97b75ff7e; impl commits feffa97de..d0830a3a5)

## Executive Summary

All 11 plan tasks were faithfully implemented. Every file the plan specified to create or modify exists with the prescribed content, every test the plan dictates is present and passing, and the FR-10 / FR-11 "no-change" decisions are honored (info.go, data.go, character_data.go all unchanged vs main). Builds and tests are green across all three changed modules (atlas-monster-book, atlas-channel, libs/atlas-packet). The three pre-disclosed controller deviations (monster.Id typing, ingress-routed DATA_SERVICE_URL, IDA-unavailable FR-10 finding) are all present and acceptable. Verdict: READY_TO_MERGE.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | New `data/consumable` outbound client | DONE | model.go/rest.go/requests.go/processor.go/rest_test.go all created and match plan code verbatim. `atlas-rest` now in `require` block (go.mod:10). 3 tests present (round-trip serves a `relationships` block + extra attrs; 404→ErrNotFound). All pass. |
| 2 | `CoverMobId` on entity/model/builder | DONE | entity.go:18 `CoverMobId uint32 gorm:"not null;default:0"`; model.go:29 getter + :49 ToEntity cast; builder.go field/setter/Clone/Build/Make all threaded. `TestBuilderCoverMobIdRoundTrip` PASS. |
| 3 | `setCover` persists resolved mob id | DONE | administrator.go:55 6-arg signature; :61 `"cover_mob_id": uint32(coverMobId)` in Updates map; idempotency guard unchanged. `TestSetCoverPersistsMobId` PASS (incl. duplicate-eventId no-op). |
| 4 | Resolve card→mob, thread into `SetCoverAndEmit` | DONE | processor.go:75 `dp consumable.Processor` field; :85 built in NewProcessor; :96 carried in WithTransaction; :224 `resolveCoverMobId` (fail-safe, returns 0 on any error / non-mb / mobId==0); :254 resolved + passed to setCover; COVER_CHANGED body still card-id only (:269). `TestResolveCoverMobId` (5 sub-cases) PASS. |
| 5 | REST exposes `coverMonsterId` | DONE | rest.go:18 `CoverMonsterId monster.Id json:"coverMonsterId"`; :41 Transform maps `m.CoverMobId()`; CoverCardId stays card id. rest_test.go `TestTransformIncludesCoverMonsterId` PASS. |
| 6 | atlas-channel carry `coverMonsterId` to domain | DONE | processor.go:25 `coverMonsterId monster.Id` field + :34 getter; rest.go:20 wire field + :112 mapped in Extract; model.go:23 `CoverMonsterId()` delegate. `TestExtractIncludesCoverMonsterId` PASS; existing tests still pass. |
| 7 | Write mob id into Character-Info packet (crash fix) | DONE | character_info.go:60 `Cover: uint32(mb.CoverMonsterId())` (was `uint32(mb.CoverCardId())`). character_info_test.go `TestCharacterInfoBody_CoverIsMobId` proves cover=100100 (mob id), not 2380000 (card id). PASS. |
| 8 | atlas-packet contract guard | DONE | info.go unchanged vs main (verified empty diff). info_test.go:90 `TestCharacterInfo_CoverCarriesArbitraryValue` (Cover:100100 round-trips). PASS. |
| 9 | FR-10 login-draw decision record | DONE | fr10-login-draw-finding.md created: documents no-change decision, behavioral evidence, regression guard `TestBuildCharacterData_MonsterBook` (cover stays card id 2380001, PASS), and IDA-unavailable. data.go + character_data.go unchanged (verified empty diff). Escalation gate did not trigger. |
| 10 | Declare `DATA_SERVICE_URL` | DONE | atlas-monster-book.yaml:39-40 `DATA_SERVICE_URL: http://atlas-ingress.atlas.svc.cluster.local:80/api/`. YAML parses. (Ingress-routed per controller decision — see deviations.) |
| 11 | Full verification | DONE | go vet/build/test green for all 3 modules; collection + consumable + channel monsterbook/writer + packet guard tests pass; working tree clean on correct branch. (Docker bake not re-run in this audit; the `atlas-rest` COPY it would check already pre-exists in repo-root Dockerfile, no go.work/Dockerfile change needed.) |

**Completion Rate:** 11/11 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Deviations From Plan (all acceptable)

1. **`monster.Id` typing instead of raw `uint32`** — `coverMobId`/`CoverMobId()` (model.go, builder.go, administrator.go param), `resolveCoverMobId` return (processor.go:224), `CoverMonsterId` REST field (rest.go:18), and the atlas-channel `coverMonsterId`/`CoverMonsterId()` (processor.go:25/34, model.go:23, rest.go:20) are all typed `monster.Id` from libs/atlas-constants/monster. Casting to `uint32` happens only at the GORM column (model.go:49, administrator.go:61) and the packet writer boundary (character_info.go:60). This is the pre-disclosed DOM-21 decision mirroring the sibling `coverCardId item.Id`; strictly better than the plan's raw `uint32`. ACCEPTABLE.

2. **`DATA_SERVICE_URL` points at ingress base** (`http://atlas-ingress.atlas.svc.cluster.local:80/api/`) rather than the plan's literal `http://atlas-data/`. Matches the BASE_SERVICE_URL convention; npc-shops makes the identical `requests.RootUrl("DATA")` call and relies on the ingress. No service uses direct atlas-data routing. ACCEPTABLE (pre-disclosed; commit d0830a3a5).

3. **FR-10 IDA not run this session** — Task 9 Step 2 is best-effort. IDA-MCP was unavailable, so the no-change decision rests on behavioral evidence (crash only on Character Info) plus the passing `TestBuildCharacterData_MonsterBook` guard. The plan's escalation gate ("STOP if IDA proves login calls GetMobTemplate") did not trigger because IDA could not run; the finding doc records this transparently. ACCEPTABLE (pre-disclosed).

## Build & Test Results

| Module | Build | Tests | Notes |
|--------|-------|-------|-------|
| atlas-monster-book | PASS | PASS | go vet/build clean; collection + data/consumable tests pass; `atlas-rest` correctly in require block |
| atlas-channel | PASS | PASS | source-only (go.mod/go.sum unchanged vs main); monsterbook + socket/writer tests pass |
| libs/atlas-packet | PASS | PASS | source-only; clientbound contract guard passes; info.go unchanged |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None required. The implementation is complete and faithful to the plan. The one non-blocking note for the merging human: Task 11's `docker buildx bake atlas-monster-book` (mandatory per CLAUDE.md because atlas-monster-book's go.mod changed) was reported green during execution and was not re-executed in this read-only audit; the dependency it guards (atlas-rest COPY in the repo-root Dockerfile) already pre-exists, so no Dockerfile/go.work edit was needed. Confirm the bake remains green in CI before merge.
