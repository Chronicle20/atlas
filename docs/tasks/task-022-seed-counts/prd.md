# Seed Counts on Bootstrap UI — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-24

---

## 1. Overview

The `/setup` page in atlas-ui already renders a **Game Data** panel where each of the three bootstrap stages (Upload WZ, Extract, Ingest) shows a live-polled badge — `"12 .wz files"`, `"2,341 XMLs extracted"`, `"18,204 documents loaded"` — that updates every five seconds. The **Seed Data** panel immediately below it has no such badges. Each of the eight seed buttons fires its POST and the user has no visible signal that anything changed: no "before" count, no tick-up while the seed runs, no "after" count.

This task adds the same observability contract to Seed Data. Every seed row grows a JSON:API count endpoint on its owning service, a React Query hook that polls it every five seconds, and a badge in the row that shows the count broken down by sub-resource (for rows that seed into multiple tables — e.g. `"148 shops / 2,341 commodities"`). Clicking Seed triggers `onSuccess` invalidation of the corresponding status query so the badge reflects the new state immediately for sync seeds and on the next poll tick for async seeds. Rows whose status endpoint has not responded or has errored show `"—"`, matching the Game Data convention.

The feature is purely additive. Seed endpoints keep their current behavior (sync returns 200 with a result body; async returns 202 and logs completion). The UI does not try to own any in-progress state beyond what React Query already gives us via `mutation.isPending`. "Updating" means the polled count ticks up as the seed processor writes rows — the same contract Game Data relies on.

## 2. Goals

Primary goals:
- Each of the eight Seed Data rows shows a live-polled, per-tenant count badge.
- Clicking Seed visibly updates the badge while the seed runs (for async seeds) and immediately on completion (for sync seeds).
- Compound seeds (Drops, Gachapons, NPC Shops) show a per-sub-resource breakdown, not a single combined number.
- All counts are tenant-scoped via the existing `TENANT_ID`/`REGION`/`MAJOR_VERSION`/`MINOR_VERSION` headers.
- Each count endpoint follows JSON:API envelope conventions consistent with `/api/wz/input`, `/api/wz/extractions`, `/api/data/status`.

Non-goals:
- SSE / websocket progress streams. Polling only.
- Percent-done progress bars. "Updating" means the count ticks up.
- Changing sync vs. async seed handler behavior. Drops and Gachapons stay 202; everything else stays 200.
- In-memory tenant counters or cached aggregates. Each status request runs a plain `SELECT COUNT(*) WHERE tenant_id = ?`.
- Adding count badges outside the `/setup` page.
- Surfacing per-row `updatedAt` in the badge. The count alone is the badge; `updatedAt` is on the wire for possible future use but unused by the initial UI.
- A combined "overall seeded" dashboard widget.
- Retry-on-failure UX. A failed count request renders `"—"` and the next five-second poll retries silently.

## 3. User Stories

- As an operator seeding a fresh tenant, I want to see the current count of seeded rows next to each Seed button, so I know at a glance what state the tenant is already in before I click.
- As an operator who just clicked Seed on Drops, I want the Monster / Continent / Reactor numbers to tick up over the next few polls, so I can tell the async seed is actually running.
- As an operator re-seeding an existing tenant, I want the pre-seed and post-seed numbers to be visible, so I can sanity-check that the seed actually changed something (not a silent no-op).
- As an operator with a compound row like NPC Shops, I want to see both `shops` and `commodities` counts, so I can tell which half of the seed failed if the numbers look wrong.
- As an operator on a tenant where a service is temporarily unavailable, I want the badge to show `"—"` rather than an angry red error banner, so transient blips don't dominate the UI.

## 4. Functional Requirements

### 4.1 Count endpoints — one per seed row

Each service that currently owns a seed endpoint gains a sibling **GET** endpoint that returns tenant-scoped row counts for the tables its seed populates. All endpoints:

