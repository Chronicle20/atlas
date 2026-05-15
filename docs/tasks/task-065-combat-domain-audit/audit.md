# Plan Audit — task-065-combat-domain-audit

**Plan Path:** docs/tasks/task-065-combat-domain-audit/plan.md
**Audit Date:** 2026-05-15
**Branch:** task-065-combat-domain-audit
**Base Branch:** main
**HEAD:** 6483ac413dfc33b52da46cd5189338654a2ab486

## Executive Summary

This PR ships a scope-reduced monster-only Phase 2a plus required tooling extensions, with all deferrals explicitly documented in `post-phase-b.md`. Tasks 0, 1, 1.5 (added mid-session), 2, 3, and 4 are complete with commit evidence; Tasks 5–10 (pet/drop/reactor sub-domains and cross-version passes) are cleanly deferred to follow-up tasks per the documented mid-session scope decision. Task 11 (closeout) is scope-reduced to match. All verification commands pass (`go build`, `go vet`, `go test -race` clean on both `libs/atlas-packet/...` and `tools/packet-audit/...`). The 109 `case "` count, 9 monster SUMMARY rows, and clean gitleaks scrub all match expectations.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 0 | Rebase gate (task-028 baseline) | DONE | `post-phase-b.md` line 9 documents baseline; `blockTerminatesWithReturn` present in `tools/packet-audit/internal/atlaspacket/analyzer.go`; `EncodeForeign` present in `registry.go`; `grep -c 'case "' run.go` = 109 (≥70); `docs/packets/audits/gms_v95/CharacterSpawn.md` present. Rebase was onto local `task-028-character-domain-audit` (acknowledged in scope deviation note). |
| 1 | Registry coverage fixture for combat sub-structs | DONE | `tools/packet-audit/internal/atlaspacket/registry_test.go:109` (`TestRegistryRegistersCombatSubStructs`) and `:133` (`TestRegistryStillRegistersMovementAfterCombatExtension`). Commit `2ae7cf590`. Tests pass under `-race`. |
| 1.5 | NEW — Sub-domain disambiguation tooling | DONE | Added not in original plan but documented in `post-phase-b.md` "Tooling improvements" §1. `candidate.pkg` field at `tools/packet-audit/cmd/run.go:133`; `locateAtlasFile(root, name, pkg, dir)` at `:608`; `qualifiedWriterName(pkg, name)` at `:141`. Test file `tools/packet-audit/cmd/disambiguation_test.go` (81 lines). Commit `eab8e64d8`. Required because sub-domain struct names collide (e.g. 4 `Spawn` types). |
| 2 | candidatesFromFName routing for 31 combat FNames | DONE (with deviation) | `grep -c 'case "' tools/packet-audit/cmd/run.go` = 109 (78 baseline + 31 combat). Commit `f38916d81`. Plan's predicted FNames were wrong — CSV-verified FNames used instead (e.g. `CMobPool::OnMobEnterField`, `CPet::OnMove`). Deviation documented in `post-phase-b.md` "Scope deviation from plan.md (FNames)" section. |
| 3 | Analyzer descent into combat sub-structs | DONE (with deviation) | `tools/packet-audit/internal/diff/combat_flatten_test.go:22` (`TestFlattenMonsterSpawnRunsToCompletion`) and `:51` (`TestFlattenMonsterStatSetCycleSafe`). Located in `internal/diff/` not `internal/atlaspacket/` (path deviation, acceptable). MonsterSpawn fixture uses weak "runs to completion" assertion instead of "expands KindRecurse" — registry struct-name collision is acknowledged as a known limitation in `post-phase-b.md` "Analyzer false positives surfaced" + `_pending.md`. Commit `57fb768f8`. |
| 4 | Phase 2a — monster sub-domain audit | DONE (9 of 10 packets) | 9 reports: `MonsterSpawn`, `MonsterControl`, `MonsterDestroy`, `MonsterDamage`, `MonsterHealth`, `MonsterMovement`, `MonsterMovementAck`, `MonsterStatSet`, `MonsterStatReset` — all present under `docs/packets/audits/gms_v95/Monster*.md` and visible in `SUMMARY.md`. Pre-flight test sweep `TestMonsterMovementRoundTrip` + `TestMonsterMovementRoundTripWithSkill` in `libs/atlas-packet/monster/clientbound/movement_test.go`. `MonsterMovementHandle` (serverbound) deferred — `CMob::GenerateMovePath` is 4 KB+ encode-side. Verdicts: ✅3 / ❌5 / 🔍1. `_pending.md` has the combat-domain monster section. Commits `544e4f44e`, `bf42c5dfd`. |
| 5 | Phase 2b — pet sub-domain | DEFERRED | Explicitly documented in `post-phase-b.md` "Out-of-scope cleanly deferred" §1. Routing entries committed; IDA exports unpopulated. |
| 6 | Phase 2c — drop sub-domain | DEFERRED | Explicitly documented in `post-phase-b.md` "Out-of-scope cleanly deferred" §2. |
| 7 | Phase 2d — reactor sub-domain | DEFERRED | Explicitly documented in `post-phase-b.md` "Out-of-scope cleanly deferred" §3. |
| 8 | Phase 3 — GMS v83 cross-version | DEFERRED | Explicitly documented in `post-phase-b.md` "Out-of-scope cleanly deferred" §4. |
| 9 | Phase 3 — GMS v87 cross-version | DEFERRED | Same as Task 8. |
| 10 | Phase 3 — JMS v185 cross-version | DEFERRED | Same as Task 8. |
| 11 | Phase 4 — closeout (scope-reduced) | DONE | `post-phase-b.md` exists, written as monster-only closeout. Verification matrix run + documented in `post-phase-b.md` "Verification matrix run" section. gitleaks scrub clean (`grep -r '/home/' docs/packets/audits/gms_v95/Monster*.md` returns no output). Commit `6483ac413`. |

