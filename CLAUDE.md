# Atlas

## Project Overview

This is a Go microservices game server project with 14+ services. The primary language is Go. TypeScript is used only for atlas-ui.

## Workflow Rules

When asked to understand or plan something, DO NOT start implementing code changes. Wait for explicit approval before making any edits. Planning and implementation are separate phases.

## Build & Verification

Before claiming a branch is "done," "ready for PR," or invoking
`superpowers:finishing-a-development-branch`:

```sh
tools/verify.sh          # flagless, must exit 0. --quick is the inner loop only.
```

Only the **flagless** invocation counts as verified. `--quick` and `--no-docker`
also exit 0 — they print a caveat and skip the bake and `-race` — so "verify.sh
exited 0" is not a pass unless it ran with no flags.

It mirrors `.github/workflows/pr-validation.yml`: per-module `go build`/`vet`/
`test -race`, `docker buildx bake atlas-<svc>` for every service whose `go.mod`
was touched, every repo guard, and `tools/lint.sh --check`. Change-gated the
same way CI is.

**Never claim verified from a subset.** The bake step in particular is not
optional — `go build` against `go.work` cannot catch a missing `COPY libs/...`
in the shared Dockerfile.

Running the gate **per task** on a long branch? Pass `--base <last-gated-commit>`
so the change set is the increment, not the accumulated branch — one `libs/`
commit otherwise fans every later run out to all 86 modules (~11 min instead of
~1). Launch it in the background and keep working; never idle waiting on it.

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

Phase commands accept fuzzy task identifiers: `task-054-slug`, `task-054`, `054`, or `54` all resolve to the same folder. They resolve it with `tools/task-resolve.sh <identifier>`, which prints `<task-id>\t<task-dir>\t<worktree>`. Never glob `.worktrees/*/docs/tasks/task-*` to find a task — every worktree carries a full copy of `docs/tasks/` from its branch point, so that pattern returns (tasks × worktrees) mostly-duplicate paths into context to resolve a single ID.

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

- Before planning or designing a task, first verify the task is not already planned/implemented, and that its task number does not collide with an in-flight task. Use `tools/task-numbers.sh next` to pick the number and `tools/task-resolve.sh --list` to see every existing task (one row per task, already deduplicated across worktrees).

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
- Long agents are the cost: context grows with turn count and every turn re-reads all of it, so one 600-turn agent costs far more than the same work split across fresh contexts. The implementer budget is **120 tool calls**, warned at 100 — enforced by `.claude/hooks/turn-budget.sh` and contracted in `.claude/agents/atlas-implementer.md`, `.claude/agents/packet-implementer.md`, and `.claude/agents/dispatcher-family-implementer.md`. At the cap an implementer commits and reports `PARTIAL`; the controller dispatches a continuation. The number lives in the hook — change it there only.
- Implementers do not run repo-wide verification. `tools/verify.sh`, `tools/lint.sh`, `-race`, and docker bake belong to the `atlas-verifier` agent in its own clean context; a `--quick` run inside a 400k-token implementer costs a large multiple of the same run in a 20k one. Implementers run module-local `go build ./... && go test ./...` and nothing more.
- Fan out with **fresh-context agents, not `subagent_type: "fork"`.** A fork inherits the parent's entire conversation and re-reads it on every turn, so a forked child that runs 70+ turns costs several times a briefed agent doing the same job. Default to a named agent type plus an explicit brief. Fork only to continue an interactive debugging thread whose brief would be longer than the context it saves — and say so, because `.claude/hooks/fork-dispatch-guard.sh` requires the justification inline.
- The same arithmetic binds the **controller**, which is the one context that lives for a whole plan — every wake-up re-reads it. During `/execute-task`, hand off to a fresh session past ~250k tokens with tasks remaining: the SDD ledger (`.superpowers/sdd/<plan>/progress.md`) is the resume point, so the cost is one plan re-read. Procedure: `.claude/commands/execute-task.md` Step 4e.

## Context Handoff

