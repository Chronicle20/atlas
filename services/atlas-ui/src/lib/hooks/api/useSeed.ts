import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationResult,
  type UseQueryResult,
} from '@tanstack/react-query';
import {
  seedService,
  type DataStatus,
  type WzExtractionStatus,
  type WzInputStatus,
} from '@/services/api/seed.service';
import { useTenant } from '@/context/tenant-context';

const wzInputStatusKey = (tenantId: string) => ['wzInputStatus', tenantId] as const;
const extractionStatusKey = (tenantId: string) => ['extractionStatus', tenantId] as const;
const dataStatusKey = (tenantId: string) => ['dataStatus', tenantId] as const;

export function useSeedDrops(): UseMutationResult<void, Error, void> {
  // seed hooks no longer need the tenant argument.
  return useMutation({ mutationFn: () => seedService.seedDrops() });
}

export function useSeedGachapons(): UseMutationResult<void, Error, void> {
  // seed hooks no longer need the tenant argument.
  return useMutation({ mutationFn: () => seedService.seedGachapons() });
}

export function useSeedNpcConversations(): UseMutationResult<unknown, Error, void> {
  // seed hooks no longer need the tenant argument.
  return useMutation({ mutationFn: () => seedService.seedNpcConversations() });
}

export function useSeedQuestConversations(): UseMutationResult<unknown, Error, void> {
  // seed hooks no longer need the tenant argument.
  return useMutation({ mutationFn: () => seedService.seedQuestConversations() });
}

export function useSeedNpcShops(): UseMutationResult<unknown, Error, void> {
  // seed hooks no longer need the tenant argument.
  return useMutation({ mutationFn: () => seedService.seedNpcShops() });
}

export function useSeedPortalScripts(): UseMutationResult<unknown, Error, void> {
  // seed hooks no longer need the tenant argument.
  return useMutation({ mutationFn: () => seedService.seedPortalScripts() });
}

export function useSeedReactorScripts(): UseMutationResult<unknown, Error, void> {
  // seed hooks no longer need the tenant argument.
  return useMutation({ mutationFn: () => seedService.seedReactorScripts() });
}

export function useUploadWzFiles(): UseMutationResult<void, Error, File> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (file: File) => seedService.uploadWzFiles(activeTenant!, file),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({ queryKey: wzInputStatusKey(activeTenant.id) });
      void queryClient.invalidateQueries({ queryKey: extractionStatusKey(activeTenant.id) });
    },
  });
}

export function useRunWzExtraction(): UseMutationResult<void, Error, void> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.runWzExtraction(activeTenant!),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({ queryKey: extractionStatusKey(activeTenant.id) });
      void queryClient.invalidateQueries({ queryKey: dataStatusKey(activeTenant.id) });
    },
  });
}

export function useRunDataProcessing(): UseMutationResult<void, Error, void> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.runDataProcessing(activeTenant!),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({ queryKey: dataStatusKey(activeTenant.id) });
    },
  });
}

export function useWzInputStatus(): UseQueryResult<WzInputStatus, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant ? wzInputStatusKey(activeTenant.id) : ['wzInputStatus', 'none'],
    queryFn: () => seedService.getWzInputStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useExtractionStatus(): UseQueryResult<WzExtractionStatus, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant ? extractionStatusKey(activeTenant.id) : ['extractionStatus', 'none'],
    queryFn: () => seedService.getExtractionStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useDataStatus(): UseQueryResult<DataStatus, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant ? dataStatusKey(activeTenant.id) : ['dataStatus', 'none'],
    queryFn: () => seedService.getDataStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}
