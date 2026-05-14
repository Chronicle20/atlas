#!/usr/bin/env bash
# SessionStart / UserPromptSubmit hook — surface task-NNN-* collisions
# to the model so they get fixed before /spec-task or /design-task land
# more work on the wrong number.
#
# Silent on the happy path. On a collision, prints a system-reminder block
# to stdout (which becomes additional context for the model in these hook
# events) and exits 0 so it never breaks the session.

set -u

repo_root="${CLAUDE_PROJECT_DIR:-$(pwd)}"
script="$repo_root/tools/task-numbers.sh"

[ -x "$script" ] || exit 0

# Drain stdin so we don't deadlock the harness on hooks that pipe JSON in.
if [ ! -t 0 ]; then
  cat >/dev/null || true
fi

out="$("$script" check 2>&1)" || true
[ -z "$out" ] && exit 0

cat <<EOF
<system-reminder>
Task-number collision detected. Resolve before running /spec-task, /design-task,
/plan-task, or /execute-task — the affected task IDs share a number and downstream
tooling (PR titles, dashboards, cross-references) cannot disambiguate them.

$out

Run \`tools/task-numbers.sh list\` to see all assignments.
Renumber the lower-priority task by renaming its branch, worktree, and
docs/tasks/<id> folder (then commit). Re-run this check to confirm.
</system-reminder>
EOF
