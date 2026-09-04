#!/usr/bin/env bash
# PostToolUse hook — raise the context-handoff question at commit boundaries.
#
# Context cost is turn count x context size: 50 turns at 190k cost ~10x the same
# 50 turns at 19k. The cheapest lever is not doing the next unit of work in a
# context that has nothing to do with it. But the moment to decide that is
# exactly the moment you are mid-flow and least inclined to stop, so this hook
# makes the boundary visible instead of leaving it to notice itself.
#
# It does NOT decide the handoff — "does the next work depend on this history?"
# is not mechanically observable. It asks, once per commit, past a floor.
#
# The floor matters. Below it, a fresh agent re-discovers files the current
# context already holds and you pay for discovery twice. FLOOR is a tool-call
# count used as a proxy for context size (the hook cannot see tokens).
#
# Reuses the per-session counter written by turn-budget.sh — same state dir,
# same session_id keying. If that hook has not run, the count reads 0 and this
# hook stays silent, which is the correct failure mode.
#
# Silent on the happy path. Always exits 0 — a nag must never break a session.
# The ESCALATE tier below is advisory here; context-handoff-guard.sh reads the
# same ESCALATE value and DENIES new-unit Agent dispatches past it. Change the
# threshold in this file only.
#
# Referenced from CLAUDE.md "Context Handoff". The floor lives in this file
# only; change it here.

set -u

FLOOR=40
# Second tier: the unconditional ~150k controller threshold from
# .claude/commands/execute-task.md Step 4e, expressed in the only unit this
# hook can see. Measured across the task-231 execute chain, a controller's
# context grows ~2.1k tokens per tool call over the 23k standing-prompt floor
# (86 calls -> 210-224k; 113 -> 236k; 145 -> 309k), so (150k - 23k) / 2.1k
# lands near 60 calls.
ESCALATE=60

# No stdin -> can't decide, stay silent.
[ -t 0 ] && exit 0

payload="$(cat)" || exit 0

command_run="$(printf '%s' "$payload" | jq -r '.tool_input.command // ""' 2>/dev/null)"
[ -n "$command_run" ] || exit 0

# Only fire on a commit. Exclude `git commit --dry-run` and the various read-only
# subcommands that merely contain the word (log, show, rev-parse ...).
printf '%s' "$command_run" | grep -Eq 'git[[:space:]]+(-[^[:space:]]+[[:space:]]+)*commit' || exit 0
printf '%s' "$command_run" | grep -Eq -- '--dry-run' && exit 0

# Skip a commit that did not actually land. PostToolUse fires on failures too,
# and "nothing to commit" / hook rejections are not boundaries.
response="$(printf '%s' "$payload" | jq -r '
  (.tool_response // empty) |
  if type == "string" then . else (.stdout // "") + "\n" + (.stderr // "") end
' 2>/dev/null)"
if [ -n "$response" ]; then
    printf '%s' "$response" |
        grep -Eqi 'nothing to commit|no changes added|working tree clean|pre-commit hook|commit failed' &&
        exit 0
fi

session="$(printf '%s' "$payload" | jq -r '.session_id // ""' 2>/dev/null)"
session="$(printf '%s' "$session" | tr -cd 'A-Za-z0-9._-')"
[ -n "$session" ] || exit 0

# Same state dir and key as turn-budget.sh — deliberately shared, not duplicated.
#
# The key MUST carry the "session-" prefix. turn-budget.sh writes
# "$state_dir/session-$session" for controllers (and "agent-$id" for subagents,
# so the two cannot collide); reading "$state_dir/$session" finds no file, reads
# 0, and silently fails the FLOOR test on every commit. This hook did exactly
# that and never fired once — which is why the task-231 controller ran to 309k
# with no boundary prompt. An unenforced rule is not a rule.
counter="${TMPDIR:-/tmp}/claude-turn-budget/session-$session"
count=0
[ -f "$counter" ] && count="$(cat "$counter" 2>/dev/null || echo 0)"
case "$count" in
    ''|*[!0-9]*) count=0 ;;
esac

# Under the floor the handoff is net-negative. Stay silent.
[ "$count" -ge "$FLOOR" ] || exit 0

if [ "$count" -ge "$ESCALATE" ]; then
    body="Commit landed at $count tool calls — you are past the handoff threshold.

At this call count the context is roughly 150k tokens (~2.1k per call over the
23k floor). CLAUDE.md and .claude/commands/execute-task.md Step 4e both say:
never carry the controller past ~150k tokens — unconditionally, no carve-out
for tasks remaining. That is here. (Step 4e also escalates after 4 completed
plan tasks regardless of token count; this hook can only see tool calls, so if
you have completed 4+ plan tasks in this conversation, treat that as
independently past the threshold even if the call count above reads low.)

The default is now HAND OFF, not continue. Continue only if the next step reads
state that exists nowhere but this conversation — and if so, say why.

Every further turn re-reads this whole context. On the task-231 chain a
controller ran to 309k; its last 44 turns cost 12.2M billed tokens, 38% of that
session's total, for work that was fully resumable from the ledger.

  1. Write the diagnosis down (one paragraph into the task folder: what was
     found, what it means, what the next step is). Reasoning that survives only
     in this conversation does not survive the handoff.
  2. Dispatch the next unit to a fresh agent with an explicit brief
     (task-implementer + task-verifier for code work; tools/task-brief.sh
     generates the brief — do not assemble one by hand out of plan.md).
  3. If the next unit is genuinely controller-shaped, say so plainly and let
     the user decide whether to /clear."
else
    body="Commit landed at $count tool calls — a durable boundary.

Before continuing, answer one question: does the next unit of work depend
materially on THIS conversation's history, or only on repository state plus
what is already written down?

If it can be resumed from repo state + task reports + a short diagnosis, hand
off now rather than continuing here. Turn cost scales with the context you are
carrying, and work that does not need this history is paying for all of it.

Handing off means DELEGATING, not clearing — you cannot /clear yourself:
  1. Write the diagnosis down first (one paragraph into the task folder: what
     was found, what it means, what the next step is). Reasoning that survives
     only in this conversation does not survive the handoff.
  2. Dispatch the next unit to a fresh agent with an explicit brief
     (task-implementer + task-verifier for code work; tools/task-brief.sh
     generates the brief — do not assemble one by hand out of plan.md).
  3. Only if the next unit is genuinely controller-shaped, say so and let the
     user decide whether to /clear.

If the next work DOES depend on this history — you are mid-debug, the next step
reads state only this conversation holds — continue. This is a question, not a
cap.

See CLAUDE.md \"Context Handoff\"."
fi

cat <<EOF
{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"$(printf '%s' "$body" | sed 's/\\/\\\\/g; s/"/\\"/g' | sed ':a;N;$!ba;s/\n/\\n/g')"}}
EOF
