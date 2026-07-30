# Plan Audit — task-187-version-aware-id-semantics

**Plan Path:** docs/tasks/task-187-version-aware-id-semantics/plan.md
**Audit Date:** 2026-07-30
**Branch:** task-187-version-aware-id-semantics
**Base Branch:** main

## Executive Summary

All 16 plan tasks and both scope additions (atlas-monsters SuperGmHide detection, atlas-data skill/reader.go SuperGM classification) have concrete file:line evidence of implementation in the current tree — 24 commits on the branch, 168 files changed, +44324/-1551. The core PRD acceptance criterion is proven end-to-end: `TestGoldenAnchors` in `libs/atlas-constants/constants/golden_test.go` passes live (verified by direct `go test -v` run), covering v48 5101004→SuperGmHide, v72 5101004→BrawlerCorkscrewBlow, v61 Pirate present-in-semantics/absent-from-availability, v48 job500→Gm, plus a Big Bang merge anchor. No task is SKIPPED; nothing is PARTIAL. Build/test verification beyond the targeted checks below is explicitly out of scope for this audit per instructions.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Multi-boundary audit (grounding deliverable) | DONE | `audit/divergences.csv` has 65 data rows (header + 65, verified via `wc -l`), `audit/availability.csv` has 99 data rows; `audit/README.md`, `v048-v062.md`, `bigbang-v092-v095.md`, `cygnus-anchor.md`, `v048-gm-supergm-skill-ranges.md` all present with cited evidence (e.g. `divergences.csv:2` cites live atlas-data query with tenant id + skill names). `libs/atlas-constants/gen/audit_validate_test.go` is the ported structural validator. |
| 2 | Identity namespace + generator scaffold | DONE | `libs/atlas-constants/gen/go.mod`, `gen/main.go`, `gen/identities.go`, `gen/identities.yaml` (2618 lines), `skill/identities_gen.go`, `job/identities_gen.go`, `skill/identity.go`, `job/identity.go` all present and building. |
| 3 | Per-version WZ id-set snapshots | DONE | `libs/atlas-constants/gen/wzsnapshot/<r>_<maj>_<min>.json` × 11 files present (gms 12/48/61/72/79/83/84/87/92/95 + jms_185_1) + `snapshots.go`/`snapshots_test.go`; `PROVENANCE.md` records drain provenance. `go test ./wzsnapshot/` passes. |
| 4 | Per-version semantics + `Set.Resolve/Wire` | DONE | `gen/semantics/<r>_<maj>_<min>.yaml` × 11 files, `gen/semantics.go`, `skill/version_<r>_<maj>_<min>_gen.go` × 11, `job/version_*_gen.go` × 11. `Resolve`/`Wire` implemented in `skill/identity.go` and `job/identity.go`. |
| 5 | Availability manifests + `Available` | DONE | `gen/availability/<r>_<maj>_<min>.yaml` × 11 (implicit inside `availability.go`/generated sets), `gen/availability.go` (303 lines), `Available`/`AvailableIdentities`/`Name` on `Set` confirmed in `for.go`/generated files; `TestGoldenAnchors/v61_Pirate_present_but_unavailable` passes live. |
| 6 | `constants.For` selector + baseline fallback | DONE | `libs/atlas-constants/constants/for.go` implements `For(region, major, minor)` with case-insensitive region match, `sync.Map`-deduped once-per-key warning log, and baseline (GMS 83.1) fallback — read in full, matches plan Task 6 spec exactly. `constants/registry_gen.go` present. `for_test.go` covers it. |
| 7 | Identity predicates + golden anchors + drift CI | DONE | `skill/identity_test.go`, `job/identity_test.go` port predicates (`IsKeyDownSkillIdentity`, `IsAIdentity`, etc. — confirmed via grep in common.go/model.go usage). `constants/golden_test.go` (116 lines) is the golden-anchor acceptance test — **ran it live, all 7 subtests PASS** (see Core Acceptance Criterion section). `gen/drift_test.go` present. CI wiring: `.github/workflows/pr-validation.yml:190` job `atlas-constants-drift-guard` runs the regen+diff check; included in the final gate at line 713. |
| 8 | atlas-channel identity-keyed registry + dispatch | DONE | `skill/handler/registry.go` — `var registry = map[skill2.Identity]Handler{}` (L32), `Register(id skill2.Identity, h Handler)` (L38), `Lookup(id skill2.Identity)` (L43). `common.go:94` — `castId, castIdOk := set.Skill.Resolve(skill2.Id(info.SkillId()))`, dispatch at L181-186 via `Lookup(castId)`. `registry_test.go` present (55 lines). |
| 9 | Registrations + hide/healdispel/keydown/buff-cancel | DONE | All 7+ registration call sites confirmed via grep: `hide/hide.go:38 Register(skill2.SuperGmHide, Apply)`, `healdispel/healdispel.go:30`, `heal/heal.go:42`, `mprecovery/mprecovery.go:19`, `mysticdoor/mysticdoor.go:26`, `timeleap/timeleap.go:33`, `resurrection/resurrection.go:27-29` (3 identities). `character_skill_prepare.go`, `character_buff_cancel.go`, `character/buff/hidden.go`, `kafka/consumer/buff/consumer.go` all modified per `git diff --stat`. New file `skill/handler/version_resolve.go` + `version_resolve_test.go` added as a shared resolve helper (not explicitly named in plan but consistent with its intent). |
| 10 | Mastery cascade + remaining sites | DONE | `socket/writer/character_attack_common.go` modified (36 lines changed) plus new `character_attack_common_mastery_test.go` (121 lines); `character_attack_combo.go`, `character_skill_use.go`, `npc_shop.go`, `mob_select.go`, `mount.go` all modified per diff stat, matching the plan's file list. |
| 11 | atlas-character job-keyed logic migration | DONE | `character/processor.go` (336 lines changed) — `resolveHPMPGainParams` at ~L1639-1661 explicitly reorders GM/SuperGM check before Pirate (comment at L1645-1653 documents the exact v48-Pirate-shadowing bug being fixed and cites the ordering fix). New `character/hp_mp_gain_test.go` (68 lines) is the v48 GM HP regression test from plan Step 1. `point_reset.go` (122 lines changed) + test. `skill/model.go` modified (10 lines) for `FromSkillId`→`FromSkillIdentity`. `character/model.go` (9 lines changed) — citation comments added per plan Step 5. |
| 12 | Scoped CI guard | DONE | `tools/skill-job-id-guard.sh` (231 lines) exists; ran it live: `skill-job-id-guard: clean (14 divergent const(s) checked)`, exit 0. Wired into CI at `.github/workflows/pr-validation.yml:337` (`skill-job-id-guard` job) and included in the final required-checks list at line 713. CLAUDE.md verification list updated — item 10 in the worktree's CLAUDE.md explicitly documents the guard, its divergent-id source (`audit/divergences.csv`), and the resolver exception. |
| 13 | atlas-data availability resources | DONE | `services/atlas-data/atlas.com/data/jobavailability/{resource,rest,processor,resource_test}.go` and `skillavailability/{resource,rest,processor}.go` present; `main.go` modified (+4 lines) to register routes. `jobavailability/resource_test.go` (94 lines) covers the GMS 48.1 → GM / GMS 72.1 → Pirate distinction per plan Step 1. |
| 14 | atlas-ui availability-gated selectors | DONE | `src/services/api/availability.service.ts`, `src/lib/hooks/api/useJobAvailability.ts`, `useSkillAvailability.ts` created; `usePresetJobOptions.ts` rewritten to source from `useJobAvailability` instead of `useJobs`/`JOB_LIST` WZ-presence gating (read in full — matches plan intent: names + gating both now version-correct, graceful "unknown ≠ empty" pending fallback preserved). `JobCombobox.tsx` modified (7 lines) for the name source. Tests present for both the hook and service. |
| 15 | v111 forward-compat review | DONE | `docs/tasks/task-187-version-aware-id-semantics/audit/v111-forward-compat.md` (342 lines) exists, explicitly headed "NOT SHIPPED", cites IDA `func_query` against a v111 IDB session and meymink anchors, documents a naming-coverage limitation distinguishing "confirmed absent" from "name-unconfirmable". No `gen/wzsnapshot/gms_111_1.json` or `versions.json` row was added, consistent with "not shipped." |
| 16 | Full verification sweep + finalize | DONE (partially out of scope for this audit) | Per this audit's explicit instructions, the full `docker buildx bake` / repo-root guard sweep is deferred to a separate verification pass. Targeted checks performed here (guard script, golden test, `go build`/`go test` on libs/atlas-constants) all pass — see Build & Test section. README.md updated (`libs/atlas-constants/README.md`, +5/-... lines) with `constants` and `Identity` rows, confirmed by reading the file. |
| Scope Addition 1 | atlas-monsters version-aware SuperGmHide detection | DONE | `character/buff/model.go` — `HasActiveGmHide(ctx, bs)` (L38-50) resolves `b.SourceId()` via `constants.For(t...).Skill.Resolve` and compares to `skill.SuperGmHide`, with an explicit doc comment (L29-37) explaining the v48 (5101004) vs v62+ (9101004) wire divergence this fixes. `kafka/consumer/buff/consumer.go` (+42/-... lines) and `character/hidden/task.go` also modified; new tests `model_test.go` (73 lines), `consumer_test.go` (63 lines). |
| Scope Addition 2 | atlas-data skill/reader.go version-aware SuperGM classification | DONE | `services/atlas-data/atlas.com/data/skill/reader.go` modified (36 lines changed) with a new 261-line `reader_test.go`. Commit `fcccc61e6 feat(atlas-data): version-aware SuperGM skill classification (task-187)`. |

