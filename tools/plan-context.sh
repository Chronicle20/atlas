#!/usr/bin/env bash
# tools/plan-context.sh — the code survey /plan-task opens with, in one call.
#
# Why this exists
# ---------------
# Measured across all 70 `/plan-task` sessions in this repo's transcript
# history: median 102 turns, median 220k peak context, and 44.8 Bash calls per
# session — 12.8 `cd`, 9.7 `grep`, 8.3 `sed`, 4.1 `cat`, 3.8 `ls`. Those ~39
# calls move a median of 0.5 MB total, so this was never a bytes problem. Each
# one is a separate turn that replays the whole 220k window, and together they
# are ~38% of the session's turns.
#
# What they are collectively computing is small and entirely mechanical: for
# every path design.md names, does it exist, what module does it build from,
# does it already have a test file, and what siblings could a task's
# "Patterns to copy:" line point at. This script answers all of that in one
# call, in under ~8 KB.
#
# It also pre-empts the two `tools/plan-lint.sh` checks that drive the rewrite
# passes. Plan-task sessions average 5.5 Write/Edit ops against plan.md for a
# ~50 KB file (median 91 KB of payload — ~1.8x re-emission), and the loop is
# write -> lint -> fix -> fix. F1 (every `### Files` path exists or is marked
# `new file`) is answered by the EXISTING/UNRESOLVED split below, before the
# first Write. F5 (symbols in ```go blocks resolve) is answered by --symbols.
#
# Usage:
#   tools/plan-context.sh <task-identifier> [options]
#
# <task-identifier> is anything tools/task-resolve.sh accepts: "task-054-slug",
# "task-054", "054", "54", or a slug fragment.
#
# Options:
#   --from <file>       Scan this file for paths instead of design.md+prd.md.
#                       Repeatable. Use it to survey a path list you already
#                       have rather than re-deriving one.
#   --symbols           For each existing .go file, list its EXPORTED func,
#                       method and type declarations. Answers plan-lint F5 up
#                       front. Off by default: it added 8.4 KB to task-241's
#                       6.0 KB survey.
#   --siblings <n>      Sibling files listed per touched directory (default 6,
#                       0 disables). Siblings are the "Patterns to copy:"
#                       candidates.
#   --max-paths <n>     Cap on surveyed paths (default 60). The cap is
#                       REPORTED, never silent — an elided count is printed.
#
# Output: plain text on stdout, sectioned. Designed to be pasted into the
# planning context whole, or into a discovery subagent's brief.
#
# Exit codes:
#   0  survey printed
#   2  usage error
#   3  no such task            (passed through from task-resolve.sh)
#   4  ambiguous identifier    (passed through, candidates on stderr)
#   5  resolved, but the task has no design.md to survey

set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"

SYMBOLS=0
SIBLINGS=6
MAX_PATHS=60
QUERY=""
FROM=()

while [ $# -gt 0 ]; do
    case "$1" in
        --symbols)  SYMBOLS=1; shift ;;
        --siblings) SIBLINGS="${2:-}"; shift 2 ;;
        --max-paths) MAX_PATHS="${2:-}"; shift 2 ;;
        --from)
            [ -n "${2:-}" ] || { echo "plan-context.sh: --from needs a file" >&2; exit 2; }
            FROM+=("$2"); shift 2 ;;
        -h|--help) sed -n '2,45p' "$0"; exit 0 ;;
        -*) echo "plan-context.sh: unknown option $1" >&2; exit 2 ;;
        *)  [ -z "$QUERY" ] || { echo "plan-context.sh: one identifier only" >&2; exit 2; }
            QUERY="$1"; shift ;;
    esac
done

[ -n "$QUERY" ] || { echo "usage: tools/plan-context.sh <task-identifier> [--symbols] [--siblings n] [--from file]" >&2; exit 2; }
case "$SIBLINGS$MAX_PATHS" in
    *[!0-9]*) echo "plan-context.sh: --siblings and --max-paths take integers" >&2; exit 2 ;;
esac

