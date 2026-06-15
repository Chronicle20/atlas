# Baseline-Only Ephemeral Bootstrap Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `atlas-pr-bootstrap` provision ephemeral PR data only by restoring a published canonical baseline, deleting the `full`-mode WZ-ingest path and adding an early fail-fast preflight that blocks bring-up when no baseline exists for the env's version.

**Architecture:** Pure deletion-plus-guard across six artifacts — `bootstrap.sh` (remove `BOOTSTRAP_MODE`/`WZ_CANONICAL`/`resolve_mode`/`full` branch, add a read-only baseline preflight that runs before any data-affecting work, simplify the data step to restore-only), `lib.sh` (cosmetic comment fix so the grep gate is clean), a bats test file (fail-fast + MinIO-unreachable coverage via `PATH`-shim `curl`/`kubectl`), `sync-bootstrap.yaml` (drop the `fetch-wz-canonical` init container, the `/opt/wz` mount, and its `emptyDir` volume), and two runbooks. No Go code, no API, no schema changes.

**Tech Stack:** Bash (`set -euo pipefail`), `jq`, `curl`, `kubectl`; bats + shellcheck for verification; Kustomize for the `pr` overlay; Markdown runbooks. `atlas-pr-bootstrap` has no `go.mod`, so the CLAUDE.md Go bake/test rules do not apply.

---

## File Structure

| File | Responsibility after change |
|---|---|
| `services/atlas-pr-bootstrap/scripts/bootstrap.sh` | Baseline-only provisioning. Early `preflight_baseline` (canonical-version, both objects, transient-aware), restore-only data step, no mode switch, no `/api/data/wz` probe. |
| `services/atlas-pr-bootstrap/scripts/lib.sh` | Shared helpers. `log()` doc comment no longer names the deleted `resolve_mode` (keeps the acceptance grep clean). |
| `services/atlas-pr-bootstrap/test/bootstrap_test.bats` | Existing `require_env` tests + new fail-fast-on-absent-baseline and MinIO-unreachable-is-distinct tests, driven by `PATH`-shim `curl`/`kubectl` and a fixture `tenant.json`. |
| `deploy/k8s/overlays/pr/sync-bootstrap.yaml` | Bootstrap Job with no init container, no `/opt/wz` mount, no `wz-canonical` volume. |
| `docs/runbooks/ephemeral-pr-deployments.md` | §9.1 rewritten to baseline-only provisioning; no `atlas.zip`/`BOOTSTRAP_MODE`/init-container; links the migration runbook. |
| `docs/runbooks/canonical-version-migration.md` | Note that bootstrap is baseline-only and publishing a baseline (step 4) is a per-version prerequisite, not an optimization. |

## Conventions used in every task

- All paths are relative to the worktree root (`<repo-root>/.worktrees/task-098-baseline-only-bootstrap`).
- Before each commit, confirm you are on the right branch in the right worktree:

```bash
git rev-parse --show-toplevel   # must end with /.worktrees/task-098-baseline-only-bootstrap
git branch --show-current       # must be task-098-baseline-only-bootstrap
```

- Run bats from the service dir: `cd services/atlas-pr-bootstrap && bats test/`.
- Run shellcheck from the worktree root: `shellcheck services/atlas-pr-bootstrap/scripts/bootstrap.sh services/atlas-pr-bootstrap/scripts/lib.sh`.

---

## Task 1: Failing tests for the baseline preflight

The preflight is the new behavior; cover it first. The tests run the **real** `bootstrap.sh` with a doctored `PATH` (mirrors the existing tests), a fixture canonical `tenant.json`, and fast probe timing so the run is sub-second once the preflight exists. Each new test wraps the script in `timeout` so that — before the preflight exists — the run is bounded (it would otherwise hang in `wait-ready`) and the assertion fails cleanly.

**Files:**
- Modify: `services/atlas-pr-bootstrap/test/bootstrap_test.bats`

- [ ] **Step 1: Add the fixture + shim helpers and the two new tests**

Append to `services/atlas-pr-bootstrap/test/bootstrap_test.bats` (keep the existing `setup()` and two `require_env` tests as-is):

