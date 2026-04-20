import { useEffect, useMemo, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ChevronDown,
  Download,
  MoreHorizontal,
  Plus,
  Trash2,
  Upload,
} from "lucide-react";
import { toast } from "sonner";

import { useTenant } from "@/context/tenant-context";
import { npcsService } from "@/services/api/npcs.service";
import { useItemBatchData } from "@/lib/hooks/useItemData";
import type { Commodity, CommodityAttributes } from "@/types/models/npc";

import {
  Card,
  CardAction,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ErrorDisplay } from "@/components/common/ErrorDisplay";
import { NpcShopCommodityWidget } from "@/components/features/npc/NpcShopCommodityWidget";
import { NpcShopCommodityDialog } from "@/components/features/npc/NpcShopCommodityDialog";

interface NpcShopCardProps {
  npcId: number;
  hasShop: boolean;
}

export function NpcShopCard({ npcId, hasShop }: NpcShopCardProps) {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();

  const shopKey = ["npcs", "shop", activeTenant?.id ?? "no-tenant", npcId] as const;

  const shopQuery = useQuery({
    queryKey: shopKey,
    queryFn: () => npcsService.getNPCShop(npcId),
    enabled: !!activeTenant && npcId > 0 && hasShop,
  });

  const shop = shopQuery.data;
  const commodities: Commodity[] = useMemo(() => shop?.included ?? [], [shop]);
  const [recharger, setRecharger] = useState<boolean>(false);

  useEffect(() => {
    setRecharger(shop?.data.attributes.recharger ?? false);
  }, [shop]);

  const templateIds = useMemo(
    () => commodities.map(c => c.attributes.templateId),
    [commodities],
  );
  const itemBatch = useItemBatchData(templateIds);
  const itemDataById = useMemo(() => {
    const m = new Map<number, { name?: string | undefined; iconUrl?: string | undefined }>();
    for (const entry of itemBatch.data) {
      m.set(entry.id, { name: entry.name, iconUrl: entry.iconUrl });
    }
    return m;
  }, [itemBatch.data]);

  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<Commodity | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Commodity | null>(null);
  const [deleteAllOpen, setDeleteAllOpen] = useState(false);
  const [bulkOpen, setBulkOpen] = useState(false);
  const [bulkJson, setBulkJson] = useState("");

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: shopKey });
    queryClient.invalidateQueries({
      queryKey: ["npcs", "detail", activeTenant?.id ?? "no-tenant", npcId],
    });
  };

  const handleCreate = async (attrs: CommodityAttributes) => {
    if (attrs.templateId <= 0) {
      toast.error("Template ID must be greater than zero");
      return;
    }
    if (attrs.mesoPrice <= 0 && attrs.tokenPrice <= 0) {
      toast.error("Either Meso Price or Token Price must be greater than zero");
      return;
    }
    try {
      await npcsService.createCommodity(npcId, attrs);
      toast.success("Commodity created successfully");
      setCreateOpen(false);
      invalidate();
    } catch (err) {
      toast.error(
        "Failed to create commodity: " +
          (err instanceof Error ? err.message : String(err)),
      );
    }
  };

  const handleUpdate = async (attrs: CommodityAttributes) => {
    if (!editing) return;
    if (attrs.templateId <= 0) {
      toast.error("Template ID must be greater than zero");
      return;
    }
    if (attrs.mesoPrice <= 0 && attrs.tokenPrice <= 0) {
      toast.error("Either Meso Price or Token Price must be greater than zero");
      return;
    }
    try {
      await npcsService.updateCommodity(npcId, editing.id, attrs);
      toast.success("Commodity updated successfully");
      setEditing(null);
      invalidate();
    } catch (err) {
      toast.error(
        "Failed to update commodity: " +
          (err instanceof Error ? err.message : String(err)),
      );
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    try {
      await npcsService.deleteCommodity(npcId, deleteTarget.id);
      toast.success("Commodity deleted");
      setDeleteTarget(null);
      invalidate();
    } catch (err) {
      toast.error(
        "Failed to delete commodity: " +
          (err instanceof Error ? err.message : String(err)),
      );
    }
  };

  const handleDeleteAll = async () => {
    try {
      await npcsService.deleteAllCommoditiesForNPC(npcId);
      toast.success("All commodities deleted");
      setDeleteAllOpen(false);
      invalidate();
    } catch (err) {
      toast.error(
        "Failed to delete all commodities: " +
          (err instanceof Error ? err.message : String(err)),
      );
    }
  };

  const handleRechargerToggle = async (checked: boolean) => {
    const previous = recharger;
    setRecharger(checked);
    try {
      await npcsService.updateShop(npcId, commodities, checked);
      toast.success("Recharger updated");
      invalidate();
    } catch (err) {
      setRecharger(previous);
      toast.error(
        "Failed to update recharger: " +
          (err instanceof Error ? err.message : String(err)),
      );
    }
  };

  const handleBulkUpdate = async () => {
    try {
      const parsed = JSON.parse(bulkJson);
      let toUpdate: Commodity[] = [];
      if (parsed.included?.length) toUpdate = parsed.included;
      else if (parsed.data?.included?.length) toUpdate = parsed.data.included;
      const rechargerValue =
        parsed.data?.attributes?.recharger ?? recharger;

      await npcsService.updateShop(npcId, toUpdate, rechargerValue);
      toast.success("Shop updated");
      setBulkOpen(false);
      setBulkJson("");
      invalidate();
    } catch (err) {
      toast.error(
        "Failed to bulk update shop: " +
          (err instanceof Error ? err.message : String(err)),
      );
    }
  };

  const handleExport = () => {
    const payload = {
      data: {
        type: "shops",
        id: `shop-${npcId}`,
        attributes: { npcId, recharger },
        relationships: {
          commodities: {
            data: commodities.map(c => ({ type: "commodities", id: c.id })),
          },
        },
      },
      included: commodities,
    };
    const blob = new Blob([JSON.stringify(payload, null, 2)], {
      type: "application/json",
    });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = `shop-${npcId}.json`;
    document.body.appendChild(anchor);
    anchor.click();
    document.body.removeChild(anchor);
    URL.revokeObjectURL(url);
    toast.success("Shop exported");
  };

  const showEmpty = hasShop && !shopQuery.isLoading && commodities.length === 0;

  return (
    <Card>
      <Collapsible defaultOpen={hasShop} className="flex flex-col gap-6">
        <CardHeader>
          <CollapsibleTrigger className="group flex items-center gap-2 cursor-pointer text-left">
            <ChevronDown className="h-4 w-4 text-muted-foreground transition-transform group-data-[state=closed]:-rotate-90" />
            <CardTitle className="text-sm font-medium">
              Shop{hasShop && commodities.length > 0 ? ` (${commodities.length})` : ""}
            </CardTitle>
          </CollapsibleTrigger>
          <CardAction>
            <div className="flex items-center gap-1">
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setCreateOpen(true)}
                title="Add Commodity"
                aria-label="Add Commodity"
              >
                <Plus className="h-4 w-4" />
              </Button>
              {hasShop && (
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon"
                      title="More actions"
                      aria-label="More shop actions"
                    >
                      <MoreHorizontal className="h-4 w-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem onSelect={() => setBulkOpen(true)}>
                      <Upload className="h-4 w-4" />
                      Bulk Update
                    </DropdownMenuItem>
                    <DropdownMenuItem onSelect={handleExport}>
                      <Download className="h-4 w-4" />
                      Export Shop
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      className="text-destructive focus:text-destructive"
                      onSelect={() => setDeleteAllOpen(true)}
                    >
                      <Trash2 className="h-4 w-4" />
                      Delete All Commodities
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              )}
            </div>
          </CardAction>
        </CardHeader>
        <CollapsibleContent>
          <CardContent className="flex flex-col gap-4">
            {hasShop && (
              <div className="flex items-center gap-2">
                <Switch
                  id={`recharger-${npcId}`}
                  checked={recharger}
                  onCheckedChange={handleRechargerToggle}
                />
                <Label
                  htmlFor={`recharger-${npcId}`}
                  className="text-sm font-medium"
                >
                  Recharger
                </Label>
              </div>
            )}
            {!hasShop ? (
              <p className="text-sm text-muted-foreground">
                No shop configured. Add a commodity to create one.
              </p>
            ) : shopQuery.isLoading ? (
              <div className="space-y-2">
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
              </div>
            ) : shopQuery.error ? (
              <ErrorDisplay
                error={shopQuery.error}
                retry={() => shopQuery.refetch()}
              />
            ) : showEmpty ? (
              <p className="text-sm text-muted-foreground">
                Shop has no commodities configured.
              </p>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-2">
                {commodities.map(commodity => {
                  const data = itemDataById.get(commodity.attributes.templateId);
                  return (
                    <NpcShopCommodityWidget
                      key={commodity.id}
                      templateId={commodity.attributes.templateId}
                      mesoPrice={commodity.attributes.mesoPrice}
                      tokenPrice={commodity.attributes.tokenPrice}
                      tokenTemplateId={commodity.attributes.tokenTemplateId}
                      {...(data?.name !== undefined && { name: data.name })}
                      {...(data?.iconUrl !== undefined && { iconUrl: data.iconUrl })}
                      onEdit={() => setEditing(commodity)}
                      onDelete={() => setDeleteTarget(commodity)}
                    />
                  );
                })}
              </div>
            )}
          </CardContent>
        </CollapsibleContent>
      </Collapsible>

      <NpcShopCommodityDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        mode="create"
        onSubmit={handleCreate}
      />

      <NpcShopCommodityDialog
        open={editing !== null}
        onOpenChange={open => {
          if (!open) setEditing(null);
        }}
        mode="edit"
        {...(editing && { initial: editing.attributes })}
        onSubmit={handleUpdate}
      />

      <AlertDialog
        open={deleteTarget !== null}
        onOpenChange={open => {
          if (!open) setDeleteTarget(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete commodity?</AlertDialogTitle>
            <AlertDialogDescription>
              Removes item #{deleteTarget?.attributes.templateId} from this shop.
              This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={deleteAllOpen} onOpenChange={setDeleteAllOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete all commodities?</AlertDialogTitle>
            <AlertDialogDescription>
              Removes every commodity from this shop. This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDeleteAll}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Delete All
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <Dialog open={bulkOpen} onOpenChange={setBulkOpen}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>Bulk Update Shop</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <Textarea
              placeholder="Paste JSON data here..."
              value={bulkJson}
              onChange={e => setBulkJson(e.target.value)}
              className="min-h-[300px] font-mono"
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setBulkOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleBulkUpdate}>Update Shop</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  );
}
