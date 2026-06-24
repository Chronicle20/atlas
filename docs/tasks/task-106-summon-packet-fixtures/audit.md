# Plan Audit — task-106-summon-packet-fixtures (plan-adherence-reviewer)

**Plan Path:** docs/tasks/task-106-summon-packet-fixtures/plan.md
**Audit Date:** 2026-06-23
**Branch:** task-106-summon-packet-fixtures
**Base Branch:** main (5d9c42ff3)
**Range audited:** 5d9c42ff3..aac895b88

## Executive Summary

This verification campaign was implemented faithfully and completely. All 6 summon clientbound packets are ✅ across all 5 versions (v83/v84/v87/v95/jms) in `STATUS.md` with zero 🟡/❌. The tier-0 cell (v95 SummonMove) carries a marker and no evidence; all 24 tier-1 cells carry both a `packet-audit:verify` marker and a pinned evidence YAML with a `verifies:` field. `matrix --check` exits 0, all 41 summon clientbound subtests pass, and `go vet`/`go build` are clean in `libs/atlas-packet`. No codec (`*.go` non-test) files were changed (the §R3 wire-fix path correctly did not trigger). Both grading-relevant deviations (the `SummonSpawn` tiers.yaml addition and the v83/v87 export re-points) are present, IDA-evidenced, and principled — not drift. Verdict: READY_TO_MERGE.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | v95 SummonMove tier-0 (marker only, NO evidence, ✅) | DONE | Marker `move_test.go:162 ida=0x759830` matches v95 SummonMove report Address 0x759830. No `gms_v95/summon.clientbound.SummonMove.yaml` exists (confirmed). Commit 352541e57. STATUS row ✅. |
| 2 | v83 × 6 packets (tier-1: marker + pinned evidence) | DONE | 6 markers in `*_test.go`, 6 evidence YAMLs under `evidence/gms_v83/`, each with `verifies:`. SummonSpawn re-pointed to active 0x95adec (commit 3254ee096). Commits 66eeb99b7, 045036515, 6c689ebce, 6c9d76e91, 38485aef7, 3254ee096. |
| 3 | v84 × 6 packets (tier-1) | DONE | 6 markers + 6 evidence YAMLs under `evidence/gms_v84/`. Commits fba0d3a25, cafe45a0f, 69c4cbeff, d0d671c92, c1072e462, 62ce46eb1. |
| 4 | v87 × 6 packets (tier-1) | DONE | 6 markers + 6 evidence YAMLs under `evidence/gms_v87/`. SummonSpawn export `calls`/`note` corrected oid-vs-cid (commit dd46b94fe). Commits dd46b94fe, eaa9f2c63, e5635ea6f, 2a329ec31, 91d89c715, c595cb930. |
| 5 | jms_v185 × 6 packets (tier-1) | DONE | 6 markers + 6 evidence YAMLs under `evidence/jms_v185/`. Commits e72464b5c, 74c82c8f2, ab6335af0, 158b01930, 17db7a3aa, aac895b88. No jms read function reported undecompilable (no escalation needed). |
| 6 | Final acceptance gate | DONE | `matrix --check` exit 0; six summon clientbound rows ✅ × 5 versions, zero 🟡/❌; `go test ./summon/clientbound/` 41 PASS / 0 FAIL; `go vet`/`go build` clean in `libs/atlas-packet`. |

**Completion Rate:** 6/6 plan tasks (100%). The 32 unchecked `- [ ]` boxes in plan.md are step-level checkboxes left un-toggled in the doc; every step has landed evidence in the commits above, so this is a cosmetic doc-hygiene gap, not skipped work.
**Skipped without approval:** 0
**Partial implementations:** 0

## Per-claim verification (matches the review brief)

