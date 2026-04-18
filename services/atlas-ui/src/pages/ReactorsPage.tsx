import { useTenant } from "@/context/tenant-context";
import { Suspense, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { reactorsService } from "@/services/api/reactors.service";
import type { ReactorData } from "@/types/models/reactor";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Zap, Search, Loader2 } from "lucide-react";
import { Link, useSearchParams } from "react-router-dom";
import { NpcImage } from "@/components/features/npc/NpcImage";
import { getAssetIconUrl } from "@/lib/utils/asset-url";

export function ReactorsPage() {
  return (
    <Suspense>
      <ReactorsPageContent />
    </Suspense>
  );
}

function ReactorsPageContent() {
  const { activeTenant } = useTenant();
  const [searchParams, setSearchParams] = useSearchParams();
  const urlQuery = searchParams.get("q") ?? "";
  const [searchInput, setSearchInput] = useState(urlQuery);

  const reactorsQuery = useQuery<ReactorData[], Error>({
    queryKey: ["reactors", "search", activeTenant?.id ?? "no-tenant", urlQuery],
    queryFn: () => reactorsService.searchReactors(urlQuery),
    enabled: !!activeTenant && urlQuery.length > 0,
    staleTime: 30 * 1000,
  });

  const reactors = reactorsQuery.data ?? [];
  const loading = reactorsQuery.isFetching;
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
        <Zap className="h-6 w-6" />
        <h2 className="text-2xl font-bold tracking-tight">Reactors</h2>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Search Reactors</CardTitle>
          <CardDescription>
            Search for reactors by ID or name. Results are limited to 50 entries.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex gap-4 items-end">
            <div className="flex-1">
              <Input
                placeholder="Enter reactor ID or name..."
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
              {reactors.length > 0 && (
                <span className="ml-2 text-muted-foreground font-normal">
                  ({reactors.length} {reactors.length === 1 ? "reactor" : "reactors"})
                </span>
              )}
            </CardTitle>
          </CardHeader>
          <CardContent className="flex-1 min-h-0 flex flex-col">
            {loading ? (
              <div className="text-center py-8 text-muted-foreground">Searching...</div>
            ) : reactors.length === 0 ? (
              <div className="text-center py-8 text-muted-foreground">
                No reactors found matching your search criteria.
              </div>
            ) : (
              <div className="rounded-md border flex-1 min-h-0 overflow-auto">
                <Table>
                  <TableHeader className="sticky top-0 bg-background z-10">
                    <TableRow>
                      <TableHead className="w-10">Icon</TableHead>
                      <TableHead>Template</TableHead>
                      <TableHead>Name</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {reactors.map((reactor) => {
                      const iconUrl = activeTenant ? getAssetIconUrl(
                        activeTenant.id,
                        activeTenant.attributes.region,
                        activeTenant.attributes.majorVersion,
                        activeTenant.attributes.minorVersion,
                        'reactor',
                        parseInt(reactor.id),
                      ) : undefined;
                      return (
                        <TableRow key={reactor.id}>
                          <TableCell>
                            <NpcImage
                              npcId={parseInt(reactor.id)}
                              iconUrl={iconUrl}
                              size={32}
                              lazy={true}
                              showRetryButton={false}
                              maxRetries={1}
                            />
                          </TableCell>
                          <TableCell>
                            <Link to={`/reactors/${reactor.id}`} className="font-mono text-primary hover:underline">
                              {reactor.id}
                            </Link>
                          </TableCell>
                          <TableCell>
                            <Link to={`/reactors/${reactor.id}`} className="font-medium hover:underline">
                              {reactor.attributes.name || `Reactor ${reactor.id}`}
                            </Link>
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
