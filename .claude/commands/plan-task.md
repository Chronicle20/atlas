---
description: Phase 3 — invoke superpowers:writing-plans to produce an implementation plan inside the task worktree
argument-hint: Task identifier — accepts "task-054-effect-duration-units", "task-054", "054", or "54"
---

You are starting Phase 3 of the Atlas four-phase development workflow. Argument: **$ARGUMENTS**

## Process

### Step 1 — Resolve the task

Same as `/design-task` Step 1 — run the resolver, do NOT glob for the folder
yourself:

```sh
tools/task-resolve.sh "$ARGUMENTS"
```

It prints one tab-separated line: `<task-id>\t<task-dir>\t<worktree>`, and
accepts exact, number-only (`54`/`054`/`task-54`/`task-054`), or slug-fragment
identifiers.

**Never glob `.worktrees/*/docs/tasks/task-*`** — every worktree carries a full
copy of `docs/tasks/`, so that pattern returns thousands of mostly-duplicate
paths into context to resolve one ID.

Exit codes: **3** → no match, ask for correction. **4** → ambiguous, the
candidates are on stderr; list them and let the user pick. If `<worktree>` is
the main repo root the task has no worktree — stop and tell the user it needs
one.

### Step 2 — Ensure we're in the right worktree

Run `pwd`. If it does NOT match `<worktree>`, `cd <worktree>` yourself and continue from there. Do NOT ask the user to re-run the command — per CLAUDE.md's "Worktree Discipline" rule, cd into the task worktree yourself.

### Step 3 — Validate inputs

1. Confirm both `prd.md` and `design.md` exist. If either is missing, stop and tell the user to complete the prior phase.
2. Confirm `plan.md` does NOT already exist. If it does, ask whether to overwrite.

### Step 4 — Load context

Read:
- `<worktree>/docs/tasks/<id>/prd.md`
- `<worktree>/docs/tasks/<id>/design.md`
- `<worktree>/CLAUDE.md`
- `<worktree>/docs/superpowers-integration.md`
- Code areas the design touches

### Step 5 — Invoke writing-plans

Use the Skill tool to invoke `superpowers:writing-plans`. Pass:

- Spec at `<worktree>/docs/tasks/<id>/design.md` (PRD at `prd.md` for reference).
- Plan output MUST be saved to `<worktree>/docs/tasks/<id>/plan.md`.
- Also produce `<worktree>/docs/tasks/<id>/context.md` summarizing key files, decisions, dependencies.
- Do NOT auto-invoke execution.

Run the `writing-plans` skill's self-review (placeholder scan, type consistency, spec coverage) before saving.

### Step 5a — Atlas plan-task format (required)

Phase 4 extracts each task section verbatim into the implementer's brief
(`tools/task-brief.sh` is an awk slice of the `## Task N` heading and its
body). Whatever you write into a task section IS the brief. Two rules follow
from that.

**Every task carries a `### Files` block.** You are reading the code to write
the plan; the implementer should not have to rediscover what you already
found. Naming the files removes the implementer's discovery phase — the phase
that inflates context before any editing starts.

```markdown
### Files

- `services/atlas-x/atlas.com/x/foo/processor.go` — add the Bar method here
- `services/atlas-x/atlas.com/x/foo/processor_test.go` — new test cases
- `libs/atlas-constants/item/id.go` — read-only; the constant to use

Patterns to copy: `services/atlas-y/atlas.com/y/baz/processor.go:88` (same shape)
```

Paths must be repo-relative and must exist (or be explicitly marked `new
file`). Mark read-only references as such so the implementer does not edit
them. Include the module root the task's `go build`/`go test` runs from when
it is not obvious from the paths.

**Size tasks to the implementer budget.** Implementers stop at 120 tool calls
and hand back `PARTIAL`; the controller then splits the task anyway, at a
worse moment and a higher cost. Split at plan time instead:

- A task touching **more than ~6 files**, or **more than one service**, gets
  split — unless the edits are the same mechanical change repeated, which
  batches fine.
- A task whose acceptance needs a new REST surface *and* a new Kafka consumer
  *and* their tests is three tasks.
- Prefer several small tasks over one large one. The review loop is per task,
  so smaller tasks also mean tighter review surfaces.
- **Before sizing a templated transformation into a second implementer task,
  check whether an AST codemod is cheaper than the manual dispatches it
  replaces** — see [docs/codemod-vs-agents.md](../../docs/codemod-vs-agents.md)
  for the break-even arithmetic and the worked example. This is dormant until
  a rewriter exists (no `tools/` codemod is built yet), but the sizing
  question still belongs at plan time, not discovered mid-execution.

Note in `context.md` any task you deliberately left large, and why.

### Step 5b — Lint the plan (required)

```
tools/plan-lint.sh docs/tasks/<id>/plan.md
```

Must exit 0 before you commit. It checks the rules Step 5a already states, and
which nothing previously enforced:

| | Check | Why |
|---|---|---|
| **F1** | every `### Files` path exists, or is marked `new file` | Step 5a's own rule |
| **F2** | the plan does not specify a stub to land | CLAUDE.md: no `// TODO`, stubbed handlers or 501s |
| **F3** | read-only commands the plan tells the implementer to run actually match something | |
| **F4** | task size (>6 files, or >1 service) — *warning* | Step 5a's splitting rule |
| **F5** | every symbol a ```` ```go ```` block calls resolves — *warning* | the code in the plan is written without a compiler |

F1–F3 are errors; fix them. F4 and F5 are advisory.

F4: a deliberately large task is allowed provided `context.md` says why, but
oversized tasks are what produce `PARTIAL` hand-backs and a mid-plan split at a
worse moment.

F5: a symbol resolves if the repo defines it, the repo calls it anywhere, or the
plan declares it. What survives is either the repo's first use of an external
API — fine, confirm the signature — or a method you invented from memory, which
an implementer will hit at dispatch time. **Grep each one before committing.**
It is advisory rather than an error only because the first case is legitimate;
it is not optional. On task-238 this check was run by hand — roughly thirty tool
calls at 250k context, the most expensive stretch of the plan session, spent
grepping for `newCtxTenant`, `SetCashScene` and `NewModelBuilder` to find out
whether the API just written into the plan exists.

F5 indexes every `.go` file in the tree, so it adds ~5s. `--no-symbols` skips it.

This exists because these defects are cheap here and expensive later. On
task-231 the controller had to append 18 `## CONTROLLER RULING` blocks patching
exactly this class of problem — a `### Files` path that did not exist, a Step-2
command matching nothing, four planned stubs — each discovered at dispatch time
at 150–250k context, most after an investigation it had to run first.

### Step 6 — Commit and summarize

```
git add docs/tasks/<id>/plan.md docs/tasks/<id>/context.md
git commit -m "plan(<id>): implementation plan and context"
```

Verify post-commit:

```
git rev-parse --show-toplevel  # must end with /.worktrees/<id>
git branch --show-current      # must be <id>
```

If either is wrong, STOP and report BLOCKED. Then tell the user:

> Plan and context saved and committed. Now run `/clear`, then `/execute-task <id>`. (You're already in the right worktree.)

## Important Rules

- All file I/O uses absolute paths under `<worktree>`.
- Never write plan artifacts under main's `docs/tasks/`.
- DO NOT begin implementation. This phase produces planning documents only.
