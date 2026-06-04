# Plan Audit — task-079-monster-book-client-display

**Plan Path:** docs/tasks/task-079-monster-book-client-display/plan.md
**Audit Date:** 2026-06-03
**Branch:** task-079-monster-book-client-display
**Base Branch:** main (base commit 92fadfdb0)

## Executive Summary

All 10 plan tasks were faithfully implemented. The 9 implementing tasks each produced exactly one commit matching the plan's commit message; Task 10 is verification-only (no commit, as expected). Every task's tests exist, assert real behavior, and pass. All four documented intentional deviations were verified as correct (corrected cover bytes, the `(GMS && >28 && <=87) || JMS` book gate preserving v28-absence and excluding v95, the doc-comment cleanup, and the `SetSp("0")` test fix). `go build`, `go vet`, and `go test -race` are clean across all three changed modules (`libs/atlas-constants`, `libs/atlas-packet`, `services/atlas-channel`). No TODO/stub/panic markers were introduced. The repo-wide redis-key-guard FAIL is pre-existing in untouched services and unrelated to this task (which adds no Redis code).

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `MonsterBookCardBase` constant | DONE | `libs/atlas-constants/item/constants.go:45-48` (`MonsterBookCardBase = Id(2380000)`); test `constants_test.go:66-76` asserts value + `==Classification(238)*10000`. Commit 94f7b4784. |
| 2 | Packet types + encode/decode rewrite | DONE | `libs/atlas-packet/character/data.go:83-96` (`MonsterBookCard`/`MonsterBookData`), `:108` (`MonsterBook` field), `:692-715` (encode cover as full item id, mode 0, short count, per-card `cardId-base`+level; symmetric decode). Tests `data_test.go:130-169`. Commit 2afedc469. |
| 3 | Split version gate | DONE | encode `data.go:147` and decode `data.go:206` both `(GMS && >28 && <=87) || JMS`; tail kept `(GMS && >28) || JMS` at `:152`/`:211`. Round-trip test `data_test.go:171-215` predicate matches. Commit 4f2f23e23 (message records v95-tail verification finding). |
| 4 | atlas-channel `/cards` REST consumption | DONE | `monsterbook/rest.go:67-101` (`CardRestModel`+`ExtractCard`), `requests.go:14-15,30-32` (`CardsResource`+builder), `processor.go:34-41,83-93` (`Card` domain type, interface methods, impl via `SliceProvider`). 3 tests `rest_test.go:191-256`. Commit d83f4f97d. |
| 5 | `character.Model` owned-card field | DONE | `model.go:64` field, `:363-372` getter/setter; `builder.go:62,111,153-156,202` threaded through struct/Clone/setter/Build. Test `builder_test.go:238-252`. Commit b044aeba6. |
| 6 | Unified `MonsterBookDecorator` | DONE | `processor.go:34` interface, `:173-190` impl (cover then cards, both fail-open with Debug log). Mock `mock/processor.go:75`, caller `character_info_request.go:31`. Doc comment `model.go:352` updated (deviation #3). Tests `processor_test.go:254-300` (fail-open + populates). Commit cd5ca3119. |
| 7 | `BuildCharacterData` populates book | DONE | `character_data.go:92-100` maps cover + cards into `cd.MonsterBook`. New test `character_data_test.go` with `SetSp("0")` (deviation #4). Commit f3dd185d5. |
| 8 | Wire decorator into login chain | DONE | `kafka/consumer/session/consumer.go:166` appends `cp.MonsterBookDecorator` to `GetById` chain. Commit 1f0499d6c. |
| 9 | CharacterInfo cover thread-through | DONE | `clientbound/info.go:34` field, `:36-43` ctor trailing arg, `:100` encode `m.monsterBookCover`, `:120` getter, `:166` decode. Caller `socket/writer/character_info.go:55` passes `uint32(c.CoverCardId())`. All existing `NewCharacterInfo` call sites updated with trailing `0`. Test `info_test.go:76-84`. Commit fd4179d20. |
| 10 | Full verification | DONE | Verification-only (no commit expected). Task 3 commit records the v95 tail finding. Build/vet/race-test reproduced clean by this audit (see below). |

**Completion Rate:** 10/10 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Intentional Deviations — Verified Correct

1. **Task 2 cover bytes (corrected).** Implementation writes the cover as the full item id (`w.WriteInt(uint32(m.MonsterBook.CoverCardId))`, `data.go:693`). The corrected test asserts `0xE1 0x50 0x24 0x00` = LE of 2380001 (0x2450E1), not the plan's erroneous `61 4C 24 00`. Comment in the test was also corrected. CONFIRMED.
2. **Task 3 gate.** Both encode (`data.go:147`) and decode (`data.go:206`) gates use `(GMS && >28 && <=87) || JMS`, preserving v28-absent (original `>28` lower bound) and excluding v95. The newYear/area/trailing tail keeps `(GMS && >28) || JMS` (`:152`/`:211`). Test predicate (`data_test.go:194`) matches exactly. Verified across all 5 variants (v28 absent, v83/v87 present, v95 absent, JMS present) — all round-trip green. CONFIRMED.
3. **Task 6 doc-comment cleanup.** `CoverCardId` doc at `model.go:352` now references `MonsterBookDecorator`. Repo-wide grep for `MonsterBookCoverDecorator` returns zero matches. CONFIRMED.
4. **Task 7 `SetSp("0")`.** Present in `character_data_test.go` to avoid the pre-existing `BuildCharacterData` panic. CONFIRMED.

## Skipped / Deferred Tasks

None. The plan's explicit out-of-scope items (`kafka/consumer/monsterbook/consumer.go`, `socket/handler/monster_book_cover.go`, atlas-monster-book persistence) remain untouched as intended.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| libs/atlas-constants | PASS | PASS | `go test -race ./item/` ok |
| libs/atlas-packet | PASS | PASS | `go vet` clean; `go test -race ./character/...` all packages ok; monster-book + v95 round-trip subtests all pass |
| services/atlas-channel | PASS | PASS | `go build ./...`, `go vet ./...` clean; tests ok for monsterbook, character, socket/handler, socket/writer; session consumer has no test files (expected) |

Additional checks:
- TODO/FIXME/panic/501/"not implemented" scan on the full diff: none introduced.
- redis-key-guard: pre-existing FAIL flags untouched services (atlas-saga-orchestrator, atlas-party-quests) plus GOWORK=off cross-module import noise; no task-079 file is flagged. This task adds no Redis usage. Not a regression.
- Docker bake (`docker buildx bake atlas-channel`) not run in this audit environment; called out as the only Task 10 sub-step not independently reproduced here.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None blocking. Optional follow-ups:
1. Run `docker buildx bake atlas-channel` from the worktree root before PR (Task 10 Step 4) — only Go `go.mod` for atlas-channel was indirectly affected via new package imports; bake confirms the shared Dockerfile COPY coverage. Not reproduced in this audit environment.
2. Complete the in-game acceptance checks (Task 10 Step 6 / PRD §10), notably the GMS v95 no-desync login, since the v95 CharacterData tail retention was defaulted (not IDA-verified) per the Task 3 commit message.

---

# Backend Guidelines Audit — task-079-monster-book-client-display

- **Service Path:** services/atlas-channel/atlas.com/channel (+ libs/atlas-constants, libs/atlas-packet)
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-06-03
- **Reviewer:** backend-guidelines-reviewer (adversarial)
- **Build:** PASS (`go build ./...` clean in channel; libs build clean)
- **Vet:** PASS (`go vet` clean on changed packages)
- **Tests:** PASS — all channel packages `ok`; `libs/atlas-constants/item`, `libs/atlas-packet/character/...` `ok`
- **Overall:** PASS

## Scope Classification

This change does NOT add a DB-backed DOM domain package. The new `monsterbook`
package is an **External HTTP Client** (REST consumer of atlas-monster-book) plus
a Kafka producer. It has no `model.go`/`entity.go`/`administrator.go`/`resource.go`
and exposes no REST endpoints, so the entity/builder/administrator/RegisterInputHandler
DOM checks are N/A. The applicable checklists are: immutable-model + Builder
(`character`), External-HTTP-Client (EXT-01..04), functional/provider patterns,
multi-tenancy/context, and DOM-21 (constant reuse).

## Checklist Results

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | No reinvented atlas-constants types | PASS | New `MonsterBookCardBase` added to the shared lib at `libs/atlas-constants/item/constants.go:48`, not in the service; tied to existing `ClassificationConsumableMonsterCard` via test `constants_test.go:72`. All card/cover ids use `item.Id` (`monsterbook/rest.go:18,70`, `data.go:86,93`). No service-local id/classification type declared. |
| Immutable model + Builder | New field threaded through every path | PASS | `character/model.go:64` field; getter/setter `model.go:365-371`; builder struct `builder.go:62`, `CloneModel` `builder.go:111`, `SetMonsterBookCards` `builder.go:153-156`, `Build()` `builder.go:202`. Clone carries the field, so no decorator loss across rebuilds. |
| Processor Interface+Impl | PASS | `monsterbook/processor.go:45-52` (interface), `:54-66` (impl + `NewProcessor(l,ctx)`); `character/processor.go:32` adds `MonsterBookDecorator` to interface and `:177-194` impl. |
| Mock updated for interface change | PASS | `character/mock/processor.go:75-77` implements `MonsterBookDecorator`. Build of dependent test packages confirms interface satisfied. |
| Provider / lazy evaluation | PASS | `monsterbook/processor.go:62` uses `requests.Provider`; `:87` uses `requests.SliceProvider[...](..., model.Filters[Card]())`. Both return `model.Provider`; execution deferred until `()` in `GetByCharacterId`/`GetCardsByCharacterId`. |
| Multi-tenancy / context | PASS | `monsterbook/processor.go:51` binds `tenant.MustFromContext(ctx)`; producer keys on `p.t.Id()` (`:56`). REST calls pass `p.ctx` to `requests.Provider` (tenant header propagated by atlas-rest). No manual tenant header parsing. |
| Fail-open error handling (decorator) | PASS | `character/processor.go:181-193`: cover-fetch error logs Debug and returns undecorated model; card-fetch error logs Debug and returns cover-only model. Login proceeds on any REST failure. Mirrors existing `PartyDecorator`/`InventoryDecorator`. |
| EXT-01 | JSON:API relationship stubs on target structs | PASS | `CollectionRestModel` `rest.go:58-65`; `CardRestModel` `rest.go:94-95` — both implement `SetToOneReferenceID`/`SetToManyReferenceIDs` (+ marshal stubs). |
| EXT-02 | httptest-backed integration test | PASS | `rest_test.go:129-168` (`GetByCharacterId_RoundTrip`) and `:213-242` (`GetCardsByCharacterId_RoundTrip`) stand up `httptest.NewServer` with real JSON:API bodies and assert populated domain structs through the unmarshal path — not FakeClient mocks. |
| EXT-03 | 404 distinguished from other failures | PASS | Transport maps 404→`requests.ErrNotFound` at `libs/atlas-rest/requests/get.go:77-78`; other errors bubble with original type. `rest_test.go:170-189` asserts `errors.Is(err, requests.ErrNotFound)` for the collection endpoint. Decorator treats all errors as fail-open, which is correct for a login-display path. |
| EXT-04 | Service URL via `RootUrl(domain)` | PASS | `monsterbook/requests.go:17` `requests.RootUrl("MONSTER_BOOK")`; falls back to `BASE_SERVICE_URL` (`libs/atlas-rest/requests/url.go:14-19`). No hardcoded DNS. Consistent with all 30+ existing `RootUrl(...)` callers in atlas-channel. |
| Packet encode/decode symmetry | PASS | `data.go:700-708` encode (cover full id, mode byte 0, short count, `cardId-MonsterBookCardBase` + level) and `:713-721` decode are inverse operations; round-trip test `data_test.go` passes. `CharacterInfo` cover wired symmetrically `clientbound/info.go:100`/`:166`. |
| SEC-* | Auth/redirect/secret concerns | N/A / PASS | Not an auth service. No secrets, tokens, redirects, or `os.Getenv` introduced in the new code. URL composition is env-driven via the shared helper. |

## Findings

### Minor (non-blocking)

- **M1 — Cards 404 test does not assert `ErrNotFound` specifically.**
  `monsterbook/rest_test.go:243-256` (`GetCardsByCharacterId_NotFound`) asserts
  only `err != nil`, whereas the collection counterpart asserts
  `errors.Is(err, requests.ErrNotFound)`. The production transport DOES map 404
  correctly, so behavior is right; the test is just weaker than its sibling.
  Tightening it to `errors.Is(err, requests.ErrNotFound)` would make the EXT-03
  guarantee explicit on both endpoints. Not blocking.

- **M2 — Two sequential REST round-trips per login decorator.**
  `MonsterBookDecorator` (`character/processor.go:177-194`) issues
  `GetByCharacterId` then `GetCardsByCharacterId` serially on every login. This
  matches the existing per-decorator pattern and is fail-open, so it is
  acceptable; noted only as a latency observation, not a guideline violation.

### Blocking

None.

## Verdict

PASS. Build, vet, and tests are clean across all three changed modules. The new
constant lives in the shared `libs/atlas-constants/item` package and is reused
(DOM-21 satisfied). The immutable `character.Model` field is threaded through the
struct, `CloneModel`, setter, and `Build()` with no clone loss. The `monsterbook`
REST client satisfies the full EXT-01..04 external-client checklist including
relationship stubs, httptest round-trip tests, 404 discrimination, and
`RootUrl`-based URL composition. Multi-tenancy and context propagation are
correct. Only two minor, non-blocking observations.
