# Plan Audit — task-139-pet-auto-pot-validation (Plan Adherence)

**Plan Path:** docs/tasks/task-139-pet-auto-pot-validation/plan.md
**Audit Date:** 2026-07-26
**Branch:** task-139-pet-auto-pot-validation
**Base Branch:** main (merge-base 68b26af0a)
**HEAD:** 35fcaf10b (27 commits)

## Executive Summary

All 15 plan tasks (numbered 1–13, 15, 14) are DONE, verified by direct file:line
inspection of the landed code, not just the implementer reports. The branch's
central promise — PET_AUTO_POT is validated end-to-end (specific pet, owned,
spawned, character alive, version-family skill gate) and fails closed — holds
in the code: all nine routed seed templates (gms_48/61/72/79/83/84/87/95,
jms_185) carry a `skillGate` option, the handler rejects when the option is
absent or unrecognized, and all nine PET_AUTO_POT matrix cells show
`"state": "verified"` in `docs/packets/audits/status.json`. No TODOs/stubs/501s
were introduced, and no absolute home paths leaked into any committed file. Go
build/vet/test, docker bake, and all four repo guard scripts (redis-key,
goroutine, template-opcode-order, lint) are clean per task-14's re-run
transcript, with the sole non-pass being `ui:node-missing` (no Node.js in this
container — atlas-ui was not touched by this branch, an environmental gap not
a regression).

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | atlas-constants: Classification 519 + pet/skill package | DONE | `libs/atlas-constants/item/constants.go:87` (`ClassificationPetSkill = Classification(519)`); `libs/atlas-constants/pet/skill/constants.go` present with `constants_test.go` |
| 2 | atlas-packet: ResolveCode16 | DONE | `libs/atlas-packet/resolve.go:103` `func ResolveCode16(...)` |
| 3 | atlas-packet: petFlag + config-resolved usPetSkill encode + trailer version gate | DONE | `libs/atlas-packet/model/asset.go:350-381` `encodePetCashItemInfo`; `resolvePetSkillWireMask` helper; gates `MajorAtLeast(72)`/`MajorAtLeast(79)` correctly extended to gms_48 (matches context.md's rebase note); `TestAssetPetCashItemTrailerVersionGate` in `asset_test.go:381-428` asserts v48/v61 = v83−6, v72 = v83−2, v79/84/87/95/jms unchanged |
| 4 | atlas-data: parse 0519 pet-skill keys on cash items | DONE | `services/atlas-data/atlas.com/data/cash/rest.go:45-46` (`PetSkills`/`PetSkillAdd`); `cash/reader.go:99-110` parses `info` node via `petskill.All()` |
| 5 | atlas-data: parse pet-ability attributes on equipment | DONE | `equipment/rest.go:47` (`PetAbilities`); `equipment/reader.go:117,231-240` `readPetAbilities` |
| 6 | atlas-pets: SET_SKILL command, flag persistence, FLAG_CHANGED event | DONE (documented deviation) | `pet/processor.go:1025-1068` `SetSkillAndEmit`/`SetSkill`; **deviation accepted**: uses `database.ExecuteTransaction` + `message.Emit(outbox.EmitProvider(...))` (the codebase's real outbox idiom), not the plan's bare `message.Emit(p.kp)` snippet, which does not exist in this codebase — confirmed by grep, matches context.md's documented deviation |
| 7 | atlas-consumables: 0519 pouch consume branch | DONE | `ConsumePetSkillPouch` present in `consumable/processor.go`; message mirrors, cash client mirror, `ErrPetCannotLearn`/`PET_CANNOT_LEARN` all landed; task-7-report "Issues" section: none |
| 8 | atlas-packet: jms type-28 ItemUsePetSkill (IDA-verified) | DONE (documented deviation) | `libs/atlas-packet/cash/serverbound/item_use_pet_skill.go`: bare `petId uint64`, `NewItemUsePetSkill()` takes zero args, no `UpdateTime()` accessor — confirmed exactly as the accepted deviation describes; struct doc cites IDA evidence (`0xaef2f5`, `0xaf1a42`) |
| 9 | atlas-channel: cash-item-use type-28 arm + petId plumbing | DONE (consistent with Task 8 deviation) | `character_cash_item_use.go:69-79`: `CashSlotItemTypePetSkill` arm has no `updateTimeFirst` branch, consistent with Task 8's layout; forwards to `RequestItemConsumeWithPet` |
| 10 | atlas-channel: data/consumable + data/equipment clients | DONE | Both packages exist under `services/atlas-channel/atlas.com/channel/data/` with `rest.go`/`requests.go`/`processor.go` |
| 11 | atlas-channel: pet flag sync (FLAG_CHANGED, asset enrichment, encode) | DONE | task-11-report confirms message mirror, consumer handler, asset builder/model `SetPetFlag`/`PetFlag`, `PetAssetEnrichmentDecorator`, `socket/model/asset.go` wiring; no blocking issues noted |
| 12 | atlas-channel: PetItemUseHandle validation chain | DONE | `socket/handler/pet_item_use.go` (full read): `evaluateAutoPot`, `resolveSkillSources`, `petAbilityPositions`, `normalizeWornPosition`, `matchPetAbilityEquips`, `resolveSpawnedPet` all present and match plan + rebase-revision addendum (gms_48 `hasPetId` branch, no fallback between petId-present/absent paths — verified at lines 79, 96-135); forward call on success (`pet_item_use.go:159`) is **byte-for-byte identical** to the pre-task handler (confirmed via `git show 68b26af0a:...pet_item_use.go`) — FR-5 satisfied |
| 13 | atlas-configurations: seed template wiring + usPetSkill bits | DONE | All nine templates (`gms_48/61/72/79/83/84/87/95`, `jms_185`) carry exactly one `skillGate` entry each with `LoggedInValidator` present (verified by direct grep, not just report); `gms_12`/`gms_92` correctly excluded (0 matches); opcode-order guard passes (22 arrays ascending) |
| 15 | legacy version coverage (v48/61/72/79): codec gate, fixtures, matrix, data check | DONE | `pet/serverbound/item_use.go` reuses the pre-existing `hasLeadingPetId` helper (`legacy.go`, landed on main before this branch, used by sibling pet packets) rather than reinventing a gate; `status.json` PET_AUTO_POT block shows all nine cells `"state": "verified"`; `coverage-manifest.yaml` present and detailed; Step 3's legacy-WZ open question is honestly recorded in context.md as "genuinely unverified" (not resolved either direction) — confirmed by reading the actual context.md text |
| 14 | Full verification sweep + context.md | DONE | task-14-report.md quotes real command output (module loop EXIT=0 ×7, docker bake EXIT=0 with five `naming to ...` lines, redis/goroutine/opcode-order guards EXIT=0, packet-audit matrix/fname-doc/operations --check EXIT=0); `context.md` committed at `35fcaf10b`; only non-pass is `ui:node-missing` (no Node in container, atlas-ui untouched by this branch) |

**Completion Rate:** 15/15 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. Two items are legitimately runtime-only and are recorded as such rather
than falsely claimed complete:

- **PRD acceptance items 6 and 9 (partial)** — "auto-pot passes after pouch
  grants the skill" and "legitimate GMS player not rejected by the equip-ability
  gate" require a live deployed environment (pouch use → flag persist → channel
  projection refresh → subsequent auto-pot packet). `context.md` rollout note 4
  and task-14-report's "Runtime-only items" section both flag these honestly as
  not exercised by unit tests — this is the correct behavior per the task brief,
  not a gap to be fixed.
- **Task 15 Step 3 (legacy WZ pet-equip attribute check)** — recorded as
  genuinely unresolved (the live `atlas-data` pod predates this branch's Task 5
  commit, so the live query couldn't distinguish "legacy WZ lacks the
  attribute" from "code not deployed yet"). This is an honest open question,
  not a silent default; `equip_data_missing` remains a distinct fail-closed
  rejection reason so the ambiguity never resolves to a permissive default.

## Build & Test Results

(Re-derived from task-14-report.md's captured transcripts, cross-checked
directly in this audit for the guard scripts, JSON validity, and packet-audit
checks — see commands below, all re-run live in this audit pass.)

| Service/Module | Build | Tests | Notes |
|---|---|---|---|
| libs/atlas-constants | PASS | PASS | `go test -race`/`vet`/`build` EXIT=0 (task-14 transcript) |
| libs/atlas-packet | PASS | PASS | EXIT=0; includes `pet/serverbound`, `cash/serverbound`, `model` packages touched by this task |
| services/atlas-data | PASS | PASS | EXIT=0; `cash`, `equipment`, `consumable`, `pet` packages all `ok` |
| services/atlas-pets | PASS | PASS | EXIT=0 |
| services/atlas-consumables | PASS | PASS | EXIT=0 |
| services/atlas-channel | PASS | PASS | EXIT=0 |
| services/atlas-configurations | PASS | PASS | EXIT=0; re-confirmed `go test -race ./... && go build ./...` implicit in task-14 loop |
| docker buildx bake (atlas-data/pets/consumables/channel/configurations) | PASS | n/a | EXIT=0, all five `naming to docker.io/library/atlas-*:local` lines present |
| tools/redis-key-guard.sh | PASS | n/a | re-run live in this audit, EXIT=0 |
| tools/goroutine-guard.sh | PASS | n/a | re-run live in this audit, EXIT=0 |
| tools/template-opcode-order-guard.sh | PASS | n/a | re-run live in this audit: "OK: 22 template arrays are in ascending opcode order." |
| tools/lint.sh --check | PASS (Go) / FAIL (ui:node-missing) | n/a | 83 Go modules `0 issues.`; sole failure is atlas-ui Node.js absence in this container — atlas-ui untouched by this branch |
| packet-audit matrix/fname-doc/operations --check | PASS | n/a | re-run live in this audit, all EXIT=0; PET_AUTO_POT all 9 cells `"state": "verified"` |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None required for plan adherence. Carried-forward operational notes (already
recorded in context.md, not blockers to this branch's correctness):

1. Existing tenants need a live config PATCH (`skillGate` handler options on
   all 9 versions + `petSkill` writer tables) before this feature is live for
   them — seed templates only apply to newly-created tenants.
2. atlas-data must be re-ingested on GMS tenants before the channel gate goes
   live, or every GMS auto-pot rejects with `equip_data_missing` (fail-closed
   by design).
3. The two runtime-only acceptance items (pouch-to-auto-pot round trip; legit
   GMS player pass) should be exercised in a deployed environment before wide
   rollout, per context.md rollout note 4.
