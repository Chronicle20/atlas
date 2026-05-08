# Atlas

## Project Overview

This is a Go microservices game server project with 14+ services. The primary language is Go. TypeScript is used only for atlas-ui.

## Workflow Rules

When asked to understand or plan something, DO NOT start implementing code changes. Wait for explicit approval before making any edits. Planning and implementation are separate phases.

## Build & Verification

Before claiming a branch is "done," "ready for PR," or invoking `superpowers:finishing-a-development-branch`, verify the affected services this way:

1. `go test -race ./...` clean in every changed module.
2. `go vet ./...` clean in every changed module.
3. `go build ./...` clean in every changed service.
4. **`docker build -f services/<svc>/Dockerfile .` from the worktree root for every service whose `go.mod` or `Dockerfile` was touched.** This is mandatory, not optional. Each service's Dockerfile maintains a hand-edited list of `Chronicle20/atlas/libs/atlas-*` libs in four places (the go.mod stage `COPY`s, the synthesized `go.work use(...)` block, the source `COPY`s, and the explicit `go mod edit -replace=...` flags). Adding a new lib dependency requires updating all four locations, and `go build`/`go test` against the workspace `go.work` will NOT catch the drift — only `docker build` will. CI catches it too, but each round-trip wastes a CI cycle and turns "verified" into a lie.

For large refactors expect multiple fix-and-rebuild cycles. Don't shortcut the Docker step.

## Code Patterns

When refactoring shared types or creating common libraries, prefer straightforward moves over re-exporting type aliases. Keep abstractions clean — don't break service boundaries by having one layer call another's internals directly.

Before defining a new domain type, alias, or numeric constant in a service, check `libs/atlas-constants/` (see its README package index) for an existing equivalent. The shared library already covers item ids/classifications, inventory/compartment types, weapon types, world/channel/map/character ids, jobs, skills, and monster ids — services should use those types directly rather than reinventing them. The `backend-guidelines-reviewer` agent enforces this as DOM-21.

## Development Workflow

The canonical flow for any non-trivial change is four phases. **`/spec-task` creates a dedicated worktree at `.worktrees/task-NNN-slug/` on a `task-NNN-slug` branch; all subsequent phases run inside that worktree** so docs, code, and the eventual PR are one unit. Each phase is a separate slash command, invoked from a fresh (`/clear`'d) session so the next phase consumes only the prior phase's documented artifacts:

1. `/spec-task <idea>` — run from the main repo. Interactive PRD interview that creates the worktree + branch and commits the PRD. Output: `<worktree>/docs/tasks/task-NNN-slug/prd.md`.
2. `cd .worktrees/task-NNN-slug`, `/clear`, then `/design-task <task-id>` — invokes `superpowers:brainstorming`. Output: `design.md` (committed on the task branch).
3. `/clear`, then `/plan-task <task-id>` — invokes `superpowers:writing-plans`. Output: `plan.md` + `context.md` (committed).
4. `/clear`, then `/execute-task <task-id>` — invokes `superpowers:subagent-driven-development`. Reuses the existing worktree; never creates a new one.

Phase commands accept fuzzy task identifiers: `task-054-slug`, `task-054`, `054`, or `54` all resolve to the same folder. They search both `docs/tasks/` (main) and `.worktrees/*/docs/tasks/` to locate the task.

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

## Design/Plan Output Style

- When producing design.md or plan.md documents, write the full document directly to the file. Do NOT walk through sections interactively or ask for per-section approval. The user will read the committed file.

## Worktree Discipline

- Tasks live in git worktrees (often siblings of the main repo). Before planning/designing/executing a task, verify cwd is the correct worktree; if not, cd into it yourself rather than asking the user.
- When searching for task PRDs/plans/designs, search across all worktrees (`git worktree list`) before concluding a file is missing.
- Never edit files in the main repo when a task worktree exists for that work.

## Code Review Before PR

- Always run the code-review step before opening a PR. Do not skip even when the task plan looks complete.

## Verification Over Memory

- For game data values (props, item IDs, skill effects, WZ data), always verify against local WZ data or repo source. Do not cite values from general MapleStory knowledge or memory.
- When uncertain about packet encoding, protocol details, or service ownership, read the source rather than speculating.

## Test Helper Pattern

- Use the project's Builder pattern for test setup. Do not create `*_testhelpers.go` files with test-only constructors.
