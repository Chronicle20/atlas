"use client"

import { useTenant } from "@/context/tenant-context";
import { Suspense, useCallback, useEffect, useRef, useState } from "react";
import { merchantsService } from "@/services/api/merchants.service";
import { itemsService } from "@/services/api/items.service";
import type { MerchantShop } from "@/types/models/merchant";
import type { ListingSearchResult } from "@/types/models/merchant";
import { toast } from "sonner";
import { createErrorFromUnknown } from "@/types/api/errors";
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
import Link from "next/link";
import { useSearchParams, useRouter, usePathname } from "next/navigation";
import Image from "next/image";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import { shouldUnoptimizeImageSrc } from "@/lib/utils/image";
import { DataTableWrapper } from "@/components/common/DataTableWrapper";
import { getColumns, hiddenColumns } from "./columns";
import { MapCell } from "@/components/map-cell";
import { ItemNameCell } from "@/components/item-name-cell";

export default function MerchantsPage() {
  return (
    <Suspense>
      <MerchantsPageContent />
    </Suspense>
  );
}

function MerchantsPageContent() {
  const { activeTenant } = useTenant();
  const searchParams = useSearchParams();
  const router = useRouter();
  const pathname = usePathname();
  const initialTab = searchParams.get("tab") ?? "shops";
  const initialQuery = searchParams.get("q") ?? "";

  const [shops, setShops] = useState<MerchantShop[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [searchQuery, setSearchQuery] = useState(initialQuery);
  const [searchResults, setSearchResults] = useState<ListingSearchResult[]>([]);
  const [searchLoading, setSearchLoading] = useState(false);
  const [hasSearched, setHasSearched] = useState(false);
  const autoSearched = useRef(false);

  const fetchShops = useCallback(async () => {
    if (!activeTenant) return;
    setLoading(true);
    setError(null);
    try {
      const data = await merchantsService.getAllShops(activeTenant);
      setShops(data);
    } catch (err: unknown) {
      const errorInfo = createErrorFromUnknown(err, "Failed to fetch merchants");
      setError(errorInfo.message);
    } finally {
      setLoading(false);
    }
  }, [activeTenant]);

  useEffect(() => {
    fetchShops();
  }, [fetchShops]);

  const handleSearch = useCallback(async () => {
    if (!activeTenant) {
      toast.error("No tenant selected");
      return;
    }
    if (!searchQuery.trim()) {
      toast.error("Please enter an item ID or name");
      return;
    }

    setSearchLoading(true);
    setHasSearched(true);
    router.replace(`${pathname}?tab=search&q=${encodeURIComponent(searchQuery.trim())}`, { scroll: false });

    try {
      const query = searchQuery.trim();
      const itemId = parseInt(query, 10);

      if (!isNaN(itemId) && String(itemId) === query) {
        const data = await merchantsService.searchListings(itemId, activeTenant);
        setSearchResults(data);
        if (data.length === 0) {
          toast.info("No listings found for this item");
        }
      } else {
        const items = await itemsService.searchItems(query, activeTenant);
        if (items.length === 0) {
          toast.info("No items found matching your search");
          setSearchResults([]);
        } else {
          const itemsToSearch = items.slice(0, 10);
          const allResults: ListingSearchResult[] = [];
          for (const item of itemsToSearch) {
            const data = await merchantsService.searchListings(parseInt(item.id, 10), activeTenant);
            allResults.push(...data);
          }
          setSearchResults(allResults);
          if (allResults.length === 0) {
            toast.info("No listings found for matching items");
          }
        }
      }
    } catch (err: unknown) {
      const errorInfo = createErrorFromUnknown(err, "Failed to search listings");
      toast.error(errorInfo.message);
    } finally {
      setSearchLoading(false);
    }
  }, [activeTenant, searchQuery, router, pathname]);

  const handleClear = () => {
    setSearchQuery("");
    setSearchResults([]);
    setHasSearched(false);
    router.replace(`${pathname}?tab=search`, { scroll: false });
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      handleSearch();
    }
  };

  useEffect(() => {
    if (activeTenant && initialQuery && initialTab === "search" && !autoSearched.current) {
      autoSearched.current = true;
      handleSearch();
    }
  }, [activeTenant, initialQuery, initialTab, handleSearch]);

  const columns = getColumns({ tenant: activeTenant });

  return (
    <div className="flex flex-col flex-1 min-h-0 space-y-6 p-10 pb-16">
      <div className="flex items-center gap-2">
        <Store className="h-6 w-6" />
        <h2 className="text-2xl font-bold tracking-tight">Merchants</h2>
      </div>

      <Tabs defaultValue={initialTab} className="flex-1 min-h-0 flex flex-col">
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
            onRefresh={fetchShops}
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
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
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
                {searchResults.length === 0 ? (
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
                          <TableHead>Map</TableHead>
                          <TableHead>Price</TableHead>
                          <TableHead>Qty</TableHead>
                          <TableHead>Bundles</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {searchResults.map((result) => (
                          <SearchResultRow key={result.id} result={result} />
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

function SearchResultRow({ result }: { result: ListingSearchResult }) {
  const { activeTenant } = useTenant();
  const a = result.attributes;

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
          <Image
            src={iconUrl}
            alt={String(a.itemId)}
            width={32}
            height={32}
            unoptimized={shouldUnoptimizeImageSrc(iconUrl)}
            className="object-contain"
          />
        ) : (
          <Store className="h-8 w-8 text-muted-foreground" />
        )}
      </TableCell>
      <TableCell>
        <Link href={`/items/${a.itemId}`}>
          <ItemNameCell itemId={String(a.itemId)} tenant={activeTenant} />
        </Link>
      </TableCell>
      <TableCell>
        <Link href={`/merchants/${a.shopId}`} className="font-medium text-primary hover:underline">
          {a.shopTitle || "Untitled"}
        </Link>
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
