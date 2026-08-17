#!/usr/bin/env bash
# doc-slice_test.sh — tests for tools/doc-slice.sh.
#
# The behaviour that matters is not "it prints something" but "it prints a small
# something, and it says so when that something is partial." An accessor that
# silently returns a truncated answer is worse than a whole-file read, because
# the caller acts on it believing it is complete.

set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SLICE="$HERE/doc-slice.sh"
[ -x "$SLICE" ] || { echo "FATAL: $SLICE not executable" >&2; exit 2; }

fails=0
assert_eq()  { if [ "$2" = "$3" ]; then echo "ok   - $1"; else echo "FAIL - $1 (want '$2', got '$3')" >&2; fails=$((fails+1)); fi; }
assert_has() { case "$3" in *"$2"*) echo "ok   - $1";; *) echo "FAIL - $1 (missing '$2')" >&2; fails=$((fails+1));; esac; }
assert_lacks(){ case "$3" in *"$2"*) echo "FAIL - $1 (unexpectedly has '$2')" >&2; fails=$((fails+1));; *) echo "ok   - $1";; esac; }

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

# A fixture shaped like the documents this exists for: a plan with numbered
# task sections, a recipe with lettered patterns, and an audit table.
doc="$tmp/fixture.md"
cat > "$doc" <<'EOF'
# Implementation plan

Preamble text.

## Task 1: wire the thing

Do the first thing.

### Files

- `a.go`

## Task 2: unwire the thing

Do the second thing. This section mentions Task 1 in prose.

```sh
# Task 99 is inside a fence and must not start a section
echo hi
```

Still Task 2.

## Task 10: the tenth

Ten.

## Pattern C — the recipe

Follow pattern C.

### Sub-step

Detail of pattern C.

## Pattern D — the other recipe

Follow pattern D.

## Query scope audit

| service | scope | status |
|---|---|---|
| atlas-account | tenant | ok |
| atlas-buddies | global | BROKEN |
| atlas-cashshop | tenant | ok |
| atlas-drops | global | BROKEN |
EOF

# --- outline ---------------------------------------------------------------

out="$("$SLICE" "$doc" --outline)"
assert_has "outline lists a task heading"    "## Task 2: unwire the thing" "$out"
assert_has "outline lists a pattern heading" "## Pattern C — the recipe"   "$out"
assert_has "outline reports the file size"   "bytes"                        "$out"
assert_lacks "outline does not include body text" "Do the first thing"      "$out"

# --- section ---------------------------------------------------------------
#
# This is the "retrieve one plan task without loading the whole plan" case.

out="$("$SLICE" "$doc" --section 'Task 2')"
assert_has  "section returns the requested task"       "Do the second thing" "$out"
assert_lacks "section stops before the next task"      "Ten."                "$out"
assert_lacks "section does not include earlier tasks"  "Do the first thing"  "$out"
assert_has  "section keeps fenced content"             "Task 99 is inside a fence" "$out"
assert_has  "a fence cannot end the section"           "Still Task 2"        "$out"

# `Task 1` must not also match `Task 10`.
out="$("$SLICE" "$doc" --section 'Task 1:')"
assert_has  "exact task heading matches"      "Do the first thing" "$out"
assert_lacks "and not the numerically similar" "Ten."              "$out"

# A section carries its subsections and stops at the next same-level heading.
out="$("$SLICE" "$doc" --section 'Pattern C')"
assert_has  "section includes its subsections" "Detail of pattern C" "$out"
assert_lacks "section stops at the next peer"  "Follow pattern D"    "$out"

# Case-insensitive, and repeatable.
out="$("$SLICE" "$doc" --section 'pattern c' --section 'pattern d')"
assert_has "case-insensitive match"        "Follow pattern C" "$out"
assert_has "two sections in one call (1)"  "Follow pattern C" "$out"
assert_has "two sections in one call (2)"  "Follow pattern D" "$out"

# The point of the whole exercise: the slice is a fraction of the document.
whole="$(wc -c < "$doc")"
part="$("$SLICE" "$doc" --section 'Task 1:' | wc -c)"
if [ "$part" -lt "$((whole / 3))" ]; then
  echo "ok   - one section is under a third of the document ($part of $whole bytes)"
else
  echo "FAIL - section is not meaningfully smaller ($part of $whole)" >&2; fails=$((fails+1))
fi

# --- rows -------------------------------------------------------------------

