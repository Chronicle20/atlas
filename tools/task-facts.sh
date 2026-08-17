#!/usr/bin/env bash
# tools/task-facts.sh — the mechanical facts about a task, in one block.
#
# Composes the resolvers that already exist. It does NOT re-implement any of
# them:
#
#   tools/task-resolve.sh     task id, task folder, worktree
#   tools/change-surfaces.sh  changed services/libs, surfaces, audit families
#   tools/verify.sh --facts   change base, module selection, guards, gates
#
# Everything here is a fact an agent would otherwise establish by hand. Measured
# on task-232: 2,191 Bash calls / 2.07 MB across 213 streams spent deriving
# branch, worktree, base sha, changed-file set, module list, and toolchain
# availability — while `tools/task-resolve.sh`, which answers the first three,
# was invoked in 16 of those 213 streams. The gap was never missing tooling; it
# was that nothing put the answer where an agent reads it. Prepend this block to
# a dispatch brief and the agent starts with it instead of spending a turn.
#
# Usage:
#   tools/task-facts.sh <task-identifier> [--base <rev>]
#
# <task-identifier> is anything tools/task-resolve.sh accepts: "task-054-slug",
# "task-054", "054", "54", or a slug fragment.
#
# --base narrows the change set the same way tools/verify.sh --base does — pass
# the last gated commit for a per-task increment, omit it for the whole branch.
#
# Output: `key=value`, one per line, stdout. Under 1 KB in the normal case.
#
# Exit codes:
#   0  block printed
#   2  usage error
#   3  no such task            (passed through from task-resolve.sh)
#   4  ambiguous identifier    (passed through, candidates on stderr)

set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"

BASE=""
QUERY=""
while [ $# -gt 0 ]; do
    case "$1" in
        --base) BASE="${2:-}"; [ -n "$BASE" ] || { echo "task-facts.sh: --base needs a rev" >&2; exit 2; }; shift 2 ;;
        -h|--help) sed -n '2,33p' "$0"; exit 0 ;;
        -*) echo "task-facts.sh: unknown option $1" >&2; exit 2 ;;
        *) [ -z "$QUERY" ] || { echo "task-facts.sh: one identifier only" >&2; exit 2; }; QUERY="$1"; shift ;;
    esac
done

[ -n "$QUERY" ] || { echo "usage: tools/task-facts.sh <task-identifier> [--base <rev>]" >&2; exit 2; }

# task-resolve.sh owns resolution, including its exit codes. Pass them through
# unchanged so a caller's existing 3/4 handling keeps working.
resolved="$("$HERE/task-resolve.sh" "$QUERY")" || exit $?
task="$(printf '%s' "$resolved" | cut -f1)"
task_dir="$(printf '%s' "$resolved" | cut -f2)"
worktree="$(printf '%s' "$resolved" | cut -f3)"

echo "task=$task"
echo "task_dir=${task_dir#"$worktree"/}"
echo "worktree=$worktree"
echo "branch=$(git -C "$worktree" rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)"
echo "head=$(git -C "$worktree" rev-parse --short HEAD 2>/dev/null || echo unknown)"

# Which phase artifacts exist. Cheaper than the `ls` every phase command opens
# with, and it is the fact that decides whether a phase can run at all.
artifacts=""
for a in prd.md design.md plan.md context.md progress.md audit.md; do
    [ -f "$task_dir/$a" ] && artifacts="$artifacts${artifacts:+,}$a"
done
echo "artifacts=${artifacts:-none}"

# --- change surfaces -------------------------------------------------------
# Run the worktree's own copy: the branch under test may have changed the
# classifier, and the answer must come from the tree being reviewed.
surfaces_args=()
[ -n "$BASE" ] && surfaces_args=(--base "$BASE")
if [ -x "$worktree/tools/change-surfaces.sh" ]; then
    ( cd "$worktree" && ./tools/change-surfaces.sh ${surfaces_args[@]+"${surfaces_args[@]}"} ) \
        || echo "change_surfaces=unavailable"
else
    echo "change_surfaces=unavailable"
fi

# --- verification selection ------------------------------------------------
# Only the guard/gate half; the change-base and service lists above already came
# from change-surfaces.sh. `--facts --quick` runs no check.
verify_args=(--facts --quick)
[ -n "$BASE" ] && verify_args+=(--base "$BASE")
if [ -x "$worktree/tools/verify.sh" ]; then
    verify_out="$( cd "$worktree" && ./tools/verify.sh "${verify_args[@]}" 2>/dev/null )" || verify_out=""
    if [ -n "$verify_out" ]; then
        printf '%s\n' "$verify_out" | grep -E '^(modules_selected|guard_suites|fanout_reason)=' || true
        guards="$(printf '%s\n' "$verify_out" | sed -n 's/^gate=//p' | paste -sd';' -)"
        echo "applicable_guards=${guards:-none}"
    else
        echo "applicable_guards=unavailable"
    fi
else
    echo "applicable_guards=unavailable"
fi

# --- toolchain -------------------------------------------------------------
# Stated once here so agents stop probing. Measured on task-232: ~65
# `command -v` / `--version` / `which` probes across 80 of 213 streams. A stale
# list is worse than a probe, so this is generated live, never hardcoded.
tc=""
for t in go docker kubectl npm node yq shellcheck bats python3 golangci-lint staticcheck; do
    command -v "$t" >/dev/null 2>&1 && tc="$tc${tc:+,}$t"
done
echo "toolchain=${tc:-none}"
echo "go_version=$(go version 2>/dev/null | awk '{print $3}' || echo unknown)"
