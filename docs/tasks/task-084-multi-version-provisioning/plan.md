# Multi-Version Tenant Provisioning Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make N game versions coexist durably in one environment by (a) deriving every per-version socket port from `majorVersion` via one shared shell formula, (b) generating the k8s LB/Deployment port set from a single declared version list with a CI drift guard, and (c) reworking `atlas-pr-bootstrap` from a clobbering template-rewrite into an additive, id-keyed `tenants[]` merge.

**Architecture:** Pure shell + JSON + k8s/CI config — **no Go module changes**. One sourced helper (`version-ports.sh`) is the sole port-derivation source, consumed by both the bootstrap image (runtime) and a `tools/gen-lb-ports.sh` generator (build/CI). A declared `deploy/k8s/base/versions.json` drives marker-delimited port blocks in `atlas-{login,channel}.yaml`; CI runs the generator in `--check` mode to fail on drift. The bootstrap's `upsert_service_config` reads the live `services` config and upserts only its canonical tenant entry (keyed by id), preserving every other entry so co-resident versions never get drained.

**Tech Stack:** bash, jq, bats (test), awk, GitHub Actions (`pr-validation.yml`), kustomize base manifests.

---

## Design Deviation (read before starting)

The design (`design.md` §4.1) places the shared port helper at `tools/lib/version-ports.sh` and plans `COPY tools/lib/version-ports.sh` into the bootstrap image, asserting "the build context for the bootstrap image is repo-root (`docker_context: "."`)". **This is factually wrong for this repo.** `docker-bake.hcl:123` pins `target "atlas-pr-bootstrap"` to `context = "services/atlas-pr-bootstrap"` (its Dockerfile uses relative COPYs), and `services/atlas-pr-bootstrap/test/dockerfile_test.bats` enforces `^COPY scripts/<name> /atlas/<name>$`. A `COPY tools/...` line cannot reach outside the service-dir build context.

**Resolution (this plan):** the single shared helper lives at **`services/atlas-pr-bootstrap/scripts/version-ports.sh`**. The bootstrap image COPYs it like every other script (satisfies the bake context + the dockerfile test); `tools/gen-lb-ports.sh` sources it by repo-relative path. This keeps FR-1.2's literal single-definition guarantee (one physical file, both consumers source it) with the least churn — no change to the bake context or the Dockerfile-test COPY convention. The bootstrap's testable merge helpers live in a sibling sourced file `services/atlas-pr-bootstrap/scripts/service-config.sh`.

This is the only intentional departure from `design.md`; everything else follows it.

---

## File Structure

**Created:**
- `services/atlas-pr-bootstrap/scripts/version-ports.sh` — port-derivation helper (FR-1).
- `services/atlas-pr-bootstrap/scripts/service-config.sh` — pure, sourceable merge helpers (`build_login_entry`, `build_channel_entry`, `merge_tenant_entry`) for the bootstrap upsert (FR-2).
- `services/atlas-pr-bootstrap/test/version-ports_test.bats` — helper unit tests.
- `services/atlas-pr-bootstrap/test/service_config_test.bats` — merge unit tests.
- `deploy/k8s/base/versions.json` — declared version set (FR-3.1).
- `deploy/k8s/base/versions.schema.json` — advisory JSON Schema for the above.
- `tools/gen-lb-ports.sh` — LB/Deployment port generator + `--check` drift mode (FR-3.2/3.5).
- `tools/gen-lb-ports_test.sh` — generator regression tests (fixture-based, mirrors `tools/task-numbers_test.sh`).

**Modified:**
- `services/atlas-pr-bootstrap/Dockerfile` — COPY the two new sourced scripts.
- `services/atlas-pr-bootstrap/test/dockerfile_test.bats` — skip the two sourced (non-exec) scripts in the chmod check.
- `services/atlas-pr-bootstrap/scripts/bootstrap.sh` — source the new helpers; rework `upsert_service_config` to additive id-keyed merge; new shape-based call sites.
- `services/atlas-pr-bootstrap/canonical/services/login-service.json` — drop the literal port (`tenants: []`).
- `services/atlas-pr-bootstrap/canonical/services/channel-service.json` — keep the worlds shell; the literal port is now an overwritten placeholder.
- `deploy/k8s/base/atlas-login.yaml`, `deploy/k8s/base/atlas-channel.yaml` — wrap port blocks in generator markers; regenerated to the complete `[12,83,84,87,92,95,185]` set.
- `.github/workflows/pr-validation.yml` — new `gen-lb-ports` drift-check job wired into the final gate.
- `docs/runbooks/ephemeral-pr-deployments.md`, `docs/onboarding.md` — add-a-version workflow + additive-bootstrap guarantee (FR-5).

---

## Task 1: Port-derivation helper (single source of truth) — FR-1

**Files:**
- Create: `services/atlas-pr-bootstrap/scripts/version-ports.sh`
- Test: `services/atlas-pr-bootstrap/test/version-ports_test.bats`

- [ ] **Step 1: Write the failing test**

Create `services/atlas-pr-bootstrap/test/version-ports_test.bats`:

```bash
#!/usr/bin/env bats

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
    # shellcheck source=../scripts/version-ports.sh
    . "$PROJECT_ROOT/scripts/version-ports.sh"
}

@test "derive_login_port: 83 -> 8300" {
    [ "$(derive_login_port 83)" = "8300" ]
}

@test "derive_channel_port: 83 -> 8301" {
    [ "$(derive_channel_port 83)" = "8301" ]
}

@test "derive_login_port: 12 -> 1200 and 185 -> 18500" {
    [ "$(derive_login_port 12)" = "1200" ]
    [ "$(derive_login_port 185)" = "18500" ]
}

@test "derive_channel_port: 12 -> 1201 and 185 -> 18501" {
    [ "$(derive_channel_port 12)" = "1201" ]
    [ "$(derive_channel_port 185)" = "18501" ]
}

@test "derive_login_port: non-integer is rejected" {
    run derive_login_port "8x"
    [ "$status" -ne 0 ]
    [[ "$output" == *"not a non-negative integer"* ]]
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `bats services/atlas-pr-bootstrap/test/version-ports_test.bats`
Expected: FAIL — `version-ports.sh` does not exist (source error).

- [ ] **Step 3: Write minimal implementation**

Create `services/atlas-pr-bootstrap/scripts/version-ports.sh`:

```bash
#!/usr/bin/env bash
# Single source of truth for Atlas per-version socket ports (task-084 FR-1).
#   loginPort(major)   = major * 100
#   channelPort(major) = loginPort(major) + 1
# Derivation is a function of majorVersion ONLY (FR-1.3). Sourced by both
# scripts/bootstrap.sh (runtime, via /atlas/version-ports.sh) and
# tools/gen-lb-ports.sh (build/CI). No port arithmetic is written anywhere
# else in the repo.

