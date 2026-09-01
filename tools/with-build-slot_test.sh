#!/usr/bin/env bash
# with-build-slot_test.sh — CLI suite for tools/with-build-slot.sh.
#
# Run directly: tools/with-build-slot_test.sh
set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="$HERE/with-build-slot.sh"
[ -x "$SCRIPT" ] || { echo "FATAL: $SCRIPT not executable" >&2; exit 2; }

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

export ATLAS_SLOT_DIR="$tmp/slots"

# -- runs the command and passes stdout through -------------------------------
out="$("$SCRIPT" t -- echo hello 2>/dev/null)"; rc=$?
assert_eq "runs the command: exit 0" 0 "$rc"
assert_eq "runs the command: stdout passthrough" "hello" "$out"

# -- propagates a non-zero exit status ----------------------------------------
"$SCRIPT" t -- sh -c 'exit 7' >/dev/null 2>&1; rc=$?
assert_eq "propagates a non-zero exit status" 7 "$rc"

# -- reports the acquired slot on stderr --------------------------------------
err="$("$SCRIPT" t -- true 2>&1 1>/dev/null)"
assert_has "reports the acquired slot on stderr" "acquired slot" "$err"
case "$err" in
    *"acquired slot 1"* | *"acquired slot 2"* | *"acquired slot 3"* | *"acquired slot 4"*)
        echo "ok   - acquired slot is in 1..4"
        ;;
    *)
        echo "FAIL - acquired slot is in 1..4 (got '$err')" >&2
        fails=$((fails + 1))
        ;;
esac

# -- a free slot is taken without waiting -------------------------------------
err="$("$SCRIPT" t -- true 2>&1 1>/dev/null)"
assert_has "a free slot is taken without waiting" "after 0s" "$err"

# -- all slots busy + --timeout fails cleanly ---------------------------------
export ATLAS_BUILD_SLOTS=2
"$SCRIPT" hold1 --timeout 30 -- sleep 5 >/dev/null 2>&1 &
holder_pids+=("$!")
"$SCRIPT" hold2 --timeout 30 -- sleep 5 >/dev/null 2>&1 &
holder_pids+=("$!")
sleep 0.3

err="$(ATLAS_BUILD_SLOTS=2 "$SCRIPT" t --timeout 1 -- true 2>&1 1>/dev/null)"; rc=$?
assert_eq "all slots busy + --timeout exits 75" 75 "$rc"
assert_has "all slots busy + --timeout stderr mentions no build capacity" "no build capacity" "$err"
for pid in "${holder_pids[@]}"; do kill "$pid" >/dev/null 2>&1 || true; done
holder_pids=()
wait 2>/dev/null || true
unset ATLAS_BUILD_SLOTS

# -- a released slot is reacquired --------------------------------------------
export ATLAS_BUILD_SLOTS=1
"$SCRIPT" hold --timeout 30 -- sleep 1 >/dev/null 2>&1 &
holder_pids+=("$!")
sleep 0.3

ATLAS_BUILD_SLOTS=1 "$SCRIPT" t --timeout 5 -- true >/dev/null 2>&1; rc=$?
assert_eq "a released slot is reacquired" 0 "$rc"
holder_pids=()
unset ATLAS_BUILD_SLOTS

# -- --slots overrides the env var --------------------------------------------
export ATLAS_BUILD_SLOTS=1
"$SCRIPT" hold --timeout 30 -- sleep 5 >/dev/null 2>&1 &
holder_pids+=("$!")
sleep 0.3

err="$("$SCRIPT" t --slots 4 -- true 2>&1 1>/dev/null)"; rc=$?
assert_eq "--slots overrides the env var: exit 0" 0 "$rc"
assert_has "--slots overrides the env var: no wait" "after 0s" "$err"
for pid in "${holder_pids[@]}"; do kill "$pid" >/dev/null 2>&1 || true; done
holder_pids=()
unset ATLAS_BUILD_SLOTS

# -- missing -- separator ------------------------------------------------------
err="$("$SCRIPT" t true 2>&1 1>/dev/null)"; rc=$?
assert_eq "missing -- separator exits 2" 2 "$rc"
assert_has "missing -- separator stderr mentions --" "--" "$err"

# -- missing command after -- --------------------------------------------------
"$SCRIPT" t -- >/dev/null 2>&1; rc=$?
assert_eq "missing command after -- exits 2" 2 "$rc"

# -- invalid slot count ---------------------------------------------------------
"$SCRIPT" t --slots 0 -- true >/dev/null 2>&1; rc=$?
assert_eq "invalid slot count exits 2" 2 "$rc"

# -- -h prints usage ------------------------------------------------------------
out="$("$SCRIPT" -h)"; rc=$?
assert_eq "-h exits 0" 0 "$rc"
assert_has "-h stdout contains usage:" "usage:" "$out"

if [ "$fails" -eq 0 ]; then
    echo "with-build-slot_test.sh: all assertions passed"
    exit 0
fi
echo "with-build-slot_test.sh: $fails assertion(s) failed" >&2
exit 1
