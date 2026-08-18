import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationResult,
  type UseQueryResult,
} from "@tanstack/react-query";
import { toast } from "sonner";
import {
  seedService,
  type DataStatus,
  type DropsSeedStatus,
  type EventDefinitionsSeedStatus,
  type GachaponsSeedStatus,
  type IngestRun,
  type InstanceRoutesSeedStatus,
  type ItemConversationsSeedStatus,
  type MapActionScriptsSeedStatus,
  type NpcConversationsSeedStatus,
  type NpcShopsSeedStatus,
  type PortalScriptsSeedStatus,
  type QuestConversationsSeedStatus,
  type ReactorScriptsSeedStatus,
  type TransportRoutesSeedStatus,
  type TransportVesselsSeedStatus,
  type WzInputStatus,
} from "@/services/api/seed.service";
import { useTenant } from "@/context/tenant-context";

// Shared WZ-upload error toast: one copy of the 409/400 wording for the
// tenant (Setup) and canonical (Baselines) upload paths.
export function showWzUploadErrorToast(error: Error): void {
  const err = error as Error & { status?: number };
  if (err.status === 409) {
    toast.error(
      "Another upload or processing job is in progress for this scope. Try again in a moment.",
    );
  } else if (err.status === 400) {
    toast.error(`Upload rejected: ${err.message}`);
  } else {
    toast.error(`Upload failed: ${err.message}`);
  }
}

const wzInputStatusKey = (tenantId: string) =>
  ["wzInputStatus", tenantId] as const;
export const dataStatusKey = (tenantId: string) =>
  ["dataStatus", tenantId] as const;
const ingestRunKey = (tenantId: string) => ["ingestRun", tenantId] as const;
const dropsSeedStatusKey = (tenantId: string) =>
  ["dropsSeedStatus", tenantId] as const;
const gachaponsSeedStatusKey = (tenantId: string) =>
  ["gachaponsSeedStatus", tenantId] as const;
const npcConversationsSeedStatusKey = (tenantId: string) =>
  ["npcConversationsSeedStatus", tenantId] as const;
const questConversationsSeedStatusKey = (tenantId: string) =>
  ["questConversationsSeedStatus", tenantId] as const;
const itemConversationsSeedStatusKey = (tenantId: string) =>
  ["itemConversationsSeedStatus", tenantId] as const;
const npcShopsSeedStatusKey = (tenantId: string) =>
  ["npcShopsSeedStatus", tenantId] as const;
const portalScriptsSeedStatusKey = (tenantId: string) =>
  ["portalScriptsSeedStatus", tenantId] as const;
const reactorScriptsSeedStatusKey = (tenantId: string) =>
  ["reactorScriptsSeedStatus", tenantId] as const;
const mapActionScriptsSeedStatusKey = (tenantId: string) =>
  ["mapActionScriptsSeedStatus", tenantId] as const;
const transportRoutesSeedStatusKey = (tenantId: string) =>
  ["transportRoutesSeedStatus", tenantId] as const;
const transportVesselsSeedStatusKey = (tenantId: string) =>
  ["transportVesselsSeedStatus", tenantId] as const;
const instanceRoutesSeedStatusKey = (tenantId: string) =>
  ["instanceRoutesSeedStatus", tenantId] as const;
const eventDefinitionsSeedStatusKey = (tenantId: string) =>
  ["eventDefinitionsSeedStatus", tenantId] as const;

export function useSeedDrops(): UseMutationResult<void, Error, void> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.seedDrops(),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({
        queryKey: dropsSeedStatusKey(activeTenant.id),
      });
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
      void queryClient.invalidateQueries({
        queryKey: gachaponsSeedStatusKey(activeTenant.id),
      });
    },
  });
}

export function useSeedNpcConversations(): UseMutationResult<
  unknown,
  Error,
  void
> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.seedNpcConversations(),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({
        queryKey: npcConversationsSeedStatusKey(activeTenant.id),
      });
    },
  });
}

