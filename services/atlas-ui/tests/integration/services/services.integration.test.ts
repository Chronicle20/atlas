/**
 * @jest-environment jsdom
 * 
 * Integration tests for API Services
 * Tests real service workflows and API client interactions
 */

import { tenantsService } from '@/services/api/tenants.service';
import { accountsService } from '@/services/api/accounts.service';
import { charactersService } from '@/services/api/characters.service';
import { api } from '@/lib/api/client';
import type { Tenant } from '@/types/models/tenant';
import type { Account } from '@/types/models/account';
import type { Character } from '@/types/models/character';

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

describe('Services Integration Tests', () => {
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

  const mockAccount: Account = {
    id: 'account-1',
    attributes: {
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
    }
  };

  const mockCharacter: Character = {
    id: 'char-1',
    attributes: {
      accountId: 'account-1',
      name: 'TestCharacter',
      level: 50,
      job: 'Warrior',
      str: 100,
      dex: 80,
      int: 60,
      luk: 70,
      hp: 5000,
      mp: 2000,
      maxHp: 5000,
      maxMp: 2000,
      exp: 12345678,
      meso: 1000000,
      mapId: 100000000,
      spawnPoint: 0,
      gender: 0,
      skinColor: 0,
      face: 20000,
      hair: 30000,
      world: 0,
      rank: 100,
      rankMove: 0,
      jobRank: 50,
      jobRankMove: 0,
      createdAt: '2023-01-01T00:00:00Z',
      updatedAt: '2023-01-01T00:00:00Z'
    }
  };

  describe('Tenants Service Integration', () => {
    it('should handle tenant management workflow', async () => {
      const tenantBasic = {
        id: 'tenant-123',
        attributes: {
          name: 'Test Tenant',
          region: 'GMS',
          majorVersion: 83,
          minorVersion: 1,
        }
      };

      // 1. Get all tenants
      mockApi.getList.mockResolvedValueOnce([tenantBasic]);
      
      const tenants = await tenantsService.getAllTenants();
      
      expect(mockApi.getList).toHaveBeenCalled();
      expect(tenants).toHaveLength(1);
      expect(tenants[0].id).toBe('tenant-123');

      // 2. Get specific tenant
      mockApi.getOne.mockResolvedValueOnce(tenantBasic);
      
      const tenant = await tenantsService.getTenantById('tenant-123');
      
      expect(mockApi.getOne).toHaveBeenCalled();
      expect(tenant.attributes.name).toBe('Test Tenant');

      // 3. Delete tenant
      mockApi.delete.mockResolvedValueOnce(undefined);
      
      await tenantsService.deleteTenant('tenant-123');
      
      expect(mockApi.delete).toHaveBeenCalled();
    });

    it('should handle tenant configuration management', async () => {
      const tenantConfig = {
        id: 'tenant-123',
        attributes: {
          region: 'GMS',
          majorVersion: 83,
          minorVersion: 1,
          usesPin: true,
          characters: { templates: [] },
          worlds: [],
          properties: {},
          handlers: [],
          writers: [],
        }
      };

      // Get tenant configuration
      mockApi.getOne.mockResolvedValueOnce(tenantConfig);
      
      const config = await tenantsService.getTenantConfigurationById('tenant-123');
      
      expect(mockApi.getOne).toHaveBeenCalled();
      expect(config.attributes.usesPin).toBe(true);

      // Get all configurations
      mockApi.getList.mockResolvedValueOnce([tenantConfig]);
      
      const configs = await tenantsService.getAllTenantConfigurations();
      
      expect(mockApi.getList).toHaveBeenCalled();
      expect(configs).toHaveLength(1);
      expect(configs[0].id).toBe('tenant-123');
    });
  });

  describe('Accounts Service Integration', () => {
    it('should handle account management with tenant context', async () => {
      // 1. Get all accounts for tenant
      mockApi.getList.mockResolvedValueOnce([mockAccount]);
      
      const accounts = await accountsService.getAllAccounts(mockTenant);
      
      expect(mockApi.setTenant).toHaveBeenCalledWith(mockTenant);
      expect(mockApi.getList).toHaveBeenCalledWith('/api/accounts', expect.any(Object));
      expect(accounts).toHaveLength(1);

      // 2. Get specific account
      mockApi.getOne.mockResolvedValueOnce(mockAccount);
      
      const account = await accountsService.getAccountById(mockTenant, 'account-1');
      
      expect(mockApi.getOne).toHaveBeenCalledWith('/api/accounts/account-1', expect.any(Object));
      expect(account.attributes.name).toBe('testuser');

      // 3. Check account existence
      mockApi.getOne.mockResolvedValueOnce(mockAccount);
      
      const exists = await accountsService.accountExists(mockTenant, 'account-1');
      
      expect(exists).toBe(true);
    });

    it('should handle batch session termination', async () => {
      const accountIds = ['account-1', 'account-2', 'account-3'];

      // Mock successful terminations
      mockApi.delete.mockResolvedValue(undefined);

      const result = await accountsService.terminateMultipleSessions(mockTenant, accountIds);

      expect(mockApi.setTenant).toHaveBeenCalledWith(mockTenant);
      expect(mockApi.delete).toHaveBeenCalledTimes(3);
      expect(result.successful).toEqual(accountIds);
      expect(result.failed).toHaveLength(0);
    });

    it('should calculate account statistics', async () => {
      const accounts = [
        { ...mockAccount, attributes: { ...mockAccount.attributes, loggedIn: 1, characterSlots: 3 } },
        { ...mockAccount, id: 'account-2', attributes: { ...mockAccount.attributes, banned: true, characterSlots: 5 } },
        { ...mockAccount, id: 'account-3', attributes: { ...mockAccount.attributes, loggedIn: 0, characterSlots: 2 } },
      ];

      mockApi.getList.mockResolvedValueOnce(accounts);

      const stats = await accountsService.getAccountStats(mockTenant);

      expect(stats).toEqual({
        total: 3,
        loggedIn: 1,
        banned: 1,
        totalCharacterSlots: 10,
        averageCharacterSlots: 10 / 3,
      });
    });
  });

  describe('Characters Service Integration', () => {
    it('should handle character operations with tenant context', async () => {
      // 1. Get all characters for tenant
      mockApi.getList.mockResolvedValueOnce([mockCharacter]);
      
      const characters = await charactersService.getAll(mockTenant);
      
      expect(mockApi.setTenant).toHaveBeenCalledWith(mockTenant);
      expect(mockApi.getList).toHaveBeenCalled();
      expect(characters).toHaveLength(1);

      // 2. Get specific character
      mockApi.getOne.mockResolvedValueOnce(mockCharacter);
      
      const character = await charactersService.getById(mockTenant, 'char-1');
      
      expect(mockApi.getOne).toHaveBeenCalled();
      expect(character.attributes.name).toBe('TestCharacter');

      // 3. Fetch character (alternative method)
      mockApi.getOne.mockResolvedValueOnce(mockCharacter);
      const fetchedCharacter = await charactersService.fetchCharacter(mockTenant, 'char-1');
      
      expect(fetchedCharacter.attributes.name).toBe('TestCharacter');
    });
  });

  describe('Cross-Service Integration', () => {
    it('should handle complete tenant-account-character workflow', async () => {
      // 1. Get tenant
      const tenantBasic = {
        id: 'tenant-123',
        attributes: {
          name: 'Test Tenant',
          region: 'GMS',
          majorVersion: 83,
          minorVersion: 1,
        }
      };
      
      mockApi.getOne.mockResolvedValueOnce(tenantBasic);
      const tenant = await tenantsService.getTenantById('tenant-123');
      
      // 2. Get accounts for tenant
      mockApi.getList.mockResolvedValueOnce([mockAccount]);
      const accounts = await accountsService.getAllAccounts(mockTenant);
      
      // 3. Get characters for tenant
      mockApi.getList.mockResolvedValueOnce([mockCharacter]);
      const characters = await charactersService.getAll(mockTenant);
      
      expect(tenant.id).toBe('tenant-123');
      expect(accounts).toHaveLength(1);
      expect(characters).toHaveLength(1);
      expect(characters[0].attributes.accountId).toBe(accounts[0].id);

      // Verify tenant context was set for each operation
      expect(mockApi.setTenant).toHaveBeenCalledTimes(2);
      expect(mockApi.setTenant).toHaveBeenCalledWith(mockTenant);
    });
  });

  describe('Error Handling Integration', () => {
    it('should handle API errors gracefully', async () => {
      // Test 404 error
      const notFoundError = new Error('Not found');
      (notFoundError as any).status = 404;
      mockApi.getOne.mockRejectedValueOnce(notFoundError);

      await expect(tenantsService.getTenantById('non-existent')).rejects.toThrow('Not found');

      // Test 500 error
      const serverError = new Error('Internal server error');
      (serverError as any).status = 500;
      mockApi.getList.mockRejectedValueOnce(serverError);

      await expect(accountsService.getAllAccounts(mockTenant)).rejects.toThrow('Internal server error');
    });

    it('should handle partial failures in batch operations', async () => {
      const accountIds = ['account-1', 'account-2', 'account-3'];

      // Mock mixed success/failure
      mockApi.delete
        .mockResolvedValueOnce(undefined) // account-1 success
        .mockRejectedValueOnce(new Error('Session not found')) // account-2 fails
        .mockResolvedValueOnce(undefined); // account-3 success

      const result = await accountsService.terminateMultipleSessions(mockTenant, accountIds);

      expect(result.successful).toEqual(['account-1', 'account-3']);
      expect(result.failed).toEqual([{ id: 'account-2', error: 'Session not found' }]);
    });
  });

  describe('Caching Integration', () => {
    it('should properly handle cache configurations', async () => {
      const tenantBasic = {
        id: 'tenant-123',
        attributes: {
          name: 'Test Tenant',
          region: 'GMS',
          majorVersion: 83,
          minorVersion: 1,
        }
      };

      mockApi.getList.mockResolvedValueOnce([tenantBasic]);

      await tenantsService.getAllTenants({
        useCache: true,
        cacheConfig: {
          ttl: 5 * 60 * 1000, // 5 minutes
          staleWhileRevalidate: true,
        }
      });

      expect(mockApi.getList).toHaveBeenCalledWith('/api/tenants', expect.objectContaining({
        cacheConfig: expect.objectContaining({
          ttl: 5 * 60 * 1000,
          staleWhileRevalidate: true,
        }),
      }));
    });
  });

  describe('Concurrent Operations Integration', () => {
    it('should handle concurrent API operations', async () => {
      const tenantBasic = {
        id: 'tenant-123',
        attributes: {
          name: 'Test Tenant',
          region: 'GMS',
          majorVersion: 83,
          minorVersion: 1,
        }
      };

      // Set up concurrent operations
      mockApi.getOne
        .mockResolvedValueOnce(tenantBasic)
        .mockResolvedValueOnce(mockAccount)
        .mockResolvedValueOnce(mockCharacter);

      // Execute concurrent requests
      const [tenant, account, character] = await Promise.all([
        tenantsService.getTenantById('tenant-1'),
        accountsService.getAccountById(mockTenant, 'account-1'),
        charactersService.getById(mockTenant, 'char-1'),
      ]);

      expect(tenant.id).toBe('tenant-123');
      expect(account.id).toBe('account-1');
      expect(character.id).toBe('char-1');
      expect(mockApi.getOne).toHaveBeenCalledTimes(3);
    });
  });
});