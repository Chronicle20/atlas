import { Card, CardContent } from "@/components/ui/card";
import type { Asset } from "@/services/api/inventory.service";
import type { Tenant } from "@/services/api/tenants.service";
import { EquipmentCell } from "./EquipmentCell";

interface Props {
  equipped: Asset[];
  tenant: Tenant;
}

interface SlotLayoutEntry {
  slotId: number;
  row: number;
  col: number;
  name: string;
}

// Layout mirrors the v83 in-game Equipment Inventory window: a 5×6 grid
// with the body's silhouette down the center column. Slot numbering matches
// the negative `slot` values reported by atlas-inventory.
const SLOT_LAYOUT: SlotLayoutEntry[] = [
  // Row 1 — head
  { slotId: -1, row: 1, col: 3, name: "Hat" },
  // Row 2 — face / accessories
  { slotId: -2, row: 2, col: 3, name: "Face Acc" },
  { slotId: -3, row: 2, col: 4, name: "Eye Acc" },
  { slotId: -4, row: 2, col: 5, name: "Earrings" },
  // Row 3 — torso
  { slotId: -111, row: 3, col: 1, name: "Pendant 2" },
  { slotId: -10, row: 3, col: 2, name: "Weapon" },
  { slotId: -5, row: 3, col: 3, name: "Top" },
  { slotId: -16, row: 3, col: 4, name: "Cape" },
  { slotId: -17, row: 3, col: 5, name: "Shield" },
  // Row 4 — waist
  { slotId: -13, row: 4, col: 1, name: "Pendant" },
  { slotId: -8, row: 4, col: 2, name: "Gloves" },
  { slotId: -6, row: 4, col: 3, name: "Bottom" },
  { slotId: -112, row: 4, col: 4, name: "Belt" },
  { slotId: -11, row: 4, col: 5, name: "Ring 1" },
  // Row 5 — feet
  { slotId: -49, row: 5, col: 1, name: "Medal" },
  { slotId: -21, row: 5, col: 2, name: "Mount" },
  { slotId: -7, row: 5, col: 3, name: "Shoes" },
  { slotId: -22, row: 5, col: 4, name: "Saddle" },
  { slotId: -12, row: 5, col: 5, name: "Ring 2" },
  // Row 6 — pet + spare ring
  { slotId: -25, row: 6, col: 2, name: "Pet Equip" },
  { slotId: -15, row: 6, col: 5, name: "Ring 3/4" },
];

export function EquipmentPanel({ equipped, tenant }: Props) {
  const bySlot = new Map<number, Asset>();
  for (const a of equipped) bySlot.set(a.attributes.slot, a);

  return (
    <Card className="w-fit">
      <CardContent className="pt-4">
        <div
          className="grid gap-1"
          style={{
            // Lock the doll to a fixed compact size so it doesn't stretch to
            // fill its parent column. ~44px cells leaves room for the icons
            // without dominating the page layout.
            gridTemplateColumns: "repeat(5, 44px)",
            gridTemplateRows: "repeat(6, 44px)",
          }}
        >
          {SLOT_LAYOUT.map((entry) => {
            const asset = bySlot.get(entry.slotId);
            return (
              <div key={entry.slotId} style={{ gridRow: entry.row, gridColumn: entry.col }}>
                <EquipmentCell
                  slotId={entry.slotId}
                  slotName={entry.name}
                  asset={asset}
                  tenant={tenant}
                />
              </div>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
}
