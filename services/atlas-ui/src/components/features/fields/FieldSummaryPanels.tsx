import { useMemo } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type { LiveMonsterData } from "@/services/api/live-monsters.service";

interface FieldSummaryPanelsProps {
  characterCount: number;
  liveMonsters: LiveMonsterData[] | undefined;
}

/**
 * FR-20: character count, live monsters grouped by monster id with counts,
 * and the tracked-object slot. The tracked-object source does not exist on
 * `main` (Task 22 supplies it) — this always renders "—", never a fabricated
 * number.
 */
export function FieldSummaryPanels({
  characterCount,
  liveMonsters,
}: FieldSummaryPanelsProps) {
  const monsterGroups = useMemo(() => {
    if (!liveMonsters) return undefined;
    const counts = new Map<number, number>();
    const order: number[] = [];
    for (const monster of liveMonsters) {
      const id = monster.attributes.monsterId;
      if (counts.has(id)) {
        counts.set(id, (counts.get(id) ?? 0) + 1);
      } else {
        counts.set(id, 1);
        order.push(id);
      }
    }
    return order.map((id) => ({ id, count: counts.get(id) ?? 1 }));
  }, [liveMonsters]);

  return (
    <Card className="h-full">
      <CardContent className="pt-6 space-y-6">
        <section>
          <h3 className="text-sm font-semibold mb-2">Characters</h3>
          <p className="text-2xl font-bold" data-testid="field-character-count">
            {characterCount}
          </p>
        </section>

        <section>
          <h3 className="text-sm font-semibold mb-2">
            Monsters {monsterGroups && `(${monsterGroups.length})`}
          </h3>
          {monsterGroups === undefined ? (
            <div className="space-y-2">
              <Skeleton className="h-6 w-full" />
              <Skeleton className="h-6 w-full" />
            </div>
          ) : monsterGroups.length === 0 ? (
            <p className="text-sm italic text-muted-foreground">
              No live monsters
            </p>
          ) : (
            <ul className="space-y-1">
              {monsterGroups.map((group) => (
                <li
                  key={group.id}
                  data-testid="field-monster-group"
                  className="flex items-center justify-between text-sm"
                >
                  <span>Monster {group.id}</span>
                  <span className="text-xs text-muted-foreground">
                    ×{group.count}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </section>

        <section>
          <h3 className="text-sm font-semibold mb-2">Tracked Objects</h3>
          <p
            className="text-2xl font-bold text-muted-foreground"
            data-testid="field-tracked-object-count"
          >
            —
          </p>
        </section>
      </CardContent>
    </Card>
  );
}
