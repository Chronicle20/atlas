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
# Point (2) is worth stating plainly, because it is load-bearing and it is the
# thing that silently stopped working in CI: the per-package fact cache lives in
# GOCACHE. A job that starts with a cold GOCACHE re-type-checks every package
# from source and pays full price. `actions/setup-go` restores GOCACHE only when
# it can find a dependency file — see the `cache-dependency-path` in
# .github/workflows/pr-validation.yml, without which the restore silently
# no-ops in a go.work repo that has no root go.sum.
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
# The individual entry points above remain the local/dev interface and the
# single-guard escape hatch. CI drives all four through
# tools/go-analyzer-guards.sh, which builds ONE vettool carrying every analyzer
# and type-checks the tree once instead of four times; the functions below are
# the shared pieces both paths use.
#
# Env:
#   GUARD_JOBS       override the parallelism (default: nproc, capped at 8)
#   GUARD_NOCACHE=1  force a rebuild of the analyzer binary
#   GUARD_MODULES    newline/space-separated module dirs to analyze, replacing
#                    the SCAN_ROOTS discovery walk. Callers that already know
#                    the affected module set (CI's change detection,
#                    tools/verify.sh) pass it here rather than re-analyzing
#                    all 86 modules. Empty/unset means "discover everything".

set -euo pipefail

# analyzer_guard_jobs — parallelism for the module walk.
analyzer_guard_jobs() {
    local jobs="${GUARD_JOBS:-}"
    if [ -z "$jobs" ]; then
        jobs="$(nproc 2>/dev/null || echo 4)"
        [ "$jobs" -gt 8 ] && jobs=8
    fi
    printf '%s\n' "$jobs"
}

# analyzer_guard_hash <src-dir>... — content key over analyzer sources.
#
# Hash NAMES as well as contents (sha256sum prints both), and fold in the
# toolchain version. Digesting concatenated bodies alone would keep the cached
# binary when code merely moves between files in the package, when a file is
# renamed, or when Go itself is upgraded — the guard would then go on enforcing
# the pre-change rules while looking like it ran.
analyzer_guard_hash() {
    { find "$@" \( -name '*.go' -o -name 'go.mod' -o -name 'go.sum' -o -name '*.txt' \) -print0 \
        | LC_ALL=C sort -z | xargs -0 -r sha256sum; go version; } \
        | sha256sum | cut -c1-16
}

# analyzer_guard_build <bin-name> <build-dir> <pkg> <src-dir>...
#
# Builds <pkg> (from <build-dir>, with the workspace off) into
# .cache/tools/bin/<bin-name>-<hash>, where <hash> keys every <src-dir>.
# Rebuilds only when that key moves. Prints the binary path on stdout; all
# progress chatter goes to stderr so the path stays capturable.
analyzer_guard_build() {
    local name="$1" builddir="$2" pkg="$3"
    shift 3

    local bindir="$ROOT/.cache/tools/bin"
    mkdir -p "$bindir"

    local hash bin
    hash="$(analyzer_guard_hash "$@")"
    bin="$bindir/${name}-${hash}"

    if [ ! -x "$bin" ] || [ "${GUARD_NOCACHE:-0}" -eq 1 ]; then
        echo "building ${name} (vettool)..." >&2
        ( cd "$builddir" && GOWORK=off go build -o "$bin" "$pkg" ) >&2
        # drop older builds of this binary so .cache/tools/bin cannot grow forever
        find "$bindir" -maxdepth 1 -name "${name}-*" ! -name "$(basename "$bin")" -delete 2>/dev/null || true
    fi

    printf '%s\n' "$bin"
}

