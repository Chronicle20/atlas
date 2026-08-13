import { useMemo } from "react";
import { ArrowDown, ArrowUp, Minus } from "lucide-react";
import { useTenant } from "@/context/tenant-context";
import { useCharacter } from "@/lib/hooks/api/useCharacters";
import { useInventory } from "@/lib/hooks/api/useInventory";
import { OptimizedCharacterRenderer } from "@/components/features/characters/OptimizedCharacterRenderer";
import { useJobNameLookup } from "@/lib/hooks/api/useJobGraph";
import { Badge } from "@/components/ui/badge";
import type { Asset } from "@/services/api/inventory.service";
import type { RankingEntry } from "@/services/api/rankings.service";

interface LeaderboardRowProps {
  entry: RankingEntry;
  /** "overall" uses rank/rankMove; "job" uses jobRank/jobRankMove. */
  view: "overall" | "job";
}

function MoveArrow({ move }: { move: number }) {
  if (move > 0)
    // no semantic success token in the palette; matches repo convention
    return <ArrowUp className="h-4 w-4 text-green-600" aria-label="moved up" />;
  if (move < 0)
    return (
      <ArrowDown className="h-4 w-4 text-destructive" aria-label="moved down" />
    );
  return (
    <Minus className="h-4 w-4 text-muted-foreground" aria-label="no change" />
  );
}

export function LeaderboardRow({ entry, view }: LeaderboardRowProps) {
  const a = entry.attributes;
  const rank = view === "overall" ? a.rank : a.jobRank;
  const move = view === "overall" ? a.rankMove : a.jobRankMove;
  const { activeTenant } = useTenant();
  const jobName = useJobNameLookup();
  // Lazy per-row appearance fetch. Fails open: when the tenant isn't ready
  // yet or the character fetch errors/hasn't resolved, render a neutral
  // placeholder instead of blocking the row's text cells.
  const characterQuery = useCharacter(activeTenant!, String(a.characterId));
  const inventoryQuery = useInventory(activeTenant!, String(a.characterId));

  // Equipped items live in the negative-slot compartment (same filter the
  // account character tiles use); passing them lets the renderer draw the
  // character WITH gear rather than a naked base body.
  const equippedAssets = useMemo<Asset[]>(
    () =>
      inventoryQuery.data?.included?.filter(
        (item): item is Asset =>
          item.type === "assets" &&
          "slot" in item.attributes &&
          item.attributes.slot < 0,
      ) ?? [],
    [inventoryQuery.data],
  );

  return (
    <tr className="border-b">
      <td className="px-3 py-2 font-mono">#{rank}</td>
      <td className="px-3 py-2">
        {/* Fixed, clipped thumbnail box so the sprite (naturally taller than a
            table row) can never spill outside the row bounds. */}
        <div className="flex h-14 w-12 items-end justify-center overflow-hidden">
          {characterQuery.data ? (
            <OptimizedCharacterRenderer
              character={characterQuery.data}
              inventory={equippedAssets}
              size="small"
              lazy
              fallbackAvatar="/logo.png"
              className="h-14 w-12"
            />
          ) : (
            <div className="h-12 w-10 rounded bg-muted" aria-hidden="true" />
          )}
        </div>
      </td>
      <td className="px-3 py-2 font-medium">{a.name}</td>
      <td className="px-3 py-2">{a.level}</td>
      <td className="px-3 py-2">
        <Badge variant="secondary">{jobName(a.jobId)}</Badge>
      </td>
      <td className="px-3 py-2">
        <MoveArrow move={move} />
      </td>
    </tr>
  );
}
