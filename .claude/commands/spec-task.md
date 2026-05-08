---
description: Phase 1 — interview a backlog idea into a PRD inside a fresh task worktree
argument-hint: Brief description of the feature idea (e.g., "recurring reminders")
---

You are a product-minded engineer turning a rough backlog idea into a structured feature spec. The idea is: **$ARGUMENTS**

## Process

### Step 1 — Refuse if already in a task worktree

Run `git rev-parse --show-toplevel` and `pwd`. If the result is under `.worktrees/`, stop and tell the user:

> You're already inside a task worktree. `/spec-task` must be run from the main repo (`/home/tumidanski/source/atlas-ms/atlas`). `cd` there and re-run.

### Step 2 — Determine task number and working slug

1. Run `tools/task-numbers.sh next`. The number it prints (zero-padded, three digits) is the next free `NNN`. Do NOT pick a number any other way — that script is the only collision-safe source: it scans `docs/tasks/`, every `.worktrees/<wt>/docs/tasks/<wt>/`, and every local `task-*` git branch in one pass. Picking by hand has caused collisions before (May 2026: two tasks both numbered 063).
2. Derive a working slug from `$ARGUMENTS` (lowercase, hyphenated, 3–4 words). Examples: "recurring reminders" → `recurring-reminders`, "household invitations" → `household-invitations`.
3. Compose the task identifier: `task-NNN-<slug>`.

### Step 3 — Lightweight context scan

Before creating anything, gather just enough context to ask intelligent questions:

1. Read project docs in `docs/` and relevant service docs that the idea touches.
2. Check `docs/TODO.md` for related items.
3. Scan existing task folder names (don't read full PRDs unless one looks directly related).
4. Identify which services would be affected.

### Step 4 — Confirm scope before creating the worktree

Present to the user:

1. **Proposed task ID:** `task-NNN-<slug>` (note: slug can be renamed at the end of the interview if scope shifts)
2. **Scope summary** — 2-3 sentences
3. **Key questions** — 3-7 ambiguities or decisions
4. **Proposed boundaries** — in scope vs. out of scope
5. **Affected services**

Wait for the user to answer. They may also propose a different slug at this point.

### Step 5 — Create the worktree

Once scope and slug are confirmed, invoke `superpowers:using-git-worktrees` to create:

- Branch: `task-NNN-<slug>` (from `main`)
- Worktree: `.worktrees/task-NNN-<slug>` (relative to main repo root)

From this point on, every file write, read, and shell command MUST use absolute paths under the worktree (e.g., `/home/tumidanski/source/atlas-ms/atlas/.worktrees/task-NNN-<slug>/docs/tasks/task-NNN-<slug>/prd.md`). Do NOT write any files under the main repo's `docs/tasks/`.

### Step 6 — Generate the PRD inside the worktree

Create `<worktree>/docs/tasks/task-NNN-<slug>/prd.md` using this structure:

```markdown
# [Feature Name] — Product Requirements Document

Version: v1
Status: Draft
Created: YYYY-MM-DD
---

## 1. Overview

[What this feature is and why it matters — 2-3 paragraphs]

## 2. Goals

Primary goals:
- [list]

Non-goals:
- [list]

## 3. User Stories

- As a [role], I want to [action] so that [outcome]
- [repeat]

## 4. Functional Requirements

[Organized by capability area. Be specific and testable.]

## 5. API Surface

[New or modified endpoints, request/response shapes, error cases]

## 6. Data Model

[New entities, fields, relationships, constraints, migration notes]

## 7. Service Impact

[Which services are affected and what changes in each]

## 8. Non-Functional Requirements

[Performance, security, observability, multi-tenancy considerations]

## 9. Open Questions

[Anything still unresolved after the conversation]

## 10. Acceptance Criteria

[Concrete checklist of what "done" looks like]
```

Optionally create supporting files in the same folder if they add value: `api-contracts.md`, `data-model.md`, `migration-plan.md`, `ux-flow.md`, `risks.md`. Don't create empty ones.

### Step 7 — Commit on the task branch

From within the worktree, commit the new files on the `task-NNN-<slug>` branch:

```
git add docs/tasks/task-NNN-<slug>/
git commit -m "spec(task-NNN): initial PRD for <slug>"
```

Verify post-commit:

```
git rev-parse --show-toplevel  # must end with /.worktrees/task-NNN-<slug>
git branch --show-current      # must be task-NNN-<slug>
```

If either is wrong, STOP and report BLOCKED — do NOT attempt destructive recovery.

### Step 8 — Summary and next step

Tell the user:

> Worktree created at `.worktrees/task-NNN-<slug>` on branch `task-NNN-<slug>`. PRD committed.
> Next: `cd .worktrees/task-NNN-<slug>`, then `/clear`, then `/design-task task-NNN-<slug>`.

## Quality Standards

- Requirements must be specific and testable — avoid vague language like "should be fast".
- Respect existing architectural patterns from project/service docs and the codebase.
- API designs follow JSON:API conventions; data models include `tenant_id` scoping.
- Keep the PRD self-contained — a developer should implement from it without clarifying questions.
- Never write task artifacts under main's `docs/tasks/` — they belong on the task branch in the worktree.
