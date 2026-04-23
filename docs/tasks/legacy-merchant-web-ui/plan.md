# Merchant Web UI - Implementation Plan

**Last Updated: 2026-03-17**

## Executive Summary

Add merchant shop browsing to atlas-ui: a shop list grid, shop detail view, and item listing search. This requires new backend API endpoints on atlas-merchant (list all open shops with listing counts, search listings by itemId) and corresponding frontend pages, service class, and types in atlas-ui.

## Current State Analysis

### Backend (atlas-merchant)
- **Existing endpoints**:
  - `GET /api/merchants?mapId=X` — shops by map (requires mapId, returns no listing counts)
  - `GET /api/merchants/{shopId}` — single shop with listings
  - `GET /api/merchants/{shopId}/relationships/listings` — listings for a shop
  - `GET /api/characters/{characterId}/merchants` — shops by character
- **Missing**:
  - No endpoint to list ALL open shops (without mapId filter)
  - No listing count included in list responses
  - No endpoint to search listings by itemId across shops
  - No visitor data exposed in REST (only in Redis registry)
- **Data model**: Shop entity has no `channelId`/`worldId` — these are implicit from tenant context (tenant = world+channel pair). The UI already has tenant selection, so "in the world" means within the selected tenant.

### Frontend (atlas-ui)
- No merchant-related pages, services, or types exist
- Established patterns: BaseService, DataTableWrapper, TanStack Table columns, tenant context, item icon rendering via `getAssetIconUrl`
- Sidebar navigation under "Operations" group

## Proposed Future State

### Feature 1: Shop List Grid (`/merchants`)
- DataTableWrapper grid showing all open/maintenance shops for the current tenant
- Columns: Channel (from tenant), Map Name, Shop Name, Owner (characterId), Shop Type badge, Item Count, State badge
- Click shop name → navigate to detail view

### Feature 2: Shop Detail (`/merchants/[id]`)
- Header card: shop info (map, title, owner, type, state, position)
- Visitors section: list of characterIds currently in the shop (from Redis via new endpoint)
- Listings grid: TanStack Table showing each listing with item icon, item name (resolved via itemStrings), templateId, quantity, bundle size, bundles remaining, price per bundle

### Feature 3: Item Listing Search (`/merchants` search tab or `/merchants/search`)
- Search input for item name/ID
- Results grid: shop name, map name, item icon, quantity available, price per bundle, bundles remaining
- Links to shop detail

### Architecture

```
Backend Changes:
  atlas-merchant/shop/resource.go     — new endpoints
  atlas-merchant/shop/processor.go    — new query methods
  atlas-merchant/shop/provider.go     — new DB queries
  atlas-merchant/shop/rest.go         — extended RestModel
  atlas-merchant/listing/provider.go  — search by itemId query

Frontend Changes:
  services/api/merchants.service.ts   — API service class
  types/models/merchant.ts            — TypeScript types
  app/merchants/page.tsx              — shop list + search
  app/merchants/columns.tsx           — table column defs
  app/merchants/[id]/page.tsx         — shop detail
  components/app-sidebar.tsx          — add "Merchants" nav item
  services/api/index.ts               — export new service
```

## Implementation Phases

### Phase 1: Backend API Extensions (M)

New endpoints and queries on atlas-merchant.

#### 1.1 Add `GetAllOpen` provider and processor method (S)
- New `getAllOpen()` in `provider.go`: `WHERE state IN (Open, Maintenance)`
- New `GetAllOpen()` on Processor interface and impl
- **Acceptance**: Returns all non-closed, non-draft shops for current tenant

#### 1.2 Add listing count to shop list response (S)
- Add `ListingCount int64` field to `RestModel` JSON tag `"listingCount"`
- Create `TransformWithListingCount` that queries count per shop
- Alternatively, use a single SQL join/subquery to get counts for all shops in one query
- **Acceptance**: Shop list responses include accurate listing counts

#### 1.3 New endpoint: `GET /api/merchants` without mapId (S)
- Modify `handleGetMerchantsByMap`: if `mapId` query param is absent, return all open shops with listing counts
- Or add a separate handler and route
- **Acceptance**: `GET /api/merchants` returns all open shops with listing counts; `GET /api/merchants?mapId=X` still works

#### 1.4 Add visitor data to shop detail response (S)
- In `handleGetMerchant`, after fetching shop and listings, also call `GetVisitors(shopId)`
- Add `Visitors []uint32` to `RestModel` JSON tag `"visitors"`
- Update `TransformWithListings` to also accept visitors (or chain transforms)
- **Acceptance**: `GET /api/merchants/{shopId}` response includes visitor characterIds

#### 1.5 New endpoint: `GET /api/merchants/search/listings?itemId=X` (M)
- New `listing` provider: `getByItemId(itemId)` queries listings table joined with shops table (state = Open)
- Returns listing + shop info (shopId, shopName, mapId)
- New REST model for search results that includes shop context
- Register route in `InitializeRoutes`
- **Acceptance**: Returns all active listings for a given itemId across all open shops

#### 1.6 Build and test (S)
- `go test ./... -count=1 && go build`
- Docker build verification
- **Acceptance**: All tests pass, service builds and starts

### Phase 2: Frontend Types and Service (S)

#### 2.1 Create merchant TypeScript types (S)
- `types/models/merchant.ts`:
  - `MerchantShop` (id, attributes: characterId, shopType, state, title, mapId, x, y, listingCount, visitors)
  - `MerchantListing` (id, attributes: shopId, itemId, itemType, quantity, bundleSize, bundlesRemaining, pricePerBundle, itemSnapshot, displayOrder)
  - `ListingSearchResult` (listing + shop context: shopId, shopTitle, mapId)
  - Badge variant helpers for shopType and state
