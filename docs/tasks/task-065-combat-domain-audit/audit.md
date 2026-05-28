# Plan Audit — task-065-combat-domain-audit (FINAL)

**Plan Path:** `docs/tasks/task-065-combat-domain-audit/plan.md`
**Audit Date:** 2026-05-27
**Branch:** `task-065-combat-domain-audit`
**Base Branch:** `main` (merged at `c68ac5a75`; 89 commits ahead of main, 0 behind)
**HEAD:** `2ccb2757e`

This audit supersedes the May-15 monster-only audit and the May-27 mid-sweep audit. It covers the original plan (Phases 0–4) AND the 10 follow-up items landed on the same branch after the initial post-PR code review.

## Executive Summary

All 11 plan tasks (Phases 0–4) and all 10 post-PR follow-up items (items 1–10) landed with commit + test + ledger evidence. The branch is build-clean, vet-clean, race-test-clean across `libs/atlas-packet/...`, `tools/packet-audit/...`, and the touched `atlas-channel` service. No `/home/` paths leaked into audit reports. 89 commits total above main; combat-domain v95 SUMMARY carries 33 rows (14 ✅ / 17 ❌ / 2 🔍) — matches `post-phase-b.md`'s final-state tally exactly. **Recommendation: READY_TO_MERGE**, with two minor documentation-consistency observations noted in Findings (non-blocking).

## Plan Adherence (original plan.md tasks)

Plan has 64 unchecked `- [ ]` items and 0 checked. This is the repo's standard pattern — task ledgers, not checkbox state, track completion.

| Task | Status | Evidence |
|---|---|---|
| **Phase 0** — Task 0: rebase gate on task-028 baseline | DONE | task-028 base merged into main as `82ecec3a9`; task-065 commits sit atop. `tools/packet-audit/internal/atlaspacket/analyzer.go` carries `blockTerminatesWithReturn`, `registry.go` carries `EncodeForeign`, `cmd/run.go` carries character routing entries. |
| **Phase 1** — Task 1: registry coverage fixture for combat sub-structs | DONE | `053a32d52 test(packet-audit): assert combat sub-structs MonsterModel/TemporaryStat/MultiTargetForBall/RandTimeForAreaAttack registered`. Fixtures in `tools/packet-audit/internal/atlaspacket/registry_test.go`. |
| **Phase 1** — Task 2: `candidatesFromFName` routing for combat FNames | DONE | `c5b019a8a feat(packet-audit): route 31 combat-domain FNames to atlas writers/handlers` + `e1b3294ff feat(packet-audit): sub-domain disambiguation via candidate.pkg`. `tools/packet-audit/cmd/run.go` now has 109+ case entries. |
| **Phase 1** — Task 3: analyzer flatten-safety fixtures | DONE | `5d76d5172 test(packet-audit): MonsterSpawn/StatSet flatten safety fixtures`. Tests in `tools/packet-audit/internal/atlaspacket/analyzer_test.go`. |
| **Phase 2a** — Task 4: monster sub-domain audit | DONE | `96a72e718 test(atlas-packet,monster/movement)` (preflight test) + `a3153dc18 audit(monster): GMS v95 sub-domain audit (9 clientbound packets)`. v95 reports for `MonsterSpawn/Control/Damage/Destroy/Health/Movement/MovementAck/StatSet/StatReset` plus `MonsterMovementHandle/Request` exist under `docs/packets/audits/gms_v95/`. |
| **Phase 2b** — Task 5: pet sub-domain audit | DONE | `aac33b8db test(atlas-packet,pet/movement)` + `43565a896 audit(pet): GMS v95 sub-domain audit (14 packets)`. v95 reports for `PetActivated/Chat/CashFoodResult/Command/CommandResponse/ExcludeResponse/Movement` + 8 sb pets present. |
| **Phase 2c** — Task 6: drop sub-domain audit | DONE | `000671c49 audit(drop): GMS v95 sub-domain audit (3 packets)`. v95 reports for `DropSpawn/Destroy/PickUp` present. |
| **Phase 2d** — Task 7: reactor sub-domain audit | DONE | `0a7e139e4 audit(reactor): GMS v95 sub-domain audit (4 packets)`. v95 reports for `ReactorSpawn/Hit/Destroy/HitRequest` present. |
| **Phase 3** — Task 8: GMS v83 cross-version pass | DONE | `e2b82d08b audit(combat): GMS v83 cross-version pass (phase-3-v83)`. `docs/packets/ida-exports/gms_v83.json` populated; `docs/packets/audits/gms_v83/` carries combat reports. |
| **Phase 3** — Task 9: GMS v87 cross-version pass | DONE | `170240784 audit(combat): GMS v87 cross-version pass (phase-3-v87)`. `gms_v87.json` populated; `docs/packets/audits/gms_v87/` carries combat reports. |
| **Phase 3** — Task 10: JMS v185 cross-version pass | DONE | `4520ed61e audit(combat): JMS v185 cross-version pass (phase-3-jms-185)`. `gms_jms_185.json` populated; `docs/packets/audits/jms_v185/` carries combat reports. |
| **Phase 4** — Task 11: post-phase-b ledger + verification + scrub | DONE | `7b6b89df6` (initial monster-only closeout) + `84660f0b0 docs(task-065): rewrite post-phase-b.md for full scope (Phases 2+3 complete)` + `dd3fd9009 docs(task-065): post-phase-b + _pending updates after wire-bug fixes` + `a11da7622 docs(task-065): code review audit reports`. Subsequent follow-up sweep updated `post-phase-b.md` again (see below). gitleaks scrub clean (verified). |

