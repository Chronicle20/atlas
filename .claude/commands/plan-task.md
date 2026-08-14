---
description: Phase 3 — invoke superpowers:writing-plans to produce an implementation plan inside the task worktree
argument-hint: Task identifier — accepts "task-054-effect-duration-units", "task-054", "054", or "54"
---

You are starting Phase 3 of the Atlas four-phase development workflow. Argument: **$ARGUMENTS**

## Process

### Step 1 — Resolve the task

Same fuzzy-match algorithm as `/design-task` Step 1:

1. Glob `docs/tasks/task-*` (main) and `.worktrees/*/docs/tasks/task-*` (sibling worktrees).
2. Match `$ARGUMENTS` against folder names — exact, number-only (`54`/`054`/`task-54`/`task-054`), or slug fragment.
3. Zero matches → ask for correction. Multiple matches → list and let the user pick.
4. If the task lives only on main with no worktree, stop and tell the user the task needs a worktree.
5. Resolve to `<worktree>/docs/tasks/<id>/`.

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
(`scripts/task-brief` is an awk slice of the `## Task N` heading and its
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

Note in `context.md` any task you deliberately left large, and why.

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
