# Defect 4 — the data-ingest guard fires a destructive restore into a shared database

Fourth and last defect found on the first sparse ephemeral environment
(PR #1411). Its three predecessors — the ApplicationSet overlay path (#1416),
the missing `uuidgen` / `ENVIRONMENT` header (#1418), and the REST gate
rejecting `PROVISIONING` (#1421) — are all confirmed fixed by the run that
surfaced this one.

## What the run proved before it died

Argo operation 19:44:08–19:47:39Z, hook Job `atlas-pr-bootstrap`:

    service-config: sparse service config 44f74713-… created (type=login,   environment=pr-1411)
    service-config: sparse service config 5e3a7cfd-… created (type=channel, environment=pr-1411)
    deployment "atlas-login"   successfully rolled out
    deployment "atlas-channel" successfully rolled out

Rows stamped `environment=pr-1411` rather than `''`, `SERVICE_ID` populated,
and the REST gate admitting a `PROVISIONING` environment. Bootstrap then died
one step later:

    data-ingest: restoring canonical baseline → POST /api/data/baseline/restore
    curl: (22) The requested URL returned error: 500

atlas-data (`atlas-main`, pod `atlas-data-65bc56d57c-7qlds`):

    restore table documents: ERROR: duplicate key value violates unique
    constraint "documents_pkey" (SQLSTATE 23505)
    restore: table loop failed for target=ec876921-c363-4cc6-9c51-5bb8d57f9553
      region=GMS ver=83.1; cleaning partial state

## Root cause

Two independent facts combine.

**1. The restore cannot target a database that already holds the canonical
rows.** `baseline/rewriter.go:15-22` rewrites *only* the `tenant_id` column as
it streams the COPY dump; every other column, primary keys included, is copied
verbatim. Restoring into a populated database therefore collides on
`documents_pkey` regardless of which tenant is targeted. Isolated environments
never hit this — they get an empty Postgres. The first sparse environment
shares the baseline's atlas-data, which already held the canonical rows.

**2. The guard asked the wrong question.** `bootstrap.sh` skipped the restore
only when the *caller's tenant* owned rows, read at the default `scope=tenant`.
Measured live against atlas-main's atlas-data:

    GET /api/data/status                       tenant ec876921-…  → documentCount 0
    GET /api/data/status?scope=shared          canonical 144ba144-… → documentCount 49049

Zero is the truthful answer for that tenant, and it is *also* the normal steady
state: `document/storage.go:73-97` falls back to the version-scoped canonical
tenant (`canonical.TenantId`) on both per-id and paged reads when the caller's
tenant owns nothing. Its own comment says so — "a tenant provisioned after
canonical ingestion has no per-tenant rows, so per-id lookups would silently
succeed via canonical". `ec876921-…` is the only GMS 83.1 tenant and is
atlas-main's real one; owning zero rows is correct, not broken.

So the guard read a truthful zero, concluded "no data", and fired a restore
that could only fail — and, because `cleanupAfterFailure` wipes the target on
failure, would have left the tenant emptied had it owned anything.

Same class as defects 1–3: **sparse reuses a shared backing service where
isolated mode had a private one.**

## The fix

Make the guard mirror atlas-data's read semantics instead of its ownership
model: restore only when the data is genuinely unreachable — the tenant owns
nothing *and* no canonical rows exist. A fresh isolated database satisfies
both, so the restore still runs there and still populates it; after it, the
tenant owns rows and re-runs skip as before.

Applied in both modes rather than behind an `ATLAS_MODE=sparse` branch. The
condition is a property of the database, not of the mode, and a mode-gated
version would leave the same landmine for any future shared-backing-store
deployment.

Secondary, in the same guard: `get_attr` pipes `curl` into `jq`, so its exit
status is jq's and a failed request surfaces as an empty string, not an error.
The old guard treated `""` as "data present" (skip) and `null` as "no data"
(restore) — both silent misreads of a failed request, on the branch that
decides whether to run a destructive operation. The new `document_count`
helper accepts only a plain integer and fails otherwise; the caller aborts.
This is the same silent-empty-value failure mode as defect 2's `uuidgen`.

## Verification

- `services/atlas-pr-bootstrap/test/data_ingest_test.bats`, 10 cases: all 10
  fail against the pre-fix script, all 10 pass after. The suite asserts both
  helpers were extracted before running, because a missing definition exits
  127 and would satisfy every "must fail" assertion for the wrong reason.
- Full bats suite: 106/106.
- `shellcheck` clean on the changed file (the four remaining findings are
  pre-existing SC1091/SC2034 on untouched lines).

### Ingress query-string pass-through (review finding, resolved)

The canonical read appends `?scope=shared` to a URL routed through
`$ATLAS_UI_BASE`, which in the PR namespace resolves to
`http://atlas-ingress.atlas-pr-1411.svc.cluster.local`. Code review flagged
that an ingress stripping query strings would hard-fail every bootstrap at
"could not read the canonical document count" — fail-closed, but an abort.

Checked live against that exact Service (not against atlas-data directly):

    GET /api/data/status                  → id ec876921-…,  documentCount 0
    GET /api/data/status?scope=shared     → id 144ba144-…,  documentCount 49049
      (with X-Atlas-Operator: 1)

Both the query string and the operator header survive the hop. With these
values the new guard computes `docs=0, canon=49049` and skips the restore —
the intended outcome for a sparse environment.

## Not addressed here

`baseline.Restorer` remains unsafe against any populated database — this
change keeps callers away from it rather than making it re-key on copy. That
is a wider change to a path isolated mode depends on, and nothing currently
needs it. Worth revisiting if a second shared-database consumer appears.
