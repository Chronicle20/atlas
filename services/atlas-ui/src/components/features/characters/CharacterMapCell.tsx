import { Link } from "react-router-dom";
import type { Tenant } from "@/types/models/tenant";
import { useCharacterLocation } from "@/lib/hooks/api/useCharacterLocation";
import { MapCell } from "@/components/map-cell";

export function CharacterMapCell({
  characterId,
  tenant,
}: {
  characterId: string;
  tenant: Tenant | null;
}) {
  const { data: location } = useCharacterLocation(tenant, characterId);
  const mapId = location?.attributes.mapId;
  if (mapId == null) {
    return <span className="text-muted-foreground">—</span>;
  }
  const mapIdStr = String(mapId);
  return (
    <Link to={"/maps/" + mapIdStr}>
      <MapCell mapId={mapIdStr} tenant={tenant} />
    </Link>
  );
}