export function useSeedQuestConversations(): UseMutationResult<
  unknown,
  Error,
  void
> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.seedQuestConversations(),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({
        queryKey: questConversationsSeedStatusKey(activeTenant.id),
      });
    },
  });
}

export function useSeedItemConversations(): UseMutationResult<
  unknown,
  Error,
  void
> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.seedItemConversations(),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({
        queryKey: itemConversationsSeedStatusKey(activeTenant.id),
      });
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
      void queryClient.invalidateQueries({
        queryKey: npcShopsSeedStatusKey(activeTenant.id),
      });
    },
  });
}

export function useSeedPortalScripts(): UseMutationResult<
  unknown,
  Error,
  void
> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.seedPortalScripts(),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({
        queryKey: portalScriptsSeedStatusKey(activeTenant.id),
      });
    },
  });
}

export function useSeedReactorScripts(): UseMutationResult<
  unknown,
  Error,
  void
> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.seedReactorScripts(),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({
        queryKey: reactorScriptsSeedStatusKey(activeTenant.id),
      });
    },
  });
}

export function useSeedMapActionScripts(): UseMutationResult<
  unknown,
  Error,
  void
> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.seedMapActionScripts(),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({
        queryKey: mapActionScriptsSeedStatusKey(activeTenant.id),
      });
    },
  });
}

export function useSeedTransportRoutes(): UseMutationResult<void, Error, void> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.seedRoutes(),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({
        queryKey: transportRoutesSeedStatusKey(activeTenant.id),
      });
    },
  });
}

export function useSeedTransportVessels(): UseMutationResult<
  void,
  Error,
  void
> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.seedVessels(),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({
        queryKey: transportVesselsSeedStatusKey(activeTenant.id),
      });
    },
  });
}

export function useSeedInstanceRoutes(): UseMutationResult<void, Error, void> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.seedInstanceRoutes(),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({
        queryKey: instanceRoutesSeedStatusKey(activeTenant.id),
      });
    },
  });
}

export function useSeedEventDefinitions(): UseMutationResult<
  void,
  Error,
  void
> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.seedEventDefinitions(),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({
        queryKey: eventDefinitionsSeedStatusKey(activeTenant.id),
      });
    },
  });
}

export interface UploadWzFilesInput {
  file: File;
}

export function useUploadWzFiles(): UseMutationResult<
  void,
  Error,
  UploadWzFilesInput
> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ file }: UploadWzFilesInput) =>
      seedService.uploadWzFiles(activeTenant!, file),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({
        queryKey: wzInputStatusKey(activeTenant.id),
      });
    },
    onError: showWzUploadErrorToast,
  });
}

export function useRunDataProcessing(): UseMutationResult<void, Error, void> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.runDataProcessing(activeTenant!),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({
        queryKey: dataStatusKey(activeTenant.id),
      });
    },
  });
}

export function useWzInputStatus(): UseQueryResult<WzInputStatus, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant
      ? wzInputStatusKey(activeTenant.id)
      : ["wzInputStatus", "none"],
    queryFn: () => seedService.getWzInputStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useDataStatus(): UseQueryResult<DataStatus, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant
      ? dataStatusKey(activeTenant.id)
      : ["dataStatus", "none"],
    queryFn: () => seedService.getDataStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useIngestRun(): UseQueryResult<IngestRun, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant
      ? ingestRunKey(activeTenant.id)
      : ["ingestRun", "none"],
    queryFn: () => seedService.getIngestRun(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
    // No retry, and the panel renders from query.isError rather than raising a
    // toast: an unreachable endpoint gives a steady "progress unavailable"
    // panel on a 5s cadence that never escalates.
    retry: false,
  });
}

