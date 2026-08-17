#!/usr/bin/env bash
# tools/doc-slice.sh — read part of a large document instead of all of it.
#
# One accessor for every "I need three rows out of a 126 KB audit" and "I need
# Pattern C out of the wiring recipe" situation. Works on Markdown, on plain
# text, and on the offloaded `tool-results/*.txt` files a large tool result
# spills to.
#
# Why it exists
# -------------
# Measured on task-232: `service-wiring-recipe.md` (23 KB, read 26 times by 25
# distinct agent streams, nearly always whole) and `query-scope-audit.md`
# (126 KB, read 48 times) together cost **69.3 MB-turns — 7.5% of the task's
# entire tool-result carry — from 74 of 11,102 tool calls.** Each of those
# agents needed one Pattern section or a handful of service rows. And the
# position makes it worse than the size suggests: the median >12 KB result
# lands at 0.10 of its stream, so it is re-billed on ~90% of that stream's
# turns.
#
# The documents are not the problem — they are the reason 40+ services were
# wired consistently. THE ACCESS PATTERN IS THE PROBLEM. This script changes the
# access pattern and leaves every document complete.
#
# Slice first; escalate when necessary. See docs/slice-first.md. If the slice
# turns out to be insufficient, read the whole file — under-reading and getting
# it wrong costs a fix round, which is far more than the read saved.
#
# Usage:
#   tools/doc-slice.sh <file> --outline
#   tools/doc-slice.sh <file> --section <pattern> [--section <pattern>]...
#   tools/doc-slice.sh <file> --rows <pattern> [--rows <pattern>]...
#   tools/doc-slice.sh <file> --grep <pattern> [--context N]
#   tools/doc-slice.sh <file> --lines <A-B>
#
#   --outline           every heading with its line number and section size, so
#                       the next call can name a section instead of guessing
#   --section <pat>     the body of each heading matching <pat> (case-insensitive
#                       ERE), through the next heading at the same or higher
#                       level. Repeatable.
#   --rows <pat>        matching lines. In a Markdown table the header row and
#                       separator are emitted first so the columns are readable.
#                       Repeatable — this is the "rows for these three services"
#                       mode.
#   --grep <pat>        matching lines with --context lines either side
#                       (default 2). The mode for offloaded tool results.
#   --context N         context lines for --grep
#   --lines A-B         a literal line range
#   --max-bytes N       truncate output at N bytes and say so (default 65536).
#                       A truncation notice always names the source file so the
#                       caller can escalate to a full read.
#
# Exit codes:
#   0  slice printed
#   2  usage error / no such file
#   3  pattern matched nothing — say so, do not print the whole file

set -euo pipefail

FILE=""
MODE=""
PATTERNS=()
CONTEXT=2
RANGE=""
MAX_BYTES=65536

die() { echo "doc-slice.sh: $1" >&2; exit "${2:-2}"; }

while [ $# -gt 0 ]; do
    case "$1" in
        --outline)   MODE="outline"; shift ;;
        --section)   MODE="section"; PATTERNS+=("${2:?--section needs a pattern}"); shift 2 ;;
        --rows)      MODE="rows";    PATTERNS+=("${2:?--rows needs a pattern}");    shift 2 ;;
        --grep)      MODE="grep";    PATTERNS+=("${2:?--grep needs a pattern}");    shift 2 ;;
        --context)   CONTEXT="${2:?--context needs a number}"; shift 2 ;;
        --lines)     MODE="lines";   RANGE="${2:?--lines needs A-B}"; shift 2 ;;
        --max-bytes) MAX_BYTES="${2:?--max-bytes needs a number}"; shift 2 ;;
        -h|--help)   sed -n '2,57p' "$0"; exit 0 ;;
        -*)          die "unknown option $1" ;;
        *)           [ -z "$FILE" ] || die "one file only"; FILE="$1"; shift ;;
    esac
done

[ -n "$FILE" ] || die "usage: tools/doc-slice.sh <file> <mode>"
[ -f "$FILE" ] || die "no such file: $FILE"
[ -n "$MODE" ] || die "pick a mode: --outline, --section, --rows, --grep, --lines"
[[ "$CONTEXT" =~ ^[0-9]+$ ]] || die "--context must be a number"
[[ "$MAX_BYTES" =~ ^[0-9]+$ ]] || die "--max-bytes must be a number"

total_lines="$(wc -l < "$FILE" | tr -d ' ')"
total_bytes="$(wc -c < "$FILE" | tr -d ' ')"

