# task-234 — Design

Spec: `prd.md`. Evidence: `~/.claude/audits/session-task-232-three-largest.md`.

## Shape of the change

Four independent levers plus a measurement harness. **None of them touch service
code**; the work is entirely in `.claude/`, `tools/`, and `docs/`. That has one
consequence worth stating up front: `tools/verify.sh` will pass trivially for most
of this task, so **the verification story is the measurement in FR-5, not the
gate**. A green gate here proves nothing about whether the change worked.

The levers are ordered below by measured impact, but they are independent and can
land in any order. FR-1 and FR-3 are the cheap ones and should land first so the
next task-sized piece of work is already cheaper while FR-2 is being built.

---

## Lever 1 (FR-1) — Per-agent tool restriction

### Current state

All 12 definitions in `.claude/agents/` declare `name`, `description`, and
`model`. **None declares `tools:`.** The agent-type listing confirms the effect:
every Atlas agent reads `Tools: All tools`, while the built-in `Explore` agent
shows a restricted set — so the mechanism works, it is simply unused here.

Measured consequence: subagent starting context is a median 37.3k with a very
tight spread (p25 35.9k, p90 39.3k). The tightness is the tell — the prefix is
dominated by a fixed component identical across agents, not by per-agent briefs.
Of that 37.3k, `CLAUDE.md` is ~2.6k and the agent definition ~2.7k
(`atlas-implementer.md`, 10,886 bytes), leaving **~32k of base prompt + tool
schemas**.

### Approach

Add a `tools:` field to each definition, derived from what the definition's own
body actually instructs the agent to do. Proposed sets:

| Agent | Tools | Rationale |
|---|---|---|
| `atlas-implementer` | Read, Write, Edit, Bash, Grep, Glob | Its body: brief-first discovery, edit, module-local `go build`/`go test`, commit. Never dispatches, never fetches. |
| `atlas-verifier` | Bash, Read | "Runs one command and quotes the output"; explicitly never edits. |
| `todo-scanner` | Read, Grep, Glob, Write, Bash | Scan and write `docs/TODO.md`. |
| `plan-adherence-reviewer`, `backend-guidelines-reviewer`, `frontend-guidelines-reviewer`, `family-auditor`, `packet-completeness-critic` | Read, Grep, Glob, Bash, Write | Read-and-report; `Write` only for their audit artifact. |
| `service-documentation` | Read, Grep, Glob, Write, Edit, Bash | Confined to one service directory. |
| `packet-implementer`, `packet-verifier`, `dispatcher-family-implementer` | wide set incl. MCP | **Keep wide, state why inline** (FR-1.3): these drive IDA through MCP. |

The three packet agents are the ones that genuinely need breadth. Everything else
is a file-and-shell agent, and `atlas-implementer` — the single largest consumer,
6,231 turns — is squarely in the narrow group.

### Why this is the right mechanism

The alternative is trimming prose (agent definitions, `CLAUDE.md`). Task-233
already took that path and hit the floor: prose is ~5.3k of the 37.3k. Tool
schemas are the larger term and the only one still addressable from the repo.

### Risk

An agent denied a tool it actually needs fails mid-run — which is a *loud* failure
(the tool call is refused), not a silent one. Mitigation is to derive each set
from the definition body rather than guessing, and to run one live dispatch per
restricted agent before considering the change done.

### Measurement (FR-5.2)

Median subagent starting context before: **37.3k**. After: re-measure across a
dispatch of each restricted agent. Target < 28k. This is a direct read of the
first assistant turn's usage in the subagent transcript, so it needs no new
tooling beyond FR-5.

---

## Lever 2 (FR-2) — Codemod for mechanical sweeps

### The evidence for what "mechanical" means

From task-232's batch 4 (`6a06ffae0`), the per-call-site transformation was:

1. `requests.RootUrl(` → `requests.RootUrlFor(ctx, ` — signature change
2. add `"context"` to the import block
3. thread `ctx` through the URL-builder function and its callers
4. `main.go`: add `service.WithEnvironmentRegistry(serviceName)` to `Bootstrap`
5. every Kafka consumer's `SetHeaderParsers` gains `consumer.EnvHeaderParser`,
   ordered after `TenantHeaderParser`
6. **error propagation**: the URL builder now returns `(string, error)`, so each
   caller gains an `if err != nil` block with a service-appropriate log message

Steps 1, 2, 4, 5 are pure AST rewrites. Step 3 is AST-derivable (call-graph
walk). **Step 6 is not** — the log message is judgment, and the audit's sample
diff shows exactly that: `"Unable to resolve base URL for character [%d]'s pets."`
is written for its call site.

So the split is roughly: mechanical for the signature/import/wiring, judgment for
the error message. That is the split FR-2.2 formalizes — rewrite what is
derivable, **list** what is not.

### Approach

A Go module at `tools/<name>/`, mirroring `tools/rediskeyguard/`
(`analyzer.go`, `analyzer_test.go`, `cmd/`, `testdata/`, own `go.mod`) so it
inherits the repo's existing analyzer conventions and can be wired into
`tools/go-analyzer-guards.sh` later.

