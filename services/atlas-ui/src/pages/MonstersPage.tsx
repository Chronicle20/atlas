import { useTenant } from "@/context/tenant-context";
import { Suspense, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { monstersService } from "@/services/api/monsters.service";
import type { MonsterData } from "@/types/models/monster";
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
import { Skull, Search, Loader2 } from "lucide-react";
import { Link, useSearchParams } from "react-router-dom";
import { getAssetIconUrl } from "@/lib/utils/asset-url";

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

  const monstersQuery = useQuery<MonsterData[], Error>({
    queryKey: ["monsters", "search", activeTenant?.id ?? "no-tenant", urlQuery],
    queryFn: () => monstersService.searchMonsters(urlQuery),
    enabled: !!activeTenant && urlQuery.length > 0,
    staleTime: 30 * 1000,
  });

  const monsters = monstersQuery.data ?? [];
  const loading = monstersQuery.isFetching;
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
            <div className="flex-1">
              <Input
                placeholder="Enter monster ID or name..."
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
              {monsters.length > 0 && (
                <span className="ml-2 text-muted-foreground font-normal">
                  ({monsters.length} {monsters.length === 1 ? "monster" : "monsters"})
                </span>
              )}
            </CardTitle>
          </CardHeader>
          <CardContent className="flex-1 min-h-0 flex flex-col">
            {loading ? (
              <div className="text-center py-8 text-muted-foreground">Searching...</div>
            ) : monsters.length === 0 ? (
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
                          <TableCell>{monster.attributes.level}</TableCell>
                          <TableCell>{monster.attributes.hp.toLocaleString()}</TableCell>
                          <TableCell>{monster.attributes.experience.toLocaleString()}</TableCell>
                          <TableCell>
                            <div className="flex gap-1">
                              {monster.attributes.boss && <Badge variant="destructive">Boss</Badge>}
                              {monster.attributes.undead && <Badge variant="secondary">Undead</Badge>}
                              {monster.attributes.friendly && <Badge variant="outline">Friendly</Badge>}
                            </div>
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
