import { Link } from "react-router-dom";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { NpcImage } from "@/components/features/npc/NpcImage";
import { MonsterTableRow } from "@/components/features/monsters/MonsterTableRow";
import { MapCell } from "@/components/map-cell";
import { useTenant } from "@/context/tenant-context";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import type {
  MapMonsterData,
  MapPortalData,
  MapReactorData,
} from "@/services/api/map-entities.service";

const NONE_MAP_ID = 999999999;

interface MapDetailTabsProps {
  mapId: string;
  portals: MapPortalData[] | undefined;
  portalsError?: unknown;
  monsters: MapMonsterData[] | undefined;
  monstersError?: unknown;
  reactors: MapReactorData[] | undefined;
  reactorsError?: unknown;
}

export function MapDetailTabs({
  mapId,
  portals,
  portalsError,
  monsters,
  monstersError,
  reactors,
  reactorsError,
}: MapDetailTabsProps) {
  const { activeTenant } = useTenant();

  return (
    <Tabs defaultValue="portals" className="flex-1 flex flex-col min-h-0">
      <TabsList>
        <TabsTrigger value="portals">
          Portals {portals && `(${portals.length})`}
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
            {portalsError ? (
              <p className="text-sm text-destructive">Failed to load portals.</p>
            ) : portals === undefined ? (
              <p className="text-sm text-muted-foreground">Loading portals...</p>
            ) : portals.length > 0 ? (
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
                          to={`/maps/${mapId}/portals/${portal.id}`}
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
                          Number(portal.attributes.targetMapId) === NONE_MAP_ID ? (
                            <Badge variant="secondary">NONE</Badge>
                          ) : (
                            <Link to={`/maps/${portal.attributes.targetMapId}`}>
                              <MapCell
                                mapId={String(portal.attributes.targetMapId)}
                                tenant={activeTenant}
                              />
                            </Link>
                          )
                        ) : (
                          "-"
                        )}
                      </TableCell>
                      <TableCell>
                        {portal.attributes.scriptName ? (
                          <Badge variant="outline">{portal.attributes.scriptName}</Badge>
                        ) : (
                          "-"
                        )}
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

      <TabsContent value="monsters" className="flex-1 min-h-0 overflow-hidden">
        <Card className="h-full flex flex-col">
          <CardContent className="pt-6 flex-1 overflow-auto">
            {monstersError ? (
              <p className="text-sm text-destructive">Failed to load monsters.</p>
            ) : monsters === undefined ? (
              <p className="text-sm text-muted-foreground">Loading monsters...</p>
            ) : monsters.length > 0 ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-10">Icon</TableHead>
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
            {reactorsError ? (
              <p className="text-sm text-destructive">Failed to load reactors.</p>
            ) : reactors === undefined ? (
              <p className="text-sm text-muted-foreground">Loading reactors...</p>
            ) : reactors.length > 0 ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-10">Icon</TableHead>
                    <TableHead>Template</TableHead>
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
                          iconUrl={
                            activeTenant
                              ? getAssetIconUrl(
                                  activeTenant.id,
                                  activeTenant.attributes.region,
                                  activeTenant.attributes.majorVersion,
                                  activeTenant.attributes.minorVersion,
                                  "reactor",
                                  reactor.attributes.classification,
                                )
                              : undefined
                          }
                          size={32}
                          lazy
                          showRetryButton={false}
                          maxRetries={1}
                        />
                      </TableCell>
                      <TableCell>
                        <Link
                          to={`/reactors/${reactor.attributes.classification}`}
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
  );
}
