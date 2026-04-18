import { useEffect, useState } from "react";
import { Map as MapIcon } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { useTenant } from "@/context/tenant-context";
import { getMapImageUrl } from "@/lib/utils/asset-url";

type ImageState = "render" | "minimap" | "placeholder";

interface MapImagePanelProps {
  mapId: string;
  mapName: string;
  /**
   * Initial image kind to attempt. Phase 1 uses `"minimap"`; Phase 2 uses
   * `"render"` with automatic fallback to `"minimap"`.
   */
  initialKind?: "render" | "minimap";
}

export function MapImagePanel({ mapId, mapName, initialKind = "minimap" }: MapImagePanelProps) {
  const { activeTenant } = useTenant();
  const [state, setState] = useState<ImageState>(initialKind);

  useEffect(() => {
    setState(initialKind);
  }, [mapId, initialKind]);

  if (!activeTenant) {
    return (
      <Card className="w-full aspect-video flex items-center justify-center">
        <CardContent className="flex flex-col items-center gap-2 text-muted-foreground">
          <MapIcon className="w-10 h-10" />
          <p className="text-sm">No active tenant</p>
        </CardContent>
      </Card>
    );
  }

  const handleError = () => {
    if (state === "render") {
      setState("minimap");
      return;
    }
    if (state === "minimap") {
      setState("placeholder");
    }
  };

  if (state === "placeholder") {
    return (
      <Card className="w-full min-h-[200px] flex items-center justify-center bg-muted/30">
        <CardContent className="flex flex-col items-center gap-2 py-10 text-muted-foreground">
          <MapIcon className="w-10 h-10" />
          <p className="text-sm">No render available</p>
        </CardContent>
      </Card>
    );
  }

  const url = getMapImageUrl(
    activeTenant.id,
    activeTenant.attributes.region,
    activeTenant.attributes.majorVersion,
    activeTenant.attributes.minorVersion,
    mapId,
    state,
  );

  return (
    <Card className="w-full overflow-hidden">
      <CardContent className="p-0">
        <img
          key={`${mapId}-${state}`}
          src={url}
          alt={`Map render for ${mapName}`}
          loading="lazy"
          className="w-full h-auto object-contain bg-muted/20"
          onError={handleError}
        />
      </CardContent>
    </Card>
  );
}
