# Admin Navigation by Blast Radius + Canonical Baselines Manager — Design

Task: task-134-admin-nav-baselines
Status: Approved PRD (`prd.md`), design v1
Created: 2026-07-04

This document records the architecture, the alternatives considered, and the
tradeoffs behind them. Product scope questions were all settled in the PRD
interview; everything here is implementation-level design.

---

## 1. System Overview

Two services change:

- **atlas-ui** — sidebar regrouping, scope-aware tenant switcher, scope banner,
  Setup page de-scoping, new Baselines page, canonical (tenant-free) service
  layer.
- **atlas-data** — one new read-only endpoint, `GET /data/baselines`, listing
  published canonical baselines from MinIO.

The load-bearing backend fact (verified in source): the shared-scope handlers
never key anything off the tenant *ID*. `ResolveScope`
(`services/atlas-data/atlas.com/data/wzinput/scope.go:21`) gates `scope=shared`
only on `X-Atlas-Operator`, and the shared prefix is built from
`t.Region()/t.MajorVersion()/t.MinorVersion()` (`wzinput/status.go:71`). The
shared REST middleware (`libs/atlas-rest/server/handler.go` `ParseTenant`)
requires syntactically valid tenant headers but performs no existence check —
`uuid.Parse` accepts the nil UUID. So the UI can drive every canonical
operation with synthetic headers: nil `TENANT_ID` + picker-derived
`REGION`/`MAJOR_VERSION`/`MINOR_VERSION`. No backend changes are needed for
the existing shared-scope endpoints.

---

## 2. atlas-data: `GET /data/baselines`

### 2.1 Placement and registration

The endpoint lives in the existing `baseline` package alongside
publish/restore. `InitResource` (`baseline/handler.go:19`) currently mounts a
`/data/baseline` subrouter; the new route is registered in the same
`InitResource` but on the parent router as `/data/baselines` (plural, per PRD
§5.1) using `rest.RegisterHandler` (the no-input-body variant, as
`wzinput/resource.go` uses for GETs) with `Methods(http.MethodGet)`.

Handler skeleton mirrors `publishInner` exactly: `mc == nil` → 503 first, then
`X-Atlas-Operator != "1"` → 403, then the listing. The `ParseTenant`
middleware runs anyway (all `rest.RegisterHandler` routes get it), which
satisfies the PRD's "nil-UUID tenant is acceptable, values ignored" clause for
free — the handler simply never reads the tenant from context.

### 2.2 MinIO listing

`storage/minio.Client` has no method that returns per-object keys
(`PrefixStats` aggregates; `RemovePrefix` deletes). Three options:

1. **Extend `minio.Client` with a `List(ctx, bucket, prefix) ([]ObjectInfo, error)`
   method** returning `{Key, Size, LastModified}` per object — **chosen**.
   Mirrors the existing `PrefixStats` style, keeps the `miniogo` dependency
   confined to the storage package, and is reusable (the same gap exists for
   any future prefix-enumeration need).
2. Use `miniogo.ListObjects` directly from the `baseline` package. Rejected:
   breaks the storage abstraction every other caller respects; the `baseline`
   package currently touches MinIO only through `MC.Put`/`MC.Get`.
3. Overload `PrefixStats` to also return keys. Rejected: changes an existing
   API and its callers for no benefit.

### 2.3 Listing algorithm

In the `baseline` package, a new `Lister` (same shape as `Publisher`/`Restorer`:
struct with `MC` + `L`):

1. `List` all objects in `BucketCanonical` under prefix `baseline/regions/`.
2. Keep keys matching `baseline/regions/<region>/versions/<major>.<minor>/documents.dump`
   (exact suffix match on `/documents.dump`, then parse the two path segments;
   `<region>` is taken verbatim, `<major>.<minor>` split on the last `.` and
   parsed as non-negative integers). Keys that don't parse are logged at WARN
   and skipped — the listing never fails on one bad key.
3. For each dump: `publishedAt` = object `LastModified` (RFC3339 UTC),
   `sizeBytes` = object size. The tar header's internal `publishedAt` is
   epoch-zero by design (`publish.go:78`) and is **never** read.
