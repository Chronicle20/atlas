# Superpowers Integration — When to Use What

This document is the quick-reference companion to `CLAUDE.md`. It tells you which command, agent, or skill to reach for in each situation. The full architectural design lives in `docs/tasks/task-016-superpowers-integration/design.md`.

This document owns *which* command, agent, or skill to reach for in a given
situation. `docs/agent-dispatch.md` owns *how* to dispatch any agent — model,
budget, isolation, handoff.

## The Four-Phase Workflow

| Phase | Command | What it does | Output |
|---|---|---|---|
| 1. Requirements | `/spec-task <idea>` | Interactive PRD interview | `docs/tasks/task-NNN-slug/prd.md` |
| 2. Design | `/design-task <task-folder>` | Architecture, alternatives, tradeoffs | `design.md` |
| 3. Plan | `/plan-task <task-folder>` | Bite-sized TDD step-by-step plan | `plan.md` + `context.md` |
| 4. Execute | `/execute-task <task-folder>` | Subagent-driven implementation | code + commits |
| 5. After implementation | `/fix-pr-bug <task> <bug-slug>` | PR validation, live testing, debugging, follow-up fixes — diagnosis to a durable file, fix in a fresh context | `bug-<slug>.md` + fix commits |

Run `/clear` between phases 1–4. Each command consumes only the prior phase's documented artifacts.

Phase 5 is not a `/clear` boundary and does not require one: it hands off by
writing a diagnosis and dispatching against it. It exists because the flow used
to stop at phase 4 while the work did not — one measured task spent 12.7% of its
entire budget after the PR opened, at 94% main thread with three subagents across
four sessions. See [`docs/post-implementation.md`](post-implementation.md).

### Task resolution

Phase commands accept fuzzy task identifiers: `task-054-slug`, `task-054`,
`054`, and `54` all resolve to the same folder.

- `tools/task-resolve.sh <identifier>` prints one tab-separated line:
  `<task-id>\t<task-dir>\t<worktree>`. Exit 3 means no match; exit 4 means
  ambiguous, with candidates on stderr.
- `tools/task-resolve.sh --list` shows every existing task, one row per task,
  already deduplicated across worktrees.
- `tools/task-numbers.sh next` picks the number for a new task; check both
  before planning so a number does not collide with an in-flight task.
- **Never glob `.worktrees/*/docs/tasks/task-*`** — every worktree carries a
  full copy of `docs/tasks/` from its branch point, so that pattern returns
  (tasks × worktrees) mostly-duplicate paths into context to resolve a single
  id.
- Searching for a task artifact: search across all worktrees
  (`git worktree list`) before concluding a file is missing.

### Artifact location override

`superpowers:brainstorming` and `superpowers:writing-plans` default to
`docs/superpowers/specs/` and `docs/superpowers/plans/`. In this project both
go under `docs/tasks/task-NNN-slug/` instead. When invoking those skills
directly, outside the phase commands, pass the task folder explicitly so
artifacts land in the right place.

### Phase 4 context budget

`atlas-implementer` replaces `general-purpose` for every Phase 4
implementation dispatch. Its contracts override the plugin's
`implementer-prompt.md` where they disagree.

The controls that keep implementer contexts small — the tool-call cap, the
verification split, and the front-loaded file inventory — are owned by
[`docs/agent-dispatch.md`](agent-dispatch.md).

## Code Review

Invoke `superpowers:requesting-code-review` after completing a logical chunk of work. The skill dispatches the relevant subset of these agents in parallel:

- `plan-adherence-reviewer` — checks every task in `plan.md` was implemented; cites file:line evidence
- `backend-guidelines-reviewer` — adversarial Go audit against the applicable families in `.claude/skills/backend-dev-guidelines/resources/audit-checklist.md` (DOM-*, FILE-*, SUB-*, EXT-*, SCAFFOLD-*, SEC-*)
- `frontend-guidelines-reviewer` — adversarial TS/React audit (FE-* checks)
- `atlas-reviewer` — per-unit / ad-hoc correctness review of one commit range against its brief. This is the named home for what used to ride bare `general-purpose`; use it rather than dispatching `general-purpose` with a review prompt.

For ad-hoc one-off checks, invoke any agent directly by name without the orchestration skill.

### Picking the roster — do not derive it by hand

Run the classifier and read the roster off it:

```sh
tools/change-surfaces.sh --base <merge-base-or-last-gated-commit>
```

- `go_changed=true` → `backend-guidelines-reviewer`
- `frontend_review=true` → `frontend-guidelines-reviewer`
- `packet_surface=true` → also `packet-completeness-critic` (packet tasks)
- a `plan.md` exists → `plan-adherence-reviewer`

Pass the whole block verbatim into each reviewer's dispatch brief. It gives the
backend reviewer its `backend_audit_families` list up front, replacing the
13.6 KB `git diff --stat` pair one measured reviewer opened with — carried
through all 83 of its turns — plus ~12 later turns spent rediscovering whether a
Dockerfile exists and whether topic env vars changed.

**The block is additive and fails open.** It states the families that are
*definitely* in scope. A reviewer may add a family; a reviewer may **not** drop
one because the block omitted it. When the classifier cannot understand the
change — an unresolvable base, a Go file in an unrecognised layout — it emits
`classification=uncertain` with every family listed, and the review runs wide.

