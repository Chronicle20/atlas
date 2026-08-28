#!/usr/bin/env bash
# scratch-sweep_test.sh — tests for tools/scratch-sweep.sh.
#
# Every case runs against $ATLAS_SCRATCH_ROOT inside a mktemp -d fixture, and
# the dangerous-root cases are asserted, not just exercised — the guard must
# be caught by a failing assertion if it is ever wrong, not by deleting /tmp.

set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SWEEP="$HERE/scratch-sweep.sh"
[ -x "$SWEEP" ] || { echo "FATAL: $SWEEP not executable" >&2; exit 2; }

fails=0
assert_eq()  { if [ "$2" = "$3" ]; then echo "ok   - $1"; else echo "FAIL - $1 (want '$2', got '$3')" >&2; fails=$((fails+1)); fi; }
assert_has() { case "$3" in *"$2"*) echo "ok   - $1";; *) echo "FAIL - $1 (missing '$2')" >&2; fails=$((fails+1));; esac; }
assert_lacks(){ case "$3" in *"$2"*) echo "FAIL - $1 (unexpectedly has '$2')" >&2; fails=$((fails+1));; *) echo "ok   - $1";; esac; }

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

# --- creates a missing root -------------------------------------------------

root="$tmp/scratch"
out="$(ATLAS_SCRATCH_ROOT="$root" "$SWEEP")"; rc=$?
assert_eq  "creates a missing root: exit 0" "0" "$rc"
[ -d "$root" ] && echo "ok   - root exists after creation" || { echo "FAIL - root was not created" >&2; fails=$((fails+1)); }
mode="$(stat -c '%a' "$root" 2>/dev/null || stat -f '%Lp' "$root")"
assert_eq "created root is mode 700" "700" "$mode"

# --- removes an entry older than the default age ---------------------------

root="$tmp/scratch1"
mkdir -p "$root"
touch -d '10 days ago' "$root/old.txt"
out="$(ATLAS_SCRATCH_ROOT="$root" "$SWEEP")"; rc=$?
assert_eq "removes stale entry: exit 0" "0" "$rc"
[ ! -e "$root/old.txt" ] && echo "ok   - old.txt gone" || { echo "FAIL - old.txt still present" >&2; fails=$((fails+1)); }

# --- keeps an entry inside the default age ----------------------------------

root="$tmp/scratch2"
mkdir -p "$root"
touch -d '2 days ago' "$root/new.txt"
out="$(ATLAS_SCRATCH_ROOT="$root" "$SWEEP")"; rc=$?
assert_eq "keeps fresh entry: exit 0" "0" "$rc"
[ -e "$root/new.txt" ] && echo "ok   - new.txt present" || { echo "FAIL - new.txt was removed" >&2; fails=$((fails+1)); }

# --- --age-days 1 removes the 2-day entry -----------------------------------

root="$tmp/scratch3"
mkdir -p "$root"
touch -d '2 days ago' "$root/new.txt"
out="$(ATLAS_SCRATCH_ROOT="$root" "$SWEEP" --age-days 1)"; rc=$?
assert_eq "--age-days 1: exit 0" "0" "$rc"
[ ! -e "$root/new.txt" ] && echo "ok   - new.txt gone under --age-days 1" || { echo "FAIL - new.txt still present" >&2; fails=$((fails+1)); }

# --- --now removes everything -----------------------------------------------

root="$tmp/scratch4"
mkdir -p "$root/dir"
touch -d '2 days ago' "$root/new.txt"
touch -d '2 days ago' "$root/dir"
out="$(ATLAS_SCRATCH_ROOT="$root" "$SWEEP" --now)"; rc=$?
assert_eq "--now: exit 0" "0" "$rc"
remaining="$(find "$root" -mindepth 1 -maxdepth 1 | wc -l | tr -d ' ')"
assert_eq "--now empties the root" "0" "$remaining"
[ -d "$root" ] && echo "ok   - root still exists after --now" || { echo "FAIL - root was removed" >&2; fails=$((fails+1)); }

# --- removes a stale directory, not just files ------------------------------

