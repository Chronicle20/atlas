#!/usr/bin/env bash
# Generate the per-version LoadBalancer/Deployment port blocks in
# deploy/k8s/base/atlas-{login,channel}.yaml from the single declared version
# set (deploy/k8s/base/versions.json) and the shared port formula
# (services/atlas-pr-bootstrap/scripts/version-ports.sh). task-084 FR-3.
#
#   gen-lb-ports.sh           rewrite the marker blocks in place
#   gen-lb-ports.sh --check   generate to temp, diff vs checked-in; exit 1 on
#                             drift (the CI guard). No files are modified.
#
# Marker contract (per file): two labelled regions, content untouched outside.
#   # BEGIN generated:container-ports (...)  ...  # END generated:container-ports
#   # BEGIN generated:service-ports (...)    ...  # END generated:service-ports
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
VERSIONS="$REPO_ROOT/deploy/k8s/base/versions.json"
PORTS_LIB="$REPO_ROOT/services/atlas-pr-bootstrap/scripts/version-ports.sh"
LOGIN_YAML="$REPO_ROOT/deploy/k8s/base/atlas-login.yaml"
CHANNEL_YAML="$REPO_ROOT/deploy/k8s/base/atlas-channel.yaml"

# shellcheck source=../services/atlas-pr-bootstrap/scripts/version-ports.sh
. "$PORTS_LIB"

CHECK=0
[ "${1:-}" = "--check" ] && CHECK=1

[ -f "$VERSIONS" ] || { echo "gen-lb-ports: missing $VERSIONS" >&2; exit 1; }

# Reject duplicate majorVersion (Q4 / FR-1.3): same major -> same port -> LB collision.
dupes=$(jq -r '[.versions[].majorVersion] | group_by(.) | map(select(length>1)) | flatten | unique | .[]' "$VERSIONS")
if [ -n "$dupes" ]; then
    echo "gen-lb-ports: duplicate majorVersion(s) in versions.json: $(echo "$dupes" | tr '\n' ' ')" >&2
    exit 1
fi

# Emit "region major" pairs sorted by majorVersion (deterministic order).
versions_sorted() {
    jq -r '.versions | sort_by(.majorVersion) | .[] | "\(.region) \(.majorVersion)"' "$VERSIONS"
}

# Render the Deployment containerPort block for a service. $1=login|channel.
render_container_ports() {
    local svc="$1" region major port
    while read -r region major; do
        [ -z "$region" ] && continue
        if [ "$svc" = login ]; then port=$(derive_login_port "$major"); else port=$(derive_channel_port "$major"); fi
        printf '        - containerPort: %s\n' "$port"
    done < <(versions_sorted)
}

# Render the Service.ports block for a service. $1=login|channel.
render_service_ports() {
    local svc="$1" region major port
    while read -r region major; do
        [ -z "$region" ] && continue
        if [ "$svc" = login ]; then port=$(derive_login_port "$major"); else port=$(derive_channel_port "$major"); fi
        printf '  - port: %s\n' "$port"
        printf '    targetPort: %s\n' "$port"
        printf '    protocol: TCP\n'
        printf '    name: atlas-%s-%s-%s\n' "$svc" "$region" "$major"
    done < <(versions_sorted)
}

# Replace the lines between BEGIN/END of $label in $file with the contents of
# $blockfile (markers preserved). Echoes the rewritten file to stdout.
replace_block() {
    local file="$1" label="$2" blockfile="$3"
    awk -v label="$label" -v blockfile="$blockfile" '
        index($0, "BEGIN generated:" label) { print; while ((getline line < blockfile) > 0) print line; close(blockfile); skip=1; next }
        index($0, "END generated:" label)   { skip=0 }
        skip==1 { next }
        { print }
    ' "$file"
}

# Regenerate one file end-to-end; echoes the new content to stdout.
regen_file() {
    local file="$1" svc="$2" cp sp tmp
    cp="$(mktemp)"; sp="$(mktemp)"; tmp="$(mktemp)"
    render_container_ports "$svc" > "$cp"
    render_service_ports   "$svc" > "$sp"
    replace_block "$file" container-ports "$cp" > "$tmp"
    replace_block "$tmp"  service-ports   "$sp"
    rm -f "$cp" "$sp" "$tmp"
}

process() {
    local file="$1" svc="$2" gen
    [ -f "$file" ] || { echo "gen-lb-ports: missing $file" >&2; exit 1; }
    grep -q "BEGIN generated:container-ports" "$file" || { echo "gen-lb-ports: $file missing container-ports marker" >&2; exit 1; }
    grep -q "BEGIN generated:service-ports"   "$file" || { echo "gen-lb-ports: $file missing service-ports marker" >&2; exit 1; }
    gen="$(regen_file "$file" "$svc")"
    if [ "$CHECK" = 1 ]; then
        if ! diff -u "$file" <(printf '%s\n' "$gen"); then
            echo "gen-lb-ports: $file is stale; run tools/gen-lb-ports.sh and commit" >&2
            exit 1
        fi
    else
        printf '%s\n' "$gen" > "$file"
        echo "gen-lb-ports: wrote $file"
    fi
}

process "$LOGIN_YAML"   login
process "$CHANNEL_YAML" channel
