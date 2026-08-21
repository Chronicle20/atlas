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

---

# Backend Audit — task-257-client-initiated-banish

- **Service Path:** services/atlas-channel, services/atlas-monsters, services/atlas-portals
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-21
- **Commit range:** 1461bfc96..4cf6d2a3c
- **Build:** PASS (all three modules)
- **Tests:** 1812 passed, 0 failed (`atlas-channel` 1456, `atlas-monsters` 311, `atlas-portals` 45; per-test counts via `go test ./... -v | grep -c '^--- PASS/FAIL'`)
- **Overall:** NEEDS-WORK

## Build & Test Results

```
services/atlas-channel/atlas.com/channel:  go build ./... -> exit 0 ; go test ./... -count=1 -> all ok, no FAIL
services/atlas-monsters/atlas.com/monsters: go build ./... -> exit 0 ; go test ./... -count=1 -> all ok, no FAIL
services/atlas-portals/atlas.com/portals:  go build ./... -> exit 0 ; go test ./... -count=1 -> all ok, no FAIL
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | Fired | `model.go` present in changed packages `atlas-channel/monster`, `atlas-monsters/monster`, `atlas-monsters/monster/information`, `atlas-portals/portal` |
| FILE placement (FILE-01..06) | Fired | Every changed Go package is in scope unconditionally |
| SUB sub-domain (SUB-01..04) | N/A | No changed package has `resource.go` without `model.go` — `atlas-monsters/monster` has both (domain, not sub-domain); the support packages (`kafka/consumer/monster`, `kafka/message/system_message`, `socket/handler`, `kafka/message/monster`) have no `resource.go` at all |
| REST (DOM-06..09,12..15,17..19,32) | Fired | `processor.go` changed in `atlas-channel/monster`, `atlas-monsters/monster`, `atlas-portals/portal` |
| Constants reuse (DOM-21) | Fired | Diff declares `BanishCommandBody`, `banishCommandBody`, `system_message.Command[E]`/`SendMessageBody` |
| Testing (DOM-10,20,24,33) | Fired | Diff adds/changes `_test.go` files in all three services and re-signs the `monster.Processor` / `portal.Processor` interfaces |
| Cache (DOM-29) | N/A | No `cache.go` touched, no processor/struct gains cached state |
| Messaging (DOM-30) | Fired (family) — rule N/A | `producer.go`/`AndEmit`-adjacent code changed (`atlas-channel/monster/producer.go`, `atlas-monsters/monster/disease.go`), but the Banish operation performs no database write — `information.Model` lookups and `Registry` reads are in-memory, not GORM |
| Multi-tenancy (DOM-31) | Fired | `rest.go` present in all three changed domain packages (package-characteristic trigger) |
| Migration hygiene (DOM-34,35) | N/A | No symbol moved between a service and `libs/atlas-*` |
| Deploy & topics (DOM-22,23) | N/A | No new `libs/atlas-*` module; `COMMAND_TOPIC_SYSTEM_MESSAGE` already exists in `deploy/k8s/base/env-configmap.yaml:92` and all three overlay generators — reused, not added or renamed |
| Runtime safety (DOM-26) | N/A | `git diff -- '*.go'` contains no added `go ` statement |
| Channel wire values (DOM-25) | N/A | `mob_banish_player.go` forwards a client-supplied template id as a domain `uint32`, not a dispatcher/fail-reason wire byte; the pre-existing `system_message` consumer that turns `messageType` into a socket write is untouched by this diff |
| Resilience (DOM-27,28) | N/A | No `resource.go` handler or `model.Decorator` touched |
| External clients (EXT-01..04) | N/A | No `requests.GetRequest[T]`/`PostRequest[T]` call added |
| Scaffolding (SCAFFOLD-01..09) | N/A | No new service directory; `MobBanishPlayerHandleFunc` registration in `main.go` pre-existed (unchanged in diff) |
| Security (SEC-01..04) | N/A | None of the three services in scope handle auth/tokens/redirects/secrets |
| `patterns-provider.md` (foundational) | Opened | `BanishCommandProvider`, `warpCommandProvider`, `sendMessageProvider` all return `model.Provider[[]kafka.Message]`; each is a thin, lazily-evaluated `producer.SingleMessageProvider(key, value)` wrapper matching every sibling provider already in the same file — no deviation found |
| `patterns-functional.md` (foundational) | N/A | No curried constructor/decorator/model combinator defined |

## Checklist Results

### atlas-channel/kafka/message/monster (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File responsibilities | PASS | `kafka.go` adds only `CommandTypeBanish` const and `BanishCommandBody` struct — a DTO-only Kafka envelope package, consistent with every sibling `kafka/message/<domain>/kafka.go` in the repo; no catch-all mixing (`services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go:23,127-140`) |
| DOM-21 | No redeclared constant/type | PASS | `BanishCommandBody.CharacterId`/`MonsterTemplateId` are raw `uint32`, matching every sibling command body in the same file (`kafka.go:36,42,70,85,105,124`) — no `libs/atlas-constants` equivalent exists for a "monster template id" wire field |

### atlas-channel/monster (domain)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor interface/impl in `processor.go` | PASS | `Banish` interface method and impl both added to `processor.go:35,177-181` |
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` unchanged, `monster/processor.go:43` |
| DOM-01 | `builder.go` present, `NewBuilder()`-equivalent, validating `Build()` | N/A | `builder.go` untouched by this diff; `Banish` constructs no new `Model` |
| DOM-02/03 | `ToEntity()`/`Make()` in `entity.go` | N/A | Package has no `entity.go` (channel-side monster is a Kafka relay model, not DB-backed) |
| DOM-04/05 | `Transform`/`TransformSlice` in `rest.go` | N/A | `rest.go` untouched by this diff; `Banish` introduces no new `RestModel` |
| DOM-11 | `provider.go` lazy evaluation | N/A | Package has no `provider.go` |
| DOM-16 | `administrator.go` for writes | N/A | Package performs no DB create/update/delete |
| DOM-31 | Tenant/trace travel in context only | PASS | `BanishCommandProvider`/`Banish` carry only `characterId`, `monsterTemplateId`, and the envelope's `field.Model` — no tenant/trace field added to any wire body (`producer.go:225-245`) |
| DOM-33 | Mock updated alongside interface change | PASS | `monster/mock/processor.go:28,151-156` adds `BanishFunc` + `Banish` impl in the same diff |
| DOM-20 | Tests table-driven | FAIL | `producer_banish_test.go:12` (`TestBanishCommandProviderShape`) and `:47` (`TestBanishCommandKeysOnCharacter`) are both single-scenario `func TestX(t *testing.T)` — neither uses `tests := []struct{...}` + `t.Run` |
| DOM-24 | Emit path stubbed in tests | N/A | `producer_banish_test.go` calls `BanishCommandProvider(...)()` directly — a pure message-builder, never `producer.ProviderImpl`/`AndEmit`/`message.Emit` |

