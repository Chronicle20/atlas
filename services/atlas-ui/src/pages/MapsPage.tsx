import { useTenant } from "@/context/tenant-context";
import { Suspense, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { mapsService, type MapData } from "@/services/api/maps.service";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { DataTableWrapper } from "@/components/common/DataTableWrapper";
import { columns, hiddenColumns } from "./maps-columns";
import { Map, Search, Loader2 } from "lucide-react";
import { useSearchParams } from "react-router-dom";

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

  const mapsQuery = useQuery<MapData[], Error>({
    queryKey: ["maps", "search", activeTenant?.id ?? "no-tenant", urlQuery],
    queryFn: () => mapsService.searchMaps(urlQuery, activeTenant!),
    enabled: !!activeTenant && urlQuery.length > 0,
    staleTime: 30 * 1000,
  });

  const maps = mapsQuery.data ?? [];
  const loading = mapsQuery.isFetching;
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
            <div className="flex-1">
              <Input
                placeholder="Enter map ID, name, or street name..."
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
