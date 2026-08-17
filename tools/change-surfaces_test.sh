#!/usr/bin/env bash
# change-surfaces_test.sh — hermetic tests for tools/change-surfaces.sh.
#
# Builds a throwaway repo with Atlas-shaped paths so the assertions never depend
# on the live repo's evolving diff. The classifier is copied in, so its
# `dirname $0/..` root resolves to the fixture.
#
# The fail-open cases are the important ones. A classifier that under-classifies
# silently narrows a review, which is worse than no classifier at all — so every
# case where the input is not fully understood must widen to every family and
# say `classification=uncertain`.

set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SRC="$HERE/change-surfaces.sh"
[ -x "$SRC" ] || { echo "FATAL: $SRC not executable" >&2; exit 2; }

fails=0
assert_eq() {
  if [ "$2" = "$3" ]; then echo "ok   - $1"
  else echo "FAIL - $1 (want '$2', got '$3')" >&2; fails=$((fails + 1)); fi
}
assert_has() { # desc key-substring output
  case "$3" in
    *"$2"*) echo "ok   - $1" ;;
    *) echo "FAIL - $1 (missing '$2')" >&2; fails=$((fails + 1)) ;;
  esac
}
assert_lacks() {
  case "$3" in
    *"$2"*) echo "FAIL - $1 (unexpectedly contains '$2')" >&2; fails=$((fails + 1)) ;;
    *) echo "ok   - $1" ;;
  esac
}

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
repo="$tmp/repo"
mkdir -p "$repo/tools"
cp "$SRC" "$repo/tools/"
git -C "$repo" init -q -b main
git -C "$repo" config user.email t@t.t
git -C "$repo" config user.name t
git -C "$repo" config commit.gpgsign false

# Base commit: an existing service with a full DDD domain package, a sub-domain
# package, and a UI.
dom="services/atlas-thing/atlas.com/thing/internal/thing"
sub="services/atlas-thing/atlas.com/thing/internal/action"
mkdir -p "$repo/$dom" "$repo/$sub" "$repo/services/atlas-ui/src" "$repo/docs"
for f in model.go entity.go rest.go processor.go provider.go resource.go; do
  echo "package thing" > "$repo/$dom/$f"
done
for f in resource.go processor.go; do
  echo "package action" > "$repo/$sub/$f"
done
echo "x" > "$repo/services/atlas-ui/src/page.tsx"
echo "readme" > "$repo/docs/readme.md"
git -C "$repo" add -A
git -C "$repo" commit -qm base
base="$(git -C "$repo" rev-parse HEAD)"

cd "$repo"
run() { ./tools/change-surfaces.sh --base "$base"; }
commit() { git -C "$repo" add -A; git -C "$repo" commit -qm "$1"; }

# --- docs-only change: nothing is in scope ---------------------------------

echo "hello" >> "$repo/docs/readme.md"
commit docs
out="$(run)"
assert_eq "docs-only: go_changed" "go_changed=false" "$(printf '%s\n' "$out" | grep '^go_changed=')"
assert_eq "docs-only: no families" "backend_audit_families=none" \
  "$(printf '%s\n' "$out" | grep '^backend_audit_families=')"
assert_eq "docs-only: confident" "classification=confident" \
  "$(printf '%s\n' "$out" | grep '^classification=')"
assert_eq "docs-only: no frontend review" "frontend_review=false" \
  "$(printf '%s\n' "$out" | grep '^frontend_review=')"

# --- a domain package changes ----------------------------------------------

cat >> "$repo/$dom/processor.go" <<'EOF'
func (p *ProcessorImpl) Do() error { return nil }
EOF
commit domain
out="$(run)"
assert_has "domain: FILE family"          "FILE"          "$out"
assert_has "domain: DOM-STRUCTURE family" "DOM-STRUCTURE" "$out"
assert_has "domain: REST family"          "REST"          "$out"
assert_has "domain: RUNTIME family"       "RUNTIME"       "$out"
assert_has "domain: rest_surface"         "rest_surface=true" "$out"
assert_has "domain: names the service"    "changed_services=atlas-thing" "$out"
assert_has "domain: counts the package"   "changed_packages=1" "$out"
assert_eq  "domain: still confident" "classification=confident" \
  "$(printf '%s\n' "$out" | grep '^classification=')"

# --- SUB is per-package, not a union ---------------------------------------
#
# The sub-domain package has resource.go and no model.go. A union over all
# changed packages would answer false here, because the domain package changed
# in the same commit and DOES have a model.go.

