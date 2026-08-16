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

## Evidence & grounding

- Never invent values, names, opcodes, output, or behavior; unverified is "unknown / unverified," not a plausible guess. "I think it's X" is a lead to check, not a finding. Quote the actual tool output before drawing a conclusion — do not paraphrase numbers from memory.
- Repo source, WZ data, IDA, and live output outrank remembered general MapleStory knowledge — for game data, packet encoding, protocol details, and service ownership, read the local WZ data or repo source.
- Confirm the exact server/tenant/client version before investigating any bug; the wrong version sends the whole investigation down the wrong path.
- Sweep, don't spot-check. A spot-check presented as a full sweep is a false "verified." State findings as hypotheses until confirmed against real evidence.
- Finish producible work: do not declare a "documented gap," "follow-up task," or "out of scope" when the blocker is a prerequisite you can produce yourself — an unnamed IDB function → name it; an unrouted template → wire it; a missing export → generate it. Do not split work into a new task to avoid finishing this one; keep triage and fix on the same branch, and produce the clean PR branch by rebase at PR time.
- A genuine external blocker, an ambiguous design decision, or an unresolved packet-audit fname is different — surface it and ask. The bar is: can I produce this myself right now?

## Where you work — branch & worktree

- Check the branch before every commit. Setup work that must precede a feature branch still goes on that branch — create it first; it branches from the same HEAD.
- Non-trivial tasks live in their own task worktree. Verify cwd is the correct worktree before planning, designing, or executing a task; cd into it yourself rather than asking the user. Search across all worktrees (`git worktree list`) before concluding a task artifact is missing.
- When dispatching subagents, ensure they operate inside the correct worktree — never write artifacts or edits into the main repo — and verify the tree is clean after they run.
- After completing a rebase, merge, or history-rewrite, always push (force-push when history was rewritten); do not stop at local-only completion.
- A plain push to a task branch triggers the PR workflows and the ephemeral rollout; do not merge `origin/main` as a routine build-triggering ritual. The one exception is a branch that conflicts with `main` — there the push does not build, and the merge is the conflict resolution.

## Done means verified

- Before claiming a branch "done," "ready for PR," or invoking `superpowers:finishing-a-development-branch`, the flagless `tools/verify.sh` must exit 0. Only the flagless invocation counts — `--quick`/`--no-docker` also exit 0 but skip the bake and `-race`.
- Always run the code-review step before opening a PR; do not skip even when the task plan looks complete.
- Code review is a different gate: a green `tools/verify.sh` does not mean the branch is correct, because it cannot see a cross-service seam defect. When a change crosses a service boundary, trace the event into its consumers by hand and check that a test asserts the NEW contract.

## Development workflow