4. Read the `.sha256` sidecar (`ShaKey`) via `MC.Get`; expect 64 hex chars. On
   any error (missing object, short read, non-hex) log WARN and emit
   `sha256: ""` — a partially-published baseline stays visible (PRD §5.1).
5. Sort results by (region, major, minor) ascending so the response — and the
   UI table — is deterministic.

Cost is one list + N sidecar GETs for N baselines (single-digit-to-tens per
PRD §8); no caching.

### 2.4 REST model

New JSON:API output model in `baseline/rest.go`, following the existing
`PublishOutputModel` pattern:

```go
type ListItemModel struct {
    Id           string `json:"-"`       // "<region>/<major>.<minor>" via PublishOutputId
    Region       string `json:"region"`
    MajorVersion int    `json:"majorVersion"`
    MinorVersion int    `json:"minorVersion"`
    Sha256       string `json:"sha256"`
    PublishedAt  string `json:"publishedAt"` // RFC3339
    SizeBytes    int64  `json:"sizeBytes"`
}
func (ListItemModel) GetName() string { return "baselines" }
```

Marshalled as a collection via `server.MarshalResponse[[]ListItemModel]` (the
same marshaller the rest of the codebase uses for slices). `PublishOutputId`
is reused for the id — one composition function for the canonical id format.

### 2.5 Testability seam

`Lister` consumes a narrow local interface rather than `*minio.Client`
directly:

```go
type objectStore interface {
    List(ctx context.Context, bucket, prefix string) ([]minio.ObjectInfo, error)
    Get(ctx context.Context, bucket, key string) (io.ReadCloser, error)
}
```

`*minio.Client` satisfies it; tests inject a map-backed fake. This is what
makes the PRD-required tests (empty list, populated list, missing sidecar)
possible without a live MinIO — the existing `handler_test.go` pattern
(sentinel non-nil client) can only exercise the 503/403 gates. The gate tests
follow `TestPublishRefusesNonOperator` verbatim; the listing-behavior tests
drive `Lister` directly with the fake.

Rejected alternative: interface-ize `minio.Client` globally. Too broad for
this task; a package-local consumer interface is the established Go idiom.

---

## 3. atlas-ui: canonical service layer

### 3.1 Header helper

`src/lib/headers.tsx` gains a sibling to `tenantHeaders`:

```ts
export interface CanonicalSelection {
  region: string;
  majorVersion: number;
  minorVersion: number;
}

export function canonicalHeaders(sel: CanonicalSelection): Headers
// TENANT_ID = 00000000-0000-0000-0000-000000000000
// REGION / MAJOR_VERSION / MINOR_VERSION from sel
// X-Atlas-Operator = 1
```

The nil UUID is a named constant (`CANONICAL_TENANT_ID`) exported from the
same module. Baking `X-Atlas-Operator` into the helper (rather than sprinkling
it per call, as the current shared-scope code does) means a canonical request
cannot be assembled without the operator header — one construction path, no
drift.

### 3.2 Service functions

Per PRD §7, canonical variants live in the existing service files (no new
`canonical.service.ts` — the split is by resource, not by scope):

**`seed.service.ts`** — tenant functions lose their `scope` parameter entirely
(the Setup page becomes incapable of shared requests, PRD §8); new canonical
functions take `CanonicalSelection`:

- `uploadCanonicalWzFiles(sel, file)` → `PATCH /api/data/wz?scope=shared`
- `runCanonicalDataProcessing(sel)` → `POST /api/data/process?scope=shared`
- `getCanonicalWzInputStatus(sel)` → `GET /api/data/wz?scope=shared`
- `getCanonicalDataStatus(sel)` → `GET /api/data/status?scope=shared`

**`baseline.service.ts`** — `publish` changes signature from
`(tenant, region, major, minor)` to `(sel: CanonicalSelection)` (it was always
a shared-scope operation; the tenant argument only fed headers). `restore`
keeps its `Tenant` (it is genuinely tenant-side). New:

- `listBaselines()` → `GET /api/data/baselines`, decoding the JSON:API
  collection into `Baseline[]`. This endpoint needs tenant headers only to
  clear middleware, so it uses `canonicalHeaders` with a fixed dummy selection
  (`region: "NONE", 0.0`) — values documented as ignored by the server.

