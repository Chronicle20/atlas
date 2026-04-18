import { useTenant } from "@/context/tenant-context";
import { Suspense, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { itemsService } from "@/services/api/items.service";
import { type ItemSearchResult, getItemTypeBadgeVariant } from "@/types/models/item";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Package, Search, Loader2 } from "lucide-react";
import { Link, useSearchParams } from "react-router-dom";
import { getAssetIconUrl } from "@/lib/utils/asset-url";

export function ItemsPage() {
  return (
    <Suspense>
      <ItemsPageContent />
    </Suspense>
  );
}

function ItemsPageContent() {
  const { activeTenant } = useTenant();
  const [searchParams, setSearchParams] = useSearchParams();
  const urlQuery = searchParams.get("q") ?? "";
  const [searchInput, setSearchInput] = useState(urlQuery);

  // The URL's `q` parameter is the source of truth for what's been searched.
  const itemsQuery = useQuery<ItemSearchResult[], Error>({
    queryKey: ["items", "search", activeTenant?.id ?? "no-tenant", urlQuery],
    queryFn: () => itemsService.searchItems(urlQuery, activeTenant!),
    enabled: !!activeTenant && urlQuery.length > 0,
    staleTime: 30 * 1000,
  });

  const items = itemsQuery.data ?? [];
  const loading = itemsQuery.isFetching;
  const hasSearched = urlQuery.length > 0;

  const handleSearch = () => {
    if (!activeTenant) {
      toast.error("No tenant selected");
      return;
    }
    if (!searchInput.trim()) {
      toast.error("Please enter a search term");
      return;
    }
    setSearchParams({ q: searchInput.trim() }, { replace: true });
  };

  const handleClear = () => {
    setSearchInput("");
    setSearchParams({}, { replace: true });
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      handleSearch();
    }
  };

  return (
    <div className="flex flex-col flex-1 min-h-0 space-y-6 p-10 pb-16">
      <div className="flex items-center gap-2">
        <Package className="h-6 w-6" />
        <h2 className="text-2xl font-bold tracking-tight">Items</h2>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Search Items</CardTitle>
          <CardDescription>
            Search for items by ID or name. Results are limited to 50 entries.
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
            <Button onClick={handleSearch} disabled={loading}>
              {loading ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Search className="mr-2 h-4 w-4" />
              )}
              Search
            </Button>
            <Button variant="outline" onClick={handleClear} disabled={loading}>
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
              {items.length > 0 && (
                <span className="ml-2 text-muted-foreground font-normal">
                  ({items.length} {items.length === 1 ? "item" : "items"})
                </span>
              )}
            </CardTitle>
          </CardHeader>
          <CardContent className="flex-1 min-h-0 flex flex-col">
            {loading ? (
              <div className="text-center py-8 text-muted-foreground">Searching...</div>
            ) : items.length === 0 ? (
              <div className="text-center py-8 text-muted-foreground">
                No items found matching your search criteria.
              </div>
            ) : (
              <div className="rounded-md border flex-1 min-h-0 overflow-auto">
                <Table>
                  <TableHeader className="sticky top-0 bg-background z-10">
                    <TableRow>
                      <TableHead className="w-10">Icon</TableHead>
                      <TableHead>Name</TableHead>
                      <TableHead>Type</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {items.map((item) => {
                      const iconUrl = activeTenant ? getAssetIconUrl(
                        activeTenant.id,
                        activeTenant.attributes.region,
                        activeTenant.attributes.majorVersion,
                        activeTenant.attributes.minorVersion,
                        'item',
                        parseInt(item.id),
                      ) : '';
                      return (
                        <TableRow key={item.id}>
                          <TableCell>
                            {iconUrl ? (
                              <img
                                src={iconUrl}
                                alt={item.name}
                                width={32}
                                height={32}
                                className="object-contain"
                              />
                            ) : (
                              <Package className="h-8 w-8 text-muted-foreground" />
                            )}
                          </TableCell>
                          <TableCell>
                            <TooltipProvider>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <Link to={`/items/${item.id}`}>
                                    <Badge variant="secondary">{item.name}</Badge>
                                  </Link>
                                </TooltipTrigger>
                                <TooltipContent copyable>
                                  <p>{item.id}</p>
                                </TooltipContent>
                              </Tooltip>
                            </TooltipProvider>
                          </TableCell>
                          <TableCell>
                            <Badge variant="secondary" className={getItemTypeBadgeVariant(item.type)}>
                              {item.type}
                            </Badge>
                          </TableCell>
                        </TableRow>
                      );
                    })}
                  </TableBody>
                </Table>
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