The unit of work is a **briefable task, not a conversation.** Context cost scales
with turn count × context size, so 50 turns carried at 190k cost roughly ten times
the same 50 turns at 19k — regardless of what they accomplish.

**At every durable boundary — a commit landing, a verification gate returning, a
fan-out of agents reporting — ask one question: does the next unit of work depend
materially on this conversation's history, or only on repository state?**

If it can be resumed from repo state + the task's own reports + a short written
diagnosis, hand off. Do not wait for a context threshold; a high threshold is a
backstop, not the trigger. The signal is dependency, not size.

- **Handing off means delegating, not clearing.** `/clear` is a user action —
  you cannot clear yourself. Dispatch the next unit to a fresh agent with a brief
  (`atlas-implementer` + `atlas-verifier` for code work). Only when the next unit
  is genuinely controller-shaped should you instead write the diagnosis down, tell
  the user this is a clean handoff point, and let them `/clear`.
- **The diagnosis must be written before the handoff, not carried in your head.**
  One paragraph into the task folder — what was found, what it means, what the
  next step is. One turn to write; it is what makes the handoff lossless. A
  handoff whose reasoning survives only in conversation is not a handoff.
- **There is a floor as well as a backstop.** Below roughly 60k a fresh agent
  re-discovers files you already hold, and you pay for that discovery twice.
  Under ~40 tool calls, prefer continuing. `.claude/hooks/commit-boundary.sh`
  encodes this floor and raises the question at commits past it. The backstop at
  the other end is ~250k for a controller — see `/execute-task` Step 4e, which is
  this same rule in its threshold form, with the measured numbers behind it.
- **The pattern already exists — reuse it.** `/execute-task` Step 4d (`PARTIAL`
  → continuation brief beside the original → same report file as the persistent
  memory across the split → fresh implementer) and Step 4e (controller → SDD
  ledger → fresh session) are both exactly this handoff, and both already carry a
  durable artifact: `task-N-report.md` and
  `.superpowers/sdd/<plan>/progress.md`. Neither is special to `/execute-task` —
  apply the same shape in any session, and where a canonical ledger already
  exists, write there rather than inventing a second artifact. Generate briefs
  with `tools/task-brief.sh`, never by hand out of `plan.md`.

The failure this prevents: one session doing four unrelated jobs — resolve a merge,
verify packet cells, run reviews, then fix a service bug — where the last job needed
exactly one sentence from the first three but re-read all of them on all 57 of its
turns.

## Shell & Editing Conventions

- Prefer portable POSIX shell in Bash commands; avoid zsh/direnv-specific constructs and batch patch loops that can produce garbled or unapplied output. When a multi-file edit is needed, prefer per-file Edit/Write over a shell patch loop.
- Preserve line endings when editing (do not normalize CRLF→LF as a side effect) — it inflates diffs with spurious changes.
- **Never sweep the filesystem to locate a Go dependency's source.** Ask the toolchain: `go list -m -f '{{.Dir}}' <module>` prints the directory in ~0.02s, whether the module resolves to the module cache or to a local `replace`. `find /` takes ~2 minutes on WSL2 and is how one task-227 session burned 6 minutes across five sweeps hunting for `atlas-rest` — which `go.mod` had `replace`d to `libs/atlas-rest` inside the worktree the agents were already working in. Guessing at module-cache case-escaping (`!chronicle20`) is the tell that you should have asked `go list`. The same applies to `go doc <pkg>` for a symbol and `go list -m all` for the version set. `find` is for paths you own, rooted at a directory you name — never at `/`.
- **Never spend inference turns waiting for a process.** Launch it once with a bound — `run_in_background: true`, or `Monitor` with an until-loop — and do something else or hand back. Repeated `sleep` / `ps aux | grep` / `echo waiting` / `for i in $(seq …); do sleep` Bash calls are the anti-pattern: each one re-reads the whole context to learn nothing, and they cluster late in a session where that is most expensive. If the process exceeds its bound, kill it and fall back; do not keep polling. When a tool has a known hang mode, the fallback belongs in that tool's agent doc, not in a longer wait.
