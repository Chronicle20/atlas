# Context — task-065 Combat-Domain Packet Audit

> Companion to `plan.md`. Captures the *as-of-planning* state an executor needs to know without re-deriving it.

## Worktree + branch

- Worktree: `<home>/source/atlas-ms/atlas/.worktrees/task-065-combat-domain-audit/`
- Branch: `task-065-combat-domain-audit`
- Merge-base with `main`: commit `414d7c872` (`fix(pr-deploy): annotate per-PR LB Services to use MetalLB's pr-pool`).
- Commits ahead of main: 2 (`spec(task-065)` + `design(task-065)`).
- `git rev-parse --show-toplevel` MUST end with `/.worktrees/task-065-combat-domain-audit`; `git branch --show-current` MUST be `task-065-combat-domain-audit`.

## Hard precondition: task-028 must be merged before Phase 1 starts

`design.md §1` assumes task-028's analyzer/registry/run-routing work is in place. As of planning (2026-05-15) it is **not** in this worktree:

- `tools/packet-audit/internal/atlaspacket/analyzer.go` has no `blockTerminatesWithReturn` and no suffix-taint logic (Phase 0 of task-028).
- `tools/packet-audit/internal/atlaspacket/registry.go` does not recognise `EncodeForeign` (Phase 1 of task-028).
- `tools/packet-audit/cmd/run.go` has 28 `case` entries (login only) — `grep -c 'case "' tools/packet-audit/cmd/run.go` → 28. Task-028's branch has 78.
- `docs/packets/audits/gms_v95/` only contains login (task-027) reports.
- `docs/packets/ida-exports/_pending.md` only mentions login-domain entries.

The task-028 branch (`.worktrees/task-028-character-domain-audit`) is rebased on top of `bea7964b8 feat: Monster Book (task-056) (#402)` and forward but not yet merged. **Task 0 of the plan is the rebase gate**: bring task-028's mainlined changes into this worktree before any combat audit work, or stop and ask the user.

## What "the audit pipeline" actually is (executor reference)

```
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_v95.json \
  --output           docs/packets/audits/gms_v95
```

Inputs:
- CSV files = canonical opcode↔FName mapping per direction.
- Template JSON = per-version opcode→atlas-writer-name mapping (one file per region+major version).
- Atlas-packet directory = the analyzer source-of-truth for what bytes Go emits.
- IDA-export JSON = recorded IDA `Decode*` ops per FName.

Outputs (combined for all FNames in the IDA export):
- `docs/packets/audits/gms_v95/<PacketName>.md` — per-packet verdict + diff prose.
- `docs/packets/audits/gms_v95/<PacketName>.json` — machine-readable verdict.
- `docs/packets/audits/gms_v95/SUMMARY.md` — table rolling up all `.md` rows.

The pipeline is **append-only**: re-running it does not delete reports for FNames absent from the current IDA export. Combat reports land under the same `gms_v95/` dir as the existing login + (post-task-028) character reports.

## Actual combat-domain inventory (PRD numbers are wrong)

Verified by `ls libs/atlas-packet/{monster,drop,reactor,pet}/{clientbound,serverbound}/`. Total = **31 logical packets**, NOT the 59 PRD §2 cites. (PRD inflated by counting headers/struct names. Adopt 31 as the source of truth.)

### monster/clientbound (8 files, 9 logical packets)
| File | Writer const | Operation FName | Notes |
|---|---|---|---|
| `control.go` | `MonsterControlWriter` | `ControlMonster` | inline `m.monster.Encode(...)` recurse (MonsterModel) |
| `damage.go` | `MonsterDamageWriter` | `MonsterDamage` | flat `MonsterDamageType` enum (sub-op drift candidate) |
| `destroy.go` | `MonsterDestroyWriter` | `DestroyMonster` | trivial |
| `health.go` | `MonsterHealthWriter` | `MonsterHealth` | trivial |
| `movement_ack.go` | `MonsterMovementAckWriter` | `MoveMonsterAck` | flat |
| `movement.go` | `MonsterMovementWriter` | `MoveMonster` | **no test file**; calls 3 sub-structs (MultiTargetForBall, RandTimeForAreaAttack, Movement); version-gated on GMS>83/JMS |
| `spawn.go` | `MonsterSpawnWriter` | `SpawnMonster` | hottest; calls `model.MonsterModel.Encode` (548 LOC); region/version-gated control byte |
| `stat.go` | `MonsterStatSetWriter` + `MonsterStatResetWriter` | `MonsterStatSet`, `MonsterStatReset` | **two logical packets, one file**; both call `*model.MonsterTemporaryStat.Encode`; sub-op enum drift candidate |

