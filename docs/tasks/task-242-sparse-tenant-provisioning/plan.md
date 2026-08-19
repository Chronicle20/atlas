# Sparse-Mode Tenant Provisioning Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give every sparse PR environment its own tenant, recorded on its
control-plane record, and make baseline pods and browser traffic agree on
which environment they are in — so `tenant_id` and `environment` actually
isolate a sparse environment and teardown can reclaim it.

**Architecture:** Three independent seams, all outside service code. (1)
`deploy/k8s` gains a `main` environment record (Argo Job) and an
`ATLAS_ENVIRONMENT` literal, so `env.Self()` is non-empty on baseline pods
and ownership/heartbeat/`EnvironmentsOwnedBy` resolve. (2) `atlas-ingress`
gains an nginx `map` that defaults an absent inbound `ENVIRONMENT` header to
the pod's own `ATLAS_ENVIRONMENT`, so browser reads are environment-scoped.
(3) `bootstrap.sh` scopes its tenant lookup/create to `$ATLAS_ENVIRONMENT`
and PATCHes the minted tenant id onto the environment record via a helper
extracted from `cleanup.sh`.

**Tech Stack:** Bash + bats (`services/atlas-pr-bootstrap`), kustomize
(`deploy/k8s`), nginx `map` + `envsubst` templates, Go
(`libs/atlas-env`, `atlas-configurations` — read-only in this plan).

**Spec:** `docs/tasks/task-242-sparse-tenant-provisioning/design.md`
(PRD: `docs/tasks/task-242-sparse-tenant-provisioning/prd.md`)

## Global Constraints

- **Isolated mode (`deploy/k8s/overlays/pr`) and local compose must be
  behaviourally unchanged.** `env.Self()` stays `""` there, the ingress
  stamps nothing, and bootstrap's curl argv for the tenant step stays
  byte-identical to today (FR-1.5, FR-2.5).
- **The baseline environment id is `main`**, equal to
  `deploy/k8s/overlays/main/kustomization.yaml`'s `namespace: atlas-main`
  with the `atlas-` prefix stripped — the same derivation
  `.github/workflows/pr-validation.yml:921` performs (FR-1.2).
- **`tools/pr-sparse-mirror-guard.sh` must pass.** No task may edit any of
  the nine files in its `MIRRORS` array
  (`tools/pr-sparse-mirror-guard.sh:31-41`): `atlas-env-tokens.yaml`,
  `ingress-route.yaml`, `sync-bootstrap.yaml`, `predelete-purge.yaml`,
  `postsync-pihole-add.yaml`, `patches/ingress-host.yaml`,
  `patches/consumer-group-env.yaml`, `patches/lb-allocate.yaml`,
  `patches/seed-catalog-ref.yaml`. No task in this plan touches any of them.
- **`services/atlas-pr-bootstrap/scripts/bootstrap.sh` is not sourceable.**
  Its bats suites extract helpers with
  `sed -n '/^name()/,/^}/p'`, so every new helper MUST open with `name()` at
  column 0 and close with `}` at column 0
  (`services/atlas-pr-bootstrap/test/data_ingest_test.bats:14-24`).
- **`services/atlas-pr-bootstrap/test/cleanup_test.bats` must keep passing
  UNMODIFIED.** It is the acceptance test for Task 1's delegation refactor.
- **`environments` PATCH bodies must carry all five attributes AND a
  non-empty `phase`.** `UpdateByName` calls `validatePhase(input.Phase)`
  *before* any backfill
  (`services/atlas-configurations/atlas.com/configurations/environments/processor.go:224-226`),
  and `phaseIndex("")` is `-1`, so a body with no `phase` is a 400. A
  same-phase transition is explicitly legal (`processor.go:82-88`).
- **Never hard-code `main` in `deploy/k8s/overlays/pr-sparse/`.** That
  overlay names the baseline only through
  `PLACEHOLDER_BASELINE_ENVIRONMENT` / `PLACEHOLDER_BASELINE_NAMESPACE`.
  No task in this plan edits `pr-sparse`.
- **Gate:** flagless `tools/verify.sh` must exit 0 before the branch is
  called done.

---

## Deviations from design.md (read before Task 4/5/7)

Two design decisions are amended by evidence found while planning. Both are
recorded in `context.md` with the same reasoning.

**A. `ATLAS_ENVIRONMENT_DEFAULT` must always be *defined*, not `optional`.**
D3 specifies `configMapKeyRef … optional: true`. The nginx image's
`docker-entrypoint.d/20-envsubst-on-templates.sh` builds its substitution
list from variables that are actually **present in the process environment**
and passes that list to `envsubst` as a SHELL-FORMAT argument. A variable
that is *undefined* is therefore not substituted at all — `envsubst` copies
`${ATLAS_ENVIRONMENT_DEFAULT}` through **verbatim**, and the rendered map
would stamp the literal string `${ATLAS_ENVIRONMENT_DEFAULT}` on every
untagged request, which `ParseEnvironment` 400s
(`libs/atlas-rest/server/handler.go:68`). That is a silent, total outage in
exactly the overlays the design wanted to leave untouched.

The fix is to guarantee the key exists everywhere rather than tolerate its
absence: Task 4 adds `ATLAS_ENVIRONMENT` to `deploy/k8s/base/env-configmap.yaml`
(empty) and to `deploy/k8s/overlays/pr/kustomization.yaml`'s generator
(empty), because that generator is `behavior: replace`
(`deploy/k8s/overlays/pr/kustomization.yaml:155-157`) and would otherwise
drop the base key. `overlays/pr-sparse` is `behavior: merge` and already
carries `ATLAS_ENVIRONMENT=pr-PLACEHOLDER_PR_NUMBER`
(`deploy/k8s/overlays/pr-sparse/kustomization.yaml:258-266`), so it needs no
change. Task 5 then omits `optional: true`, so a future overlay that drops
the key fails loudly at pod start instead of 400ing every request.

`ATLAS_ENVIRONMENT=""` is semantically identical to unset for
`env.Self()` (`os.Getenv` returns `""` either way,
`libs/atlas-env/env.go:78-81`), so FR-1.5 still holds. The design's manifest
assertion "`overlays/pr` renders no `ATLAS_ENVIRONMENT` anywhere" becomes
"`overlays/pr` renders `ATLAS_ENVIRONMENT` with an **empty** value, and no
`ATLAS_ENVIRONMENT_DEFAULT` other than empty".

**B. D6's Go regression pin already exists; no new Go test is written.**
Design §7 asks for "one regression test in `atlas-configurations` pinning
that a template lookup from a non-baseline environment still resolves the
baseline's row", homed in `templates/overlay_test.go`. That test is already
there and asserts exactly that:
`TestTemplatesFallBackToTheBaselineRow`
(`services/atlas-configurations/atlas.com/configurations/templates/overlay_test.go:65-77`)
seeds only a `main` v83.1 row, reads as `pr-123`, and fails unless the
baseline row is returned — which is precisely the call
`bootstrap.sh:290` makes and precisely what a `scope.Strict` refactor would
break. Landing a second copy would be duplicate coverage, so this plan adds
none and instead cites the existing pin in the runbook (Task 7).

---

## Task 1: Extract the environment-record helper out of cleanup.sh

**Files:**

- `services/atlas-pr-bootstrap/scripts/env-record.sh` — **new file**; the two
  shared helpers
- `services/atlas-pr-bootstrap/scripts/cleanup.sh` — source the new file;
  `_dcp_env_get` (line 91) and `_dcp_patch_phase` (line 150) become one-line
  delegations
- `services/atlas-pr-bootstrap/Dockerfile` — `COPY` the new script
- `services/atlas-pr-bootstrap/test/dockerfile_test.bats` — add
  `env-record.sh` to the chmod-exclusion list (it is sourced, not executed)
- `services/atlas-pr-bootstrap/test/env_record_test.bats` — **new file**
- `services/atlas-pr-bootstrap/test/cleanup_test.bats` — **read-only**; must
  keep passing unmodified, do not edit it

Patterns to copy: `services/atlas-pr-bootstrap/scripts/service-config.sh`
(same "sourced sibling helper" shape, no shebang-driven execution);
`services/atlas-pr-bootstrap/test/service_config_test.bats:3-17` (sources
`lib.sh` first, then the helper under test).

