# Review — fix-atlas-drops-service-id (`0b6917b6e..HEAD`)

Commits reviewed: `d03d34a2b` (Task 1, resolveGroup) and `5c6993106` (Task 2,
environment fallback/reject).

## Scope

`git diff --stat 0b6917b6e..HEAD`:

```
services/administrator.go        | 20 +++++-
services/processor_test.go       | 72 +++++++++++++++++++++-
services/resource.go             | 12 ++++
servicesuniq/migration.go        | 48 +------------
servicesuniq/migration_test.go   | 62 +++++++++++++------
```

Matches the brief's file list exactly. `scope_confirmed`: reviewed both
commits in full (all five changed files), plus `libs/atlas-env/env.go` (the
`env.Self()`/`env.MustFromContext` contract Fix 2 depends on) and
`services/atlas-pr-bootstrap/scripts/bootstrap.sh` +
`deploy/k8s/overlays/{pr,pr-sparse,main}/kustomization.yaml` (the callers and
pod-environment contract Fix 2's fallback depends on, per the specific ask).

## Task 1 — `resolveGroup` fails loudly

**PASS.** `servicesuniq/migration.go:174-224`: `resolveGroup` now only walks
the derived-id match; the newest-history and lowest-id fallback blocks are
deleted entirely (diff `-40 +8`). When no row matches the derived id it
returns `(uuid.Nil, nil, err)` naming every candidate id — nothing else
changed about the derived-id rule, and `Preflight` (read-only) and the
`CREATE UNIQUE INDEX` step are untouched, as required.

**Transaction-boundary check (explicitly requested).** `Migration`
(`migration.go:69-118`):

```go
_, losers, err := resolveGroup(g, rows)
if err != nil {
    return err
}
if len(losers) == 0 {
    continue
}
... database.ExecuteTransaction(db, func(tx *gorm.DB) error { ... DELETE ... })
```

`resolveGroup`'s error is returned *before* `database.ExecuteTransaction` is
ever called for that group — no transaction opens, no `DELETE` statement is
issued, no outbox tombstone is enqueued. `Migration` itself then returns the
error immediately, so the unique-index creation step is also skipped. This
is verified structurally (the delete call is textually inside a closure that
is never reached), not just by the return value. Confirmed further by the
new regression test `TestDedupeErrorsAndDeletesNothingWhenCanonicalRowLosesOnHistory`
(`migration_test.go`), which reproduces the exact production row set
(canonical `00000000-…` with older history vs. two `2026-08-19` interlopers)
and asserts `rowCount == 3` after `Migration` errors.

One pre-existing property worth noting (not a defect, since out of scope):
group-by-group transactions mean a group processed *before* an unresolvable
one can still commit its own deletes in the same `Migration` call. That
matches "no row is removed **for the unresolvable group**" — the brief's ask
— not "the whole migration is one all-or-nothing transaction," which the
brief did not request.

**Test rewrite honesty (explicitly requested).** Both rewritten tests
(`TestDedupeErrorsWhenNoDerivedIdMatchesEvenWithHistory`,
`TestDedupeErrorsWhenNoDerivedIdMatchesAndNoHistory`) assert `err != nil` and
`rowCount` unchanged (2, not 1). Under the pre-fix code both would fail:
`Migration` returned `nil` and deleted down to 1 row. These are real
assertions of the new contract, not passes-either-way edits.

## Task 2 — never insert an empty-environment row

`services/administrator.go:16-24,28-35`:

```go
environment := string(env.MustFromContext(ctx))
if environment == "" {
    environment = string(env.Self())
}
if environment == "" {
    return ErrEnvironmentRequired
}
```

`resource.go` adds `isValidationError` mapping `ErrEnvironmentRequired` to
`server.WriteBadRequest`, mirroring `environments/resource.go:38-42`'s
`isValidationError` / `ErrInvalidPhase` pattern exactly, per the brief's
"patterns to copy" pointer.

### Blocking finding — the fallback breaks the isolated-mode bootstrap path

The brief's Task 2 description reads as if `env.Self()` (the pod's own
environment) is always a meaningful fallback. It is not, for one of the
three modes `bootstrap.sh` drives.

- `services/atlas-pr-bootstrap/scripts/bootstrap.sh:56-59` (`env_header_init`)
  only builds `ENV_HEADER` when `ATLAS_MODE=sparse`; in the default
  (`ATLAS_MODE=isolated`, i.e. legacy non-sparse per-PR environments)
  `ENV_HEADER=()`.
