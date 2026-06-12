# Plan Audit — task-081-ida-export-reharvest (per-branch-verification-plan)

**Plan Path:** docs/tasks/task-081-ida-export-reharvest/per-branch-verification-plan.md
**Audit Date:** 2026-06-09
**Branch:** task-081-ida-export-reharvest
**Base Branch:** main (review range 6c4bf73 → ec78c8d)

## Executive Summary

All 8 plan tasks are implemented, committed, and covered by tests that genuinely
exercise the new behavior. The build/test gates from `tools/packet-audit/`
(`go build ./...`, `go vet ./...`, `go test -race ./...`) are green. The three
documented deviations (Task 2 hand-crafted structural fixtures; Task 4 lossless
surgical WriteDispatch; Task 6 per-base-handler bijection grouping) are all sound,
honestly documented in the results/triage docs, and tested. The Task 8 live-run
artifacts match the documented claims exactly: 77 persisted dispatch selectors
(14/18/31/14) and 251 allowlist entries (33/59/123/36). The only non-code finding is
cosmetic: the plan file's 47 step checkboxes were never ticked (`- [ ]` throughout),
though every step's deliverable is present in the diff. Recommendation: READY_TO_MERGE.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `Selector.Default` + default-arm extraction | DONE | `internal/idasrc/extract.go:11` (`DefaultGuardToken`), `:17-21` (`Selector.Default`), `:67-75` (`clauseMatches` default handling). Test `extract_test.go:89` `TestExtractShape_DefaultArm` asserts both default-match and case-must-not-match-default. |
| 2 | if/else dispatch guard emission in `ParseDecompile` | DONE (deviation, documented) | `parse.go:70-73` (`reIfEq`/`reElse`), `:451-457` (`ifChainEntry`), `:570-608` (arm detection + binding), `:691-705` (chain pop). Tests `parse_test.go:393` `TestParseDecompile_IfElseDispatch` (full else-if chain + bare-else default) and `:423` `TestParseDecompile_IfElseTrailingElse`. Fixtures `testdata/ifelse_chain.c`, `ifelse_else.c` are hand-crafted structural (NOT real Hex-Rays) — see Deviations. |
| 3 | full case-label-set enumeration on `Fields` | DONE | `idasrc.go:84` (`Fields.CaseLabels`), `:89-119` (`CaseSet`/`NewCaseSet`/`add`/`Has`/`Values`), `parse.go:716` (`ParseDecompileFields`), `:752-845` (`collectCaseLabels` second pass). Test `parse_test.go:439` `TestParseDecompile_CaseLabelSet` with `testdata/switch_emptycase.c` (asserts label 3 from a read-less `case 3:`). |
| 4 | WriteDispatch persisted dispatch writer | DONE (deviation, documented) | `internal/idasrc/baseline_write.go` — lossless surgical raw-JSON inserter (`WriteDispatch:28`, `orderedFunctionRaws:75`, `insertDispatchField:110`, `matchPythonEscaping:176`). 6 tests in `baseline_write_test.go` incl. round-trip, unknown-FName, order/indent preservation, no-op byte-stability, legacy `note` key, Python escaping. See Deviations. |
| 5 | `resolve-dispatch` subcommand (infer + agent-confirmation gate) | DONE | `cmd/resolve_dispatch.go` (`resolveDispatchRun:40`, `writeWorklist:125`); wired in `cmd/root.go:41-42` + `runResolveDispatch:228`. Test `cmd/resolve_dispatch_test.go:13` asserts high-confidence picks written to baseline + "auto-accepted" roll-up + worklist emitted. |
| 6 | bijection missing/extra-mode buckets in validate | DONE (deviation, documented) | `internal/idasrc/bijection.go` (pure diff); `cmd/validate.go:80-196` (per-base-handler `handlerAgg` grouping across addresses, allowlist subtraction, buckets); `live.go:136` populates `CaseLabels`. Tests: `bijection_test.go` (missing/extra, nil-client), `validate_test.go:157` (`TestValidate_BijectionMissingExtra`), `:199` (`TestValidate_BijectionMultiAddressNoFalseMissing`), `live_test.go:98` (`TestResolveLiveCaseLabels`). See Deviations. |
| 7 | per-version allowlist | DONE | `internal/idasrc/allowlist.go` (`LoadAllowlist`, `Suppressed`, missing-file = empty). Applied `cmd/validate.go:66,179-182`; default path + jms_v185 quirk `cmd/root.go:138,159-166`. Tests `allowlist_test.go` (suppress, missing-file), `validate_test.go:175` (`TestValidate_AllowlistSuppressesMissing`). |
| 8 | live four-IDB run + record results | DONE | `per-branch-verification-results.md`, `missing-mode-triage.md`. Baselines carry 77 `"dispatch"` fields (v83=14, v87=18, v95=31, jms=14); allowlists carry 251 entries (33/59/123/36). All counts verified against the docs. Baseline diff is +509/−0 (purely additive, lossless claim confirmed). All modified JSON parses. |

**Completion Rate:** 8/8 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Deviations (all documented and sound)

