# Plan Audit — task-044-packet-opcode-name-alignment

**Plan Path:** docs/tasks/task-044-packet-opcode-name-alignment/plan.md
**Audit Date:** 2026-05-02
**Branch:** task-044-packet-opcode-name-alignment
**Base Branch:** main
**Range:** 926bc2a7..ec940100 (single commit `ec94010`)

## Executive Summary

Plan executed faithfully. All four constant flips (Step 1) match the target values at the specified lines. Both Warnf calls (Step 2) added in `libs/atlas-opcodes/producer.go` with all four required focused tests (positive and negative paths). Step 3 is correctly left as a manual smoke task and the plan's Definition-of-done flags it as such (no false claim of completion). All affected modules build and test green.

## Task Completion

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 1a | `CompartmentMergeRequestHandle = "CompartmentMergeHandle"` | DONE | libs/atlas-packet/inventory/serverbound/compartment_merge.go:12 |
| 1b | `CompartmentSortRequestHandle = "CompartmentSortHandle"` | DONE | libs/atlas-packet/inventory/serverbound/compartment_sort.go:12 |
| 1c | `CharacterChatMultiHandle = "CharacterMultiChatHandle"` | DONE | libs/atlas-packet/chat/serverbound/multi.go:13 |
| 1d | `MultiChatWriter = "CharacterMultiChat"` | DONE | libs/atlas-packet/chat/clientbound/multi.go:12 |
| 2a | Warnf in BuildHandlerMap on handlerMap miss | DONE | libs/atlas-opcodes/producer.go:54-56 |
| 2b | Post-loop Warnf for unconfigured availableWriters | DONE | libs/atlas-opcodes/producer.go:29-33 |
| 2-test | `TestBuildHandlerMap_WarnsOnUnknownHandler` | DONE | libs/atlas-opcodes/producer_test.go:52 |
| 2-test | `TestBuildHandlerMap_NoWarnWhenHandlerKnown` | DONE | libs/atlas-opcodes/producer_test.go:68 |
| 2-test | `TestBuildWriterProducer_WarnsOnUnconfiguredAvailableWriter` | DONE | libs/atlas-opcodes/producer_test.go:84 |
| 2-test | `TestBuildWriterProducer_NoWarnWhenAllAvailableConfigured` | DONE | libs/atlas-opcodes/producer_test.go:102 |
| 3 | Manual v83 smoke (merge / sort / multi-chat) | DEFERRED | Plan Step 3 explicitly requires live client + docker; Definition-of-done lists it as a separate item, no false claim made. |

**Completion Rate:** 10/10 in-scope tasks (100%); Step 3 correctly out-of-scope-for-code.
**Skipped without approval:** 0
**Partial implementations:** 0

## Notes / Minor deviations

- 2b warning text reads "tenant config has no opcode mapping for it" instead of the plan's draft "no opcode is configured for it in tenant config." Same semantics, contains writer name; tests assert substring `OrphanWriter`, so test passes. No functional impact.
- Regression grep matched only the two intentionally-preserved writer constants (`CompartmentMergeWriter`, `CompartmentSortWriter`) per plan §Out-of-scope.

## Build & Test Results

| Module | Build | Tests | Notes |
|--------|-------|-------|-------|
| libs/atlas-opcodes | PASS | PASS | 4 new tests pass; full package green |
| libs/atlas-packet | PASS | PASS | All sub-packages green |
| services/atlas-channel | PASS | PASS | Full build + test suite green |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None. Manual smoke (Step 3) remains the only outstanding item and is explicitly an out-of-band verification step requiring a live v83 client + docker stack.

## Backend guidelines audit

- **Date:** 2026-05-02
- **Scope:** SHA range 926bc2a..ec94010 — string-value flips in `libs/atlas-packet/{chat,inventory}` + Warnf additions and new tests in `libs/atlas-opcodes`.
- **Build:** PASS — `go test ./libs/atlas-opcodes/... ./libs/atlas-packet/...` clean; `services/atlas-channel/atlas.com/channel: go build ./... && go test ./...` clean.
- **Overall:** PASS

### Applicable checks

| ID | Status | Evidence |
|----|--------|----------|
| DOM-21 (libs/atlas-constants reuse) | N/A | No new domain types, ids, or numeric constants introduced. |
| Log-level appropriateness | PASS | `producer.go:31` and `producer.go:55` use `Warnf`, matching the existing convention at `producer.go:49` (missing validator) and `producer.go:61` (parse failure). All three are recoverable misconfigurations the loop intentionally `continue`s past — `Warn` is correct (not `Error`, since execution proceeds; not `Debug`, since they signal operator-visible drift). |
| Test coverage — positive + negative | PASS | `producer_test.go:52` (handler missing → warn naming handler+opcode), `:68` (handler present → zero warns), `:84` (orphan available writer → warn naming only the orphan), `:102` (all configured → zero warns). Both branches of both new `Warnf` paths covered. |
| Convention drift in producer.go | PASS | New warns mirror the message shape of the prior `Warnf` at `producer.go:49` (`"Unable to locate validator [%s] for handler [%s]."`) — bracketed identifiers, period terminator. |
| Template alignment | PASS | `services/atlas-configurations/seed-data/templates/template_gms_83_1.json:344,349,459` match the new handler constants; `:1358,1362,1519` match the writer-side strings (note `MultiChatWriter` flipped to `CharacterMultiChat` while `CompartmentMergeWriter`/`CompartmentSortWriter` were intentionally preserved per plan §Out-of-scope). |
| Stale literal references | PASS | Grep for `"CompartmentMerge"` / `"CompartmentSort"` / `"CharacterChatMulti"` / `"CharacterChatMultiHandle"` returns only the two legitimate clientbound writer constants (`libs/atlas-packet/inventory/clientbound/compartment_merge.go:12`, `compartment_sort.go:12`), which correctly remain. |

### Skipped (irrelevant to this change set)

DOM-01..DOM-20 (DDD layering, builder/entity/Transform, JSON:API, Kafka producer patterns, processor/administrator separation, multi-tenancy, tenant-callback test setup) — none apply to a string-constant flip + log-statement change in a non-domain shared library. SEC-01..SEC-04 — no auth/redirect/secret surface touched.

### Blocking
- None.

### Non-Blocking
- None.
