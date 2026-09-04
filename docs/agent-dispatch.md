# Agent Dispatch

This document owns *how* to dispatch any agent — model, budget, isolation,
handoff — for every dispatch in every session, including ad-hoc ones outside
the four-phase workflow. `docs/superpowers-integration.md` owns *which*
command, agent, or skill to reach for in a given situation.

**Historical-name cutoff.** Task-266 renamed `atlas-implementer`,
`atlas-verifier`, and `atlas-reviewer` to the generic `task-implementer`,
`task-verifier`, and `task-reviewer` for cross-repository process parity
(`docs/process-parity.md` §5.1). Artifacts under `docs/tasks/` from before that
task use the old `atlas-*` names because those dispatches actually happened
under them; they are historical records, not rewritten to assert a name that
wasn't in use at the time.

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
| Run the verification gate (`task-verifier`) | `haiku` | Frontmatter pin; it runs one command and quotes the output |
| Implement a plan task (`task-implementer`) | `sonnet` | Default; frontmatter pin |
| Implement a packet codec (`packet-implementer`) or dispatcher family (`dispatcher-family-implementer`) | `sonnet` | Frontmatter pin; both also carry the 120-call PARTIAL budget |
| Implement a plan task tagged `model: opus` in plan.md | `opus` | Opt-in only — see below; pass `model: opus` on the dispatch to override the frontmatter |

A plan task may be tagged `model: opus` in `plan.md` when it is genuinely
derivation-heavy: IDA/packet field-order derivation, saga orchestration
across services, or a cross-service contract change. `/plan-task` should
apply that tag sparingly and justify it in one line. Everything else — REST
surfaces, GORM entities, Kafka consumers, tests, template routing — runs
Sonnet.

If an implementer comes back wrong twice on Sonnet, escalate that one task to
Opus and note it, rather than raising the default.

Never use Fable for background or review workflows.

## The implementer budget

The implementer budget is **120 tool calls, warned at 100**, counted by
`.claude/hooks/turn-budget.sh` and contracted in
`.claude/agents/task-implementer.md`, `.claude/agents/packet-implementer.md`,
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

Before dispatching a second implementer at the same templated transformation,
check whether an AST codemod is cheaper than the remaining manual dispatches
— see [docs/codemod-vs-agents.md](codemod-vs-agents.md).

## Verification split

Implementers never run `tools/verify.sh`, `tools/lint.sh`, `-race`, or
docker bake — a `--quick` run inside a 400k-token implementer costs a large
multiple of the same run in a clean 20k one, and its output is the biggest
avoidable consumer of an implementer's window. Implementers run module-local
`go build ./... && go test ./...` and nothing more; the repo gate belongs to
`task-verifier`, in its own clean context.

For the concurrency procedure that runs that gate against the rest of the
plan — launch, keep going, reconcile, at most one gate in flight — see
`/execute-task` Step 4c. That is command mechanics and stays there.

## Inline vs delegate

Delegation is strongly preferred when it replaces a meaningful sequence of
expensive turns in an already-large context. It is a loss when it replaces one
or two cheap ones.

A fresh subagent carries a **~35–38k dispatch floor** before it has done
anything (measured: an agent's turn-1 input is 38,178 tokens; a whole two-turn
agent cost 52,857). So the decision is arithmetic, not conceptual:

> **If you can answer the question in roughly one or two targeted tool calls,
> answer it yourself. Break-even is about four to five turns of your own
> work** — a ~35k floor against a parent turn at 100–150k.

Decide on *expected turns and context*, not on how big the task sounds. "Audit
the saga step handler coverage" sounds like a delegation and is a grep. "Fix
this one-line bug" sounds trivial and is a fresh implementer, because your own
context is 300k.

What this rules out, with the measurement behind it. One
`backend-guidelines-reviewer` dispatched six children for checklist questions:

| Child | Turns | Billed input | Output tokens |
|---|---|---|---|
| Pending_change domain DOM/FILE checklist | 20 | 2,009,604 | 2,997 |
| DOM-21 atlas-constants reuse check | 10 | 713,475 | **25** |
| Orphan reconciliation severity assessment | 10 | 634,682 | **39** |
| Saga step handler coverage audit | 12 | 576,718 | 1,581 |
| Hand-mirrored cashshop kafka struct parity | 6 | 335,081 | **22** |
| Cashshop Purchase tx boundary audit | 2 | 52,857 | **5** |
| **Total** | 60 | **4.32M** | 4,669 |

Four of the six produced fewer than 40 output tokens. Their *returns* were
maximally compact — the cost was the floor plus each child's own context growth.
The last one made a single tool call and cost 52,857.

