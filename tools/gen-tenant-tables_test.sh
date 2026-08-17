#!/usr/bin/env bash
# gen-tenant-tables_test.sh — hermetic regression tests for
# tools/gen-tenant-tables.sh. Builds a throwaway git repo in a temp dir with
# a fake query-scope-audit.md and fake deploy/k8s/base/atlas-*.yaml
# manifests (the script resolves paths via `git rev-parse --show-toplevel`,
# same pattern as tools/gen-lb-ports_test.sh — prefer this hermetic shape
# over tools/gen-routes_test.sh, which asserts against live checked-in
# output and is a recorded deferred defect on this branch).
# Run directly: tools/gen-tenant-tables_test.sh
set -euo pipefail

SCRIPT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/gen-tenant-tables.sh"
[ -x "$SCRIPT" ] || { echo "FATAL: $SCRIPT not executable" >&2; exit 2; }

fails=0
assert_eq() { if [ "$2" = "$3" ]; then echo "ok   - $1"; else echo "FAIL - $1 (want '$2', got '$3')" >&2; fails=$((fails+1)); fi; }
assert_contains() { if printf '%s\n' "$3" | grep -qF -- "$2"; then echo "ok   - $1"; else echo "FAIL - $1 (missing '$2')" >&2; fails=$((fails+1)); fi; }
assert_not_contains() { if printf '%s\n' "$3" | grep -qF -- "$2"; then echo "FAIL - $1 (unexpectedly has '$2')" >&2; fails=$((fails+1)); else echo "ok   - $1"; fi; }

tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" config user.email t@t.t; git -C "$tmp" config user.name t
mkdir -p "$tmp/docs/tasks/task-232-sparse-ephemeral-environments" \
         "$tmp/deploy/k8s/base" \
         "$tmp/services/atlas-pr-bootstrap/scripts" \
         "$tmp/tools"
cp "$SCRIPT" "$tmp/tools/gen-tenant-tables.sh"

write_audit() {
    cat > "$tmp/docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md" <<'MD'
# Query-path scope audit (FR-8.1)

Some prose before the table that must not be parsed as rows.

| Service | Table / entity | Plane | Verdict | Evidence (file:line) | Notes |
|---|---|---|---|---|---|
| atlas-account | accounts (`account.Entity`) | Data | SCOPED | ev | notes |
| atlas-account | environments (`env.Entity`) | Control | SCOPED | ev | control rows must never appear |
| atlas-cashshop | accounts / cash wallets (`wallet.Entity`) | Data | SCOPED | ev | alt-descriptor row, one table not two |
| atlas-trades | trade_ledger_entries, trade_ledger_sides, trade_ledger_items (`ledger.Entity`) | Data | SCOPED | ev | comma list splits into 3 tables |
| atlas-noservice | orphan_table (`orphan.Entity`) | Data | SCOPED | ev | no deploy manifest — must be skipped, not guessed |

Trailing prose after the table that must also not be parsed as rows.
MD
}

write_manifests() {
    cat > "$tmp/deploy/k8s/base/atlas-account.yaml" <<'YAML'
        - name: DB_NAME
          value: "atlas-accounts"
YAML
    cat > "$tmp/deploy/k8s/base/atlas-cashshop.yaml" <<'YAML'
        - name: DB_NAME
          value: "atlas-cashshop"
YAML
    cat > "$tmp/deploy/k8s/base/atlas-trades.yaml" <<'YAML'
        - name: DB_NAME
          value: "atlas-trades"
YAML
    # Deliberately no deploy/k8s/base/atlas-noservice.yaml.
}

# --- Test 1: generate produces the expected rows and excludes what it must ---
write_audit; write_manifests
out="$( cd "$tmp" && ./tools/gen-tenant-tables.sh 2>&1 )"
gen="$(cat "$tmp/services/atlas-pr-bootstrap/scripts/tenant-tables.txt")"
assert_contains "account row present"          "atlas-accounts accounts"           "$gen"
assert_not_contains "control-plane row absent" "atlas-accounts environments"       "$gen"
assert_contains "alt-descriptor collapses to one table" "atlas-cashshop accounts"  "$gen"
assert_not_contains "alt-descriptor 2nd half not a table" "cash wallets"           "$gen"
assert_contains "comma-split table 1" "atlas-trades trade_ledger_entries"          "$gen"
assert_contains "comma-split table 2" "atlas-trades trade_ledger_sides"            "$gen"
assert_contains "comma-split table 3" "atlas-trades trade_ledger_items"            "$gen"
assert_not_contains "no manifest -> not guessed" "orphan_table"                    "$gen"
assert_contains "warns about unresolved service" "atlas-noservice"                 "$out"
assert_contains "generated-file marker present" "DO NOT EDIT BY HAND"              "$gen"

# --- Test 2: re-running is a no-op (idempotent / deterministic) ---
before="$(cat "$tmp/services/atlas-pr-bootstrap/scripts/tenant-tables.txt")"
( cd "$tmp" && ./tools/gen-tenant-tables.sh >/dev/null 2>&1 )
after="$(cat "$tmp/services/atlas-pr-bootstrap/scripts/tenant-tables.txt")"
assert_eq "second run is byte-identical" "$before" "$after"

# --- Test 3: --check passes when in sync, fails on drift ---
set +e
( cd "$tmp" && ./tools/gen-tenant-tables.sh --check >/dev/null 2>&1 ); rc=$?
set -e
assert_eq "--check exit 0 when in sync" "0" "$rc"

echo "stale entry that does not match generation" >> "$tmp/services/atlas-pr-bootstrap/scripts/tenant-tables.txt"
set +e
( cd "$tmp" && ./tools/gen-tenant-tables.sh --check >/dev/null 2>&1 ); rc=$?
set -e
assert_eq "--check exit 1 on drift" "1" "$rc"

# --- Test 4: missing audit file is a hard failure ---
rm "$tmp/docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md"
set +e
out="$( cd "$tmp" && ./tools/gen-tenant-tables.sh 2>&1 )"; rc=$?
set -e
assert_eq "missing audit exit 1" "1" "$rc"
assert_contains "missing audit message" "missing" "$out"

echo; [ "$fails" -eq 0 ] && echo "ALL PASS" || { echo "$fails FAILED" >&2; exit 1; }
