import { useMemo } from "react";
import { useParams } from "react-router-dom";
import { useMonster, useMonsterMaps } from "@/lib/hooks/api/useMonsters";
import { useMonsterDrops } from "@/lib/hooks/api/useDrops";
import { useMobData } from "@/lib/hooks/useMobData";
import { useItemBatchData } from "@/lib/hooks/useItemData";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { PageLoader } from "@/components/common/PageLoader";
import { ErrorDisplay } from "@/components/common/ErrorDisplay";
import { MonsterHeader } from "@/components/features/monsters/MonsterHeader";
import { MonsterDropWidget } from "@/components/features/monsters/MonsterDropWidget";
import { MonsterMesoWidget } from "@/components/features/monsters/MonsterMesoWidget";
import { MonsterSpawnMapWidget } from "@/components/features/monsters/MonsterSpawnMapWidget";
import { MonsterSkillChip } from "@/components/features/monsters/MonsterSkillChip";
import { getItemType } from "@/types/models/item";
import type { DropData } from "@/types/models/drop";

type DropGroupKey = "equipment" | "consumable" | "setup" | "etc" | "cash" | "other";

const DROP_GROUP_ORDER: DropGroupKey[] = [
  "equipment",
  "consumable",
  "setup",
  "etc",
  "cash",
  "other",
];

const DROP_GROUP_LABEL: Record<DropGroupKey, string> = {
  equipment: "Equipment",
  consumable: "Consumable",
  setup: "Setup",
  etc: "Etc",
  cash: "Cash",
  other: "Other",
};

function groupDrops(drops: DropData[]) {
  const mesos: DropData[] = [];
  const groups: Record<DropGroupKey, DropData[]> = {
    equipment: [],
    consumable: [],
    setup: [],
    etc: [],
    cash: [],
    other: [],
  };
  for (const drop of drops) {
    if (drop.attributes.itemId === 0) {
      mesos.push(drop);
      continue;
    }
    const type = getItemType(String(drop.attributes.itemId));
    switch (type) {
      case "Equipment":
        groups.equipment.push(drop);
        break;
      case "Consumable":
        groups.consumable.push(drop);
        break;
      case "Setup":
        groups.setup.push(drop);
        break;
      case "Etc":
        groups.etc.push(drop);
        break;
      case "Cash":
      case "Pet":
        groups.cash.push(drop);
        break;
      default:
        groups.other.push(drop);
    }
  }
  return { mesos, groups };
}

