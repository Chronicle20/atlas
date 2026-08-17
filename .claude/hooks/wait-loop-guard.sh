#!/usr/bin/env bash
# PreToolUse hook — refuse turns that exist only to wait.
#
# CLAUDE.md and docs/tooling-conventions.md already forbid polling a process.
# The rule was advisory, and one task violated it in three separate sessions.
# This makes it machine-checked, the same way fork-dispatch-guard.sh made the
# fork rule machine-checked. Measured evidence for both halves:
#
#   No-op waits. In one `backend-guidelines-reviewer` (83 turns, 92 tool calls,
#   9,994,299 billed input), 30 of the 92 tool calls were `true` returning 31
#   bytes — clustered immediately after six async child dispatches, burning
#   turns to stay alive while they ran. At that agent's ~120k mean per turn
#   that is ≈3.6M tokens, ~36% of the entire agent, for zero information.
#
#   Process polling. One main thread issued 20 `pgrep`/`ps`/`sleep` calls at
#   170–290k context (≈4.6M); another 11; a third, 22 consecutive. The single
#   largest result was a 9.8 KB `pgrep -af` dump. The correct pattern was in
#   use in the SAME task three sessions earlier: `Monitor` with an `until [ -f
#   … ]` loop, returning 209 bytes.
#
# What it denies, and what it deliberately does not
# -------------------------------------------------
# Denied: a command that is only `true`/`:`; a bare `sleep N` with nothing else
# to do; a sleep-driven poll loop; a broad process listing (`ps aux`, `ps -ef`,
# `pgrep`) used as a wait.
#
# Allowed, always — these are legitimate process debugging, and a guard that
# blocked them would be worse than the polling it prevents:
#   - `ps -p <pid>` / `ps -o …` — inspecting a specific process
#   - `kill` / `pkill` on its own — terminating something is not waiting for it
#   - `kubectl`, `docker ps`, `systemctl`, `top -b -n1`, journalctl
#   - anything inside a heredoc, a quoted string, or a file being written (the
#     guard reads the command, and a script containing `sleep` is content, not
#     a poll — matched only when the sleep is the shell command itself)
#   - anything with a `POLL-JUSTIFIED:` prefix or comment, mirroring
#     `FORK-JUSTIFIED:` — a considered wait costs one sentence
#
# Silent on the happy path. Always exits 0.

set -u

[ -t 0 ] && exit 0

input="$(cat)"

decision="$(printf '%s' "$input" | jq -rc '
  (.tool_input.command // "") as $cmd |

  # An explicit justification always passes, wherever it appears in the command.
  if ($cmd | test("POLL-JUSTIFIED:")) then empty

  # A pure no-op turn: `true`, `:`, `true && true`, `echo waiting`.
  elif ($cmd | test("^\\s*(true|:)\\s*(&&\\s*(true|:)\\s*)*$"))
    or ($cmd | test("^\\s*echo\\s+[\"'"'"']?(waiting|wait|still waiting|polling)[\"'"'"']?\\s*$"; "i"))
  then
    {hookSpecificOutput: {
      hookEventName: "PreToolUse",
      permissionDecision: "deny",
      permissionDecisionReason: "Refused: no-op tool call. This turn produces no information and still costs a full context re-read — measured at ~120k tokens per turn inside a large agent, where 30 such calls consumed 36% of the agent.\n\nIf you are waiting on a child agent: you do not need to. Agent completions arrive as notifications; do other work, or end your turn and be re-invoked. If you are waiting on a process: launch it with `run_in_background: true`, or use `Monitor` with an until-loop and a timeout.\n\nIf you truly need a no-op, prefix the command with `POLL-JUSTIFIED: <reason>` as a comment."
    }}

  # Sleep as the shell command: a bare sleep, or a sleep-driven poll loop.
  elif ($cmd | test("^\\s*sleep\\s+[0-9.]+\\s*$"))
    or ($cmd | test("^\\s*sleep\\s+[0-9.]+\\s*(;|&&)"))
    or ($cmd | test("(while|until|for)\\b[^;]*;.*\\bsleep\\s+[0-9.]"))
  then
    {hookSpecificOutput: {
      hookEventName: "PreToolUse",
      permissionDecision: "deny",
      permissionDecisionReason: "Refused: sleeping to wait. Each poll turn re-reads the whole context to learn nothing, and they cluster late in a session where that is most expensive — 20 of them at 170–290k context cost ≈4.6M tokens in one measured session.\n\nUse instead: `run_in_background: true` on the long command (you are re-invoked when it exits), or `Monitor` with an `until [ -f <sentinel> ]` loop and an explicit timeout — that returned 209 bytes per call in the same task.\n\nIf this sleep is part of real work (a rate limit, a settle delay inside a test), prefix the command with `POLL-JUSTIFIED: <reason>`."
    }}

  # Broad process listing used as a wait. `ps -p`/`ps -o` (a named process) and
  # kill/pkill (terminating, not waiting) are not matched.
  elif ($cmd | test("(^|[|;&]\\s*)(pgrep\\b|ps\\s+(aux|-ef|-e)\\b)"))
    and ($cmd | test("\\bkill\\b") | not)
  then
    {hookSpecificOutput: {
      hookEventName: "PreToolUse",
      permissionDecision: "deny",
      permissionDecisionReason: "Refused: broad process listing as a wait. A `ps aux` / `pgrep` sweep to check whether something is still running is the polling anti-pattern CLAUDE.md forbids; one such call returned a 9.8 KB dump into a 290k context.\n\nTo know whether a background command finished: it notifies you when it exits. To wait on a condition: `Monitor` with an until-loop. To inspect ONE known process: `ps -p <pid>` or `ps -o …`, which this guard allows.\n\nIf you are genuinely debugging process state rather than waiting, prefix with `POLL-JUSTIFIED: <reason>`."
    }}

  else empty
  end
' 2>/dev/null)"

[ -z "$decision" ] && exit 0

printf '%s\n' "$decision"
exit 0
