# Service Layer

Atlas UI's service layer is a collection of plain objects under `src/services/api/` that call `api.*` from `src/lib/api/client.ts`. React Query hooks under `src/lib/hooks/api/` wrap the services with caching, invalidation, and optimistic-update plumbing.

There is no `BaseService` class. There was one in the Next.js era — during task-004 it was deleted and every service became a plain object literal that calls `api.getList` / `api.getOne` / `api.post` / `api.put` / `api.patch` / `api.delete` directly.

## Directory

```
src/
├── lib/api/
│   ├── client.ts         # ApiClient + `api` wrapper (tenant headers, retries, JSON:API)
│   ├── query-params.ts   # ServiceOptions / QueryOptions / BatchOptions types + helpers
│   └── errors.ts         # sanitizeErrorData, normalized error construction
├── lib/hooks/api/
│   ├── useAccounts.ts
│   ├── useBans.ts
│   ├── useCharacters.ts
│   ├── useConversations.ts
│   ├── …                 # one per resource
│   └── query-keys.ts     # (follow-up) single source for query keys
└── services/api/
    ├── accounts.service.ts
    ├── bans.service.ts
    ├── characters.service.ts
    ├── conversations.service.ts
    ├── fields.service.ts        # /api/fields — live field occupancy + field character roster
    ├── worlds.service.ts        # /api/worlds — world/channel topology
    ├── live-monsters.service.ts # live monsters for a running field
    ├── map-entities.service.ts  # map-scoped definition data; getObjects() added alongside portals/npcs/reactors/monsters
    ├── …
    └── index.ts          # re-exports every service + its types
```

## Shape of a service

```ts
import { api } from "@/lib/api/client";
import { buildQueryString, type QueryOptions, type ServiceOptions } from "@/lib/api/query-params";
import type { Tenant } from "@/types/models/tenant";
import type { Account } from "@/types/models/account";

const BASE_PATH = "/api/accounts";

export const accountsService = {
  async getAllAccounts(_tenant: Tenant, options?: QueryOptions): Promise<Account[]> {
    return api.getList<Account>(`${BASE_PATH}${buildQueryString(options)}`, options);
  },

  async getAccountById(_tenant: Tenant, id: string, options?: ServiceOptions): Promise<Account> {
    return api.getOne<Account>(`${BASE_PATH}/${id}`, options);
  },

  async deleteAccount(_tenant: Tenant, id: string, options?: ServiceOptions): Promise<void> {
    return api.delete(`${BASE_PATH}/${id}`, options);
  },
};
```

The `_tenant` parameter is legacy signature back-compat — `TenantProvider` already pushed the active tenant into the API client before the service method is called (see `src/context/tenant-context.tsx`), so the parameter is ignored inside the service. The `_` prefix is what keeps `noUnusedParameters` quiet. Callers can pass `undefined` or their tenant; it doesn't matter.

## Shape of a hook

```ts
import { useMutation, useQuery, useQueryClient, type UseMutationResult, type UseQueryResult } from "@tanstack/react-query";
import { accountsService } from "@/services/api/accounts.service";
import type { Account } from "@/types/models/account";
import type { Tenant } from "@/types/models/tenant";

export const accountKeys = {
  all: ["accounts"] as const,
  lists: () => [...accountKeys.all, "list"] as const,
  list: (tenant: Tenant | null) => [...accountKeys.lists(), tenant?.id ?? "no-tenant"] as const,
  details: () => [...accountKeys.all, "detail"] as const,
  detail: (tenant: Tenant | null, id: string) => [...accountKeys.details(), tenant?.id ?? "no-tenant", id] as const,
};

export function useAccounts(tenant: Tenant): UseQueryResult<Account[], Error> {
  return useQuery({
    queryKey: accountKeys.list(tenant),
    queryFn: () => accountsService.getAllAccounts(tenant),
    enabled: !!tenant?.id,
    staleTime: 2 * 60 * 1000,
  });
}

export function useDeleteAccount(): UseMutationResult<void, Error, { tenant: Tenant; id: string }> {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ tenant, id }) => accountsService.deleteAccount(tenant, id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: accountKeys.all }),
  });
}

export function useInvalidateAccounts() {
  const queryClient = useQueryClient();
  return {
    invalidateAll: () => queryClient.invalidateQueries({ queryKey: accountKeys.all }),
  };
}
```

## Cache profiles: definition vs runtime

Every hook falls into one of two cache profiles, chosen by how often the underlying data actually
changes:

|  | `staleTime` | `gcTime` |
|---|---|---|
| Definition (maps, portals, npcs, reactors, monsters, objects, worlds, channels) | `10 * 60 * 1000` | `10 * 60 * 1000` |
| Runtime (fields, field characters, live monsters) | `5 * 1000` | `60 * 1000` |

Definition data — a map's static layout, or the world/channel topology — barely changes, so it can
sit in cache for ten minutes. Runtime data — which fields currently exist, who's in them, which
monsters are alive — changes continuously, so its cache window is five seconds with a one-minute
GC.

The runtime constants (`RUNTIME_STALE_TIME = 5 * 1000`, `RUNTIME_GC_TIME = 60 * 1000`) are declared
independently in both `useFields.ts` and `useFieldRuntime.ts` — they're duplicated, not shared from
a common constant, so keep them in sync by hand if either changes. The definition profile is a
literal `10 * 60 * 1000` inline at each `useQuery` call in `useMapEntities.ts` and `useWorlds.ts`.

