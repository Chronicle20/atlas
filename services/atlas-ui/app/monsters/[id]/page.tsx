"use client"

import { useParams } from "next/navigation";
import { useMonster } from "@/lib/hooks/api/useMonsters";
import { useMonsterDrops } from "@/lib/hooks/api/useDrops";
import { useMobData } from "@/lib/hooks/useMobData";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { PageLoader } from "@/components/common/PageLoader";
import { ErrorDisplay } from "@/components/common/ErrorDisplay";
import Image from "next/image";
import { shouldUnoptimizeImageSrc } from "@/lib/utils/image";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

export default function MonsterDetailPage() {
  const params = useParams();
  const id = params.id as string;

  const { data: monster, isLoading, error, refetch } = useMonster(id);
  const { data: drops, isLoading: dropsLoading } = useMonsterDrops(id);
  const { iconUrl: monsterIconUrl } = useMobData(parseInt(id));

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
      <div className="flex items-center gap-3">
        {monsterIconUrl && (
          <Image
            src={monsterIconUrl}
            alt={attrs.name}
            width={64}
            height={64}
            unoptimized={shouldUnoptimizeImageSrc(monsterIconUrl)}
            className="object-contain"
          />
        )}
        <h2 className="text-2xl font-bold tracking-tight">{attrs.name}</h2>
        <span className="text-muted-foreground font-mono">#{monster.id}</span>
        {attrs.boss && <Badge variant="destructive">Boss</Badge>}
        {attrs.undead && <Badge variant="secondary">Undead</Badge>}
        {attrs.friendly && <Badge variant="outline">Friendly</Badge>}
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        <Card>
          <CardHeader><CardTitle className="text-sm font-medium">Combat Stats</CardTitle></CardHeader>
          <CardContent className="space-y-2 text-sm">
            <div className="flex justify-between"><span className="text-muted-foreground">Level</span><span>{attrs.level}</span></div>
            <div className="flex justify-between"><span className="text-muted-foreground">HP</span><span>{attrs.hp.toLocaleString()}</span></div>
            <div className="flex justify-between"><span className="text-muted-foreground">MP</span><span>{attrs.mp.toLocaleString()}</span></div>
            <div className="flex justify-between"><span className="text-muted-foreground">EXP</span><span>{attrs.experience.toLocaleString()}</span></div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader><CardTitle className="text-sm font-medium">Attack / Defense</CardTitle></CardHeader>
          <CardContent className="space-y-2 text-sm">
            <div className="flex justify-between"><span className="text-muted-foreground">Weapon Attack</span><span>{attrs.weapon_attack}</span></div>
            <div className="flex justify-between"><span className="text-muted-foreground">Weapon Defense</span><span>{attrs.weapon_defense}</span></div>
            <div className="flex justify-between"><span className="text-muted-foreground">Magic Attack</span><span>{attrs.magic_attack}</span></div>
            <div className="flex justify-between"><span className="text-muted-foreground">Magic Defense</span><span>{attrs.magic_defense}</span></div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader><CardTitle className="text-sm font-medium">Properties</CardTitle></CardHeader>
          <CardContent className="space-y-2 text-sm">
            <div className="flex justify-between"><span className="text-muted-foreground">First Attack</span><span>{attrs.first_attack ? "Yes" : "No"}</span></div>
            <div className="flex justify-between"><span className="text-muted-foreground">FFA Loot</span><span>{attrs.ffa_loot ? "Yes" : "No"}</span></div>
            <div className="flex justify-between"><span className="text-muted-foreground">Explosive Reward</span><span>{attrs.explosive_reward ? "Yes" : "No"}</span></div>
            <div className="flex justify-between"><span className="text-muted-foreground">CP</span><span>{attrs.cp}</span></div>
          </CardContent>
        </Card>
      </div>

      {attrs.skills && attrs.skills.length > 0 && (
        <Card>
          <CardHeader><CardTitle className="text-sm font-medium">Skills</CardTitle></CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Skill ID</TableHead>
                  <TableHead>Level</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {attrs.skills.map((skill, i) => (
                  <TableRow key={i}>
                    <TableCell className="font-mono">{skill.id}</TableCell>
                    <TableCell>{skill.level}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium">
            Drops {drops && `(${drops.length})`}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {dropsLoading ? (
            <p className="text-sm text-muted-foreground">Loading drops...</p>
          ) : drops && drops.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Item ID</TableHead>
                  <TableHead>Chance</TableHead>
                  <TableHead>Min Qty</TableHead>
                  <TableHead>Max Qty</TableHead>
                  <TableHead>Quest ID</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {drops.map((drop) => (
                  <TableRow key={drop.id}>
                    <TableCell className="font-mono">{drop.attributes.itemId}</TableCell>
                    <TableCell>{drop.attributes.chance.toLocaleString()}</TableCell>
                    <TableCell>{drop.attributes.minimumQuantity}</TableCell>
                    <TableCell>{drop.attributes.maximumQuantity}</TableCell>
                    <TableCell className="font-mono">{drop.attributes.questId || "-"}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <p className="text-sm text-muted-foreground">No drops configured for this monster.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
