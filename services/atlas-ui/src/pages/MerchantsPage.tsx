import { useTenant } from "@/context/tenant-context";
import { Suspense, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { merchantsService } from "@/services/api/merchants.service";
import { itemsService } from "@/services/api/items.service";
import type { MerchantShop, ListingSearchResult } from "@/types/models/merchant";
import type { TenantConfig } from "@/services/api/tenants.service";
import { useTenantConfiguration } from "@/lib/hooks/api/useTenants";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Store, Search, Loader2 } from "lucide-react";
import { Link, useSearchParams } from "react-router-dom";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import { DataTableWrapper } from "@/components/common/DataTableWrapper";
import { getColumns, hiddenColumns } from "./merchants-columns";
import { MapCell } from "@/components/map-cell";
import { ItemNameCell } from "@/components/item-name-cell";

export function MerchantsPage() {
  return (
    <Suspense>
      <MerchantsPageContent />
    </Suspense>
  );
}

async function searchListingsByQuery(query: string, activeTenant: NonNullable<ReturnType<typeof useTenant>["activeTenant"]>): Promise<ListingSearchResult[]> {
  const itemId = parseInt(query, 10);
  if (!isNaN(itemId) && String(itemId) === query) {
    return merchantsService.searchListings(itemId, activeTenant);
  }
  const items = await itemsService.searchItems(query, activeTenant);
  if (items.length === 0) return [];
  const itemsToSearch = items.slice(0, 10);
  const allResults: ListingSearchResult[] = [];
  for (const item of itemsToSearch) {
    const data = await merchantsService.searchListings(parseInt(item.id, 10), activeTenant);
    allResults.push(...data);
  }
  return allResults;
}

