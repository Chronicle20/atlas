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
# Per-process, not fixed: two concurrent runs of this suite — or one run
# racing a live `verify.sh` gate — must not collide on the same paths. Each
# leaf name carries the PID of this run; the directory each probe sits in is
# unchanged, because several assertions depend on the probes' RELATIVE
# locations (inside services/, deploy/, tools/) and cannot simply move to a
# shared temp dir.
probe_tag="$$"
probe_suite="$HERE/zz-verify-probe-${probe_tag}_test.sh"
probe_deploy="$HERE/../deploy/.zz-verify-probe-${probe_tag}.tmp"
probe_ban_dir="$HERE/../services/atlas-ban/zz-verify-probe-${probe_tag}"
probe_account_dir="$HERE/../services/atlas-account/zz-verify-probe-${probe_tag}"
probe_bake_ban="$probe_ban_dir/go.mod"
probe_bake_account="$probe_account_dir/go.mod"
probe_broken_dir="$HERE/../services/zz-verify-probe-broken-${probe_tag}"
probe_jobs0_err="$HERE/zz-verify-jobs0-${probe_tag}.err"
# Task 9 (Layer 5): a real libs/ file, per-process like every other probe
# above — two concurrent runs of this file must not collide on the same
# libs/ path.
probe_libs="$HERE/../libs/atlas-tenant/zz-verify-probe-${probe_tag}.go"
# Two assertions below don't read this run's OWN probe by name — they read a
# category `--facts` derives from shared, not-per-process, state: "does any
# deploy/ path exist in the changed set" and "which SERVICES have a changed
# go.mod," which bake_targets() dedupes by service name. A per-process leaf
# name can't isolate those; a concurrent run's own probe under the same
# category is indistinguishable from this run's, so the add/assert/remove/
# assert-gone sequence around them takes an exclusive lock instead — the same
# tool (flock) tools/tidy-all-go.sh already uses for the one other genuinely
# shared, not-per-process, mutable resource in this suite.
shared_state_lock="${TMPDIR:-/tmp}/atlas-verify-test-shared-state.lock"
exec 8>"$shared_state_lock"
# The broken-module run below is a real `verify.sh --quick` invocation, so the
# Go toolchain resolves the workspace and appends hash lines to go.work.sum.
# That file must come back byte-identical whether the assertions pass, fail,
# or the run is interrupted by SIGINT/SIGTERM, or every later gate
# misclassifies it as a shared-lib change and rebuilds all modules (see the
# comment at the broken-module block). Snapshot/restore is folded into the
# same cleanup trap as the rest of the probe cleanup, and the trap fires on
# EXIT, INT, and TERM so a signal-delivered interruption (e.g. from a
# `timeout`-bounded foreground child) still restores it.
gowork_sum="$HERE/../go.work.sum"
gowork_sum_backup="$HERE/zz-verify-probe-broken-${probe_tag}.go.work.sum.bak"
gowork_sum_backup_absent="$HERE/zz-verify-probe-broken-${probe_tag}.go.work.sum.absent"
cleanup() {
  rm -f "$probe_suite" "$probe_deploy" "$probe_bake_ban" "$probe_bake_account" \
    "$probe_jobs0_err" "$probe_libs"
  rmdir "$probe_ban_dir" "$probe_account_dir" 2>/dev/null || true
  rm -rf "$probe_broken_dir"
  if [ -f "$gowork_sum_backup" ]; then
    mv -f "$gowork_sum_backup" "$gowork_sum"
  elif [ -f "$gowork_sum_backup_absent" ]; then
    rm -f "$gowork_sum" "$gowork_sum_backup_absent"
  fi
}
trap cleanup EXIT INT TERM
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
flock 8
: > "$probe_deploy"
assert_true "a deploy/ change selects the LB port gate" \
  "$(facts_selected --quick --base HEAD | grep 'LB port drift' >/dev/null && echo true)"
rm -f "$probe_deploy"
assert_true "removing it deselects the LB port gate" \
  "$(facts_selected --quick --base HEAD | grep 'LB port drift' >/dev/null && echo false || echo true)"
