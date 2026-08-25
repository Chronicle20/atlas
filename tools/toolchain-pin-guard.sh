#!/usr/bin/env bash
# tools/toolchain-pin-guard.sh — asserts every pinned copy of the repo's
# Go/Alpine/golangci-lint versions agrees with tools/toolchain.versions.
#
# Why this exists (task-261): the repo pins its Go/Alpine/golangci-lint
# versions in ~110 places that no single format can cross-reference — go.mod
# `go` directives, go.work, Dockerfile ARGs, docker-bake.hcl variables, CI
# workflow env vars, and README.md all carry independent copies. Before this
# guard, a partial Renovate bump left the tree building against three
# different Go versions at once with nothing failing: no build, no test, no
# lint caught the drift because each pin site is syntactically valid on its
# own. tools/toolchain.versions is the single source of truth; this guard is
# the only thing that reads every copy and compares it back.
#
# Run from anywhere; drift → non-zero exit, one `path:line: expected X, got Y`
# line per violation on stdout.
#
# --selftest: proves the guard can actually detect a mutation, against a
# throwaway mktemp copy of the checked files. Mutates nothing under $ROOT.

set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
# shellcheck source=toolchain.versions
source "$ROOT/tools/toolchain.versions"

# FR-7 (task-261): fixture data, NOT build inputs. tools/cideps' tests assert
# against these trees and the version string is incidental to what they
# exercise; bumping them would break `go test ./...` in tools/cideps for no
# gain. Do NOT "fix" these to match the pin file.
#
# Two more fixture strings live outside this list because they are not tracked
# go.mod files and `git ls-files '*go.mod'` never yields them — recorded here
# so a future sweep does not mistake them for misses:
#   tools/cideps/graph_test.go:14   `go 1.25.5` inside a go.mod string literal
#   tools/plan-context_test.sh:50   `go 1.24` written into a temp fixture repo
EXEMPT_GOMOD=(
    tools/cideps/testdata/  # both simple/ and transitive/ trees (8 files)
)

# The six explicit CI sites plus the one added by the task-261 session-2
# amendment (.github/config/services.json), giving seven. Enumerated
# explicitly, never by pattern — .github/** contains many unrelated version
# strings and an over-broad pattern there produces false positives that get
# "fixed" by loosening the guard.
CI_SITES=(
    ".github/workflows/pr-validation.yml|GO_VERSION:"
    ".github/workflows/main-publish.yml|GO_VERSION:"
    ".github/workflows/catalog-lint.yml|GO_VERSION:"
    ".github/workflows/packet-matrix.yml|GO_VERSION:"
    ".github/actions/go-test/action.yml|default:"
    ".github/actions/detect-changes/action.yml|go-version:"
    ".github/config/services.json|\"go_version\":"
)

# Dockerfile:94's synthesized-workspace `printf` is deliberately NOT checked.
# It derives its value from `ARG GO_VERSION` at build time; parsing a printf
# format string inside a line-continued RUN would produce a guard that passes
# while the real image build fails. Do not "fix" this omission.

VIOLATIONS=()
CHECKED_GOMOD=0

add_violation() {
    VIOLATIONS+=("$1")
}

# GO_VERSION with `.` escaped for use inside a regex — never interpolate the
# raw value, or a prefix match like `go 1.27` would slip past a `go 1.27.1`
# tree and pass it as clean.
ver_regex() {
    printf '%s' "${GO_VERSION//./\\.}"
}

gomod_check() {
    local checkdir="$1"
    local ver_esc
    ver_esc="$(ver_regex)"
    CHECKED_GOMOD=0

    while IFS= read -r -d '' rel; do
        local exempt=0
        for prefix in "${EXEMPT_GOMOD[@]}"; do
            case "$rel" in
                "$prefix"*) exempt=1 ;;
            esac
        done
        [ "$exempt" -eq 1 ] && continue

        CHECKED_GOMOD=$((CHECKED_GOMOD + 1))
        local f="$checkdir/$rel"
        local line
        line="$(grep -n '^go ' "$f" | head -1)"
        local lineno="${line%%:*}"
        local content="${line#*:}"
        if ! printf '%s\n' "$content" | grep -qE "^go ${ver_esc}\$"; then
            add_violation "${rel}:${lineno}: expected go ${GO_VERSION}, got ${content}"
        fi
    done < <(git -C "$ROOT" ls-files -z '*go.mod' 'go.mod')
}

workspace_check() {
    local checkdir="$1"
    local ver_esc
    ver_esc="$(ver_regex)"
    local f="$checkdir/go.work"
    local content
    content="$(sed -n '1p' "$f")"
    if ! printf '%s\n' "$content" | grep -qE "^go ${ver_esc}\$"; then
        add_violation "go.work:1: expected go ${GO_VERSION}, got ${content}"
    fi
}

