import type { Asset } from "@/services/api/inventory.service";

interface Props {
  asset: Asset;
  itemName?: string | undefined;
  slotName?: string | undefined;
}

type Field = {
  label: string;
  key: keyof Asset["attributes"];
  format?: "date" | "number";
};

const ZERO_DATE = "0001-01-01T00:00:00Z";

const EQUIPABLE_FIELDS: Field[] = [
  { label: "Quantity", key: "quantity", format: "number" },
  { label: "STR", key: "strength", format: "number" },
  { label: "DEX", key: "dexterity", format: "number" },
  { label: "INT", key: "intelligence", format: "number" },
  { label: "LUK", key: "luck", format: "number" },
  { label: "HP", key: "hp", format: "number" },
  { label: "MP", key: "mp", format: "number" },
  { label: "WATK", key: "weaponAttack", format: "number" },
  { label: "MATK", key: "magicAttack", format: "number" },
  { label: "WDEF", key: "weaponDefense", format: "number" },
  { label: "MDEF", key: "magicDefense", format: "number" },
  { label: "ACC", key: "accuracy", format: "number" },
  { label: "AVOID", key: "avoidability", format: "number" },
  { label: "Hands", key: "hands", format: "number" },
  { label: "Speed", key: "speed", format: "number" },
  { label: "Jump", key: "jump", format: "number" },
  { label: "Slots", key: "slots", format: "number" },
  { label: "Lv Type", key: "levelType", format: "number" },
  { label: "Level", key: "level", format: "number" },
  { label: "Experience", key: "experience", format: "number" },
  { label: "Hammers", key: "hammersApplied", format: "number" },
  { label: "Equipped Since", key: "equippedSince", format: "date" },
  { label: "Expires", key: "expiration", format: "date" },
];

const CONSUMABLE_FIELDS: Field[] = [
  { label: "Quantity", key: "quantity", format: "number" },
  { label: "Cash ID", key: "cashId" },
  { label: "Commodity", key: "commodityId", format: "number" },
  { label: "Pet ID", key: "petId", format: "number" },
  { label: "Expires", key: "expiration", format: "date" },
];

function shouldRender(value: unknown, format?: Field["format"]): boolean {
  if (format === "date") return typeof value === "string" && value !== "" && value !== ZERO_DATE;
  if (typeof value === "number") return value !== 0;
  if (typeof value === "string") return value !== "";
  return value != null;
}

function formatValue(value: unknown, format?: Field["format"]): string {
  if (format === "date" && typeof value === "string") return new Date(value).toLocaleDateString();
  if (typeof value === "number") return value > 9999 ? new Intl.NumberFormat().format(value) : String(value);
  return String(value);
}

export function AssetTooltipContent({ asset, itemName, slotName }: Props) {
  const fields = asset.attributes.slot < 0 ? EQUIPABLE_FIELDS : CONSUMABLE_FIELDS;
  return (
    <div className="space-y-2 max-w-xs">
      <div className="font-semibold">
        {itemName ?? `Item #${asset.attributes.templateId}`}
        {slotName && <span className="ml-1 text-muted-foreground text-xs">({slotName})</span>}
      </div>
      <div className="grid grid-cols-2 gap-x-3 gap-y-1 text-xs">
        {fields
          .filter((f) => shouldRender(asset.attributes[f.key], f.format))
          .map((f) => (
            <div key={f.key as string} className="flex justify-between">
              <span className="text-muted-foreground">{f.label}</span>
              <span>{formatValue(asset.attributes[f.key], f.format)}</span>
            </div>
          ))}
      </div>
      <div className="text-xs text-muted-foreground border-t pt-1">
        Asset ID: {asset.id} | Slot: {asset.attributes.slot}
      </div>
    </div>
  );
}