# task-resolve.sh owns resolution and its exit codes; pass them through so a
# caller's existing 3/4 handling keeps working.
resolved="$("$HERE/task-resolve.sh" "$QUERY")" || exit $?
task="$(printf '%s' "$resolved" | cut -f1)"
task_dir="$(printf '%s' "$resolved" | cut -f2)"
worktree="$(printf '%s' "$resolved" | cut -f3)"

cd "$worktree"

sources=()
if [ "${#FROM[@]}" -gt 0 ]; then
    for f in "${FROM[@]}"; do
        [ -f "$f" ] || { echo "plan-context.sh: no such file: $f" >&2; exit 2; }
        sources+=("$f")
    done
else
    [ -f "$task_dir/design.md" ] || {
        echo "plan-context.sh: $task has no design.md — run /design-task first" >&2
        exit 5
    }
    sources+=("$task_dir/design.md")
    # `if`, not `[ ... ] && ...`: under `set -e` a false one-liner test is the
    # statement's exit status, which aborts the script. Every conditional
    # statement in this file is written as an `if` for that reason.
    if [ -f "$task_dir/prd.md" ]; then sources+=("$task_dir/prd.md"); fi
fi

echo "task=$task"
echo "worktree=$worktree"
echo "scanned=$(for s in "${sources[@]}"; do printf '%s ' "${s#"$worktree"/}"; done)"

# --- path extraction -------------------------------------------------------
# Pull repo-relative-looking paths out of the design prose. Deliberately
# permissive: a false positive costs one UNRESOLVED line, while a miss costs
# the planner the grep turn this script exists to remove.
extract() {
    grep -ohE '(services|libs|tools|scripts|docs|cmd|internal)/[A-Za-z0-9_@.+-]+(/[A-Za-z0-9_@.+-]+)*' "$@" 2>/dev/null |
        sed 's/[.,;:)]*$//' |
        # Drop line-wrap artefacts: prose that broke `libs/atlas-kafka` across a
        # newline leaves a bare `libs/atlas-`, which would otherwise be reported
        # as an unresolved path and read as a genuine missing file.
        grep -vE '[-_/]$' |
        sort -u
}

all_paths="$(extract "${sources[@]}" || true)"
total="$(printf '%s' "$all_paths" | grep -c . || true)"

# Existing paths first — they are the ones with real facts attached. Truncation
# therefore drops speculative prose matches before it drops real files.
existing=""
unresolved=""
while IFS= read -r p; do
    [ -n "$p" ] || continue
    if [ -e "$p" ]; then
        existing="$existing$p
"
    else
        unresolved="$unresolved$p
"
    fi
done <<EOF
$all_paths
EOF

ex_count="$(printf '%s' "$existing" | grep -c . || true)"
shown="$(printf '%s' "$existing" | grep -c . || true)"
if [ "$ex_count" -gt "$MAX_PATHS" ]; then
    existing="$(printf '%s' "$existing" | head -n "$MAX_PATHS")"
    shown="$MAX_PATHS"
fi

echo "paths_found=$total existing=$ex_count surveyed=$shown"
if [ "$ex_count" -gt "$shown" ]; then
    echo "TRUNCATED: $((ex_count - shown)) existing path(s) not surveyed — raise --max-paths"
fi

# --- module roots ----------------------------------------------------------
# The directory a task's `go build ./... && go test ./...` actually runs from.
# Step 5a of /plan-task requires this in the ### Files block whenever it is not
# obvious from the paths; deriving it is a per-file upward walk nobody should
# spend a turn on.
module_root() {
    d="$1"
    [ -d "$d" ] || d="$(dirname "$d")"
    while [ "$d" != "." ] && [ "$d" != "/" ]; do
        if [ -f "$d/go.mod" ]; then printf '%s\n' "$d"; return 0; fi
        d="$(dirname "$d")"
    done
    return 1
}

roots=""
while IFS= read -r p; do
    [ -n "$p" ] || continue
    r="$(module_root "$p" || true)"
    if [ -n "$r" ]; then
        roots="$roots$r
"
    fi
done <<EOF
$existing
EOF

if [ -n "$roots" ]; then
    echo
    echo "## Module roots (go build/go test cwd)"
    printf '%s' "$roots" | sort -u | while IFS= read -r r; do
        [ -n "$r" ] || continue
        echo "- $r  ($(sed -n 's/^module[[:space:]]*//p' "$r/go.mod" | head -1))"
    done
fi

# --- the file survey -------------------------------------------------------
echo
echo "## Existing files"
while IFS= read -r p; do
    [ -n "$p" ] || continue
    if [ -d "$p" ]; then
        n="$(find "$p" -maxdepth 1 -type f | wc -l | tr -d ' ')"
        echo "- $p/  — directory, $n file(s)"
        continue
    fi
    lines="$(wc -l < "$p" | tr -d ' ')"
    note=""
    case "$p" in
        *_test.go) note=" [test file]" ;;
        *.go)
            t="${p%.go}_test.go"
            if [ -f "$t" ]; then
                note=" [has test: $t]"
            else
                note=" [NO test file — a new one is a new file]"
            fi
            ;;
    esac
    echo "- $p — ${lines} lines$note"