root="$tmp/scratch5"
mkdir -p "$root/dir"
touch "$root/dir/a"
touch -d '10 days ago' "$root/dir"
out="$(ATLAS_SCRATCH_ROOT="$root" "$SWEEP")"; rc=$?
assert_eq "removes stale dir: exit 0" "0" "$rc"
[ ! -e "$root/dir" ] && echo "ok   - dir gone" || { echo "FAIL - dir still present" >&2; fails=$((fails+1)); }

# --- --dry-run removes nothing -----------------------------------------------

root="$tmp/scratch6"
mkdir -p "$root"
touch -d '10 days ago' "$root/old.txt"
out="$(ATLAS_SCRATCH_ROOT="$root" "$SWEEP" --dry-run)"; rc=$?
assert_eq "--dry-run: exit 0" "0" "$rc"
[ -e "$root/old.txt" ] && echo "ok   - old.txt present after --dry-run" || { echo "FAIL - old.txt was removed by --dry-run" >&2; fails=$((fails+1)); }
assert_has "--dry-run stdout names the candidate" "old.txt" "$out"

# --- refuses dangerous roots -------------------------------------------------

out="$(ATLAS_SCRATCH_ROOT=/ "$SWEEP" --now 2>&1)"; rc=$?
assert_eq "refuses / : exit 2" "2" "$rc"
assert_has "refuses / : stderr says refusing" "refusing" "$out"

out="$(ATLAS_SCRATCH_ROOT=/tmp "$SWEEP" --now 2>&1)"; rc=$?
assert_eq "refuses /tmp : exit 2" "2" "$rc"
assert_has "refuses /tmp : stderr says refusing" "refusing" "$out"

out="$(ATLAS_SCRATCH_ROOT=/var/tmp "$SWEEP" --now 2>&1)"; rc=$?
assert_eq "refuses /var/tmp : exit 2" "2" "$rc"
assert_has "refuses /var/tmp : stderr says refusing" "refusing" "$out"

out="$(ATLAS_SCRATCH_ROOT="$HOME" "$SWEEP" --now 2>&1)"; rc=$?
assert_eq "refuses home dir: exit 2" "2" "$rc"
assert_has "refuses home dir: stderr says refusing" "refusing" "$out"

# --- unknown option -----------------------------------------------------------

out="$("$SWEEP" --nope 2>&1)"; rc=$?
assert_eq "unknown option: exit 2" "2" "$rc"
assert_has "unknown option: stderr says unknown option" "unknown option" "$out"

# --- --root overrides the env var --------------------------------------------

root="$tmp/other"
mkdir -p "$root"
touch -d '10 days ago' "$root/old.txt"
out="$(ATLAS_SCRATCH_ROOT="$tmp/unused" "$SWEEP" --root "$root")"; rc=$?
assert_eq "--root overrides env: exit 0" "0" "$rc"
[ ! -e "$root/old.txt" ] && echo "ok   - old.txt gone via --root" || { echo "FAIL - old.txt still present" >&2; fails=$((fails+1)); }

# --- summary names the count and the root ------------------------------------

root="$tmp/scratch7"
mkdir -p "$root"
touch -d '10 days ago' "$root/a.txt"
touch -d '10 days ago' "$root/b.txt"
out="$(ATLAS_SCRATCH_ROOT="$root" "$SWEEP")"; rc=$?
assert_eq "summary run: exit 0" "0" "$rc"
case "$out" in
    *"removed 2 entr"*) echo "ok   - summary names the count" ;;
    *) echo "FAIL - summary does not name the count (got: $out)" >&2; fails=$((fails+1)) ;;
esac
assert_has "summary names the root" "$root" "$out"

# --- -h prints usage and exits 0 ---------------------------------------------

out="$("$SWEEP" -h)"; rc=$?
assert_eq "-h: exit 0" "0" "$rc"
assert_has "-h: stdout contains usage:" "usage:" "$out"

echo
if [ "$fails" -eq 0 ]; then echo "scratch-sweep_test.sh: all assertions passed"
else echo "scratch-sweep_test.sh: $fails failure(s)" >&2; fi
[ "$fails" -eq 0 ]
