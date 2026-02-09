"use client"

import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { usePortalScript } from "@/lib/hooks/api/usePortalScripts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { PageLoader } from "@/components/common/PageLoader";
import { ErrorDisplay } from "@/components/common/ErrorDisplay";
import Link from "next/link";
import { useTenant } from "@/context/tenant-context";
import { mapEntitiesService, type MapPortalData } from "@/services/api/map-entities.service";

export default function PortalDetailPage() {
  const params = useParams();
  const mapId = params.id as string;
  const portalId = params.portalId as string;
  const { activeTenant } = useTenant();

  const { data: portal, isLoading, error, refetch } = useQuery({
    queryKey: ['maps', mapId, 'portals', portalId],
    queryFn: () => mapEntitiesService.getPortal(mapId, portalId, activeTenant!),
    enabled: !!mapId && !!portalId && !!activeTenant,
    staleTime: 5 * 60 * 1000,
  });

  const { data: script, isLoading: scriptLoading } = usePortalScript(portalId);

  if (isLoading) {
    return <PageLoader />;
  }

  if (error || !portal) {
    return (
      <div className="p-10">
        <ErrorDisplay error={error ?? "Portal not found"} retry={() => refetch()} />
      </div>
    );
  }

  const attrs = portal.attributes;

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      <div className="flex items-center gap-3">
        <h2 className="text-2xl font-bold tracking-tight">{attrs.name}</h2>
        <span className="text-muted-foreground font-mono">Portal #{portal.id}</span>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card>
          <CardHeader><CardTitle className="text-sm font-medium">Portal Info</CardTitle></CardHeader>
          <CardContent className="space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Type</span>
              <span>{attrs.type}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Position</span>
              <span className="font-mono">({attrs.x}, {attrs.y})</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Source Map</span>
              <Link href={`/maps/${mapId}`} className="font-mono text-primary hover:underline">
                {mapId}
              </Link>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Target</span>
              <span>{attrs.target || "-"}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Target Map</span>
              {attrs.targetMapId ? (
                <Link href={`/maps/${attrs.targetMapId}`} className="font-mono text-primary hover:underline">
                  {attrs.targetMapId}
                </Link>
              ) : (
                <span>-</span>
              )}
            </div>
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
                {attrs.scriptName && (
                  <p className="text-muted-foreground font-mono">{attrs.scriptName}</p>
                )}
                {script.attributes.description && (
                  <p className="text-muted-foreground">{script.attributes.description}</p>
                )}
              </div>
            ) : attrs.scriptName ? (
              <div className="space-y-2 text-sm">
                <Badge variant="destructive">Missing Script</Badge>
                <p className="text-muted-foreground font-mono">{attrs.scriptName}</p>
                <p className="text-xs text-muted-foreground">
                  This portal references a script that has not been seeded.
                </p>
              </div>
            ) : (
              <Badge variant="secondary">No Script</Badge>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