### monster/serverbound (1 file)
| File | Handle const | Operation FName |
|---|---|---|
| `movement.go` | `MonsterMovementHandle` | `MonsterMovementHandle` |

### drop/clientbound (2 files)
| File | Writer const | Operation FName |
|---|---|---|
| `destroy.go` | `DropDestroyWriter` | `DropDestroy` |
| `spawn.go` | `DropSpawnWriter` | `DropSpawn` |

### drop/serverbound (1 file)
| File | Handle const | Operation FName |
|---|---|---|
| `pick_up.go` | `DropPickUpHandle` | `DropPickUpHandle` |

### reactor/clientbound (3 files)
| File | Writer const | Operation FName |
|---|---|---|
| `destroy.go` | `ReactorDestroyWriter` | `ReactorDestroy` |
| `hit.go` | `ReactorHitWriter` | `ReactorHit` |
| `spawn.go` | `ReactorSpawnWriter` | `ReactorSpawn` |

### reactor/serverbound (1 file)
| File | Handle const | Operation FName |
|---|---|---|
| `hit.go` | `ReactorHitHandle` | `ReactorHitHandle` |

### pet/clientbound (6 logical packets across 7 files)
| File | Writer const | Operation FName | Notes |
|---|---|---|---|
| `activated.go` | `PetActivatedWriter` | `PetActivated` | sub-op on `active` bool; self+foreign perspective candidate |
| `activated_body.go` | — | — | helper: `PetSpawnBody`/`PetDespawnBody` wrap `NewPetSpawnActivated`/`NewPetDespawnActivated`. NOT an independent encoder — flag as "consumed by activated.go" in audit report, NOT a `_pending.md` deferral. |
| `cash_food_result.go` | `PetCashFoodResultWriter` | `PetCashFoodResult` | trivial |
| `chat.go` | `PetChatWriter` | `PetChat` | flat |
| `command.go` | `PetCommandResponseWriter` | `PetCommandResponse` | uses `mode` field for two constructors (`NewPetCommandResponse`, `NewPetFoodResponse`); flat encoder. |
| `exclude.go` | `PetExcludeResponseWriter` | `PetExcludeResponse` | linearised count→loop (analyzer FP risk) |
| `movement.go` | `PetMovementWriter` | `PetMovement` | **no test file**; calls `model.Movement.Encode` |

### pet/serverbound (8 files)
| File | Handle const | Operation FName |
|---|---|---|
| `chat.go` | `PetChatHandle` | `PetChatHandle` |
| `command.go` | `PetCommandHandle` | `PetCommandHandle` |
| `drop_pick_up.go` | `PetDropPickUpHandle` | `PetDropPickUpHandle` |
| `exclude_item.go` | `PetItemExcludeHandle` | `PetItemExcludeHandle` |
| `food.go` | `PetFoodHandle` | `PetFoodHandle` |
| `item_use.go` | `PetItemUseHandle` | `PetItemUseHandle` |
| `movement.go` | `PetMovementHandle` | `PetMovementHandle` |
| `spawn.go` | `PetSpawnHandle` | `PetSpawnHandle` |

Pet serverbound count is **8** files (8 distinct handlers), NOT 16. PRD §13.2's open question resolves: each file is its own decoder; audit each as a discrete row. There is no shared dispatcher.