```bash
# --- task-098: baseline preflight ---------------------------------------

# Common env for a script run that should reach the preflight. TENANT_ID
# must be UUID-shaped or the earlier shape-check exits first.
prq_env() {
    echo ATLAS_ENV=test
    echo ATLAS_UI_BASE=http://atlas-ingress.test.svc.cluster.local
    echo TENANT_ID=00000000-0000-0000-0000-000000000001
    echo REGION=GMS
    echo MAJOR_VERSION=83
    echo MINOR_VERSION=1
    echo MINIO_ENDPOINT=http://minio.test:9000
    echo MINIO_PROBE_RETRIES=1
    echo MINIO_PROBE_SLEEP=0
    echo "CANONICAL_TENANT_JSON=$BATS_TEST_TMPDIR/tenant.json"
}

# Build a PATH dir containing a curl shim (emits $1 for every HEAD probe)
# and a kubectl shim (touches a sentinel so we can prove it never ran).
# Real jq is symlinked through so the script can still parse tenant.json.
make_shims() {
    local curl_code="$1"
    local dir="$BATS_TEST_TMPDIR/bin"
    mkdir -p "$dir"

    cat >"$dir/curl" <<EOF
#!/usr/bin/env bash
# baseline_object_status calls: curl -sS -o /dev/null -w '%{http_code}' -I <url>
if [ "$curl_code" = "000" ]; then echo 000; exit 7; fi
echo "$curl_code"
EOF

    cat >"$dir/kubectl" <<EOF
#!/usr/bin/env bash
touch "$BATS_TEST_TMPDIR/kubectl-ran"
exit 0
EOF

    ln -sf "$(command -v jq)" "$dir/jq"
    chmod +x "$dir/curl" "$dir/kubectl"
    echo "$dir"
}

write_fixture_tenant() {
    cat >"$BATS_TEST_TMPDIR/tenant.json" <<'EOF'
{"data":{"attributes":{"region":"GMS","majorVersion":83,"minorVersion":1}}}
EOF
}

@test "bootstrap.sh fails fast when no canonical baseline (404)" {
    command -v jq >/dev/null || skip "jq required"
    command -v timeout >/dev/null || skip "timeout required"
    write_fixture_tenant
    local bindir; bindir="$(make_shims 404)"
    run timeout 15 env $(prq_env) PATH="$bindir:$PATH" \
        bash "$PROJECT_ROOT/scripts/bootstrap.sh"
    [ "$status" -ne 0 ]
    [[ "$output" == *"no canonical baseline"* ]]
    [[ "$output" == *"83.1"* ]]
    [[ "$output" == *"canonical-version-migration"* ]]
    # Preflight must run BEFORE any cluster mutation.
    [ ! -f "$BATS_TEST_TMPDIR/kubectl-ran" ]
}

@test "bootstrap.sh reports MinIO-unreachable distinctly (000)" {
    command -v jq >/dev/null || skip "jq required"
    command -v timeout >/dev/null || skip "timeout required"
    write_fixture_tenant
    local bindir; bindir="$(make_shims 000)"
    run timeout 15 env $(prq_env) PATH="$bindir:$PATH" \
        bash "$PROJECT_ROOT/scripts/bootstrap.sh"
    [ "$status" -ne 0 ]
    [[ "$output" == *"MinIO unreachable"* ]]
    [[ "$output" != *"no canonical baseline"* ]]
    [ ! -f "$BATS_TEST_TMPDIR/kubectl-ran" ]
}
```

- [ ] **Step 2: Run the new tests and verify they FAIL**

Run: `cd services/atlas-pr-bootstrap && bats test/bootstrap_test.bats`
Expected: the two existing `require_env` tests PASS; the two new tests FAIL. Because `preflight_baseline` does not exist yet, the unmodified script skips straight to `wait-ready`, where the `curl` shim never returns 200, so `timeout` kills the run at 15s — the output contains neither `"no canonical baseline"` nor `"MinIO unreachable"`, so the assertions fail. (A ~15s hang per new test here is expected and one-time.)

- [ ] **Step 3: Commit the failing tests**

```bash
git add services/atlas-pr-bootstrap/test/bootstrap_test.bats
git commit -m "test(atlas-pr-bootstrap): cover baseline preflight fail-fast and MinIO-unreachable"
```

---

## Task 2: Implement the baseline preflight in bootstrap.sh

Add the canonical-tenant-path variable, the probe primitives, and the `preflight_baseline` gate, and wire it in immediately after the TENANT_ID shape check (before `wait-ready`). This task makes Task 1's tests pass. It does **not** yet delete `resolve_mode`/`full` (Task 3) — both can coexist for one commit.

**Files:**
- Modify: `services/atlas-pr-bootstrap/scripts/bootstrap.sh`

- [ ] **Step 1: Add `CANONICAL_TENANT_JSON` and probe-timing seams next to the other env defaults**

Find (lines ~31-33):

```bash
WZ_CANONICAL="${WZ_CANONICAL:-/opt/wz/atlas.zip}"
BOOTSTRAP_MODE="${BOOTSTRAP_MODE:-auto}"
MINIO_ENDPOINT="${MINIO_ENDPOINT:-http://minio.minio.svc.cluster.local:9000}"
```

Replace with:

```bash
WZ_CANONICAL="${WZ_CANONICAL:-/opt/wz/atlas.zip}"
BOOTSTRAP_MODE="${BOOTSTRAP_MODE:-auto}"
MINIO_ENDPOINT="${MINIO_ENDPOINT:-http://minio.minio.svc.cluster.local:9000}"
# Canonical tenant descriptor baked into the image. Single source of truth
# for the (region, major, minor) the baseline preflight probes and the
# tenant-create step uses. Overridable so bats can point at a fixture.
CANONICAL_TENANT_JSON="${CANONICAL_TENANT_JSON:-/atlas/canonical/tenant.json}"
# Baseline-probe retry budget. Only transient (000) connection failures are
# retried; a 404 is decisive. Test seam: bats sets these to 1/0.
MINIO_PROBE_RETRIES="${MINIO_PROBE_RETRIES:-5}"
MINIO_PROBE_SLEEP="${MINIO_PROBE_SLEEP:-5}"
```

(`WZ_CANONICAL`/`BOOTSTRAP_MODE` stay for now; Task 3 deletes them.)

- [ ] **Step 2: Add the probe primitives and `preflight_baseline` right after `canonical_baseline_exists` (after line ~132)**

