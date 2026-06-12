# Plan Audit — task-081-ida-export-reharvest

**Plan Paths:** prd.md (§10), plan.md, plan-validation.md (the pivot), non-equality-dispatch-plan.md, divergent-offbyone-plan.md, per-branch-verification-plan.md
**Audit Date:** 2026-06-11
**Branch:** task-081-ida-export-reharvest
**Base (NET diff):** 3ab5d1dc5 → c12f30985

## Executive Summary

Task-081 did NOT execute the original plan.md as written. After Phases 0/1/1.5 (build a real
automated exporter), an empirical v83 measurement showed that *replacing* the hand-authored
baselines with exporter output **regresses** the audit (26 ✅→❌ vs 3 ❌→✅), so the team
explicitly **abandoned plan.md Phases 2–7** and pivoted to a *validation* approach
(`design-validation-pivot.md`), then ran three further sub-plans (per-branch verification,
non-equality dispatch, off-by-one divergent). This is a documented, well-justified scope pivot —
not silent skipping. The four sub-plans' code is essentially all present, builds clean, vets
clean, and passes `go test -race` in both changed modules (`tools/packet-audit`,
`libs/atlas-packet`). Two genuine gaps remain: (1) the single Atlas wire fix
(`pet/serverbound/chat.go` gate `>=87`→`>=95`) shipped with **no WireShape byte-test oracle** —
only a symmetric RoundTrip test that cannot catch the off-by-one the fix corrects (violates the
plan's load-bearing FR-3.3 discipline); (2) the **FR-6 ledger/guide phase (V7) was not done** —
`STARTING_A_NEW_VERSION_PASS.md`, both `_pending.md`, and `TOTAL.md` were last touched by
task-080, and the per-version validation reports (`docs/packets/validation/*.md`) were never
committed.

## Plan / FR Completion

| FR / Phase | Status | Evidence / Notes |
|---|---|---|
| Phase 0 rebase + verdict snapshot (FR-7.1) | DONE | `verdict-snapshot-080.md` present (33KB, "DO NOT EDIT after Phase 0"); commit c0e8e2a0d |
| Phase 1 exporter (FR-1.1–1.6, 2.3) | DONE | `Unresolved` prim `idasrc.go:21`; `ParseDecompile` descent+loop+switch `parse.go:406`; `Harvest` BFS cycle/depth guard `harvest.go:28`; MCP-HTTP client `mcphttp.go:39` (lookup_funcs/select_instance new API); `VerdictUnresolved` 🚫 `diff/diff.go:17`. Full TDD fixtures committed. |
| Phase 1.5 real-decompile hardening | DONE | commits 631b37db8, cdbffa5db, 1cca19a32; real v83 fixtures `testdata/real_onfriendresult_v83.c`, `real_sub_a40028_v83.c` |
| plan.md Phase 2 full re-export (FR-2.1/2.2) | NOT_APPLICABLE | Explicitly abandoned by `design-validation-pivot.md` §1 (empirical regression measurement). Baselines stay hand-authored + live-VALIDATED, not replaced. |
| plan.md Phase 3 verdict-delta triage (FR-3.2/7.x) | PARTIAL / superseded | `verdict-delta-081.md` not created (Phase 3 artifact, abandoned). Triage replaced by the validation/divergent sub-plans' per-shape disposition. |
| FR-3.3 fix surfaced bugs with per-version byte tests | PARTIAL | Real wire fix landed: `pet/serverbound/chat.go` gate `MajorAtLeast(87)`→`(95)` symmetric Encode/Decode (commit 15de0b8c2). BUT `chat_test.go` NOT modified in range; only a symmetric `TestChatRoundTrip` exists — no exact-offset WireShape test asserting the v95 gate. The plan calls this exact oracle out as load-bearing. |
| FR-4 opaque decomposition (plan Phase 5) | PARTIAL / superseded | No `opaque-set-081.md`; opaque/mask packets (`OnCharacterInfo`, `OnAvatarModified`, CMovePath, SecondaryStat) modeled via Delegate splicing / opaque-buffer equivalence (commits 6e0b1c46a, 3891c23a1) and left "honest divergent" by the validation design. No dedicated per-type byte-test exception ledger as FR-4.3 specified. |
| FR-5 template completeness (plan Phase 6) | DONE (mostly in base) | JMS NPC-shop/interaction op-byte routing landed in task-080 (B5.1, commits in base: 2fae37786, 38b31491b). task-081's bijection (missing-mode/extra-mode + per-version `_unimplemented.json` allowlists, all 4 present) is the structural "what's unrouted" signal. No `template-gaps-081.md`. |
| FR-6.1 re-curate both `_pending.md` | SKIPPED | Neither `docs/packets/ida-exports/_pending.md` nor `docs/packets/audits/gms_v95/_pending.md` was touched by task-081 (last commits d0ae6e902/9d3c99097 = task-080). No "live-verified"/"validation" annotation. |
| FR-6.2 update TOTAL.md + new-version-pass guide | SKIPPED | `TOTAL.md` and `STARTING_A_NEW_VERSION_PASS.md` last touched by task-080 (d4af9b42f). Guide has 0 mentions of `validate`/`dispatch selector`/`resolve-dispatch`. The new exporter/validate workflow is undocumented in the canonical guide. |
| Validation pivot V1–V4 (offline core) | DONE | `ExtractShape`/`Selector` `extract.go:30`; `Dispatch` schema + `ResolveShape`; `InferDispatch`/`InferDispatchJoint` `infer.go:112`; `validate` cmd wired `root.go:29`. |
| Validation pivot V5/V6 (bootstrap + validate live) | DONE (data) | dispatch selectors persisted in all 4 baselines (v83:39, v87:54, v95:72, jms:40). Live results in sub-plan results docs. |
| Validation pivot V7 (ledger + docs) | SKIPPED | `docs/packets/validation/` dir does not exist; guide/_pending/TOTAL not updated (see FR-6). |
| Per-branch verification plan (Tasks 1–8) | DONE | `Selector.Default` + `<default>` token; if/else guard emission `parse.go`; `CaseLabels`/`ParseDecompileFields`; `WriteDispatch` (lossless surgical writer, 774e33e9); `resolve-dispatch` cmd `root.go:41`; `Bijection`/`ModeBinding` `bijection.go`; `LoadAllowlist` `allowlist.go:25`; results `per-branch-verification-results.md` + `missing-mode-triage.md`. |
| Non-equality dispatch plan (Tasks 1–6) | DONE | `Selector.Guard` verbatim match `extract.go:23`; verbatim-arm emission + `isSinglePredicate`; `Fields.HasMultiwayDispatch`; `enumerateArms` `infer.go:289`; leaf flat-validation in `validate.go`; results `non-equality-dispatch-results.md` (FAIL-1 review correction 410→407 documented). |
| Off-by-one divergent plan (Tasks 1–6) | DONE | `classifyDiff`/`shapeDiff` + `diff-shape` cmd `root.go:44`; `PrependCall` `baseline_write.go:92` + review hardening (9c5b9327f); 54-entry prepend landed in baselines (v83:10/v87:8/v95:29/jms:9); results `divergent-offbyone-results.md`. NOTE: results doc claims v95:27 but baseline shows 29 leading-byte comments — minor count drift, not material. `divergent-findings.md` absent but results state "no genuine encoder bugs isolated" so nothing to record. |
| Code review before PR (acceptance criterion) | DONE | `audit-backend.md`, `audit.md`, `audit-backend-noneq.md`, `audit-noneq.md`, `audit-backend-offbyone.md`, `audit-offbyone.md` present per sub-plan. |
| CLAUDE.md verify gates | DONE (tool scope) | See Build & Test below. `tools/packet-audit` is a dev tool, no service target → no docker bake required. `libs/atlas-packet` consumers would bake; not run here (read-only audit). |

## Skipped / Partial — impact

1. **FR-3.3 byte-test oracle missing for the pet-chat wire fix (PARTIAL — most material).**
   `libs/atlas-packet/pet/serverbound/chat.go` changed the `updateTime` gate from
   `MajorAtLeast(87)` to `MajorAtLeast(95)` in both Encode and Decode. `chat_test.go` was not
   touched in the task-081 range; the only test is `TestChatRoundTrip`, which encode→decodes with
   the SAME context and asserts field round-trip — it passes identically whether the gate is 87 or
   95, so it cannot prove the v87/v95 boundary the fix turns on. The plan (Phase 4 Step 2 /
   validation V6) explicitly requires a per-version exact-offset WireShape test as the oracle
   precisely because "a round-trip misses a wrong-but-symmetric bug." Impact: the headline wire
   correction ships unguarded against regression; a future edit could silently revert it.

2. **FR-6 ledger + guide (SKIPPED).** `STARTING_A_NEW_VERSION_PASS.md`, both `_pending.md`, and
   `gms_v95/TOTAL.md` still reflect task-080. The new `export`/`validate`/`resolve-dispatch`
   workflow, the `dispatch`/`Guard` selector schema, the by-address decompile model, and the
   `unresolved`-over-guess invariant are undocumented in the canonical onboarding guide, and the
   `#`-mode entries are not annotated as live-verified. Impact: a future maintainer onboarding a
   5th version won't find the new tooling from the guide; the registries overstate "hand-trusted."

3. **Validation reports not committed under `docs/packets/validation/` (SKIPPED).** The V6
   deliverable per-version reports live only in `/tmp` and the task-folder `*-results.md`. Impact:
   the live-verification evidence is not a durable repo artifact at the path the plan specified.

4. **FR-4 opaque per-type exception ledger (PARTIAL).** Opaque/mask packets are tolerated via
   buffer-equivalence and left honest-divergent, but no `opaque-set-081.md` enumerates each type's
   disposition with a byte-test-backed verified exception as FR-4.3 demanded. Lower impact (the
   validation design absorbs most), but the PRD's "no type in an unexamined skipped state" bar is
   not provably met per-type.

## Build & Test Results

| Module | Build | Vet | Tests (`-race`) | Notes |
|---|---|---|---|---|
| tools/packet-audit | PASS | PASS | PASS | all packages green; `validate` subcommand runs + enforces required flags |
| libs/atlas-packet | PASS | PASS | PASS | full tree incl. pet/serverbound green |

(Worktree clean except `go.work.sum` build artifact. `docker buildx bake` not run — read-only
audit; `tools/packet-audit` has no service target, and no `libs/atlas-packet` consumer `go.mod`
was touched beyond the lib itself.)

## Overall Assessment

- **Plan Adherence:** MOSTLY_COMPLETE — the engineering core (exporter + four sub-plans) is fully
  implemented, tested, and committed; the documented pivot away from plan.md Phases 2–7 is sound
  and evidence-backed. Two genuine gaps: the missing WireShape oracle on the one wire fix, and the
  entire FR-6/V7 documentation+ledger phase.
- **Recommendation:** NEEDS_FIXES (small, bounded) — add the pet-chat WireShape byte test and
  complete the FR-6/V7 docs/ledger before declaring the PRD acceptance criteria met.

## Action Items

1. Add a `TestChatRequestWireShape` (or equivalent) byte-level test over `pt.Variants` asserting
   the exact `updateTime` presence/offset gated at GMS major ≥ 95, per FR-3.3 — the symmetric
   RoundTrip test is not an oracle for this fix. (`libs/atlas-packet/pet/serverbound/chat_test.go`)
2. Update `docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md` with the `export` (new-version
   bootstrap) vs `validate` (verify-existing) workflow, the `dispatch`/`Guard` selector schema +
   `resolve-dispatch` annotation procedure, the by-address decompile model, and the
   unresolved-over-guess invariant (FR-6.2 / V7 Step 1).
3. Re-curate both `_pending.md` registries to annotate the `#`-mode entries as live-verified and
   cite the validation results, and refresh `gms_v95/TOTAL.md` (FR-6.1/6.2 / V7 Step 2).
4. Commit the per-version validation reports under `docs/packets/validation/<version>.md` (or
   document that the task-folder `*-results.md` files are the canonical record and adjust the plan
   reference) (V6 deliverable).
5. (Optional/low) Reconcile the off-by-one results-doc v95 count (27) with the baseline (29) and
   either create the empty/`no-findings` `divergent-findings.md` or note in the results doc that it
   was intentionally omitted.
