/**
 * Bulk-generate a batch of coupon codes.
 *
 * The plaintext codes come back exactly once, on the POST response
 * (`attributes.codes`); a later GET of the same batch will not carry them.
 * The dialog therefore holds that response and offers a CSV built client-side
 * from it — there is no CSV endpoint to fall back on, so navigating away
 * before downloading loses the codes, which is why the panel says so.
 */

import { useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Download } from "lucide-react";
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
import { Textarea } from "@/components/ui/textarea";
import { ScrollArea } from "@/components/ui/scroll-area";
import { RewardRowsField } from "@/components/features/coupons/RewardRowsField";
import { splitRewardErrors } from "@/components/features/coupons/reward-errors";
import { useGenerateCouponBatch } from "@/lib/hooks/api/useCoupons";
import {
  couponBatchFormSchema,
  emptyCouponBatchForm,
  type CouponBatchFormInput,
  type CouponBatchFormOutput,
} from "@/lib/schemas/coupons.schema";
import { localInputToIso } from "@/lib/utils/coupon-dates";
import { downloadCodesCsv } from "@/lib/utils/coupons";
import type {
  CouponBatch,
  GenerateCouponBatchInput,
} from "@/services/api/coupons.service";
import { createErrorFromUnknown } from "@/types/api/errors";

interface GenerateCouponBatchDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function GenerateCouponBatchDialog({
  open,
  onOpenChange,
}: GenerateCouponBatchDialogProps) {
  const generateBatch = useGenerateCouponBatch();
  const [generated, setGenerated] = useState<CouponBatch | null>(null);

  const {
    control,
    register,
    handleSubmit,
    reset,
    watch,
    formState: { errors },
  } = useForm<CouponBatchFormInput, unknown, CouponBatchFormOutput>({
    resolver: zodResolver(couponBatchFormSchema),
    defaultValues: emptyCouponBatchForm(),
  });

  const rows = watch("rewards");
  const { rowErrors, arrayError } = splitRewardErrors(
    errors.rewards,
    rows.length,
  );

  const codes = generated?.attributes.codes ?? [];

  const handleOpenChange = (next: boolean) => {
    if (!next) {
      reset(emptyCouponBatchForm());
      setGenerated(null);
    }
    onOpenChange(next);
  };

  const onSubmit = handleSubmit(async (values) => {
    const input: GenerateCouponBatchInput = {
      count: values.count,
      rewards: values.rewards,
    };
    if (values.prefix) input.prefix = values.prefix;
    if (values.length !== null) input.length = values.length;
    if (values.description) input.description = values.description;
    if (values.startsAt) input.startsAt = localInputToIso(values.startsAt);
    if (values.expiresAt) input.expiresAt = localInputToIso(values.expiresAt);

    try {
      const batch = await generateBatch.mutateAsync({ input });
      setGenerated(batch);
      toast.success(`Generated ${batch.attributes.generatedCount} codes`);
    } catch (e) {
      toast.error(createErrorFromUnknown(e).message);
    }
  });

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Generate Coupon Batch</DialogTitle>
          <DialogDescription>
            Every code in the batch grants the same rewards.
          </DialogDescription>
        </DialogHeader>

        {generated ? (
          <div className="space-y-3">
            <p className="text-sm">
              {codes.length} codes generated. They are shown only once —
              download the CSV before closing this dialog.
            </p>
            <ScrollArea className="h-48 rounded-md border p-3">
              <ul className="space-y-1 font-mono text-sm">
                {codes.map((code) => (
                  <li key={code}>{code}</li>
                ))}
              </ul>
            </ScrollArea>
            <DialogFooter>
              <Button
                type="button"
                onClick={() =>
                  downloadCodesCsv(`coupon-batch-${generated.id}.csv`, codes)
                }
              >
                <Download className="mr-1 h-4 w-4" />
                Download CSV
              </Button>
              <Button
                type="button"
                variant="outline"
                onClick={() => handleOpenChange(false)}
              >
                Close
              </Button>
            </DialogFooter>
          </div>
        ) : (
          <form onSubmit={onSubmit} className="space-y-4">
            <div className="grid grid-cols-3 gap-3">
              <div className="space-y-1">
                <Label htmlFor="batch-count">Count</Label>
                <Input id="batch-count" type="number" {...register("count")} />
                {errors.count && (
                  <p className="text-sm text-destructive">
                    {errors.count.message}
                  </p>
                )}
              </div>
              <div className="space-y-1">
                <Label htmlFor="batch-prefix">Prefix</Label>
                <Input id="batch-prefix" {...register("prefix")} />
                {errors.prefix && (
                  <p className="text-sm text-destructive">
                    {errors.prefix.message}
                  </p>
                )}
              </div>
              <div className="space-y-1">
                <Label htmlFor="batch-length">Length</Label>
                <Input
                  id="batch-length"
                  type="number"
                  placeholder="Default"
                  {...register("length")}
                />
                {errors.length && (
                  <p className="text-sm text-destructive">
                    {errors.length.message}
                  </p>
                )}
              </div>
            </div>

            <div className="space-y-1">
              <Label htmlFor="batch-description">Description</Label>
              <Textarea id="batch-description" {...register("description")} />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <Label htmlFor="batch-starts-at">Starts at</Label>
                <Input
                  id="batch-starts-at"
                  type="datetime-local"
                  {...register("startsAt")}
                />
              </div>
              <div className="space-y-1">
                <Label htmlFor="batch-expires-at">Expires at</Label>
                <Input
                  id="batch-expires-at"
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

            <Controller
              control={control}
              name="rewards"
              render={({ field }) => (
                <RewardRowsField
                  idPrefix="batch"
                  rows={field.value}
                  onChange={field.onChange}
                  rowErrors={rowErrors}
                  error={arrayError}
                />
              )}
            />

            <DialogFooter>
              <Button type="submit" disabled={generateBatch.isPending}>
                Generate
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}
