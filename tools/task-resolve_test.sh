#!/usr/bin/env bash
# task-resolve_test.sh — hermetic regression tests for tools/task-resolve.sh
# and tools/task-brief.sh.
#
# Builds a throwaway git repo with real linked worktrees so the assertions
# never depend on the live repo's evolving task set. Run directly:
#
#     tools/task-resolve_test.sh
#
# Exits non-zero if any assertion failed.

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOLVE="$HERE/task-resolve.sh"
BRIEF="$HERE/task-brief.sh"
[ -x "$RESOLVE" ] || { echo "FATAL: $RESOLVE not executable" >&2; exit 2; }
[ -x "$BRIEF" ]   || { echo "FATAL: $BRIEF not executable" >&2; exit 2; }

fails=0
assert_eq() { # desc want got
  if [ "$2" = "$3" ]; then
    echo "ok   - $1"
  else
    echo "FAIL - $1 (want '$2', got '$3')" >&2
    fails=$((fails + 1))
  fi
}

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

repo="$tmp/repo"
mkdir -p "$repo"
git -C "$repo" init -q -b main
git -C "$repo" config user.email t@t.t
git -C "$repo" config user.name t
git -C "$repo" config commit.gpgsign false

# Three tasks on main.
for t in task-001-alpha task-002-beta task-205-player-trade; do
  mkdir -p "$repo/docs/tasks/$t"
  echo "prd for $t" > "$repo/docs/tasks/$t/prd.md"
done
git -C "$repo" add -A
git -C "$repo" commit -qm "tasks 001, 002, 205"

# Two worktrees. Each branches from the commit above, so each carries its OWN
# full copy of docs/tasks/ — including task-205's folder. That duplication is
# the thing the old `.worktrees/*/docs/tasks/task-*` glob multiplied out.
for t in task-010-gamma task-011-delta; do
  git -C "$repo" worktree add -q -b "$t" "$repo/.worktrees/$t" main
  mkdir -p "$repo/.worktrees/$t/docs/tasks/$t"
  echo "prd for $t" > "$repo/.worktrees/$t/docs/tasks/$t/prd.md"
  git -C "$repo/.worktrees/$t" add -A
  git -C "$repo/.worktrees/$t" commit -qm "add $t"
done

cd "$repo"

# --- resolution ----------------------------------------------------------

got="$("$RESOLVE" task-001-alpha | cut -f1)"
assert_eq "exact task ID resolves" "task-001-alpha" "$got"

got="$("$RESOLVE" 2 | cut -f1)"
assert_eq "bare number resolves (2 -> 002)" "task-002-beta" "$got"

got="$("$RESOLVE" 002 | cut -f1)"
assert_eq "zero-padded number resolves" "task-002-beta" "$got"

got="$("$RESOLVE" task-02 | cut -f1)"
assert_eq "task-NN prefix form resolves" "task-002-beta" "$got"

got="$("$RESOLVE" player-trade | cut -f1)"
assert_eq "slug fragment resolves" "task-205-player-trade" "$got"

# THE REGRESSION. task-205's folder physically exists in three places: main
# plus both worktrees. The old glob returned all three; resolution must yield
# exactly one, and it must be main's — no worktree is named task-205-*, so no
# worktree owns it.
got="$("$RESOLVE" 205 | wc -l | tr -d ' ')"
assert_eq "duplicated task resolves to exactly one home" "1" "$got"

got="$("$RESOLVE" 205 | cut -f3)"
assert_eq "unowned task resolves to the main repo root" "$repo" "$got"

# A task whose worktree exists resolves to the WORKTREE, not to any copy.
got="$("$RESOLVE" 010 | cut -f3)"
assert_eq "owned task resolves to its own worktree" "$repo/.worktrees/task-010-gamma" "$got"

got="$("$RESOLVE" 010 | cut -f2)"
assert_eq "owned task dir is inside its worktree" \
  "$repo/.worktrees/task-010-gamma/docs/tasks/task-010-gamma" "$got"

