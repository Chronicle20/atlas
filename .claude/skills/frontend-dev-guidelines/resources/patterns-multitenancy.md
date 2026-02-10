# Multi-Tenancy Patterns

## Overview

Atlas UI is a multi-tenant admin application. Tenant context flows through a React context provider, localStorage persistence, and HTTP headers injected into every API request.

## TenantProvider

The `TenantProvider` wraps the entire application (outermost provider):

```typescript
// context/tenant-context.tsx
type TenantContextType = {
  tenants: Tenant[];                    // All available tenants
  activeTenant: Tenant | null;          // Currently selected tenant
  loading: boolean;                     // Initial load state
  setActiveTenant: (tenant: Tenant) => void;           // Switch tenant
  refreshTenants: () => Promise<void>;                 // Reload tenant list
  refreshAndSelectTenant: (tenantId: string) => Promise<Tenant | null>;
  fetchTenantConfiguration: (tenantId: string) => Promise<TenantConfig>;
};
```

## Tenant Persistence

Active tenant ID is stored in `localStorage` under key `"activeTenantId"`:

```typescript
// On load: restore from localStorage or fall back to first tenant
const storedId = localStorage.getItem("activeTenantId");
const storedTenant = data.find(t => t.id === storedId);
setActiveTenantState(storedTenant ?? data[0] ?? null);

// On change: persist to localStorage
const setActiveTenant = (tenant: Tenant) => {
  setActiveTenantState(tenant);
  localStorage.setItem("activeTenantId", tenant.id);
};
```

## useTenant() Hook

```typescript
export function useTenant() {
  const context = useContext(TenantContext);
  if (!context) {
    throw new Error("useTenant must be used within a TenantProvider");
  }
  return context;
}
```

**Destructure what you need:**
```tsx
const { activeTenant } = useTenant();
const { activeTenant, refreshTenants } = useTenant();
const { activeTenant, fetchTenantConfiguration } = useTenant();
```

## Tenant in API Calls

### Service Layer
Every tenant-scoped service method takes `tenant` as first parameter:

```typescript
async getAllBans(tenant: Tenant, options?: QueryOptions): Promise<Ban[]> {
  api.setTenant(tenant);    // ← Injects headers: TENANT_ID, REGION, MAJOR_VERSION, MINOR_VERSION
  return api.getList<Ban>(this.basePath, options);
}
```

### Headers Injected
```
TENANT_ID: "uuid-string"
REGION: "GMS"
MAJOR_VERSION: "83"
MINOR_VERSION: "1"
```

### Hook Layer — Two Patterns

**Pattern A: Explicit tenant parameter** (resource-specific hooks)
```typescript
export function useBans(tenant: Tenant | null, options?: QueryOptions) {
  return useQuery({
    queryKey: banKeys.list(tenant, options),
    queryFn: () => bansService.getAllBans(tenant!, options),
    enabled: !!tenant?.id,     // ← Guard: don't fetch without tenant
  });
}
```

**Pattern B: useTenant() context** (global resource hooks)
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

**Pattern C: No tenant** (tenant-management hooks)
```typescript
export function useTenants(options?: QueryOptions) {
  return useQuery({
    queryKey: tenantKeys.list(options),
    queryFn: () => tenantsService.getAllTenants(options),
  });
}
```

### Page Layer
Pages get tenant from context and pass to services/hooks:

```tsx
export default function BansPage() {
  const { activeTenant } = useTenant();

  const fetchBans = useCallback(async () => {
    if (!activeTenant) return;           // ← Guard
    const data = await bansService.getAllBans(activeTenant);
    setBans(data);
  }, [activeTenant]);

  useEffect(() => { fetchBans(); }, [fetchBans]);
}
```

## Query Keys Include Tenant

**Always include tenant ID in query keys** to prevent cache cross-contamination between tenants:

```typescript
export const banKeys = {
  list: (tenant: Tenant | null, options?: QueryOptions) =>
    [...banKeys.lists(), tenant?.id || 'no-tenant', options] as const,
  detail: (tenant: Tenant | null, id: string) =>
    [...banKeys.details(), tenant?.id || 'no-tenant', id] as const,
};
```

When the user switches tenants, React Query automatically refetches because the query keys change.

## Tenant Switching

The `AppTenantSwitcher` component in the sidebar allows users to switch active tenant. When switched:

1. `setActiveTenant(newTenant)` updates context and localStorage
2. React components re-render with new `activeTenant`
3. React Query hooks fire new queries (key changed)
4. UI updates with new tenant's data

## Tenant Configuration

Some pages need the full tenant configuration (not just the basic tenant):

```tsx
const { activeTenant, fetchTenantConfiguration } = useTenant();

useEffect(() => {
  if (!activeTenant) return;
  fetchTenantConfiguration(activeTenant.id)
    .then(setTenantConfig)
    .catch(err => setError(createErrorFromUnknown(err).message));
}, [activeTenant, fetchTenantConfiguration]);
```
