import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { itemStringsService } from '@/services/api/item-strings.service';
import { useTenant } from '@/context/tenant-context';

export const itemStringKeys = {
  all: ['item-strings'] as const,
  byId: (id: string) => [...itemStringKeys.all, 'name', id] as const,
};

export function useItemName(itemId: string): UseQueryResult<string, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: itemStringKeys.byId(itemId),
    queryFn: async () => {
      const item = await itemStringsService.getItemString(itemId);
      return item.attributes.name;
    },
    enabled: !!itemId && !!activeTenant,
    staleTime: 10 * 60 * 1000,
    gcTime: 30 * 60 * 1000,
  });
}