**Completion Rate:** 16/16 tasks (100%) + 2/2 scope additions (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None found. Every plan task has direct file:line evidence in the current tree. Task 15 is intentionally "not shipped" as designed by the plan itself (a findings-only deliverable), not a skip.

## Core Acceptance Criterion — Verified End-to-End

Ran `go test ./constants/... -run TestGoldenAnchors -v` in `libs/atlas-constants` directly (not just grepped):

```
=== RUN   TestGoldenAnchors/v48_skill5101004_is_SuperGmHide         --- PASS
=== RUN   TestGoldenAnchors/v48_job500_is_Gm                        --- PASS
=== RUN   TestGoldenAnchors/v72_skill5101004_is_BrawlerCorkscrewBlow --- PASS
=== RUN   TestGoldenAnchors/v72_job500_is_Pirate                    --- PASS
=== RUN   TestGoldenAnchors/v61_Pirate_present_but_unavailable      --- PASS
=== RUN   TestGoldenAnchors/bigbang_WarriorHpBoost_absent_v92_present_v95 --- PASS
=== RUN   TestGoldenAnchors/v48_dispatch_intent_hide_not_keydown    --- PASS
```

- **v48 5101004 → SuperGmHide (not BrawlerCorkscrewBlow):** proven by `golden_test.go:19-25` and passing live.
- **v72 5101004 → BrawlerCorkscrewBlow (the opposite mapping):** proven by `golden_test.go:36-42` and passing live — confirms the resolution genuinely branches on version rather than hardcoding a v48 special case, since both directions are asserted against the *same* generated code path (`constants.For(...).Skill.Resolve`).
- **v61 Pirate present-in-semantics / absent-from-availability:** proven by `golden_test.go:55-63` (`Job.Wire(Pirate)` succeeds, `Job.Available(Pirate)` is false) and passing live.
- **v48 job 500 → GM (not Pirate):** proven by `golden_test.go:26-32` and passing live; additionally the consumer-side fix is verified in `services/atlas-character/atlas.com/character/character/processor.go` around L1639-1661, which explicitly documents (in an inline comment) the pre-fix bug — a v48 GM matching `job.GmId`/`job.SuperGmId` (900/910, version-blind constants) fell through to the Pirate branch because Pirate's raw-500 check ran first — and fixes it by resolving to `job.Identity` and ordering the GM/SuperGm `IsAIdentity` check before Pirate's.
- The atlas-channel dispatch path (`skill/handler/registry.go` + `common.go:94,181-186`) was read directly, not just grepped: the registry is keyed on `skill2.Identity`, and `UseSkill` resolves the incoming wire id via `set.Skill.Resolve` **before** calling `Lookup`, so a v48 Super GM pressing wire 5101004 dispatches to whatever handler is registered under `skill.SuperGmHide` (the Hide handler, registered at `hide/hide.go:38`), never the keydown-attack path that Corkscrew Blow uses on v72+.

**Verdict: all four criteria are proven, not merely asserted.**

## Build & Test Results

Out of scope per this audit's instructions (a separate verification pass covers the full `docker buildx bake` / repo-root guard sweep). Targeted checks performed as part of evidence-gathering:

| Check | Result | Notes |
|---|---|---|
| `libs/atlas-constants`: `go build ./...` | PASS | Clean build. |
| `libs/atlas-constants`: `go test ./...` | PASS | All packages pass (constants, field, item, job, map, merchant, monster, skill, summon; several have no test files, unrelated to this task). |
| `libs/atlas-constants/gen`: `go test ./...` | PASS | Generator + wzsnapshot tests pass. |
| `tools/skill-job-id-guard.sh` | PASS | `skill-job-id-guard: clean (14 divergent const(s) checked)`, exit 0. |
| `constants/golden_test.go` (`TestGoldenAnchors`) | PASS | All 7 subtests pass — see Core Acceptance Criterion. |
| Full `go test -race ./...` per changed service (atlas-channel, atlas-character, atlas-data) | NOT RUN (out of scope) | Defer to the separate build/test verification pass per audit instructions. |
| `docker buildx bake atlas-channel atlas-character atlas-data` | NOT RUN (out of scope) | Defer to the separate build/test verification pass per audit instructions. |
| atlas-ui `npm run build` / `vitest` | NOT RUN (out of scope) | Defer to the separate build/test verification pass per audit instructions. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE, pending the separate full build/test/docker-bake verification pass (out of scope here) coming back clean.

## Action Items

None required based on plan-adherence evidence. Recommend:
1. Run the full CLAUDE.md verification suite (`go test -race ./...`, `go vet ./...`, `go build ./...`, `docker buildx bake atlas-channel atlas-character atlas-data`, `tools/lint.sh --check`, `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, atlas-ui `npm run build` + `vitest`) as the separate verification pass this audit was told to skip, before merging.
2. No plan-adherence gaps to fix.
