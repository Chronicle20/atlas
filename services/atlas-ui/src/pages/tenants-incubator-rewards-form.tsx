import { Fragment, useState } from "react";
import { useParams } from "react-router-dom";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { useTenant } from "@/context/tenant-context";
import {
  useIncubatorRewards,
  useCreateIncubatorReward,
  useUpdateIncubatorReward,
  useDeleteIncubatorReward,
  useSeedIncubatorRewards,
} from "@/lib/hooks/api/useIncubatorRewards";
import type { IncubatorReward } from "@/services/api/incubator-rewards.service";
import { incubatorRewardSchema, type IncubatorRewardFormData } from "@/lib/schemas/incubator-rewards.schema";
import { createErrorFromUnknown } from "@/types/api/errors";
import { ItemNameCell } from "@/components/item-name-cell";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
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
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

/**
 * Region label per egg item id (interim — 4170008 is omitted, region unknown).
 */
const REGION_LABELS: Record<number, string> = {
  4170000: "Henesys",
  4170001: "Ellinia",
  4170002: "Perion",
  4170003: "Kerning City",
  4170004: "El Nath",
  4170005: "Ludibrium",
  4170006: "Orbis",
  4170007: "Aqua Road",
  4170009: "Nautilus",
};

const EGG_OPTIONS = Object.entries(REGION_LABELS).map(([eggId, label]) => ({
  value: eggId,
  label: `${label} (${eggId})`,
}));

function regionLabel(eggId: number): string {
  return REGION_LABELS[eggId] ?? `Egg ${eggId}`;
}

const EMPTY_DEFAULTS: IncubatorRewardFormData = {
  eggId: 4170000,
  itemId: 0,
  quantity: 1,
  weight: 1,
};

const FIELDS: { name: keyof IncubatorRewardFormData; label: string }[] = [
  { name: "itemId", label: "Item ID" },
  { name: "quantity", label: "Quantity" },
  { name: "weight", label: "Weight" },
];