- `upsert_service_config` (`bootstrap.sh:455-514`, the isolated-mode
  login/channel/drops upsert, called from the `ATLAS_MODE != sparse` branch
  at `bootstrap.sh:656-662`) never includes `"${ENV_HEADER[@]}"` on its POST
  (`bootstrap.sh:509-513`) — confirmed by direct read of the POST call,
  unlike the sparse path's `upsert_sparse_service_config` at
  `bootstrap.sh:620-631`, which does send `"${ENV_HEADER[@]}"`.
- `deploy/k8s/overlays/pr/kustomization.yaml:158-165` — the isolated-mode
  `atlas-env` ConfigMap generator — sets `ATLAS_ENVIRONMENT=` (empty) for
  every pod in an isolated PR namespace, and the comment says this is
  deliberate: *"Isolated mode registers no control-plane environment record,
  so it must keep env.Self()=="" (FR-1.5)."*
- `deploy/k8s/base/atlas-configurations.yaml:19-23` — `atlas-configurations`
  gets this value via `envFrom: configMapRef: atlas-env`, so
  `env.Self()` inside `atlas-configurations` in an isolated PR pod is `""`
  by design, not by accident.

Chaining these: for every isolated-mode PR bootstrap (still the *default*
`ATLAS_MODE`, not a deprecated path — `${ATLAS_MODE:-isolated}` appears at
`bootstrap.sh:57,381,633,774`), the login/channel/drops service-config POST
now hits `environment := env.MustFromContext(ctx)` (empty, no header) →
`env.Self()` (empty, by FR-1.5 design) → `ErrEnvironmentRequired` → HTTP 400.
`bootstrap.sh` runs under `set -euo pipefail`
(`bootstrap.sh:13`) and `upsert_service_config`'s POST has no `||` fallback
(unlike the sparse path, which does: `bootstrap.sh:629-631`), so the first
`curl -fsS -X POST` failure aborts the whole bootstrap script.

Before this fix, that same POST inserted `environment=''`, which is exactly
what `FR-1.5` designed for (isolated mode is intentionally "not
environment-aware" — see `libs/atlas-env/env.go:38-39`, "the empty Id is the
legacy value: it means 'not environment-aware'"). After this fix, the
identical request is unconditionally rejected. This is not the incident's
defect 2 recurring — it is a new regression the fix introduces in a mode the
brief did not mention and the diff's tests never exercise.

**Verdict on this point:** wrong for isolated mode, as currently wired. The
`create`-level fallback-then-reject is correct for `main` (whose pod
`ATLAS_ENVIRONMENT=main`, `overlays/main/kustomization.yaml:48`) and for
`sparse` (which always sends the header), but it silently turns every
isolated-mode bootstrap run into a hard failure. Either `administrator.go`'s
`create` needs a way to know "isolated legacy caller, empty environment is
intentional" is still valid, or `bootstrap.sh`'s isolated-mode
`upsert_service_config` needs to send an explicit
`ENVIRONMENT: <legacy-sentinel>` header that the fix's resolution chain can
accept as non-empty — this diff does neither, and ships no test that would
have caught it (all three new `processor_test.go` cases construct contexts
directly; none reproduces the isolated-mode "no header, no pod env" call
shape against the actual bootstrap caller contract).

### Test honesty (explicitly requested)

`services/processor_test.go`'s three new tests
(`TestProcessor_Create_StampsEnvironmentFromContext`,
`TestProcessor_Create_FallsBackToServiceOwnEnvironment`,
`TestProcessor_Create_RejectsWhenNoEnvironmentResolves`) each assert a
distinct, real outcome (stamped value, fallback value, rejection + zero
rows) that the pre-fix `create` (`string(env.MustFromContext(ctx))` with no
fallback/reject) would not produce — genuine new-contract tests, not
edited-to-pass. They are, however, all written against
`NewProcessor(...)`/`envContext(t, ...)` directly; none goes through
`bootstrap.sh`'s actual isolated-mode call shape (no header, pod
`ATLAS_ENVIRONMENT=""`), which is exactly the gap that let the finding above
through a green test suite.

## Not evaluable

- Whether the `pr-cleanup`/reclaim path (`deploy/k8s/overlays/pr-cleanup/postdelete-cleanup.yaml`)
  or any other consumer depends on isolated-mode rows carrying
  `environment=''` — out of this diff's surface; flagged only insofar as it
  reinforces that `''` was a load-bearing value for isolated mode, not
  merely an oversight.

## Summary

Task 1 is correct and directly verified at the transaction boundary, per the
specific ask — approve as-is.

Task 2 is correct for `main` and `sparse` callers but introduces a
regression for the isolated-mode bootstrap path, which the brief's own
target caller list did not include but which is a real, currently-active
(`ATLAS_MODE` defaults to `isolated`) production caller of exactly the
function this commit changed. This is blocking: shipping it as-is will
break every non-sparse PR-environment bootstrap the next time one runs.
