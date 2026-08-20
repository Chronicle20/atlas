# Review: fix(deploy): sparse envs must use the baseline's substrate names, not base's defaults

Commit reviewed: `bffb7db8e9da06bfe0fc7c76c602394458add0d2` on branch
`fix-sparse-baseline-scoping`, worktree
`.worktrees/fix-sparse-baseline-scoping`. Confirmed via `git log --oneline -1`
and `git show --stat` before starting.

Brief: `docs/tasks/task-232-sparse-ephemeral-environments/bug-sparse-baseline-scoping.md`

## Scope

Diff surface (`git diff --stat bffb7db8e~1 bffb7db8e`):

- `libs/atlas-redis/keys.go`, `keys_test.go`
- `deploy/k8s/overlays/pr-sparse/{README.md,kustomization.yaml,patches/db-name-suffix.yaml}`
- `deploy/k8s/overlays/pr/{patches/db-name-suffix.yaml,scripts/gen-db-name-suffix.sh,scripts/gen-topic-config.sh}`
- `tools/sparse-baseline-scoping-guard.sh` (new), `tools/verify.sh` (wiring)
- `docs/tasks/task-232-.../{bug-sparse-baseline-scoping.md (new),design.md,isolation-audit.md,plan.md,prd.md}`

Files read for correctness but not part of the diff (contract dependencies,
per the brief): `libs/atlas-lock/leader.go`,
`services/atlas-data/atlas.com/data/runtime/rest/jobs.go`,
`deploy/k8s/base/atlas-data-ingest-job-template.yaml`,
`.github/workflows/pr-validation.yml`, `tools/pr-sparse-mirror-guard.sh`,
`deploy/k8s/overlays/pr-cleanup/postdelete-cleanup.yaml`,
`services/atlas-pr-bootstrap/scripts/cleanup.sh`, `deploy/k8s/base/kafka-precreate.sh`.

`scope_confirmed`: matches the brief exactly — Kafka topic suffixing, Postgres
DB_NAME suffixing, and the Redis `ATLAS_REDIS_ENV` split, all resolved from
`PLACEHOLDER_BASELINE_ENVIRONMENT`, plus the new guard script and its
`verify.sh` wiring.

## Findings

### 1. Redis: `keyEnv()` fallback is behavior-preserving — PASS

`libs/atlas-redis/keys.go:34-39` reads `ATLAS_REDIS_ENV` first, falling back
to `ATLAS_ENV`. Diffed against the pre-fix code
(`var keyPrefix = computeKeyPrefix(os.Getenv("ATLAS_ENV"))`): when
`ATLAS_REDIS_ENV` is unset (every deployment except the new sparse overlay),
`keyEnv()` returns exactly `os.Getenv("ATLAS_ENV")` — byte-identical input to
`computeKeyPrefix`. Verified with `git show bffb7db8e -- libs/atlas-redis/keys.go`.

`keyPrefix` is still a package-level `var` initialized at import time
(`keys.go:17`), same as before — this commit does not change *when* it's
read, only *what* env var feeds it. Swept the repo for other readers of the
Redis key prefix (`grep -rn "ATLAS_ENV"` across `.go` files, excluding tests):
the only three are `libs/atlas-redis/keys.go`, `libs/atlas-lock/leader.go`,
and `services/atlas-data/.../jobs.go:314`. Other `ATLAS_ENV`-looking hits
(`libs/atlas-env/env.go`, `atlas-configurations/environmentcol/migration.go`,
`atlas-tenants/tenant/environment_migration.go`, `atlas-service/logger.go`)
all read `ATLAS_ENVIRONMENT` (the environment-record id, e.g. `pr-1411`), a
distinct variable — confirmed by grepping each file directly. So the reader
list in the brief is complete.

Test honesty check: reverted `keys.go` to `bffb7db8e~1` with `keys_test.go`
at HEAD and ran `go test -run TestKeyEnv`. Result: build failure,
`./keys_test.go:28:12: undefined: keyEnv` — the new tests do not compile,
let alone pass, without the fix. Genuine regression test.

### 2. `ATLAS_ENV` readers and the `jobs.go` propagation — PASS

`libs/atlas-lock/leader.go:85` (`const EnvVar = "ATLAS_ENV"`) scopes leader
leases and is untouched by this commit — confirmed it must stay
per-deployment, which is exactly why `ATLAS_REDIS_ENV` was split out rather
than repointing `ATLAS_ENV` itself.

