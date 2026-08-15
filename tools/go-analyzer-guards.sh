#!/usr/bin/env bash
# tools/go-analyzer-guards.sh — every go/analysis guard in ONE sweep.
#
# rediskeyguard, outboxguard, goroutineguard and buffdurationguard each used to
# own a CI job. Four jobs, four runners, four cold `go vet` passes over the same
# 64 service modules — and what those passes actually spent their time on was
# type-checking, not analysis. Measured on run 31889679664: 376s, 384s, 359s and
# 285s, of which the analyzer build was ~10s.
#
# This script registers all four analyzers with a single unitchecker binary
# (tools/atlasguards), so the tree is parsed and type-checked once and each
# analyzer walks the same syntax trees. What each guard DETECTS is unchanged,
# and so is what each guard SCANS — services-only guards stay services-only via
# a second, narrower binary. See tools/atlasguards/guards.go.
#
# The individual tools/<name>-guard.sh entry points remain, unchanged, as the
# local iteration path and the single-guard escape hatch.
#
# Usage:
#     tools/go-analyzer-guards.sh
#
# Env:
#   GUARD_JOBS       override the parallelism (default: nproc, capped at 8)
#   GUARD_NOCACHE=1  force a rebuild of the vettool binaries
#   GUARD_SERVICE_MODULES  newline/space-separated module dirs to analyze in the
#                    services/ pass instead of all 64. Unset means "sweep
#                    everything under services/".
#   GUARD_LIB_MODULES      the same, for the libs/ pass.
#
#                    The two are separate on purpose. The affected-module
#                    matrix CI computes covers services/ exactly (64 of 64),
#                    but .github/config/services.json lists only 9 of the 22
#                    Go modules under libs/ — scoping the libs pass to that
#                    matrix would silently drop 13 modules from goroutineguard
#                    and buffdurationguard coverage. CI therefore scopes
#                    services/ and always sweeps libs/, which is cheap: the
#                    libs modules are small and the services pass has already
#                    warmed their compiled form in GOCACHE.
#   GUARD_SKIP_SELFTEST=1  skip the analyzers' own unit tests (they are the
#                    cheap part; CI runs them, `--quick` paths may not need to)

set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tools/lib/analyzer-guard.sh"

SRC="$ROOT/tools/atlasguards"
# The content key spans the combined module AND every analyzer it links, so an
# edit to any analyzer's source rebuilds the shared binaries. Keying on
# tools/atlasguards alone would leave a stale binary enforcing the pre-change
# rules while looking like it ran.
GUARD_SRCS=(
    "$SRC"
    "$ROOT/tools/rediskeyguard"
    "$ROOT/tools/outboxguard"
    "$ROOT/tools/goroutineguard"
    "$ROOT/tools/buffdurationguard"
)

# Guards that ship their own unit tests. rediskeyguard and outboxguard have
# none to run — this mirrors the SELFTEST flags on the per-guard wrappers.
SELFTEST_GUARDS=(goroutineguard buffdurationguard)

if [ "${GUARD_SKIP_SELFTEST:-0}" -ne 1 ]; then
    for g in "${SELFTEST_GUARDS[@]}"; do
        echo "self-testing $g..."
        ( cd "$ROOT/tools/$g" && GOWORK=off go test ./... )
    done
fi

SERVICES_BIN="$(analyzer_guard_build atlasguards-services-vet "$SRC" \
    ./cmd/atlasguards-services-vet "${GUARD_SRCS[@]}")"
LIBS_BIN="$(analyzer_guard_build atlasguards-libs-vet "$SRC" \
    ./cmd/atlasguards-libs-vet "${GUARD_SRCS[@]}")"

LOG="$(mktemp)"
trap 'rm -f "$LOG"' EXIT

# Resolve both scopes up front. An unresolvable module list is a caller bug —
# a relative path, an unexpanded $GITHUB_WORKSPACE — and must stop the run
# rather than silently analyze zero modules and report PASS.
SERVICES_SCOPE="$( GUARD_MODULES="${GUARD_SERVICE_MODULES:-}"; analyzer_guard_scope "$ROOT/services" )"
LIBS_SCOPE="$( GUARD_MODULES="${GUARD_LIB_MODULES:-}"; analyzer_guard_scope "$ROOT/libs" )"

rc=0

# services/: rediskeyguard + outboxguard + goroutineguard + buffdurationguard
printf '%s\n' "$SERVICES_SCOPE" \
    | analyzer_guard_vet "$SERVICES_BIN" "go-analyzer-guards (services)" 2>>"$LOG" || rc=1

# libs/: goroutineguard + buffdurationguard only
printf '%s\n' "$LIBS_SCOPE" \
    | analyzer_guard_vet "$LIBS_BIN" "go-analyzer-guards (libs)" 2>>"$LOG" || rc=1

if [ "$rc" -eq 0 ]; then
    echo "go-analyzer-guards: PASS"
    exit 0
fi

cat "$LOG" >&2
echo ""
echo "go-analyzer-guards: FAIL"

# Every analyzer prefixes its diagnostics with its own name, so the remediation
# hint below is keyed off what actually fired rather than dumping all four.
if grep -q '^\s*.*rediskeyguard:' "$LOG"; then
    echo "  rediskeyguard — raw keyed redis client calls found (use a libs/atlas-redis type)"
fi
if grep -q '^\s*.*outboxguard:' "$LOG"; then
    echo "  outboxguard — direct producer calls inside DB transactions (use outbox.EmitProvider)"
fi
if grep -q '^\s*.*goroutineguard:' "$LOG"; then
    echo "  goroutineguard — bare go statements found (use routine.Go from libs/atlas-routine)"
fi
if grep -q '^\s*.*buffdurationguard:' "$LOG"; then
    echo "  buffdurationguard — seconds-valued buff duration emitter found."
    echo "    The COMMAND_TOPIC_CHARACTER_BUFF duration field is MILLISECONDS."
    echo "    Contract owner: services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go"
fi

echo ""
echo "  To re-run one guard on its own: tools/<name>-guard.sh"

exit 1
