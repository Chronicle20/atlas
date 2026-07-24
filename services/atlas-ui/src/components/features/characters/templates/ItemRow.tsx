import { useState } from "react";
import { Package, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useItemName } from "@/lib/hooks/api/useItemStrings";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import { useTenant } from "@/context/tenant-context";

interface ItemRowProps {
  id: number;
  onRemove: () => void;
  removeAriaLabel: string;
}

/** Icon + display name + mono id + remove ×. Bad ids degrade, never block. */
export function ItemRow({ id, onRemove, removeAriaLabel }: ItemRowProps) {
  const { activeTenant } = useTenant();
  const name = useItemName(String(id));
  const [iconFailed, setIconFailed] = useState(false);

  return (
    <div className="flex items-center gap-2 rounded-md border px-2 py-1.5">
      {activeTenant && !iconFailed ? (
        <img
          src={getAssetIconUrl(
            activeTenant.id,
            activeTenant.attributes.region,
            activeTenant.attributes.majorVersion,
            activeTenant.attributes.minorVersion,
            "item",
            id,
          )}
          alt=""
          width={28}
          height={28}
          loading="lazy"
          onError={() => setIconFailed(true)}
          className="[image-rendering:pixelated]"
        />
      ) : (
        <Package className="size-7 p-1 text-muted-foreground" />
      )}
      <span className="flex-1 truncate text-sm">
        {name.data ?? (name.isError ? "Unknown item" : "…")}
      </span>
      <span className="font-mono text-xs text-muted-foreground">{id}</span>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        aria-label={removeAriaLabel}
        onClick={onRemove}
      >
        <X className="size-4" />
      </Button>
    </div>
  );
}
