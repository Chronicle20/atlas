#!/usr/bin/env bash
# plan-context_test.sh — hermetic tests for tools/plan-context.sh.
#
# plan-context.sh exists to replace ~39 discovery turns with one call, so the
# assertions are about the three properties that make that substitution safe:
#
#   1. It passes task-resolve.sh's exit codes through, like task-facts.sh does.
#   2. Its EXISTING/UNRESOLVED split is correct — that split is what a planner
#      turns into `### Files` entries and `new file` markers, and it is what
#      plan-lint F1 later checks. A path misfiled here becomes a lint failure
#      at the exact moment this script exists to avoid.
#   3. It stays small. A survey that grows past a few KB stops being cheaper
#      than the turns it replaces.
#
# The `set -e` regression tests are deliberate: the first working draft of this
# script silently dropped its whole --symbols section because a `[ -n "$x" ] &&
# echo` at the tail of a loop returned 1 and errexit killed the script with
# status 0 already printed. Both a design with no prd.md and a design whose
# last surveyed directory has no siblings reproduce that class of bug.

set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
for s in plan-context.sh task-resolve.sh; do
  [ -x "$HERE/$s" ] || { echo "FATAL: $HERE/$s not executable" >&2; exit 2; }
done

fails=0
assert_eq()  { if [ "$2" = "$3" ]; then echo "ok   - $1"; else echo "FAIL - $1 (want '$2', got '$3')" >&2; fails=$((fails+1)); fi; }
assert_has() { case "$3" in *"$2"*) echo "ok   - $1";; *) echo "FAIL - $1 (missing '$2')" >&2; fails=$((fails+1));; esac; }
assert_not() { case "$3" in *"$2"*) echo "FAIL - $1 (unexpected '$2')" >&2; fails=$((fails+1));; *) echo "ok   - $1";; esac; }

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
repo="$tmp/repo"
mkdir -p "$repo/tools"
cp "$HERE/plan-context.sh" "$HERE/task-resolve.sh" "$repo/tools/"
git -C "$repo" init -q -b main
git -C "$repo" config user.email t@t.t
git -C "$repo" config user.name t
git -C "$repo" config commit.gpgsign false

# A fixture task whose design names: one real .go with a test, one real .go
# without, one real directory, and one path that does not exist.
mkdir -p "$repo/docs/tasks/task-001-fixture"
mkdir -p "$repo/services/atlas-demo/atlas.com/demo/widget"
cat > "$repo/services/atlas-demo/go.mod" <<'EOF'
module atlas-demo

go 1.24
EOF
cat > "$repo/services/atlas-demo/atlas.com/demo/widget/processor.go" <<'EOF'
package widget

type Processor struct{}

func NewProcessor() Processor { return Processor{} }

func (p Processor) Build() error { return nil }

func unexportedHelper() {}
EOF
touch "$repo/services/atlas-demo/atlas.com/demo/widget/processor_test.go"
cat > "$repo/services/atlas-demo/atlas.com/demo/widget/model.go" <<'EOF'
package widget

type Model struct{}
EOF
cat > "$repo/docs/tasks/task-001-fixture/design.md" <<'EOF'
# Design

Touches `services/atlas-demo/atlas.com/demo/widget/processor.go` and
`services/atlas-demo/atlas.com/demo/widget/model.go`. A new file lands at
services/atlas-demo/atlas.com/demo/widget/emitter.go. See also the directory
services/atlas-demo/atlas.com/demo/widget for context, and a wrapped path like
libs/atlas-
that prose broke across a line.
EOF
git -C "$repo" add -A >/dev/null
git -C "$repo" commit -qm init

run() { ( cd "$repo" && ./tools/plan-context.sh "$@" 2>&1 ); }

# --- resolution passthrough ------------------------------------------------
( cd "$repo" && ./tools/plan-context.sh nosuchtask >/dev/null 2>&1 )
assert_eq "unknown task exits 3 (task-resolve passthrough)" "3" "$?"

( cd "$repo" && ./tools/plan-context.sh >/dev/null 2>&1 )
assert_eq "no identifier exits 2" "2" "$?"

( cd "$repo" && ./tools/plan-context.sh 001 --siblings abc >/dev/null 2>&1 )
assert_eq "non-numeric --siblings exits 2" "2" "$?"