Run bats from the repo root: `bats services/atlas-pr-bootstrap/test`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-pr-bootstrap/test/env_record_test.bats`. Setup
sources `lib.sh` then `env-record.sh` (that order — the helpers call `log`),
copying `service_config_test.bats:3-17`. `curl` is shimmed as a shell
function that records its argv to `$CURL_ARGS` and echoes `$CURL_BODY`,
copying `data_ingest_test.bats:41-48`. After sourcing, guard the extraction
the way `data_ingest_test.bats:25-34` does — assert
`declare -F env_record_get` and `declare -F env_record_patch` both succeed,
so a missing definition fails the suite instead of turning every "must fail"
assertion green via exit 127.

Fixed environment for every case:
`ATLAS_UI_BASE=http://ui`, `ATLAS_ENVIRONMENT=pr-1411`.

`TestEnvRecordGet` cases — bats test names given verbatim:

| bats test name | env / stub | expected |
|---|---|---|
| `env_record_get GETs this environment's record with the ENVIRONMENT header` | `CURL_BODY='{"data":{"id":"pr-1411"}}'` | status 0; output `{"data":{"id":"pr-1411"}}`; `$CURL_ARGS` contains the line `ENVIRONMENT: pr-1411` and the line `http://ui/api/configurations/environments/pr-1411` |
| `env_record_get fails when ATLAS_UI_BASE is unset` | `unset ATLAS_UI_BASE` | status 1; `$CURL_ARGS` does not exist (curl never ran) |
| `env_record_get fails when ATLAS_ENVIRONMENT is unset` | `unset ATLAS_ENVIRONMENT` | status 1; `$CURL_ARGS` does not exist |
| `env_record_get mirrors curl's exit status on a 404` | `CURL_RC=22` | status 22 |

`TestEnvRecordPatch` cases:

| bats test name | args | expected |
|---|---|---|
| `env_record_patch sends all five attributes plus the record id` | `env_record_patch ACTIVE main atlas-main 8f14e45f-ceea-467a-9c2a-1b3f4c5d6e70 '{"atlas-login":"atlas-pr-1411"}'` | status 0; the `-d` payload (the `$CURL_ARGS` line that parses as JSON) satisfies: `.data.type == "environments"`, `.data.id == "pr-1411"`, `.data.attributes.phase == "ACTIVE"`, `.data.attributes.baseline == "main"`, `.data.attributes.namespace == "atlas-main"`, `.data.attributes.tenant == "8f14e45f-ceea-467a-9c2a-1b3f4c5d6e70"`, `.data.attributes.overrides["atlas-login"] == "atlas-pr-1411"` |
| `env_record_patch targets the environments PATCH route with the ENVIRONMENT header` | same args | `$CURL_ARGS` contains the lines `-X`, `PATCH`, `ENVIRONMENT: pr-1411`, and `http://ui/api/configurations/environments/pr-1411` |
| `env_record_patch accepts an empty overrides object` | `env_record_patch PROVISIONING main atlas-main "" '{}'` | status 0; payload `.data.attributes.overrides == {}` and `.data.attributes.tenant == ""` |
| `env_record_patch propagates a failing PATCH` | `CURL_RC=22`, same args as the first case | status 22 |

Helper for reading the payload back out of the recorded argv (define it in
the bats file):

```bash
# patch_payload — echoes the recorded `-d` argument (the only argv line that
# parses as JSON). The curl shim records one argument per line.
patch_payload() {
    while IFS= read -r line; do
        if printf '%s' "$line" | jq -e . >/dev/null 2>&1; then
            printf '%s' "$line"
            return 0
        fi
    done <"$CURL_ARGS"
    return 1
}
```

- [ ] **Step 2: Run the test and verify it fails**

Run: `bats services/atlas-pr-bootstrap/test/env_record_test.bats`
Expected: FAIL — `setup` aborts because
`services/atlas-pr-bootstrap/scripts/env-record.sh` does not exist.

- [ ] **Step 3: Create `services/atlas-pr-bootstrap/scripts/env-record.sh`**

Move the two bodies out of `cleanup.sh` **verbatim** — same curl flags, same
`jq -nc` payload construction, same header set. Do not re-derive them; the
whole point of the extraction is that `cleanup_test.bats` cannot tell the
difference.

```bash
#!/usr/bin/env bash
# Shared control-plane environment-record helpers, sourced by BOTH
# bootstrap.sh (which records the environment's tenant, FR-3) and
# cleanup.sh (which flips phase during teardown). Extracted from cleanup.sh
# so the two callers cannot drift on the one thing that is easy to get
# wrong: a PATCH body that omits an attribute.
#
# Sourced, never executed — no `set` here (lib.sh owns option state) and no
# executable bit (see test/dockerfile_test.bats).

# env_record_get — echoes this environment's environments record GET
# response body (the full JSON:API document), or nothing if no record exists
# (a 404 from the GET) or ATLAS_UI_BASE/ATLAS_ENVIRONMENT are unset. Exit
# status mirrors curl's.
env_record_get() {
    [ -z "${ATLAS_UI_BASE:-}" ] && return 1
    [ -z "${ATLAS_ENVIRONMENT:-}" ] && return 1
    curl -fsS -H 'Accept: application/vnd.api+json' \
        -H "ENVIRONMENT: $ATLAS_ENVIRONMENT" \
        "$ATLAS_UI_BASE/api/configurations/environments/$ATLAS_ENVIRONMENT" 2>/dev/null
}

# env_record_patch <phase> <baseline> <namespace> <tenant> <overrides_json>
# — PATCHes the environments record, sending ALL five attributes.
# environments.RestModel's fields are non-pointer (environments/rest.go), so
# ParseInput unmarshals a PATCH body into a fresh zero-value struct first:
# any attribute omitted from the body is zeroed, not left alone
# (environments/administrator.go's update() doc comment is explicit about
# this). The processor now ALSO backfills omitted fields from the existing
# record (environments/processor.go:243-255), so the two layers disagree
# about who is responsible — sending everything is the move that is correct
# under both.
#
# <phase> must be non-empty: UpdateByName calls validatePhase BEFORE any
# backfill (processor.go:224-226) and phaseIndex("") is -1, so a
# phase-less body is a 400. A same-phase value is a legal transition
# (processor.go:82-88), which is what makes a tenant-only PATCH possible.
env_record_patch() {
    local phase="$1" baseline="$2" namespace="$3" tenant="$4" overrides="$5"
    local payload
    payload=$(jq -nc \
        --arg id "$ATLAS_ENVIRONMENT" \
        --arg baseline "$baseline" \
        --arg namespace "$namespace" \
        --arg tenant "$tenant" \
        --argjson overrides "$overrides" \
        --arg phase "$phase" \
        '{data:{type:"environments",id:$id,attributes:{baseline:$baseline,namespace:$namespace,tenant:$tenant,overrides:$overrides,phase:$phase}}}')
    curl -fsS -X PATCH \
        -H 'Content-Type: application/vnd.api+json' \
        -H "ENVIRONMENT: $ATLAS_ENVIRONMENT" \
        -d "$payload" \
        "$ATLAS_UI_BASE/api/configurations/environments/$ATLAS_ENVIRONMENT" >/dev/null
}
```

- [ ] **Step 4: Run the new test and verify it passes**

Run: `bats services/atlas-pr-bootstrap/test/env_record_test.bats`
Expected: PASS (8 tests).

- [ ] **Step 5: Delegate from cleanup.sh**

In `services/atlas-pr-bootstrap/scripts/cleanup.sh`, source the new file
immediately after `lib.sh` (currently line 19-20):

```bash
# shellcheck source=env-record.sh
. "$(dirname "$0")/env-record.sh"
```

Replace the *body* of `_dcp_env_get` (line 91) with `env_record_get`, and
the body of `_dcp_patch_phase` (line 150) with `env_record_patch "$@"`.
**Keep both function names and both doc comments** — `cleanup_test.bats`
runs `cleanup.sh` as a whole script and has 30 KB of pinned assertions that
must not move. Add one line to each comment noting the body now lives in
`env-record.sh`.

- [ ] **Step 6: Ship the script in the image**

