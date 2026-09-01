#!/usr/bin/env bash
# gen-topics_test.sh — regression tests for tools/gen-topics.sh.
#
# Unlike gen-tenant-tables_test.sh (hermetic, throwaway repo), this suite
# mutates and restores the real libs/atlas-kafka/gen/topics.yaml in place:
# the generator resolves every path relative to its own module and to
# `git rev-parse --show-toplevel`, and its drift check reads the live
# service/lib tree, so a throwaway repo would not have anything to scan.
# Run directly: tools/gen-topics_test.sh
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT="$REPO_ROOT/tools/gen-topics.sh"
MANIFEST="$REPO_ROOT/libs/atlas-kafka/gen/topics.yaml"
[ -x "$SCRIPT" ] || { echo "FATAL: $SCRIPT not executable" >&2; exit 2; }

fails=0
assert_eq() { if [ "$2" = "$3" ]; then echo "ok   - $1"; else echo "FAIL - $1 (want '$2', got '$3')" >&2; fails=$((fails+1)); fi; }
assert_contains() { if printf '%s\n' "$3" | grep -qF -- "$2"; then echo "ok   - $1"; else echo "FAIL - $1 (missing '$2')" >&2; fails=$((fails+1)); fi; }

teardown() { git -C "$REPO_ROOT" checkout -- "$MANIFEST"; }
trap teardown EXIT

# --- Test 1: --check exits 0 on a clean tree ---
set +e
out="$( "$SCRIPT" --check 2>&1 )"; rc=$?
set -e
assert_eq "gen-topics.sh --check exits 0 on a clean tree" "0" "$rc"

# --- Test 2: --check exits 1 on a dirty manifest, names topics.yaml ---
printf '  - token: EVENT_TOPIC_FABRICATED\n    cleanup: delete\n' >> "$MANIFEST"
set +e
out="$( "$SCRIPT" --check 2>&1 )"; rc=$?
set -e
assert_eq "gen-topics.sh --check exits 1 on a dirty manifest" "1" "$rc"
assert_contains "gen-topics.sh --check names topics.yaml" "topics.yaml" "$out"
teardown

# --- Test 3: --check writes no files ---
before="$(git -C "$REPO_ROOT" status --porcelain)"
"$SCRIPT" --check >/dev/null 2>&1
after="$(git -C "$REPO_ROOT" status --porcelain)"
assert_eq "gen-topics.sh --check writes no files" "$before" "$after"

echo; [ "$fails" -eq 0 ] && echo "ALL PASS" || { echo "$fails FAILED" >&2; exit 1; }