- Route mount path mirrors the seed endpoint's sibling (see table below).
- Read tenant from `tenant.MustFromContext(ctx)`.
- Respond with `200 OK`, `Content-Type: application/vnd.api+json`, and a JSON:API envelope as in 4.2.
- Back each attribute with a plain `SELECT COUNT(*) FROM <table> WHERE tenant_id = ?`. No caching, no in-memory counters.
- `updatedAt` attribute — ISO 8601 timestamp of the most recent `UPDATED_AT`/`CREATED_AT` across the counted rows for this tenant. `null` if the table is empty. Use whichever timestamp column already exists; if none exists, return `null`. (The UI does not consume `updatedAt` in v1 but it goes on the wire for parity with Game Data status endpoints.)
- Respond `500 Internal Server Error` with a plain `{"error": "..."}` JSON body on unexpected DB errors. No 202 / async story — the endpoint is trivially synchronous.

| Service | Route | Resource type | Attribute keys (counts) |
|---|---|---|---|
| atlas-drop-information | `GET /drops/seed/status` | `dropsSeedStatus` | `monsterDropCount`, `continentDropCount`, `reactorDropCount` |
| atlas-gachapons | `GET /gachapons/seed/status` | `gachaponsSeedStatus` | `gachaponCount`, `itemCount`, `globalItemCount` |
| atlas-npc-conversations | `GET /npcs/conversations/seed/status` | `npcConversationsSeedStatus` | `conversationCount` |
| atlas-npc-conversations | `GET /quests/conversations/seed/status` | `questConversationsSeedStatus` | `conversationCount` |
| atlas-npc-shops | `GET /shops/seed/status` | `npcShopsSeedStatus` | `shopCount`, `commodityCount` |
| atlas-portal-actions | `GET /portals/scripts/seed/status` | `portalScriptsSeedStatus` | `scriptCount` |
| atlas-reactor-actions | `GET /reactors/actions/seed/status` | `reactorScriptsSeedStatus` | `scriptCount` |
| atlas-map-actions | `GET /maps/actions/seed/status` | `mapActionScriptsSeedStatus` | `scriptCount` |

See `api-contracts.md` for envelope samples per endpoint.

### 4.2 Envelope shape

Every count endpoint returns the same shape, parameterised by attribute keys:

```json
{
  "data": {
    "type": "<resourceType>",
    "id": "<tenantId>",
    "attributes": {
      "<countKey1>": <int>,
      "<countKey2>": <int>,
      "updatedAt": "<rfc3339 string or null>"
    }
  }
}
```

- `id` is the tenant UUID — matches the pattern used by `/api/wz/input` et al. It is not a new synthetic ID.
- Counts are non-negative integers. Empty tables return `0`, not `null`.
- `updatedAt` is a single top-level attribute. Per-sub-resource `updatedAt` is out of scope (a future change can split if it ever matters).

### 4.3 Ingress routing

Every new route falls under an existing nginx `location` block in `deploy/shared/routes.conf` and `deploy/k8s/ingress.yaml`:

- `/drops/seed/status` → `location ~ ^/api/drops(/.*)?$` (drops already routes to `atlas-drop-information`).
- `/gachapons/seed/status` → `location ~ ^/api/gachapons(/.*)?$`.
- `/npcs/conversations/seed/status` → `location ~ ^/api/npcs/conversations(/.*)?$`.
- `/quests/conversations/seed/status` → `location ~ ^/api/quests/conversations(/.*)?$`.
- `/shops/seed/status` → `location ~ ^/api/shops(/.*)?$`.
- `/portals/scripts/seed/status` → (verify — there is a `^/api/portals(/.*)?$` block; `/portals/scripts/...` must land on atlas-portal-actions, which currently owns `/api/portals/scripts/seed`).
- `/reactors/actions/seed/status` → `location ~ ^/api/reactors/actions(/.*)?$`.
- `/maps/actions/seed/status` → `location ~ ^/api/maps/actions(/.*)?$`.

**Requirement:** verify during implementation that each of these paths already routes to the correct upstream. If any path falls through to a wrong upstream, add a more-specific `location` block. This should not require any new upstream blocks — only potentially more-specific patterns.

### 4.4 atlas-ui service layer

Extend `src/services/api/seed.service.ts`:

- Add typed result interfaces, one per endpoint, matching the attribute keys in 4.1.
- Add one getter per endpoint, each using the existing `fetchJsonApi<A>(url, tenant)` helper. No new HTTP machinery required.
- Expose each interface via named export so the hook file can import the types.

### 4.5 atlas-ui React Query hooks

Extend `src/lib/hooks/api/useSeed.ts`:

