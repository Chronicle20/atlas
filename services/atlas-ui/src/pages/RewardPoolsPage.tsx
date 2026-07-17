import { useMemo, useState } from "react";
import { RefreshCw } from "lucide-react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog";
import { DataTableWrapper } from "@/components/common/DataTableWrapper";
import { ErrorDisplay } from "@/components/common/ErrorDisplay";
import { PageLoader } from "@/components/common/PageLoader";
import { useGridRefresh } from "@/lib/hooks/useGridRefresh";
import { useTenant } from "@/context/tenant-context";
import { useRewardPools, useGlobalRewardItems, useDeleteGlobalItem } from "@/lib/hooks/api/useRewardPools";
import { poolColumns } from "./reward-pools-columns";
import { PoolFormDialog } from "@/components/features/reward-pools/PoolFormDialog";
import { PoolItemDialog } from "@/components/features/reward-pools/PoolItemDialog";
import { ItemNameCell } from "@/components/item-name-cell";
import { toast } from "sonner";
import { cn } from "@/lib/utils";
import { createErrorFromUnknown } from "@/types/api/errors";
import type { GlobalRewardItemData } from "@/types/models/global-reward-item";
import type { RewardPoolData } from "@/types/models/reward-pool";

export function RewardPoolsPage() {
  const { activeTenant } = useTenant();
  const poolsQuery = useRewardPools();
  const globalQuery = useGlobalRewardItems();
  const { isRefreshing, onRefresh } = useGridRefresh([poolsQuery, globalQuery]);
  const deleteGlobal = useDeleteGlobalItem();

  const [createOpen, setCreateOpen] = useState(false);
  const [globalDialog, setGlobalDialog] = useState<{ open: boolean; item?: GlobalRewardItemData }>({ open: false });
  const [globalDelete, setGlobalDelete] = useState<GlobalRewardItemData | null>(null);

  const pools = useMemo(() => poolsQuery.data ?? [], [poolsQuery.data]);
  const gachapons = useMemo(() => pools.filter((p) => p.attributes.kind === "gachapon"), [pools]);
  const incubators = useMemo(() => pools.filter((p) => p.attributes.kind === "incubator"), [pools]);
  const globalItems = globalQuery.data ?? [];
  const error = poolsQuery.error?.message ?? null;

  if (poolsQuery.isLoading) return <PageLoader />;

  const poolTable = (data: RewardPoolData[], emptyTitle: string, emptyDescription: string) => (
    <DataTableWrapper
      columns={poolColumns}
      data={data}
      error={error}
      emptyState={{ title: emptyTitle, description: emptyDescription }}
    />
  );

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold tracking-tight">Reward Pools</h2>
        <Button onClick={() => setCreateOpen(true)}>New Pool</Button>
      </div>

      <Tabs defaultValue="all">
        <div className="flex items-center justify-between">
          <TabsList>
            <TabsTrigger value="all">All ({pools.length})</TabsTrigger>
            <TabsTrigger value="gachapons">Gachapons ({gachapons.length})</TabsTrigger>
            <TabsTrigger value="incubators">Incubators ({incubators.length})</TabsTrigger>
            <TabsTrigger value="global">Global Pool ({globalItems.length})</TabsTrigger>
          </TabsList>
          <Button
            variant="outline"
            size="icon"
            onClick={onRefresh}
            disabled={isRefreshing}
            title="Refresh"
            aria-busy={isRefreshing}
          >
            <RefreshCw className={cn("h-4 w-4", isRefreshing && "animate-spin")} />
          </Button>
        </div>

        <TabsContent value="all" className="mt-4">
          {poolTable(pools, "No reward pools found", "Seed defaults from Setup, or create a pool.")}
        </TabsContent>
        <TabsContent value="gachapons" className="mt-4">
          {poolTable(gachapons, "No gachapon pools", "Seed defaults from Setup, or create one.")}
        </TabsContent>
        <TabsContent value="incubators" className="mt-4">
          {poolTable(incubators, "No incubator pools", "Seed defaults from Setup, or create one.")}
        </TabsContent>

        <TabsContent value="global" className="mt-4 space-y-4">
          <p className="text-sm text-muted-foreground">
            Global items merge into every gachapon machine's roll for their tier. They never apply to incubator pools.
          </p>
          <div className="flex justify-end">
            <Button onClick={() => setGlobalDialog({ open: true })}>Add Item</Button>
          </div>
          {globalQuery.isLoading ? (
            <PageLoader />
          ) : globalQuery.error ? (
            <ErrorDisplay error={globalQuery.error} retry={() => void globalQuery.refetch()} />
          ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Item</TableHead>
                <TableHead>Quantity</TableHead>
                <TableHead>Tier</TableHead>
                <TableHead className="w-24" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {globalItems.map((gi) => (
                <TableRow key={gi.id}>
                  <TableCell><ItemNameCell itemId={String(gi.attributes.itemId)} tenant={activeTenant} /></TableCell>
                  <TableCell>{gi.attributes.quantity}</TableCell>
                  <TableCell><Badge variant="outline">{gi.attributes.tier}</Badge></TableCell>
                  <TableCell className="space-x-2 text-right">
                    <Button variant="ghost" size="sm" onClick={() => setGlobalDialog({ open: true, item: gi })}>Edit</Button>
                    <Button variant="ghost" size="sm" onClick={() => setGlobalDelete(gi)}>Delete</Button>
                  </TableCell>
                </TableRow>
              ))}
              {globalItems.length === 0 && (
                <TableRow><TableCell colSpan={4} className="text-muted-foreground">No global items.</TableCell></TableRow>
              )}
            </TableBody>
          </Table>
          )}
        </TabsContent>
      </Tabs>

      <PoolFormDialog open={createOpen} onOpenChange={setCreateOpen} mode="create" />
      <PoolItemDialog
        open={globalDialog.open}
        onOpenChange={(open) => setGlobalDialog((s) => ({ ...s, open }))}
        kind="global"
        {...(globalDialog.item !== undefined && { item: globalDialog.item })}
      />
      <AlertDialog open={!!globalDelete} onOpenChange={(open) => !open && setGlobalDelete(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete global item?</AlertDialogTitle>
            <AlertDialogDescription>
              Item {globalDelete?.attributes.itemId} will stop appearing in every gachapon's {globalDelete?.attributes.tier} rolls.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={async () => {
                try {
                  await deleteGlobal.mutateAsync({ itemRecordId: globalDelete!.id });
                  toast.success("Global item deleted");
                } catch (e) {
                  toast.error(createErrorFromUnknown(e).message);
                } finally {
                  setGlobalDelete(null);
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
