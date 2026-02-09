import { useMutation, type UseMutationResult } from '@tanstack/react-query';
import { seedService } from '@/services/api/seed.service';
import { useTenant } from '@/context/tenant-context';

export function useSeedDrops(): UseMutationResult<void, Error, void> {
  const { activeTenant } = useTenant();
  return useMutation({ mutationFn: () => seedService.seedDrops(activeTenant!) });
}

export function useSeedGachapons(): UseMutationResult<void, Error, void> {
  const { activeTenant } = useTenant();
  return useMutation({ mutationFn: () => seedService.seedGachapons(activeTenant!) });
}

export function useSeedNpcConversations(): UseMutationResult<unknown, Error, void> {
  const { activeTenant } = useTenant();
  return useMutation({ mutationFn: () => seedService.seedNpcConversations(activeTenant!) });
}

export function useSeedQuestConversations(): UseMutationResult<unknown, Error, void> {
  const { activeTenant } = useTenant();
  return useMutation({ mutationFn: () => seedService.seedQuestConversations(activeTenant!) });
}

export function useSeedNpcShops(): UseMutationResult<unknown, Error, void> {
  const { activeTenant } = useTenant();
  return useMutation({ mutationFn: () => seedService.seedNpcShops(activeTenant!) });
}

export function useSeedPortalScripts(): UseMutationResult<unknown, Error, void> {
  const { activeTenant } = useTenant();
  return useMutation({ mutationFn: () => seedService.seedPortalScripts(activeTenant!) });
}

export function useSeedReactorScripts(): UseMutationResult<unknown, Error, void> {
  const { activeTenant } = useTenant();
  return useMutation({ mutationFn: () => seedService.seedReactorScripts(activeTenant!) });
}

export function useUploadGameData(): UseMutationResult<void, Error, File> {
  const { activeTenant } = useTenant();
  return useMutation({ mutationFn: (file: File) => seedService.uploadGameData(activeTenant!, file) });
}
