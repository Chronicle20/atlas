# Plan Audit — task-234-inference-efficiency-levers

**Plan Path:** docs/tasks/task-234-inference-efficiency-levers/plan.md
**Audit Date:** 2026-08-16
**Branch:** task-234-inference-efficiency-levers
**Base Branch:** main (merge base `4df8f00a5`)
**HEAD:** `1d1cd45d4` (8 commits)

## Executive Summary

All six work-carrying plan tasks (1, 2, 3, 5a, 6, 7) are implemented and traceable in the diff; Task 4 is a recorded decision with no edits, and Task 5b is correctly dropped (no `tools/` module created). The prose is careful about the plan's own trap — nowhere does `plan.md` claim gate-passing as evidence, and both places where a figure could be misread as an "after" result (Task 2's steps 4–5, Task 6's 28.4k) are explicitly labeled "pending" / "functional check, not the after figure." The re-derived break-even arithmetic in `docs/codemod-vs-agents.md` checks out (760M / 6,231 ≈ 122,010/turn; 120 × 122,010 ≈ 14.6M on both sides of the break-even), and a repo-wide search found no stale "third implementer" / old-threshold wording left behind for that rule. One genuine gap was found outside the plan's own file list: `.claude/hooks/commit-boundary.sh` still hard-codes the pre-Task-1 ~250k/110-tool-call threshold and its user-facing message actively misquotes `execute-task.md` Step 4e as still saying "~250k" — Task 1's own design/plan never considered this second live enforcement surface. No Go or TypeScript changed, so no build/test run applies.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Controller context ceiling (FR-3) | DONE | `.claude/commands/execute-task.md:246-249` — unconditional "150k tokens — or 4 plan tasks... whichever comes first," "no carve-out." Line 255 "Stop: no further tool calls after the handoff line." Delegate-branch closed via prose at `:259-265`, citing `CLAUDE.md`'s handoff rule. Cross-ref present in `docs/agent-dispatch.md:105-122`. See Important finding below re: `commit-boundary.sh`. |
| 2 | Per-agent tool restriction (FR-1.1–1.3) | DONE | `grep -n '^tools:' .claude/agents/*.md` → exactly 9 hits across 12 files (verified). Three packet agents (`packet-implementer.md:31-36`, `packet-verifier.md:26-31`, `dispatcher-family-implementer.md:30-35`) omit `tools:` with an inline comment naming omission as the documented mechanism and citing `https://code.claude.com/docs/en/sub-agents.md` (Ruling 10). Steps 4–5 (live validation, before/after context) recorded as pending, not verified: `plan.md:161`. |
| 3 | Shell glob-quoting note (FR-1.4) | DONE | `docs/tooling-conventions.md:46-48` (diff: +4 lines) — landed at the Ruling-2 owner, not the skill or `atlas-implementer.md`. |
| 4 | Resolve Q3 | DONE (decision only) | `plan.md:86-107` records option (b); no code/doc edits expected or found. |
| 5a | Codemod-vs-agents decision rule (FR-2 spec) | DONE | `docs/codemod-vs-agents.md` (138 lines, standalone per Ruling 3), `CLAUDE.md:89` router row, `docs/agent-dispatch.md:60-63` cross-ref, `execute-task.md:67-71` and `plan-task.md:99-104` dispatch/plan-time hooks. Arithmetic re-verified (see below). |
| 5b | Retrospective rewriter | DROPPED (correct) | `git diff 4df8f00a5 HEAD --name-only \| grep '^tools/'` → empty. No new Go module. |
| 6 | Measurement (FR-5) | DONE | Committed half: `plan.md:157-166` before/after table, commit `9e5d465c1`. User-scope half (`~/.claude/tools/session-digest.sh`) correctly uncommitted (Ruling 7) and produces no diff, as the plan itself specifies. |
| 7 | Review-agent right-sizing (FR-4) | DONE | `execute-task.md:164-173` — reduced-review path gated on "codemod-produced and `--check`-confirmed," explicitly "dormant" today since no rewriter/`--check` exists. Not self-declarable (see Adversarial check below). |

