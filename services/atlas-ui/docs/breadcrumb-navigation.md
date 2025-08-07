# Breadcrumb Navigation System

## Overview

The Atlas UI breadcrumb navigation system provides dynamic, hierarchical navigation that reflects the current page's location within the application structure. The system integrates with multi-tenant context, supports dynamic label resolution for entities, and follows accessibility best practices.

## Architecture

### Core Components

#### 1. Base UI Components (`components/ui/breadcrumb.tsx`)
Following shadcn/ui patterns, the base components provide building blocks:

- **`Breadcrumb`** - Root navigation container with semantic HTML
- **`BreadcrumbList`** - Ordered list of breadcrumb items
- **`BreadcrumbItem`** - Individual breadcrumb container
- **`BreadcrumbLink`** - Clickable breadcrumb links
- **`BreadcrumbPage`** - Current page indicator (non-clickable)
- **`BreadcrumbSeparator`** - Visual separator between items
- **`BreadcrumbEllipsis`** - Truncation indicator

#### 2. Feature Component (`components/features/navigation/BreadcrumbBar.tsx`)
The main breadcrumb bar component that integrates all functionality:

- Dynamic label resolution
- Tenant context integration
- Responsive design with mobile optimization
- Loading states and error handling
- Customizable display options

#### 3. React Hook (`lib/hooks/useBreadcrumbs.ts`)
Core logic for breadcrumb management:

- Route parsing and hierarchy generation
- Dynamic label resolution with caching
- Performance optimizations
- Error handling

## Usage

### Basic Implementation

```tsx
import { BreadcrumbBar } from '@/components/features/navigation/BreadcrumbBar';

export default function Layout({ children }) {
  return (
    <div>
      <header>
        {/* Your header content */}
      </header>
      <BreadcrumbBar />
      <main>{children}</main>
    </div>
  );
}
```

### Advanced Configuration

```tsx
<BreadcrumbBar
  maxItems={5}
  maxItemsMobile={2}
  showEllipsis={true}
  hiddenRoutes={['/admin/debug']}
  showLoadingStates={true}
  labelOverrides={{
    '/dashboard': 'Home',
    '/characters': 'My Characters'
  }}
  className="px-4 py-2"
/>
```

### Using the Hook Directly

```tsx
import { useBreadcrumbs } from '@/lib/hooks/useBreadcrumbs';

export function CustomBreadcrumbs() {
  const { breadcrumbs, loading, error } = useBreadcrumbs({
    maxItems: 3,
    autoResolve: true,
    resolverOptions: {
      fallback: 'Loading...',
      timeout: 5000,
    }
  });

  return (
    <nav aria-label="breadcrumb">
      {/* Custom breadcrumb rendering */}
    </nav>
  );
}
```

## Features

### Dynamic Label Resolution

The system automatically resolves entity names for dynamic routes:

- **Characters**: Resolves character names from IDs
- **Guilds**: Fetches guild names and details
- **NPCs**: Retrieves NPC names and locations
- **Templates**: Shows template names with version info
- **Tenants**: Displays tenant names and regions

```tsx
// Route: /characters/123/inventory
// Breadcrumbs: Home > Characters > "PlayerName" > Inventory
```

### Multi-Tenant Support

Breadcrumbs automatically respect tenant context:

- Tenant-specific data fetching
- Proper header injection for API calls
- Context preservation during navigation
- Graceful handling of tenant switches

### Responsive Design

Mobile-optimized display with adaptive behavior:

- Reduced max items on mobile (2 vs 5)
- Responsive text sizing (`text-xs` → `text-sm`)
- Touch-friendly spacing and click areas
- Label truncation for long names

### Caching and Performance

- **Label Caching**: Resolved entity names are cached per tenant
- **Request Deduplication**: Prevents duplicate API calls
- **Lazy Loading**: Only resolves labels when needed
- **Preloading**: Optional preloading for predictable navigation

### Accessibility

Full WCAG compliance with:

- Semantic `<nav>` element with `aria-label="breadcrumb"`
- Current page marked with `aria-current="page"`
- Screen reader optimized separators (`aria-hidden="true"`)
- Proper focus management and keyboard navigation

## Configuration

### Route Configuration (`lib/breadcrumbs/routes.ts`)

Define breadcrumb mappings for your routes:

```tsx
export const routeConfig: RouteConfig[] = [
  {
    path: '/dashboard',
    label: 'Dashboard',
    showInBreadcrumbs: true,
  },
  {
    path: '/characters',
    label: 'Characters',
    showInBreadcrumbs: true,
    children: [
      {
        path: '/characters/[characterId]',
        label: 'Character Details',
        showInBreadcrumbs: true,
        dynamic: {
          param: 'characterId',
          entityType: 'character',
        },
        children: [
          {
            path: '/characters/[characterId]/inventory',
            label: 'Inventory',
            showInBreadcrumbs: true,
          }
        ]
      }
    ]
  }
];
```

### Entity Resolvers (`lib/breadcrumbs/resolvers.ts`)

Custom resolvers for dynamic labels:

```tsx
// Built-in resolvers available:
- resolveCharacterLabel(characterId, tenantId)
- resolveGuildLabel(guildId, tenantId)
- resolveNPCLabel(npcId, tenantId)
- resolveTemplateLabel(templateId, tenantId)
- resolveTenantLabel(tenantId)

// Custom resolver example:
export async function resolveCustomEntityLabel(
  entityId: string,
  tenantId: string
): Promise<ResolvedLabel> {
  try {
    const entity = await customService.getById(entityId);
    return {
      success: true,
      label: entity.name,
      metadata: { /* additional data */ }
    };
  } catch (error) {
    return {
      success: false,
      error: error as Error,
      fallback: `Entity ${entityId}`
    };
  }
}
```

## Component Props

### BreadcrumbBar Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `className` | `string` | - | Custom CSS classes |
| `maxItems` | `number` | `5` | Maximum items on desktop |
| `maxItemsMobile` | `number` | `2` | Maximum items on mobile |
| `showEllipsis` | `boolean` | `true` | Show truncation indicator |
| `hiddenRoutes` | `string[]` | `[]` | Routes to exclude |
| `showLoadingStates` | `boolean` | `true` | Show loading indicators |
| `labelOverrides` | `Record<string, string>` | `{}` | Custom route labels |

### useBreadcrumbs Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `maxItems` | `number` | `5` | Maximum breadcrumb items |
| `showEllipsis` | `boolean` | `true` | Enable truncation |
| `hiddenRoutes` | `string[]` | `[]` | Routes to hide |
| `autoResolve` | `boolean` | `true` | Auto-resolve dynamic labels |
| `enablePreloading` | `boolean` | `false` | Preload entity labels |
| `resolverOptions` | `ResolverOptions` | - | Custom resolver config |

## Styling

The breadcrumb system uses Tailwind CSS with responsive design:

```css
/* Base list styling */
.breadcrumb-list {
  @apply flex flex-wrap items-center gap-1 break-words text-xs text-muted-foreground;
  @apply sm:gap-1.5 sm:text-sm md:gap-2.5;
}

/* Individual items */
.breadcrumb-item {
  @apply inline-flex items-center gap-1 sm:gap-1.5;
}

/* Links with hover states */
.breadcrumb-link {
  @apply transition-colors hover:text-foreground;
}

/* Current page styling */
.breadcrumb-page {
  @apply font-normal text-foreground;
}

/* Mobile-specific truncation */
.breadcrumb-label {
  @apply truncate max-w-[100px] sm:max-w-none;
}
```

## Testing

### Unit Tests
Located in `lib/breadcrumbs/__tests__/`:

- `utils.test.ts` - Path parsing and breadcrumb generation
- `routes.test.ts` - Route configuration and matching
- `resolvers.test.ts` - Entity label resolution

### Component Tests
Located in component directories:

- `components/ui/__tests__/breadcrumb.test.tsx` - Base UI components
- `components/features/navigation/__tests__/BreadcrumbBar.test.tsx` - Feature component
- `lib/hooks/__tests__/useBreadcrumbs.test.tsx` - React hook