derive_login_port() {
    local major="$1"
    if ! printf '%s' "$major" | grep -Eq '^[0-9]+$'; then
        echo "derive_login_port: majorVersion '$major' is not a non-negative integer" >&2
        return 1
    fi
    echo $((major * 100))
}

derive_channel_port() {
    local major="$1"
    local login
    login=$(derive_login_port "$major") || return 1
    echo $((login + 1))
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `bats services/atlas-pr-bootstrap/test/version-ports_test.bats`
Expected: PASS (5 tests).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-pr-bootstrap/scripts/version-ports.sh \
        services/atlas-pr-bootstrap/test/version-ports_test.bats
git commit -m "feat(task-084): add shared version-port derivation helper"
```

---

## Task 2: Ship sourced helpers in the bootstrap image — FR-1.2

This task ships **only** `version-ports.sh` into the image (COPY + dockerfile-test skip). The second sourced helper, `service-config.sh`, is created later in Task 8 and gets its own COPY + skip there, alongside the file itself — so there is never an empty-file commit.

**Files:**
- Modify: `services/atlas-pr-bootstrap/Dockerfile`
- Modify: `services/atlas-pr-bootstrap/test/dockerfile_test.bats`

- [ ] **Step 1: Update the failing Dockerfile test for the sourced helper**

`dockerfile_test.bats` test #1 ("copies every script under scripts/") will now require a COPY for `version-ports.sh` — that part passes once we add the COPY. Test #2 (chmod) will **fail** for `version-ports.sh` because it is sourced, not executed. Add it to the skip list. Edit `services/atlas-pr-bootstrap/test/dockerfile_test.bats`, in the second test's loop, change the lib.sh skip line:

```bash
        [ "$base" = "lib.sh" ] && continue
        [ "$base" = "version-ports.sh" ] && continue
```

- [ ] **Step 2: Run the dockerfile test to verify it now fails on the missing COPY**

Run: `bats services/atlas-pr-bootstrap/test/dockerfile_test.bats`
Expected: FAIL — test #1 reports `Dockerfile missing COPY for: version-ports.sh`.

- [ ] **Step 3: Add the COPY line to the Dockerfile**

In `services/atlas-pr-bootstrap/Dockerfile`, add a COPY for the sourced helper next to `lib.sh` (order matters only for cache; keep it with the other `scripts/` COPYs):

```dockerfile
COPY scripts/lib.sh /atlas/lib.sh
COPY scripts/version-ports.sh /atlas/version-ports.sh
COPY scripts/bootstrap.sh /atlas/bootstrap.sh
```

(Leave the `RUN chmod +x` line unchanged — `version-ports.sh` is sourced, not executed.)

- [ ] **Step 4: Run the dockerfile test to verify it passes**

Run: `bats services/atlas-pr-bootstrap/test/dockerfile_test.bats`
Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-pr-bootstrap/Dockerfile \
        services/atlas-pr-bootstrap/test/dockerfile_test.bats
git commit -m "build(task-084): ship version-ports.sh in the bootstrap image"
```

---

## Task 3: Declared version set + schema — FR-3.1

**Files:**
- Create: `deploy/k8s/base/versions.json`
- Create: `deploy/k8s/base/versions.schema.json`

- [ ] **Step 1: Create the declared version set**

Create `deploy/k8s/base/versions.json` (the complete intended post-#711 set; `region`+`majorVersion` drive the port name, `minorVersion` is carried for clarity and does not affect ports):

```json
{
  "$schema": "./versions.schema.json",
  "description": "Game versions this environment exposes on the login/channel LoadBalancers. Edit this list + run tools/gen-lb-ports.sh to (re)generate the port blocks in atlas-login.yaml/atlas-channel.yaml.",
  "versions": [
    { "region": "gms", "majorVersion": 12,  "minorVersion": 1 },
    { "region": "gms", "majorVersion": 83,  "minorVersion": 1 },
    { "region": "gms", "majorVersion": 84,  "minorVersion": 1 },
    { "region": "gms", "majorVersion": 87,  "minorVersion": 1 },
    { "region": "gms", "majorVersion": 92,  "minorVersion": 1 },
    { "region": "gms", "majorVersion": 95,  "minorVersion": 1 },
    { "region": "jms", "majorVersion": 185, "minorVersion": 1 }
  ]
}
```

- [ ] **Step 2: Create the advisory schema**

Create `deploy/k8s/base/versions.schema.json` (editor assist + documentation; the generator in Task 4 is the enforced guard — there is no JSON-schema validator in CI):

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Atlas declared version set",
  "description": "Versions a cluster exposes on the login/channel LoadBalancers. Consumed by tools/gen-lb-ports.sh.",
  "type": "object",
  "required": ["versions"],
  "properties": {
    "$schema": { "type": "string" },
    "description": { "type": "string" },
    "versions": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "object",
        "required": ["region", "majorVersion", "minorVersion"],
        "additionalProperties": false,
        "properties": {
          "region": { "type": "string", "minLength": 1 },
          "majorVersion": { "type": "integer", "minimum": 1 },
          "minorVersion": { "type": "integer", "minimum": 0 }
        }
      }
    }
  }
}
```

- [ ] **Step 3: Verify the JSON parses**

Run: `jq -e '.versions | length == 7' deploy/k8s/base/versions.json`
Expected: prints `true`, exit 0.

- [ ] **Step 4: Commit**

```bash
git add deploy/k8s/base/versions.json deploy/k8s/base/versions.schema.json
git commit -m "feat(task-084): declare the per-environment game version set"
```

---

## Task 4: LB/Deployment port generator + tests — FR-3.2 / FR-3.5

The generator owns marker-delimited blocks in the two base YAMLs. It uses **two distinct marker labels per file** so each region renders the right shape:
- `container-ports` → Deployment `containerPort:` list (8-space indent).
- `service-ports` → `Service.ports` named entries (2-space indent).

Tests are fixture-based (mirroring `tools/task-numbers_test.sh`): the generator runs against throwaway YAML fixtures, never the real manifests, so the test is hermetic.

**Files:**
- Create: `tools/gen-lb-ports.sh`
- Create: `tools/gen-lb-ports_test.sh`

- [ ] **Step 1: Write the failing test harness**

Create `tools/gen-lb-ports_test.sh`:

```bash
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
assert_contains() { if printf '%s\n' "$3" | grep -qF "$2"; then echo "ok   - $1"; else echo "FAIL - $1 (missing '$2')" >&2; fails=$((fails+1)); fi; }

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
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `chmod +x tools/gen-lb-ports_test.sh && tools/gen-lb-ports_test.sh`
Expected: FAIL — `gen-lb-ports.sh` not executable / missing.