In `services/atlas-pr-bootstrap/Dockerfile`, add below the
`COPY scripts/service-config.sh /atlas/service-config.sh` line:

```dockerfile
COPY scripts/env-record.sh /atlas/env-record.sh
```

Do **not** add it to the `RUN chmod +x` line — it is sourced, not executed.
In `services/atlas-pr-bootstrap/test/dockerfile_test.bats`, add the matching
exclusion beside the existing three, inside the second test:

```bash
    [ "$base" = "env-record.sh" ] && continue
```

- [ ] **Step 7: Run the whole bats suite**

Run: `bats services/atlas-pr-bootstrap/test`
Expected: PASS. `cleanup_test.bats` must pass with **zero edits** — that is
the acceptance test for this refactor. If it fails, the extraction changed
behaviour; fix `env-record.sh`, never `cleanup_test.bats`.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-pr-bootstrap/scripts/env-record.sh \
        services/atlas-pr-bootstrap/scripts/cleanup.sh \
        services/atlas-pr-bootstrap/Dockerfile \
        services/atlas-pr-bootstrap/test/dockerfile_test.bats \
        services/atlas-pr-bootstrap/test/env_record_test.bats
git commit -m "refactor(pr-bootstrap): extract the environment-record GET/PATCH helper"
```

---

## Task 2: Scope bootstrap's tenant lookup and create to this environment

**Files:**

- `services/atlas-pr-bootstrap/scripts/bootstrap.sh` — add `env_header_init`,
  `find_environment_tenant`, `create_environment_tenant`; replace the
  `tenant-create` block at lines 228-266
- `services/atlas-pr-bootstrap/test/tenant_provisioning_test.bats` — **new
  file**
- `services/atlas-pr-bootstrap/scripts/lib.sh` — **read-only**; `require_env`
  (line 27) exits 1 on a missing variable, `log` (line 6) writes to stderr
- `services/atlas-pr-bootstrap/canonical/tenant.json` — **read-only**; the
  canonical POST payload (`GMS`, `majorVersion: 83`, `minorVersion: 1`)

**Interfaces:**

- Consumes: nothing from Task 1.
- Produces: `find_environment_tenant <region> <major> <minor>` echoes this
  environment's matching tenant id or nothing;
  `create_environment_tenant` echoes the newly minted id and returns
  non-zero on failure; `env_header_init` sets the global bash array
  `ENV_HEADER`. Task 3 relies on `TENANT_ID` holding the resolved id at the
  end of the `tenant-create` step.

Patterns to copy: `services/atlas-pr-bootstrap/test/data_ingest_test.bats:9-48`
(the sed-extraction harness, the `declare -F` guard, and the curl shim).

- [ ] **Step 1: Write the failing test**

Create `services/atlas-pr-bootstrap/test/tenant_provisioning_test.bats`.
`setup` extracts the three helpers out of `bootstrap.sh` with
`sed -n '/^name()/,/^}/p'` into `$BATS_TEST_TMPDIR/helpers.sh`, sources
`lib.sh` first and then that file, and asserts
`declare -F env_header_init`, `declare -F find_environment_tenant`,
`declare -F create_environment_tenant` — copying
`data_ingest_test.bats:14-34` exactly, including the failure messages.

The `curl` shim records one argv element per line to `$CURL_ARGS` and echoes
`$CURL_BODY`, honouring `$CURL_RC` — `data_ingest_test.bats:41-48` verbatim.

Fixed environment: `ATLAS_UI_BASE=http://ui`,
`ATLAS_ENVIRONMENT=pr-1411`,
`CANONICAL_TENANT_JSON="$PROJECT_ROOT/canonical/tenant.json"`.
Each case sets `ATLAS_MODE` and calls `env_header_init` before the helper
under test, since `ENV_HEADER` is what the helpers expand.

Fixture bodies (define these as bats helper functions):

```bash
# A tenant listing carrying exactly one GMS 83.1 row.
one_tenant() {
    printf '{"data":[{"type":"tenants","id":"%s","attributes":{"region":"GMS","majorVersion":83,"minorVersion":1}}]}' "$1"
}
# Two GMS 83.1 rows, to pin "first wins".
two_tenants() {
    printf '{"data":[{"type":"tenants","id":"%s","attributes":{"region":"GMS","majorVersion":83,"minorVersion":1}},{"type":"tenants","id":"%s","attributes":{"region":"GMS","majorVersion":83,"minorVersion":1}}]}' "$1" "$2"
}
```

Constants used below:
`ENV_TENANT=6a5f0c1e-9d2b-4a77-8c31-0f2e5b7a9d40`,
`OTHER_TENANT=1c9d3f42-7b60-4e18-9a5c-2d8f6e0b31a7`.

`env_header_init` cases:

| bats test name | env | expected |
|---|---|---|
| `env_header_init builds the ENVIRONMENT header in sparse mode` | `ATLAS_MODE=sparse` | `${#ENV_HEADER[@]}` is `2`; `${ENV_HEADER[0]}` is `-H`; `${ENV_HEADER[1]}` is `ENVIRONMENT: pr-1411` |
| `env_header_init leaves ENV_HEADER empty in isolated mode` | `unset ATLAS_MODE` | `${#ENV_HEADER[@]}` is `0` |
| `env_header_init leaves ENV_HEADER empty when ATLAS_MODE is explicitly isolated` | `ATLAS_MODE=isolated` | `${#ENV_HEADER[@]}` is `0` |
| `env_header_init fails loudly when sparse mode has no ATLAS_ENVIRONMENT` | `ATLAS_MODE=sparse`, `unset ATLAS_ENVIRONMENT` | status 1; stderr contains `missing required env: ATLAS_ENVIRONMENT` |

`find_environment_tenant` cases (all call
`find_environment_tenant GMS 83 1`):

| bats test name | mode / stub | expected |
|---|---|---|
| `find_environment_tenant scopes the listing with the ENVIRONMENT header in sparse mode` | sparse, `CURL_BODY='{"data":[]}'` | status 0; output empty; `$CURL_ARGS` contains a line exactly equal to `ENVIRONMENT: pr-1411` |
| `find_environment_tenant sends no ENVIRONMENT header in isolated mode` | isolated, `CURL_BODY='{"data":[]}'` | status 0; `$CURL_ARGS` contains **no** line starting with `ENVIRONMENT:`; it does contain `http://ui/api/tenants` |
| `find_environment_tenant echoes the environment's own tenant id` | sparse, `CURL_BODY="$(one_tenant $ENV_TENANT)"` | output is `$ENV_TENANT` |
| `find_environment_tenant echoes the first match when several exist` | sparse, `CURL_BODY="$(two_tenants $ENV_TENANT $OTHER_TENANT)"` | output is `$ENV_TENANT` |
| `find_environment_tenant does not adopt a tenant when the scoped listing is empty` | sparse, `CURL_BODY='{"data":[]}'` | output empty — **the core regression**: the version triple alone must never resolve a tenant; only the server's environment-scoped listing may |
| `find_environment_tenant ignores a tenant on a different version triple` | sparse, `CURL_BODY='{"data":[{"type":"tenants","id":"'$OTHER_TENANT'","attributes":{"region":"GMS","majorVersion":95,"minorVersion":1}}]}'` | output empty |
| `find_environment_tenant expands an empty ENV_HEADER without tripping set -u` | isolated, `CURL_BODY='{"data":[]}'`, run under `set -u` | status 0 (no `unbound variable`) |

`create_environment_tenant` cases:

| bats test name | mode / stub | expected |
|---|---|---|
| `create_environment_tenant POSTs the canonical payload with the ENVIRONMENT header` | sparse, `CURL_BODY='{"data":{"type":"tenants","id":"'$ENV_TENANT'"}}'` | status 0; output `$ENV_TENANT`; `$CURL_ARGS` contains `-X`, `POST`, `ENVIRONMENT: pr-1411`, `@'"$PROJECT_ROOT"'/canonical/tenant.json` (as the `-d` argument) and `http://ui/api/tenants` |
| `create_environment_tenant sends no ENVIRONMENT header in isolated mode` | isolated, same body | status 0; `$CURL_ARGS` contains no line starting with `ENVIRONMENT:` |
| `create_environment_tenant fails when the POST returns no id` | sparse, `CURL_BODY='{"data":{"type":"tenants"}}'` | status 1; stderr contains `tenant POST returned no id` |
| `create_environment_tenant fails when the POST itself fails` | sparse, `CURL_RC=22` | status 1; stderr contains `tenant POST failed` |

