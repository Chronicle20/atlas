import { useMemo } from "react";
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
import { cn } from "@/lib/utils";
import type {
  MapMonsterData,
  MapPortalData,
  MapReactorData,
} from "@/services/api/map-entities.service";
import { useHoverHighlight } from "./HoverHighlightContext";

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
  const sortedPortals = useMemo(() => {
    if (!portals) return portals;
    return [...portals].sort((a, b) => {
      const an = a.attributes.name || "";
      const bn = b.attributes.name || "";
      if (an && bn) return an.localeCompare(bn);
      if (an) return -1;
      if (bn) return 1;
      return a.id.localeCompare(b.id);
    });
  }, [portals]);

  const sortedMonsters = useMemo(() => {
    if (!monsters) return monsters;
    return [...monsters].sort((a, b) => {
      if (a.attributes.template !== b.attributes.template) {
        return a.attributes.template - b.attributes.template;
      }
      if (a.attributes.x !== b.attributes.x) return a.attributes.x - b.attributes.x;
      return a.attributes.y - b.attributes.y;
    });
  }, [monsters]);

  const sortedReactors = useMemo(() => {
    if (!reactors) return reactors;
    return [...reactors].sort((a, b) => {
      const an = a.attributes.name || "";
      const bn = b.attributes.name || "";
      if (an && bn && an !== bn) return an.localeCompare(bn);
      if (an && !bn) return -1;
      if (!an && bn) return 1;
      return a.attributes.classification - b.attributes.classification;
    });
  }, [reactors]);

  return (
    <Tabs defaultValue="portals" className="flex flex-col">
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

      <TabsContent value="portals">
        <Card>
          <CardContent className="pt-6">
            {portalsError ? (
              <p className="text-sm text-destructive">Failed to load portals.</p>
            ) : sortedPortals === undefined ? (
              <p className="text-sm text-muted-foreground">Loading portals...</p>
            ) : sortedPortals.length > 0 ? (
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
                  {sortedPortals.map((portal) => (
                    <PortalRow key={portal.id} mapId={mapId} portal={portal} />
                  ))}
                </TableBody>
              </Table>
            ) : (
              <p className="text-sm text-muted-foreground">No portals on this map.</p>
            )}
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="monsters">
        <Card>
          <CardContent className="pt-6">
            {monstersError ? (
              <p className="text-sm text-destructive">Failed to load monsters.</p>
            ) : sortedMonsters === undefined ? (
              <p className="text-sm text-muted-foreground">Loading monsters...</p>
            ) : sortedMonsters.length > 0 ? (
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
                  {sortedMonsters.map((monster) => (
                    <MonsterTableRow
                      key={monster.id}
                      monster={monster}
                    />
                  ))}
                </TableBody>
              </Table>
            ) : (
              <p className="text-sm text-muted-foreground">No monsters on this map.</p>
            )}
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="reactors">
        <Card>
          <CardContent className="pt-6">
            {reactorsError ? (
              <p className="text-sm text-destructive">Failed to load reactors.</p>
            ) : sortedReactors === undefined ? (
              <p className="text-sm text-muted-foreground">Loading reactors...</p>
            ) : sortedReactors.length > 0 ? (
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
                  {sortedReactors.map((reactor) => (
                    <ReactorRow key={reactor.id} reactor={reactor} />
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

function PortalRow({ mapId, portal }: { mapId: string; portal: MapPortalData }) {
  const { activeTenant } = useTenant();
  const { setHovered, isHovered } = useHoverHighlight();
  const highlighted = isHovered({ kind: "portal", portalId: portal.id });
  return (
    <TableRow
      onPointerEnter={() => setHovered({ kind: "portal", portalId: portal.id })}
      onPointerLeave={() => setHovered(null)}
      className={cn(
        "!border-l-2 border-l-transparent",
        highlighted && "bg-muted/60 !border-l-emerald-500",
      )}
    >
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
  );
}

function ReactorRow({ reactor }: { reactor: MapReactorData }) {
  const { activeTenant } = useTenant();
  const { setHovered, isHovered } = useHoverHighlight();
  const highlighted = isHovered({ kind: "reactor", reactorId: reactor.id });
  return (
    <TableRow
      onPointerEnter={() => setHovered({ kind: "reactor", reactorId: reactor.id })}
      onPointerLeave={() => setHovered(null)}
      className={cn(
        "!border-l-2 border-l-transparent",
        highlighted && "bg-muted/60 !border-l-amber-500",
      )}
    >
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
  );
}
