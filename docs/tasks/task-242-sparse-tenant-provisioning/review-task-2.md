# Review — Task 2 of 8: Scope bootstrap's tenant lookup and create to this environment

Range reviewed: `1147693f0..82a34ffe9` (commit `82a34ffe9`).

## Scope confirmed

Diff touches exactly the two files the brief named:

- `services/atlas-pr-bootstrap/scripts/bootstrap.sh` (+70/-24)
- `services/atlas-pr-bootstrap/test/tenant_provisioning_test.bats` (new, +210)

`services/atlas-pr-bootstrap/scripts/lib.sh` and
`services/atlas-pr-bootstrap/canonical/tenant.json` have empty diffs in this
range — confirmed read-only. No file in
`tools/pr-sparse-mirror-guard.sh:31-41`'s `MIRRORS` array (`atlas-env-tokens.yaml`,
`ingress-route.yaml`, `sync-bootstrap.yaml`, `predelete-purge.yaml`,
`postsync-pihole-add.yaml`, `patches/ingress-host.yaml`,
`patches/consumer-group-env.yaml`, `patches/lb-allocate.yaml`,
`patches/seed-catalog-ref.yaml`) is touched. Scope matches the brief.

## Requirement-by-requirement

### Isolated/local-compose byte-identical argv

- `env_header_init` (`bootstrap.sh:55-61`) sets `ENV_HEADER=()` and only
  populates it when `ATLAS_MODE` is `sparse`; default (`isolated`, unset, or
  explicit `isolated`) leaves the array empty.
- Both curl call sites expand `"${ENV_HEADER[@]}"` (`bootstrap.sh:76`,
  `:92`), which is zero words for an empty array under bash ≥4.4 — no header
  is appended, no reordering.
- GET argv diff vs. pre-existing: unchanged except the array splice (empty in
  isolated mode) — byte-identical.
- POST argv diff vs. pre-existing: one real change — `-d @/atlas/canonical/tenant.json`
  (hardcoded) became `-d @"$CANONICAL_TENANT_JSON"`. `CANONICAL_TENANT_JSON`
  defaults to that exact literal (`bootstrap.sh:33`,
  `CANONICAL_TENANT_JSON="${CANONICAL_TENANT_JSON:-/atlas/canonical/tenant.json}"`),
  and was already the single source of truth used elsewhere in the same file
  for the baseline preflight and the `canonical_region`/`canonical_major`/
  `canonical_minor` extraction (`bootstrap.sh:294-296`) — so in the
  unmodified-env production path this produces the identical string; only a
  bats fixture (or a future intentional override) would differ. This is the
  correct fix for what the implementer's report calls a latent bug (the old
  POST call was the only place bypassing the override), not scope creep, and
  it does not violate "byte-identical argv" for the actual production default.
- Test coverage: `find_environment_tenant sends no ENVIRONMENT header in
  isolated mode` (`tenant_provisioning_test.bats:111-119`) and
  `create_environment_tenant sends no ENVIRONMENT header in isolated mode`
  (`:184-190`) both assert `! grep -q '^ENVIRONMENT:' "$CURL_ARGS"`, and the
  GET case additionally pins `http://ui/api/tenants` as present, so the
  header-omission is genuinely pinned, not just inferred. PASS.

### Extractability (`sed -n '/^name()/,/^}/p'`)

- `env_header_init()` opens at `bootstrap.sh:55` col 0, closes `}` at `:61`
  col 0 (verified via `awk`).
- `find_environment_tenant()` opens at `:74` col 0; `create_environment_tenant()`
  opens at `:87` col 0 — both grep-confirmed at column 0.
- `tenant_provisioning_test.bats:12-16` extracts all three with the exact
  `sed -n '/^name()/,/^}/p'` pattern and pipes them into one `helpers.sh`.
  PASS.

### `ENV_HEADER` is an array, populated by an extractable function

- `env_header_init` is itself one of the three extracted helpers (not a bare
  top-level assignment) and is the sole writer of `ENV_HEADER`
  (`bootstrap.sh:41-61`). It is called once at top level immediately after
  its own definition (`bootstrap.sh:63`, `env_header_init` bare call) —
  before the tenant-create block runs later in the file. PASS.
- Every expansion site uses `"${ENV_HEADER[@]}"` (`:76`, `:92`) — grep swept
  the whole file for `ENV_HEADER`; only the two writes and two correctly
  quoted `[@]` expansions exist. PASS.

### `set -e` interaction / non-zero-tolerant call sites

- `find_environment_tenant`'s body is `curl | jq | head -1` under `pipefail`
  (inherited from `set -euo pipefail` at `bootstrap.sh:13`, restored to
  `set -e` at `:21` after `lib.sh` resets to `set -uo pipefail` — `pipefail`
  itself is never unset in between). `head -1` on empty input still exits 0,
  and `jq` on an empty `.data[]` selection exits 0, so the "no match" case is
  a *zero-exit, empty-output* result, not a non-zero one — the function never
  legitimately returns non-zero for "no match." The call site
  `existing=$(find_environment_tenant ...)` (`:300`) is therefore safe under
  `set -e` without needing an explicit `|| true`. (It would still abort on a
  genuine curl network failure, matching pre-existing behavior — no
  regression.)