- When asked to understand or plan something, do not start implementing; wait for explicit approval before making any edits. Planning and implementation are separate phases.
- Any non-trivial change runs the four-phase flow — `/spec-task` → `/design-task` → `/plan-task` → `/execute-task` — each a separate slash command invoked from a fresh (`/clear`'d) session, so the next phase consumes only the prior phase's documented artifacts. `/spec-task` runs from the main repo and creates a dedicated worktree at `.worktrees/task-NNN-slug/`; every later phase runs inside that worktree and never creates a new one, so docs, code, and the eventual PR are one unit.
- Skip `/spec-task` only for trivial fixes that don't warrant a PRD; document those directly via a brainstorming session.
- Before planning or designing a task, verify the task is not already planned/implemented and its number does not collide with an in-flight task.

**Task lifecycle mechanics** — each phase's command form and output artifact, fuzzy task identifiers, the resolver's output format, the artifact-location override, and the code-review reviewer roster: see [docs/superpowers-integration.md](docs/superpowers-integration.md).

## Dispatching agents

- Pass an explicit `model` on every Agent/Task dispatch; unspecified inherits Opus, and an Opus subagent turn costs a large multiple of a Sonnet one.
- The pin follows the job, not the `subagent_type` — review/verify/audit runs Sonnet even for an ad-hoc general-purpose agent, scans and inventories run Haiku, implementers run Sonnet unless the plan task is tagged `model: opus`. Never use Fable for background or review workflows.
- Implementers do not run repo-wide verification; they run module-local `go build ./... && go test ./...` and nothing more.
- Fan out with fresh-context agents, not `subagent_type: "fork"` — a named agent type plus an explicit brief. Fork only to continue an interactive debugging thread, and say why inline. *(enforced)*

## Handing off context

- At every durable boundary — a commit landing, a verification gate returning, a fan-out of agents reporting — ask one question: does the next unit of work depend materially on this conversation's history, or only on repository state? Dependency is the signal, not size; thresholds are backstops.
- If it can be resumed from repo state, the task's own reports, and a short written diagnosis, hand off. Handing off means delegating, not clearing: dispatch the next unit to a fresh agent with a brief. `/clear` is a user action — an agent cannot clear itself, so only when the next unit is genuinely controller-shaped do you write the diagnosis down and let the user `/clear`.
- Write the diagnosis before the handoff, not carried in your head — one paragraph into the task folder. A handoff whose reasoning survives only in conversation is not a handoff.

**Dispatch and handoff mechanics** — the full job→model table and its opt-in Opus escalation, the implementer tool-call budget, the verification split, the fork policy, and the handoff thresholds: see [docs/agent-dispatch.md](docs/agent-dispatch.md).

## Repository conventions

- Before defining a new domain type, alias, or numeric constant, check `libs/atlas-constants/` for an existing equivalent.
- Prefer straightforward moves over re-exported type aliases when refactoring shared types or common libraries; don't call another layer's internals across a service boundary.
- Use the project's Builder pattern for test setup; do not create `*_testhelpers.go` files with test-only constructors.
- When producing `design.md`/`plan.md`, write the full document directly to file; do not walk through sections interactively or ask for per-section approval.
- Use repo-relative paths or placeholders in committed files; never a literal home/absolute path. *(enforced under `docs/`)*
- Preserve existing line endings when editing; do not normalize CRLF→LF as a side effect.
- Ask the toolchain (`go list -m -f '{{.Dir}}' <module>`) rather than sweeping the filesystem for a Go dependency's source.
- Never spend inference turns polling a process — launch it with a bound (`run_in_background`, or Monitor with an until-loop) and hand back or do something else in the meantime.
- When updating `TODO.md` or other tracking docs, use `Glob`/`Grep` to find the file first rather than assuming a path.
- Send substantive content as its own text-only message before an `AskUserQuestion` — text emitted in the same turn does not render reliably.
- Do not proactively pitch paid features; if one is genuinely the right tool, lead with what is known and unknown about billing before mentioning it.
- Prefer portable POSIX shell, and per-file Edit/Write over a shell patch loop.

## Where the procedures live

| Trigger | Owner | What's there |
|---|---|---|
| **A `verify.sh` guard failed, or the script and CI disagree** | [docs/verification.md](docs/verification.md) | Why each check exists, per-guard invariants, escape hatches, known CI drift |
| **Adding a new service** | [docs/adding-a-new-service.md](docs/adding-a-new-service.md) | Service onboarding checklist |
| **Packet or protocol work** — new codec, version bring-up, dispatcher family, verifying a cell | [docs/packets/PROCESS.md](docs/packets/PROCESS.md) | Task-type → entry point → canonical playbook |
| **You have a bare task number, or are invoking a superpowers skill outside a phase command** | [docs/superpowers-integration.md](docs/superpowers-integration.md) | Fuzzy task resolution, artifact-location override, code-review roster, maintenance commands |
| **You committed to `main`, a push didn't trigger a build, or `gh` returns 401** | [docs/git-workflow.md](docs/git-workflow.md) | Stray-commit recovery, the conflicts-with-main build exception, token auth |
| **You need to read a client binary** | [docs/reverse-engineering.md](docs/reverse-engineering.md) | `func_query` usage, `idb_list` session resolution |
| **You need a Go dependency's source, or are waiting on a long-running process** | [docs/tooling-conventions.md](docs/tooling-conventions.md) | Toolchain lookups, the polling anti-pattern |
| **You are about to dispatch an agent, or to decide whether to hand off** | [docs/agent-dispatch.md](docs/agent-dispatch.md) | Full job→model table, implementer budget, controller handoff thresholds |
| **You are about to dispatch a second implementer at the same templated transformation** | [docs/codemod-vs-agents.md](docs/codemod-vs-agents.md) | The break-even rule (evaluate before the second dispatch), the task-232 batch-4 worked example, the deferred rewriter's contract |
| **Runtime or Kubernetes debugging** — diagnosing a wedged deploy or crash-loop | [docs/observability.md](docs/observability.md) | Read pod logs first, and how to reach them |
| **Writing or changing a Go service** | `backend-dev-guidelines` skill, `backend-guidelines-reviewer` agent | DOM-* checklist, constants reuse, service boundaries, test builders |
