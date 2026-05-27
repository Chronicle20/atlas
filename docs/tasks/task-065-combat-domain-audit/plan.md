# Combat-Domain Packet Audit — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Apply the audit pipeline from task-027/028 to the 31 combat-domain packets in `libs/atlas-packet/{monster,drop,reactor,pet}/{clientbound,serverbound}/`, ship wire-bug + template fixes against GMS v95 IDA, re-verify across v83/v87/JMS v185, and produce per-packet audit reports + a closing memo.

**Architecture:** Phase 0 rebases onto a post-task-028 main and confirms the analyzer/registry/run-routing baseline. Phase 1 extends `tools/packet-audit/` with combat sub-struct registrations (`MonsterModel`, `MonsterTemporaryStat`, `MultiTargetForBall`, `RandTimeForAreaAttack`) and `candidatesFromFName` routing entries. Phase 2 audits the four combat sub-domains in four tracking sub-tasks (monster → pet → drop → reactor, hot-first). Phase 3 re-runs the audit against v83, v87, and JMS v185 IDA. Phase 4 ships `post-phase-b.md`, full verification, code review, and PR.

**Tech Stack:** Go 1.24 (`go/parser` + `go/ast`), `mcp__ida-pro__*` MCP tools for live IDA decompiles, `libs/atlas-socket` reader/writer for round-trip tests, `libs/atlas-packet/test/` `pt.Variants` + `RoundTrip` helpers, GORM JSON-blob columns in `services/atlas-configurations` for template overrides. No new runtime dependencies.

---

## Conventions used by every task

- **Worktree.** All work happens in `.worktrees/task-065-combat-domain-audit/` on branch `task-065-combat-domain-audit`. Before *every* commit run `git rev-parse --show-toplevel` (must end with `/.worktrees/task-065-combat-domain-audit`) and `git branch --show-current` (must be `task-065-combat-domain-audit`); if either disagrees, STOP.
- **TDD cadence.** Test first → run-to-fail → minimal implementation → run-to-pass → commit.
- **Verification cadence (registry / run.go).** `go test -race ./tools/packet-audit/...` clean before commit.
- **Verification cadence (atlas-packet edits).** `go test -race ./libs/atlas-packet/...` clean. Every encoder fix lands with a 4-variant test sweep covering GMS v28 / v83 / v95 + JMS v185 (use `test.Variants` + `test.RoundTrip` from `libs/atlas-packet/test/context.go` and `libs/atlas-packet/test/roundtrip.go`).
- **No `*_testhelpers.go`** — use the Builder pattern already present in `libs/atlas-packet/model/`.
- **No `reflect`**, no new `interface{}` params, no benchmarks added to CI (design §10).
- **Hard guard cap.** No encoder/decoder grows beyond 2 nested region/version guards. 3+ nested → STOP, append a row to `docs/packets/ida-exports/_pending.md`.
- **gitleaks.** Absolute paths `/home/<user>/` must not appear in `docs/packets/audits/gms_v95/{monster,drop,reactor,pet}/`. Phase 4 has the mandatory scrub.
- **Tracking sub-tasks vs PR-sized commits.** Phase 2 and Phase 3 sub-tasks (Tasks 5–11) are *tracking* units, not single commits. Each ❌ verdict inside a sub-task ships its own fix commit (one fix = one commit) with a 4-variant test sweep and an IDA citation. A sub-task is "done" when every packet in its bucket has a `SUMMARY.md` row and every ❌ has either a fix commit or a `_pending.md` row.

---

## Phase 0 — Rebase gate (task-028 baseline)

`design.md §1` and Phase 1+ assume task-028's tooling (analyzer early-return suffix-taint, registry `EncodeForeign`, character `candidatesFromFName` entries, character audit reports, character IDA-export entries) is present on the branch's parent. As of planning it is NOT (see `context.md` precondition). Task 0 brings the baseline in or halts.

### Task 0: Confirm task-028 is on main; rebase if needed

**Files:**
- Read: `tools/packet-audit/internal/atlaspacket/analyzer.go`
- Read: `tools/packet-audit/internal/atlaspacket/registry.go`
- Read: `tools/packet-audit/cmd/run.go`
- Read: `docs/packets/audits/gms_v95/SUMMARY.md`

- [ ] **Step 1: Fetch main and check for task-028 merge**

```bash
git fetch origin
git log origin/main --oneline | grep -iE 'task-028|character.domain.audit' | head -5
```

Expected after merge: one or more commits with subject like `feat(task-028): character-domain packet audit ...` or PR-style merge subjects citing task-028.

If no task-028 commits appear on `origin/main`:
- STOP. Task-028 has not merged. Report to user:
  > Task-065 cannot start the audit pipeline work until task-028 merges, because Phase 1 depends on the analyzer early-return fix, `EncodeForeign` registry support, and character `candidatesFromFName` entries shipped there. Block this task or invert order — open task-028 PR first.
- Do not proceed to Step 2.

- [ ] **Step 2: Rebase task-065 onto post-task-028 main**

```bash
git rebase origin/main
```