# Every task appears exactly once in --list, despite the on-disk duplication.
got="$("$RESOLVE" --list | wc -l | tr -d ' ')"
assert_eq "--list emits one row per task, not per copy" "5" "$got"

got="$("$RESOLVE" --list | cut -f1 | sort -u | wc -l | tr -d ' ')"
assert_eq "--list has no duplicate task IDs" "5" "$got"

# Resolution works from inside a worktree too (phase commands run there).
got="$(cd "$repo/.worktrees/task-011-delta" && "$RESOLVE" 010 | cut -f1)"
assert_eq "resolves correctly from inside another worktree" "task-010-gamma" "$got"

# --- failure modes -------------------------------------------------------

set +e
"$RESOLVE" task-999-nope >/dev/null 2>&1; rc=$?
set -e
assert_eq "no match exits 3" "3" "$rc"

set +e
"$RESOLVE" task >/dev/null 2>&1; rc=$?
set -e
assert_eq "ambiguous fragment exits 4" "4" "$rc"

set +e
"$RESOLVE" >/dev/null 2>&1; rc=$?
set -e
assert_eq "missing argument exits 2" "2" "$rc"

# --- task-brief ----------------------------------------------------------

plan="$repo/docs/tasks/task-001-alpha/plan.md"
cat > "$plan" <<'PLAN'
# Plan

Preamble that belongs to no task.

## Task 1: First thing

Body of task one.

```sh
# Task 2: this is a decoy inside a fence
```

Still task one.

## Task 2: Second thing

Body of task two.

## Task 10: Tenth thing

Body of task ten.
PLAN

"$BRIEF" "$plan" 1 "$tmp/brief1.md" >/dev/null
got="$(grep -c 'Body of task one' "$tmp/brief1.md")"
assert_eq "brief captures its own task body" "1" "$got"

got="$(grep -c 'Body of task two' "$tmp/brief1.md" || true)"
assert_eq "brief stops at the next task heading" "0" "$got"

got="$(grep -c 'decoy inside a fence' "$tmp/brief1.md")"
assert_eq "fenced heading does not terminate the section" "1" "$got"

got="$(grep -c 'Preamble' "$tmp/brief1.md" || true)"
assert_eq "brief excludes the pre-task preamble" "0" "$got"

# Task 1 must not swallow Task 10 — the heading match is number-exact.
"$BRIEF" "$plan" 10 "$tmp/brief10.md" >/dev/null
got="$(grep -c 'Body of task ten' "$tmp/brief10.md")"
assert_eq "two-digit task number resolves to its own section" "1" "$got"

got="$(grep -c 'Body of task ten' "$tmp/brief1.md" || true)"
assert_eq "Task 1 does not swallow Task 10" "0" "$got"

set +e
"$BRIEF" "$plan" 7 "$tmp/brief7.md" >/dev/null 2>&1; rc=$?
set -e
assert_eq "missing task number exits 3" "3" "$rc"
assert_eq "no empty brief file left behind" "0" \
  "$([ -e "$tmp/brief7.md" ] && echo 1 || echo 0)"

# Default output location: per-plan workspace, git-ignored.
"$BRIEF" "$plan" 1 >/dev/null
assert_eq "default brief lands in the per-plan workspace" "1" \
  "$([ -s "$repo/.superpowers/sdd/plan/task-1-brief.md" ] && echo 1 || echo 0)"
assert_eq "workspace is self-ignoring" "1" \
  "$([ -s "$repo/.superpowers/sdd/.gitignore" ] && echo 1 || echo 0)"
assert_eq "workspace does not dirty git status" "" "$(git -C "$repo" status --porcelain --untracked-files=all | grep superpowers || true)"

if [ "$fails" -ne 0 ]; then
  echo "$fails assertion(s) failed" >&2
  exit 1
fi
echo "all task-resolve.sh / task-brief.sh tests passed"
