import { useTenant } from "@/context/tenant-context";
import { Suspense, useEffect, useMemo, useState } from "react";
import { keepPreviousData, useQueries, useQuery } from "@tanstack/react-query";
import { monstersService } from "@/services/api/monsters.service";
import type { MonsterData } from "@/types/models/monster";
import { Skeleton } from "@/components/ui/skeleton";
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
import { Skull, Loader2 } from "lucide-react";
import { Link, useSearchParams } from "react-router-dom";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import { useDebounce } from "@/lib/utils/debounce";

const MIN_QUERY_LENGTH = 2;
const DEBOUNCE_MS = 250;

export function MonstersPage() {
  return (
    <Suspense>
      <MonstersPageContent />
    </Suspense>
  );
}

function MonstersPageContent() {
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

  const monstersQuery = useQuery<MonsterData[], Error>({
    queryKey: ["monsters", "search", activeTenant?.id ?? "no-tenant", urlQuery],
    queryFn: () => monstersService.searchMonsters(urlQuery),
    enabled: !!activeTenant && urlQuery.length >= MIN_QUERY_LENGTH,
    staleTime: 30 * 1000,
    placeholderData: keepPreviousData,
  });

  const monsters = monstersQuery.data ?? [];
  const fetching = monstersQuery.isFetching;
  const hasSearched = urlQuery.length >= MIN_QUERY_LENGTH;

  const tenantId = activeTenant?.id ?? "no-tenant";
  const detailQueries = useQueries({
    queries: monsters.map((m) => ({
      queryKey: ["monsters", "detail", tenantId, m.id],
      queryFn: () => monstersService.getMonsterById(m.id),
      enabled: !!activeTenant,
      staleTime: 5 * 60 * 1000,
    })),
  });

  const detailsById = useMemo(() => {
    const out = new Map<string, MonsterData>();
    for (const q of detailQueries) {
      if (q.data) out.set(q.data.id, q.data);
    }
    return out;
  }, [detailQueries]);

  const handleClear = () => {
    setSearchInput("");
    setSearchParams({}, { replace: true });
  };

  return (
    <div className="flex flex-col flex-1 min-h-0 space-y-6 p-10 pb-16">
      <div className="flex items-center gap-2">
        <Skull className="h-6 w-6" />
        <h2 className="text-2xl font-bold tracking-tight">Monsters</h2>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Search Monsters</CardTitle>
          <CardDescription>
            Search for monsters by ID or name. Results are limited to 50 entries.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex gap-4 items-end">
            <div className="flex-1 relative">
              <Input
                placeholder="Enter monster ID or name..."
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
              {monsters.length > 0 && (
                <span className="ml-2 text-muted-foreground font-normal">
                  ({monsters.length} {monsters.length === 1 ? "monster" : "monsters"})
                </span>
              )}
            </CardTitle>
          </CardHeader>
          <CardContent className="flex-1 min-h-0 flex flex-col">
            {monsters.length === 0 ? (
              <div className="text-center py-8 text-muted-foreground">
                No monsters found matching your search criteria.
              </div>
            ) : (
              <div className="rounded-md border flex-1 min-h-0 overflow-auto">
                <Table>
                  <TableHeader className="sticky top-0 bg-background z-10">
                    <TableRow>
                      <TableHead className="w-10">Icon</TableHead>
                      <TableHead>Name</TableHead>
                      <TableHead>Level</TableHead>
                      <TableHead>HP</TableHead>
                      <TableHead>EXP</TableHead>
                      <TableHead>Tags</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {monsters.map((monster) => {
                      const iconUrl = activeTenant ? getAssetIconUrl(
                        activeTenant.id,
                        activeTenant.attributes.region,
                        activeTenant.attributes.majorVersion,
                        activeTenant.attributes.minorVersion,
                        'mob',
                        parseInt(monster.id),
                      ) : undefined;
                      const detail = detailsById.get(monster.id)?.attributes;
                      return (
                        <TableRow key={monster.id}>
                          <TableCell>
                            {iconUrl ? (
                              <img
                                src={iconUrl}
                                alt={monster.attributes.name}
                                width={32}
                                height={32}
                                className="object-contain"
                              />
                            ) : null}
                          </TableCell>
                          <TableCell>
                            <TooltipProvider>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <Link to={`/monsters/${monster.id}`}>
                                    <Badge variant="secondary">{monster.attributes.name}</Badge>
                                  </Link>
                                </TooltipTrigger>
                                <TooltipContent copyable>
                                  <p>{monster.id}</p>
                                </TooltipContent>
                              </Tooltip>
                            </TooltipProvider>
                          </TableCell>
                          <TableCell>{detail ? detail.level : <Skeleton className="h-4 w-8" />}</TableCell>
                          <TableCell>{detail ? detail.hp.toLocaleString() : <Skeleton className="h-4 w-16" />}</TableCell>
                          <TableCell>{detail ? detail.experience.toLocaleString() : <Skeleton className="h-4 w-16" />}</TableCell>
                          <TableCell>
                            {detail ? (
                              <div className="flex gap-1">
                                {detail.boss && <Badge variant="destructive">Boss</Badge>}
                                {detail.undead && <Badge variant="secondary">Undead</Badge>}
                                {detail.friendly && <Badge variant="outline">Friendly</Badge>}
                              </div>
                            ) : (
                              <Skeleton className="h-4 w-20" />
                            )}
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