### Test-file gaps to close before any encoder mutation

- `libs/atlas-packet/monster/clientbound/movement.go` — no `movement_test.go`.
- `libs/atlas-packet/pet/clientbound/movement.go` — no `movement_test.go`.

Both are full encoders, not stubs (verified). Phase 2 sub-tasks fill the gaps with 4-variant `pt.Variants` round-trip coverage **before** any fix lands per design §6.

## Sub-struct (model) inventory

`libs/atlas-packet/model/` types referenced by the combat domain:
- `model.MonsterModel` (548 LOC, `monster.go:459`) — called from `monster/clientbound/spawn.go:54` and `monster/clientbound/control.go:61`.
- `model.MonsterTemporaryStat` (`monster.go:200`) — called from `monster/clientbound/stat.go` (both StatSet and StatReset). Has an `EncodeMask` method (`monster.go:231`) plus standard `Encode`. Sub-op enum drift candidate.
- `model.MultiTargetForBall` (`multi_target_for_ball.go`, 36 LOC) — called from `monster/clientbound/movement.go:63`.
- `model.RandTimeForAreaAttack` (`rand_time_for_area_attack.go`, 34 LOC) — called from `monster/clientbound/movement.go:64`.
- `model.Movement` (`movement.go`, 320 LOC) — called from `monster/clientbound/movement.go:66` and `pet/clientbound/movement.go:35`. Already registered by task-028 (top-level + element sub-types).

Predicted Phase 1 registrations:
- `model.MonsterModel` (NEW)
- `model.MonsterTemporaryStat` (NEW)
- `model.MultiTargetForBall` (NEW)
- `model.RandTimeForAreaAttack` (NEW)

`model.Movement` and its element sub-types are already registered by task-028. Phase 1 includes a sanity fixture that asserts they remain registered after the rebase.

## File-level coordination with adjacent tasks

Per design §13 last bullet, coordinate at branch-ordering time, NOT at design/plan time. The following branches touch `services/atlas-monsters/` (business logic, not wire shape) and may conflict at the *atlas-packet handler caller* level when fixes land:

- task-033 (monster aggro controller)
- task-034 (monster skill picker)
- task-035 (mob skill firing and regen)
- task-036 (monster skill effects completion)
- task-057 (monster movement)
- task-060 (monster data TTL cache)
- task-061 (data cache invalidation)

This audit touches `libs/atlas-packet/monster/` (wire-shape), `libs/atlas-packet/drop/`, `libs/atlas-packet/reactor/`, `libs/atlas-packet/pet/`. If a fix changes a constructor signature, downstream callers in `services/atlas-channel/`, `services/atlas-monsters/`, `services/atlas-pets/`, `services/atlas-drops/`, `services/atlas-reactors/` must rebuild clean.

## Key reference points

