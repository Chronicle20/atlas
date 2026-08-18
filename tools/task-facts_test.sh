#!/usr/bin/env bash
# task-facts_test.sh — hermetic tests for tools/task-facts.sh.
#
# task-facts.sh is a composer: its job is to call the existing resolvers and
# concatenate their answers, not to answer anything itself. So the assertions
# are about composition — does it pass the resolver's exit codes through, does
# it degrade gracefully when a component is absent, and is the result actually
# small enough to prepend to every dispatch brief.

set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
for s in task-facts.sh task-resolve.sh change-surfaces.sh; do
  [ -x "$HERE/$s" ] || { echo "FATAL: $HERE/$s not executable" >&2; exit 2; }
done

fails=0
assert_eq()  { if [ "$2" = "$3" ]; then echo "ok   - $1"; else echo "FAIL - $1 (want '$2', got '$3')" >&2; fails=$((fails+1)); fi; }
assert_has() { case "$3" in *"$2"*) echo "ok   - $1";; *) echo "FAIL - $1 (missing '$2')" >&2; fails=$((fails+1));; esac; }

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
repo="$tmp/repo"
mkdir -p "$repo/tools"
cp "$HERE/task-facts.sh" "$HERE/task-resolve.sh" "$HERE/change-surfaces.sh" "$repo/tools/"
git -C "$repo" init -q -b main
git -C "$repo" config user.email t@t.t
git -C "$repo" config user.name t
git -C "$repo" config commit.gpgsign false

mkdir -p "$repo/docs/tasks/task-100-alpha" "$repo/docs/tasks/task-101-beta"
echo prd > "$repo/docs/tasks/task-100-alpha/prd.md"
echo prd > "$repo/docs/tasks/task-101-beta/prd.md"
git -C "$repo" add -A
git -C "$repo" commit -qm base

# One task gets a real worktree with a fuller artifact set, mirroring a task
# mid-flight through the four phases.
git -C "$repo" worktree add -q -b task-102-gamma "$repo/.worktrees/task-102-gamma" main
wt="$repo/.worktrees/task-102-gamma"
mkdir -p "$wt/docs/tasks/task-102-gamma"
for a in prd.md design.md plan.md context.md; do echo "$a" > "$wt/docs/tasks/task-102-gamma/$a"; done
git -C "$wt" add -A
git -C "$wt" commit -qm gamma

cd "$repo"
FACTS="./tools/task-facts.sh"

# --- resolution and identity ------------------------------------------------

out="$($FACTS 102)"; rc=$?
assert_eq  "bare number resolves" "0" "$rc"
assert_has "names the task"       "task=task-102-gamma"      "$out"
assert_has "names the branch"     "branch=task-102-gamma"    "$out"
assert_has "names the worktree"   "worktree=$wt"             "$out"
assert_has "task_dir is repo-relative" "task_dir=docs/tasks/task-102-gamma" "$out"
assert_has "reports HEAD"         "head="                    "$out"

# --- artifacts --------------------------------------------------------------

assert_has "lists the phase artifacts that exist" "artifacts=prd.md,design.md,plan.md,context.md" "$out"
out2="$($FACTS 100)"
assert_has "lists only what exists" "artifacts=prd.md" "$out2"

# --- composed blocks --------------------------------------------------------

assert_has "embeds the change-surface block" "classification=" "$out"
assert_has "embeds the surface keys"         "rest_surface="   "$out"
assert_has "states the toolchain"            "toolchain="      "$out"
assert_has "states the go version"           "go_version="     "$out"

# verify.sh is absent from this fixture: the block must degrade to a named
# unavailable rather than omitting the key or failing.
assert_has "missing verify.sh degrades loudly" "applicable_guards=unavailable" "$out"

# --- exit codes pass through unchanged --------------------------------------
#
# Callers already branch on task-resolve.sh's 3 and 4; the composer must not
# swallow them.

$FACTS nosuchtask >/dev/null 2>&1; rc=$?
assert_eq "unknown task passes through exit 3" "3" "$rc"

# task-100-alpha and task-101-beta both contain "task-10" as a fragment.
$FACTS 'task-10' >/dev/null 2>&1; rc=$?
assert_eq "ambiguous identifier passes through exit 4" "4" "$rc"

# --- usage ------------------------------------------------------------------

$FACTS >/dev/null 2>&1; rc=$?
assert_eq "no identifier is a usage error" "2" "$rc"
$FACTS 102 --base >/dev/null 2>&1; rc=$?
assert_eq "--base without a value is a usage error" "2" "$rc"

# --- size -------------------------------------------------------------------
#
# The whole point is that this is cheap enough to prepend to every brief. If it
# grows past ~1KB it stops being a fact block and becomes another document.

bytes="$(printf '%s' "$out" | wc -c)"
if [ "$bytes" -lt 1200 ]; then
  echo "ok   - fact block is $bytes bytes (budget 1200)"
else
  echo "FAIL - fact block is $bytes bytes, over the 1200-byte budget" >&2; fails=$((fails+1))
fi

# --- output contract --------------------------------------------------------

assert_eq "output is key=value lines only" "" \
  "$(printf '%s\n' "$out" | grep -v '^[a-z_]*=' || true)"

echo
if [ "$fails" -eq 0 ]; then echo "task-facts_test.sh: all assertions passed"
else echo "task-facts_test.sh: $fails failure(s)" >&2; fi
[ "$fails" -eq 0 ]
