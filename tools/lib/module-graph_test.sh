#!/usr/bin/env bash
# module-graph_test.sh — unit tests for module_consumers over a synthetic
# workspace, so this suite does not depend on the repo's real module graph
# and does not get slower as the repo grows.
#
# Fixture:
#
#   libs/a       (no requires)
#   libs/b       requires libs/a
#   svc/one      requires libs/b
#   svc/two      requires libs/a
#   svc/three    (no requires, no consumers)
#
# Run directly: tools/lib/module-graph_test.sh
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(mktemp -d)"
trap 'rm -rf "$ROOT"' EXIT

# shellcheck source=./module-graph.sh
. "$HERE/module-graph.sh"

fails=0
assert_eq() {
    if [ "$2" = "$3" ]; then
        echo "ok   - $1"
    else
        echo "FAIL - $1" >&2
        echo "        want: $2" >&2
        echo "        got:  $3" >&2
        fails=$((fails + 1))
    fi
}
assert_not_contains() {
    if printf '%s\n' "$3" | grep -qF -- "$2"; then
        echo "FAIL - $1 (unexpectedly present: '$2' in: $3)" >&2
        fails=$((fails + 1))
    else
        echo "ok   - $1"
    fi
}

mkdir -p "$ROOT/libs/a" "$ROOT/libs/b" "$ROOT/svc/one" "$ROOT/svc/two" "$ROOT/svc/three"

cat >"$ROOT/libs/a/go.mod" <<'EOF'
module example.test/libs/a

go 1.21
EOF

cat >"$ROOT/libs/b/go.mod" <<'EOF'
module example.test/libs/b

go 1.21

require example.test/libs/a v0.0.0
EOF

cat >"$ROOT/svc/one/go.mod" <<'EOF'
module example.test/svc/one

go 1.21

require example.test/libs/b v0.0.0
EOF

cat >"$ROOT/svc/two/go.mod" <<'EOF'
module example.test/svc/two

go 1.21

require example.test/libs/a v0.0.0
EOF

cat >"$ROOT/svc/three/go.mod" <<'EOF'
module example.test/svc/three

go 1.21
EOF

rel() {
    # rel <root> <abs-dir-list-on-stdin> — print each line relative to root
    sed "s|^${1}/||"
}

# --- direct consumer only ---------------------------------------------------

got="$(module_consumers "$ROOT" "$ROOT/libs/b" | rel "$ROOT")"
want="$(printf '%s\n' libs/b svc/one | sort)"
assert_eq "direct consumer only" "$want" "$got"

# --- transitive closure -----------------------------------------------------

got="$(module_consumers "$ROOT" "$ROOT/libs/a" | rel "$ROOT")"
want="$(printf '%s\n' libs/a libs/b svc/one svc/two | sort)"
assert_eq "transitive closure" "$want" "$got"

# --- unrelated module never selected ----------------------------------------

assert_not_contains "unrelated module never selected" "svc/three" "$got"

# --- two changed libs union --------------------------------------------------

got="$(module_consumers "$ROOT" "$ROOT/libs/a" "$ROOT/libs/b" | rel "$ROOT")"
want="$(printf '%s\n' libs/a libs/b svc/one svc/two | sort)"
assert_eq "two changed libs union" "$want" "$got"

# --- a changed dir with no consumers -----------------------------------------

got="$(module_consumers "$ROOT" "$ROOT/svc/three" | rel "$ROOT")"
want="svc/three"
assert_eq "a changed dir with no consumers" "$want" "$got"

# --- a changed dir that is not a module --------------------------------------

mkdir -p "$ROOT/nope"
got="$(module_consumers "$ROOT" "$ROOT/nope"; echo "rc=$?")"
assert_eq "a changed dir that is not a module exits 0" "rc=0" "$(printf '%s\n' "$got" | tail -1)"
assert_not_contains "a changed dir that is not a module is absent from output" \
    "$ROOT/nope" "$got"

# --- a require block, not a single line --------------------------------------

cat >"$ROOT/svc/one/go.mod" <<'EOF'
module example.test/svc/one

go 1.21

require (
	example.test/libs/b v0.0.0
)
EOF

got="$(module_consumers "$ROOT" "$ROOT/libs/b" | rel "$ROOT")"
want="$(printf '%s\n' libs/b svc/one | sort)"
assert_eq "a require block, not a single line" "$want" "$got"

# --- an // indirect require is still an edge ---------------------------------

cat >"$ROOT/svc/two/go.mod" <<'EOF'
module example.test/svc/two

go 1.21

require example.test/libs/a v0.0.0 // indirect
EOF

got="$(module_consumers "$ROOT" "$ROOT/libs/a" | rel "$ROOT")"
want="$(printf '%s\n' libs/a libs/b svc/one svc/two | sort)"
assert_eq "an // indirect require is still an edge" "$want" "$got"

# --- cycle-safe --------------------------------------------------------------

cat >"$ROOT/libs/a/go.mod" <<'EOF'
module example.test/libs/a

go 1.21

require example.test/libs/b v0.0.0
EOF

start="$(date +%s)"
got="$(module_consumers "$ROOT" "$ROOT/libs/a" | rel "$ROOT")"
elapsed=$(( $(date +%s) - start ))
want="$(printf '%s\n' libs/a libs/b svc/one svc/two | sort)"
assert_eq "cycle-safe: terminates with the expected closure" "$want" "$got"
if [ "$elapsed" -ge 10 ]; then
    echo "FAIL - cycle-safe: took ${elapsed}s, looks like it did not terminate" >&2
    fails=$((fails + 1))
fi

echo
if [ "$fails" -eq 0 ]; then
    echo "module-graph_test.sh: all assertions passed"
else
    echo "module-graph_test.sh: $fails failure(s)" >&2
fi
[ "$fails" -eq 0 ]
