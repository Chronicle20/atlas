#!/usr/bin/env bash
# task-brief.sh — extract one task's full text from an implementation plan
# into a standalone brief file that an implementer subagent reads in one call.
#
# Why this lives in the repo
# --------------------------
# `.claude/commands/execute-task.md` Step 4b instructs the controller to run
# `scripts/task-brief`. That path does not exist here — the script it means is
# the superpowers plugin's, at:
#
#     ~/.claude/plugins/cache/claude-plugins-official/superpowers/
#         <VERSION>/skills/subagent-driven-development/scripts/task-brief
#
# Two problems with pointing at that: the path is version-pinned, so it breaks
# on every plugin upgrade; and when the invocation fails, the controller's
# fallback is to assemble briefs by hand out of the full plan.md — which is
# exactly the context bloat the brief exists to prevent. Vendoring it makes the
# invocation in execute-task.md resolve to a real, stable path.
#
# Behaviour matches the plugin script, including the default output location,
# so the two can be used interchangeably against the same workspace. The
# workspace-resolution logic (upstream's separate `sdd-workspace`) is inlined
# here so this file has no sibling-script dependency.
#
# Usage: tools/task-brief.sh PLAN_FILE TASK_NUMBER [OUTFILE]
#
# Default OUTFILE: <repo-root>/.superpowers/sdd/<plan-basename>/task-<N>-brief.md
# One workspace directory per plan, so a follow-up plan in the same working
# tree cannot read or overwrite another plan's briefs and reports.
#
# The workspace lives in the working tree rather than under .git/ because
# Claude Code treats .git/ as a protected path and denies agent writes there,
# which would block an implementer from writing its report file. A
# self-ignoring .gitignore at .superpowers/sdd/ keeps it out of `git status`.
#
# Exit codes:
#   0  brief written
#   2  usage error / no such plan file
#   3  no heading matching "Task <N>" in the plan

set -euo pipefail

if [ $# -lt 2 ] || [ $# -gt 3 ]; then
  echo "usage: tools/task-brief.sh PLAN_FILE TASK_NUMBER [OUTFILE]" >&2
  exit 2
fi

plan=$1
n=$2
[ -f "$plan" ] || { echo "no such plan file: $plan" >&2; exit 2; }
[[ "$n" =~ ^[0-9]+$ ]] || { echo "task number must be numeric, got: $n" >&2; exit 2; }

if [ $# -eq 3 ]; then
  out=$3
  mkdir -p "$(dirname "$out")"
else
  slug="$(basename "$plan" .md)"
  [ -n "$slug" ] && [ "$slug" != "." ] && [ "$slug" != ".." ] \
    || { echo "cannot derive a workspace name from: $plan" >&2; exit 2; }

  root="$(git rev-parse --show-toplevel)"
  base="$root/.superpowers/sdd"
  dir="$base/$slug"
  mkdir -p "$dir"
  printf '*\n' > "$base/.gitignore"
  out="$dir/task-${n}-brief.md"
fi

# Print every line from the "## Task <N>" heading up to the next task heading
# at any level. Fenced code blocks are tracked so a "# Task 7" line inside a
# sample snippet cannot terminate (or start) a section.
awk -v n="$n" '
  /^```/ { infence = !infence }
  !infence && /^#+[ \t]+Task[ \t]+[0-9]+/ {
    intask = ($0 ~ ("^#+[ \t]+Task[ \t]+" n "([^0-9]|$)"))
  }
  intask { print }
' "$plan" > "$out"

if [ ! -s "$out" ]; then
  rm -f "$out"
  echo "task ${n} not found in ${plan} (no heading matching 'Task ${n}')" >&2
  exit 3
fi

echo "wrote ${out}: $(wc -l < "$out" | tr -d ' ') lines"