- [ ] **Step 3: Write the generator**

Create `tools/gen-lb-ports.sh`:

```bash
#!/usr/bin/env bash
# Generate the per-version LoadBalancer/Deployment port blocks in
# deploy/k8s/base/atlas-{login,channel}.yaml from the single declared version
# set (deploy/k8s/base/versions.json) and the shared port formula
# (services/atlas-pr-bootstrap/scripts/version-ports.sh). task-084 FR-3.
#
#   gen-lb-ports.sh           rewrite the marker blocks in place
#   gen-lb-ports.sh --check   generate to temp, diff vs checked-in; exit 1 on
#                             drift (the CI guard). No files are modified.
#
# Marker contract (per file): two labelled regions, content untouched outside.
#   # BEGIN generated:container-ports (...)  ...  # END generated:container-ports
#   # BEGIN generated:service-ports (...)    ...  # END generated:service-ports
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
VERSIONS="$REPO_ROOT/deploy/k8s/base/versions.json"
PORTS_LIB="$REPO_ROOT/services/atlas-pr-bootstrap/scripts/version-ports.sh"
LOGIN_YAML="$REPO_ROOT/deploy/k8s/base/atlas-login.yaml"
CHANNEL_YAML="$REPO_ROOT/deploy/k8s/base/atlas-channel.yaml"

# shellcheck source=../services/atlas-pr-bootstrap/scripts/version-ports.sh
. "$PORTS_LIB"

CHECK=0
[ "${1:-}" = "--check" ] && CHECK=1

[ -f "$VERSIONS" ] || { echo "gen-lb-ports: missing $VERSIONS" >&2; exit 1; }

# Reject duplicate majorVersion (Q4 / FR-1.3): same major -> same port -> LB collision.
dupes=$(jq -r '[.versions[].majorVersion] | group_by(.) | map(select(length>1)) | flatten | unique | .[]' "$VERSIONS")
if [ -n "$dupes" ]; then
    echo "gen-lb-ports: duplicate majorVersion(s) in versions.json: $(echo "$dupes" | tr '\n' ' ')" >&2
    exit 1
fi

# Emit "region major" pairs sorted by majorVersion (deterministic order).
versions_sorted() {
    jq -r '.versions | sort_by(.majorVersion) | .[] | "\(.region) \(.majorVersion)"' "$VERSIONS"
}

# Render the Deployment containerPort block for a service. $1=login|channel.
render_container_ports() {
    local svc="$1" region major port
    while read -r region major; do
        [ -z "$region" ] && continue
        if [ "$svc" = login ]; then port=$(derive_login_port "$major"); else port=$(derive_channel_port "$major"); fi
        printf '        - containerPort: %s\n' "$port"
    done < <(versions_sorted)
}

# Render the Service.ports block for a service. $1=login|channel.
render_service_ports() {
    local svc="$1" region major port
    while read -r region major; do
        [ -z "$region" ] && continue
        if [ "$svc" = login ]; then port=$(derive_login_port "$major"); else port=$(derive_channel_port "$major"); fi
        printf '  - port: %s\n' "$port"
        printf '    targetPort: %s\n' "$port"
        printf '    protocol: TCP\n'
        printf '    name: atlas-%s-%s-%s\n' "$svc" "$region" "$major"
    done < <(versions_sorted)
}

# Replace the lines between BEGIN/END of $label in $file with the contents of
# $blockfile (markers preserved). Echoes the rewritten file to stdout.
replace_block() {
    local file="$1" label="$2" blockfile="$3"
    awk -v label="$label" -v blockfile="$blockfile" '
        index($0, "BEGIN generated:" label) { print; while ((getline line < blockfile) > 0) print line; close(blockfile); skip=1; next }
        index($0, "END generated:" label)   { skip=0 }
        skip==1 { next }
        { print }
    ' "$file"
}

# Regenerate one file end-to-end; echoes the new content to stdout.
regen_file() {
    local file="$1" svc="$2" cp sp tmp
    cp="$(mktemp)"; sp="$(mktemp)"; tmp="$(mktemp)"
    render_container_ports "$svc" > "$cp"
    render_service_ports   "$svc" > "$sp"
    replace_block "$file" container-ports "$cp" > "$tmp"
    replace_block "$tmp"  service-ports   "$sp"
    rm -f "$cp" "$sp" "$tmp"
}

process() {
    local file="$1" svc="$2" gen
    [ -f "$file" ] || { echo "gen-lb-ports: missing $file" >&2; exit 1; }
    grep -q "BEGIN generated:container-ports" "$file" || { echo "gen-lb-ports: $file missing container-ports marker" >&2; exit 1; }
    grep -q "BEGIN generated:service-ports"   "$file" || { echo "gen-lb-ports: $file missing service-ports marker" >&2; exit 1; }
    gen="$(regen_file "$file" "$svc")"
    if [ "$CHECK" = 1 ]; then
        if ! diff -u "$file" <(printf '%s\n' "$gen"); then
            echo "gen-lb-ports: $file is stale; run tools/gen-lb-ports.sh and commit" >&2
            exit 1
        fi
    else
        printf '%s\n' "$gen" > "$file"
        echo "gen-lb-ports: wrote $file"
    fi
}

process "$LOGIN_YAML"   login
process "$CHANNEL_YAML" channel
```

