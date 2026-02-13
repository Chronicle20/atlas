"use client"

import { useParams } from "next/navigation";
import { useMap } from "@/lib/hooks/api/useMaps";
import { useMapPortals, useMapNpcs, useMapReactors, useMapMonsters } from "@/lib/hooks/api/useMapEntities";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { PageLoader } from "@/components/common/PageLoader";
import { ErrorDisplay } from "@/components/common/ErrorDisplay";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import Link from "next/link";
import { NpcImage } from "@/components/features/npc/NpcImage";
import { MonsterTableRow } from "@/components/features/monsters/MonsterTableRow";
import { useTenant } from "@/context/tenant-context";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

export default function MapDetailPage() {
  const params = useParams();
  const id = params.id as string;

  const { activeTenant } = useTenant();
  const { data: map, isLoading, error, refetch } = useMap(id);
  const { data: portals, isLoading: portalsLoading } = useMapPortals(id);
  const { data: npcs, isLoading: npcsLoading } = useMapNpcs(id);
  const { data: reactors, isLoading: reactorsLoading } = useMapReactors(id);
  const { data: monsters, isLoading: monstersLoading } = useMapMonsters(id);

  if (isLoading) {
    return <PageLoader />;
  }

  if (error || !map) {
    return (
      <div className="p-10">
        <ErrorDisplay error={error ?? "Map not found"} retry={() => refetch()} />
      </div>
    );
  }

  const attrs = map.attributes;

  return (
    <div className="flex flex-col flex-1 min-h-0 space-y-6 p-10 pb-16">
      <div className="flex items-center gap-3">
        <h2 className="text-2xl font-bold tracking-tight">{attrs.name}</h2>
        <span className="text-muted-foreground font-mono">#{map.id}</span>
      </div>
      {attrs.streetName && (
        <p className="text-muted-foreground">{attrs.streetName}</p>
      )}

      <Tabs defaultValue="portals" className="flex-1 flex flex-col min-h-0">
        <TabsList>
          <TabsTrigger value="portals">
            Portals {portals && `(${portals.length})`}
          </TabsTrigger>
          <TabsTrigger value="npcs">
            NPCs {npcs && `(${npcs.length})`}
          </TabsTrigger>
          <TabsTrigger value="monsters">
            Monsters {monsters && `(${monsters.length})`}
          </TabsTrigger>
          <TabsTrigger value="reactors">
            Reactors {reactors && `(${reactors.length})`}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="portals" className="flex-1 min-h-0 overflow-hidden">
          <Card className="h-full flex flex-col">
            <CardContent className="pt-6 flex-1 overflow-auto">
              {portalsLoading ? (
                <p className="text-sm text-muted-foreground">Loading portals...</p>
              ) : portals && portals.length > 0 ? (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Name</TableHead>
                      <TableHead>Type</TableHead>
                      <TableHead>Position</TableHead>
                      <TableHead>Target Map</TableHead>
                      <TableHead>Script</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {portals.map((portal) => (
                      <TableRow key={portal.id}>
                        <TableCell>
                          <Link
                            href={`/maps/${id}/portals/${portal.id}`}
                            className="text-primary hover:underline"
                          >
                            {portal.attributes.name || portal.id}
                          </Link>
                        </TableCell>
                        <TableCell>{portal.attributes.type}</TableCell>
                        <TableCell className="font-mono">
                          ({portal.attributes.x}, {portal.attributes.y})
                        </TableCell>
                        <TableCell>
                          {portal.attributes.targetMapId ? (
                            <Link
                              href={`/maps/${portal.attributes.targetMapId}`}
                              className="font-mono text-primary hover:underline"
                            >
                              {portal.attributes.targetMapId}
                            </Link>
                          ) : "-"}
                        </TableCell>
                        <TableCell>
                          {portal.attributes.scriptName ? (
                            <Badge variant="outline">{portal.attributes.scriptName}</Badge>
                          ) : "-"}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              ) : (
                <p className="text-sm text-muted-foreground">No portals on this map.</p>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="npcs" className="flex-1 min-h-0 overflow-hidden">
          <Card className="h-full flex flex-col">
            <CardContent className="pt-6 flex-1 overflow-auto">
              {npcsLoading ? (
                <p className="text-sm text-muted-foreground">Loading NPCs...</p>
              ) : npcs && npcs.length > 0 ? (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-10">Icon</TableHead>
                      <TableHead>NPC ID</TableHead>
                      <TableHead>Name</TableHead>
                      <TableHead>Position</TableHead>
                      <TableHead>Hidden</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {npcs.map((npc) => (
                      <TableRow key={npc.id}>
                        <TableCell>
                          <NpcImage
                            npcId={npc.attributes.template}
                            iconUrl={activeTenant ? getAssetIconUrl(
                              activeTenant.id,
                              activeTenant.attributes.region,
                              activeTenant.attributes.majorVersion,
                              activeTenant.attributes.minorVersion,
                              'npc',
                              npc.attributes.template,
                            ) : undefined}
                            size={32}
                            lazy={true}
                            showRetryButton={false}
                            maxRetries={1}
                          />
                        </TableCell>
                        <TableCell>
                          <Link href={`/npcs/${npc.attributes.template}`} className="font-mono text-primary hover:underline">
                            {npc.attributes.template}
                          </Link>
                        </TableCell>
                        <TableCell>{npc.attributes.name}</TableCell>
                        <TableCell className="font-mono">
                          ({npc.attributes.x}, {npc.attributes.y})
                        </TableCell>
                        <TableCell>
                          {npc.attributes.hide && <Badge variant="secondary">Hidden</Badge>}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              ) : (
                <p className="text-sm text-muted-foreground">No NPCs on this map.</p>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="monsters" className="flex-1 min-h-0 overflow-hidden">
          <Card className="h-full flex flex-col">
            <CardContent className="pt-6 flex-1 overflow-auto">
              {monstersLoading ? (
                <p className="text-sm text-muted-foreground">Loading monsters...</p>
              ) : monsters && monsters.length > 0 ? (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-10">Icon</TableHead>
                      <TableHead>Monster ID</TableHead>
                      <TableHead>Name</TableHead>
                      <TableHead>Position</TableHead>
                      <TableHead>Mob Time</TableHead>
                      <TableHead>Team</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {monsters.map((monster) => (
                      <MonsterTableRow key={monster.id} monster={monster} />
                    ))}
                  </TableBody>
                </Table>
              ) : (
                <p className="text-sm text-muted-foreground">No monsters on this map.</p>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="reactors" className="flex-1 min-h-0 overflow-hidden">
          <Card className="h-full flex flex-col">
            <CardContent className="pt-6 flex-1 overflow-auto">
              {reactorsLoading ? (
                <p className="text-sm text-muted-foreground">Loading reactors...</p>
              ) : reactors && reactors.length > 0 ? (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-10">Icon</TableHead>
                      <TableHead>Reactor ID</TableHead>
                      <TableHead>Name</TableHead>
                      <TableHead>Position</TableHead>
                      <TableHead>Delay</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {reactors.map((reactor) => (
                      <TableRow key={reactor.id}>
                        <TableCell>
                          <NpcImage
                            npcId={reactor.attributes.classification}
                            iconUrl={activeTenant ? getAssetIconUrl(
                              activeTenant.id,
                              activeTenant.attributes.region,
                              activeTenant.attributes.majorVersion,
                              activeTenant.attributes.minorVersion,
                              'reactor',
                              reactor.attributes.classification,
                            ) : undefined}
                            size={32}
                            lazy={true}
                            showRetryButton={false}
                            maxRetries={1}
                          />
                        </TableCell>
                        <TableCell>
                          <Link
                            href={`/reactors/${reactor.attributes.classification}`}
                            className="font-mono text-primary hover:underline"
                          >
                            {reactor.attributes.classification}
                          </Link>
                        </TableCell>
                        <TableCell>{reactor.attributes.name}</TableCell>
                        <TableCell className="font-mono">
                          ({reactor.attributes.x}, {reactor.attributes.y})
                        </TableCell>
                        <TableCell>{reactor.attributes.delay}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              ) : (
                <p className="text-sm text-muted-foreground">No reactors on this map.</p>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
