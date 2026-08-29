#!/usr/bin/env bash
# tools/with-build-slot.sh — CLI wrapper around tools/lib/build-slot.sh for
# external callers that run a command as a subprocess.
#
# tools/verify.sh does NOT use this wrapper: it needs to hold a slot around
# the shell function launch_go_layers, and a slot held by a subprocess would
# release the instant that subprocess exits — before the build it is meant
# to guard even starts. verify.sh sources tools/lib/build-slot.sh directly.
# This wrapper exists for everyone else: a human at a terminal, a Makefile
# target, CI steps that only need to gate one external command.
#
# usage: tools/with-build-slot.sh [--slots N] [--timeout SEC] <label> -- <command...>
#
#   --slots N       number of machine-wide slots (default: $ATLAS_BUILD_SLOTS,
#                   else 4)
#   --timeout SEC   give up after SEC seconds waiting for a slot and exit 75
#                   (EX_TEMPFAIL) instead of blocking forever
#   <label>         what this slot is for; appears in the stderr diagnostics
#   -h, --help      this message
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

usage() {
    cat <<'EOF'
usage: tools/with-build-slot.sh [--slots N] [--timeout SEC] <label> -- <command...>

  --slots N       number of machine-wide slots (default: $ATLAS_BUILD_SLOTS,
                  else 4)
  --timeout SEC   give up after SEC seconds waiting for a slot and exit 75
                  (EX_TEMPFAIL) instead of blocking forever
  <label>         what this slot is for; appears in the stderr diagnostics
  -h, --help      this message
EOF
}

slots=""
slot_timeout=""
label=""
saw_dashdash=0

# Flags and the <label> may appear in any order before --; every invocation
# in the acceptance table puts <label> first, ahead of --slots/--timeout.
while [ $# -gt 0 ]; do
    case "$1" in
        -h | --help)
            usage
            exit 0
            ;;
        --slots)
            [ $# -ge 2 ] || { echo "with-build-slot: --slots requires a value" >&2; exit 2; }
            slots="$2"
            shift 2
            ;;
        --timeout)
            [ $# -ge 2 ] || { echo "with-build-slot: --timeout requires a value" >&2; exit 2; }
            slot_timeout="$2"
            shift 2
            ;;
        --)
            saw_dashdash=1
            shift
            break
            ;;
        -*)
            echo "with-build-slot: unknown option: $1" >&2
            exit 2
            ;;
        *)
            label="$1"
            shift
            ;;
    esac
done

if [ "$saw_dashdash" -ne 1 ]; then
    echo "with-build-slot: expected a '--' separator before the command" >&2
    usage >&2
    exit 2
fi

if [ -z "$label" ]; then
    echo "with-build-slot: missing <label>" >&2
    usage >&2
    exit 2
fi

if [ $# -eq 0 ]; then
    echo "with-build-slot: missing <command...> after --" >&2
    exit 2
fi

[ -n "$slots" ] && export ATLAS_BUILD_SLOTS="$slots"
[ -n "$slot_timeout" ] && export ATLAS_BUILD_SLOT_TIMEOUT="$slot_timeout"

# shellcheck source=./lib/build-slot.sh
. "$ROOT/tools/lib/build-slot.sh"

acquire_build_slot "$label" || exit $?

"$@"