- Add one `useXxxSeedStatus()` hook per endpoint, each built exactly like the existing `useWzInputStatus`/`useExtractionStatus`/`useDataStatus`:
  - `queryKey` = `['<resourceType>', tenantId]` (or `['<resourceType>', 'none']` when no active tenant).
  - `queryFn` calls the service getter with `activeTenant!`.
  - `enabled: !!activeTenant`, `staleTime: 0`, `refetchInterval: 5000`.
- Extend each `useSeed<Row>()` mutation's `onSuccess` to invalidate the matching status query key. The hooks currently lack any `onSuccess` for most seeds (see `useSeed.ts` lines 20–57) — add one. This makes sync seeds reflect instantly and async seeds refetch on the next five-second tick.
- `useUploadWzFiles`, `useRunWzExtraction`, `useRunDataProcessing` already demonstrate the invalidation pattern; reuse it.

### 4.6 atlas-ui SetupPage

Rework the Seed Data section of `src/pages/SetupPage.tsx`:

- Replace the existing `SeedButton` card renderer (which has no badge slot) with a `GameDataRow`-style row renderer. The eight seed rows and the three game-data rows should look visually consistent — same layout primitive.
- For each seed row:
  - Wire the row's status hook. Data landing in the cached `.data` object drives the badge.
  - Render the badge via a per-row `formatBadge(data)` function that produces strings like:
    - `"—"` — when `data` is undefined (initial load, service unavailable, query error).
    - `"148 shops / 2,341 commodities"` — compound rows, with Intl.NumberFormat separators and pluralisation via the existing `pluralize()` helper.
    - `"184 conversations"` — single-count rows.
    - `"0 scripts"` — when the table is empty (not `"—"` — `0` is a real, polled value).
  - Preserve the existing Seed button behavior (mutation.mutate + toast). Disable the button only while that row's own mutation is pending, matching today's behavior. The polled count keeps updating while the button is disabled.
- Remove the old `SeedButton` component if it has no remaining callers.

### 4.7 Error-state rendering

- Query errors (status !== 2xx, network failure, tenant missing): badge renders `"—"`. No toast, no banner. The next five-second tick silently retries.
- If **every** status query for a row returns 500 and `error` is set, still render `"—"`. The error is observable via `query.error` but intentionally not surfaced to the UI.
- Mutation errors on the Seed button itself continue to render via the existing toast path (unchanged from current behavior).

### 4.8 Pluralisation & formatting

- Existing `formatCount(n)` (uses `Intl.NumberFormat()`) is reused verbatim. No new locale work.
- Existing `pluralize(n, singular, plural)` is reused for every sub-resource label.
- Singular forms for each attribute:
  - drops: `monster drop` / `continent drop` / `reactor drop`
  - gachapons: `gachapon` / `item` / `global item`
  - npc conversations: `conversation`
  - quest conversations: `conversation`
  - shops: `shop` / `commodity`
  - scripts: `script`
- Plural forms are `<singular>s`. Keep them in the UI; not on the wire.

## 5. API Surface

Eight new GET endpoints, all returning the envelope described in 4.2.

- `GET /api/drops/seed/status` → atlas-drop-information
- `GET /api/gachapons/seed/status` → atlas-gachapons
- `GET /api/npcs/conversations/seed/status` → atlas-npc-conversations
- `GET /api/quests/conversations/seed/status` → atlas-npc-conversations
- `GET /api/shops/seed/status` → atlas-npc-shops
- `GET /api/portals/scripts/seed/status` → atlas-portal-actions
- `GET /api/reactors/actions/seed/status` → atlas-reactor-actions
- `GET /api/maps/actions/seed/status` → atlas-map-actions

All require the four tenant headers. Missing headers → same error handling the existing `tenant.MustFromContext` path produces today. Full request/response samples live in `api-contracts.md`.

No existing endpoints change. Seed POST behaviors, routes, and response bodies are untouched.

## 6. Data Model

No new columns, tables, or migrations. Every count is a `SELECT COUNT(*)` over an existing table filtered by `tenant_id`. Tables referenced:

- atlas-drop-information: monster drop table, continent drop table, reactor drop table.
- atlas-gachapons: `gachapon.Model`, `item.Model` (gachapon items), `global.Model` (global items).
- atlas-npc-conversations: npc conversations table, quest conversations table.
- atlas-npc-shops: `shops.Model`, `commodities.Model`.
- atlas-portal-actions: scripts table.
- atlas-reactor-actions: scripts table.
- atlas-map-actions: scripts table.

