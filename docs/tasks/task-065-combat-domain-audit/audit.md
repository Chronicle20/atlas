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
