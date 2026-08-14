#!/usr/bin/env bash
# tools/lib/analyzer-guard.sh — shared driver for the go/analysis guards.
#
# Every analyzer guard (rediskeyguard, goroutineguard, outboxguard,
# buffdurationguard) used to do the same three slow things:
#
#   1. rebuild its analyzer into `mktemp -d` on every invocation;
#   2. run it as a standalone singlechecker, which re-parses and re-type-checks
#      every package from source each time — a warm run cost exactly what a
#      cold one did (measured 46.4s then 47.5s back to back);
#   3. walk ~60 module directories strictly sequentially.
#
# This helper fixes all three:
#
#   1. the analyzer is built once into .cache/tools/bin, keyed by a hash of its
#      own source, and reused until that source changes;
#   2. it runs via `go vet -vettool=`, so the go command caches the analyzer's
#      facts and diagnostics per package in GOCACHE and skips unchanged
#      packages entirely;
#   3. modules are analyzed in parallel.
#
# The analyzer itself is unchanged — cmd/<guard>vet and cmd/<guard> wrap the
# same Analyzer value, so what the guard *detects* is identical. Only the
# driver differs.
#
# Usage (from tools/<name>-guard.sh):
#
#     ROOT="$(cd "$(dirname "$0")/.." && pwd)"
#     . "$ROOT/tools/lib/analyzer-guard.sh"
#     GUARD=rediskeyguard
#     SCAN_ROOTS=("$ROOT/services")
#     SELFTEST=0
#     FAIL_MSG=("rediskeyguard: FAIL — ...")
#     analyzer_guard_main
#
# Env:
#   GUARD_JOBS   override the parallelism (default: nproc, capped at 8)
#   GUARD_NOCACHE=1  force a rebuild of the analyzer binary

set -euo pipefail

analyzer_guard_main() {
    : "${GUARD:?analyzer-guard: GUARD must be set}"
    : "${ROOT:?analyzer-guard: ROOT must be set}"

    local src="$ROOT/tools/$GUARD"
    local bindir="$ROOT/.cache/tools/bin"
    mkdir -p "$bindir"

    if [ "${SELFTEST:-0}" -eq 1 ]; then
        echo "self-testing $GUARD..."
        ( cd "$src" && GOWORK=off go test ./... )
    fi

    # Content-keyed binary: rebuild only when the analyzer source changes.
    local hash
    hash="$(find "$src" -name '*.go' -o -name 'go.mod' -o -name 'go.sum' \
        | LC_ALL=C sort | xargs cat 2>/dev/null | sha256sum | cut -c1-16)"
    local bin="$bindir/${GUARD}vet-$hash"

    if [ ! -x "$bin" ] || [ "${GUARD_NOCACHE:-0}" -eq 1 ]; then
        echo "building $GUARD (vettool)..."
        ( cd "$src" && GOWORK=off go build -o "$bin" "./cmd/${GUARD}vet" )
        # drop older builds of this guard so .cache/tools/bin cannot grow forever
        find "$bindir" -maxdepth 1 -name "${GUARD}vet-*" ! -name "$(basename "$bin")" -delete 2>/dev/null || true
    fi

    local jobs="${GUARD_JOBS:-}"
    if [ -z "$jobs" ]; then
        jobs="$(nproc 2>/dev/null || echo 4)"
        [ "$jobs" -gt 8 ] && jobs=8
    fi

    local mods
    mods="$(find "${SCAN_ROOTS[@]}" -name go.mod -not -path '*/node_modules/*' \
        -not -path '*/.worktrees/*' -print0 | xargs -0 -n1 dirname | LC_ALL=C sort -u)"

    local count
    count="$(printf '%s\n' "$mods" | sed '/^$/d' | wc -l)"
    echo "$GUARD: $count module(s), $jobs parallel"

    local rc=0
    if ! printf '%s\n' "$mods" | sed '/^$/d' \
        | VETTOOL="$bin" GUARD_NAME="$GUARD" xargs -P "$jobs" -I{} \
            sh -c 'cd "$1" || exit 1; out="$(go vet -vettool="$VETTOOL" ./... 2>&1)" || { printf "%s: %s\n%s\n" "$GUARD_NAME" "$1" "$out" >&2; exit 1; }' _ {}
    then
        rc=1
    fi

    if [ "$rc" -ne 0 ]; then
        local line
        for line in "${FAIL_MSG[@]}"; do
            echo "$line"
        done
    fi
    return "$rc"
}