**Completion Rate:** 6/12 implemented + 1 added (Task 1.5) + 6 deferred = 12 of 12 accounted for.
**Skipped without approval:** 0
**Partial implementations:** 1 (Task 4 is 9/10 packets; serverbound `MonsterMovementHandle` deliberately deferred and documented).

## Skipped / Deferred Tasks

All deferrals are explicitly accounted for in `post-phase-b.md`:

- **Task 4 (partial)** — `MonsterMovementHandle` serverbound packet deferred. `post-phase-b.md` "Out-of-scope cleanly deferred" §5 explains: `CMob::GenerateMovePath` IDA function is 4 KB+ encode-side; deferred pending decision on how to model Encode→Decode equivalence for `Send*` sources. Impact: monster audit is 9/10 instead of 10/10; the missing packet is serverbound (lower-risk).
- **Tasks 5, 6, 7 (Phase 2b–2d)** — Pet (14), Drop (3), Reactor (4) sub-domains deferred. Routing entries committed but IDA exports unpopulated. Impact: 21 of 31 planned packets not yet audited; tooling infrastructure is in place so follow-ups reuse it.
- **Tasks 8, 9, 10 (Phase 3)** — Cross-version passes (v83, v87, JMS v185) deferred. Impact: cross-version verification still pending; pre-task-027/028 cross-version reports for non-combat domains are unaffected.
- **Task 11 (Phase 4)** — Scope-reduced. The closeout document is monster-only; full ledger across all sub-domains will be produced as deferred phases complete.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| `tools/packet-audit/...` | PASS | PASS | All sub-packages pass under `-race`. |
| `libs/atlas-packet/...` | PASS | PASS | All sub-packages pass under `-race`. |
| `go vet ./libs/atlas-packet/... ./tools/packet-audit/...` | PASS | n/a | Clean. |

Additional verifications:
- `grep -c 'case "' tools/packet-audit/cmd/run.go` = **109** (matches plan expectation of 78+31).
- `grep -E '\[Monster' docs/packets/audits/gms_v95/SUMMARY.md \| wc -l` = **9** (matches scope-reduced expectation).
- `grep -r '/home/' docs/packets/audits/gms_v95/Monster*.md` returns **no output** (gitleaks clean).
- No `go.mod` or `Dockerfile` files were touched (verified via `git diff --name-only main..HEAD`), so `docker build` step correctly skipped per CLAUDE.md §4.

## Overall Assessment

