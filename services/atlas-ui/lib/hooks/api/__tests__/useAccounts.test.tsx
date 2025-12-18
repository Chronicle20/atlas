/**
 * Tests for account React Query hooks
 */

import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act } from 'react';
import { accountsService } from '@/services/api/accounts.service';
import type { Account } from '@/types/models/account';
import type { Tenant } from '@/types/models/tenant';
import {
  useAccounts,
  useAccount,
  useAccountExists,
  useAccountSearch,
  useLoggedInAccounts,
  useBannedAccounts,
  useAccountStats,
  useTerminateAccountSession,
  useTerminateMultipleSessions,
  useInvalidateAccounts,
  usePrefetchAccounts,
} from '../useAccounts';

// Mock the accounts service
jest.mock('@/services/api/accounts.service');
const mockAccountsService = accountsService as jest.Mocked<typeof accountsService>;

// Test data
const mockTenant: Tenant = {
  id: 'tenant-1',
  attributes: {
    regionName: 'Test Region',
    majorVersion: 83,
    minorVersion: 1,
    locale: 'en_US',
  },
};

const mockAccount: Account = {
  id: 'account-1',
  attributes: {
    name: 'testuser',
    pin: '1234',
    pic: '5678',
    loggedIn: 1,
    lastLogin: Date.now(),
    gender: 0,
    banned: false,
    tos: true,
    language: 'en',
    country: 'US',
    characterSlots: 6,
  },
};

const mockAccounts: Account[] = [
  mockAccount,
  {
    id: 'account-2',
    attributes: {
      name: 'banneduser',
      pin: '8765',
      pic: '4321',
      loggedIn: 0,
      lastLogin: Date.now() - 86400000,
      gender: 1,
      banned: true,
      tos: true,
      language: 'en',
      country: 'US',
      characterSlots: 3,
    },
  },
];

const mockStats = {
  total: 2,
  loggedIn: 1,
  banned: 1,
  totalCharacterSlots: 9,
  averageCharacterSlots: 4.5,
};

// Test wrapper component
function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
        staleTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
  });

  const Wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  Wrapper.displayName = 'ReactQueryWrapper';
  return Wrapper;
}

