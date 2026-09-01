# shellcheck shell=bash
# tools/lib/go-work.sh — go.work `use`-entry membership, shared by
# tools/lint.sh, tools/verify.sh, and tools/lib/analyzer-guard.sh.
#
# go.work's `use` entries (both the single-line `use ./path` form and the
# parenthesized `use ( ... )` block form), resolved to absolute directories.
# Every caller here intersects its own module walk under services/ or libs/
# against this set: a tool that walks `go.work`-scoped modules cannot
# type-check a directory outside it ("directory prefix . does not contain
# modules listed in go.work"). check_workspace_drift() below is the other
# half of that contract — a module found under services/ or libs/ but absent
# from this set must fail the caller loudly by name, never drop out of the
# intersection silently (see plan.md:18 for task-276, Fix G5).
#
# Fails loudly (non-zero, naming go.work) rather than returning an empty set:
# an unreadable go.work or a parse with zero `use` entries must stop the
# sweep, never silently verify nothing while reporting success.
#
# Source it; do not execute it:
#
#     ROOT="$(cd "$(dirname "$0")/.." && pwd)"
#     . "$ROOT/tools/lib/go-work.sh"
#     workspace="$(workspace_module_dirs "lint.sh")" || exit 1
#
# The caller name is used to prefix this helper's own error messages
# ("lint.sh: ERROR — ..." vs "verify.sh: ERROR — ..."), so each script's
# failures still read as its own rather than a shared library's.

workspace_module_dirs() {
    local caller="${1:?workspace_module_dirs: caller name required}"
    if [ ! -r "$ROOT/go.work" ]; then
        echo "${caller}: ERROR — cannot read $ROOT/go.work" >&2
        return 1
    fi
    local entries
    entries="$(awk '
        /^use \(/ { inuse=1; next }
        inuse && /^\)/ { inuse=0; next }
        inuse {
            gsub(/^[ \t]+|[ \t]+$/, "")
            if ($0 != "") print
            next
        }
        /^use[ \t]+/ {
            line = $0
            sub(/^use[ \t]+/, "", line)
            gsub(/^[ \t]+|[ \t]+$/, "", line)
            if (line != "") print line
        }
    ' "$ROOT/go.work")"
    if [ -z "$entries" ]; then
        echo "${caller}: ERROR — $ROOT/go.work parsed to zero 'use' entries" >&2
        return 1
    fi
    printf '%s\n' "$entries" | while IFS= read -r p; do
        case "$p" in
            /*) printf '%s\n' "$p" ;;
            *)  printf '%s\n' "$ROOT/${p#./}" ;;
        esac
    done | sort -u
}

# check_workspace_drift <caller> <found> <workspace> — reports, on stdout,
# every offender when <found> (a newline-separated set of module
# directories a caller discovered under services/ or libs/) contains a
# directory absent from <workspace> (workspace_module_dirs()'s go.work
# `use` set). Nothing is printed to stdout when there is no drift.
#
# The intersection callers take between found and workspace (`comm -12`)
# must never silently drop a module nobody added to go.work: that module
# then goes unbuilt, unvetted, and unlinted while the gate still reports
# green. It already happened once on this branch (libs/atlas-kafka/gen —
# see plan.md:18 for task-276, Fix G5), so this check runs before the
# intersection is taken, not after.
#
# This function reports; it does not decide fatality. It returns 1 (and
# names every offender on stderr, unchanged) whenever drift is found — the
# caller decides whether that return is fatal. Every current caller in
# tools/verify.sh, tools/lint.sh, and tools/lib/analyzer-guard.sh treats it
# as fatal (`|| exit 1` / `|| return 1`) on a real run; tools/verify.sh's
# --facts path is the one exception, capturing this function's stdout to
# report the drift as a fact instead of aborting.
#
# Both sets are re-sorted LC_ALL=C here so `comm` never refuses regardless
# of which locale/order the caller collated them in.
check_workspace_drift() {
    local caller="${1:?check_workspace_drift: caller name required}"
    local found="$2"
    local workspace="$3"
    [ -z "$found" ] && return 0
    local dropped
    dropped="$(LC_ALL=C comm -23 <(printf '%s\n' "$found" | LC_ALL=C sort -u) <(printf '%s\n' "$workspace" | LC_ALL=C sort -u))"
    if [ -n "$dropped" ]; then
        echo "${caller}: ERROR — the following module(s) under services/ or libs/ are not in go.work's 'use' list:" >&2
        printf '%s\n' "$dropped" | while IFS= read -r d; do
            echo "${caller}: ERROR —   $d" >&2
        done
        echo "${caller}: ERROR — add the missing module(s) to go.work's 'use' list to fix this" >&2
        printf '%s\n' "$dropped"
        return 1
    fi
}