- [ ] **Step 4: Make it executable and run the test to verify it passes**

Run: `chmod +x tools/gen-lb-ports.sh && tools/gen-lb-ports_test.sh`
Expected: `ALL PASS`.

- [ ] **Step 5: Commit**

```bash
git add tools/gen-lb-ports.sh tools/gen-lb-ports_test.sh
git commit -m "feat(task-084): add LB/Deployment port generator with drift --check"
```

---

## Task 5: Wrap the real base manifests in generator markers

Adds the BEGIN/END markers around the existing port blocks **without changing port values yet**, so the generator (Task 4) has valid targets. Task 6 then runs the generator to complete/normalize the set.

**Files:**
- Modify: `deploy/k8s/base/atlas-login.yaml`
- Modify: `deploy/k8s/base/atlas-channel.yaml`

- [ ] **Step 1: Add markers to `atlas-login.yaml`**

In the Deployment `ports:` list, wrap the version containerPorts (leave `8080` outside):

```yaml
        ports:
        # BEGIN generated:container-ports (tools/gen-lb-ports.sh — edit deploy/k8s/base/versions.json)
        - containerPort: 1200
        - containerPort: 8300
        - containerPort: 8700
        - containerPort: 9200
        - containerPort: 9500
        - containerPort: 18500
        # END generated:container-ports
        - containerPort: 8080
```

In the Service `ports:` list, wrap all named entries (leave `loadBalancerIP` outside):

```yaml
  ports:
  # BEGIN generated:service-ports (tools/gen-lb-ports.sh — edit deploy/k8s/base/versions.json)
  - port: 1200
    targetPort: 1200
    protocol: TCP
    name: atlas-login-gms-12
  - port: 8300
    targetPort: 8300
    protocol: TCP
    name: atlas-login-gms-83
  - port: 8700
    targetPort: 8700
    protocol: TCP
    name: atlas-login-gms-87
  - port: 9200
    targetPort: 9200
    protocol: TCP
    name: atlas-login-gms-92
  - port: 9500
    targetPort: 9500
    protocol: TCP
    name: atlas-login-gms-95
  - port: 18500
    targetPort: 18500
    protocol: TCP
    name: atlas-login-jms-185
  # END generated:service-ports
  loadBalancerIP: 192.168.23.231
```

- [ ] **Step 2: Add markers to `atlas-channel.yaml`**

Deployment block:

```yaml
        ports:
        # BEGIN generated:container-ports (tools/gen-lb-ports.sh — edit deploy/k8s/base/versions.json)
        - containerPort: 1201
        - containerPort: 8301
        - containerPort: 8701
        - containerPort: 18501
        # END generated:container-ports
        - containerPort: 8080
```

Service block:

```yaml
  ports:
  # BEGIN generated:service-ports (tools/gen-lb-ports.sh — edit deploy/k8s/base/versions.json)
  - port: 1201
    targetPort: 1201
    protocol: TCP
    name: atlas-channel-gms-12
  - port: 8301
    targetPort: 8301
    protocol: TCP
    name: atlas-channel-gms-83
  - port: 8701
    targetPort: 8701
    protocol: TCP
    name: atlas-channel-gms-87
  - port: 18501
    targetPort: 18501
    protocol: TCP
    name: atlas-channel-jms-185
  # END generated:service-ports
  loadBalancerIP: 192.168.23.232
```

- [ ] **Step 3: Sanity-check the manifests still parse as YAML and markers are present**

Run:
```bash
grep -c "BEGIN generated:" deploy/k8s/base/atlas-login.yaml deploy/k8s/base/atlas-channel.yaml
```
Expected: each file reports `2`.

- [ ] **Step 4: Commit**

```bash
git add deploy/k8s/base/atlas-login.yaml deploy/k8s/base/atlas-channel.yaml
git commit -m "chore(task-084): wrap LB port blocks in generator markers"
```

---

## Task 6: Regenerate the manifests to the complete version set — FR-3.3

Per `design.md` §3: the current base manifests are **incomplete** (missing login `8400`; channel `8401, 9201, 9501`), so this generator run is **not** a no-op — it *completes* the set and normalizes whitespace. FR-3.3's "no-op" guarantee is re-anchored to this generated baseline going forward (Task 7's CI check enforces it).

**Files:**
- Modify: `deploy/k8s/base/atlas-login.yaml` (generator output)
- Modify: `deploy/k8s/base/atlas-channel.yaml` (generator output)

- [ ] **Step 1: Run the generator**

Run: `tools/gen-lb-ports.sh`
Expected: prints `gen-lb-ports: wrote .../atlas-login.yaml` and `.../atlas-channel.yaml`.

- [ ] **Step 2: Verify the completed set**

Run:
```bash
grep "containerPort:" deploy/k8s/base/atlas-login.yaml
grep "containerPort:" deploy/k8s/base/atlas-channel.yaml
```
Expected login: `1200, 8300, 8400, 8700, 9200, 9500, 18500, 8080`.
Expected channel: `1201, 8301, 8401, 8701, 9201, 9501, 18501, 8080`.

- [ ] **Step 3: Verify the generator is now idempotent (no-op on re-run)**

Run: `tools/gen-lb-ports.sh --check`
Expected: exit 0, no diff output.

- [ ] **Step 4: Review the diff and commit**

Run: `git --no-pager diff --stat deploy/k8s/base/`
Then:
```bash
git add deploy/k8s/base/atlas-login.yaml deploy/k8s/base/atlas-channel.yaml
git commit -m "fix(task-084): backfill gms-84/92/95 LB ports via generator (#711 intent)"
```

---

## Task 7: CI drift-check job — FR-3.5

**Files:**
- Modify: `.github/workflows/pr-validation.yml`

- [ ] **Step 1: Add the drift-check job**

