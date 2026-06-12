# Plan Audit — Leaf Flat-Validation + Verbatim-Guard Dispatch (task-081)

**Plan Path:** docs/tasks/task-081-ida-export-reharvest/non-equality-dispatch-plan.md
**Audit Date:** 2026-06-10
**Branch:** task-081-ida-export-reharvest
**Base Branch:** main
**Review Range:** 8eba7578 (parent of design commit) → 958457f8 (HEAD)

## Executive Summary

All 7 plan tasks are faithfully implemented. The four offline code tasks (1–5) land exactly
as specified with TDD tests that exercise their targets; the two IDA-gated tasks (0, 6) were
honestly consolidated into the live-run results doc, which carries the coverage data. The live
E2E caught two integration bugs (zero-read leaf false-divergence; verbatim-selector false
`case<0>` extra-mode); both fixes exist, are correct, and carry regression tests. All gates are
green: `go build ./...`, `go vet ./...`, and `go test -race -count=1 ./...` pass clean in
`tools/packet-audit/`. Recommendation: READY_TO_MERGE.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 0 | Live characterization of 251 shared-address handlers (IDA-gated) | DONE (consolidated) | No separate `non-equality-characterization.md`; coverage data lives in `non-equality-dispatch-results.md:6-18` (before→after table). Deviation honestly stated in results doc:4. |
| 1 | `Selector.Guard` + verbatim clause matching | DONE | `extract.go:23` (field), `extract.go:79-90` (verbatim match before equality scan). Test `extract_test.go:119` asserts match, composed-clause inclusion, sibling-arm rejection. Commit 7b3765ae0. |
| 2 | Parser verbatim-guard emission for non-equality arms | DONE | `parse.go:79` (`reIfCond`), `parse.go:890` (`isSinglePredicate`), `parse.go:588-599` (verbatim arm branch between reIfEq and reElse). Fixture `testdata/ifelse_noneq.c`. Test `parse_test.go:455` asserts `v5 < 5`, `v5 & 0x10`, default token. Commit db72d1332. |
| 3 | `Fields.HasMultiwayDispatch` | DONE | `idasrc.go:90` (field), `parse.go:772` (`collectCaseLabels` → `(map,bool)`), set in `ParseDecompileFields` (`parse.go:744`) and `ResolveLive` (`live.go:136-138`). Fixtures `leaf_linear.c`, `multiway_if.c`. Test `parse_test.go:475`. Commit fb40cb8b9. |
| 4 | `enumerateArms` + verbatim selector proposals | DONE (minor deviation) | `infer.go:289` (`enumerateArms`); `InferDispatchJoint` uses it at `infer.go:113`. Test `infer_test.go:180` asserts verbatim `{Guard}` and equality `v5==9` assignment. Deviation: plan said also switch `InferDispatch` (singular) to `enumerateArms`; it still uses `enumerateCases` (`infer.go:19`). Harmless — both live callers (`cmd/resolve_dispatch.go:88`, `cmd/infer.go:93`) use `InferDispatchJoint`; `InferDispatch` is test-only. |
| 5 | Validate leaf flat-validation | DONE | `cmd/validate.go:123-146` — exact 4-arm switch from plan, including leaf branch (`:132-140`) and multiway-stays-unverifiable branch (`:141-143`). Tests `validate_test.go:219` (leaf→verified), `:233` (multiway→unverifiable). Commit 00ed8c354. |
| 6 | E2E re-validate on four IDBs + results (IDA-gated) | DONE | Results doc present with before→after table; 145 selectors persisted (commit fe1131b5f); results commit 958457f8. Per-version figures sum exactly to headline (verified 410, unverifiable 348, divergent 339, allowlisted 254). |

**Completion Rate:** 7/7 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Deviation Verification (all sound, all documented)

1. **Task 0 consolidated into results doc.** Confirmed: no `non-equality-characterization.md`
   exists; `non-equality-dispatch-results.md:6-18` carries the before/after coverage table
   (verified 352→410, unverifiable 434→348, of which 373 per-mode-not-extractable). The live
   E2E numbers serve as the coverage measurement as intended. Honestly stated at results doc:4.