describe('useAccounts hooks', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('useAccounts', () => {
    it('should fetch all accounts for a tenant', async () => {
      mockAccountsService.getAllAccounts.mockResolvedValue(mockAccounts);

      const { result } = renderHook(
        () => useAccounts(mockTenant),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockAccounts);
      expect(mockAccountsService.getAllAccounts).toHaveBeenCalledWith(mockTenant, undefined);
    });

    it('should not fetch when tenant is not provided', () => {
      const { result } = renderHook(
        () => useAccounts(null as unknown as Tenant),
        { wrapper: createWrapper() }
      );

      expect(result.current.status).toBe('pending');
      expect(mockAccountsService.getAllAccounts).not.toHaveBeenCalled();
    });

    it('should pass query options to service', async () => {
      const options = { name: 'test', banned: false };
      mockAccountsService.getAllAccounts.mockResolvedValue(mockAccounts);

      renderHook(
        () => useAccounts(mockTenant, options),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(mockAccountsService.getAllAccounts).toHaveBeenCalledWith(mockTenant, options);
      });
    });
  });

  describe('useAccount', () => {
    it('should fetch specific account by ID', async () => {
      mockAccountsService.getAccountById.mockResolvedValue(mockAccount);

      const { result } = renderHook(
        () => useAccount(mockTenant, 'account-1'),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockAccount);
      expect(mockAccountsService.getAccountById).toHaveBeenCalledWith(mockTenant, 'account-1', undefined);
    });

    it('should not fetch when tenant or ID is not provided', () => {
      const { result } = renderHook(
        () => useAccount(mockTenant, ''),
        { wrapper: createWrapper() }
      );

      expect(result.current.status).toBe('pending');
      expect(mockAccountsService.getAccountById).not.toHaveBeenCalled();
    });
  });

  describe('useAccountExists', () => {
    it('should check if account exists', async () => {
      mockAccountsService.accountExists.mockResolvedValue(true);

      const { result } = renderHook(
        () => useAccountExists(mockTenant, 'account-1'),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toBe(true);
      expect(mockAccountsService.accountExists).toHaveBeenCalledWith(mockTenant, 'account-1', undefined);
    });
  });

  describe('useAccountSearch', () => {
    it('should search accounts by name pattern', async () => {
      mockAccountsService.searchAccountsByName.mockResolvedValue([mockAccount]);

      const { result } = renderHook(
        () => useAccountSearch(mockTenant, 'test'),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual([mockAccount]);
      expect(mockAccountsService.searchAccountsByName).toHaveBeenCalledWith(mockTenant, 'test', undefined);
    });

    it('should not search with empty pattern', () => {
      const { result } = renderHook(
        () => useAccountSearch(mockTenant, ''),
        { wrapper: createWrapper() }
      );

      expect(result.current.status).toBe('pending');
      expect(mockAccountsService.searchAccountsByName).not.toHaveBeenCalled();
    });
  });

  describe('useLoggedInAccounts', () => {
    it('should fetch logged-in accounts', async () => {
      const loggedInAccounts = [mockAccount];
      mockAccountsService.getLoggedInAccounts.mockResolvedValue(loggedInAccounts);

      const { result } = renderHook(
        () => useLoggedInAccounts(mockTenant),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(loggedInAccounts);
      expect(mockAccountsService.getLoggedInAccounts).toHaveBeenCalledWith(mockTenant, undefined);
    });
  });

  describe('useBannedAccounts', () => {
    it('should fetch banned accounts', async () => {
      const bannedAccounts = [mockAccounts[1]];
      mockAccountsService.getBannedAccounts.mockResolvedValue(bannedAccounts);

      const { result } = renderHook(
        () => useBannedAccounts(mockTenant),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(bannedAccounts);
      expect(mockAccountsService.getBannedAccounts).toHaveBeenCalledWith(mockTenant, undefined);
    });
  });

  describe('useAccountStats', () => {
    it('should fetch account statistics', async () => {
      mockAccountsService.getAccountStats.mockResolvedValue(mockStats);

      const { result } = renderHook(
        () => useAccountStats(mockTenant),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockStats);
      expect(mockAccountsService.getAccountStats).toHaveBeenCalledWith(mockTenant, undefined);
    });
  });

  describe('useTerminateAccountSession', () => {
    it('should terminate account session', async () => {
      mockAccountsService.terminateAccountSession.mockResolvedValue();

      const { result } = renderHook(
        () => useTerminateAccountSession(),
        { wrapper: createWrapper() }
      );

      await act(async () => {
        result.current.mutate({
          tenant: mockTenant,
          accountId: 'account-1',
        });
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockAccountsService.terminateAccountSession).toHaveBeenCalledWith(
        mockTenant,
        'account-1',
        undefined
      );
    });

    it('should handle termination errors', async () => {
      const error = new Error('Session termination failed');
      mockAccountsService.terminateAccountSession.mockRejectedValue(error);

      const { result } = renderHook(
        () => useTerminateAccountSession(),
        { wrapper: createWrapper() }
      );

      await act(async () => {
        result.current.mutate({
          tenant: mockTenant,
          accountId: 'account-1',
        });
      });

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });

      expect(result.current.error).toBe(error);
    });
  });

  describe('useTerminateMultipleSessions', () => {
    it('should terminate multiple account sessions', async () => {
      const mockResult = {
        successful: ['account-1', 'account-2'],
        failed: [],
      };
      mockAccountsService.terminateMultipleSessions.mockResolvedValue(mockResult);

      const { result } = renderHook(
        () => useTerminateMultipleSessions(),
        { wrapper: createWrapper() }
      );

      await act(async () => {
        result.current.mutate({
          tenant: mockTenant,
          accountIds: ['account-1', 'account-2'],
        });
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockResult);
      expect(mockAccountsService.terminateMultipleSessions).toHaveBeenCalledWith(
        mockTenant,
        ['account-1', 'account-2'],
        undefined
      );
    });

    it('should handle partial failures', async () => {
      const mockResult = {
        successful: ['account-1'],
        failed: [{ id: 'account-2', error: 'Already logged out' }],
      };
      mockAccountsService.terminateMultipleSessions.mockResolvedValue(mockResult);

      const { result } = renderHook(
        () => useTerminateMultipleSessions(),
        { wrapper: createWrapper() }
      );

      await act(async () => {
        result.current.mutate({
          tenant: mockTenant,
          accountIds: ['account-1', 'account-2'],
        });
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockResult);
    });
  });

  describe('useInvalidateAccounts', () => {
    it('should provide invalidation functions', () => {
      const { result } = renderHook(
        () => useInvalidateAccounts(),
        { wrapper: createWrapper() }
      );

      expect(typeof result.current.invalidateAll).toBe('function');
      expect(typeof result.current.invalidateLists).toBe('function');
      expect(typeof result.current.invalidateList).toBe('function');
      expect(typeof result.current.invalidateAccount).toBe('function');
      expect(typeof result.current.invalidateLoggedIn).toBe('function');
      expect(typeof result.current.invalidateBanned).toBe('function');
      expect(typeof result.current.invalidateStats).toBe('function');
      expect(typeof result.current.invalidateAllForTenant).toBe('function');
    });
  });

  describe('usePrefetchAccounts', () => {
    it('should provide prefetch functions', () => {
      const { result } = renderHook(
        () => usePrefetchAccounts(),
        { wrapper: createWrapper() }
      );

      expect(typeof result.current.prefetchAccounts).toBe('function');
      expect(typeof result.current.prefetchAccount).toBe('function');
      expect(typeof result.current.prefetchLoggedIn).toBe('function');
      expect(typeof result.current.prefetchStats).toBe('function');
    });
  });
});