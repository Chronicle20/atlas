import { useTenant } from "@/context/tenant-context";
import { Suspense, useEffect, useState } from "react";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { reactorsService } from "@/services/api/reactors.service";
import type { ReactorData } from "@/types/models/reactor";
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
import { Zap, Loader2 } from "lucide-react";
import { Link, useSearchParams } from "react-router-dom";
import { NpcImage } from "@/components/features/npc/NpcImage";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import { useDebounce } from "@/lib/utils/debounce";

const MIN_QUERY_LENGTH = 2;
const DEBOUNCE_MS = 250;

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

  const reactorsQuery = useQuery<ReactorData[], Error>({
    queryKey: ["reactors", "search", activeTenant?.id ?? "no-tenant", urlQuery],
    queryFn: () => reactorsService.searchReactors(urlQuery),
    enabled: !!activeTenant && urlQuery.length >= MIN_QUERY_LENGTH,
    staleTime: 30 * 1000,
    placeholderData: keepPreviousData,
  });

  const reactors = reactorsQuery.data ?? [];
  const fetching = reactorsQuery.isFetching;
  const hasSearched = urlQuery.length >= MIN_QUERY_LENGTH;

  const handleClear = () => {
    setSearchInput("");
    setSearchParams({}, { replace: true });
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
            <div className="flex-1 relative">
              <Input
                placeholder="Enter reactor ID or name..."
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
              {reactors.length > 0 && (
                <span className="ml-2 text-muted-foreground font-normal">
                  ({reactors.length} {reactors.length === 1 ? "reactor" : "reactors"})
                </span>
              )}
            </CardTitle>
          </CardHeader>
          <CardContent className="flex-1 min-h-0 flex flex-col">
            {reactors.length === 0 ? (
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
