#!/usr/bin/env bash
# gen-lb-ports_test.sh — hermetic regression tests for tools/gen-lb-ports.sh.
# Builds throwaway versions.json + marker-delimited YAML fixtures inside a
# temp dir wired as a git repo (the script resolves paths via
# `git rev-parse --show-toplevel`). Run directly:
#     tools/gen-lb-ports_test.sh
set -euo pipefail

SCRIPT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/gen-lb-ports.sh"
PORTS_LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/services/atlas-pr-bootstrap/scripts/version-ports.sh"
[ -x "$SCRIPT" ] || { echo "FATAL: $SCRIPT not executable" >&2; exit 2; }
[ -f "$PORTS_LIB" ] || { echo "FATAL: $PORTS_LIB missing" >&2; exit 2; }

fails=0
assert_eq() { if [ "$2" = "$3" ]; then echo "ok   - $1"; else echo "FAIL - $1 (want '$2', got '$3')" >&2; fails=$((fails+1)); fi; }
assert_contains() { if printf '%s\n' "$3" | grep -qF -- "$2"; then echo "ok   - $1"; else echo "FAIL - $1 (missing '$2')" >&2; fails=$((fails+1)); fi; }

# Build a throwaway repo that mirrors the real layout the script expects.
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" config user.email t@t.t; git -C "$tmp" config user.name t
mkdir -p "$tmp/deploy/k8s/base" "$tmp/services/atlas-pr-bootstrap/scripts" "$tmp/tools"
cp "$SCRIPT" "$tmp/tools/gen-lb-ports.sh"
cp "$PORTS_LIB" "$tmp/services/atlas-pr-bootstrap/scripts/version-ports.sh"

cat > "$tmp/deploy/k8s/base/versions.json" <<'JSON'
{ "versions": [
  { "region": "gms", "majorVersion": 12, "minorVersion": 1 },
  { "region": "gms", "majorVersion": 83, "minorVersion": 1 }
] }
JSON

write_login_yaml() { cat > "$tmp/deploy/k8s/base/atlas-login.yaml" <<'YAML'
        ports:
        # BEGIN generated:container-ports (tools/gen-lb-ports.sh)
        - containerPort: 9999
        # END generated:container-ports
        - containerPort: 8080
---
  ports:
  # BEGIN generated:service-ports (tools/gen-lb-ports.sh)
  - port: 9999
    targetPort: 9999
    protocol: TCP
    name: atlas-login-stale
  # END generated:service-ports
  loadBalancerIP: 1.2.3.4
YAML
}
# channel fixture: same marker shape (the script keys off file path, see below)
write_channel_yaml() { cat > "$tmp/deploy/k8s/base/atlas-channel.yaml" <<'YAML'
        ports:
        # BEGIN generated:container-ports (tools/gen-lb-ports.sh)
        - containerPort: 7777
        # END generated:container-ports
        - containerPort: 8080
---
  ports:
  # BEGIN generated:service-ports (tools/gen-lb-ports.sh)
  - port: 7777
    targetPort: 7777
    protocol: TCP
    name: atlas-channel-stale
  # END generated:service-ports
  loadBalancerIP: 1.2.3.5
YAML
}

# --- Test 1: generate fills both blocks with derived ports ---
write_login_yaml; write_channel_yaml
( cd "$tmp" && ./tools/gen-lb-ports.sh >/dev/null )
login="$(cat "$tmp/deploy/k8s/base/atlas-login.yaml")"
assert_contains "login container 1200" "- containerPort: 1200" "$login"
assert_contains "login container 8300" "- containerPort: 8300" "$login"
assert_contains "login service name gms-83" "name: atlas-login-gms-83" "$login"
assert_contains "login keeps static 8080" "- containerPort: 8080" "$login"
assert_contains "login keeps loadBalancerIP" "loadBalancerIP: 1.2.3.4" "$login"
chan="$(cat "$tmp/deploy/k8s/base/atlas-channel.yaml")"
assert_contains "channel container 1201" "- containerPort: 1201" "$chan"
assert_contains "channel service name gms-83" "name: atlas-channel-gms-83" "$chan"

