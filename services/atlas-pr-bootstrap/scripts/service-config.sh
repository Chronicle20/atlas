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

# Echo a fresh RFC 4122 UUID, or fail.
#
# `uuidgen` (util-linux) was missing from the image, and because
# `--arg id "$(uuidgen)"` swallows a missing-binary failure into an empty
# string, sparse mode POSTed `.data.id = ""`. atlas-configurations then
# minted its own id server-side (configurations/services/processor.go —
# "Use ID from input if provided and valid, otherwise generate a new one"),
# so bootstrap never learned the real row id and pushed the empty string on
# to `kubectl set env SERVICE_ID=`, which writes an env entry with NO value.
# atlas-channel and atlas-login then panicked at startup on
# `uuid.MustParse("")` and crash-looped.
#
# Three independent layers now prevent that: the Dockerfile installs
# util-linux; /proc/sys/kernel/random/uuid is a kernel-provided fallback
# needing no package at all; and the shape check below means a malformed or
# empty value fails the caller loudly instead of travelling onward as "".
# _SC_UUID_PROC is a test seam, not a tunable: it exists so the
# no-UUID-source-available failure path is reachable from bats (point it at
# a nonexistent file). Production never sets it.
new_uuid() {
    local id="" proc="${_SC_UUID_PROC-/proc/sys/kernel/random/uuid}"
    if command -v uuidgen >/dev/null 2>&1; then
        id=$(uuidgen)
    elif [ -r "$proc" ]; then
        # `read` builtin rather than cat(1): new_uuid must not itself depend
        # on a binary being present on PATH.
        read -r id < "$proc" || id=""
    fi
    # Bash-native match, for the same reason.
    if [[ ! "$id" =~ ^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$ ]]; then
        log error "new_uuid: could not generate a UUID (got '${id}'); is util-linux installed?"
        return 1
    fi
    printf '%s' "$id"
}

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

# Build the service-config payload for this environment. $1 = shape
# (login | channel | none), $2 = canonical template path.
#
# ATLAS_MODE=sparse: a NEW row with a fresh UUID carrying exactly this
# environment's single tenant entry (or none, for a tenant-agnostic shape).
# main's row is never read, written or merged into (G7, NG6). The services
# table is Id-keyed and consumers select by SERVICE_ID, so multiple rows of
# one Type are already representable (design C1) — no key change is needed
# to make this work.
#
# ATLAS_MODE=isolated (default): unchanged. Reproduces today's first-write
# POST body exactly — the pinned canonical id, with .tenants UNCONDITIONALLY
# REPLACED by [entry] (NOT merge_tenant_entry — some canonical templates
# ship a non-empty seeded tenants[], e.g. channel-service.json's placeholder
# ec876921…/port 0, and merging would append this environment's entry beside
# that placeholder instead of replacing it, as the original code did). The
# GET-merge-PATCH network sequence for an EXISTING live row stays in
# bootstrap.sh's upsert_service_config, unchanged, and IS correct to use
# merge_tenant_entry there — that merge is against live data with possibly
# many real foreign tenants, not this pure function's canonical fixture.
build_service_config() {
    local shape="$1" tmpl="$2" entry
    case "$shape" in
        login)   entry=$(build_login_entry) ;;
        channel) entry=$(build_channel_entry "$tmpl") ;;
        none)    entry="" ;;
        *)       log error "build_service_config: unknown shape '$shape'"; return 1 ;;
    esac

    if [ "${ATLAS_MODE:-isolated}" = "sparse" ]; then
        local id
        # Not inlined as `--arg id "$(uuidgen)"`: command substitution
        # swallows a missing-binary failure into an empty string, which is
        # exactly how this shipped broken. See new_uuid's header.
        id=$(new_uuid) || return 1
        jq -c --arg id "$id" --arg envn "$ATLAS_ENVIRONMENT" --argjson entry "${entry:-null}" '
            .data.id = $id
            | .data.attributes.environment = $envn
            | if $entry == null then . else .data.attributes.tenants = [$entry] end
        ' "$tmpl"
    else
        if [ -n "$entry" ]; then
            jq -c --argjson entry "$entry" '.data.attributes.tenants = [$entry]' "$tmpl"
        else
            cat "$tmpl"
        fi
    fi
}
