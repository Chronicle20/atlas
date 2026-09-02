#!/usr/bin/env bash
# buildx-bootstrap_test.sh — tests for tools/buildx-bootstrap.sh.
#
# Cases that would touch docker (create/use/rm a real builder) are gated on
# `command -v docker >/dev/null` so this suite still runs somewhere without
# docker, at the cost of skipping the cases that matter most there.
#
# The docker-gated block saves/restores the host's ambient buildx builder
# selection: this is a shared host, `docker buildx use` is a machine-wide
# side effect, and the suite must not silently steal the ambient builder back
# to `default` when it removes its own throwaway test builder.

set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$HERE/.." && pwd)"
SCRIPT="$HERE/buildx-bootstrap.sh"
[ -x "$SCRIPT" ] || { echo "FATAL: $SCRIPT not executable" >&2; exit 2; }

fails=0
assert_eq()  { if [ "$2" = "$3" ]; then echo "ok   - $1"; else echo "FAIL - $1 (want '$2', got '$3')" >&2; fails=$((fails+1)); fi; }
assert_has() { case "$3" in *"$2"*) echo "ok   - $1";; *) echo "FAIL - $1 (missing '$2')" >&2; fails=$((fails+1));; esac; }

# -- -h prints usage ---------------------------------------------------------
out="$("$SCRIPT" -h)"; rc=$?
assert_eq "-h exits 0" 0 "$rc"
assert_has "-h stdout contains usage:" "usage:" "$out"

# -- unknown option -----------------------------------------------------------
err="$("$SCRIPT" --nope 2>&1 1>/dev/null)"; rc=$?
assert_eq "unknown option exits 2" 2 "$rc"
assert_has "unknown option stderr mentions it" "unknown option" "$err"

# -- --check fails with the remedy when the builder is absent -----------------
err="$(ATLAS_BUILDER=zz-atlas-absent "$SCRIPT" --check 2>&1 1>/dev/null)"; rc=$?
if [ "$rc" -eq 0 ]; then
    echo "FAIL - --check on an absent builder exits non-zero (got 0)" >&2
    fails=$((fails+1))
else
    echo "ok   - --check on an absent builder exits non-zero"
fi
assert_has "--check remedy names this script" "tools/buildx-bootstrap.sh" "$err"

# -- the config path the script names resolves ---------------------------------
config_path="$(grep -o 'deploy/buildkit/buildkitd\.toml' "$SCRIPT" | head -1)"
assert_eq "script names deploy/buildkit/buildkitd.toml" "deploy/buildkit/buildkitd.toml" "$config_path"
if [ -f "$ROOT/deploy/buildkit/buildkitd.toml" ]; then
    echo "ok   - $ROOT/deploy/buildkit/buildkitd.toml exists"
else
    echo "FAIL - $ROOT/deploy/buildkit/buildkitd.toml does not exist" >&2
    fails=$((fails+1))
fi

# -- config declares the parallelism budget ------------------------------------
cfg="$ROOT/deploy/buildkit/buildkitd.toml"
assert_has "config declares max-parallelism = 8" "max-parallelism = 8" "$(cat "$cfg")"

# -- config declares a hard ceiling --------------------------------------------
cfgcontent="$(cat "$cfg")"
assert_has "config declares all = true" "all = true" "$cfgcontent"
assert_has "config declares keepBytes = 60000000000" "keepBytes = 60000000000" "$cfgcontent"

# -- docker-gated cases ---------------------------------------------------------
if command -v docker >/dev/null 2>&1; then
    # Capture the builder selected before this suite touches anything, so the
    # trap can restore it. `docker buildx ls` marks the ambient selection with
    # a `*` suffix on its NAME column; fall back to no-op restore (today's
    # existing behaviour, "leave it on default") if nothing was selected or
    # the query fails, rather than erroring the suite over it.
    PREV_BUILDER="$(docker buildx ls 2>/dev/null | awk '$1 ~ /\*$/ {print $1; exit}' | sed 's/\*$//')"

    TEST_BUILDER="zz-atlas-test-$$"
    export ATLAS_BUILDER="$TEST_BUILDER"
    cleanup() {
        docker buildx rm "$TEST_BUILDER" >/dev/null 2>&1 || true
        if [ -n "$PREV_BUILDER" ]; then
            docker buildx use "$PREV_BUILDER" >/dev/null 2>&1 || true
        fi
    }
    trap cleanup EXIT

    "$SCRIPT" >/tmp/zz-bootstrap-out.$$ 2>&1
    rc1=$?
    assert_eq "bootstrap exits 0" 0 "$rc1"

    "$SCRIPT" --check >/dev/null 2>&1
    rc2=$?
    assert_eq "--check after bootstrap exits 0" 0 "$rc2"

    "$SCRIPT" >/tmp/zz-bootstrap-out2.$$ 2>&1
    rc3=$?
    assert_eq "second bootstrap (idempotent) exits 0" 0 "$rc3"

    count="$(docker buildx ls | grep -c "^${TEST_BUILDER}[* ]" || true)"
    assert_eq "docker buildx ls lists the test builder exactly once" "1" "$count"

    rm -f /tmp/zz-bootstrap-out.$$ /tmp/zz-bootstrap-out2.$$
    unset ATLAS_BUILDER
else
    echo "skip - docker-gated cases (no docker on PATH)"
fi

if [ "$fails" -eq 0 ]; then
    echo "buildx-bootstrap_test: all assertions passed"
    exit 0
fi
echo "buildx-bootstrap_test: $fails assertion(s) failed" >&2
exit 1