`services/atlas-data/.../jobs.go:314` propagates `os.Getenv("ATLAS_ENV")`
onto spawned ingest Jobs, with a comment explaining it exists so the Job's
Redis key prefix matches the parent atlas-data pod's. Traced whether this
comment is stale post-fix: the Job template
(`deploy/k8s/base/atlas-data-ingest-job-template.yaml:56-58`) already carries
`envFrom: configMapRef: atlas-env`, and in `pr-sparse` that ConfigMap sets
`ATLAS_REDIS_ENV=PLACEHOLDER_BASELINE_ENVIRONMENT`
(`kustomization.yaml:316`) while deliberately keeping `ATLAS_ENV` **out** of
the ConfigMap (comment at `kustomization.yaml:289-303`) — `ATLAS_ENV` lives
only on individual container envs via `patches/consumer-group-env.yaml`,
which does include `atlas-data`
(`patches/consumer-group-env.yaml:188`). So on the atlas-data pod:
`ATLAS_ENV=<per-PR>` (container patch), `ATLAS_REDIS_ENV=main` (ConfigMap
envFrom) → `keyEnv()` returns `main`. On the spawned Job pod: `ATLAS_REDIS_ENV`
is inherited from the same ConfigMap via the Job template's own `envFrom` (no
explicit override needed), and `jobs.go` explicitly appends `ATLAS_ENV` to
match the parent's per-deployment value. Both pods land on the same
`keyEnv()` result (`main`) despite different `ATLAS_ENV` values, and the
propagation logic requires no change. Confirmed this also holds for isolated
`overlays/pr` (no `ATLAS_REDIS_ENV` anywhere there, so `keyEnv()` falls back
to `ATLAS_ENV` on both parent and spawned Job, which `jobs.go` already
matches). No defect found here — this was the highest-risk item I checked and
it holds up.

### 3. `pr-sparse` topic/DB_NAME suffixing and CI substitution — PASS

Counted directly against the rendered files:
- `grep -c "^      - .*TOPIC_.*=.*-PLACEHOLDER_BASELINE_ENVIRONMENT$" deploy/k8s/overlays/pr-sparse/kustomization.yaml` → 170.
- `grep -c "value:.*-PLACEHOLDER_BASELINE_ENVIRONMENT" deploy/k8s/overlays/pr-sparse/patches/db-name-suffix.yaml` → 36, matching `grep -c "name: DB_NAME"` → 36.

Ran `tools/sparse-baseline-scoping-guard.sh` directly against the live tree
(not just read it) — all four invariants PASS:
```
sparse-baseline-scoping-guard: PASS - all 170 topic vars end with -PLACEHOLDER_BASELINE_ENVIRONMENT
sparse-baseline-scoping-guard: PASS - atlas-env ATLAS_REDIS_ENV is the baseline's environment
sparse-baseline-scoping-guard: PASS - all 36 DB_NAME values end with -PLACEHOLDER_BASELINE_ENVIRONMENT
sparse-baseline-scoping-guard: PASS - ATLAS_ENV is 'PLACEHOLDER_ATLAS_ENV' everywhere it is set (per-deployment)
```

CI substitution: `.github/workflows/pr-validation.yml:948-956`'s `find`
targets `"$OVERLAY_DIR" deploy/k8s/overlays/pr-cleanup -type f \( -name
'*.yaml' -o -name '*.yml' \)` and runs the sed in-place on every match — this
is a glob over the whole overlay tree, not an enumerated file list, so the
new `patches/db-name-suffix.yaml` is covered automatically. Confirmed the sed
includes `-e "s|PLACEHOLDER_BASELINE_ENVIRONMENT|${BASELINE_ENVIRONMENT}|g"`
(`pr-validation.yml:955`).