out="$("$SLICE" "$doc" --rows 'atlas-buddies' --rows 'atlas-drops')"
assert_has  "rows keeps the table header"   "| service | scope | status |" "$out"
assert_has  "rows keeps the separator"      "|---|---|---|"                "$out"
assert_has  "rows returns the first match"  "atlas-buddies"                "$out"
assert_has  "rows returns the second match" "atlas-drops"                  "$out"
assert_lacks "rows omits unrequested rows"  "atlas-account"                "$out"

# --- grep -------------------------------------------------------------------

out="$("$SLICE" "$doc" --grep 'BROKEN' --context 0)"
assert_has  "grep finds both broken rows (1)" "atlas-buddies" "$out"
assert_has  "grep finds both broken rows (2)" "atlas-drops"   "$out"
assert_lacks "grep with context 0 has no neighbours" "atlas-cashshop" "$out"

out="$("$SLICE" "$doc" --grep 'atlas-buddies' --context 1)"
assert_has "grep context 1 pulls the neighbour" "atlas-account" "$out"

# --- lines ------------------------------------------------------------------

out="$("$SLICE" "$doc" --lines 1-3)"
assert_has  "lines returns the range"        "# Implementation plan" "$out"
assert_lacks "lines stops at the range end"  "Do the first thing"    "$out"

# --- no match: say so, do not dump the file --------------------------------

out="$("$SLICE" "$doc" --section 'nonexistent heading' 2>&1)"; rc=$?
assert_eq   "no matching section exits 3"       "3" "$rc"
assert_lacks "no matching section prints no body" "Do the first thing" "$out"
assert_has  "no matching section suggests --outline" "--outline" "$out"

out="$("$SLICE" "$doc" --rows 'zzz-not-a-service' 2>&1)"; rc=$?
assert_eq "no matching row exits 3" "3" "$rc"

# --- truncation is announced ------------------------------------------------

out="$("$SLICE" "$doc" --lines 1-999 --max-bytes 100)"
assert_has "truncation is announced"        "truncated" "$out"
assert_has "truncation names the source"    "fixture.md" "$out"
assert_has "truncation offers escalation"   "read the file directly" "$out"

# The cap is in bytes, not characters. A document of multi-byte characters
# would truncate somewhere other than the flag says if this used ${#var} /
# ${var:0:n}, which count characters under a UTF-8 locale.
utf8="$tmp/utf8.md"
{ echo "# Heading"
  for i in $(seq 1 40); do echo "row $i — em-dash … ellipsis … ünïcödé"; done; } > "$utf8"
out="$(LC_ALL=en_US.UTF-8 "$SLICE" "$utf8" --lines 1-999 --max-bytes 200)"
body="${out%%$'\n\n[doc-slice:'*}"
bytes="$(printf '%s' "$body" | wc -c | tr -d ' ')"
if [ "$bytes" -le 200 ]; then
  echo "ok   - multi-byte truncation caps at $bytes bytes, not characters"
else
  echo "FAIL - multi-byte truncation emitted $bytes bytes for --max-bytes 200" >&2
  fails=$((fails+1))
fi

# --- usage errors -----------------------------------------------------------

"$SLICE" "$tmp/nope.md" --outline >/dev/null 2>&1; rc=$?
assert_eq "missing file exits 2" "2" "$rc"
"$SLICE" "$doc" >/dev/null 2>&1; rc=$?
assert_eq "no mode exits 2" "2" "$rc"

# --- works on a non-markdown offload ---------------------------------------
#
# The `tool-results/*.txt` case from the audit: a large plain-text result that
# should be grepped from disk rather than read whole.

log="$tmp/offload.txt"
{ for i in $(seq 1 500); do echo "line $i noise"; done
  echo "PANIC: nil map write at foo.go:42"
  for i in $(seq 1 500); do echo "more noise $i"; done; } > "$log"
out="$("$SLICE" "$log" --grep 'PANIC' --context 1)"
assert_has "offload: finds the needle" "nil map write at foo.go:42" "$out"
bytes="$(printf '%s' "$out" | wc -c)"
if [ "$bytes" -lt 500 ]; then
  echo "ok   - offload slice is small ($bytes bytes vs $(wc -c < "$log") on disk)"
else
  echo "FAIL - offload slice is $bytes bytes" >&2; fails=$((fails+1))
fi

echo
if [ "$fails" -eq 0 ]; then echo "doc-slice_test.sh: all assertions passed"
else echo "doc-slice_test.sh: $fails failure(s)" >&2; fi
[ "$fails" -eq 0 ]