2. **Tasks 1–5 use hand-crafted structural `.c` fixtures; real-fixture hardening deferred.**
   Confirmed — fixtures `ifelse_noneq.c`, `leaf_linear.c`, `multiway_if.c` are synthetic
   Hex-Rays-style. Consistent with the per-branch convention; deferral is noted in the plan
   self-review (plan:807-809).

3. **Bug fix (a) — leaf with zero live reads → unverifiable, not divergent.** Confirmed at
   `cmd/validate.go:133-137`. Commit 3dde25269. Regression test
   `TestValidate_LeafEmptyLiveIsUnverifiable` (`validate_test.go:249`) feeds an empty-body leaf
   decompile and asserts `unverifiable` — exercises the exact target.

4. **Bug fix (b) — verbatim `{Guard}` selectors (Case==0) polluting the equality bijection.**
   Confirmed at `cmd/validate.go:157-158`: binding collection skips both `Default` and `Guard`
   selectors. Commit 76ef6db90. Regression test
   `TestValidate_VerbatimSelectorNotBijectionBinding` (`validate_test.go:266`) with fixture
   `testdata/verbatim_nobij.json` (a `{guard:"mode < 9"}` selector alongside an equality
   `case:1` binding) asserts no false `#case<0>` extra-mode — exercises the exact target.

5. **145 dispatch selectors persisted, additive/lossless, with verbatim guards.** Confirmed:
   per-version dispatch-entry counts v83:30, v87:39, v95:49, jms:27 = 145 (matches results
   doc:46). Verbatim `{guard}` selectors present in every baseline (54 total: 12/20/13/9).
   Commit fe1131b5f shows 0 deletions in `docs/packets/ida-exports/` (additive, lossless;
   matches the "+789 / −0" claim).

6. **+7 newly-allowlisted missing-mode.** Confirmed: exactly 7 `fname` entries added across
   `docs/packets/audits/{gms_v83,gms_v87,jms_v185}/_unimplemented.json` in commit fe1131b5f
   (CLogin::OnViewAllCharResult ×2, CWvsContext::OnGuildResult ×3, CWvsContext::OnPartyResult
   ×2). jms dir is `jms_v185` per plan:773.

7. **Claimed verified 410 / unverifiable 348.** Confirmed internally consistent: per-version
   breakdown (results doc:14) sums to verified 410, unverifiable 348, divergent 339,
   allowlisted 254 — matching the headline table exactly.

## Minor Notes (non-blocking)

- **Task 3 multiway detection differs from the plan's suggested algorithm.** The plan suggested
  a per-discriminator arm counter (`chainArms[disc]++`). The implementation instead treats any
  `else`/`else if` as a 2nd-arm signal (`parse.go:818-824`) plus switch ≥2 cases /
  ≥1 case+default (`parse.go:876`). Functionally equivalent for the tested cases and **safe**:
  it errs toward `multiway=true` (which keeps an entry `unverifiable`, never a false
  divergence). An optional-field `if {...} else {...}` would be flagged multiway — conservative,
  not a correctness regression. All five `TestParseDecompileFields_HasMultiwayDispatch` cases
  pass.

- **`enumerateCases` retained as dead-for-live code** (`infer.go:336`). Used only by
  `InferDispatch` (test-only) and its own tests. `go vet`/build clean, so not flagged unused;
  acceptable per plan:589-590 ("keep `enumerateCases` … if vet flags it unused, delete it").

## Build & Test Results

| Component | Build | Vet | Tests (-race -count=1) | Notes |
|-----------|-------|-----|------------------------|-------|
| tools/packet-audit | PASS | PASS | PASS | All packages green; idasrc 1.23s, cmd 3.09s. Tool, not a service → no docker bake / redis guard required. |

The four offline task tests and the four validate/fix tests were re-run verbosely; all RUN→PASS,
none skipped.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None required. Optional (not blocking):
1. If `InferDispatch` (singular, test-only) is to track the design, switch it to `enumerateArms`
   for parity — currently a no-op divergence since no command calls it.
2. The standing real-fixture hardening (deferred across the per-branch and this lever) remains
   the largest open gap, but is explicitly out of scope here.
