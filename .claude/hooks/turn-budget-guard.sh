#!/usr/bin/env bash
# PreToolUse hook — make the implementer tool-call cap BINDING.
#
# `.claude/hooks/turn-budget.sh` counts tool calls and nags at 100 and 120. It
# is a PostToolUse hook, so it can only inject text — and text is advice the
# model may decline. On the task-231 execute chain it was declined three times:
# implementers ran to 127, 127 and 130 calls, and the ledger recorded "cap
# reached at 130 tool calls" as a fait accompli rather than a stop. Those eight
# over-long agents billed 130.8M input tokens between them — a third of all
# subagent spend on the branch — because an agent's per-turn cost grows with its
# own context, so the last 60 turns of a 120-turn agent cost roughly twice what
# the first 60 did.
#
# This hook is the enforcement half. Past CAP + GRACE it DENIES further tool
# calls, with a narrow allowlist for the commands an agent needs to land its
# work and hand back cleanly. The grace band is deliberate: it lets a
# well-behaved agent wrap up voluntarily after the PostToolUse nag, so the deny
# only fires on the ones that ignored it.
#
# Scope: SUBAGENTS ONLY. The controller is never blocked — its budget is the
# dispatch loop, not a single task, and denying it mid-plan would strand the
# whole run. The controller's over-cap guidance stays advisory in turn-budget.sh.
#
# The cap is read out of turn-budget.sh at runtime so it remains stated exactly
# once, per that file's own note. Change it there, not here.
#
# Silent on the happy path. Always exits 0 — a guard must never break a session
# by failing itself.

set -u

GRACE=5   # calls past CAP before the deny engages

# No stdin -> can't decide, allow.
[ -t 0 ] && exit 0
input="$(cat)" || exit 0

hook_dir="$(cd "$(dirname "$0")" && pwd -P)"
budget_hook="$hook_dir/turn-budget.sh"

# Single source of truth for the cap: parse it out of the counting hook.
CAP="$(sed -n 's/^CAP=\([0-9][0-9]*\).*/\1/p' "$budget_hook" 2>/dev/null | head -1)"
case "${CAP:-}" in ''|*[!0-9]*) CAP=120 ;; esac

limit=$((CAP + GRACE))

agent="$(printf '%s' "$input" | jq -r '.agent_id // ""' 2>/dev/null)"
sanitize() { printf '%s' "$1" | tr -cd 'A-Za-z0-9._-'; }
agent="$(sanitize "$agent")"

# No agent_id -> controller session. Never block it.
[ -n "$agent" ] || exit 0

state_dir="${TMPDIR:-/tmp}/claude-turn-budget"
counter="$state_dir/agent-$agent"
[ -f "$counter" ] || exit 0

count="$(cat "$counter" 2>/dev/null || echo 0)"
case "$count" in ''|*[!0-9]*) count=0 ;; esac

# turn-budget.sh increments AFTER each call, so at PreToolUse time the counter
# holds the number of calls already completed.
[ "$count" -ge "$limit" ] || exit 0

# --- allowlist: what an agent may still do in order to stop cleanly ----------
tool="$(printf '%s' "$input" | jq -r '.tool_name // ""' 2>/dev/null)"
cmd="$(printf '%s' "$input"  | jq -r '.tool_input.command // ""' 2>/dev/null)"
path="$(printf '%s' "$input" | jq -r '.tool_input.file_path // ""' 2>/dev/null)"

case "$tool" in
    Bash)
        # Landing committed work, and reading enough git state to do it safely.
        case "$cmd" in
            git\ add*|git\ commit*|git\ status*|git\ diff*|git\ log*) exit 0 ;;
        esac
        ;;
    Write|Edit)
        # Writing the hand-back report itself.
        case "$path" in
            *-report.md|*/.superpowers/sdd/*) exit 0 ;;
        esac
        ;;
esac

reason="Tool-call budget exhausted: $count calls, $GRACE past the implementer cap of $CAP.

You were warned at $CAP by the turn-budget hook and did not stop. Further work
is denied because an agent's cost grows with its own context — the turns past
the cap are the most expensive ones in the run, and the controller can do this
work far more cheaply in a fresh context.

STOP NOW and hand back:
  1. Commit whatever works  (\`git add\` / \`git commit\` are still allowed)
  2. Report status PARTIAL with (a) what is done and committed, file by file,
     (b) what remains, (c) the exact next step for the continuation agent.

Writing your report file is still allowed. Reporting PARTIAL at the cap is the
contracted, correct outcome — see .claude/agents/task-implementer.md — not a
failure. The controller will dispatch a continuation with fresh context."

jq -nc --arg r "$reason" '{
  hookSpecificOutput: {
    hookEventName: "PreToolUse",
    permissionDecision: "deny",
    permissionDecisionReason: $r
  }
}' 2>/dev/null || exit 0

exit 0
