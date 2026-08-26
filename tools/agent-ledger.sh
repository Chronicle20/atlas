#!/usr/bin/env bash
# tools/agent-ledger.sh — one append-only TSV line per agent, per task.
#
# Both cost audits ended the same way: the findings were real, but recovering
# them took hundreds of lines of ad-hoc transcript parsing, and the most
# important questions could not be answered at all. This is the minimum that
# makes a future audit a `sort | awk` instead of a reconstruction.
#
# It is deliberately NOT an observability system. One file per task, one line
# per agent, nine core columns, no daemon, no schema migration.
#
# What it answers that the transcripts do not
# -------------------------------------------
#   - agent identity <-> plan unit. Linking `agent-a64868a95292e8c3c` to
#     "Task 9E review" required matching dispatch prompts by hand.
#   - reviewer verdict, and whether a review caused a fix. 84 reviews in one
#     task produced 1 explicit Critical; how many of the rest were load-bearing
#     is simply unknown.
#   - the cost of a declined handoff. There is no marker anywhere for "a
#     handoff was written and the session kept going" — which is the exact
#     failure the controller ceiling exists to prevent. `--kind handoff` with
#     `--context-tokens` records it.
#
# UNKNOWN IS `-`, NEVER A GUESS. If the runtime does not hand you a turn count,
# write nothing for it and the column stays `-`. A fabricated number here
# silently poisons the next audit, which is worse than a gap. Do not build a
# transcript parser to fill these in; partial data beats invented data, and
# beats none.
#
# Usage:
#   tools/agent-ledger.sh append <task> [field flags...]
#   tools/agent-ledger.sh summary <task>
#   tools/agent-ledger.sh path <task>
#
# <task> is anything tools/task-resolve.sh accepts.
#
# Field flags (all optional except --unit and --agent-type):
#   --unit <s>              plan task / review unit this agent served
#   --agent-type <s>        task-implementer, task-reviewer, ...
#   --model <s>             sonnet | haiku | opus
#   --turns <n>             assistant turns
#   --tool-calls <n>
#   --tool-result-bytes <n>
#   --return-bytes <n>      size of the return that entered the controller
#   --status <s>            DONE | PARTIAL | BLOCKED | PASS | FAIL | ...
#   --commit <sha>
#   --verdict <s>           reviewers: APPROVED | APPROVED_WITH_FINDINGS | CHANGES_REQUIRED
#   --caused-fix <yes|no>   reviewers: did this review produce a fix commit
#   --artifact <path>       durable artifact this agent wrote
#   --context-tokens <n>    controller context at this moment (handoff rows)
#   --kind <agent|handoff>  default: agent
#
# Exit codes: 0 ok · 2 usage · 3/4 passed through from task-resolve.sh

set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"

COLUMNS="ts	kind	unit	agent_type	model	turns	tool_calls	tool_result_bytes	return_bytes	status	commit	verdict	caused_fix	artifact	context_tokens"

usage() { sed -n '2,55p' "$0"; }

cmd="${1:-}"
case "$cmd" in
    append|summary|path) shift ;;
    -h|--help) usage; exit 0 ;;
    *) usage >&2; exit 2 ;;
esac

task_query="${1:-}"
[ -n "$task_query" ] || { echo "agent-ledger.sh: need a task identifier" >&2; exit 2; }
shift

resolved="$("$HERE/task-resolve.sh" "$task_query")" || exit $?
task_dir="$(printf '%s' "$resolved" | cut -f2)"
ledger="$task_dir/agent-ledger.tsv"

if [ "$cmd" = "path" ]; then
    echo "$ledger"
    exit 0
fi

# --------------------------------------------------------------------- append

