import { useEffect, useState } from "react";
import type { CharacterPresetInventoryEntry } from "@/types/models/template";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ItemRow } from "../templates/ItemRow";
import { ItemSearchCombobox } from "../templates/ItemSearchCombobox";

interface InventorySectionProps {
  inventory: CharacterPresetInventoryEntry[];
  onAdd: (templateId: number) => void;
  onRemove: (index: number) => void;
  onSetQty: (index: number, value: number) => void;
}

/**
 * Uncontrolled-feeling quantity input: keeps a local text draft so mid-edit
 * keystrokes (e.g. clearing before typing a new value) aren't clobbered by
 * the parent's clamped-to-1 prop echo. Resyncs from `value` whenever it
 * changes out from under the input (preset switch, external clamp).
 */
function QuantityInput({
  value,
  onChange,
}: {
  value: number;
  onChange: (value: number) => void;
}) {
  const [draft, setDraft] = useState(String(value));

  useEffect(() => setDraft(String(value)), [value]);

  return (
    <Input
      type="number"
      min={1}
      aria-label="Quantity"
      className="w-20"
      value={draft}
      onChange={(e) => {
        setDraft(e.target.value);
        const parsed = Number(e.target.value);
        if (!Number.isNaN(parsed)) {
          onChange(Math.max(1, parsed));
        }
      }}
    />
  );
}

export function InventorySection({
  inventory,
  onAdd,
  onRemove,
  onSetQty,
}: InventorySectionProps) {
  const [manualId, setManualId] = useState("");

  const existingIds = inventory.map((e) => e.templateId);

  const handleManualAdd = () => {
    const parsed = Number(manualId);
    if (Number.isInteger(parsed) && parsed > 0) {
      onAdd(parsed);
      setManualId("");
    }
  };

  return (
    <section className="space-y-2">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold">Granted items</h3>
        <ItemSearchCombobox
          poolKey="items"
          existingIds={existingIds}
          onAdd={onAdd}
        />
      </div>

      <div className="flex items-center gap-2">
        <Input
          aria-label="Manual item id"
          value={manualId}
          onChange={(e) => setManualId(e.target.value)}
          placeholder="Enter item id…"
          className="w-40"
        />
        <Button
          type="button"
          variant="outline"
          size="sm"
          aria-label="Add item id"
          onClick={handleManualAdd}
        >
          Add
        </Button>
      </div>

      <div className="space-y-1">
        {inventory.length === 0 && (
          <p className="text-sm text-muted-foreground">No granted items.</p>
        )}
        {inventory.map((e, i) => (
          <ItemRow
            key={`${e.templateId}-${i}`}
            id={e.templateId}
            removeAriaLabel={`Remove item ${e.templateId}`}
            onRemove={() => onRemove(i)}
            trailing={
              <QuantityInput
                value={e.quantity}
                onChange={(v) => onSetQty(i, v)}
              />
            }
          />
        ))}
      </div>
    </section>
  );
}
