# Seed Counts on Bootstrap UI — Design Document

Version: v1
Status: Approved
Created: 2026-04-24

Companion to `prd.md`, `api-contracts.md`, and `ux-flow.md`. This design pins the architectural decisions made during Phase 2 brainstorming. The `/plan-task` phase consumes this document plus the PRD as its input.

---

## 1. Overview

Add eight new tenant-scoped JSON:API `GET /…/seed/status` endpoints across seven backend services and a matching React Query + `SetupRow` rendering layer in atlas-ui. Every new endpoint is additive; no existing seed endpoint, route, or response body changes.

---

## 2. Backend architecture

### 2.1 Pattern (applied to every service)

For each sub-resource package that owns a counted table, add one new `Count` method to its existing `Processor` interface returning `(count int64, maxUpdatedAt *time.Time, err error)`. The method stays inside the sub-resource package that owns the entity, keeping the `entity` struct unexported.

Implementation shape (mirrors `atlas-data/atlas.com/data/data/status.go#queryStatus`):

```go
func (p *ProcessorImpl) Count() (int64, *time.Time, error) {
    var count int64
    if err := p.db.WithContext(p.ctx).
        Model(&entity{}).
        Where("tenant_id = ?", p.t.Id()).
        Count(&count).Error; err != nil {
        return 0, nil, err
    }
    if count == 0 {
        return 0, nil, nil
    }
    // Tables without an updated_at column return (count, nil, nil) here.
    row := p.db.WithContext(p.ctx).
        Model(&entity{}).
        Where("tenant_id = ?", p.t.Id()).
        Select("MAX(updated_at)").
        Row()
    var raw sql.NullString
    if err := row.Scan(&raw); err != nil {
        return 0, nil, err
    }
    if !raw.Valid || raw.String == "" {
        return count, nil, nil
    }
    t, err := parseDBTime(raw.String)
    if err != nil || t.IsZero() {
        return count, nil, nil
    }
    return count, &t, nil
}
```

Reuse `parseDBTime` semantics from `atlas-data/atlas.com/data/data/status.go` (tolerant format list; returns zero on failure). Copy the helper into each service rather than introducing a shared lib — roughly 15 lines of duplicated code is cheaper than a new dependency.

### 2.2 Status handler (co-located with seed)

Each `seed` package gets a new `status.go` alongside `resource.go`/`processor.go`. It defines:

- A JSON:API `<ResourceType>RestModel` matching the attribute keys in PRD §4.1. `GetName()` returns the resource type (e.g., `"dropsSeedStatus"`). `GetID()` returns the tenant UUID string.
- `handleGetSeedStatus` — reads tenant from `tenant.MustFromContext(ctx)`, calls each sub-resource processor's `Count()` in parallel via `errgroup.WithContext` guarded by `sync.Mutex`, aggregates into the RestModel, and marshals via `server.MarshalResponse`.
- Extension of the existing `InitResource` in the seed package to register the `GET` route alongside the existing `POST`.

### 2.3 Concurrency

Compound handlers (drops: 3, gachapons: 3, shops: 2) execute their sub-counts in parallel using `errgroup.WithContext` — the same pattern `atlas-drops-information/atlas.com/dis/seed/processor.go` already uses for the seed side. Results accumulate into a struct guarded by a `sync.Mutex`. Single-count handlers (portal-scripts, reactor-scripts, map-scripts, npc-conversations, quest-conversations) skip the errgroup and call `Count()` directly.

### 2.4 Error handling

| Failure | Response |
|---|---|
| Any `Count()` returns `err` | `errgroup.Wait()` propagates; handler logs `d.Logger().WithError(err).Errorf(...)`, writes `500`, body `{"error": "<err.Error()>"}` encoded as plain JSON. |
| `tenant.MustFromContext` panic (missing headers) | `rest.RegisterHandler`'s existing recover middleware returns 500. Matches existing seed POST behavior; no new path needed. |
| `Select("MAX(updated_at)")` returns a null/unparseable string | Return `(count, nil, nil)` — the status endpoint succeeds with `updatedAt: null`. |

No retry loop. No partial response on errgroup failure. No caching layer.

### 2.5 `updatedAt` aggregation

For compound handlers, aggregate = `MAX` across non-nil timestamps; `nil` if all are nil. Serialised as `time.Time.UTC().Format(time.RFC3339)`.

Structural outcome:

