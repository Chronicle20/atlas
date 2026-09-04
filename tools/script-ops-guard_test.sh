#!/usr/bin/env bash
# script-ops-guard_test.sh — regression test for script-ops-guard.sh.
#
# Builds a throwaway tree under mktemp -d mirroring services/atlas-x/ and
# runs the guard against it via the SCRIPT_OPS_GUARD_ROOT env override,
# asserting the guard fires on a direct shared-payload construction under a
# script service, exempts atlas-saga-orchestrator (which legitimately
# consumes the payloads), and ignores commented-out occurrences.
#
# Run directly: tools/script-ops-guard_test.sh
set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GUARD="$HERE/script-ops-guard.sh"

fails=0
pass() { echo "ok   - $1"; }
fail() { echo "FAIL - $1" >&2; fails=$((fails + 1)); }

run_case() {
    local desc="$1" root="$2" want_rc="$3" want_substr="$4"
    local out rc
    out="$(SCRIPT_OPS_GUARD_ROOT="$root" "$GUARD" 2>&1)"
    rc=$?
    if [ "$rc" -ne "$want_rc" ]; then
        fail "$desc (exit $rc, wanted $want_rc; output: $out)"
        return
    fi
    if [ -n "$want_substr" ] && ! printf '%s\n' "$out" | grep -qF -- "$want_substr"; then
        fail "$desc (missing '$want_substr' in: $out)"
        return
    fi
    pass "$desc"
}

# Case: clean tree — no banned literal anywhere.
ROOT1="$(mktemp -d)"
mkdir -p "$ROOT1/services/atlas-map-actions/atlas.com/map"
cat > "$ROOT1/services/atlas-map-actions/atlas.com/map/x.go" <<'EOF'
package map

func doThing() {
	ops.Build()
}
EOF
run_case "clean tree exits 0 with OK" "$ROOT1" 0 "OK"
rm -rf "$ROOT1"

# Case: banned literal in a service file.
ROOT2="$(mktemp -d)"
mkdir -p "$ROOT2/services/atlas-map-actions/atlas.com/map"
cat > "$ROOT2/services/atlas-map-actions/atlas.com/map/x.go" <<'EOF'
package map

func doThing() {
	p := saga.SpawnMonsterPayload{}
	_ = p
}
EOF
run_case "banned literal fails and names file+type" "$ROOT2" 1 "SpawnMonsterPayload"
out2="$(SCRIPT_OPS_GUARD_ROOT="$ROOT2" "$GUARD" 2>&1)"
if printf '%s\n' "$out2" | grep -qF "services/atlas-map-actions/atlas.com/map/x.go"; then
    pass "banned literal names the offending file"
else
    fail "banned literal names the offending file (output: $out2)"
fi
rm -rf "$ROOT2"

# Case: atlas-saga-orchestrator is exempt (it legitimately consumes the payloads).
ROOT3="$(mktemp -d)"
mkdir -p "$ROOT3/services/atlas-saga-orchestrator/atlas.com/saga-orchestrator"
cat > "$ROOT3/services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/y.go" <<'EOF'
package sagaorchestrator

func doThing() {
	p := saga.SpawnMonsterPayload{}
	_ = p
}
EOF
run_case "saga-orchestrator is exempt" "$ROOT3" 0 "OK"
rm -rf "$ROOT3"

# Case: a _test.go file is not exempt — a test that constructs the payload
# directly is the same duplication as production code doing it.
ROOT4="$(mktemp -d)"
mkdir -p "$ROOT4/services/atlas-map-actions/atlas.com/map"
cat > "$ROOT4/services/atlas-map-actions/atlas.com/map/x_test.go" <<'EOF'
package map

func TestThing(t *testing.T) {
	p := saga.SpawnMonsterPayload{}
	_ = p
}
EOF
run_case "test file is not exempt" "$ROOT4" 1 "SpawnMonsterPayload"
rm -rf "$ROOT4"

# Case: a commented-out occurrence does not count.
ROOT5="$(mktemp -d)"
mkdir -p "$ROOT5/services/atlas-map-actions/atlas.com/map"
cat > "$ROOT5/services/atlas-map-actions/atlas.com/map/x.go" <<'EOF'
package map

// saga.SpawnMonsterPayload{} was the old way; build via ops instead.
func doThing() {
	ops.Build()
}
EOF
run_case "commented-out occurrence does not count" "$ROOT5" 0 "OK"
rm -rf "$ROOT5"

# Case: a line carrying the inline `script-ops-guard:allow` marker is a
# documented, reviewed exemption (a call site that legitimately can't use
# ops.* because it lacks script params), not a comment or a clean tree.
ROOT6="$(mktemp -d)"
mkdir -p "$ROOT6/services/atlas-map-actions/atlas.com/map"
cat > "$ROOT6/services/atlas-map-actions/atlas.com/map/x.go" <<'EOF'
package map

func doThing() {
	p := saga.SpawnMonsterPayload{ // script-ops-guard:allow — test fixture, not a real exemption.
	}
	_ = p
}
EOF
run_case "inline allow marker suppresses the match" "$ROOT6" 0 "OK"
rm -rf "$ROOT6"

# Case: a service outside the four script-operation-table services is out
# of scope — it builds these same payload types for its own domain logic,
# not by re-implementing a script operation, so it is never flagged.
ROOT7="$(mktemp -d)"
mkdir -p "$ROOT7/services/atlas-quest/atlas.com/quest"
cat > "$ROOT7/services/atlas-quest/atlas.com/quest/x.go" <<'EOF'
package quest

func doThing() {
	p := saga.StartQuestPayload{}
	_ = p
}
EOF
run_case "a non-script-operation-table service is out of scope" "$ROOT7" 0 "OK"
rm -rf "$ROOT7"

if [ "$fails" -ne 0 ]; then
    echo "FAILED: $fails assertion(s)" >&2
    exit 1
fi
echo "script-ops-guard_test.sh: all assertions passed"
