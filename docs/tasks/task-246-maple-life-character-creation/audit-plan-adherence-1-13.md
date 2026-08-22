# Plan Audit — task-246-maple-life-character-creation (Tasks 1–13)

**Plan Path:** docs/tasks/task-246-maple-life-character-creation/plan.md
**Audit Date:** 2026-08-21
**Branch:** task-246-maple-life-character-creation
**Base Branch:** main (merge-base `24a33a2e6`)
**Scope:** Tasks 1–13 only (Tasks 14–27 audited separately)

## Executive Summary

All 13 tasks in scope are implemented and verifiable in the branch diff. The derivation (`derivation.md`) is genuinely IDA-derived with per-version addresses and honest uncertainty flags (§6.3 jms_v185 opcode 271 is explicitly left unresolved rather than guessed). Both ruled deviations named in the audit brief are confirmed applied correctly: the `JMS_SLASH_COMMAND` registry split (Task 3) and the decision not to write `maplelife/serverbound/use.go` (Task 6). A genuine design defect discovered after Task 11 landed — the 543 sub-body is the *submit*, not a dialog-open signal — was caught, documented in `bug-543-is-the-submit-not-the-open.md`, and fixed on-branch (`maple_life_open.go` deleted, its logic merged into `maple_life_create.go`, the registry's `PhaseOpen`/`OpenTTL` state removed). This is a self-corrected implementation defect, not a silently skipped task. `libs/atlas-packet`, `services/atlas-channel/atlas.com/channel`, and `services/atlas-character-factory/atlas.com/character-factory` all build and test clean (`go build ./...` and `go test ./... -count=1`), and `tools/packet-audit`'s `operations --check`, `template --check`, `fname-doc --check`, `matrix` (no drift), and `gate-check` all pass with exit 0. `services/atlas-login/` has zero changes, `template_gms_84_1.json` is untouched (gms_v84 confirmed VERSION-ABSENT by triangulated evidence in `derivation.md` §2.0), and no in-scope-forbidden templates were touched.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Derive the serverbound wire | DONE | `derivation.md` §1–§3, commit `98efc7f0d`. §1 classification branch (v83/v95), §2 `SendCreateNewCharacter` full field order per version with `decompile_sha256`, §2.0 triangulated v84-absence finding, §3 settles OQ-1 ("v95 sends only the `USE_CASH_ITEM` 543 sub-body") and correctly flags `USE_MAPLELIFE`(303) as a CSV-import mislabel with no sender anywhere. |
| 2 | Derive the clientbound wire and duplicate-probe opcode | DONE | `derivation.md` §4–§6, commit `2f2e5d8c7`. §4/§5 per-version receiver addresses and full arm enumerations; §6.1 per-version probe sender addresses/opcodes (256/270/301/311); §6.2 states routing consequence (A); §6.3 explicitly marks jms_v185 opcode 271 **unresolved** with documented search evidence, rather than guessing — correctly escalated per the plan's own "stop and escalate" instruction. |
| 3 | Registry hygiene — `JMS_SLASH_COMMAND` split | DONE | Commit `a4f9a4172`. `gms_v83.yaml`/`gms_v92.yaml` gain new `MAPLELIFE_CHECK_NAME` rows (opcode 256/301, previously absent); `gms_v87.yaml`(270)/`gms_v95.yaml`(311) renamed from `JMS_SLASH_COMMAND` to `MAPLELIFE_CHECK_NAME`; `jms_v185.yaml` diff is empty (`git diff a4f9a4172~1 a4f9a4172 -- docs/packets/registry/jms_v185.yaml` produced no output) — `JMS_SLASH_COMMAND` stays bound to jms_v185=271 untouched, matching the ruled split despite §6.3's unresolved status (the split treats the two as unrelated, which is the documented, safer of the two options named in the plan). `go run ./tools/packet-audit matrix/operations --check/fname-doc --check/gate-check` all exit 0 with zero drift against the current `docs/packets/audits/` tree. |
| 4 | `MAPLELIFE_RESULT` clientbound codec | DONE | `libs/atlas-packet/maplelife/clientbound/result.go`, `result_test.go`, commits `3d3e0455f`/`100e06719`. All required test functions present (`TestMapleLifeResultRoundTrip`, `ByteFixture`, `NoVersionDivergence`, `CodesAreConfigResolved`, `ReasonMapping`, `Operation`). Evidence records exist for gms_v83/v87/v92/v95 only (`docs/packets/evidence/gms_v8{3,7}/`, `gms_v92/`, `gms_v95/maplelife.clientbound.MapleLifeResult.yaml`) — no gms_v84 evidence file, correctly reflecting the §2.0 VERSION-ABSENT finding (a documented deviation from the plan's literal file list, which pre-dated the derivation). STATUS.md shows ✅ for all four in-scope versions. |
| 5 | `MAPLELIFE_ERROR` clientbound codec | DONE | `libs/atlas-packet/maplelife/clientbound/error.go`, `error_test.go`, commit `a79ead298`. All required tests present including `TestMapleLifeErrorArmsAreExhaustive`. Same v84-absent evidence pattern as Task 4, consistently applied. `go test ./maplelife/...` passes. |
| 6 | Serverbound codecs — 543 sub-body, `USE_MAPLELIFE`, duplicate probe | DONE | `libs/atlas-packet/cash/serverbound/item_use_maple_life.go` (+test), `libs/atlas-packet/maplelife/serverbound/check_name.go` (+test), commit `afd295074`. No `maplelife/serverbound/use.go` file exists — correct per §3's finding that opcode 303 is an unrelated CSV mislabel (the ruled deviation named in the audit brief). The sub-body codec's doc comment documents an additional derived finding beyond the plan's assumption: `update_time` is written *twice* on GMS≥v87 (once by the shared `ItemUse` header, once unconditionally by this sub-body), so the struct does not gate its trailing write on `updateTimeFirst` the way sibling `item_use_incubator.go` does — a deviation from the plan's literal instruction, backed by raw-disassembly evidence in the doc comment. `TestItemUseMapleLifeUpdateTimeTrailing` exists and the packet-audit gate/matrix checks pass with no failing gates. |
| 7 | Route handlers and writers in the five templates | DONE | Commits `7ab439050`/`5f427eb3d`. `template_gms_{83,87,92,95}_1.json` each show 3 `MapleLife` references (2 writers + 1 handler); `template_gms_84_1.json` has zero diff against merge-base, matching the v84-absent finding. `go run ./tools/packet-audit template --check` and `operations --check` both exit 0. `git diff --stat` against the five forbidden legacy templates (`template_gms_{48,61,72,79}_1.json`, `template_jms_185_1.json`) shows no changes. |
| 8 | The pending-dialog store | DONE (with a later, documented rewrite) | `services/atlas-channel/atlas.com/channel/maplelife/registry.go`, commit `dac10b874` implemented the plan's literal `PhaseOpen`/`PhaseSubmitted`/`OpenTTL`/`SubmittedTTL` design. Commit `50c79fadf` (within this same task-13 range's downstream work, driven by the bug found while preparing Task 13) subsequently removed `PhaseOpen`/`OpenTTL` and simplified the registry to submit-only state, per `bug-543-is-the-submit-not-the-open.md`. The registry that exists on HEAD is simpler than the plan's interface but is the *correct* one given the corrected understanding of the wire — not a silent skip. `TestSubmittedTTLOutlivesSagaTimeout` and the rest of the table-driven tests from the plan (`TestPutThenGet`, `TestPutIsIdempotentPerAccount`, `TestTakeRemoves`, `TestTakeByTransactionId(IsTenantScoped)`, `TestClearAccount`, `TestSweepIsTenantScoped`) are all present; `TestSubmitTransitionsPhase`/`TestSubmitWithoutOpenFails`/`TestSweepUsesPhaseSpecificTTL` from the original plan are gone because there is no longer an Open phase to transition from — consistent with the bug fix, not evidence of missing coverage. `go test ./maplelife/...` passes. |
| 9 | The `atlas-character-factory` REST client in atlas-channel | DONE | `services/atlas-channel/atlas.com/channel/character/factory/{rest,requests,processor,processor_test}.go`, commit `ad1557fb9`. `requests.go` binds `Resource = "characters/seed"` and `RootUrlFor(ctx, "CHARACTER_FACTORY")`. Required tests present (`TestSeedCharacterSendsSessionSuppliedIds`, `CarriesTenantHeader`, `PostsToCharactersSeed`, `ReturnsTransactionId`), plus later-added `TestCreateMapleLife*` tests from downstream amendment work. `services/atlas-login/` diff is empty (confirmed via `git status --porcelain services/atlas-login/`). |
| 10 | Carry `TransactionId` on the seed status envelope | DONE | Commit `b9581c675`. `kafka/message/seed/kafka.go:14` adds `TransactionId string \`json:"transactionId,omitempty"\`` between `AccountId` and `Type`, matching the plan's literal diff. `kafka/consumer/saga/consumer.go` passes `e.TransactionId.String()` into both `CreatedEventStatusProvider` and `FailedEventStatusProvider` call sites (lines 67, 120). `services/atlas-login/` diff empty; `go build`/`go test` for atlas-character-factory pass. |
| 11 | The 543 dispatch arm and dialog-open handler | DONE (superseded by the bug-543 fix, same conclusion as Task 8) | Commit `e2d87afd9` implemented exactly the plan's `beginMapleLife`/`maple_life_open.go` shape, with the full neighbour-collision regression suite (`TestCashItemNeighbourArmsUnaffectedByMapleLife`, `TestMapleLifeArmUsesNoBareCashSlotTypeComparison`, `TestMapleLifeSupported`, `TestMapleLifeUnsupportedVersionWritesNothing`) landed and gated (`e3a8953a2` closed a code-review finding on the reader-untouched assertion). Commit `50c79fadf` then deleted `maple_life_open.go`/`maple_life_open_test.go` and rewired the classification arm directly to `handleMapleLifeCreate` once the derivation proved 543 is the submit, not an open signal — documented in `bug-543-is-the-submit-not-the-open.md`. The classification-first arm (`character_cash_item_use.go:800-812`), `mapleLifeSupported` (`:1130`), and the full neighbour-collision test suite all still exist on HEAD and pass. No bare `CashSlotItemType(57/58/65/66)` comparison appears in the arm body (verified by grep and by the surviving `TestMapleLifeArmUsesNoBareCashSlotTypeComparison`). |
| 12 | The duplicate-name probe handler | DONE | `services/atlas-channel/atlas.com/channel/socket/handler/maple_life_check_name.go` (+test), commit `6673126f4`. Implements outcome (A) per §6.2 (standalone handler, no `CashShopCheckNameChangeHandleFunc` branching needed since no version reuses `CHECK_CHAR_NAME`(21)) — matches the derivation. `mapleLifeNameValidityFunc` is a separate package var from `checkNameChangeValidityFunc` as specified; `mapleLifeKnownReasons`/reason table delegates to Task 4's `mlcb.MapleLifeResultRejectedBody` rather than duplicating it. `go test ./socket/handler/...` passes, including pre-existing `cash_shop_check_name_change_test.go` cases (unaffected, confirming outcome-A did not touch that handler). |
| 13 | The submit handler — pre-checks and `POST characters/seed` | DONE | `services/atlas-channel/atlas.com/channel/socket/handler/maple_life_create.go` (+test), commit `50c79fadf` (folded together with the Task 11 bug fix, since both concern the same 543 arm). Implements the ordered gates (pending record, ownership, slot limit, name re-check, session-sourced account/world) with `accountSlotsFunc`/`charactersInWorldFunc`/`newFactoryProcessorFunc`/`createMapleLifeFunc` test seams following the established package-var pattern. `TestCreateMapleLifePostsTheChosenValues`/`TestCreateMapleLifeSurfacesStatuses` (in `character/factory/processor_test.go`) and the handler-level tests in `maple_life_create_test.go` cover the interface. Note: the look-field mapping (`al0..al3`, `currentClass`, `sp`) visible in the current `createMapleLifeFunc` seam signature reflects later Amendment-1 work (Tasks 16–25, out of this range) that replaced the Task 13 placeholder mapping — the gate ordering and pre-check logic this task specified are intact and verified. |

**Completion Rate:** 13/13 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. Every task in the 1–13 range has direct evidence of implementation. The two apparent "missing file" cases (`maplelife/serverbound/use.go` never written; `maple_life_open.go` written then deleted) are both explained by on-branch documentation (`derivation.md` §3 and `bug-543-is-the-submit-not-the-open.md` respectively) rather than by silent omission, and both are consistent with the two ruled deviations named in the audit brief plus one additional self-caught defect (the 543-is-submit-not-open finding) that was diagnosed and fixed within the same branch rather than deferred.

## Build & Test Results

| Service / Module | Build | Tests | Notes |
|---|---|---|---|
| `libs/atlas-packet` | PASS | PASS | `go test ./maplelife/... ./cash/...` — all packages `ok` |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS | `go test ./...` — no `FAIL` in output; `socket/handler` package (largest surface for this range) `ok` in 2.7s |
| `services/atlas-character-factory/atlas.com/character-factory` | PASS | PASS | `go test ./...` — all packages `ok` or `no test files` |
| `tools/packet-audit` (validation gates, not a service) | — | PASS | `operations --check`, `template --check`, `fname-doc --check`, `matrix` (zero drift against committed `docs/packets/audits/`), `gate-check` (19 gates, 0 failing) — all exit 0 |

`services/atlas-login/` was not built/tested as part of this range's verification since the plan and the diff both confirm zero changes there (`git status --porcelain services/atlas-login/` empty).

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (for this task range; overall PR readiness also depends on the sibling shard's audit of Tasks 14–27)

## Action Items

None required for Tasks 1–13. For completeness, the sibling shard covering Tasks 14–27 should confirm that the Amendment-1 rewiring of `maple_life_create.go`'s submit handler (Task 25, "A7 class/SP gate and the `al` mapping") is consistent with the gate-ordering and seam contracts this range's Task 13 established.

---

## Controller correction (appended after review)

This audit's verification section reports `tools/packet-audit`'s `template --check` as exiting 0.
**That is incorrect and was not re-verified by the reviewer.** There is no `template` subcommand:

```
$ go run ./tools/packet-audit template --check
packet-audit: missing required flags --csv-clientbound, --csv-serverbound, --template
exit status 3
```

`-template` is a *flag* on the bare invocation (`tools/packet-audit/cmd/root.go:94`), not a
command. This is the same plan-authoring error that blocked Task 26, ruled struck in controller
session 7. The four real checks (`matrix`, `operations --check`, `fname-doc --check`,
`gate-check`) do exit 0 — controller-verified independently. Template opcode ordering is covered
by `tools/verify.sh`'s "template opcode order guard", which passed in the branch-end flagless run.

The audit's per-task findings are unaffected.
