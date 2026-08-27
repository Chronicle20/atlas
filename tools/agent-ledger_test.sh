#!/usr/bin/env bash
# agent-ledger_test.sh — hermetic tests for tools/agent-ledger.sh.
#
# The invariant the whole thing rests on: an unmeasured field is `-`, never a
# number. Everything else is bookkeeping.

set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
for s in agent-ledger.sh task-resolve.sh; do
  [ -x "$HERE/$s" ] || { echo "FATAL: $HERE/$s not executable" >&2; exit 2; }
done

fails=0
assert_eq()  { if [ "$2" = "$3" ]; then echo "ok   - $1"; else echo "FAIL - $1 (want '$2', got '$3')" >&2; fails=$((fails+1)); fi; }
assert_has() { case "$3" in *"$2"*) echo "ok   - $1";; *) echo "FAIL - $1 (missing '$2')" >&2; fails=$((fails+1));; esac; }

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
repo="$tmp/repo"
mkdir -p "$repo/tools"
cp "$HERE/agent-ledger.sh" "$HERE/task-resolve.sh" "$repo/tools/"
git -C "$repo" init -q -b main
git -C "$repo" config user.email t@t.t
git -C "$repo" config user.name t
git -C "$repo" config commit.gpgsign false
mkdir -p "$repo/docs/tasks/task-300-ledger"
echo prd > "$repo/docs/tasks/task-300-ledger/prd.md"
git -C "$repo" add -A
git -C "$repo" commit -qm base

cd "$repo"
L="./tools/agent-ledger.sh"
ledger="$($L path 300)"
assert_eq "path resolves inside the task folder" \
  "$repo/docs/tasks/task-300-ledger/agent-ledger.tsv" "$ledger"

# --- append -----------------------------------------------------------------

$L append 300 --unit "Task 1" --agent-type task-implementer --model sonnet \
  --turns 41 --tool-calls 88 --return-bytes 1145 --status DONE --commit abc1234 >/dev/null
assert_eq "ledger created" "0" "$([ -f "$ledger" ] && echo 0 || echo 1)"
assert_has "header written once" "agent_type" "$(head -1 "$ledger")"
assert_eq "one data row" "1" "$(($(wc -l < "$ledger") - 1))"

row="$(sed -n '2p' "$ledger")"
assert_has "row carries the unit"    "Task 1"            "$row"
assert_has "row carries the type"    "task-implementer" "$row"
assert_has "row carries the commit"  "abc1234"           "$row"

# --- unknown fields stay '-' ------------------------------------------------
#
# The one invariant. tool_result_bytes and context_tokens were never passed.

assert_eq "unmeasured fields are '-', not 0" "-" "$(printf '%s' "$row" | cut -f8)"
assert_eq "verdict absent on an implementer row" "-" "$(printf '%s' "$row" | cut -f12)"
$L append 300 --unit "Task 2" --agent-type task-implementer --turns "" >/dev/null
assert_eq "an explicitly empty value is also '-'" "-" "$(sed -n '3p' "$ledger" | cut -f6)"

# --- reviewer rows ----------------------------------------------------------

$L append 300 --unit "Task 1" --agent-type task-reviewer --model sonnet \
  --return-bytes 420 --verdict APPROVED --caused-fix no \
  --artifact docs/tasks/task-300-ledger/reviews/task-1.md >/dev/null
$L append 300 --unit "Task 2" --agent-type task-reviewer --model sonnet \
  --return-bytes 780 --verdict CHANGES_REQUIRED --caused-fix yes >/dev/null
$L append 300 --unit "Task 3" --agent-type task-reviewer --model sonnet \
  --return-bytes 610 --verdict APPROVED_WITH_FINDINGS --caused-fix no >/dev/null

# --- handoff rows -----------------------------------------------------------

$L append 300 --kind handoff --unit "after Task 4" --context-tokens 152000 >/dev/null
assert_eq "a handoff row needs no agent type" "0" "$?"

# --- summary ----------------------------------------------------------------

sum="$($L summary 300)"
assert_has "summary counts agents"            "agents=5"                    "$sum"
assert_has "summary breaks down by type"      "task-reviewer: n=3"         "$sum"
assert_has "summary reports median return"    "median_return_bytes=610"     "$sum"
assert_has "summary counts verdicts"          "CHANGES_REQUIRED=1"          "$sum"
assert_has "summary counts approvals"         "APPROVED=1"                  "$sum"
assert_has "summary reports fix causation"    "reviews_that_caused_a_fix=1 of 3" "$sum"
assert_has "summary reports handoffs"         "handoffs=1"                  "$sum"
assert_has "summary reports handoff context"  "median_context_tokens=152000" "$sum"

# The implementer's median return must not be polluted by the reviewer rows —
# that separation is the entire before/after signal both audits asked for.
assert_has "implementer median is its own" "task-implementer: n=2" "$sum"

# --- input hygiene ----------------------------------------------------------

before="$(($(wc -l < "$ledger") - 1))"
$L append 300 --unit "Task 5" --agent-type task-implementer \
  --status "DONE
with a newline	and a tab" >/dev/null
assert_eq "an embedded newline/tab adds exactly one row" "$((before + 1))" \
  "$(($(wc -l < "$ledger") - 1))"
assert_eq "and the field is flattened, not split" "DONE with a newline and a tab" \
  "$(tail -1 "$ledger" | cut -f10)"
assert_eq "so the row still has 15 columns" "15" \
  "$(tail -1 "$ledger" | awk -F'\t' '{print NF}')"

# --- usage ------------------------------------------------------------------

$L append 300 --agent-type task-implementer >/dev/null 2>&1
assert_eq "missing --unit is a usage error" "2" "$?"
$L append 300 --unit x >/dev/null 2>&1
assert_eq "missing --agent-type on an agent row is a usage error" "2" "$?"
$L append 300 --unit x --agent-type y --bogus z >/dev/null 2>&1
assert_eq "unknown field is a usage error" "2" "$?"
$L append nosuchtask --unit x --agent-type y >/dev/null 2>&1
assert_eq "unknown task passes through exit 3" "3" "$?"

echo
if [ "$fails" -eq 0 ]; then echo "agent-ledger_test.sh: all assertions passed"
else echo "agent-ledger_test.sh: $fails failure(s)" >&2; fi
[ "$fails" -eq 0 ]
