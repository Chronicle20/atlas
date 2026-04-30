import { vi, type Mocked } from 'vitest';

import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act } from 'react';
import { factoryService } from '@/services/api/factory.service';
import type { CreateFromPresetPayload } from '@/services/api/factory.service';
import { useCreateCharacterFromPreset } from '../useCharacterFromPresetMutation';
import { characterKeys } from '../useCharacters';
import type { Tenant } from '@/types/models/tenant';
import type { ReactNode } from 'react';

// Mock the factory service
vi.mock('@/services/api/factory.service', () => ({
  factoryService: {
    createFromPreset: vi.fn(),
    checkNameValidity: vi.fn(),
  },
}));

const mockFactoryService = factoryService as Mocked<typeof factoryService>;

const mockTenant: Tenant = {
  id: 'tenant-123',
  attributes: {
    name: 'Test Tenant',
    region: 'GMS',
    majorVersion: 83,
    minorVersion: 1,
  },
};

const mockPayload: CreateFromPresetPayload = {
  presetId: 'preset-abc',
  accountId: 42,
  worldId: 0,
  name: 'NewHero',
};

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  const TestWrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  TestWrapper.displayName = 'TestWrapper';
  return { queryClient, TestWrapper };
}

describe('useCreateCharacterFromPreset', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should call createFromPreset with the correct payload on success', async () => {
    const mockResponse = { transactionId: 'txn-001' };
    mockFactoryService.createFromPreset.mockResolvedValue(mockResponse);

    const { TestWrapper } = createWrapper();
    const { result } = renderHook(
      () => useCreateCharacterFromPreset(mockTenant),
      { wrapper: TestWrapper },
    );

    await act(async () => {
      result.current.mutate(mockPayload);
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(mockFactoryService.createFromPreset).toHaveBeenCalledWith(mockTenant, mockPayload);
    expect(result.current.data).toEqual(mockResponse);
  });

  it('should invalidate character list queries on success', async () => {
    const mockResponse = { transactionId: 'txn-002' };
    mockFactoryService.createFromPreset.mockResolvedValue(mockResponse);

    const { queryClient, TestWrapper } = createWrapper();
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');

    const { result } = renderHook(
      () => useCreateCharacterFromPreset(mockTenant),
      { wrapper: TestWrapper },
    );

    await act(async () => {
      result.current.mutate(mockPayload);
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: characterKeys.lists(),
    });
  });

  it('should expose the error on mutation failure', async () => {
    const error = new Error('Creation failed');
    mockFactoryService.createFromPreset.mockRejectedValue(error);

    const { TestWrapper } = createWrapper();
    const { result } = renderHook(
      () => useCreateCharacterFromPreset(mockTenant),
      { wrapper: TestWrapper },
    );

    await act(async () => {
      result.current.mutate(mockPayload);
    });

    await waitFor(() => {
      expect(result.current.isError).toBe(true);
    });

    expect(result.current.error).toBe(error);
  });
});