- [ ] **Step 2: Run the test and verify it fails**

Run: `bats services/atlas-pr-bootstrap/test/tenant_provisioning_test.bats`
Expected: FAIL in `setup` — `env_header_init not extracted from
.../bootstrap.sh`.

- [ ] **Step 3: Add the three helpers to bootstrap.sh**

Insert immediately after the `MINIO_PROBE_SLEEP="${MINIO_PROBE_SLEEP:-5}"`
line (currently `services/atlas-pr-bootstrap/scripts/bootstrap.sh:39`), i.e.
before the `TENANT_ID` shape check. All three must start at column 0 so the
bats sed extraction finds them.

```bash
# ENV_HEADER is the control-plane environment tag every scoped request
# carries. It is an ARRAY, not a string: a string would word-split on the
# space in "ENVIRONMENT: pr-1411" and curl would see two broken arguments.
#
# Gating on ATLAS_MODE rather than on "does an environment record exist"
# (cleanup.sh:135-137's live check) is deliberate. Teardown must not trust a
# build-time flag because it is *reacting* to whatever got deployed;
# bootstrap is *establishing* the state and must send the header on its very
# first call, before any record could answer the question. ATLAS_MODE is
# also the signal the neighbouring create_service_config already keys on.
#
# In isolated mode the array is empty, so every curl argv below is
# byte-identical to what it was before this existed (FR-2.5) — one code
# path, not two.
env_header_init() {
    ENV_HEADER=()
    if [ "${ATLAS_MODE:-isolated}" = "sparse" ]; then
        require_env ATLAS_ENVIRONMENT
        ENV_HEADER=(-H "ENVIRONMENT: $ATLAS_ENVIRONMENT")
    fi
}
env_header_init

# find_environment_tenant <region> <major> <minor> — echoes THIS
# environment's tenant id for that version triple, or nothing.
#
# The ENVIRONMENT header is the entire fix for the adopt-main's-tenant
# defect. atlas-tenants' getAll already applies scope.Strict to the caller's
# environment (tenant/provider.go:27-32); without the header the caller is
# the legacy "" environment and sees the unfiltered union, in which main's
# canonical tenant ALWAYS matches the canonical version triple — because a
# sparse tenant deliberately shares that triple and is distinguished only by
# its environment and its generated UUID.
find_environment_tenant() {
    curl -fsS -H 'Accept: application/vnd.api+json' \
        "${ENV_HEADER[@]}" \
        "$ATLAS_UI_BASE/api/tenants" \
        | jq -r --arg r "$1" --arg M "$2" --arg m "$3" \
            '.data[] | select(.attributes.region == $r and (.attributes.majorVersion|tostring) == $M and (.attributes.minorVersion|tostring) == $m) | .id' \
        | head -1
}

# create_environment_tenant — POSTs the canonical tenant payload and echoes
# the assigned id. Entity.Environment is server-owned from request context
# (tenant/entity.go:16), so the ENVIRONMENT header is the ONLY way to stamp
# the new row with this environment; the request body cannot carry it.
create_environment_tenant() {
    local created id
    created=$(curl -fsS -X POST \
        -H 'Accept: application/vnd.api+json' \
        -H 'Content-Type: application/vnd.api+json' \
        "${ENV_HEADER[@]}" \
        -d @"$CANONICAL_TENANT_JSON" \
        "$ATLAS_UI_BASE/api/tenants") || { log error "tenant POST failed"; return 1; }
    id=$(printf '%s' "$created" | jq -r '.data.id // empty')
    if [ -z "$id" ] || [ "$id" = "null" ]; then
        log error "tenant POST returned no id"
        return 1
    fi
    printf '%s' "$id"
}
```

- [ ] **Step 4: Replace the tenant-create block**

Replace `services/atlas-pr-bootstrap/scripts/bootstrap.sh:234-266` — from
`existing=$(curl -fsS -H 'Accept: application/vnd.api+json' \` through the
closing `fi` of the `if [ -n "$existing" ] …` block, inclusive — with:

```bash
existing=$(find_environment_tenant "$canonical_region" "$canonical_major" "$canonical_minor")

if [ -n "$existing" ] && [ "$existing" != "null" ]; then
    log info "tenant already present for environment=${ATLAS_ENVIRONMENT:-<legacy>}: $existing"
    TENANT_ID="$existing"
else
    log info "creating tenant for environment=${ATLAS_ENVIRONMENT:-<legacy>} ($canonical_region v$canonical_major.$canonical_minor)"
    TENANT_ID=$(create_environment_tenant) || exit 1

    # Wait for the tenant.status Kafka event to settle. atlas-tenants writes
    # the DB row before the emit; if Kafka is unreachable the emit fails and
    # the next caller would see a tenant via REST with no event published.
    # This mirrors the onboarding doc's pitfall #1.
    sleep 10
fi
```

Leave the three `canonical_*` assignments above it and the
`REGION=/MAJOR_VERSION=/MINOR_VERSION=` reassignments below it untouched.
`TENANT_ID` is reassigned in place and every later step already reads it, so
FR-2.6 needs no further work.

- [ ] **Step 5: Run the test and verify it passes**

Run: `bats services/atlas-pr-bootstrap/test/tenant_provisioning_test.bats`
Expected: PASS (15 tests).

- [ ] **Step 6: Run the whole suite and shellcheck**

Run: `bats services/atlas-pr-bootstrap/test`
Expected: PASS.

Run: `shellcheck services/atlas-pr-bootstrap/scripts/bootstrap.sh`
Expected: no new findings versus `git stash`-ing the change (the file has
pre-existing informational output; only regressions matter).

- [ ] **Step 7: Commit**

```bash
git add services/atlas-pr-bootstrap/scripts/bootstrap.sh \
        services/atlas-pr-bootstrap/test/tenant_provisioning_test.bats
git commit -m "fix(pr-bootstrap): scope the tenant lookup and create to this environment"
```

---

## Task 3: Record the resolved tenant on the environment record

**Files:**

- `services/atlas-pr-bootstrap/scripts/bootstrap.sh` — source `env-record.sh`;
  add `record_environment_tenant`; call it right after the `tenant-create`
  step
- `services/atlas-pr-bootstrap/test/env_record_test.bats` — extend with the
  `record_environment_tenant` cases
- `services/atlas-pr-bootstrap/scripts/env-record.sh` — **read-only** here;
  Task 1 created it
- `services/atlas-pr-bootstrap/scripts/cleanup.sh` — **read-only**;
  `do_sweep_tenant` (line 334) is the consumer of the attribute this task
  writes

**Interfaces:**

- Consumes: `env_record_get` and
  `env_record_patch <phase> <baseline> <namespace> <tenant> <overrides_json>`
  from Task 1; `TENANT_ID` as resolved by Task 2.
- Produces: `record_environment_tenant <tenant_id>`, returning non-zero on
  any failure.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-pr-bootstrap/test/env_record_test.bats`. The
existing `setup` sources `lib.sh` and `env-record.sh`; extend it to also
extract `record_environment_tenant` from `bootstrap.sh` with
`sed -n '/^record_environment_tenant()/,/^}/p'` and add the matching
`declare -F record_environment_tenant` guard.

The curl shim must now distinguish GET from PATCH. Replace the shim in this
file with one that echoes `$GET_BODY` when the argv contains no `-X PATCH`,
and records the PATCH payload otherwise:

```bash
curl() {
    printf '%s\n' "$@" >>"$CURL_ARGS"
    for a in "$@"; do
        if [ "$a" = "PATCH" ]; then
            [ "${PATCH_RC:-0}" -eq 0 ] || return "$PATCH_RC"
            return 0
        fi
    done
    [ "${GET_RC:-0}" -eq 0 ] || return "$GET_RC"
    printf '%s' "$GET_BODY"
}
```

Record fixture (a bats helper in the same file):