1. **Task 1 (v95 SummonMove tier-0):** PASS. Marker present, no evidence record, cell ✅.
2. **Tasks 2-5 (24 tier-1 cells):** PASS. Each cell has a marker in its `*_test.go` AND a pinned `evidence/<version>/summon.clientbound.Summon<Packet>.yaml` with `verifies:`. Cross-check: every `verifies:` reference resolves to a real `func Test...` in the named test file (0 missing).
3. **Marker/report/export address sync:** PASS. `matrix --check` exits 0 (no orphan/dangling/stale/drift). Spot-checked: v83 SummonSpawn marker `0x95adec` = report Address `0x95adec` = re-pointed export address; v87 SummonSpawn marker `0x9b3749` = report Address `0x9b3749`.
4. **The two grading-relevant deviations:** PASS, both principled.
   - (a) `summon/clientbound/SummonSpawn` added to `docs/packets/evidence/tiers.yaml` `packets:` list, making ALL Spawn cells uniformly tier-1; this required a v95 SummonSpawn evidence record (commit 3254ee096) so the already-✅ v95 cell does not regress. This is why the repo now has 30 summon markers (not the plan's stated 25 — the plan miscounted the 5 v95 markers for Spawn/Remove/Attack/Damage/Skill that already lived on main) and 25 summon-clientbound evidence files (24 planned + the required v95 SummonSpawn). Internally consistent, not scope creep.
   - (b) v83 SummonSpawn export re-pointed from inactive `0x938f61` to active `0x95ADEC`; v87 SummonSpawn export `calls`/`note` corrected so the int after the upstream cid is labeled `oid` (was mislabeled `skillId`). Both carry full IDA decompile-line evidence in the export `note`. No address drift: `matrix --check` exit 0 and regenerating the matrix produced no diff against the committed STATUS.md/status.json.
5. **Acceptance gate (Task 6):** PASS. `matrix --check` exit 0; six summon clientbound rows ✅ × 5 versions, zero 🟡/❌; `go test ./libs/atlas-packet/summon/clientbound/` 41 PASS / 0 FAIL; `go vet ./...` and `go build ./...` clean in `libs/atlas-packet`.
6. **No silent gaps:** PASS. `git diff 5d9c42ff3..aac895b88 -- 'libs/atlas-packet/summon/clientbound/*.go' ':!*_test.go'` is EMPTY (no codec changes — §R3 correctly untriggered). No `// TODO`/`FIXME`/`t.Skip` in the changed test files.

## Skipped / Deferred Tasks

None. Nothing was skipped, deferred, or stubbed.

## Build & Test Results

| Module | Build | Tests | Notes |
|--------|-------|-------|-------|
| libs/atlas-packet | PASS | PASS | `go build ./...` exit 0; `go vet ./...` exit 0; `go test ./summon/clientbound/ -count=1` ok, 41/41 subtests pass. |
| tools/packet-audit | PASS | n/a | `matrix` regenerates with no diff; `matrix --check` exit 0. |

`services/atlas-channel` was NOT rebuilt/baked, correctly — no codec change landed, so the §R3 follow-on (build/vet/test/bake of atlas-channel) did not apply. Test-only fixture changes do not trigger a bake.

## Known non-blocking item (not a task-106 defect)

`GOWORK=off tools/redis-key-guard.sh` exits 1. Pre-existing repo-wide: the tool emits an info line per scanned service package (57 listed) and the non-zero exit is identical on `main`. This branch changed zero service/redis code (diffstat is entirely `docs/` + `libs/atlas-packet/.../*_test.go` + evidence/audit/export files), so the exit is not attributable to task-106. Not a blocker.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None required for correctness. Optional housekeeping only:

1. (Cosmetic) The 32 step-level `- [ ]` checkboxes in `plan.md` were never toggled to `- [x]` despite all steps landing. Optionally check them off so the doc reflects completed state. Does not affect mergeability.
2. (Doc accuracy) Task 6 Step 2 of the plan asserts "expect 25 markers / 24 evidence files." The correct counts are 30 markers and 25 summon-clientbound evidence files (5 v95 markers pre-existed on main; the SummonSpawn tiers.yaml promotion legitimately required a v95 SummonSpawn evidence record). The implementation is right; only the plan's pre-stated expectation was off.

---

# Backend Guidelines Audit — task-106-summon-packet-fixtures (backend-guidelines-reviewer)

- **Scope:** Go test-fixture changes only — `libs/atlas-packet/summon/clientbound/{spawn,remove,move,attack,damage,skill}_test.go`
- **Diff audited:** `5d9c42ff3..aac895b88` (branch tip `aac895b88`)
- **Date:** 2026-06-23
- **Build (go vet ./summon/...):** PASS (exit 0)
- **Tests (go test -race ./summon/clientbound/):** PASS (all green, isolated detached worktree at `aac895b88`)
- **Overall (scoped to test-fixture changes):** PASS

## Verification method note

The Bash tool's working directory resets to the repo root (`main` checkout) between
calls, so the task-106 commits are not in that tree. To verify the actual branch tip,
I added a throwaway detached `git worktree` at `aac895b88`, ran `go vet ./summon/...`
(exit 0) and `go test -race ./summon/clientbound/ -count=1` (`ok`, all tests pass),
then removed the temporary worktree. No files in the task worktree were mutated.

## Checklist Results (test-fixture scope)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Build/vet | `go vet ./summon/...` clean | PASS | exit 0 at `aac895b88` |
| Test/race | `go test -race ./summon/clientbound/` clean | PASS | `ok ...summon/clientbound 1.019s` at `aac895b88` |
| Builder/helper pattern | Uses shared `test.CreateContext` / `test.Encode` + production constructors, not test-only constructors | PASS | spawn_test.go:1-9 imports `atlas-packet/test`; `test.CreateContext` (test/context.go:34), `test.Encode` (test/roundtrip.go:15); constructors are production `NewSummon*` (spawn.go:28, remove.go:20, move.go:32, damage.go:38, attack.go:21/39, skill.go:32) |
| No `*_testhelpers.go` added | No test-only constructor helper files introduced | PASS | `git diff --name-only` shows only the six `*_test.go` files |
| DOM-21 type reuse | No new domain types/aliases/numeric constants duplicating atlas-constants | PASS (N/A) | Fixtures assert raw `[]byte`; reuse pre-existing `summonSpawnV83Body`/`summonAttackV83Body`/`summonDamageV83Body` vars (present at base 5d9c42ff3); no new `type`/`const` added |
| No production smells in tests | No `panic`, `t.Skip`, `TODO/FIXME`, dead/commented-out code | PASS | grep of all six files at `aac895b88` returns no matches |
| Naming convention | Funcs follow `TestSummon<Packet>Bytes<VER>` | PASS | All 24 new funcs match (V83/V84/V87/JMS185 × 6 packets, minus spawn-V95 pre-existing); see func list below |
| No renamed/removed existing funcs | Existing tests untouched (tests reference internals) | PASS | `comm` of base vs branch func sets shows zero removals; `TestSummon*`, `*RoundTrip`, `*Bytes`, `*BytesV95` all retained |
| Byte traceability ("no inventing") | Fixture bytes carry per-byte tracing comments to decompile field/address | PASS | Inline `want` slices annotate every field with `// <field> (Decode_@0x...)` e.g. move_test.go, remove_test.go, skill_test.go; GMS-identical cells reuse the already-annotated `*V83Body` vars |

### New test functions added (all named per convention, all passing)

SummonAttack: BytesV83, BytesV84, BytesV87, BytesJMS185 (BytesV95 pre-existing)
SummonDamage: BytesV83, BytesV84, BytesJMS185 (BytesV87, BytesV95 pre-existing)
SummonMove:   BytesV83, BytesV84, BytesV87, BytesJMS185 (BytesV95 pre-existing)
SummonRemove: BytesV83, BytesV84, BytesV87, BytesJMS185 (BytesV95 pre-existing)
SummonSkill:  BytesV83, BytesV84, BytesV87, BytesJMS185 (BytesV95 pre-existing)
SummonSpawn:  BytesV83, BytesV84, BytesV87 (BytesV95, BytesJMS185 pre-existing)

### N/A checklist items (do not apply to pure test-fixture changes)

DOM-01..DOM-20, DOM-22..DOM-24 (builder/entity/processor/REST/Kafka/multi-tenancy/
Dockerfile/topic/producer-stub), all SUB-*, EXT-*, SCAFFOLD-*, SEC-* — marked N/A:
the change is six packet byte-fixture test files in a shared lib, with no
production code, no new types, no service/processor/REST/Kafka surface. DOM-24
(Kafka producer stub) does not apply: these are pure codec `Encode` byte-assertion
tests with no emit path (no `AndEmit`/`message.Emit`/`producer.Produce`, no
consumer/saga entry points).

## Strengths

- Every fixture builds the packet through the production constructor + codec
  `Encode` and asserts exact wire bytes — the correct verification altitude.
- GMS-identical version cells (V84/V87 etc.) reuse the single pre-existing
  `summon<Packet>V83Body` var and assert equality to it, so there is one source of
  truth per packet and a divergence in any version would fail loudly. No copied
  magic-byte blocks.
- Inline `want` slices (move/remove/skill) annotate every byte group with the
  decompile `Decode_@0x...` address it traces to — satisfies the project's
  "no inventing / cite the source" rule for fixture bytes.
- `// packet-audit:verify packet=... version=... ida=0x...` markers present on each
  new fixture, tying it to the coverage-matrix campaign.
- Test-only setup goes through the shared `atlas-packet/test` helpers
  (`CreateContext`/`Encode`); no bespoke test-only constructors and no
  `*_testhelpers.go` file — compliant with the project Builder/test-helper rule.

## Issues

### Critical
- None.

### Important
- None.

### Minor
- None blocking. Observation only: several doc comments candidly flag the
  attractive "OnHit/OnSkill mangled-symbol naming swap" in the IDA export and that
  the body (not the symbol) is authoritative. This is correct disclosure, not a
  defect — the fixtures pin the codec output, which is the contract under test.

## Assessment (scoped to the test-fixture changes)

PASS. The change is limited to additive packet byte-fixture tests across six summon
clientbound packets for v83/v84/v87/jms185 (v95 was pre-existing). `go vet` and
`go test -race` are clean at the branch tip. No production code, no new
types/constants (DOM-21 clean), no forbidden test-helper files, no panics/skips/dead
code, correct naming, and full per-byte traceability to decompile addresses. The
vast majority of DOM/SUB/EXT/SCAFFOLD/SEC checks are N/A to pure test-fixture work
and are marked as such rather than forced into findings.
