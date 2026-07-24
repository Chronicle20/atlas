# Plan Audit — task-124-teleport-rocks (Plan Adherence)

**Plan Path:** docs/tasks/task-124-teleport-rocks/plan.md
**Audit Date:** 2026-07-17
**Branch:** task-124-teleport-rocks
**Base:** c9490b724 (origin/main) → HEAD 2585df656 (33 commits)

## Executive Summary

All 23 plan tasks are faithfully implemented with strong file:line evidence; nothing was silently skipped, stubbed, or deferred. Every changed Go module (`libs/atlas-saga`, `libs/atlas-constants`, `libs/atlas-packet`, `atlas-character`, `atlas-channel`) builds, vets, and passes `go test -race ./...` clean. `docker buildx bake` for `atlas-character` and `atlas-channel` succeeds; `packet-audit matrix --check` and `operations --check` both exit 0; redis-key-guard, goroutine-guard, and service-registration-guard are all clean; `git status` is clean. Task 22's packet-verification campaign went beyond the plan's conservative scope (which sanctioned stop-and-ask for v84/v87/jms) and fully verified all three ops plus the cash sub-payload across v83/v84/v87/v95/jms_v185 via live IDA decompiles, uncovering and fixing a real cross-version bug in the cash teleport-rock sub-payload codec (commit 8c91089ba) along the way. Plan faithfully executed: **yes**.

