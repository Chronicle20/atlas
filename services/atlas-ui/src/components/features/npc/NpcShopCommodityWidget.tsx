import { Link } from "react-router-dom";
import { Package } from "lucide-react";

interface NpcShopCommodityWidgetProps {
  templateId: number;
  mesoPrice: number;
  tokenPrice: number;
  tokenTemplateId: number;
  name?: string;
  iconUrl?: string;
}

export function NpcShopCommodityWidget({
  templateId,
  mesoPrice,
  tokenPrice,
  tokenTemplateId,
  name,
  iconUrl,
}: NpcShopCommodityWidgetProps) {
  const priceLine = formatPrice(mesoPrice, tokenPrice, tokenTemplateId);
  const displayName = name || `Item #${templateId}`;
  return (
    <Link
      to={`/items/${templateId}`}
      className="flex items-center gap-3 rounded-md border bg-card p-2 hover:bg-accent transition-colors"
    >
      <div className="flex h-8 w-8 shrink-0 items-center justify-center">
        {iconUrl ? (
          <img
            src={iconUrl}
            alt={displayName}
            width={32}
            height={32}
            loading="lazy"
            className="max-h-full max-w-full object-contain"
          />
        ) : (
          <Package className="h-5 w-5 text-muted-foreground" />
        )}
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium truncate">{displayName}</p>
        <p className="text-xs text-muted-foreground truncate">{priceLine}</p>
      </div>
    </Link>
  );
}

function formatPrice(
  mesoPrice: number,
  tokenPrice: number,
  tokenTemplateId: number,
): string {
  const parts: string[] = [];
  if (mesoPrice > 0) parts.push(`${mesoPrice.toLocaleString()} mesos`);
  if (tokenPrice > 0 && tokenTemplateId > 0) {
    parts.push(`${tokenPrice.toLocaleString()} × item ${tokenTemplateId}`);
  }
  if (parts.length === 0) return "Free";
  return parts.join(" · ");
}
