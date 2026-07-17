#!/usr/bin/env bash
# PostToolUse hook — format the file a Write/Edit just touched (task-171).
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
fp="$(printf '%s' "$input" | jq -r '.tool_input.file_path // empty' 2>/dev/null)" || exit 0
[ -z "$fp" ] && exit 0
[ -f "$fp" ] || exit 0

ROOT="${CLAUDE_PROJECT_DIR:-$(pwd)}"

case "$fp" in
    *.go)
        # shellcheck source=../../tools/lint.versions
        source "$ROOT/tools/lint.versions" 2>/dev/null || exit 0
        GOLANGCI="$ROOT/.cache/tools/bin/golangci-lint-${GOLANGCI_LINT_VERSION:-}"
        [ -x "$GOLANGCI" ] || exit 0
        # Format from the file's own module dir so gofumpt sees its go.mod.
        moddir="$(dirname "$fp")"
        while [ "$moddir" != "/" ] && [ ! -f "$moddir/go.mod" ]; do
            moddir="$(dirname "$moddir")"
        done
        [ -f "$moddir/go.mod" ] || exit 0
        (cd "$moddir" && "$GOLANGCI" fmt -c "$ROOT/.golangci.yml" "$fp") >/dev/null 2>&1 || true
        ;;
    */services/atlas-ui/*.ts|*/services/atlas-ui/*.tsx)
        (cd "$ROOT/services/atlas-ui" && npx --no-install prettier --write "$fp") >/dev/null 2>&1 || true
        ;;
esac

exit 0
