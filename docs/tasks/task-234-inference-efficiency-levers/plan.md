# task-234 — Implementation plan

Spec: `prd.md`. Design: `design.md`.

**Character of this task:** almost all edits land in `.claude/`, `tools/`, and
`docs/`. `tools/verify.sh` will pass trivially for Tasks 1–3, so a green gate is
**not** evidence the change worked — the evidence is the measurement in Task 6.
Do not report these tasks as verified on the gate alone.

---

## Phase A — the cheap levers

### Task 1: Controller context ceiling (FR-3)

**Files:**
- Edit: `.claude/commands/execute-task.md` (the gate-reconcile step)
- Edit: `docs/agent-dispatch.md` (handoff thresholds section, cross-reference)

**Steps:**
1. Read the gate-reconcile step in `.claude/commands/execute-task.md` and locate
   where a plan task's completion is recorded before the next begins.
2. Add the ceiling: after a gate reconciliation, if controller context exceeds the
   threshold, do not start another plan task — append the handoff paragraph to
   `progress.md`, then either delegate the next unit to a fresh agent or emit the
   handoff line as the **final** output of the turn with no further tool calls.
3. State both triggers (Q2 resolution): a token threshold as primary, "after N
   completed plan tasks" as the fallback self-assessment.
4. Include the operative sentence: *a handoff the same context then works past is
   not a handoff.*
5. Cross-reference from `docs/agent-dispatch.md` so the rule is discoverable from
   the dispatch doc as well as the command.

**Verification:** prose only — confirm the rule is unambiguous about *stopping*
rather than recommending. No build impact.

**Evidence to cite in the commit:** `854e6e87` wrote HANDOFF #10 at 243k and ran
26 more turns at 259k avg (6.73M) for one plan task; all 17 task-232 sessions
ended at their peak context.

---

### Task 2: Per-agent tool restriction (FR-1.1–1.3)

**Files:** all 12 of `.claude/agents/*.md` (frontmatter only)

**Steps:**
1. For each definition, read its body and derive the minimum tool set its own
   instructions require. Do **not** guess from the agent's name.
2. Add an explicit `tools:` frontmatter field per the table in `design.md`.
3. For `packet-implementer`, `packet-verifier`, and
   `dispatcher-family-implementer`, keep the wide set and add an inline comment
   stating why (they drive IDA through MCP) — FR-1.3.
4. Validate by dispatching each restricted agent once on a trivial real task and
   confirming it completes without hitting a denied tool. A denied tool is a loud
   failure, so this validation is cheap and conclusive.
5. Record each agent's starting context before and after.

**Verification:** one live dispatch per restricted agent; `tools/verify.sh
--quick` for repo hygiene.

**Baseline to beat:** median subagent starting context **37.3k** (n=197, p25
35.9k, p90 39.3k). Target < 28k.

---

### Task 3: Shell glob quoting note (FR-1.4)

**Files:** the `backend-dev-guidelines` skill, or `.claude/agents/atlas-implementer.md`
— whichever the repo treats as the shell-conventions owner. Find it with `Grep`
rather than assuming a path.

**Steps:**
1. Add: quote glob arguments in shell tool calls — `--include='*.go'`, not
   `--include=*.go` — because zsh expands them before `grep` sees them, producing
   `no matches found` and a wasted retry.
2. Keep it to one or two sentences; this is a 60-occurrence annoyance, not a
   structural problem.

**Verification:** none needed beyond the edit landing.

---

## Phase B — the decision gate

### Task 4: Resolve Q3 — retrospective codemod, or decision rule? ✅ RESOLVED

**Decision: option (b) — ship the decision rule now; build the real rewriter
against the next sweep-shaped task.** Confirmed by the user, matching the design
recommendation.

Rationale of record: the measured waste was not "we lacked a rewriter," it was
that nobody asked whether a rewriter was cheaper than 6,231 implementer turns. A
rule at the point of dispatch captures that. A rewriter built speculatively
against reconstructed testdata costs inference now, has no live work to validate
it, and may not match the next sweep's shape.

**Consequences:**
- **Task 5a is active.** Task 5b is dropped — do not create `tools/<name>/`.
- Task 7 (FR-4) likely reduces to documentation, since "mechanically
  transformed" will be defined by the rule rather than by a `--check` mode that
  now does not exist. Re-read Task 7 with that in mind.
- FR-2.1/2.2/2.3 (the module layout, residue-list contract, and `--check` mode)
  become **specification for the future rewriter**, documented by Task 5a rather
  than implemented. They are not dropped from the PRD; they are deferred with a
  written contract so the next task builds to a known shape.

---

## Phase C

### Task 5a: Decision rule *(active — Q3 resolved to option b)*

**Files:**
- Edit: `.claude/commands/plan-task.md` and/or `.claude/commands/execute-task.md`
- Create: `docs/codemod-vs-agents.md` (or a section in `docs/agent-dispatch.md`)