- **Plan Adherence:** MOSTLY_COMPLETE (scope-reduced per documented mid-session decision)
- **Recommendation:** READY_TO_MERGE

Every plan task is either completed with commit evidence or explicitly deferred to a follow-up in `post-phase-b.md`. No silent skips. Mid-session scope reduction is well-documented with a clear rationale (the plan's predicted FNames were wrong against v95 IDA; per-packet decompile work is genuinely multi-session; user explicitly chose "option B — monster-only"). The added Task 1.5 (sub-domain disambiguation tooling) was a necessary tooling extension and is documented in `post-phase-b.md` "Tooling improvements". Plan deviations from `--output` flag (plan said `docs/packets/audits/gms_v95`, executor used `docs/packets/audits` since the tool appends `gms_v95/` itself) are minor and produce identical output paths.

## Action Items

None blocking for this PR. For the follow-up task that picks up Phase 2b/2c/2d/3:

1. Populate `docs/packets/ida-exports/gms_v95.json` for pet, drop, and reactor FNames (routing entries already committed).
2. Resolve the registry qualified-name limitation called out in `post-phase-b.md` "Audit-tool follow-ups recommended" §1 — needed to fully expand sub-struct fields in MonsterSpawn / MonsterMovement / MonsterStat* reports.
3. Address `MonsterMovementHandle` serverbound packet once the Encode→Decode equivalence model is decided (`post-phase-b.md` §4).
4. The two real wire bugs identified but not fixed (`MonsterDestroy` swallow optional and `MonsterControl` wire-shape divergence) need fix commits + hex baseline tests in a follow-up.

---

## Backend Guidelines Audit

- **Scope:** Go changes in task-065 above the task-028 baseline (`2ae7cf590^..HEAD`).
- **Files in scope (5):**
  - `tools/packet-audit/cmd/run.go` (modified)
  - `tools/packet-audit/cmd/disambiguation_test.go` (new)
  - `tools/packet-audit/internal/atlaspacket/registry_test.go` (modified — `TestRegistryRegistersCombatSubStructs`, `TestRegistryStillRegistersMovementAfterCombatExtension`)
  - `tools/packet-audit/internal/diff/combat_flatten_test.go` (new)
  - `libs/atlas-packet/monster/clientbound/movement_test.go` (new)
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-05-15
- **Build:** PASS
- **Tests:** all `go test -race` targets PASS, all suites OK
- **Overall:** PASS

### Build & Test Results (this section)

```
go vet ./libs/atlas-packet/...            -> clean (no output)
go vet ./tools/packet-audit/...           -> clean (no output)
go test -race ./tools/packet-audit/...    -> ok (cmd 1.742s, atlaspacket 2.692s, csv 1.024s, diff 1.532s, idasrc 1.012s, report 1.012s, template 1.014s)
go test -race ./libs/atlas-packet/monster/clientbound/... -> ok 1.020s
```

### Scope Classification

Task-065 changed only audit tooling (`tools/packet-audit/`) and added one test file under a shared lib (`libs/atlas-packet/monster/clientbound/movement_test.go`). **No `services/atlas-*` Go code changed.** Per task description, the DOM-* service-layer checklist (processor/handler/REST/Kafka/multi-tenancy/configurations/Dockerfile/k8s) does not apply. SEC-* and SUB-* do not apply (no new entrypoints, no input parsing, no subscriptions/streams). Service Scaffolding does not apply (no new service).

The relevant gates for this scope are: build/vet/test cleanliness, test-helper pattern, no `interface{}`/`reflect` introduction, and DOM-21 shared-constants reuse for any new types/constants.

### Applicable Checks

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Build/vet/test gate (Phase 1) | `go build`, `go vet`, `go test -race` clean | PASS | All four commands above produced zero errors / failures. |
| DOM-20 | Table-driven tests | PASS | `tools/packet-audit/cmd/disambiguation_test.go:13-31` (`TestQualifiedWriterName` table), `:42-65` (`TestLocateAtlasFileDisambiguatesByPkg` table); `tools/packet-audit/internal/atlaspacket/registry_test.go:94-107` (movement element table), `:116-131` (combat sub-struct table), `:140-153` (regression table). |
| DOM-21 | No duplication of atlas-constants types | PASS | `git diff 2ae7cf590^..HEAD -- '*.go' \| grep -E '^\+\s*(type \|const )'` returns zero new top-level type/const declarations. The only `type ` occurrences in `+` lines are inside string literals (test fixture strings like `"type X struct"`). No numeric-literal classifications were introduced. |
| Test Helper Pattern (project rule) | No `*_testhelpers.go` files | PASS | `find .worktrees/task-065-combat-domain-audit -name '*_testhelpers.go'` returns zero hits. `libs/atlas-packet/monster/clientbound/movement_test.go:13` uses `model.MultiTargetForBall{}` / `model.RandTimeForAreaAttack{}` zero-values plus the existing `NewMonsterMovement(...)` constructor — no test-only constructor introduced. |
| No `interface{}` introduced | grep across 5 task files | PASS | `grep -nE 'interface\{\}'` across all 5 task-065 Go files returns zero matches. |
| No `reflect.` introduced | grep across 5 task files | PASS | `grep -nE 'reflect\.'` across all 5 task-065 Go files returns zero matches. |
| Variant coverage | `movement_test.go` exercises all `test.Variants` | PASS | `libs/atlas-packet/monster/clientbound/movement_test.go:11`, `:21` loop over `test.Variants` (5 region/version variants — GMS v28/v83/v87/v95, JMS v185) for both no-skill and with-skill paths. |
| Package boundary (tooling vs service) | `tools/packet-audit/cmd/run.go` does not import `services/...` | PASS | `tools/packet-audit/cmd/run.go:1-30` imports only `tools/packet-audit/internal/...` siblings plus stdlib; no `services/atlas-*` reach-through. |
| Package boundary (test in shared lib) | `movement_test.go` lives in same package, no cross-domain imports | PASS | `libs/atlas-packet/monster/clientbound/movement_test.go:3-8` imports only `testing`, the sibling `model` package, and the shared `test` helper — all within `libs/atlas-packet/`. |
| Comments-on-why | New routing entries explain the IDA->atlas mapping | PASS | `tools/packet-audit/cmd/run.go:445-540` — each `case` block has a leading comment naming the CSV opcode and the canonical IDA dispatcher (e.g. `:482` "CSV: REACTOR_HIT — atlas Hit", `:519` "CSV: SPAWN_PET (serverbound) — atlas Spawn"). The `candidate.pkg` field is documented with intent (`:128-132`) and `qualifiedWriterName` explains the empty-pkg fallback (`:137-140`). |
| Known FP documented in test fixture | `combat_flatten_test.go` documents design §3 expectation | PASS | `tools/packet-audit/internal/diff/combat_flatten_test.go:12-21` explicitly calls out the registry-name-collision FP and pins safe completion (not full expansion) as the assertion. |

### Checks Not Applicable (justified)

- **DOM-01..DOM-19** (`builder.go`, `ToEntity`, `Make`, REST `Transform`, processor/handler/administrator layering, `RegisterInputHandler`, tenant callbacks, lazy providers, `os.Getenv`, JSON:API interfaces, request flatness): no service domain package was added or modified. No `model.go`, `processor.go`, `resource.go`, `administrator.go`, or `provider.go` exist in scope.
- **DOM-22** (Dockerfile 4-block sync): no `services/<svc>/go.mod` or `services/<svc>/Dockerfile` touched in this task's commits.
- **DOM-23** (Kafka topic naming via configmap): no `COMMAND_TOPIC_*` / `EVENT_TOPIC_*` constants added; no `deploy/k8s/env-configmap.yaml` change.
- **SUB-01..SUB-04** (action-event sub-packages): no `resource.go` added.
- **EXT-01..EXT-04** (external HTTP client): no new `requests.GetRequest[T]` / `requests.PostRequest[T]` call site.
- **SCAFFOLD-01..SCAFFOLD-08** (new service scaffolding): no new `services/atlas-<service>/` directory; no new `Writer`/`Handler` constant registered in `services/atlas-channel/atlas.com/channel/main.go` in scope.
- **SEC-01..SEC-04** (auth/security): not an auth-related service; no new input parsing or token handling.

### Findings

#### Blocking
None.

#### Non-Blocking
None.

### Summary

Task-065 is in-scope a tooling-only audit change plus a single shared-lib test file. All applicable backend guidelines pass with specific file:line evidence:

- The new IDA->atlas routing entries in `tools/packet-audit/cmd/run.go:445-540` introduce the `candidate.pkg` disambiguation field and the `qualifiedWriterName` helper; both are documented inline and covered by `disambiguation_test.go`'s tables.
- The combat-sub-struct registry assertions (`registry_test.go:109-131`) and movement-regression assertion (`:133-154`) protect task-028's movement work from accidental drop by the new combat extension.
- `combat_flatten_test.go:12-21` documents the design §3 FP as a deliberate, pinned behavior rather than a silent regression.
- The `monster/clientbound/movement_test.go` 5-variant round-trip baseline uses the existing `test.Variants` and constructor, not a `*_testhelpers.go` file, satisfying the project's Test Helper Pattern.

No `services/atlas-*` audit dimensions were exercised by this change, and the service-layer DOM-* / SUB-* / EXT-* / SCAFFOLD-* / SEC-* checks are documented above as Not Applicable.

## Backend Guidelines Re-review (Phases 2+3 complete)

**Re-review Date:** 2026-05-15
**Prior Review Baseline:** `09b198006`
**Current HEAD:** `a1309f2bc`

### Scope

Re-audit of Go changes landed since the prior review baseline, covering Phase 2b/c/d sub-domain audits (pet, drop, reactor) and Phase 3 cross-version passes (v83, v87, JMS v185).

### Go Diff Since Prior Review

`git diff --name-only 09b198006..HEAD -- '*.go'` reports two files (NOT empty — initial assumption was wrong; both were touched in commits after the prior review baseline):

1. `libs/atlas-packet/pet/clientbound/movement_test.go` — new file, 18 lines (commit `13c4e4f6a`).
2. `tools/packet-audit/cmd/run.go` — single switch-case + comment update (commit `36b719ffe`).

### Gate Commands

| Command | Result |
|---------|--------|
| `go vet ./libs/atlas-packet/...` | PASS (no output) |
| `go vet ./tools/packet-audit/...` | PASS (no output) |
| `go test -race ./libs/atlas-packet/...` | PASS (all packages `ok`, `pet/clientbound` cached pass) |
| `go test -race ./tools/packet-audit/...` | PASS (all packages `ok`) |

### Per-File Findings

#### `libs/atlas-packet/pet/clientbound/movement_test.go`

| Check | Status | Evidence |
|-------|--------|----------|
| Test Helper Pattern (no `*_testhelpers.go`) | PASS | File is a standard `_test.go`; uses shared `libs/atlas-packet/test` package (`test.Variants`, `test.CreateContext`, `test.RoundTrip`) instead of test-only constructors. `movement_test.go:7,13-15`. |
| Table-driven style (DOM-20 analogue for libs) | PASS | `movement_test.go:11-17` iterates `test.Variants` with `t.Run(v.Name, ...)`, matching the 5-variant baseline pattern used across `libs/atlas-packet/*/clientbound/*_test.go`. |
| Production code touched | N/A | Test-only addition; no `Encode`/`Decode` changes. |
| Service-layer DOM-* / SUB-* / EXT-* / SCAFFOLD-* / SEC-* | N/A | `libs/atlas-packet/` is a shared library, not a service domain. No `model.go`, `processor.go`, `resource.go`, `administrator.go`, Kafka topic, or HTTP client involved. |
| Atlas-constants duplication (DOM-21) | PASS | No new types, aliases, or numeric constants declared. |

#### `tools/packet-audit/cmd/run.go`

| Check | Status | Evidence |
|-------|--------|----------|
| Single switch-case rewrite | PASS | `run.go:485-491` re-keys the FName entry from `CUser::OnPetPacket` to `CUserRemote::OnPetActivated`; same `candidate{name: "Activated", pkg: "pet", dir: csvpkg.DirClientbound}` value. |
| Justification documented | PASS | `run.go:484-490` inline comment cites the v95 IDA dispatcher (`OnPetPacket@0x8e02a0`) and leaf (`CUserRemote::OnPetActivated@0x9547d0`), satisfying the project's "verify against source" rule for protocol mappings. |
| Service-layer DOM-* / SUB-* / EXT-* / SCAFFOLD-* / SEC-* | N/A | `tools/packet-audit/` is a CLI auditor, not a microservice. No domain code, no Kafka, no HTTP, no Dockerfile. |
| Atlas-constants duplication (DOM-21) | PASS | No new types or numeric constants. |

### Findings

#### Blocking
None.

#### Non-Blocking
None.

### Re-review Conclusion

The two Go files touched since the prior review baseline are (a) a shared-lib round-trip test that conforms to the existing 5-variant pattern and the project's Test Helper Pattern, and (b) a single CLI switch-case rewrite with an inline IDA-cited justification. Neither file is service-layer code, so DOM-* / SUB-* / EXT-* / SCAFFOLD-* / SEC-* checklists do not apply (consistent with the original review's Not-Applicable rationale). All four gate commands are clean.