Two modes:
- **rewrite** — applies the mechanical transformation, writes a residue file
  listing every call site it declined to complete, with file:line and reason.
- **`--check`** — exits non-zero if any un-migrated site remains. This is what
  makes it a guard after the migration lands (FR-2.3), replacing the
  hand-maintained scopeguard allowlists that caused two separate gate failures in
  task-232.

### The open question this design cannot settle

**Q3 in the PRD is real and blocking for this lever only.** Task-232's wiring
sweep is *finished* — there are no remaining call sites. Writing the codemod now
means writing it against `testdata/` reconstructed from git history, with no live
work to validate it. Two honest options:

- **(a) Build it retrospectively** against the task-232 diffs as testdata. Proves
  the concept, gives the repo a reusable AST-rewriter skeleton, but the first real
  use is speculative and it may not fit the next sweep's shape.
- **(b) Defer the rewriter, land the skeleton + the decision rule.** Ship
  `tools/<name>/` as a documented pattern plus a rule in the plan/execute
  commands: *before dispatching more than N implementers at one templated
  transformation, write the codemod instead.* The next sweep-shaped task builds
  the real thing against live work.

**Recommendation: (b).** The measured waste is not "we lacked a rewriter" — it is
"nobody asked whether a rewriter was cheaper than 6,231 agent turns." A decision
rule at the point of dispatch captures that; a speculative rewriter might not fit
and would itself cost inference to build. This needs the user's call before Lever 2
proceeds — see the plan's Task 4.

---

## Lever 3 (FR-3) — Controller context ceiling

### Current state

`~/.claude/CLAUDE.md` already carries the correct rule ("At every durable
boundary… does the next unit of work depend materially on this conversation's
history, or only on repository state?"). It fired correctly in task-232 —
`854e6e87` wrote HANDOFF #10 — and was then worked past, because the rule's own
clause ("an agent cannot clear itself") makes the reset contingent on a human
choosing it mid-run.

Measured cost of that single declined reset: the marker lands at turn 138/164 with
context at **243k**; the 26 turns after it cost **6.73M at an average of 259k per
turn**, and bought exactly one plan task (Task 37 batch 5) done correctly. A fresh
controller resuming from `progress.md` produces the same three commits for ≈1.7M.

Corroborating, across all 17 sessions: **`last context == peak context` in every
single one.** No session ever finished a unit of work and stopped while still
small.

### Approach

Amend `.claude/commands/execute-task.md` at the gate-reconcile step (Step 4c) with
a ceiling whose operative clause is *do not continue past a handoff you have
written*. The threshold question (Q2) resolves in favor of **both**: a token
threshold as the primary trigger, since it is what actually drives cost, with
"after N completed plan tasks" as a fallback for a controller that cannot read its
own context size.

This is the cheapest lever in the task — a prose edit to one command file — and it
is worth landing first.

### Why not automate the reset

Because the harness genuinely does not let an agent clear itself. The design
therefore targets the *reachable* behavior: the controller must **stop**, not
continue, once it has written the handoff. Stopping is fully within the agent's
control; clearing is not.

---

## Lever 4 (FR-4) — Review-agent right-sizing

Review and audit subagents accounted for **2,616 turns / 227M / 17.6%** of
task-232. Several returned "0 Critical, 0 Important, 0 Minor" — e.g. the Task 4
review, 521k billed input for 1.4k output tokens. One re-review agent produced
**15 output tokens for 402k billed input**.

But the audit also found review chains that *were* load-bearing, and one review
found a real false-PASS hazard in the analyzer guard cache key. So this is not a
"delete the reviews" lever.

The design position: **gate the review on the nature of the change, not on the
existence of a plan task.** A batch whose transformation was applied by a codemod
and confirmed by `--check` has a machine-checkable correctness argument already;
the review agent adds little. A batch with hand-written judgment (the error
messages of Lever 2, step 6) needs the review.

This lever depends on Lever 2 existing to define "mechanically transformed," so it
sequences last and may reduce to a documentation change if Lever 2 lands as
option (b).

---

## Lever 5 (FR-5) — Measurement harness

`~/.claude/tools/session-digest.sh` already provides totals, context growth, the
tool ledger, and the subagent roster, and was sufficient for the entire audit. The
gap is narrow: it reports cache-read at turn 1 rather than the full starting
context (cache-read + cache-creation + input), which understates the prefix — the
audit initially read 23k where the true figure was 59k.

So FR-5 is a small addition, not a new tool: report **starting context** and
**median subagent starting context** per session. Everything else the audit needed
was already there.

## Sequencing

```
FR-3 (prose, one file)  ──┐
FR-1 (12 frontmatter edits + validation dispatches) ──┤── independent, land first
FR-5 (small digest addition) ──┘
                              │
FR-2 (needs the Q3 decision) ─┴─> FR-4 (depends on FR-2's definition)
```