In `.github/workflows/pr-validation.yml`, add a new job after the `redis-key-guard` job (around line 99), mirroring its structure (no Go setup needed — pure shell + jq, both present on `ubuntu-latest`):

```yaml
  gen-lb-ports:
    name: LB Port Drift Guard
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: LB port manifests match versions.json
        run: ./tools/gen-lb-ports.sh --check
```

- [ ] **Step 2: Wire it into the final gate**

In the `pr-validation-complete` job (line ~462), add `gen-lb-ports` to `needs`:

```yaml
    needs: [detect-changes, test-go-libraries, test-go-services, test-ui, build-docker, update-pr-overlay, redis-key-guard, gen-lb-ports]
```

In the "Check results" step, capture and fail on its result. Add after the `GUARD_RESULT` line (~478):

```bash
          LBPORTS_RESULT="${{ needs.gen-lb-ports.result }}"
```

Add a summary row after the Redis Key Guard row (~487):

```bash
          echo "| LB Port Drift Guard | $LBPORTS_RESULT |" >> $GITHUB_STEP_SUMMARY
```

And extend the failure condition (~492) to include it:

```bash
          if [ "$LIBS_RESULT" == "failure" ] || [ "$SERVICES_RESULT" == "failure" ] || [ "$UI_RESULT" == "failure" ] || [ "$DOCKER_RESULT" == "failure" ] || [ "$OVERLAY_RESULT" == "failure" ] || [ "$GUARD_RESULT" == "failure" ] || [ "$LBPORTS_RESULT" == "failure" ]; then
```

- [ ] **Step 3: Verify the workflow YAML parses and locally re-confirm no drift**

Run:
```bash
python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/pr-validation.yml')); print('yaml ok')"
tools/gen-lb-ports.sh --check && echo "drift check clean"
```
Expected: `yaml ok` then `drift check clean` (exit 0).

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/pr-validation.yml
git commit -m "ci(task-084): fail PR validation on LB port / versions.json drift"
```

---

## Task 8: Additive id-keyed bootstrap merge — FR-2 (helpers + unit tests)

Extract the pure, network-free merge logic into a sourceable `service-config.sh` so it can be unit-tested with bats. Wire it into the image (COPY + dockerfile-test skip) here, alongside the file.

**Files:**
- Create: `services/atlas-pr-bootstrap/scripts/service-config.sh`
- Create: `services/atlas-pr-bootstrap/test/service_config_test.bats`
- Modify: `services/atlas-pr-bootstrap/Dockerfile`
- Modify: `services/atlas-pr-bootstrap/test/dockerfile_test.bats`
- Modify: `services/atlas-pr-bootstrap/canonical/services/login-service.json`
- Modify: `services/atlas-pr-bootstrap/canonical/services/channel-service.json`

- [ ] **Step 1: Write the failing merge unit tests**

Create `services/atlas-pr-bootstrap/test/service_config_test.bats`:

```bash
#!/usr/bin/env bats

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
    # shellcheck source=../scripts/service-config.sh
    . "$PROJECT_ROOT/scripts/service-config.sh"
    export TENANT_ID="11111111-1111-1111-1111-111111111111"
    export MAJOR_VERSION="84"
    export LB_IP="10.0.0.9"
    CHANNEL_TMPL="$PROJECT_ROOT/canonical/services/channel-service.json"
}

@test "build_login_entry: derived port, given id" {
    run build_login_entry
    [ "$status" -eq 0 ]
    [ "$(echo "$output" | jq -r '.id')" = "$TENANT_ID" ]
    [ "$(echo "$output" | jq -r '.port')" = "8400" ]
}

@test "build_channel_entry: derived channel port, id, ipAddress, worlds shell preserved" {
    run build_channel_entry "$CHANNEL_TMPL"
    [ "$status" -eq 0 ]
    [ "$(echo "$output" | jq -r '.id')" = "$TENANT_ID" ]
    [ "$(echo "$output" | jq -r '.ipAddress')" = "$LB_IP" ]
    [ "$(echo "$output" | jq -r '.worlds[0].channels[0].port')" = "8401" ]
    [ "$(echo "$output" | jq -r '.worlds[0].id')" = "0" ]
}

@test "merge_tenant_entry: appends when id absent, preserves foreign entries verbatim" {
    live='{"type":"login-service","tenants":[{"id":"aaaa","port":8300}]}'
    entry='{"id":"bbbb","port":8400}'
    merged="$(printf '%s' "$live" | merge_tenant_entry "$entry")"
    [ "$(echo "$merged" | jq -r '.tenants | length')" = "2" ]
    [ "$(echo "$merged" | jq -r '.tenants[0].id')" = "aaaa" ]
    [ "$(echo "$merged" | jq -r '.tenants[0].port')" = "8300" ]
    [ "$(echo "$merged" | jq -r '.tenants[1].id')" = "bbbb" ]
}

@test "merge_tenant_entry: replaces in place by id, preserving array order" {
    live='{"tenants":[{"id":"aaaa","port":8300},{"id":"bbbb","port":1}]}'
    entry='{"id":"bbbb","port":8400}'
    merged="$(printf '%s' "$live" | merge_tenant_entry "$entry")"
    [ "$(echo "$merged" | jq -r '.tenants | length')" = "2" ]
    [ "$(echo "$merged" | jq -r '.tenants[0].id')" = "aaaa" ]
    [ "$(echo "$merged" | jq -r '.tenants[1].id')" = "bbbb" ]
    [ "$(echo "$merged" | jq -r '.tenants[1].port')" = "8400" ]
}

@test "merge_tenant_entry: preserves a foreign channel entry's ipAddress" {
    live='{"tenants":[{"id":"aaaa","ipAddress":"9.9.9.9","worlds":[]},{"id":"bbbb","ipAddress":"1.1.1.1","worlds":[]}]}'
    entry='{"id":"bbbb","ipAddress":"10.0.0.9","worlds":[]}'
    merged="$(printf '%s' "$live" | merge_tenant_entry "$entry")"
    [ "$(echo "$merged" | jq -r '.tenants[0].ipAddress')" = "9.9.9.9" ]
    [ "$(echo "$merged" | jq -r '.tenants[1].ipAddress')" = "10.0.0.9" ]
}

