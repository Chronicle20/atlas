import { Link } from "react-router-dom";
import { TableCell, TableRow } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { NpcImage } from "@/components/features/npc/NpcImage";
import { useHoverHighlight } from "@/components/features/maps/HoverHighlightContext";
import { useMobData } from "@/lib/hooks/useMobData";
import { cn } from "@/lib/utils";
import type { MapMonsterData } from "@/services/api/map-entities.service";

interface MonsterTableRowProps {
  monster: MapMonsterData;
  spawnIndex?: number;
}

export function MonsterTableRow({ monster, spawnIndex }: MonsterTableRowProps) {
  const { name, iconUrl } = useMobData(monster.attributes.template);
  const { setHovered, isHovered } = useHoverHighlight();
  const template = monster.attributes.template;
  const target =
    spawnIndex !== undefined
      ? ({ kind: "monster", template, spawnIndex } as const)
      : ({ kind: "monster", template } as const);
  const highlighted = isHovered(target);

  return (
    <TableRow
      onPointerEnter={() => setHovered(target)}
      onPointerLeave={() => setHovered(null)}
      className={cn(
        "!border-l-2 border-l-transparent",
        highlighted && "bg-muted/60 !border-l-rose-500",
      )}
    >
      <TableCell>
        <NpcImage
          npcId={monster.attributes.template}
          iconUrl={iconUrl}
          size={32}
          lazy={true}
          showRetryButton={false}
          maxRetries={1}
        />
      </TableCell>
      <TableCell>
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <Link to={`/monsters/${monster.attributes.template}`}>
                <Badge variant="secondary">{name ?? "\u2014"}</Badge>
              </Link>
            </TooltipTrigger>
            <TooltipContent copyable>
              <p>{monster.attributes.template}</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </TableCell>
      <TableCell className="font-mono">
        ({monster.attributes.x}, {monster.attributes.y})
      </TableCell>
      <TableCell>{monster.attributes.mobTime}</TableCell>
      <TableCell>{monster.attributes.team}</TableCell>
    </TableRow>
  );
}