Rejected alternative: keep `scope` parameters and have the Baselines page pass
`'shared'`. Rejected because the PRD's multi-tenancy-safety NFR demands the
capability be *removed* from the tenant path, not defaulted away — separate
functions make the two blast radii two different types.

### 3.3 Hooks

`useSeed.ts` sheds its `Scope` import and the scope key segment (tenant hooks
key on `[name, tenantId]` again). New `useCanonicalData.ts` hook module:

- Query keys: `['canonical', 'wzInput' | 'dataStatus', region, major, minor]`
  and `['baselines']`.
- `useCanonicalWzInputStatus(sel | null)` / `useCanonicalDataStatus(sel | null)`
  — `enabled: !!sel`, same 5s `refetchInterval` as the Setup page equivalents.
- `useUploadCanonicalWz(sel)`, `useRunCanonicalProcessing(sel)` — mutations
  invalidating the matching canonical status keys.
- `useBaselines()` — plain query, no polling (mutations invalidate it).
- `usePublishCanonicalBaseline(sel)` — invalidates `['baselines']` and the
  selection's `dataStatus` key on success (PRD FR-5.5).

`useBaseline.ts`'s `usePublishBaseline(tenant)` is deleted with the Setup-page
publish row; `useRestoreBaseline` is unchanged.

---

## 4. atlas-ui: navigation and scope awareness

### 4.1 Single route predicate

New `src/lib/deployment-routes.ts`:

```ts
export const DEPLOYMENT_ROUTE_PREFIXES = ['/templates', '/tenants', '/services', '/baselines'];
export function isDeploymentRoute(pathname: string): boolean
// true when pathname === prefix or pathname.startsWith(prefix + '/')
```

This is the **only** definition of "Deployment route"; the tenant switcher and
the scope banner both consume it. The `prefix + '/'` guard prevents false
positives like a hypothetical `/servicesx`. `/tenants/:id/*` subpages match by
construction (FR-2.1).

Rejected alternative: deriving the set from the sidebar items array. Cute but
inverts the dependency — the sidebar renders *links*, the predicate governs
*behavior*; a caption-bearing nav structure shouldn't be the type feeding a
routing predicate. A unit test instead asserts the two stay in sync (every
Deployment-group child URL satisfies `isDeploymentRoute`).

### 4.2 Sidebar (`app-sidebar.tsx`)

The `items` array is regrouped per FR-1.1/1.3 (Operations, Security, Setup,
Deployment; Deployment children ordered Templates, Tenants, Services,
Baselines; "Bootstrap" renamed "Setup"). The item type gains two optional
fields consumed by the render loop:

- `separated?: boolean` — renders a `SidebarSeparator` above the group
  (Deployment only).
- `caption?: string` — muted caption line under the group label inside the
  `CollapsibleTrigger` ("Applies to all tenants").

Icons: Setup group gets its own icon (`Wrench`); Deployment keeps
`MonitorCog` (FR-1.4 allows it). The existing active-state logic
(`pathname === url || startsWith(url + '/')`) is untouched.

### 4.3 Tenant switcher (`app-tenant-switcher.tsx`)

The component reads `useLocation()` and branches **on render only**:

- `isDeploymentRoute(pathname)` → return an inert block: a `SidebarMenuButton`
  styled muted, `aria-disabled`, no `DropdownMenu` wrapper, no chevron, label
  "Deployment-wide" with a secondary line "tenant selection inactive". Not
  focusable as a picker (it is not a button-with-menu; render as a `div` via
  `asChild`).
- Otherwise → the existing dropdown, byte-for-byte.

Because the inert state is purely presentational, FR-2.3 (selection preserved)
is satisfied structurally: `TenantContext` state and `localStorage`
(`tenant-context.tsx`) are never written on route change. No context changes
at all. This is the decisive reason to reject the alternative of modelling
"deployment mode" in `TenantContext` (e.g. `activeTenant = null` on those
routes): that would ripple through every `enabled: !!activeTenant` hook and
risk cache churn and refetch storms on navigation, for zero functional gain.

### 4.4 Scope banner

New `src/components/common/deployment-scope-banner.tsx`: a slim, non-dismissible
callout ("Changes on this page affect all tenants", `Globe` icon, muted-warning
styling consistent with shadcn `Alert`).

