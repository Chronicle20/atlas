import { useState } from "react";
import type { CharacterPresetEquipmentEntry } from "@/types/models/template";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ItemRow } from "../templates/ItemRow";
import { ItemSearchCombobox } from "../templates/ItemSearchCombobox";
import type { SearchPoolKey } from "../templates/poolSearchConfig";

interface EquipmentSectionProps {
  equipment: CharacterPresetEquipmentEntry[];
  onAdd: (templateId: number) => void;
  onRemove: (index: number) => void;
  onSetAvg: (index: number, value: boolean) => void;
}

const SUBCATEGORY_OPTIONS: { value: SearchPoolKey; label: string }[] = [
  { value: "items", label: "All" },
  { value: "tops", label: "Tops" },
  { value: "bottoms", label: "Bottoms" },
  { value: "shoes", label: "Shoes" },
  { value: "weapons", label: "Weapons" },
];

export function EquipmentSection({
  equipment,
  onAdd,
  onRemove,
  onSetAvg,
}: EquipmentSectionProps) {
  const [poolKey, setPoolKey] = useState<SearchPoolKey>("items");

  const existingIds = equipment.map((e) => e.templateId);

  return (
    <section className="space-y-2">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold">Worn items</h3>
        <div className="flex items-center gap-2">
          <Select
            value={poolKey}
            onValueChange={(v) => setPoolKey(v as SearchPoolKey)}
          >
            <SelectTrigger className="w-32" aria-label="Item subcategory">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {SUBCATEGORY_OPTIONS.map((opt) => (
                <SelectItem key={opt.value} value={opt.value}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <ItemSearchCombobox
            poolKey={poolKey}
            existingIds={existingIds}
            onAdd={onAdd}
          />
        </div>
      </div>

      <div className="space-y-1">
        {equipment.length === 0 && (
          <p className="text-sm text-muted-foreground">No worn items.</p>
        )}
        {equipment.map((e, i) => (
          <ItemRow
            key={`${e.templateId}-${i}`}
            id={e.templateId}
            removeAriaLabel={`Remove equipment ${e.templateId}`}
            onRemove={() => onRemove(i)}
            trailing={
              <label className="flex items-center gap-1.5 text-xs text-muted-foreground">
                Avg stats
                <Switch
                  aria-label="Use average stats"
                  checked={e.useAverageStats}
                  onCheckedChange={(v) => onSetAvg(i, v)}
                />
              </label>
            }
          />
        ))}
      </div>
    </section>
  );
}
