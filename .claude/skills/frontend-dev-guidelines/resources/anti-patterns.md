# Anti-Patterns

## Quick Reference

| Anti-Pattern | Correct Pattern |
|-------------|-----------------|
| Manual class concatenation | Use `cn()` utility |
| `any` type | Proper types or type guards |
| Direct API calls in components | Use service layer → hooks |
| Inline Zod schemas in components | Define in `lib/schemas/` |
| Spinner for page loading | Skeleton components |
| `console.log` for errors | `toast.error()` + `createErrorFromUnknown()` |
| Hardcoded color values | Semantic CSS variables (`bg-background`) |
| Missing tenant guard | `enabled: !!tenant?.id` in hooks |
| State mutation | Spread operator for immutable updates |
| Default export for components | Named export |

---

## Detailed Anti-Patterns

### 1. Skipping Tenant Context

```tsx
// ❌ Bad — fetches without tenant, causes wrong data
useEffect(() => {
  bansService.getAllBans(null as any).then(setBans);
}, []);

// ✅ Good — guards against missing tenant
const { activeTenant } = useTenant();

useEffect(() => {
  if (!activeTenant) return;
  bansService.getAllBans(activeTenant).then(setBans);
}, [activeTenant]);
```

### 2. Calling API Client Directly from Components

```tsx
// ❌ Bad — bypasses service layer
import { api } from "@/lib/api/client";
const data = await api.get('/api/bans');

// ✅ Good — uses service abstraction
import { bansService } from "@/services/api";
const data = await bansService.getAllBans(tenant);
```

### 3. Missing Query Key Tenant Isolation

```typescript
// ❌ Bad — same cache key for all tenants
export const banKeys = {
  list: (options?: QueryOptions) => ['bans', 'list', options] as const,
};

// ✅ Good — tenant ID in key prevents cross-contamination
export const banKeys = {
  list: (tenant: Tenant | null, options?: QueryOptions) =>
    ['bans', 'list', tenant?.id || 'no-tenant', options] as const,
};
```

### 4. Manual Class String Concatenation

```tsx
// ❌ Bad — no merge, duplicates not resolved
<div className={"flex items-center " + (active ? "bg-primary" : "")} />

// ✅ Good — cn() handles merging and deduplication
<div className={cn("flex items-center", active && "bg-primary")} />
```

### 5. Using `any` Type

```typescript
// ❌ Bad — defeats TypeScript
const handleData = (data: any) => { ... };

// ✅ Good — proper typing
const handleData = (data: Ban) => { ... };

// ✅ Good — unknown + type guard for dynamic data
const handleData = (data: unknown) => {
  if (isBan(data)) { /* typed access */ }
};
```

### 6. Inline Schema Definition

```tsx
// ❌ Bad — schema buried in component, not reusable
export function CreateDialog() {
  const schema = z.object({ name: z.string().min(1) });
  // ...
}

// ✅ Good — schema in dedicated file
// lib/schemas/resource.schema.ts
export const createResourceSchema = z.object({ name: z.string().min(1) });
export type CreateResourceFormData = z.infer<typeof createResourceSchema>;

// Component imports schema
import { createResourceSchema, type CreateResourceFormData } from "@/lib/schemas/resource.schema";
```

**Exception:** Cross-field `.refine()` validations that are form-specific can live in the component file.

### 7. Spinner for Content Loading

```tsx
// ❌ Bad — jarring, no layout stability
if (loading) return <div className="animate-spin">Loading...</div>;

// ✅ Good — preserves layout during loading
if (loading) return <PageSkeleton />;

// ✅ Good — spinner only for submit buttons
<Button disabled={isSubmitting}>
  {isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
  Submit
</Button>
```

### 8. Hardcoded Colors

```tsx
// ❌ Bad — ignores theme, breaks dark mode
<div className="bg-white text-gray-900 border-gray-200" />

// ✅ Good — uses semantic CSS variables
<div className="bg-background text-foreground border-border" />
<p className="text-muted-foreground" />
```

### 9. Mutating State

```tsx
// ❌ Bad — direct mutation
const updatedBans = bans;
updatedBans.push(newBan);
setBans(updatedBans);

// ✅ Good — immutable update
setBans([...bans, newBan]);

// ✅ Good — immutable object update
setCharacter({ ...character, attributes: { ...character.attributes, ...updates } });
```

### 10. Missing Error Handling in Async Operations

```tsx
// ❌ Bad — unhandled rejection
useEffect(() => {
  bansService.getAllBans(tenant).then(setBans);
}, [tenant]);

// ✅ Good — proper error handling with user feedback
useEffect(() => {
  if (!tenant) return;
  bansService.getAllBans(tenant)
    .then(data => { setBans(data); setError(null); })
    .catch(err => {
      const errorInfo = createErrorFromUnknown(err, "Failed to fetch bans");
      setError(errorInfo.message);
    })
    .finally(() => setLoading(false));
}, [tenant]);
```

### 11. Default Exports for Components

```tsx
// ❌ Bad — unnamed in imports, harder to refactor
export default function BanList() { ... }

// ✅ Good — explicit naming, better IDE support
export function BanList() { ... }
```

**Exception:** Next.js `page.tsx` and `layout.tsx` files use default exports as required by the framework.

### 12. Forgetting `enabled` Guard in Hooks

```typescript
// ❌ Bad — fires request with null tenant
export function useBans(tenant: Tenant | null) {
  return useQuery({
    queryKey: banKeys.list(tenant),
    queryFn: () => bansService.getAllBans(tenant!),  // ← Will crash
  });
}

// ✅ Good — guarded with enabled
export function useBans(tenant: Tenant | null) {
  return useQuery({
    queryKey: banKeys.list(tenant),
    queryFn: () => bansService.getAllBans(tenant!),
    enabled: !!tenant?.id,  // ← Only fetches when tenant exists
  });
}
```
