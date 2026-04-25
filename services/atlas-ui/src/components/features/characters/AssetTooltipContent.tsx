import type { Asset } from "@/services/api/inventory.service";
import { getItemType } from "@/types/models/item";
import { useItemData } from "@/lib/hooks/useItemData";
import { useEquipmentData } from "@/lib/hooks/api/useEquipmentData";
import { cn } from "@/lib/utils";

interface Props {
  asset: Asset;
  itemName?: string | undefined;
  slotName?: string | undefined;
}

const ZERO_DATE = "0001-01-01T00:00:00Z";

// Order matches the in-game class label row.
const JOB_BITS: Array<{ name: string; bit: number }> = [
  { name: "BEGINNER", bit: 0 }, // bit 0 == "any" — see isJobActive
  { name: "WARRIOR", bit: 1 },
  { name: "MAGICIAN", bit: 2 },
  { name: "BOWMAN", bit: 4 },
  { name: "THIEF", bit: 8 },
  { name: "PIRATE", bit: 16 },
];

// In v83, reqJob === 0 means "Beginner-tier — any class can equip" (the
// item shows all six job badges highlighted). Non-zero values are bit
// flags for the five advanced classes; Beginner is NOT highlighted in
// that case because the item gates on a specific advanced class.
function isJobActive(reqJob: number, bit: number): boolean {
  if (reqJob === 0) return true;
  if (bit === 0) return false; // Beginner only highlighted on the "any" reqJob
  return (reqJob & bit) !== 0;
}

function getEquipmentCategory(templateId: number): string {
  const id = templateId;
  if (id >= 1000000 && id < 1010000) return "HAT";
  if (id >= 1010000 && id < 1020000) return "FACE ACCESSORY";
  if (id >= 1020000 && id < 1030000) return "EYE ACCESSORY";
  if (id >= 1030000 && id < 1040000) return "EARRING";
  if (id >= 1040000 && id < 1050000) return "TOP";
  if (id >= 1050000 && id < 1060000) return "OVERALL";
  if (id >= 1060000 && id < 1070000) return "BOTTOM";
  if (id >= 1070000 && id < 1080000) return "SHOES";
  if (id >= 1080000 && id < 1090000) return "GLOVES";
  if (id >= 1090000 && id < 1100000) return "CAPE";
  if (id >= 1100000 && id < 1110000) return "SHIELD";
  if (id >= 1110000 && id < 1120000) return "PENDANT";
  if (id >= 1120000 && id < 1130000) return "BELT";
  if (id >= 1130000 && id < 1140000) return "RING";
  if (id >= 1140000 && id < 1150000) return "MEDAL";
  if (id >= 1300000 && id < 1700000) return "WEAPON";
  if (id >= 1900000 && id < 2000000) return "MOUNT";
  return "EQUIPMENT";
}

function getCategory(templateId: number): string {
  const type = getItemType(String(templateId));
  if (type === "Equipment") return getEquipmentCategory(templateId);
  return type.toUpperCase();
}

