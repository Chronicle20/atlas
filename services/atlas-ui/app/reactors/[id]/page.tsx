"use client"

import { useParams } from "next/navigation";
import { useReactor } from "@/lib/hooks/api/useReactors";
import { useReactorDrops } from "@/lib/hooks/api/useDrops";
import { useReactorScript } from "@/lib/hooks/api/useReactorScripts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { PageLoader } from "@/components/common/PageLoader";
import { ErrorDisplay } from "@/components/common/ErrorDisplay";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { NpcImage } from "@/components/features/npc/NpcImage";
import { useTenant } from "@/context/tenant-context";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import Link from "next/link";

export default function ReactorDetailPage() {
  const params = useParams();
  const id = params.id as string;

  const { activeTenant } = useTenant();
  const { data: reactor, isLoading, error, refetch } = useReactor(id);
  const { data: drops, isLoading: dropsLoading } = useReactorDrops(id);
  const { data: script, isLoading: scriptLoading } = useReactorScript(id);

  if (isLoading) {
    return <PageLoader />;
  }

  if (error || !reactor) {
    return (
      <div className="p-10">
        <ErrorDisplay error={error ?? "Reactor not found"} retry={() => refetch()} />
      </div>
    );
  }

  const attrs = reactor.attributes;
  const stateEntries = attrs.stateInfo ? Object.entries(attrs.stateInfo) : [];

  const iconUrl = activeTenant ? getAssetIconUrl(
    activeTenant.id,
    activeTenant.attributes.region,
    activeTenant.attributes.majorVersion,
    activeTenant.attributes.minorVersion,
    'reactor',
    parseInt(reactor.id),
  ) : undefined;

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      <div className="flex items-center gap-3">
        <NpcImage
          npcId={parseInt(reactor.id)}
          iconUrl={iconUrl}
          size={40}
          lazy={false}
          showRetryButton={false}
          maxRetries={2}
        />
        <div>
          <div className="flex items-center gap-2">
            <h2 className="text-2xl font-bold tracking-tight">{attrs.name || `Reactor ${reactor.id}`}</h2>
            <span className="text-muted-foreground font-mono">#{reactor.id}</span>
          </div>
          <p className="text-sm text-muted-foreground">
            <Link href="/reactors" className="hover:underline">Reactors</Link>
            {" > "}
            <span>{attrs.name || `Reactor ${reactor.id}`}</span>
          </p>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card>
          <CardHeader><CardTitle className="text-sm font-medium">Bounds</CardTitle></CardHeader>
          <CardContent className="space-y-2 text-sm">
            {attrs.tl && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">Top-Left</span>
                <span className="font-mono">({attrs.tl.x}, {attrs.tl.y})</span>
              </div>
            )}
            {attrs.br && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">Bottom-Right</span>
                <span className="font-mono">({attrs.br.x}, {attrs.br.y})</span>
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader><CardTitle className="text-sm font-medium">Script Status</CardTitle></CardHeader>
          <CardContent>
            {scriptLoading ? (
              <p className="text-sm text-muted-foreground">Loading...</p>
            ) : script ? (
              <div className="space-y-2 text-sm">
                <Badge variant="default">Script Available</Badge>
                {script.attributes.description && (
                  <p className="text-muted-foreground">{script.attributes.description}</p>
                )}
              </div>
            ) : (
              <Badge variant="secondary">No Script</Badge>
            )}
          </CardContent>
        </Card>
      </div>

      {stateEntries.length > 0 && (
        <Card>
          <CardHeader><CardTitle className="text-sm font-medium">States</CardTitle></CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>State</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Next State</TableHead>
                  <TableHead>Item</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {stateEntries.flatMap(([stateKey, states]) =>
                  states.map((state, i) => (
                    <TableRow key={`${stateKey}-${i}`}>
                      <TableCell className="font-mono">{stateKey}</TableCell>
                      <TableCell>{state.type}</TableCell>
                      <TableCell>{state.nextState}</TableCell>
                      <TableCell className="font-mono">
                        {state.reactorItem ? `${state.reactorItem.itemId} x${state.reactorItem.quantity}` : "-"}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium">
            Drops {drops && `(${drops.length})`}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {dropsLoading ? (
            <p className="text-sm text-muted-foreground">Loading drops...</p>
          ) : drops && drops.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Item ID</TableHead>
                  <TableHead>Chance</TableHead>
                  <TableHead>Quest ID</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {drops.map((drop) => (
                  <TableRow key={drop.id}>
                    <TableCell className="font-mono">{drop.attributes.itemId}</TableCell>
                    <TableCell>{drop.attributes.chance.toLocaleString()}</TableCell>
                    <TableCell className="font-mono">{drop.attributes.questId || "-"}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <p className="text-sm text-muted-foreground">No drops configured for this reactor.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