export function useDropsSeedStatus(): UseQueryResult<DropsSeedStatus, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant
      ? dropsSeedStatusKey(activeTenant.id)
      : ["dropsSeedStatus", "none"],
    queryFn: () => seedService.getDropsSeedStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useGachaponsSeedStatus(): UseQueryResult<
  GachaponsSeedStatus,
  Error
> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant
      ? gachaponsSeedStatusKey(activeTenant.id)
      : ["gachaponsSeedStatus", "none"],
    queryFn: () => seedService.getGachaponsSeedStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useNpcConversationsSeedStatus(): UseQueryResult<
  NpcConversationsSeedStatus,
  Error
> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant
      ? npcConversationsSeedStatusKey(activeTenant.id)
      : ["npcConversationsSeedStatus", "none"],
    queryFn: () => seedService.getNpcConversationsSeedStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useQuestConversationsSeedStatus(): UseQueryResult<
  QuestConversationsSeedStatus,
  Error
> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant
      ? questConversationsSeedStatusKey(activeTenant.id)
      : ["questConversationsSeedStatus", "none"],
    queryFn: () => seedService.getQuestConversationsSeedStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useItemConversationsSeedStatus(): UseQueryResult<
  ItemConversationsSeedStatus,
  Error
> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant
      ? itemConversationsSeedStatusKey(activeTenant.id)
      : ["itemConversationsSeedStatus", "none"],
    queryFn: () => seedService.getItemConversationsSeedStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useNpcShopsSeedStatus(): UseQueryResult<
  NpcShopsSeedStatus,
  Error
> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant
      ? npcShopsSeedStatusKey(activeTenant.id)
      : ["npcShopsSeedStatus", "none"],
    queryFn: () => seedService.getNpcShopsSeedStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function usePortalScriptsSeedStatus(): UseQueryResult<
  PortalScriptsSeedStatus,
  Error
> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant
      ? portalScriptsSeedStatusKey(activeTenant.id)
      : ["portalScriptsSeedStatus", "none"],
    queryFn: () => seedService.getPortalScriptsSeedStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useReactorScriptsSeedStatus(): UseQueryResult<
  ReactorScriptsSeedStatus,
  Error
> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant
      ? reactorScriptsSeedStatusKey(activeTenant.id)
      : ["reactorScriptsSeedStatus", "none"],
    queryFn: () => seedService.getReactorScriptsSeedStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useMapActionScriptsSeedStatus(): UseQueryResult<
  MapActionScriptsSeedStatus,
  Error
> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant
      ? mapActionScriptsSeedStatusKey(activeTenant.id)
      : ["mapActionScriptsSeedStatus", "none"],
    queryFn: () => seedService.getMapActionScriptsSeedStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useTransportRoutesSeedStatus(): UseQueryResult<
  TransportRoutesSeedStatus,
  Error
> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant
      ? transportRoutesSeedStatusKey(activeTenant.id)
      : ["transportRoutesSeedStatus", "none"],
    queryFn: () => seedService.getTransportRoutesSeedStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useTransportVesselsSeedStatus(): UseQueryResult<
  TransportVesselsSeedStatus,
  Error
> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant
      ? transportVesselsSeedStatusKey(activeTenant.id)
      : ["transportVesselsSeedStatus", "none"],
    queryFn: () => seedService.getTransportVesselsSeedStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useInstanceRoutesSeedStatus(): UseQueryResult<
  InstanceRoutesSeedStatus,
  Error
> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant
      ? instanceRoutesSeedStatusKey(activeTenant.id)
      : ["instanceRoutesSeedStatus", "none"],
    queryFn: () => seedService.getInstanceRoutesSeedStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useEventDefinitionsSeedStatus(): UseQueryResult<
  EventDefinitionsSeedStatus,
  Error
> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant
      ? eventDefinitionsSeedStatusKey(activeTenant.id)
      : ["eventDefinitionsSeedStatus", "none"],
    queryFn: () => seedService.getEventDefinitionsSeedStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}
