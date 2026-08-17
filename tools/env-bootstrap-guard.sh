#!/usr/bin/env bash
# tools/env-bootstrap-guard.sh — the gate on enabling sparse mode at all
# (design.md §12): sparse mode is not selectable until every service wires
# the environment registry into its Bootstrap call, and CI proves it.
#
# Fails when a services/*/atlas.com/*/main.go calls
# github.com/Chronicle20/atlas/libs/atlas-service's Bootstrap without also
# passing WithEnvironmentRegistry. This is a source scan (shell + grep, in
# the style of tools/service-registration-guard.sh), not a go/analysis pass:
# main.go is a fixed, easy-to-grep call site, and an AST pass buys nothing
# here that a regex over the import alias + two call sites doesn't already
# give.
#
# Handles the atlas-service import under any local alias (e.g. atlas-character
# and atlas-storage import it as "lifecycle") — matching the literal string
# "service.Bootstrap" would silently pass those two forever.
#
# Expected to FAIL today, listing all services not yet migrated to
# WithEnvironmentRegistry (Phase C). Unlike env-domain-guard.sh, this guard
# is NOT wired into tools/verify.sh yet — Task 52 adds that once Phase C
# reaches zero.
#
# Usage: tools/env-bootstrap-guard.sh   (from the repo root)
# Exit 0 = every non-exempt service wires the registry, 1 = violations found.

set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

ALLOWLIST="tools/envguard/bootstrap-allowlist.txt"

is_allowlisted() {
    local svc="$1"
    grep -vE '^\s*#|^\s*$' "$ALLOWLIST" | grep -qxF "$svc"
}

fail=0
checked=0
missing=0

for main in services/*/atlas.com/*/main.go; do
    [ -f "$main" ] || continue
    svc="$(printf '%s\n' "$main" | cut -d/ -f2)"

    if is_allowlisted "$svc"; then
        continue
    fi

    # Resolve the local alias atlas-service is imported under (usually
    # "service", sometimes "lifecycle") so "<alias>.Bootstrap(" and
    # "<alias>.WithEnvironmentRegistry(" are matched correctly regardless of
    # naming.
    alias="$(grep -oE '\w+ "github\.com/Chronicle20/atlas/libs/atlas-service"' "$main" | awk '{print $1}' | head -1)"
    if [ -z "$alias" ]; then
        if grep -q '"github.com/Chronicle20/atlas/libs/atlas-service"' "$main"; then
            alias="service"
        else
            # Genuinely does not import atlas-service at all — not this
            # guard's concern (and not currently true for any service; see
            # bootstrap-allowlist.txt).
            continue
        fi
    fi

    if ! grep -qE "${alias}\.Bootstrap\(" "$main"; then
        # Imports atlas-service but never calls Bootstrap — not this
        # guard's concern.
        continue
    fi

    checked=$((checked + 1))
    if ! grep -qE "${alias}\.WithEnvironmentRegistry\(" "$main"; then
        echo "FAIL: $svc ($main) calls ${alias}.Bootstrap without ${alias}.WithEnvironmentRegistry"
        missing=$((missing + 1))
        fail=1
    fi
done

echo ""
if [ "$fail" -ne 0 ]; then
    echo "env-bootstrap-guard: $missing/$checked service(s) missing WithEnvironmentRegistry"
    echo "See libs/atlas-service/envregistry.go and design.md §12 (Phase C)."
    exit 1
fi
echo "env-bootstrap-guard: clean ($checked service(s) checked)"