flock -u 8

# --- module selection agrees ----------------------------------------------

want_mods="$("$VERIFY" --quick --base HEAD 2>/dev/null \
  | sed 's/\x1b\[[0-9;]*m//g' | sed -n 's/^verify.sh: \([0-9]*\) changed Go module(s)$/\1/p')"
want_mods="${want_mods:-0}"
got_mods="$(facts_key modules_selected --quick --base HEAD)"
assert_eq "module count agrees" "$want_mods" "$got_mods"

# --- libs/ fan-out narrows to the reverse-dependency closure (Task 9) ------
#
# The real repo graph, not a synthetic one: libs/atlas-tenant has dozens of
# consumers, which is exactly what makes the old "any libs/ change reaches
# every module" behaviour so expensive.

total_modules="$(find "$HERE/../services" "$HERE/../libs" -name go.mod -not -path '*/node_modules/*' \
  | wc -l | tr -d ' ')"

printf 'package atlastenant\n' > "$probe_libs"

closure_selected="$(facts_key modules_selected --quick --base HEAD)"
assert_true "a real libs/ change no longer selects every module" \
  "$([ "$closure_selected" -lt "$total_modules" ] && echo true)"

closure_reason="$(facts_key fanout_reason --quick --base HEAD)"
assert_true "the fan-out reason names the closure" \
  "$(printf '%s\n' "$closure_reason" | grep -qE '^shared-lib-closure:libs/atlas-tenant' \
     && printf '%s\n' "$closure_reason" | grep -qE '\([0-9]+ consumers\)$' \
     && echo true)"

all_selected="$(ATLAS_LIBS_FANOUT=all facts_key modules_selected --quick --base HEAD)"
assert_eq "the escape hatch restores the old behaviour (module count)" \
  "$total_modules" "$all_selected"
all_reason="$(ATLAS_LIBS_FANOUT=all facts_key fanout_reason --quick --base HEAD)"
assert_true "the escape hatch's fan-out reason carries the shared-lib: prefix" \
  "$(printf '%s\n' "$all_reason" | grep -qE '^shared-lib:' && echo true)"

rm -f "$probe_libs"
assert_eq "no libs change, no fan-out" "none" "$(facts_key fanout_reason --quick --base HEAD)"

# go.work.sum is a checksum artifact of resolving the workspace — an ordinary
# local `go build`/`go mod tidy` dirties it with no require-graph edge
# actually changing. The old `^(go\.work|libs/)` match was unanchored at its
# end, so it also matched go.work.sum and sent every such run through
# all_modules — a second, independent path to the full fan-out, and the
# common one in daily use. go.work.sum is shared, not per-process, state (it
# is THE workspace sum file, not a probe), so this takes the same lock this
# suite already uses for go.work.sum below.
flock 8
gowork_dirty_backup="$HERE/zz-verify-probe-gowork-dirty-${probe_tag}.bak"
if [ -f "$gowork_sum" ]; then
  cp -p "$gowork_sum" "$gowork_dirty_backup"
else
  : > "$gowork_dirty_backup.absent"
fi
printf '// zz-verify-probe-%s dirty line\n' "$probe_tag" >> "$gowork_sum"

dirty_reason="$(facts_key fanout_reason --quick --base HEAD)"
dirty_selected="$(facts_key modules_selected --quick --base HEAD)"

if [ -f "$gowork_dirty_backup" ]; then
  mv -f "$gowork_dirty_backup" "$gowork_sum"
else
  rm -f "$gowork_sum" "$gowork_dirty_backup.absent"
fi
flock -u 8

assert_eq "a dirty go.work.sum alone does not fan out" "none" "$dirty_reason"
assert_eq "a dirty go.work.sum alone selects zero modules" "0" "$dirty_selected"

# --- structural: go.work still fans out to everything (Task 9) -------------

changed_modules_body="$(awk '/^changed_modules\(\) \{/,/^\}$/' "$VERIFY")"
assert_true "the go.work branch of changed_modules still calls all_modules" \
  "$(printf '%s\n' "$changed_modules_body" \
     | awk '/gowork_changed/,0' | grep -F 'all_modules' >/dev/null && echo true)"