1. **Task 2 fixtures are hand-crafted structural `.c`, not harvested Hex-Rays.**
   `testdata/ifelse_chain.c` and `ifelse_else.c` are clean, idealized C (perfect
   comments, no `/* line: N */` annotations, no aliased scratch vars) — clearly not
   real Hex-Rays output, contradicting the plan's emphatic "fixtures MUST be real
   Hex-Rays text" (plan §Task 2, Phase-1.5 lesson). The user flagged this as a known
   deviation: IDA-MCP was unavailable mid-execution and real-fixture hardening is
   deferred. Assessment: **honest and acceptable.** The parser logic is genuinely
   exercised — the chain fixture drives an `if (result==2) / else if (result==5) /
   else` cascade and the test asserts the exact guard on each read incl. the
   `<default>` token; the trailing-else fixture is a separate case. The if/else state
   machine, `clearActiveArm`, and the sibling-block `startDepth` pop (an improvement
   over the plan's `bodyDepth` pseudocode — `parse.go:451-456` documents why) are all
   covered. The residual risk is that real Hex-Rays idioms (else-if with casts,
   recycled discriminators) aren't fixture-tested — but those arms intentionally fall
   through to "no guard / honest unverifiable" by the documented bail rule
   (`parse.go:66-69`), so the failure mode is conservative, not a fabricated read.
   The deferral is recorded in MEMORY (`project_packet_audit_exporter_real_decompile_gaps`)
   and the results doc's "what remains" section. Recommend a follow-up to re-harvest
   real if/else fixtures before relying on if/else extraction at scale.

2. **Task 4 WriteDispatch rewritten from typed-marshal to lossless surgical inserter.**
   The plan's pseudocode used `json.Unmarshal` into `exportFile` + `json.MarshalIndent`
   round-trip. That was found lossy during Task 8 (dropped `region` ×156, `note`/`_note`
   ×173, `size` ×1, churned indent + escaping — `results.md:91-98`). The committed
   version (`baseline_write.go`) streams function object bytes via `json.Decoder`/
   `RawMessage` and string-replaces only the target object, inserting the `dispatch`
   field before `"calls"` with matched indentation and Python-style escaping. Assessment:
   **correct and well-tested.** `TestWriteDispatch_PreservesOrderAndIndent` proves a
   no-op write is byte-identical and a real write preserves key order; the +509/−0
   baseline diff confirms zero non-dispatch bytes changed in the live run. The test was
   adapted to newline-formatted input (the surgical writer anchors on the `"calls"`
   line) — a necessary change from the plan's single-line fixture.

3. **Task 6 bijection grouped per base handler (not per address).**
   The pure `Bijection` function is per-CaseSet as planned; the per-address→per-handler
   correction lives in `validate.go:80-196` (`handlerAgg` accumulates each base's client
   case-set + bound cases across all its addresses, diffs once). Without it the v95
   outlined party/guild bodies (e.g. `OnPartyResult` at 8 addresses) double-counted and
   false-reported cases bound at sibling addresses (`missing-mode-triage.md:7-16`;
   403→251). Assessment: **correct and regression-tested** by
   `TestValidate_BijectionMultiAddressNoFalseMissing` (`validate_test.go:199`), which
   binds case 2 at 0x201 and asserts it is NOT reported missing at 0x200.

## Build & Test Results

| Module | Build | Vet | Tests (-race) | Notes |
|--------|-------|-----|---------------|-------|
| tools/packet-audit | PASS | PASS | PASS | `go build ./...` exit 0; `go vet ./...` exit 0; `go test ./... -count=1` exit 0. New task tests run & pass (TestParseDecompile_IfElse*, _CaseLabelSet, TestExtractShape_DefaultArm, TestWriteDispatch_*, TestBijection_*, TestAllowlist_*, TestResolveLiveCaseLabels, TestValidate_Bijection*/Allowlist*, TestResolveDispatch_AutoAcceptsHighConfidence). |

This is a tool, not a service: no `docker buildx bake`, no `redis-key-guard` (per task
instructions and CLAUDE.md's service-only verification rules). Scope is confined to
`tools/packet-audit/` and `docs/` — no other Go module was touched.

## Data Artifact Verification (Task 8)

- Dispatch selectors persisted: 77 total (gms_v83=14, gms_v87=18, gms_v95=31,
  gms_jms_185=14) — matches `results.md` and the persist commit b7fe9607.
- Allowlist entries: 251 total (gms_v83=33, gms_v87=59, gms_v95=123, jms_v185=36) —
  matches `missing-mode-triage.md`. Single distinct reason string
  `"partial implementation — sub-op not built (task-081 triage 2026-06-09)"`,
  em-dash correctly `—`-escaped.
- All four baselines + four allowlists parse as valid JSON.
- Baseline diff over the full range is +509/−0 (purely additive), confirming the
  lossless-writer claim.
- Inserted blocks carry both `"dispatch"` and a provenance `"notes"` field
  (e.g. `"inferred-high-confidence (1.00) @0x5facca"`).

## Skipped / Deferred Tasks

None skipped. Explicitly deferred (out of scope, documented in `results.md`
"what remains"): real-Hex-Rays if/else fixture hardening; demangled `Class::Method`
helper-name resolution (the dominant recall lever); non-equality dispatch modeling;
manual IDA confirmation of the ~19/version ambiguous-with-proposal picks;
loop/opaque-block divergence modeling. These are future-task material, correctly
scoped out, and do not block this plan's completion.

## Overall Assessment

- **Plan Adherence:** FULL — every task's required types, functions, wiring, and tests
  are present; deviations are improvements with honest documentation, not silent skips.
- **Recommendation:** READY_TO_MERGE

## Action Items

1. (Cosmetic) The plan file still shows 47 `- [ ]` unchecked step boxes and 0 checked.
   Consider ticking them to reflect the committed state, or note in the plan that
   completion is tracked via the results doc + commits. Non-blocking.
2. (Follow-up, already deferred) Re-harvest real Hex-Rays if/else fixtures and add them
   as `parse_test.go` cases before relying on if/else extraction at production scale,
   per the documented Phase-1.5 lesson. Non-blocking for this plan.

---

> Note: a prior `backend-guidelines-reviewer` audit previously occupied this file
> (DOM-* checklist for `tools/packet-audit`). This plan-adherence audit replaced it
> per the task instruction to write to `audit.md`. The backend audit's substance is
> independent; re-run `backend-audit` if its findings are needed alongside this one.
