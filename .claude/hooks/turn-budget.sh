#!/usr/bin/env bash
# PostToolUse hook — per-agent tool-call budget.
#
# Context cost scales with turn count: every turn re-reads the whole context,
# so a 600-turn implementer costs far more than four 150-turn ones. This hook
# counts tool calls and nags at the thresholds.
#
# Counting is keyed per AGENT, not per session. A dispatched subagent shares
# its parent's session_id, transcript_path, cwd and prompt_id — keying on
# session_id charged every implementer for the controller's calls, so a fresh
# implementer could hit the cap on its first tool call and hand back a bogus
# PARTIAL. The harness identifies a subagent by `agent_id` (plus `agent_type`),
# which is absent in the main loop, so:
#
#   agent_id present -> a dispatched subagent; its own counter, its own budget
#   agent_id absent  -> the controller session; counted separately
#
# Verified live on 2.1.232 by capturing both payloads. Builds that do not emit
# agent_id fall back to session_id, i.e. the old behavior.
#
# Silent on the happy path. Prints a system-reminder as additionalContext at
# the soft warning, the hard cap, and every REPEAT calls past the cap. Always
# exits 0 — a counter must never break a session.
#
# The cap is stated once here and referenced from CLAUDE.md,
# .claude/agents/atlas-implementer.md, and .claude/hooks/turn-budget-guard.sh
# (which parses CAP out of this file at runtime). Change it in this file only.
#
# This hook only NAGS. Enforcement lives in the PreToolUse companion
# turn-budget-guard.sh, which denies further tool calls past CAP+5 for
# subagents. Both are needed: this one gives the agent the chance to stop
# voluntarily, the guard handles the ones that don't.

set -u

WARN=100
CAP=120
REPEAT=20

state_dir="${TMPDIR:-/tmp}/claude-turn-budget"
mkdir -p "$state_dir" 2>/dev/null || exit 0

# Prune counters from agents that finished more than a day ago.
find "$state_dir" -type f -mtime +1 -delete 2>/dev/null || true

payload=""
if [ ! -t 0 ]; then
    payload="$(cat)" || exit 0
fi

# First occurrence wins: the harness emits its own fields ahead of tool_input,
# so a tool argument that happens to contain the same key cannot shadow them.
json_str_field() {
    printf '%s' "$payload" |
        grep -o "\"$1\"[[:space:]]*:[[:space:]]*\"[^\"]*\"" |
        head -1 |
        sed 's/.*:[[:space:]]*"//; s/"$//'
}

session="$(json_str_field session_id)"
[ -n "$session" ] || exit 0

agent="$(json_str_field agent_id)"

# Both ids come from the harness; strip anything else defensively so a counter
# key can never escape the state directory.
sanitize() { printf '%s' "$1" | tr -cd 'A-Za-z0-9._-'; }

session="$(sanitize "$session")"
agent="$(sanitize "$agent")"
[ -n "$session" ] || exit 0

if [ -n "$agent" ]; then
    key="agent-$agent"
    role="subagent"
else
    key="session-$session"
    role="controller"
fi

counter="$state_dir/$key"
count=0
[ -f "$counter" ] && count="$(cat "$counter" 2>/dev/null || echo 0)"
case "$count" in
    ''|*[!0-9]*) count=0 ;;
esac
count=$((count + 1))
printf '%s' "$count" > "$counter" 2>/dev/null || exit 0

emit=""
if [ "$count" -eq "$WARN" ]; then
    emit="soft"
elif [ "$count" -eq "$CAP" ]; then
    emit="cap"
elif [ "$count" -gt "$CAP" ] && [ $(((count - CAP) % REPEAT)) -eq 0 ]; then
    emit="over"
fi
[ -n "$emit" ] || exit 0

if [ "$role" = "controller" ]; then
    # The controller's budget is the dispatch loop, not a single task, so the
    # soft warning does not apply to it. Past the cap, the relevant guidance is
    # the context-handoff rule, not PARTIAL.
    [ "$emit" = "soft" ] && exit 0
    body="You are the controller session, at $count tool calls.

This is not the implementer cap — your budget is the dispatch loop. It is a
prompt to check the context-handoff rule in CLAUDE.md: if the next unit of work
depends only on repository state, write the diagnosis down and hand it to a
fresh agent rather than carrying this context further."
else
    case "$emit" in
        soft)
            body="You are at $count tool calls (soft warning at $WARN, cap at $CAP).

Start converging now. Commit what already works, and plan to stop at $CAP
rather than pushing through."
            ;;
        cap)
            body="You are at $count tool calls — the implementer cap.

STOP taking on new work. Commit what works, then report status PARTIAL with
(a) what is done and committed, (b) what remains, file by file, (c) the exact
next step. Do not push through the cap; the controller will dispatch a
continuation with fresh context. Reporting PARTIAL at the cap is the correct
outcome, not a failure."
            ;;
        over)
            body="You are at $count tool calls — $((count - CAP)) past the implementer cap of $CAP.

You should already have reported PARTIAL. Commit what works and report now."
            ;;
    esac
fi

cat <<EOF
{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"$(printf '%s' "$body" | sed 's/\\/\\\\/g; s/"/\\"/g' | sed ':a;N;$!ba;s/\n/\\n/g')"}}
EOF
