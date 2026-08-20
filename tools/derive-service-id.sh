#!/usr/bin/env sh
# derive-service-id.sh — the single derivation site for a sparse
# environment's SERVICE_ID.
#
# Usage: derive-service-id.sh <service-type> <environment>
#
# Pure function of (service type, environment): deterministic, no
# random/uuidgen dependency, so CI can bake the result into a manifest
# instead of minting it at bootstrap and fighting Argo's selfHeal over it.
#
# Prints the derived UUID to stdout with no trailing newline.
set -eu

usage() {
    echo "Usage: derive-service-id.sh <service-type> <environment>" >&2
}

if [ "$#" -lt 2 ] || [ -z "${1:-}" ] || [ -z "${2:-}" ]; then
    usage
    exit 2
fi

SERVICE_TYPE="$1"
ENVIRONMENT="$2"

# ATLAS_SERVICE_NS — the UUIDv5 namespace every derived SERVICE_ID depends
# on. It appears here and in exactly one other place, atlasServiceNS in
# services/atlas-configurations/atlas.com/configurations/servicesuniq/migration.go,
# which carries the reciprocal reference. Never regenerate it: changing it
# re-keys every sparse environment's service-config row.
# Reproducible rather than arbitrary, so the value can be re-derived if this
# line is ever lost:
#   uuid5(NAMESPACE_DNS, "service-config.atlas.chronicle20")
ATLAS_SERVICE_NS=c8f90111-a0cf-513e-95e6-c54609e5dec0

if ! command -v python3 >/dev/null 2>&1; then
    echo "FATAL: derive-service-id.sh requires python3 on PATH" >&2
    exit 1
fi

python3 - "$ATLAS_SERVICE_NS" "$SERVICE_TYPE/$ENVIRONMENT" <<'PY'
import sys, uuid
sys.stdout.write(str(uuid.uuid5(uuid.UUID(sys.argv[1]), sys.argv[2])))
PY
