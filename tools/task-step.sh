#!/usr/bin/env bash
# task-step.sh — advance one task boundary in a subagent-driven-development run
# with a SINGLE tool call.
#
# Why this exists
# ---------------
# Across the task-231 execute chain the controller spent 710 assistant turns on
# 588 tool calls — 1.21 turns per call, i.e. essentially no batching. The
# dominant shape was three consecutive single-call turns at every task boundary:
#
#     turn N     sed -n '/verify.sh summary/,$p' gate-27.log    (read the gate)
#     turn N+1   cat >> progress.md <<'EOF' Task 27: ...        (write the ledger)
#     turn N+2   Agent -> task-implementer  Task 28            (dispatch next)
#
# 130 of the 588 calls were that middle one — pure bookkeeping, 2-byte results,
# each its own turn. Every turn re-reads the whole context, so at the 150-220k
# where these sat, the two extra turns cost roughly 370k billed tokens per task.
# Across the branch that is 22-34M tokens, a fifth to a third of all
# main-thread input, spent on turn overhead rather than on work.
#
# This script does the ledger append, the gate read, and the next brief in one
# call, so the boundary costs one turn instead of three. The note text still
# comes from the controller (it is a judgment summary of the implementer's
# report) — but the controller already holds that when it calls this, so
# nothing is round-tripped.
#
# Usage:
#   tools/task-step.sh --plan PLAN --task N [options]
#
#   --plan PLAN        implementation plan (e.g. docs/tasks/task-231-x/plan.md)
#   --task N           the task just completed
#   --note TEXT        ledger entry to append under a "Task N" header
#   --note-file FILE   read the ledger entry from a file instead ("-" = stdin)
#   --gate LOG         verify.sh log to summarise (delegates to gate-summary.sh)
#   --next             also ensure + show the brief for task N+1 (default on)
#   --no-next          skip the next-brief step (use at the end of a plan)
#
# Exit codes:
#   0  all requested steps succeeded, and the gate (if given) passed
#   1  the gate FAILED — ledger and brief steps still ran
#   2  usage error
#   3  the gate log had no summary yet (run in progress or died)

set -u

here="$(cd "$(dirname "$0")" && pwd -P)"
root="$(git rev-parse --show-toplevel 2>/dev/null || printf '%s' "$here/..")"

plan=""; task=""; note=""; note_file=""; gate=""; do_next=1

while [ $# -gt 0 ]; do
    case "$1" in
        --plan)      plan="${2:-}"; shift 2 ;;
        --task)      task="${2:-}"; shift 2 ;;
        --note)      note="${2:-}"; shift 2 ;;
        --note-file) note_file="${2:-}"; shift 2 ;;
        --gate)      gate="${2:-}"; shift 2 ;;
        --next)      do_next=1; shift ;;
        --no-next)   do_next=0; shift ;;
        -h|--help)   sed -n '30,45p' "$0"; exit 0 ;;
        *) printf 'task-step.sh: unexpected argument: %s\n' "$1" >&2; exit 2 ;;
    esac
done

[ -n "$plan" ] || { printf 'task-step.sh: --plan is required\n' >&2; exit 2; }
[ -n "$task" ] || { printf 'task-step.sh: --task is required\n' >&2; exit 2; }
case "$task" in ''|*[!0-9]*) printf 'task-step.sh: --task must be a number\n' >&2; exit 2 ;; esac
[ -f "$plan" ] || { printf 'task-step.sh: no such plan: %s\n' "$plan" >&2; exit 2; }

# Workspace layout matches tools/task-brief.sh: one directory per plan.
plan_base="$(basename "$plan")"; plan_base="${plan_base%.md}"
ws="$root/.superpowers/sdd/$plan_base"
progress="$ws/progress.md"
mkdir -p "$ws" 2>/dev/null || true

rc=0

# ---- 1. ledger append ------------------------------------------------------
if [ -n "$note_file" ]; then
    if [ "$note_file" = "-" ]; then
        note="$(cat)"
    elif [ -f "$note_file" ]; then
        note="$(cat "$note_file")"
    else
        printf 'task-step.sh: no such note file: %s\n' "$note_file" >&2
        exit 2
    fi
fi

if [ -n "$note" ]; then
    {
        printf '\nTask %s: %s\n' "$task" "$note"
    } >> "$progress"
    printf 'LEDGER  appended %s lines to %s\n' \
        "$(printf '%s\n' "$note" | wc -l | tr -d ' ')" \
        "${progress#"$root/"}"
else
    printf 'LEDGER  (no --note given, nothing appended)\n'
fi

# ---- 2. gate verdict -------------------------------------------------------
if [ -n "$gate" ]; then
    printf '\nGATE    %s\n' "$gate"
    "$here/gate-summary.sh" "$gate" || rc=$?
fi

# ---- 3. next brief ---------------------------------------------------------
if [ "$do_next" -eq 1 ]; then
    next=$((task + 1))
    brief="$ws/task-$next-brief.md"
    printf '\nNEXT    task %s\n' "$next"
    if [ ! -f "$brief" ]; then
        if ! "$here/task-brief.sh" "$plan" "$next" >/dev/null 2>&1; then
            printf '  no task %s in %s — plan complete?\n' "$next" "${plan#"$root/"}"
            exit "$rc"
        fi
    fi
    if [ -f "$brief" ]; then
        printf '  brief: %s (%s lines)\n' "${brief#"$root/"}" \
            "$(wc -l < "$brief" | tr -d ' ')"
        printf '  ---- Files section ----\n'
        awk '/^### Files/{f=1} f{print "  " $0} f&&/^$/{n++; if(n>1) exit}' "$brief" \
            | head -30
    fi
fi

exit "$rc"
