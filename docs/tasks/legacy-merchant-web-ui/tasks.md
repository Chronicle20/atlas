# Merchant Web UI - Tasks

**Last Updated: 2026-03-17**

## Phase 1: Backend API Extensions

- [x] 1.1 Add `getAllOpen()` provider in `shop/provider.go` (WHERE state IN Open, Maintenance)
- [x] 1.2 Add `GetAllOpen()` to Processor interface and ProcessorImpl
- [x] 1.3 Add listing count subquery — batch count via `CountByShopIds`
- [x] 1.4 Add `ListingCount` field to `shop/rest.go` RestModel
- [x] 1.5 Create `TransformWithListingCount` for list responses
- [x] 1.6 Modify handler to return all open shops when mapId is absent
- [x] 1.7 Add visitor characterIds to detail response
- [x] 1.7a Add `Visitors []uint32` field to RestModel
- [x] 1.7b Extend `handleGetMerchant` to call `GetVisitors(shopId)` and include in response
- [x] 1.8 Add `searchListingsByItemId` provider (join listings with shops)
- [x] 1.9 Add `SearchListingsByItemId` processor method
- [x] 1.10 Create `ListingSearchRestModel` REST model (listing + shop context)
- [x] 1.11 Add `handleSearchListings` handler in `shop/resource.go`
- [x] 1.12 Register `/merchants/search/listings` route
- [x] 1.13 Add index on `listings.item_id` column
- [x] 1.14 Run `go test ./... -count=1 && go build`
- [x] 1.15 Docker build verification (pre-existing go.sum issue, not from our changes)

## Phase 2: Frontend Types and Service

- [x] 2.1 Create `types/models/merchant.ts` with MerchantShop, MerchantListing, ListingSearchResult types
- [x] 2.2 Add shop type helpers (shopType badge variant, state badge variant)
- [x] 2.3 Create `services/api/merchants.service.ts` extending BaseService
- [x] 2.3a Implement `getAllShops(tenant)` method
- [x] 2.3b Implement `getShopById(shopId, tenant)` method
- [x] 2.3c Implement `searchListings(itemId, tenant)` method
- [x] 2.4 Export service and types from `services/api/index.ts`

## Phase 3: Shop List Page

- [x] 3.1 Add "Merchants" to sidebar navigation in `components/app-sidebar.tsx`
- [x] 3.2 Create `app/merchants/columns.tsx` with column definitions
- [x] 3.2a Map column (resolve mapId → map name via MapCell)
- [x] 3.2b Shop Name column (link to `/merchants/{id}`)
- [x] 3.2c Owner column (characterId, link to `/characters/{characterId}`)
- [x] 3.2d Type column (badge: Character Shop / Hired Merchant)
- [x] 3.2e State column (badge: Open / Maintenance)
- [x] 3.2f Items column (listing count)
- [x] 3.3 Create `app/merchants/page.tsx`
- [x] 3.3a Fetch shops via merchantsService.getAllShops
- [x] 3.3b Map name resolution via MapCell component
- [x] 3.3c Render DataTableWrapper with columns
- [x] 3.3d Handle loading/error/empty states

## Phase 4: Shop Detail Page

- [x] 4.1 Create `app/merchants/[id]/page.tsx`
- [x] 4.1a Fetch shop data with listings and visitors
- [x] 4.1b Resolve map name via MapCell
- [x] 4.1c Shop info header card (map, title, owner, type, state, coords, meso balance)
- [x] 4.1d Visitors section (list of characterIds with links)
- [x] 4.1e Listings table with columns (icon, item name, templateId, quantity, bundle info, price)
- [x] 4.1f Item icon rendering via getAssetIconUrl
- [x] 4.1g Item name resolution via ItemNameCell component
- [x] 4.1h Loading/error states (PageLoader, error display)

## Phase 5: Item Listing Search

- [x] 5.1 Add search UI to merchants page
- [x] 5.1a Search card with input field and search/clear buttons
- [x] 5.1b Item name/ID search input
- [x] 5.1c Search results table (item icon, item name, shop name, map name, quantity, price, bundles remaining)
- [x] 5.1d Shop name links to detail page
- [x] 5.2 URL state management (?tab=search&q= query params)
- [x] 5.3 Auto-search on page load when params present

## Post-Implementation

- [ ] 6.1 Verify all pages work end-to-end with real data
- [ ] 6.2 Test empty states (no shops, no listings, no search results)
- [ ] 6.3 Test error states (service down, invalid shopId)
- [x] 6.4 Verify sidebar navigation active state
- [x] 6.5 Frontend build (`npm run build`)
- [ ] 6.6 Update docs/TODO.md if applicable