Insert this block immediately after the closing `}` of `canonical_baseline_exists()`:

```bash
# baseline_object_status <url> → echo the HTTP status of an anonymous HEAD
# (e.g. 200/404); echo 000 on a connection-level failure. Anonymous read is
# enabled on the atlas-canonical bucket, so no credentials are needed.
baseline_object_status() {
    local url="$1" code
    code=$(curl -sS -o /dev/null -w '%{http_code}' -I "$url" 2>/dev/null) || code=000
    printf '%s' "$code"
}

# baseline_reachable <url> — retry()-friendly predicate. Sets the global
# BASELINE_PROBE_CODE to the HTTP status and returns 0 whenever MinIO
# answered at all (even 404); returns 1 ONLY on a 000 connection failure so
# retry() rides out a cold-start MinIO blip without masking a real 404.
BASELINE_PROBE_CODE=""
baseline_reachable() {
    BASELINE_PROBE_CODE=$(baseline_object_status "$1")
    [ "$BASELINE_PROBE_CODE" != "000" ]
}

# probe_baseline_object <url> — drive baseline_reachable through retry(). On
# success leaves the HTTP code in BASELINE_PROBE_CODE. If MinIO stays
# unreachable through the retry budget, log a DISTINCT "unreachable" error
# and exit non-zero (do NOT tell the operator to publish a baseline that may
# already exist). Called directly (never in $()) so its exit halts the script.
probe_baseline_object() {
    local url="$1"
    if ! retry "$MINIO_PROBE_RETRIES" "$MINIO_PROBE_SLEEP" baseline_reachable "$url"; then
        log error "MinIO unreachable at $MINIO_ENDPOINT ($url) — cannot verify canonical baseline; check MinIO, do not assume the baseline is missing"
        exit 1
    fi
}

# preflight_baseline — hard-gate the bootstrap on a published canonical
# baseline BEFORE any data-affecting work. Reads (region, major, minor) from
# CANONICAL_TENANT_JSON so the probe targets exactly the version the later
# restore requests (not the initial env-injected values). HEADs BOTH the
# documents.dump.sha256 sidecar AND the documents.dump object, so a
# half-published baseline fails here rather than breaking the restore later.
preflight_baseline() {
    local region major minor base sha_code dump_code
    region=$(jq -r '.data.attributes.region' "$CANONICAL_TENANT_JSON")
    major=$(jq -r '.data.attributes.majorVersion' "$CANONICAL_TENANT_JSON")
    minor=$(jq -r '.data.attributes.minorVersion' "$CANONICAL_TENANT_JSON")
    base="$MINIO_ENDPOINT/atlas-canonical/baseline/regions/$region/versions/$major.$minor"

    probe_baseline_object "$base/documents.dump.sha256"
    sha_code="$BASELINE_PROBE_CODE"
    probe_baseline_object "$base/documents.dump"
    dump_code="$BASELINE_PROBE_CODE"

    if [ "$sha_code" = "200" ] && [ "$dump_code" = "200" ]; then
        log info "canonical baseline present for $region $major.$minor"
        return 0
    fi
    log error "no canonical baseline for $region $major.$minor (documents.dump.sha256=$sha_code documents.dump=$dump_code) — publish one (see docs/runbooks/canonical-version-migration.md) before deploying this env"
    exit 1
}
```

- [ ] **Step 3: Wire the preflight in right after the TENANT_ID shape check (after line ~45)**

Find the closing `fi` of the TENANT_ID UUID-shape guard:

```bash
    log error "TENANT_ID '$TENANT_ID' is not UUID-shaped; tenant-aware probes will 400. Fix Phase 7's Helm chart to inject a UUID."
    exit 1
fi
```

Insert immediately after that `fi`:

```bash

# Fail fast, before any data-affecting work (tenant create, config clone,
# restarts, restore), when no canonical baseline exists for this version.
# A read-only MinIO probe with no dependency on atlas-data being up.
ATLAS_STEP=preflight-baseline preflight_baseline
```

- [ ] **Step 4: Run the bats suite and verify all four tests PASS**

Run: `cd services/atlas-pr-bootstrap && bats test/bootstrap_test.bats`
Expected: 4 passing — the two `require_env` tests, plus both new tests now exit sub-second via the preflight (the 404 test prints `"no canonical baseline … 83.1 … canonical-version-migration"`, the 000 test prints `"MinIO unreachable"`, and neither creates the `kubectl-ran` sentinel).

- [ ] **Step 5: shellcheck the script**

