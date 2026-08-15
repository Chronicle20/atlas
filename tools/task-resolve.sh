#!/usr/bin/env bash
# task-resolve.sh — resolve a fuzzy task identifier to its home directory.
#
# The four phase commands (/design-task, /plan-task, /execute-task) all begin
# by turning "54" / "task-054" / "effect-duration" into a concrete task folder.
# They used to do it by globbing:
#
#     docs/tasks/task-*  and  .worktrees/*/docs/tasks/task-*
#
# That second pattern is quadratic. Every worktree carries a full copy of
# docs/tasks/ from its branch point, so the glob returns (tasks x worktrees)
# paths — 2253 paths / 177KB in this repo at 217 tasks and 12 worktrees — and
# every one of them lands in the model's context just to resolve one ID. It
# also returns the SAME task many times over, since task-205's folder exists
# inside every worktree created after it.
#
# This script applies the ownership rule already used by task-numbers.sh:
#
#   A worktree owns exactly one task — the one whose ID matches the worktree
#   directory name. Every other task folder inside that worktree is a
#   branch-history copy of main's docs/tasks/ and is never the answer.
#
# That collapses the candidate set from (tasks x worktrees) to
# (tasks + worktrees) and makes each task resolve to exactly one home.
#
# Usage:
#   task-resolve.sh <identifier>          Print "<task-id>\t<task-dir>\t<worktree>"
#   task-resolve.sh --list                Print every known task, one per line
#
# <worktree> is the absolute path of the tree the task lives in — the main
# repo root for a task that has no worktree yet.
#
# Exit codes:
#   0  resolved (exactly one match)
#   2  usage error
#   3  no match
#   4  ambiguous — candidates listed on stderr for the caller to disambiguate
#
# Matching, in precedence order. The first tier that yields any match wins, so
# an exact ID never loses to a slug fragment that happens to match more folders.
#   1. Exact task ID          task-054-effect-duration-units
#   2. Number                 54 / 054 / task-54 / task-054
#   3. Slug fragment          effect-duration (substring of the slug)

set -euo pipefail

# Resolve the MAIN repo root regardless of cwd or whether we are inside a
# worktree. Same derivation as task-numbers.sh: a linked worktree's
# --git-common-dir points back at the primary .git directory.
if repo_root="$(git rev-parse --show-toplevel 2>/dev/null)"; then
  common_dir="$(git rev-parse --git-common-dir 2>/dev/null)"
  case "$common_dir" in
    /*) ;;
    *)  common_dir="$repo_root/$common_dir" ;;
  esac
  main_root="$(cd "$common_dir/.." && pwd)"
else
  main_root="$(pwd)"
fi

# Emit "<task-id>\t<task-dir>\t<worktree>" for every task that has a home.
#
# A task owned by a worktree is emitted ONLY from that worktree, never also
# from main-docs, so each task ID appears exactly once and the caller never has
# to break a tie between a task and its own stale copy on main.
candidates() {
  owned=""

  if [ -d "$main_root/.worktrees" ]; then
    for wt in "$main_root/.worktrees"/*; do
      [ -d "$wt" ] || continue
      wt_name="$(basename "$wt")"
      case "$wt_name" in task-*) ;; *) continue ;; esac
      d="$wt/docs/tasks/$wt_name"
      [ -d "$d" ] || continue
      printf '%s\t%s\t%s\n' "$wt_name" "$d" "$wt"
      owned="$owned $wt_name"
    done
  fi

  if [ -d "$main_root/docs/tasks" ]; then
    for d in "$main_root/docs/tasks"/task-*; do
      [ -d "$d" ] || continue
      tid="$(basename "$d")"
      # Skip tasks already emitted from their own worktree.
      case " $owned " in *" $tid "*) continue ;; esac
      printf '%s\t%s\t%s\n' "$tid" "$d" "$main_root"
    done
  fi
}

if [ "${1:-}" = "--list" ]; then
  candidates | sort -u
  exit 0
fi

if [ $# -ne 1 ] || [ -z "$1" ]; then
  echo "usage: task-resolve.sh <identifier> | --list" >&2
  exit 2
fi

query="$1"
all="$(candidates | sort -u)"
[ -n "$all" ] || { echo "task-resolve: no task folders found under $main_root" >&2; exit 3; }

# Normalise a number-shaped query to a zero-padded 3-digit number, so 54,
# 054, task-54 and task-054 all reduce to "054". Empty if not number-shaped.
num=""
probe="${query#task-}"
if [[ "$probe" =~ ^[0-9]+$ ]]; then
  num="$(printf '%03d' "$((10#$probe))")"
fi

match_exact()    { printf '%s\n' "$all" | awk -F'\t' -v q="$query" '$1 == q'; }
match_number()   {
  [ -n "$num" ] || return 0
  printf '%s\n' "$all" | awk -F'\t' -v n="$num" '
    { id = $1; sub(/^task-/, "", id); sub(/-.*$/, "", id)
      if (id != "" && sprintf("%03d", id + 0) == n) print }'
}
match_fragment() { printf '%s\n' "$all" | awk -F'\t' -v q="$query" 'index($1, q)'; }

hits=""
for tier in match_exact match_number match_fragment; do
  hits="$($tier)"
  [ -n "$hits" ] && break
done

count="$(printf '%s' "$hits" | grep -c . || true)"

if [ "$count" -eq 0 ]; then
  echo "task-resolve: no task matches '$query'" >&2
  exit 3
fi

if [ "$count" -gt 1 ]; then
  echo "task-resolve: '$query' is ambiguous — $count matches:" >&2
  printf '%s\n' "$hits" | awk -F'\t' '{print "  - " $1 "  (" $2 ")"}' >&2
  exit 4
fi

printf '%s\n' "$hits"