The design phase (`/design-task`) is responsible for pinning exact table names and the `updatedAt` column per table. Where a table has no `updated_at` equivalent, its endpoint returns `updatedAt: null` rather than synthesising a value.

## 7. Service Impact

| Service | Change |
|---|---|
| atlas-drop-information | New handler + processor + route at `GET /drops/seed/status`. New repository method per sub-table to `COUNT(*)` and aggregate `MAX(updatedAt)`. |
| atlas-gachapons | Same pattern — new status handler covering three sub-tables. |
| atlas-npc-conversations | Two new handlers (NPC + Quest) over their two seed tables. |
| atlas-npc-shops | New status handler covering shops + commodities. |
| atlas-portal-actions | New status handler, single count. |
| atlas-reactor-actions | New status handler, single count. |
| atlas-map-actions | New status handler, single count. |
| atlas-ui | Service-layer getters, React Query hooks with polling + invalidation, `SetupPage.tsx` rendering rework. |
| deploy/shared/routes.conf + deploy/k8s/ingress.yaml | No edits expected; verify existing catch-alls cover the new paths. |

## 8. Non-Functional Requirements

- **Performance.** Each count query is a single indexed `COUNT(*)` per table. With polling every 5s across eight rows and ~13 tables total, the incremental load is <3 queries/s per connected UI session. No caching layer is required; if a specific table ever grows large enough that `COUNT(*)` hurts, the fix is a stats-table cache and is out of scope here.
- **Multi-tenancy.** Every query is scoped by `tenant_id = MustFromContext(ctx).Id()`. Cross-tenant leakage tests live alongside the existing repository tests.
- **Security.** Endpoints are GET-only, tenant-scoped, no write side-effects. Same auth posture as the existing status endpoints — there is no auth layer today, so no change.
- **Observability.** Handlers log errors via the injected `logrus.FieldLogger`. No new metrics required; if one ever matters, follow the pattern of the existing seed handlers.
- **UI responsiveness.** Badge transitions are driven by React Query cache updates; no layout shift — the row layout matches the stable-height pattern `GameDataRow` already uses.
- **Accessibility.** Badges live inside the existing `aria-live="polite"` slot on each row. Updates announce to screen readers the same way Game Data updates already do.

## 9. Open Questions

None blocking scope. Pinned during the design phase:

- Exact `updated_at` column names per table — whichever column already exists. If none, `null`.
- Exact table names and repository package paths — design phase will enumerate.
- Whether the existing `/api/portals(/.*)?$` ingress block is a strict-enough pattern to route `/api/portals/scripts/seed/status` to atlas-portal-actions rather than atlas-portals — verify during implementation.

## 10. Acceptance Criteria

- [ ] Eight new GET endpoints are reachable via the ingress and return the envelope in §4.2 with correct attribute keys.
- [ ] Every endpoint is tenant-scoped — calling the same endpoint for two tenants returns two distinct count sets when the tables diverge.
- [ ] Empty tables return `0`, not `null`, for their count attributes, and return `updatedAt: null`.
- [ ] `SetupPage.tsx` renders a badge on every seed row, consistent with the Game Data rows' layout.
- [ ] On a freshly bootstrapped tenant with no seeded data, every seed row badge reads `0 ...` once its first poll completes.
- [ ] Clicking Seed on a sync row (e.g. NPC Shops) causes the badge to update within one poll tick — typically under 5 seconds — after the mutation resolves.
- [ ] Clicking Seed on an async row (Drops, Gachapons) causes the badge numbers to tick up across multiple polls while the async seed runs.
- [ ] Compound rows (Drops, Gachapons, NPC Shops) display a `/`-separated breakdown with pluralised labels.
- [ ] If a service is down, its row's badge renders `"—"` without a toast or banner; the badge recovers on its own when the service is back.
- [ ] No existing seed endpoint behavior or response body is changed.
- [ ] `docker compose` (or the equivalent dev bring-up) does not require any new volumes, env vars, or migrations.
- [ ] Unit tests exist for each new count processor method (tenant isolation + empty-table case), matching the style of existing processor tests in each service.
- [ ] React Query hooks have tests asserting: polling is enabled only when a tenant is active, and the query key includes the tenant id.
