#!/usr/bin/env bash
# build-slot_test.sh — function-level suite for tools/lib/build-slot.sh.
#
# Drives acquire_build_slot / release_build_slot directly (no CLI wrapper);
# tools/with-build-slot_test.sh covers the CLI.
#
# Run directly: tools/lib/build-slot_test.sh
set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=./build-slot.sh
. "$HERE/build-slot.sh"

fails=0
assert_eq() { if [ "$2" = "$3" ]; then echo "ok   - $1"; else echo "FAIL - $1 (want '$2', got '$3')" >&2; fails=$((fails + 1)); fi; }
assert_has() { case "$3" in *"$2"*) echo "ok   - $1" ;; *) echo "FAIL - $1 (missing '$2' in '$3')" >&2; fails=$((fails + 1)) ;; esac; }

tmp="$(mktemp -d)"
holder_pids=()
cleanup() {
    for pid in "${holder_pids[@]:-}"; do
        [ -n "$pid" ] && kill "$pid" >/dev/null 2>&1 || true
    done
    rm -rf "$tmp"
}
trap cleanup EXIT

# -- acquire succeeds and reports a slot -------------------------------------
# K is pinned: the default is derived from the host's physical cores (tested
# separately below), and this case is about acquire/release mechanics.
(
    export ATLAS_SLOT_DIR="$tmp/case1"
    export ATLAS_BUILD_SLOTS=4
    . "$HERE/build-slot.sh"
    acquire_build_slot t
    echo "rc=$? slot=$BUILD_SLOT"
) >"$tmp/case1.out" 2>"$tmp/case1.err"
out="$(cat "$tmp/case1.out")"
assert_has "acquire reports rc=0" "rc=0" "$out"
case "$out" in
    *"slot=1"* | *"slot=2"* | *"slot=3"* | *"slot=4"*)
        echo "ok   - acquire sets BUILD_SLOT in 1..4"
        ;;
    *)
        echo "FAIL - acquire sets BUILD_SLOT in 1..4 (got '$out')" >&2
        fails=$((fails + 1))
        ;;
esac

# -- the slot dir and files are created --------------------------------------
count="$(find "$tmp/case1" -maxdepth 1 -name 'slot.*' -type f 2>/dev/null | wc -l | tr -d ' ')"
assert_eq "slot dir has 4 slot.N files" "4" "$count"

# -- the default K is derived from physical cores, floored at 1 --------------
(
    export ATLAS_SLOT_DIR="$tmp/case1b"
    unset ATLAS_BUILD_SLOTS
    . "$HERE/build-slot.sh"
    cores="$(_build_slot_physical_cores)"
    k="$(_build_slot_count)"
    echo "cores=$cores k=$k"
) >"$tmp/case1b.out" 2>&1
cores="$(grep -o 'cores=[0-9]*' "$tmp/case1b.out" | cut -d= -f2)"
k="$(grep -o 'k=[0-9]*' "$tmp/case1b.out" | cut -d= -f2)"
want_k=$(( ${cores:-0} / 6 )); [ "$want_k" -lt 1 ] && want_k=1
assert_eq "default K is physical_cores/6 floored at 1 (cores=$cores)" "$want_k" "${k:-unset}"
assert_true_k=$([ "${k:-0}" -ge 1 ] && echo yes || echo no)
assert_eq "default K is at least 1" "yes" "$assert_true_k"

# -- K follows the slot thread budget: doubling it halves K, floored at 1 ----
(
    export ATLAS_SLOT_DIR="$tmp/case1c"
    unset ATLAS_BUILD_SLOTS
    export ATLAS_SLOT_THREADS=$(( ${cores:-6} * 2 ))
    . "$HERE/build-slot.sh"
    echo "k=$(_build_slot_count) threads=$(_build_slot_threads)"
) >"$tmp/case1c.out" 2>&1
assert_eq "slot budget above the core count floors K at 1" "1" \
  "$(grep -o 'k=[0-9]*' "$tmp/case1c.out" | cut -d= -f2)"
(
    export ATLAS_SLOT_DIR="$tmp/case1d"
    unset ATLAS_BUILD_SLOTS
    export ATLAS_SLOT_THREADS=1
    . "$HERE/build-slot.sh"
    echo "k=$(_build_slot_count)"
) >"$tmp/case1d.out" 2>&1
assert_eq "a 1-thread slot budget yields K = physical cores" "${cores:-unset}" \
  "$(grep -o 'k=[0-9]*' "$tmp/case1d.out" | cut -d= -f2)"

# -- release frees the slot ---------------------------------------------------
(
    export ATLAS_SLOT_DIR="$tmp/case2"
    . "$HERE/build-slot.sh"
    acquire_build_slot t >/dev/null 2>&1
    release_build_slot
    echo "release_rc=$? slot=${BUILD_SLOT:-}"
) >"$tmp/case2.out" 2>&1
rc="$(grep -o 'release_rc=[0-9]*' "$tmp/case2.out" | cut -d= -f2)"
assert_eq "release_build_slot returns 0" "0" "${rc:-unset}"

# After release, an independent flock -n on slot.1 (the deterministic first
# try) must succeed for at least one of the 4 slot files.
freed=0
for f in "$tmp/case2"/slot.*; do
    if ( flock -n 9 9>"$f" ) 2>/dev/null; then
        freed=1
        break
    fi
done
assert_eq "a released slot file is lockable from a fresh subshell" "1" "$freed"

# -- a second acquire past K fails under a timeout ---------------------------
(
    export ATLAS_SLOT_DIR="$tmp/case3"
    export ATLAS_BUILD_SLOTS=1
    . "$HERE/build-slot.sh"
    acquire_build_slot holder
    sleep 5
) >/dev/null 2>&1 &
holder_pid=$!
holder_pids+=("$holder_pid")
sleep 0.3

(
    export ATLAS_SLOT_DIR="$tmp/case3"
    export ATLAS_BUILD_SLOTS=1
    export ATLAS_BUILD_SLOT_TIMEOUT=1
    . "$HERE/build-slot.sh"
    acquire_build_slot t
    echo "rc=$?"
) >"$tmp/case3.out" 2>"$tmp/case3.err"
rc="$(grep -o 'rc=[0-9]*' "$tmp/case3.out" | cut -d= -f2)"
assert_eq "acquire past capacity under a timeout returns 75" "75" "${rc:-unset}"
kill "$holder_pid" >/dev/null 2>&1 || true

# -- an invalid slot count is rejected ----------------------------------------
(
    export ATLAS_SLOT_DIR="$tmp/case4"
    export ATLAS_BUILD_SLOTS=0
    . "$HERE/build-slot.sh"
    acquire_build_slot t
    echo "rc=$?"
) >"$tmp/case4.out" 2>"$tmp/case4.err"
rc="$(grep -o 'rc=[0-9]*' "$tmp/case4.out" | cut -d= -f2)"
assert_eq "ATLAS_BUILD_SLOTS=0 is rejected with rc 2" "2" "${rc:-unset}"

if [ "$fails" -eq 0 ]; then
    echo "build-slot_test.sh: all assertions passed"
    exit 0
fi
echo "build-slot_test.sh: $fails assertion(s) failed" >&2
exit 1