**Overall: PASS.**

---

## Re-review (Phases 2+3 complete)

**Re-review Date:** 2026-05-15
**Branch:** task-065-combat-domain-audit
**Base Branch:** main
**HEAD:** a1309f2bc
**Prior monster-only audit:** see top half of this file (frozen at HEAD `6483ac413`).

### Executive Summary

The earlier monster-only audit's MOSTLY_COMPLETE / READY_TO_MERGE verdict is now upgraded after Phases 2b/2c/2d (pet, drop, reactor) and Phase 3 (v83, v87, JMS v185) landed in-branch. Every plan task except a single, well-documented deferral (monster serverbound `MonsterMovementHandle` ← 4 KB+ `CMob::GenerateMovePath` encode-side function) is implemented with commit evidence. `post-phase-b.md` has been rewritten to reflect the full scope (no longer "monster-only"). All five verification matrix commands are clean. The branch is **READY_TO_MERGE**.

### Task Completion (updated)

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 0 | Rebase gate | DONE | (unchanged from prior review) |
| 1 | Registry coverage fixture | DONE | Commit `2ae7cf590`. (unchanged) |
| 1.5 | Sub-domain disambiguation tooling | DONE | Commit `eab8e64d8`. (unchanged) |
| 2 | candidatesFromFName routing for 31 combat FNames | DONE | `grep -c 'case "' tools/packet-audit/cmd/run.go` = 109. Commit `f38916d81`. (unchanged) |
| 3 | Analyzer descent into combat sub-structs | DONE | `tools/packet-audit/internal/diff/combat_flatten_test.go`. Commit `57fb768f8`. (unchanged) |
| 4 | Phase 2a — monster sub-domain (9 of 10 cb) | DONE (partial — see Task 4.x) | 9 reports under `docs/packets/audits/gms_v95/Monster*.md` (excluding the deferred Handle). Commits `544e4f44e`, `bf42c5dfd`. |
| 4.x | `MonsterMovementHandle` serverbound | DEFERRED | Documented in `post-phase-b.md` "Out-of-scope cleanly deferred" §1. `CMob::GenerateMovePath` is 4 KB+ encode-side and requires Encode→Decode equivalence modeling not yet in the audit pipeline. |
| 5 | Phase 2b — pet sub-domain (14 packets) | DONE | 14 reports under `docs/packets/audits/gms_v95/Pet*.md` (`PetActivated`, `PetCashFoodResult`, `PetChat`, `PetChatRequest`, `PetCommand`, `PetCommandResponse`, `PetDropPickUp`, `PetExcludeItem`, `PetExcludeResponse`, `PetFood`, `PetItemUse`, `PetMovement`, `PetMovementRequest`, `PetSpawn`). Pre-flight test sweep `libs/atlas-packet/pet/clientbound/movement_test.go` (commit `13c4e4f6a`). Bucket audit commit `36b719ffe`. Pet-domain row in `_pending.md:367` documents 4 ✅ / 10 ❌. |
| 6 | Phase 2c — drop sub-domain (3 packets) | DONE | 3 reports `DropSpawn.md`, `DropDestroy.md`, `DropPickUp.md`. Bucket commit `0d617d1b6`. `_pending.md:394` documents 1 ✅ / 2 ❌. |
| 7 | Phase 2d — reactor sub-domain (4 packets) | DONE | 4 reports `ReactorSpawn.md`, `ReactorHit.md`, `ReactorDestroy.md`, `ReactorHitRequest.md`. Bucket commit `db7da6540`. `_pending.md:404` documents 3 ✅ / 1 ❌. |
| 8 | Phase 3 — GMS v83 cross-version | DONE | 30 combat reports under `docs/packets/audits/gms_v83/` (9 Monster + 13 Pet + 4 Drop + 4 Reactor). `_pending.md:415` documents 11 ✅ / 19 ❌. Bucket commit `f345c30b5`. |
| 9 | Phase 3 — GMS v87 cross-version | DONE | 30 combat reports under `docs/packets/audits/gms_v87/`. `_pending.md:437` documents 12 ✅ / 18 ❌. Bucket commit `bb730d66d`. |
| 10 | Phase 3 — JMS v185 cross-version | DONE | 30 combat reports under `docs/packets/audits/jms_v185/`. `_pending.md:449` documents 11 ✅ / 1 🔍 / 18 ❌. Bucket commit `8d18e7ffe`. |
| 11 | Phase 4 — closeout | DONE | `post-phase-b.md` rewritten for full scope (commit `a1309f2bc`); replaces the earlier monster-only snapshot (`6483ac413`). Verification matrix + gitleaks scrub both clean (see below). |

