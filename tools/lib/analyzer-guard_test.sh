#!/usr/bin/env bash
# analyzer-guard_test.sh — regression test for analyzer_guard_discover's
# .worktrees handling.
#
# Guards against the false-green fixed by this task: a root whose OWN
# absolute path sits inside a `.worktrees/` segment (the normal case for
# every task in this repo, which does all its work in git worktrees) must
# still discover its modules. A nested worktree genuinely reachable BELOW the
# scanned root must still be excluded, so a run from the main repo does not
# descend into every task branch's checked-out modules.
#
# Run directly: tools/lib/analyzer-guard_test.sh
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(mktemp -d)"
trap 'rm -rf "$ROOT"' EXIT

# shellcheck source=./analyzer-guard.sh
. "$HERE/analyzer-guard.sh"

fails=0
assert_contains() {
    if printf '%s\n' "$3" | grep -qF -- "$2"; then
        echo "ok   - $1"
    else
        echo "FAIL - $1 (missing '$2' in: $3)" >&2
        fails=$((fails + 1))
    fi
}
assert_not_contains() {
    if printf '%s\n' "$3" | grep -qF -- "$2"; then
        echo "FAIL - $1 (unexpectedly present: '$2' in: $3)" >&2
        fails=$((fails + 1))
    else
        echo "ok   - $1"
    fi
}

# Case 1: the root being scanned is itself inside a .worktrees/ segment (this
# is literally what running from inside a task worktree looks like). A module
# directly under it must still be discovered.
WT_ROOT="$ROOT/.worktrees/task-232-fake/services"
mkdir -p "$WT_ROOT/atlas-fake/atlas.com/fake"
: > "$WT_ROOT/atlas-fake/atlas.com/fake/go.mod"

got="$(analyzer_guard_discover "$WT_ROOT")"
assert_contains "root-inside-.worktrees discovers its own module" \
    "$WT_ROOT/atlas-fake/atlas.com/fake" "$got"

# Case 2: a nested worktree genuinely reachable BELOW the scanned root must
# still be excluded (the pre-existing intent this exclusion always had).
MAIN_ROOT="$ROOT/main/services"
mkdir -p "$MAIN_ROOT/atlas-real/atlas.com/real"
: > "$MAIN_ROOT/atlas-real/atlas.com/real/go.mod"
mkdir -p "$MAIN_ROOT/../.worktrees/task-999-nested/services/atlas-nested/atlas.com/nested"
: > "$MAIN_ROOT/../.worktrees/task-999-nested/services/atlas-nested/atlas.com/nested/go.mod"

got="$(analyzer_guard_discover "$MAIN_ROOT")"
assert_contains "main-repo run discovers its own module" \
    "$MAIN_ROOT/atlas-real/atlas.com/real" "$got"
assert_not_contains "main-repo run excludes a nested worktree below root" \
    "task-999-nested" "$got"

if [ "$fails" -ne 0 ]; then
    echo "FAILED: $fails assertion(s)" >&2
    exit 1
fi
echo "analyzer-guard_test.sh: all assertions passed"
