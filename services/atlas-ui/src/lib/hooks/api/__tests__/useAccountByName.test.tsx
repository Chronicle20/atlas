import { vi, type Mocked } from 'vitest';

import { renderHook, waitFor, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { accountsService } from '@/services/api/accounts.service';
import { useAccountByName, accountByNameKeys } from '../useAccountByName';
import type { Account } from '@/types/models/account';
import type { Tenant } from '@/types/models/tenant';
import type { ReactNode } from 'react';

// Mock the accounts service
vi.mock('@/services/api/accounts.service', () => ({
  accountsService: {
    getAllAccounts: vi.fn(),
    getAccountById: vi.fn(),
    accountExists: vi.fn(),
    searchAccountsByName: vi.fn(),
    getLoggedInAccounts: vi.fn(),
    terminateAccountSession: vi.fn(),
    deleteAccount: vi.fn(),
    getAccountStats: vi.fn(),
    terminateMultipleSessions: vi.fn(),
  },
}));

const mockAccountsService = accountsService as Mocked<typeof accountsService>;

const mockTenant: Tenant = {
  id: 'tenant-123',
  attributes: {
    name: 'Test Tenant',
    region: 'GMS',
    majorVersion: 83,
    minorVersion: 1,
  },
};

const mockAccount: Account = {
  id: 'account-1',
  attributes: {
    name: 'chronicle',
    pin: '1234',
    pic: '5678',
    pinAttempts: 0,
    picAttempts: 0,
    loggedIn: 0,
    lastLogin: Date.now(),
    gender: 0,
    tos: true,
    language: 'en',
    country: 'US',
    characterSlots: 6,
  },
};

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  });

  const TestWrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  TestWrapper.displayName = 'TestWrapper';
  return TestWrapper;
}

describe('useAccountByName', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useRealTimers();
  });

  describe('query key generation', () => {
    it('should generate correct query keys', () => {
      expect(accountByNameKeys.all).toEqual(['account', 'by-name']);
      expect(accountByNameKeys.query('tenant-1', 'chronicle')).toEqual([
        'account',
        'by-name',
        'tenant-1',
        'chronicle',
      ]);
    });
  });

  describe('basic fetching', () => {
    it('should fetch accounts by name successfully on first poll', async () => {
      mockAccountsService.getAllAccounts.mockResolvedValue([mockAccount]);

      const { result } = renderHook(
        () => useAccountByName(mockTenant, 'chronicle'),
        { wrapper: createWrapper() },
      );

      await waitFor(() => {
        expect(result.current.query.isSuccess).toBe(true);
      });

      expect(mockAccountsService.getAllAccounts).toHaveBeenCalledWith({ name: 'chronicle' });
      expect(result.current.query.data).toEqual([mockAccount]);
      expect(result.current.timedOut).toBe(false);
    });

    it('should not fetch when name is empty', () => {
      const { result } = renderHook(
        () => useAccountByName(mockTenant, ''),
        { wrapper: createWrapper() },
      );

      expect(result.current.query.fetchStatus).toBe('idle');
      expect(mockAccountsService.getAllAccounts).not.toHaveBeenCalled();
    });

    it('should handle fetch errors', async () => {
      const error = new Error('Network error');
      mockAccountsService.getAllAccounts.mockRejectedValue(error);

      const { result } = renderHook(
        () => useAccountByName(mockTenant, 'chronicle'),
        { wrapper: createWrapper() },
      );

      await waitFor(() => {
        expect(result.current.query.isError).toBe(true);
      });

      expect(result.current.query.error).toBe(error);
    });
  });

  describe('polling behaviour', () => {
    it('should not poll when pollUntilFound is not set', async () => {
      mockAccountsService.getAllAccounts.mockResolvedValue([]);

      const { result } = renderHook(
        () => useAccountByName(mockTenant, 'chronicle'),
        { wrapper: createWrapper() },
      );

      await waitFor(() => {
        expect(result.current.query.isSuccess).toBe(true);
      });

      // Only called once — no polling
      expect(mockAccountsService.getAllAccounts).toHaveBeenCalledTimes(1);
      expect(result.current.timedOut).toBe(false);
    });
  });

  describe('timeout behaviour', () => {
    it('should set timedOut to true after the timeout elapses', async () => {
      vi.useFakeTimers();
      mockAccountsService.getAllAccounts.mockResolvedValue([]);

      const { result } = renderHook(
        () =>
          useAccountByName(mockTenant, 'chronicle', {
            pollUntilFound: true,
            timeoutMs: 100,
            intervalMs: 50,
          }),
        { wrapper: createWrapper() },
      );

      expect(result.current.timedOut).toBe(false);

      await act(async () => {
        vi.advanceTimersByTime(150);
      });

      expect(result.current.timedOut).toBe(true);

      vi.useRealTimers();
    });

    it('should keep timedOut false when account is found before timeout', async () => {
      mockAccountsService.getAllAccounts.mockResolvedValue([mockAccount]);

      const { result } = renderHook(
        () =>
          useAccountByName(mockTenant, 'chronicle', {
            pollUntilFound: true,
            timeoutMs: 30000,
            intervalMs: 1000,
          }),
        { wrapper: createWrapper() },
      );

      await waitFor(() => {
        expect(result.current.query.isSuccess).toBe(true);
      });

      // Account found immediately — timeout watchdog should not have fired
      expect(result.current.timedOut).toBe(false);
    });
  });
});
