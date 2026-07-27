import { useEffect } from "react";
import { useForm, Controller, type DefaultValues } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { createErrorFromUnknown } from "@/types/api/errors";
import {
  tierItemSchema,
  weightItemSchema,
  type TierItemFormData,
  type WeightItemFormData,
} from "@/lib/schemas/reward-pools.schema";
import {
  useCreatePoolItem,
  useUpdatePoolItem,
  useCreateGlobalItem,
  useUpdateGlobalItem,
} from "@/lib/hooks/api/useRewardPools";
import type { RewardPoolItemData } from "@/types/models/reward-pool-item";
import type { GlobalRewardItemData } from "@/types/models/global-reward-item";

interface PoolItemDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  kind: "gachapon" | "incubator" | "global";
  poolId?: string;
  item?: RewardPoolItemData | GlobalRewardItemData;
}

export function PoolItemDialog({
  open,
  onOpenChange,
  kind,
  poolId,
  item,
}: PoolItemDialogProps) {
  const isEdit = !!item;
  const weighted = kind === "incubator";
  const schema = weighted ? weightItemSchema : tierItemSchema;

  // Create mode leaves the numeric fields blank (keys omitted, so RHF starts
  // them as undefined); DefaultValues<T> makes that representable under
  // exactOptionalPropertyTypes, where the exact z.infer type would force 0s.
  const defaultValues: DefaultValues<TierItemFormData | WeightItemFormData> =
    item
      ? {
          itemId: item.attributes.itemId,
          quantity: item.attributes.quantity,
          ...(weighted
            ? { weight: (item as RewardPoolItemData).attributes.weight }
            : {
                tier: (item.attributes.tier || "common") as
                  "common" | "uncommon" | "rare",
              }),
        }
      : weighted
        ? {}
        : { tier: "common" as const };

  const form = useForm<TierItemFormData | WeightItemFormData>({
    resolver: zodResolver(schema),
    defaultValues,
  });
  useEffect(() => {
    if (open) form.reset(defaultValues);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const createItem = useCreatePoolItem();
  const updateItem = useUpdatePoolItem();
  const createGlobal = useCreateGlobalItem();
  const updateGlobal = useUpdateGlobalItem();
  const pending =
    createItem.isPending ||
    updateItem.isPending ||
    createGlobal.isPending ||
    updateGlobal.isPending;

  const onSubmit = form.handleSubmit(async (values) => {
    try {
      if (kind === "global") {
        const attrs = {
          itemId: values.itemId,
          quantity: values.quantity,
          tier: (values as TierItemFormData).tier,
        };
        if (isEdit)
          await updateGlobal.mutateAsync({
            itemRecordId: item!.id,
            attributes: attrs,
          });
        else await createGlobal.mutateAsync({ attributes: attrs });
      } else {
        const attrs = weighted
          ? {
              itemId: values.itemId,
              quantity: values.quantity,
              tier: "common",
              weight: (values as WeightItemFormData).weight,
            }
          : {
              itemId: values.itemId,
              quantity: values.quantity,
              tier: (values as TierItemFormData).tier,
              weight: 0,
            };
        if (isEdit)
          await updateItem.mutateAsync({
            poolId: poolId!,
            itemRecordId: item!.id,
            attributes: attrs,
          });
        else
          await createItem.mutateAsync({ poolId: poolId!, attributes: attrs });
      }
      toast.success(isEdit ? "Item updated" : "Item added");
      onOpenChange(false);
    } catch (e) {
      toast.error(createErrorFromUnknown(e).message);
    }
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit Item" : "Add Item"}</DialogTitle>
        </DialogHeader>
        <form onSubmit={onSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="pi-itemId">Item Id</Label>
            <Input
              id="pi-itemId"
              type="number"
              {...form.register("itemId", { valueAsNumber: true })}
            />
            {form.formState.errors.itemId && (
              <p className="text-sm text-destructive">
                {form.formState.errors.itemId.message}
              </p>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor="pi-quantity">Quantity</Label>
            <Input
              id="pi-quantity"
              type="number"
              {...form.register("quantity", { valueAsNumber: true })}
            />
            {form.formState.errors.quantity && (
              <p className="text-sm text-destructive">
                {form.formState.errors.quantity.message}
              </p>
            )}
          </div>
          {weighted ? (
            <div className="space-y-2">
              <Label htmlFor="pi-weight">Weight</Label>
              <Input
                id="pi-weight"
                type="number"
                {...form.register("weight" as const, { valueAsNumber: true })}
              />
              {"weight" in form.formState.errors &&
                form.formState.errors.weight && (
                  <p className="text-sm text-destructive">
                    {form.formState.errors.weight.message}
                  </p>
                )}
            </div>
          ) : (
            <div className="space-y-2">
              <Label htmlFor="pi-tier">Tier</Label>
              <Controller
                control={form.control}
                name={"tier" as const}
                render={({ field }) => (
                  <Select
                    value={field.value as "common" | "uncommon" | "rare"}
                    onValueChange={field.onChange}
                  >
                    <SelectTrigger id="pi-tier" aria-label="Tier">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="common">common</SelectItem>
                      <SelectItem value="uncommon">uncommon</SelectItem>
                      <SelectItem value="rare">rare</SelectItem>
                    </SelectContent>
                  </Select>
                )}
              />
            </div>
          )}
          <DialogFooter>
            <Button type="submit" disabled={pending}>
              {isEdit ? "Save" : "Add"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