function MerchantsPageContent() {
  const { activeTenant } = useTenant();
  const [searchParams, setSearchParams] = useSearchParams();
  const initialTab = searchParams.get("tab") ?? "shops";
  const urlQuery = searchParams.get("q") ?? "";
  const [searchInput, setSearchInput] = useState(urlQuery);

  const shopsQuery = useQuery<MerchantShop[], Error>({
    queryKey: ["merchants", "shops", activeTenant?.id ?? "no-tenant"],
    queryFn: () => merchantsService.getAllShops(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 60 * 1000,
  });

  const tenantConfigQuery = useTenantConfiguration(activeTenant?.id ?? "");

  const searchResultsQuery = useQuery<ListingSearchResult[], Error>({
    queryKey: ["merchants", "search-listings", activeTenant?.id ?? "no-tenant", urlQuery],
    queryFn: () => searchListingsByQuery(urlQuery, activeTenant!),
    enabled: !!activeTenant && urlQuery.length > 0,
    staleTime: 30 * 1000,
  });

  const shops = shopsQuery.data ?? [];
  const loading = shopsQuery.isLoading;
  const error = shopsQuery.error?.message ?? null;
  const tenantConfig: TenantConfig | null = tenantConfigQuery.data ?? null;

  const searchResults = searchResultsQuery.data ?? [];
  const searchLoading = searchResultsQuery.isFetching;
  const hasSearched = urlQuery.length > 0;

  const handleSearch = () => {
    if (!activeTenant) {
      toast.error("No tenant selected");
      return;
    }
    if (!searchInput.trim()) {
      toast.error("Please enter an item ID or name");
      return;
    }
    setSearchParams({ tab: "search", q: searchInput.trim() }, { replace: true });
  };

  const handleClear = () => {
    setSearchInput("");
    setSearchParams({ tab: "search" }, { replace: true });
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      handleSearch();
    }
  };

  const handleTabChange = (value: string) => {
    if (value === "search") {
      const next: Record<string, string> = { tab: "search" };
      if (urlQuery) next["q"] = urlQuery;
      setSearchParams(next, { replace: true });
    } else {
      setSearchParams({ tab: "shops" }, { replace: true });
    }
  };

  const columns = getColumns({ tenant: activeTenant, tenantConfig });

  return (
    <div className="flex flex-col flex-1 min-h-0 space-y-6 p-10 pb-16">
      <div className="flex items-center gap-2">
        <Store className="h-6 w-6" />
        <h2 className="text-2xl font-bold tracking-tight">Merchants</h2>
      </div>

      <Tabs defaultValue={initialTab} onValueChange={handleTabChange} className="flex-1 min-h-0 flex flex-col">
        <TabsList>
          <TabsTrigger value="shops">Shops</TabsTrigger>
          <TabsTrigger value="search">Search Listings</TabsTrigger>
        </TabsList>

        <TabsContent value="shops" className="flex-1 min-h-0">
          <DataTableWrapper
            columns={columns}
            data={shops}
            loading={loading}
            error={error}
            onRefresh={() => shopsQuery.refetch()}
            initialVisibilityState={hiddenColumns}
            emptyState={{ title: "No merchant shops found", description: "There are no active merchant shops for this tenant." }}
          />
        </TabsContent>

        <TabsContent value="search" className="flex-1 min-h-0 flex flex-col space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Search Item Listings</CardTitle>
              <CardDescription>
                Search for items available in merchant shops by item ID or name.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex gap-4 items-end">
                <div className="flex-1">
                  <Input
                    placeholder="Enter item ID or name..."
                    value={searchInput}
                    onChange={(e) => setSearchInput(e.target.value)}
                    onKeyDown={handleKeyDown}
                  />
                </div>
                <Button onClick={handleSearch} disabled={searchLoading}>
                  {searchLoading ? (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  ) : (
                    <Search className="mr-2 h-4 w-4" />
                  )}
                  Search
                </Button>
                <Button variant="outline" onClick={handleClear} disabled={searchLoading}>
                  Clear
                </Button>
              </div>
            </CardContent>
          </Card>

          {hasSearched && (
            <Card className="flex-1 min-h-0 flex flex-col">
              <CardHeader className="shrink-0">
                <CardTitle>
                  Results
                  {searchResults.length > 0 && (
                    <span className="ml-2 text-muted-foreground font-normal">
                      ({searchResults.length} {searchResults.length === 1 ? "listing" : "listings"})
                    </span>
                  )}
                </CardTitle>
              </CardHeader>
              <CardContent className="flex-1 min-h-0 flex flex-col">
                {searchLoading ? (
                  <div className="text-center py-8 text-muted-foreground">Searching...</div>
                ) : searchResults.length === 0 ? (
                  <div className="text-center py-8 text-muted-foreground">
                    No listings found matching your search criteria.
                  </div>
                ) : (
                  <div className="rounded-md border flex-1 min-h-0 overflow-auto">
                    <Table>
                      <TableHeader className="sticky top-0 bg-background z-10">
                        <TableRow>
                          <TableHead className="w-10">Icon</TableHead>
                          <TableHead>Item</TableHead>
                          <TableHead>Shop</TableHead>
                          <TableHead>Channel</TableHead>
                          <TableHead>Map</TableHead>
                          <TableHead>Price</TableHead>
                          <TableHead>Qty</TableHead>
                          <TableHead>Bundles</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {searchResults.map((result) => (
                          <SearchResultRow key={result.id} result={result} tenantConfig={tenantConfig} />
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                )}
              </CardContent>
            </Card>
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
}

function SearchResultRow({ result, tenantConfig }: { result: ListingSearchResult; tenantConfig: TenantConfig | null }) {
  const { activeTenant } = useTenant();
  const a = result.attributes;
  const worldName = tenantConfig?.attributes.worlds[a.worldId]?.name || `World ${a.worldId}`;

  const iconUrl = activeTenant ? getAssetIconUrl(
    activeTenant.id,
    activeTenant.attributes.region,
    activeTenant.attributes.majorVersion,
    activeTenant.attributes.minorVersion,
    'item',
    a.itemId,
  ) : '';

  return (
    <TableRow>
      <TableCell>
        {iconUrl ? (
          <img
            src={iconUrl}
            alt={String(a.itemId)}
            width={32}
            height={32}
            className="object-contain"
          />
        ) : (
          <Store className="h-8 w-8 text-muted-foreground" />
        )}
      </TableCell>
      <TableCell>
        <Link to={`/items/${a.itemId}`}>
          <ItemNameCell itemId={String(a.itemId)} tenant={activeTenant} />
        </Link>
      </TableCell>
      <TableCell>
        <Link to={`/merchants/${a.shopId}`} className="font-medium text-primary hover:underline">
          {a.shopTitle || "Untitled"}
        </Link>
      </TableCell>
      <TableCell>
        <Badge variant="secondary">
          {worldName} Ch. {a.channelId + 1}
        </Badge>
      </TableCell>
      <TableCell>
        <MapCell mapId={String(a.mapId)} tenant={activeTenant} />
      </TableCell>
      <TableCell>
        <span className="font-mono">{a.pricePerBundle.toLocaleString()}</span>
      </TableCell>
      <TableCell>{a.quantity}</TableCell>
      <TableCell>{a.bundlesRemaining}</TableCell>
    </TableRow>
  );
}
