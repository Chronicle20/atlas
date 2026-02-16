"use client"

import Link from "next/link";
import { TableCell, TableRow } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { NpcImage } from "@/components/features/npc/NpcImage";
import { useMobData } from "@/lib/hooks/useMobData";
import type { MapMonsterData } from "@/services/api/map-entities.service";

interface MonsterTableRowProps {
  monster: MapMonsterData;
}

export function MonsterTableRow({ monster }: MonsterTableRowProps) {
  const { name, iconUrl } = useMobData(monster.attributes.template);

  return (
    <TableRow>
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
              <Link href={`/monsters/${monster.attributes.template}`}>
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