### atlas-channel/socket/handler (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File responsibilities | PASS | `mob_banish_player.go` is a single handler function only; no mixed responsibility |
| DOM-25 | No client wire byte literal | PASS | `p.MobTemplateId()` is forwarded as a domain `uint32`, not a dispatcher mode/fail-reason byte resolved from a literal; `mob_banish_player.go:20` |
| (informal) error handling | consistent with sibling pattern | PASS | `_ = monster.NewProcessor(l, ctx).Banish(...)` discards the error the same way the existing `monster_damage_friendly.go:21` does — no guideline requires socket handlers (non-`resource.go`) to check this return |

### atlas-monsters/kafka/consumer/monster (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File responsibilities | PASS | `handleBanishCommand` added to `consumer.go:232-239`; `banishCommandBody` added to `kafka.go:168-180` — correct split, no catch-all |
| DOM-21 | No redeclared constant/type | PASS | `banishCommandBody` fields are raw `uint32`, consistent with every sibling command body in `kafka.go` |

### atlas-monsters/kafka/message/system_message (new, support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File responsibilities | PASS | New package holds only `Command[E]`/`SendMessageBody` DTOs and the two topic/type consts — a DTO-only Kafka envelope package, matching the sibling pattern the file's own doc-comment cites (`kafka.go:1-6`) |
| DOM-22/23 | Topic env var lifecycle | N/A | `COMMAND_TOPIC_SYSTEM_MESSAGE` is reused, not newly declared — already present in `deploy/k8s/base/env-configmap.yaml:92` and all three overlay generators (`main:128`, `pr:248`, `pr-sparse:414`); this diff touches no `deploy/` file |

