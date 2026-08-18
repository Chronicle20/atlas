#!/usr/bin/env bash
# tools/shell-guard.sh — syntax + shellcheck gate for the repo's shell tooling.
#
# Why this exists: tools/*.sh had no gate of any kind. verify.sh's only shell
# check was hardcoded to `^tools/task-(resolve|brief)(_test)?\.sh$`, so every
# other script in tools/ — including ones that execute text extracted from a
# file — could change with nothing verifying it. A branch touching three new
# tools/ scripts produced a flagless verify.sh run in which all 14 checks
# skipped and the gate still exited 0.
#
# Checks, per script:
#   1. Parse check with the interpreter its shebang names (bash -n / sh -n).
#   2. shellcheck at severity `error`.
#
# Severity is `error`, not `warning`, deliberately: `error` is clean across the
# tree today, so the gate lands with zero legacy debt and any failure is a real
# regression. `warning` currently reports 19 pre-existing findings; raising the
# bar means fixing those first, which is a separate change.
#
# usage: tools/shell-guard.sh [--require-shellcheck] [file ...]
#
#   No file arguments: check every tools/**/*.sh.
#   --require-shellcheck: fail if shellcheck is absent instead of degrading to
#   a syntax-only run. CI passes this so the gate cannot silently weaken into
#   a check that always passes.

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

require_shellcheck=0
files=()

while [ $# -gt 0 ]; do
    case "$1" in
        --require-shellcheck) require_shellcheck=1; shift ;;
        -h|--help) sed -n '2,26p' "$0"; exit 0 ;;
        -*) echo "shell-guard.sh: unknown option $1" >&2; exit 2 ;;
        *) files+=("$1"); shift ;;
    esac
done

if [ "${#files[@]}" -eq 0 ]; then
    while IFS= read -r f; do files+=("$f"); done < <(
        find tools -name '*.sh' -type f | sort
    )
fi

if [ "${#files[@]}" -eq 0 ]; then
    echo "shell-guard: no shell scripts to check"
    exit 0
fi

have_shellcheck=0
if command -v shellcheck >/dev/null 2>&1; then
    have_shellcheck=1
elif [ "$require_shellcheck" -eq 1 ]; then
    echo "shell-guard: FAIL — shellcheck is not installed and --require-shellcheck was passed." >&2
    echo "  Install it (apt-get install shellcheck / brew install shellcheck) or drop the flag." >&2
    exit 1
fi

failed=0

for f in "${files[@]}"; do
    [ -f "$f" ] || continue

    # Parse with the interpreter the script actually declares. Checking a bash
    # script with `sh -n` produces false failures on arrays and [[ ]].
    interp=sh
    case "$(head -1 "$f")" in
        *bash*) interp=bash ;;
    esac

    if ! out="$("$interp" -n "$f" 2>&1)"; then
        echo "shell-guard: syntax FAIL — $f"
        printf '%s\n' "$out" | sed 's/^/    /'
        failed=$((failed + 1))
        continue
    fi

    if [ "$have_shellcheck" -eq 1 ]; then
        if ! out="$(shellcheck -S error "$f" 2>&1)"; then
            echo "shell-guard: shellcheck FAIL — $f"
            printf '%s\n' "$out" | sed 's/^/    /'
            failed=$((failed + 1))
        fi
    fi
done

if [ "$failed" -ne 0 ]; then
    echo
    echo "shell-guard: $failed of ${#files[@]} script(s) failed."
    exit 1
fi

if [ "$have_shellcheck" -eq 1 ]; then
    echo "shell-guard: ${#files[@]} script(s) OK (syntax + shellcheck -S error)."
else
    echo "shell-guard: ${#files[@]} script(s) OK (syntax only — shellcheck not installed)."
fi
