#!/usr/bin/env bash
# PreToolUse hook (Agent) — make the ~150k controller handoff BINDING.
#
# commit-boundary.sh is PostToolUse: past ESCALATE tool calls it can only
# inject text saying "the default is now HAND OFF." Two consecutive weekly
# audits (2026-08-24, 2026-08-29) measured that text being declined: sampled
# controllers ran 60–90% of their context above 150k after the prompt fired,
# and in one 263-turn session the USER twice asked "should we hand off?" and
# the controller kept dispatching. An unenforced rule is not a rule.
#
# This hook is the enforcement half, shaped like turn-budget-guard.sh. Past
# the ESCALATE call count (read from commit-boundary.sh so the threshold is
# stated once) it DENIES dispatching a NEW unit of work — an implementer, an
# explorer, a planner, a bare general-purpose agent — and allows everything a
# controller needs to FINISH the unit it is on and hand off: reviewers,
# verifiers, adherence and guideline auditors, documentation. A finishing
# dispatch at 200k is the plan working; a fresh implementer at 200k is the
# controller paying full freight for context the implementer never sees.
#
# Escape hatch: a prompt containing `CONTEXT-JUSTIFIED: <reason>` passes,
# mirroring FORK-JUSTIFIED / POLL-JUSTIFIED — a considered exception costs
# one sentence stating what the next unit needs from this conversation that
# the ledger cannot carry.
#
# Scope: CONTROLLER ONLY (no agent_id). Subagents are leaf or near-leaf and
# are already bounded by turn-budget-guard.sh.
#
# Silent on the happy path. Always exits 0.

set -u

[ -t 0 ] && exit 0
input="$(cat)" || exit 0

agent="$(printf '%s' "$input" | jq -r '.agent_id // ""' 2>/dev/null | tr -cd 'A-Za-z0-9._-')"
[ -z "$agent" ] || exit 0            # subagent: not our concern

session="$(printf '%s' "$input" | jq -r '.session_id // ""' 2>/dev/null | tr -cd 'A-Za-z0-9._-')"
[ -n "$session" ] || exit 0

hook_dir="$(cd "$(dirname "$0")" && pwd -P)"
ESCALATE="$(sed -n 's/^ESCALATE=\([0-9][0-9]*\).*/\1/p' "$hook_dir/commit-boundary.sh" 2>/dev/null | head -1)"
case "${ESCALATE:-}" in ''|*[!0-9]*) ESCALATE=60 ;; esac

counter="${TMPDIR:-/tmp}/claude-turn-budget/session-$session"
[ -f "$counter" ] || exit 0
count="$(cat "$counter" 2>/dev/null || echo 0)"
case "$count" in ''|*[!0-9]*) count=0 ;; esac
[ "$count" -ge "$ESCALATE" ] || exit 0

type="$(printf '%s' "$input" | jq -r '.tool_input.subagent_type // ""' 2>/dev/null)"
brief="$(printf '%s' "$input" | jq -r '(.tool_input.prompt // "") + " " + (.tool_input.description // "")' 2>/dev/null)"

printf '%s' "$brief" | grep -q 'CONTEXT-JUSTIFIED:' && exit 0

# Finishing dispatches: they close the unit in flight and feed the ledger.
case "$type" in
    task-reviewer|task-verifier|plan-adherence-reviewer|backend-guidelines-reviewer|\
    frontend-guidelines-reviewer|packet-completeness-critic|family-auditor|\
    service-documentation|todo-scanner|statusline-setup|claude-code-guide)
        exit 0 ;;
esac

reason="Refused: new work unit dispatched at $count controller tool calls (handoff threshold $ESCALATE ≈ 150k context).

\`$type\` starts a NEW unit. CLAUDE.md \"Handing off context\" and execute-task.md Step 4e say the controller never carries past ~150k — unconditionally, no carve-out for tasks remaining. Every turn from here re-reads all of it; in the measured sessions 60–90% of total spend accrued after this point, for work that was fully resumable from the ledger.

Finishing the unit in flight is still allowed (task-reviewer, task-verifier, plan-adherence-reviewer, guideline reviewers, service-documentation). Then:
  1. Write the diagnosis / progress down (progress.md, task folder).
  2. Record it: tools/agent-ledger.sh --kind handoff --unit \"after <unit>\" --context-tokens <n>.
  3. Tell the user: \`/clear\` and re-run the phase command — it resumes from the ledger. Then stop.

If the next unit genuinely needs state that exists only in this conversation, retry with a line starting \`CONTEXT-JUSTIFIED:\` in the prompt saying what that state is."

jq -nc --arg r "$reason" '{
  hookSpecificOutput: {
    hookEventName: "PreToolUse",
    permissionDecision: "deny",
    permissionDecisionReason: $r
  }
}' 2>/dev/null || exit 0
exit 0
