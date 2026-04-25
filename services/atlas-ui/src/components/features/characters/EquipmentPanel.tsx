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

// Layout mirrors the v83 in-game Equipment Inventory window:
// Medal sits top-left, the body silhouette runs down the center, the left
// column carries Pendant / Pendant2 / Pet Equip, and the right column is
// the ring/medal stack. Cash equipment (slots <=-100, except -111/-112) is
// out of scope for v1.
const SLOT_LAYOUT: SlotLayoutEntry[] = [
  // Row 1 — head + medal
  { slotId: -49, row: 1, col: 1, name: "Medal" },
  { slotId: -1, row: 1, col: 3, name: "Hat" },
  // Row 2 — face/eye/earrings
  { slotId: -2, row: 2, col: 3, name: "Face Acc" },
  { slotId: -3, row: 2, col: 4, name: "Eye Acc" },
  { slotId: -4, row: 2, col: 5, name: "Earrings" },
  // Row 3 — torso (pendant left of weapon, then body / cape / shield)
  { slotId: -13, row: 3, col: 1, name: "Pendant" },
  { slotId: -10, row: 3, col: 2, name: "Weapon" },
  { slotId: -5, row: 3, col: 3, name: "Top" },
  { slotId: -16, row: 3, col: 4, name: "Cape" },
  { slotId: -17, row: 3, col: 5, name: "Shield" },
  // Row 4 — waist (second pendant under the first, gloves / bottom / belt / ring1)
  { slotId: -111, row: 4, col: 1, name: "Pendant 2" },
  { slotId: -8, row: 4, col: 2, name: "Gloves" },
  { slotId: -6, row: 4, col: 3, name: "Bottom" },
  { slotId: -112, row: 4, col: 4, name: "Belt" },
  { slotId: -11, row: 4, col: 5, name: "Ring 1" },
  // Row 5 — feet (mount / shoes / saddle / ring2)
  { slotId: -21, row: 5, col: 2, name: "Mount" },
  { slotId: -7, row: 5, col: 3, name: "Shoes" },
  { slotId: -22, row: 5, col: 4, name: "Saddle" },
  { slotId: -12, row: 5, col: 5, name: "Ring 2" },
  // Row 6 — pet equip + spare ring
  { slotId: -25, row: 6, col: 1, name: "Pet Equip" },
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
            // Fixed-size doll so it doesn't stretch with its parent. 60×60
            // cells comfortably fit the asset icons + an empty-slot watermark.
            gridTemplateColumns: "repeat(5, 60px)",
            gridTemplateRows: "repeat(6, 60px)",
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
