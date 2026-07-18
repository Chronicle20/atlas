#!/usr/bin/env bash
# tools/lint.sh — shared lint & format guard (task-171).
#
# One entry point for both local use (fix mode) and CI (--check mode), so the
# two can never disagree. golangci-lint v2 is the single authority for Go
# formatting (gofumpt + goimports via .golangci.yml `formatters`) and linting
# (`standard` group). atlas-ui uses Prettier + ESLint via its npm scripts.
#
# Formatting is enforced TREE-WIDE. Linter findings are gated to NEW code via
# --new-from-rev (burn-down tracked in docs/TODO.md "Lint burn-down").
#
# golangci-lint runs per-module in WORKSPACE MODE (root go.work active):
# service go.mod files are not standalone-consistent, so GOWORK=off would
# fail type-loading (verified — see docs/tasks/task-171-lint-format-enforcement/context.md).
# The guard never requires `go work sync`.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck source=lint.versions
source "$ROOT/tools/lint.versions"

NODE_MAJOR_REQUIRED=22

usage() {
    cat <<'EOF'
Usage: tools/lint.sh [--check] [--fmt] [--go|--ui] [--base <rev>] [path ...]

  (no flags)    fix mode: rewrite files in place (formatters + lint --fix)
  --check       check mode: mutate nothing; non-zero exit on any violation
  --fmt         formatter layer only (produces the baseline reformat)
  --go / --ui   restrict to one ecosystem (default: both)
  --base <rev>  diff base for the Go *linter* layer (--new-from-rev).
                Default: merge-base of HEAD and origin/main (fallback: main).
                Formatting is never rev-gated — it is enforced tree-wide.
  path ...      restrict Go module discovery to modules under these paths
                (CI passes changed module paths). No paths = whole tree.

Versions are pinned in tools/lint.versions. Exit: 0 clean, 1 violations, 2 usage.
EOF
}

CHECK=0
FMT_ONLY=0
DO_GO=1
DO_UI=1
BASE=""
PATHS=()

while [ $# -gt 0 ]; do
    case "$1" in
        --check) CHECK=1 ;;
        --fmt)   FMT_ONLY=1 ;;
        --go)    DO_UI=0 ;;
        --ui)    DO_GO=0 ;;
        --base)  BASE="${2:?--base requires a revision}"; shift ;;
        -h|--help) usage; exit 0 ;;
        -*) echo "lint.sh: unknown flag: $1" >&2; usage >&2; exit 2 ;;
        *) PATHS+=("$1") ;;
    esac
    shift
done

TOOLS_BIN="$ROOT/.cache/tools/bin"
GOLANGCI="$TOOLS_BIN/golangci-lint-$GOLANGCI_LINT_VERSION"

GO_RC=0
UI_RC=0
FAILED=()

ensure_golangci() {
    [ -x "$GOLANGCI" ] && return 0
    mkdir -p "$TOOLS_BIN"

    # Fast path: download the pinned prebuilt release binary and verify it
    # against the release's published SHA256 checksums. This is ~10s vs the
    # multi-minute `go install` source build — it dominates cold-cache CI time
    # (task-171). Falls back to `go install` when the download path is
    # unavailable (no curl/sha256sum, unknown platform, or offline).
    local ver="${GOLANGCI_LINT_VERSION#v}" os="" arch="" asset url tmp
    case "$(uname -s)" in
        Linux) os=linux ;;
        Darwin) os=darwin ;;
    esac
    case "$(uname -m)" in
        x86_64 | amd64) arch=amd64 ;;
        arm64 | aarch64) arch=arm64 ;;
    esac

    if [ -n "$os" ] && [ -n "$arch" ] \
        && command -v curl >/dev/null 2>&1 && command -v sha256sum >/dev/null 2>&1; then
        asset="golangci-lint-${ver}-${os}-${arch}.tar.gz"
        url="https://github.com/golangci/golangci-lint/releases/download/${GOLANGCI_LINT_VERSION}"
        echo "lint.sh: downloading golangci-lint $GOLANGCI_LINT_VERSION prebuilt ($os-$arch) into $TOOLS_BIN ..."
        tmp="$(mktemp -d)"
        if curl -sSfL "$url/$asset" -o "$tmp/$asset" \
            && curl -sSfL "$url/golangci-lint-${ver}-checksums.txt" -o "$tmp/checksums.txt" \
            && (cd "$tmp" && grep " ${asset}\$" checksums.txt | sha256sum -c - >/dev/null 2>&1) \
            && tar -xzf "$tmp/$asset" -C "$tmp" \
            && mv "$tmp/golangci-lint-${ver}-${os}-${arch}/golangci-lint" "$GOLANGCI"; then
            chmod +x "$GOLANGCI"
            rm -rf "$tmp"
            return 0
        fi
        echo "lint.sh: WARNING — prebuilt download/verify failed; falling back to 'go install' (slower)." >&2
        rm -rf "$tmp"
    fi

    # Fallback: build from source (requires the Go toolchain).
    if ! command -v go >/dev/null 2>&1; then
        echo "lint.sh: ERROR — cannot fetch prebuilt golangci-lint and no go toolchain for the source fallback" >&2
        exit 1
    fi
    echo "lint.sh: installing golangci-lint $GOLANGCI_LINT_VERSION from source into $TOOLS_BIN ..."
    tmp="$(mktemp -d)"
    GOBIN="$tmp" go install "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$GOLANGCI_LINT_VERSION"
    mv "$tmp/golangci-lint" "$GOLANGCI"
    rm -rf "$tmp"
}

