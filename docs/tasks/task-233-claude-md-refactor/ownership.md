# Documentation ownership map (FR-2)

One authoritative owner per concern. `CLAUDE.md` links to these; it does not
compete with them. This table reproduces design §4's resolved table verbatim
— the PRD's open questions (§9) are already answered there; this task does
not re-litigate them.

| Concern | Owner | Action |
|---|---|---|
| Verification gate | `docs/verification.md` | Extend / consolidate |
| Packet & protocol work | `docs/packets/PROCESS.md` | No change — confirmed sufficient |
| Task lifecycle, task resolution, code-review dispatch | `docs/superpowers-integration.md` | Extend |
| Adding a service | `docs/adding-a-new-service.md` | No change |
| Git / branch / PR workflow | `docs/git-workflow.md` | New |
| Reverse engineering / IDA | `docs/reverse-engineering.md` | New, narrow |
| Shell, editing, tooling conventions | `docs/tooling-conventions.md` | New |
| Agent dispatch, model selection, context handoff | `docs/agent-dispatch.md` | New |
| Runtime / Kubernetes debugging | `docs/observability.md` | Extend |
| Go service patterns (constants, boundaries, builders) | `backend-dev-guidelines` skill + `backend-guidelines-reviewer` | Existing; root keeps one-line bullets |

New-document count: four (`docs/git-workflow.md`, `docs/reverse-engineering.md`,
`docs/tooling-conventions.md`, `docs/agent-dispatch.md`). Within the PRD
budget; no fifth document is proposed.

## The boundary a maintainer will get wrong

`docs/superpowers-integration.md` and `docs/agent-dispatch.md` sound like they
overlap. They don't — they answer different questions:

> `superpowers-integration.md` — *which* command/agent/skill for a situation.
> `agent-dispatch.md` — *how* to dispatch any agent: model, budget, isolation, handoff.

`superpowers-integration.md` is scoped to the four-phase Superpowers workflow.
`agent-dispatch.md`'s rules apply to **every** dispatch in **every** session,
including ad-hoc ones that never enter the four-phase workflow.

## Where does a new rule go?

A short decision list, so root context cannot silently re-accumulate
procedure the way it did before this refactor:

1. **Is it an invariant** — would an agent that hasn't read any owner
   document make an unrecoverable mistake without it (wrong branch, fabricated
   finding, false verification claim, edit in the wrong worktree)? → Stays
   directly in `CLAUDE.md`, stated in full.
2. **Is it a broadly-applicable default**, one sentence, cheaper to keep
   loaded than to route to? → A compact bullet in `CLAUDE.md` §Repository
   conventions (or the nearest Tier-2 section); detail and rationale go to the
   matching owner document below.
3. **Does it only matter once the agent is already inside a specific
   workflow** (packet work, git/PR mechanics, IDA/RE, shell/tooling
   mechanics, agent dispatch mechanics, verification internals, task
   lifecycle, runtime debugging, Go service patterns)? → Match it to the one
   row above whose Concern column covers it, and write it there. Never split
   the same procedure across two of these documents.
4. **Is it rationale, a measurement, an incident, or a cost comparison** that
   doesn't change current behavior? → Goes in the owner document as
   supporting detail, never in `CLAUDE.md`.
5. **Does no concern in the table above cover it?** → That is the signal to
   stop and raise it as an explicit ownership decision (as design §4 did for
   the four new documents) rather than default it into `CLAUDE.md` because
   there's nowhere else to put it.
