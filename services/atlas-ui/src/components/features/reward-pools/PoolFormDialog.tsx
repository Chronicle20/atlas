import { useState, useEffect } from "react";
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
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { ItemPicker } from "@/components/features/items/item-search/ItemPicker";
import { createErrorFromUnknown } from "@/types/api/errors";
import {
  gachaponPoolSchema,
  incubatorPoolSchema,
  cashSurprisePoolSchema,
  type GachaponPoolFormData,
  type IncubatorPoolFormData,
  type CashSurprisePoolFormData,
} from "@/lib/schemas/reward-pools.schema";
import {
  useCreateRewardPool,
  useUpdateRewardPool,
} from "@/lib/hooks/api/useRewardPools";
import { useSurpriseBoxTemplateIds } from "@/lib/hooks/api/useSurpriseBoxes";
import type {
  RewardPoolData,
  RewardPoolKind,
} from "@/types/models/reward-pool";

interface PoolFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  mode: "create" | "edit";
  pool?: RewardPoolData;
}

export function PoolFormDialog({
  open,
  onOpenChange,
  mode,
  pool,
}: PoolFormDialogProps) {
  const isEdit = mode === "edit";
  const [kind, setKind] = useState<RewardPoolKind>(
    pool?.attributes.kind ?? "gachapon",
  );
  // Reset the selected kind whenever the dialog transitions from closed to
  // open (adjust state during render per https://react.dev/learn/you-might-not-need-an-effect
  // instead of a useEffect, so this doesn't fire a synchronous setState from an effect).
  const [wasOpen, setWasOpen] = useState(open);
  if (open !== wasOpen) {
    setWasOpen(open);
    if (open) setKind(pool?.attributes.kind ?? "gachapon");
  }

  const createPool = useCreateRewardPool();
  const updatePool = useUpdateRewardPool();
  const pending = createPool.isPending || updatePool.isPending;

  const gachaponDefaults: GachaponPoolFormData =
    pool && pool.attributes.kind === "gachapon"
      ? {
          name: pool.attributes.name,
          npcIds: pool.attributes.npcIds,
          commonWeight: pool.attributes.commonWeight,
          uncommonWeight: pool.attributes.uncommonWeight,
          rareWeight: pool.attributes.rareWeight,
        }
      : {
          name: "",
          npcIds: [],
          commonWeight: 70,
          uncommonWeight: 25,
          rareWeight: 5,
        };

  // Create mode leaves eggItemId/successNpcId blank (keys omitted, so RHF
  // starts them as undefined); DefaultValues<T> makes that representable under
  // exactOptionalPropertyTypes, where the exact z.infer type would force 0s.
  const incubatorDefaults: DefaultValues<IncubatorPoolFormData> =
    pool && pool.attributes.kind === "incubator"
      ? {
          eggItemId: Number(pool.id),
          name: pool.attributes.name,
          successNpcId: pool.attributes.npcIds[0] ?? 0,
        }
      : { name: "" };

  const gachaponForm = useForm<GachaponPoolFormData>({
    resolver: zodResolver(gachaponPoolSchema),
    defaultValues: gachaponDefaults,
  });
  const incubatorForm = useForm<IncubatorPoolFormData>({
    resolver: zodResolver(incubatorPoolSchema),
    defaultValues: incubatorDefaults,
  });

  // Create mode leaves boxItemId blank, same rationale as incubatorDefaults.
  const cashSurpriseDefaults: DefaultValues<CashSurprisePoolFormData> =
    pool && pool.attributes.kind === "cash-surprise"
      ? { boxItemId: Number(pool.id), name: pool.attributes.name }
      : { name: "" };

  const cashSurpriseForm = useForm<CashSurprisePoolFormData>({
    resolver: zodResolver(cashSurprisePoolSchema),
    defaultValues: cashSurpriseDefaults,
  });

  // A pool whose id isn't a configured box is never rolled (atlas-cashshop
  // rejects the open with NOT_A_SURPRISE_BOX before it ever reaches the pool).
  // Warned about, not blocked: the operator may add the id to the tenant
  // configuration afterwards. Gated on isResolved so a still-loading (or
  // failed) configuration read can't accuse a perfectly good box.
  const surpriseBoxes = useSurpriseBoxTemplateIds();
  const boxItemId = cashSurpriseForm.watch("boxItemId");
  const unconfiguredBox =
    surpriseBoxes.isResolved &&
    !!boxItemId &&
    !surpriseBoxes.ids.includes(boxItemId);

  // npcIds is edited as a comma-separated string for gachapons
  const [npcIdsText, setNpcIdsText] = useState(
    (pool?.attributes.npcIds ?? []).join(", "),
  );
  if (open !== wasOpen && open) {
    setNpcIdsText((pool?.attributes.npcIds ?? []).join(", "));
  }
  // form.reset() is an imperative call into react-hook-form's external store
  // (not a React state setter), so it stays in an effect rather than the
  // render-time adjustment above.
  useEffect(() => {
    if (open) {
      gachaponForm.reset(gachaponDefaults);
      incubatorForm.reset(incubatorDefaults);
      cashSurpriseForm.reset(cashSurpriseDefaults);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, pool]);

  const submitGachapon = gachaponForm.handleSubmit(async (values) => {
    const npcIds = npcIdsText
      .split(",")
      .map((s) => Number(s.trim()))
      .filter((n) => Number.isInteger(n) && n > 0);
    const attributes = {
      name: values.name,
      kind: "gachapon" as const,
      npcIds,
      commonWeight: values.commonWeight,
      uncommonWeight: values.uncommonWeight,
      rareWeight: values.rareWeight,
    };
    try {
      if (isEdit) await updatePool.mutateAsync({ id: pool!.id, attributes });
      else await createPool.mutateAsync({ attributes });
      toast.success(isEdit ? "Pool updated" : "Pool created");
      onOpenChange(false);
    } catch (e) {
      toast.error(createErrorFromUnknown(e).message);
    }
  });

  const submitIncubator = incubatorForm.handleSubmit(async (values) => {
    const attributes = {
      name: values.name,
      kind: "incubator" as const,
      npcIds: [values.successNpcId],
      commonWeight: 0,
      uncommonWeight: 0,
      rareWeight: 0,
    };
    try {
      if (isEdit) await updatePool.mutateAsync({ id: pool!.id, attributes });
      else
        await createPool.mutateAsync({
          id: String(values.eggItemId),
          attributes,
        });
      toast.success(isEdit ? "Pool updated" : "Pool created");
      onOpenChange(false);
    } catch (e) {
      toast.error(createErrorFromUnknown(e).message);
    }
  });

  const submitCashSurprise = cashSurpriseForm.handleSubmit(async (values) => {
    const attributes = {
      name: values.name,
      kind: "cash-surprise" as const,
      npcIds: [],
      commonWeight: 0,
      uncommonWeight: 0,
      rareWeight: 0,
    };
    try {
      if (isEdit) await updatePool.mutateAsync({ id: pool!.id, attributes });
      else
        await createPool.mutateAsync({
          id: String(values.boxItemId),
          attributes,
        });
      toast.success(isEdit ? "Pool updated" : "Pool created");
      onOpenChange(false);
    } catch (e) {
      toast.error(createErrorFromUnknown(e).message);
    }
  });

  // Each kind renders its own form with its own useForm; select via an
  // exhaustive switch (rather than a ternary/if-chain) so a future fourth
  // kind that forgets a branch fails to compile — activeForm would be used
  // before being definitely assigned. Mirrors the KindBadge Record idiom,
  // adapted for JSX blocks that close over per-kind form objects.
  let activeForm: React.ReactElement;
  switch (kind) {
    case "gachapon":
      activeForm = (
        <form onSubmit={submitGachapon} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="pf-name">Name</Label>
            <Input id="pf-name" {...gachaponForm.register("name")} />
            {gachaponForm.formState.errors.name && (
              <p className="text-sm text-destructive">
                {gachaponForm.formState.errors.name.message}
              </p>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor="pf-npcs">NPC Ids (comma-separated)</Label>
            <Input
              id="pf-npcs"
              value={npcIdsText}
              onChange={(e) => setNpcIdsText(e.target.value)}
            />
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div className="space-y-2">
              <Label htmlFor="pf-cw">Common Weight</Label>
              <Input
                id="pf-cw"
                type="number"
                {...gachaponForm.register("commonWeight", {
                  valueAsNumber: true,
                })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="pf-uw">Uncommon Weight</Label>
              <Input
                id="pf-uw"
                type="number"
                {...gachaponForm.register("uncommonWeight", {
                  valueAsNumber: true,
                })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="pf-rw">Rare Weight</Label>
              <Input
                id="pf-rw"
                type="number"
                {...gachaponForm.register("rareWeight", {
                  valueAsNumber: true,
                })}
              />
            </div>
          </div>
          {gachaponForm.formState.errors.commonWeight && (
            <p className="text-sm text-destructive">
              {gachaponForm.formState.errors.commonWeight.message}
            </p>
          )}
          <DialogFooter>
            <Button type="submit" disabled={pending}>
              {isEdit ? "Save" : "Create"}
            </Button>
          </DialogFooter>
        </form>
      );
      break;
    case "incubator":
      activeForm = (
        <form onSubmit={submitIncubator} className="space-y-4">
          {!isEdit && (
            <div className="space-y-2">
              <Label htmlFor="pf-egg">Egg Item Id</Label>
              <Input
                id="pf-egg"
                type="number"
                {...incubatorForm.register("eggItemId", {
                  valueAsNumber: true,
                })}
              />
              {incubatorForm.formState.errors.eggItemId && (
                <p className="text-sm text-destructive">
                  {incubatorForm.formState.errors.eggItemId.message}
                </p>
              )}
              <p className="text-xs text-muted-foreground">
                The egg item id becomes the pool id (e.g. 4170001).
              </p>
            </div>
          )}
          <div className="space-y-2">
            <Label htmlFor="pf-iname">Name</Label>
            <Input id="pf-iname" {...incubatorForm.register("name")} />
            {incubatorForm.formState.errors.name && (
              <p className="text-sm text-destructive">
                {incubatorForm.formState.errors.name.message}
              </p>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor="pf-snpc">Success NPC Id</Label>
            <Input
              id="pf-snpc"
              type="number"
              {...incubatorForm.register("successNpcId", {
                valueAsNumber: true,
              })}
            />
            {incubatorForm.formState.errors.successNpcId && (
              <p className="text-sm text-destructive">
                {incubatorForm.formState.errors.successNpcId.message}
              </p>
            )}
          </div>
          <DialogFooter>
            <Button type="submit" disabled={pending}>
              {isEdit ? "Save" : "Create"}
            </Button>
          </DialogFooter>
        </form>
      );
      break;
    case "cash-surprise":
      activeForm = (
        <form onSubmit={submitCashSurprise} className="space-y-4">
          {!isEdit && (
            /* The picker trigger is a labelable <button>, so a <label
               for="pf-box"> would override its text as the accessible name.
               The field name is carried by a role="group"/aria-labelledby pair
               instead — same wiring as PoolItemDialog. */
            <div
              role="group"
              aria-labelledby="pf-box-label"
              className="space-y-2"
            >
              <Label
                id="pf-box-label"
                onClick={() => document.getElementById("pf-box")?.focus()}
              >
                Box Item
              </Label>
              <Controller
                control={cashSurpriseForm.control}
                name="boxItemId"
                render={({ field }) => (
                  <ItemPicker
                    id="pf-box"
                    poolKey="cashSurpriseBoxes"
                    placeholder="Select a box item…"
                    value={field.value ?? 0}
                    onChange={field.onChange}
                  />
                )}
              />
              {cashSurpriseForm.formState.errors.boxItemId && (
                <p className="text-sm text-destructive">
                  {cashSurpriseForm.formState.errors.boxItemId.message}
                </p>
              )}
              <p className="text-xs text-muted-foreground">
                The box item id becomes the pool id (e.g. 5222000).
              </p>
              {unconfiguredBox && (
                <p className="text-sm text-warning-foreground">
                  Item {boxItemId} is not one of this tenant&apos;s Surprise
                  boxes ({surpriseBoxes.ids.join(", ")}), so opening it will
                  never roll this pool. Add it to the tenant
                  configuration&apos;s cashShop.surprise.boxTemplateIds, or pick
                  a configured box.
                </p>
              )}
              <p className="text-sm text-muted-foreground">
                A pool that awards a Surprise box will produce an endless box.
                This is allowed; check your entries.
              </p>
            </div>
          )}
          <div className="space-y-2">
            <Label htmlFor="pf-csname">Name</Label>
            <Input id="pf-csname" {...cashSurpriseForm.register("name")} />
            {cashSurpriseForm.formState.errors.name && (
              <p className="text-sm text-destructive">
                {cashSurpriseForm.formState.errors.name.message}
              </p>
            )}
          </div>
          <DialogFooter>
            <Button type="submit" disabled={pending}>
              {isEdit ? "Save" : "Create"}
            </Button>
          </DialogFooter>
        </form>
      );
      break;
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit Pool" : "New Pool"}</DialogTitle>
        </DialogHeader>

        {!isEdit && (
          <RadioGroup
            value={kind}
            onValueChange={(v) => setKind(v as RewardPoolKind)}
            className="flex gap-6"
          >
            <div className="flex items-center gap-2">
              <RadioGroupItem value="gachapon" id="kind-gachapon" />
              <Label htmlFor="kind-gachapon">Gachapon</Label>
            </div>
            <div className="flex items-center gap-2">
              <RadioGroupItem value="incubator" id="kind-incubator" />
              <Label htmlFor="kind-incubator">Incubator</Label>
            </div>
            <div className="flex items-center gap-2">
              <RadioGroupItem value="cash-surprise" id="kind-cash-surprise" />
              <Label htmlFor="kind-cash-surprise">Cash Surprise</Label>
            </div>
          </RadioGroup>
        )}

        {activeForm}
      </DialogContent>
    </Dialog>
  );
}