# analyzer_guard_discover <root>... — every Go module directory under <root>s.
#
# Excludes nested worktrees *reachable below* each root — e.g. a run from the
# main repo must not descend into .worktrees/task-xxx/services/... and analyze
# every task branch's modules. That is a statement about nesting, not about
# substring: a plain `-not -path '*/.worktrees/*'` also matches when the ROOT
# itself sits inside .worktrees/ (i.e. this run IS a worktree, the normal case
# for every task in this repo), because the excluded segment then appears in
# the root's own absolute path rather than below it. That produced a run that
# discovered zero modules and returned success having analyzed nothing.
#
# Fixed by matching the exclusion against the path RELATIVE to the root being
# scanned, not the raw absolute path — a `.worktrees` segment in the root
# itself is invisible once stripped, while one appearing below the root still
# excludes.
analyzer_guard_discover() {
    local root
    for root in "$@"; do
        [ -d "$root" ] || continue
        local rootnorm="${root%/}"
        find "$root" -name go.mod -not -path '*/node_modules/*' -print0 \
            | while IFS= read -r -d '' f; do
                local dir rel
                dir="$(dirname "$f")"
                if [ "$dir" = "$rootnorm" ]; then
                    rel=""
                else
                    rel="${dir#"$rootnorm"/}"
                fi
                case "$rel" in
                    .worktrees | .worktrees/* | */.worktrees | */.worktrees/*)
                        continue ;;
                esac
                printf '%s\n' "$dir"
            done
    done | LC_ALL=C sort -u
}

# analyzer_guard_scope <root>... — the module set to analyze under <root>s.
#
# GUARD_MODULES, when set, replaces the discovery walk — but is still filtered
# to the given roots, so a caller passing one repo-wide affected-module list can
# hand the same list to a services-only guard and to a services+libs one and get
# the right subset each time. Filtering (rather than trusting the caller) is
# also what keeps a guard's scope from silently widening: rediskeyguard must not
# start reporting on libs/ just because a caller included libs/ in the list.
#
# Fail CLOSED on a bad path. The intersection legitimately comes back empty when
# the caller's list contains no module under this guard's roots — a services-only
# guard handed a libs-only change set, say. It also comes back empty when every
# path in the list is wrong: a relative path where an absolute one was meant, a
# $GITHUB_WORKSPACE that did not expand, a stale module directory. Those two
# cases are indistinguishable downstream, and the second one reads as "no module
# in scope — nothing to analyze", i.e. a green guard that analyzed nothing.
# Every entry must therefore resolve to a real module directory, whatever root
# it lives under; an entry that does not is an error, not a filter.
analyzer_guard_scope() {
    if [ -z "${GUARD_MODULES:-}" ]; then
        analyzer_guard_discover "$@"
        return
    fi

    # Normalize first: entries may be repo-relative, and the discovered set is
    # absolute. A relative entry that merely *happens* to resolve against the
    # current directory would pass the existence check below and then match
    # nothing in the intersection — the exact silent-pass this guards against.
    local m raw normalized=""
    while IFS= read -r raw; do
        [ -n "$raw" ] || continue
        case "$raw" in
            /*) m="$raw" ;;
            *)  m="$ROOT/$raw" ;;
        esac
        m="${m%/}"
        normalized="$normalized$m"$'\n'
    done < <(printf '%s\n' "$GUARD_MODULES" | tr ' \t' '\n\n' | sed '/^$/d')

    local requested
    requested="$(printf '%s' "$normalized" | LC_ALL=C sort -u)"

    local bad=0
    while IFS= read -r m; do
        [ -n "$m" ] || continue
        if [ ! -f "$m/go.mod" ]; then
            echo "analyzer-guard: GUARD_MODULES entry is not a Go module directory: $m" >&2
            bad=1
        fi
    done <<<"$requested"
    if [ "$bad" -ne 0 ]; then
        echo "analyzer-guard: refusing to run against an unresolvable module list" >&2
        return 1
    fi

    local discovered
    discovered="$(analyzer_guard_discover "$@")"

    # Intersect the caller's list with what actually exists under the roots.
    # Both sides are absolute, sorted, de-duplicated module directories.
    LC_ALL=C comm -12 \
        <(printf '%s\n' "$discovered") \
        <(printf '%s\n' "$requested")
}

# analyzer_guard_vet <bin> <label> — module dirs on stdin.
#
# Runs `go vet -vettool=<bin> ./...` in each module, in parallel. Returns 1 if
# any module reported a diagnostic; the analyzer's own output is echoed to
# stderr under the offending module path.
analyzer_guard_vet() {
    local bin="$1" label="$2"

    local mods
    mods="$(cat | sed '/^$/d')"

    local count
    count="$(printf '%s\n' "$mods" | sed '/^$/d' | wc -l | tr -d ' ')"
    if [ "$count" -eq 0 ]; then
        echo "$label: no module in scope — nothing to analyze"
        return 0
    fi

    local jobs
    jobs="$(analyzer_guard_jobs)"
    echo "$label: $count module(s), $jobs parallel"

    if ! printf '%s\n' "$mods" \
        | VETTOOL="$bin" GUARD_NAME="$label" xargs -P "$jobs" -I{} \
            sh -c 'cd "$1" || exit 1; out="$(go vet -vettool="$VETTOOL" ./... 2>&1)" || { printf "%s: %s\n%s\n" "$GUARD_NAME" "$1" "$out" >&2; exit 1; }' _ {}
    then
        return 1
    fi
    return 0
}

# analyzer_guard_main — the single-guard entry point used by
# tools/<name>-guard.sh.
analyzer_guard_main() {
    : "${GUARD:?analyzer-guard: GUARD must be set}"
    : "${ROOT:?analyzer-guard: ROOT must be set}"

    local src="$ROOT/tools/$GUARD"

    if [ "${SELFTEST:-0}" -eq 1 ]; then
        echo "self-testing $GUARD..."
        ( cd "$src" && GOWORK=off go test ./... )
    fi

    local bin
    bin="$(analyzer_guard_build "${GUARD}vet" "$src" "./cmd/${GUARD}vet" "$src")"

    # Resolve the scope BEFORE analyzing, so an unresolvable module list fails
    # with its own error rather than borrowing the guard's remediation text.
    local scope
    scope="$(analyzer_guard_scope "${SCAN_ROOTS[@]}")" || return 1

    local rc=0
    printf '%s\n' "$scope" | analyzer_guard_vet "$bin" "$GUARD" || rc=1

    if [ "$rc" -ne 0 ]; then
        local line
        for line in "${FAIL_MSG[@]}"; do
            echo "$line"
        done
    fi
    return "$rc"
}
