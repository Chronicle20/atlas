# Plan Adherence Audit — task-224-pet-name-tag

**Plan Path:** `docs/tasks/task-224-pet-name-tag/plan.md`
**Audit Date:** 2026-08-13
**Branch:** `task-224-pet-name-tag`
**Base Branch:** `main` (merge-base `723519dc4`, head `2c9942bba`, 23 commits)

## Executive Summary

All 16 tasks in the plan were faithfully implemented, and all four flagged plan deviations were verified to match their rulings exactly (v48 is `✅` not `n-a`, Task 15 was absorbed into Task 4's commit, the mirror-guard fixture uses `slot:3`, and `packet-gap-inference.md` was correctly left untouched since it does not exist on this branch). Every affected module — `libs/atlas-constants`, `libs/atlas-packet`, `libs/atlas-saga`, `atlas-pets`, `atlas-channel`, `atlas-saga-orchestrator` — builds clean, vets clean, and passes `go test -race -count=1 ./...` with zero failures across 200+ packages. All nine template-and-guard scripts (opcode order, duplicate binding, movement types, redis-key, goroutine, skill-job-id, buff-duration, trade-mirror, service-registration) exit 0; `tools/lint.sh --check` reports "0 issues" on every Go module and fails only on the known `ui:node-version` false-fail (node v24 vs required v22). The six items on the deferred-minors triage list were all independently confirmed present as described; none block merge, though the two genuinely uncovered guard branches in `handleNameChangedEvent` (Task 10) are a real gap worth a follow-up test.

## Task Completion

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 1 | Shared pet-name bounds in `libs/atlas-constants` | DONE | `libs/atlas-constants/pet/name.go:19-22` (`MinNameLength=4`, `MaxNameLength=12`), `:27-28` (errors), `:34,42` (`NormalizeName`/`ValidateName`); all 6 tests pass (`go test ./pet/... -v`) |
| 2 | Serverbound type-17 sub-body codec | DONE | `libs/atlas-packet/cash/serverbound/item_use_pet_name_tag.go:37-59` (`ItemUsePetNameTag`, `NewItemUsePetNameTag`, `Encode`/`Decode`); tests pass. Minor gap: round-trip test inlines `request.Request`/`request.NewRequestReader` (`item_use_pet_name_tag_test.go:56-58`) instead of calling a shared `newReaderFor` helper — see Deferred Minors §1 |
| 3 | Clientbound `PET_NAMECHANGE` codec | DONE | `libs/atlas-packet/pet/clientbound/name_changed.go:15,25,33,42` (`PetNameChangedWriter`, `NameTagLayer`, `NameChanged`, `NewPetNameChanged`), `:47-59` (GMS-only `t.IsRegion("GMS")` flag gate on both Encode/Decode); `activated.go` sets `nameTag: NameTagLayer` |
| 4 | Per-version derivation and matrix promotion | DONE (matches ruling #1) | `docs/packets/audits/STATUS.md:205` — all 10 cells `✅` including `gms_v48` (0x071); ten byte fixtures in `name_changed_test.go` all pass |
| 5 | atlas-pets Kafka contract — `RENAME`/`NAME_CHANGED` | DONE | `services/atlas-pets/.../kafka/message/pet/kafka.go:66-68` (`RenameCommandBody`), `:207-212` (`NameChangedStatusEventBody`) |
| 6 | atlas-pets `SetName`/`updateName` | DONE | `pet/builder.go:65` (`SetName`), `pet/administrator.go:93` (`updateName`, deliberately no RowsAffected check for idempotency) |
| 7 | atlas-pets rename processor/producer/consumer | DONE | `pet/processor.go:957` (`RenameAndEmit`), `:972` (`Rename`); `kafka/consumer/pet/consumer.go` registers `handleRenameCommand`; 4 tests (`TestRenameAppliesAndEmits`, `TestRenameIsIdempotent`, `TestRenameRejectsInvalidName`, `TestRenameRejectsNonOwner`) all pass |
| 8 | atlas-pets `PATCH /pets/{petId}` | DONE | `pet/resource.go:30` (route registration), `:183-222` (`handleUpdate`); `resource_test.go:35,65` (`TestPatchPetRejectsInvalidName`, `TestPatchPetRenamesPet`) both pass. Known gap: 500 not 404 for nonexistent pet — see Deferred Minors §2 |
| 9 | Shared saga contract — `PetNameTagUse`/`RenamePet` | DONE | `libs/atlas-saga/model.go:45,95` (type/action), `payloads.go:302` (`RenamePetPayload`), `unmarshal.go:192-193` (unmarshal arm) |
| 10 | Orchestrator wiring | DONE | mirror (`kafka/message/pet/kafka.go`), producer/processor (`pet/producer.go`, `pet/processor.go`), `saga/model.go:50,97,260,1397-1398` (re-exports + unmarshal arm), `saga/event_acceptance.go:56,151,360` (3-block registration), `saga/handler.go:117,828-829,1370` (interface, dispatch, impl), `kafka/consumer/pet/consumer.go:38,72-98` (`handleNameChangedEvent`). Gap: no direct test for the two guard branches (wrong event type / nil transaction id) — see Deferred Minors §3 |
| 11 | Orchestrator compensation and timeout classification | DONE | `pet/mock/processor.go` created; `saga/compensator.go:43,60,134-144,259,325-326,1745-1817` (`WithPetProcessor`, `DispatchPetNameTagRollbacks`, `compensatePetNameTagUse`, `petNameTagCharacterId`); `saga/timer.go:181,208,253` (both classification lists + dispatch arm); `saga/pet_name_tag_compensation_test.go` (2 tests, both pass under `-tags test`). Minor: `producer.go:195-197` computes `mesoSackCharacterId(s)` unconditionally then discards it on the `PetNameTagUse` path — see Deferred Minors §4 |
| 12 | atlas-channel Kafka mirrors + round-trip guard | DONE (matches ruling #3) | Channel and orchestrator mirrors byte-identical to atlas-pets' owner copy (same struct names, json tags, verified by diff); `contract_mirror_test.go` in both modules uses `"slot":3` (not the plan's `"slot":0`), both pass |
| 13 | atlas-channel dispatch — 517 predicate fix + handler | DONE | `character_cash_item_use.go:957` (`itemId%10000 != 0` predicate fix, replacing the overflowing `10000*itemId/10000` form), `:736` (`CashSlotItemTypePetNameTag`); `character_cash_item_use_pet_name_tag.go` (`buildPetNameTagUseSaga`, `handlePetNameTagUse`, `petsForOwnerFunc` seam); `character_cash_item_use_pet_name_tag.go:124` belt-and-braces `target.OwnerId() != s.CharacterId()` check present — see Deferred Minors §5 |
| 14 | atlas-channel broadcast, writer registration, failure rendering | DONE | `kafka/consumer/pet/consumer.go:97,459-499` (`handleNameChanged`, `ForSessionsInMap` broadcast); `main.go:733` (`petcb.PetNameChangedWriter` registered); `kafka/consumer/saga/consumer.go:385,410-417` (`petNameTagFailureMessage`, `errorCode` param left named not `_`) — see Deferred Minors §6 |
| 15 | Tenant socket templates — writer registration | DONE (matches ruling #2 — absorbed into Task 4's commit `a2615b050`, verification-only here) | 10 templates carry exactly 1 `PetNameChanged` entry each (incl. `gms_48`), 0 in `gms_12`; all 10 carry `fname: CPet::OnNameChanged`; opcodes match plan/registry exactly (`0x71/0x83/0x9D/0xA1/0xAC/0xB0/0xB9/0xC8/0xCB/0xB2`); all 3 template guards exit 0 |
| 16 | Documentation, rollout note, full verification gate | DONE | `docs/tasks/task-224-pet-name-tag/rollout.md` created; `docs/research/missing-features/items-and-consumables.md:130` updated and accurately describes the v48 ruling; `packet-gap-inference.md` correctly not touched (does not exist in this repo — matches ruling #4); no `go.mod`/`go.sum` changed on the branch, so no docker-bake round required (verified: `git diff --name-only` shows nothing matching `go.mod|go.sum`) |

**Completion Rate:** 16/16 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0 (all deviations were pre-ruled and match their rulings)

## Requirement Traceability Table Walk

Walked every row of `plan.md`'s Requirement Traceability table (lines 2602-2640) against the code:

- **FR-1.1–1.5** (type-17 constant, predicate fix, unit test, dispatch arm, ownership guard retained) — all present in `character_cash_item_use.go` / `character_cash_item_use_pet_name_tag.go`; `TestGetCashSlotItemTypePetNameTag` present and passing.
- **FR-2.1–2.4** (serverbound codec + per-version re-derivation) — Tasks 2 & 4 confirmed.
- **FR-3.1–3.4** (lead-pet resolution, ownership) — `handlePetNameTagUse` resolves `Slot()==0`, checks ownership; both confirmed at `character_cash_item_use_pet_name_tag.go:100-125`.
- **FR-4.1–4.5** (bounds, trim, server-side enforcement, no profanity filter) — `petconst.NormalizeName`/`ValidateName` called at both the channel (`character_cash_item_use_pet_name_tag.go:127-130`) and atlas-pets (`pet/processor.go` `Rename`) layers, satisfying "atlas-pets re-validates" (FR-5.6) independently. Grepped for profanity/curse/badword across all new pet-name files — none found, confirming FR-4.5's explicit scope boundary is honored.
- **FR-5.1–5.6** (command/event contract, administrator, builder, idempotency, re-validation) — all confirmed in Tasks 5-7; idempotency proven by `TestRenameIsIdempotent`.
- **FR-6.1–6.6** (clientbound codec, writer registration, event-driven broadcast, flag provenance) — all confirmed; `NameTagLayer` doc comment at `name_changed.go` cites design A5 provenance directly, not a config key.
- **FR-7.1–7.6** (saga type, rename-first ordering, fail-closed pre-flight, revert-on-failure, full registration set, no-warp unlock) — all confirmed; step order in `buildPetNameTagUseSaga` is `rename_pet` then `consume_pet_name_tag` (character_cash_item_use_pet_name_tag.go).
- **FR-8.1–8.5** (nine[ten] templates, fname, sorted position, no duplicate handler, live-tenant reconciliation doc) — all confirmed; `rollout.md` documents the reconciliation step per `bug_new_opcodes_not_in_live_tenant_config`.
- **FR-9.1–9.4** (matrix promotion, v48 handling, serverbound verification, real derivation) — confirmed; v48 promoted to `✅` (superseding the plan's `n-a` assumption per ruling #1), all cells carry real IDA addresses in evidence records (spot-checked `docs/packets/evidence/gms_v83/cash.serverbound.CashItemUsePetNameTag.yaml` exists).
- **Design C-1/C-2/C-3, §3.6, §5 A5** — all confirmed in code comments and structural gates (region gate, bounds constants, mirror test, shared `NameTagLayer`).
- **PRD §10 build & guard gates** — confirmed below.

No gap found spanning task boundaries beyond what the triage list below already names.

## Build & Test Results

| Module | Build | Vet | Test (`-race -count=1`) | Notes |
|---|---|---|---|---|
| `libs/atlas-constants` | PASS | PASS | PASS | all packages ok, incl. `pet` and `pet/skill` |
| `libs/atlas-packet` | PASS | PASS | PASS | incl. `cash/serverbound`, `pet/clientbound` |
| `libs/atlas-saga` | PASS | PASS | PASS | |
| `services/atlas-pets/atlas.com/pets` | PASS | PASS | PASS | 123 packages, 0 failures |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS | PASS | 123 packages ok, 0 FAIL lines |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator` | PASS | PASS | PASS (plain and `-tags test`) | 33 packages ok both ways; compensation tests require `-tags test` (matches `meso_sack_compensation_test.go` convention) |

**Guard scripts (all from repo root):**

| Guard | Result |
|---|---|
| `tools/redis-key-guard.sh` | exit 0 |
| `tools/goroutine-guard.sh` | exit 0 |
| `tools/skill-job-id-guard.sh` | exit 0 |
| `tools/buff-duration-guard.sh` | exit 0 |
| `tools/template-opcode-order-guard.sh` | exit 0 ("22 template arrays are in ascending opcode order") |
| `tools/template-duplicate-binding-guard.sh` | exit 0 ("22 template arrays carry no duplicate (name, opCode) binding") |
| `tools/template-movement-types-guard.sh` | exit 0 (54 move handlers across 11 templates) |
| `tools/trade-contract-mirror-guard.sh` | exit 0 |
| `tools/service-registration-guard.sh` | exit 0 |
| `tools/lint.sh --check` | FAIL only on `ui:node-version` (node v24 present, v22 required); every Go module reports "0 issues." Known environment false-fail, not a code defect. |

`git diff --name-only 723519dc4...2c9942bba \| grep 'go\.mod\|go\.sum'` returned nothing — no module dependency moved, so `docker buildx bake` was correctly not required for this branch.

## Deviation Ruling Verification

All four pre-identified deliberate deviations were checked against their stated rulings and match exactly:

1. **`gms_v48` implemented, not `n-a`.** `STATUS.md:205` shows ten `✅` cells, zero `n-a`. Ten templates (not nine) carry the `PetNameChanged` writer, including `template_gms_48_1.json` at opcode `0x71`. **MATCHES RULING.**
2. **Task 15 absorbed into Task 4.** Task 4's commit (`a2615b050`, "verify PET_NAMECHANGE across all ten versions; promote matrix row") is where the template wiring landed based on the ten-template state confirmed above; the plan's Task 15 checklist items (JSON validity, one entry per template, guard exits) were all independently re-verified and pass. **MATCHES RULING.**
3. **Task 12 fixture uses `slot:3`.** Both `services/atlas-channel/.../contract_mirror_test.go:19` and the orchestrator's equivalent use `"slot":3` in the wire fixture, not the plan's `slot:0`. **MATCHES RULING.**
4. **`packet-gap-inference.md` not updated.** `git ls-files docs/research/missing-features/` on this branch returns only `items-and-consumables.md`; no history for `packet-gap-inference.md` exists on this branch. Task 16 correctly updated only `items-and-consumables.md`. **MATCHES RULING.**

## Deferred Minors Triage

1. **Task 2 — inlined round-trip test (`item_use_pet_name_tag_test.go:56-58`).** Confirmed: `TestItemUsePetNameTagDecodeRoundTrip` constructs `request.Request(b)` / `request.NewRequestReader(&req, 0)` directly instead of calling a shared helper. This is cosmetic — the test still exercises the real `Decode` path and passes — but it does lose whatever "unconsumed bytes" assertion a shared helper might carry. **Verdict: does not need to block merge.** Low-risk style nit; recommend folding into a future test-helper consolidation pass rather than reopening this branch.

2. **Task 8 — PATCH `/pets/{petId}` returns 500 not 404 for a nonexistent pet.** Confirmed at `resource.go:194-197`: `GetById` error routes through `server.WriteErrorResponse`, same as the pre-existing `handleGetPet` (`resource.go:38-44`). This is a genuine pre-existing pattern, not a regression introduced by this task. The stated rationale (registering a not-found classifier in `libs/atlas-rest` is a repo-wide behavior change with no mandate; special-casing one handler would make it inconsistent with its sibling) is sound. **Verdict: correctly parked, not a blocker.**

3. **Task 10 — `handleNameChangedEvent`'s two guard branches are untested.** Confirmed: `kafka/consumer/pet/consumer.go:72-98` has two early-return guards (`e.Type != pet2.StatusEventTypeNameChanged` and `e.Body.TransactionId == uuid.Nil`), and `grep -n "func Test" kafka/consumer/pet/consumer_test.go` shows only `TestHandleClosenessChangedEvent_*` — no test names `handleNameChangedEvent` or `RenamePet`/`NameChanged` in that file. **Verdict: this is the one item on the triage list I'd flag as worth fixing before merge is ideal, though not a hard blocker** — the happy path is exercised end-to-end by `TestRenamePetStepCompletesOnMatchingTransaction` (Task 10 Step 7) via `AcceptEvent`, but the consumer's own guard logic (a stray non-saga rename event, or a redelivered event racing an already-cleared transaction) has zero direct coverage. This is a real, if small, coverage gap — recommend a 10-line follow-up test before or shortly after merge, not a plan-adherence failure.

4. **Task 11 — `producer.go:188-192` computes `mesoSackCharacterId(s)` then discards it on the `PetNameTagUse` path.** Confirmed at `saga/producer.go:194-198`: `characterId := mesoSackCharacterId(s)` runs unconditionally, then is overwritten by `petNameTagCharacterId(s)` when `s.SagaType() == PetNameTagUse`. Functionally harmless (the correct value is used), just a wasted call and mildly confusing to read. **Verdict: pure style nit, does not block merge.**

5. **Task 13 — `target.OwnerId() != s.CharacterId()` is an untested unreachable branch.** Confirmed at `character_cash_item_use_pet_name_tag.go:124-127`; `GetByOwner`'s contract (verified at `pet/processor.go:110`, filters by `ownerId` parameter) makes this branch unreachable through the seam as currently wired. Defense-in-depth against a future refactor of `petsForOwnerFunc`/`GetByOwner` that stops filtering by owner. **Verdict: acceptable, belt-and-braces pattern is a recognized idiom in this codebase (plan explicitly calls it out as such); not a blocker.**

6. **Task 14 — `petNameTagFailureMessage`'s `errorCode` param left named, not `_`.** Confirmed at `kafka/consumer/saga/consumer.go:415`: `func petNameTagFailureMessage(errorCode string) string`. `tools/lint.sh --check` reports 0 issues for `atlas-channel`, so the linter is not tripping on this — the plan's own escape hatch condition ("if an unused parameter trips `tools/lint.sh`, name it `_`") was correctly not triggered. **Verdict: no action needed; matches the plan's own conditional guidance.**

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

1. (Optional, non-blocking) Add direct unit tests for `handleNameChangedEvent`'s two guard branches in `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/pet/consumer_test.go` — one asserting a wrong-`Type` event is a no-op, one asserting a nil-`TransactionId` event is a no-op. This is the only item from the deferred-minors list with genuinely zero test coverage; everything else on that list is either intentional, harmless, or already covered indirectly.
2. No other action items. All 16 plan tasks are DONE, all four deviations match their rulings, and all builds/tests/guards are green.
