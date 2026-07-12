import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { ReactNode } from 'react';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useRestoreBaseline } from '@/lib/hooks/api/useBaseline';
import { baselineService } from '@/services/api/baseline.service';
import { dataStatusKey } from '@/lib/hooks/api/useSeed';
import type { Tenant } from '@/types/models/tenant';

vi.mock('@/services/api/baseline.service', () => ({
  baselineService: {
    restore: vi.fn(),
    publish: vi.fn(),
  },
}));

const mockTenant: Tenant = {
  id: 'tenant-1',
  attributes: {
    name: 'Tenant 1',
    region: 'GMS',
    majorVersion: 83,
    minorVersion: 1,
  },
};

function makeWrapper(qc: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
  };
}

describe('useRestoreBaseline', () => {
  beforeEach(() => vi.clearAllMocks());

  it('rejects when tenant is null', async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { result } = renderHook(() => useRestoreBaseline(null), { wrapper: makeWrapper(qc) });
    result.current.mutate({
      region: 'GMS',
      majorVersion: 83,
      minorVersion: 1,
      tenantId: 't1',
    });
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.message).toMatch(/tenant is not yet resolved/);
    expect(baselineService.restore).not.toHaveBeenCalled();
  });

  it('invalidates dataStatus query on success', async () => {
    (baselineService.restore as ReturnType<typeof vi.fn>).mockResolvedValue(undefined);
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const invalidateSpy = vi.spyOn(qc, 'invalidateQueries');
    const { result } = renderHook(() => useRestoreBaseline(mockTenant), {
      wrapper: makeWrapper(qc),
    });
    result.current.mutate({
      region: 'GMS',
      majorVersion: 83,
      minorVersion: 1,
      tenantId: mockTenant.id,
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: dataStatusKey(mockTenant.id) });
  });

  it('forwards body to baselineService.restore with the tenant', async () => {
    (baselineService.restore as ReturnType<typeof vi.fn>).mockResolvedValue(undefined);
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { result } = renderHook(() => useRestoreBaseline(mockTenant), {
      wrapper: makeWrapper(qc),
    });
    const body = {
      region: 'GMS',
      majorVersion: 83,
      minorVersion: 1,
      tenantId: mockTenant.id,
    };
    result.current.mutate(body);
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(baselineService.restore).toHaveBeenCalledWith(mockTenant, body);
  });
});