@test "merge_tenant_entry: idempotent — second merge of same entry is byte-identical" {
    live='{"tenants":[{"id":"aaaa","port":8300}]}'
    entry='{"id":"bbbb","port":8400}'
    once="$(printf '%s' "$live" | merge_tenant_entry "$entry")"
    twice="$(printf '%s' "$once" | merge_tenant_entry "$entry")"
    [ "$once" = "$twice" ]
}

@test "merge_tenant_entry: tenant-agnostic config (no tenants key) is unchanged" {
    live='{"type":"drops-service","tasks":[]}'
    entry='{"id":"bbbb","port":8400}'
    merged="$(printf '%s' "$live" | merge_tenant_entry "$entry")"
    [ "$(echo "$merged" | jq -cS .)" = "$(echo "$live" | jq -cS .)" ]
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `bats services/atlas-pr-bootstrap/test/service_config_test.bats`
Expected: FAIL — `service-config.sh` does not exist (source error).

- [ ] **Step 3: Write `service-config.sh`**

Create `services/atlas-pr-bootstrap/scripts/service-config.sh`:

```bash
#!/usr/bin/env bash
# Pure (network-free) helpers for the additive services-config upsert
# (task-084 FR-2). Sourced by bootstrap.sh and unit-tested directly by
# test/service_config_test.bats. Depends on version-ports.sh and the env
# vars TENANT_ID / MAJOR_VERSION / LB_IP set by the caller.

# Resolve version-ports.sh whether running from the image (/atlas) or from a
# checkout (scripts/ sibling).
_sc_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ -f "$_sc_dir/version-ports.sh" ]; then
    . "$_sc_dir/version-ports.sh"
else
    . /atlas/version-ports.sh
fi

# Echo the login tenant entry {id, port} with the version-derived login port.
build_login_entry() {
    local port
    port=$(derive_login_port "$MAJOR_VERSION") || return 1
    jq -cn --arg id "$TENANT_ID" --argjson port "$port" '{id:$id, port:$port}'
}

# Echo the channel tenant entry from the canonical template's worlds shell,
# with id / ipAddress / the first channel's port overwritten. $1 = template path.
build_channel_entry() {
    local tmpl="$1" port
    port=$(derive_channel_port "$MAJOR_VERSION") || return 1
    jq -c --arg id "$TENANT_ID" --arg ip "$LB_IP" --argjson port "$port" '
        .data.attributes.tenants[0]
        | .id = $id
        | .ipAddress = $ip
        | .worlds[0].channels[0].port = $port
    ' "$tmpl"
}

# Upsert $1 (an entry JSON) into the tenants[] of the attributes JSON read on
# stdin, keyed by .id, preserving order and foreign entries. Tenant-agnostic
# attributes (no "tenants" key) pass through unchanged. Echoes merged attributes.
merge_tenant_entry() {
    local entry="$1"
    jq -c --argjson entry "$entry" '
        if has("tenants") then
          .tenants = (
            if any(.tenants[]?; .id == $entry.id)
            then (.tenants | map(if .id == $entry.id then $entry else . end))
            else (.tenants + [$entry]) end )
        else . end'
}
```

- [ ] **Step 4: Update the canonical templates**

Edit `services/atlas-pr-bootstrap/canonical/services/login-service.json` — replace the `tenants` array (which held the stale `8300`) with an empty list; the entry is now built in-script and set on first-run POST:

```json
      "tenants": []
```

Edit `services/atlas-pr-bootstrap/canonical/services/channel-service.json` — keep the `worlds` shell (`build_channel_entry` reads it) but make the placeholder port obviously non-authoritative (it is always overwritten by the derived value). Change the channel `port` to `0`:

```json
                {
                  "id": 0,
                  "port": 0
                }
```

(Leave `id` and `ipAddress` in the channel template as-is; both are overwritten in-script.)

- [ ] **Step 5: Wire `service-config.sh` into the image**

In `services/atlas-pr-bootstrap/Dockerfile`, add the COPY next to the other sourced helpers:

```dockerfile
COPY scripts/version-ports.sh /atlas/version-ports.sh
COPY scripts/service-config.sh /atlas/service-config.sh
```

In `services/atlas-pr-bootstrap/test/dockerfile_test.bats`, add `service-config.sh` to the chmod skip list (sourced, not executed), next to the `version-ports.sh` line added in Task 2:

```bash
        [ "$base" = "version-ports.sh" ] && continue
        [ "$base" = "service-config.sh" ] && continue
```

- [ ] **Step 6: Run the merge + dockerfile tests to verify they pass**

Run:
```bash
bats services/atlas-pr-bootstrap/test/service_config_test.bats
bats services/atlas-pr-bootstrap/test/dockerfile_test.bats
```
Expected: both PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-pr-bootstrap/scripts/service-config.sh \
        services/atlas-pr-bootstrap/test/service_config_test.bats \
        services/atlas-pr-bootstrap/Dockerfile \
        services/atlas-pr-bootstrap/test/dockerfile_test.bats \
        services/atlas-pr-bootstrap/canonical/services/login-service.json \
        services/atlas-pr-bootstrap/canonical/services/channel-service.json
git commit -m "feat(task-084): additive id-keyed services-config merge helpers"
```

---

## Task 9: Rewire `bootstrap.sh` to use the additive merge — FR-2

Replace the clobbering `upsert_service_config` body and its `rewrite_ip` call convention with the new shape-based, read-merge-write flow. The three call sites stay but switch to a `shape` argument (`login|channel|none`).

**Files:**
- Modify: `services/atlas-pr-bootstrap/scripts/bootstrap.sh`

- [ ] **Step 1: Source the new helpers**

In `bootstrap.sh`, after the existing `. "$(dirname "$0")/lib.sh"` block (line ~19) and the `set -e` restore, add:

```bash
# shellcheck source=version-ports.sh
. "$(dirname "$0")/version-ports.sh"
# shellcheck source=service-config.sh
. "$(dirname "$0")/service-config.sh"
```

- [ ] **Step 2: Replace `upsert_service_config`**

Replace the entire `upsert_service_config() { ... }` function (lines ~272–332) with:

```bash
# Read the live services config, upsert this PR's canonical tenant entry
# (keyed by id), and write back the merged result. Preserves every other
# tenants[] entry so co-resident versions are never drained (task-084 FR-2).
#   $1 = canonical template path
#   $2 = shape: login | channel | none(tenant-agnostic, e.g. drops)
upsert_service_config() {
    local payload_path="$1" shape="$2" svc_id entry
    svc_id=$(jq -r '.data.id' "$payload_path")
    if [ -z "$svc_id" ] || [ "$svc_id" = "null" ]; then
        log error "missing data.id in $payload_path"
        return 1
    fi

    # Build the canonical tenant entry (version-derived port). 'none' shapes
    # (drops-service) have no tenant entry and skip the merge entirely.
    case "$shape" in
        login)   entry=$(build_login_entry) ;;
        channel) entry=$(build_channel_entry "$payload_path") ;;
        none)    entry="" ;;
        *)       log error "upsert_service_config: unknown shape '$shape'"; return 1 ;;
    esac

    local existing
    existing=$(curl -fsS -H 'Accept: application/vnd.api+json' \
        "$ATLAS_UI_BASE/api/configurations/services/$svc_id" 2>/dev/null || true)

    if echo "$existing" | jq -e '.data.id' >/dev/null 2>&1; then
        # PRESENT — merge the canonical entry onto the LIVE attributes.
        local live_attrs new_attrs
        live_attrs=$(echo "$existing" | jq -c '.data.attributes')
        if [ -n "$entry" ]; then
            new_attrs=$(printf '%s' "$live_attrs" | merge_tenant_entry "$entry")
        else
            new_attrs="$live_attrs"
        fi
        # Idempotency guard (FR-2.5): skip the PATCH when nothing changed.
        # Also dodges the atlas-configurations PATCH panic on tenant-agnostic
        # configs (reflect.Value.Set using unaddressable value).
        if [ "$(printf '%s' "$live_attrs" | jq -cS .)" = "$(printf '%s' "$new_attrs" | jq -cS .)" ]; then
            log info "service config $svc_id matches; skipping PATCH"
        else
            log info "service config $svc_id exists; PATCH (merged)"
            local body
            body=$(echo "$existing" | jq -c --argjson a "$new_attrs" '.data.attributes = $a')
            curl -fsS -X PATCH \
                -H 'Accept: application/vnd.api+json' \
                -H 'Content-Type: application/vnd.api+json' \
                -d "$body" \
                "$ATLAS_UI_BASE/api/configurations/services/$svc_id" >/dev/null
        fi
    else
        # ABSENT (first run, FR-2.7) — POST the template with the canonical
        # entry as the sole tenant (none shapes POST the template as-is).
        log info "service config $svc_id absent; POST"
        local body
        if [ -n "$entry" ]; then
            body=$(jq -c --argjson entry "$entry" '.data.attributes.tenants = [$entry]' "$payload_path")
        else
            body=$(cat "$payload_path")
        fi
        curl -fsS -X POST \
            -H 'Accept: application/vnd.api+json' \
            -H 'Content-Type: application/vnd.api+json' \
            -d "$body" \
            "$ATLAS_UI_BASE/api/configurations/services" >/dev/null
    fi
}
```

- [ ] **Step 3: Update the three call sites**

Replace the three calls (lines ~335–342) with the shape-based form:

```bash
# login-service: version-derived {id, port}
upsert_service_config /atlas/canonical/services/login-service.json login

