import { ItemSearchCombobox } from "./ItemSearchCombobox";
import { ItemRow } from "./ItemRow";
import type { EquipmentPoolKey } from "./previewLoadout";

interface EquipmentPoolSectionProps {
  poolKey: EquipmentPoolKey;
  title: string;
  ids: number[];
  onAdd: (id: number) => void;
  onRemove: (entryIndex: number) => void;
}

export function EquipmentPoolSection({
  poolKey,
  title,
  ids,
  onAdd,
  onRemove,
}: EquipmentPoolSectionProps) {
  return (
    <section className="space-y-2">
      <div className="flex items-center justify-between">
        <div className="flex items-baseline gap-2">
          <h3 className="text-sm font-semibold">{title}</h3>
          <span className="text-xs text-muted-foreground">
            {ids.length} options · player picks one
          </span>
        </div>
        <ItemSearchCombobox poolKey={poolKey} existingIds={ids} onAdd={onAdd} />
      </div>
      <div className="space-y-1">
        {ids.map((id, idx) => (
          <ItemRow
            key={`${id}-${idx}`}
            id={id}
            onRemove={() => onRemove(idx)}
            removeAriaLabel={`Remove ${id}`}
          />
        ))}
      </div>
    </section>
  );
}