Generator byte-identity claim: diffed `kustomize build overlays/pr` and
`kustomize build overlays/main` before (`bffb7db8e~1`) vs after (`bffb7db8e`)
— both `diff` calls produced no output (confirmed "PR IDENTICAL" / "MAIN
IDENTICAL"). Default generator invocations reproduce checked-in output.

*(Note: mid-review I accidentally popped an unrelated pre-existing stash
while doing this before/after comparison, which produced a `go.work.sum`
merge conflict. I reverted it immediately with `git checkout HEAD --
go.work.sum`; the stash was preserved (git keeps a stash on a conflicting
pop) and the worktree verified clean and back at `bffb7db8e` afterward. No
lasting effect on the reviewed tree.)*

### 4. `patches/db-name-suffix.yaml` addition vs. mirror-guard and teardown — PASS

`tools/pr-sparse-mirror-guard.sh`'s `MIRRORS` array (lines 32-42) does **not**
include `patches/db-name-suffix.yaml` — it enumerates
`atlas-env-tokens.yaml`, `ingress-route.yaml`, `sync-bootstrap.yaml`,
`predelete-purge.yaml`, `postsync-pihole-add.yaml`, and three other
`patches/*.yaml` files, none of which is `db-name-suffix.yaml`. So the guard
will not flag the new file as a stale mirror even though it's deliberately
not byte-identical to `../pr`'s version (different suffix token). Correct.

`deploy/k8s/overlays/pr-cleanup/postdelete-cleanup.yaml` was not touched by
this commit (not in the diff) — it invokes
`services/atlas-pr-bootstrap/scripts/cleanup.sh`. Read `cleanup.sh`'s
`do_drop_dbs` (line 358): gated on `if [ "${ATLAS_MODE:-isolated}" =
"sparse" ]` and returns 0 immediately, logging "skipped (sparse) — databases
are shared with main" — it never reaches the `DROP DATABASE` loop for
sparse-mode teardown regardless of how the databases are named. So the
baseline-suffix naming this commit introduces cannot cause teardown to drop a
shared baseline database; the sparse path was already a no-op before this
change and stays a no-op after it. Same pattern confirmed for
`do_drop_topics` and `do_drop_redis` (both gated identically). No regression.

One non-blocking finding here (see below): the `do_drop_redis` comment
(`cleanup.sh:434-438`) is now stale.

### 5. `tools/sparse-baseline-scoping-guard.sh` asserts what it claims, non-vacuously — PASS

Read the script fully (`tools/sparse-baseline-scoping-guard.sh:76-147`) and
ran it live rather than trusting the read:

- Invariant 0 (precondition): `if env_cm is None: fail(...); sys.exit(status)`
  (lines 82-84) — if the overlay stopped rendering an `atlas-env` ConfigMap
  entirely, the guard fails immediately rather than silently skipping the
  remaining checks. Not vacuous.
- Invariant 1 (topics): `if not topics: fail(...)` (line 91) before checking
  suffixes — an overlay that stopped emitting topic vars at all fails, it
  doesn't pass by having nothing to iterate.
- Invariant 2 (Redis): `data.get("ATLAS_REDIS_ENV") != baseline token` fails
  on `None` (unset) just as much as on a wrong value (line 102).
  Invariant 3 (DB_NAME): `if not db_names: fail(...)` (line 118) — same shape.
- Invariant 4 (ATLAS_ENV): `if not atlas_envs: fail(...)` (line 139) — an
  overlay where nothing sets `ATLAS_ENV` fails outright.

All four guard branches require evidence to pass, not merely absence of
contradicting evidence. Live run above (§3) confirms all four actually PASS
against the current rendered `pr-sparse`.

### Documentation consistency

`design.md`, `isolation-audit.md`, `plan.md`, `prd.md` (FR-4.8), and
`pr-sparse/README.md` are all updated with explicit "Corrected 2026-08-20"
notes retracting the "unsuffixed"/"inert" premise the bug diagnosis
identified, each cross-linking `bug-sparse-baseline-scoping.md`. Read the
full diff for each; no residual "unsuffixed baseline topics" or "inert"
language survives uncorrected in the touched files.

## Non-blocking

- `services/atlas-pr-bootstrap/scripts/cleanup.sh:434-438` — the
  `do_drop_redis` comment still reads: "Sparse mode never prefixes Redis keys
  with ATLAS_ENV — the per-env key prefix is inert there (design §9,
  computeKeyPrefix("") is the legacy/shared path)." This is exactly the
  "inert" premise this commit's own diagnosis (`bug-sparse-baseline-scoping.md`)
  and `design.md`'s "Corrected 2026-08-20" note prove false — the prefix in
  sparse mode is now `main:atlas:...` (or the resolved baseline id), not
  inert, and reaches that value via `ATLAS_REDIS_ENV`, not by leaving
  `ATLAS_ENV` unset. The function's *behavior* is still correct by
  coincidence (it scans `${ATLAS_ENV}:*`, the per-PR value, which never
  matches sparse-mode keys — those are keyed by the baseline's
  `ATLAS_REDIS_ENV` value instead — so the skip-in-sparse-mode branch above it
  makes the stale reasoning moot at runtime), but the comment documents a
  premise this same commit retracted everywhere else it appears. Since
  `cleanup.sh` is outside this commit's diff, this is a leftover the fix
  didn't sweep, not a functional regression — flagging for a follow-up
  touch-up rather than blocking.

## Not evaluable

- MinIO bucket naming under sparse mode — explicitly out of scope per the
  brief's diagnosis doc ("Not investigated") and not touched by this commit.
- Whether `PLACEHOLDER_BASELINE_ENVIRONMENT` is actually resolved correctly
  end-to-end in a live CI run (the "Resolve baseline environment" step in
  `pr-validation.yml` upstream of the substitution step) — this is CI
  runtime behavior outside a static diff review; the static wiring (glob
  coverage, sed token) checks out, but I did not execute the workflow.

## Verdict rationale

Every item the brief flagged as highest-risk was traced by hand and holds:
the Redis fallback is behavior-preserving and test-verified, the reader list
is complete, the `jobs.go` propagation still produces matching key prefixes
in sparse mode via ConfigMap `envFrom` + explicit `ATLAS_ENV` override, the
mirror-guard correctly excludes the new file, and sparse-mode teardown is a
no-op regardless of the naming change. The guard script asserts real,
non-vacuous invariants and was confirmed to pass by direct execution. The one
finding (stale comment in `cleanup.sh`) is documentation-only and outside the
diff; it does not affect correctness.
