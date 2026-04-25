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

const SLOT_LAYOUT: SlotLayoutEntry[] = [
  { slotId: -49, row: 1, col: 1, name: "Medal" },
  { slotId: -1, row: 1, col: 3, name: "Hat" },
  { slotId: -3, row: 1, col: 4, name: "Eye Accessory" },
  { slotId: -2, row: 2, col: 3, name: "Face Accessory" },
  { slotId: -4, row: 2, col: 5, name: "Earrings" },
  { slotId: -25, row: 3, col: 1, name: "Pet Equip" },
  { slotId: -10, row: 3, col: 2, name: "Weapon" },
  { slotId: -5, row: 3, col: 3, name: "Top / Overall" },
  { slotId: -17, row: 3, col: 4, name: "Shield" },
  { slotId: -16, row: 3, col: 5, name: "Cape" },
  { slotId: -112, row: 4, col: 1, name: "Belt" },
  { slotId: -8, row: 4, col: 2, name: "Gloves" },
  { slotId: -6, row: 4, col: 3, name: "Bottom" },
  { slotId: -13, row: 4, col: 4, name: "Pendant" },
  { slotId: -11, row: 4, col: 5, name: "Ring 1" },
  { slotId: -21, row: 5, col: 1, name: "Mount" },
  { slotId: -22, row: 5, col: 2, name: "Saddle" },
  { slotId: -7, row: 5, col: 3, name: "Shoes" },
  { slotId: -111, row: 5, col: 4, name: "Pendant 2" },
  { slotId: -12, row: 5, col: 5, name: "Ring 2" },
  { slotId: -15, row: 6, col: 5, name: "Ring 3 / 4" },
];

export function EquipmentPanel({ equipped, tenant }: Props) {
  const bySlot = new Map<number, Asset>();
  for (const a of equipped) bySlot.set(a.attributes.slot, a);

  return (
    <Card>
      <CardContent className="pt-4">
        <div
          className="grid gap-2"
          style={{
            gridTemplateColumns: "repeat(5, minmax(0, 1fr))",
            gridTemplateRows: "repeat(6, minmax(0, 1fr))",
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
