# Superpowers Integration — When to Use What

This document is the quick-reference companion to `CLAUDE.md`. It tells you which command, agent, or skill to reach for in each situation. The full architectural design lives in `docs/tasks/task-016-superpowers-integration/design.md`.

## The Four-Phase Workflow

| Phase | Command | What it does | Output |
|---|---|---|---|
| 1. Requirements | `/spec-task <idea>` | Interactive PRD interview | `docs/tasks/task-NNN-slug/prd.md` |
| 2. Design | `/design-task <task-folder>` | Architecture, alternatives, tradeoffs | `design.md` |
| 3. Plan | `/plan-task <task-folder>` | Bite-sized TDD step-by-step plan | `plan.md` + `context.md` |
| 4. Execute | `/execute-task <task-folder>` | Subagent-driven implementation | code + commits |

Run `/clear` between phases. Each command consumes only the prior phase's documented artifacts.

## Code Review

Invoke `superpowers:requesting-code-review` after completing a logical chunk of work. The skill dispatches the relevant subset of these agents in parallel:

- `plan-adherence-reviewer` — checks every task in `plan.md` was implemented; cites file:line evidence
- `backend-guidelines-reviewer` — adversarial Go audit (DOM-*, SUB-*, SEC-* checks)
- `frontend-guidelines-reviewer` — adversarial TS/React audit (FE-* checks)

For ad-hoc one-off checks, invoke any agent directly by name without the orchestration skill.

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

Packet-audit work has ONE canonical playbook per task type and an executable entry point that drives it. Start at [`docs/packets/PROCESS.md`](packets/PROCESS.md) — the source of truth for the version set, baseline status, and CI gates — then pick your entry point:

| Task type | Entry point | Canonical playbook |
|---|---|---|
| Implement a new feature codec (clientbound or serverbound) | `/implement-packet` command + `packet-implementer` agent | [`IMPLEMENTING_A_PACKET.md`](packets/IMPLEMENTING_A_PACKET.md) |
| Bring up a new client-version column | `/bringup-version` command | [`STARTING_A_NEW_VERSION_PASS.md`](packets/audits/STARTING_A_NEW_VERSION_PASS.md) |
| Audit / implement a mode-prefix dispatcher family | `family-auditor` agent (read-only triage) · `dispatcher-family-implementer` agent (do-mode) | [`DISPATCHER_FAMILY.md`](packets/DISPATCHER_FAMILY.md) |

The leaf step shared by all of the above — promoting one packet × version matrix cell to `✅` — is `/verify-packet` + the `packet-verifier` agent, driving [`VERIFYING_A_PACKET.md`](packets/audits/VERIFYING_A_PACKET.md). Each entry point cites its playbook rather than restating the procedure; keep it that way.

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