### What a reviewer returns

Every reviewer writes its full reasoning to a durable artifact and returns a
compact verdict-first block. The contract, the verdict semantics, and the
controller's read rule are in [`docs/review-protocol.md`](review-protocol.md).
Short version: `verdict` is the first line, blocking findings are enumerated
with `file:line`, everything else is a count, and the controller opens the
artifact only when the verdict is not `APPROVED`.

All three are **scoped to the change under review**: the diff is the review surface, repo surveying is off, and anything a reviewer could not evaluate within that surface is reported under `## Not evaluable from the diff` rather than passed silently. Each agent's own `## Scope` section is the contract — you do not need to restate it in the dispatch prompt. Measured on a 67-file Go diff, this cost the same as an unscoped review and returned a strict superset of its findings.

Each agent writes findings to `docs/tasks/task-NNN-slug/audit.md` (backend
also writes `audit.json`).

Code review is mandatory before opening a PR and is a **different gate**
from verification: a green `tools/verify.sh` does not mean the branch is
correct. Every module can build, vet, test, and bake clean while the branch
carries blocking defects, because each service is self-consistent in
isolation. The gate cannot see: a producer emptying a compartment the
consumer still reads; a new saga action with no step handler in the
orchestrator; a class of missing emits that existing tests actively pin as
the old behavior. When a change crosses a service boundary, trace the event
into its consumers by hand and check that a test asserts the NEW contract,
not the old silent drop.

## Maintenance Commands

| Command | What it does | Underlying agent |
|---|---|---|
| `/review-todos` | Whole-codebase TODO/FIXME scan; updates `docs/TODO.md` | `todo-scanner` |
| `/service-doc <service>` | Generates/updates documentation for one service | `service-documentation` |
| `/convert-map` | Convert map entry JavaScript script to JSON rules format | (direct command) |
| `/convert-npc` | Convert NPC conversation JavaScript script to JSON state machine format | (direct command) |
| `/convert-portal` | Convert portal JavaScript script to JSON rules format | (direct command) |
| `/convert-quest` | Convert quest conversation JavaScript script to JSON state machine format | (direct command) |
| `/convert-reactor` | Convert reactor JavaScript script to JSON rules format | (direct command) |

## Packet Work

Packet-audit work has ONE canonical playbook per task type and an executable entry point that drives it. Start at [`docs/packets/PROCESS.md`](packets/PROCESS.md) — the source of truth for the version set, baseline status, CI gates, and the task-type → entry-point → playbook table. Do not restate a playbook's procedure in prose elsewhere — link to it.

**Before opening a packet-task PR**, run the `packet-completeness-critic` agent alongside the guideline reviewers. It is the packet-specific review companion: it diffs the task's `docs/tasks/<task>/coverage-manifest.yaml` (schema in [`PROCESS.md`](packets/PROCESS.md)) against the branch's git + matrix delta and flags **CHANGED-BUT-UNCLAIMED** (a codec/gate moved but the task never declared it — the class-8 scope hole) and **CLAIMED-BUT-UNVERIFIED** (a manifest op×version with no verified cell). Read-only; writes `completeness-critic.md`.

## Domain Skills

These activate via the project hook (`skill-activation-prompt.py`) when you mention relevant keywords or work on relevant files:

- `backend-dev-guidelines` — Go service patterns
- `frontend-dev-guidelines` — React/TypeScript patterns

The hook produces a visible "🎯 SKILL ACTIVATION CHECK" banner. Heed it before responding.

## Superpowers Skills (Self-Activating)

Reach for these explicitly when relevant; they also self-activate via Claude's native skill matching:

- `using-superpowers` — invoke at the start of any conversation
- `brainstorming` — used inside `/design-task`
- `writing-plans` — used inside `/plan-task`
- `subagent-driven-development` — used inside `/execute-task`
- `executing-plans` — fallback for inline execution
- `systematic-debugging` — for any bug, test failure, or unexpected behavior
- `test-driven-development` — when implementing any feature or bugfix
- `verification-before-completion` — before claiming work is complete
- `using-git-worktrees` — for isolated workspaces
- `finishing-a-development-branch` — when implementation is complete and tests pass
- `requesting-code-review` — used at the end of a chunk of work
- `receiving-code-review` — when processing review feedback
- `dispatching-parallel-agents` — used by code-review orchestration
- `writing-skills` — when authoring new skills

## When NOT to Use Superpowers

- **Trivial fixes** (typo, version bump, one-line change) — no workflow needed; commit directly.
- **Documentation-only updates** that don't need a PRD — go straight to editing.
- **Domain script conversion** — use the appropriate `/convert-*` command directly (no workflow overhead).

## File Locations Cheat Sheet

| Artifact | Location |
|---|---|
| PRD, design, plan, context, audit | `docs/tasks/task-NNN-slug/` |
| Audit JSON output (backend) | `docs/tasks/task-NNN-slug/audit.json` |
| Per-service docs | `services/<service>/docs/` |
| TODO list | `docs/TODO.md` |
| atlas-ui frontend | `services/atlas-ui/` |
