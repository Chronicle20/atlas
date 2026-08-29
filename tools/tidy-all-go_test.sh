#!/usr/bin/env bash
# tidy-all-go_test.sh — CLI suite for tools/tidy-all-go.sh.
#
# Run directly: tools/tidy-all-go_test.sh
#
# These cases point ATLAS_GOMODCACHE_LOCK at a scratch path and run the real
# script under a short timeout so it never gets far enough to actually tidy
# any of the 89 workspace modules — the "blocks while the lock is held" case
# is asserted by the ABSENCE of any `==> ` line in the output, not by the
# `timeout` exit code, because both the locked and unlocked cases are killed
# with 124: an exit-code assertion would pass vacuously and prove nothing.
set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="$HERE/tidy-all-go.sh"
REPO_ROOT="$(cd "$HERE/.." && pwd)"
[ -x "$SCRIPT" ] || { echo "FATAL: $SCRIPT not executable" >&2; exit 2; }

fails=0
assert_eq() { if [ "$2" = "$3" ]; then echo "ok   - $1"; else echo "FAIL - $1 (want '$2', got '$3')" >&2; fails=$((fails + 1)); fi; }
assert_has() { case "$3" in *"$2"*) echo "ok   - $1" ;; *) echo "FAIL - $1 (missing '$2' in '$3')" >&2; fails=$((fails + 1)) ;; esac; }
assert_not_has() { case "$3" in *"$2"*) echo "FAIL - $1 (unexpected '$2' in '$3')" >&2; fails=$((fails + 1)) ;; *) echo "ok   - $1" ;; esac; }

tmp="$(mktemp -d)"
holder_pids=()
cleanup() {
    for pid in "${holder_pids[@]:-}"; do
        [ -n "$pid" ] && kill "$pid" >/dev/null 2>&1 || true
    done
    rm -rf "$tmp"
}
trap cleanup EXIT

cd "$REPO_ROOT" || exit 2

# Snapshot dirty-state BEFORE the suite runs anything, so the restore step
# below can tell "dirtied by this suite" apart from "was already dirty when
# the suite started" (e.g. a developer's uncommitted go.mod edit in
# progress, or a concurrent tools/verify.sh gate mutating go.work.sum in
# this same worktree). Anything dirty at baseline is left exactly as found;
# see the restore step for how the two cases diverge.
restore_paths=(go.work.sum '**/go.sum' '**/go.mod')
declare -A baseline_dirty_set
baseline_dirty="$(git status --porcelain -- "${restore_paths[@]}" 2>/dev/null)"
if [ -n "$baseline_dirty" ]; then
    while IFS= read -r line; do
        baseline_dirty_set["${line:3}"]=1
    done <<< "$baseline_dirty"
fi

# run_bounded SECS -- runs "$SCRIPT" [args...] in its own process group and
# hard-kills the whole group (not just the direct child) once SECS elapses.
# Plain `timeout` only signals its direct child; tools/tidy-all-go.sh spawns
# `go mod tidy` as a further child, which would otherwise survive the
# timeout as an orphan and keep mutating go.sum/go.work.sum in the
# background, racing the restore step below. Blocks in the foreground: no
# state survives past the `wait`.
run_bounded() {
    local secs="$1"; shift
    local outfile watchdog pid
    outfile="$(mktemp)"
    setsid "$@" >"$outfile" 2>&1 &
    pid=$!
    ( sleep "$secs"; kill -KILL -- "-$pid" >/dev/null 2>&1 ) &
    watchdog=$!
    wait "$pid" 2>/dev/null
    kill "$watchdog" >/dev/null 2>&1 || true
    wait "$watchdog" 2>/dev/null || true
    cat "$outfile"
    rm -f "$outfile"
}

# -- blocks while the lock is held --------------------------------------------
export ATLAS_GOMODCACHE_LOCK="$tmp/lock"
flock -x "$tmp/lock" sleep 5 &
holder_pids+=("$!")
sleep 0.3

out="$(run_bounded 3 "$SCRIPT")"
assert_not_has "blocks while the lock is held: no ==> line" "==> " "$out"
kill "${holder_pids[-1]}" >/dev/null 2>&1 || true
wait "${holder_pids[-1]}" 2>/dev/null || true
holder_pids=()
unset ATLAS_GOMODCACHE_LOCK