**Placement: rendered once in `AppShell`** (`features/navigation/app-shell.tsx`),
directly under the `BreadcrumbBar`, conditioned on `isDeploymentRoute(pathname)`.

- Chosen because FR-3.1 spans four page families *and all their detail/config
  subpages* — Templates alone has six subpages. Per-page placement would be
  ~15 call sites that silently rot when a new subpage is added; the shell
  placement inherits the exact same route predicate as the switcher, so the
  two scope signals can never disagree.
- Tradeoff accepted: the banner sits above the page `<h2>` rather than inside
  each page's header block. That still reads as "near the page header"
  (FR-3.1) and buys structural consistency.

### 4.5 Routing and breadcrumbs

`App.tsx`: one new lazy route `/baselines` → `BaselinesPage`.
`lib/breadcrumbs/routes.ts`: one new `ROUTE_CONFIGS` entry
(`pattern: '/baselines'`, `label: 'Baselines'`, `parent: '/'`). The Setup
breadcrumb label is updated to "Setup" alongside the sidebar rename.

---

## 5. atlas-ui: Setup page (tenant-only)

`SetupPage.tsx` changes are all deletions plus retitling:

- `ScopeToggle` import, `scope` state, and both scope-dependent branches go;
  `useWzInputStatus()`/`useDataStatus()` are called without arguments (the
  hooks lose the parameter).
- The publish row (`showPublishRow`, `handlePublishBaseline`,
  `usePublishBaseline`) is removed.
- Title "Bootstrap" → "Setup"; description per FR-4.4. The Game Data card
  description drops the scope-choice sentence.
- Restore row and all eight seed rows are untouched.
- `ScopeToggle.tsx` and its test are deleted. Its red re-publish warning text
  migrates to the Baselines page confirmation dialog (§6.4).

The `formatBytes` helper currently local to `SetupPage` moves to a shared
`lib/format.ts` since the Baselines page needs identical formatting for the
size column and WZ badge — the only touched shared utility.

---

## 6. atlas-ui: Baselines page

`src/pages/BaselinesPage.tsx`, composed of three sections top-to-bottom.
Page-local state is exactly one value: the current `CanonicalSelection | null`.
It is not persisted (YAGNI — this is a rarely-visited operator page).

### 6.1 Published baselines table

Driven by `useBaselines()`. Columns: Region, Version (`major.minor`), SHA-256
(first 12 chars + copy-full-value button, em-dash when empty), Published-at
(locale-formatted), Size (`formatBytes`). A plain shadcn `Table` — the
`data-table.tsx` machinery (sorting/filtering/pagination) is overkill for tens
of rows and would be the page's only complexity driver. Empty state: short
explanation of what a baseline is and that publishing creates one.

### 6.2 Region/version picker

A `Select` whose options are the deduplicated union of
(region, major, minor) from `useTemplates()` and `useTenants()` (dedup key =
`region/major.minor`, sorted), labelled e.g. `GMS 83.1`, with source badge
omitted (provenance doesn't matter — they're just seeds for the picker).
Plus a final "Custom…" option that reveals three inline fields (region text,
major/minor numeric) with the PRD's validation (non-empty region,
non-negative integers); the custom selection becomes active only once valid.
Zero-tenant deployments degrade gracefully: templates still populate options,
and custom entry always works (FR-5.6).

The picker is a small local component (`BaselineTargetPicker`) inside the
`features/baselines/` folder, taking `value`/`onChange` of
`CanonicalSelection | null` — keeps `BaselinesPage` readable and the picker
independently testable.

### 6.3 Workflow rows

Three `SetupRow`s (component reused as-is per PRD §8), all disabled until a
selection exists:

1. **Upload WZ** — `useUploadCanonicalWz(sel)`; badge from
   `useCanonicalWzInputStatus(sel)` (file count / bytes). Same zip validation
   and 409/400 toast handling as SetupPage (extracted into the mutation's
   `onError` in the hook so both pages share it — no copy-paste divergence).
2. **Process Data** — `useRunCanonicalProcessing(sel)`; disabled until the
   shared WZ status reports files; badge = shared document count from
   `useCanonicalDataStatus(sel)`.
3. **Publish Baseline** — enabled when shared document count > 0.

