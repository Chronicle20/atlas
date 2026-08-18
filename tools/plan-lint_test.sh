#!/usr/bin/env bash
# plan-lint_test.sh — hermetic regression tests for tools/plan-lint.sh.
#
# The focus is F3, which EXECUTES commands extracted from a plan file. Its
# safety rests entirely on the extraction regex plus the allowlist and
# metacharacter filters, so each of those needs an assertion pinning it.
#
# The headline regression: the extraction regex excludes `;`, which blocks
# `find ... -exec rm {} \;` — but `-exec ... +` terminates without a `;` and
# `-delete` takes no subcommand at all, so both cleared every filter and ran
# with cwd at the repo root.
#
# Run directly:
#
#     tools/plan-lint_test.sh
#
# Exits non-zero on the first failed assertion.

set -euo pipefail

SCRIPT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/plan-lint.sh"
[ -x "$SCRIPT" ] || { echo "FATAL: $SCRIPT not executable" >&2; exit 2; }

fails=0
assert_contains() { # desc needle output
  if printf '%s\n' "$3" | grep -qF -- "$2"; then
    echo "ok   - $1"
  else
    echo "FAIL - $1 (missing '$2' in:)" >&2
    printf '%s\n' "$3" | sed 's/^/       /' >&2
    fails=$((fails + 1))
  fi
}
assert_not_contains() { # desc needle output
  if printf '%s\n' "$3" | grep -qF -- "$2"; then
    echo "FAIL - $1 (unexpected '$2' in:)" >&2
    printf '%s\n' "$3" | sed 's/^/       /' >&2
    fails=$((fails + 1))
  else
    echo "ok   - $1"
  fi
}
assert_file_exists() { # desc path
  if [ -e "$2" ]; then
    echo "ok   - $1"
  else
    echo "FAIL - $1 ($2 was destroyed)" >&2
    fails=$((fails + 1))
  fi
}

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

# F3 executes extracted commands with cwd at `git rev-parse --show-toplevel`,
# falling back to pwd. Make $tmp a repo so that root is definitively $tmp and
# this suite's deliberately destructive plans are contained BY CONSTRUCTION,
# not by the fallback happening to land somewhere harmless. Without this, a
# regression in the primary filter would run `find . -delete` at whatever
# toplevel the caller happened to be inside.
git -C "$tmp" init -q
git -C "$tmp" config user.email t@t.t
git -C "$tmp" config user.name t

run_lint() { # plan-file -> stdout+stderr, never fails the suite
  set +e
  ( cd "$tmp" && "$SCRIPT" "$1" 2>&1 )
  set -e
}

# --------------------------------------------------------------------- F3 ---
# A canary the destructive forms would delete if F3 ran them. plan-lint runs
# extracted commands with cwd at the repo root it resolves, so the canary sits
# where a `find . -delete` rooted anywhere above would reach it.
mkdir -p "$tmp/canary"
echo "do not delete me" > "$tmp/canary/keep.md"

cat > "$tmp/destructive-plan.md" <<'PLAN'
# Task 1: something

Step 2 — verify with:

find . -name "*.md" -delete
find . -name "*.md" -exec rm {} +
find . -name "*.md" -execdir rm {} +
PLAN

out="$(run_lint destructive-plan.md)"

assert_file_exists "F3 does not execute 'find -delete'"        "$tmp/canary/keep.md"
assert_contains    "F3 reports -delete as an acting primary" \
                   "acting primary" "$out"
assert_not_contains "F3 does not claim -delete 'matches nothing'" \
                   "F3 matches nothing: find . -name \"*.md\" -delete" "$out"

for primary in -delete -exec -execdir; do
  assert_contains "F3 refuses $primary" "$primary" "$out"
done

# A benign, genuinely read-only command must still be executed and reported
# when it matches nothing — otherwise the fix has disabled the check it guards.
cat > "$tmp/benign-plan.md" <<'PLAN'
# Task 1: something

grep -n "ThisSymbolDoesNotExistAnywhere" nonexistent-file.go
PLAN

out="$(run_lint benign-plan.md)"
assert_contains "F3 still runs benign read-only commands" \
                "F3 matches nothing" "$out"

# --no-commands must skip F3 entirely.
out="$(cd "$tmp" && "$SCRIPT" --no-commands destructive-plan.md 2>&1 || true)"
assert_not_contains "--no-commands skips F3 execution" \
                    "acting primary" "$out"
assert_file_exists  "--no-commands leaves the canary intact" "$tmp/canary/keep.md"

if [ "$fails" -ne 0 ]; then
  echo "$fails assertion(s) failed" >&2
  exit 1
fi
echo "all plan-lint.sh tests passed"