resolve_base() {
    if [ -n "$BASE" ]; then
        echo "$BASE"
        return 0
    fi
    git -C "$ROOT" merge-base HEAD origin/main 2>/dev/null && return 0
    git -C "$ROOT" merge-base HEAD main 2>/dev/null && return 0
    return 1
}

discover_modules() {
    if [ "${#PATHS[@]}" -eq 0 ]; then
        find "$ROOT/services" "$ROOT/libs" -name go.mod -not -path '*/node_modules/*' -print0 \
            | xargs -0 -n1 dirname | sort -u
    else
        local p target
        for p in "${PATHS[@]}"; do
            case "$p" in
                /*) target="$p" ;;
                *)  target="$ROOT/${p#./}" ;;
            esac
            find "$target" -name go.mod -not -path '*/node_modules/*' -print0 2>/dev/null \
                | xargs -0 -r -n1 dirname
        done | sort -u
    fi
}

run_go() {
    ensure_golangci
    local base=""
    if [ "$FMT_ONLY" -eq 0 ]; then
        if ! base="$(resolve_base)"; then
            echo "lint.sh: WARNING — cannot resolve a merge base with origin/main or main;" >&2
            echo "lint.sh: WARNING — running the linter UN-GATED (whole-module findings, never fewer)." >&2
            base=""
        fi
    fi

    local moddir rel fmt_out
    while IFS= read -r moddir; do
        rel="${moddir#"$ROOT"/}"

        # ---- formatter layer: tree-wide, never rev-gated -------------------
        if [ "$CHECK" -eq 1 ]; then
            if fmt_out="$(cd "$moddir" && "$GOLANGCI" fmt --diff -c "$ROOT/.golangci.yml" ./... 2>&1)" \
                    && [ -z "$fmt_out" ]; then
                : # clean
            else
                echo "lint.sh: FMT FAIL — $rel"
                printf '%s\n' "$fmt_out" | head -40 || true
                GO_RC=1
                FAILED+=("fmt:$rel")
            fi
        else
            if ! (cd "$moddir" && "$GOLANGCI" fmt -c "$ROOT/.golangci.yml" ./...); then
                echo "lint.sh: FMT ERROR — $rel"
                GO_RC=1
                FAILED+=("fmt:$rel")
            fi
        fi

        # ---- linter layer: rev-gated to new code (design.md §5) ------------
        if [ "$FMT_ONLY" -eq 0 ]; then
            local -a lintargs=(run -c "$ROOT/.golangci.yml")
            if [ "$CHECK" -eq 0 ]; then
                lintargs+=(--fix)
            fi
            if [ -n "$base" ]; then
                lintargs+=(--new-from-rev "$base")
            fi
            if ! (cd "$moddir" && "$GOLANGCI" "${lintargs[@]}" ./...); then
                echo "lint.sh: LINT FAIL — $rel"
                GO_RC=1
                FAILED+=("lint:$rel")
            fi
        fi
    done < <(discover_modules)
}

run_ui() {
    local uidir="$ROOT/services/atlas-ui"
    if ! command -v node >/dev/null 2>&1; then
        echo "lint.sh: ERROR — node not found; atlas-ui checks need Node $NODE_MAJOR_REQUIRED (try: nvm use $NODE_MAJOR_REQUIRED)" >&2
        UI_RC=1
        FAILED+=("ui:node-missing")
        return
    fi
    local major
    major="$(node --version | sed 's/^v//' | cut -d. -f1)"
    if [ "$major" != "$NODE_MAJOR_REQUIRED" ]; then
        echo "lint.sh: ERROR — node v$major found, need v$NODE_MAJOR_REQUIRED (try: nvm use $NODE_MAJOR_REQUIRED)" >&2
        UI_RC=1
        FAILED+=("ui:node-version")
        return
    fi
    if [ ! -d "$uidir/node_modules" ]; then
        echo "lint.sh: bootstrapping atlas-ui dev tooling (npm ci) ..."
        (cd "$uidir" && npm ci)
    fi

    if [ "$CHECK" -eq 1 ]; then
        if ! (cd "$uidir" && npm run format:check); then
            echo "lint.sh: UI FMT FAIL — services/atlas-ui"
            UI_RC=1
            FAILED+=("ui:prettier")
        fi
        if [ "$FMT_ONLY" -eq 0 ]; then
            if ! (cd "$uidir" && npm run lint); then
                echo "lint.sh: UI LINT FAIL — services/atlas-ui"
                UI_RC=1
                FAILED+=("ui:eslint")
            fi
        fi
    else
        if ! (cd "$uidir" && npm run format); then
            UI_RC=1
            FAILED+=("ui:prettier")
        fi
        if [ "$FMT_ONLY" -eq 0 ]; then
            if ! (cd "$uidir" && npm run lint -- --fix); then
                echo "lint.sh: UI LINT FAIL — unfixable findings remain (services/atlas-ui)"
                UI_RC=1
                FAILED+=("ui:eslint")
            fi
        fi
    fi
}

if [ "$DO_GO" -eq 1 ]; then
    run_go
fi
if [ "$DO_UI" -eq 1 ]; then
    run_ui
fi

if [ "$GO_RC" -ne 0 ] || [ "$UI_RC" -ne 0 ]; then
    echo ""
    echo "lint.sh: FAIL — ${#FAILED[@]} failing target(s):"
    printf 'lint.sh:   %s\n' "${FAILED[@]}"
    if [ "$CHECK" -eq 1 ]; then
        echo "lint.sh: run 'tools/lint.sh' (fix mode) locally, then commit the result."
    fi
    exit 1
fi
echo "lint.sh: OK"
