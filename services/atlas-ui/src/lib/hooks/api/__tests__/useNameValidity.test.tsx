import { vi, type Mocked } from 'vitest';

import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { factoryService } from '@/services/api/factory.service';
import type { NameValidityResponse } from '@/services/api/factory.service';
import { useNameValidity, nameValidityKeys } from '../useNameValidity';
import type { Tenant } from '@/types/models/tenant';
import type { ReactNode } from 'react';

// Mock the factory service
vi.mock('@/services/api/factory.service', () => ({
  factoryService: {
    createFromPreset: vi.fn(),
    checkNameValidity: vi.fn(),
  },
}));

// Mock the debounce hook to return the value synchronously (bypass delay in tests)
vi.mock('@/lib/utils/debounce', () => ({
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  useDebounce: (value: any, _delay: number) => value,
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
  return TestWrapper;
}

describe('useNameValidity', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('query key generation', () => {
    it('should generate correct query keys', () => {
      expect(nameValidityKeys.all).toEqual(['name-validity']);
      expect(nameValidityKeys.query('tenant-1', 0, 'Hero')).toEqual([
        'name-validity',
        'tenant-1',
        0,
        'Hero',
      ]);
    });
  });

  describe('when name is shorter than 3 characters', () => {
    it('should be disabled and not call the service', () => {
      const { result } = renderHook(
        () => useNameValidity(mockTenant, 'ab', 0),
        { wrapper: createWrapper() },
      );

      expect(result.current.fetchStatus).toBe('idle');
      expect(mockFactoryService.checkNameValidity).not.toHaveBeenCalled();
    });

    it('should be disabled for empty name', () => {
      const { result } = renderHook(
        () => useNameValidity(mockTenant, '', 0),
        { wrapper: createWrapper() },
      );

      expect(result.current.fetchStatus).toBe('idle');
      expect(mockFactoryService.checkNameValidity).not.toHaveBeenCalled();
    });
  });

  describe('when name is 3 or more characters', () => {
    it('should call checkNameValidity and return result', async () => {
      const mockResponse: NameValidityResponse = { valid: true };
      mockFactoryService.checkNameValidity.mockResolvedValue(mockResponse);

      const { result } = renderHook(
        () => useNameValidity(mockTenant, 'Hero', 0),
        { wrapper: createWrapper() },
      );

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockFactoryService.checkNameValidity).toHaveBeenCalledWith(mockTenant, 'Hero', 0);
      expect(result.current.data).toEqual(mockResponse);
    });

    it('should report invalid names', async () => {
      const mockResponse: NameValidityResponse = {
        valid: false,
        reason: 'duplicate',
        detail: 'Name already taken',
      };
      mockFactoryService.checkNameValidity.mockResolvedValue(mockResponse);

      const { result } = renderHook(
        () => useNameValidity(mockTenant, 'TakenName', 0),
        { wrapper: createWrapper() },
      );

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockResponse);
    });
  });

  describe('when enabled option is false', () => {
    it('should not run the query even with a valid name', () => {
      const { result } = renderHook(
        () => useNameValidity(mockTenant, 'ValidName', 0, { enabled: false }),
        { wrapper: createWrapper() },
      );

      expect(result.current.fetchStatus).toBe('idle');
      expect(mockFactoryService.checkNameValidity).not.toHaveBeenCalled();
    });
  });
});