**Completion Rate:** 11 of 12 original tasks DONE + 1 added (Task 1.5 — sub-domain disambiguation tooling) + 1 sub-deferral (MonsterMovementHandle). Effective: **30 of 31 planned packets audited**.

**Skipped without approval:** 0.
**Partial implementations:** 1 (Task 4 — `MonsterMovementHandle` serverbound packet deferred and documented).

### Deferral Detail (single remaining)

- **`MonsterMovementHandle` (serverbound)** — `CMob::GenerateMovePath` is the v95 IDA encode-side function and is 4 KB+. The audit pipeline currently has no Encode→Decode equivalence model that would let the diff engine bind atlas's `Decode×N` reads against the IDA `Encode×N` source. Listed as `post-phase-b.md` "Out-of-scope cleanly deferred" §1 and in "Audit-tool follow-ups recommended" §4. Impact: 30/31 packets audited; the unaudited packet is serverbound (lower-risk than clientbound).

### Verification Matrix

| Command | Result |
|---------|--------|
| `go build ./libs/atlas-packet/... ./tools/packet-audit/...` | PASS (no output) |
| `go test -race ./tools/packet-audit/...` | PASS (all 7 packages `ok`) |
| `go test -race ./libs/atlas-packet/...` | PASS (all packages `ok`) |
| `grep -cE '^\| \[(Monster\|Pet\|Drop\|Reactor)' docs/packets/audits/gms_v95/SUMMARY.md` | **31** (30 combat + the pre-existing `DropMeso` character-domain row that lexically matches the regex) |
| `ls docs/packets/audits/gms_v83/Monster*.md \| wc -l` | **9** (matches expectation) |
| `ls docs/packets/audits/gms_v87/Monster*.md \| wc -l` | **9** |
| `ls docs/packets/audits/jms_v185/Monster*.md \| wc -l` | **9** |
| `grep -rE '/home/[a-z]' docs/packets/audits/{gms_v83,gms_v87,gms_v95,jms_v185}/{Monster,Pet,Drop,Reactor}*.md` | (no output — gitleaks clean) |

