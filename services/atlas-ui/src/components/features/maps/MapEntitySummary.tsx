import { Link } from "react-router-dom";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { NpcImage } from "@/components/features/npc/NpcImage";
import { useTenant } from "@/context/tenant-context";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import { useMobData } from "@/lib/hooks/useMobData";
import type { MapMonsterData, MapNpcData } from "@/services/api/map-entities.service";

interface MapEntitySummaryProps {
  npcs: MapNpcData[] | undefined;
  npcsError?: unknown;
  monsters: MapMonsterData[] | undefined;
  monstersError?: unknown;
}

export function MapEntitySummary({ npcs, npcsError, monsters, monstersError }: MapEntitySummaryProps) {
  return (
    <Card className="h-full">
      <CardContent className="pt-6 space-y-6">
        <NpcsSection npcs={npcs} error={npcsError} />
        <MonstersSection monsters={monsters} error={monstersError} />
      </CardContent>
    </Card>
  );
}

function NpcsSection({ npcs, error }: { npcs: MapNpcData[] | undefined; error?: unknown }) {
  const { activeTenant } = useTenant();

  if (error) {
    return (
      <section>
        <h3 className="text-sm font-semibold mb-2">NPCs</h3>
        <p className="text-sm text-destructive">Failed to load NPCs</p>
      </section>
    );
  }

  if (npcs === undefined) {
    return (
      <section>
        <h3 className="text-sm font-semibold mb-2">NPCs</h3>
        <div className="space-y-2">
          <Skeleton className="h-8 w-full" />
          <Skeleton className="h-8 w-full" />
          <Skeleton className="h-8 w-full" />
        </div>
      </section>
    );
  }

  const deduped = Array.from(
    new Map(npcs.map((n) => [n.attributes.template, n])).values(),
  );

  return (
    <section>
      <h3 className="text-sm font-semibold mb-2">NPCs ({deduped.length})</h3>
      {deduped.length === 0 ? (
        <p className="text-sm italic text-muted-foreground">No NPCs</p>
      ) : (
        <ul className="max-h-[400px] overflow-y-auto space-y-1 pr-2">
          {deduped.map((n) => {
            const iconUrl = activeTenant
              ? getAssetIconUrl(
                  activeTenant.id,
                  activeTenant.attributes.region,
                  activeTenant.attributes.majorVersion,
                  activeTenant.attributes.minorVersion,
                  "npc",
                  n.attributes.template,
                )
              : undefined;
            return (
              <li key={n.attributes.template} className="flex items-center gap-2">
                <NpcImage
                  npcId={n.attributes.template}
                  iconUrl={iconUrl}
                  size={32}
                  lazy
                  showRetryButton={false}
                  maxRetries={1}
                />
                <Link
                  to={`/npcs/${n.attributes.template}`}
                  className="text-sm text-primary hover:underline"
                >
                  {n.attributes.name}
                </Link>
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}

function MonstersSection({ monsters, error }: { monsters: MapMonsterData[] | undefined; error?: unknown }) {
  if (error) {
    return (
      <section>
        <h3 className="text-sm font-semibold mb-2">Monsters</h3>
        <p className="text-sm text-destructive">Failed to load monsters</p>
      </section>
    );
  }

  if (monsters === undefined) {
    return (
      <section>
        <h3 className="text-sm font-semibold mb-2">Monsters</h3>
        <div className="space-y-2">
          <Skeleton className="h-8 w-full" />
          <Skeleton className="h-8 w-full" />
          <Skeleton className="h-8 w-full" />
        </div>
      </section>
    );
  }

  const counts = new Map<number, number>();
  const order: number[] = [];
  for (const m of monsters) {
    const t = m.attributes.template;
    if (counts.has(t)) {
      counts.set(t, (counts.get(t) ?? 0) + 1);
    } else {
      counts.set(t, 1);
      order.push(t);
    }
  }

  return (
    <section>
      <h3 className="text-sm font-semibold mb-2">Monsters ({order.length})</h3>
      {order.length === 0 ? (
        <p className="text-sm italic text-muted-foreground">No monsters</p>
      ) : (
        <ul className="max-h-[400px] overflow-y-auto space-y-1 pr-2">
          {order.map((template) => (
            <MonsterRow key={template} template={template} count={counts.get(template) ?? 1} />
          ))}
        </ul>
      )}
    </section>
  );
}

function MonsterRow({ template, count }: { template: number; count: number }) {
  const { name, iconUrl } = useMobData(template);
  return (
    <li className="flex items-center gap-2">
      <NpcImage
        npcId={template}
        iconUrl={iconUrl}
        size={32}
        lazy
        showRetryButton={false}
        maxRetries={1}
      />
      <Link
        to={`/monsters/${template}`}
        className="text-sm text-primary hover:underline"
      >
        {name ?? "\u2014"}
      </Link>
      <span className="text-xs text-muted-foreground">×{count}</span>
    </li>
  );
}