`useFields.ts` exports `fieldQueryOptions(filters)` as the one place the runtime profile is
assembled for fields; both `useFields` and `useFieldsForMap` spread it, so the two can't drift
apart from each other even though the constants above aren't shared with `useFieldRuntime.ts`.

Neither profile uses `refetchInterval` anywhere — runtime freshness comes from a short `staleTime`
plus normal refetch-on-mount/refocus behavior, not polling.

### Key namespaces

Each hook module owns its own key factory, and the roots are deliberately disjoint so that
invalidating one never touches another:

- `mapEntityKeys` (`useMapEntities.ts`) → `["maps", mapId, "portals" | "npcs" | "reactors" | "monsters" | "objects"]`
- `worldKeys` (`useWorlds.ts`) → `["worlds"]`, `["worlds", "list"]`, `["worlds", worldId, "channels"]`
- `fieldKeys` (`useFields.ts`) → `["fields"]`, `["fields", "list", filters]`
- `fieldRuntimeKeys` (`useFieldRuntime.ts`) → `["fields", worldId, channelId, mapId, instanceId, "monsters" | "characters"]`

`worldKeys` is its own namespace — it carries the definition cache profile, but its key root is
`["worlds", …]`, not `["maps", …]`. The "definition vs runtime" split above is about the cache
profile a hook uses, not which key root it lives under.

The `["maps", …]` and `["fields", …]` roots being disjoint matters because a field-runtime refetch
(e.g. a live monster count changing) must never invalidate the map's definition cache, and vice
versa — a stale live-occupancy number should never force a portal/NPC refetch.

## Tenant contract

Every API request carries four headers (SCREAMING_SNAKE_CASE, see `src/lib/headers.tsx`):

- `TENANT_ID`
- `REGION`
- `MAJOR_VERSION`
- `MINOR_VERSION`

`TenantProvider` owns the wiring. Whenever `activeTenant` changes (after the initial null mount):

1. `api.setTenant(activeTenant)` updates the client's tenant reference so subsequent requests pick up the new headers.
2. `queryClient.clear()` invalidates every React Query cache entry so tenant A never sees tenant B's data.

Both steps fire together in a single `useEffect` in `src/context/tenant-context.tsx:68`. Covered by `src/context/__tests__/tenant-context.test.tsx`.

Because `queryClient.clear()` already wipes every cache entry on tenant change, the tenant is
deliberately **not** part of any query key in `fieldKeys`, `fieldRuntimeKeys`, or `worldKeys` — one
more field in a key tuple would be redundant with the clear. What every one of those hooks does
keep is the `enabled: !!activeTenant` guard, so a query never fires before a tenant is selected.

## Shared types and helpers

`src/lib/api/query-params.ts`:

- `ServiceOptions` — base options accepted by every service method (extends `ApiRequestOptions`).
- `QueryOptions` — `ServiceOptions` + `search` / `sortBy` / `sortOrder` / `limit` / `offset` / `filters` / `fields`.
- `BatchOptions` / `BatchResult<T>` — shapes for bulk operations.
- `ValidationError` — `{ field, message, value? }`.
- `buildQueryString(options?)` — serialises `QueryOptions` to `?foo=bar&filter[x]=y&fields[maps]=name,streetName`.
- `runBatch(items, fn, options?)` — bounded-concurrency batch runner with `BatchResult<T>` reporting.

`src/lib/api/client.ts` exports `api` (convenience wrapper) and `apiClient` (the singleton). The client does JSON:API error parsing, retries 5xx/429 up to 3 times with exponential backoff, and injects tenant headers unless `skipTenantHeaders` is passed. It does not own caching — React Query does. The `cacheConfig` / `staleWhileRevalidate` / `skipDeduplication` / `onProgress` fields on `ApiRequestOptions` are kept as compatibility shims and ignored.

## Patterns for pages

- **List page**: `const { data, isLoading, error } = useAccounts(activeTenant!)` — drive the `DataTableWrapper` directly, call `useInvalidateAccounts` for refresh.
- **Detail page**: `const { data } = useAccount(activeTenant!, id ?? "")` — the `enabled` guard in the hook handles the null-tenant / empty-id case.
- **Filter/search page**: read `?q=…` from `useSearchParams`, pass it to the hook's query function, make the URL the single source of truth. No `autoSearched` ref, no initial-load `useEffect`.
- **Form page**: `useX(id)` for the read, `useUpdateX` for the write. A small `useEffect` keyed on the query's `data` drives `form.reset()`.
- **Mutation on success**: invalidate the list keyspace in the hook's `onSuccess`, or use the `useInvalidateX` return object for ad-hoc invalidation.

## Follow-ups tracked in docs/TODO.md

- Drop the `_tenant` parameter from service method signatures once every caller is updated.
- Centralise query keys in `src/lib/hooks/api/query-keys.ts` (today each hook module declares its own).
- Delete the legacy `lib/hooks/` wrappers (`useNpcData`, `useItemData`, `useMobData`, `useSkillData`) once their callers move to `lib/hooks/api/`.
