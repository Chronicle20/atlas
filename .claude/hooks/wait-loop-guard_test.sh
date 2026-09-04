#!/usr/bin/env bash
# Tests for .claude/hooks/wait-loop-guard.sh.
#
# The allow-cases matter as much as the deny-cases: a guard that blocks
# legitimate process debugging is worse than the polling it prevents, so every
# allow-case below is a command a real debugging session issues.

set -u

HOOK="$(cd "$(dirname "$0")" && pwd)/wait-loop-guard.sh"
pass=0
fail=0

run() { printf '{"tool_input":{"command":%s}}' "$(printf '%s' "$1" | jq -Rs .)" | "$HOOK"; }

deny() {
    local out; out="$(run "$1")"
    if printf '%s' "$out" | grep -q '"permissionDecision":"deny"'; then
        pass=$((pass + 1))
    else
        fail=$((fail + 1)); echo "FAIL (expected deny): $1"
    fi
}

allow() {
    local out; out="$(run "$1")"
    if [ -z "$out" ]; then
        pass=$((pass + 1))
    else
        fail=$((fail + 1)); echo "FAIL (expected allow): $1"
    fi
}

echo "== denied: no-op turns =="
deny 'true'
deny '  true  '
deny ':'
deny 'true && true'
deny 'echo waiting'
deny 'echo "still waiting"'

echo "== denied: sleeping to wait =="
deny 'sleep 5'
deny 'sleep 30; cat /tmp/gate.log'
deny 'sleep 2 && ls'
deny 'while ! test -f /tmp/done; do sleep 5; done'
deny 'until [ -f /tmp/done ]; do sleep 10; done'
deny 'for i in $(seq 1 20); do sleep 3; done'

echo "== denied: broad process listing as a wait =="
deny 'pgrep -af "lint[.]sh"'
deny 'ps aux | grep verify'
deny 'ps -ef | grep -c golangci'

echo "== allowed: legitimate process debugging =="
allow 'ps -p 12345'
allow 'ps -o pid,etime,cmd -p 4242'
allow 'pgrep -af "lint[.]sh" | awk "{print \$1}" | xargs -r kill -9'
allow 'pkill -f stale-gate'
allow 'kill -9 4242'
allow 'docker ps'
allow 'kubectl get pods -n atlas-pr-1370'
allow 'top -b -n1 | head -20'
allow 'journalctl -u atlas --since "5 min ago" | tail -50'

echo "== allowed: ordinary work =="
allow 'go build ./...'
allow 'git status --short'
allow 'tools/verify.sh --facts --quick'
allow 'grep -rn "sleep" tools/ | head'
# `sleep` as CONTENT of a file being written is not a poll.
allow 'cat > /tmp/x.sh <<EOF
sleep 5
EOF'
# A test suite that legitimately contains a settle delay.
allow 'go test ./... -run TestRetryAfterSleep'

echo "== allowed: explicitly justified =="
allow 'POLL-JUSTIFIED: the deploy webhook has no completion signal
sleep 30'
allow 'sleep 60 # POLL-JUSTIFIED: external rate limit, no callback available'
allow 'true # POLL-JUSTIFIED: demonstrating the escape hatch'

echo "== file-tail polling: third identical read-only command denied =="
export TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT
runk() { jq -nc --arg s "$1" --arg a "$2" --arg c "$3" '{session_id:$s, agent_id:$a, tool_input:{command:$c}}' | "$HOOK"; }
denyk()  { out="$(runk "$@")"; if printf '%s' "$out" | grep -q '"permissionDecision":"deny"'; then pass=$((pass+1)); else fail=$((fail+1)); echo "FAIL (expected deny): $3"; fi; }
allowk() { out="$(runk "$@")"; if [ -z "$out" ]; then pass=$((pass+1)); else fail=$((fail+1)); echo "FAIL (expected allow): $3"; fi; }
allowk s1 "" 'tail -1 /tmp/gate.log'
allowk s1 "" 'tail  -1 /tmp/gate.log'          # 2nd (whitespace-normalized): a re-check is fine
denyk  s1 "" 'tail -1 /tmp/gate.log'           # 3rd: poll
denyk  s1 "" 'tail -1 /tmp/gate.log'           # stays denied
allowk s1 "" 'go build ./...'                  # different command resets the streak
allowk s1 "" 'tail -1 /tmp/gate.log'
allowk s1 "" 'tail -1 /tmp/gate.log'
denyk  s1 "" 'tail -1 /tmp/gate.log'
# Pipelines of readers count; other sessions/agents are keyed separately.
allowk s2 "" 'grep -c "EXIT=" /tmp/verify.log | tail -1'
allowk s2 "" 'grep -c "EXIT=" /tmp/verify.log | tail -1'
denyk  s2 "" 'grep -c "EXIT=" /tmp/verify.log | tail -1'
allowk s2 agentX 'grep -c "EXIT=" /tmp/verify.log | tail -1'
allowk s2 agentX 'wc -l /tmp/verify.log'
allowk s2 agentX 'wc -l /tmp/verify.log'
denyk  s2 agentX 'wc -l /tmp/verify.log'
# Work that happens to repeat is not a poll: it writes, chains, or is not a reader.
allowk s3 "" 'git status --short'; allowk s3 "" 'git status --short'; allowk s3 "" 'git status --short'
allowk s3 "" 'go test ./...'; allowk s3 "" 'go test ./...'; allowk s3 "" 'go test ./...'
allowk s3 "" 'cat a.go > b.go'; allowk s3 "" 'cat a.go > b.go'; allowk s3 "" 'cat a.go > b.go'
# Justified repeats pass.
allowk s4 "" 'tail -1 /tmp/x.log # POLL-JUSTIFIED: x'; allowk s4 "" 'tail -1 /tmp/x.log # POLL-JUSTIFIED: x'; allowk s4 "" 'tail -1 /tmp/x.log # POLL-JUSTIFIED: x'
# No session/agent id: stateless only, never denied for repetition.
allow 'tail -1 /tmp/gate.log'; allow 'tail -1 /tmp/gate.log'; allow 'tail -1 /tmp/gate.log'

echo
echo "passed: $pass  failed: $fail"
[ "$fail" -eq 0 ]