The parent then had nothing to do while its async children ran and emitted **30
`Bash true` no-op turns** — 33% of its tool calls, ≈3.6M tokens, ≈36% of the
whole agent, for zero information.
`.claude/hooks/wait-loop-guard.sh` now refuses those calls and the polling
equivalents; this rule removes the reason to make them.

**Reviewers do not fan out at all.** A reviewer answers its own checklist. See
[docs/review-protocol.md](review-protocol.md).

### Shrinking the floor itself

Two parts of the floor are ours, not the harness's:

- **The agent roster.** The custom-agent listing costs ~5.3k (measured via
  `/context`) and is delivered as an "Available agent types for the Agent tool"
  reminder. Whether denying the tool also suppresses the listing is **inferred,
  not measured** — the emitting code has no named gate. Leaf agents deny it
  regardless, because it enforces the no-fan-out rule structurally:
  `disallowedTools: [Agent]`. Use `disallowedTools`, never a `tools:` allowlist
  — the packet agents must keep `tools:` omitted, which is the only documented
  way to inherit MCP tools (`ida-pro-mcp`). `packet-implementer` is the
  exception: it legitimately dispatches `packet-verifier` per cell.
- **Bundled skills.** Claude Code's built-in skills (`dataviz`, `claude-api`,
  `design`, `update-config`, …) are ~2.5k of listing this repo never invokes
  (summed from `/context`'s built-in skill listing: dataviz 380, claude-api 360,
  design 340, code-review 270, update-config 240, and eleven smaller entries).
  `disableBundledSkills: true` in `.claude/settings.json` removes them. Plugin
  skills (`superpowers:*`) and the four-phase commands are unaffected; the
  built-in `/code-review` skill goes with them, and repo code review runs
  through the reviewer agents anyway.

**Never idle waiting on a child.** Agent completions arrive as notifications —
do other work, or end the turn and be re-invoked. There is no wait primitive
because none is needed.

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
raises the question at commits past it. Past its ESCALATE tier (~150k),
`.claude/hooks/context-handoff-guard.sh` denies dispatching a *new* unit
(implementer, explorer, planner, bare general-purpose) while still allowing
the dispatches that finish the unit in flight (reviewers, verifiers,
auditors); a `CONTEXT-JUSTIFIED:` line in the prompt is the escape hatch.

**The backstop.** ~150k tokens for a controller, or 4 completed plan tasks
in one session, whichever comes first — the one context that lives for a
whole plan, where every wake-up re-reads it, and the second trigger exists
for a controller that cannot read its own context size. Apply the ceiling
unconditionally: past the threshold, the controller does not start another
plan task, however many remain — a carve-out for "only a couple left" is
exactly the shape of the failure below. Measured on a real 18-task run: the
controller finished at 402k tokens having produced only 165KB of its own
tool output across 157 calls. Its last 42 turns — a self-contained segment
sharing no state with the preceding tasks — ran at 360-400k each; in a fresh
session those same turns would have run at ~80k. A second run, `854e6e87`,
wrote a handoff marker (HANDOFF #10) at 243k tokens and then ran 26 more
turns at an average of 259k (6.73M tokens) to finish one more plan task
anyway; all 17 sessions in that task's run ended at their peak context. A
handoff the same context then works past is not a handoff — the marker on
disk is meaningless if the session that wrote it keeps going.

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

**The rule does not stop when implementation does.** PR validation, live
testing, debugging, regression investigation, and follow-up fixes are the same
question at the same boundaries — and they are where it was measured to be
ignored: one task's post-PR phase was 12.7% of its total spend at **94% main
thread**, three subagents across four sessions, peaking at 328k and 274k solo.
Against the execute phase's 19% main-thread share, that is the workflow's
largest single structural regression. The concrete loop — reproduce inline,
diagnose into `docs/tasks/<task>/bug-<slug>.md`, dispatch a fresh implementer
against that file, verify in a clean context — is
[docs/post-implementation.md](post-implementation.md), mechanized as
`/fix-pr-bug`.

## Recording what a dispatch cost

Append one line per agent to the task's ledger at reconcile time:

```sh
tools/agent-ledger.sh append <task> --unit "<plan task or bug slug>" \
  --agent-type <type> --model <model> --status <status> --commit <sha>
```

Reviewer rows add `--verdict` and `--caused-fix`; a handoff records
`--kind handoff --context-tokens <n>`, which is the only marker anywhere for a
handoff that was written and then worked past.

**Unknown is `-`, never a guess.** If the runtime does not hand you a turn count
or a byte size, leave the flag off. Both cost audits were reconstructed from
transcripts by hand precisely because nothing aggregated this; a fabricated
number would be worse than the gap it fills.
