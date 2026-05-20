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
  type DropsSeedStatus,
  type GachaponsSeedStatus,
  type MapActionScriptsSeedStatus,
  type NpcConversationsSeedStatus,
  type NpcShopsSeedStatus,
  type PortalScriptsSeedStatus,
  type QuestConversationsSeedStatus,
  type ReactorScriptsSeedStatus,
  type WzInputStatus,
} from '@/services/api/seed.service';
import type { Scope } from '@/components/features/setup/ScopeToggle';
import { useTenant } from '@/context/tenant-context';

const wzInputStatusKey = (tenantId: string) => ['wzInputStatus', tenantId] as const;
const dataStatusKey = (tenantId: string) => ['dataStatus', tenantId] as const;
const dropsSeedStatusKey = (tenantId: string) => ['dropsSeedStatus', tenantId] as const;
const gachaponsSeedStatusKey = (tenantId: string) => ['gachaponsSeedStatus', tenantId] as const;
const npcConversationsSeedStatusKey = (tenantId: string) => ['npcConversationsSeedStatus', tenantId] as const;
const questConversationsSeedStatusKey = (tenantId: string) => ['questConversationsSeedStatus', tenantId] as const;
const npcShopsSeedStatusKey = (tenantId: string) => ['npcShopsSeedStatus', tenantId] as const;
const portalScriptsSeedStatusKey = (tenantId: string) => ['portalScriptsSeedStatus', tenantId] as const;
const reactorScriptsSeedStatusKey = (tenantId: string) => ['reactorScriptsSeedStatus', tenantId] as const;
const mapActionScriptsSeedStatusKey = (tenantId: string) => ['mapActionScriptsSeedStatus', tenantId] as const;

export function useSeedDrops(): UseMutationResult<void, Error, void> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.seedDrops(),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({ queryKey: dropsSeedStatusKey(activeTenant.id) });
    },
  });
}

export function useSeedGachapons(): UseMutationResult<void, Error, void> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.seedGachapons(),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({ queryKey: gachaponsSeedStatusKey(activeTenant.id) });
    },
  });
}

export function useSeedNpcConversations(): UseMutationResult<unknown, Error, void> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.seedNpcConversations(),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({ queryKey: npcConversationsSeedStatusKey(activeTenant.id) });
    },
  });
}

export function useSeedQuestConversations(): UseMutationResult<unknown, Error, void> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.seedQuestConversations(),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({ queryKey: questConversationsSeedStatusKey(activeTenant.id) });
    },
  });
}

export function useSeedNpcShops(): UseMutationResult<unknown, Error, void> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.seedNpcShops(),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({ queryKey: npcShopsSeedStatusKey(activeTenant.id) });
    },
  });
}

export function useSeedPortalScripts(): UseMutationResult<unknown, Error, void> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.seedPortalScripts(),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({ queryKey: portalScriptsSeedStatusKey(activeTenant.id) });
    },
  });
}

export function useSeedReactorScripts(): UseMutationResult<unknown, Error, void> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.seedReactorScripts(),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({ queryKey: reactorScriptsSeedStatusKey(activeTenant.id) });
    },
  });
}

export function useSeedMapActionScripts(): UseMutationResult<unknown, Error, void> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.seedMapActionScripts(),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({ queryKey: mapActionScriptsSeedStatusKey(activeTenant.id) });
    },
  });
}

export interface UploadWzFilesInput {
  file: File;
  scope: Scope;
}

export function useUploadWzFiles(): UseMutationResult<void, Error, UploadWzFilesInput> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ file, scope }: UploadWzFilesInput) =>
      seedService.uploadWzFiles(activeTenant!, file, scope),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({ queryKey: wzInputStatusKey(activeTenant.id) });
    },
  });
}

export function useRunDataProcessing(): UseMutationResult<void, Error, Scope> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (scope: Scope) => seedService.runDataProcessing(activeTenant!, scope),
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

export function useDropsSeedStatus(): UseQueryResult<DropsSeedStatus, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant ? dropsSeedStatusKey(activeTenant.id) : ['dropsSeedStatus', 'none'],
    queryFn: () => seedService.getDropsSeedStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useGachaponsSeedStatus(): UseQueryResult<GachaponsSeedStatus, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant ? gachaponsSeedStatusKey(activeTenant.id) : ['gachaponsSeedStatus', 'none'],
    queryFn: () => seedService.getGachaponsSeedStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useNpcConversationsSeedStatus(): UseQueryResult<NpcConversationsSeedStatus, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant
      ? npcConversationsSeedStatusKey(activeTenant.id)
      : ['npcConversationsSeedStatus', 'none'],
    queryFn: () => seedService.getNpcConversationsSeedStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useQuestConversationsSeedStatus(): UseQueryResult<QuestConversationsSeedStatus, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant
      ? questConversationsSeedStatusKey(activeTenant.id)
      : ['questConversationsSeedStatus', 'none'],
    queryFn: () => seedService.getQuestConversationsSeedStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useNpcShopsSeedStatus(): UseQueryResult<NpcShopsSeedStatus, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant ? npcShopsSeedStatusKey(activeTenant.id) : ['npcShopsSeedStatus', 'none'],
    queryFn: () => seedService.getNpcShopsSeedStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function usePortalScriptsSeedStatus(): UseQueryResult<PortalScriptsSeedStatus, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant
      ? portalScriptsSeedStatusKey(activeTenant.id)
      : ['portalScriptsSeedStatus', 'none'],
    queryFn: () => seedService.getPortalScriptsSeedStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useReactorScriptsSeedStatus(): UseQueryResult<ReactorScriptsSeedStatus, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant
      ? reactorScriptsSeedStatusKey(activeTenant.id)
      : ['reactorScriptsSeedStatus', 'none'],
    queryFn: () => seedService.getReactorScriptsSeedStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useMapActionScriptsSeedStatus(): UseQueryResult<MapActionScriptsSeedStatus, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant
      ? mapActionScriptsSeedStatusKey(activeTenant.id)
      : ['mapActionScriptsSeedStatus', 'none'],
    queryFn: () => seedService.getMapActionScriptsSeedStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}
