/**
 * Create one coupon.
 *
 * `code` is optional: a blank field means "server, generate one", and the
 * attribute is then omitted from the POST body entirely rather than sent as
 * "" (which the backend would take as a literal, empty, code).
 */

import { Controller, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { RewardRowsField } from "@/components/features/coupons/RewardRowsField";
import { splitRewardErrors } from "@/components/features/coupons/reward-errors";
import { useCreateCoupon } from "@/lib/hooks/api/useCoupons";
import {
  couponFormSchema,
  emptyCouponForm,
  type CouponFormInput,
  type CouponFormOutput,
} from "@/lib/schemas/coupons.schema";
import { localInputToIso } from "@/lib/utils/coupon-dates";
import type { CreateCouponInput } from "@/services/api/coupons.service";
import { createErrorFromUnknown } from "@/types/api/errors";

interface CreateCouponDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function CreateCouponDialog({
  open,
  onOpenChange,
}: CreateCouponDialogProps) {
  const createCoupon = useCreateCoupon();
  const {
    control,
    register,
    handleSubmit,
    reset,
    watch,
    setValue,
    formState: { errors },
  } = useForm<CouponFormInput, unknown, CouponFormOutput>({
    resolver: zodResolver(couponFormSchema),
    defaultValues: emptyCouponForm(),
  });

  const rows = watch("rewards");
  const { rowErrors, arrayError } = splitRewardErrors(
    errors.rewards,
    rows.length,
  );

  const onSubmit = handleSubmit(async (values) => {
    const input: CreateCouponInput = {
      active: values.active,
      rewards: values.rewards,
    };
    // Every optional key is added only when the operator supplied it — an
    // omitted `code` is what makes the server generate one.
    if (values.code) input.code = values.code;
    if (values.description) input.description = values.description;
    if (values.startsAt) input.startsAt = localInputToIso(values.startsAt);
    if (values.expiresAt) input.expiresAt = localInputToIso(values.expiresAt);
    if (values.maxUses !== null) input.maxUses = values.maxUses;

    try {
      const created = await createCoupon.mutateAsync({ input });
      toast.success(`Coupon ${created.attributes.code} created`);
      reset(emptyCouponForm());
      onOpenChange(false);
    } catch (e) {
      toast.error(createErrorFromUnknown(e).message);
    }
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>New Coupon</DialogTitle>
          <DialogDescription>
            Leave the code blank to have one generated.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={onSubmit} className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <Label htmlFor="coupon-code">Code (optional)</Label>
              <Input id="coupon-code" {...register("code")} />
              {errors.code && (
                <p className="text-sm text-destructive">
                  {errors.code.message}
                </p>
              )}
            </div>
            <div className="space-y-1">
              <Label htmlFor="coupon-max-uses">Max uses</Label>
              <Input
                id="coupon-max-uses"
                type="number"
                placeholder="Unlimited"
                {...register("maxUses")}
              />
              {errors.maxUses && (
                <p className="text-sm text-destructive">
                  {errors.maxUses.message}
                </p>
              )}
            </div>
          </div>

          <div className="space-y-1">
            <Label htmlFor="coupon-description">Description</Label>
            <Textarea id="coupon-description" {...register("description")} />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <Label htmlFor="coupon-starts-at">Starts at</Label>
              <Input
                id="coupon-starts-at"
                type="datetime-local"
                {...register("startsAt")}
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="coupon-expires-at">Expires at</Label>
              <Input
                id="coupon-expires-at"
                type="datetime-local"
                {...register("expiresAt")}
              />
            </div>
          </div>
          {errors.expiresAt && (
            <p className="text-sm text-destructive">
              {errors.expiresAt.message}
            </p>
          )}

          <div className="flex items-center gap-2">
            <Switch
              id="coupon-active"
              checked={watch("active")}
              onCheckedChange={(checked) =>
                setValue("active", checked, { shouldDirty: true })
              }
            />
            <Label htmlFor="coupon-active">Active</Label>
          </div>

          <Controller
            control={control}
            name="rewards"
            render={({ field }) => (
              <RewardRowsField
                idPrefix="coupon"
                rows={field.value}
                onChange={field.onChange}
                rowErrors={rowErrors}
                error={arrayError}
              />
            )}
          />

          <DialogFooter>
            <Button type="submit" disabled={createCoupon.isPending}>
              Create
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