# --- Test 2: re-running is a no-op (idempotent / deterministic) ---
before="$(cat "$tmp/deploy/k8s/base/atlas-login.yaml")"
( cd "$tmp" && ./tools/gen-lb-ports.sh >/dev/null )
after="$(cat "$tmp/deploy/k8s/base/atlas-login.yaml")"
assert_eq "second run is byte-identical" "$before" "$after"

# --- Test 3: --check passes when in sync, fails on drift ---
set +e
( cd "$tmp" && ./tools/gen-lb-ports.sh --check >/dev/null 2>&1 ); rc=$?
set -e
assert_eq "--check exit 0 when in sync" "0" "$rc"
# Hand-edit a generated block → drift.
write_login_yaml   # stale 9999 again
set +e
( cd "$tmp" && ./tools/gen-lb-ports.sh --check >/dev/null 2>&1 ); rc=$?
set -e
assert_eq "--check exit 1 on drift" "1" "$rc"

# --- Test 4: duplicate majorVersion is rejected ---
write_login_yaml; write_channel_yaml
cat > "$tmp/deploy/k8s/base/versions.json" <<'JSON'
{ "versions": [
  { "region": "gms", "majorVersion": 83, "minorVersion": 1 },
  { "region": "gms", "majorVersion": 83, "minorVersion": 2 }
] }
JSON
set +e
out="$( cd "$tmp" && ./tools/gen-lb-ports.sh 2>&1 )"; rc=$?
set -e
assert_eq "duplicate major exit 1" "1" "$rc"
assert_contains "duplicate major message" "duplicate majorVersion" "$out"

# --- Test 5: missing markers is rejected ---
cat > "$tmp/deploy/k8s/base/versions.json" <<'JSON'
{ "versions": [ { "region": "gms", "majorVersion": 83, "minorVersion": 1 } ] }
JSON
printf 'ports:\n- containerPort: 1\n' > "$tmp/deploy/k8s/base/atlas-login.yaml"
write_channel_yaml
set +e
out="$( cd "$tmp" && ./tools/gen-lb-ports.sh 2>&1 )"; rc=$?
set -e
assert_eq "missing markers exit 1" "1" "$rc"
assert_contains "missing markers message" "marker" "$out"

# --- Test 5b: BEGIN marker present but its END marker missing is rejected ---
# Silent data-loss path: replace_block sets skip=1 at BEGIN and only clears it
# at END; a missing END drops every line to EOF. Must be caught explicitly.
cat > "$tmp/deploy/k8s/base/versions.json" <<'JSON'
{ "versions": [ { "region": "gms", "majorVersion": 83, "minorVersion": 1 } ] }
JSON
# login YAML with the container-ports END marker deleted (service-ports intact).
cat > "$tmp/deploy/k8s/base/atlas-login.yaml" <<'YAML'
        ports:
        # BEGIN generated:container-ports (tools/gen-lb-ports.sh)
        - containerPort: 9999
        - containerPort: 8080
---
  ports:
  # BEGIN generated:service-ports (tools/gen-lb-ports.sh)
  - port: 9999
    targetPort: 9999
    protocol: TCP
    name: atlas-login-stale
  # END generated:service-ports
  loadBalancerIP: 1.2.3.4
YAML
write_channel_yaml
set +e
out="$( cd "$tmp" && ./tools/gen-lb-ports.sh 2>&1 )"; rc=$?
set -e
assert_eq "missing END marker exit 1" "1" "$rc"
assert_contains "missing END marker message" "marker" "$out"

# --- Test 6: adding a version produces the expected new entry ---
write_login_yaml; write_channel_yaml
cat > "$tmp/deploy/k8s/base/versions.json" <<'JSON'
{ "versions": [
  { "region": "gms", "majorVersion": 83, "minorVersion": 1 },
  { "region": "gms", "majorVersion": 99, "minorVersion": 1 }
] }
JSON
( cd "$tmp" && ./tools/gen-lb-ports.sh >/dev/null )
login="$(cat "$tmp/deploy/k8s/base/atlas-login.yaml")"
assert_contains "added v99 login port" "- containerPort: 9900" "$login"
assert_contains "added v99 service name" "name: atlas-login-gms-99" "$login"

echo; [ "$fails" -eq 0 ] && echo "ALL PASS" || { echo "$fails FAILED" >&2; exit 1; }
