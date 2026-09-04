#!/usr/bin/env bash
# Tests for .claude/hooks/context-handoff-guard.sh.
#
# The guard denies NEW work units past the controller handoff threshold and
# allows finishing dispatches, subagents, justified exceptions, and everything
# below the threshold. Each case sets the turn-budget counter it needs.

set -u

HOOK="$(cd "$(dirname "$0")" && pwd)/context-handoff-guard.sh"
export TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT
state="$TMPDIR/claude-turn-budget"
mkdir -p "$state"
pass=0
fail=0

# run <count> <session> <agent_id> <subagent_type> <prompt>
run() {
    [ -n "$2" ] && printf '%s' "$1" > "$state/session-$2"
    jq -nc --arg s "$2" --arg a "$3" --arg t "$4" --arg p "$5" \
        '{session_id:$s, agent_id:$a, tool_name:"Agent", tool_input:{subagent_type:$t, prompt:$p}}' | "$HOOK"
}
deny()  { out="$(run "$@")"; if printf '%s' "$out" | grep -q '"permissionDecision":"deny"'; then pass=$((pass+1)); else fail=$((fail+1)); echo "FAIL (expected deny): count=$1 type=$4"; fi; }
allow() { out="$(run "$@")"; if [ -z "$out" ]; then pass=$((pass+1)); else fail=$((fail+1)); echo "FAIL (expected allow): count=$1 type=$4"; fi; }

echo "== below threshold: everything passes =="
allow 10 s1 "" task-implementer "Implement Task 3"
allow 59 s1 "" general-purpose "survey the seam"

echo "== past threshold: new units denied =="
deny 60  s2 "" task-implementer "Implement Task 5"
deny 90  s2 "" general-purpose "investigate"
deny 120 s2 "" Explore "find the handlers"
deny 75  s2 "" packet-implementer "implement the codec"
deny 75  s2 "" Plan "plan the next stage"

echo "== past threshold: finishing dispatches allowed =="
allow 90 s3 "" task-reviewer "Review commits a..b"
allow 90 s3 "" task-verifier "run tools/verify.sh --quick"
allow 90 s3 "" plan-adherence-reviewer "audit plan.md"
allow 90 s3 "" backend-guidelines-reviewer "audit atlas-account"
allow 90 s3 "" service-documentation "document atlas-maps"

echo "== past threshold: justified exception allowed =="
allow 90 s4 "" task-implementer "CONTEXT-JUSTIFIED: the fix depends on the live k3s trace only this session holds. Implement..."

echo "== subagents and unkeyed calls are never blocked =="
allow 200 s5 agent-abc task-implementer "child dispatch"
out="$(jq -nc '{tool_name:"Agent", tool_input:{subagent_type:"task-implementer", prompt:"x"}}' | "$HOOK")"
if [ -z "$out" ]; then pass=$((pass+1)); else fail=$((fail+1)); echo "FAIL (expected allow): no session id"; fi

echo "== no counter file: silent =="
out="$(jq -nc '{session_id:"nofile", tool_name:"Agent", tool_input:{subagent_type:"task-implementer", prompt:"x"}}' | "$HOOK")"
if [ -z "$out" ]; then pass=$((pass+1)); else fail=$((fail+1)); echo "FAIL (expected allow): no counter"; fi

echo
echo "passed: $pass  failed: $fail"
[ "$fail" -eq 0 ]
