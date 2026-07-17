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
4. **`docker buildx bake atlas-<svc>` from the worktree root for every service whose `go.mod` was touched.** This is mandatory, not optional. The shared `Dockerfile` at the repo root is parameterized by `ARG SERVICE`; `docker-bake.hcl` enumerates one target per Go service driven by `.github/config/services.json` (single source of truth). `go build`/`go test` against the workspace `go.work` will NOT catch a missing `COPY libs/...` line in the shared Dockerfile — only `docker buildx bake` will. CI catches it too, but each round-trip wastes a CI cycle and turns "verified" into a lie.

To build everything locally: `docker buildx bake all-go-services` (or `tools/build-services.sh` — a thin wrapper).

Adding a new shared lib requires appending two `COPY` lines to the repo-root `Dockerfile` (one in the mod-only block, one in the source block) and one `./libs/<name>` line to `go.work`. That's it — no per-service edits.

For large refactors expect multiple fix-and-rebuild cycles. Don't shortcut the bake step.
5. **`tools/redis-key-guard.sh` clean from the repo root.** Bans keyed Redis
   commands on the raw `go-redis` client outside `libs/atlas-redis` (FR-1.5,
   task-045). Runs alongside `go vet ./...`.
6. **`tools/goroutine-guard.sh` clean from the repo root.** Bans bare `go`
   statements outside `libs/atlas-routine` and justified
   `//goroutine-guard:allow` sites (RR-6, task-115) — every goroutine must be
   spawned via `routine.Go`. Runs alongside `go vet ./...`.
7. **`tools/lint.sh --check` clean from the repo root.** The shared lint &
   format guard (task-171): golangci-lint v2 formatters (gofumpt + goimports,
   tree-wide) and `standard` linters (rev-gated to new code) across every Go
   module, plus Prettier + ESLint for atlas-ui. Fix mode (`tools/lint.sh`,
   no flags) rewrites files in place — run it before committing. Item 2's
   standalone `go vet` is intentionally retained (it runs full-module;
   the guard's govet is diff-gated).

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

## Grounding & Honesty (No Inventing)

- Never invent values, names, opcodes, command output, or behavior. If something is not verified from source, WZ data, IDA, or live output, say "unknown / unverified" — do not fill the gap with a plausible guess.
- When reading tool output (e.g. `top`, pod metrics, decompiled code), quote the actual values before drawing a conclusion. Do not paraphrase numbers from memory or infer them — misreading and then asserting is worse than saying "let me re-check."
- "I think it's X" is not a finding. Either verify X and state it plainly, or flag it as unverified and say what you'd need to confirm it.

## No Deferring Producible Work

- Do not declare a "documented gap," "follow-up task," or "out of scope" when the blocker is a prerequisite you can produce yourself (an unnamed IDB function → name it; an unrouted template → wire it; a missing export/report → generate it). Attempt the unblock before calling it terminal. The user should not have to prod you to finish bounded work.
- Do not split work into a new task to avoid completing the current one. Keep triage and fix on the same branch/worktree; produce the clean PR branch via rebase at PR-time, not by forking partway through.
- No `// TODO`, stubbed handlers, or 501s in landed commits. Finish the bounded work or escalate explicitly — never leave a silent stub.
- Genuine stop-and-ask cases (a true external blocker, an ambiguous design decision, an unresolved packet-audit fname) are different: surface them and ask. The bar is "can I produce this myself right now?" — if yes, do it.

## File Writing / Conventions

- When writing files, always use repo-relative paths or placeholders; never write literal home/absolute paths like `/Users/<name>/...` or `/home/<name>/...` into committed files.

## Packet work

Packet-audit work has ONE canonical playbook per task type and an executable entry point that drives it. Start at [`docs/packets/PROCESS.md`](docs/packets/PROCESS.md) (the source of truth for the version set, baseline status, and CI gates), then pick your entry point:

| Task type | Entry point | Canonical playbook |
|---|---|---|
| Implement a new feature codec (clientbound or serverbound) | `/implement-packet` command + `packet-implementer` agent | [`docs/packets/IMPLEMENTING_A_PACKET.md`](docs/packets/IMPLEMENTING_A_PACKET.md) |
| Bring up a new client-version column | `/bringup-version` command | [`docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md`](docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md) |
| Audit / implement a mode-prefix dispatcher family | `family-auditor` agent (read-only triage) · `dispatcher-family-implementer` agent (do-mode) | [`docs/packets/DISPATCHER_FAMILY.md`](docs/packets/DISPATCHER_FAMILY.md) |

Every task type's leaf step — promoting one packet × version matrix cell to `✅` — is the single-cell verify procedure: `/verify-packet` command + `packet-verifier` agent, driving [`docs/packets/audits/VERIFYING_A_PACKET.md`](docs/packets/audits/VERIFYING_A_PACKET.md). Do not restate a playbook's procedure in prose elsewhere — link to it.

## Reverse Engineering / IDA

- For IDA Pro lookups, use the `func_query` tool with `name_regex` (the documented method); do not improvise alternate lookup approaches. See the IDA-MCP notes in project memory for the current API.
- Confirm the IDA instance/version under investigation matches the version you're targeting before reading (use `select_instance(port)` for v83/v87/v95/jms).

## Task Workflow

- Before planning or designing a task, first verify the task is not already planned/implemented, and that its task number does not collide with an in-flight task. Use `tools/task-numbers.sh next` and search both `docs/tasks/` and `.worktrees/*/docs/tasks/`.

## Debugging / Verification

- When asked to verify or fix something, confirm the exact server/tenant version the user is testing (e.g. v83 vs v87) before investigating. Do not assume — ask or check, because the wrong version sends the whole investigation down the wrong path.
- Do a full sweep, not spot-checking, unless explicitly told otherwise. A spot-check presented as a full sweep is a false "verified" — and live PATCHes built on it get rejected at validation time.

## Debugging / Kubernetes

- For diagnosing wedged deploys or runtime failures, read the relevant pod logs early (e.g. `atlas-character-factory`, `atlas-world`) via `mcp__kubernetes__pods_log` rather than starting at packet-level fixes or bare pod listings. The logs usually name the real root cause directly.