# channel-service: version-derived port + LB_IP, worlds shell from template
upsert_service_config /atlas/canonical/services/channel-service.json channel

# drops-service: tenant-agnostic — no tenants[] entry, POST/merge is a no-op
upsert_service_config /atlas/canonical/services/drops-service.json none
```

- [ ] **Step 4: Verify bootstrap.sh still passes shellcheck and its env-guard tests**

Run:
```bash
bash -n services/atlas-pr-bootstrap/scripts/bootstrap.sh && echo "syntax ok"
bats services/atlas-pr-bootstrap/test/bootstrap_test.bats
```
Expected: `syntax ok`, then the 2 env-guard tests PASS (they exit before any service-config call).

- [ ] **Step 5: Run the full bootstrap test suite**

Run: `bats services/atlas-pr-bootstrap/test/`
Expected: all bats files PASS (bootstrap, service_config, version-ports, dockerfile, lib, cleanup, sweep, reclaim, predelete).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-pr-bootstrap/scripts/bootstrap.sh
git commit -m "feat(task-084): make bootstrap services-config upsert additive and version-derived"
```

---

## Task 10: Build the bootstrap image to validate the COPY/source chain — FR-1.2

`go build`/bats can't catch a missing image COPY. Build the bake target to confirm `version-ports.sh` + `service-config.sh` are present and sourceable at runtime.

**Files:** none (verification only)

- [ ] **Step 1: Bake the image**

Run from the worktree root: `docker buildx bake atlas-pr-bootstrap`
Expected: build succeeds.

- [ ] **Step 2: Confirm both helpers are in the image and source cleanly**

Run:
```bash
docker run --rm --entrypoint /bin/bash atlas-pr-bootstrap:latest -lc \
  '. /atlas/version-ports.sh; . /atlas/service-config.sh; derive_login_port 84; derive_channel_port 84'
```
Expected: prints `8400` then `8401`.

- [ ] **Step 3: Commit (only if Step 1 required a Dockerfile fix)**

If the bake surfaced a missing COPY or path bug, fix it and:
```bash
git add services/atlas-pr-bootstrap/Dockerfile
git commit -m "fix(task-084): correct bootstrap image helper COPY"
```
Otherwise no commit — this task is verification.

---

## Task 11: Operator docs — FR-5

**Files:**
- Modify: `docs/runbooks/ephemeral-pr-deployments.md`
- Modify: `docs/onboarding.md`

- [ ] **Step 1: Add the add-a-version section to the runbook**

Append a new top-level section to `docs/runbooks/ephemeral-pr-deployments.md`:

```markdown
## Adding (or removing) a game version

A version's login/channel ports are derived from its `majorVersion`
(`loginPort = major × 100`, `channelPort = loginPort + 1`) by one shared
formula (`services/atlas-pr-bootstrap/scripts/version-ports.sh`). Two places
that used to be hand-maintained are now generated from a single declared list.

**To expose a new version on the LoadBalancers:**

1. Add the version to `deploy/k8s/base/versions.json`:
   ```json
   { "region": "gms", "majorVersion": 84, "minorVersion": 1 }
   ```
   (Two versions may not share a `majorVersion` — they would collide on the
   same port; the generator rejects this.)
2. Regenerate the manifests:
   ```bash
   tools/gen-lb-ports.sh
   ```
   This rewrites the `# BEGIN/END generated:*` blocks in
   `deploy/k8s/base/atlas-{login,channel}.yaml`. Nothing outside the markers
   changes. CI (`gen-lb-ports --check`) fails any PR where these drift.
3. Commit both the `versions.json` edit and the regenerated manifests, then
   redeploy the base.

The tenant row and its per-tenant configuration still have to exist:
ephemeral envs get them from `atlas-pr-bootstrap`; persistent envs from the
UI Templates → Clone flow. The declared version set only controls **LB
exposure**.

**Additive bootstrap guarantee.** `atlas-pr-bootstrap` now upserts only its
canonical tenant into the live `services` config (keyed by tenant id) and
leaves every other tenant entry untouched. A second version added by hand in
an ephemeral env **survives every bootstrap re-run** — its socket listener
and its per-tenant Kafka consumers are no longer drained. (Previously the
bootstrap rebuilt `tenants[]` from a template and clobbered the second
version, leaving its consumers drained so clients logged in and hung.)

**Coexistence verification (manual repro).** With v83 (canonical) + v84
(hand-added) both present in the `services` config, re-run the bootstrap and
confirm in the login/channel logs:
- `projection.applied op=add` for **both** tenants, and
- **no** `projection.applied op=drain` for v84,
then connect a v84 client and confirm the login handshake completes (no hang).
```

- [ ] **Step 2: Add the version-port note to onboarding**

In `docs/onboarding.md`, under **Step 2 — Per-service configs** (after the
login/channel `tenants` bullet lines, ~line 58), add:

```markdown
> **Ports are version-derived.** Each tenant's `port` is `majorVersion × 100`
> (login) and `+1` (channel) — one formula in
> `services/atlas-pr-bootstrap/scripts/version-ports.sh`, consumed by both the
> bootstrap and the LB-port generator. The set of versions a cluster exposes
> on its LoadBalancers is declared once in `deploy/k8s/base/versions.json`;
> run `tools/gen-lb-ports.sh` to regenerate the k8s port blocks. See
> `docs/runbooks/ephemeral-pr-deployments.md` → "Adding (or removing) a game
> version". The bootstrap upserts only its own tenant into the live `services`
> config, so additional co-resident versions persist across re-runs.
```

- [ ] **Step 3: Commit**

```bash
git add docs/runbooks/ephemeral-pr-deployments.md docs/onboarding.md
git commit -m "docs(task-084): document version-derived ports and additive bootstrap"
```

---

## Task 12: Final verification gate

**Files:** none (verification only)

- [ ] **Step 1: Run every shell test**

Run:
```bash
bats services/atlas-pr-bootstrap/test/
tools/gen-lb-ports_test.sh
```
Expected: all bats PASS; generator test prints `ALL PASS`.

- [ ] **Step 2: Re-confirm no LB drift and clean YAML**

Run:
```bash
tools/gen-lb-ports.sh --check && echo "no drift"
python3 -c "import yaml;[yaml.safe_load_all(open(f)) and None for f in ('deploy/k8s/base/atlas-login.yaml','deploy/k8s/base/atlas-channel.yaml')];print('yaml ok')"
```
Expected: `no drift`, `yaml ok`.

- [ ] **Step 3: Confirm no Go module was touched (acceptance note)**

Run: `git --no-pager diff --name-only main... | grep -E '\.go$|go\.(mod|sum)$' || echo "no Go changes"`
Expected: `no Go changes` — so the Go build/test/bake and redis-key-guard gates are N/A for this branch (the bootstrap image bake in Task 10 is the only image build, already done).

- [ ] **Step 4: Code review**

Per CLAUDE.md "Code Review Before PR", invoke `superpowers:requesting-code-review` before opening a PR. No Go or TS files changed, so the backend/frontend guideline reviewers are N/A; the `plan-adherence-reviewer` verifies every task above landed. Address findings, then proceed to `superpowers:finishing-a-development-branch`.

---

## Requirement → Task Traceability

| PRD FR | Task(s) |
|--------|---------|
| FR-1.1 / 1.2 / 1.3 (single port formula) | 1 (helper), 4 (generator consumes it), 8/9 (bootstrap consumes it) |
| FR-2.1 / 2.2 (read-merge-write, id-keyed) | 8 (`merge_tenant_entry`), 9 (wired in) |
| FR-2.3 (version-derived canonical port) | 8 (`build_login/channel_entry`), 9 |
| FR-2.4 (preserve foreign ipAddress) | 8 (merge test), 9 |
| FR-2.5 (idempotent, no churn) | 8 (idempotency test), 9 (skip-PATCH guard) |
| FR-2.6 (login + channel + drops) | 9 (three shape-based call sites) |
| FR-2.7 (first-run POST) | 8/9 (absent → POST `[entry]`) |
| FR-3.1 (declared version set) | 3 |
| FR-3.2 (build-time generator) | 4 |
| FR-3.3 (completes set; no-op re-anchored) | 5 (markers), 6 (regenerate) |
| FR-3.4 (one-list-edit to add) | 4 + 11 (runbook) |
| FR-3.5 (CI drift guard) | 4 (`--check`), 7 (CI job) |
| FR-4.1 / 4.2 (coexistence, no cross-drain) | 8 (preserve-foreign tests), 9, 11 (manual repro) |
| FR-4.3 (drain only the removed tenant) | unchanged projection; 11 (documented) |
| FR-5.1 / 5.2 (operator workflow + runbook) | 11 |
| §10 build/test gates | 12 |