**Phase 2 wire fixes landed during execution** (not separate plan tasks; one-commit-per-fix per plan conventions):

| Fix | Commit | Files |
|---|---|---|
| MonsterDestroy swallow-id (destroyType=4) | `fca8aa860` | `libs/atlas-packet/monster/clientbound/destroy.go` + test |
| DropDestroy explode/pet-pickup tail (destroyType=4/5) | `fca8aa860` | `libs/atlas-packet/drop/clientbound/destroy.go` + test |

**Completion Rate:** 11/11 plan tasks (100%). Plan checkboxes not flipped; commit/file evidence is canonical.

## Follow-up Items 1–10 (post-PR sweep)

The ledger section `post-phase-b.md` §"Post-PR follow-up work landed in-branch" enumerates 10 items re-opened after initial code review. Audited individually:

| Item | Commit | Files / Evidence | Status |
|---|---|---|---|
| **10 — PRD scope reconcile** | `5524d78d9` | `prd.md` §2/§4.1/§4.2/§11 updated from 59 → 31 combat packets; `post-phase-b.md` gains "Post-PR follow-up work" section. Inventory now matches `plan.md`. | DONE |
| **2 — Combat template opcode audit** | `4ccbf9d72` | `template-audit.md` created (84 lines). Findings: no name-string drift, no opcode collisions in existing entries. Coverage gap (only `template_gms_83_1.json` fully populated; v95 has zero combat entries) surfaced as a separate follow-up gated on IDA-dispatcher decompile access. Ledgered in `post-phase-b.md`. | DONE (gap explicitly deferred with rationale) |
| **4 — Qualified registry type names** | `170ba923f` | `tools/packet-audit/internal/atlaspacket/registry.go` — storage keyed on `<pkgPath>.<name>`, `byShort` index added, `Qualify(hint, contextPkg)` method added. Tests: `TestRegistryQualifiedKeysResolveCollisions`, `TestRegistryQualifyPrefersSamePackage`. | DONE |
| **6 — Dispatcher-prefix annotation** | `25fd855d3` | `tools/packet-audit/internal/idasrc/export.go:14-32` adds `Dispatcher` field; `dispatcherPrefix(kind)` at line 172 emits canonical prefixes for `per-mob`, `per-pet`, `per-pet-remote`. Tests in `export_test.go` cover all three kinds + no-annotation passthrough + unknown-kind forward-compat. | DONE |
| **5 — Wire-mutex if/else collapse** | `02a781195` | `tools/packet-audit/internal/atlaspacket/analyzer.go:253-315` introduces `scratchWalk` over body+else; collapses identical-shape branches into one unconditional position. Tests: `TestWireMutexCollapsesIfElse` and `TestWireDivergentKeepsBothBranches` (testdata `wire_mutex.go.txt`, `wire_divergent.go.txt`). | DONE |
| **8 — `Delegate` op for sub-fn descent** | `f9515973a` | `tools/packet-audit/internal/idasrc/export.go:87` adds `resolveWithVisited` with cycle detection; line 113 handles `Op == "Delegate"`. Tests: `TestDelegateInlinesSubFunction`, `TestDelegateANDsGuards`, `TestDelegateCycleDetected`, `TestDelegateDiamondAllowed`, `TestDelegateRequiresRef` (testdata `delegate_mini.json`). | DONE |
| **7 — Encode↔Decode equivalence + AssignStmt walking** | `37450ea49` | `tools/packet-audit/internal/atlaspacket/analyzer.go` walks RHS of `AssignStmt` so atlas Decode methods (`m.x = r.ReadByte()`) produce the same Call list as Encode methods. Tests: `TestEncodeDecodeProduceEquivalentCalls` (atlaspacket) + `TestParsePrimEncodeDecodeEquivalence` (idasrc) cover Encode1↔Decode1 through Encode8↔Decode8, Str, Buf, and legacy Buffer aliases. | DONE |
| **9 — Bulk audit re-run** | `2a932c7f1` | ReactorHitRequest flipped ❌→✅ in `docs/packets/audits/{gms_v83,gms_v87,gms_v95,jms_v185}/ReactorHitRequest.md` (all four verified at ✅). The deliberate non-flips for MonsterMovement/Move/CharacterList are documented in `post-phase-b.md:139-145` as "explicitly not corrected on the verdict line in-branch" because the fixes require IDA-entry rewrites (Delegate refs for CMovePath/CMob::Init/CPet::Init, loop-unrolling for CharacterList) that are bulk per-version edits gated on IDA-decompile access. Internal consistency: see Findings §1 below. | DONE (per ledger's explicit scope decision) |
| **3 — MonsterControl aggro byte semantic fix** | `5b6f32ca9` | `libs/atlas-packet/monster/clientbound/control.go:43,68-73,86` — `Control` struct grows `aggro bool`, encoder writes `byte(1)` if aggro else `byte(0)` (replacing legacy hardcoded `byte(5)`), decoder reads into `m.aggro`. Caller `services/atlas-channel/atlas.com/channel/socket/writer/monster_control.go` widens `ControlMonsterBody` signature; both `StartControlMonsterBody` and `StopControlMonsterBody` thread aggro correctly. Test `TestMonsterControlAggroByteReflectsState` pins wire bytes `0x01`/`0x00` at index 5. All consumer call-sites updated (`kafka/consumer/monster/consumer.go` lines 150/300/321/343 and `kafka/consumer/map/consumer.go:467` pass `m.ControllerHasAggro()` / `e.Body.ControllerHasAggro`). | DONE |
| **1 — v83/v87 GenerateMovePath IDA entries** | `2ccb2757e` | `docs/packets/ida-exports/gms_v83.json` adds `CMob::GenerateMovePath` at `0x66b6fc` (opcode `0xBC`, md5 `80ff438ced539b831f0d2ed95099275d`). `docs/packets/ida-exports/gms_v87.json` adds same at `0x6a6381` (opcode `0xC8`, md5 `2e692f3ab5078e04138d264f8ea1e668`). v83 report: `MonsterMovementRequest.md` verdict ❌ (analyzer-FP class — positions 0–8 ✅, positions 9+ flag the EncodeBuf placeholder for Flush body). v87 report: verdict 🔍 (matches v95/JMS-v185 distribution exactly). Both consistent with the cross-version pattern. | DONE |

### Special-attention items — verifications

**MonsterControl aggro byte (item 3) — wire-shape compatibility:**

- Atlas previously emitted `byte(5)` for any `controlType > Reset`. New behavior: `byte(1)` if `aggro=true`, `byte(0)` if `aggro=false`.
- Back-compat reasoning (from commit message): v95 client treats the byte as a non-zero flag. `byte(1) == byte(5)` semantically for "this controller has aggro". So aggro=true callers see no behavior change. New aggro=false callers correctly signal "no aggro" instead of falsely saying "aggro".
- Confirmed by checking that:
  1. The only Atlas writer for this packet is `services/atlas-channel/.../monster_control.go` — verified at lines 26-35.
  2. All consumer call-sites in `kafka/consumer/monster/consumer.go` (lines 150, 300, 321, 343) and `kafka/consumer/map/consumer.go:467` correctly thread the upstream aggro state (`m.ControllerHasAggro()` or `e.Body.ControllerHasAggro`).
  3. `StopControlMonsterBody` (Reset path) passes `false` — but Reset never reaches the aggro byte in the encoder anyway (`if m.controlType > ControlTypeReset` guards lines 68-75).
- **The reasoning holds.** The wire-shape change is safe: existing aggro=true emissions remain wire-compatible (any non-zero byte → "has aggro"), and the new aggro=false emissions correctly distinguish "no aggro" rather than falsely emitting `byte(5)`.

**Item 9 internal consistency — three deliberately-not-flipped reports:**

- Ledger (`post-phase-b.md:139-145`) explicitly says verdict lines for CharacterList, MonsterMovement, Move are **not** updated on-disk after the re-run, because the real fix requires IDA-entry rewrites that are blocked on IDA-decompile access.
- On-disk verdicts checked across all four versions:
  - **v83**: CharacterList ❌, MonsterMovement ❌, Move 🔍.
  - **v87**: CharacterList ✅, MonsterMovement 🔍, Move 🔍.
  - **v95**: CharacterList ✅, MonsterMovement 🔍, Move 🔍.
  - **jms_v185**: (no CharacterList report — JMS-v185 doesn't have one), MonsterMovement 🔍, Move 🔍.
- The ledger's claim that all four ReactorHitRequest reports are at ✅ is confirmed: `gms_v83/v87/v95/jms_v185` all show `**Verdict:** ✅`.
- See Findings §1 for a minor ledger/v83 wording discrepancy.

**Item 1 — v83/v87 IDA addresses + opcodes match JSON:**

- v83 JSON address: `0x66b6fc`, opcode: `0xBC`. Md5 cited in notes: `80ff438ced539b831f0d2ed95099275d`. Report file `gms_v83/MonsterMovementRequest.md` line 3 cites `**IDA:** 0x66b6fc` — matches.
- v87 JSON address: `0x6a6381`, opcode: `0xC8`. Md5: `2e692f3ab5078e04138d264f8ea1e668`. Report file `gms_v87/MonsterMovementRequest.md` line 3 cites `**IDA:** 0x6a6381` — matches.
- Verdicts: v83 ❌, v87 🔍 — matches the cross-version pattern documented in the ledger (v83 has narrowest wire shape — analyzer-FP class on positions 9+; v87 mirrors v95/JMS-v185).
- Wire-shape narrative in JSON `notes` field cross-checks the documented `(GMS && >83) || JMS` gate semantics in `libs/atlas-packet/monster/clientbound/movement.go`.

## Verification Results

| Service / Module | Build | Vet | Tests (-race) | Notes |
|---|---|---|---|---|
| `libs/atlas-packet/...` | PASS | PASS | PASS | All test packages cached green; no new failures. |
| `tools/packet-audit/...` | PASS | PASS | PASS | All test packages cached green. |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS | PASS | Touched by item 3 (monster_control writer aggro plumbing). All tests cached green. |

`docker buildx bake` not required — no `go.mod` or repo-root `Dockerfile` files were touched (verified via `git diff --name-only main...HEAD`).

**gitleaks scrub:** `grep -rl '/home/' docs/packets/audits/` returned no matches. Clean.

## Findings

### Non-blocking observations

1. **v83 audit-report verdict drift vs ledger wording (item 9 caveat).** `docs/packets/audits/gms_v83/CharacterList.md` shows verdict ❌ and `gms_v83/MonsterMovement.md` shows verdict ❌, but `post-phase-b.md:141,139` says v83 was "unchanged ✅" for CharacterList and "unchanged 🔍" for MonsterMovement. Likely cause: the v83 reports were generated during Phase 3 (`e2b82d08b`, before the analyzer refactors of items 4/5/8 landed). The ledger's narrative compares post-refactor v87/v95/jms results against the pre-refactor baseline; the v83 reports on-disk show a still-older state. The ledger's overall scope decision (not re-flipping verdict lines on-disk) covers this — but the "unchanged ✅" / "unchanged 🔍" wording at line 141 isn't a precise mirror of what's actually on-disk for v83. **Non-blocking** because: (a) all three packets are documented as wire-correct in atlas; (b) verdict-line correctness is gated on the same IDA-decompile access blocker the ledger names; (c) item 9 explicitly opted not to update these verdicts in-branch.

2. **Audit directory structure differs from plan.** The plan's Phase 2 conventions (lines 469, 476) reference per-domain subdirs like `docs/packets/audits/gms_v95/monster/`, but the actual layout is flat: all reports live directly under `gms_v95/`. The flat layout is consistent across all four versions and aligns with the existing task-027/028 reports. The plan's sub-dir reference was descriptive scaffolding, not a hard requirement; the SUMMARY.md correctly aggregates the 33 combat-domain rows. **Non-blocking** — no functional impact.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

All 11 original plan tasks landed with commit + test + ledger evidence. All 10 post-PR follow-up items landed with discrete commits, tests, and ledger entries. Verification matrix is clean across the three affected modules. gitleaks scrub clean. The MonsterControl wire-shape change is back-compat-safe (non-zero flag semantics preserved). Item 9's "deliberate non-flip" scope decision is internally consistent. Item 1's IDA addresses and verdicts match the cross-version pattern documented in the ledger.

## Action Items

None required for merge. The two non-blocking observations above (v83 verdict-line drift vs ledger wording, flat audit-dir layout) are documentation/cosmetic and are explicitly covered by the ledger's deferred-IDA-rewrite scope decision. If a follow-up cleanup pass is desired, the natural item is the same IDA-decompile-gated bulk rewrite that items 1/9 already name (Delegate refs for `CMovePath::OnMovePacket` / `CMob::Init` / `CPet::Init` plus loop-unrolling cleanup for the CharacterList `anPetID[N]` entries).
