# Plan Audit — task-237-agentic-cost-levers

**Plan Path:** docs/tasks/task-237-agentic-cost-levers/plan.md
**Audit Date:** 2026-08-16
**Branch:** task-237-agentic-cost-levers
**Base Branch:** main

## Executive Summary

Six of seven tasks are fully implemented and match the plan; all six shell
test suites pass with real exit status 0. Two genuine gaps were found: Task
1.2 explicitly names `family-auditor.md` and `packet-completeness-critic.md`
for the "Return to the controller" append, and neither file was touched —
report.md documents this as a deliberate skip with rationale ("already return
a one-line verdict... changing them would have been churn"), so it is a
documented deviation, not a silent one. Task 2.4 explicitly names "the two
guideline reviewers" (`backend-guidelines-reviewer.md`,
`frontend-guidelines-reviewer.md`) for a slice-first reference; neither file
contains one, and unlike the Task 1.2 gap, report.md never mentions or
justifies this omission — that is a silent partial. The plan.md ↔ diff
filename mismatch (`tools/verify-facts_test.sh` named in plan.md line 103 vs.
`tools/verify_test.sh` actually shipped) is functionally covered — the file
exists, asserts `--facts` vs. real-run agreement, and passes — but the rename
itself is not called out anywhere in report.md. The Task 7 `.envrc` commit
bullet was explicitly not done, with a written rationale in report.md
(personal config with a home path; the real problem is solved by
`tools/lib/node-env.sh` instead) — a documented, reasoned deviation.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Reviewer return protocol (WS1) | PARTIAL | `docs/review-protocol.md` new (203 lines); `atlas-reviewer.md` new; `## Return to the controller` appended to `backend-guidelines-reviewer.md`, `frontend-guidelines-reviewer.md`, `plan-adherence-reviewer.md` (confirmed via grep); `.claude/commands/execute-task.md` Step 4c routes to `atlas-reviewer`; `docs/superpowers-integration.md` links the protocol. **Gap:** `family-auditor.md` and `packet-completeness-critic.md` were not touched, though named in plan.md:49-50. Documented in report.md:66-68 as an intentional skip with rationale — not silent. |
| 2 | Slice-first artifact access (WS2) | PARTIAL | `tools/doc-slice.sh` (188 lines) + `tools/doc-slice_test.sh` (13/13 assertions pass) new; `docs/slice-first.md` (107 lines) new; referenced from `.claude/agents/atlas-implementer.md`, `atlas-reviewer.md`, `plan-adherence-reviewer.md`, `docs/tooling-conventions.md:64`. **Gap:** plan.md:76-78 explicitly names "the two guideline reviewers" as reference targets — `grep -n "slice-first\|doc-slice" .claude/agents/backend-guidelines-reviewer.md .claude/agents/frontend-guidelines-reviewer.md` returns nothing, and report.md's WS2 file table (report.md:78-87) omits both files with no explanation anywhere in the report. |
| 3 | Deterministic facts (WS3) | DONE | `tools/verify.sh --facts` implemented (tools/verify.sh:47,65,89-91,511); `tools/change-surfaces.sh` (285 lines) + `_test.sh` (13/13 pass); `tools/task-facts.sh` (113 lines) + `_test.sh` (13/13 pass); briefs injected per `.claude/commands/execute-task.md` Step 4b. Filename note: plan.md:103 names `tools/verify-facts_test.sh`; the diff ships `tools/verify_test.sh` instead (199 lines, 6/6 shown assertions pass, header says "29 assertions" per report.md:103). Functionally satisfies validation row 5 (asserts `--facts` selection sets agree with a real run — tools/verify_test.sh:86-160), but the rename from the plan's named filename is not flagged as a deviation anywhere in report.md. |
| 4 | Waiting / polling / micro-delegation (WS4) | DONE | `.claude/hooks/wait-loop-guard.sh` (91 lines) new; `.claude/hooks/wait-loop-guard_test.sh` (86 lines, 33/33 pass, 15 deny + 18 allow); registered on `Bash` matcher in `.claude/settings.json` (confirmed: PreToolUse → matcher "Bash" → `wait-loop-guard.sh`); `docs/agent-dispatch.md:77` new "## Inline vs delegate" section; "Do not fan out" sections added to `backend-guidelines-reviewer.md:305`, `frontend-guidelines-reviewer.md:226`, `atlas-reviewer.md:89` (3 reviewer defs, matching report.md:132's claim of "three reviewer agent definitions"). |
| 5 | Post-implementation fresh context (WS5) | DONE | `docs/post-implementation.md` (160 lines) new; `.claude/commands/fix-pr-bug.md` (111 lines) new, references `docs/post-implementation.md`; CLAUDE.md:95 router row added; `docs/superpowers-integration.md:25` phase table row added; `docs/agent-dispatch.md:189,195` names the post-PR case in §Context handoff. |
| 6 | Measurement (WS6) | DONE | `tools/agent-ledger.sh` (191 lines) new, fields match plan.md:151-154 (`unit, agent_type, model, turns, tool_calls, tool_result_bytes, return_bytes, status, commit` + `verdict`/`caused_fix`/HANDOFF row type — confirmed via 13/13 passing `tools/agent-ledger_test.sh` cases covering verdicts, approvals, fix causation, handoffs); `.claude/commands/execute-task.md:341` reconcile step appends a row per agent. |
| 7 | Small findings | DONE (with one explicit, justified non-delivery) | `docs/reverse-engineering.md:25` adds the `select:` ToolSearch line and (line 34/40) prefers `func_query`/bounded reads over `decompile`; `docs/observability.md:147-148` "never list a whole namespace" bullet added; `docs/tooling-conventions.md:70-73` covers the wrapper-swallows-stdout rule (worded generically as "a token-optimizing shell wrapper" rather than literally "rtk") and line 65-67 covers toolchain-from-task-facts. **`.envrc` was NOT committed** (`git status --porcelain` shows it untracked) — plan.md:170 calls for committing it, but report.md:190,240 documents this as a deliberate reversal with rationale (CLAUDE.md forbids a committed home path; `tools/lib/node-env.sh` solves the underlying nvm-prefix problem instead). Documented deviation, not silent. |

**Completion Rate:** 6/7 tasks DONE, 1/7 PARTIAL (Tasks 1 and 2 both carry a gap; counted once each below) — by task-count: 5 DONE, 2 PARTIAL, 0 SKIPPED, 0 NOT_APPLICABLE (Task 7's `.envrc` line item is a documented non-delivery within an otherwise-DONE task, not counted as a separate SKIPPED task)
**Skipped without approval:** 1 (Task 2.4's guideline-reviewer references — undocumented in report.md)
**Partial implementations:** 2 (Task 1, Task 2)

## Skipped / Deferred Tasks

**Task 1 — `family-auditor.md` / `packet-completeness-critic.md` not updated.**
Plan.md:49-50 lists both files alongside the three that were updated. Neither
carries a "Return to the controller" section or any reference to
`review-protocol.md`. Impact is low: report.md:66-68 explains both agents
already return a one-line verdict-plus-counts and name their own artifact
file, so the protocol's substance is arguably already met even without the
formal section — but this is an unreviewed, unilateral judgment call made
mid-implementation, not something the plan authorized in advance. A future
reader of the plan without report.md would reasonably expect these files to
carry the new section.

**Task 2 — guideline reviewers not wired to slice-first.**
Plan.md:76-78: "Reference it from `.claude/agents/atlas-implementer.md`,
`atlas-reviewer.md`, `plan-adherence-reviewer.md`, the two guideline
reviewers, and `docs/tooling-conventions.md`." Three of five targets got the
reference; `backend-guidelines-reviewer.md` and `frontend-guidelines-reviewer.md`
did not, and report.md's own change table for WS2 (report.md:78-87) silently
omits them — no rationale is offered anywhere in the report, unlike the
Task 1 gap which is explicitly called out. Impact: these two reviewers are the
ones most likely to walk a large diff or a large guidelines doc (backend
`resources/audit-checklist.md`, frontend guideline set) and are exactly the
kind of consumer the slice-first default targets; leaving them out reduces
the token-savings surface the workstream is optimizing for.

**Task 3 — filename drift, `verify-facts_test.sh` → `verify_test.sh`, unremarked.**
Not a functional gap (validation row 5 is satisfied — `tools/verify_test.sh`
asserts `--facts` selection agrees with a real `--quick` run, tools/verify_test.sh:86-160,
6/6 sampled assertions pass), but the rename from the plan's literal filename
is not called out as a deviation in report.md, so a reader diffing plan vs.
report cannot tell whether this was intentional consolidation (folding the
`--facts` tests into the general `verify.sh` test file) or an oversight.

**Task 7 — `.envrc` not committed.**
Explicitly reversed with a written rationale (report.md:190,240); not a gap,
included here only because the deliverable literally named in plan.md:170 is
absent from the diff and a naive diff-only audit would flag it as SKIPPED.

## Build & Test Results

This branch is documentation + POSIX shell tooling only — no Go or TypeScript
services were touched, so no `go build`/`go test`/`npm` runs apply. Shell test
suites run directly, actual exit codes captured:

| Suite | Result | Notes |
|---|---|---|
| `tools/agent-ledger_test.sh` | PASS (exit 0) | "agent-ledger_test.sh: all assertions passed" |
| `tools/change-surfaces_test.sh` | PASS (exit 0) | "change-surfaces_test.sh: all assertions passed" |
| `tools/doc-slice_test.sh` | PASS (exit 0) | "doc-slice_test.sh: all assertions passed" |
| `tools/task-facts_test.sh` | PASS (exit 0) | "task-facts_test.sh: all assertions passed" |
| `tools/verify_test.sh` | PASS (exit 0) | "verify_test.sh: all assertions passed" |
| `.claude/hooks/wait-loop-guard_test.sh` | PASS (exit 0) | "passed: 33  failed: 0" |

## Overall Assessment

- **Plan Adherence:** MOSTLY_COMPLETE
- **Recommendation:** NEEDS_FIXES

## Action Items

1. Either add the "Return to the controller" section to `family-auditor.md`
   and `packet-completeness-critic.md` per plan.md:49-50, or amend the plan
   itself to record the exemption (report.md already argues the case; the
   plan document should reflect the accepted deviation so it isn't lost).
2. Add a slice-first reference to `.claude/agents/backend-guidelines-reviewer.md`
   and `.claude/agents/frontend-guidelines-reviewer.md` per plan.md:76-78, or
   document in report.md why they were excluded.
3. Note the `tools/verify-facts_test.sh` → `tools/verify_test.sh` rename
   explicitly in report.md (or plan.md) so the deviation is traceable.