dockerfile_arg_check() {
    local checkdir="$1"
    for f in "Dockerfile" "services/atlas-kafka-precreate/Dockerfile"; do
        local path="$checkdir/$f"
        if [ ! -f "$path" ]; then
            add_violation "$f: expected file to exist, got missing"
            continue
        fi

        local goline
        goline="$(grep -n '^ARG GO_VERSION=' "$path" | head -1 || true)"
        if [ -z "$goline" ]; then
            add_violation "$f: expected 'ARG GO_VERSION=${GO_VERSION}', got no ARG GO_VERSION= line"
        else
            local lineno="${goline%%:*}"
            local content="${goline#*:}"
            if [ "$content" != "ARG GO_VERSION=${GO_VERSION}" ]; then
                add_violation "$f:${lineno}: expected ARG GO_VERSION=${GO_VERSION}, got ${content}"
            fi
        fi

        local alpineline
        alpineline="$(grep -n '^ARG ALPINE_VERSION=' "$path" | head -1 || true)"
        if [ -z "$alpineline" ]; then
            add_violation "$f: expected 'ARG ALPINE_VERSION=${ALPINE_VERSION}', got no ARG ALPINE_VERSION= line"
        else
            local alineno="${alpineline%%:*}"
            local acontent="${alpineline#*:}"
            if [ "$acontent" != "ARG ALPINE_VERSION=${ALPINE_VERSION}" ]; then
                add_violation "$f:${alineno}: expected ARG ALPINE_VERSION=${ALPINE_VERSION}, got ${acontent}"
            fi
        fi
    done
}

bake_check() {
    local checkdir="$1"
    local f="$checkdir/docker-bake.hcl"

    # GO_VERSION block: the `default = "X"` line inside `variable "GO_VERSION" { ... }`.
    local go_default
    go_default="$(awk '/^variable "GO_VERSION" \{/{f=1;next} f && /default *=/{print NR": "$0; exit} f && /\}/{exit}' "$f")"
    local go_lineno="${go_default%%:*}"
    local go_content
    go_content="$(printf '%s' "$go_default" | sed 's/^[0-9]*: //')"
    if [ "$go_content" != "  default = \"${GO_VERSION}\"" ]; then
        add_violation "docker-bake.hcl:${go_lineno}: expected default = \"${GO_VERSION}\", got ${go_content# }"
    fi

    # ALPINE_VERSION block.
    local alpine_default
    alpine_default="$(awk '/^variable "ALPINE_VERSION" \{/{f=1;next} f && /default *=/{print NR": "$0; exit} f && /\}/{exit}' "$f")"
    local alpine_lineno="${alpine_default%%:*}"
    local alpine_content
    alpine_content="$(printf '%s' "$alpine_default" | sed 's/^[0-9]*: //')"
    if [ "$alpine_content" != "  default = \"${ALPINE_VERSION}\"" ]; then
        add_violation "docker-bake.hcl:${alpine_lineno}: expected default = \"${ALPINE_VERSION}\", got ${alpine_content# }"
    fi
}

ci_check() {
    local checkdir="$1"

    for entry in "${CI_SITES[@]}"; do
        local file="${entry%%|*}"
        local key="${entry#*|}"
        local path="$checkdir/$file"

        if [ ! -f "$path" ]; then
            add_violation "$file: expected file to exist, got missing"
            continue
        fi

        case "$file" in
            .github/config/services.json)
                local jline
                jline="$(grep -n '"go_version":' "$path" | head -1)"
                local lineno="${jline%%:*}"
                local val
                val="$(sed -n "${lineno}p" "$path" | sed -E 's/.*"go_version": *"([^"]*)".*/\1/')"
                if [ "$val" != "$GO_VERSION" ]; then
                    add_violation "$file:${lineno}: expected \"go_version\": \"${GO_VERSION}\", got \"go_version\": \"${val}\""
                fi
                ;;
            .github/actions/go-test/action.yml)
                # `default:` appears more than once in this file (race-detection
                # also has one); scope the match to the go-version input's block —
                # the `default: 'X'` line immediately following the `  go-version:`
                # key.
                local gv_line
                gv_line="$(grep -n '^  go-version:' "$path" | head -1)"
                local gv_lineno="${gv_line%%:*}"
                local def_lineno=$((gv_lineno + 3))
                local def_content
                def_content="$(sed -n "${def_lineno}p" "$path")"
                local val
                val="$(printf '%s\n' "$def_content" | sed -E "s/^[[:space:]]*default: *'([^']*)'.*/\1/")"
                if [ "$val" != "$GO_VERSION" ]; then
                    add_violation "$file:${def_lineno}: expected default: '${GO_VERSION}', got ${def_content# }"
                fi
                ;;
            *)
                local kline
                kline="$(grep -n -F "$key" "$path" | head -1 || true)"
                if [ -z "$kline" ]; then
                    add_violation "$file: expected a '${key}' line, got none found"
                    continue
                fi
                local lineno="${kline%%:*}"
                local content="${kline#*:}"
                local val
                val="$(printf '%s\n' "$content" | sed -E "s/.*${key}[[:space:]]*'?\"?([0-9]+\.[0-9]+\.[0-9]+).*/\1/")"
                if [ "$val" != "$GO_VERSION" ]; then
                    add_violation "$file:${lineno}: expected ${key} '${GO_VERSION}', got ${key} '${val}'"
                fi
                ;;
        esac
    done
}

