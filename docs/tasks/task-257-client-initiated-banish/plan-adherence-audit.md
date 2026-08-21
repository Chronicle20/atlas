# Plan Audit — task-257-client-initiated-banish

**Plan Path:** docs/tasks/task-257-client-initiated-banish/plan.md
**Audit Date:** 2026-08-21
**Branch:** task-257-client-initiated-banish
**Base Branch:** main (commit range audited: 1461bfc96..4cf6d2a3c)
**Scope:** Tasks 1-4 (Task 5, the verification gate, is running separately and was not executed by this audit)

## Executive Summary

Tasks 1-4 were implemented faithfully against the plan: every named file was touched as specified, every producer/consumer/handler shape matches the plan's code blocks essentially verbatim, and all four Global Constraints (fail-closed validation, warp-then-message ordering, no direct socket write, no `*_testhelpers.go` files) hold in the diff. `go build ./... && go test ./...` pass clean in all three touched modules (atlas-portals, atlas-channel, atlas-monsters), with no `FAIL` lines anywhere. The one pre-existing known issue — the "random spawn when all unset" subtest in `atlas-portals/portal/consumer_test.go` proving only absence-of-log rather than a successful drain — is confirmed still present and still a cosmetic-only gap (the real fallback path is independently proven by `TestWarpByName_MissFallsBackToRandomSpawn`).

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `atlas-portals` — resolve a warp by portal name | DONE | `portal/kafka.go` adds `TargetPortalName string \`json:"targetPortalName"\`` (no omitempty, per constraint); `portal/processor.go` adds `WarpByName` interface method + impl matching the plan's resolve-warn-fallback shape; `portal/consumer.go` inserts the precedence branch between `TargetPortalId` and random-spawn; `portal/warp_by_name_test.go` (new, 178 lines) implements `TestWarpByName_Hit`/`TestWarpByName_MissFallsBackToRandomSpawn`; `consumer_test.go` adds `TestHandleWarpCommand_Precedence` (4 subtests); `docs/kafka.md` row added. `go test ./portal/... -run 'TestHandleWarpCommand_Precedence' -v` — all 4 subtests PASS. |
| 2 | `atlas-channel` — emit the `BANISH` command | DONE | `kafka/message/monster/kafka.go` adds `CommandTypeBanish` and `BanishCommandBody`; `monster/producer.go` adds `BanishCommandProvider` keyed on character id (verified by `TestBanishCommandKeysOnCharacter`, PASS); `monster/processor.go` adds `Banish`; `socket/handler/mob_banish_player.go` replaces the deferred comment with `_ = monster.NewProcessor(l, ctx).Banish(s.Field(), s.CharacterId(), p.MobTemplateId())` — confirmed `MobTemplateId()` is a real accessor on `serverbound.MobBanishPlayer` via `go doc`; `docs/kafka.md` updated. `producer_mock` regenerated in `monster/mock/processor.go`. Tests: `TestBanishCommandProviderShape`, `TestBanishCommandKeysOnCharacter` — both PASS. |
| 3 | `atlas-monsters` — banish plumbing | DONE | New `kafka/message/system_message/kafka.go` mirrors the party-quests template with matching JSON tags; `monster/disease.go` widens `warpBody`/`warpCommandProvider` with `TargetPortalName string \`json:"targetPortalName,omitempty"\`` (omitempty on producer side, per constraint) and adds `sendMessageProvider`; `monster/processor.go`'s pre-existing `executeBanish` call site updated to pass `ma.Banish().PortalName`; `monster/information/builder.go` adds `banish Banish` field + `SetBanish`. Tests `TestWarpCommandProviderCarriesPortalName`, `TestWarpCommandProviderOmitsEmptyPortalName`, `TestSendMessageProviderShape`, `TestModelBuilderSetBanish` all present in `banish_producer_test.go` and pass as part of the full module test run. |
| 4 | `atlas-monsters` — validate and execute the banish | DONE | `kafka/consumer/monster/kafka.go` adds `CommandTypeBanish` + `banishCommandBody`; `kafka/consumer/monster/consumer.go` registers `handleBanishCommand` and defines it exactly per the plan (routes through `monster.NewProcessor(...).Banish`, logs+swallows rejection at Debug); `monster/processor.go` adds `monsterInformation` test hook, `banishCharacter` shared executor (warp-then-message, message failure logged/swallowed), `Banish` (fail-closed: field membership check → info fetch → zero-map check, each returning a named error and taking no action on failure), and rewrites `executeBanish` onto the shared executor. `docs/kafka.md` gets the `BANISH` block, the `targetPortalName` field in `WARP`, and the new `COMMAND_TOPIC_SYSTEM_MESSAGE` section. `banish_test.go` (324 lines) implements the full table: 4 fail-closed rejections (`TestBanish`), portal-name present/absent, message present/absent, and `TestExecuteBanish_ConvergesOnSharedExecutor` proving skill-129 and client-initiated paths emit byte-identical events in the same order. `go test ./monster/... -run TestBanish -v` and the full module suite both PASS. |
| 5 | Verification gate | NOT_APPLICABLE (out of audit scope) | Explicitly excluded per task instructions — running separately. |

**Completion Rate:** 4/4 in-scope tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

Note: all 41 checkbox items across Tasks 1-4 in `plan.md` remain unchecked (`- [ ]`) despite the underlying work being present and verified in the diff — this is a plan-bookkeeping gap, not a code gap, and does not affect the Task Completion verdicts above, which are evidenced directly against the diff and test runs.

