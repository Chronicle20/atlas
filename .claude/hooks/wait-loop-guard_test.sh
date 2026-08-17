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

echo
echo "passed: $pass  failed: $fail"
[ "$fail" -eq 0 ]