# -- proceeds when the lock is free -------------------------------------------
# Kept short on purpose: long enough to observe the first `==> ` line, short
# enough that `go mod tidy` on module 1 has little time to actually mutate
# the machine-global module cache or a tracked go.sum before run_bounded
# kills the whole process group.
export ATLAS_GOMODCACHE_LOCK="$tmp/free-lock"
out="$(run_bounded 2 "$SCRIPT")"
assert_has "proceeds when the lock is free: has ==> line" "==> " "$out"
unset ATLAS_GOMODCACHE_LOCK

# -- creates the lock's parent dir --------------------------------------------
export ATLAS_GOMODCACHE_LOCK="$tmp/nested/deep/lock"
run_bounded 2 "$SCRIPT" >/dev/null 2>&1
if [ -d "$tmp/nested/deep" ]; then
    echo "ok   - creates the lock's parent dir"
else
    echo "FAIL - creates the lock's parent dir" >&2
    fails=$((fails + 1))
fi
unset ATLAS_GOMODCACHE_LOCK

# -- the lock is distinct from the build slots --------------------------------
default_lock="$(grep -o '/var/tmp/atlas/gomodcache\.lock' "$SCRIPT" | head -n1)"
assert_eq "default lock path is /var/tmp/atlas/gomodcache.lock" "/var/tmp/atlas/gomodcache.lock" "$default_lock"
assert_not_has "default lock path is not under slots/" "slots/" "$default_lock"

# -- restore any machine state the real tidy run mutated -----------------------
# `go mod tidy`/`go mod download` can rewrite go.mod (require-block
# reformatting) and go.work.sum/go.sum (new indirect-dep hashes) purely from
# the local module cache, with no network round trip to wait out. A one-time
# settle gives a killed-but-not-yet-reaped subprocess a moment to finish an
# in-flight write before the restore reads git status, so a single pass
# reliably lands on a clean tree.
#
# Only paths that were CLEAN at the baseline snapshot above and are dirty
# now are restored (checked out if tracked, removed if untracked-new). A
# path that was already dirty at baseline is never touched here, even if it
# is still dirty now — there is no way to tell "the suite mutated it further"
# from "it just sat there dirty the whole time" without hashing content,
# and silently reverting a pre-existing change (a developer's uncommitted
# work, or a concurrent gate's own in-flight write) is exactly the defect
# this snapshot exists to avoid. Such paths are reported with a warning
# instead and do not fail the suite.
sleep 1
dirty="$(git status --porcelain -- "${restore_paths[@]}" 2>/dev/null)"
if [ -n "$dirty" ]; then
    while IFS= read -r line; do
        status="${line:0:2}"
        f="${line:3}"
        if [ -n "${baseline_dirty_set[$f]:-}" ]; then
            echo "WARN - $f was already dirty before this suite ran; leaving it untouched (cannot separate pre-existing changes from anything the suite may have added)" >&2
            continue
        fi
        echo "WARN - tidy run touched $f, restoring:" >&2
        if [ "$status" = "??" ]; then
            rm -f -- "$f"
        else
            git checkout -- "$f"
        fi
    done <<< "$dirty"
fi
still_dirty="$(git status --porcelain -- "${restore_paths[@]}" 2>/dev/null)"
# Only fail the suite over paths it is responsible for restoring. A path
# that was already dirty at baseline is expected to still show up here (see
# the warning above) and must not be counted as a suite failure.
unexpected_dirty=""
if [ -n "$still_dirty" ]; then
    while IFS= read -r line; do
        f="${line:3}"
        if [ -z "${baseline_dirty_set[$f]:-}" ]; then
            unexpected_dirty="${unexpected_dirty}${line}
"
        fi
    done <<< "$still_dirty"
fi
if [ -n "$unexpected_dirty" ]; then
    echo "FAIL - tree still dirty after restore:" >&2
    printf '%s' "$unexpected_dirty" >&2
    fails=$((fails + 1))
fi

if [ "$fails" -eq 0 ]; then
    echo "tidy-all-go_test.sh: all assertions passed"
    exit 0
fi
echo "tidy-all-go_test.sh: $fails assertion(s) failed" >&2
exit 1