export function MonsterDetailPage() {
  const params = useParams();
  const id = params.id as string;

  const { data: monster, isLoading, error, refetch } = useMonster(id);
  const { data: drops, isLoading: dropsLoading } = useMonsterDrops(id);
  const {
    data: spawnMaps,
    isLoading: spawnMapsLoading,
    error: spawnMapsError,
    refetch: refetchSpawnMaps,
  } = useMonsterMaps(id);
  const { iconUrl: monsterIconUrl } = useMobData(parseInt(id));

  const itemIds = useMemo(
    () =>
      drops
        ?.filter((d) => d.attributes.itemId !== 0)
        .map((d) => d.attributes.itemId) ?? [],
    [drops],
  );
  useItemBatchData(itemIds);

  const { mesos, groups } = useMemo(
    () => groupDrops(drops ?? []),
    [drops],
  );

  const sortedSpawnMaps = useMemo(() => {
    if (!spawnMaps) return [];
    return [...spawnMaps].sort((a, b) => {
      const diff = b.attributes.spawnCount - a.attributes.spawnCount;
      if (diff !== 0) return diff;
      return a.attributes.name.localeCompare(b.attributes.name);
    });
  }, [spawnMaps]);

  if (isLoading) {
    return <PageLoader />;
  }

  if (error || !monster) {
    return (
      <div className="p-10">
        <ErrorDisplay error={error ?? "Monster not found"} retry={() => refetch()} />
      </div>
    );
  }

  const attrs = monster.attributes;

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16 overflow-y-auto">
      <MonsterHeader
        monsterId={monster.id}
        name={attrs.name}
        iconUrl={monsterIconUrl}
        boss={attrs.boss}
        undead={attrs.undead}
        friendly={attrs.friendly}
      />

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
        <Card>
          <CardHeader className="py-2 px-4">
            <CardTitle className="text-sm font-medium">Combat Stats</CardTitle>
          </CardHeader>
          <CardContent className="py-2 px-4 space-y-1 text-sm">
            <div className="flex justify-between"><span className="text-muted-foreground">Level</span><span>{attrs.level}</span></div>
            <div className="flex justify-between"><span className="text-muted-foreground">HP</span><span>{attrs.hp.toLocaleString()}</span></div>
            <div className="flex justify-between"><span className="text-muted-foreground">MP</span><span>{attrs.mp.toLocaleString()}</span></div>
            <div className="flex justify-between"><span className="text-muted-foreground">EXP</span><span>{attrs.experience.toLocaleString()}</span></div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="py-2 px-4">
            <CardTitle className="text-sm font-medium">Attack / Defense</CardTitle>
          </CardHeader>
          <CardContent className="py-2 px-4 space-y-1 text-sm">
            <div className="flex justify-between"><span className="text-muted-foreground">Weapon Attack</span><span>{attrs.weapon_attack}</span></div>
            <div className="flex justify-between"><span className="text-muted-foreground">Weapon Defense</span><span>{attrs.weapon_defense}</span></div>
            <div className="flex justify-between"><span className="text-muted-foreground">Magic Attack</span><span>{attrs.magic_attack}</span></div>
            <div className="flex justify-between"><span className="text-muted-foreground">Magic Defense</span><span>{attrs.magic_defense}</span></div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="py-2 px-4">
            <CardTitle className="text-sm font-medium">Properties</CardTitle>
          </CardHeader>
          <CardContent className="py-2 px-4 space-y-1 text-sm">
            <div className="flex justify-between"><span className="text-muted-foreground">First Attack</span><span>{attrs.first_attack ? "Yes" : "No"}</span></div>
            <div className="flex justify-between"><span className="text-muted-foreground">FFA Loot</span><span>{attrs.ffa_loot ? "Yes" : "No"}</span></div>
            <div className="flex justify-between"><span className="text-muted-foreground">Explosive Reward</span><span>{attrs.explosive_reward ? "Yes" : "No"}</span></div>
            <div className="flex justify-between"><span className="text-muted-foreground">CP</span><span>{attrs.cp}</span></div>
          </CardContent>
        </Card>
      </div>

      {attrs.skills && attrs.skills.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium">Skills</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-2">
              {attrs.skills.map((skill, i) => (
                <MonsterSkillChip
                  key={`${skill.id}-${skill.level}-${i}`}
                  skillId={skill.id}
                  level={skill.level}
                />
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium">
            Drops {drops && `(${drops.length})`}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {dropsLoading ? (
            <p className="text-sm text-muted-foreground">Loading drops...</p>
          ) : drops && drops.length > 0 ? (
            <>
              {mesos.length > 0 && (
                <div className="space-y-2">
                  <h3 className="text-sm font-medium text-muted-foreground uppercase tracking-wide">
                    Mesos
                  </h3>
                  <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
                    {mesos.map((drop) => (
                      <MonsterMesoWidget key={drop.id} drop={drop} />
                    ))}
                  </div>
                </div>
              )}
              {DROP_GROUP_ORDER.map((key) => {
                const items = groups[key];
                if (items.length === 0) return null;
                return (
                  <div key={key} className="space-y-2">
                    <h3 className="text-sm font-medium text-muted-foreground uppercase tracking-wide">
                      {DROP_GROUP_LABEL[key]} ({items.length})
                    </h3>
                    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
                      {items.map((drop) => (
                        <MonsterDropWidget key={drop.id} drop={drop} />
                      ))}
                    </div>
                  </div>
                );
              })}
            </>
          ) : (
            <p className="text-sm text-muted-foreground">
              No drops configured for this monster.
            </p>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium">
            Spawn Locations {sortedSpawnMaps.length > 0 && `(${sortedSpawnMaps.length})`}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {spawnMapsLoading ? (
            <p className="text-sm text-muted-foreground">Loading spawn locations...</p>
          ) : spawnMapsError ? (
            <ErrorDisplay error={spawnMapsError} retry={() => refetchSpawnMaps()} />
          ) : sortedSpawnMaps.length > 0 ? (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
              {sortedSpawnMaps.map((entry) => (
                <MonsterSpawnMapWidget key={entry.id} entry={entry} />
              ))}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">
              This monster does not spawn on any loaded map.
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
