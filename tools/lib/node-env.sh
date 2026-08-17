# shellcheck shell=bash
# tools/lib/node-env.sh — put the required Node on PATH, if it is not already.
#
# Source it; do not execute it:
#
#     . "$(dirname "$0")/lib/node-env.sh"
#
# Why this exists
# ---------------
# Every gate invocation that could reach the atlas-ui layer was being written as
#
#     export NVM_DIR=… && . "$NVM_DIR/nvm.sh" && nvm use 22 >/dev/null && tools/verify.sh …
#
# — an environment bootstrap replicated into ~12 command strings per session,
# measured across two audited tasks. The bytes are trivial; the fragility is
# not. A prefix that lives in the command string is a prefix that gets
# mistyped, dropped from one call in ten, and re-derived by every new session.
#
# It is a no-op when a suitable `node` is already on PATH — CI, a direnv/asdf
# shell, a devcontainer — so it costs nothing where it is not needed and never
# overrides a Node the caller deliberately selected.
#
# NODE_MAJOR_REQUIRED may be set by the caller; tools/lint.sh owns the value of
# record. Nothing here hardcodes a home directory: nvm is located from $NVM_DIR
# or the standard XDG/HOME locations at runtime.

atlas_node_env() {
    local want="${NODE_MAJOR_REQUIRED:-22}"

    # Already good? Do nothing.
    if command -v node >/dev/null 2>&1; then
        local have
        have="$(node --version 2>/dev/null | sed 's/^v//' | cut -d. -f1)"
        [ "$have" = "$want" ] && return 0
    fi

    local nvm_sh=""
    for candidate in \
        "${NVM_DIR:-}/nvm.sh" \
        "${XDG_CONFIG_HOME:-$HOME/.config}/nvm/nvm.sh" \
        "$HOME/.nvm/nvm.sh"
    do
        case "$candidate" in /nvm.sh) continue ;; esac
        if [ -s "$candidate" ]; then nvm_sh="$candidate"; break; fi
    done

    [ -n "$nvm_sh" ] || return 0   # no nvm here; the caller's own check reports it

    # nvm.sh is not `set -e` clean, and a failure to select a version is not a
    # failure of the caller — lint.sh already reports a missing/wrong node with
    # a better message than anything this shim could produce.
    # shellcheck disable=SC1090
    . "$nvm_sh" >/dev/null 2>&1 || return 0
    nvm use "$want" >/dev/null 2>&1 || return 0
    return 0
}

atlas_node_env
