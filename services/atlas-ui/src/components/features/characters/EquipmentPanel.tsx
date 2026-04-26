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

// Slot positions match the layout the user pinned from the v83 in-game
// Equipment Inventory window. Mount / Saddle / Pet Equip / Pendant 2 are
// intentionally omitted for now — they'll be added in a follow-up.
//
// Ring 4 has no standard v83 slot id (the 4-ring system was added in a
// later client version); the cell renders as a placeholder so the grid
// stays symmetric, and we'll wire it up if/when the backend supports it.
const RING4_PLACEHOLDER_SLOT_ID = -9999;

const SLOT_LAYOUT: SlotLayoutEntry[] = [
  // Row 1
  { slotId: -1, row: 1, col: 2, name: "Hat" },
  // Row 2
  { slotId: -49, row: 2, col: 1, name: "Medal" },
  { slotId: -2, row: 2, col: 2, name: "Forehead" },
  { slotId: -11, row: 2, col: 4, name: "Ring 1" },
  { slotId: -12, row: 2, col: 5, name: "Ring 2" },
  // Row 3
  { slotId: -3, row: 3, col: 3, name: "Eye" },
  { slotId: -4, row: 3, col: 4, name: "Earring" },
  // Row 4 — body row
  { slotId: -16, row: 4, col: 1, name: "Cape" },
  { slotId: -5, row: 4, col: 2, name: "Top" },
  { slotId: -13, row: 4, col: 3, name: "Pendant" },
  { slotId: -10, row: 4, col: 4, name: "Weapon" },
  { slotId: -17, row: 4, col: 5, name: "Shield" },
  // Row 5
  { slotId: -8, row: 5, col: 1, name: "Gloves" },
  { slotId: -6, row: 5, col: 2, name: "Pants" },
  { slotId: -112, row: 5, col: 3, name: "Belt" },
  { slotId: -15, row: 5, col: 4, name: "Ring 3" },
  { slotId: RING4_PLACEHOLDER_SLOT_ID, row: 5, col: 5, name: "Ring 4" },
  // Row 6
  { slotId: -7, row: 6, col: 3, name: "Shoes" },
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