lint_pin_check() {
    if [ -z "${GOLANGCI_LINT_VERSION:-}" ]; then
        add_violation "tools/toolchain.versions: expected GOLANGCI_LINT_VERSION to be set, got empty/unset"
    fi
}

readme_check() {
    local checkdir="$1"
    local ver_esc
    ver_esc="$(ver_regex)"
    local f="$checkdir/README.md"
    local rline
    rline="$(grep -n -E "^\| Go \| ${ver_esc}\+ \|" "$f" || true)"
    if [ -z "$rline" ]; then
        local anyline
        anyline="$(grep -n -E '^\| Go \|' "$f" | head -1 || true)"
        local lineno="${anyline%%:*}"
        local content="${anyline#*:}"
        add_violation "README.md:${lineno:-?}: expected | Go | ${GO_VERSION}+ | ..., got ${content:-no '| Go |' line found}"
    fi
}

run_all_checks() {
    local checkdir="$1"
    VIOLATIONS=()
    CHECKED_GOMOD=0

    gomod_check "$checkdir"

    # Vacuous-pass protection: a guard that enumerated zero files must fail,
    # not pass.
    if [ "$CHECKED_GOMOD" -lt 1 ]; then
        echo "toolchain-pin-guard: FAIL — enumerated 0 go.mod files; the git ls-files glob is broken" >&2
        return 2
    fi

    workspace_check "$checkdir"
    dockerfile_arg_check "$checkdir"
    bake_check "$checkdir"
    ci_check "$checkdir"
    lint_pin_check
    readme_check "$checkdir"

    if [ "${#VIOLATIONS[@]}" -eq 0 ]; then
        echo "toolchain-pin-guard: clean (${CHECKED_GOMOD} go.mod + go.work + 4 Dockerfile ARGs + 2 bake vars + 7 CI pins + README checked)"
        return 0
    fi

    for v in "${VIOLATIONS[@]}"; do
        echo "$v"
    done
    echo "toolchain-pin-guard: violations found (see above)"
    return 1
}

selftest() {
    local tmp
    tmp="$(mktemp -d)"
    trap 'rm -rf "$tmp"' RETURN

    while IFS= read -r f; do
        case "$f" in
            *go.mod|go.work|Dockerfile|services/atlas-kafka-precreate/Dockerfile|docker-bake.hcl|\
.github/workflows/pr-validation.yml|.github/workflows/main-publish.yml|\
.github/workflows/catalog-lint.yml|.github/workflows/packet-matrix.yml|\
.github/actions/go-test/action.yml|.github/actions/detect-changes/action.yml|\
.github/config/services.json|README.md)
                mkdir -p "$tmp/$(dirname "$f")"
                cp "$ROOT/$f" "$tmp/$f"
                ;;
        esac
    done < <(git ls-files)

    # Mutate one copied go.mod to a wrong version.
    local target="$tmp/libs/atlas-retry/go.mod"
    if [ ! -f "$target" ]; then
        target="$(find "$tmp" -name go.mod ! -path '*/tools/cideps/testdata/*' | head -1)"
    fi
    sed -i 's/^go .*/go 1.26.0/' "$target"

    local output
    local exit_code=0
    output="$(run_all_checks "$tmp" 2>&1)" || exit_code=$?

    local ver_esc
    ver_esc="$(ver_regex)"
    if [ "$exit_code" -eq 1 ] && printf '%s\n' "$output" | grep -qE "expected go ${ver_esc}, got go 1\\.26\\.0"; then
        echo "toolchain-pin-guard: selftest PASS"
        return 0
    fi
    echo "toolchain-pin-guard: selftest FAIL — mutation not detected" >&2
    echo "--- selftest run output ---" >&2
    printf '%s\n' "$output" >&2
    return 1
}

if [ "${1:-}" = "--selftest" ]; then
    selftest
    exit $?
fi

run_all_checks "$ROOT"
exit $?
