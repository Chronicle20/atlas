# Agent Dispatch

This document owns *how* to dispatch any agent — model, budget, isolation,
handoff — for every dispatch in every session, including ad-hoc ones outside
the four-phase workflow. `docs/superpowers-integration.md` owns *which*
command, agent, or skill to reach for in a given situation.

---

## Model selection

Pass an explicit `model` on **every** Agent/Task dispatch. Never rely on
inheritance: an unspecified model inherits the main-loop model (Opus), and an
Opus subagent turn costs ~7x a Sonnet one.

The pin is chosen by the **job the agent is doing**, not by its
`subagent_type`. Named reviewer agents carry a Sonnet pin in their
frontmatter, but an ad-hoc `general-purpose` dispatch carrying a review
prompt does not — that is the hole this rule closes.

| Job | Model | Notes |
|---|---|---|
| Review, verify, audit, re-review, whole-branch review | **`sonnet`** — always | No exceptions. Reviewing is reading against a checklist; Opus buys nothing and these run long |
| Scan, inventory, doc sweep, file-finding | `haiku` | |
| Run the verification gate (`atlas-verifier`) | `haiku` | Frontmatter pin; it runs one command and quotes the output |
| Implement a plan task (`atlas-implementer`) | `sonnet` | Default; frontmatter pin |
| Implement a packet codec (`packet-implementer`) or dispatcher family (`dispatcher-family-implementer`) | `sonnet` | Frontmatter pin; both also carry the 120-call PARTIAL budget |
| Implement a plan task tagged `model: opus` in plan.md | `opus` | Opt-in only — see below; pass `model: opus` on the dispatch to override the frontmatter |

A plan task may be tagged `model: opus` in `plan.md` when it is genuinely
derivation-heavy: IDA/packet field-order derivation, saga orchestration
across services, or a cross-service contract change. Everything else — REST
surfaces, GORM entities, Kafka consumers, tests, template routing — runs
Sonnet.

If an implementer comes back wrong twice on Sonnet, escalate that one task to
Opus and note it, rather than raising the default.

Never use Fable for background or review workflows.

## The implementer budget

The implementer budget is **120 tool calls, warned at 100**, counted by
`.claude/hooks/turn-budget.sh` and contracted in
`.claude/agents/atlas-implementer.md`, `.claude/agents/packet-implementer.md`,
and `.claude/agents/dispatcher-family-implementer.md`. At the cap the
implementer commits what works and reports `PARTIAL`; the controller
dispatches a continuation. The number is changed in the counting hook only.

The cap is **binding**: `.claude/hooks/turn-budget-guard.sh` (PreToolUse)
denies subagent calls past CAP+5, exempting the commit-and-report path —
controllers are never blocked by it.

The underlying arithmetic: context grows with turn count and every turn
re-reads all of it, so one 600-turn agent costs far more than the same work
split across fresh contexts. Splitting is the designed outcome, not a
failure.

## Verification split

Implementers never run `tools/verify.sh`, `tools/lint.sh`, `-race`, or
docker bake — a `--quick` run inside a 400k-token implementer costs a large
multiple of the same run in a clean 20k one, and its output is the biggest
avoidable consumer of an implementer's window. Implementers run module-local
`go build ./... && go test ./...` and nothing more; the repo gate belongs to
`atlas-verifier`, in its own clean context.

For the concurrency procedure that runs that gate against the rest of the
plan — launch, keep going, reconcile, at most one gate in flight — see
`/execute-task` Step 4c. That is command mechanics and stays there.

## Fork vs fresh context

Fan out with **fresh-context agents, not `subagent_type: "fork"`** — a named
agent type plus an explicit brief. Fork only to continue an interactive
debugging thread, and say why inline. `.claude/hooks/fork-dispatch-guard.sh`
denies an unjustified fork and states the cost.

## Context handoff

The unit of work is a **briefable task, not a conversation.** Context cost
scales with turn count × context size, so 50 turns carried at 190k cost
roughly ten times the same 50 turns at 19k — regardless of what they
accomplish.

At every durable boundary — a commit landing, a verification gate returning,
a fan-out of agents reporting — the decision criterion is whether the next
unit of work depends materially on this conversation's history, or only on
repository state. If it can be resumed from repo state + the task's own
reports + a short written diagnosis, hand off.

Handing off means delegating, not clearing — `/clear` is a user action, and
no agent can clear itself. The diagnosis is written down *before* the
handoff, not carried in your head: one paragraph into the task folder, so
the handoff is lossless even though the reasoning does not survive in
conversation.

**The floor.** Below roughly 60k tokens a fresh agent re-discovers files you
already hold, and you pay for that discovery twice; under ~40 tool calls,
prefer continuing. `.claude/hooks/commit-boundary.sh` encodes this floor and
raises the question at commits past it.

**The backstop.** ~250k tokens for a controller — the one context that lives
for a whole plan, where every wake-up re-reads it. Measured on a real
18-task run: the controller finished at 402k tokens having produced only
165KB of its own tool output across 157 calls. Its last 42 turns — a
self-contained segment sharing no state with the preceding tasks — ran at
360-400k each; in a fresh session those same turns would have run at ~80k.

Generate briefs with `tools/task-brief.sh`, never by hand out of `plan.md` —
assembling them by hand is exactly the context bloat the brief exists to
prevent. The durable artifacts that make a handoff resumable are
`task-N-report.md` per task and the SDD ledger
`.superpowers/sdd/<plan>/progress.md` for the whole plan.

`/execute-task` Steps 4d–4e are this handoff in its two concrete procedural
forms — `PARTIAL` handling and controller handoff — each keyed to one of
those durable artifacts. Apply the same shape in any session; where a
canonical ledger already exists, write there rather than inventing a second
artifact.