```bash
# env_record <phase> <tenant> — a full environments GET document.
env_record() {
    printf '{"data":{"type":"environments","id":"pr-1411","attributes":{"name":"pr-1411","baseline":"main","namespace":"atlas-pr-1411","tenant":"%s","overrides":{"atlas-login":"atlas-pr-1411","atlas-channel":"atlas-pr-1411"},"phase":"%s"}}}' "$2" "$1"
}
```

Cases (all with `ATLAS_UI_BASE=http://ui`, `ATLAS_ENVIRONMENT=pr-1411`;
`NEW_TENANT=6a5f0c1e-9d2b-4a77-8c31-0f2e5b7a9d40`):

| bats test name | stub state | expected |
|---|---|---|
| `record_environment_tenant PATCHes the new tenant onto the record` | `GET_BODY="$(env_record ACTIVE '')"`; call `record_environment_tenant "$NEW_TENANT"` | status 0; payload `.data.attributes.tenant == "$NEW_TENANT"` |
| `record_environment_tenant carries the record's current phase, never an empty one` | same | payload `.data.attributes.phase == "ACTIVE"` — a phase-less body is a 400 (`processor.go:224-226`) |
| `record_environment_tenant carries a PROVISIONING phase through unchanged` | `GET_BODY="$(env_record PROVISIONING '')"` | payload `.data.attributes.phase == "PROVISIONING"` — bootstrap runs while the environment is PROVISIONING; it must not promote it |
| `record_environment_tenant carries baseline, namespace and overrides through unchanged` | `GET_BODY="$(env_record ACTIVE '')"` | payload `.data.attributes.baseline == "main"`, `.data.attributes.namespace == "atlas-pr-1411"`, `.data.attributes.overrides["atlas-channel"] == "atlas-pr-1411"` |
| `record_environment_tenant is a no-op-shaped same-phase PATCH when the tenant is already recorded` | `GET_BODY="$(env_record ACTIVE "$NEW_TENANT")"` | status 0; payload `.data.attributes.tenant == "$NEW_TENANT"` and `.data.attributes.phase == "ACTIVE"` (FR-3.4) |
| `record_environment_tenant fails when no environment record exists` | `GET_RC=22` | status 1; stderr contains `no control-plane environment record for pr-1411` |
| `record_environment_tenant fails when the record has no phase` | `GET_BODY='{"data":{"type":"environments","id":"pr-1411","attributes":{}}}'` | status 1; stderr contains `no control-plane environment record for pr-1411` |
| `record_environment_tenant propagates a failing PATCH` | `GET_BODY="$(env_record ACTIVE '')"`, `PATCH_RC=22` | status 22 (non-zero) — FR-3.5 |

- [ ] **Step 2: Run the test and verify it fails**

Run: `bats services/atlas-pr-bootstrap/test/env_record_test.bats`
Expected: FAIL in `setup` — `record_environment_tenant not extracted`.

- [ ] **Step 3: Source env-record.sh from bootstrap.sh**

After the `. "$(dirname "$0")/service-config.sh"` line
(`services/atlas-pr-bootstrap/scripts/bootstrap.sh:26-27`), add:

```bash
# shellcheck source=env-record.sh
. "$(dirname "$0")/env-record.sh"
```

- [ ] **Step 4: Add `record_environment_tenant`**

Place it directly after `create_environment_tenant` (Task 2's insertion
block), at column 0.

```bash
# record_environment_tenant <tenant_id> — writes the tenant id onto this
# environment's control-plane record (FR-3), which is the ONLY thing
# cleanup.sh's sweep-tenant phase reads to know what to reclaim
# (cleanup.sh:352-358). Without it a sparse environment's gameplay rows
# survive teardown forever, silently.
#
# The PATCH must carry the record's CURRENT phase, not a chosen one:
# bootstrap runs while the environment is PROVISIONING and must not promote
# it, and a body with no phase is a 400 (UpdateByName validates phase before
# it backfills anything). A same-phase transition is explicitly legal, which
# is also what makes re-running this idempotent (FR-3.4).
record_environment_tenant() {
    local tenant="$1" body phase baseline namespace overrides
    body=$(env_record_get) || body=""
    phase=$(printf '%s' "${body:-}" | jq -r '.data.attributes.phase // empty' 2>/dev/null)
    if [ -z "$phase" ]; then
        log error "no control-plane environment record for ${ATLAS_ENVIRONMENT:-<unset>}; cannot record tenant=$tenant"
        return 1
    fi
    baseline=$(printf '%s' "$body" | jq -r '.data.attributes.baseline // ""' 2>/dev/null)
    namespace=$(printf '%s' "$body" | jq -r '.data.attributes.namespace // ""' 2>/dev/null)
    overrides=$(printf '%s' "$body" | jq -c '.data.attributes.overrides // {}' 2>/dev/null)
    env_record_patch "$phase" "$baseline" "$namespace" "$tenant" "$overrides"
}
```

- [ ] **Step 5: Call it, immediately after the tenant is resolved**

Insert directly below the
`log info "using TENANT_ID=$TENANT_ID for downstream calls"` line
(currently `bootstrap.sh:271`, just above the `ATLAS_STEP=tenant-config`
comment block):

```bash
# Record the tenant on the control-plane environment record BEFORE any
# tenant-keyed write happens. The reverse order leaks on every partial
# failure: a bootstrap that dies after the config clone but before the PATCH
# leaves rows under a tenant teardown has no way to name.
#
# Sparse-only, and gated on the same ATLAS_MODE fact env_header_init uses:
# an isolated environment registers no control-plane record at all
# (cleanup.sh:120-133), so there is nothing to PATCH and do_drop_dbs reclaims
# its rows by dropping the databases outright.
#
# `|| exit 1` is belt-and-braces for `set -e` (restored at line 22 after
# lib.sh relaxes it), stated at the call site for the same reason the
# create_service_config calls below do. An environment that provisions a
# tenant but never records it produces exactly the residue this exists to
# prevent (FR-3.5).
if [ "${ATLAS_MODE:-isolated}" = "sparse" ]; then
    ATLAS_STEP=record-tenant record_environment_tenant "$TENANT_ID" || exit 1
    ATLAS_STEP=record-tenant log info "recorded tenant=$TENANT_ID on environment record $ATLAS_ENVIRONMENT"
fi
```

- [ ] **Step 6: Run the test and verify it passes**

