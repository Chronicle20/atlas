# Phase 5 — After Implementation

The four-phase flow ends at `/execute-task`. The work does not: PR validation,
live-environment testing, bug reproduction, regression investigation, and
follow-up fixes all happen afterwards, and nothing told those sessions to
delegate. This document is the missing phase.

It introduces no new context-clearing rule and requires nothing of the user. It
generalizes the handoff principle the execute loop already applies — see
[`docs/agent-dispatch.md`](agent-dispatch.md) §Context handoff — to the work
that follows implementation.

---

## Why

Measured across one task's post-PR phase — four sessions of live testing and bug
fixing:

| | Execute phase | Post-PR phase |
|---|---|---|
| Billed input | 800.0M (84.3% of the task) | **120.5M (12.7%)** |
| Main-thread share | 19% | **94%** |
| Subagents | 133 | **3** |
| Peak context | 349k across 8 sessions | 328k and 274k, solo |

Main-thread tokens are the expensive kind: the context grows monotonically and
every turn re-reads all of it. A turn at 200–330k costs several times the same
investigation inside a fresh subagent at 36–120k. Two of those sessions burned
76.7M between them with no subagents at all.

The habit was half-formed already. That same task folder contains four bug
diagnosis files — `bug-purchase-path-sets-assetid.md`,
`bug-world-transfer-client-crash.md`, `name-check-gap.md`,
`followup-check-time-eligibility.md`. **The artifact habit existed; the
delegation habit did not.** The one session that came closest to the right shape
opened by reading a bug file and dispatched two agents.

The rediscovery cost is visible too: one post-PR session re-grepped the repo for
`namechange|name_change` to relocate code the same branch had written 48 hours
earlier, and another touched `plan.md` four times for 19.6 KB to re-establish
what the plan had said.

---

## The loop

**Reproduce inline. Diagnose into a file. Delegate the fix. Verify fresh.**

### 1. Reproduce — stay in your own context

Reproduction is interactive: the operator is in the loop with a live client, a
running cluster, round-trip latency matters more than tokens, and each step
depends on what the last one showed. Do this yourself. Do not delegate it.

Read pod logs first — see [`docs/observability.md`](observability.md). Confirm
the tenant and client version before anything else; the wrong version sends the
whole investigation down the wrong path.

### 2. Write the diagnosis to `docs/tasks/<task>/bug-<slug>.md`

Before dispatching anything. This is the boundary: everything after it must be
resumable from repository state plus this file.

```markdown
# bug: <one-line symptom>

**Reproduced:** <tenant, client version, exact steps>
**Observed:** <what happens, with the log line / packet / error verbatim>
**Expected:** <what should happen, and where that is specified — PRD/FR, plan task>
**Root cause:** <what you established, with file:line>  — or: "not yet established; <what is ruled out>"

## Fix

- `path/to/file.go:120` — <what changes here>
- `path/to/other_test.go` — <the test that must fail before and pass after>

## Not yet answered

- <anything the fix agent must decide, and what it should do if unsure>
```

The `## Fix` section is a `### Files` inventory by another name, and it does the
same job: it removes the implementer's discovery phase — the phase that inflates
context before a single edit happens. You already know these paths from
reproducing; the fix agent would otherwise pay to rediscover them, at its own
context depth, on top of what you already paid at yours.

If the root cause is not established, say so explicitly and name what is ruled
out. An honest "not yet established" is a fine brief; a guessed root cause is
not.

### 3. Delegate the fix to a fresh agent

```text
subagent_type: atlas-implementer
model: sonnet
brief: docs/tasks/<task>/bug-<slug>.md
```

The bug file is the brief. Do not restate it in the dispatch prompt — add only
what the file cannot carry: the worktree path, and any ruling you have made
since writing it.

This is the step the audit found missing. It is also where the saving is: the
fix agent starts near 36k instead of inheriting your 300k.

### 4. Verify in a clean context

`atlas-verifier` (`model: haiku`) for the gate. `atlas-reviewer` (`model:
sonnet`) if the fix crosses a service boundary or touches a contract — the gate
cannot see a seam defect.

Launch the gate backgrounded and keep going; do not poll it.
`.claude/hooks/wait-loop-guard.sh` will refuse the poll anyway.

### 5. Ledger it

```sh
tools/agent-ledger.sh append <task> --unit "bug-<slug>" --agent-type atlas-implementer \
  --model sonnet --status <status> --commit <sha>
```

So the next audit can answer "what did the post-PR phase actually cost" without
reconstructing transcripts.

---

## When to hand off your own context

The same question as every other durable boundary: **does the next unit depend
materially on this conversation's history, or only on repository state plus a
short written diagnosis?**

After a bug file is written, the answer is almost always "repository state".
Once you have written the diagnosis, the reproduction conversation that produced
it is no longer load-bearing.

Concretely, in a debugging session:

- **After each bug file is written and its fix dispatched**, ask the question.
  If the next bug is unrelated to the one you just fixed, it is a fresh unit —
  dispatch it against its own bug file rather than continuing to accumulate.
- **Past ~150k**, stop starting new investigations in this context. Write the
  remaining leads into the task folder and hand off. The `/execute-task` ceiling
  applies here for the same reason it applies there: the marker on disk is
  meaningless if the session that wrote it keeps going.

`/fix-pr-bug` mechanizes steps 2–5 for a single bug.

---

## What does not change

- **Reproduction stays inline.** An over-delegated interactive debugging session
  is worse than an expensive one.
- **The bug-file habit is already right** — this document adds the delegation
  step after it, not a new artifact format.
- **`/execute-task`'s ceiling and ledger are unchanged.** Phase 5 borrows them;
  it does not redefine them.
