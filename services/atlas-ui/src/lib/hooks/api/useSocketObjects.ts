/**
 * React Query hooks for the Packet Matrix's data layer.
 *
 * Two read paths, deliberately kept apart:
 *   - useSocketMatrixTemplates / useSocketMatrixTenants: SPARSE reads
 *     (region/majorVersion/minorVersion/socket only) that feed the grid.
 *   - useSocketMutation: the single write path. It always re-fetches the
 *     FULL document by id before writing - see the query-key comment below
 *     for why the sparse documents can never reach it directly.
 */
import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationResult,
  type UseQueryResult,
} from "@tanstack/react-query";

import { templatesService } from "@/services/api/templates.service";
import { tenantsService } from "@/services/api/tenants.service";
import { templateKeys } from "@/lib/hooks/api/useTemplates";
import { tenantKeys } from "@/lib/hooks/api/useTenants";
import { fromTemplate, fromTenantConfig } from "@/lib/socket/normalize";
import type { SocketObject } from "@/lib/socket/model";
import type { SocketConfig } from "@/types/models/socket";

/**
 * Query keys for the SPARSE socket reads.
 *
 * These deliberately do NOT reuse templateKeys.detail / tenantKeys.configDetail.
 * A sparse document that reached a mutation's attribute spread would silently
 * erase characters, worlds and cashShop (see tenantsService.updateTenantConfiguration
 * and templatesService.update, both of which write the whole document), so the
 * two live under separate keys and the sparse one is never a write input -
 * useSocketMutation below never reads from socketKeys.* at all.
 */
export const socketKeys = {
  all: ["socket"] as const,
  matrix: () => [...socketKeys.all, "matrix", "templates"] as const,
  tenantMatrix: () => [...socketKeys.all, "matrix", "tenants"] as const,
};

/** Sparse Template read for the grid. Never pass this query's data to useSocketMutation. */
export function useSocketMatrixTemplates(): UseQueryResult<
  SocketObject[],
  Error
> {
  return useQuery({
    queryKey: socketKeys.matrix(),
    queryFn: async () =>
      (await templatesService.getSocketMatrix()).map(fromTemplate),
    staleTime: 30_000,
  });
}

/** Sparse TenantConfig read for the grid. Never pass this query's data to useSocketMutation. */
export function useSocketMatrixTenants(): UseQueryResult<
  SocketObject[],
  Error
> {
  return useQuery({
    queryKey: socketKeys.tenantMatrix(),
    queryFn: async () =>
      (await tenantsService.getSocketMatrix()).map(fromTenantConfig),
    staleTime: 30_000,
  });
}

export interface SocketTarget {
  source: "template" | "tenant";
  id: string;
}

export interface SocketMutationInput {
  target: SocketTarget;
  /** A pure splice from lib/socket/mutate. May throw MutationError. */
  apply: (cfg: SocketConfig) => SocketConfig;
}

/**
 * The single write path for every socket dialog and bulk flow.
 *
 * It re-fetches the FULL document by id, applies the pure splice to that
 * fresh copy, and PATCHes the whole attribute document back. Re-fetching is
 * not belt-and-braces: the grid's cache (socketKeys.matrix/tenantMatrix)
 * holds sparse documents and this hook never reads from it, and re-fetching
 * also narrows the last-write-wins window the PRD accepts to the duration of
 * one request.
 *
 * If `apply` throws (typically a MutationError from lib/socket/mutate, e.g. a
 * binding that no longer resolves to exactly one entry), nothing is written
 * and the error propagates to the caller (mutateAsync rejects / onError fires).
 */
export function useSocketMutation(): UseMutationResult<
  void,
  Error,
  SocketMutationInput
> {
  const queryClient = useQueryClient();

  return useMutation<void, Error, SocketMutationInput>({
    mutationFn: async ({ target, apply }) => {
      if (target.source === "template") {
        const fresh = await templatesService.getById(target.id);
        const socket = apply(fresh.attributes.socket);
        await templatesService.update(target.id, {
          ...fresh.attributes,
          socket,
        });
        return;
      }
      const fresh = await tenantsService.getTenantConfigurationById(target.id);
      const socket = apply(fresh.attributes.socket);
      await tenantsService.updateTenantConfiguration(fresh, { socket });
    },
    onSuccess: (_data, { target }) => {
      void queryClient.invalidateQueries({ queryKey: socketKeys.all });
      if (target.source === "template") {
        void queryClient.invalidateQueries({
          queryKey: templateKeys.detail(target.id),
        });
        void queryClient.invalidateQueries({ queryKey: templateKeys.lists() });
      } else {
        void queryClient.invalidateQueries({
          queryKey: tenantKeys.configDetail(target.id),
        });
        void queryClient.invalidateQueries({
          queryKey: tenantKeys.configLists(),
        });
      }
    },
  });
}
