#!/usr/bin/env bash
# check-version-coverage_test.sh — hermetic regression tests for
# tools/check-version-coverage.sh. Builds a throwaway git repo with fixture
# templates + versions.json (the script resolves paths via git rev-parse).
#     tools/check-version-coverage_test.sh
set -euo pipefail

SCRIPT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/check-version-coverage.sh"
[ -x "$SCRIPT" ] || { echo "FATAL: $SCRIPT not executable" >&2; exit 2; }

fails=0
assert_eq() { if [ "$2" = "$3" ]; then echo "ok   - $1"; else echo "FAIL - $1 (want '$2', got '$3')" >&2; fails=$((fails+1)); fi; }
assert_contains() { if printf '%s\n' "$3" | grep -qF -- "$2"; then echo "ok   - $1"; else echo "FAIL - $1 (missing '$2')" >&2; fails=$((fails+1)); fi; }

tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" config user.email t@t.t; git -C "$tmp" config user.name t
TPL="$tmp/services/atlas-configurations/seed-data/templates"
mkdir -p "$tmp/deploy/k8s/base" "$TPL" "$tmp/tools"
cp "$SCRIPT" "$tmp/tools/check-version-coverage.sh"

set_versions()    { printf '{ "versions": [ %s ] }\n' "$1" > "$tmp/deploy/k8s/base/versions.json"; }
reset_templates() { rm -f "$TPL"/template_*.json; }
add_template()    { : > "$TPL/template_$1.json"; }   # $1 = gms_83_1
run() { set +e; out="$( cd "$tmp" && ./tools/check-version-coverage.sh 2>&1 )"; rc=$?; set -e; }

# --- Test 1: in-sync sets pass ---
reset_templates; add_template gms_12_1; add_template gms_83_1
set_versions '{ "region": "gms", "majorVersion": 12, "minorVersion": 1 }, { "region": "gms", "majorVersion": 83, "minorVersion": 1 }'
run
assert_eq "in-sync exit 0" "0" "$rc"

# --- Test 2: template without a versions.json entry fails + names it ---
reset_templates; add_template gms_48_1; add_template gms_83_1
set_versions '{ "region": "gms", "majorVersion": 83, "minorVersion": 1 }'
run
assert_eq "template-without-version exit 1" "1" "$rc"
assert_contains "names gms 48" "gms 48" "$out"

# --- Test 3: versions.json entry without a template fails + names it ---
reset_templates; add_template gms_83_1
set_versions '{ "region": "gms", "majorVersion": 83, "minorVersion": 1 }, { "region": "gms", "majorVersion": 92, "minorVersion": 1 }'
run
assert_eq "version-without-template exit 1" "1" "$rc"
assert_contains "names gms 92" "gms 92" "$out"

# --- Test 4: two minors of one major, single version entry → no false mismatch ---
reset_templates; add_template gms_83_1; add_template gms_83_2
set_versions '{ "region": "gms", "majorVersion": 83, "minorVersion": 1 }'
run
assert_eq "two-minors-one-major exit 0" "0" "$rc"

echo; [ "$fails" -eq 0 ] && echo "ALL PASS" || { echo "$fails FAILED" >&2; exit 1; }
