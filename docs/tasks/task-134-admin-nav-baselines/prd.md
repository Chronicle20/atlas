# Admin Navigation by Blast Radius + Canonical Baselines Manager — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-04
---

## 1. Overview

The atlas-ui sidebar's "Administration" group lumps four pages together alphabetically with no
organizing principle. Three of them (Templates, Tenants, Services) are deployment-wide and ignore
the tenant switcher entirely; the fourth (Bootstrap, `/setup`) is mostly tenant-scoped but hides a
"Canonical (shared)" scope toggle that flips the same upload/process buttons into deployment-wide
canonical baseline operations. The result is context leakage: the sidebar tenant switcher is
visible and interactive on pages it does not affect, and one page mixes two blast radii behind a
toggle.

This task reorganizes the navigation by blast radius and gives canonical game data a real home.
Bootstrap splits into a tenant-scoped **Setup** page (WZ upload/process for the active tenant,
restore-from-baseline, seed rows) and a deployment-wide **Baselines** page (a full canonical
baseline manager keyed by explicit region/version — no active tenant required). The Administration
group is renamed **Deployment** and gains distinct visual treatment; the tenant switcher renders an
inert "Deployment-wide" state on Deployment routes, following the AWS "Global" region-picker
pattern combined with Grafana's separated server-admin area.

A secondary goal falls out naturally: operators can publish canonical data for a region/version
that has **no live tenant yet**. Today canonical operations are hostage to the active tenant purely
because the UI derives REGION/MAJOR_VERSION/MINOR_VERSION headers from it. The backend's shared
scope never reads the tenant ID (`services/atlas-data/atlas.com/data/wzinput/scope.go` gates only
on `X-Atlas-Operator`; the shared prefix is keyed by region/version alone), so the UI can send
synthetic headers instead.

## 2. Goals

Primary goals:

- One navigation rule: everything outside the Deployment group follows the tenant switcher;
  nothing inside it does.
- Remove the dual-scope toggle from the Setup page entirely; each page has exactly one blast
  radius.
- Make the tenant switcher scope-aware: inert "Deployment-wide" state on Deployment routes
  (including `/tenants/:id/*`), normal picker elsewhere.
- Provide a Baselines page that answers "what canonical baselines exist?" and supports
  upload → process → publish for any region/version without an active tenant.
- Add the missing atlas-data endpoint to list published baselines.

Non-goals:

- Moving tenant setup into `/tenants/:id/*` subpages (considered and rejected — heavier refactor
  of the active-tenant-bound seed hooks for little gain in a single-operator tool).
- Any change to seed endpoints or seeding backends.
- Operator authentication hardening beyond the existing `X-Atlas-Operator: 1` header convention.
- Baseline deletion, retention, or versioning-history management.
- Restore-into-tenant from the Baselines page (restore remains a tenant-side action on Setup).

## 3. User Stories

- As an operator, I want the sidebar grouped by blast radius so I always know whether a page
  affects one tenant or the whole deployment.
- As an operator, I want the tenant switcher to visibly disengage on deployment-wide pages so I
  never wonder whether my selection influences what I'm editing.
- As an operator, I want to upload and publish canonical game data for a region/version before any
  tenant of that version exists, so new tenants can bootstrap from the baseline on day one.
- As an operator, I want to see all published canonical baselines (region, version, sha256,
  published-at) in one table so I can verify what new tenants will restore from.
- As an operator, I want tenant-scoped setup (WZ ingest, restore, seeding) on a page that plainly
  operates on the currently selected tenant.

## 4. Functional Requirements

### 4.1 Sidebar reorganization

- FR-1.1: Sidebar groups top-to-bottom: **Operations**, **Security**, **Setup**, **Deployment**.
  Operations and Security contents are unchanged.
- FR-1.2: The Setup group contains a single entry, "Setup", routing to `/setup`.
- FR-1.3: The Deployment group (renamed from Administration) contains, in this order: Templates,
  Tenants, Services, Baselines. Baselines routes to `/baselines`.
