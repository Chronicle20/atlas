#!/usr/bin/env bash
# PostToolUse hook — format the file a Write/Edit just touched (task-171), or
# the .go files an in-place Bash edit (sed -i, perl -i, python3, awk, redirect)
# named.
#
# DELIBERATELY FAIL-OPEN (design.md §3.6): a local convenience hook must never
# block an edit. Missing toolchain, missing cached binary, unparseable input,
# tool error — all exit 0 silently. CI (lint-go / lint-ui) is the enforcement
# point. To avoid a multi-minute stall on first Write, the hook never
# bootstraps golangci-lint itself; it uses the binary only if tools/lint.sh
# has already cached it.
set -u

[ -t 0 ] && exit 0

input="$(cat)"
ROOT="${CLAUDE_PROJECT_DIR:-$(pwd)}"

format_one() {
    local fp="$1"
    [ -f "$fp" ] || return 0
    # Fail-open on a non-absolute path: the hook resolves nothing relative to
    # the repo, and dirname-walk on a relative path can spin. First-party
    # Write/Edit always pass an absolute file_path.
    case "$fp" in /*) ;; *) return 0 ;; esac
    case "$fp" in
    *.go)
        # shellcheck source=../../tools/toolchain.versions
        source "$ROOT/tools/toolchain.versions" 2>/dev/null || return 0
        GOLANGCI="$ROOT/.cache/tools/bin/golangci-lint-${GOLANGCI_LINT_VERSION:-}"
        [ -x "$GOLANGCI" ] || return 0
        # Format from the file's own module dir so gofumpt sees its go.mod.
        moddir="$(dirname "$fp")"
        while [ "$moddir" != "/" ] && [ ! -f "$moddir/go.mod" ]; do
            moddir="$(dirname "$moddir")"
        done
        [ -f "$moddir/go.mod" ] || return 0
        (cd "$moddir" && "$GOLANGCI" fmt -c "$ROOT/.golangci.yml" "$fp") >/dev/null 2>&1 || true
        ;;
    */services/atlas-ui/*.ts|*/services/atlas-ui/*.tsx)
        (cd "$ROOT/services/atlas-ui" && npx --no-install prettier --write "$fp") >/dev/null 2>&1 || true
        ;;
    esac
}

fp="$(printf '%s' "$input" | jq -r '.tool_input.file_path // empty' 2>/dev/null)" || exit 0
if [ -n "$fp" ]; then
    format_one "$fp"
    exit 0
fi

# Bash edits bypass Write/Edit. A rebase-conflict session resolved every
# conflict with python3/perl/awk one-liners, never tripped this hook, and paid
# ~22% of its total for a gofmt failure caught only at the verify gate. When a
# Bash command looks like an in-place edit and names .go files, format those.
cmd="$(printf '%s' "$input" | jq -r '.tool_input.command // empty' 2>/dev/null)" || exit 0
[ -n "$cmd" ] || exit 0
printf '%s' "$cmd" | grep -Eq '(sed +-i|perl +-[a-zA-Z]*i|python3?\b|awk\b|>>?|\btee\b|\bpatch\b|\bmv\b|\bcp\b)' || exit 0
cwd="$(printf '%s' "$input" | jq -r '.cwd // empty' 2>/dev/null)"

# Only ever format inside this repo. A Bash command can name an absolute path
# anywhere, and `cwd` can sit outside the checkout, so the Write/Edit path's
# implicit trust does not carry over here. Anything that does not resolve under
# $ROOT is skipped — skipping is the fail-open direction.
root_real="$(realpath -m "$ROOT" 2>/dev/null || printf '%s' "$ROOT")"
n=0
for tok in $(printf '%s' "$cmd" | grep -oE "[A-Za-z0-9_./-]+\.go\b" | sort -u); do
    case "$tok" in /*) p="$tok" ;; *) p="${cwd:-$ROOT}/$tok" ;; esac
    p="$(realpath -m "$p" 2>/dev/null || printf '%s' "$p")"
    case "$p" in
        "$root_real"/*) ;;
        *) continue ;;
    esac
    format_one "$p"
    n=$((n + 1)); [ "$n" -ge 20 ] && break
done

exit 0
