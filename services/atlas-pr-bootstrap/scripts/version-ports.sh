#!/usr/bin/env bash
# Single source of truth for Atlas per-version socket ports (task-084 FR-1).
#   loginPort(major)   = major * 100
#   channelPort(major) = loginPort(major) + 1
# Derivation is a function of majorVersion ONLY (FR-1.3). Sourced by both
# scripts/bootstrap.sh (runtime, via /atlas/version-ports.sh) and
# tools/gen-lb-ports.sh (build/CI). No port arithmetic is written anywhere
# else in the repo.

derive_login_port() {
    local major="$1"
    if ! printf '%s' "$major" | grep -Eq '^[0-9]+$'; then
        echo "derive_login_port: majorVersion '$major' is not a non-negative integer" >&2
        return 1
    fi
    echo $((major * 100))
}

derive_channel_port() {
    local major="$1"
    local login
    login=$(derive_login_port "$major") || return 1
    echo $((login + 1))
}
