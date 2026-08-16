# task-234 — Inference efficiency levers

**Status:** spec
**Related:** task-233 (CLAUDE.md restructure — the prefix lever, already landed)
**Evidence:** `~/.claude/audits/session-task-232-three-largest.md` (session audit of
task-232, 17 sessions)

## 1. Problem

Task-232 consumed **1,294M billed input tokens** across 17 sessions
(302M main-thread + 993M subagent). A session-level audit established that this
was *not* caused by bad tool use — tool errors run 1.9% on main threads and 3.4%
in subagents, retry chains account for 1.4% of calls, and the work itself was
correct. The cost is structural, and it decomposes into two terms:

**Term 1 — the fixed prefix, re-read on every turn.**

| | measured |
|---|---|
| Main-thread starting context | 59.2k → 49.3k (after task-233) |
| Subagent starting context | median **37.3k** (n=197; p25 35.9k, p90 39.3k) |
| Turns | 1,877 main + **8,894 subagent** = 10,771 |

The prefix alone accounts for **≈434M of 1,294M — 34%** of the task. 331M of that
is subagent prefix, because 83% of all turns are subagent turns.

**Term 2 — the turn count itself.** 6,231 implementer turns cost **760M (59% of
the task)** at ~122k billed input per turn. A large share went to mechanically
repetitive edits: read `requests.go` → edit → read `processor.go` → edit, ~60
times across 8 services in a single batch, following a templated recipe the task
had already written down (`service-wiring-recipe.md`, Patterns A/B/C/D).

Task-233 addressed the prefix from the `CLAUDE.md` side and delivered a measured
~10k reduction ≈ **107M per task of this size (8%)**. That lever is now largely
spent: `CLAUDE.md` is 10,521 bytes ≈ 2.6k tokens of a 37.3k subagent prefix. The
remaining prefix is base system prompt plus **tool schemas**, and the remaining
term-2 cost is untouched entirely.

## 2. Goals

- **G1** — Cut the subagent prefix by restricting per-agent tool sets.
- **G2** — Remove mechanically-derivable implementer turns by providing codemod
  tooling for repeated AST-shaped sweeps.
- **G3** — Stop controller threads from running to 250–370k when their state is
  already durable on disk.
- **G4** — Right-size review-agent spend (2,616 turns / 227M / 17.6% of the task).
- **G5** — Make all four measurable and re-measurable, so the next audit compares
  numbers rather than impressions.

## 3. Non-goals

- Disabling MCP servers by default — the user is doing this directly; G1 measures
  the result rather than implementing it.
- Further `CLAUDE.md` trimming — task-233 owns that and it has reached diminishing
  returns.
- Tool-call hygiene micro-optimization. Measured at <2% of billed input; not worth
  a work item. The single exception is folded into FR-1.4 because it costs one
  line.
- Changing the 120-call implementer budget, the verification split, or the
  four-phase task flow. The audit found all three earning their cost:
  continuation agents re-orient in ~18KB, ≈1% of their spend.

## 4. Functional requirements

### FR-1 — Per-agent tool restriction

- **FR-1.1** Every agent definition in `.claude/agents/` declares an explicit
  `tools:` frontmatter field. Today **none of the 12 do**, so all inherit the full
  tool set including `Agent`, `Workflow`, `Artifact`, `WebFetch`, `NotebookEdit`,
  the Cron family, and every MCP tool.
- **FR-1.2** Each agent's declared set is the minimum its own instructions
  require. An implementer needs file and shell tools; it does not need to dispatch
  agents, publish artifacts, or fetch the web.
- **FR-1.3** Agents that legitimately need a wide set (`packet-implementer`,
  which drives IDA via MCP) keep it, and the definition says why inline.
- **FR-1.4** The backend guidelines note that shell glob arguments must be quoted
  (`--include='*.go'`), since zsh expands them before `grep` sees them — measured
  at 60 occurrences across 5 sessions.

### FR-2 — Codemod tooling for mechanical sweeps

- **FR-2.1** A Go AST rewriter under `tools/`, following the existing analyzer
  module layout (`tools/rediskeyguard/` — `analyzer.go`, `analyzer_test.go`,
  `cmd/`, `testdata/`, own `go.mod`).
- **FR-2.2** It applies the mechanical share of a templated sweep and **emits a
  residue list** of call sites needing human or agent judgment. It never silently
  skips a site: every site is rewritten, or listed.
- **FR-2.3** It is idempotent and has a `--check` mode, so it can run under
  `tools/verify.sh` as a guard once a migration has landed.
- **FR-2.4** The task's plan documents which task-232 patterns were mechanical and
  which were not, so the split is evidence-backed rather than asserted.

### FR-3 — Controller context ceiling

- **FR-3.1** `/execute-task` gains a hard ceiling: after a gate reconciliation, if
  controller context exceeds a stated threshold, the controller does not start
  another plan task.
- **FR-3.2** At the ceiling the controller appends its handoff paragraph to
  `progress.md` and then either delegates the next unit to a fresh agent or stops
  with the handoff line as the final output of the turn — **no further tool
  calls**.
- **FR-3.3** The rule states that a handoff the same context then works past is
  not a handoff. In task-232 this is exactly what happened: `854e6e87` wrote
  HANDOFF #10 at 243k and then ran 26 more turns at an average of 259k.

### FR-4 — Review-agent right-sizing

- **FR-4.1** The review step distinguishes a batch that is mechanically
  transformed (codemod output, verified by `--check`) from one requiring
  judgment, and does not spend a full review agent on the former.
- **FR-4.2** Any reduction preserves the existing gate: `tools/verify.sh` and the
  guideline reviewers still run before a PR. This is about *review agent count per
  plan task*, not about removing the pre-PR review.

### FR-5 — Measurement

- **FR-5.1** A repeatable way to report, for any session: starting context,
  turn count, and billed input split main vs subagent.
- **FR-5.2** The task records before/after numbers for FR-1 and FR-3 so the next
  audit can verify the change landed rather than assuming it.

## 5. Success metrics

| Metric | Baseline (task-232) | Target |
|---|---|---|
| Subagent starting context (median) | 37.3k | **< 28k** |
| Fixed prefix share of a task's billed input | 34% | **< 22%** |
| Controller peak context, execute sessions | 170k–370k; **all 17 sessions ended at their peak** | no execute session exceeds ~180k |
| Implementer turns on a templated sweep | ~60 read/edit pairs per 8-service batch | codemod handles the mechanical share; agents touch only the residue |

## 6. Open questions

- **Q1** What is the actual token cost of the MCP tool schemas in the subagent
  prefix? Needs a measured before/after once MCPs are disabled — this sizes FR-1.
- **Q2** Is the ceiling in FR-3.1 best expressed as a token threshold, a completed
  plan-task count, or both? A token threshold is directly tied to the measured
  cost; a task count is easier for a controller to self-assess.
- **Q3** Is the first codemod worth writing *retrospectively* for the task-232
  wiring pattern (no remaining call sites to convert), or should FR-2 wait for the
  next sweep-shaped task and be built against live work?
