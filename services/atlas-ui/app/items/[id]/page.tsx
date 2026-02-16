"use client"

import { useTenant } from "@/context/tenant-context";
import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { itemsService } from "@/services/api/items.service";
import {
  getItemType,
  getItemTypeBadgeVariant,
  type ItemType,
  type ItemDetailData,
  type EquipmentData,
  type ConsumableData,
  type SetupData,
  type EtcData,
  type CashItemData,
} from "@/types/models/item";
import { toast } from "sonner";
import { createErrorFromUnknown } from "@/types/api/errors";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { PageLoader } from "@/components/common/PageLoader";
import { Package } from "lucide-react";
import Link from "next/link";
import Image from "next/image";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import { shouldUnoptimizeImageSrc } from "@/lib/utils/image";

export default function ItemDetailPage() {
  const { activeTenant } = useTenant();
  const params = useParams();
  const itemId = params.id as string;
  const itemType = getItemType(itemId);

  const [itemName, setItemName] = useState<string | null>(null);
  const [detail, setDetail] = useState<ItemDetailData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!activeTenant || !itemId) return;

    const fetchData = async () => {
      setLoading(true);
      setError(null);

      try {
        const [name, detailData] = await Promise.all([
          itemsService.getItemName(itemId, activeTenant),
          itemType !== "Unknown"
            ? itemsService.getItemDetail(itemId, activeTenant)
            : Promise.resolve(null),
        ]);
        setItemName(name);
        setDetail(detailData);
      } catch (err: unknown) {
        const errorInfo = createErrorFromUnknown(err, "Failed to load item details");
        setError(errorInfo.message);
        toast.error(errorInfo.message);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [activeTenant, itemId, itemType]);

  if (loading) {
    return <PageLoader />;
  }

  if (error) {
    return (
      <div className="flex flex-col flex-1 min-h-0 space-y-6 p-10 pb-16">
        <div className="text-center py-8 text-muted-foreground">{error}</div>
      </div>
    );
  }

  return (
    <div className="flex flex-col flex-1 min-h-0 space-y-6 p-10 pb-16">
      <div className="flex items-center gap-2">
        {activeTenant ? (
          <Image
            src={getAssetIconUrl(
              activeTenant.id,
              activeTenant.attributes.region,
              activeTenant.attributes.majorVersion,
              activeTenant.attributes.minorVersion,
              'item',
              parseInt(itemId),
            )}
            alt={itemName || itemId}
            width={40}
            height={40}
            unoptimized={shouldUnoptimizeImageSrc(getAssetIconUrl(
              activeTenant.id,
              activeTenant.attributes.region,
              activeTenant.attributes.majorVersion,
              activeTenant.attributes.minorVersion,
              'item',
              parseInt(itemId),
            ))}
            className="object-contain"
          />
        ) : (
          <Package className="h-6 w-6" />
        )}
        <div>
          <div className="flex items-center gap-2">
            <h2 className="text-2xl font-bold tracking-tight">{itemName || itemId}</h2>
            <Badge variant="secondary" className={getItemTypeBadgeVariant(itemType)}>
              {itemType}
            </Badge>
          </div>
          <p className="text-sm text-muted-foreground">
            <Link href="/items" className="hover:underline">Items</Link>
            {" > "}
            <span>{itemName || itemId}</span>
          </p>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>General</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            <InfoField label="Template" value={itemId} mono />
            <InfoField label="Name" value={itemName || "Unknown"} />
            <InfoField label="Type" value={itemType} />
          </div>
        </CardContent>
      </Card>

      {detail && renderTypeSpecificSection(itemType, detail)}
    </div>
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
            <InfoField label="Speed" value={a.speed} />
            <InfoField label="Jump" value={a.jump} />
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Combat</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-3 lg:grid-cols-4">
            <InfoField label="Weapon Attack" value={a.weaponAttack} />
            <InfoField label="Magic Attack" value={a.magicAttack} />
            <InfoField label="Weapon Defense" value={a.weaponDefense} />
            <InfoField label="Magic Defense" value={a.magicDefense} />
            <InfoField label="Accuracy" value={a.accuracy} />
            <InfoField label="Avoidability" value={a.avoidability} />
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Properties</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-3 lg:grid-cols-4">
            <InfoField label="Upgrade Slots" value={a.slots} />
            <InfoField label="Price" value={a.price} />
            <InfoField label="Cash" value={a.cash} />
            <InfoField label="Time Limited" value={a.timeLimited} />
          </div>
        </CardContent>
      </Card>
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
            <InfoField label="Price" value={a.price} />
            <InfoField label="Unit Price" value={a.unitPrice} />
            <InfoField label="Slot Max" value={a.slotMax} />
            <InfoField label="Required Level" value={a.reqLevel} />
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
          <InfoField label="Price" value={a.price} />
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
          <InfoField label="Price" value={a.price} />
          <InfoField label="Unit Price" value={a.unitPrice} />
          <InfoField label="Slot Max" value={a.slotMax} />
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
