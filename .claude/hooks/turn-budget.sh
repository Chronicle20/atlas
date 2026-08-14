#!/usr/bin/env bash
# PostToolUse hook — per-session tool-call budget.
#
# Context cost scales with turn count: every turn re-reads the whole context,
# so a 600-turn implementer costs far more than four 150-turn ones. This hook
# counts tool calls per session (subagents get their own session_id, so it
# fires inside dispatched implementers too) and nags at the thresholds.
#
# Silent on the happy path. Prints a system-reminder as additionalContext at
# the soft warning, the hard cap, and every REPEAT calls past the cap. Always
# exits 0 — a counter must never break a session.
#
# The cap is stated once here and referenced from CLAUDE.md and
# .claude/agents/atlas-implementer.md. Change it in this file only.

set -u

WARN=100
CAP=120
REPEAT=20

state_dir="${TMPDIR:-/tmp}/claude-turn-budget"
mkdir -p "$state_dir" 2>/dev/null || exit 0

# Prune counters from sessions that ended more than a day ago.
find "$state_dir" -type f -mtime +1 -delete 2>/dev/null || true

payload=""
if [ ! -t 0 ]; then
    payload="$(cat)" || exit 0
fi

session="$(printf '%s' "$payload" |
    sed -n 's/.*"session_id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' |
    head -1)"
[ -n "$session" ] || exit 0

# session_id is a uuid from the harness; strip anything else defensively so it
# can never escape the state directory.
session="$(printf '%s' "$session" | tr -cd 'A-Za-z0-9._-')"
[ -n "$session" ] || exit 0

counter="$state_dir/$session"
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

case "$emit" in
    soft)
        body="You are at $count tool calls in this session (soft warning at $WARN, cap at $CAP).

If you are an implementer subagent: start converging now. Commit what already
works, and plan to stop at $CAP rather than pushing through.

If you are the controller session: this is informational — your budget is the
dispatch loop, not a single task."
        ;;
    cap)
        body="You are at $count tool calls — the implementer cap.

If you are an implementer subagent: STOP taking on new work. Commit what
works, then report status PARTIAL with (a) what is done and committed,
(b) what remains, file by file, (c) the exact next step. Do not push through
the cap; the controller will dispatch a continuation with fresh context.
Reporting PARTIAL at the cap is the correct outcome, not a failure.

If you are the controller session: informational only."
        ;;
    over)
        body="You are at $count tool calls — $((count - CAP)) past the implementer cap of $CAP.

If you are an implementer subagent you should already have reported PARTIAL.
Commit what works and report now."
        ;;
esac

cat <<EOF
{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"$(printf '%s' "$body" | sed 's/\\/\\\\/g; s/"/\\"/g' | sed ':a;N;$!ba;s/\n/\\n/g')"}}
EOF
