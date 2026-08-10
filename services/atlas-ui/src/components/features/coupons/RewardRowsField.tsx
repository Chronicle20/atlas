/**
 * The reward field array shared by the create-coupon and generate-batch
 * dialogs. Presentational and fully controlled: it owns no state, so the two
 * dialogs can drive it from their own react-hook-form `Controller` without
 * either form's type leaking in here.
 *
 * Switching a row's `type` swaps which inputs are shown; the values for the
 * other branch are kept in the row object (so toggling back does not lose
 * typing) and are discarded by `rewardRowSchema`'s transform, which only reads
 * the fields belonging to the selected branch.
 */

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Plus, Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";
import {
  emptyRewardRow,
  type RewardRowInput,
  type RewardType,
} from "@/lib/schemas/coupons.schema";

export interface RewardRowsFieldProps {
  rows: RewardRowInput[];
  onChange: (rows: RewardRowInput[]) => void;
  /** Per-row messages, indexed alongside `rows`. */
  rowErrors?: (string | undefined)[];
  /** Message for the array itself (e.g. "at least one reward"). */
  error?: string | undefined;
  /** Distinguishes input ids when both dialogs are mounted at once. */
  idPrefix: string;
}

export function RewardRowsField({
  rows,
  onChange,
  rowErrors,
  error,
  idPrefix,
}: RewardRowsFieldProps) {
  const patch = (index: number, changes: Partial<RewardRowInput>) => {
    onChange(
      rows.map((row, i) => (i === index ? { ...row, ...changes } : row)),
    );
  };

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <Label>Rewards</Label>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => onChange([...rows, emptyRewardRow()])}
        >
          <Plus className="mr-1 h-3 w-3" />
          Add reward
        </Button>
      </div>

      {rows.map((row, index) => {
        const rowId = `${idPrefix}-reward-${index}`;
        return (
          <div key={rowId} className="rounded-md border p-3 space-y-3">
            <div className="flex items-center justify-between gap-2">
              <div className="flex gap-1" role="group" aria-label="Reward type">
                {(
                  [
                    ["CURRENCY", "Currency"],
                    ["CASH_ITEM", "Cash Item"],
                  ] as [RewardType, string][]
                ).map(([value, label]) => (
                  <Button
                    key={value}
                    type="button"
                    size="sm"
                    variant={row.type === value ? "default" : "outline"}
                    aria-pressed={row.type === value}
                    onClick={() => patch(index, { type: value })}
                  >
                    {label}
                  </Button>
                ))}
              </div>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                aria-label={`Remove reward ${index + 1}`}
                disabled={rows.length <= 1}
                onClick={() => onChange(rows.filter((_, i) => i !== index))}
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>

            {row.type === "CURRENCY" ? (
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <Label htmlFor={`${rowId}-currency`}>Currency</Label>
                  <Input
                    id={`${rowId}-currency`}
                    type="number"
                    value={row.currency}
                    onChange={(e) => patch(index, { currency: e.target.value })}
                  />
                  <p className="text-xs text-muted-foreground">
                    1 = NX, 2 = Maple Points, 3 = Prepaid
                  </p>
                </div>
                <div className="space-y-1">
                  <Label htmlFor={`${rowId}-amount`}>Amount</Label>
                  <Input
                    id={`${rowId}-amount`}
                    type="number"
                    value={row.amount}
                    onChange={(e) => patch(index, { amount: e.target.value })}
                  />
                </div>
              </div>
            ) : (
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <Label htmlFor={`${rowId}-serial`}>Serial number</Label>
                  <Input
                    id={`${rowId}-serial`}
                    type="number"
                    value={row.serialNumber}
                    onChange={(e) =>
                      patch(index, { serialNumber: e.target.value })
                    }
                  />
                </div>
                <div className="space-y-1">
                  <Label htmlFor={`${rowId}-quantity`}>Quantity</Label>
                  <Input
                    id={`${rowId}-quantity`}
                    type="number"
                    value={row.quantity}
                    onChange={(e) => patch(index, { quantity: e.target.value })}
                  />
                </div>
              </div>
            )}

            {rowErrors?.[index] && (
              <p className={cn("text-sm", "text-destructive")}>
                {rowErrors[index]}
              </p>
            )}
          </div>
        );
      })}

      {error && <p className="text-sm text-destructive">{error}</p>}
    </div>
  );
}
