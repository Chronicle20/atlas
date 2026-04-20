import { useEffect, useState } from "react";
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

const FIELDS: Array<{ key: keyof CommodityAttributes; label: string }> = [
  { key: "templateId", label: "Template ID" },
  { key: "mesoPrice", label: "Meso Price" },
  { key: "discountRate", label: "Discount Rate" },
  { key: "tokenTemplateId", label: "Token Template ID" },
  { key: "tokenPrice", label: "Token Price" },
  { key: "period", label: "Period" },
  { key: "levelLimit", label: "Level Limit" },
];

export function NpcShopCommodityDialog({
  open,
  onOpenChange,
  mode,
  initial,
  onSubmit,
}: NpcShopCommodityDialogProps) {
  const [form, setForm] = useState<CommodityAttributes>(initial ?? EMPTY);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (open) setForm(initial ?? EMPTY);
  }, [open, initial]);

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
          {FIELDS.map(({ key, label }) => {
            const disabled = mode === "edit" && key === "templateId";
            return (
              <div key={key} className="grid grid-cols-4 items-center gap-4">
                <Label htmlFor={`commodity-${key}`} className="text-right">
                  {label}
                </Label>
                <Input
                  id={`commodity-${key}`}
                  name={key}
                  type="number"
                  value={form[key]}
                  disabled={disabled}
                  onChange={e =>
                    setForm(prev => ({ ...prev, [key]: Number(e.target.value) }))
                  }
                  className="col-span-3"
                />
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