export function IncubatorRewardsForm() {
  const { id: tenantId = "" } = useParams();
  const { activeTenant } = useTenant();

  const rewardsQuery = useIncubatorRewards(tenantId);
  const rewards = rewardsQuery.data ?? [];
  const loading = rewardsQuery.isLoading;

  const createMut = useCreateIncubatorReward();
  const updateMut = useUpdateIncubatorReward();
  const deleteMut = useDeleteIncubatorReward();
  const seedMut = useSeedIncubatorRewards();

  // Reward pools are per-egg (region): group rows by eggId and compute each
  // group's chance % against its own group weight total, not the grand total.
  const groupedByEgg = new Map<number, IncubatorReward[]>();
  for (const r of rewards) {
    const group = groupedByEgg.get(r.attributes.eggId);
    if (group) {
      group.push(r);
    } else {
      groupedByEgg.set(r.attributes.eggId, [r]);
    }
  }
  const sortedEggIds = Array.from(groupedByEgg.keys()).sort((a, b) => a - b);

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<IncubatorReward | null>(null);

  const [deleteTarget, setDeleteTarget] = useState<IncubatorReward | null>(null);
  const [seedDialogOpen, setSeedDialogOpen] = useState(false);

  const form = useForm<IncubatorRewardFormData>({
    resolver: zodResolver(incubatorRewardSchema),
    defaultValues: EMPTY_DEFAULTS,
  });

  const openAdd = () => {
    setEditing(null);
    form.reset(EMPTY_DEFAULTS);
    setDialogOpen(true);
  };

  const openEdit = (reward: IncubatorReward) => {
    setEditing(reward);
    form.reset(reward.attributes);
    setDialogOpen(true);
  };

  const handleDialogOpenChange = (open: boolean) => {
    setDialogOpen(open);
    if (!open) {
      setEditing(null);
      form.reset(EMPTY_DEFAULTS);
    }
  };

  const onSubmit = (data: IncubatorRewardFormData) => {
    if (editing) {
      updateMut.mutate(
        { tenantId, id: editing.id, attributes: data },
        {
          onSuccess: () => {
            toast.success("Incubator reward updated.");
            handleDialogOpenChange(false);
          },
          onError: (error: unknown) =>
            toast.error(createErrorFromUnknown(error, "Failed to update incubator reward").message),
        },
      );
    } else {
      createMut.mutate(
        { tenantId, attributes: data },
        {
          onSuccess: () => {
            toast.success("Incubator reward created.");
            handleDialogOpenChange(false);
          },
          onError: (error: unknown) =>
            toast.error(createErrorFromUnknown(error, "Failed to create incubator reward").message),
        },
      );
    }
  };

  const handleDelete = () => {
    if (!deleteTarget) return;
    deleteMut.mutate(
      { tenantId, id: deleteTarget.id },
      {
        onSuccess: () => {
          toast.success("Incubator reward deleted.");
          setDeleteTarget(null);
        },
        onError: (error: unknown) =>
          toast.error(createErrorFromUnknown(error, "Failed to delete incubator reward").message),
      },
    );
  };

  const handleSeed = () => {
    seedMut.mutate(
      { tenantId },
      {
        onSuccess: () => {
          toast.success("Incubator rewards seeded.");
          setSeedDialogOpen(false);
        },
        onError: (error: unknown) =>
          toast.error(createErrorFromUnknown(error, "Failed to seed incubator rewards").message),
      },
    );
  };

  if (loading) {
    return <div className="flex justify-center items-center p-8">Loading incubator rewards...</div>;
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-row justify-end gap-2">
        <Button type="button" variant="outline" onClick={() => setSeedDialogOpen(true)}>
          Seed defaults
        </Button>
        <Button type="button" onClick={openAdd}>
          Add
        </Button>
      </div>

      {rewards.length === 0 ? (
        <div className="flex justify-center items-center p-8 text-muted-foreground">
          No incubator rewards found for this tenant.
        </div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Item</TableHead>
              <TableHead>Quantity</TableHead>
              <TableHead>Weight</TableHead>
              <TableHead>Chance</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {sortedEggIds.map((eggId) => {
              const groupRewards = groupedByEgg.get(eggId) ?? [];
              const groupWeight = groupRewards.reduce((s, r) => s + r.attributes.weight, 0);
              return (
                <Fragment key={eggId}>
                  <TableRow className="bg-muted/50 hover:bg-muted/50">
                    <TableCell colSpan={5} className="font-medium">
                      {regionLabel(eggId)}
                    </TableCell>
                  </TableRow>
                  {groupRewards.map((r) => {
                    const chance =
                      groupWeight > 0 ? ((r.attributes.weight / groupWeight) * 100).toFixed(1) + "%" : "—";
                    return (
                      <TableRow key={r.id}>
                        <TableCell>
                          <ItemNameCell itemId={String(r.attributes.itemId)} tenant={activeTenant} />
                        </TableCell>
                        <TableCell>{r.attributes.quantity}</TableCell>
                        <TableCell>{r.attributes.weight}</TableCell>
                        <TableCell>{chance}</TableCell>
                        <TableCell className="text-right space-x-2">
                          <Button type="button" variant="outline" size="sm" onClick={() => openEdit(r)}>
                            Edit
                          </Button>
                          <Button type="button" variant="destructive" size="sm" onClick={() => setDeleteTarget(r)}>
                            Delete
                          </Button>
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </Fragment>
              );
            })}
          </TableBody>
        </Table>
      )}

      {/* Add/Edit Dialog */}
      <Dialog open={dialogOpen} onOpenChange={handleDialogOpenChange}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editing ? "Edit Incubator Reward" : "Add Incubator Reward"}</DialogTitle>
            <DialogDescription>
              Configure an item reward and its relative weight in the incubator reward pool.
            </DialogDescription>
          </DialogHeader>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
              <FormField
                control={form.control}
                name="eggId"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Region (Egg)</FormLabel>
                    <Select
                      onValueChange={(value) => field.onChange(Number(value))}
                      defaultValue={field.value.toString()}
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder="Select a region" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        {EGG_OPTIONS.map((opt) => (
                          <SelectItem key={opt.value} value={opt.value}>
                            {opt.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />
              {FIELDS.map((f) => (
                <FormField
                  key={f.name}
                  control={form.control}
                  name={f.name}
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{f.label}</FormLabel>
                      <FormControl>
                        <Input
                          type="number"
                          {...field}
                          onChange={(e) => field.onChange(e.target.valueAsNumber)}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              ))}
              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => handleDialogOpenChange(false)}>
                  Cancel
                </Button>
                <Button type="submit" disabled={createMut.isPending || updateMut.isPending}>
                  Save
                </Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>

      {/* Seed Confirmation */}
      <AlertDialog open={seedDialogOpen} onOpenChange={setSeedDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Seed default incubator rewards?</AlertDialogTitle>
            <AlertDialogDescription>
              This repopulates the reward pool from the built-in default set. Existing entries are
              not removed.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={handleSeed} disabled={seedMut.isPending}>
              Confirm
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Delete Confirmation */}
      <AlertDialog open={!!deleteTarget} onOpenChange={(open) => !open && setDeleteTarget(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete incubator reward?</AlertDialogTitle>
            <AlertDialogDescription>
              This action cannot be undone. This will permanently remove the reward entry.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              disabled={deleteMut.isPending}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Confirm
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
