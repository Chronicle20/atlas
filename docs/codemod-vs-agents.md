# Codemod vs. agents — when a templated transformation earns a tool

This doc answers one question at dispatch time: **you are about to send
several implementers at the same mechanical, repeated change — should one of
them write a codemod instead?** It exists because task-232 answered "no"
without ever asking: a single templated transformation (batch 4, task-232)
consumed 6,231 implementer turns / 760M billed input tokens / 59% of the
session's total spend, and nobody evaluated whether an AST rewrite would have
been cheaper.

This document is the rule, the worked example, and the specification for the
rewriter that would apply it — the rewriter itself is **not built yet** (see
"Current status" below).

## The rule

> Before dispatching more than **2 implementers** at the same templated
> transformation, evaluate whether an AST codemod is cheaper than the
> remaining manual dispatches.

"Templated transformation" means: the same multi-step edit, repeated across
call sites/files/services, where most steps are syntactic (a rename, an
added import, a threaded parameter, a fixed call inserted at a fixed
location) and at most one step requires per-site judgment (a log message, a
comment, a domain-specific choice).

### The arithmetic that sets N = 2

Two figures are measured, from task-232's batch 4:

- **6,231 implementer turns cost 760M billed input tokens** → 760,000,000 /
  6,231 ≈ **122,010 tokens per turn** (the brief's own "~122k per turn"
  figure — this recomputation is a consistency check, not a new figure).

One figure is a standing contract, not a measurement: an `atlas-implementer`
dispatch is capped at **120 tool calls** before it must hand back `PARTIAL`
(`docs/agent-dispatch.md` "The implementer budget"). Writing a codemod is
itself exactly this shape of task — a small, self-contained Go module
mirroring `tools/rediskeyguard/` (four or five files: `go.mod`, `analyzer.go`,
`analyzer_test.go`, `cmd/`, `testdata/`) — so its cost ceiling is bounded by
that same budget: at most 120 turns × ~122,010 tokens/turn ≈ **14.6M tokens**
to build and test, worst case.

Two implementer dispatches manually applying the same templated
transformation therefore have a cost ceiling of 2 × 120 × 122,010 ≈ **29.3M
tokens** — a full budget's worth each, worst case. That is the break-even
point: past it, a single bounded dispatch to write the codemod (≤14.6M
tokens, once) costs less than the *manual* dispatches it replaces, and every
site beyond what the codemod covers is a `--check`-verified mechanical
rewrite instead of another implementer turn.

**N = 2** is therefore the threshold: before a third implementer would be
dispatched at the same templated transformation, stop and evaluate the
codemod option.

Measured against what actually happened: batch 4's 6,231 turns / 760M tokens
is **~26×** the 29.3M-token ceiling that two implementer-equivalents of
codemod-writing would have cost (760M / 29.3M ≈ 25.9). The rule would have
fired at turn budget #241 (the start of the third implementer's worth of
work); the session ran roughly 26 implementer-budgets past that point before
anyone asked the question.

## Worked example: task-232 batch 4

Source: `docs/tasks/task-234-inference-efficiency-levers/design.md` Lever 2
(FR-2), citing commits `6a06ffae0`, `54e7e0c3d`, `8776709b8`.

The per-call-site transformation was six steps:

1. `requests.RootUrl(` → `requests.RootUrlFor(ctx, ` — signature change —
   **AST**
2. add `"context"` to the import block — **AST**
3. thread `ctx` through the URL-builder function and its callers — **AST**
   (call-graph walk, AST-derivable)
4. `main.go`: add `service.WithEnvironmentRegistry(serviceName)` to
   `Bootstrap` — **AST**
5. every Kafka consumer's `SetHeaderParsers` gains `consumer.EnvHeaderParser`,
   ordered after `TenantHeaderParser` — **AST**
6. error propagation: the URL builder now returns `(string, error)`, so each
   caller gains an `if err != nil` block with a service-appropriate log
   message — **judgment**. The audit's sample diff shows exactly why:
   `"Unable to resolve base URL for character [%d]'s pets."` is written for
   its call site.

Four of six steps are pure AST rewrites, one (step 3) is AST-derivable via a
call-graph walk, and one (step 6) is irreducibly judgment. That is the split
FR-2.2 formalizes: **rewrite what is derivable, list what is not, and never
silently skip a site.** A codemod covering steps 1-5 would have turned every
site's remaining work into "confirm/write one log message," reviewable from a
residue list rather than dispatched as a full implementer turn per site.

## The deferred rewriter's contract (specification only)

If a templated transformation clears the N = 2 threshold, the rewriter that
gets written should follow this shape — this is a specification for future
work, not a description of an existing tool. Nothing under `tools/` currently
implements it.

**Module layout**, mirroring `tools/rediskeyguard/`:

- `tools/<name>/go.mod` — own module, same as `rediskeyguard`
- `tools/<name>/analyzer.go` — the AST rewrite logic
- `tools/<name>/analyzer_test.go` — table-driven tests over `testdata/`
- `tools/<name>/cmd/` — the CLI entry point
- `tools/<name>/testdata/` — before/after fixture pairs, built from the real
  diffs at `6a06ffae0`, `54e7e0c3d`, `8776709b8`

**Two contracts it must honor:**

- **FR-2.2 — every site is rewritten or listed, never silently skipped.**
  A site the tool cannot safely rewrite (the judgment step, or any pattern
  it doesn't recognize) goes into a residue report with file:line and reason.
  Silent omission is the failure mode that makes a codemod untrustworthy —
  a human has to be able to trust that "not in the residue list" means
  "rewritten," not "not looked at."
- **FR-2.3 — `--check` mode for use as a guard afterward.** The same
  analyzer, run in a mode that exits non-zero if any un-migrated site
  remains, becomes the regression guard once the migration lands — replacing
  hand-maintained allowlists (the kind that caused two separate gate failures
  in task-232) with a mechanical check.

## Current status — dormant

**No rewriter exists yet.** This document specifies what one would look like
and the threshold at which writing one pays for itself; it does not claim one
is available to run. Because no `--check` mode exists, no batch today can be
verified as codemod-applied — see the review-step guidance in
`.claude/commands/execute-task.md`: until a rewriter with `--check` lands,
every batch is treated as judgment-bearing and gets the full per-task review
agent.
