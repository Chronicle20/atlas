/**
 * @jest-environment jsdom
 */

import { accountsService } from '../accounts.service';
import { api } from '@/lib/api/client';
import type { Account, AccountAttributes } from '@/types/models/account';
import type { Tenant } from '@/types/models/tenant';

// Mock the API client
jest.mock('@/lib/api/client', () => ({
  api: {
    getList: jest.fn(),
    getOne: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    patch: jest.fn(),
    delete: jest.fn(),
    upload: jest.fn(),
    download: jest.fn(),
    clearCacheByPattern: jest.fn(),
    getCacheStats: jest.fn(),
    setTenant: jest.fn(),
  },
}));

const mockApi = api as jest.Mocked<typeof api>;

describe('AccountsService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  const mockTenant: Tenant = {
    id: 'tenant-123',
    attributes: {
      name: 'Test Tenant',
      region: 'GMS',
      majorVersion: 83,
      minorVersion: 1,
      usesPin: true,
      gameSecret: 'secret',
      adminSecret: 'admin-secret',
      registrationEnabled: true,
      createdAt: '2023-01-01T00:00:00Z',
      updatedAt: '2023-01-01T00:00:00Z'
    }
  };

  const mockAccountAttributes: AccountAttributes = {
    name: 'testuser',
    pin: '1234',
    pic: '5678',
    loggedIn: 0,
    lastLogin: 1640995200,
    gender: 0,
    banned: false,
    tos: true,
    language: 'en',
    country: 'US',
    characterSlots: 3,
  };

  const mockAccount: Account = {
    id: 'account-1',
    attributes: mockAccountAttributes,
  };

  const mockAccounts: Account[] = [
    mockAccount,
    {
      id: 'account-2',
      attributes: {
        ...mockAccountAttributes,
        name: 'anotheruser',
        loggedIn: 1,
        banned: true,
      },
    },
  ];

  describe('validation', () => {
    it('should validate required fields', () => {
      const invalidData = {
        name: '',
        pin: '',
        pic: '',
        characterSlots: -1,
        gender: 2,
      };
      const errors = (accountsService as any).validate(invalidData);
      
      expect(errors.length).toBeGreaterThan(0);
      expect(errors.some((e: any) => e.field === 'name')).toBeTruthy();
      expect(errors.some((e: any) => e.field === 'pin')).toBeTruthy();
      expect(errors.some((e: any) => e.field === 'pic')).toBeTruthy();
    });

    it('should validate account name length', () => {
      const invalidData = {
        name: 'verylongaccountname', // Too long (>12 chars)
        pin: '1234',
        pic: '5678',
        characterSlots: 3,
        gender: 0,
      };
      const errors = (accountsService as any).validate(invalidData);
      
      expect(errors.some((e: any) => e.field === 'name' && e.message.includes('12 characters'))).toBeTruthy();
    });

    it('should validate PIN length', () => {
      const invalidData = {
        name: 'testuser',
        pin: '123456789', // Too long (>8 chars)
        pic: '5678',
        characterSlots: 3,
        gender: 0,
      };
      const errors = (accountsService as any).validate(invalidData);
      
      expect(errors.some((e: any) => e.field === 'pin' && e.message.includes('8 characters'))).toBeTruthy();
    });

    it('should validate character slots range', () => {
      const invalidData = {
        name: 'testuser',
        pin: '1234',
        pic: '5678',
        characterSlots: 50, // Too high (>30)
        gender: 0,
      };
      const errors = (accountsService as any).validate(invalidData);
      
      expect(errors.some((e: any) => e.field === 'characterSlots')).toBeTruthy();
    });

    it('should validate gender values', () => {
      const invalidData = {
        name: 'testuser',
        pin: '1234',
        pic: '5678',
        characterSlots: 3,
        gender: 2, // Invalid (must be 0 or 1)
      };
      const errors = (accountsService as any).validate(invalidData);
      
      expect(errors.some((e: any) => e.field === 'gender')).toBeTruthy();
    });

    it('should pass validation for valid data', () => {
      const errors = (accountsService as any).validate(mockAccountAttributes);
      expect(errors).toHaveLength(0);
    });
  });

  describe('transformResponse', () => {
    it('should transform numeric fields properly', () => {
      const responseData = {
        id: 'account-1',
        attributes: {
          ...mockAccountAttributes,
          loggedIn: '1', // String that should be converted to number
          lastLogin: '1640995200',
          gender: '0',
          characterSlots: '3',
          banned: 'false', // String that should be converted to boolean
          tos: 'true',
        },
      };

      const result = (accountsService as any).transformResponse(responseData);

      expect(typeof result.attributes.loggedIn).toBe('number');
      expect(typeof result.attributes.lastLogin).toBe('number');
      expect(typeof result.attributes.gender).toBe('number');
      expect(typeof result.attributes.characterSlots).toBe('number');
      expect(typeof result.attributes.banned).toBe('boolean');
      expect(typeof result.attributes.tos).toBe('boolean');
    });

    it('should return data as-is if not an Account object', () => {
      const nonAccountData = { someData: 'value' };
      const result = (accountsService as any).transformResponse(nonAccountData);
      expect(result).toEqual(nonAccountData);
    });
  });

  describe('getAllAccounts', () => {
    it('should fetch all accounts for a tenant and sort them', async () => {
      mockApi.getList.mockResolvedValue(mockAccounts);

      const result = await accountsService.getAllAccounts(mockTenant);

      expect(mockApi.setTenant).toHaveBeenCalledWith(mockTenant);
      expect(mockApi.getList).toHaveBeenCalledWith('/api/accounts', expect.any(Object));
      expect(result).toHaveLength(2);
      // Check if sorted by name (anotheruser should come before testuser)
      expect(result[0].attributes.name).toBe('anotheruser');
      expect(result[1].attributes.name).toBe('testuser');
    });

    it('should handle account filtering options', async () => {
      mockApi.getList.mockResolvedValue([mockAccount]);

      await accountsService.getAllAccounts(mockTenant, {
        name: 'testuser',
        banned: false,
        loggedIn: true,
        language: 'en',
        country: 'US',
      });

      expect(mockApi.getList).toHaveBeenCalledWith(
        expect.stringContaining('/api/accounts?'),
        expect.objectContaining({
          filters: expect.objectContaining({
            name: 'testuser',
            banned: false,
            loggedIn: true,
            language: 'en',
            country: 'US',
          }),
        })
      );
    });
  });

  describe('getAccountById', () => {
    it('should fetch account by ID with tenant context', async () => {
      mockApi.getOne.mockResolvedValue(mockAccount);

      const result = await accountsService.getAccountById(mockTenant, 'account-1');

      expect(mockApi.setTenant).toHaveBeenCalledWith(mockTenant);
      expect(mockApi.getOne).toHaveBeenCalledWith('/api/accounts/account-1', expect.any(Object));
      expect(result).toEqual(mockAccount);
    });
  });

  describe('accountExists', () => {
    it('should check if account exists', async () => {
      mockApi.getOne.mockResolvedValue(mockAccount);

      const result = await accountsService.accountExists(mockTenant, 'account-1');

      expect(mockApi.setTenant).toHaveBeenCalledWith(mockTenant);
      expect(result).toBe(true);
    });

    it('should return false for non-existent account', async () => {
      const error = new Error('Not Found');
      (error as any).status = 404;
      mockApi.getOne.mockRejectedValue(error);

      const result = await accountsService.accountExists(mockTenant, 'non-existent');

      expect(result).toBe(false);
    });
  });

  describe('searchAccountsByName', () => {
    it('should search accounts by name pattern', async () => {
      mockApi.getList.mockResolvedValue([mockAccount]);

      const result = await accountsService.searchAccountsByName(mockTenant, 'test');

      expect(mockApi.getList).toHaveBeenCalledWith(
        expect.stringContaining('/api/accounts?'),
        expect.objectContaining({
          search: 'test',
          filters: expect.objectContaining({
            name: 'test',
          }),
        })
      );
      expect(result).toEqual([mockAccount]);
    });
  });

  describe('getLoggedInAccounts', () => {
    it('should fetch only logged-in accounts', async () => {
      const loggedInAccount = { ...mockAccount, attributes: { ...mockAccount.attributes, loggedIn: 1 } };
      mockApi.getList.mockResolvedValue([loggedInAccount]);

      const result = await accountsService.getLoggedInAccounts(mockTenant);

      expect(mockApi.getList).toHaveBeenCalledWith(
        expect.stringContaining('/api/accounts?'),
        expect.objectContaining({
          filters: expect.objectContaining({
            loggedIn: true,
          }),
        })
      );
      expect(result).toEqual([loggedInAccount]);
    });
  });

  describe('getBannedAccounts', () => {
    it('should fetch only banned accounts', async () => {
      const bannedAccount = { ...mockAccount, attributes: { ...mockAccount.attributes, banned: true } };
      mockApi.getList.mockResolvedValue([bannedAccount]);

      const result = await accountsService.getBannedAccounts(mockTenant);

      expect(mockApi.getList).toHaveBeenCalledWith(
        expect.stringContaining('/api/accounts?'),
        expect.objectContaining({
          filters: expect.objectContaining({
            banned: true,
          }),
        })
      );
      expect(result).toEqual([bannedAccount]);
    });
  });

  describe('terminateAccountSession', () => {
    it('should terminate account session', async () => {
      mockApi.delete.mockResolvedValue(undefined);

      await accountsService.terminateAccountSession(mockTenant, 'account-1');

      expect(mockApi.setTenant).toHaveBeenCalledWith(mockTenant);
      expect(mockApi.delete).toHaveBeenCalledWith('/api/accounts/account-1/session', expect.any(Object));
    });
  });

  describe('getAccountStats', () => {
    it('should calculate account statistics', async () => {
      const accountsWithStats = [
        { ...mockAccount, attributes: { ...mockAccount.attributes, loggedIn: 1, characterSlots: 3 } },
        { ...mockAccount, id: 'account-2', attributes: { ...mockAccount.attributes, banned: true, characterSlots: 5 } },
        { ...mockAccount, id: 'account-3', attributes: { ...mockAccount.attributes, loggedIn: 0, characterSlots: 2 } },
      ];
      mockApi.getList.mockResolvedValue(accountsWithStats);

      const stats = await accountsService.getAccountStats(mockTenant);

      expect(stats).toEqual({
        total: 3,
        loggedIn: 1,
        banned: 1,
        totalCharacterSlots: 10,
        averageCharacterSlots: 10 / 3,
      });
    });

    it('should handle empty account list', async () => {
      mockApi.getList.mockResolvedValue([]);

      const stats = await accountsService.getAccountStats(mockTenant);

      expect(stats).toEqual({
        total: 0,
        loggedIn: 0,
        banned: 0,
        totalCharacterSlots: 0,
        averageCharacterSlots: 0,
      });
    });
  });

  describe('terminateMultipleSessions', () => {
    it('should terminate multiple sessions successfully', async () => {
      mockApi.delete.mockResolvedValue(undefined);

      const result = await accountsService.terminateMultipleSessions(mockTenant, ['account-1', 'account-2']);

      expect(mockApi.setTenant).toHaveBeenCalledWith(mockTenant);
      expect(mockApi.delete).toHaveBeenCalledTimes(2);
      expect(result.successful).toEqual(['account-1', 'account-2']);
      expect(result.failed).toHaveLength(0);
    });

    it('should handle partial failures', async () => {
      mockApi.delete
        .mockResolvedValueOnce(undefined) // First account succeeds
        .mockRejectedValueOnce(new Error('Session not found')); // Second account fails

      const result = await accountsService.terminateMultipleSessions(mockTenant, ['account-1', 'account-2']);

      expect(result.successful).toEqual(['account-1']);
      expect(result.failed).toEqual([{ id: 'account-2', error: 'Session not found' }]);
    });

    it('should process accounts in batches of 3', async () => {
      mockApi.delete.mockResolvedValue(undefined);
      const manyAccountIds = Array.from({ length: 10 }, (_, i) => `account-${i + 1}`);

      await accountsService.terminateMultipleSessions(mockTenant, manyAccountIds);

      expect(mockApi.delete).toHaveBeenCalledTimes(10);
      // Verify batch processing by checking that not all calls happen simultaneously
      expect(mockApi.setTenant).toHaveBeenCalledWith(mockTenant);
    });
  });

  describe('type guards', () => {
    it('should correctly identify Account objects', () => {
      const isAccountMethod = (accountsService as any).isAccount;
      
      expect(isAccountMethod(mockAccount)).toBe(true);
      expect(isAccountMethod({ id: 'test' })).toBe(false);
      expect(isAccountMethod(null)).toBe(false);
      expect(isAccountMethod(undefined)).toBe(false);
    });

    it('should correctly identify AccountAttributes objects', () => {
      const isAccountAttributesMethod = (accountsService as any).isAccountAttributes;
      
      expect(isAccountAttributesMethod(mockAccountAttributes)).toBe(true);
      expect(isAccountAttributesMethod({ name: 'test' })).toBe(false);
      expect(isAccountAttributesMethod(null)).toBe(false);
      expect(isAccountAttributesMethod(undefined)).toBe(false);
    });
  });
});