### 6.4 Publish confirmation

Publish checks the already-loaded baselines list for an entry matching the
selection:

- No existing entry → publish immediately (FR-5.4).
- Existing entry → shadcn `AlertDialog`: "This will replace the shared
  canonical baseline for {region} v{major}.{minor}." (the migrated ScopeToggle
  warning), destructive-styled confirm.

The check intentionally uses the client-side list rather than a server
round-trip: publish is operator-initiated and idempotent-overwrite by design;
a stale list at worst skips one confirmation, it cannot corrupt data. On
success: toast + invalidation of `['baselines']` and the selection's status
keys (FR-5.5).

---

## 7. Error handling summary

| Surface | Failure | Behavior |
|---|---|---|
| `GET /data/baselines` | MinIO nil/unreachable | 503 (gate identical to publish) |
| `GET /data/baselines` | missing/garbled sidecar | row listed with `sha256:""`, WARN log |
| `GET /data/baselines` | unparseable key under prefix | key skipped, WARN log |
| Baselines table | query error | inline error state in the table card, no toast spam (it polls nothing) |
| Canonical upload | 409 concurrent job / 400 rejected | same toast copy as Setup, "for this scope" wording |
| Publish | 5xx | destructive toast with server message (existing `decodeErrorMessage`) |
| Picker custom entry | invalid input | inline field errors; workflow rows stay disabled |

---

## 8. Testing strategy

**atlas-data (Go, table-driven, no live MinIO):**

- Gate tests for the list handler (nil-mc 503, non-operator 403) following the
  existing sentinel-client pattern in `handler_test.go`.
- `Lister` tests against the map-backed `objectStore` fake: empty bucket →
  empty collection; two baselines → parsed, sorted, correct attributes;
  missing sidecar → `sha256:""` present; junk key under prefix → skipped;
  `List` error → error surfaced (→ 500).
- Key-parse unit tests (round-trip with `DumpKey`, malformed segments).
- Standard gates: `go test -race`, `go vet`, `go build`,
  `docker buildx bake atlas-data`, `tools/redis-key-guard.sh`.

**atlas-ui (vitest + testing-library):**

- `deployment-routes`: prefix/exact/subpath matching incl. `/tenants/:id/config`,
  and the sidebar-sync assertion (§4.1).
- Sidebar: four groups in order, Deployment children in order, separator +
  caption render.
- TenantSwitcher: inert on a Deployment route (no dropdown trigger, shows
  "Deployment-wide"), interactive elsewhere, and selection survives a
  Deployment-route round-trip (context value unchanged).
- AppShell: banner present on Deployment routes (incl. a subpage), absent
  elsewhere.
- SetupPage: no scope toggle, no publish row, restore row still gated on
  document count 0.
- `canonicalHeaders`: nil UUID, selection values, operator header.
- BaselinesPage: picker dedup + custom validation + zero-tenant fallback;
  workflow enable/disable ladder; re-publish confirmation appears only when
  the selection already has a baseline; table render incl. empty state and
  blank-sha row.
- Service tests for `listBaselines` decode.

---

## 9. Explicitly out of scope (from PRD non-goals)

Tenant setup under `/tenants/:id/*`, seed-endpoint changes, operator auth
hardening, baseline deletion/retention, restore-from-Baselines-page.

## 10. Decision log

| # | Decision | Alternatives rejected |
|---|---|---|
| 1 | Extend `minio.Client` with `List` | direct miniogo in `baseline` pkg; overloading `PrefixStats` |
| 2 | `Lister` behind package-local `objectStore` interface | global client interface; live-MinIO tests |
| 3 | Inert switcher is render-only; context untouched | "deployment mode" in TenantContext (cache churn, hook ripple) |
| 4 | One `isDeploymentRoute` predicate module | deriving the set from sidebar items (dependency inversion) |
| 5 | Banner mounted once in AppShell | per-page placement (~15 rot-prone call sites) |
| 6 | Separate canonical service functions; tenant functions lose `scope` param | keeping `scope` params (violates capability-removal NFR) |
| 7 | Re-publish confirmation from client-side list | server pre-check round-trip (no correctness gain) |
| 8 | Plain table, single non-persisted selection state | data-table machinery; persisted selection (YAGNI) |