# --- preflight: capacity gate (Task 7) --------------------------------------
#
# The starved-run case is the important one: it must prove the gate FAILS
# rather than proceeding, which is the whole point of the preflight.

assert_true "preflight is selected on a full run" \
  "$(facts_selected --base HEAD --no-ui | grep -x 'preflight (capacity)' >/dev/null && echo true)"
assert_true "preflight is NOT selected under --quick" \
  "$(facts_selected --quick --base HEAD | grep -x 'preflight (capacity)' >/dev/null && echo false || echo true)"

starved_out="$(ATLAS_MIN_FREE_MB=99999999 "$VERIFY" --base HEAD --no-ui --no-docker 2>&1)"
starved_rc=$?
sane_out="$(TMPDIR=/tmp ATLAS_MIN_FREE_MB=1 ATLAS_MIN_TMP_MB=1 "$VERIFY" --base HEAD --no-ui --no-docker 2>&1)"
sane_rc=$?

assert_true "a starved run fails rather than proceeding (non-zero exit)" \
  "$([ "$starved_rc" -ne 0 ] && echo true)"
assert_true "the same command without the override exits 0 (preflight is not permanently on)" \
  "$([ "$sane_rc" -eq 0 ] && echo true)"
assert_true "the starved run's summary reports a preflight failure" \
  "$(printf '%s\n' "$starved_out" | grep '✗' | grep -i preflight >/dev/null && echo true)"
assert_true "the preflight message names the free-RAM shortfall" \
  "$(printf '%s\n' "$starved_out" | grep 'free RAM' | grep -E '[0-9]+ MiB' >/dev/null && echo true)"

tuning_out="$(TMPDIR=/tmp ATLAS_MIN_FREE_MB=1 ATLAS_MIN_TMP_MB=1 "$VERIFY" --base HEAD --no-docker --no-ui 2>&1)"
assert_true "an un-tuned host is reported, not assumed (names Host tuning)" \
  "$(printf '%s\n' "$tuning_out" | grep 'Host tuning' >/dev/null && echo true)"
assert_true "the un-tuned-host report points at docs/verification.md" \
  "$(printf '%s\n' "$tuning_out" | grep 'docs/verification.md' >/dev/null && echo true)"

# --- per-slot budgets: applied in go_layer, and only there ------------------

go_layer_body="$(awk '/^go_layer\(\) \{/,/^\}$/' "$VERIFY")"
assert_true "go_layer exports GOMAXPROCS" \
  "$(printf '%s\n' "$go_layer_body" | grep -F 'export GOMAXPROCS' >/dev/null && echo true)"
