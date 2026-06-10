#!/usr/bin/env bash
# Pure (network-free) helpers for the additive services-config upsert
# (task-084 FR-2). Sourced by bootstrap.sh and unit-tested directly by
# test/service_config_test.bats. Depends on version-ports.sh and the env
# vars TENANT_ID / MAJOR_VERSION / LB_IP set by the caller.

# Resolve version-ports.sh whether running from the image (/atlas) or from a
# checkout (scripts/ sibling).
_sc_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ -f "$_sc_dir/version-ports.sh" ]; then
    . "$_sc_dir/version-ports.sh"
else
    . /atlas/version-ports.sh
fi
unset _sc_dir

# Echo the login tenant entry {id, port} with the version-derived login port.
build_login_entry() {
    local port
    port=$(derive_login_port "$MAJOR_VERSION") || return 1
    jq -cn --arg id "$TENANT_ID" --argjson port "$port" '{id:$id, port:$port}'
}

# Echo the channel tenant entry from the canonical template's worlds shell,
# with id / ipAddress / the first channel's port overwritten. $1 = template path.
build_channel_entry() {
    local tmpl="$1" port
    port=$(derive_channel_port "$MAJOR_VERSION") || return 1
    jq -c --arg id "$TENANT_ID" --arg ip "$LB_IP" --argjson port "$port" '
        .data.attributes.tenants[0]
        | .id = $id
        | .ipAddress = $ip
        | .worlds[0].channels[0].port = $port
    ' "$tmpl"
}

# Upsert $1 (an entry JSON) into the tenants[] of the attributes JSON read on
# stdin, keyed by .id, preserving order and foreign entries. Tenant-agnostic
# attributes (no "tenants" key) pass through unchanged. Echoes merged attributes.
# The entry MUST have an .id (callers build it with one); an id-less entry is not supported.
merge_tenant_entry() {
    local entry="$1"
    jq -c --argjson entry "$entry" '
        if has("tenants") then
          .tenants = (
            if any(.tenants[]?; .id == $entry.id)
            then (.tenants | map(if .id == $entry.id then $entry else . end))
            else (.tenants + [$entry]) end )
        else . end'
}
