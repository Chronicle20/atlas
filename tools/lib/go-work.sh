# shellcheck shell=bash
# tools/lib/go-work.sh — go.work `use`-entry membership, shared by
# tools/lint.sh, tools/verify.sh, and tools/lib/analyzer-guard.sh.
#
# go.work's `use` entries (both the single-line `use ./path` form and the
# parenthesized `use ( ... )` block form), resolved to absolute directories. A
# module directory under services/ or libs/ that is NOT listed here is a tool
# module, deliberately kept out of the workspace (e.g. libs/atlas-kafka/gen —
# see plan.md:18 for task-276): a tool that walks `go.work`-scoped modules
# cannot type-check a non-member ("directory prefix . does not contain
# modules listed in go.work"). It is verified by its own explicit GOWORK=off
# step instead, so every caller here filters it out of its module walk
# rather than folding it into the workspace sweep.
#
# Fails loudly (non-zero, naming go.work) rather than returning an empty set:
# an unreadable go.work or a parse with zero `use` entries must stop the
# sweep, never silently verify nothing while reporting success.
#
# Source it; do not execute it:
#
#     ROOT="$(cd "$(dirname "$0")/.." && pwd)"
#     . "$ROOT/tools/lib/go-work.sh"
#     workspace="$(workspace_module_dirs "lint.sh")" || exit 1
#
# The caller name is used to prefix this helper's own error messages
# ("lint.sh: ERROR — ..." vs "verify.sh: ERROR — ..."), so each script's
# failures still read as its own rather than a shared library's.

workspace_module_dirs() {
    local caller="${1:?workspace_module_dirs: caller name required}"
    if [ ! -r "$ROOT/go.work" ]; then
        echo "${caller}: ERROR — cannot read $ROOT/go.work" >&2
        return 1
    fi
    local entries
    entries="$(awk '
        /^use \(/ { inuse=1; next }
        inuse && /^\)/ { inuse=0; next }
        inuse {
            gsub(/^[ \t]+|[ \t]+$/, "")
            if ($0 != "") print
            next
        }
        /^use[ \t]+/ {
            line = $0
            sub(/^use[ \t]+/, "", line)
            gsub(/^[ \t]+|[ \t]+$/, "", line)
            if (line != "") print line
        }
    ' "$ROOT/go.work")"
    if [ -z "$entries" ]; then
        echo "${caller}: ERROR — $ROOT/go.work parsed to zero 'use' entries" >&2
        return 1
    fi
    printf '%s\n' "$entries" | while IFS= read -r p; do
        case "$p" in
            /*) printf '%s\n' "$p" ;;
            *)  printf '%s\n' "$ROOT/${p#./}" ;;
        esac
    done | sort -u
}
