# Task 134 — Context for Implementers

Task: task-134-admin-nav-baselines
Worktree: `.worktrees/task-134-admin-nav-baselines` (branch `task-134-admin-nav-baselines`)
Inputs: `prd.md` (requirements), `design.md` (architecture + decision log), `plan.md` (task-by-task steps).

## What this task builds

1. **atlas-data**: one new read-only endpoint, `GET /data/baselines`, listing published canonical
   baselines from MinIO (region, version, sha256, publishedAt from object LastModified, sizeBytes).
2. **atlas-ui**: sidebar regrouped by blast radius (Operations / Security / Setup / Deployment),
   tenant switcher inert ("Deployment-wide") on Deployment routes, a shared scope banner, a
   tenant-only Setup page (scope toggle and publish row removed), and a new deployment-wide
   `/baselines` page with picker-driven upload → process → publish using synthetic nil-tenant
   headers.

## Load-bearing facts (verified from source)

- **Shared scope never reads the tenant id.** `ResolveScope`
  (`services/atlas-data/atlas.com/data/wzinput/scope.go:21`) gates `scope=shared` only on
  `X-Atlas-Operator: 1`; the shared prefix is `shared/regions/<region>/versions/<maj>.<min>/`
  built from tenant *headers* (`wzinput/status.go:71`). The REST middleware only needs
  syntactically valid tenant headers — the nil UUID parses. This is why the UI can send
  `TENANT_ID: 00000000-0000-0000-0000-000000000000` + picker-derived REGION/MAJOR_VERSION/
  MINOR_VERSION and drive every canonical operation with no live tenant.
- **Baseline object keys** (`baseline/dump.go`): dump
  `baseline/regions/<region>/versions/<major>.<minor>/documents.dump` (`DumpKey`), sidecar
  `...dump.sha256` (`ShaKey`), bucket `mc.Cfg().BucketCanonical`.
- **The tar header's internal `publishedAt` is epoch-zero on purpose** (`publish.go:78`, hash
  reproducibility). The list endpoint must use the MinIO object's LastModified.
- **Gate order convention** (`baseline/handler.go` `publishInner`): nil-mc → 503 first, then
  non-operator → 403, then work. The sentinel-client test pattern
  (`handler_test.go` `nonNilSentinelClient`) depends on the handler not dereferencing `mc`
  before the operator gate.
- **`minio.Client` had no per-key listing** — `PrefixStats` aggregates only. Task 1 adds
  `List(ctx, bucket, prefix) ([]ObjectInfo, error)`; the `baseline` package consumes it through
  a package-local `objectStore` interface so tests inject a map-backed fake.
- **Collection marshalling**: `server.MarshalResponse[[]ListItemModel]` — same as
  `commodity/resource.go:43`. A slice model only needs `GetName`/`GetID`/`SetID`.
- **Route registration**: `baseline.InitResource(db, mc)` is already wired in `main.go:164`;
  the new GET route registers inside it (on the parent router — the existing subrouter is the
  singular `/data/baseline`).
- **UI JSON:API write envelope gotcha**: POSTs to `RegisterInputHandler` endpoints must wrap the
  body as `{ data: { type: GetName(), attributes } }` — publish type is `baselinePublishes`
  (see memory `bug_ui_jsonapi_envelope_required_for_input_handlers`).
- **TenantContext must not be touched for the inert switcher.** `tenant-context.tsx` clears the
  entire React Query cache on tenant-id change (`applyTenant` → `queryClient.clear()`).
  Modelling "deployment mode" as `activeTenant = null` would refetch-storm every
  `enabled: !!activeTenant` hook on navigation. The inert state is render-only (design decision 3).

## Key interfaces produced (cross-task contracts)

