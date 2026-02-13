import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { itemStringsService } from '@/services/api/item-strings.service';
import { useTenant } from '@/context/tenant-context';
import type { ItemStringData } from '@/types/models/item-string';

export const itemStringKeys = {
  all: ['item-strings'] as const,
  lists: () => [...itemStringKeys.all, 'list'] as const,
};

export function useItemStrings(): UseQueryResult<ItemStringData[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: itemStringKeys.lists(),
    queryFn: () => itemStringsService.getAllItemStrings(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 10 * 60 * 1000,
    gcTime: 30 * 60 * 1000,
  });
}

export function useItemName(itemId: string): UseQueryResult<string, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: [...itemStringKeys.all, 'name', itemId],
    queryFn: async () => {
      const item = await itemStringsService.getItemString(itemId, activeTenant!);
      return item.attributes.name;
    },
    enabled: !!itemId && !!activeTenant,
    staleTime: 10 * 60 * 1000,
    gcTime: 30 * 60 * 1000,
  });
}
