import { useTenant } from "@/context/tenant-context";
import { Suspense, useEffect, useState } from "react";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { itemsService } from "@/services/api/items.service";
import { type ItemSearchResult, getItemTypeBadgeVariant } from "@/types/models/item";
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
import { Package, Loader2 } from "lucide-react";
import { Link, useSearchParams } from "react-router-dom";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import { useDebounce } from "@/lib/utils/debounce";

const MIN_QUERY_LENGTH = 2;
const DEBOUNCE_MS = 250;

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
  const debounced = useDebounce(searchInput.trim(), DEBOUNCE_MS);

  useEffect(() => {
    const next = debounced;
    if (next.length >= MIN_QUERY_LENGTH) {
      if (next !== urlQuery) {
        setSearchParams({ q: next }, { replace: true });
      }
    } else if (urlQuery !== "") {
      setSearchParams({}, { replace: true });
    }
  }, [debounced, urlQuery, setSearchParams]);

  const itemsQuery = useQuery<ItemSearchResult[], Error>({
    queryKey: ["items", "search", activeTenant?.id ?? "no-tenant", urlQuery],
    queryFn: () => itemsService.searchItems(urlQuery),
    enabled: !!activeTenant && urlQuery.length >= MIN_QUERY_LENGTH,
    staleTime: 30 * 1000,
    placeholderData: keepPreviousData,
  });

  const items = itemsQuery.data ?? [];
  const fetching = itemsQuery.isFetching;
  const hasSearched = urlQuery.length >= MIN_QUERY_LENGTH;

  const handleClear = () => {
    setSearchInput("");
    setSearchParams({}, { replace: true });
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
            <div className="flex-1 relative">
              <Input
                placeholder="Enter item ID or name..."
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
              />
              {fetching && (
                <Loader2 className="absolute right-3 top-1/2 -translate-y-1/2 h-4 w-4 animate-spin text-muted-foreground" />
              )}
            </div>
            <Button variant="outline" onClick={handleClear}>
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
            {items.length === 0 ? (
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