## Task Completion

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 1 | Saga type + field-limit const | DONE | `libs/atlas-saga/model.go:24` (`TeleportRockUse`), `services/atlas-channel/.../saga/model.go:26,50,76` (re-exports incl. pre-existing `WarpToRandomPortal`), `libs/atlas-constants/map/field_limit.go:12-14` (`FieldLimitNoTeleportItem = 0x40`) |
| 2 | Shared `Target` codec | DONE | `libs/atlas-packet/teleportrock/target.go` (110 lines); `target_test.go` passes |
| 3 | Serverbound `Use` | DONE | `libs/atlas-packet/teleportrock/serverbound/use.go`; `TeleportRockUseHandle` const; tests pass |
| 4 | Serverbound `AddMap` | DONE | `libs/atlas-packet/teleportrock/serverbound/add_map.go`; register sends no map id (line 668-676 logic); tests pass |
| 5 | Cash sub-payload `ItemUseTeleportRock` | DONE | `libs/atlas-packet/cash/serverbound/item_use_teleport_rock.go`; later hardened by fix commit 8c91089ba (see Beyond-Plan Work) |
| 6 | Clientbound `MapTransferResult` writer+bodies | DONE | `libs/atlas-packet/teleportrock/clientbound/result.go`, `result_body.go`; nine mode-key consts; config-resolved via `WithResolvedCode`; golden + VIP-pad + error tests pass |
| 7 | `CharacterData` real lists | DONE | `libs/atlas-packet/character/data.go:107-110,704-737` — `TeleportMaps`/`VipTeleportMaps` fields, `encodeTeleports`/`decodeTeleports` no longer hardcode `EmptyMapId` |
| 8 | atlas-character `teleport_rock` domain | DONE | `entity.go` (table `teleport_rock_maps`, unique idx tenant/char/list/slot), `model.go`, `builder.go` |
| 9 | Processor + validations + status events | DONE | `teleport_rock/processor.go` — `AddMap`/`RemoveMap` validate eligibility/capacity/duplicate/not-found, emit `ListUpdated`/`Error` status events via `message.Buffer` |
| 10 | Command consumer + main wiring | DONE | `kafka/consumer/teleportrock/consumer.go`; `main.go:66,94,107` (migration + consumer + handler registration) |
| 11 | REST resource | DONE | `teleport_rock/rest.go` (Transform/Extract), `resource.go` (`GET /characters/{id}/teleport-rock-maps`); `main.go:123` route registered |
| 12 | Delete-cleanup + mock | DONE | `character/processor.go:354` calls `teleport_rock.DeleteForCharacter` inside the delete transaction; `teleport_rock/mock/processor.go` (func-field mock, `var _ teleport_rock.Processor = (*ProcessorMock)(nil)`); `administrator_test.go:91` `TestDeleteForCharacter` |
| 13 | Channel read model + command package | DONE | `character/teleportrock/{model,rest,processor,producer,requests}.go`; `kafka/message/teleportrock/kafka.go` |
| 14 | Thread real lists into character-data writer | DONE | `socket/writer/character_data.go:18,48-50` (`BuildCharacterData` 4th param `trm`); call sites `set_field.go`, `cash_shop_open.go`, `set_itc.go` all updated with fail-open fetch |
| 15 | `TROCK_ADD_MAP` handler | DONE | `socket/handler/teleport_rock_add_map.go`; register uses `s.Field()` (session map, no wire map id per locked decision #1); `main.go:877` registered |
| 16 | Status consumer + writer registration | DONE | `kafka/consumer/teleportrock/consumer.go` — `handleListUpdated`/`handleError`, error-reason mapping matches locked decision #10 exactly; `main.go:755` writer registered |
| 17 | Use-flow validate→saga | DONE | `teleportrock/use.go` (186 lines) — full 5-step validation chain matches plan's table exactly (source bar→target resolve→same-map→target bar→continent), warp-then-destroy saga steps, VIP selector `itemId/1000==5041`, regular-consume gate `itemId/10000==232`; `use_test.go` table test (`TestUseRockRejections` + 3 success cases) all pass |
| 18 | `USE_TELEPORT_ROCK` handler | DONE | `socket/handler/teleport_rock_use.go`; slot/item ownership check, `Valid()` gate, delegates to shared `UseRock`; `main.go:878` registered |
| 19 | Cash type-12 branch | DONE | `socket/handler/character_cash_item_use.go:118-131` — gated on `item.GetClassification(itemId) == item.ClassificationTeleportRock` (504) per locked decision #11; megaphone alias falls through unchanged; disambiguation test `TestCharacterCashItemUseHandleFunc_MegaphoneEnum12NotInvoked` present |
| 20 | Seed templates ×6 | DONE | All six templates carry the two handler rows (`LoggedInValidator` present on both) + the nine-key `MapTransferResult` writer row; opcodes verified to match the plan's table exactly (v83/84: 0x54/0x66/0x2A; v87: 0x57/0x69/0x2A; v92: 0x5B/0x71/0x2B; v95: 0x5B/0x72/0x29; jms: 0x4C/0x61/0x27) |
| 21 | Deploy manifests (2 Kafka topics) | DONE | `deploy/k8s/base/env-configmap.yaml`, both kustomize overlays (`-main`/`-PLACEHOLDER_ATLAS_ENV` suffixes), `deploy/compose/.env.example` — all four files updated correctly; `kubectl kustomize` main/pr both build clean |
| 22 | Packet verification campaign | DONE (exceeds plan) | v83 (5fd03fa7a), v95 (9892f46af), v84 (ac8000d32), v87 (78ae5ac75), jms_v185 (2585df656) all verified live against IDA with renamed fnames + `idb_save`; `docs/packets/audits/STATUS.md` shows ✅ for `USE_TELEPORT_ROCK`, `TROCK_ADD_MAP`, `MAP_TRANSFER_RESULT` across all 5 target versions; gms_v92 correctly absent from the matrix entirely (not part of the tracked version set — see Findings); `matrix --check`/`operations --check` exit 0 |
| 23 | Final verification gates | DONE | See Build & Test Results below; all green; `git status --short` clean |

**Completion Rate:** 23/23 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. No task was skipped, stubbed, or deferred. No `// TODO` markers found in any new teleport-rock file (`grep` swept `libs/atlas-packet/teleportrock`, `cash/serverbound/item_use_teleport_rock.go`, `atlas-character/teleport_rock`, `atlas-channel/teleportrock`, and all related handler/consumer packages — zero hits).

## Beyond-Plan Work (reviewer awareness)

1. **Cash sub-payload `updateTimeFirst` bug fix (commit 8c91089ba, mid-Task-22).** Live v95 decompile of `CWvsContext::SendConsumeCashItemUseRequest` revealed the original `ItemUseTeleportRock` codec unconditionally reserved a phantom trailing 4-byte `updateTime` budget in `teleportrock.Target.Decode`, which would have silently misdecoded every genuine 5-byte by-map cash-rock payload on v87+ as "no selection." The fix gave `Target.Decode` an explicit `hasTrailingUpdateTime bool` parameter. Verified consistent across both call sites: `teleportrock/serverbound/Use.Decode` (`use.go:65`) passes `true` unconditionally (Use always has a genuine trailing updateTime on every version, IDA-verified both v83 and v95), and `cash/serverbound.ItemUseTeleportRock.Decode` (`item_use_teleport_rock.go:66`) passes `!m.updateTimeFirst`, where `updateTimeFirst := t.MajorVersion() >= 87` is computed identically to the existing `CashItemUsePointReset` convention at `character_cash_item_use.go:42`. This does not contradict any locked decision in context.md — it refines Q1's wire-contract detail with corrected live evidence and is exactly the kind of self-discovered, self-fixed defect the project's "no deferring producible work" policy asks for.
2. **Task 22 exceeded its own conservative bar.** The plan explicitly sanctioned stop-and-ask for v84/v87/jms ("fnames absent from checked-in exports — needs IDB"). All three were fully verified live (IDBs were available and fnames resolved/renamed), so no cell was left at "stopped." STATUS.md/status.json also gained a `cash/serverbound/CashItemUseTeleportRock` row (not explicitly itemized in the plan's Task-22 table, which only listed the three teleportrock-package ops) — confirmed ✅ for v83/v84/v87/v95/jms_v185 and correctly ❌ for the pre-83 legacy columns (teleport rocks/cash items don't exist that early). This is additive coverage, not a gap.

## Build & Test Results

| Module | Build | Vet | Test (-race) | Notes |
|---|---|---|---|---|
| libs/atlas-saga | PASS | PASS | PASS | |
| libs/atlas-constants | PASS | PASS | PASS | |
| libs/atlas-packet | PASS | PASS | PASS | includes teleportrock, teleportrock/serverbound, teleportrock/clientbound, cash/serverbound |
| services/atlas-character/.../character | PASS | PASS | PASS | includes teleport_rock, kafka/consumer/teleportrock |
| services/atlas-channel/.../channel | PASS | PASS | PASS | includes teleportrock, character/teleportrock, kafka/consumer/teleportrock, socket/handler, socket/writer |

| Gate | Result |
|---|---|
| `docker buildx bake atlas-character` | PASS (built/cached) |
| `docker buildx bake atlas-channel` | PASS (built/cached) |
| `kubectl kustomize deploy/k8s/overlays/main` | PASS |
| `kubectl kustomize deploy/k8s/overlays/pr` | PASS |
| `tools/redis-key-guard.sh` | PASS (clean) |
| `tools/goroutine-guard.sh` | PASS (clean) |
| `tools/service-registration-guard.sh` | PASS (clean) |
| `packet-audit matrix --check` | PASS (exit 0) |
| `packet-audit operations --check` | PASS (exit 0, "0 absent-writer notes") |
| `git status --short` | clean |

(atlas-saga-orchestrator and atlas-login bakes were not independently re-run in this audit — no lib.-ripple risk was identified beyond the packages already covered by the above module test sweep, and the controller's prior run was reported green per the audit brief.)

## Findings

**None are blocking.** One informational note:

- **Task 20's own ad-hoc opcode-collision script (as literally written in plan.md Step 3) reports false-positive duplicates.** Running it verbatim against all six templates reports duplicate writer opcodes (e.g. `0x00` shared by `AuthSuccess`/`AuthTemporaryBan`/`AuthPermanentBan`/`AuthLoginFailed`; `0x0a` shared by `ServerListEntry`/`ServerListEnd`). These duplicates pre-date this branch (confirmed present in `git show c9490b724:.../template_gms_83_1.json` and every other base template) and are apparently legitimate multi-writer-per-opcode patterns elsewhere in the codebase, unrelated to teleport rocks. The task-124-introduced opcodes (`0x54`/`0x66`/`0x2A` for gms_83, and the version-specific equivalents for the other five templates) were individually confirmed absent from every template at the base commit, so no real collision was introduced. This is a latent imprecision in the plan's own verification script, not an implementation defect — flagging only so a future reader doesn't mistake "the Step-3 script as literally run would fail" for a regression in this branch.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending the backend-guidelines-reviewer pass, which writes to a separate `audit.md` per the task's code-review convention)

## Action Items

None required for plan adherence. No fixes needed before proceeding to the backend-guidelines review / PR.
