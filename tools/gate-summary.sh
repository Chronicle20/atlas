#!/usr/bin/env bash
# gate-summary.sh — print the verdict block from a tools/verify.sh log.
#
# Why this exists
# ---------------
# verify.sh writes tens of KB to its log; the controller needs the ~15 lines at
# the end that say what passed, what failed, and whether the run was gated.
# Across the task-231 execute chain, 57 tool calls sampled gate logs and the
# *how* was reinvented almost every time — `tail -3`, `tail -25`, `tail -40`,
# `tail -80`, `grep -E "✓|✗|FAIL|All checks"`, `grep -n -A40 "lint & format
# guard (87"` — twelve-plus distinct incantations for one log format, before
# session 4 converged on `sed -n '/verify.sh summary/,$p'`.
#
# That is a script, not a judgment call. One invocation, one predictable ~1KB
# result, and no per-gate decision about which flags to use. On failure it also
# prints the failing checks' surrounding output, which is the follow-up call the
# ad-hoc approach always needed next.
#
# ANSI escapes are stripped: they cost context and render as noise in a
# transcript.
#
# Usage: tools/gate-summary.sh LOGFILE [--full]
#          --full   also print the failing checks' detail (default when failing)
#
# Exit codes:
#   0  log contains a summary and all checks passed
#   1  log contains a summary and at least one check FAILED
#   2  usage error / no such log
#   3  log exists but has no summary yet (run still in progress, or it died)

set -u

MARKER='════ verify.sh summary ════'

log=""
full=0
for a in "$@"; do
    case "$a" in
        --full) full=1 ;;
        -h|--help) sed -n '28,36p' "$0"; exit 0 ;;
        -*) printf 'gate-summary.sh: unknown flag %s\n' "$a" >&2; exit 2 ;;
        *)  log="$a" ;;
    esac
done

if [ -z "$log" ]; then
    printf 'usage: tools/gate-summary.sh LOGFILE [--full]\n' >&2
    exit 2
fi

if [ ! -f "$log" ]; then
    printf 'gate-summary.sh: no such log: %s\n' "$log" >&2
    printf '(if verify.sh was just launched, the log appears once it starts writing)\n' >&2
    exit 2
fi

strip_ansi() { sed 's/\x1b\[[0-9;]*[a-zA-Z]//g'; }

if ! grep -qF "$MARKER" "$log" 2>/dev/null; then
    # No summary: either still running, or it died before finishing.
    if pgrep -f 'tools/verify\.sh' >/dev/null 2>&1; then
        printf 'STILL RUNNING — no summary in %s yet.\n' "$log"
        printf 'Do not poll this in a loop; launch with run_in_background and\n'
        printf 'come back when the task notification arrives.\n\n'
    else
        printf 'INCOMPLETE — no summary in %s, and no verify.sh is running.\n' "$log"
        printf 'The run probably died. Last lines:\n\n'
    fi
    tail -15 "$log" | strip_ansi
    exit 3
fi

block="$(sed -n "/$MARKER/,\$p" "$log" | strip_ansi)"

failed="$(printf '%s\n' "$block"  | sed -n 's/^  ✗ \(.*\)$/\1/p')"
n_pass="$(printf '%s\n' "$block"  | grep -c '^  ✓ ' || true)"
n_skip="$(printf '%s\n' "$block"  | grep -c '^  − ' || true)"
n_fail="$(printf '%s\n' "$failed" | grep -c . || true)"
verdict="$(printf '%s\n' "$block" | grep -E 'All checks passed|check\(s\) FAILED' | head -1)"

# A flagless run lists ~170 checks. The controller needs the verdict, not the
# roster — dumping the whole block is the very cost this script exists to avoid.
# Full listing is opt-in via --full; failures always print by name.
if [ "$full" -eq 1 ]; then
    printf '%s\n' "$block"
else
    printf '%s\n' "${verdict:-verify.sh: no verdict line found}"
    printf '  %s passed, %s failed, %s skipped\n' "$n_pass" "$n_fail" "$n_skip"
    if [ -n "$failed" ]; then
        printf '%s\n' "$failed" | sed 's/^/  ✗ /'
    fi
fi

if [ -z "$failed" ]; then
    exit 0
fi

if true; then
    printf '\n──── failing check detail ────\n'
    printf '%s\n' "$failed" | while IFS= read -r name; do
        [ -n "$name" ] || continue
        # Match the check's own header line earlier in the log, then show the
        # block that follows it. Strip a trailing parenthetical (e.g. counts)
        # so the header match is not defeated by a number that varies per run.
        base="$(printf '%s' "$name" | sed 's/ *([^)]*)$//')"
        printf '\n=== %s ===\n' "$name"
        # Show the block after the check's header, but stop at the summary
        # marker — spilling into the roster re-introduces the bloat this script
        # exists to remove.
        grep -F -m1 -A 25 "$base" "$log" 2>/dev/null \
            | strip_ansi \
            | sed -n "2,\$p" \
            | sed "/$MARKER/q" \
            | sed '/^════/d' \
            || printf '(no surrounding output found for "%s")\n' "$base"
    done
fi

exit 1
