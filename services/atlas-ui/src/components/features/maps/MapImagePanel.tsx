import { useEffect, useState } from "react";
import { Download, Map as MapIcon, Maximize2 } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useTenant } from "@/context/tenant-context";
import { getMapImageUrl } from "@/lib/utils/asset-url";
import type { MapArea } from "@/services/api/maps.service";
import type {
  MapMonsterData,
  MapNpcData,
  MapPortalData,
  MapReactorData,
} from "@/services/api/map-entities.service";
import { MapImageOverlay } from "./MapImageOverlay";
import { useHoverHighlight } from "./HoverHighlightContext";

type ImageState = "render" | "minimap" | "placeholder";

interface MapImagePanelProps {
  mapId: string;
  mapName: string;
  /**
   * Initial image kind to attempt. `"render"` prefers the full composite and
   * falls back to `"minimap"` then `"placeholder"` on 404.
   */
  initialKind?: "render" | "minimap";
  mapArea?: MapArea | null;
  portals?: MapPortalData[] | undefined;
  npcs?: MapNpcData[] | undefined;
  monsters?: MapMonsterData[] | undefined;
  reactors?: MapReactorData[] | undefined;
}

const PREVIEW_MAX_HEIGHT = "max-h-[320px]";

export function MapImagePanel({
  mapId,
  mapName,
  initialKind = "render",
  mapArea = null,
  portals,
  npcs,
  monsters,
  reactors,
}: MapImagePanelProps) {
  const { activeTenant } = useTenant();
  const { setHovered } = useHoverHighlight();
  const [state, setState] = useState<ImageState>(initialKind);
  const [expanded, setExpanded] = useState(false);

  useEffect(() => {
    setState(initialKind);
    setExpanded(false);
  }, [mapId, initialKind]);

  if (!activeTenant) {
    return (
      <Card className={`w-full ${PREVIEW_MAX_HEIGHT} flex items-center justify-center`}>
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
      <Card className={`w-full ${PREVIEW_MAX_HEIGHT} flex items-center justify-center bg-muted/30`}>
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
  const altText = `Map render for ${mapName}`;
  const downloadName = `${mapName || mapId}-${state}.png`;
  const overlayEnabled = state === "render" && mapArea != null;

  const handleDialogOpenChange = (open: boolean) => {
    setExpanded(open);
    if (!open) {
      setHovered(null);
    }
  };

  return (
    <>
      <Card className="w-full overflow-hidden">
        <CardContent className="p-0">
          <button
            type="button"
            onClick={() => setExpanded(true)}
            className="group relative block w-full cursor-zoom-in focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            aria-label={`Expand ${altText}`}
          >
            {overlayEnabled && mapArea ? (
              <div
                className={`relative mx-auto max-w-full ${PREVIEW_MAX_HEIGHT} bg-muted/20`}
                style={{
                  aspectRatio: `${mapArea.width} / ${mapArea.height}`,
                  width: "fit-content",
                }}
              >
                <img
                  key={`${mapId}-${state}`}
                  src={url}
                  alt={altText}
                  loading="lazy"
                  className="block w-full h-full object-cover"
                  onError={handleError}
                />
                <MapImageOverlay
                  bounds={mapArea}
                  portals={portals}
                  npcs={npcs}
                  monsters={monsters}
                  reactors={reactors}
                />
              </div>
            ) : (
              <img
                key={`${mapId}-${state}`}
                src={url}
                alt={altText}
                loading="lazy"
                className={`w-full ${PREVIEW_MAX_HEIGHT} object-contain bg-muted/20`}
                onError={handleError}
              />
            )}
            <span className="absolute bottom-2 right-2 flex items-center gap-1 rounded-md bg-background/80 px-2 py-1 text-xs font-medium text-foreground shadow-sm opacity-0 group-hover:opacity-100 group-focus-visible:opacity-100 transition-opacity">
              <Maximize2 className="w-3 h-3" />
              Expand
            </span>
          </button>
        </CardContent>
      </Card>

      <Dialog open={expanded} onOpenChange={handleDialogOpenChange}>
        <DialogContent className="max-w-[95vw] max-h-[95vh] w-auto p-4 sm:p-6 flex flex-col">
          <DialogHeader className="flex-row items-center justify-between gap-4 pr-8">
            <div className="min-w-0">
              <DialogTitle className="truncate">{mapName}</DialogTitle>
              <DialogDescription className="font-mono text-xs">
                #{mapId} · {state === "render" ? "full render" : "minimap"}
              </DialogDescription>
            </div>
            <a
              href={url}
              download={downloadName}
              className="inline-flex items-center gap-1 rounded-md border px-2 py-1 text-xs hover:bg-muted"
            >
              <Download className="w-3 h-3" />
              Download
            </a>
          </DialogHeader>
          <div className="flex-1 overflow-auto rounded-md border bg-muted/20 min-h-0">
            {overlayEnabled && mapArea ? (
              <div
                className="relative"
                style={{
                  aspectRatio: `${mapArea.width} / ${mapArea.height}`,
                  width: mapArea.width,
                  maxWidth: "none",
                }}
              >
                <img src={url} alt={altText} className="block w-full h-full" />
                <MapImageOverlay
                  bounds={mapArea}
                  portals={portals}
                  npcs={npcs}
                  monsters={monsters}
                  reactors={reactors}
                />
              </div>
            ) : (
              <img src={url} alt={altText} className="max-w-none block" />
            )}
          </div>
          <DialogClose className="sr-only">Close</DialogClose>
        </DialogContent>
      </Dialog>
    </>
  );
}
