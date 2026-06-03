#!/usr/bin/env bash
# PreToolUse hook — block Write/Edit ops that bake absolute home paths
# (/home/<user>/...) into anything under docs/. The literal username trips
# gitleaks downstream and leaks PII into committed plan/design/audit docs.
#
# Silent on the happy path. On a hit, emits a JSON deny with guidance so
# Claude rewrites the path as repo-relative before retrying.

set -u

# No stdin → can't decide, allow.
[ -t 0 ] && exit 0

input="$(cat)"

decision="$(printf '%s' "$input" | jq -rc '
  (.tool_input.file_path // "") as $fp |
  (.tool_input.content // .tool_input.new_string // "") as $c |
  if ($fp | test("(^|/)docs/")) and ($c | test("/home/[A-Za-z0-9_.-]+/"))
  then
    {hookSpecificOutput: {
      hookEventName: "PreToolUse",
      permissionDecision: "deny",
      permissionDecisionReason: "Refused: absolute home path (/home/<user>/...) detected in a docs/ file. Rewrite as a repo-relative path (e.g. docs/tasks/<task>/...) or a <repo-root>/... placeholder before retrying — literal usernames trip gitleaks."
    }}
  else
    empty
  end
' 2>/dev/null)"

[ -z "$decision" ] && exit 0

printf '%s\n' "$decision"
exit 0