**Completion Rate:** 6/6 work-carrying tasks (100%); Task 4 decision-only and Task 5b correctly dropped are excluded from the denominator.
**Skipped without approval:** 0
**Partial implementations:** 0 (Task 2 steps 4–5 and Task 6's "after" figure are plan-anticipated, explicitly-labeled deferrals, not silent partials — see notes below)

## Skipped / Deferred Tasks

None skipped. Two deliberate, plan-anticipated deferrals, both honestly labeled:

- **Task 2, steps 4–5** (live dispatch validation, before/after subagent context). `plan.md:161` states plainly: "pending a fresh session — the restricted agent definitions landed in `8b1513933`... but this session's agent registry... predates them," and separately labels the 28.4k figure exercised in Task 6 as "a functional check of the script, not... the FR-5.2 'after' figure." A careless reader cannot mistake this for a verified result — the sentence is explicit. Root cause (agent registry resolves from the session's project root, built at session start) is documented in `progress.md:120-132` and is a harness limitation, not an implementation gap; it will resolve itself once this branch merges and a fresh session starts.
- **Task 6, "no execute session peak above ~180k" target** — never mentioned in the before/after table, not even to say "not checked" (the task's own flagged known-minor). Confirmed: `plan.md:157-167` lists two targets in the "Targets:" line (28k, 22%) with the 180k target present in the same sentence but not reflected as a row or footnote in the table above it, and not addressed anywhere else in `plan.md`. The `progress.md` ledger (Ruling 13, lines 297-306) does capture a real data point — this controller session's own peak context measured at 186,525, over the target — but that datum lives only in the ledger, not in `plan.md`. Impact: low. It is evidence *for* Task 1's ceiling, not evidence the shipped levers failed; the session measured predates Task 1's ceiling taking effect and is a different session shape (SDD controller, not `/execute-task`). Still, a reader of `plan.md` alone has no way to know this target was even considered. **Classification: Minor**, as flagged.

## Adversarial / Deeper Checks

- **Task 7 dormancy exploit check:** `execute-task.md:164-171` gates the reduced-review path on "codemod-produced and `--check`-confirmed (`docs/codemod-vs-agents.md`)" — an objective artifact check, not a self-declaration ("mechanical" is never used as the trigger word; "hand-applied 'mechanical' batch with no `--check` PASS behind it" is explicitly named as still judgment-bearing and gets full review). Since no rewriter or `--check` mode exists anywhere under `tools/` (confirmed empty in the diff), there is no path today for a controller to self-declare its way into reduced review. **Not exploitable — confirmed non-Critical.**
- **Codemod break-even re-derivation:** 760,000,000 / 6,231 ≈ 122,010 tokens/turn (matches `docs/codemod-vs-agents.md:30-32`). 120 × 122,010 ≈ 14.6M — the doc states this is the *same* bound on both the manual-dispatch side and the rewriter-build side, and explicitly derives N=2 (not N=1 or N=3) from two separate arguments: (a) a single site can't prove "templated" (precondition), (b) after dispatch 1 the shape is known and one further manual dispatch reaches the rewriter's own worst-case build cost — the break-even point. This re-derivation is internally consistent and matches `docs/codemod-vs-agents.md:41-59`. The doc also correctly downgrades its own "~52×" sanity check to not-pinning-the-threshold (`:64-69`), which is the right epistemic move.
- **Stale-threshold sweep:** `grep -rn "third implementer\|N=2\|before the third"` across `docs/` and `.claude/` returned no hits outside unrelated task folders (task-102's `N=2` is an unrelated saga-timeout parameter). `CLAUDE.md:89`, `docs/agent-dispatch.md:60-63`, `.claude/commands/execute-task.md:67-71`, and `.claude/commands/plan-task.md:99-104` all consistently say "second implementer" / "second dispatch." No disagreeing numbers found for the codemod rule.
- **Repo conventions:** no literal home/absolute paths introduced under `docs/` (the one `/home/` / `/Users/` hit is inside `docs/tooling-conventions.md`'s own convention text describing the *pattern* to avoid, not a violation). No CRLF→LF normalization detected in any changed file. Every cited figure in `plan.md`'s before/after table and in `context.md`'s "Numbers to cite" table matches (37.3k median subagent starting context, 34%/≈434M-of-1,294M prefix share, 59.2k→49.3k main-thread context, batch-4 commits `6a06ffae0`/`54e7e0c3d`/`8776709b8` all confirmed as real commits in this repo's history via `git cat-file -t`).

## Important Finding — stale threshold in a live enforcement surface outside the plan's file list

`.claude/hooks/commit-boundary.sh:29-34` and `:83-91` still encode the **pre-Task-1** threshold:

```
FLOOR=40
# Second tier: the ~250k controller threshold from .claude/commands/execute-task.md
# Step 4e, expressed in the only unit this hook can see. ...
ESCALATE=110
...
body="... At this call count the context is roughly 250k tokens (~2.1k per call over the
23k floor). CLAUDE.md and .claude/commands/execute-task.md Step 4e both say:
never carry the controller past ~250k with tasks remaining. That is here.
```

This is a live `PostToolUse` hook that fires at real commit boundaries during real sessions and directly quotes `execute-task.md` Step 4e as authority. Task 1 changed Step 4e's threshold to **150k (or 4 completed plan tasks), unconditionally, no carve-out** (`execute-task.md:246-249`) — the hook's comment and message now misquote the very step they cite, and its `ESCALATE=110` mapping still targets ~250k, not ~150k. Neither `prd.md`, `design.md`, nor `plan.md` for task-234 ever mentions `commit-boundary.sh` (confirmed via grep — zero hits), so this is a genuine scoping blind spot, not a deliberate exclusion: the task changed the canonical policy text in two places (`execute-task.md`, `agent-dispatch.md`) but missed a third live artifact that encodes and *enforces* the same number.

Impact: a controller that trusts the hook's paraphrase (which is exactly why the hook exists — to avoid re-reading policy text at every commit) will not be nagged until ~250k-equivalent tool-call count, well past the 150k ceiling Task 1 just wrote and past the PRD's own ~180k session-peak target. This directly undercuts FR-3's purpose. The authoritative text in Step 4e is itself correct and unconditional, so a controller that reads Step 4e directly is unaffected — but the whole point of the hook is that controllers don't re-read Step 4e at every commit.

**Classification: Important.** Not Critical because the authoritative rule text (`execute-task.md` Step 4e) is correct and unconditional on its own, and the hook fails safe (silent on error, never blocks). But it is a real, functional inconsistency between two enforcement surfaces on the exact number this task exists to fix, and it actively misinforms whoever reads its message.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| N/A | N/A | N/A | Diff is markdown/frontmatter-only (`.claude/`, `docs/`, `CLAUDE.md`) — no Go module or TypeScript changed. `tools/verify.sh --quick` was run per-task by the controller and passed trivially each time ("no Go module changed"); per the plan's own preamble and Ruling 8, this is hygiene-only and is not treated here as evidence of correctness. |

## Overall Assessment

- **Plan Adherence:** FULL — every task the plan assigned work to is implemented, and every deliberate deferral is legibly labeled as such in `plan.md` itself (not just the ledger).
- **Recommendation:** NEEDS_FIXES — one Important finding (`commit-boundary.sh` stale threshold) should be corrected before/soon after merge; the branch is otherwise ready.

## Action Items

1. Update `.claude/hooks/commit-boundary.sh` — `ESCALATE` and the accompanying comment/message — to reflect the 150k / 4-completed-plan-task rule from `execute-task.md` Step 4e, or explicitly document why the hook intentionally uses a different (looser) tool-call-based proxy threshold than the token-based Step 4e rule.
2. Optional, low-priority: add one line to `plan.md`'s Task 6 section (or the Targets line) explicitly stating the ~180k session-peak target was not measured against these levers in this session, mirroring the honesty already present for the other two targets. (Confirmed Minor, already flagged by the executing session as a known deferred item.)
