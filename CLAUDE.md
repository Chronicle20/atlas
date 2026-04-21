# Atlas

## Project Overview

This is a Go microservices game server project with 14+ services. The primary language is Go. TypeScript is used only for atlas-ui. Always verify Docker builds when changing shared libraries.

## Workflow Rules

When asked to understand or plan something, DO NOT start implementing code changes. Wait for explicit approval before making any edits. Planning and implementation are separate phases.

## Build & Verification

After making changes across multiple services, always run builds and tests for ALL affected services before reporting completion. Expect multiple fix-and-rebuild cycles for large refactors.

## Code Patterns

When refactoring shared types or creating common libraries, prefer straightforward moves over re-exporting type aliases. Keep abstractions clean — don't break service boundaries by having one layer call another's internals directly.

## Development Workflow

The canonical flow for any non-trivial change is four phases. Each phase is a separate slash command, each invoked from a fresh (`/clear`'d) session so the next phase consumes only the prior phase's documented artifacts:

1. `/spec-task <idea>` — interactive PRD interview. Output: `docs/tasks/task-NNN-slug/prd.md`.
2. `/clear`, then `/design-task <task-folder>` — invokes `superpowers:brainstorming` for architecture/tradeoffs. Output: `design.md` in same folder.
3. `/clear`, then `/plan-task <task-folder>` — invokes `superpowers:writing-plans` for bite-sized TDD steps. Output: `plan.md` + `context.md`.
4. `/clear`, then `/execute-task <task-folder>` — invokes `superpowers:subagent-driven-development` (default) or `superpowers:executing-plans` (fallback).

Skip `/spec-task` only for trivial fixes that don't warrant a PRD; document those directly via a brainstorming session.

### Artifact Location Override

Both `superpowers:brainstorming` and `superpowers:writing-plans` default to `docs/superpowers/specs/` and `docs/superpowers/plans/`. **In this project, both go under `docs/tasks/task-NNN-slug/` instead.** When invoking those skills directly (outside the phase commands), pass the task folder explicitly so artifacts land in the right place.

### Code Review Pattern

Code review uses three modular reviewer agents, dispatched in parallel:

- `plan-adherence-reviewer` — verifies plan tasks were actually implemented
- `backend-guidelines-reviewer` — Go DOM-* checklist (when Go files changed)
- `frontend-guidelines-reviewer` — TS/React FE-* checklist (when atlas-ui TS files changed)

Invoke via `superpowers:requesting-code-review` (it dispatches the appropriate subset), or invoke an individual agent directly for ad-hoc checks. Each agent writes its findings to `docs/tasks/task-NNN-slug/audit.md`.

See `docs/superpowers-integration.md` for a complete when-to-use-what reference.

## Documentation

When updating TODO.md or other tracking docs, always use `Glob` or `Grep` to find the file first rather than assuming a path. Documentation updates should follow the /dev-docs format.