- `create_environment_tenant` genuinely can return 1 (POST failure or missing
  id) and its one call site is `TENANT_ID=$(create_environment_tenant) ||
  exit 1` (`:306`), which explicitly tolerates the non-zero and exits with a
  clear, deliberate `exit 1` rather than letting `set -e` fire silently.
  PASS.
- `env_header_init`'s failure path calls `lib.sh`'s `require_env`, which
  itself calls `exit 1` directly (`lib.sh:27-32`) rather than `return`, so
  the top-level bare call at `bootstrap.sh:63` needs no special handling —
  it terminates the whole script deliberately on missing
  `ATLAS_ENVIRONMENT` in sparse mode, which is the intended fail-loud
  behavior. PASS.

### bats `declare -F` extraction guard

- `tenant_provisioning_test.bats:24-37` guards all three helpers
  (`env_header_init`, `find_environment_tenant`, `create_environment_tenant`)
  with `declare -F ... || { echo "... not extracted ..." >&2; return 1; }`,
  matching the `data_ingest_test.bats:25-34` pattern including the
  failure-message wording style. PASS.

### Canonical POST payload

- `create_environment_tenant` POSTs `-d @"$CANONICAL_TENANT_JSON"`
  (`bootstrap.sh:91`), which resolves to
  `services/atlas-pr-bootstrap/canonical/tenant.json` by default — untouched
  by this diff (confirmed empty diff on that path). PASS.

### `TENANT_ID` / `ENV_HEADER` lifecycle for Task 3

- `ENV_HEADER` is initialized once, unconditionally, at top level
  (`bootstrap.sh:63`), long before the `tenant-create` step
  (`ATLAS_STEP=tenant-create` at `:291`) runs — true on every path, since
  it's not inside any conditional.
- `TENANT_ID` is assigned on both branches of the `tenant-create` `if`:
  `TENANT_ID="$existing"` (`:302`, already-exists path) and
  `TENANT_ID=$(create_environment_tenant) || exit 1` (`:306`, create path).
  Both leave `TENANT_ID` holding a resolved id by the time the block exits;
  downstream `log info "using TENANT_ID=$TENANT_ID" ...` (`:314`) and the
  `REGION=`/`MAJOR_VERSION=`/`MINOR_VERSION=` reassignments (`:311-313`) read
  it unconditionally afterward. PASS.

### Test honesty

- The "core regression" case, `find_environment_tenant does not adopt a
  tenant when the scoped listing is empty` (`tenant_provisioning_test.bats:144-152`),
  genuinely exercises the defect being fixed: it feeds `CURL_BODY='{"data":[]}'`
  and asserts empty output — this only passes because the new code no longer
  falls back to an unscoped listing.
- `find_environment_tenant scopes the listing with the ENVIRONMENT header in
  sparse mode` asserts `grep -qx 'ENVIRONMENT: pr-1411' "$CURL_ARGS"` — pins
  the header is actually sent, not just that the function returns the right
  value.
- Isolated-mode tests assert header *absence* via negated grep, which is the
  correct shape to catch a regression that re-adds the header unconditionally.
- All 15 cases from the brief's table are present with matching names,
  matching the report's stated 15/15 pass count. I did not re-run bats per
  the task instructions; the assertions read as genuinely coupled to the
  implementation (not passing vacuously), and the `declare -F` guard prevents
  the sed-extraction-failure false-pass mode.

## Non-blocking notes

- The POST argv is not byte-identical to the pre-existing script in the
  literal sense of "same source line" (it now interpolates a variable
  instead of a hardcoded path); it is byte-identical in the sense that
  matters — the constraint is about the ENVIRONMENT header not being
  appended/reordered, and the resolved string is identical under the
  production default. Flagging this only so it's a documented, deliberate
  choice rather than a silent side effect the next reviewer has to
  rediscover.
- `data_ingest_test.bats` was not re-read line-by-line to diff the guard
  wording verbatim; the extracted text in this diff matches the described
  pattern (`declare -F ... || { echo "... not extracted from $BOOTSTRAP" >&2; return 1; }`)
  closely enough that a verbatim mismatch, if any, would be cosmetic only
  (message wording), not a functional gap. Noted as not fully cross-checked
  rather than silently assumed identical.

## Not evaluable

- Live bats run and shellcheck diff were not re-executed per the task
  instructions ("Do not re-run the bats suite"); the implementer's report
  carries that evidence (15/15 pass, full suite 129/129, shellcheck
  unchanged finding count). Static review of the test file's assertions
  supports the report's claims but does not independently re-verify runtime
  behavior.

## Verdict rationale

Every constraint in the review brief is satisfied with cited evidence:
isolated-mode argv is unchanged in the way that matters and pinned by tests;
helpers are properly extractable and array-typed; `set -e` is handled
correctly on every new call site; the `declare -F` guard is present; the
canonical payload path is untouched; `TENANT_ID`/`ENV_HEADER` hold on every
path through the block; no mirror-guarded file was touched. No blocking
defect found.