Per-version combat-domain coverage (only `{Monster,Pet,Drop,Reactor}*.md` reports, `DropMeso` excluded):

| Version | Monster | Pet | Drop | Reactor | Total |
|---------|---------|-----|------|---------|-------|
| GMS v83 | 9 | 13 | 4 | 4 | 30 |
| GMS v87 | 9 | 13 | 4 | 4 | 30 |
| GMS v95 | 9 + 1 (`MonsterDamageFriendly`, pre-existing from task-028) | 14 | 3 | 4 | 31 (30 combat-domain task-065 reports + 1 pre-existing) |
| JMS v185 | 9 | 13 | 4 | 4 | 30 |

The per-version `Pet`/`Drop`/`Reactor` count differs by ±1 across versions because some FNames (e.g. `SendActivatePetRequest` for `PetSpawn` serverbound in v83) are absent in older binaries — documented in `post-phase-b.md` "Per-version cross-cutting notes" §GMS v83 and `_pending.md:467`.

### `docker build` Check

`git diff --name-only main..HEAD -- '**go.mod' '**Dockerfile'` returns empty. No service `go.mod` or `Dockerfile` touched; `docker build` correctly skipped per CLAUDE.md §4.

### Comparison to Prior (monster-only) Audit

The prior monster-only audit (top half of this file) marked Tasks 5–10 as `DEFERRED` and stated the closeout was "scope-reduced". This re-review confirms:

