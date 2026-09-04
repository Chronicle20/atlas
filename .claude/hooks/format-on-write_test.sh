#!/usr/bin/env bash
# Tests for .claude/hooks/format-on-write.sh.
#
# The hook shells out to a pinned golangci-lint, so these tests stand up a fake
# repo root whose cached "golangci-lint" is a script that logs the file it was
# asked to format. That makes the interesting property directly observable:
# WHICH files the hook decided to touch, rather than just that it exited 0.
#
# The property under test is containment. A Bash command can name an absolute
# path anywhere, and `cwd` can sit outside the checkout, so the Bash path must
# format only files resolving under $CLAUDE_PROJECT_DIR.

set -u

HOOK="$(cd "$(dirname "$0")" && pwd)/format-on-write.sh"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
pass=0
fail=0

# --- fake repo root ---------------------------------------------------------
ROOT="$TMP/repo"
OUTSIDE="$TMP/outside"
mkdir -p "$ROOT/tools" "$ROOT/.cache/tools/bin" "$ROOT/pkg" "$OUTSIDE"
echo 'GOLANGCI_LINT_VERSION=v9.9.9' > "$ROOT/tools/toolchain.versions"
: > "$ROOT/.golangci.yml"
echo 'module example.com/x' > "$ROOT/go.mod"
LOG="$TMP/formatted.log"
cat > "$ROOT/.cache/tools/bin/golangci-lint-v9.9.9" <<EOF
#!/usr/bin/env bash
# args: fmt -c <cfg> <file>
echo "\${@: -1}" >> "$LOG"
EOF
chmod +x "$ROOT/.cache/tools/bin/golangci-lint-v9.9.9"

: > "$ROOT/pkg/in_repo.go"
: > "$OUTSIDE/out_of_repo.go"
mkdir -p "$OUTSIDE/tools" "$OUTSIDE/.cache/tools/bin"

run() {  # run <json>
    : > "$LOG"
    printf '%s' "$1" | CLAUDE_PROJECT_DIR="$ROOT" "$HOOK" >/dev/null 2>&1
}
formatted() { grep -Fq "$1" "$LOG" 2>/dev/null; }

check() {  # check <description> <expect-formatted|expect-skipped> <path>
    local desc="$1" mode="$2" path="$3"
    if [ "$mode" = formatted ]; then
        if formatted "$path"; then pass=$((pass+1)); else fail=$((fail+1)); echo "FAIL ($desc): expected to format $path"; fi
    else
        if formatted "$path"; then fail=$((fail+1)); echo "FAIL ($desc): formatted $path but should not have"; else pass=$((pass+1)); fi
    fi
}

echo "== Write/Edit path is unchanged =="
run "$(jq -nc --arg f "$ROOT/pkg/in_repo.go" '{tool_input:{file_path:$f}}')"
check "write in repo" formatted "$ROOT/pkg/in_repo.go"

echo "== Bash in-place edits format the .go files they name =="
run "$(jq -nc --arg c "sed -i s/a/b/ pkg/in_repo.go" --arg w "$ROOT" '{cwd:$w, tool_input:{command:$c}}')"
check "sed -i relative" formatted "$ROOT/pkg/in_repo.go"

run "$(jq -nc --arg c "python3 - <<EOF
rewrite('pkg/in_repo.go')
EOF" --arg w "$ROOT" '{cwd:$w, tool_input:{command:$c}}')"
check "python3 heredoc" formatted "$ROOT/pkg/in_repo.go"

echo "== containment: nothing outside the repo is ever formatted =="
run "$(jq -nc --arg c "sed -i s/a/b/ $OUTSIDE/out_of_repo.go" --arg w "$ROOT" '{cwd:$w, tool_input:{command:$c}}')"
check "absolute path outside root" skipped "$OUTSIDE/out_of_repo.go"

run "$(jq -nc --arg c "sed -i s/a/b/ out_of_repo.go" --arg w "$OUTSIDE" '{cwd:$w, tool_input:{command:$c}}')"
check "cwd outside root" skipped "$OUTSIDE/out_of_repo.go"

run "$(jq -nc --arg c "sed -i s/a/b/ ../outside/out_of_repo.go" --arg w "$ROOT" '{cwd:$w, tool_input:{command:$c}}')"
check "dot-dot escape" skipped "$OUTSIDE/out_of_repo.go"

echo "== a command naming both formats only the in-repo file =="
run "$(jq -nc --arg c "sed -i s/a/b/ pkg/in_repo.go $OUTSIDE/out_of_repo.go" --arg w "$ROOT" '{cwd:$w, tool_input:{command:$c}}')"
check "mixed, inside" formatted "$ROOT/pkg/in_repo.go"
check "mixed, outside" skipped "$OUTSIDE/out_of_repo.go"

echo "== read-only commands do not trigger formatting =="
run "$(jq -nc --arg c "cat pkg/in_repo.go" --arg w "$ROOT" '{cwd:$w, tool_input:{command:$c}}')"
check "cat is not an edit" skipped "$ROOT/pkg/in_repo.go"
run "$(jq -nc --arg c "go build ./..." --arg w "$ROOT" '{cwd:$w, tool_input:{command:$c}}')"
check "build names no .go file" skipped "$ROOT/pkg/in_repo.go"

echo "== fail-open: exit 0 on every shape =="
for payload in '{}' '{"tool_input":{}}' 'not json' '{"tool_input":{"command":"sed -i s/a/b/ missing.go"}}'; do
    printf '%s' "$payload" | CLAUDE_PROJECT_DIR="$ROOT" "$HOOK" >/dev/null 2>&1
    if [ $? -eq 0 ]; then pass=$((pass+1)); else fail=$((fail+1)); echo "FAIL: non-zero exit on payload: $payload"; fi
done

echo
echo "passed: $pass  failed: $fail"
[ "$fail" -eq 0 ]
