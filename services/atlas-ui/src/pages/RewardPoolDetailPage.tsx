import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog";
import { PageLoader } from "@/components/common/PageLoader";
import { ErrorDisplay } from "@/components/common/ErrorDisplay";
import { useTenant } from "@/context/tenant-context";
import { useRewardPool, useRewardPoolItems, useGlobalRewardItems, useDeletePoolItem, useDeleteRewardPool } from "@/lib/hooks/api/useRewardPools";
import { useItemName } from "@/lib/hooks/api/useItemStrings";
import { useNPC } from "@/lib/hooks/api/useNpcs";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import { createErrorFromUnknown } from "@/types/api/errors";
import { PoolFormDialog } from "@/components/features/reward-pools/PoolFormDialog";
import { PoolItemDialog } from "@/components/features/reward-pools/PoolItemDialog";
import { PoolItemsTable } from "@/components/features/reward-pools/PoolItemsTable";
import type { RewardPoolItemData } from "@/types/models/reward-pool-item";
import type { Tenant } from "@/types/models/tenant";

function NpcChip({ npcId, tenant }: { npcId: number; tenant: Tenant }) {
  const { data: npc } = useNPC(tenant, npcId);
  return (
    <Link to={`/npcs/${npcId}`} className="hover:underline">
      <Badge variant="secondary">{npc?.name ?? npcId}</Badge>
    </Link>
  );
}

