# Plan Audit — Systematic Off-By-One Divergent Remediation

**Plan Path:** docs/tasks/task-081-ida-export-reharvest/divergent-offbyone-plan.md
**Audit Date:** 2026-06-10
**Branch:** task-081-ida-export-reharvest
**Review Range:** 3afb59460c2abade3c5f68b4bf838fda41a38e50 (BASE) → 95d8f886deab158d5798feac6fb1108e4b3cfa11 (HEAD)

## Executive Summary

All six plan tasks were faithfully implemented. The offline diagnostic (`diff-shape`
classifier + driver + subcommand wiring + determinism/read-only guards, Tasks 1–3) is complete
and tested; the IDA-gated characterization, remediation, and re-validation (Tasks 4–6) were
executed live against all four IDBs. The three named deviations are all sound and honestly
documented: (a) scope narrowed from ~109 leading off-by-one entries to the 54 clean
single-leading-byte serverbound entries; (b) remediation used a NEW surgical `PrependCall` in
`baseline_write.go` instead of a new dispatcher kind; (c) `ValidateShape`/`shapediff.go` was not
modified (option 3). Gates are green: `go build ./...`, `go vet ./...`, and `go test -race ./...`
all pass on `tools/packet-audit`. No claimed-but-not-implemented task and no test that fails to
exercise its target were found.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | diff-shape classifier (`classifyDiff`/`shapeDiff`) | DONE | `cmd/diff_shape.go:103-166` (`shapeDiff` struct, `classifyDiff`, `eqOps`, `min2`). Test `cmd/diff_shape_test.go:21-44` (`TestClassifyDiff`, 4 cases leading/trailing/interior/none) PASS. Commit c5544bed1. |
| 2 | diff-shape driver `diffShapeRun` + `cmd/root.go` wiring | DONE | `cmd/diff_shape.go:15-101` (`diffShapeOpts`, `diffShapeRun`, `opsLine`). Dispatch `cmd/root.go:44-46`; `runDiffShape` flag wrapper `cmd/root.go:226-268` (copies `runResolveDispatch` pattern). Fixture `cmd/testdata/diffshape_mini.json`. Test `TestDiffShapeRun_EmitsDivergentRows` PASS. Commit 896d00d8b. |
| 3 | Determinism / read-only guards | DONE | `cmd/diff_shape_test.go:66-83` (`TestDiffShape_DeterministicAndReadOnly`): byte-stability across two runs + baseline-unchanged assertion. PASS. Commit 9c7ed99e8. |
| 4 | Live characterization (IDA-gated) | DONE | `divergent-characterization.md` records the live diff-shape run (ports 13337–13340), the 109 leading / 59 interior / 46 trailing ±1 breakdown, the identical `hand == live[1:]` signature, and the remediation-category recommendation. Commit 0d2b91a9f. |
| 5 | Remediation blend (IDA-gated / code + data) | DONE (deviated, documented) | Mechanism chosen = baseline `calls` correction via NEW `PrependCall` (`internal/idasrc/baseline_write.go:80-177`), not a dispatcher kind. 54 baseline entries prepended a leading `Encode1` (v83:10 v87:8 v95:27 jms:9). Commits 0eaff2fce (PrependCall) + 491c7fd5c (54 edits). Step 3 (`divergent-findings.md`) N/A — no genuine encoder bugs isolated (results doc §Findings). |
| 6 | Re-validate + results (IDA-gated) | DONE | `divergent-offbyone-results.md`: before 407/338 → after 461/284, Δ +54/−54, per-version after-counts, 3-handler IDA spot-check recorded. Commit 95d8f886d. |

**Completion Rate:** 6/6 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Deviation Verification (per audit brief)

1. **Scope narrowing 109 → 54.** Documented and defensible. `divergent-characterization.md:15-23`
   names the 109 leading-`Decode1` entries; `divergent-offbyone-results.md:23-31` narrows to the
   54 clean single-leading-byte *serverbound* dialog handlers (the IDA-confirmed pattern) and
   explicitly leaves "~55 non-single-byte leading omissions" and the interior/trailing ±1 as
   honest divergent (`results.md:41-47`). The narrowing is an honest tightening of scope, not a
   silent skip.

