#!/usr/bin/env bash
# tools/reactor-seed-lint_test.sh — suite for reactor-seed-lint.sh, and the
# gate's only entry point for the Go tests of tools/reactor-seed-lint and
# tools/reactor-seed-gen. verify.sh's all_modules() walks services/ and
# libs/ only (verify.sh:392-398), so tools/ modules are not in the go
# build/vet/test sweep; changed_tool_suites() (verify.sh:225-243) runs this
# file whenever reactor-seed-lint.sh changes, which is where they belong.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
fail() { echo "FAIL: $*" >&2; exit 1; }

# 1. The tools' own Go tests.
(cd tools/reactor-seed-lint && go test ./...) || fail "reactor-seed-lint go test"
if [ -d tools/reactor-seed-gen ]; then
    (cd tools/reactor-seed-gen && go test ./...) || fail "reactor-seed-gen go test"
fi

# 2. The real corpus passes.
./tools/reactor-seed-lint.sh >/dev/null || fail "reactor-seed-lint.sh over deploy/seed"

# 3. A missing seed root is an error, not a silent pass.
if ./tools/reactor-seed-lint.sh no/such/dir >/dev/null 2>&1; then
    fail "missing seed root should exit non-zero"
fi

echo "reactor-seed-lint_test.sh: OK"