- All six previously-deferred tasks (5, 6, 7, 8, 9, 10) are now DONE with committed audit reports, IDA-export entries, and `_pending.md` rows.
- The "scope-reduced" closeout has been replaced. The current `post-phase-b.md` (commit `a1309f2bc`) lists full 30-packet × 4-version coverage with verdict roll-ups per version, real wire bugs identified (3 deferred fixes documented), and audit-tool follow-ups.
- The added Task 1.5 (sub-domain disambiguation tooling, commit `eab8e64d8`) is still acknowledged as a documented mid-execution scope extension; no new tooling was needed for Phases 2b/2c/2d/3.

### Documentation Inconsistencies (minor, non-blocking)

1. `post-phase-b.md:18` states "Total commits on branch above task-028 baseline: 13" but the embedded block at `:88-104` lists 15 commits, and `:106` says "18 total commits ahead of task-028" (the discrepancy is because :106 includes the three earlier docs commits — spec, design, plan). Recommend reconciling the line :18 number to 15 (or 18 if docs commits are counted).
2. `post-phase-b.md:5` claims "30 packets … drop (2 cb + 1 sb)" but `docs/packets/audits/gms_v95/Drop*.md` (excluding `DropMeso`) contains 3 reports: `DropSpawn` (cb), `DropDestroy` (cb), `DropPickUp` (sb). The "(2 cb + 1 sb)" formulation is correct as totals (2+1=3) — no fix required.
3. `_pending.md` doesn't include a row that names `MonsterMovementHandle` explicitly under "Still pending — combat domain (monster)". The deferral is named in `post-phase-b.md` "Out-of-scope cleanly deferred" §1 (commit `a1309f2bc`). Recommend adding a one-row entry to `_pending.md` for symmetry with the per-packet rows.