- FR-1.4: The Deployment group is visually distinct from the other groups: a separator above the
  group and a muted caption under the group label reading "Applies to all tenants". Icon may
  remain `MonitorCog`.
- FR-1.5: Route paths `/templates*`, `/tenants*`, `/services*`, `/setup` are unchanged; no
  redirects are required. `/baselines` is the only new route.

### 4.2 Scope-aware tenant switcher

- FR-2.1: Define the Deployment route set as any pathname matching `/templates`, `/tenants`,
  `/services`, `/baselines`, or any subpath of them (including `/tenants/:id/*` detail and config
  subpages).
- FR-2.2: On Deployment routes, the tenant switcher renders an inert, non-interactive state
  labeled "Deployment-wide" (visually distinct from the normal picker: muted styling, no dropdown
  affordance, not focusable as a picker).
- FR-2.3: The active tenant selection is preserved, not cleared, while on Deployment routes.
  Navigating back to any non-Deployment route restores the normal interactive picker with the
  prior selection intact.
- FR-2.4: On all non-Deployment routes the switcher behavior is unchanged.

### 4.3 Scope banner

- FR-3.1: Each Deployment page (Templates, Tenants, Services, Baselines — list pages and their
  detail/config subpages) shows a slim, non-dismissible callout near the page header:
  "Changes on this page affect all tenants." Implemented once as a shared component.

### 4.4 Setup page (tenant-scoped, `/setup`)

- FR-4.1: The ScopeToggle component and all `scope === 'shared'` branches are removed from the
  Setup page. All Setup-page requests use `scope=tenant` semantics (the existing default).