Conflict expected files (none of these are touched by task-065's two existing commits, which are docs-only): the rebase should be conflict-free. If a conflict appears, resolve in favour of `origin/main` (we have nothing to keep on the task-028 side here) and continue.

After rebase, re-verify:
```bash
git rev-parse --show-toplevel  # ends in /.worktrees/task-065-combat-domain-audit
git branch --show-current      # task-065-combat-domain-audit
git log --oneline main..HEAD   # only spec + design commits
```

- [ ] **Step 3: Verify task-028 artifacts present**

Run these four checks, all must succeed:

```bash
grep -q 'blockTerminatesWithReturn' tools/packet-audit/internal/atlaspacket/analyzer.go
grep -q 'EncodeForeign' tools/packet-audit/internal/atlaspacket/registry.go
test "$(grep -c 'case "' tools/packet-audit/cmd/run.go)" -ge 70
ls docs/packets/audits/gms_v95/CharacterSpawn.md 2>/dev/null
```

If any check fails: the rebase pulled in only partial task-028 work. STOP and ask the user to confirm task-028 status.

- [ ] **Step 4: Smoke-run the existing pipeline (login + character) to confirm the baseline is clean**

```bash
go test -race ./tools/packet-audit/...
```

Expected: all green.

```bash
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_v95.json \
  --output           docs/packets/audits/gms_v95
```

Expected: ≤ 60 s runtime, reports for all task-027 + task-028 FNames refreshed (git diff on `docs/packets/audits/gms_v95/SUMMARY.md` should show only re-ordering or no change).

- [ ] **Step 5: No commit — Phase 0 is a confirmation gate only.**

If Steps 2–4 succeeded, proceed to Phase 1. If anything was modified accidentally during the smoke run (e.g. SUMMARY.md re-ordered), `git checkout -- docs/packets/audits/gms_v95/` to restore.

---

## Phase 1 — TypeRegistry extensions + `candidatesFromFName` routing

Three tasks. Each adds combat-domain coverage to the audit tooling without touching `libs/atlas-packet/` itself. Phase 1 exit: registry tests cover all four predicted combat sub-structs, run.go routes every combat FName.

### Task 1: Registry coverage fixture for combat sub-structs

**Files:**
- Modify: `tools/packet-audit/internal/atlaspacket/registry_test.go`

The registry auto-discovers any struct with an `Encode` method (registry.go pass-2 switch case `"Encode"`). All four combat sub-structs already have `Encode` methods:

| Type | File | Method line |
|---|---|---|
| `MonsterModel` | `libs/atlas-packet/model/monster.go:493` | `Encode` |
| `MonsterTemporaryStat` | `libs/atlas-packet/model/monster.go:241` | `Encode` (pointer receiver) |
| `MultiTargetForBall` | `libs/atlas-packet/model/multi_target_for_ball.go` | `Encode` |
| `RandTimeForAreaAttack` | `libs/atlas-packet/model/rand_time_for_area_attack.go` | `Encode` |

Expected: pass-2 picks them up automatically. The test pins this so a future registry refactor that breaks pointer-receiver dispatch fails loudly.

- [ ] **Step 1: Write the fixture**

Append to `tools/packet-audit/internal/atlaspacket/registry_test.go`:

```go
func TestRegistryRegistersCombatSubStructs(t *testing.T) {
    _, thisFile, _, _ := runtime.Caller(0)
    root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
    reg, err := NewTypeRegistry(root)
    if err != nil {
        t.Fatal(err)
    }
    for _, name := range []string{
        "MonsterModel",
        "MonsterTemporaryStat",
        "MultiTargetForBall",
        "RandTimeForAreaAttack",
    } {
        if !reg.HasType(name) {
            t.Errorf("registry missing combat sub-struct %s", name)
            continue
        }
        calls, ok := reg.Calls(name)
        if !ok || len(calls) == 0 {
            t.Errorf("%s.Encode produced no calls (ok=%v len=%d)", name, ok, len(calls))
        }
    }
}

func TestRegistryStillRegistersMovementAfterCombatExtension(t *testing.T) {
    _, thisFile, _, _ := runtime.Caller(0)
    root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
    reg, err := NewTypeRegistry(root)
    if err != nil {
        t.Fatal(err)
    }
    // Regression guard: task-028 registered Movement + element sub-types.
    // Combat domain re-uses Movement; if a refactor drops these, surface here.
    for _, name := range []string{
        "Movement",
        "Element",
        "NormalElement",
        "TeleportElement",
        "StartFallDownElement",
        "FlyingBlockElement",
        "JumpElement",
        "StatChangeElement",
    } {
        if !reg.HasType(name) {
            t.Errorf("registry missing movement sub-type %s (task-028 regression)", name)
        }
    }
}
```

- [ ] **Step 2: Run the test**

```
go test -race ./tools/packet-audit/internal/atlaspacket/ -run 'TestRegistryRegistersCombatSubStructs|TestRegistryStillRegistersMovementAfterCombatExtension' -v
```

Expected: PASS — auto-discovery should have already picked these up. If any FAIL:
- `MonsterModel` is a value receiver (`func (m MonsterModel) Encode...`)? Verified `monster.go:493` declares `func (m *MonsterModel) Encode` (pointer receiver). `receiverIdent` in `registry.go` already handles `*Foo` via the StarExpr branch — verify if a test FAILS.
- For any FAIL: open `tools/packet-audit/internal/atlaspacket/registry.go` `receiverIdent` and check it covers all observed receiver forms. Fix the receiver-parsing path BEFORE editing the test.

- [ ] **Step 3: Commit**

```bash
git add tools/packet-audit/internal/atlaspacket/registry_test.go
git commit -m "test(packet-audit): assert combat sub-structs MonsterModel/TemporaryStat/MultiTargetForBall/RandTimeForAreaAttack registered

Pins auto-discovery coverage for the four combat-domain sub-structs the
analyzer must descend into (monster spawn/control, monster movement
multi-target + area-attack, monster stat set/reset). Also regression-
guards task-028's Movement element sub-types."
```

---

### Task 2: `candidatesFromFName` routing for combat FNames

**Files:**
- Modify: `tools/packet-audit/cmd/run.go`

The combat FNames the audit will resolve. Atlas writer/handler names below come from the `Operation()`/handler-const enumeration in `context.md`'s inventory tables. IDA FNames are the *expected* names; the executor MUST verify each against the live IDA decompile before the routing entry is finalised — wrong IDA names produce silent diff-engine misses.

Predicted routing table (verify each FName via `mcp__ida-pro__get_function_by_name` during execution; adjust if IDA uses a different mangled name):

| IDA FName (predicted) | Atlas name | Direction |
|---|---|---|
| `CMobPool::OnSpawnMob` | `MonsterSpawn` | clientbound |
| `CMobPool::OnControlMob` | `MonsterControl` | clientbound |
| `CMobPool::OnDestroyMob` | `MonsterDestroy` | clientbound |
| `CMob::OnMobDamaged` | `MonsterDamage` | clientbound |
| `CMob::OnMobHpIndicator` | `MonsterHealth` | clientbound |
| `CMobPool::OnMoveMob` | `MonsterMovement` | clientbound |
| `CMobPool::OnMoveMobResult` | `MonsterMovementAck` | clientbound |
| `CMob::OnSetTemporaryStat` | `MonsterStatSet` | clientbound |
| `CMob::OnResetTemporaryStat` | `MonsterStatReset` | clientbound |
| `CMobPool::OnMobMove` | `MonsterMovementHandle` | serverbound |
| `CDropPool::OnDropEnterField` | `DropSpawn` | clientbound |
| `CDropPool::OnDropLeaveField` | `DropDestroy` | clientbound |
| `CDropPool::OnDropPickUpRequest` | `DropPickUpHandle` | serverbound |
| `CReactorPool::OnReactorEnterField` | `ReactorSpawn` | clientbound |
| `CReactorPool::OnReactorChangeState` | `ReactorHit` | clientbound |
| `CReactorPool::OnReactorLeaveField` | `ReactorDestroy` | clientbound |
| `CReactorPool::OnHitReactor` | `ReactorHitHandle` | serverbound |
| `CUserLocal::OnPetActivated` | `PetActivated` | clientbound |
| `CUserLocal::OnPetMove` | `PetMovement` | clientbound |
| `CUserLocal::OnPetAction` | `PetCommandResponse` | clientbound |
| `CUserLocal::OnPetChat` | `PetChat` | clientbound |
| `CUserLocal::OnPetExceptionList` | `PetExcludeResponse` | clientbound |
| `CCashShop::OnCashPetFoodResult` | `PetCashFoodResult` | clientbound |
| `CWvsContext::SendActivatePetPacket` | `PetSpawnHandle` | serverbound |
| `CWvsContext::SendPetMovePacket` | `PetMovementHandle` | serverbound |
| `CWvsContext::SendPetActionPacket` | `PetCommandHandle` | serverbound |
| `CWvsContext::SendPetChatPacket` | `PetChatHandle` | serverbound |
| `CWvsContext::SendPetExceptionList` | `PetItemExcludeHandle` | serverbound |
| `CWvsContext::SendPetFoodPacket` | `PetFoodHandle` | serverbound |
| `CWvsContext::SendPetItemPacket` | `PetItemUseHandle` | serverbound |
| `CWvsContext::SendPetDropPickUpPacket` | `PetDropPickUpHandle` | serverbound |

That's 31 predicted routing entries (matches the 31-packet inventory in context.md).

- [ ] **Step 1: Resolve real FNames in IDA (v95 first)**

Confirm v95 IDA is loaded:
```
mcp__ida-pro__get_metadata
```

For each row in the predicted table:
```
mcp__ida-pro__get_function_by_name("<predicted FName>")
```

If it returns the function, the predicted name is correct. If it returns "not found":
- Try variants: replace `OnXxx` with `SendXxx`, swap class prefix (`CMob` ↔ `CMobPool`).
- Use `mcp__ida-pro__list_functions_filter` with a substring like `Mob` / `Reactor` / `Pet` / `Drop` to enumerate candidates.
- Record the actual FName in the audit notes — the entry committed in Step 3 uses the verified name, not the prediction.

- [ ] **Step 2: Append combat routing entries to `candidatesFromFName`**

Insert new `case` entries inside `candidatesFromFName` in `tools/packet-audit/cmd/run.go`, right before the `return nil` at line 196 (in current numbering; line drift after task-028 rebase — locate by `return nil` at end of function). One block per sub-domain to keep the diff scannable:

```go
// --- monster domain ---
case "CMobPool::OnSpawnMob":
    return []candidate{{name: "MonsterSpawn", dir: csvpkg.DirClientbound}}
case "CMobPool::OnControlMob":
    return []candidate{{name: "MonsterControl", dir: csvpkg.DirClientbound}}
case "CMobPool::OnDestroyMob":
    return []candidate{{name: "MonsterDestroy", dir: csvpkg.DirClientbound}}
case "CMob::OnMobDamaged":
    return []candidate{{name: "MonsterDamage", dir: csvpkg.DirClientbound}}
case "CMob::OnMobHpIndicator":
    return []candidate{{name: "MonsterHealth", dir: csvpkg.DirClientbound}}
case "CMobPool::OnMoveMob":
    return []candidate{{name: "MonsterMovement", dir: csvpkg.DirClientbound}}
case "CMobPool::OnMoveMobResult":
    return []candidate{{name: "MonsterMovementAck", dir: csvpkg.DirClientbound}}
case "CMob::OnSetTemporaryStat":
    return []candidate{{name: "MonsterStatSet", dir: csvpkg.DirClientbound}}
case "CMob::OnResetTemporaryStat":
    return []candidate{{name: "MonsterStatReset", dir: csvpkg.DirClientbound}}
case "CMobPool::OnMobMove":
    return []candidate{{name: "MonsterMovementHandle", dir: csvpkg.DirServerbound}}

// --- drop domain ---
case "CDropPool::OnDropEnterField":
    return []candidate{{name: "DropSpawn", dir: csvpkg.DirClientbound}}
case "CDropPool::OnDropLeaveField":
    return []candidate{{name: "DropDestroy", dir: csvpkg.DirClientbound}}
case "CDropPool::OnDropPickUpRequest":
    return []candidate{{name: "DropPickUpHandle", dir: csvpkg.DirServerbound}}

// --- reactor domain ---
case "CReactorPool::OnReactorEnterField":
    return []candidate{{name: "ReactorSpawn", dir: csvpkg.DirClientbound}}
case "CReactorPool::OnReactorChangeState":
    return []candidate{{name: "ReactorHit", dir: csvpkg.DirClientbound}}
case "CReactorPool::OnReactorLeaveField":
    return []candidate{{name: "ReactorDestroy", dir: csvpkg.DirClientbound}}
case "CReactorPool::OnHitReactor":
    return []candidate{{name: "ReactorHitHandle", dir: csvpkg.DirServerbound}}

// --- pet domain ---
case "CUserLocal::OnPetActivated":
    return []candidate{{name: "PetActivated", dir: csvpkg.DirClientbound}}
case "CUserLocal::OnPetMove":
    return []candidate{{name: "PetMovement", dir: csvpkg.DirClientbound}}
case "CUserLocal::OnPetAction":
    return []candidate{{name: "PetCommandResponse", dir: csvpkg.DirClientbound}}
case "CUserLocal::OnPetChat":
    return []candidate{{name: "PetChat", dir: csvpkg.DirClientbound}}
case "CUserLocal::OnPetExceptionList":
    return []candidate{{name: "PetExcludeResponse", dir: csvpkg.DirClientbound}}
case "CCashShop::OnCashPetFoodResult":
    return []candidate{{name: "PetCashFoodResult", dir: csvpkg.DirClientbound}}
case "CWvsContext::SendActivatePetPacket":
    return []candidate{{name: "PetSpawnHandle", dir: csvpkg.DirServerbound}}
case "CWvsContext::SendPetMovePacket":
    return []candidate{{name: "PetMovementHandle", dir: csvpkg.DirServerbound}}
case "CWvsContext::SendPetActionPacket":
    return []candidate{{name: "PetCommandHandle", dir: csvpkg.DirServerbound}}
case "CWvsContext::SendPetChatPacket":
    return []candidate{{name: "PetChatHandle", dir: csvpkg.DirServerbound}}
case "CWvsContext::SendPetExceptionList":
    return []candidate{{name: "PetItemExcludeHandle", dir: csvpkg.DirServerbound}}
case "CWvsContext::SendPetFoodPacket":
    return []candidate{{name: "PetFoodHandle", dir: csvpkg.DirServerbound}}
case "CWvsContext::SendPetItemPacket":
    return []candidate{{name: "PetItemUseHandle", dir: csvpkg.DirServerbound}}
case "CWvsContext::SendPetDropPickUpPacket":
    return []candidate{{name: "PetDropPickUpHandle", dir: csvpkg.DirServerbound}}
```

**Replace any predicted FName above with the verified IDA name from Step 1.** Do not commit a guessed FName.

- [ ] **Step 3: Verify case-count**

```
grep -c 'case "' tools/packet-audit/cmd/run.go
```

Expected: 78 (task-028 baseline) + 31 = **109**.

If lower: a routing entry didn't take. Re-check the inserted block.

- [ ] **Step 4: Compile + test**

```
go test -race ./tools/packet-audit/...
```

Expected: green. No new tests yet — Task 1's fixtures cover registry. Diff-engine routing is exercised end-to-end in Phase 2.

- [ ] **Step 5: Commit**

```bash
git add tools/packet-audit/cmd/run.go
git commit -m "feat(packet-audit): route combat-domain FNames to atlas writers/handlers

Adds 31 candidatesFromFName entries covering monster (10), drop (3),
reactor (4), pet (14) packets. FNames verified against GMS v95 IDA.
Required for Phase 2 audit to bind IDA decompiles to atlas-packet
structs."
```

---

### Task 3: Compatibility check — analyzer descends into combat sub-structs

**Files:**
- Modify: `tools/packet-audit/internal/atlaspacket/analyzer_test.go` (test only — no analyzer change expected)

The analyzer's `FlattenWithRegistry` cycle guard (shipped task-028) should already handle combat sub-structs. This task is a *guard* — if combat sub-structs trigger a regression, we catch it here before Phase 2.

- [ ] **Step 1: Write a flatten fixture for `MonsterModel`**

Append to `tools/packet-audit/internal/atlaspacket/analyzer_test.go`:

```go
func TestFlattenMonsterSpawnExpandsMonsterModel(t *testing.T) {
    _, thisFile, _, _ := runtime.Caller(0)
    root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
    reg, err := NewTypeRegistry(root)
    if err != nil {
        t.Fatal(err)
    }
    src := filepath.Join(root, "monster", "clientbound", "spawn.go")
    calls, err := AnalyzeFile(src, "Spawn", "Encode")
    if err != nil {
        t.Fatal(err)
    }
    flat := FlattenWithRegistry(calls, reg)
    // Pre-flatten there should be a KindRecurse marker for MonsterModel;
    // post-flatten the marker must be gone and the MonsterModel primitives
    // must appear inline. Use any well-known MonsterModel field as the canary —
    // e.g. the first Encode2 emitted for foothold.
    sawRecurse := false
    for _, c := range flat {
        if c.Kind == KindRecurse {
            sawRecurse = true
            break
        }
    }
    if sawRecurse {
        t.Fatalf("FlattenWithRegistry left a KindRecurse marker in MonsterSpawn after expansion")
    }
    if len(flat) <= len(calls) {
        t.Fatalf("flatten did not expand sub-struct: pre=%d post=%d", len(calls), len(flat))
    }
}

func TestFlattenMonsterStatSetCycleSafe(t *testing.T) {
    _, thisFile, _, _ := runtime.Caller(0)
    root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
    reg, err := NewTypeRegistry(root)
    if err != nil {
        t.Fatal(err)
    }
    src := filepath.Join(root, "monster", "clientbound", "stat.go")
    calls, err := AnalyzeFile(src, "StatSet", "Encode")
    if err != nil {
        t.Fatal(err)
    }
    // Should not hang; cycle guard ensures bounded recursion.
    _ = FlattenWithRegistry(calls, reg)
}
```

- [ ] **Step 2: Run**

```
go test -race ./tools/packet-audit/internal/atlaspacket/ -run 'TestFlattenMonsterSpawn|TestFlattenMonsterStatSet' -v
```

Expected: PASS. Specifically `TestFlattenMonsterStatSetCycleSafe` should complete in well under 1 s; if it hangs, the cycle guard isn't catching a MonsterTemporaryStat → MonsterTemporaryStat self-edge. STOP and inspect `FlattenWithRegistry`'s visited-set.

If `TestFlattenMonsterSpawnExpandsMonsterModel` fails because `flat` still contains a KindRecurse marker:
- Inspect with `t.Logf("%+v", flat)`. The most likely cause is a method-name mismatch in registry pass-2. Fix `registry.go` rather than the test.

- [ ] **Step 3: Commit**

```bash
git add tools/packet-audit/internal/atlaspacket/analyzer_test.go
git commit -m "test(packet-audit): assert MonsterSpawn/StatSet flatten through MonsterModel + MonsterTemporaryStat

Guards Phase 2 against silent regressions in FlattenWithRegistry's
cycle-safety + sub-struct descent when combat hot-path encoders are
analysed."
```

---

## Phase 2 — v95 combat audit per sub-domain

Four tracking sub-tasks. Hot-path-first order: monster → pet → drop → reactor. Each sub-task is a *tracking* unit:

1. Run audit, scoped output captured per sub-domain dir (`docs/packets/audits/gms_v95/<domain>/`).
2. Triage each report:
   - **✅** → nothing to do.
   - **⚠️** → annotate the `.md` with a one-line ack footer; commit alone.
   - **❌** → one of: real wire bug → fix + 4-variant test sweep; template drift → template fix + IDA case-statement cite; analyzer FP → `_pending.md` row.
3. One commit per fix. One bucket commit per sub-domain audit report batch.

Before any encoder mutation in monster or pet, fill the missing test files (`monster/clientbound/movement_test.go`, `pet/clientbound/movement_test.go`) per design §6 + §10. This is a sub-step in Tasks 4 and 6 respectively.

The audit command for all of Phase 2 is identical (re-uses Phase 0 Step 4 invocation). Per-domain rows land under `docs/packets/audits/gms_v95/<domain>/`. `SUMMARY.md` accumulates.

### Task 4: Phase 2a — monster sub-domain (9 cb + 1 sb = 10 packets)

Hottest combat sub-domain. Run before pet/drop/reactor so analyzer false positives and registry gaps surface early.

- [ ] **Step 1: Fill the `monster/clientbound/movement.go` test gap (preflight)**

`libs/atlas-packet/monster/clientbound/movement.go` has no `_test.go` sibling. Add one BEFORE running the audit so any fix that drops out of triage has a byte-baseline.

Create `libs/atlas-packet/monster/clientbound/movement_test.go`:

```go
package clientbound

import (
    "testing"

    "github.com/Chronicle20/atlas/libs/atlas-packet/model"
    "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestMonsterMovementRoundTrip(t *testing.T) {
    mv := model.Movement{} // zero value; round-trip baseline only
    mt := model.MultiTargetForBall{}
    rt := model.RandTimeForAreaAttack{}
    input := NewMonsterMovement(5001, false, true, false, 0, 0, 0, mt, rt, mv)
    for _, v := range test.Variants {
        t.Run(v.Name, func(t *testing.T) {
            ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
            test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
        })
    }
}

func TestMonsterMovementRoundTripWithSkill(t *testing.T) {
    mv := model.Movement{}
    mt := model.MultiTargetForBall{}
    rt := model.RandTimeForAreaAttack{}
    input := NewMonsterMovement(5001, true, true, true, 1, 100, 5, mt, rt, mv)
    for _, v := range test.Variants {
        t.Run(v.Name, func(t *testing.T) {
            ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
            test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
        })
    }
}
```

If `model.Movement`, `model.MultiTargetForBall`, or `model.RandTimeForAreaAttack` zero-value isn't decodeable (e.g. needs a discriminator byte set), inspect each `_test.go` for an existing usable constructor; fall back to that. The point is byte-baseline coverage, not exhaustive scenarios.

Run:
```
go test -race ./libs/atlas-packet/monster/clientbound/... -run 'TestMonsterMovement' -v
```
Expected: green. If any variant fails, EITHER the zero-value is invalid (use a working constructor) OR there's a real round-trip bug — STOP and triage before continuing the audit.

Commit:
```bash
git add libs/atlas-packet/monster/clientbound/movement_test.go
git commit -m "test(atlas-packet,monster/movement): add 4-variant round-trip baseline

Closes the missing-test gap flagged in task-065 design §6. Establishes
byte-output baseline before Phase 2a triage may mutate the encoder."
```

- [ ] **Step 2: Run the audit (full pipeline; monster reports appear under `gms_v95/`)**

```
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_v95.json \
  --output           docs/packets/audits/gms_v95
```

Expected: 10 new `.md`/`.json` files for monster packets (`MonsterSpawn.md`, `MonsterControl.md`, `MonsterDamage.md`, `MonsterDestroy.md`, `MonsterHealth.md`, `MonsterMovement.md`, `MonsterMovementAck.md`, `MonsterStatSet.md`, `MonsterStatReset.md`, `MonsterMovementHandle.md`). 10 new rows in `SUMMARY.md`.

If fewer reports appear: the matching IDA-export entries are missing. Populate `docs/packets/ida-exports/gms_v95.json` for any FName that didn't resolve (Step 3 below).

- [ ] **Step 3: For each monster FName not yet in `gms_v95.json`, capture IDA evidence and append**

For each missing FName from the Phase 1 Task 2 verified list:

```
mcp__ida-pro__get_function_by_name("<FName>")          # confirm address
mcp__ida-pro__decompile_function(<addr>)               # capture body
```

Append an entry to `docs/packets/ida-exports/gms_v95.json` matching the existing schema. Run the pipeline again (Step 2) to refresh.

- [ ] **Step 4: Triage each monster ❌ verdict**

For each `❌` row in `SUMMARY.md` whose name starts with `Monster`:
- **Atlas wire bug** (width / order / missing field) → fix in `libs/atlas-packet/monster/{clientbound,serverbound}/<pkt>.go`. Extend `<pkt>_test.go` with a hex-baseline assertion using `pt.Variants`. Cite IDA in commit message.
- **Template opcode drift** → fix `services/atlas-configurations/seed-data/templates/template_gms_*_1.json` + `template_jms_185_1.json` if applicable. Cite the IDA case-statement value.
- **Analyzer descent gap** — sub-struct unregistered → fix in Phase 1 work (regression — return to Task 1).
- **Sub-op enum drift** (`MonsterDamage` flat enum, `MonsterStat*` mask logic) → defer to `docs/packets/ida-exports/_pending.md` under a new heading `## Still pending — combat domain` (create heading on first deferral).
- **Bare handler / no atlas-packet decoder** → defer to `_pending.md` per task-028 §1 working assumption.

For sub-struct-descent FPs on `MonsterSpawn` and `MonsterControl` (predicted per design §3): manual IDA verdict captured in the `.md`, `_pending.md` row added, SUMMARY stays ❌ tagged "(analyzer FP — manual IDA confirms ✅)". Do NOT mutate the encoder.

- [ ] **Step 5: For each Atlas wire-bug fix, ship a 4-variant test sweep**

Hex-output template (replace `<...>` with IDA-derived values):

```go
import (
    "context"
    "encoding/hex"
    "testing"

    "github.com/Chronicle20/atlas/libs/atlas-packet/model"
    "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestMonsterSpawnByteForByte(t *testing.T) {
    m := model.NewMonster(100, 200, 5, 300, model.MonsterAppearTypeRegen, 0)
    input := NewMonsterSpawn(5001, true, 100100, m)
    cases := []struct {
        name string
        v    test.TenantVariant
        want string // hex
    }{
        {"gms_v83", test.Variants[1], "<...>"},
        {"gms_v95", test.Variants[2], "<...>"},
        {"jms_v185", test.Variants[3], "<...>"},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            ctx := test.CreateContext(tc.v.Region, tc.v.MajorVersion, tc.v.MinorVersion)
            // testLogger() lives in the existing _test.go files; reuse.
            got := input.Encode(testLogger(), ctx)(nil)
            if hex.EncodeToString(got) != tc.want {
                t.Fatalf("encode mismatch\n got %s\nwant %s", hex.EncodeToString(got), tc.want)
            }
        })
    }
}
```

Hex values come from IDA decompiles or from the pre-fix analyzer run.

- [ ] **Step 6: Run targeted tests per fix**

```
go test -race ./libs/atlas-packet/monster/... -run '<TestName>' -v
```

Expected: green.

- [ ] **Step 7: Per-fix commit format**

Atlas wire-bug fixes:
```bash
git add libs/atlas-packet/monster/clientbound/<pkt>.go \
        libs/atlas-packet/monster/clientbound/<pkt>_test.go
git commit -m "fix(atlas-packet,monster/<pkt>): <one-line summary>

Cites IDA <CMobPool::OnXxx>@<addr>: <one-line evidence>."
```

Template fixes:
```bash
git add services/atlas-configurations/seed-data/templates/template_*.json
git commit -m "fix(configurations,templates): <pkt> opcode <old>→<new> for <region/version>

IDA case-statement value at <FName>@<addr>."
```

`_pending.md` deferrals:
```bash
git add docs/packets/ida-exports/_pending.md
git commit -m "audit(monster/<pkt>): defer — <one-line reason>"
```

- [ ] **Step 8: Bucket commit for monster audit reports**

```bash
git add docs/packets/audits/gms_v95/Monster*.{md,json} \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(monster): GMS v95 sub-domain audit (10 packets)"
```

- [ ] **Step 9: Exit gate**

```
grep -E '^\| (Monster|Move)' docs/packets/audits/gms_v95/SUMMARY.md | wc -l
```

Expected: 10. Every row carries ✅, ⚠️, or ❌ + (if ❌) a matching fix commit or `_pending.md` row visible in `git log monster..HEAD --oneline | grep -i monster`.

Verify clean:
```
go test -race ./libs/atlas-packet/monster/... ./tools/packet-audit/...
```

---

### Task 5: Phase 2b — pet sub-domain (6 cb + 8 sb = 14 packets)

Largest combat sub-domain. Pet command sub-op dispatch + self/foreign perspective candidate (design §5).

- [ ] **Step 1: Confirm `pet/clientbound/activated_body.go` is a wrapper, not an independent encoder**

```
grep -n 'func\|return\|Encode' libs/atlas-packet/pet/clientbound/activated_body.go
```

Verified per context.md: `PetSpawnBody` and `PetDespawnBody` call `NewPetSpawnActivated.Encode` / `NewPetDespawnActivated.Encode`. There is NO standalone encoder body in `activated_body.go`. The audit report `PetActivated.md` (produced by Step 3) documents this in prose; do not create a separate audit row for `activated_body.go`.

- [ ] **Step 2: Fill the `pet/clientbound/movement.go` test gap (preflight)**

Create `libs/atlas-packet/pet/clientbound/movement_test.go`:

```go
package clientbound

import (
    "testing"

    "github.com/Chronicle20/atlas/libs/atlas-packet/model"
    "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestPetMovementRoundTrip(t *testing.T) {
    mv := model.Movement{}
    input := NewPetMovement(2001, 0, mv)
    for _, v := range test.Variants {
        t.Run(v.Name, func(t *testing.T) {
            ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
            test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
        })
    }
}
```

Run + commit identical to Task 4 Step 1:

```
go test -race ./libs/atlas-packet/pet/clientbound/... -run 'TestPetMovement' -v
```

```bash
git add libs/atlas-packet/pet/clientbound/movement_test.go
git commit -m "test(atlas-packet,pet/movement): add 4-variant round-trip baseline

Closes the missing-test gap flagged in task-065 design §6. Establishes
byte-output baseline before Phase 2b triage may mutate the encoder."
```

- [ ] **Step 3: Run the audit (same command as Task 4 Step 2)**

Expected: 14 new `.md`/`.json` files for pet packets (`PetActivated.md`, `PetCashFoodResult.md`, `PetChat.md`, `PetCommandResponse.md`, `PetExcludeResponse.md`, `PetMovement.md`, `PetSpawnHandle.md`, `PetMovementHandle.md`, `PetCommandHandle.md`, `PetChatHandle.md`, `PetItemExcludeHandle.md`, `PetFoodHandle.md`, `PetItemUseHandle.md`, `PetDropPickUpHandle.md`).

If IDA exports are missing for pet FNames, repeat Task 4 Step 3's MCP loop to populate `gms_v95.json`.

- [ ] **Step 4: Triage per Task 4 Step 4**

Anticipate these specific outcomes (verify each in the actual report):
- **`PetActivated`** — sub-op on `active` bool. Likely ✅ if perspective is local-only; if IDA shows `CUserPool::On*` dispatcher offset, the wire prepends an extra `ownerCharacterId` — that's a real bug, fix in encoder + 4-variant test (per design §5.1).
- **`PetCommandResponse`** — `mode` byte uses values 0 (action) and 1 (food). Verify against IDA case-statement; flag as sub-op drift if IDA has more modes.
- **`PetExcludeResponse`** — `byte(len(excludeIds))` count→loop. Likely linearised in the analyzer FP. Defer to `_pending.md` if ❌.
- **8 serverbound pets** — each has its own dispatcher case in `CUserPool::OnRemotePacket` or `CWvsContext::On*`. Confirm each decoder maps 1:1.

Apply triage flavours per Task 4 Step 4.

- [ ] **Step 5: Per-fix commits + bucket commit**

Per-fix commits identical to Task 4 Step 7 (substitute `pet` for `monster` in paths and commit subjects).

Bucket commit:
```bash
git add docs/packets/audits/gms_v95/Pet*.{md,json} \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(pet): GMS v95 sub-domain audit (14 packets)"
```

- [ ] **Step 6: Exit gate**

```
grep -E '^\| Pet' docs/packets/audits/gms_v95/SUMMARY.md | wc -l
```

Expected: 14. Every row carries ✅/⚠️/❌. Verify clean:
```
go test -race ./libs/atlas-packet/pet/... ./tools/packet-audit/...
```

---

### Task 6: Phase 2c — drop sub-domain (2 cb + 1 sb = 3 packets)

Smallest after reactor. `DropSpawn` is hot-path (per-drop landing).

- [ ] **Step 1: Run the audit (same command as Task 4 Step 2)**

Expected: 3 new `.md`/`.json` files (`DropSpawn.md`, `DropDestroy.md`, `DropPickUpHandle.md`). Populate IDA exports per Task 4 Step 3 if any FName misses.

- [ ] **Step 2: Triage per Task 4 Step 4**

Anticipated:
- **`DropSpawn`** — branched on `meso > 0` (item vs meso); branched on `enterType != 2`. Analyzer may flatten both branches; manual IDA confirms.
- **`DropDestroy`** — branched on `destroyType >= 2`; `petSlot >= 0` optional byte. Sub-op-ish; may need `_pending.md` row.
- **`DropPickUpHandle`** — serverbound; usually trivial.

- [ ] **Step 3: Per-fix + bucket commit**

Per-fix commits (Task 4 Step 7 pattern). Bucket commit:

```bash
git add docs/packets/audits/gms_v95/Drop*.{md,json} \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(drop): GMS v95 sub-domain audit (3 packets)"
```

- [ ] **Step 4: Exit gate**

```
grep -E '^\| Drop' docs/packets/audits/gms_v95/SUMMARY.md | wc -l
```
Expected: 3. Verify clean.

---

### Task 7: Phase 2d — reactor sub-domain (3 cb + 1 sb = 4 packets)

Cold-path. Quick.

- [ ] **Step 1: Run the audit (same command as Task 4 Step 2)**

Expected: 4 new `.md`/`.json` files (`ReactorSpawn.md`, `ReactorHit.md`, `ReactorDestroy.md`, `ReactorHitHandle.md`). Populate IDA exports if missing.

- [ ] **Step 2: Triage per Task 4 Step 4**

Anticipated all ✅ or trivial fixes — reactor packets are flat.

- [ ] **Step 3: Per-fix + bucket commit**

Per-fix commits per Task 4 Step 7. Bucket commit:

```bash
git add docs/packets/audits/gms_v95/Reactor*.{md,json} \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(reactor): GMS v95 sub-domain audit (4 packets)"
```

- [ ] **Step 4: Exit gate**

```
grep -E '^\| Reactor' docs/packets/audits/gms_v95/SUMMARY.md | wc -l
```
Expected: 4.

**Phase 2 exit:** `grep -cE '^\| (Monster|Move|Drop|Reactor|Pet)' docs/packets/audits/gms_v95/SUMMARY.md` → at least 31. Every ❌ has a fix commit OR a `_pending.md` row.

```
go build ./...
go test -race ./libs/atlas-packet/... ./tools/packet-audit/...
```
Both clean.

---

## Phase 3 — Cross-version pass

Three tracking sub-tasks. One IDA load per version, user-driven. Each sub-task is "done" when:
- `docs/packets/ida-exports/<version>.json` exists with combat-domain entries for every FName audited in Phase 2.
- Audit re-run against version's template + IDA export under `docs/packets/audits/<version>/`.
- Every divergence vs v95 atlas-packet has a `Region/MajorVersion` gate that handles it (audit report only), a gate fix on this branch (+ test sweep), OR a template fix.

If any encoder hits 3+ nested `if t.Region()` / `if t.MajorVersion()` levels (design §7), STOP, append a `_pending.md` row, do not refactor in this task.

### Task 8: GMS v83 cross-version pass

**Files:**
- Modify: `docs/packets/ida-exports/gms_v83.json` (exists — append combat entries)
- Modify (per fix): `libs/atlas-packet/{monster,drop,reactor,pet}/**/*.go` + matching `_test.go`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- Create (if not present): `docs/packets/audits/gms_v83/` directory + per-packet reports

- [ ] **Step 1: Confirm v83 IDA loaded**

```
mcp__ida-pro__get_metadata
```

Expected: `binary` field corresponds to GMS v83. If not, ask user to swap before continuing.

- [ ] **Step 2: Populate `gms_v83.json` for combat FNames**

For each FName entered in Phase 1 Task 2 routing table (verified GMS v95 names), repeat in v83 IDA:
```
mcp__ida-pro__get_function_by_name("<FName>")
mcp__ida-pro__decompile_function(<addr>)
```

Append the function entry to `docs/packets/ida-exports/gms_v83.json` (same schema). Do not reorder existing entries.

If a FName doesn't exist in v83 IDA: that packet is a v95-or-newer addition. Skip (no entry needed); document in `post-phase-b.md` "remaining work".

- [ ] **Step 3: Re-run the audit against v83**

```
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_gms_83_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_v83.json \
  --output           docs/packets/audits/gms_v83
```

If `docs/packets/audits/gms_v83/` doesn't exist yet (only login-domain reports from task-027 are likely present), the tool creates it.

- [ ] **Step 4: Triage divergences (Phase 3 buckets)**

For each ❌ in v83 audit:
- v95 fix gated `MajorVersion() >= 95`? → no v83 regression. Audit-report only.
- v95 fix gated `Region() == "GMS"` (no major-version filter)? → v83 IDA confirms same behaviour? Yes: tighten the gate to `>=95`; recompile + re-run tests. No: leave + document.
- *New* v83-only mismatch v95 didn't surface? → genuine cross-version bug. Fix with 4-variant test sweep + `Region/MajorVersion` gate. Hard cap on nested guards still applies (design §7).

- [ ] **Step 5: Per-fix commits + bucket commit**

Per-fix commit format:
```
fix(atlas-packet,<domain>/<pkt>): widen/narrow v83 gate for <field>

Cites IDA v83 <FName>@<addr>: <one-line evidence>.
```

Bucket commit:
```bash
git add docs/packets/ida-exports/gms_v83.json \
        docs/packets/audits/gms_v83/
git commit -m "audit(combat): GMS v83 cross-version pass (phase-3-v83)"
```

- [ ] **Step 6: Hard-cap check**

If any combat encoder now has 3+ nested `if t.Region()` / `if t.MajorVersion()` guards, STOP per design §7. Append to `_pending.md`:
```
## Phase 3 v83 — hard-cap stops

| Encoder | Triggering chain | Notes |
|---|---|---|
| <path> | <regions/versions> | Refactor needed; sibling task. |
```
Do not refactor in this task.

---

### Task 9: GMS v87 cross-version pass

Identical shape to Task 8. Replace `v83` with `v87` everywhere. Template: `template_gms_87_1.json`. Export file: `docs/packets/ida-exports/gms_v87.json` (CREATE — file does not exist as of planning; verify with `ls docs/packets/ida-exports/` first).

- [ ] **Step 1: Confirm v87 IDA loaded** (same as Task 8 Step 1).
- [ ] **Step 2: If `gms_v87.json` is absent, create with the schema shown in `gms_v95.json`; populate combat FNames.**
- [ ] **Step 3: Re-run audit (substitute `v87` everywhere in Task 8 Step 3).**
- [ ] **Step 4: Triage (same as Task 8 Step 4).**
- [ ] **Step 5: Per-fix commits; bucket commit.**

Bucket commit:
```bash
git add docs/packets/ida-exports/gms_v87.json \
        docs/packets/audits/gms_v87/
git commit -m "audit(combat): GMS v87 cross-version pass (phase-3-v87)"
```

- [ ] **Step 6: Hard-cap check** (same as Task 8 Step 6).

---

### Task 10: JMS v185 cross-version pass

JMS v185 had a separate opcode space for login (task-027 finding) and divergent character widths (task-028 finding). Expect similar for combat — `MoveMonster` multi-target sub-struct + `MonsterStat*` enum drift are likely.

**Files:**
- Create (if not present): `docs/packets/ida-exports/gms_jms_185.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_jms_185_1.json`
- Possibly modify: `libs/atlas-packet/{monster,drop,reactor,pet}/**/*.go` per design §9.

- [ ] **Step 1: Confirm JMS v185 IDA loaded**.
- [ ] **Step 2: Populate `gms_jms_185.json` for combat FNames** (create if absent). If a FName has no JMS equivalent, record the JMS-side FName + address with `"region": "JMS"`. Do NOT reuse GMS FNames for unrelated JMS functions.
- [ ] **Step 3: Re-run audit**:

```
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_jms_185_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_jms_185.json \
  --output           docs/packets/audits/jms_v185
```

- [ ] **Step 4: Triage per design §9.1**:
- In scope: atlas-packet emits bytes the JMS client decodes wrong.
- Out of scope: JMS-specific feature the service doesn't wire through.
- In scope: width mismatch on a field both versions decode.
- Out of scope: JMS template opcode wrong when GMS is right → template-only fix.

- [ ] **Step 5: Per-fix commits + bucket commit**.

Bucket commit:
```bash
git add docs/packets/ida-exports/gms_jms_185.json \
        docs/packets/audits/jms_v185/
git commit -m "audit(combat): JMS v185 cross-version pass (phase-3-jms-185)"
```

- [ ] **Step 6: Hard-cap check** (same as Task 8 Step 6).

---

## Phase 4 — Closeout

### Task 11: `post-phase-b.md`, full verification, code review, PR

**Files:**
- Create: `docs/tasks/task-065-combat-domain-audit/post-phase-b.md`
- Modify: `docs/packets/audits/gms_v95/SUMMARY.md` (final tallies row)
- Modify: `docs/packets/ida-exports/_pending.md` (final combat-domain section)

- [ ] **Step 1: Write `post-phase-b.md`**

Mirror task-028's structure. Five sections. Use `git log --oneline 414d7c872..HEAD` (the merge-base) to enumerate the fix commits.

```markdown
# Task-065 Post-Phase-B — Combat-Domain Audit Closeout

## Final state

- Packets audited: 31 (24 clientbound + 7 serverbound) across monster (10), pet (14), drop (3), reactor (4).
- Cross-version passes: GMS v83, GMS v87, JMS v185 — combat FNames covered.
- Combat-domain verdicts (GMS v95): ✅ <n_pass> / ❌ <n_fail> / ⚠️ <n_warn> / 🔍 <n_review> / pending <n_pending>.
- Combined `SUMMARY.md` (login + character + combat, GMS v95): <total> packets — <breakdown>.
- IDA-export coverage: GMS v95 / GMS v83 / GMS v87 / JMS v185 — combat FNames populated.
- Total commits on branch: <n>.

## Real wire bugs fixed

| Packet | File | IDA citation | Fix one-liner | Versions affected |
|---|---|---|---|---|
(one row per fix commit — `git log --grep='fix(atlas-packet,monster\|fix(atlas-packet,drop\|fix(atlas-packet,reactor\|fix(atlas-packet,pet' --oneline`)

## Template opcode / enum fixes

| Template file | Old → New | IDA case-statement | Reason |
|---|---|---|---|

## Tooling improvements

(only if any landed; this task has NO planned analyzer/registry work — expected empty)

## Remaining work

| Area | What | Why deferred |
|---|---|---|
(rows from `_pending.md` combat-domain section, hard-cap stops, JMS divergences left for sibling tasks)
```

Fill actual numbers from `SUMMARY.md` and commit history.

- [ ] **Step 2: Full verification matrix**

```
go build ./...
go vet ./libs/atlas-packet/...
go vet ./tools/packet-audit/...
go test -race ./libs/atlas-packet/...
go test -race ./tools/packet-audit/...
```

All five must be clean.

- [ ] **Step 3: Guarded `docker build`**

```
git diff --name-only main..HEAD -- '**go.mod' '**Dockerfile'
```

If empty: skip docker build (expected).
If non-empty: for each affected service `<svc>`:
```
docker build -f services/<svc>/Dockerfile .
```
Expected: clean. If it fails on workspace replace lines, fix per CLAUDE.md Build & Verification §4 (update the four hand-edited COPY / use(...) / replace blocks in the Dockerfile).

- [ ] **Step 4: gitleaks scrub**

```
grep -r '/home/' docs/packets/audits/gms_v95/{monster,drop,reactor,pet}/ docs/packets/audits/gms_v83/ docs/packets/audits/gms_v87/ docs/packets/audits/jms_v185/ 2>/dev/null
```

Expected: no output. If any user-home path appears in a report, scrub it:
```
sed -i 's|/home/[^/]*/source/atlas-ms/atlas/||g' <file>
git commit -am "audit: scrub absolute user-home paths from combat-domain reports"
```

- [ ] **Step 5: Commit `post-phase-b.md`**

```bash
git add docs/tasks/task-065-combat-domain-audit/post-phase-b.md \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/_pending.md
git commit -m "docs(task-065): post-phase-b closeout"
```

- [ ] **Step 6: Code review**

Invoke `superpowers:requesting-code-review`. Expect it to dispatch:
- `plan-adherence-reviewer` — verifies every checkbox in this plan has commit evidence.
- `backend-guidelines-reviewer` — DOM-* Go audit on `libs/atlas-packet/` + `tools/packet-audit/` changes.

(No `frontend-guidelines-reviewer` — no TS/React files touched.)

Read `docs/tasks/task-065-combat-domain-audit/audit.md` when each reviewer finishes. Act on every BLOCKER / MAJOR finding before opening the PR. Re-run review after fix commits land.

- [ ] **Step 7: Open the PR**

Title: `task-065: combat-domain packet audit (v83/v87/v95/JMS185) — monster, pet, drop, reactor`

Use `superpowers:finishing-a-development-branch` to drive PR creation. Body: short summary + link to `post-phase-b.md` for the full bug ledger.

---

## Self-review notes

**1. Spec coverage**
- PRD §3 (goals): ✅ Phase 2 covers all 31 packets; Phase 3 covers cross-version; Phase 4 ships post-phase-b ledger.
- PRD §4.1 (coverage matrix): ✅ Tasks 4–7 produce SUMMARY rows per sub-domain.
- PRD §4.2 (IDA exports): ✅ Task 4–7 Step 3 + Phase 3 tasks populate v83/v87/v95/JMS.
- PRD §4.3 (wire bug fixes): ✅ Embedded per sub-domain Task with 4-variant tests.
- PRD §4.4 (template fixes): ✅ Same Tasks, per-fix commits.
- PRD §4.5 (TypeRegistry extensions): ✅ Phase 1 Tasks 1–3.
- PRD §4.6 (cross-version re-verification): ✅ Phase 3 Tasks 8–10.
- PRD §10 acceptance: covered by Phase 4 Task 11 (verification matrix, gitleaks, code review).
- Design §3 (monster spawn analyzer FP): ✅ Task 4 Step 4 documents the expected ❌ + `_pending.md` row.
- Design §4 (Movement sub-struct registration): ✅ Task 1 fixture pins existing registration + adds combat sub-structs.
- Design §5 (pet self vs foreign + sub-op deferrals): ✅ Task 5 Step 4 anticipates.
- Design §6 (missing test gaps): ✅ Task 4 Step 1 + Task 5 Step 2 fill before any encoder mutation.
- Design §7 (phasing): mirrored exactly.
- Design §8 (v28): out per design. Variants table includes v28 for tests; no IDA pass.
- Design §9 (JMS): policy carried into Task 10.
- Design §10 (hot-path testing): ✅ Conventions section + per-fix commits enforce.
- Design §11 (template opcodes vs sub-ops): ✅ Task 4 Step 4 + per-fix commits.
- Design §13 (out-of-scope): respected. Plan does not extend analyzer or modify service-layer business logic.
- Design §15 ("what plan should do next"): plan is 12 tasks (1 + 3 + 4 + 3 + 1) — within the 10–14 target.

**2. Placeholder scan**
- No "TBD" / "implement later" / "fill in details" anywhere.
- No "similar to Task N" — each task has its own commands and code blocks.
- Every fix-commit step shows the exact commit-message format.
- IDA FName names in Task 2's routing table are PREDICTED; the plan explicitly requires verification via MCP before commit. This is not a placeholder — it's a runtime input the plan can't pre-compute.
- Hex values in Task 4 Step 5's test template are `<...>` — flagged as IDA-derived per-fix inputs the plan can't pre-compute. Same convention task-028 used.

**3. Type consistency**
- `test.Variants` / `test.RoundTrip` / `test.CreateContext` referenced consistently (Tasks 4, 5).
- `candidatesFromFName` referenced consistently (Tasks 2, 3).
- `FlattenWithRegistry` / `AnalyzeFile` / `NewTypeRegistry` / `HasType` / `Calls` referenced consistently in Phase 1 fixtures.
- Sub-struct names: `MonsterModel`, `MonsterTemporaryStat`, `MultiTargetForBall`, `RandTimeForAreaAttack`, `Movement` — consistent everywhere.
- `_pending.md` path: `docs/packets/ida-exports/_pending.md` consistently (correction noted in `context.md`).

**4. Loop-internal early-return** — out of scope. Plan does not touch the analyzer beyond Phase 1 fixtures.

**5. No `reflect`, no `interface{}`, no benchmarks** — Conventions section enforces.

**6. Gitleaks** — Phase 4 Task 11 Step 4 is the mandatory scrub.

**7. Hard precondition** — Phase 0 Task 0 gates everything on task-028 being merged. If task-028 isn't on `origin/main` when Task 0 runs, the plan correctly stops and asks the user.