| Handler | `updatedAt` in practice |
|---|---|
| drops | Always `null` (all three sub-tables lack the column) |
| gachapons | Always `null` (all three sub-tables lack the column) |
| shops | ISO-8601 string from `MAX(shops.updated_at, commodities.updated_at)` via `gorm.Model`; `null` only when empty |
| npc-conversations | ISO-8601 from `MAX(updated_at)`; `null` when empty |
| quest-conversations | same |
| portal-scripts | same |
| reactor-scripts | same |
| map-scripts | same |

The UI ignores `updatedAt` in v1; it ships on the wire for parity with Game Data and possible future use. No migration to backfill timestamp columns on drops/gachapons is part of this task — re-seeding is the operator's workaround until a follow-up task migrates those tables.

### 2.6 No shared library

Each service owns its own `Count` methods and its own `StatusRestModel`. A cross-service helper would force every service onto a new dependency for ~30 lines of duplicated code, and the JSON:API RestModel types are already per-service. Duplication is clearer than abstraction here.

---

## 3. Per-service change inventory

### 3.1 atlas-drop-information

- `monster/drop/processor.go` — add `Count()` to `Processor` interface + `ProcessorImpl`. Always returns `(n, nil, nil)` (no `updated_at` column).
- `continent/drop/processor.go` — same shape.
- `reactor/drop/processor.go` — same shape.
- `seed/status.go` (new) — `DropsSeedStatusRestModel` (`MonsterDropCount`, `ContinentDropCount`, `ReactorDropCount`, `UpdatedAt`), `handleGetSeedStatus`, errgroup of three `Count()` calls.
- `seed/resource.go` — register `GET /drops/seed/status` in `InitResource`.

### 3.2 atlas-gachapons

- `gachapon/processor.go`, `item/processor.go`, `global/processor.go` — add `Count()`. All three tables lack `updated_at`; always `(n, nil, nil)`.
- `seed/status.go` (new) — `GachaponsSeedStatusRestModel` (`GachaponCount`, `ItemCount`, `GlobalItemCount`, `UpdatedAt`), errgroup of three.
- `seed/resource.go` — register `GET /gachapons/seed/status`.

### 3.3 atlas-npc-conversations

Two independent status handlers, one per sub-package. No cross-package aggregation — each conversation type reports only its own table.

- `npc/conversation/npc/processor.go` — add `Count()` (table has explicit `updated_at`).
- `npc/conversation/quest/processor.go` — same shape.
- `npc/conversation/npc/seed.go` — add `handleGetSeedStatus` returning `npcConversationsSeedStatus`; register `GET /npcs/conversations/seed/status`.
- `npc/conversation/quest/seed.go` — symmetric; register `GET /quests/conversations/seed/status` returning `questConversationsSeedStatus`.

### 3.4 atlas-npc-shops

- `shops/processor.go` — add `Count()` (`gorm.Model` supplies `updated_at`).
- `commodities/processor.go` — same.
- `npc/seed/status.go` (new) — `NpcShopsSeedStatusRestModel` (`ShopCount`, `CommodityCount`, `UpdatedAt`), errgroup of two.
- `npc/seed/resource.go` — register `GET /shops/seed/status`.

### 3.5 atlas-portal-actions

- `portal/script/processor.go` — add `Count()` (has explicit `CreatedAt`/`UpdatedAt`).
- `portal/script/seed.go` — add `handleGetSeedStatus` and route registration for `GET /portals/scripts/seed/status`. Single-count handler; no errgroup.

### 3.6 atlas-reactor-actions

- `reactor/script/processor.go` — add `Count()`.
- `reactor/script/seed.go` — handler + `GET /reactors/actions/seed/status`. Single-count.

### 3.7 atlas-map-actions

- `map-actions/script/processor.go` — add `Count()`.
- `map-actions/script/seed.go` — handler + `GET /maps/actions/seed/status`. Single-count.

### 3.8 Ingress

Only the drops route needs a change. `deploy/shared/routes.conf` line 151 is currently:

```nginx
location ~ ^/api/drops/seed$ {
  set $u "atlas-drop-information:8080";
  proxy_pass http://$u$request_uri;
}
```

The `$` anchor means `/api/drops/seed/status` falls through to `^/api/drops(/.*)?$` → `atlas-drops` (wrong upstream). Fix: broaden to `^/api/drops/seed(/.*)?$`. Apply the same edit in `deploy/k8s/ingress.yaml`.

