"use client"

import { useMemo } from "react";
import { useParams } from "next/navigation";
import { useGachapon, useGachaponPrizePool } from "@/lib/hooks/api/useGachapons";
import { useItemStrings } from "@/lib/hooks/api/useItemStrings";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { PageLoader } from "@/components/common/PageLoader";
import { ErrorDisplay } from "@/components/common/ErrorDisplay";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

function tierBadgeVariant(tier: string) {
  switch (tier) {
    case "rare":
      return "destructive";
    case "uncommon":
      return "secondary";
    default:
      return "outline";
  }
}

export default function GachaponDetailPage() {
  const params = useParams();
  const id = params.id as string;

  const { data: gachapon, isLoading, error, refetch } = useGachapon(id);
  const { data: prizePool, isLoading: poolLoading } = useGachaponPrizePool(id);
  const { data: itemStrings } = useItemStrings();

  const itemNameMap = useMemo(() => {
    if (!itemStrings) return new Map<string, string>();
    const map = new Map<string, string>();
    for (const item of itemStrings) {
      map.set(item.id, item.attributes.name);
    }
    return map;
  }, [itemStrings]);

  if (isLoading) {
    return <PageLoader />;
  }

  if (error || !gachapon) {
    return (
      <div className="p-10">
        <ErrorDisplay error={error ?? "Gachapon not found"} retry={() => refetch()} />
      </div>
    );
  }

  const attrs = gachapon.attributes;

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16 overflow-y-auto">
      <div className="flex items-center gap-3">
        <h2 className="text-2xl font-bold tracking-tight">{attrs.name}</h2>
        <span className="text-muted-foreground font-mono">#{gachapon.id}</span>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card>
          <CardHeader><CardTitle className="text-sm font-medium">Tier Weights</CardTitle></CardHeader>
          <CardContent className="space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Common</span>
              <span>{attrs.commonWeight}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Uncommon</span>
              <span>{attrs.uncommonWeight}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Rare</span>
              <span>{attrs.rareWeight}</span>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader><CardTitle className="text-sm font-medium">NPCs</CardTitle></CardHeader>
          <CardContent className="text-sm">
            {attrs.npcIds && attrs.npcIds.length > 0 ? (
              <div className="flex flex-wrap gap-2">
                {attrs.npcIds.map((npcId) => (
                  <span key={npcId} className="font-mono text-muted-foreground">{npcId}</span>
                ))}
              </div>
            ) : (
              <span className="text-muted-foreground">No NPCs assigned</span>
            )}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium">
            Prize Pool {prizePool && `(${prizePool.length})`}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {poolLoading ? (
            <p className="text-sm text-muted-foreground">Loading prize pool...</p>
          ) : prizePool && prizePool.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Item ID</TableHead>
                  <TableHead>Item Name</TableHead>
                  <TableHead>Quantity</TableHead>
                  <TableHead>Tier</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {prizePool.map((reward) => (
                  <TableRow key={`${reward.attributes.itemId}-${reward.attributes.tier}`}>
                    <TableCell className="font-mono">{reward.attributes.itemId}</TableCell>
                    <TableCell>{itemNameMap.get(String(reward.attributes.itemId)) ?? "-"}</TableCell>
                    <TableCell>{reward.attributes.quantity}</TableCell>
                    <TableCell>
                      <Badge variant={tierBadgeVariant(reward.attributes.tier)}>
                        {reward.attributes.tier}
                      </Badge>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <p className="text-sm text-muted-foreground">No items in the prize pool.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