2. **PrependCall instead of dispatcher kind.** Confirmed. `export.go` is UNCHANGED in the review
   range (no `dispatcherPrefix` kind added) — Task 5 Step 1's conditional code path was not taken.
   Instead a new surgical `PrependCall` (`baseline_write.go:80-133`) + `prependCallToCalls`
   (`baseline_write.go:135-177`) were added. The implementation mirrors `WriteDispatch`'s
   positional cursor walk so byte-identical sibling objects are disambiguated by position
   (`baseline_write.go:114-132`), preserves unmodeled fields/formatting, errors on unknown FName,
   and no-ops on empty updates. It is correct and tested: `TestPrependCall_SurgicalLeadingByte`
   (`baseline_write_test.go:196-237`) asserts the prepended `Encode1` resolves to `Decode1` at
   index 0, the existing `EncodeStr`→`DecodeStr` remains at index 1, and the untouched sibling
   `B::Other` is unchanged (positional safety). PASS.

3. **3-handler IDA spot-check.** Recorded. `divergent-offbyone-results.md:25-28` documents the
   spot-check of 3 samples confirming each is a genuine leading `COutPacket::Encode1` sub-action
   byte (0x1D / 0x06 / 0x1E), distinct from the constructor opcode.

4. **ValidateShape not modified (option 3).** Confirmed. `internal/idasrc/shapediff.go` is
   UNCHANGED in the review range (empty diff). No task touched the verdict comparison.

5. **54 additive edits, +216/−0, leading Encode1.** Confirmed by inspection of commit 491c7fd5c:
   - `git diff --numstat` over `docs/packets/ida-exports/`: +36/+40/+32/+108 = **+216 added, 0
     deleted** (matches results claim).
   - All **54** added `"op"` lines are `Encode1` (17 at 10-space indent + 37 at 5-space indent);
     **no added line is outside op/comment/brace structure** → "0 non-call content changed"
     verified.
   - Each `Encode1` is prepended as the FIRST element immediately after `"calls": [` (sample
     hunks at `gms_v83.json` addresses 0x65f438, 0x7c37ca, 0x6fdeda) with comment
     "leading sub-action byte (task-081 off-by-one remediation 2026-06-10)".
   - All four baselines still parse as valid JSON post-edit.
   - All 54 targets are `serverbound` (matches the serverbound-dialog-handler claim).

## Test-Exercises-Target Check

- `TestClassifyDiff` exercises `classifyDiff` directly over all four position classes — target hit.
  Note: the implemented `classifyDiff` dropped the plan-draft `case p+s >= n` branch and collapses
  the "shared-on-both / no-shared-edge" cases into `interior` (`diff_shape.go:135-146`). The 4 test
  cases still pass and the simplification is behavior-equivalent for the tested inputs; the
  ±1 leading-omission signal the remediation depends on (`p==0 && s>0` → leading) is correctly
  classified.
- `TestDiffShapeRun_EmitsDivergentRows` exercises the full `diffShapeRun` path (load → resolve →
  ExtractShape → ValidateShape → classify → report), asserts the divergent `#Short` entry appears
  with a delta and that the verified `#A` entry is excluded — target hit. The fixture was adjusted
  from the plan draft (`#A`=[Decode1,Decode4] verified, `#Short`=[Decode1] divergent), which the
  plan explicitly permitted.
- `TestDiffShape_DeterministicAndReadOnly` exercises byte-stability and baseline-immutability —
  target hit.
- `TestPrependCall_SurgicalLeadingByte` exercises `PrependCall` end-to-end including sibling-safety
  — target hit.

No test was found that fails to exercise its stated target.

## Build & Test Results

| Component | Build | Vet | Tests | Notes |
|-----------|-------|-----|-------|-------|
| tools/packet-audit | PASS | PASS | PASS | `go build ./...` exit 0; `go vet ./...` exit 0; `go test -race ./...` all packages ok (cmd 3.42s, idasrc 1.24s). Tool, not a service → no docker bake / no redis-key-guard per plan §22-24. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

All six tasks are implemented with evidence; all three intentional deviations are sound and
honestly documented in the characterization/results docs; `ValidateShape` was not touched; the 54
baseline edits are additive leading-`Encode1` prepends matching the +216/−0 claim exactly; the
+54 verified / −54 divergent re-validation result is recorded; gates are green.

## Action Items

None required for plan completion. Optional follow-ups, both already flagged as deferred in the
results doc (`results.md:41-44`), not gaps in this plan:

1. The ~55 non-single-byte leading omissions (leading `Decode4`/`DecodeStr`) were not spot-checked
   and remain honest divergent — a future pass.
2. The 59 interior + 46 trailing ±1 (width regrouping / trailing optionals) remain divergent by
   the option-3 decision.