assert_true "go build and go test take distinct -p budgets" \
  "$(printf '%s\n' "$go_layer_body" | grep -F 'go build -p "${ATLAS_GO_P' >/dev/null \
     && printf '%s\n' "$go_layer_body" | grep -F 'go test -p "${ATLAS_GO_TEST_P' >/dev/null \
     && echo true)"
assert_true "go vet takes no -p (not a valid go vet flag)" \
  "$(printf '%s\n' "$go_layer_body" | grep -F 'go vet -p' >/dev/null && echo false || echo true)"

# --- heavy phases are slotted; cheap phases are not --------------------------
#
# Sectioned by the file's own "# ---" banner comments: the docker layer
# (bake) and the go-modules layer (the pool) must each hold exactly one slot
# reference; nothing from guards through the facts printer may hold one.

docker_section="$(sed -n '/^# ----*.*docker$/,/^# ----*.*guards$/p' "$VERIFY")"
go_modules_section="$(sed -n '/^# ----*.*go modules$/,/^# ----*.*docker$/p' "$VERIFY")"
guards_through_facts="$(sed -n '/^# ----*.*guards$/,/^# ----*.*summary$/p' "$VERIFY")"

assert_eq "the bake (docker layer) holds exactly one slot reference" "1" \
  "$(printf '%s\n' "$docker_section" | grep -cE 'acquire_build_slot|with-build-slot\.sh')"
assert_eq "the Go pool (go-modules layer) holds exactly one slot reference" "1" \
  "$(printf '%s\n' "$go_modules_section" | grep -c 'acquire_build_slot')"
assert_eq "acquire_build_slot/with-build-slot.sh appears exactly twice total (bake + Go pool)" "2" \
  "$(grep -cE 'acquire_build_slot|with-build-slot\.sh' "$VERIFY")"
assert_eq "no slot acquisition on the guard, lint, --facts, or summary paths" "0" \
  "$(printf '%s\n' "$guards_through_facts" | grep -cE 'acquire_build_slot|with-build-slot\.sh')"

# --- bake selection: one solve, not one per target --------------------------
#
# bake_targets() matches a changed path against each service's `path` from
# .github/config/services.json, requiring the path to end in go.mod. CHANGED
# includes untracked files, so two untracked go.mods select exactly two
# targets — and must produce exactly one bake gate, not two.
#
# bake_targets() dedupes by SERVICE (atlas-ban/atlas-account), so a
# concurrent verify_test.sh run's own probe under either service is
# indistinguishable, from this block's perspective, from this run's — the
# lock above (held for the whole add/assert/remove/assert-gone sequence)
# is what makes "no probes, no bake gate" hold; bake_gate_lines_baseline is
# belt-and-suspenders for any pre-existing litter the lock can't see.
flock 8
bake_gate_lines_baseline="$(facts_selected --base HEAD | grep -c '^docker buildx bake' || true)"
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
assert_eq "no probes, no bake gate" "$bake_gate_lines_baseline" "$no_bake_lines"
flock -u 8

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

# A genuinely failing module driven through launch_go_layers/replay_go_layer.
# real_selected() strips the ✓/✗ glyph before comparing labels, so a
# regression in the pool's per-worker .rc propagation that silently turned a
# FAILED module into a reported PASS would not show up in the label-agreement
# assertions above. Of the two assertions below, only the second — the broken
# module still showing up as FAILED on the unstripped output — actually guards
# that regression class; a sabotaged `.rc` read can still leave the overall
# exit status non-zero for unrelated reasons, so the first assertion alone
# would not catch it.
# go.work.sum itself (unlike the backup file names above) is one real,
# shared, not-per-process path: it is THE workspace sum file, not a probe.
# A concurrent verify_test.sh run mutating it (via its own real `--quick`
# invocation below) while this run is mid snapshot/restore would corrupt
# either run's restore, so this whole snapshot/run/restore sequence is
# lock-protected too — held across a possible signal-driven interruption,
# since the EXIT/INT/TERM trap's own restore runs before this fd closes.
flock 8
rm -f "$gowork_sum_backup" "$gowork_sum_backup_absent"
if [ -f "$gowork_sum" ]; then
  cp -p "$gowork_sum" "$gowork_sum_backup"
else
  : > "$gowork_sum_backup_absent"
fi

mkdir -p "$probe_broken_dir"
cat > "$probe_broken_dir/go.mod" <<'EOF'
module zz.verify.probe.broken

go 1.21
EOF
cat > "$probe_broken_dir/main.go" <<'EOF'
package main

func main() {}

this is not valid Go syntax
EOF

broken_out="$("$VERIFY" --quick --base HEAD 2>&1)"
broken_rc=$?
rm -rf "$probe_broken_dir"
if [ -f "$gowork_sum_backup" ]; then
  mv -f "$gowork_sum_backup" "$gowork_sum"
elif [ -f "$gowork_sum_backup_absent" ]; then
  rm -f "$gowork_sum" "$gowork_sum_backup_absent"
fi
flock -u 8

assert_eq "a genuinely broken module makes the run exit non-zero" "1" "$broken_rc"
assert_true "the broken module is reported FAILED, unstripped" \
  "$(printf '%s\n' "$broken_out" \
     | grep 'FAILED' | grep "zz-verify-probe-broken-${probe_tag}" >/dev/null && echo true)"

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