if [ "$cmd" = "append" ]; then
    kind="agent"; unit="-"; agent_type="-"; model="-"; turns="-"; tool_calls="-"
    tool_result_bytes="-"; return_bytes="-"; status="-"; commit="-"; verdict="-"
    caused_fix="-"; artifact="-"; context_tokens="-"

    # A field given an empty value stays `-`. That is the difference between
    # "not measured" and "measured as zero", and the audit needs both.
    set_field() { [ -n "$2" ] && printf '%s' "$2" || printf '%s' "-"; }

    while [ $# -gt 0 ]; do
        case "$1" in
            --kind)              kind="$(set_field "$1" "${2:-}")"; shift 2 ;;
            --unit)              unit="$(set_field "$1" "${2:-}")"; shift 2 ;;
            --agent-type)        agent_type="$(set_field "$1" "${2:-}")"; shift 2 ;;
            --model)             model="$(set_field "$1" "${2:-}")"; shift 2 ;;
            --turns)             turns="$(set_field "$1" "${2:-}")"; shift 2 ;;
            --tool-calls)        tool_calls="$(set_field "$1" "${2:-}")"; shift 2 ;;
            --tool-result-bytes) tool_result_bytes="$(set_field "$1" "${2:-}")"; shift 2 ;;
            --return-bytes)      return_bytes="$(set_field "$1" "${2:-}")"; shift 2 ;;
            --status)            status="$(set_field "$1" "${2:-}")"; shift 2 ;;
            --commit)            commit="$(set_field "$1" "${2:-}")"; shift 2 ;;
            --verdict)           verdict="$(set_field "$1" "${2:-}")"; shift 2 ;;
            --caused-fix)        caused_fix="$(set_field "$1" "${2:-}")"; shift 2 ;;
            --artifact)          artifact="$(set_field "$1" "${2:-}")"; shift 2 ;;
            --context-tokens)    context_tokens="$(set_field "$1" "${2:-}")"; shift 2 ;;
            *) echo "agent-ledger.sh: unknown field $1" >&2; exit 2 ;;
        esac
    done

    case "$kind" in agent|handoff) ;; *) echo "agent-ledger.sh: --kind must be agent or handoff" >&2; exit 2 ;; esac
    [ "$unit" != "-" ] || { echo "agent-ledger.sh: --unit is required" >&2; exit 2; }
    if [ "$kind" = "agent" ] && [ "$agent_type" = "-" ]; then
        echo "agent-ledger.sh: --agent-type is required for an agent row" >&2; exit 2
    fi

    # Tabs and newlines would corrupt the row; a field carrying one is a caller
    # bug, so squash rather than reject — a lost ledger line is worse than a
    # flattened one.
    clean() { printf '%s' "$1" | tr '\t\n' '  ' | sed 's/  */ /g; s/^ //; s/ $//'; }

    mkdir -p "$task_dir"
    [ -f "$ledger" ] || printf '%s\n' "$COLUMNS" > "$ledger"

    printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
        "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        "$(clean "$kind")" "$(clean "$unit")" "$(clean "$agent_type")" "$(clean "$model")" \
        "$(clean "$turns")" "$(clean "$tool_calls")" "$(clean "$tool_result_bytes")" \
        "$(clean "$return_bytes")" "$(clean "$status")" "$(clean "$commit")" \
        "$(clean "$verdict")" "$(clean "$caused_fix")" "$(clean "$artifact")" \
        "$(clean "$context_tokens")" >> "$ledger"

    echo "$ledger"
    exit 0
fi

# -------------------------------------------------------------------- summary

if [ ! -f "$ledger" ]; then
    echo "agent-ledger.sh: no ledger at $ledger" >&2
    exit 0
fi

awk -F'\t' '
    function med(arr, n,   i, tmp, c) {
        c = 0
        for (i = 1; i <= n; i++) if (arr[i] != "-" && arr[i] != "") tmp[++c] = arr[i] + 0
        if (c == 0) return "-"
        # insertion sort; these lists are dozens of rows, not thousands
        for (i = 2; i <= c; i++) { v = tmp[i]; j = i - 1
            while (j > 0 && tmp[j] > v) { tmp[j+1] = tmp[j]; j-- } ; tmp[j+1] = v }
        return (c % 2) ? tmp[(c+1)/2] : int((tmp[c/2] + tmp[c/2+1]) / 2)
    }
    NR == 1 { next }
    $2 == "handoff" { handoffs++; if ($15 != "-") { hctx[++hn] = $15 }; next }
    {
        t = $4
        n[t]++
        if ($6  != "-") turns[t] += $6
        if ($7  != "-") calls[t] += $7
        rb[t, n[t]] = $9
        if ($12 == "APPROVED")               approved++
        else if ($12 == "APPROVED_WITH_FINDINGS") approved_f++
        else if ($12 == "CHANGES_REQUIRED")  changes++
        if ($13 == "yes") fixes++
        if ($13 != "-")   reviewed++
        rows++
    }
    END {
        printf "agents=%d\n", rows
        for (t in n) {
            k = 0; delete list
            for (i = 1; i <= n[t]; i++) list[++k] = rb[t, i]
            printf "%s: n=%d turns=%s tool_calls=%s median_return_bytes=%s\n",
                   t, n[t], (turns[t] ? turns[t] : "-"), (calls[t] ? calls[t] : "-"), med(list, k)
        }
        if (approved || approved_f || changes)
            printf "verdicts: APPROVED=%d APPROVED_WITH_FINDINGS=%d CHANGES_REQUIRED=%d\n",
                   approved, approved_f, changes
        if (reviewed)
            printf "reviews_that_caused_a_fix=%d of %d\n", fixes, reviewed
        if (handoffs) {
            k = 0; delete list
            for (i = 1; i <= hn; i++) list[++k] = hctx[i]
            printf "handoffs=%d median_context_tokens=%s\n", handoffs, med(list, k)
        }
    }
' "$ledger"
