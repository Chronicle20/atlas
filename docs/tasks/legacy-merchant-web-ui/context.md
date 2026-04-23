# Merchant Web UI - Context

**Last Updated: 2026-03-17**

## Reference Templates (Existing Patterns to Follow)

### Backend REST Resource Pattern
- **Reference**: `services/atlas-merchant/atlas.com/merchant/shop/resource.go`
- Routes registered via `InitializeRoutes` → `mux.Router`
- Handlers follow `rest.GetHandler` signature: `func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc`
- Response marshaling: `server.MarshalResponse[T](d.Logger())(w)(c.ServerInformation())(queryParams)(res)`

### Backend Provider Pattern
- **Reference**: `services/atlas-merchant/atlas.com/merchant/shop/provider.go`
- GORM queries wrapped as `database.EntityProvider[T]`
- String-based `.Where("col = ?", val)` (not struct-based — GORM zero-value gotcha)
- Results converted via `model.SliceMap(Make)` or `model.Map(Make)`

### Frontend Service Class Pattern
- **Reference**: `services/api/maps.service.ts`
- Extends `BaseService`, sets `basePath`
- Methods accept `tenant: Tenant` and call `api.setTenant(tenant)` before requests
- Singleton export: `export const merchantsService = new MerchantsService();`

### Frontend List Page Pattern
- **Reference**: `app/items/page.tsx` (search-based), `app/characters/page.tsx` (auto-load)
- `"use client"` directive, `useTenant()` hook
- `useState` for data/loading/error, `useCallback` + `useEffect` for fetch
- `DataTableWrapper` with `columns.tsx` for grid display
- Error handling via `createErrorFromUnknown()`

### Frontend Detail Page Pattern
- **Reference**: `app/items/[id]/page.tsx`, `app/characters/[id]/page.tsx`
- `useParams()` for route params
- `Promise.all()` for parallel data fetching
- `PageLoader` component for loading state
- Card-based layout with `InfoField` helper components

### Item Icon Pattern
- **Reference**: `app/items/page.tsx:167-174`
- `getAssetIconUrl(tenantId, region, majorVersion, minorVersion, 'item', itemId)`
- `<Image>` with `unoptimized={shouldUnoptimizeImageSrc(iconUrl)}`

## Integration Points

### atlas-merchant REST API
| Current Endpoint | Method | Purpose |
|-----------------|--------|---------|
| `/api/merchants?mapId=X` | GET | Shops by map |
| `/api/merchants/{shopId}` | GET | Single shop + listings |
| `/api/merchants/{shopId}/relationships/listings` | GET | Listings only |
| `/api/characters/{characterId}/merchants` | GET | Shops by owner |

| New Endpoint | Method | Purpose |
|-------------|--------|---------|
| `/api/merchants` (no mapId) | GET | All open shops with listing counts |
| `/api/merchants/{shopId}` (extended) | GET | Add visitors to response |
| `/api/merchants/search/listings?itemId=X` | GET | Search listings by item |

### Frontend Services Used
| Service | Purpose |
|---------|---------|
| `mapsService.getAllMaps()` | Map name resolution (mapId → name) |
| `itemStringsService` or `itemsService` | Item name resolution (itemId → name) |
| `getAssetIconUrl()` | Item icon URLs |
| `charactersService` | Optional: character name resolution |

## Key Files

### Backend (atlas-merchant)
| File | Purpose |
|------|---------|
| `shop/resource.go` | REST route registration and handlers |
| `shop/rest.go` | RestModel definition and transforms |
| `shop/processor.go` | Business logic interface and implementation |
| `shop/provider.go` | Database query functions |
| `shop/entity.go` | GORM entity (shops table) |
| `shop/model.go` | Domain model |
| `listing/rest.go` | Listing RestModel |
| `listing/entity.go` | GORM entity (listings table) |
| `listing/exports.go` | Exported listing query functions |
| `listing/provider.go` | Listing database queries |
| `visitor/registry.go` | Redis visitor registry |
| `rest/handler.go` | Handler utilities (ParseShopId, etc.) |

### Frontend (atlas-ui)
| File | Purpose |
|------|---------|
| `services/api/base.service.ts` | BaseService class to extend |
| `services/api/index.ts` | Service export registry |
| `components/app-sidebar.tsx` | Sidebar navigation menu |
| `components/data-table.tsx` | TanStack table implementation |
| `components/common/DataTableWrapper.tsx` | Smart table wrapper |
| `context/tenant-context.tsx` | Multi-tenant context |
| `lib/utils/asset-url.ts` | Item icon URL builder |
| `types/api/errors.ts` | Error handling utilities |

## New Files to Create

### Backend
- None (all changes are modifications to existing files)

### Frontend
| File | Purpose |
|------|---------|
| `types/models/merchant.ts` | Merchant TypeScript types |
| `services/api/merchants.service.ts` | API service class |
| `app/merchants/page.tsx` | Shop list + item search page |
| `app/merchants/columns.tsx` | Table column definitions |
| `app/merchants/[id]/page.tsx` | Shop detail page |

## Resolved Design Decisions

### Decision 1: Tenant Scope (2026-03-17)
Shops are scoped to the active tenant (world+channel pair). "In the world" means within the selected tenant. Cross-tenant (all-channels) browsing is out of scope for this iteration.

### Decision 2: Map Name Resolution (2026-03-17)
Resolve map names client-side by fetching all maps once and building a lookup map, rather than denormalizing map names into the merchant API response. This follows the existing pattern used elsewhere in the UI.

### Decision 3: Item Name Resolution (2026-03-17)
Use existing `itemStringsService` to resolve item names from itemIds. Cache responses via React Query. Display itemId as fallback if name resolution fails.

### Decision 4: Search by Item (2026-03-17)
The backend search endpoint accepts `itemId` (numeric template ID). The frontend first resolves item name → itemId via `itemsService.searchItems()` if the user enters a name, then queries listings by itemId. Alternatively, the frontend can accept direct itemId input.

### Decision 5: Listing Count Strategy (2026-03-17)
Use a SQL subquery or join to fetch listing counts in the same query as shops, avoiding N+1. Add `listing_count` as a computed field in the REST response rather than requiring a separate call per shop.

### Decision 6: Visitor Data (2026-03-17)
Visitors are only shown on the detail page (not in the list grid). They come from Redis via the existing `GetVisitors` processor method. Accept eventual consistency — visitors are transient.
