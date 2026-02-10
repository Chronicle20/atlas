# AI Code Generation Guidance

## Mandatory Implementation Workflow

When generating or modifying Atlas UI code, **always** follow this sequence:

1. **Read existing files** before editing — understand current patterns
2. **Check the component location** — `ui/`, `common/`, or `features/`?
3. **Check tenant context** — Does this resource need tenant scoping?
4. **Implement changes** following established patterns
5. **Run tests**: `npm test`
6. **Fix failures** — Do not proceed with failing tests
7. **Verify build**: `npm run build`
8. **Report actual results** — Never assume success

## Core Rules

### 1. Follow Existing Patterns
When adding code to an existing file or directory, match the patterns already in use. Don't introduce new conventions without explicit instruction.

### 2. Read Before Write
Always read a file before editing it. Understand what's already there to avoid breaking changes or duplicating functionality.

### 3. Type Everything
TypeScript strict mode is enabled. Never use `any`. Use proper interfaces, type guards, and generics. Leverage `z.infer<>` for form types.

### 4. JSON:API Model Structure
All domain models use `{ id: string; attributes: ModelAttributes }`. Don't flatten or restructure. Access data through `.attributes.fieldName`.

### 5. Tenant Context Is Required
Every tenant-scoped operation needs tenant context. Check:
- Hook has `enabled: !!tenant?.id` guard
- Service method calls `api.setTenant(tenant)` first
- Query key includes tenant ID

### 6. Use the Component Library
Use shadcn/ui components from `components/ui/` — Button, Card, Dialog, Select, Input, Badge, etc. Don't create custom primitives that duplicate existing components.

### 7. Use cn() for Classnames
Always `cn()` for conditional or merged classes. Never string concatenation.

### 8. Use Sonner for User Feedback
Toast notifications via `toast.success()`, `toast.error()` from `sonner`. Not `alert()`, not `console.log()`.

### 9. Skeleton Loading
Use `<Skeleton>` components for loading states in content areas. Only use `<Loader2>` spinner inside submit buttons.

### 10. Forms Use react-hook-form + Zod
Define Zod schemas in `lib/schemas/`. Use `zodResolver`. Use shadcn/ui `Form` components for field rendering.

### 11. Immutable State Updates
Never mutate state. Use spread operators, `Array.filter()`, `Array.map()` for new arrays.

### 12. Named Exports
Use named exports for all components. Default exports only for Next.js pages/layouts.

## Generation Workflow

When creating a **new feature** (e.g., a new resource CRUD), create files in this order:

### Step 1: Types
```
types/models/resource.ts
```
- Define model interface with `id` + `attributes`
- Define attribute interface
- Define enums with label maps if needed
- Define create/update request types
- Add helper functions (status checks, formatting)

### Step 2: Service
```
services/api/resource.service.ts
```
- Create class extending `BaseService` or direct pattern
- Implement CRUD methods with tenant parameter
- Override `validate()` and `transformResponse()` if needed
- Export singleton instance
- Add to `services/api/index.ts` exports

### Step 3: React Query Hooks
```
lib/hooks/api/useResources.ts
```
- Define key factory with `as const`
- Create query hooks with `enabled` guards
- Create mutation hooks with cache invalidation
- Export invalidation helpers
- Choose appropriate stale times

### Step 4: Zod Schema (if forms needed)
```
lib/schemas/resource.schema.ts
```
- Define creation schema
- Define update schema (if different)
- Export inferred types
- Export default values

### Step 5: Feature Components
```
components/features/resource/
├── ResourceBadge.tsx        (if status/type badges needed)
├── CreateResourceDialog.tsx (form dialog)
├── DeleteResourceDialog.tsx (confirmation dialog)
└── ResourceDetail.tsx       (detail card, if complex)
```

### Step 6: Pages
```
app/resources/
├── page.tsx               (list page)
└── [id]/
    └── page.tsx           (detail page)
```

### Step 7: Navigation
- Add route to `components/app-sidebar.tsx` items array
- Add breadcrumb route to `lib/breadcrumbs/routes.ts`
- Add entity name resolver to `lib/breadcrumbs/resolvers.ts`

### Step 8: Tests
- Service tests (if validation/transformation)
- Component tests for dialogs
- Hook tests for complex query logic

## Validation Rules

Before submitting code, verify:

| Rule | Check |
|------|-------|
| No `any` types | `grep -r ": any" --include="*.ts" --include="*.tsx"` |
| No hardcoded colors | No `bg-white`, `text-gray-*` etc. |
| Tenant guards present | All hooks have `enabled: !!tenant` |
| `cn()` used for classes | No manual string concatenation |
| Forms use Zod | All `useForm` have `zodResolver` |
| Named exports | No `export default function` (except pages) |
| Error handling | All async ops have catch/error handling |
| Skeleton loading | No raw spinners in content areas |

## Common Composition Examples

### Adding a Column to DataTable
```typescript
const columns: ColumnDef<Resource>[] = [
  // Text column
  { accessorKey: "attributes.name", header: "Name" },

  // Custom render column
  {
    accessorKey: "attributes.status",
    header: "Status",
    cell: ({ row }) => <StatusBadge status={row.original.attributes.status} />,
  },

  // Actions column
  {
    id: "actions",
    cell: ({ row }) => (
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="icon"><MoreHorizontal className="h-4 w-4" /></Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuItem onClick={() => handleView(row.original)}>View</DropdownMenuItem>
          <DropdownMenuItem onClick={() => handleDelete(row.original)} className="text-destructive">Delete</DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    ),
  },
];
```

### Adding a Dialog with Form
```tsx
<Dialog open={open} onOpenChange={onOpenChange}>
  <DialogContent className="sm:max-w-[425px]">
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
        <DialogHeader>
          <DialogTitle>Title</DialogTitle>
          <DialogDescription>Description</DialogDescription>
        </DialogHeader>
        {/* FormField components */}
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button type="submit" disabled={form.formState.isSubmitting}>
            {form.formState.isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Submit
          </Button>
        </DialogFooter>
      </form>
    </Form>
  </DialogContent>
</Dialog>
```

### Adding a New Hook
```typescript
export const resourceKeys = {
  all: ['resources'] as const,
  lists: () => [...resourceKeys.all, 'list'] as const,
  list: (tenant: Tenant | null, options?: QueryOptions) =>
    [...resourceKeys.lists(), tenant?.id || 'no-tenant', options] as const,
  details: () => [...resourceKeys.all, 'detail'] as const,
  detail: (tenant: Tenant | null, id: string) =>
    [...resourceKeys.details(), tenant?.id || 'no-tenant', id] as const,
};

export function useResources(tenant: Tenant | null, options?: QueryOptions) {
  return useQuery({
    queryKey: resourceKeys.list(tenant, options),
    queryFn: () => resourceService.getAll(tenant!, options),
    enabled: !!tenant?.id,
    staleTime: 3 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}
```
