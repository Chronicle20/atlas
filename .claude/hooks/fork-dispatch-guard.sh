#!/usr/bin/env bash
# PreToolUse hook — make the cost of `subagent_type: "fork"` visible at dispatch.
#
# A fork inherits the parent's entire conversation and re-reads it on EVERY
# turn. Nothing at the call site hints at this: spawning a fork looks identical
# to spawning any other agent, and it needs no brief, which is exactly what
# makes it the tempting choice mid-task. The cost lands invisibly and later.
#
# Observed shape of the problem: a review that fanned out to four forks, one of
# which forked three more, peaked at ~236k tokens in a child -- above the parent
# session's own peak -- and ran 76 turns there. Every one of those turns re-read
# a context assembled for a different purpose. The same audit, sharded to fresh
# agents given a task-folder path and a line range, starts near 20k.
#
# So: fresh-context agent + explicit brief is the default. Fork is for
# continuing an interactive thread whose brief would be longer than the context
# it saves -- rare, but real.
#
# This is a WARN with an escape hatch, not a ban. It denies a fork dispatch that
# carries no justification, and tells you how to justify one. A fork you have
# actually thought about goes through on the retry. That asymmetry is the point:
# the reflexive fork is blocked, the considered fork costs one sentence.
#
# Silent on the happy path (any non-fork dispatch). Always exits 0.
#
# Referenced from CLAUDE.md "Model & Cost Preferences".

set -u

# No stdin -> can't decide, allow.
[ -t 0 ] && exit 0

input="$(cat)"

decision="$(printf '%s' "$input" | jq -rc '
  (.tool_input.subagent_type // "") as $type |
  ((.tool_input.prompt // "") + " " + (.tool_input.description // "")) as $brief |
  if ($type == "fork") and ($brief | test("FORK-JUSTIFIED:") | not)
  then
    {hookSpecificOutput: {
      hookEventName: "PreToolUse",
      permissionDecision: "deny",
      permissionDecisionReason: "Refused: unjustified fork dispatch. A fork inherits this entire conversation and re-reads it on every turn, so a forked child running 70+ turns costs several times a briefed agent doing the same work — and none of that is visible at the call site.\n\nDefault instead: dispatch a named agent type with an explicit brief. If you are sharding an audit or a review, give each child the artifact path plus its own scope (a line range, a file list, a task range) — that is a complete brief and it starts near-empty.\n\nIf a fork is genuinely right — you are continuing an interactive debugging thread whose brief would be longer than the context it saves — retry with a line starting `FORK-JUSTIFIED:` in the prompt, stating what the child needs from this conversation that a brief cannot carry."
    }}
  else
    empty
  end
' 2>/dev/null)"

[ -z "$decision" ] && exit 0

printf '%s\n' "$decision"
exit 0