done <<EOF
$existing
EOF

# --- unresolved ------------------------------------------------------------
# These are what plan-lint F1 fails on. Every one is either a genuine new file
# (mark it `new file` in the ### Files block) or a path the design got wrong
# (fix it here, not after the first lint run).
if [ -n "$unresolved" ]; then
    un_count="$(printf '%s' "$unresolved" | grep -c . || true)"
    echo
    echo "## Unresolved paths ($un_count) — mark 'new file' or correct before writing plan.md"
    printf '%s' "$unresolved" | head -n "$MAX_PATHS" | while IFS= read -r p; do
        [ -n "$p" ] || continue
        echo "- $p"
    done
    if [ "$un_count" -gt "$MAX_PATHS" ]; then
        echo "  (+$((un_count - MAX_PATHS)) more, elided)"
    fi
fi

# --- siblings --------------------------------------------------------------
# "Patterns to copy:" candidates. A task section is far cheaper to write when
# the planner can point at the adjacent file that already has the shape.
if [ "$SIBLINGS" -gt 0 ]; then
    dirs=""
    while IFS= read -r p; do
        [ -n "$p" ] || continue
        if [ -d "$p" ]; then dirs="$dirs$p
"; else dirs="$dirs$(dirname "$p")
"; fi
    done <<EOF
$existing
EOF
    if [ -n "$dirs" ]; then
        echo
        echo "## Siblings (Patterns-to-copy candidates, $SIBLINGS per dir)"
        printf '%s' "$dirs" | sort -u | while IFS= read -r d; do
            [ -n "$d" ] && [ -d "$d" ] || continue
            sib="$(find "$d" -maxdepth 1 -type f \( -name '*.go' -o -name '*.ts' -o -name '*.tsx' \) \
                    -printf '%f\n' 2>/dev/null | sort | head -n "$SIBLINGS" | paste -sd' ' -)"
            if [ -n "$sib" ]; then echo "- $d/: $sib"; fi
        done
    fi
fi

# --- symbols ---------------------------------------------------------------
# plan-lint F5 checks that every symbol a ```go block calls resolves. The
# planner writes those blocks without a compiler, so the cheap moment to learn
# the real signatures is here, not at dispatch time at 150-250k context.
if [ "$SYMBOLS" -eq 1 ]; then
    echo
    echo "## Top-level declarations in touched .go files"
    while IFS= read -r p; do
        [ -n "$p" ] || continue
        case "$p" in *.go) ;; *) continue ;; esac
        [ -f "$p" ] || continue
        # Exported declarations only. An unexported helper is not something a
        # ```go block in the plan can call from another package, so listing it
        # buys nothing and costs context: on task-241 the unfiltered form added
        # 22.8 KB to the survey, the filtered form 8.4 KB.
        decls="$(grep -nE '^(func([[:space:]]+|[[:space:]]*\([^)]*\)[[:space:]]*)[A-Z]|type[[:space:]]+[A-Z])' "$p" 2>/dev/null | head -n 25 || true)"
        [ -n "$decls" ] || continue
        echo "### $p"
        printf '%s\n' "$decls" | sed 's/[[:space:]]*{[[:space:]]*$//; s/^/  /'
    done <<EOF
$existing
EOF
fi