Run: `shellcheck services/atlas-pr-bootstrap/scripts/bootstrap.sh services/atlas-pr-bootstrap/scripts/lib.sh`
Expected: no output (clean). `BASELINE_PROBE_CODE` is intentionally a global set in one function and read in another; if shellcheck flags SC2034 on it, that is a false positive for this cross-function pattern — leave it (matches `DATA_PROCESSING_*`, which shellcheck does not flag because they are read in the same file).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-pr-bootstrap/scripts/bootstrap.sh
git commit -m "feat(atlas-pr-bootstrap): add fail-fast canonical baseline preflight"
```

---

## Task 3: Remove the full-mode path and simplify the data step to restore-only

With the preflight in place, delete the mode switch, the `full` branch, the `WZ_CANONICAL`/`BOOTSTRAP_MODE`/`resolve_mode` machinery, the now-dead `canonical_baseline_exists` helper (its job is the preflight's), and the `/api/data/wz` wait-ready probe. Refresh the header comment.

**Files:**
- Modify: `services/atlas-pr-bootstrap/scripts/bootstrap.sh`

- [ ] **Step 1: Rewrite the header doc comment (lines ~1-14)**

Find:

```bash
#!/usr/bin/env bash
# Atlas PR-env bootstrap (task-071: MinIO-backed ingest). Idempotent —
# short-circuits each step that is already complete. Reads:
#   ATLAS_ENV          — env hash, REQUIRED
#   ATLAS_UI_BASE      — http://atlas-ingress.<ns>.svc.cluster.local
#   BOOTSTRAP_MODE     — auto|baseline|full (default auto)
#     baseline — restore from canonical baseline in MinIO (fast: ~60s).
#     full     — upload WZ zip, run ingest (~10min).
#     auto     — probe canonical baseline; fall back to full on absence.
#   WZ_CANONICAL       — path to canonical zip (default /opt/wz/atlas.zip,
#                        only used in full mode)
#   MINIO_ENDPOINT     — http://minio.minio.svc.cluster.local:9000
#                        (for baseline-detect HEAD)
#   TENANT_ID/REGION/MAJOR_VERSION/MINOR_VERSION — required for tenant headers
```

Replace with:

```bash
#!/usr/bin/env bash
# Atlas PR-env bootstrap (task-098: baseline-only). Idempotent —
# short-circuits each step that is already complete. Data provisioning is
# baseline-restore ONLY: a read-only preflight hard-fails before any
# data-affecting work when no published canonical baseline exists for the
# env's version (cold-start a new version via the canonical-version-migration
# runbook). Reads:
#   ATLAS_ENV          — env hash, REQUIRED
#   ATLAS_UI_BASE      — http://atlas-ingress.<ns>.svc.cluster.local
#   MINIO_ENDPOINT     — http://minio.minio.svc.cluster.local:9000
#                        (for the baseline-presence HEAD probe)
#   TENANT_ID/REGION/MAJOR_VERSION/MINOR_VERSION — required for tenant headers
```

- [ ] **Step 2: Delete the `WZ_CANONICAL` and `BOOTSTRAP_MODE` defaults**

Find:

```bash
WZ_CANONICAL="${WZ_CANONICAL:-/opt/wz/atlas.zip}"
BOOTSTRAP_MODE="${BOOTSTRAP_MODE:-auto}"
MINIO_ENDPOINT="${MINIO_ENDPOINT:-http://minio.minio.svc.cluster.local:9000}"
```

Replace with:

```bash
MINIO_ENDPOINT="${MINIO_ENDPOINT:-http://minio.minio.svc.cluster.local:9000}"
```

- [ ] **Step 3: Delete `canonical_baseline_exists` and `resolve_mode`**

Delete the entire `canonical_baseline_exists()` function (the block starting at the `# Probe whether a canonical baseline exists …` comment through its closing `}`) and the entire `resolve_mode()` function (the `# Resolve BOOTSTRAP_MODE=auto …` comment through its closing `}`). Leave `baseline_object_status` / `baseline_reachable` / `probe_baseline_object` / `preflight_baseline` (added in Task 2) untouched.

- [ ] **Step 4: Drop the `/api/data/wz` wait-ready probe**

Find:

```bash
retry 60 5 http_ok_tenant "$ATLAS_UI_BASE/api/data/status"
retry 60 5 http_ok_tenant "$ATLAS_UI_BASE/api/data/wz"
```

Replace with:

```bash
retry 60 5 http_ok_tenant "$ATLAS_UI_BASE/api/data/status"
```

- [ ] **Step 5: Use `CANONICAL_TENANT_JSON` in the tenant-create step**

Find:

```bash
canonical_region=$(jq -r '.data.attributes.region' /atlas/canonical/tenant.json)
canonical_major=$(jq -r '.data.attributes.majorVersion' /atlas/canonical/tenant.json)
canonical_minor=$(jq -r '.data.attributes.minorVersion' /atlas/canonical/tenant.json)
```

Replace with:

```bash
canonical_region=$(jq -r '.data.attributes.region' "$CANONICAL_TENANT_JSON")
canonical_major=$(jq -r '.data.attributes.majorVersion' "$CANONICAL_TENANT_JSON")
canonical_minor=$(jq -r '.data.attributes.minorVersion' "$CANONICAL_TENANT_JSON")
```

