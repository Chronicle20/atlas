# task-174 — Implementation Context

Codebase pointers, patterns, and gotchas for executing `plan.md`. Read this
first; it names the exact files the plan's "mirror the existing idiom" steps
refer to.

## Where things live

- **atlas-data service root:** `services/atlas-data/atlas.com/data/` (module
  `atlas-data`; imports are `atlas-data/<pkg>`).
- **MinIO client:** `services/atlas-data/atlas.com/data/storage/minio/client.go`
  — `Client` wraps `*miniogo.Client` (`minio-go/v7`). Existing methods:
  `Put/Get/Stat/RemovePrefix/PrefixStats/List`, `Cfg() Config`. `Config` has
  `BucketWZ/BucketAssets/BucketRenders` (env `MINIO_BUCKET_WZ/ASSETS/RENDERS`,
  defaults `atlas-wz/atlas-assets/atlas-renders`). `ObjectInfo{Key,Size,LastModified}`.
- **Existing purge (pattern to mirror):** `services/atlas-data/atlas.com/data/tenantpurge/`
  (`handler.go` = route + operator gate; `purge.go` = `tenants/<id>/` prefix
  removal across the 3 buckets). This is the single-tenant analogue of the new
  bulk reconcile.
- **Route registration:** `main.go:163-189` — `server.New(l)....AddRouteInitializer(<pkg>.InitResource(...)(GetServer()))`.
  Add the new `minioreconcile.InitResource(mc)(GetServer())` after
  `tenantpurge.InitResource` (line 174).
- **JSON:API input-handler idiom:** `baseline/handler.go` uses
  `rest.RegisterInputHandler[InputModel](l)(si)("name", innerFn)` where `innerFn`
  is `func(d *rest.HandlerDependency, c *rest.HandlerContext, input InputModel) http.HandlerFunc`.
  `baseline/rest.go` shows the `GetName/GetID/SetID/SetToOneReferenceID/SetToManyReferenceIDs`
  boilerplate a JSON:API model needs (copy it verbatim, change the type name).
- **canonical package:** `services/atlas-data/atlas.com/data/canonical/canonical.go`
  — `const TenantUUID = "00000000-0000-0000-0000-000000000000"`,
  `IsCanonical(id uuid.UUID, region string, major, minor uint16) bool`. The
  reconcile executor uses only `TenantUUID` (no per-uuid version available), and
  that is sufficient: canonical version data is under `shared/` and
  `atlas-canonical/baseline/`, never `tenants/<uuid>/`.

## Orchestrator + hook

- **Shell toolbox:** `services/atlas-pr-bootstrap/scripts/lib.sh` — `log`,
  `require_env`, `retry`, `record_error`, `run_phase`, `summarize_phases`,
  `http_ok`, `compute_atlas_env`. The new orchestrator reuses these; the
  hardened hook reuses `retry`.
- **Purge hook to harden:** `services/atlas-pr-bootstrap/scripts/predelete-purge.sh`
  — `do_purge_tenants` enumerates `/api/tenants`, DELETEs each. Add retry around
  the DELETE only; keep the empty-list refusal and non-zero-exit guarantees.
- **JSON building idiom:** `bootstrap.sh` (~line 415) builds request bodies with
  `jq -cn --arg ... '{data:{type:...,attributes:{...}}}'`. Mirror it; use
  `--argjson` for numbers/bools (`minAgeHours`, `dryRun`).
- **bats harness:** `services/atlas-pr-bootstrap/test/*.bats` (e.g.
  `predelete_test.bats`, `reclaim_test.bats`). Tests stub `kubectl`/`curl` via
  executables in `$BATS_TEST_TMPDIR` and export `KUBECTL`/`CURL`. The new
  scripts read those env vars (default to real binaries) precisely so bats can
  inject stubs.

## K8s

- **Base manifests:** `deploy/k8s/base/*.yaml`, listed in
  `deploy/k8s/base/kustomization.yaml`. There is currently **no** CronJob in the
  tree — the new one is the first; follow the ServiceAccount/ConfigMap style of
  neighboring manifests.
- **atlas-pr-bootstrap image:** pinned in `deploy/k8s/overlays/pr/sync-bootstrap.yaml:96`
  (`ghcr.io/chronicle20/atlas-pr-bootstrap/atlas-pr-bootstrap:<tag>`). Use the
  same repo; confirm the script install path from
  `services/atlas-pr-bootstrap/Dockerfile`.
- **Overlays:** `deploy/k8s/overlays/main/` and `.../pr/`. The CronJob must run
  only in `main`; check whether `pr` pulls base wholesale and exclude it there
  if so (see plan Task 5 Step 4).

## Live-environment facts (for reference, not to hardcode)

- Shared MinIO: `minio.minio.svc.cluster.local:9000`, `minio` namespace,
  single-node `xl-single`, data at `/data`, no `mc` binary in the pod.
- atlas-data internal service: port **8080** (`atlas-data.atlas-main.svc.cluster.local:8080`).
- Live `atlas-main` tenants at time of writing (6): `4936dff2` v84, `86da65d2`
  v87, `abedf3b4` jms185, `c794c706` v95, `db1dbfb3` v92, `ec876921` v83. Only
  `ec876921` currently has per-tenant `tenants/<id>/` objects (in renders); the
  others use the shared/canonical scope. The reconcile keep-list is computed
  live per run — do not hardcode these.

## Gotchas

- **Go workspace/guards:** run guards from repo root; per project memory, guards
  work from root without a global `GOWORK=off`, but a raw `go` command that needs
  module isolation may want `GOWORK=off`. `go test/vet/build` run inside
  `services/atlas-data/atlas.com/data` (the module dir).
- **No goroutines / no raw redis:** reconcile spawns neither — the goroutine and
  redis guards should stay clean without `//goroutine-guard:allow`.
- **Do not create `*_testhelpers.go`** (project rule) — the `newTestHandler`
  helper lives inside `handler_test.go`.
- **Injected clock, not `time.Now()`** inside `Reconcile` (deterministic age
  tests). The handler passes `time.Now` (the function) and calls it once.
- **Reconcile is MinIO-only** — no `*gorm.DB` parameter anywhere in the new
  package.