export function RewardPoolDetailPage() {
  const params = useParams();
  const id = params.id as string;
  const navigate = useNavigate();
  const { activeTenant } = useTenant();

  const { data: pool, isLoading, error, refetch } = useRewardPool(id);
  const itemsQuery = useRewardPoolItems(id);
  const isIncubator = pool?.attributes.kind === "incubator";
  const globalQuery = useGlobalRewardItems();
  const deleteItem = useDeletePoolItem();
  const deletePool = useDeleteRewardPool();

  const { data: eggName } = useItemName(isIncubator ? id : "");

  const [editPoolOpen, setEditPoolOpen] = useState(false);
  const [itemDialog, setItemDialog] = useState<{ open: boolean; item?: RewardPoolItemData }>({ open: false });
  const [itemDelete, setItemDelete] = useState<RewardPoolItemData | null>(null);
  const [poolDeleteOpen, setPoolDeleteOpen] = useState(false);

  if (isLoading) return <PageLoader />;
  if (error || !pool) {
    return (
      <div className="p-10">
        <ErrorDisplay error={error ?? "Reward pool not found"} retry={() => void refetch()} />
      </div>
    );
  }

  const attrs = pool.attributes;
  const items = itemsQuery.data ?? [];
  const globalItems = isIncubator ? [] : (globalQuery.data ?? []);
  const totalWeight = items.reduce((s, i) => s + i.attributes.weight, 0);
  const tierTotal = attrs.commonWeight + attrs.uncommonWeight + attrs.rareWeight;
  const eggIconUrl =
    isIncubator && activeTenant
      ? getAssetIconUrl(activeTenant.id, activeTenant.attributes.region, activeTenant.attributes.majorVersion, activeTenant.attributes.minorVersion, "item", parseInt(id, 10))
      : null;

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16 overflow-y-auto">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          {eggIconUrl && <img src={eggIconUrl} alt="" width={32} height={32} />}
          <h2 className="text-2xl font-bold tracking-tight">{attrs.name}</h2>
          {isIncubator ? (
            <Badge className="bg-amber-500/15 text-amber-600 dark:text-amber-400 border-transparent">Incubator</Badge>
          ) : (
            <Badge variant="secondary">Gachapon</Badge>
          )}
          <span className="text-muted-foreground font-mono">#{pool.id}</span>
        </div>
        <Button variant="outline" onClick={() => setEditPoolOpen(true)}>Edit Pool</Button>
      </div>

      {isIncubator ? (
        <Card>
          <CardHeader><CardTitle className="text-sm font-medium">Egg</CardTitle></CardHeader>
          <CardContent className="space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Egg item</span>
              <Link to={`/items/${pool.id}`} className="hover:underline">{eggName ?? pool.id}</Link>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-muted-foreground">Success NPC</span>
              {attrs.npcIds.length > 0 && activeTenant ? (
                <NpcChip npcId={attrs.npcIds[0] as number} tenant={activeTenant} />
              ) : (
                <span className="text-muted-foreground">none</span>
              )}
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Total weight</span>
              <span>{totalWeight}</span>
            </div>
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Card>
            <CardHeader><CardTitle className="text-sm font-medium">Tier Weights</CardTitle></CardHeader>
            <CardContent className="space-y-2 text-sm">
              {([["Common", attrs.commonWeight], ["Uncommon", attrs.uncommonWeight], ["Rare", attrs.rareWeight]] as const).map(([label, w]) => (
                <div key={label} className="flex justify-between">
                  <span className="text-muted-foreground">{label}</span>
                  <span>
                    {w}
                    <span className="text-muted-foreground ml-2">({tierTotal > 0 ? ((w / tierTotal) * 100).toFixed(1) : "0.0"}%)</span>
                  </span>
                </div>
              ))}
            </CardContent>
          </Card>
          <Card>
            <CardHeader><CardTitle className="text-sm font-medium">NPCs</CardTitle></CardHeader>
            <CardContent className="text-sm">
              {attrs.npcIds.length > 0 && activeTenant ? (
                <div className="flex flex-wrap gap-2">
                  {attrs.npcIds.map((npcId) => <NpcChip key={npcId} npcId={npcId} tenant={activeTenant} />)}
                </div>
              ) : (
                <span className="text-muted-foreground">No NPCs assigned</span>
              )}
            </CardContent>
          </Card>
        </div>
      )}

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="text-sm font-medium">Pool Items ({items.length})</CardTitle>
          <Button size="sm" onClick={() => setItemDialog({ open: true })}>Add Item</Button>
        </CardHeader>
        <CardContent>
          {itemsQuery.isLoading ? (
            <p className="text-sm text-muted-foreground">Loading pool items...</p>
          ) : (
            <PoolItemsTable
              kind={isIncubator ? "incubator" : "gachapon"}
              poolId={id}
              tierWeights={{ common: attrs.commonWeight, uncommon: attrs.uncommonWeight, rare: attrs.rareWeight }}
              items={items}
              globalItems={globalItems}
              onEdit={(item) => setItemDialog({ open: true, item })}
              onDelete={setItemDelete}
            />
          )}
        </CardContent>
      </Card>

      <Card className="border-destructive/40">
        <CardHeader><CardTitle className="text-sm font-medium">Danger Zone</CardTitle></CardHeader>
        <CardContent>
          <Button variant="destructive" onClick={() => setPoolDeleteOpen(true)}>Delete Pool</Button>
        </CardContent>
      </Card>

      <PoolFormDialog open={editPoolOpen} onOpenChange={setEditPoolOpen} mode="edit" pool={pool} />
      <PoolItemDialog
        open={itemDialog.open}
        onOpenChange={(open) => setItemDialog((s) => ({ ...s, open }))}
        kind={isIncubator ? "incubator" : "gachapon"}
        poolId={id}
        {...(itemDialog.item !== undefined && { item: itemDialog.item })}
      />

      <AlertDialog open={!!itemDelete} onOpenChange={(open) => !open && setItemDelete(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete item?</AlertDialogTitle>
            <AlertDialogDescription>Item {itemDelete?.attributes.itemId} will be removed from this pool.</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={async () => {
                try {
                  await deleteItem.mutateAsync({ poolId: id, itemRecordId: itemDelete!.id });
                  toast.success("Item deleted");
                } catch (e) {
                  toast.error(createErrorFromUnknown(e).message);
                } finally {
                  setItemDelete(null);
                }
              }}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={poolDeleteOpen} onOpenChange={setPoolDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete this pool?</AlertDialogTitle>
            <AlertDialogDescription>
              "{attrs.name}" and its reward assignments will be removed. This cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={async () => {
                try {
                  await deletePool.mutateAsync({ id });
                  toast.success("Pool deleted");
                  navigate("/reward-pools");
                } catch (e) {
                  toast.error(createErrorFromUnknown(e).message);
                }
              }}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
