import { useMutation } from '@tanstack/react-query';
import { baselineService, type BaselineRestoreInput } from '@/services/api/baseline.service';
import type { Tenant } from '@/types/models/tenant';

export const useRestoreBaseline = (tenant: Tenant) =>
  useMutation({
    mutationFn: (body: BaselineRestoreInput) => baselineService.restore(tenant, body),
  });

export const usePublishBaseline = (tenant: Tenant) =>
  useMutation({
    mutationFn: (input: { region: string; majorVersion: number; minorVersion: number }) =>
      baselineService.publish(tenant, input.region, input.majorVersion, input.minorVersion),
  });
