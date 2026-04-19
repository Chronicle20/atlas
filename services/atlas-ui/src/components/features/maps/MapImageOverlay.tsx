import { useEffect, useMemo } from "react";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { useMobData } from "@/lib/hooks/useMobData";
import { worldToOverlayPercent, type MapBounds } from "@/lib/utils/map-overlay";
import type {
  MapMonsterData,
  MapNpcData,
  MapPortalData,
  MapReactorData,
} from "@/services/api/map-entities.service";
import { useHoverHighlight, type HoverTarget } from "./HoverHighlightContext";

export type MarkerSize = "default" | "large";

interface MapImageOverlayProps {
  bounds: MapBounds;
  portals?: MapPortalData[] | undefined;
  npcs?: MapNpcData[] | undefined;
  monsters?: MapMonsterData[] | undefined;
  reactors?: MapReactorData[] | undefined;
  size?: MarkerSize;
}

interface MarkerSizing {
  primary: string;
  monster: string;
}

const MARKER_SIZES: Record<MarkerSize, MarkerSizing> = {
  default: {
    primary: "h-[10px] w-[10px]",
    monster: "h-[8px] w-[8px]",
  },
  large: {
    primary: "h-[18px] w-[18px]",
    monster: "h-[14px] w-[14px]",
  },
};

interface ComputedMarker<T> {
  key: string;
  entity: T;
  pos: { left: string; top: string };
  outOfBounds: boolean;
}

function computeMarkers<T extends { id: string; attributes: { x: number; y: number } }>(
  entities: T[] | undefined,
  bounds: MapBounds,
): ComputedMarker<T>[] {
  if (!entities) return [];
  return entities.map((e, i) => {
    const pos = worldToOverlayPercent(e.attributes.x, e.attributes.y, bounds);
    const leftNum = parseFloat(pos.left);
    const topNum = parseFloat(pos.top);
    const outOfBounds = leftNum < 0 || leftNum > 100 || topNum < 0 || topNum > 100;
    return { key: `${e.id}-${i}`, entity: e, pos, outOfBounds };
  });
}

export function MapImageOverlay({
  bounds,
  portals,
  npcs,
  monsters,
  reactors,
  size = "default",
}: MapImageOverlayProps) {
  const sizing = MARKER_SIZES[size];
  const monsterMarkers = useMemo(() => computeMarkers(monsters, bounds), [monsters, bounds]);
  const reactorMarkers = useMemo(() => computeMarkers(reactors, bounds), [reactors, bounds]);
  const npcMarkers = useMemo(() => computeMarkers(npcs, bounds), [npcs, bounds]);
  const portalMarkers = useMemo(() => computeMarkers(portals, bounds), [portals, bounds]);

  useEffect(() => {
    if (!import.meta.env.DEV) return;
    const oob: string[] = [];
    for (const m of monsterMarkers) if (m.outOfBounds) oob.push(`monster:${m.entity.id}`);
    for (const m of reactorMarkers) if (m.outOfBounds) oob.push(`reactor:${m.entity.id}`);
    for (const m of npcMarkers) if (m.outOfBounds) oob.push(`npc:${m.entity.id}`);
    for (const m of portalMarkers) if (m.outOfBounds) oob.push(`portal:${m.entity.id}`);
    if (oob.length > 0) {
      console.warn("[MapImageOverlay] entities outside bounds:", oob);
    }
  }, [monsterMarkers, reactorMarkers, npcMarkers, portalMarkers]);

  return (
    <TooltipProvider delayDuration={100}>
      <div className="absolute inset-0 pointer-events-none">
        {monsterMarkers.map((m, i) => (
          <MonsterMarker
            key={m.key}
            monster={m.entity}
            spawnIndex={i}
            pos={m.pos}
            sizing={sizing}
          />
        ))}
        {reactorMarkers.map((m) => (
          <ReactorMarker key={m.key} reactor={m.entity} pos={m.pos} sizing={sizing} />
        ))}
        {npcMarkers.map((m, i) => (
          <NpcMarker key={m.key} npc={m.entity} spawnIndex={i} pos={m.pos} sizing={sizing} />
        ))}
        {portalMarkers.map((m) => (
          <PortalMarker key={m.key} portal={m.entity} pos={m.pos} sizing={sizing} />
        ))}
      </div>
    </TooltipProvider>
  );
}

