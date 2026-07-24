import { ArrowDown, ArrowUp, Minus } from "lucide-react";
import { useTenant } from "@/context/tenant-context";
import { useCharacter } from "@/lib/hooks/api/useCharacters";
import { OptimizedCharacterRenderer } from "@/components/features/characters/OptimizedCharacterRenderer";
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
  // Lazy per-row appearance fetch. Fails open: when the tenant isn't ready
  // yet or the character fetch errors/hasn't resolved, render a neutral
  // placeholder instead of blocking the row's text cells.
  const characterQuery = useCharacter(activeTenant!, String(a.characterId));

  return (
    <tr className="border-b">
      <td className="px-3 py-2 font-mono">#{rank}</td>
      <td className="px-3 py-2">
        {characterQuery.data ? (
          <OptimizedCharacterRenderer
            character={characterQuery.data}
            size="small"
            lazy
            fallbackAvatar="/logo.png"
          />
        ) : (
          <div className="h-12 w-12 rounded bg-muted" aria-hidden="true" />
        )}
      </td>
      <td className="px-3 py-2 font-medium">{a.name}</td>
      <td className="px-3 py-2">{a.level}</td>
      <td className="px-3 py-2">{a.jobId}</td>
      <td className="px-3 py-2">
        <MoveArrow move={move} />
      </td>
    </tr>
  );
}
