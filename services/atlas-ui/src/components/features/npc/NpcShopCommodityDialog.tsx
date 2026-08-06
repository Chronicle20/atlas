import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ItemPicker } from "@/components/features/items/item-search/ItemPicker";
import { useItemName } from "@/lib/hooks/api/useItemStrings";
import type { CommodityAttributes } from "@/types/models/npc";

const EMPTY: CommodityAttributes = {
  templateId: 0,
  mesoPrice: 0,
  discountRate: 0,
  tokenTemplateId: 0,
  tokenPrice: 0,
  period: 0,
  levelLimit: 0,
};

interface NpcShopCommodityDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  mode: "create" | "edit";
  initial?: CommodityAttributes;
  onSubmit: (attrs: CommodityAttributes) => Promise<void> | void;
}

// Field order is preserved from the all-numeric version; `kind` decides which
// control renders, not where the row sits. `key` is narrowed to the two
// required-number id fields for "item" rows (rather than the full
// `keyof CommodityAttributes`) so `form[key]` types as `number`, not
// `number | undefined` — CommodityAttributes' optional `unitPrice`/`slotMax`
// would otherwise widen the index type and fail ItemPicker's `value: number`.
type ItemFieldKey = "templateId" | "tokenTemplateId";
type NumberFieldKey =
  "mesoPrice" | "discountRate" | "tokenPrice" | "period" | "levelLimit";
const FIELDS: Array<
  | { key: ItemFieldKey; label: string; kind: "item" }
  | { key: NumberFieldKey; label: string; kind: "number" }
> = [
  { key: "templateId", label: "Template ID", kind: "item" },
  { key: "mesoPrice", label: "Meso Price", kind: "number" },
  { key: "discountRate", label: "Discount Rate", kind: "number" },
  { key: "tokenTemplateId", label: "Token Template ID", kind: "item" },
  { key: "tokenPrice", label: "Token Price", kind: "number" },
  { key: "period", label: "Period", kind: "number" },
  { key: "levelLimit", label: "Level Limit", kind: "number" },
];

/** Read-only rendering of an item id, used for the non-editable edit-mode
 *  Template ID row. Falls back to the raw id while loading or on failure. */
function ResolvedItemName({ value, id }: { value: number; id: string }) {
  const current = useItemName(value > 0 ? String(value) : "");
  return (
    <span id={id} className="block truncate text-sm">
      {current.data ? `${current.data} · ${value}` : `Item ${value}`}
    </span>
  );
}

export function NpcShopCommodityDialog({
  open,
  onOpenChange,
  mode,
  initial,
  onSubmit,
}: NpcShopCommodityDialogProps) {
  const [form, setForm] = useState<CommodityAttributes>(initial ?? EMPTY);
  const [submitting, setSubmitting] = useState(false);

  // Reset the form when the dialog opens (or the target commodity changes
  // while open). Adjusted during render instead of in an effect.
  const [prevSync, setPrevSync] = useState({ open, initial });
  if (open !== prevSync.open || initial !== prevSync.initial) {
    setPrevSync({ open, initial });
    if (open) setForm(initial ?? EMPTY);
  }

  const handleSubmit = async () => {
    setSubmitting(true);
    try {
      await onSubmit(form);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>
            {mode === "create" ? "Add Commodity" : "Edit Commodity"}
          </DialogTitle>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          {FIELDS.map(({ key, label, kind }) => {
            const controlId = `commodity-${key}`;
            // The ItemPicker's trigger is a labelable <button>: a <label for>
            // pointing at it would override its own text content ("Select an
            // item…" / the resolved item name) as its accessible name per
            // HTML-AAM. Only associate the Label with plain form controls
            // (number inputs, and the read-only span in edit mode) where
            // that override is the desired behavior.
            const isPickerTrigger =
              kind === "item" && !(mode === "edit" && key === "templateId");
            return (
              <div key={key} className="grid grid-cols-4 items-center gap-4">
                <Label
                  className="text-right"
                  {...(isPickerTrigger ? {} : { htmlFor: controlId })}
                >
                  {label}
                </Label>
                {kind === "number" ? (
                  <Input
                    id={controlId}
                    name={key}
                    type="number"
                    value={form[key]}
                    onChange={(e) =>
                      setForm((prev) => ({
                        ...prev,
                        [key]: Number(e.target.value),
                      }))
                    }
                    className="col-span-3"
                  />
                ) : (
                  <div className="col-span-3">
                    {mode === "edit" && key === "templateId" ? (
                      <ResolvedItemName
                        value={form.templateId}
                        id={controlId}
                      />
                    ) : (
                      <ItemPicker
                        id={controlId}
                        value={form[key]}
                        onChange={(next) =>
                          setForm((prev) => ({ ...prev, [key]: next }))
                        }
                        {...(key === "tokenTemplateId"
                          ? { allowClear: true, placeholder: "None" }
                          : {})}
                      />
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={submitting}
          >
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={submitting}>
            {mode === "create" ? "Create" : "Update"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
