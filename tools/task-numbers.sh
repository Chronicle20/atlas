#!/usr/bin/env bash
# task-numbers.sh — single source of truth for task-NNN-* numbering.
#
# Scans every place a `task-NNN-*` identifier can live:
#   - docs/tasks/           (main repo)
#   - .worktrees/*/docs/tasks/
#   - local git branches matching `task-*`
#   - remote git branches   (refs/remotes/*/task-*)
#   - git history           (task-NNN in any commit subject, all refs)
#
# The history source is what stops `next` from re-issuing the number of a task
# whose branch was merged-and-deleted and whose docs/tasks folder no longer
# lives in the working tree — the only durable record left is the merge-commit
# subject. (e.g. 052/058/059 shipped via PRs #380/#408/#412 yet vanished from
# every current-state source; see task-numbers_test.sh.)
#
# Subcommands:
#   next   Print the smallest unused 3-digit NNN.
#   check  Exit 1 + report on stderr if any NNN has more than one distinct task ID.
#   list   Print "NNN <task-id> <source>" for every assignment seen.
#
# spec-task.md MUST call `tools/task-numbers.sh next` to pick the next number.
# Picking by hand has caused collisions (two tasks both numbered 063, May 2026).

set -euo pipefail

# Resolve the main repo root regardless of cwd or whether we're inside a worktree.
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

scan() {
  if [ -d "$main_root/docs/tasks" ]; then
    for d in "$main_root/docs/tasks"/task-*; do
      [ -d "$d" ] || continue
      tid="$(basename "$d")"
      num="${tid#task-}"; num="${num%%-*}"
      [[ "$num" =~ ^[0-9]+$ ]] && printf '%03d %s main-docs\n' "$((10#$num))" "$tid"
    done
  fi
  if [ -d "$main_root/.worktrees" ]; then
    # A worktree only "owns" the task whose ID matches the worktree directory
    # name. Other task folders inside the worktree are just branch-history
    # copies of main's docs/tasks/ and would flood the scan with noise.
    for wt in "$main_root/.worktrees"/*; do
      wt_name="$(basename "$wt")"
      case "$wt_name" in task-*) ;; *) continue ;; esac
      d="$wt/docs/tasks/$wt_name"
      [ -d "$d" ] || continue
      tid="$wt_name"
      num="${tid#task-}"; num="${num%%-*}"
      [[ "$num" =~ ^[0-9]+$ ]] && printf '%03d %s worktree:%s\n' "$((10#$num))" "$tid" "$wt_name"
    done
  fi
  if git -C "$main_root" rev-parse --git-dir >/dev/null 2>&1; then
    while read -r ref; do
      [ -n "$ref" ] || continue
      tid="${ref#refs/heads/}"
      num="${tid#task-}"; num="${num%%-*}"
      [[ "$num" =~ ^[0-9]+$ ]] && printf '%03d %s branch\n' "$((10#$num))" "$tid"
    done < <(git -C "$main_root" for-each-ref --format='%(refname)' 'refs/heads/task-*')

    # Remote task branches: pushed-but-not-checked-out work (e.g. a teammate's
    # open PR, or a branch this clone never fetched as a local head) that a
    # local-only scan would miss.
    while read -r ref; do
      [ -n "$ref" ] || continue
      tid="${ref##*/}"
      num="${tid#task-}"; num="${num%%-*}"
      [[ "$num" =~ ^[0-9]+$ ]] && printf '%03d %s remote\n' "$((10#$num))" "$tid"
    done < <(git -C "$main_root" for-each-ref --format='%(refname)' 'refs/remotes/*/task-*')

    # Git history (all refs): the only durable record of a merged-and-deleted
    # task's number once its branch and docs/tasks folder are gone. We mine
    # `task-NNN` tokens out of commit subjects (merge subjects carry the branch
    # name; conventional-commit scopes carry `task-NNN`). Numbers only — the
    # full slug isn't reliably present, and `next`/dedup care only about NNN.
    while read -r num; do
      [ -n "$num" ] || continue
      printf '%03d task-%03d history\n' "$((10#$num))" "$((10#$num))"
    done < <(git -C "$main_root" log --all --pretty=%s 2>/dev/null \
               | grep -oiE 'task-0*[0-9]+' \
               | grep -oE '[0-9]+' | sort -un)
  fi
}

cmd="${1:-next}"
case "$cmd" in
  list)
    scan | sort -u
    ;;
  next)
    used="$(scan | awk '{print $1}' | sort -un)"
    n=1
    while printf '%s\n' "$used" | grep -qx "$(printf '%03d' "$n")"; do
      n=$((n + 1))
    done
    printf '%03d\n' "$n"
    ;;
  check)
    # Only flag a collision when at least one of the colliding task IDs is
    # currently in-flight (worktree or local branch source). Collisions among
    # already-landed sources (main-docs, remote, history) are historical
    # (e.g. tasks 014 and 016) and can't be un-shipped, so flagging them every
    # session would just be noise.
    # Drop history rows: they carry a number-only synthetic ID (`task-NNN`,
    # no slug) that exists solely to mark a number as used for `next`. Counting
    # it as a task ID distinct from the real slug'd source would false-flag
    # every in-flight task (e.g. history `task-081` vs branch `task-081-foo`).
    data="$(scan | sort -u | awk '$3 != "history"')"
    inflight="$(printf '%s\n' "$data" | awk '$3 == "branch" || $3 ~ /^worktree:/ {print $2}' | sort -u)"
    bad=0
    while read -r num; do
      [ -n "$num" ] || continue
      tids="$(printf '%s\n' "$data" | awk -v n="$num" '$1==n {print $2}' | sort -u)"
      count="$(printf '%s\n' "$tids" | grep -c .)"
      [ "$count" -le 1 ] && continue
      hit=0
      while read -r tid; do
        [ -n "$tid" ] || continue
        if printf '%s\n' "$inflight" | grep -qx "$tid"; then
          hit=1; break
        fi
      done < <(printf '%s\n' "$tids")
      [ "$hit" -eq 1 ] || continue
      echo "task-num collision: $num has $count distinct task IDs (at least one in-flight):" >&2
      printf '%s\n' "$tids" | sed 's/^/  - /' >&2
      echo "  sources:" >&2
      printf '%s\n' "$data" | awk -v n="$num" '$1==n {print "    " $2 "  (" $3 ")"}' | sort -u >&2
      bad=1
    done < <(printf '%s\n' "$data" | awk '{print $1}' | sort -u)
    exit $bad
    ;;
  *)
    echo "usage: $0 {next|check|list}" >&2
    exit 2
    ;;
esac
