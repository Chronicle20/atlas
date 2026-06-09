#!/usr/bin/env bash
# task-numbers_test.sh — hermetic regression tests for tools/task-numbers.sh.
#
# Builds a throwaway git repo so the assertions never depend on the live
# repo's evolving task history. Run directly:
#
#     tools/task-numbers_test.sh
#
# Exits non-zero on the first failed assertion.

set -euo pipefail

SCRIPT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/task-numbers.sh"
[ -x "$SCRIPT" ] || { echo "FATAL: $SCRIPT not executable" >&2; exit 2; }

fails=0
assert_eq() { # desc want got
  if [ "$2" = "$3" ]; then
    echo "ok   - $1"
  else
    echo "FAIL - $1 (want '$2', got '$3')" >&2
    fails=$((fails + 1))
  fi
}
assert_contains() { # desc needle haystack
  if printf '%s\n' "$3" | grep -qx "$2"; then
    echo "ok   - $1"
  else
    echo "FAIL - $1 (missing '$2' in:\n$3)" >&2
    fails=$((fails + 1))
  fi
}

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

git -C "$tmp" init -q
git -C "$tmp" config user.email t@t.t
git -C "$tmp" config user.name t
git -C "$tmp" config commit.gpgsign false

# task-001 — a normal, still-present task (docs folder survives in the tree).
mkdir -p "$tmp/docs/tasks/task-001-alpha"
echo prd > "$tmp/docs/tasks/task-001-alpha/prd.md"
git -C "$tmp" add -A
git -C "$tmp" commit -qm "feat(task-001): alpha"

# task-002 — a MERGED-AND-DELETED task. Its number lives ONLY in a merge-commit
# subject; the docs folder never survived in the working tree and the branch is
# gone. This is the regression: `next` must NOT re-issue 002.
git -C "$tmp" commit -q --allow-empty \
  -m "Merge pull request #2 from someone/task-002-beta"

# task-004 — another present task, leaving 003 as the true smallest free gap.
mkdir -p "$tmp/docs/tasks/task-004-delta"
echo prd > "$tmp/docs/tasks/task-004-delta/prd.md"
git -C "$tmp" add -A
git -C "$tmp" commit -qm "feat(task-004): delta"

# task-005 — an in-flight local branch whose number ALSO appears bare in a
# commit subject. The history source renders it as a number-only `task-005`,
# which must NOT be counted as a task ID distinct from the branch's
# `task-005-echo` (otherwise `check` false-flags every in-flight task).
git -C "$tmp" branch task-005-echo
git -C "$tmp" commit -q --allow-empty -m "wip(task-005): echo groundwork"

cd "$tmp"
list="$("$SCRIPT" list | awk '{print $1}' | sort -u)"
next="$("$SCRIPT" next)"

# The merged-and-deleted number is recognised as used...
assert_contains "history number 002 reported as used" "002" "$list"
assert_contains "present number 001 reported as used"  "001" "$list"
assert_contains "present number 004 reported as used"  "004" "$list"

# ...so `next` skips it and returns the true smallest free gap (003), NOT 002.
assert_eq "next returns true smallest free gap" "003" "$next"

# `check` must not false-flag the in-flight task-005 just because the history
# source also renders its number bare as `task-005`.
set +e
"$SCRIPT" check 2>/tmp/.tn_check_err
check_rc=$?
set -e
assert_eq "check is clean (no false history collision)" "0" "$check_rc"

if [ "$fails" -ne 0 ]; then
  echo "$fails assertion(s) failed" >&2
  exit 1
fi
echo "all task-numbers.sh tests passed"