The other seven paths route correctly today via existing catch-all regex blocks:

| Path | Matches block |
|---|---|
| `/api/gachapons/seed/status` | `^/api/gachapons(/.*)?$` → `atlas-gachapons` |
| `/api/npcs/conversations/seed/status` | `^/api/npcs/conversations(/.*)?$` → `atlas-npc-conversations` |
| `/api/quests/conversations/seed/status` | `^/api/quests/conversations(/.*)?$` → `atlas-npc-conversations` |
| `/api/shops/seed/status` | `^/api/shops(/.*)?$` → `atlas-npc-shops` |
| `/api/portals/scripts/seed/status` | `^/api/portals(/.*)?$` → `atlas-portal-actions` (specific `portals/blocked` block precedes it) |
| `/api/reactors/actions/seed/status` | `^/api/reactors/actions(/.*)?$` → `atlas-reactor-actions` |
| `/api/maps/actions/seed/status` | `^/api/maps/actions(/.*)?$` → `atlas-map-actions` |

### 3.9 Summary

7 services · 13 new `Count` methods · 8 new status handlers · 1 ingress broadening.

---

## 4. Frontend architecture

### 4.1 File layout

New file `src/components/features/setup/SetupRow.tsx` houses the row primitive plus its formatting helpers. Named exports:

```tsx
export function SetupRow({ icon, label, badge, action, warning }: SetupRowProps);
export function formatCount(n: number): string;
export function pluralize(n: number, singular: string, plural: string): string;
```

The component body is the current `GameDataRow` implementation verbatim. `formatCount` and `pluralize` are the two helpers already defined inline in `SetupPage.tsx`.

`SetupPage.tsx` imports `SetupRow`, `formatCount`, `pluralize` from the new file; deletes the inline `GameDataRow` component, the `formatCount`/`pluralize` helpers, and the old `SeedButton` component (unused after the refactor). Game Data rows switch from `<GameDataRow …/>` to `<SetupRow …/>`. `formatBytes` stays in `SetupPage.tsx` — only the WZ upload row uses it.

### 4.2 Service layer (`src/services/api/seed.service.ts`)

Add 8 result interfaces and 8 getters. Every getter is one line of `fetchJsonApi<A>` reuse:

```tsx
export interface DropsSeedStatus {
  monsterDropCount: number;
  continentDropCount: number;
  reactorDropCount: number;
  updatedAt: string | null;
}
export interface GachaponsSeedStatus {
  gachaponCount: number;
  itemCount: number;
  globalItemCount: number;
  updatedAt: string | null;
}
export interface NpcConversationsSeedStatus { conversationCount: number; updatedAt: string | null; }
export interface QuestConversationsSeedStatus { conversationCount: number; updatedAt: string | null; }
export interface NpcShopsSeedStatus { shopCount: number; commodityCount: number; updatedAt: string | null; }
export interface PortalScriptsSeedStatus { scriptCount: number; updatedAt: string | null; }
export interface ReactorScriptsSeedStatus { scriptCount: number; updatedAt: string | null; }
export interface MapActionScriptsSeedStatus { scriptCount: number; updatedAt: string | null; }
```

Getter example:

```tsx
async getDropsSeedStatus(tenant: Tenant): Promise<DropsSeedStatus> {
  return fetchJsonApi<DropsSeedStatus>('/api/drops/seed/status', tenant);
}
```

### 4.3 React Query hooks (`src/lib/hooks/api/useSeed.ts`)

Add 8 `useXxxSeedStatus()` hooks, each mirroring `useWzInputStatus`:

```tsx
const dropsSeedStatusKey = (tenantId: string) => ['dropsSeedStatus', tenantId] as const;

export function useDropsSeedStatus(): UseQueryResult<DropsSeedStatus, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant ? dropsSeedStatusKey(activeTenant.id) : ['dropsSeedStatus', 'none'],
    queryFn: () => seedService.getDropsSeedStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}
```

Extend each existing mutation hook with an `onSuccess` that invalidates the matching status key:

```tsx
export function useSeedDrops(): UseMutationResult<void, Error, void> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.seedDrops(),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({ queryKey: dropsSeedStatusKey(activeTenant.id) });
    },
  });
}
```

The seven other `useSeedXxx` mutations get the same treatment against their respective status keys.

### 4.4 SetupPage Seed Data rendering