None of these block the PR.

### Overall Assessment (re-review)

- **Plan Adherence:** **MOSTLY_COMPLETE** — every plan task has commit evidence except the single documented `MonsterMovementHandle` deferral.
- **Recommendation:** **READY_TO_MERGE**.

### Action Items

None blocking. For the follow-up that picks up `MonsterMovementHandle` and the analyzer-tool improvements:

1. Add Encode→Decode equivalence support to the audit pipeline so `Send*` IDA sources can be diffed against atlas `Decode×N` handlers (covers the `MonsterMovementHandle` deferral plus the other Phase-2 ❌ verdicts traced to "DecodeBuf placeholder" — `post-phase-b.md` "Analyzer false positives surfaced" §3).
2. Implement registry qualified type names so `r.types["monster/clientbound.Spawn"]` ≠ `r.types["pet/clientbound.Spawn"]` (covers the registry struct-name collision FP — `post-phase-b.md` "Audit-tool follow-ups recommended" §1).
3. Add `_pending.md` row for `MonsterMovementHandle` (cosmetic — see Documentation Inconsistency #3).
4. Reconcile commit-count statements at `post-phase-b.md:18` and `:106` (cosmetic — see Documentation Inconsistency #1).
5. Three real wire bugs surfaced and documented but **not fixed** in this PR (`MonsterDestroy` swallow optional, `MonsterControl` wire-shape divergence, `DropDestroy` explode-delay): each needs a fix commit + 4-variant hex test + constructor update in `services/atlas-channel` callers, per `post-phase-b.md` "Real wire bugs identified (none fixed in this PR)".