out="$(run 001)"
assert_eq "survey of a resolvable task exits 0" "0" "$?"

# --- the EXISTING / UNRESOLVED split ---------------------------------------
assert_has "existing .go is surveyed" "widget/processor.go" "$out"
assert_has "existing file reports its test" "[has test:" "$out"
assert_has "test-less .go is flagged" "NO test file" "$out"
assert_has "missing path is unresolved" "emitter.go" "$out"
assert_has "unresolved section is titled for the planner" "mark 'new file'" "$out"
assert_not "line-wrap artefact is not reported as a missing file" "- libs/atlas-" "$out"

# emitter.go must be UNRESOLVED, not EXISTING. Check it appears after the
# unresolved heading rather than merely somewhere in the output.
# Slice out just that section — the Siblings section further down legitimately
# names processor.go, so an open-ended tail would fail on correct output.
section="$(printf '%s\n' "$out" | awk '/^## Unresolved paths/{f=1;next} /^## /{f=0} f')"
assert_has "emitter.go is filed under unresolved" "emitter.go" "$section"
assert_not "processor.go is not filed under unresolved" "processor.go" "$section"

# --- module roots ----------------------------------------------------------
assert_has "module root is derived by upward walk" "services/atlas-demo  (atlas-demo)" "$out"

# --- siblings --------------------------------------------------------------
assert_has "siblings are offered as patterns to copy" "model.go" "$out"
out0="$(run 001 --siblings 0)"
assert_not "--siblings 0 disables the section" "Patterns-to-copy" "$out0"

# --- symbols ---------------------------------------------------------------
assert_not "symbols are off by default" "Top-level declarations" "$out"
outs="$(run 001 --symbols)"
assert_has "--symbols emits the section" "Top-level declarations" "$outs"
assert_has "--symbols lists exported funcs" "func NewProcessor" "$outs"
assert_has "--symbols lists exported methods" "func (p Processor) Build" "$outs"
assert_not "--symbols omits unexported helpers" "unexportedHelper" "$outs"

# --- set -e regressions ----------------------------------------------------
# A task with no prd.md must still survey. The original draft aborted here.
assert_has "design-only task still surveys" "widget/processor.go" "$out"

# The last surveyed directory having no listable siblings must not truncate the
# run: --symbols comes after the siblings loop, so a silent abort shows up as a
# missing symbols section with exit status 0.
mkdir -p "$repo/docs/tasks/task-002-nosib/"
mkdir -p "$repo/services/atlas-empty"
cat > "$repo/docs/tasks/task-002-nosib/design.md" <<'EOF'
Touches services/atlas-demo/atlas.com/demo/widget/processor.go and the
sibling-less directory services/atlas-empty.
EOF
out2="$(run 002 --symbols)"
assert_eq "sibling-less dir does not abort the run" "0" "$?"
assert_has "sections after a sibling-less dir still print" "Top-level declarations" "$out2"

# --- no design.md ----------------------------------------------------------
mkdir -p "$repo/docs/tasks/task-003-nodesign"
( cd "$repo" && ./tools/plan-context.sh 003 >/dev/null 2>&1 )
assert_eq "task without design.md exits 5" "5" "$?"

# --- --from override -------------------------------------------------------
printf 'services/atlas-demo/atlas.com/demo/widget/model.go\n' > "$tmp/list.txt"
outf="$(run 001 --from "$tmp/list.txt")"
assert_has "--from scans the given file" "widget/model.go" "$outf"
assert_not "--from ignores design.md" "emitter.go" "$outf"

# --- size ------------------------------------------------------------------
# The whole point is to be cheaper than the turns it replaces. This fixture is
# tiny; the real guard is that the script never dumps file CONTENT, only facts
# about files, so output scales with path count and not with repo size.
bytes="$(printf '%s' "$out" | wc -c | tr -d ' ')"
if [ "$bytes" -lt 4096 ]; then
  echo "ok   - fixture survey stays under 4 KB ($bytes bytes)"
else
  echo "FAIL - fixture survey is $bytes bytes, expected < 4096" >&2
  fails=$((fails+1))
fi

echo
if [ "$fails" -ne 0 ]; then
  echo "plan-context_test.sh: $fails assertion(s) failed." >&2
  exit 1
fi
echo "plan-context_test.sh: all assertions passed."
