---
description: Phase 4 — invoke superpowers:subagent-driven-development to implement a planned task in its existing worktree
argument-hint: Task identifier — accepts "task-054-effect-duration-units", "task-054", "054", or "54"
---

You are starting Phase 4 of the Atlas four-phase development workflow. Argument: **$ARGUMENTS**

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

Do NOT create a new worktree — the worktree was created by `/spec-task` and must be reused so phase artifacts stay co-located.

### Step 3 — Validate inputs

Confirm `<worktree>/docs/tasks/<id>/plan.md` AND `context.md` exist. If either is missing, tell the user to complete `/plan-task` first.

### Step 4 — Invoke subagent-driven-development

Use the Skill tool to invoke `superpowers:subagent-driven-development` (default). Pass:

- Plan path: `<worktree>/docs/tasks/<id>/plan.md`
- Context path: `<worktree>/docs/tasks/<id>/context.md`
- Project conventions: `<worktree>/CLAUDE.md`
- **Worktree absolute path** (`<worktree>`) for every dispatched implementer subagent. Subagent prompts MUST follow the cwd-discipline template from memory `feedback_subagent_worktree_cwd.md` — every Bash call prefixed with `cd <worktree> && ...`, post-commit branch verification, no destructive git ops, no `git add -A` / `git add .`.

### Step 4a — Model discipline for every dispatch

Pass an explicit `model` on **every** Agent/Task dispatch. Never rely on
inheritance: an unspecified model inherits the main-loop model (Opus), and an
Opus subagent turn costs ~7x a Sonnet one.

The pin is chosen by the **job the agent is doing**, not by its
`subagent_type`. Named reviewer agents carry a Sonnet pin in their frontmatter,
but an ad-hoc `general-purpose` dispatch carrying a review prompt does not —
that is the hole this rule closes.

| Job | Model | Notes |
|---|---|---|
| Review, verify, audit, re-review, whole-branch review | **`sonnet`** — always | No exceptions. Reviewing is reading against a checklist; Opus buys nothing and these run long |
| Scan, inventory, doc sweep, file-finding | `haiku` | |
| Implement a plan task | `sonnet` | Default |
| Implement a plan task tagged `model: opus` in plan.md | `opus` | Opt-in only — see below |

A plan task may be tagged `model: opus` in `plan.md` when it is genuinely
derivation-heavy: IDA/packet field-order derivation, saga orchestration across
services, or a cross-service contract change. `/plan-task` should apply that tag
sparingly and justify it in one line. Everything else — REST surfaces, GORM
entities, Kafka consumers, tests, template routing — runs Sonnet.

If an implementer comes back wrong twice on Sonnet, escalate that one task to
Opus and note it, rather than raising the default.

If the user explicitly requests inline mode this session (rare), invoke `superpowers:executing-plans` instead.

### Step 5 — On completion

After all plan tasks complete and verify, the chosen skill hands off to `superpowers:finishing-a-development-branch`. Honor that handoff. Then suggest:

> All plan tasks complete. Recommend running `superpowers:requesting-code-review` next, which dispatches the appropriate reviewer agents (plan-adherence, backend-guidelines, frontend-guidelines) in parallel.

## Important Rules

- The worktree was created by `/spec-task`. NEVER create a new one here.
- Never start implementation outside the task worktree.
- Follow plan steps exactly; stop and ask when blocked rather than guessing.
- Run the verification commands the plan specifies; don't claim completion based on assumption.