(The `-d @/atlas/canonical/tenant.json` on the tenant POST a few lines below stays as a literal `curl` data ref — it is the image path, not a parsed var, and is out of the grep gate's terms.)

- [ ] **Step 6: Collapse the data-ingest step to restore-only**

Find the whole data-ingest block (the `# Data ingest: branch on resolved BOOTSTRAP_MODE.` comment through the `fi` that closes the `if [ "$docs" = "0" ] …` at line ~420):

```bash
# Data ingest: branch on resolved BOOTSTRAP_MODE.
#   baseline → POST /api/data/baseline/restore (fast, ~60s).
#   full     → PATCH /api/data/wz upload + POST /api/data/process
#              (~10min; ingest now runs inside atlas-data, no separate
#              WZ-extraction step).
ATLAS_STEP=data-ingest
mode=$(resolve_mode)
log info "bootstrap mode: $mode"

docs=$(get_attr "$ATLAS_UI_BASE/api/data/status" documentCount)
if [ "$docs" = "0" ] || [ "$docs" = "null" ]; then
    case "$mode" in
        baseline)
            log info "restoring canonical baseline → POST /api/data/baseline/restore"
            restore_body=$(jq -cn \
                --arg r "$REGION" \
                --arg M "$MAJOR_VERSION" \
                --arg m "$MINOR_VERSION" \
                --arg t "$TENANT_ID" \
                '{data:{type:"baselineRestores",attributes:{region:$r,majorVersion:($M|tonumber),minorVersion:($m|tonumber),tenantId:$t}}}')
            curl -fsS -X POST \
                -H "TENANT_ID: $TENANT_ID" \
                -H "REGION: $REGION" \
                -H "MAJOR_VERSION: $MAJOR_VERSION" \
                -H "MINOR_VERSION: $MINOR_VERSION" \
                -H "X-Atlas-Operator: 1" \
                -H "Content-Type: application/vnd.api+json" \
                -d "$restore_body" \
                "$ATLAS_UI_BASE/api/data/baseline/restore" >/dev/null
            retry 60 5 data_processing_done
            ;;
        full)
            log info "uploading canonical WZ zip → PATCH /api/data/wz"
            curl -fsS -X PATCH \
                -H "TENANT_ID: $TENANT_ID" \
                -H "REGION: $REGION" \
                -H "MAJOR_VERSION: $MAJOR_VERSION" \
                -H "MINOR_VERSION: $MINOR_VERSION" \
                -F "zip_file=@$WZ_CANONICAL" \
                "$ATLAS_UI_BASE/api/data/wz" >/dev/null
            log info "running data processing → POST /api/data/process"
            post "$ATLAS_UI_BASE/api/data/process"
            retry 240 10 data_processing_done
            ;;
    esac
else
    log info "data already processed (documentCount=$docs); skipping ingest"
fi
```

Replace with:

```bash
# Data ingest: baseline-restore only. The preflight already proved the
# baseline exists for this version, so there is no "what if absent" branch;
# a non-zero documentCount means a prior sync already restored (idempotent).
ATLAS_STEP=data-ingest
docs=$(get_attr "$ATLAS_UI_BASE/api/data/status" documentCount)
if [ "$docs" = "0" ] || [ "$docs" = "null" ]; then
    log info "restoring canonical baseline → POST /api/data/baseline/restore"
    restore_body=$(jq -cn \
        --arg r "$REGION" \
        --arg M "$MAJOR_VERSION" \
        --arg m "$MINOR_VERSION" \
        --arg t "$TENANT_ID" \
        '{data:{type:"baselineRestores",attributes:{region:$r,majorVersion:($M|tonumber),minorVersion:($m|tonumber),tenantId:$t}}}')
    curl -fsS -X POST \
        -H "TENANT_ID: $TENANT_ID" \
        -H "REGION: $REGION" \
        -H "MAJOR_VERSION: $MAJOR_VERSION" \
        -H "MINOR_VERSION: $MINOR_VERSION" \
        -H "X-Atlas-Operator: 1" \
        -H "Content-Type: application/vnd.api+json" \
        -d "$restore_body" \
        "$ATLAS_UI_BASE/api/data/baseline/restore" >/dev/null
    retry 60 5 data_processing_done
else
    log info "data already processed (documentCount=$docs); skipping ingest"
fi
```

- [ ] **Step 7: Run bats + shellcheck**

Run: `cd services/atlas-pr-bootstrap && bats test/bootstrap_test.bats && cd ../.. && shellcheck services/atlas-pr-bootstrap/scripts/bootstrap.sh services/atlas-pr-bootstrap/scripts/lib.sh`
Expected: 4 bats tests PASS; shellcheck clean. (The deletions removed the only callers of `WZ_CANONICAL`/`BOOTSTRAP_MODE`/`resolve_mode`/`canonical_baseline_exists`, so no unused-var/unused-func warnings remain.)

- [ ] **Step 8: Verify the grep gate over the script**

Run: `grep -nE 'BOOTSTRAP_MODE|WZ_CANONICAL|resolve_mode|/opt/wz|atlas-canonical/atlas\.zip|api/data/wz|api/data/process|canonical_baseline_exists' services/atlas-pr-bootstrap/scripts/bootstrap.sh`
Expected: no output. (`lib.sh` still mentions `resolve_mode` in a comment — fixed in Task 4.)

- [ ] **Step 9: Commit**

```bash
git add services/atlas-pr-bootstrap/scripts/bootstrap.sh
git commit -m "refactor(atlas-pr-bootstrap): remove full-mode ingest; baseline-restore only"
```

---

## Task 4: Clean the stale `resolve_mode` reference in lib.sh

The acceptance grep bans `resolve_mode` anywhere under `services/atlas-pr-bootstrap/`. `lib.sh`'s `log()` doc comment names it; generalize the comment.

**Files:**
- Modify: `services/atlas-pr-bootstrap/scripts/lib.sh`

- [ ] **Step 1: Generalize the `log()` comment (lines ~7-16)**

Find:

```bash
    # Logs are diagnostic output and MUST go to stderr. The fallback branch
    # already does; the jq branch did not, which caused subtle bugs when a
    # caller captured a function's stdout via $(): e.g. resolve_mode echoes
    # the resolved mode on stdout, but a `log warn` inside it would prepend
    # a JSON line into the captured value and break the subsequent `case`
    # match. PR-544 hit this — auto-mode resolution silently produced a
    # multi-line mode value, the case statement no-op'd, and the bootstrap
    # exited cleanly without running ingest. Fixed by redirecting both
    # branches to stderr.
```

Replace with:

```bash
    # Logs are diagnostic output and MUST go to stderr. The fallback branch
    # already does; the jq branch did not, which caused subtle bugs when a
    # caller captured a function's stdout via $(): a helper that echoes a
    # value on stdout would get a `log` line prepended into the captured
    # value, breaking the caller's parse. PR-544 hit this in a since-removed
    # mode-resolution helper. Fixed by redirecting both branches to stderr.
```

- [ ] **Step 2: Verify the full grep gate is clean across the service + overlay**

Run:

```bash
grep -rnE 'BOOTSTRAP_MODE|WZ_CANONICAL|fetch-wz-canonical|/opt/wz|resolve_mode|atlas-canonical/atlas\.zip' \
    services/atlas-pr-bootstrap/ deploy/k8s/overlays/pr/
```

Expected: no output. (The `sync-bootstrap.yaml` hits clear in Task 5; if Task 5 is not yet done, `fetch-wz-canonical`/`/opt/wz` still appear there — re-run this after Task 5.)

- [ ] **Step 3: shellcheck + commit**

Run: `shellcheck services/atlas-pr-bootstrap/scripts/lib.sh`
Expected: clean.

```bash
git add services/atlas-pr-bootstrap/scripts/lib.sh
git commit -m "docs(atlas-pr-bootstrap): drop stale resolve_mode reference in lib.sh comment"
```

---

## Task 5: Remove the WZ init container and volume from the bootstrap Job

Delete the three manifest blocks that exist only to fetch and mount `atlas.zip`. Per the design, OQ-1 is resolved: the volume is an `emptyDir` (not a PVC), and `grep -rn atlas-wz-canonical deploy` is empty, so there is no PVC to delete.

**Files:**
- Modify: `deploy/k8s/overlays/pr/sync-bootstrap.yaml`

- [ ] **Step 1: Remove the main container's `volumeMounts`**

Find:

```yaml
          envFrom:
            - configMapRef:
                name: atlas-env-tokens
          volumeMounts:
            - name: wz-canonical
              mountPath: /opt/wz
              readOnly: true
      initContainers:
```

Replace with:

```yaml
          envFrom:
            - configMapRef:
                name: atlas-env-tokens
      initContainers:
```

- [ ] **Step 2: Remove the `initContainers` list and the `volumes` list**

Find (the tail of the file, from `initContainers:` through the end):

```yaml
      initContainers:
        # Canonical WZ atlas.zip lives in cluster-internal MinIO (namespace
        # `minio`, bucket `atlas-canonical`, anonymous read). PVC mounts can't
        # cross namespaces, so we fetch into an emptyDir instead.
        # See runbook §9.1 for MinIO setup.
        - name: fetch-wz-canonical
          image: curlimages/curl:8.10.1
          command:
            - sh
            - -c
            - |
              set -euo pipefail
              curl -fsSL --connect-timeout 10 \
                  --retry 5 --retry-delay 5 --retry-connrefused \
                  -o /opt/wz/atlas.zip \
                  "http://minio.minio.svc.cluster.local:9000/atlas-canonical/atlas.zip"
          volumeMounts:
            - name: wz-canonical
              mountPath: /opt/wz
      volumes:
        - name: wz-canonical
          # atlas.zip is ~1GB as of v83 canonical; 8Gi cap matches the prior
          # PVC budget and gives ample headroom for future WZ revisions.
          emptyDir:
            sizeLimit: 8Gi
```

Replace with: *(delete the entire block — the pod `spec.template.spec` now ends with the `containers:` list.)* The last non-blank line of the file becomes the `envFrom` block's `name: atlas-env-tokens`.

- [ ] **Step 3: Verify the overlay renders and is free of removed references**

Run:

```bash
kustomize build deploy/k8s/overlays/pr >/tmp/pr-render.yaml && \
grep -nE 'wz-canonical|fetch-wz-canonical|/opt/wz|atlas-canonical/atlas\.zip' /tmp/pr-render.yaml
```

Expected: `kustomize build` exits 0; the `grep` prints nothing. (If `kustomize` is not installed, use `kubectl kustomize deploy/k8s/overlays/pr`.)

- [ ] **Step 4: Commit**

```bash
git add deploy/k8s/overlays/pr/sync-bootstrap.yaml
git commit -m "chore(deploy): drop fetch-wz-canonical init container and /opt/wz volume from PR bootstrap"
```

---

## Task 6: Rewrite the ephemeral-PR runbook to baseline-only

Replace the `atlas.zip`/`BOOTSTRAP_MODE` content of §9.1 with a baseline-only description, keep the MinIO stand-up steps (baselines still live there), drop the `mc cp … atlas.zip` upload + "refreshing the zip" subsection, and fix the top-of-file cross-reference.

**Files:**
- Modify: `docs/runbooks/ephemeral-pr-deployments.md`

- [ ] **Step 1: Fix the top "See also" cross-reference (lines ~6-8)**

Find:

```markdown
> **See also:** [canonical-version-migration.md](canonical-version-migration.md)
> for the one-time migration that provisions per-version canonical baselines consumed
> by the `auto`-mode bootstrap this runbook describes.
```

Replace with:

```markdown
> **See also:** [canonical-version-migration.md](canonical-version-migration.md)
> for the one-time migration that provisions per-version canonical baselines consumed
> by the baseline-only bootstrap this runbook describes.
```

- [ ] **Step 2: Replace §9.1 (from the `## §9.1 First-time setup` heading through the end of the "Refreshing the canonical zip" subsection, i.e. lines ~10-102) up to but NOT including `## §9.1b`**

Replace that span with:

````markdown
## §9.1 Data provisioning: baseline-only

Ephemeral PR envs provision game data **only** by restoring the published
canonical baseline for their `(region, major, minor)`. There is no WZ
re-ingest path in the bootstrap — `bootstrap.sh` calls
`POST /api/data/baseline/restore` and nothing else. This keeps PR envs fast
(~60s restore, no ~1 GB download, no ~10-min ingest) and guarantees no
ephemeral env ever writes a per-tenant WZ/asset tree into shared MinIO.

### Fail-fast on a missing baseline

Before any data-affecting work, `bootstrap.sh` runs a read-only preflight
that HEADs both the baseline dump and its sha256 sidecar in MinIO:

```
HEAD $MINIO_ENDPOINT/atlas-canonical/baseline/regions/<region>/versions/<major>.<minor>/documents.dump
HEAD $MINIO_ENDPOINT/atlas-canonical/baseline/regions/<region>/versions/<major>.<minor>/documents.dump.sha256
```

- **Both 200** → the bootstrap proceeds.
- **Either 404** → the Job exits non-zero with a single greppable line:
  `no canonical baseline for <region> <major>.<minor> … publish one … before deploying this env`.
  Argo CD surfaces the failed Job; the env does not come up half-seeded.
- **MinIO unreachable (000)** after a bounded retry → a *distinct* `MinIO unreachable`
  error (so a transient blip is never misread as "go publish a baseline").

Cold-starting a brand-new version therefore requires publishing its baseline
**first** — that is the [canonical-version-migration](canonical-version-migration.md)
runbook (its step 4, `POST /api/data/baseline/publish`). Publishing the
baseline is a prerequisite for any PR env on that version, not an optimization.

### Stand up MinIO (one-time)

MinIO is where the canonical baselines live. Apply the manifest from the
cluster-infra repo, then wait for the Deployment:

```sh
kubectl apply -f <infra-repo>/minio.yml
kubectl rollout status -n minio deployment/minio --timeout=120s
```

The `atlas-canonical` bucket has an anonymous-read policy (set by
`atlas-minio-init`), so the bootstrap's preflight needs no credentials. The
baselines themselves are produced and consumed by `baseline/publish` and
`baseline/restore` (atlas-data) — see the migration runbook for the publish
procedure. There is no `atlas.zip` to upload.
````

- [ ] **Step 3: Verify the runbook no longer mentions the removed mechanics**

Run:

```bash
grep -nE 'atlas\.zip|BOOTSTRAP_MODE|WZ_CANONICAL|fetch-wz-canonical|/opt/wz|auto.?mode' docs/runbooks/ephemeral-pr-deployments.md
```

Expected: no output. (If `## §9.1b` or later sections legitimately reference something, confirm it is unrelated to the removed bootstrap mechanics; the line-490 `tenant_baselines` mention is unrelated and must remain.)

- [ ] **Step 4: Commit**

```bash
git add docs/runbooks/ephemeral-pr-deployments.md
git commit -m "docs(runbook): rewrite ephemeral PR §9.1 to baseline-only provisioning"
```

---

## Task 7: Update the canonical-version-migration runbook note

The migration runbook currently tells operators the bootstrap impact is "none" because `auto` mode restores automatically. With baseline-only bootstrap, publishing a baseline is a hard prerequisite. Update the two spots that say `auto`.

**Files:**
- Modify: `docs/runbooks/canonical-version-migration.md`

- [ ] **Step 1: Update the impact note (lines ~29-31)**

Find:

```markdown
**`atlas-pr-bootstrap` impact:** none. Once per-version baselines are
published (step 4), `auto` mode restores the correct version automatically —
no changes to the bootstrap job are required.
```

Replace with:

```markdown
**`atlas-pr-bootstrap` impact:** the bootstrap is **baseline-only** — it
provisions PR-env data exclusively by restoring the published canonical
baseline for the env's version and **fails fast** (before bringing services
up) when none exists. Publishing the per-version baseline (step 4) is
therefore a **prerequisite** for any ephemeral env on that version, not an
optimization. See [ephemeral-pr-deployments.md](ephemeral-pr-deployments.md) §9.1.
```

- [ ] **Step 2: Fix the step-4 `auto`-mode wording (line ~107)**

Find:

```markdown
Publish the version-correct dump and sha256 sidecar so ephemeral `auto`-mode
```

Replace with:

```markdown
Publish the version-correct dump and sha256 sidecar so the ephemeral baseline-only
```

- [ ] **Step 3: Verify and commit**

Run: `grep -nE 'auto.?mode' docs/runbooks/canonical-version-migration.md`
Expected: no output.

```bash
git add docs/runbooks/canonical-version-migration.md
git commit -m "docs(runbook): note canonical migration baseline is a bootstrap prerequisite"
```

---

## Task 8: Full verification sweep

Run the complete acceptance-gate set from the design §7 across all changed artifacts.

**Files:** none (verification only).

- [ ] **Step 1: bats suite green**

Run: `cd services/atlas-pr-bootstrap && bats test/ && cd ../..`
Expected: all tests pass (the two `require_env`, the two new preflight tests, and any pre-existing `lib_test.bats`).

- [ ] **Step 2: shellcheck clean**

Run: `shellcheck services/atlas-pr-bootstrap/scripts/*.sh`
Expected: no output. (Adjust to skip non-bash files if the glob includes `version-ports.sh`/`service-config.sh` and they were already clean — they are unchanged.)

- [ ] **Step 3: Kustomize renders**

Run: `kustomize build deploy/k8s/overlays/pr >/dev/null && echo OK` (or `kubectl kustomize`).
Expected: `OK`.

- [ ] **Step 4: Acceptance grep gate (the design §7.4 set)**

Run:

```bash
grep -rnE 'BOOTSTRAP_MODE|WZ_CANONICAL|fetch-wz-canonical|/opt/wz|resolve_mode|atlas-canonical/atlas\.zip' \
    services/atlas-pr-bootstrap/ deploy/k8s/overlays/pr/
```

Expected: no output. (Historical `docs/tasks/task-063|071/` references are pre-existing and out of scope — not in the scanned paths.)

- [ ] **Step 5: Docker image still packages (no Go bake target applies)**

Run: `docker build -t atlas-pr-bootstrap:task098 services/atlas-pr-bootstrap`
Expected: build succeeds. (`atlas-pr-bootstrap` has its own Dockerfile that `COPY`s the scripts; it is NOT a `docker buildx bake` Go target, so the CLAUDE.md bake rule does not apply. If Docker is unavailable in this environment, note that and rely on the bats + shellcheck + kustomize gates, which fully cover the changed files; flag it for CI.)

- [ ] **Step 6: Final review of the whole diff**

Run: `git log --oneline main..HEAD` and `git diff main...HEAD --stat`
Expected: commits for tasks 1-7 present; only the six files from the File Structure table changed (plus this `plan.md`/`context.md` from the planning phase). No stray edits in the main repo or other services.

---

## Self-Review

**Spec coverage (PRD §4 FRs + design §9 files):**

| Requirement | Task |
|---|---|
| FR-1.1 data step restores baseline only | Task 3 (Step 6) |
| FR-1.2 remove `BOOTSTRAP_MODE`/`WZ_CANONICAL`/`full`/`auto→full` | Task 3 (Steps 1-3, 6) |
| FR-1.3 remove `fetch-wz-canonical` init container + `/opt/wz` (no PVC per OQ-1) | Task 5 |
| FR-2.1 early preflight HEAD probe | Task 2 (Steps 2-3) |
| FR-2.2 fail fast with version + runbook message | Task 2 (Step 2, `preflight_baseline`); Task 1 asserts it |
| FR-2.3 deterministic/idempotent re-sync | Task 2 (read-only probe); Task 3 (idempotent restore on documentCount) |
| FR-2 transient-vs-absent distinction (design §3.3) | Task 2 (`baseline_reachable`/`probe_baseline_object`); Task 1 second test |
| OQ-3 probe BOTH dump + sha256 | Task 2 (`preflight_baseline`) |
| design §3.2 canonical-version source via `CANONICAL_TENANT_JSON` | Task 2 (Step 1); Task 3 (Step 5) |
| design §3.5 drop `/api/data/wz` wait-ready probe | Task 3 (Step 4) |
| FR-3.1 rewrite ephemeral-pr-deployments §9.1 | Task 6 |
| design §5 migration-runbook note | Task 7 |
| lib.sh cosmetic `resolve_mode` cleanup (grep gate) | Task 4 |
| Acceptance grep clean | Task 4 (Step 2), Task 8 (Step 4) |
| Verification (bats/shellcheck/kustomize/docker) | Task 8 |

**Placeholder scan:** No TBD/TODO/"handle edge cases"/"similar to" — every code/edit step shows exact find/replace content.

**Type/identifier consistency:** `baseline_object_status`, `baseline_reachable`, `BASELINE_PROBE_CODE`, `probe_baseline_object`, `preflight_baseline`, `CANONICAL_TENANT_JSON`, `MINIO_PROBE_RETRIES`, `MINIO_PROBE_SLEEP` are defined in Task 2 and used consistently in Tasks 2-3 and asserted in Task 1. Test helper names (`prq_env`, `make_shims`, `write_fixture_tenant`) are defined and used within Task 1.

**One ordering note for the executor:** Task 4 Step 2's full grep only goes fully clean after Task 5 removes the manifest hits — both tasks are in this plan, so run the gate again in Task 8 Step 4 for the authoritative result.
