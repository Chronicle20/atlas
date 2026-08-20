#!/usr/bin/env sh
# derive-service-id_test.sh — pinned-value regression suite for
# tools/derive-service-id.sh. Invokes the script as a subprocess (not
# sourced). Run directly: sh tools/derive-service-id_test.sh
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SCRIPT="$SCRIPT_DIR/derive-service-id.sh"

fails=0

pass() {
    echo "PASS: $1"
}

fail() {
    echo "FAIL: $1" >&2
    fails=$((fails + 1))
}

assert_eq() {
    name="$1"
    expected="$2"
    actual="$3"
    if [ "$expected" = "$actual" ]; then
        pass "$name"
    else
        fail "$name (want '$expected', got '$actual')"
    fi
}

# --- Pinned-value assertions (design §"Pinned values") ---
assert_eq "login-service pr-1411" "6439ca9c-d28d-5db9-821b-8dd93d318a25" "$("$SCRIPT" login-service pr-1411)"
assert_eq "channel-service pr-1411" "5a86d8e6-3167-5e74-9fc5-021d94001da2" "$("$SCRIPT" channel-service pr-1411)"
assert_eq "drops-service pr-1411" "cbce66aa-facb-5766-8583-84c3478a6ba2" "$("$SCRIPT" drops-service pr-1411)"
assert_eq "world-service pr-1411" "f80c02bc-2ac4-598e-a8e6-298e7e1d72b5" "$("$SCRIPT" world-service pr-1411)"
assert_eq "character-factory pr-1411" "a0bb4ad4-0c2b-5941-b297-fa4b6cf9403e" "$("$SCRIPT" character-factory pr-1411)"
assert_eq "drops-information-service pr-1411" "87d2d5a6-f37d-5a1e-8e81-bfed3a239e69" "$("$SCRIPT" drops-information-service pr-1411)"
assert_eq "login-service pr-999" "e7ae96a2-c484-5617-8e28-2178b60a8378" "$("$SCRIPT" login-service pr-999)"
assert_eq "login-service main" "78d4984e-22dd-5284-8729-61627a5e603f" "$("$SCRIPT" login-service main)"
assert_eq "channel-service pr-999" "2e3b50b4-fb89-5af0-bb51-19749ecb734f" "$("$SCRIPT" channel-service pr-999)"
assert_eq "channel-service main" "dff6f040-d4aa-51fa-914b-ff1dff6f6a76" "$("$SCRIPT" channel-service main)"

# --- §1.2 regression: the four nil-UUID services derive four DISTINCT ids ---
distinct_count=$(
    {
        "$SCRIPT" drops-service pr-1411
        echo
        "$SCRIPT" world-service pr-1411
        echo
        "$SCRIPT" character-factory pr-1411
        echo
        "$SCRIPT" drops-information-service pr-1411
        echo
    } | sort -u | wc -l
)
assert_eq "nil-UUID services collided (design §1.2)" "4" "$distinct_count"

# --- Determinism: two consecutive invocations, identical args, byte-identical output ---
first="$("$SCRIPT" login-service pr-1411)"
second="$("$SCRIPT" login-service pr-1411)"
assert_eq "determinism" "$first" "$second"

# --- Environment sensitivity ---
a="$("$SCRIPT" login-service pr-1411)"
b="$("$SCRIPT" login-service pr-1412)"
if [ "$a" != "$b" ]; then
    pass "environment sensitivity"
else
    fail "environment sensitivity (pr-1411 and pr-1412 produced the same id)"
fi

# --- Argument validation: zero args, one arg, empty-string second arg ---
set +e
out="$("$SCRIPT" 2>&1)"; rc=$?
set -e
if [ "$rc" -ne 0 ] && [ -n "$out" ]; then
    pass "zero args exits non-zero with usage on stderr"
else
    fail "zero args exits non-zero with usage on stderr (rc=$rc, out='$out')"
fi

set +e
out="$("$SCRIPT" login-service 2>&1)"; rc=$?
set -e
if [ "$rc" -ne 0 ] && [ -n "$out" ]; then
    pass "one arg exits non-zero with usage on stderr"
else
    fail "one arg exits non-zero with usage on stderr (rc=$rc, out='$out')"
fi

set +e
out="$("$SCRIPT" login-service "" 2>&1)"; rc=$?
set -e
if [ "$rc" -ne 0 ] && [ -n "$out" ]; then
    pass "empty-string second arg exits non-zero with usage on stderr"
else
    fail "empty-string second arg exits non-zero with usage on stderr (rc=$rc, out='$out')"
fi

echo
if [ "$fails" -eq 0 ]; then
    echo "ALL PASS"
    exit 0
else
    echo "$fails FAILED" >&2
    exit 1
fi