### Integration Tests
Located in `tests/integration/navigation/`:

- `breadcrumb-navigation.test.tsx` - End-to-end navigation scenarios
- `multi-tenant-breadcrumb.test.tsx` - Multi-tenant context handling

## Best Practices

### Performance

1. **Enable Caching**: Use the built-in caching for entity labels
```tsx
resolverOptions={{ useCache: true }}
```

2. **Limit API Calls**: Use `autoResolve: false` when entity data isn't needed
```tsx
const { breadcrumbs } = useBreadcrumbs({ autoResolve: false });
```

3. **Optimize Mobile**: Use lower `maxItemsMobile` for better mobile UX
```tsx
<BreadcrumbBar maxItemsMobile={2} />
```

### Accessibility

1. **Always use semantic HTML**: The components automatically provide proper ARIA attributes
2. **Test with screen readers**: Verify breadcrumb navigation is announced correctly
3. **Maintain focus order**: Ensure logical tab order through breadcrumb links

### Multi-Tenant

1. **Respect tenant context**: Always check `activeTenant` before resolving labels
2. **Handle tenant switches**: The system automatically invalidates cache on tenant change
3. **Provide fallbacks**: Use meaningful fallback labels for missing entities

### Error Handling

1. **Graceful degradation**: System continues working even if some labels fail to resolve
2. **User feedback**: Loading states inform users of background resolution
3. **Logging**: Errors are logged for debugging but don't break the UI

## Troubleshooting

### Common Issues

**Breadcrumbs not showing**
- Ensure route is configured in `routeConfig`
- Check that `showInBreadcrumbs: true` is set
- Verify the route pattern matches exactly

**Dynamic labels not resolving**
- Confirm tenant context is available
- Check API service integration
- Verify resolver function is properly implemented

**Performance issues**
- Disable preloading: `enablePreloading: false`
- Reduce max items: `maxItems={3}`
- Check for excessive re-renders in parent components

**Mobile display problems**
- Adjust `maxItemsMobile` prop
- Check responsive CSS classes
- Test on actual mobile devices

### Debug Mode

Enable debug logging for troubleshooting:

```tsx
const { breadcrumbs, utils } = useBreadcrumbs({
  resolverOptions: { debug: true }
});

// Log current route parsing
console.log('Route segments:', utils.parseCurrentRoute());
```

## Migration Guide

### From Manual Breadcrumbs

Replace manual breadcrumb implementations:

```tsx
// Before
<div className="breadcrumbs">
  <a href="/dashboard">Home</a> &gt;
  <span>Current Page</span>
</div>

// After
<BreadcrumbBar />
```

### Adding New Routes

1. Add route configuration:
```tsx
// lib/breadcrumbs/routes.ts
{
  path: '/new-section/[id]',
  label: 'New Section',
  dynamic: { param: 'id', entityType: 'newEntity' }
}
```

2. Implement resolver if needed:
```tsx
// lib/breadcrumbs/resolvers.ts
export async function resolveNewEntityLabel(id: string, tenantId: string) {
  // Implementation
}
```

3. Add to entity type mapping:
```tsx
// lib/breadcrumbs/resolvers.ts
export function getEntityTypeFromRoute(route: string): EntityType | null {
  if (route.includes('/new-section/')) return 'newEntity';
  // ... existing mappings
}
```

## Future Enhancements

Potential improvements and extensions:

1. **Breadcrumb Themes**: Support for different visual themes
2. **Custom Icons**: Icon support for different entity types
3. **Breadcrumb History**: Navigation history integration
4. **Advanced Caching**: Persistent caching across sessions
5. **Analytics**: Breadcrumb navigation tracking
6. **Internationalization**: Multi-language label support

## Related Documentation

- [Service Layer Architecture](./service-layer.md) - API integration patterns
- [Error Handling](./error-handling.md) - Error handling strategies
- [Reusable Components](./reusable-components.md) - Component design patterns
- [CLAUDE.md](../CLAUDE.md) - Overall architecture guidelines