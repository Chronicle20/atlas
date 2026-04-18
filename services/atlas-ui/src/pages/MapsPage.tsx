import { useTenant } from "@/context/tenant-context";
import { Suspense, useEffect, useState } from "react";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { mapsService, type MapData } from "@/services/api/maps.service";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { DataTableWrapper } from "@/components/common/DataTableWrapper";
import { columns, hiddenColumns } from "./maps-columns";
import { Map, Loader2 } from "lucide-react";
import { useSearchParams } from "react-router-dom";
import { useDebounce } from "@/lib/utils/debounce";

const MIN_QUERY_LENGTH = 2;
const DEBOUNCE_MS = 250;

export function MapsPage() {
  return (
    <Suspense>
      <MapsPageContent />
    </Suspense>
  );
}

function MapsPageContent() {
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

  const mapsQuery = useQuery<MapData[], Error>({
    queryKey: ["maps", "search", activeTenant?.id ?? "no-tenant", urlQuery],
    queryFn: () => mapsService.searchMaps(urlQuery),
    enabled: !!activeTenant && urlQuery.length >= MIN_QUERY_LENGTH,
    staleTime: 30 * 1000,
    placeholderData: keepPreviousData,
  });

  const maps = mapsQuery.data ?? [];
  const fetching = mapsQuery.isFetching;
  const hasSearched = urlQuery.length >= MIN_QUERY_LENGTH;

  const handleClear = () => {
    setSearchInput("");
    setSearchParams({}, { replace: true });
  };

  return (
    <div className="flex flex-col flex-1 min-h-0 space-y-6 p-10 pb-16">
      <div className="flex items-center gap-2">
        <Map className="h-6 w-6" />
        <h2 className="text-2xl font-bold tracking-tight">Maps</h2>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Search Maps</CardTitle>
          <CardDescription>
            Search for maps by ID, name, or street name. Results are limited to 50 entries.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex gap-4 items-end">
            <div className="flex-1 relative">
              <Input
                placeholder="Enter map ID, name, or street name..."
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
        <div className="flex-1 min-h-0">
          <DataTableWrapper
            columns={columns}
            data={maps}
            error={mapsQuery.error?.message ?? null}
            onRefresh={() => mapsQuery.refetch()}
            initialVisibilityState={hiddenColumns}
            emptyState={{
              title: "No maps found",
              description: "Try a different search term.",
            }}
          />
        </div>
      )}
    </div>
  );
}
