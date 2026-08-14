# Atlas

## Project Overview

This is a Go microservices game server project with 14+ services. The primary language is Go. TypeScript is used only for atlas-ui.

## Workflow Rules

When asked to understand or plan something, DO NOT start implementing code changes. Wait for explicit approval before making any edits. Planning and implementation are separate phases.

## Build & Verification

Before claiming a branch is "done," "ready for PR," or invoking
`superpowers:finishing-a-development-branch`:

```sh
tools/verify.sh          # must exit 0. Use --quick for the inner loop.
```

It mirrors `.github/workflows/pr-validation.yml`: per-module `go build`/`vet`/
`test -race`, `docker buildx bake atlas-<svc>` for every service whose `go.mod`
was touched, every repo guard, and `tools/lint.sh --check`. Change-gated the
same way CI is.

**Never claim verified from a subset.** The bake step in particular is not
optional — `go build` against `go.work` cannot catch a missing `COPY libs/...`
in the shared Dockerfile.

Rationale, per-guard invariants, escape hatches, and the known CI drift:
[`docs/verification.md`](docs/verification.md). Adding a service:
[`docs/adding-a-new-service.md`](docs/adding-a-new-service.md).

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

## Worktrees & Subagents

- Tasks live in git worktrees (often siblings of the main repo). Before planning/designing/executing a task, verify cwd is the correct worktree; if not, cd into it yourself rather than asking the user.
- When searching for task PRDs/plans/designs, search across all worktrees (`git worktree list`) before concluding a file is missing.
- Never edit files in the main repo when a task worktree exists for that work.
- When dispatching subagents (reviewers, doc agents), ensure they operate inside the correct worktree — never write artifacts or edits into the main repo. Verify the tree is clean after subagent runs.

## Code Review Before PR

- Always run the code-review step before opening a PR. Do not skip even when the task plan looks complete.
- **A green `tools/verify.sh` does not mean the branch is correct.** Every module can build, vet, test and bake clean while the branch carries blocking defects, because each service is self-consistent in isolation. The gate cannot see: a seam between two services (producer empties a compartment the consumer still reads), a new saga action with no step handler in the orchestrator, or a class of missing emits that existing tests actively pin as the old behavior. When a change crosses a service boundary, trace the event into its consumers by hand and check that a test asserts the NEW contract, not the old silent drop.

## Grounding, Verification & Finishing

**Never invent.** Values, names, opcodes, command output, behavior — if it is
not verified from source, WZ data, IDA, or live output, say "unknown /
unverified." Do not fill the gap with a plausible guess. "I think it's X" is a
lead to check, not a finding. Quote the actual tool output before drawing a
conclusion; do not paraphrase numbers from memory.

**Verify, don't recall.** For game data (props, item IDs, skill effects, WZ
data), packet encoding, protocol details, and service ownership: read the local
WZ data or repo source. General MapleStory knowledge is not a source.

**Confirm the version first.** Before investigating any bug, confirm the exact
server/tenant version being tested (v83 vs v87 …). The wrong version sends the
whole investigation down the wrong path.

**Sweep, don't spot-check.** A spot-check presented as a full sweep is a false
"verified," and live PATCHes built on it get rejected at validation time.
State findings as hypotheses until confirmed against real evidence — diffs,
logs, live k8s state.

**Finish producible work.** Do not declare a "documented gap," "follow-up
task," or "out of scope" when the blocker is a prerequisite you can produce
yourself: an unnamed IDB function → name it; an unrouted template → wire it; a
missing export → generate it. Do not split work into a new task to avoid
finishing this one — keep triage and fix on the same branch, and produce the
clean PR branch by rebase at PR time. No `// TODO`, stubbed handlers, or 501s
in landed commits.

A genuine external blocker, an ambiguous design decision, or an unresolved
packet-audit fname is different — surface it and ask. The bar is "can I produce
this myself right now?" If yes, do it.

## Test Helper Pattern

- Use the project's Builder pattern for test setup. Do not create `*_testhelpers.go` files with test-only constructors.

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
- Confirm the IDA instance/version under investigation matches the version you're targeting before reading. `select_instance(port)` and port-based selection are dead (since task-138); resolve the session from `idb_list` by binary **name** and pass it as the `database` parameter to subsequent calls.

## Task Workflow

- Before planning or designing a task, first verify the task is not already planned/implemented, and that its task number does not collide with an in-flight task. Use `tools/task-numbers.sh next` and search both `docs/tasks/` and `.worktrees/*/docs/tasks/`.

## Debugging / Kubernetes

- For diagnosing wedged deploys or runtime failures, read the relevant pod logs early (e.g. `atlas-character-factory`, `atlas-world`) via `mcp__kubernetes__pods_log` rather than starting at packet-level fixes or bare pod listings. The logs usually name the real root cause directly.

## Git Operations

- **Never commit or push directly to `main`.** Branch protection blocks the push, so a commit made on local `main` is stranded and never reaches the remote. Check the branch before every `git commit`. Setup work that must precede a feature branch still goes *on* the feature branch — create it first; it branches from the same HEAD. Recovery from a stray main commit: preserve the content on a branch (cherry-pick if needed), then `git fetch origin main && git reset --hard origin/main`.
- After completing a rebase/merge/history-rewrite, always push (force-push when history was rewritten) so the PR reflects the resolved state. Do not stop at local-only completion — a rebase resolved only locally leaves the PR still showing conflicts.
- A plain push to a task branch DOES trigger the PR workflows and the `atlas-pr-<N>` ephemeral rollout. Do not merge `origin/main` as a routine build-triggering ritual. The one exception: when the branch **conflicts with main**, the push does not start the build — merge `origin/main`, resolve, push the merge commit. The merge is the conflict resolution, not the trigger.
- Run `gh` with the token env explicitly cleared so it uses stored `hosts.yml` auth (account `Chronicle20`): `env -u GH_TOKEN -u GITHUB_TOKEN gh …`. Do NOT source `~/.config/atlas/gh.env` — its `GH_TOKEN` is expired and takes precedence, causing 401s. Never echo the token.

## Interaction

- Text emitted before an `AskUserQuestion` in the same turn does not render reliably in this user's terminal. Send substantive content (findings, proposals, designs) as its own text-only message, then ask the question in a following turn.
- Do not proactively pitch paid features (`/schedule`, remote/cloud agents, `/ultrareview`, scheduled routines). If one is genuinely the right tool, lead with what is known and unknown about how it bills before mentioning it. End-of-turn suggestions should be free/local actions.

## Model & Cost Preferences

- Pass an explicit `model` on **every** Agent/Task dispatch. Unspecified inherits Opus, and an Opus subagent turn costs ~7x a Sonnet one.
- The pin follows the **job**, not the `subagent_type`. Any dispatch whose job is review / verify / audit runs `sonnet`, always — including ad-hoc `general-purpose` agents carrying a review prompt, which frontmatter pins do not cover. Scans and inventories run `haiku`. Implementers run `sonnet` unless the plan task is tagged `model: opus`. Full table: `.claude/commands/execute-task.md` Step 4a.
- Never use Fable for background/review workflows.
- Long agents are the cost: keep any one subagent under ~150 turns. Past that, split the task — context grows with turn count and every turn re-reads all of it.

## Shell & Editing Conventions

- Prefer portable POSIX shell in Bash commands; avoid zsh/direnv-specific constructs and batch patch loops that can produce garbled or unapplied output. When a multi-file edit is needed, prefer per-file Edit/Write over a shell patch loop.
- Preserve line endings when editing (do not normalize CRLF→LF as a side effect) — it inflates diffs with spurious changes.
