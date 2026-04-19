import { useQuery } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { useTenant } from "@/context/tenant-context";
import { itemsService } from "@/services/api/items.service";
import {
  getItemType,
  type ItemType,
  type ItemDetailData,
  type EquipmentData,
  type ConsumableData,
  type SetupData,
  type EtcData,
  type CashItemData,
} from "@/types/models/item";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { PageLoader } from "@/components/common/PageLoader";
import { useItemDrops } from "@/lib/hooks/api/useDrops";
import { useItemSellers } from "@/lib/hooks/api/useItemSellers";
import { useItemCommodities } from "@/lib/hooks/api/useItemCommodities";
import { ItemHeader } from "@/components/features/items/ItemHeader";
import { EquipmentRequirementsCard } from "@/components/features/items/EquipmentRequirementsCard";
import { ItemNpcShopWidget } from "@/components/features/items/ItemNpcShopWidget";
import { ItemCashShopWidget } from "@/components/features/items/ItemCashShopWidget";
import { DroppedByWidget } from "@/components/features/items/DroppedByWidget";

export function ItemDetailPage() {
  const { activeTenant } = useTenant();
  const params = useParams();
  const itemId = params.id as string;
  const itemType = getItemType(itemId);

  const nameQuery = useQuery({
    queryKey: ["items", "name", activeTenant?.id ?? "no-tenant", itemId],
    queryFn: () => itemsService.getItemName(itemId),
    enabled: !!activeTenant && !!itemId,
    staleTime: 10 * 60 * 1000,
  });

  const detailQuery = useQuery({
    queryKey: ["items", "detail", activeTenant?.id ?? "no-tenant", itemId, itemType],
    queryFn: () => itemsService.getItemDetail(itemId),
    enabled: !!activeTenant && !!itemId && itemType !== "Unknown",
    staleTime: 10 * 60 * 1000,
  });

  const { data: drops, isLoading: dropsLoading } = useItemDrops(itemId);
  const { data: sellers, isLoading: sellersLoading } = useItemSellers(itemId);
  const { data: commodities, isLoading: commoditiesLoading } = useItemCommodities(itemId);

  const itemName = nameQuery.data ?? null;
  const detail = detailQuery.data ?? null;
  const loading = nameQuery.isLoading || detailQuery.isLoading;
  const error = nameQuery.error?.message ?? detailQuery.error?.message ?? null;

  if (loading) {
    return <PageLoader />;
  }

  if (error) {
    return (
      <div className="flex flex-col flex-1 min-h-0 overflow-y-auto space-y-6 p-10 pb-16">
        <div className="text-center py-8 text-muted-foreground">{error}</div>
      </div>
    );
  }

  const price = detail ? getDetailPrice(itemType, detail) : 0;

  const sortedDrops = drops
    ? [...drops].sort((a, b) => {
        if (b.attributes.chance !== a.attributes.chance) {
          return b.attributes.chance - a.attributes.chance;
        }
        return a.attributes.monsterId - b.attributes.monsterId;
      })
    : [];

  return (
    <div className="flex flex-col flex-1 min-h-0 overflow-y-auto space-y-6 p-10 pb-16">
      <ItemHeader itemId={itemId} itemName={itemName} itemType={itemType} />

      {detail && renderTypeSpecificSection(itemType, detail)}

      <SoldByCard
        sellers={sellers}
        sellersLoading={sellersLoading}
        commodities={commodities}
        commoditiesLoading={commoditiesLoading}
        price={price}
      />

      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium">
            Dropped By {drops && `(${drops.length})`}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {dropsLoading ? (
            <p className="text-sm text-muted-foreground">Loading drop sources...</p>
          ) : sortedDrops.length > 0 ? (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
              {sortedDrops.map((drop) => (
                <DroppedByWidget key={drop.id} drop={drop} />
              ))}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">No monsters drop this item.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function getDetailPrice(type: ItemType, detail: ItemDetailData): number {
  switch (type) {
    case "Consumable":
      return (detail as ConsumableData).attributes.price ?? 0;
    case "Setup":
      return (detail as SetupData).attributes.price ?? 0;
    case "Etc":
      return (detail as EtcData).attributes.price ?? 0;
    default:
      return 0;
  }
}

interface SoldByCardProps {
  sellers: import("@/types/models/npc").ItemSellerCommodity[] | undefined;
  sellersLoading: boolean;
  commodities: import("@/types/models/npc").ItemCashShopCommodity[] | undefined;
  commoditiesLoading: boolean;
  price: number;
}

function SoldByCard({
  sellers,
  sellersLoading,
  commodities,
  commoditiesLoading,
  price,
}: SoldByCardProps) {
  const hasSellers = sellers && sellers.length > 0;
  const hasCommodities = commodities && commodities.length > 0;
  const loading = sellersLoading || commoditiesLoading;

  const title = hasSellers || hasCommodities
    ? `Sold By (NPC: ${sellers?.length ?? 0}, Cash Shop: ${commodities?.length ?? 0})`
    : "Sold By";

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {loading ? (
          <p className="text-sm text-muted-foreground">Loading shop data...</p>
        ) : !hasSellers && !hasCommodities ? (
          <p className="text-sm text-muted-foreground">No shops or commodities sell this item.</p>
        ) : (
          <>
            {hasSellers && (
              <div className="space-y-2">
                <h3 className="text-sm font-medium text-muted-foreground uppercase tracking-wide">
                  NPC Shops ({sellers!.length})
                </h3>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                  {sellers!.map((commodity) => (
                    <ItemNpcShopWidget key={commodity.id} commodity={commodity} />
                  ))}
                </div>
              </div>
            )}
            {hasCommodities && (
              <div className="space-y-2">
                <h3 className="text-sm font-medium text-muted-foreground uppercase tracking-wide">
                  Cash Shop ({commodities!.length})
                </h3>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                  {commodities!.map((commodity) => (
                    <ItemCashShopWidget key={commodity.id} commodity={commodity} />
                  ))}
                </div>
              </div>
            )}
            {price > 0 && (
              <p className="text-xs text-muted-foreground">
                Base price: {price.toLocaleString()} mesos
              </p>
            )}
          </>
        )}
      </CardContent>
    </Card>
  );
}

function InfoField({ label, value, mono }: { label: string; value: string | number | boolean; mono?: boolean }) {
  const displayValue = typeof value === "boolean" ? (value ? "Yes" : "No") : String(value);
  return (
    <div className="space-y-1">
      <p className="text-sm text-muted-foreground">{label}</p>
      <p className={`text-sm font-medium ${mono ? "font-mono" : ""}`}>{displayValue}</p>
    </div>
  );
}

function renderTypeSpecificSection(type: ItemType, detail: ItemDetailData) {
  switch (type) {
    case "Equipment": return <EquipmentSection data={detail as EquipmentData} />;
    case "Consumable": return <ConsumableSection data={detail as ConsumableData} />;
    case "Setup": return <SetupSection data={detail as SetupData} />;
    case "Etc": return <EtcSection data={detail as EtcData} />;
    case "Cash": return <CashSection data={detail as CashItemData} />;
    default: return null;
  }
}

function EquipmentSection({ data }: { data: EquipmentData }) {
  const a = data.attributes;
  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>Stats</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-3 lg:grid-cols-4">
            <InfoField label="STR" value={a.strength} />
            <InfoField label="DEX" value={a.dexterity} />
            <InfoField label="INT" value={a.intelligence} />
            <InfoField label="LUK" value={a.luck} />
            <InfoField label="HP" value={a.hp} />
            <InfoField label="MP" value={a.mp} />
            <InfoField label="Weapon Attack" value={a.weaponAttack} />
            <InfoField label="Magic Attack" value={a.magicAttack} />
            <InfoField label="Weapon Defense" value={a.weaponDefense} />
            <InfoField label="Magic Defense" value={a.magicDefense} />
            <InfoField label="Accuracy" value={a.accuracy} />
            <InfoField label="Avoidability" value={a.avoidability} />
            <InfoField label="Speed" value={a.speed} />
            <InfoField label="Jump" value={a.jump} />
            <InfoField label="Upgrade Slots" value={a.slots} />
            <InfoField label="Cash" value={a.cash} />
            <InfoField label="Time Limited" value={a.timeLimited} />
          </div>
        </CardContent>
      </Card>
      <EquipmentRequirementsCard attributes={a} />
    </>
  );
}

function ConsumableSection({ data }: { data: ConsumableData }) {
  const a = data.attributes;
  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>Properties</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-3 lg:grid-cols-4">
            <InfoField label="Slot Max" value={a.slotMax} />
            <InfoField label="Required Level" value={a.reqLevel} />
            <InfoField label="Unit Price" value={a.unitPrice} />
            <InfoField label="Quest Item" value={a.quest} />
            <InfoField label="Trade Block" value={a.tradeBlock} />
            <InfoField label="Not For Sale" value={a.notSale} />
            <InfoField label="Time Limited" value={a.timeLimited} />
            <InfoField label="Rechargeable" value={a.rechargeable} />
          </div>
        </CardContent>
      </Card>
      {a.success > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Scroll Effects</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-3">
              <InfoField label="Success Rate" value={`${a.success}%`} />
              <InfoField label="Curse Rate" value={`${a.cursed}%`} />
            </div>
          </CardContent>
        </Card>
      )}
      {a.spec && Object.keys(a.spec).length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Spec</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-3 lg:grid-cols-4">
              {Object.entries(a.spec).map(([key, value]) => (
                <InfoField key={key} label={key} value={value} />
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </>
  );
}

function SetupSection({ data }: { data: SetupData }) {
  const a = data.attributes;
  return (
    <Card>
      <CardHeader>
        <CardTitle>Properties</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid gap-4 md:grid-cols-3 lg:grid-cols-4">
          <InfoField label="Slot Max" value={a.slotMax} />
          <InfoField label="Recovery HP" value={a.recoveryHP} />
          <InfoField label="Required Level" value={a.reqLevel} />
          <InfoField label="Trade Block" value={a.tradeBlock} />
          <InfoField label="Not For Sale" value={a.notSale} />
          <InfoField label="Time Limited" value={a.timeLimited} />
        </div>
      </CardContent>
    </Card>
  );
}

function EtcSection({ data }: { data: EtcData }) {
  const a = data.attributes;
  return (
    <Card>
      <CardHeader>
        <CardTitle>Properties</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid gap-4 md:grid-cols-3 lg:grid-cols-4">
          <InfoField label="Slot Max" value={a.slotMax} />
          <InfoField label="Unit Price" value={a.unitPrice} />
          <InfoField label="Time Limited" value={a.timeLimited} />
        </div>
      </CardContent>
    </Card>
  );
}

function CashSection({ data }: { data: CashItemData }) {
  const a = data.attributes;
  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>Properties</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-3">
            <InfoField label="Slot Max" value={a.slotMax} />
          </div>
        </CardContent>
      </Card>
      {a.spec && Object.keys(a.spec).length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Spec</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-3 lg:grid-cols-4">
              {Object.entries(a.spec).map(([key, value]) => (
                <InfoField key={key} label={key} value={value} />
              ))}
            </div>
          </CardContent>
        </Card>
      )}
      {a.timeWindows && a.timeWindows.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Time Windows</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {a.timeWindows.map((tw, idx) => (
                <InfoField key={idx} label={tw.day} value={`${tw.startHour}:00 - ${tw.endHour}:00`} />
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </>
  );
}
