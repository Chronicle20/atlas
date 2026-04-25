import { useQuery } from "@tanstack/react-query";
import { useTenant } from "@/context/tenant-context";
import { itemsService } from "@/services/api/items.service";

interface MaterialRowProps {
  itemId: number;
  quantity: number;
}

export function MaterialRow({ itemId, quantity }: MaterialRowProps) {
  const { activeTenant } = useTenant();
  const { data, isError } = useQuery({
    queryKey: ["items", "name", activeTenant?.id ?? "no-tenant", String(itemId)],
    queryFn: () => itemsService.getItemName(String(itemId)),
    enabled: !!activeTenant && itemId > 0,
    staleTime: 10 * 60 * 1000,
  });

  const display = isError || !data ? `Item #${itemId}` : data;
  return (
    <li className="text-sm text-muted-foreground">
      {display} × {quantity.toLocaleString()}
    </li>
  );
}