| Path | Why it matters |
|---|---|
| `libs/atlas-packet/monster/clientbound/spawn.go:54` | Load-bearing `m.monster.Encode(l, ctx)(options)` call into MonsterModel. |
| `libs/atlas-packet/monster/clientbound/movement.go` | 3 sub-struct calls + version gates; no test file. |
| `libs/atlas-packet/monster/clientbound/damage.go:14-20` | Flat `MonsterDamageType` enum — sub-op drift candidate. |
| `libs/atlas-packet/monster/clientbound/stat.go` | `*model.MonsterTemporaryStat.Encode` recurse with EncodeMask path. |
| `libs/atlas-packet/drop/clientbound/spawn.go` | Hot-path; not gated; coordinate block analysis. |
| `libs/atlas-packet/reactor/clientbound/hit.go` | Hot-path reactor. |
| `libs/atlas-packet/pet/clientbound/activated.go:47-69` | Self vs foreign perspective candidate. |
| `libs/atlas-packet/pet/clientbound/activated_body.go` | Helper wrapper, NOT a standalone packet — document, do not audit separately. |
| `libs/atlas-packet/pet/clientbound/movement.go` | Calls `model.Movement.Encode`; no test file. |
| `libs/atlas-packet/model/monster.go` | MonsterModel + MonsterTemporaryStat. Registry inputs. |
| `libs/atlas-packet/model/movement.go` | Already registered by task-028. |
| `libs/atlas-packet/test/context.go` | `pt.Variants` 4-variant table (`GMS v28`, `GMS v83`, `GMS v95`, `JMS v185`). |
| `libs/atlas-packet/test/roundtrip.go` | `RoundTrip(t, ctx, encode, decode, opts)` helper. |
| `tools/packet-audit/internal/atlaspacket/registry.go` | Sub-struct registration site. |
| `tools/packet-audit/internal/atlaspacket/registry_test.go` | Fixture pattern (per task-028). |
| `tools/packet-audit/cmd/run.go` | `candidatesFromFName` routing. Combat additions land here. |
| `docs/packets/audits/gms_v95/SUMMARY.md` | Combat domain rows appended. |
| `docs/packets/ida-exports/_pending.md` | Bare-handler + sub-op deferrals. (Note: design.md references `docs/packets/audits/_pending.md`; the existing file is at `docs/packets/ida-exports/_pending.md`. Use the existing path.) |
| `docs/packets/ida-exports/{gms_v83,gms_v95}.json` | Existing IDA exports — append combat FNames here. **No `gms_v87.json` or `gms_jms_185.json` files exist yet.** Phase 3 creates them. |
| `../task-028-character-domain-audit/post-phase-b.md` | Closeout template + lessons-learned. |
| `../task-028-character-domain-audit/plan.md` | Reference plan structure. |

## Verification matrix

Every Phase 1/2/3 commit MUST pass:
```
go test -race ./tools/packet-audit/...
```

Every Phase 2/3 atlas-packet edit MUST additionally pass:
```
go test -race ./libs/atlas-packet/...
```

Phase 4 closeout adds:
```
go build ./...
go vet ./libs/atlas-packet/...
go vet ./tools/packet-audit/...
```

`docker build` is NOT expected (no `go.mod` / `Dockerfile` changes anticipated). The Phase 4 task includes a guarded `git diff --name-only main..HEAD -- '**go.mod' '**Dockerfile'` check that triggers a `docker build` if any file matches.

## Gitleaks scrub

Final pre-PR check:
```
grep -r '/home/' docs/packets/audits/gms_v95/{monster,drop,reactor,pet}/ 2>/dev/null
```
Expected: no output. Scrub with `sed -i 's|/home/[^/]*/source/atlas-ms/atlas/||g' <file>` if any hit.

## Decisions locked at plan time

- **Plan task granularity:** ~12 tasks. Phase 0 (rebase gate, 1 task) + Phase 1 (registry + routing, 3 tasks) + Phase 2 (per sub-domain audit, 4 tasks) + Phase 3 (cross-version, 3 tasks) + Phase 4 (closeout + PR, 1 task).
- **Per-sub-domain commit cadence:** one commit per fix; one bucket commit per sub-domain audit report batch.
- **PR strategy:** single PR at Phase 4, NOT per-sub-domain PRs. Design.md §15 listed both options; plan picks single PR to match task-028's shipping cadence.
- **v28 inclusion:** OUT per design §8. Variants table includes v28 for round-trip tests, but no v28 IDA pass.
- **Phase 3 commit naming:** `phase-3-v83`, `phase-3-v87`, `phase-3-jms-185` (carries forward from task-028).
- **`_pending.md` location:** `docs/packets/ida-exports/_pending.md` (existing file). Design.md mentioned `docs/packets/audits/_pending.md` but that path doesn't exist on disk.
- **`activated_body.go`:** documented in `activated.md` audit report as "consumed by activated.go's `PetSpawnBody`/`PetDespawnBody`"; NOT a `_pending.md` row.
- **MonsterTemporaryStat:** registered with `Encode` only. `EncodeMask` (mask-only path) is internal to MonsterTemporaryStat.Encode and does not need a separate registry key.
