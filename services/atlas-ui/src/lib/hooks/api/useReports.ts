/**
 * React Query hooks for player-report review.
 */

import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationResult,
  type UseQueryResult,
} from "@tanstack/react-query";
import {
  reportsService,
  type ReportQueryOptions,
} from "@/services/api/reports.service";
import type { Report, ReportStatus } from "@/types/models/report";
import type { Tenant } from "@/types/models/tenant";
import type { ServiceOptions } from "@/lib/api/query-params";

export const reportKeys = {
  all: ["reports"] as const,
  lists: () => [...reportKeys.all, "list"] as const,
  list: (tenant: Tenant | null, options?: ReportQueryOptions) =>
    [...reportKeys.lists(), tenant?.id ?? "no-tenant", options] as const,
  details: () => [...reportKeys.all, "detail"] as const,
  detail: (tenant: Tenant | null, id: string) =>
    [...reportKeys.details(), tenant?.id ?? "no-tenant", id] as const,
};

export function useReports(
  tenant: Tenant | null,
  options?: ReportQueryOptions,
): UseQueryResult<Report[], Error> {
  return useQuery({
    queryKey: reportKeys.list(tenant, options),
    queryFn: () => reportsService.getAllReports(options),
    enabled: !!tenant?.id,
    gcTime: 5 * 60 * 1000,
  });
}

export function useReport(
  tenant: Tenant | null,
  id: string,
  options?: ServiceOptions,
): UseQueryResult<Report, Error> {
  return useQuery({
    queryKey: reportKeys.detail(tenant, id),
    queryFn: () => reportsService.getReportById(id, options),
    enabled: !!tenant?.id && !!id,
    gcTime: 5 * 60 * 1000,
  });
}

export function useUpdateReportStatus(): UseMutationResult<
  void,
  Error,
  { id: string; status: ReportStatus }
> {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, status }) =>
      reportsService.updateReportStatus(id, status),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: reportKeys.all });
    },
  });
}

export function useInvalidateReports() {
  const queryClient = useQueryClient();
  return {
    invalidateAll: () =>
      queryClient.invalidateQueries({ queryKey: reportKeys.all }),
  };
}
