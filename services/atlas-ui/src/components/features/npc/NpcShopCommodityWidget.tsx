import { Link } from "react-router-dom";
import { Package, Pencil, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";

interface NpcShopCommodityWidgetProps {
  templateId: number;
  mesoPrice: number;
  tokenPrice: number;
  tokenTemplateId: number;
  name?: string;
  iconUrl?: string;
  onEdit?: () => void;
  onDelete?: () => void;
}

export function NpcShopCommodityWidget({
  templateId,
  mesoPrice,
  tokenPrice,
  tokenTemplateId,
  name,
  iconUrl,
  onEdit,
  onDelete,
}: NpcShopCommodityWidgetProps) {
  const priceLine = formatPrice(mesoPrice, tokenPrice, tokenTemplateId);
  const displayName = name || `Item #${templateId}`;
  const hasActions = onEdit || onDelete;
  return (
    <div className="group flex items-center gap-3 rounded-md border bg-card p-2 hover:bg-accent transition-colors">
      <Link
        to={`/items/${templateId}`}
        className="flex flex-1 items-center gap-3 min-w-0"
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
      {hasActions && (
        <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
          {onEdit && (
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="h-7 w-7"
              onClick={onEdit}
              title="Edit Commodity"
              aria-label="Edit Commodity"
            >
              <Pencil className="h-3.5 w-3.5" />
            </Button>
          )}
          {onDelete && (
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="h-7 w-7 text-destructive hover:text-destructive"
              onClick={onDelete}
              title="Delete Commodity"
              aria-label="Delete Commodity"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          )}
        </div>
      )}
    </div>
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
