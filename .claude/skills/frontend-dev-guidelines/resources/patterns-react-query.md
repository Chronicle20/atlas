# React Query & Hooks Patterns

## Overview

All server state is managed through TanStack React Query hooks in `lib/hooks/api/`. Each resource has its own hook file exporting query key factories, query hooks, mutation hooks, invalidation helpers, and prefetch utilities.

## Query Key Factory Pattern

**Every hook file exports a hierarchical key factory using `as const`:**

```typescript
// lib/hooks/api/useBans.ts
export const banKeys = {
  all: ['bans'] as const,
  lists: () => [...banKeys.all, 'list'] as const,
  list: (tenant: Tenant | null, options?: QueryOptions) =>
    [...banKeys.lists(), tenant?.id || 'no-tenant', options] as const,
  details: () => [...banKeys.all, 'detail'] as const,
  detail: (tenant: Tenant | null, id: string) =>
    [...banKeys.details(), tenant?.id || 'no-tenant', id] as const,
};
```

**Extended factories for complex resources:**

```typescript
export const conversationKeys = {
  all: ['conversations'] as const,
  lists: () => [...conversationKeys.all, 'list'] as const,
  list: (tenant: Tenant, options?: QueryOptions) => [...conversationKeys.lists(), tenant.id, options] as const,
  details: () => [...conversationKeys.all, 'detail'] as const,
  detail: (id: string) => [...conversationKeys.details(), id] as const,
  // Specialized branches
  byNpc: () => [...conversationKeys.all, 'byNpc'] as const,
  search: () => [...conversationKeys.all, 'search'] as const,
  validation: () => [...conversationKeys.all, 'validation'] as const,
  export: () => [...conversationKeys.all, 'export'] as const,
};
```

**Key principles:**
- Always use `as const` for immutable tuple types
- Build hierarchically using spread
- Include tenant ID in keys for multi-tenant safety
- Use `'no-tenant'` fallback to prevent key collisions

## Query Hook Pattern

```typescript
export function useBans(tenant: Tenant | null, options?: QueryOptions) {
  return useQuery({
    queryKey: banKeys.list(tenant, options),
    queryFn: () => bansService.getAllBans(tenant!, options),
    enabled: !!tenant?.id,      // ← Don't fetch without tenant
    staleTime: 2 * 60 * 1000,   // ← Per-resource stale time
    gcTime: 5 * 60 * 1000,
  });
}

export function useBanById(tenant: Tenant | null, id: string) {
  return useQuery({
    queryKey: banKeys.detail(tenant, id),
    queryFn: () => bansService.getBanById(tenant!, id),
    enabled: !!tenant?.id && !!id,
  });
}
```

## Tenant Injection Patterns

### Pattern A: Explicit Tenant Parameter (Resource-specific hooks)
```typescript
export function useCharacters(tenant: Tenant, options?: QueryOptions) {
  return useQuery({
    queryKey: characterKeys.list(tenant, options),
    queryFn: () => charactersService.getAll(tenant, options),
    enabled: !!tenant?.id,
  });
}
```

**Used by:** useConversations, useInventory, useCharacters, useAccounts, useNPCs, useGuilds

### Pattern B: useTenant() Context Hook (Global resources)
```typescript
export function useMaps(options?: QueryOptions) {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: mapKeys.list(options),
    queryFn: () => mapsService.getAllMaps(activeTenant!, options),
    enabled: !!activeTenant,
  });
}
```

**Used by:** useMaps, useReactors, useMonsters, useGachapons, usePortalScripts, useDrops

### Pattern C: No Tenant (Tenant-agnostic)
```typescript
export function useTenants(options?: QueryOptions) {
  return useQuery({
    queryKey: tenantKeys.list(options),
    queryFn: () => tenantsService.getAllTenants(options),
  });
}
```

**Used by:** useTenants, useServices, useTemplates

## Mutation Hook Pattern

```typescript
export function useCreateBan() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ tenant, data }: { tenant: Tenant; data: CreateBanRequest }) =>
      bansService.createBan(tenant, data),
    onSettled: (data, error, variables) => {
      // Invalidate list to refetch
      queryClient.invalidateQueries({ queryKey: banKeys.lists() });
    },
  });
}
```

### Optimistic Update Pattern

```typescript
export function useDeleteAsset() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ tenant, characterId, compartmentId, assetId }) =>
      inventoryService.deleteAsset(tenant, characterId, compartmentId, assetId),

    onMutate: async ({ tenant, characterId, assetId }) => {
      // Cancel in-flight queries
      await queryClient.cancelQueries({
        queryKey: inventoryKeys.inventory(tenant, characterId),
      });

      // Snapshot previous value
      const previous = queryClient.getQueryData(
        inventoryKeys.inventory(tenant, characterId)
      );

      // Optimistically remove asset
      if (previous) {
        queryClient.setQueryData(inventoryKeys.inventory(tenant, characterId), {
          ...previous,
          included: previous.included.filter(item =>
            !(item.type === 'assets' && item.id === assetId)
          ),
        });
      }

      return { previous };
    },

    onError: (error, variables, context) => {
      // Rollback on error
      if (context?.previous) {
        queryClient.setQueryData(
          inventoryKeys.inventory(variables.tenant, variables.characterId),
          context.previous,
        );
      }
    },

    onSettled: (data, error, { tenant, characterId }) => {
      queryClient.invalidateQueries({
        queryKey: inventoryKeys.inventory(tenant, characterId),
      });
    },
  });
}
```

## Invalidation Helper Pattern

**Every hook file exports invalidation utilities:**

```typescript
export function useInvalidateBans() {
  const queryClient = useQueryClient();

  return {
    invalidateAll: () =>
      queryClient.invalidateQueries({ queryKey: banKeys.all }),
    invalidateLists: () =>
      queryClient.invalidateQueries({ queryKey: banKeys.lists() }),
    invalidateBan: (tenant: Tenant, id: string) =>
      queryClient.invalidateQueries({ queryKey: banKeys.detail(tenant, id) }),
    clearCache: () => {
      bansService.clearServiceCache();
      queryClient.invalidateQueries({ queryKey: banKeys.all });
    },
  };
}
```

## Prefetch Pattern

```typescript
export function usePrefetchConversations() {
  const queryClient = useQueryClient();

  return {
    prefetch: (tenant: Tenant, options?: QueryOptions) =>
      queryClient.prefetchQuery({
        queryKey: conversationKeys.list(tenant, options),
        queryFn: () => conversationsService.getAll(options),
        staleTime: 3 * 60 * 1000,
      }),
  };
}
```

## Stale Time Guidelines

| Data Volatility | Stale Time | GC Time | Examples |
|----------------|-----------|---------|----------|
| High frequency | 30s–1min | 2min | Inventory, logged-in accounts |
| Medium frequency | 1–2min | 5min | Characters, accounts, search results |
| Low frequency | 3–5min | 10min | Conversations, templates, maps |
| Static data | 5–10min | 15min | Guilds, services, rankings |
| Validation/existence | 2–5min | 5min | Existence checks, state consistency |

## Hook File Structure

Each hook file follows this order:
1. Key factory export
2. Query hooks (read operations)
3. Specialized query hooks (filtered, by-relation)
4. Mutation hooks (create, update, delete)
5. Batch operation hooks
6. Invalidation helper hook
7. Prefetch helper hook
8. Cache stats hook (if applicable)
