# Atlas

Go microservices game server monorepo, 14+ services. Go is the primary language; TypeScript is used only in atlas-ui.

## Never do this

- Never commit or push directly to `main`.
- Never edit files in the main repo when a task worktree exists for that work.
- Never invent a value, name, opcode, output, or behavior.
- Never claim verified from a flagged or partial run.
- Never open a PR without code review.
- Never dispatch an agent without an explicit `model`.
- Never land a placeholder comment, a stubbed handler, or an unimplemented status response.
- Never spend inference turns polling a process or waiting on a child agent. *(enforced)*

## Evidence & grounding

- Unverified is "unknown / unverified," never a plausible guess. "I think it's X" is a lead to check, not a finding. Quote the actual tool output before concluding; never paraphrase numbers from memory.
- Repo source, WZ data, IDA, and live output outrank remembered MapleStory knowledge. For game data, packet encoding, protocol details, and service ownership, read the local WZ data or repo source.
- Confirm the exact server/tenant/client version before investigating any bug.
- Sweep, don't spot-check. A spot-check presented as a full sweep is a false "verified." Findings are hypotheses until confirmed against real evidence.
- Finish producible work. Never declare a "documented gap," "follow-up task," or "out of scope" when the blocker is a prerequisite you can produce yourself — an unnamed IDB function, an unrouted template, a missing export. The bar is: can I produce this myself right now? A genuine external blocker, an ambiguous design decision, or an unresolved packet-audit fname is different: surface it and ask.

## Development workflow

- Asked to understand or plan? Do not implement. Wait for explicit approval before any edit.
- Any non-trivial change runs the four-phase flow — `/spec-task` → `/design-task` → `/plan-task` → `/execute-task` — each a separate command from a fresh (`/clear`'d) session, consuming only the prior phase's artifacts.
- `/spec-task` runs from the main repo and creates the worktree at `.worktrees/task-NNN-slug/`; every later phase runs inside it and never creates a new one. Skip `/spec-task` only for trivial fixes that don't warrant a PRD; document those via a brainstorming session.
- Before planning or designing, verify the task is not already planned/implemented and its number does not collide with an in-flight task.
- Verify cwd is the correct worktree before planning, designing, or executing; cd there yourself. Subagents work inside that same worktree, and the tree must be clean after they run.
- Keep triage and fix on the same branch; produce the clean PR branch by rebase at PR time.
- Write `design.md`/`plan.md` directly to file; do not walk sections interactively or ask for per-section approval.

## Done means verified

- Before calling a branch "done," "ready for PR," or invoking `superpowers:finishing-a-development-branch`, the flagless `tools/verify.sh` must exit 0. `--quick`/`--no-docker` also exit 0 but skip the bake and `-race`; they do not count.
- Always run code review before opening a PR, even when the plan looks complete.
- Code review is a separate gate: a green `tools/verify.sh` cannot see a cross-service seam defect. When a change crosses a service boundary, trace the event into its consumers by hand and confirm a test asserts the NEW contract.

## Dispatching agents

- The `model` pin follows the job, not the `subagent_type`; unspecified inherits Opus, which costs a large multiple of Sonnet per turn. Never use Fable for background or review workflows.
- Fan out with fresh-context agents — a named agent type plus an explicit brief. Fork only to continue an interactive debugging thread, and say why inline. *(enforced)*
- Per-unit review is `task-reviewer`, never a bare `general-purpose` dispatch; reviewers follow their own protocol.
- Read `docs/agent-dispatch.md` before dispatching, and `docs/review-protocol.md` before dispatching a reviewer.

## Handing off context

- At every durable boundary — a commit landing, a gate returning, a fan-out reporting — ask: does the next unit depend materially on this conversation's history, or only on repository state? Dependency is the signal, not size.
- If it is resumable from repo state, task reports, and a short written diagnosis, hand off. Handing off means delegating to a fresh agent with a brief, not clearing; an agent cannot clear itself.
- Always write the diagnosis down first — one paragraph into the task folder. A handoff whose reasoning survives only in conversation is not a handoff.

## Repository conventions

- Check `libs/atlas-constants/` before defining a new domain type, alias, or numeric constant.
- Prefer straightforward moves over re-exported type aliases when refactoring shared code; never call another layer's internals across a service boundary.
- Use the project's Builder pattern for test setup; no `*_testhelpers.go` test-only constructors.
- Use repo-relative paths or placeholders in committed files; never a literal home/absolute path. *(enforced under `docs/`)*
- Preserve existing line endings; never normalize CRLF→LF as a side effect.
- Ask the tooling for a mechanical fact rather than deriving it; find tracking docs with `Glob`/`Grep` rather than assuming a path. Slice a large artifact before reading it whole.
- Batch a gate-log or review-artifact read with the `progress.md`/`agent-ledger.sh` append recording its verdict into the *same* tool call.
- Do not proactively pitch paid features; if one is the right tool, lead with what is known and unknown about billing.
- Send substantive content as its own text-only message before an `AskUserQuestion`.

## Where the procedures live

Load the owning document before acting in its area; it holds the mechanics this file omits.

| Trigger | Owner |
|---|---|
| Dispatching any agent, or deciding whether to hand off | [docs/agent-dispatch.md](docs/agent-dispatch.md) |
| Dispatching a reviewer, or writing up a review | [docs/review-protocol.md](docs/review-protocol.md) |
| A `verify.sh` guard failed, or script and CI disagree | [docs/verification.md](docs/verification.md) |
| A bare task number, or a superpowers skill outside a phase command | [docs/superpowers-integration.md](docs/superpowers-integration.md) |
| Committing, pushing, rebasing; a stray `main` commit, a push that didn't build, `gh` 401 | [docs/git-workflow.md](docs/git-workflow.md) |
| A Go dependency's source, a long-running process, a mechanical repo fact, shell/editing conventions | [docs/tooling-conventions.md](docs/tooling-conventions.md) |
| About to read a large document, diff, plan, or tool result | [docs/slice-first.md](docs/slice-first.md) |
| The PR is open and something is wrong (Phase 5, `/fix-pr-bug`) | [docs/post-implementation.md](docs/post-implementation.md) |
| About to dispatch a second implementer at the same transformation | [docs/codemod-vs-agents.md](docs/codemod-vs-agents.md) |
| Packet or protocol work | [docs/packets/PROCESS.md](docs/packets/PROCESS.md) |
| Reading a client binary | [docs/reverse-engineering.md](docs/reverse-engineering.md) |
| Adding a new service | [docs/adding-a-new-service.md](docs/adding-a-new-service.md) |
| A wedged deploy or crash-loop | [docs/observability.md](docs/observability.md) |
| Writing or changing a Go service | `backend-dev-guidelines` skill, `backend-guidelines-reviewer` agent |
| A cross-repository process-parity question (`atlas`, `home-hub`, `Harbormaster`, `MyFleet`) | [docs/process-parity.md](docs/process-parity.md) |