### atlas-monsters/monster (domain)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor interface/impl in `processor.go` | PASS | `Banish` interface method (`processor.go:73`), `banishCharacter` executor (`:1265-1279`), `Banish` impl (`:1284-1311`) all in `processor.go` |
| FILE-01 (producer placement) | Kafka message-creation providers | PASS | `warpCommandProvider` (pre-existing, signature widened) and new `sendMessageProvider` both live in `disease.go`, the pre-existing feature-scoped provider file for this exact command family — not a new catch-all |
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`, `processor.go:108` |
| DOM-33 | Mock updated alongside interface change | N/A | `monster.Processor` has no mock package anywhere in `atlas-monsters` (confirmed by `grep -rn "type.*Mock struct"` returning no hit for this package) — nothing to update |
| DOM-31 | Tenant/trace travel in context only | PASS | `BanishCommandBody`/`warpBody`/`SendMessageBody` carry only domain ids and strings — no tenant/trace field |
| DOM-20 | Tests table-driven | FAIL | `banish_producer_test.go:18,48,70,103` (4 single-scenario funcs) and `banish_test.go:104,150,186,228,260` (5 single-scenario funcs) are not table-driven. `banish_test.go:31` (`TestBanish`) **is** correctly table-driven (`tests := []struct{...}` + `t.Run`, lines 33-73) |
| DOM-24 | Emit path stubbed in tests | N/A | All new tests build `ProcessorImpl` via `newRecordingProcessorWithBodies` (`processor_test.go:236`), which injects a fake `emitter` closure — no real `producer.ProviderImpl`/Kafka path is reached |
| DOM-10 | GORM test setup calls `RegisterTenantCallbacks` | N/A | No test opens a GORM DB directly; `Registry` is in-memory |

### atlas-monsters/monster/information (domain)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` — `NewBuilder()`-equivalent, fluent setters, validating `Build()` | WARN | `builder.go:16` (`NewModelBuilder`) and fluent `SetBanish` (`:59-63`) are present, matching the codebase-wide `NewModelBuilder` naming convention. `Build()` (`:69-84`) performs no invariant validation — it only defaults nil slices. This is a genuine gap against the rule's literal text ("a `Build()` that enforces invariants"), but `information.Model` is a read-only template value object (skills/attacks/hpRecovery/boss/resistances/banish) with no combination of fields that is actually invalid, so there is no invariant for `Build()` to enforce. Flagged non-blocking rather than FAIL because the diff only added a field+setter to a pre-existing, already-unvalidated builder; it introduced no new invariant to skip. |
| FILE-01..06 | File responsibilities | PASS | `builder.go` carries only builder responsibility; no mixing |

### atlas-portals/portal (domain)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor interface/impl in `processor.go` | PASS | `WarpByName` interface method (`processor.go:29`) and impl (`:142-154`) added to `processor.go` |
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`, `processor.go:39` |
| DOM-33 | Mock updated alongside interface change | PASS | `portal/mock/processor.go:20,80-85` adds `WarpByNameFunc` + `WarpByName` impl in the same diff |
| DOM-31 | Tenant/trace travel in context only | PASS | `warpBody.TargetPortalName` (`kafka.go:39-44`) carries only a portal name string — no tenant/trace field |
| DOM-01/02/03/04/05/11/16 | Domain structure | N/A | `builder.go`/`entity.go`/`rest.go`/`provider.go`/`administrator.go` all untouched by this diff; `WarpByName` introduces no new `Model`/`RestModel`/`Entity` |
| DOM-20 | Tests table-driven | FAIL | `warp_by_name_test.go:94` (`TestWarpByName_Hit`) and `:126` (`TestWarpByName_MissFallsBackToRandomSpawn`) are single-scenario, not table-driven |
| DOM-20 | Tests table-driven (new addition only) | PASS | `consumer_test.go:231` (`TestHandleWarpCommand_Precedence`, added by this diff) uses `tests := []struct{...}` + `t.Run` (lines 250-259, 293-...) — the file's other `Test*` functions predate this diff and are out of scope |
| DOM-24 | Emit path stubbed in tests | PASS | `portal/testmain_test.go:10-13` installs `producertest.InstallNoop()` in `TestMain`, which governs the whole `portal`/`portal_test` test binary including the new `warp_by_name_test.go` (external `portal_test` package) and the `consumer_test.go` additions |

## Not evaluable from the diff

- none

## Summary

### Blocking (must fix)

- DOM-20: `services/atlas-channel/atlas.com/channel/monster/producer_banish_test.go:12,47` — `TestBanishCommandProviderShape` and `TestBanishCommandKeysOnCharacter` are single-scenario tests, not table-driven.
- DOM-20: `services/atlas-monsters/atlas.com/monsters/monster/banish_producer_test.go:18,48,70,103` — `TestWarpCommandProviderCarriesPortalName`, `TestWarpCommandProviderOmitsEmptyPortalName`, `TestSendMessageProviderShape`, `TestModelBuilderSetBanish` are single-scenario, not table-driven.
- DOM-20: `services/atlas-monsters/atlas.com/monsters/monster/banish_test.go:104,150,186,228,260` — `TestBanish_PortalNamePresent`, `TestBanish_PortalNameAbsent`, `TestBanish_MessagePresent`, `TestBanish_MessageAbsent`, `TestExecuteBanish_ConvergesOnSharedExecutor` are single-scenario, not table-driven.
- DOM-20: `services/atlas-portals/atlas.com/portals/portal/warp_by_name_test.go:94,126` — `TestWarpByName_Hit` and `TestWarpByName_MissFallsBackToRandomSpawn` are single-scenario, not table-driven.

### Non-Blocking (should fix)

- DOM-01: `services/atlas-monsters/atlas.com/monsters/monster/information/builder.go:69-84` — `Build()` performs no invariant validation; acceptable here because the value object has no invalid-state combination, but the rule's literal text is unmet.