## Skipped / Deferred Tasks

None in scope. No task was skipped, partially implemented, or silently deferred.

### Known non-blocking test gap (carried forward from Task 1 review)

`services/atlas-portals/atlas.com/portals/portal/consumer_test.go:102-106` (subtest `"random spawn when all unset"` inside `TestHandleWarpCommand_Precedence`) — confirmed still present as described. `setupMockDataServerForConsumer` matches only the exact request path including query string, while the random-spawn drain path (`portal/requests.go`: `DrainProvider`) appends its own `page[number]`/`page[size]` query parameters. The bare `/api/data/maps/200000000/portals` registered by the test therefore never matches the actual drain request, which 404s. The subtest passes only on the absence of `portal [` / `position` substrings in the debug log — it does not prove the drain succeeds. This is cosmetic: the real fallback behavior (successful drain landing on the fallback portal, with the correct warning) is independently and fully proven by `TestWarpByName_MissFallsBackToRandomSpawn` in `warp_by_name_test.go`, which uses the recording mock and asserts both the warning content and that the drain path was actually requested. No functional risk; still stands as a minor test-quality gap, not a blocking finding.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-portals (`services/atlas-portals/atlas.com/portals`) | PASS | PASS | `go build ./...` clean; `go test ./...` all packages `ok`, including `portal` (8.9s, includes new `TestWarpByName_*` and `TestHandleWarpCommand_Precedence`). |
| atlas-channel (`services/atlas-channel/atlas.com/channel`) | PASS | PASS | `go build ./...` clean; `go test ./...` all packages `ok`, no FAIL lines; `TestBanishCommandProviderShape`/`TestBanishCommandKeysOnCharacter` PASS. |
| atlas-monsters (`services/atlas-monsters/atlas.com/monsters`) | PASS | PASS | `go build ./...` clean; `go test ./...` all packages `ok` (monster package 21s, information package 15.9s); `TestBanish*`/`TestExecuteBanish_ConvergesOnSharedExecutor` PASS. |

## Global Constraints Check

| Constraint | Status | Evidence |
|---|---|---|
| Fail-closed validation (no banish on packet alone) | HOLDS | `monster.ProcessorImpl.Banish` in `atlas-monsters/monster/processor.go`: rejects when no live monster of the template is in the field, when the information fetch fails, and when `MapId == 0` — each path returns a named error and calls neither `banishCharacter` nor any emit. Proven by `TestBanish`'s 4-row table, all asserting `err != nil` and `len(*events) == 0`. |
| Warp-then-message ordering | HOLDS | `banishCharacter` emits `WARP` first and returns immediately on its failure (message never sent); a message emit failure after a successful warp is logged at `Warn` and swallowed, matching the plan's stated behavior exactly. Proven by `TestBanish_MessagePresent` and `TestExecuteBanish_ConvergesOnSharedExecutor` asserting event order `[0]=WARP, [1]=SEND_MESSAGE`. |
| No direct socket write | HOLDS | `mob_banish_player.go` takes `_ writer.Producer` (discarded) and only forwards to the Kafka-backed `monster.Processor.Banish`; no `writer.` call site references the parameter. |
| No `*_testhelpers.go` files | HOLDS | `git diff --name-only` over the full range contains no path matching `*_testhelpers.go`; new test files (`warp_by_name_test.go`, `producer_banish_test.go`, `banish_producer_test.go`, `banish_test.go`) all reuse existing per-package harnesses (`setupMockDataServer`/`setupMockDataServerForConsumer`, `newRecordingProcessorWithBodies`, `magnetTestField`) rather than introducing new helper-constructor files. |
| Do-not-touch list (`atlas-data`, `libs/atlas-packet`, `atlas-channel`'s `data/monster` projection, `atlas-channel`'s local `WarpBody` copy, deploy manifests) | HOLDS | `git diff --name-only 1461bfc96..4cf6d2a3c` grepped against `atlas-data\|libs/atlas-packet\|atlas-channel/.*data/monster\|WarpBody\|deploy` returns no matches. `serverbound.MobBanishPlayer` (from `libs/atlas-packet`) is *used* by the new handler line but the library itself is unmodified. |
| `TargetPortalName` JSON tag asymmetry (producer `omitempty`, consumer none) | HOLDS | Producer (`atlas-monsters/monster/disease.go`): `TargetPortalName string \`json:"targetPortalName,omitempty"\``, proven never emitted when empty by `TestWarpCommandProviderOmitsEmptyPortalName` and `TestBanish_PortalNameAbsent`. Consumer (`atlas-portals/portal/kafka.go`): `TargetPortalName string \`json:"targetPortalName"\`` with no `omitempty`, matching plan §Global Constraints line 2 exactly. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending Task 5's separate flagless `tools/verify.sh` run and the required code-review step before opening a PR)

## Action Items

1. (Optional, non-blocking) Update the 41 checkboxes in `plan.md` from `- [ ]` to `- [x]` for Tasks 1-4 to keep the plan document's own bookkeeping consistent with the verified state of the branch.
2. (Optional, non-blocking) If a future pass touches `atlas-portals/portal/consumer_test.go`, consider fixing `setupMockDataServerForConsumer` to strip/ignore pagination query params (or matching by path prefix) so the "random spawn when all unset" subtest actually exercises the drain rather than 404ing silently.
