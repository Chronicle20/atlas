import type { EquipmentEntry, InventoryEntry } from "@/types/models/template";
import { EquipmentSection } from "../presets/EquipmentSection";
import { InventorySection } from "../presets/InventorySection";

interface StartingKitSectionProps {
  equipment: EquipmentEntry[];
  inventory: InventoryEntry[];
  onAddEquipment: (templateId: number) => void;
  onRemoveEquipment: (index: number) => void;
  onSetEquipmentAvg: (index: number, value: boolean) => void;
  onAddInventory: (templateId: number) => void;
  onRemoveInventory: (index: number) => void;
  onSetInventoryQty: (index: number, value: number) => void;
}

/** Worn equipment and granted inventory for the selected class row (FR-9),
 * composed unchanged from the presets kit sections. */
export function StartingKitSection({
  equipment,
  inventory,
  onAddEquipment,
  onRemoveEquipment,
  onSetEquipmentAvg,
  onAddInventory,
  onRemoveInventory,
  onSetInventoryQty,
}: StartingKitSectionProps) {
  return (
    <section className="space-y-4">
      <h3 className="text-sm font-semibold">Starting kit</h3>
      <EquipmentSection
        equipment={equipment}
        onAdd={onAddEquipment}
        onRemove={onRemoveEquipment}
        onSetAvg={onSetEquipmentAvg}
      />
      <InventorySection
        inventory={inventory}
        onAdd={onAddInventory}
        onRemove={onRemoveInventory}
        onSetQty={onSetInventoryQty}
      />
    </section>
  );
}