export function AssetTooltipContent({ asset, itemName, slotName }: Props) {
  const templateId = asset.attributes.templateId;
  const isEquipment = getItemType(String(templateId)) === "Equipment";

  const itemDataQuery = useItemData(templateId);
  const equipmentQuery = useEquipmentData(templateId, { enabled: isEquipment });

  const resolvedName = itemName ?? itemDataQuery.name ?? `Item #${templateId}`;
  const eq = equipmentQuery.data?.attributes;
  const a = asset.attributes;
  const category = getCategory(templateId);

  // Two-column key/value grid where labels in column 1 and column 3 each
  // size to their widest content (`max-content`) — that's what makes the
  // values line up vertically across rows like the in-game tooltip.
  const colsStyle = { gridTemplateColumns: "max-content auto max-content auto" };

  return (
    <div className="space-y-2 w-[400px]">
      {/* Header — icon + name (+ optional slot label) */}
      <div className="flex items-center gap-2">
        {itemDataQuery.iconUrl && (
          <img
            src={itemDataQuery.iconUrl}
            alt={resolvedName}
            className="w-8 h-8 object-contain"
          />
        )}
        <span className="font-semibold">{resolvedName}</span>
        {slotName && (
          <span className="text-muted-foreground text-xs">({slotName})</span>
        )}
      </div>

      {/* Equipment — required stats (always shown as a fixed list) */}
      {isEquipment && eq && (
        <dl className="grid gap-x-3 gap-y-0.5 text-xs" style={colsStyle}>
          <ReqRow label="REQ LEV" value={eq.reqLevel} />
          <ReqRow label="REQ POP" value={eq.reqPop} />
          <ReqRow label="REQ STR" value={eq.reqStr} />
          <ReqRow label="REQ DEX" value={eq.reqDex} />
          <ReqRow label="REQ INT" value={eq.reqInt} />
          <ReqRow label="REQ LUK" value={eq.reqLuk} />
          {eq.reqFame > 0 && <ReqRow label="REQ FAM" value={eq.reqFame} />}
        </dl>
      )}

      {/* Equipment — class eligibility row */}
      {isEquipment && eq && (
        <div className="flex flex-wrap gap-1 text-[10px]">
          {JOB_BITS.map((j) => (
            <span
              key={j.name}
              className={cn(
                "rounded px-1.5 py-0.5 font-bold",
                isJobActive(eq.reqJob, j.bit)
                  ? "bg-orange-500/80 text-white"
                  : "bg-muted text-muted-foreground/40 line-through",
              )}
            >
              {j.name}
            </span>
          ))}
        </div>
      )}

      {/* Category */}
      <div className="text-xs">
        <span className="text-muted-foreground">CATEGORY: </span>
        <span>{category}</span>
      </div>

      {/* Equipment — actual stats from the asset (only non-zero rendered) */}
      {isEquipment && (
        <dl className="grid gap-x-3 gap-y-0.5 text-xs" style={colsStyle}>
          <StatRow label="STR" value={a.strength} />
          <StatRow label="DEX" value={a.dexterity} />
          <StatRow label="INT" value={a.intelligence} />
          <StatRow label="LUK" value={a.luck} />
          <StatRow label="HP" value={a.hp} />
          <StatRow label="MP" value={a.mp} />
          <StatRow label="WEAPON ATK" value={a.weaponAttack} />
          <StatRow label="MAGIC ATK" value={a.magicAttack} />
          <StatRow label="WEAPON DEF" value={a.weaponDefense} />
          <StatRow label="MAGIC DEF" value={a.magicDefense} />
          <StatRow label="ACCURACY" value={a.accuracy} />
          <StatRow label="AVOIDABILITY" value={a.avoidability} />
          <StatRow label="SPEED" value={a.speed} />
          <StatRow label="JUMP" value={a.jump} />
          {a.level > 0 && <StatRow label="ITEM LEV" value={a.level} />}
          {a.experience > 0 && <StatRow label="ITEM EXP" value={a.experience} />}
        </dl>
      )}

      {/* Equipment — upgrade slots & hammers (always shown for equipment) */}
      {isEquipment && (
        <dl className="grid gap-x-3 gap-y-0.5 text-xs" style={colsStyle}>
          <SimpleRow label="UPGRADES AVAILABLE" value={a.slots} />
          <SimpleRow label="HAMMERS APPLIED" value={a.hammersApplied} />
        </dl>
      )}

      {/* Non-equipment — quantity + expiration only */}
      {!isEquipment && a.quantity > 1 && (
        <div className="text-xs">
          <span className="text-muted-foreground">QUANTITY: </span>
          <span>{a.quantity}</span>
        </div>
      )}

      {a.expiration && a.expiration !== "" && a.expiration !== ZERO_DATE && (
        <div className="text-xs">
          <span className="text-muted-foreground">EXPIRES: </span>
          <span>{new Date(a.expiration).toLocaleDateString()}</span>
        </div>
      )}
    </div>
  );
}

// `<dt>` and `<dd>` participate in the parent grid's column layout so labels
// across rows line up at a consistent width. The wrapping `<></>` keeps the
// pair in DOM order without injecting an extra grid item.
function ReqRow({ label, value }: { label: string; value: number }) {
  return (
    <>
      <dt className="text-muted-foreground">{label}</dt>
      <dd>{value}</dd>
    </>
  );
}

function StatRow({ label, value }: { label: string; value: number }) {
  if (value === 0) return null;
  return (
    <>
      <dt className="text-muted-foreground">{label}</dt>
      <dd>+{value}</dd>
    </>
  );
}

function SimpleRow({ label, value }: { label: string; value: number }) {
  return (
    <>
      <dt className="text-muted-foreground">{label}</dt>
      <dd>{value}</dd>
    </>
  );
}