echo "// touch" >> "$repo/$sub/resource.go"
commit subdomain
out="$(run)"
assert_has "sub-domain: SUB family fires despite a sibling domain package" "SUB" "$out"

# --- database and messaging surfaces ---------------------------------------

cat >> "$repo/$dom/entity.go" <<'EOF'
func Migration() error { return nil }
EOF
commit entity
out="$(run)"
assert_has "entity change: db_surface" "db_surface=true" "$out"

mkdir -p "$repo/services/atlas-thing/atlas.com/thing/internal/kafka/message/thing"
echo "package thing" > "$repo/services/atlas-thing/atlas.com/thing/internal/kafka/message/thing/kafka.go"
commit kafka
out="$(run)"
assert_has "kafka path: kafka_surface" "kafka_surface=true" "$out"
assert_has "kafka path: MESSAGING family" "MESSAGING" "$out"

# --- frontend ---------------------------------------------------------------

echo "y" >> "$repo/services/atlas-ui/src/page.tsx"
commit ui
out="$(run)"
assert_has "ui change: frontend_review" "frontend_review=true" "$out"
assert_has "ui change: ts_changed"      "ts_changed=true"      "$out"

# --- a new service ----------------------------------------------------------

mkdir -p "$repo/services/atlas-new/atlas.com/new"
echo "module atlas.com/new" > "$repo/services/atlas-new/atlas.com/new/go.mod"
echo "package main" > "$repo/services/atlas-new/atlas.com/new/main.go"
commit newsvc
out="$(run)"
assert_has "new service: new_service=true" "new_service=true" "$out"
assert_has "new service: SCAFFOLD family" "SCAFFOLD" "$out"

# --- FAIL OPEN: a Go file in an unrecognised layout -------------------------
#
# services/, libs/ and tools/ are the layouts this classifier understands.
# Anything else must widen rather than guess.

mkdir -p "$repo/pkg/novel"
echo "package novel" > "$repo/pkg/novel/thing.go"
commit novel
out="$(run)"
assert_eq "novel layout: uncertain" "classification=uncertain" \
  "$(printf '%s\n' "$out" | grep '^classification=')"
for fam in DOM-STRUCTURE FILE SUB REST CONSTANTS TESTING CACHE MESSAGING \
           MULTITENANCY MIGRATION DEPLOY RUNTIME CHANNEL-WIRE RESILIENCE EXT SCAFFOLD SEC; do
  assert_has "novel layout: widened to $fam" "$fam" \
    "$(printf '%s\n' "$out" | grep '^backend_audit_families=')"
done
assert_has "novel layout: names the reason" "uncertain_reason=" "$out"
git -C "$repo" rm -rq pkg
commit drop-novel

# --- FAIL OPEN: a base that does not resolve --------------------------------

out="$(./tools/change-surfaces.sh --base deadbeefdeadbeef 2>/dev/null)"
rc=$?
assert_eq "bad base: still exits 0" "0" "$rc"
assert_eq "bad base: uncertain" "classification=uncertain" \
  "$(printf '%s\n' "$out" | grep '^classification=')"
assert_has "bad base: every surface true" "rest_surface=true" "$out"
assert_has "bad base: frontend review true" "frontend_review=true" "$out"

# --- FAIL OPEN: --all --------------------------------------------------------

out="$(./tools/change-surfaces.sh --all)"
assert_eq "--all: uncertain" "classification=uncertain" \
  "$(printf '%s\n' "$out" | grep '^classification=')"
assert_has "--all: every family" "SEC" "$out"

# --- tools/ is a known layout, not a novel one ------------------------------
#
# It must NOT widen (tools/ carries no DDD packages), and it must be reported
# rather than swallowed, because a change there changes the guards themselves.

mkdir -p "$repo/tools/thingguard"
echo "package thingguard" > "$repo/tools/thingguard/analyzer.go"
commit toolsgo
out="$(run)"
assert_eq "tools/ Go: still confident" "classification=confident" \
  "$(printf '%s\n' "$out" | grep '^classification=')"
assert_has "tools/ Go: tooling_surface reported" "tooling_surface=true" "$out"

# --- output contract --------------------------------------------------------

out="$(run)"
assert_eq "output is key=value lines only" "" \
  "$(printf '%s\n' "$out" | grep -v '^[a-z_]*=' || true)"
assert_lacks "no absolute paths leak into the block" "$tmp" "$out"

echo
if [ "$fails" -eq 0 ]; then echo "change-surfaces_test.sh: all assertions passed"
else echo "change-surfaces_test.sh: $fails failure(s)" >&2; fi
[ "$fails" -eq 0 ]