- FR-4.2: The Setup page retains: Upload WZ (tenant), Process Data (tenant), Restore Canonical
  Baseline (shown when the tenant's document count is 0, as today), and the eight Seed Data rows.
- FR-4.3: The "Publish Canonical Baseline" row is removed from the Setup page (it moves to
  Baselines).
- FR-4.4: Page title/description updated to reflect tenant scope (e.g. "Setup — prepare the
  selected tenant's game data and seeded services").

### 4.5 Baselines page (deployment-wide, `/baselines`)

- FR-5.1: **Published baselines table.** Lists all published canonical baselines from the new
  atlas-data endpoint (§5.1): columns region, version (`major.minor`), sha256 (truncated with
  copy-to-clipboard for the full value), published-at, size. Empty state with a short explanation
  of what a baseline is.
- FR-5.2: **Region/version picker.** A selector whose options are the deduplicated union of
  (region, majorVersion, minorVersion) combos from existing templates and tenants, plus a
  "Custom…" option exposing free-entry fields (region string, major/minor integers, validated:
  region non-empty, versions non-negative integers). No active tenant is consulted.
- FR-5.3: **Canonical workflow rows**, driven by the picker selection (all requests use synthetic
  headers per §4.6 and `scope=shared`):
  - Upload WZ: `PATCH /api/data/wz?scope=shared` with the zip; status badge from
    `GET /api/data/wz?scope=shared` (file count / bytes for the selected region/version).
  - Process Data: `POST /api/data/process?scope=shared`; disabled until the shared WZ input for
    the selection reports files; status badge from `GET /api/data/status?scope=shared`
    (document count).
  - Publish Baseline: `POST /api/data/baseline/publish` with region/major/minor from the picker;
    enabled when the shared document count for the selection is > 0.
- FR-5.4: **Publish confirmation.** Publishing when a baseline already exists for the selected
  region/version requires an explicit confirmation dialog stating it replaces the shared canonical
  baseline for that region/version (migrating the red warning text the ScopeToggle showed today).
  Publishing a first baseline for a selection needs no confirmation.
- FR-5.5: After a successful publish, the baselines table and the selection's status badges
  refresh (React Query invalidation).
- FR-5.6: The page is fully functional with zero configured tenants (picker falls back to
  templates and custom entry; table and workflow operate normally).

### 4.6 Synthetic headers for canonical requests

- FR-6.1: A canonical-scope header helper produces: `TENANT_ID` = nil UUID
  (`00000000-0000-0000-0000-000000000000`), `REGION`/`MAJOR_VERSION`/`MINOR_VERSION` from the
  picker selection, and `X-Atlas-Operator: 1`.
- FR-6.2: All Baselines-page service calls (wz upload/status, process, data status, baseline
  publish, baselines list) use the canonical helper and take `{region, majorVersion,
  minorVersion}` parameters instead of a `Tenant`.
- FR-6.3: Existing tenant-scoped service functions are unchanged in behavior; shared-scope
  variants no longer accept a `Tenant`.

## 5. API Surface

### 5.1 New: list published baselines (atlas-data)

`GET /data/baselines`

- Auth: requires `X-Atlas-Operator: 1`, else 403 (matching publish/restore). Standard tenant
  headers are required by the shared REST middleware; the nil-UUID tenant is acceptable and the
  tenant values do not affect the response.
- 503 when MinIO is unavailable (matching publish/restore).
- Response: JSON:API collection, resource type `baselines`, id `"<region>/<major>.<minor>"`:

```json
{
  "data": [
    {
      "type": "baselines",
      "id": "GMS/83.1",
      "attributes": {
        "region": "GMS",
        "majorVersion": 83,
        "minorVersion": 1,
        "sha256": "<64 hex chars>",
        "publishedAt": "2026-07-04T12:34:56Z",
        "sizeBytes": 123456789
      }
    }
  ]
}
```

- Implementation notes: list MinIO objects in the canonical bucket under prefix
  `baseline/regions/` (client already supports prefix listing); for each `documents.dump` object,
  parse region/major/minor from the key (`baseline/regions/<region>/versions/<major>.<minor>/documents.dump`,
  per `baseline.DumpKey`), take `publishedAt` from the object's LastModified and `sizeBytes` from
  object size, and read the `.sha256` sidecar for the hash. The tar header's internal
  `publishedAt` is intentionally epoch-zero for hash reproducibility and MUST NOT be used.
  A dump object with a missing/unreadable sidecar is still listed, with `sha256: ""` and a warning
  logged, so a partially-published baseline is visible rather than hidden.

### 5.2 Existing endpoints (consumed, unchanged)

- `PATCH /data/wz?scope=shared` — canonical WZ zip upload (operator-gated via ResolveScope).
- `POST /data/process?scope=shared` — canonical ingest.
- `GET /data/wz?scope=shared`, `GET /data/status?scope=shared` — shared-scope status reads.
- `POST /data/baseline/publish` — body carries region/majorVersion/minorVersion.
- `POST /data/baseline/restore` — unchanged, still invoked from the tenant-scoped Setup page.
- Templates/tenants list endpoints — source the region/version picker options.

No changes to any existing atlas-data handler are required; ResolveScope and the operator gates
already support tenant-ID-independent shared scope.

## 6. Data Model

No database changes. No new persisted entities.

The baselines list is derived at request time from existing MinIO canonical-bucket objects:

- `baseline/regions/<region>/versions/<major>.<minor>/documents.dump` — the baseline tar.
- `baseline/regions/<region>/versions/<major>.<minor>/documents.dump.sha256` — hash sidecar.

Shared WZ input continues to live under `shared/regions/<region>/versions/<major>.<minor>/`.

## 7. Service Impact

### atlas-ui (bulk of the work)

- `src/components/app-sidebar.tsx` — regroup menu items (Operations / Security / Setup /
  Deployment), Deployment visual treatment + caption.
- `src/components/app-tenant-switcher.tsx` — route-aware inert "Deployment-wide" state.
- New shared scope-banner component; applied to Deployment pages.
- `src/pages/SetupPage.tsx` — remove ScopeToggle and shared-scope branches and the publish row;
  retitle. `ScopeToggle.tsx` deleted.
- New `src/pages/BaselinesPage.tsx` (+ route in `App.tsx`, breadcrumb entry) with picker,
  workflow rows, and baselines table.
- `src/services/api/seed.service.ts` / `baseline.service.ts` — canonical variants taking
  region/version; canonical header helper (nil UUID + explicit region/version + operator header);
  new `baselines list` service + React Query hook; hooks for shared-scope status keyed by
  region/version instead of tenant.
- Tests: sidebar grouping, switcher inert-state routing (including `/tenants/:id/*`), SetupPage
  toggle removal, BaselinesPage picker/workflow/table behavior, header-helper correctness.

### atlas-data

- New `GET /data/baselines` handler in the `baseline` package (list from MinIO as §5.1),
  registered in `InitResource`, plus handler tests (operator gate, empty list, populated list,
  missing sidecar, MinIO unavailable).

No other services are affected. `docker buildx bake atlas-data` required before completion
(Go module touched); atlas-ui build + vitest for the frontend.

## 8. Non-Functional Requirements

- **Multi-tenancy safety:** no code path may write tenant-scoped data under the shared prefix or
  vice versa; the Setup page must be incapable of issuing `scope=shared` requests after this
  change (the capability is removed, not hidden).
- **Operator gating parity:** the new list endpoint enforces the same `X-Atlas-Operator` gate as
  publish/restore.
- **Performance:** baselines listing is O(number of baselines) MinIO operations (one list + one
  sidecar read per baseline); baseline count is expected to be single-digit-to-tens. No caching
  required beyond React Query defaults.
- **Observability:** list endpoint logs warnings for unparseable keys or missing sidecars rather
  than failing the whole listing.
- **UX consistency:** shadcn/ui components, existing SetupRow pattern reused on the Baselines
  page; React Query for all server state; toasts via sonner as on the current Setup page.

## 9. Open Questions

None. All scope decisions were resolved during the brainstorming interview (2026-07-04):
split-by-scope layout, own Setup group, full baselines manager with new list endpoint, synthetic
nil-UUID headers, restore stays tenant-side, route names, picker sources, group naming.

## 10. Acceptance Criteria

- [ ] Sidebar shows Operations / Security / Setup / Deployment; Deployment contains Templates,
      Tenants, Services, Baselines with separator + "Applies to all tenants" caption.
- [ ] Tenant switcher is inert and reads "Deployment-wide" on `/templates*`, `/tenants*`
      (including `/tenants/:id/*` subpages), `/services*`, `/baselines*`; normal elsewhere;
      selection survives round-trips.
- [ ] Deployment pages render the shared scope banner.
- [ ] `/setup` has no scope toggle, no publish row, and can only operate on the active tenant;
      restore row still appears when the tenant's document count is 0; all eight seed rows work
      as before.
- [ ] `/baselines` lists published baselines (region, version, sha256, published-at, size) from
      the new endpoint, including an empty state.
- [ ] `/baselines` picker offers template/tenant-derived region/version combos plus validated
      custom entry, and works with zero tenants configured.
- [ ] Upload → process → publish flow on `/baselines` succeeds end-to-end for a selection with no
      active tenant involvement; re-publish over an existing baseline requires confirmation;
      table and badges refresh after publish.
- [ ] All shared-scope requests carry nil-UUID `TENANT_ID`, picker-derived
      `REGION`/`MAJOR_VERSION`/`MINOR_VERSION`, and `X-Atlas-Operator: 1`.
- [ ] `GET /data/baselines` returns the JSON:API collection per §5.1; 403 without the operator
      header; 503 without MinIO; tolerates missing sidecars.
- [ ] atlas-ui: `npm run build` + `npm run test` clean. atlas-data: `go test -race ./...`,
      `go vet ./...`, `go build ./...`, and `docker buildx bake atlas-data` clean;
      `tools/redis-key-guard.sh` clean.
