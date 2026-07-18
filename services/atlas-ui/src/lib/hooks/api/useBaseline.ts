import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  baselineService,
  type BaselineRestoreInput,
} from "@/services/api/baseline.service";
import type { Tenant } from "@/types/models/tenant";
import { dataStatusKey } from "@/lib/hooks/api/useSeed";

export const useRestoreBaseline = (tenant: Tenant | null) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: BaselineRestoreInput) => {
      if (!tenant) {
        throw new Error("useRestoreBaseline: tenant is not yet resolved");
      }
      return baselineService.restore(tenant, body);
    },
    onSuccess: () => {
      if (!tenant) return;
      void qc.invalidateQueries({ queryKey: dataStatusKey(tenant.id) });
    },
  });
};
