#!/usr/bin/env bash
# verify_test.sh — tests for `tools/verify.sh --facts`.
#
# The claim under test is the one that makes --facts worth trusting:
#
#   --facts does not re-implement the selection logic; it IS the selection
#   logic, with the work removed.
#
# Two kinds of assertion enforce that. The behavioural ones run --facts and a
# real gate over the same change set and require the selected/skipped label sets
# to be identical. The structural ones read verify.sh and require that no gate
# label can be produced anywhere except inside step() — which is what makes the
# behavioural agreement hold for change sets this test never exercises.
#
# RECURSION. verify.sh runs every changed tools/*_test.sh, and this file is one
# of them, so a real gate invoked from here would run this test again, which
# would invoke the gate again. ATLAS_VERIFY_TEST_INNER breaks that at depth one:
# the inner copy runs the structural assertions (which need no subprocess) and
# stops. Removing the guard reintroduces an unbounded fork loop, so do not.

set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VERIFY="$HERE/verify.sh"
[ -x "$VERIFY" ] || { echo "FATAL: $VERIFY not executable" >&2; exit 2; }

fails=0
assert_eq() {
  if [ "$2" = "$3" ]; then echo "ok   - $1"
  else echo "FAIL - $1"; echo "        want: $2"; echo "        got:  $3"; fails=$((fails + 1)); fi
}
assert_true() {
  if [ "$2" = "true" ] || [ "$2" = "0" ]; then echo "ok   - $1"
  else echo "FAIL - $1 (got '$2')"; fails=$((fails + 1)); fi
}

# --- structural: gates come only from step() -------------------------------
#
# The anti-drift assertions. If a future edit prints a gate name in the fact
# block directly, --facts starts answering from a second source of truth and
# these fail. They run in both the outer and the inner invocation.

structural() {
  local appends in_step
  appends="$(grep -c 'SELECTED+=' "$VERIFY")"
  assert_eq "SELECTED is appended in exactly one place" "1" "$appends"

  in_step="$(awk '/^step\(\) \{/,/^\}/' "$VERIFY" | grep -c 'SELECTED+=')"
  assert_eq "that one place is inside step()" "1" "$in_step"

  # The pipefail/grep hazard: grep exits on the first match, the writing
  # printf takes SIGPIPE, and pipefail reports 141 — so a MATCH reads as "no
  # match" and a path-gated guard silently skips while the gate exits 0. It
  # only bites past the 64KB pipe buffer, i.e. on large sweep branches.
  # Comments are allowed to name the hazard; code is not allowed to use it.
  assert_eq "no 'grep -q' in the change-detection path" "0" \
    "$(grep -v '^[[:space:]]*#' "$VERIFY" | grep -c 'grep -q' || true)"
}

if [ "${ATLAS_VERIFY_TEST_INNER:-0}" = "1" ]; then
  echo "verify_test.sh: inner invocation (dispatched by verify.sh) — structural assertions only"
  structural
  echo
  [ "$fails" -eq 0 ] || { echo "verify_test.sh: $fails failure(s)" >&2; exit 1; }
  exit 0
fi

structural
export ATLAS_VERIFY_TEST_INNER=1

# --- helpers ---------------------------------------------------------------
#
# Gate labels a real run reports, normalised: ✓ passed, ✗ failed, − skipped.

real_selected() {
  "$VERIFY" "$@" 2>/dev/null \
    | sed 's/\x1b\[[0-9;]*m//g' \
    | sed -n 's/^  [✓✗] *//p' \
    | sed 's/^ *//; s/ *$//' | sort
}
real_skipped_count() {
  "$VERIFY" "$@" 2>/dev/null \
    | sed 's/\x1b\[[0-9;]*m//g' \
    | grep -c '^  − ' || true
}
facts_selected() { "$VERIFY" --facts "$@" 2>/dev/null | sed -n 's/^gate=//p' | sort; }
facts_key() { local k="$1"; shift; "$VERIFY" --facts "$@" 2>/dev/null | sed -n "s/^${k}=//p"; }