- **Acceptance**: Types match backend RestModel JSON structure

#### 2.2 Create merchants service class (S)
- `services/api/merchants.service.ts` extending BaseService
- `basePath = '/api/merchants'` (routed through atlas-merchant service)
- Methods:
  - `getAllShops(tenant)` → `MerchantShop[]`
  - `getShopById(shopId, tenant)` → `MerchantShop` (with listings and visitors)
  - `searchListings(itemId, tenant)` → `ListingSearchResult[]`
- **Acceptance**: Service methods work with tenant context and match API contract

#### 2.3 Export from service index (S)
- Add `merchantsService` export to `services/api/index.ts`
- Export types
- **Acceptance**: Importable from `@/services/api`

### Phase 3: Shop List Page (M)

#### 3.1 Create `app/merchants/page.tsx` (M)
- Follow established list page pattern (accounts/characters)
- Fetch all shops via `merchantsService.getAllShops(activeTenant)`
- Use `DataTableWrapper` with columns
- Add search/filter tab for item listing search (Feature 3)
- **Acceptance**: Page loads, shows all open shops, handles loading/error/empty states

#### 3.2 Create `app/merchants/columns.tsx` (M)
- Columns:
  - Map: resolve mapId → map name via `mapsService` or inline lookup
  - Shop Name: link to `/merchants/{id}`
  - Owner: characterId (link to `/characters/{characterId}`)
  - Type: badge (Character Shop / Hired Merchant)
  - State: badge (Open / Maintenance)
  - Items: listing count number
- **Acceptance**: All columns render correctly, shop name links work

#### 3.3 Add sidebar navigation (S)
- Add "Merchants" entry under "Operations" in `app-sidebar.tsx`
- Position after "Items" or similar logical spot
- **Acceptance**: Sidebar shows Merchants link, active state works

### Phase 4: Shop Detail Page (M)

#### 4.1 Create `app/merchants/[id]/page.tsx` (M)
- Fetch shop by ID via `merchantsService.getShopById(id, activeTenant)`
- Header section: Card with shop info fields (map, title, owner, type, state, position, meso balance for hired merchants)
- Visitors section: list of characterIds (link to character pages)
- Listings table: DataTableWrapper or manual Table with columns (icon, item name, templateId, quantity, bundle size, bundles remaining, price per bundle)
- Item icons via `getAssetIconUrl` using listing's `itemId`
- Item names via `itemStringsService` or item name resolution
- **Acceptance**: Detail page renders all shop data, listings with icons, visitor list

### Phase 5: Item Listing Search (M)

#### 5.1 Add search UI to merchants page (M)
- Search card (similar to items page pattern): input for item name/ID, search button
- Results rendered in a table below search card
- Columns: Item Icon, Item Name, Shop Name (link), Map Name, Quantity, Price Per Bundle, Bundles Remaining
- Use `merchantsService.searchListings(itemId, tenant)`
- Need to resolve item name from itemId if searching by ID, or search by name then look up listings
- **Acceptance**: Can search by item ID, results show all shops selling that item with prices

#### 5.2 URL state management (S)
- Persist search query in URL params (`?itemId=X` or `?q=X`)
- Auto-search on page load if params present (like items page pattern)
- **Acceptance**: Search state survives page refresh, shareable URLs

## Risk Assessment

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Listing search performance (full table scan) | Medium | Medium | Add index on `listings.item_id`; limit results |
| Map name resolution N+1 queries | Low | High | Batch fetch all maps once, build lookup map client-side |
| Item name resolution for listings | Medium | Medium | Use existing `itemStringsService` with client-side cache |
| Visitor data stale (Redis vs reality) | Low | Low | Accept eventual consistency; visitors are transient |
| Cross-tenant search (all channels) not supported | Medium | Low | Document as current-tenant-only; cross-tenant is future work |

## Success Metrics

1. Shop list page loads and displays all open/maintenance shops with correct listing counts
2. Shop detail page shows complete shop info, visitors, and listing grid with item icons
3. Item search returns all active listings for a given item across shops
4. All pages handle loading, error, and empty states gracefully
5. Navigation integrates seamlessly with existing sidebar
6. Backend builds and passes all tests
7. Frontend builds without TypeScript errors

## Required Resources and Dependencies

### Backend
- atlas-merchant service (existing, `services/atlas-merchant/`)
- GORM database queries (existing patterns in `provider.go`)
- Redis visitor registry (existing `visitor.GetRegistry()`)

### Frontend
- `BaseService` pattern (`services/api/base.service.ts`)
- `DataTableWrapper` component
- `mapsService` — for map name resolution
- `itemStringsService` / `itemsService` — for item name resolution
- `getAssetIconUrl` — for item icons
- Tenant context

### External
- No new infrastructure or services required
- No new npm packages needed

## Effort Estimates

| Phase | Description | Effort | Dependencies |
|-------|-------------|--------|--------------|
| Phase 1 | Backend API Extensions | M (3-5 days) | None |
| Phase 2 | Frontend Types & Service | S (1-2 days) | Phase 1 |
| Phase 3 | Shop List Page | M (3-5 days) | Phase 2 |
| Phase 4 | Shop Detail Page | M (3-5 days) | Phase 2 |
| Phase 5 | Item Listing Search | M (3-5 days) | Phase 2 |

**Total**: L (10-15 days)

Phases 3-5 can be parallelized after Phase 2 is complete.