**Steps:**
1. Write the rule: before dispatching more than N implementers at one templated
   transformation, evaluate whether an AST codemod is cheaper. Ground it in the
   measured figure — 6,231 implementer turns cost 760M, ~122k per turn.
2. Document the mechanical/judgment split from task-232 batch 4 (design.md Lever 2
   lists the six steps and which four are AST-derivable) as the worked example.
   Cite the batch commits (`6a06ffae0`, `54e7e0c3d`, `8776709b8`) so the next task
   has real diffs to build testdata from.
3. Document the intended module layout for when one is written, mirroring
   `tools/rediskeyguard/` (`go.mod`, `analyzer.go`, `analyzer_test.go`, `cmd/`,
   `testdata/`), plus the two contracts the rewriter must honor: **every site is
   rewritten or listed, never silently skipped** (FR-2.2), and **`--check` mode
   for use as a guard afterwards** (FR-2.3).

**Verification:** prose only. The rule must be specific enough to be actionable at
dispatch time — a threshold and a worked example, not "consider whether a codemod
would help."

### Task 5b: Retrospective rewriter — **DROPPED** (Q3 → option b)

Not implemented. Its contract survives as specification in Task 5a step 3.

---

### Task 6: Measurement (FR-5)

**Files:** `~/.claude/tools/session-digest.sh` (user-scope, outside the repo — do
not commit)

**Steps:**
1. Add **starting context** (cache-read + cache-creation + input on the first
   assistant turn) to the session totals block. The current turn-1 cache-read
   figure understates the prefix — it read 23k where the true value was 59k.
2. Add **median subagent starting context** to the subagent roster block.
3. Re-measure after Tasks 1–2 and record before/after in this plan (FR-5.2).

**This task is the actual verification for the whole change.** Tasks 1–3 cannot be
proven by the gate.

**Before/after (FR-5.2):**

| Metric | Before (task-232 baseline) | After |
|---|---|---|
| Subagent starting context, median | 37.3k (n=197, p25 35.9k, p90 39.3k) | pending a fresh session — the restricted agent definitions landed in `8b1513933` (amended by `ff4882ed7`), but this session's agent registry (built at session start, from the main repo's project root) predates them. A same-session probe against `agent-a86d72fe824002390` (atlas-verifier, tool-restriction probe) confirmed the restricted `tools:` list is not yet live: it reported the full unrestricted 16-tool set despite the on-branch definition declaring only `Bash, Read`. The digest's new `START-CTX` column and median line were exercised against this session's own subagent roster (n=11, median 28.4k) as a functional check of the script, not as the FR-5.2 "after" figure — those 11 agents ran under the same pre-restriction registry as the baseline. |
| Main-thread starting context | 59.2k → 49.3k after task-233 | 49,352 tokens, measured this session (`session-digest.sh digest 6bf93293-e0b0-4afb-b462-247405411473`) — consistent with the task-233 figure; no further change expected from Task 6 itself, which only adds instrumentation. |
| Fixed prefix share of billed input | 34% (≈434M of 1,294M) | not remeasured — no script support for this metric was added in this task; out of scope for FR-5's starting-context instrumentation. |
| No execute session peak above ~180k | n/a (target introduced this task) | not yet met / not re-measured — this task-234 execute controller's own peak was measured at 186,525 tokens by the Task 6 reviewer: the session that wrote the ~150k ceiling (Task 1) exceeded the ~180k target that ceiling exists to enforce, because the ceiling landed on the branch but did not govern the already-running session. Corroborating evidence for the Task 1 fix (`commit-boundary.sh` threshold drift) — not itself a fix. |

Targets: subagent starting context **< 28k**; prefix share **< 22%**; no execute
session peak above ~180k.

---

### Task 7: Review-agent right-sizing (FR-4)

**Depends on:** Task 5a/5b — "mechanically transformed" must be defined first.

**Files:** `.claude/commands/execute-task.md` (review step)

**Steps:**
1. Distinguish a codemod-applied, `--check`-confirmed batch from a
   judgment-bearing one.
2. For the former, reduce or skip the per-task review agent — **preserving** the
   pre-PR gate and the guideline reviewers (FR-4.2). This is about review agents
   per plan task, not about the PR review.
3. If Task 4 chose option (b), this likely reduces to documentation.

**Baseline:** review/audit agents were 2,616 turns / 227M / 17.6% of task-232.

---

## Out of scope

Disabling MCPs (user-owned; Task 6 measures the result), further `CLAUDE.md`
trimming (task-233, at diminishing returns), and tool-call hygiene beyond Task 3
(<2% of billed input). The 120-call implementer budget, the verification split,
and the four-phase flow are explicitly **not** changed — the audit found all three
earning their cost.