Replace the current `seedActions` array with one that binds icon + label + mutation + status hook + badge formatter per row. Each row renders as a `<SetupRow>` with:

- `icon` / `label` — from the row spec
- `badge` — `row.formatBadge(row.status.data)`; returns `"—"` when `data === undefined`
- `action` — the existing seed `<Button disabled={row.mutation.isPending} onClick={() => handleSeed(row.mutation, row.label)}>`

`handleSeed` stays unchanged. Only that row's mutation's `isPending` disables its button; polls keep running across all rows regardless of which one is mid-mutation.

Badge formatters use `formatCount` + `pluralize` from `SetupRow.tsx`:

| Row | `formatBadge` produces |
|---|---|
| drops | `"12,040 monster drops / 48 continent drops / 6,116 reactor drops"` |
| gachapons | `"17 gachapons / 842 items / 60 global items"` |
| npc-conversations | `"1,284 conversations"` |
| quest-conversations | `"517 conversations"` |
| npc-shops | `"148 shops / 2,341 commodities"` |
| portal-scripts | `"61 scripts"` |
| reactor-scripts | `"89 scripts"` |
| map-action-scripts | `"210 scripts"` |

Pluralisation singular forms are listed in PRD §4.8.

### 4.5 Hooks-in-array note

Hoist each `useXxxSeedStatus()` call to a named variable at the top of the `SetupPage` function body (next to the existing `useWzInputStatus` / `useExtractionStatus` / `useDataStatus` calls), then pass the query result into the row spec. This keeps the hook calls unconditional and easy to scan, and avoids the refactor hazard of inlining hook calls inside an array literal. Example:

```tsx
const drops = useDropsSeedStatus();
const gachapons = useGachaponsSeedStatus();
// …six more

const seedRows = [
  { label: "…", icon: <…/>, mutation: seedDrops, status: drops, formatBadge: (d?: DropsSeedStatus) => …  },
  // …
];
```

---

## 5. Data flow & error handling

### 5.1 Happy-path poll (compound example)

```
TenantProvider sets active tenant
  │
  ▼
useDropsSeedStatus() fires queryFn every 5s
  │
  ▼
seedService.getDropsSeedStatus(tenant)
  │  GET /api/drops/seed/status
  │  Headers: TENANT_ID, REGION, MAJOR_VERSION, MINOR_VERSION, Accept: application/vnd.api+json
  ▼
nginx: ^/api/drops/seed(/.*)?$ → atlas-drop-information:8080
  │
  ▼
atlas-drop-information seed/status.go handler
  │  tenant.MustFromContext(ctx)
  │  errgroup.WithContext:
  │    ├── monsterdrop.NewProcessor(l, ctx, db).Count()
  │    ├── continentdrop.NewProcessor(l, ctx, db).Count()
  │    └── reactordrop.NewProcessor(l, ctx, db).Count()
  │  Wait; aggregate into DropsSeedStatusRestModel
  ▼
200 { data: { type: "dropsSeedStatus", id: tenantId, attributes: {…, updatedAt: null} } }
  │
  ▼
fetchJsonApi unwraps body.data.attributes
  │
  ▼
React Query cache keyed ['dropsSeedStatus', tenantId]
  │
  ▼
SetupRow.badge = formatBadge(query.data)
```

### 5.2 Mutation → status refresh

```
User clicks Seed
  │
  ▼
seedDrops.mutate()  (async; returns 202)
  │
  ▼ onSuccess
queryClient.invalidateQueries({ queryKey: ['dropsSeedStatus', tenantId] })
  │
  ▼
React Query immediately refetches the invalidated key
  │
  ▼
First tick after invalidation shows 0s (async seed just cleared old rows)
  │
  ▼
Subsequent 5s ticks show counts climbing as the goroutine writes rows
```

Sync seeds (`/shops/seed`, `/npcs/conversations/seed`, …) follow the same arrow — the invalidated refetch lands after the POST resolves, so the new count appears within ~1 s.

### 5.3 Error handling matrix