- Go: `minio.ObjectInfo{Key,Size,LastModified}`, `(c *Client) List(ctx,bucket,prefix)`,
  `parseDumpKey(key) (region,major,minor,ok)`, `Lister{MC objectStore, Bucket string, L}` with
  `List(ctx) ([]ListItemModel, error)`, `ListItemModel` (`GetName()=="baselines"`, id
  `PublishOutputId(region,major,minor)` = `"GMS/83.1"`), `listInner(mc)` handler.
- TS: `CANONICAL_TENANT_ID`, `CanonicalSelection{region,majorVersion,minorVersion}`,
  `canonicalHeaders(sel)` (only way to build canonical headers; operator header baked in),
  `formatBytes` moved to `@/lib/format`, `isDeploymentRoute(pathname)` in
  `@/lib/deployment-routes` (the ONLY definition of Deployment routes),
  `seedService.{uploadCanonicalWzFiles,runCanonicalDataProcessing,getCanonicalWzInputStatus,getCanonicalDataStatus}(sel,…)`,
  `baselineService.publish(sel)` / `baselineService.listBaselines(): Promise<Baseline[]>`,
  hooks in `@/lib/hooks/api/useCanonicalData` (`useBaselines`, `useCanonicalWzInputStatus`,
  `useCanonicalDataStatus`, `useUploadCanonicalWz`, `useRunCanonicalProcessing`,
  `usePublishCanonicalBaseline`), `showWzUploadErrorToast` exported from `useSeed.ts`,
  `sidebarItems` exported from `app-sidebar.tsx` (sync test), `BaselineTargetPicker` +
  pure helpers `dedupeSelections`/`parseCustomSelection`/`selectionKey`.

## Decisions already made (do not relitigate — see design.md §10)

1. Extend `minio.Client` with `List` (not direct miniogo use in `baseline`, not `PrefixStats` overload).
2. `Lister` behind package-local `objectStore` interface; map-backed fake in tests.
3. Inert switcher is render-only; `TenantContext` untouched.
4. One `isDeploymentRoute` predicate module; sidebar/predicate agreement enforced by unit test.
5. Banner mounted once in `AppShell` (self-conditioning component), not per-page.
6. Separate canonical service functions; tenant functions LOSE the `scope` parameter entirely
   (capability removal, not defaulting — PRD §8 NFR).
7. Re-publish confirmation checks the client-side baselines list (no server pre-check).
8. Plain shadcn `Table` (no data-table machinery); single non-persisted `CanonicalSelection` state.

## Sequencing constraints

- Tasks 1→5 (Go) are independent of Tasks 6→16 (UI); within each service the order matters.
- Task 8 keeps tenant service signatures unchanged (internals refactored); Task 9 migrates
  `publish` + deletes `usePublishBaseline` + removes the Setup publish row in ONE commit;
  Task 10 removes `scope` from the tenant path + deletes `ScopeToggle` in ONE commit.
  Every intermediate commit compiles and passes tests.
- Restore stays tenant-side on Setup (`useRestoreBaseline` untouched throughout).

## Verification commands

- Go (from `services/atlas-data/atlas.com/data/`): `go test -race ./...`, `go vet ./...`,
  `go build ./...`.
- From worktree root: `docker buildx bake atlas-data` (mandatory), `tools/redis-key-guard.sh`.
- UI (from `services/atlas-ui/`): `npm run test` (vitest, jsdom, setup `src/test/setup.ts`),
  `npm run build`, `npx tsc -b --noEmit` for quick typechecks.
- jsdom lacks `window.matchMedia` — sidebar/switcher tests stub it (pattern in plan Tasks 12–13).
- Radix `Select` is jsdom-hostile → picker behavior tested via exported pure helpers;
  BaselinesPage tests stub the picker.

## Out of scope (PRD non-goals)

Tenant setup under `/tenants/:id/*`, seed-endpoint changes, operator auth hardening beyond the
existing header convention, baseline deletion/retention, restore-from-Baselines-page.

## After implementation

Run `superpowers:requesting-code-review` (dispatches plan-adherence + backend + frontend
reviewers) BEFORE opening a PR — mandatory per CLAUDE.md.