# --- probes ----------------------------------------------------------------
#
# Every real run below is scoped with `--base HEAD`, so the change set is only
# the working tree. An unscoped run would be the whole-branch diff — minutes
# of lint and static analysis, in a test whose subject is selection rather
# than execution. (Do not start a comment line with the word that shellcheck
# reads as a directive; it parses one and fails with SC1072/SC1073.)
#
# Fixed names, not $$: a crashed run leaves exactly one stale file, which the
# next run removes, instead of one per attempt.
probe_suite="$HERE/zz-verify-probe_test.sh"
probe_deploy="$HERE/../deploy/.zz-verify-probe.tmp"
probe_ban_dir="$HERE/../services/atlas-ban/zz-verify-probe"
probe_account_dir="$HERE/../services/atlas-account/zz-verify-probe"
probe_bake_ban="$probe_ban_dir/go.mod"
probe_bake_account="$probe_account_dir/go.mod"
cleanup() {
  rm -f "$probe_suite" "$probe_deploy" "$probe_bake_ban" "$probe_bake_account" \
    "$HERE/zz-verify-jobs0.err"
  rmdir "$probe_ban_dir" "$probe_account_dir" 2>/dev/null || true
}
trap cleanup EXIT
cleanup
printf '#!/usr/bin/env bash\nexit 0\n' > "$probe_suite"
chmod +x "$probe_suite"

# --- agreement: --facts vs a real run -------------------------------------
#
# Compared as sets. The one transformation --facts applies is collapsing the
# per-module `go build/vet …` steps into a single line, so those are dropped
# from both sides and asserted separately via modules_selected.

drop_module_gates() { grep -v '^go build/vet' || true; }

for args in "--quick --base HEAD" "--quick --base HEAD --no-ui"; do
  # shellcheck disable=SC2086
  want="$(real_selected $args | drop_module_gates)"
  # shellcheck disable=SC2086
  got="$(facts_selected $args | drop_module_gates)"
  assert_eq "selected gates agree ($args)" "$want" "$got"

  # shellcheck disable=SC2086
  want_n="$(real_skipped_count $args)"
  # shellcheck disable=SC2086
  got_n="$(facts_key gates_skipped $args)"
  assert_eq "skipped gate count agrees ($args)" "$want_n" "$got_n"
done

# The untracked probe suite must be selected, by both, by name.
#
# Captured to a variable first, deliberately. Piping a helper straight into
# grep would put the helper's exit status in the pipeline, and under pipefail a
# real gate that reports a FAILED check makes the whole expression non-zero —
# so a present label would read as absent. Selection is the claim here, not the
# gate's verdict.
probe_name="$(basename "$probe_suite")"
facts_out="$(facts_selected --quick --base HEAD)"
real_out="$(real_selected --quick --base HEAD)"
assert_true "--facts selects an untracked tools/*_test.sh" \
  "$(printf '%s\n' "$facts_out" | grep -x "$probe_name" >/dev/null && echo true)"
assert_true "the real run selects it too" \
  "$(printf '%s\n' "$real_out" | grep -x "$probe_name" >/dev/null && echo true)"
assert_true "--facts lists it under guard_suites" \
  "$(facts_key guard_suites --quick --base HEAD | grep "$(basename "$probe_suite")" >/dev/null && echo true)"

# A deploy/ path selects the deploy gates. Only --facts is exercised here: a
# real gen-lb-ports.sh --check walks the whole tree, and selection is the claim.
: > "$probe_deploy"
assert_true "a deploy/ change selects the LB port gate" \
  "$(facts_selected --quick --base HEAD | grep 'LB port drift' >/dev/null && echo true)"
rm -f "$probe_deploy"
assert_true "removing it deselects the LB port gate" \
  "$(facts_selected --quick --base HEAD | grep 'LB port drift' >/dev/null && echo false || echo true)"

# --- module selection agrees ----------------------------------------------

want_mods="$("$VERIFY" --quick --base HEAD 2>/dev/null \
  | sed 's/\x1b\[[0-9;]*m//g' | sed -n 's/^verify.sh: \([0-9]*\) changed Go module(s)$/\1/p')"
want_mods="${want_mods:-0}"
got_mods="$(facts_key modules_selected --quick --base HEAD)"
assert_eq "module count agrees" "$want_mods" "$got_mods"

# --- bake selection: one solve, not one per target --------------------------
#
# bake_targets() matches a changed path against each service's `path` from
# .github/config/services.json, requiring the path to end in go.mod. CHANGED
# includes untracked files, so two untracked go.mods select exactly two
# targets — and must produce exactly one bake gate, not two.
mkdir -p "$probe_ban_dir" "$probe_account_dir"
: > "$probe_bake_ban"
: > "$probe_bake_account"

assert_eq "two changed go.mods select two bake targets" "atlas-account,atlas-ban" \
  "$(facts_key bake_targets --base HEAD)"

bake_gate_lines="$(facts_selected --base HEAD | grep -c '^docker buildx bake' || true)"
assert_eq "two bake targets produce exactly one bake gate" "1" "$bake_gate_lines"

bake_gate_name="$(facts_selected --base HEAD | grep '^docker buildx bake')"
assert_eq "the gate names the target count" "docker buildx bake (2 target(s))" "$bake_gate_name"

