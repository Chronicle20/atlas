/**
 * React Query hooks for the transports surface.
 *
 * Every query is gated on an active tenant and polls on a 30-second interval.
 * Countdowns tick locally off the shared clock store between polls, so this is
 * the only network cadence on the page. Tenant switching is already handled by
 * TenantProvider's cache clear — there is no transport-specific invalidation.
 */

import {
  keepPreviousData,
  useQueries,
  useQuery,
  type UseQueryResult,
} from "@tanstack/react-query";

import { useTenant } from "@/context/tenant-context";
import { transportsService } from "@/services/api/transports.service";
import type {
  InstanceRoute,
  InstanceStatus,
  ScheduledRoute,
  ScheduledRouteDetail,
  Vessel,
} from "@/types/models/transport";

export const TRANSPORT_POLL_MS = 30_000;

export const transportKeys = {
  all: ["transports"] as const,
  scheduled: (tenantId: string) =>
    [...transportKeys.all, "scheduled", tenantId] as const,
  scheduledDetail: (tenantId: string, routeId: string) =>
    [...transportKeys.all, "scheduled", tenantId, "detail", routeId] as const,
  instanceRoutes: (tenantId: string) =>
    [...transportKeys.all, "instance-routes", tenantId] as const,
  instanceStatus: (tenantId: string, routeId: string) =>
    [...transportKeys.all, "instance-status", tenantId, routeId] as const,
  vessels: (tenantId: string) =>
    [...transportKeys.all, "vessels", tenantId] as const,
};

const pollDefaults = {
  refetchInterval: TRANSPORT_POLL_MS,
  refetchIntervalInBackground: false,
  placeholderData: keepPreviousData,
} as const;

export function useScheduledRoutes(): UseQueryResult<ScheduledRoute[], Error> {
  const { activeTenant } = useTenant();
  const tenantId = activeTenant?.id ?? "no-tenant";

  return useQuery({
    queryKey: transportKeys.scheduled(tenantId),
    queryFn: () => transportsService.getScheduledRoutes(),
    enabled: !!activeTenant,
    ...pollDefaults,
  });
}

export function useScheduledRoute(
  routeId: string,
): UseQueryResult<ScheduledRouteDetail, Error> {
  const { activeTenant } = useTenant();
  const tenantId = activeTenant?.id ?? "no-tenant";

  return useQuery({
    queryKey: transportKeys.scheduledDetail(tenantId, routeId),
    queryFn: () => transportsService.getScheduledRoute(routeId),
    enabled: !!activeTenant && !!routeId,
    ...pollDefaults,
  });
}

export function useInstanceRoutes(): UseQueryResult<InstanceRoute[], Error> {
  const { activeTenant } = useTenant();
  const tenantId = activeTenant?.id ?? "no-tenant";

  return useQuery({
    queryKey: transportKeys.instanceRoutes(tenantId),
    queryFn: () => transportsService.getInstanceRoutes(),
    enabled: !!activeTenant,
    ...pollDefaults,
  });
}

/**
 * One status query per instance route, index-aligned with `routeIds`.
 *
 * The Instance tab needs a live count for every route, including collapsed
 * rows, so fetching only for expanded rows will not do. Twelve small,
 * usually-empty responses per 30s is 0.4 rps — cheaper than adding an
 * aggregate endpoint and a Redis scan for a twelve-item list.
 */
export function useInstanceStatuses(
  routeIds: string[],
): UseQueryResult<InstanceStatus[], Error>[] {
  const { activeTenant } = useTenant();
  const tenantId = activeTenant?.id ?? "no-tenant";

  return useQueries({
    queries: routeIds.map((routeId) => ({
      queryKey: transportKeys.instanceStatus(tenantId, routeId),
      queryFn: () => transportsService.getInstanceStatuses(routeId),
      enabled: !!activeTenant,
      ...pollDefaults,
    })),
  });
}

export function useVessels(): UseQueryResult<Vessel[], Error> {
  const { activeTenant } = useTenant();
  const tenantId = activeTenant?.id ?? "no-tenant";

  return useQuery({
    queryKey: transportKeys.vessels(tenantId),
    queryFn: () => transportsService.getVessels(tenantId),
    enabled: !!activeTenant,
    ...pollDefaults,
  });
}