interface MarkerShellProps {
  pos: { left: string; top: string };
  target: NonNullable<HoverTarget>;
  ariaLabel: string;
  tooltip: string;
  className: string;
  highlightRingColor: string;
}

function MarkerShell({
  pos,
  target,
  ariaLabel,
  tooltip,
  className,
  highlightRingColor,
}: MarkerShellProps) {
  const { setHovered, isHovered } = useHoverHighlight();
  const highlighted = isHovered(target);

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          aria-label={ariaLabel}
          onPointerEnter={() => setHovered(target)}
          onPointerLeave={() => setHovered(null)}
          onFocus={() => setHovered(target)}
          onBlur={() => setHovered(null)}
          style={{ left: pos.left, top: pos.top }}
          className={cn(
            "absolute -translate-x-1/2 -translate-y-1/2 pointer-events-auto transition-transform outline-none",
            className,
            highlighted && "scale-150 opacity-100 ring-2",
            highlighted && highlightRingColor,
          )}
        />
      </TooltipTrigger>
      <TooltipContent>{tooltip}</TooltipContent>
    </Tooltip>
  );
}

function PortalMarker({
  portal,
  pos,
  sizing,
}: {
  portal: MapPortalData;
  pos: { left: string; top: string };
  sizing: MarkerSizing;
}) {
  return (
    <MarkerShell
      pos={pos}
      target={{ kind: "portal", portalId: portal.id }}
      ariaLabel={`Portal: ${portal.attributes.name || portal.id}`}
      tooltip={portal.attributes.name || portal.id}
      className={cn(sizing.primary, "rotate-45 bg-emerald-500/70 border-2 border-white")}
      highlightRingColor="ring-yellow-400"
    />
  );
}

function NpcMarker({
  npc,
  spawnIndex,
  pos,
  sizing,
}: {
  npc: MapNpcData;
  spawnIndex: number;
  pos: { left: string; top: string };
  sizing: MarkerSizing;
}) {
  return (
    <MarkerShell
      pos={pos}
      target={{ kind: "npc", template: npc.attributes.template, spawnIndex }}
      ariaLabel={`NPC: ${npc.attributes.name}`}
      tooltip={npc.attributes.name}
      className={cn(sizing.primary, "rounded-full bg-sky-500/70 border-2 border-white")}
      highlightRingColor="ring-yellow-400"
    />
  );
}

function ReactorMarker({
  reactor,
  pos,
  sizing,
}: {
  reactor: MapReactorData;
  pos: { left: string; top: string };
  sizing: MarkerSizing;
}) {
  return (
    <MarkerShell
      pos={pos}
      target={{ kind: "reactor", reactorId: reactor.id }}
      ariaLabel={`Reactor: ${reactor.attributes.name || reactor.attributes.classification}`}
      tooltip={reactor.attributes.name || String(reactor.attributes.classification)}
      className={cn(sizing.primary, "bg-amber-500/70 border-2 border-white")}
      highlightRingColor="ring-yellow-400"
    />
  );
}

function MonsterMarker({
  monster,
  spawnIndex,
  pos,
  sizing,
}: {
  monster: MapMonsterData;
  spawnIndex: number;
  pos: { left: string; top: string };
  sizing: MarkerSizing;
}) {
  const { name } = useMobData(monster.attributes.template);
  return (
    <MarkerShell
      pos={pos}
      target={{ kind: "monster", template: monster.attributes.template, spawnIndex }}
      ariaLabel={`Monster: ${name ?? monster.attributes.template}`}
      tooltip={name ?? String(monster.attributes.template)}
      className={cn(sizing.monster, "rounded-full bg-rose-500/70 border border-white")}
      highlightRingColor="ring-yellow-400"
    />
  );
}