| Layer | Failure mode | Handling |
|---|---|---|
| Backend COUNT | DB error in any sub-count | errgroup propagates; handler logs, 500 with `{"error":"…"}` |
| Backend | Missing tenant headers | `MustFromContext` panic → recover middleware returns 500 (existing behavior) |
| Backend | `MAX(updated_at)` unparseable | Return `(count, nil, nil)`; response includes `updatedAt: null` |
| nginx | Upstream down | 502/504 → React Query marks errored |
| Frontend | `response.ok === false` / network failure | `fetchJsonApi` throws; `query.data === undefined`; badge renders `"—"` |
| Frontend | No active tenant | `enabled: false`; no queryFn call; badge `"—"` |
| Frontend | Tenant switch mid-poll | `TenantProvider` calls `queryClient.clear()`; badges flash `"—"` until next poll (`staleTime: 0` triggers immediate refetch) |
| Frontend | Mutation failure | Existing toast path; unchanged |

No toasts for status-query errors. No retry logic. Next 5-second tick retries silently.

---

## 6. Testing strategy

### 6.1 Backend unit tests — per sub-resource package

For every `Count()` method added (13 total), extend the package's existing `processor_test.go`. Minimum three cases per `Count()`:

1. **Empty table for tenant** — returns `(0, nil, nil)`.
2. **Rows present for tenant** — returns correct count; for tables with `updated_at`, returns a non-nil `*time.Time` ≥ the max row's timestamp.
3. **Tenant isolation** — seed rows for tenant A and tenant B; call `Count()` in tenant A's context; assert only A's rows are counted.

Tables without `updated_at` (drops + gachapons sub-tables) skip the timestamp-shape assertion in case 2.

### 6.2 Backend handler tests — per status endpoint

One `status_test.go` per new status handler (8 files), modelled on `atlas-data/atlas.com/data/data/status_test.go` and `atlas-wz-extractor/atlas.com/wz-extractor/extraction/status_test.go`. Per handler:

1. **Happy path** — seed N rows, `GET …/seed/status`, assert 200, JSON:API envelope, `id = tenantId`, attribute keys match PRD §4.1, counts correct, `updatedAt` ISO-8601 or null as applicable.
2. **Empty tenant** — no rows; assert `200` with zero counts and `updatedAt: null`.
3. **Tenant isolation** — two tenants, each hitting its own endpoint; assert responses differ.
4. **DB error** — swap in a failing DB or close the connection mid-request; assert `500`. Skip for services whose existing handler-test harness doesn't already support DB-error fixtures.

Compound handlers (drops, gachapons, shops) also assert every sub-count reflects its own table.

### 6.3 Frontend hook tests

Extend `src/lib/hooks/api/__tests__/useSeed.test.ts` (create if absent). For each of the 8 new status hooks:

1. **Query key includes tenant id** — mount with a tenant; assert the cache key `['<resourceType>', tenantId]` is present.
2. **Polling disabled without tenant** — mount with `activeTenant = null`; assert `enabled === false` and queryFn is not called.

For each of the 8 mutation hooks (extending existing mutation tests):

3. **onSuccess invalidates the matching status key** — stub the mutation, fire, assert `queryClient.invalidateQueries` is called with the right key.

No unit test per badge-formatter (trivial string concatenation).

### 6.4 Frontend component tests — out of scope

`SetupPage.tsx` has no existing component test. Adding one is out of scope. Manual smoke testing (run the dev server against a seeded tenant) is sufficient per PRD §10.

### 6.5 Ingress — manual verification only

Manual check during implementation: `curl -H 'TENANT_ID: …' …` each new path against the ingress and confirm it lands on the expected upstream. No integration-test fixture.

### 6.6 Coverage target

Not tracked numerically. Every new `Count()` method and every new handler has at least one assertion per PRD §10 acceptance criterion.

---

## 7. Open questions

None. All open questions from PRD §9 are resolved:

- Exact table names: enumerated in §3 above (`monster_drops`, `continent_drops`, `reactor_drops`, `gachapons`, `gachapon_items`, `global_gachapon_items`, `shops`, `commodities`, `conversations`, `quest_conversations`, `portal_scripts`, `reactor_scripts`, `map_scripts`).
- `updated_at` column presence: covered in §2.5 — drops + gachapons permanently `null`, others real.
- Portals ingress specificity: verified in §3.8 — `^/api/portals/blocked(/.*)?$` precedes `^/api/portals(/.*)?$`, so `/api/portals/scripts/seed/status` correctly lands on `atlas-portal-actions`.
- Drops ingress collision: identified and fixed (§3.8) by broadening `^/api/drops/seed$` → `^/api/drops/seed(/.*)?$` in both `routes.conf` and `ingress.yaml`.