Run: `bats services/atlas-pr-bootstrap/test/env_record_test.bats`
Expected: PASS (16 tests — Task 1's 8 plus these 8).

- [ ] **Step 7: Run the whole suite**

Run: `bats services/atlas-pr-bootstrap/test`
Expected: PASS, `cleanup_test.bats` still unmodified.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-pr-bootstrap/scripts/bootstrap.sh \
        services/atlas-pr-bootstrap/test/env_record_test.bats
git commit -m "feat(pr-bootstrap): record the environment's tenant on its control-plane record"
```

---

## Task 4: Give baseline pods their own environment id, and main a record

**Files:**

- `deploy/k8s/base/env-configmap.yaml` — add `ATLAS_ENVIRONMENT: ""` (see
  Deviation A)
- `deploy/k8s/overlays/main/kustomization.yaml` — add
  `- ATLAS_ENVIRONMENT=main` to the `behavior: replace` `atlas-env`
  literals (block starts line 36); add `environment-record.yaml` to
  `resources`
- `deploy/k8s/overlays/pr/kustomization.yaml` — add `- ATLAS_ENVIRONMENT=`
  to the `behavior: replace` `atlas-env` literals (block starts line 155)
- `deploy/k8s/overlays/main/environment-record.yaml` — **new file**; the
  wave-11 idempotent GET-then-POST Job
- `deploy/k8s/overlays/pr-sparse/kustomization.yaml` — **read-only**; already
  carries `ATLAS_ENVIRONMENT=pr-PLACEHOLDER_PR_NUMBER` at line 266 under
  `behavior: merge`, so it inherits nothing that needs changing

Patterns to copy: `deploy/k8s/overlays/pr-sparse/environment-record.yaml`
(the Job's whole shape — `Force=true,Replace=true`, `backoffLimit: 3`,
`alpine:3.20` + `apk add curl`, GET-then-POST idempotency).

- [ ] **Step 1: Add the ATLAS_ENVIRONMENT keys**

In `deploy/k8s/base/env-configmap.yaml`, insert as the first entry under
`data:` (before `BASE_SERVICE_URL`):

```yaml
  # env.Self() reads this (libs/atlas-env SelfVar). Empty in the base and in
  # the isolated PR overlay: an environment that registers no control-plane
  # record must keep Self()=="" so EnvironmentsOwnedBy still returns [""]
  # and the legacy single-iteration path is preserved
  # (libs/atlas-env/registry.go:227).
  #
  # It must be DEFINED even when empty. atlas-ingress renders its
  # environment default from this key through the nginx image's envsubst
  # entrypoint, and envsubst only substitutes variables that are actually
  # present in the process environment — an undefined one is copied through
  # as the literal "${ATLAS_ENVIRONMENT_DEFAULT}", which every REST call
  # would then 400 on. Empty-here is what keeps that from happening.
  ATLAS_ENVIRONMENT: ""
```

In `deploy/k8s/overlays/main/kustomization.yaml`, add as the first literal
of the `atlas-env` generator (immediately above `BASE_SERVICE_URL=…`):

```yaml
      # env.Self() reads this. It MUST equal `namespace` with the `atlas-`
      # prefix stripped, which is exactly how CI derives
      # BASELINE_ENVIRONMENT (.github/workflows/pr-validation.yml:921) and
      # what pr-sparse's environment-record.yaml writes into `baseline`.
      # MapRegistry.IsOwner compares rec.Baseline == r.self by string
      # equality (libs/atlas-env/registry.go:218), so any disagreement makes
      # the baseline own nothing. tools/overlay-env-guard.sh pins the two
      # together.
      - ATLAS_ENVIRONMENT=main
```

In `deploy/k8s/overlays/pr/kustomization.yaml`, add under the
`# Infrastructure literals.` comment, above `BASE_SERVICE_URL=…`:

```yaml
      # Isolated mode registers no control-plane environment record, so it
      # must keep env.Self()=="" (FR-1.5). Present-but-empty rather than
      # absent, because this generator is `behavior: replace` and would
      # otherwise drop the base key that atlas-ingress's envsubst needs to
      # be defined — see deploy/k8s/base/env-configmap.yaml.
      - ATLAS_ENVIRONMENT=
```

- [ ] **Step 2: Create the main environment-record Job**

Create `deploy/k8s/overlays/main/environment-record.yaml`:

```yaml
# The baseline's own control-plane environment record. Three independent
# mechanisms need it to exist, and all three are broken today because
# nothing creates it:
#
#   - MapRegistry.EnvironmentsOwnedBy (libs/atlas-env/registry.go:225-247)
#     returns the legacy [""] ONLY while no records are projected. The
#     moment any sparse PR registers one, main's pods have a non-empty
#     record set and admit a record only when rec.Baseline == r.self — so
#     without a `main` record AND ATLAS_ENVIRONMENT=main, every
#     ForEachOwnedEnvironment ticker in main iterates nothing for the whole
#     lifetime of that PR environment.
#   - environments.StartHeartbeat republishes envlib.Self()
#     (atlas-configurations/.../environments/heartbeat.go:27). Republish on a
#     name with no record fails, so nothing is published, so every pod's
#     registry ages into Stale() and the Kafka gate starts dropping messages
#     as unresolvable rather than skipping them as not-owned.
#   - ParseEnvironment (libs/atlas-rest/server/handler.go:68) 400s an
#     unknown environment id, and once the ingress default lands main stamps
#     ENVIRONMENT: main on browser traffic.
#
# POSTs to the ClusterIP, NOT the ingress. Once the ingress default lands it
# stamps ENVIRONMENT: main on everything — including the very request that
# creates the `main` record, which ParseEnvironment would then 400. An
# in-cluster caller that sets no header is the legacy "" environment, which
# ParseEnvironment admits unconditionally.
#
# sync-wave 11: the base pushes every Deployment to wave 10
# (deploy/k8s/base/kustomization.yaml:97-106), so this runs after
# atlas-configurations is healthy. On a FRESH atlas-main bring-up that
# leaves a seconds-long window where atlas-ingress is Ready and stamping
# `main` before this Job has created the record, and every REST call 400s.
# Argo gates wave 11 on wave-10 health, so the window is bounded and
# self-healing; it is accepted deliberately (design D1).
#
# `Force=true,Replace=true` (delete+recreate every sync, the same idiom as
# the base Kafka precreate Jobs and pr-sparse's own environment-record Job)
# means the body must be idempotent: it GETs by name first and only POSTs
# when absent, so a re-sync against an existing record is a no-op rather
# than a duplicate-name 500 (environments.Name carries a uniqueIndex).
#
# `tenant` stays "" here on purpose. do_sweep_tenant is sparse-only
# (services/atlas-pr-bootstrap/scripts/cleanup.sh:335-341) and never runs
# against main; populating it would add a value nothing reads and one more
# way to point a sweep at main's own tenant.
apiVersion: batch/v1
kind: Job
metadata:
  name: atlas-environment-record
  annotations:
    argocd.argoproj.io/sync-wave: "11"
    argocd.argoproj.io/sync-options: Force=true,Replace=true
spec:
  backoffLimit: 3
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: environment-record
          image: alpine:3.20
          command: ["/bin/sh", "-c"]
          args:
            - |
              set -eu
              apk add --no-cache --quiet curl >/dev/null
              base="http://atlas-configurations.atlas-main.svc.cluster.local:8080/api/configurations/environments"
              name="main"
              if curl -fsS "$base/$name" >/dev/null 2>&1; then
                echo "environment record $name already exists, skipping"
                exit 0
              fi
              curl -fsS -X POST "$base" \
                -H "Content-Type: application/vnd.api+json" \
                -d '{
                      "data": {
                        "type": "environments",
                        "attributes": {
                          "name": "main",
                          "baseline": "main",
                          "namespace": "atlas-main",
                          "tenant": "",
                          "overrides": {},
                          "phase": "ACTIVE"
                        }
                      }
                    }'
              echo "created environment record $name"
```

Then add it to `deploy/k8s/overlays/main/kustomization.yaml`'s `resources`
list, below `atlas-env-tokens.yaml`:

```yaml
  - environment-record.yaml
```

- [ ] **Step 3: Verify the overlays still render**

```sh
kustomize build deploy/k8s/overlays/main >/dev/null
kustomize build deploy/k8s/overlays/pr >/dev/null
kustomize build deploy/k8s/overlays/pr-sparse >/dev/null
```

Expected: all three exit 0.

Then confirm the values landed:

```sh
kustomize build deploy/k8s/overlays/main | grep -n 'ATLAS_ENVIRONMENT'
kustomize build deploy/k8s/overlays/pr   | grep -n 'ATLAS_ENVIRONMENT'
```

Expected: main shows `ATLAS_ENVIRONMENT: main` in the `atlas-env-*`
ConfigMap; `pr` shows `ATLAS_ENVIRONMENT: ""` and nothing else.

- [ ] **Step 4: Verify the mirror guard still passes**

Run: `./tools/pr-sparse-mirror-guard.sh`
Expected: exit 0 (this task edits no mirrored file).

- [ ] **Step 5: Commit**

```bash
git add deploy/k8s/base/env-configmap.yaml \
        deploy/k8s/overlays/main/kustomization.yaml \
        deploy/k8s/overlays/main/environment-record.yaml \
        deploy/k8s/overlays/pr/kustomization.yaml
git commit -m "feat(deploy): give main its own environment id and control-plane record"
```

---

## Task 5: Default the ingress ENVIRONMENT header per overlay

**Files:**

- `deploy/k8s/base/env-default.conf.template` — **new file**; the nginx `map`
- `deploy/k8s/base/kustomization.yaml` — add the template to the
  `atlas-ingress-routes` `configMapGenerator` (block at lines 82-85)
- `deploy/k8s/base/atlas-ingress.yaml` — `nginx.conf` include + log format +
  `proxy_set_header` (lines 28-53); `NGINX_ENVSUBST_FILTER` (line 108-109);
  new container env var; new `volumeMount` (lines 252-254)
- `deploy/k8s/base/routes.conf.template.generated` — **read-only**; generated
  by `tools/gen-routes.sh`, must not be hand-edited and is not touched here

- [ ] **Step 1: Create the template**

Create `deploy/k8s/base/env-default.conf.template`:

```nginx
# Rendered to /etc/nginx/conf.d/env-default.conf by the nginx image's
# envsubst entrypoint (docker-entrypoint.d/20-envsubst-on-templates.sh) and
# included from nginx.conf's http {} block.
#
# $atlas_environment is the RESOLVED environment for a request: the caller's
# own ENVIRONMENT header when it sent one, this deployment's environment
# otherwise. In-cluster service-to-service calls already set the header via
# EnvHeaderDecorator (libs/atlas-rest/requests/header.go:46) and must win,
# which is what the `default` arm does. A browser sends nothing, which is
# what the "" arm catches: without it the request reaches atlas-tenants as
# the legacy "" environment and scope.Strict applies no filter at all,
# returning the unfiltered union of every environment's rows.
#
# ${ATLAS_ENVIRONMENT_DEFAULT} comes from the atlas-env ConfigMap's
# ATLAS_ENVIRONMENT key — the SAME key env.Self() reads — so the ingress
# default can never drift from what the pods behind it believe they are.
# Empty in the base and in overlays/pr, which renders `"" "";` and makes
# this file a no-op: nginx does not emit a proxy_set_header with an empty
# value, so untagged requests stay untagged exactly as they are today.
map $http_environment $atlas_environment {
    ""      "${ATLAS_ENVIRONMENT_DEFAULT}";
    default $http_environment;
}
```

- [ ] **Step 2: Generate it into the ingress ConfigMap**

In `deploy/k8s/base/kustomization.yaml`, extend the existing
`atlas-ingress-routes` generator (lines 83-85) to carry both files:

```yaml
configMapGenerator:
  - name: atlas-ingress-routes
    files:
      - routes.conf.template=routes.conf.template.generated
      - env-default.conf.template
```

- [ ] **Step 3: Wire it into the ingress**

Four edits in `deploy/k8s/base/atlas-ingress.yaml`.

(a) In `nginx.conf`'s `http {}` block, immediately above the
`# $http_environment resolves…` comment that precedes `log_format`
(currently line 28), add:

```
      # Resolves $atlas_environment: the caller's ENVIRONMENT header, or
      # this deployment's own environment when the caller sent none. See
      # env-default.conf.template.
      include /etc/nginx/conf.d/env-default.conf;
```

(b) Change the `log_format main_env` last line (line 36) from
`'env=$http_environment';` to `'env=$atlas_environment';`, and update the
comment above it (lines 28-31) to say `$atlas_environment` records the
*resolved* environment rather than only the inbound header.

(c) Change line 53 from
`proxy_set_header ENVIRONMENT $http_environment;` to
`proxy_set_header ENVIRONMENT $atlas_environment;`.

(d) Add the container env var immediately after the
`NGINX_ENVSUBST_FILTER` entry, and extend the filter itself:

```yaml
        - name: NGINX_ENVSUBST_FILTER
          value: "POD_NAMESPACE|NS_|ATLAS_ENVIRONMENT_DEFAULT"
        # The per-overlay ENVIRONMENT default, sourced from the SAME
        # atlas-env key env.Self() reads so the two can never disagree.
        # Deliberately NOT `optional: true`: envsubst only substitutes
        # variables that are present in the process environment, so an
        # undefined one is copied through as the literal
        # "${ATLAS_ENVIRONMENT_DEFAULT}" and every untagged request would
        # then 400 in ParseEnvironment. A hard CreateContainerConfigError on
        # an overlay that drops the key is strictly better than that silent
        # outage; deploy/k8s/base/env-configmap.yaml guarantees the key
        # exists (empty) for every overlay that does not set it.
        - name: ATLAS_ENVIRONMENT_DEFAULT
          valueFrom:
            configMapKeyRef:
              name: atlas-env
              key: ATLAS_ENVIRONMENT
```

(e) Add the mount beside the existing `routes.conf.template` one (after
line 254):

```yaml
        - name: nginx-routes-volume
          mountPath: /etc/nginx/templates/env-default.conf.template
          subPath: env-default.conf.template
```

- [ ] **Step 4: Verify the render**

```sh
kustomize build deploy/k8s/overlays/main | grep -c 'env-default.conf.template'
kustomize build deploy/k8s/overlays/pr   | grep -n 'ATLAS_ENVIRONMENT_DEFAULT'
./tools/gen-routes.sh --check
./tools/pr-sparse-mirror-guard.sh
```

Expected: the first prints a non-zero count; the second shows the
`configMapKeyRef` block; `gen-routes.sh --check` exits 0 (the new template
is a sibling of `routes.conf.template.generated`, not part of its generated
content); the mirror guard exits 0.

- [ ] **Step 5: Commit**

```bash
git add deploy/k8s/base/env-default.conf.template \
        deploy/k8s/base/kustomization.yaml \
        deploy/k8s/base/atlas-ingress.yaml
git commit -m "feat(ingress): default an absent ENVIRONMENT header to the deployment's own environment"
```

---

## Task 6: Pin the rendered manifests with a guard script

**Files:**

- `tools/overlay-env-guard.sh` — **new file**
- `tools/verify.sh` — wire the guard into the existing `deploy/` block
  (lines 554-561)
- `tools/pr-sparse-mirror-guard.sh` — **read-only**; the closest existing
  guard-script pattern to copy (argument handling, `status=0` accumulation,
  message style)

- [ ] **Step 1: Write the guard**

Create `tools/overlay-env-guard.sh`. It renders the three overlays once each
and asserts the facts this task's siblings depend on. Every failure must
print what it expected and what it found, then set `status=1` and keep
going, so one run reports every drift.

Assertions, in order:

| # | Overlay | Assertion | Requirement |
|---|---|---|---|
| 1 | main | the `atlas-env-*` ConfigMap has `ATLAS_ENVIRONMENT: main` | FR-1.1 |
| 2 | main | that value equals the overlay's `namespace` (`atlas-main`) with a leading `atlas-` stripped | FR-1.2 |
| 3 | main | a Job named `atlas-environment-record` exists, with `argocd.argoproj.io/sync-wave: "11"` | D1 |
| 4 | main | that Job's script contains `atlas-configurations.atlas-main.svc.cluster.local` and **not** `atlas-ingress` | D1 (must bypass the ingress) |
| 5 | main | that Job's POST body contains `"phase": "ACTIVE"` and `"name": "main"` | D1 |
| 6 | pr | the `atlas-env-*` ConfigMap has `ATLAS_ENVIRONMENT` present with an **empty** value | FR-1.5 |
| 7 | pr | no Job named `atlas-environment-record` is rendered | FR-1.5 |
| 8 | pr-sparse | the `atlas-env-*` ConfigMap has `ATLAS_ENVIRONMENT: pr-PLACEHOLDER_PR_NUMBER` | FR-4.2 |
| 9 | all three | the `atlas-ingress` container carries an `ATLAS_ENVIRONMENT_DEFAULT` env var whose `configMapKeyRef` names `atlas-env` / `ATLAS_ENVIRONMENT` | FR-4.1 |
| 10 | all three | that container's `NGINX_ENVSUBST_FILTER` value contains `ATLAS_ENVIRONMENT_DEFAULT` | D3 |
| 11 | all three | the `atlas-ingress-routes-*` ConfigMap carries an `env-default.conf.template` key | D3 |
| 12 | all three | the `atlas-ingress-configmap` `nginx.conf` contains `proxy_set_header ENVIRONMENT $atlas_environment;` and does **not** contain `proxy_set_header ENVIRONMENT $http_environment;` | FR-4.1 |

Implementation notes for the writer:

- Take the repo root from `git rev-parse --show-toplevel`, as
  `tools/pr-sparse-mirror-guard.sh` does.
- Render each overlay once into a temp file
  (`kustomize build deploy/k8s/overlays/<name>`); fail the whole script if
  `kustomize` is not on `PATH`, with the message
  `overlay-env-guard: kustomize not found on PATH`.
- Parse with `yq` if available, otherwise `grep`/`awk` — prefer plain
  `grep -F` on the rendered stream, since the assertions above are all
  literal-string facts and the repo has no `yq` dependency today. Do **not**
  add a new toolchain dependency.
- Assertion 2 must *derive* the expected value:
  read `namespace: atlas-main` from
  `deploy/k8s/overlays/main/kustomization.yaml`, strip the `atlas-` prefix,
  and compare against the rendered `ATLAS_ENVIRONMENT`. A hard-coded `main`
  on both sides would pass while proving nothing.
- Support `--help` printing one line of usage; no other flags.
- `exit "$status"`.

- [ ] **Step 2: Run it and verify it passes**

Run: `./tools/overlay-env-guard.sh`
Expected: exit 0, one PASS line per assertion group.

Then prove it actually fails on drift: temporarily change
`deploy/k8s/overlays/main/kustomization.yaml`'s literal to
`ATLAS_ENVIRONMENT=mian`, re-run, confirm assertions 1 and 2 both report a
failure and the script exits 1, then revert.

- [ ] **Step 3: Wire it into verify.sh**

In `tools/verify.sh`, inside the existing block that starts
`if touched '^(deploy/|tools/gen-lb-ports\.sh|.*versions\.json)'; then`
(line 554), add a fourth `step` alongside the three already there, and
extend the `touched` pattern so a change to the guard itself also runs it:

```sh
if touched '^(deploy/|tools/gen-lb-ports\.sh|tools/overlay-env-guard\.sh|.*versions\.json)'; then
    step "LB port drift"       ./tools/gen-lb-ports.sh --check
    step "routes drift"        ./tools/gen-routes.sh --check
    step "version coverage"    ./tools/check-version-coverage.sh
    step "overlay env drift"   ./tools/overlay-env-guard.sh
else
    skip "LB port / version coverage / overlay env (no deploy or versions.json change)"
fi
```

- [ ] **Step 4: Confirm verify.sh selects it**

Run: `./tools/verify.sh --facts`
Expected: the printed selection includes the `deploy/` block (this branch
has changed `deploy/k8s/...`).

- [ ] **Step 5: Commit**

```bash
chmod +x tools/overlay-env-guard.sh
git add tools/overlay-env-guard.sh tools/verify.sh
git commit -m "test(deploy): pin the per-overlay environment id and ingress default"
```

---

## Task 7: Document the boundary in the runbook

**Files:**

- `docs/runbooks/ephemeral-pr-deployments.md` — add a new section after
  `## §9.15 Sparse vs. isolated mode, and the per-PR override labels`
  (line 596)

- [ ] **Step 1: Write the section**

Add `## §9.16 Sparse-mode tenant ownership and the environment boundary`
covering, in this order:

1. **What a sparse environment owns.** One tenant, minted by `bootstrap.sh`
   under its own `ENVIRONMENT`, distinct from the baseline's, recorded on
   the control-plane record's `tenant` attribute. State that the tenant
   deliberately shares the baseline's `(region, majorVersion, minorVersion)`
   triple and is distinguished only by environment and generated UUID.

2. **What it does *not* duplicate.** Correct the widespread assumption that
   each sparse environment gets a ~48k-document `atlas-data` restore. It
   does not: the data-ingest guard
   (`services/atlas-pr-bootstrap/scripts/bootstrap.sh:541`, merged in
   `c5e88320a`) skips the restore when the shared database already holds the
   canonical rows, and the sparse tenant reads the shared corpus through
   `document/storage.go`'s canonical fallback — an id derived from the
   version triple, not the environment. So expect the PR tenant to report
   **0 owned documents** and the shared/canonical scope to report the full
   total. Mutable gameplay state is what is tenant-keyed and what
   `sweep-orphans.sh --sweep-tenant` reclaims.

3. **How to confirm the boundary held**, as a checklist an operator can run:
   the environment record carries a non-empty `tenant` that is not the
   baseline's; `GET /api/tenants` against `<N>.atlas.home` returns exactly
   one tenant; `GET /api/tenants` against `dev.atlas.home` returns only
   main's and does not list the PR's; bootstrap logged
   `recorded tenant=<id> on environment record pr-<N>`.

4. **The two metrics to watch, not one.** A stale registry produces
   `atlas_kafka_gate_dropped_unresolvable_total`, a *different* verdict from
   `atlas_kafka_gate_skipped_not_owner_total`. Both must stay flat for a
   non-overridden service (e.g. `atlas-ban`) processing a PR-environment
   operation. Name the heartbeat as the reason: with no `main` record and no
   `ATLAS_ENVIRONMENT`, `environments.StartHeartbeat` published nothing and
   every registry aged into `Stale()`.

5. **The `main` prerequisite and the bring-up window.** `overlays/main` now
   deploys `environment-record.yaml` at sync-wave 11 and sets
   `ATLAS_ENVIRONMENT=main`. On a *fresh* `atlas-main` bring-up there is a
   seconds-long window where atlas-ingress is Ready and stamping `main`
   before the record exists, and every REST call 400s; it is bounded by
   Argo's wave gating and self-heals. Also record the one-time operator step
   for the **existing** live cluster: after the sync completes, run
   `kubectl rollout restart deployment/atlas-ingress -n atlas-main`, because
   `atlas-ingress-configmap` is a plain resource (not generated) and editing
   `nginx.conf` does not roll the pods on its own.

6. **A pointer to the regression pin.** Note that sparse bootstrap depends
   on templates having a baseline-fallback scope rather than a strict one,
   and that the pin for it is
   `TestTemplatesFallBackToTheBaselineRow`
   (`services/atlas-configurations/atlas.com/configurations/templates/overlay_test.go:65`).
   A refactor that makes `templates` strictly scoped turns every sparse
   bootstrap into `no template found … cluster setup issue`.

Use repo-relative paths throughout; no absolute or home paths.

- [ ] **Step 2: Commit**

```bash
git add docs/runbooks/ephemeral-pr-deployments.md
git commit -m "docs(runbooks): document sparse-mode tenant ownership and the environment boundary"
```

---

## Task 8: Full verification gate

**Files:** none — this task only runs commands.

- [ ] **Step 1: Run the flagless gate**

Run: `tools/verify.sh`
Expected: exit 0. Only the flagless invocation counts — `--quick` and
`--no-docker` skip the bake and `-race`.

If it fails, fix the cause in the task that owns it and re-run. Do not
commit a green claim from a flagged run.

- [ ] **Step 2: Confirm the guards individually**

```sh
./tools/pr-sparse-mirror-guard.sh
./tools/gen-routes.sh --check
./tools/overlay-env-guard.sh
bats services/atlas-pr-bootstrap/test
```

Expected: all four exit 0.

- [ ] **Step 3: Confirm cleanup_test.bats was never edited**

Run: `git log --oneline -- services/atlas-pr-bootstrap/test/cleanup_test.bats`
Expected: no commit from this branch. If one appears, the Task 1 refactor
changed behaviour and was papered over — revert the test edit and fix
`env-record.sh` instead.

---

## Post-merge acceptance (operator, not the implementer)

Not part of any task's definition of done — recorded here so it is not lost.
The design's §6 corrections apply.

- [ ] The environment record for `pr-<N>` carries a non-empty `tenant` that
      is not the baseline's tenant id.
- [ ] `GET /api/tenants` against `<N>.atlas.home` returns exactly one tenant.
- [ ] `GET /api/tenants` against `dev.atlas.home` returns only main's tenants.
- [ ] `GET /api/data/status` under the PR tenant reports **0** owned
      documents; the shared/canonical total is unchanged; bootstrap logged
      the ingest skip.
- [ ] `atlas-ban` (deployed only in main) demonstrably processes an
      operation issued in the PR environment, with **both**
      `atlas_kafka_gate_skipped_not_owner_total{environment="pr-<N>"}` and
      `atlas_kafka_gate_dropped_unresolvable_total` flat for it.
- [ ] Main's own periodic per-tenant work continues while the sparse
      environment is live (design F3 — invisible to every other check).
- [ ] A game client reaches character select through the PR environment's
      login LB IP under the PR tenant.
- [ ] After PR closure, `sweep-tenant` reports a successful reclaim with no
      `cannot reclaim tenant-keyed rows` warning, and a direct query
      confirms zero rows remain for that tenant across `tenant-tables.txt`.