emit() {
    # Cap the output and always say when it was capped, naming the source so the
    # caller can escalate deliberately rather than silently acting on a partial.
    local out; out="$(cat)"
    if [ "${#out}" -gt "$MAX_BYTES" ]; then
        printf '%s\n' "${out:0:$MAX_BYTES}"
        printf '\n[doc-slice: output truncated at %s bytes of %s. Source: %s (%s lines, %s bytes). Narrow the pattern, or read the file directly if you need all of it.]\n' \
            "$MAX_BYTES" "${#out}" "$FILE" "$total_lines" "$total_bytes"
    else
        printf '%s\n' "$out"
    fi
}

case "$MODE" in

outline)
    # Heading, its line, and how big the section is — enough to choose the next
    # call without reading anything else.
    awk -v file="$FILE" -v tl="$total_lines" -v tb="$total_bytes" '
        /^```/ { infence = !infence }
        !infence && /^#{1,6}[ \t]/ {
            if (prev != "") printf "%-6s %-5s %s\n", prevline, (NR - prevline), prev
            prev = $0; prevline = NR
        }
        { last = NR }
        END {
            if (prev != "") printf "%-6s %-5s %s\n", prevline, (last - prevline + 1), prev
            printf "\n[doc-slice: %s — %s lines, %s bytes. Columns: line, section lines, heading.]\n", file, tl, tb
        }
    ' "$FILE" | emit
    ;;

section)
    acc=""
    for pat in "${PATTERNS[@]}"; do
        # A section runs from its heading to the next heading at the same or a
        # higher level, so `--section 'Pattern C'` gets Pattern C's subsections
        # and stops at Pattern D. Fenced blocks are tracked so a `# comment`
        # inside a shell snippet cannot end the section.
        out="$(awk -v pat="$pat" '
            function level(s,   n) { n = 0; while (substr(s, n+1, 1) == "#") n++; return n }
            /^```/ { infence = !infence; if (intask) print; next }
            !infence && /^#{1,6}[ \t]/ {
                if (intask && level($0) <= mylevel) { intask = 0 }
                if (!intask && tolower($0) ~ tolower(pat)) { intask = 1; mylevel = level($0) }
            }
            intask { print }
        ' "$FILE")"
        [ -n "$out" ] && acc="$acc$out"$'\n'
    done
    [ -n "$acc" ] || die "no heading matches: ${PATTERNS[*]} (try --outline)" 3
    {
        printf '%s' "$acc"
        printf '\n[doc-slice: section(s) of %s — %s lines, %s bytes on disk.]\n' "$FILE" "$total_lines" "$total_bytes"
    } | emit
    ;;

rows)
    # Preserve the table header so the columns mean something. A Markdown table
    # header is the line before the |---|---| separator.
    header="$(awk '/^\|[ \t]*:?-+/ { print prev; print $0; exit } { prev = $0 }' "$FILE" || true)"
    body=""
    for pat in "${PATTERNS[@]}"; do
        m="$(grep -nE "$pat" "$FILE" || true)"
        [ -n "$m" ] && body="$body$m"$'\n'
    done
    body="$(printf '%s' "$body" | sed '/^$/d' | sort -t: -k1,1n -u)"
    [ -n "$body" ] || die "no line matches: ${PATTERNS[*]}" 3
    {
        [ -n "$header" ] && printf '%s\n' "$header"
        printf '%s\n' "$body"
        printf '\n[doc-slice: %s matching row(s) from %s (%s lines, %s bytes). Line numbers prefix each row.]\n' \
            "$(printf '%s\n' "$body" | wc -l | tr -d ' ')" "$FILE" "$total_lines" "$total_bytes"
    } | emit
    ;;

grep)
    out=""
    for pat in "${PATTERNS[@]}"; do
        m="$(grep -nE -C "$CONTEXT" "$pat" "$FILE" || true)"
        [ -n "$m" ] && out="$out$m"$'\n'
    done
    [ -n "$out" ] || die "no line matches: ${PATTERNS[*]}" 3
    {
        printf '%s' "$out"
        printf '\n[doc-slice: %s (%s lines, %s bytes) — context %s.]\n' "$FILE" "$total_lines" "$total_bytes" "$CONTEXT"
    } | emit
    ;;

lines)
    [[ "$RANGE" =~ ^([0-9]+)-([0-9]+)$ ]] || die "--lines wants A-B, got: $RANGE"
    { sed -n "${BASH_REMATCH[1]},${BASH_REMATCH[2]}p" "$FILE"
      printf '\n[doc-slice: lines %s of %s (%s bytes).]\n' "$RANGE" "$total_lines" "$total_bytes"
    } | emit
    ;;

esac
