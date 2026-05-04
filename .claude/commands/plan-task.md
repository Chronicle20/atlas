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

### Step 2 — Verify we're in the right worktree

Run `pwd`. If it does NOT match `<worktree>`, tell the user:

> Task `<id>` lives in `<worktree>`. Please `cd <worktree>` and re-run `/plan-task <id>`.

Stop. Do not proceed from the wrong cwd.

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