rm -f "$probe_bake_ban" "$probe_bake_account"
rmdir "$probe_ban_dir" "$probe_account_dir" 2>/dev/null || true

no_bake_lines="$(facts_selected --base HEAD | grep -c '^docker buildx bake' || true)"
assert_eq "no probes, no bake gate" "0" "$no_bake_lines"

assert_eq "no per-target bake loop remains" "0" \
  "$(grep -c 'for t in .\{0,4\}TARGETS' "$VERIFY")"

# --- pool: GO_JOBS changes wall time, not what is reported -----------------
#
# The claim under test: a bounded worker pool must be observably identical to
# the serial loop it replaces — same gate labels, same module count, same
# ordering guarantees — for every job count. Only wall time may differ.

labels_j1="$(ATLAS_VERIFY_GO_JOBS=1 real_selected --quick --base HEAD)"
labels_j4="$(ATLAS_VERIFY_GO_JOBS=4 real_selected --quick --base HEAD)"
assert_eq "gate labels are job-count invariant (1 vs 4)" "$labels_j1" "$labels_j4"

mods_j1="$(ATLAS_VERIFY_GO_JOBS=1 facts_key modules_selected --quick --base HEAD)"
mods_j4="$(ATLAS_VERIFY_GO_JOBS=4 facts_key modules_selected --quick --base HEAD)"
assert_eq "module count is job-count invariant (1 vs 4)" "$mods_j1" "$mods_j4"

probe_jobs0_err="$HERE/zz-verify-jobs0.err"
ATLAS_VERIFY_GO_JOBS=0 "$VERIFY" --quick --base HEAD >/dev/null 2>"$probe_jobs0_err"
job0_rc=$?
assert_eq "GO_JOBS=0 is rejected (exit 2)" "2" "$job0_rc"
assert_true "GO_JOBS=0 rejection names ATLAS_VERIFY_GO_JOBS" \
  "$(grep 'ATLAS_VERIFY_GO_JOBS' "$probe_jobs0_err" >/dev/null && echo true)"
rm -f "$probe_jobs0_err"

ATLAS_VERIFY_GO_JOBS=x "$VERIFY" --quick --base HEAD >/dev/null 2>/dev/null
jobx_rc=$?
assert_eq "a non-numeric GO_JOBS is rejected (exit 2)" "2" "$jobx_rc"

assert_true "the Go layer's log dir is created under TMPDIR and cleaned up on exit" \
  "$(grep -F 'mktemp -d "${TMPDIR:-/tmp}/verify-go.XXXXXX"' "$VERIFY" >/dev/null \
     && grep -F "trap 'rm -rf \"\$GO_LOG_DIR\"' EXIT" "$VERIFY" >/dev/null \
     && echo true)"

# --- --facts runs nothing --------------------------------------------------
#
# `--all` selects every gate including the module builds. If any of it actually
# executed this would take minutes; a fast exit is the evidence that step() was
# neutered rather than merely quiet.
start="$(date +%s)"
out="$("$VERIFY" --facts --all --quick 2>/dev/null)"
rc=$?
elapsed=$(( $(date +%s) - start ))
assert_true "--facts --all exits 0" "$rc"
if [ "$elapsed" -lt 60 ]; then
  echo "ok   - --facts --all runs no check (${elapsed}s)"
else
  echo "FAIL - --facts --all took ${elapsed}s; a check appears to have executed"; fails=$((fails + 1))
fi
assert_true "--facts --all reports the fan-out reason" \
  "$(printf '%s\n' "$out" | grep '^fanout_reason=--all' >/dev/null && echo true)"

# --- output contract -------------------------------------------------------

out="$("$VERIFY" --facts --quick --base HEAD 2>/dev/null)"
for k in base changed_paths changed_services changed_libs go_changed ui_changed \
         fanout_reason modules_selected guard_suites gates_selected gates_skipped; do
  assert_true "fact block carries '$k'" \
    "$(printf '%s\n' "$out" | grep "^${k}=" >/dev/null && echo true)"
done
assert_true "fact block is key=value lines only" \
  "$(printf '%s\n' "$out" | grep -v '^[a-z_]*=' >/dev/null && echo false || echo true)"

# stdout must stay clean: informational chatter goes to stderr under --facts,
# or a caller parsing key=value gets a surprise line.
assert_true "no informational chatter on stdout" \
  "$(printf '%s\n' "$out" | grep '^verify\.sh:' >/dev/null && echo false || echo true)"

echo
if [ "$fails" -eq 0 ]; then echo "verify_test.sh: all assertions passed"; else echo "verify_test.sh: $fails failure(s)" >&2; fi
[ "$fails" -eq 0 ]
