/**
 * Tests for dynamic label resolver service with caching
 */

import {
  resolveEntityLabel,
  resolveEntityLabels,
  preloadEntityLabels,
  invalidateEntityLabels,
  clearResolverCache,
  getResolverCacheStats,
  getEntityTypeFromRoute,
  isValidEntityType,
  EntityType,
  resolverCache,
  type ResolverOptions,
} from '../resolvers';

// Mock the services
jest.mock('@/services/api', () => ({
  accountsService: {
    getById: jest.fn(),
  },
  charactersService: {
    getById: jest.fn(),
  },
  guildsService: {
    getById: jest.fn(),
  },
  npcsService: {
    getById: jest.fn(),
  },
  templatesService: {
    getById: jest.fn(),
  },
  tenantsService: {
    getById: jest.fn(),
  },
}));

// Mock tenant for testing
const mockTenant = {
  id: 'test-tenant-id',
  type: 'tenant',
  attributes: {
    name: 'Test Tenant',
    region: 'test',
    majorVersion: 1,
    minorVersion: 0,
  },
};

describe('Entity Type Utilities', () => {
  describe('getEntityTypeFromRoute', () => {
    it('should identify account routes', () => {
      expect(getEntityTypeFromRoute('/accounts/123')).toBe(EntityType.ACCOUNT);
      expect(getEntityTypeFromRoute('/accounts/uuid-123/settings')).toBe(EntityType.ACCOUNT);
    });

    it('should identify character routes', () => {
      expect(getEntityTypeFromRoute('/characters/456')).toBe(EntityType.CHARACTER);
      expect(getEntityTypeFromRoute('/characters/456/inventory')).toBe(EntityType.CHARACTER);
    });

    it('should identify guild routes', () => {
      expect(getEntityTypeFromRoute('/guilds/789')).toBe(EntityType.GUILD);
      expect(getEntityTypeFromRoute('/guilds/789/members')).toBe(EntityType.GUILD);
    });

    it('should identify NPC routes', () => {
      expect(getEntityTypeFromRoute('/npcs/101112')).toBe(EntityType.NPC);
      expect(getEntityTypeFromRoute('/npcs/101112/conversations')).toBe(EntityType.NPC);
    });

    it('should identify template routes', () => {
      expect(getEntityTypeFromRoute('/templates/template-id')).toBe(EntityType.TEMPLATE);
      expect(getEntityTypeFromRoute('/templates/template-id/properties')).toBe(EntityType.TEMPLATE);
    });

    it('should identify tenant routes', () => {
      expect(getEntityTypeFromRoute('/tenants/tenant-123')).toBe(EntityType.TENANT);
      expect(getEntityTypeFromRoute('/tenants/tenant-123/settings')).toBe(EntityType.TENANT);
    });

    it('should return null for unknown routes', () => {
      expect(getEntityTypeFromRoute('/unknown/route')).toBe(null);
      expect(getEntityTypeFromRoute('/')).toBe(null);
      expect(getEntityTypeFromRoute('/settings')).toBe(null);
    });
  });

  describe('isValidEntityType', () => {
    it('should validate known entity types', () => {
      expect(isValidEntityType('account')).toBe(true);
      expect(isValidEntityType('character')).toBe(true);
      expect(isValidEntityType('guild')).toBe(true);
      expect(isValidEntityType('npc')).toBe(true);
      expect(isValidEntityType('template')).toBe(true);
      expect(isValidEntityType('tenant')).toBe(true);
    });

    it('should reject invalid entity types', () => {
      expect(isValidEntityType('invalid')).toBe(false);
      expect(isValidEntityType('user')).toBe(false);
      expect(isValidEntityType('')).toBe(false);
      expect(isValidEntityType('Account')).toBe(false); // case sensitive
    });
  });
});

describe('Entity Label Resolution', () => {
  beforeEach(() => {
    // Clear cache before each test
    clearResolverCache();
    jest.clearAllMocks();
  });

  describe('resolveEntityLabel', () => {
    it('should resolve character names successfully', async () => {
      const { charactersService } = await import('@/services/api');
      (charactersService.getById as jest.Mock).mockResolvedValue({
        id: '123',
        type: 'character',
        attributes: { name: 'TestCharacter' },
      });

      const result = await resolveEntityLabel(
        EntityType.CHARACTER,
        '123',
        mockTenant
      );

      expect(result).toEqual({
        label: 'TestCharacter',
        fromCache: false,
        resolvedAt: expect.any(Number),
        isFallback: false,
      });

      expect(charactersService.getById).toHaveBeenCalledWith(
        mockTenant,
        '123',
        expect.any(Object)
      );
    });

    it('should resolve guild names successfully', async () => {
      const { guildsService } = await import('@/services/api');
      (guildsService.getById as jest.Mock).mockResolvedValue({
        id: '456',
        type: 'guild',
        attributes: { name: 'TestGuild' },
      });

      const result = await resolveEntityLabel(
        EntityType.GUILD,
        '456',
        mockTenant
      );

      expect(result).toEqual({
        label: 'TestGuild',
        fromCache: false,
        resolvedAt: expect.any(Number),
        isFallback: false,
      });
    });

    it('should use fallback for failed resolution', async () => {
      const { charactersService } = await import('@/services/api');
      (charactersService.getById as jest.Mock).mockRejectedValue(
        new Error('Character not found')
      );

      const result = await resolveEntityLabel(
        EntityType.CHARACTER,
        '999',
        mockTenant,
        { fallback: 'Unknown Character' }
      );

      expect(result).toEqual({
        label: 'Unknown Character',
        fromCache: false,
        resolvedAt: expect.any(Number),
        isFallback: true,
      });
    });

    it('should use default fallback when no custom fallback provided', async () => {
      const { charactersService } = await import('@/services/api');
      (charactersService.getById as jest.Mock).mockRejectedValue(
        new Error('Character not found')
      );

      const result = await resolveEntityLabel(
        EntityType.CHARACTER,
        '999',
        mockTenant
      );

      expect(result).toEqual({
        label: 'Unknown',
        fromCache: false,
        resolvedAt: expect.any(Number),
        isFallback: true,
      });
    });

    it('should handle timeout scenarios', async () => {
      const { charactersService } = await import('@/services/api');
      (charactersService.getById as jest.Mock).mockImplementation(
        () => new Promise(resolve => setTimeout(resolve, 10000)) // 10s delay
      );

      const result = await resolveEntityLabel(
        EntityType.CHARACTER,
        '123',
        mockTenant,
        { timeout: 100 } // 100ms timeout
      );

      expect(result.isFallback).toBe(true);
      expect(result.label).toBe('Unknown');
    });
  });

  describe('Caching', () => {
    it('should cache successful resolutions', async () => {
      const { charactersService } = await import('@/services/api');
      (charactersService.getById as jest.Mock).mockResolvedValue({
        id: '123',
        type: 'character',
        attributes: { name: 'TestCharacter' },
      });

      // First call
      const result1 = await resolveEntityLabel(
        EntityType.CHARACTER,
        '123',
        mockTenant
      );
      expect(result1.fromCache).toBe(false);

      // Second call should use cache
      const result2 = await resolveEntityLabel(
        EntityType.CHARACTER,
        '123',
        mockTenant
      );
      expect(result2.fromCache).toBe(true);
      expect(result2.label).toBe('TestCharacter');

      // Should only call service once
      expect(charactersService.getById).toHaveBeenCalledTimes(1);
    });

    it('should respect cache disable option', async () => {
      const { charactersService } = await import('@/services/api');
      (charactersService.getById as jest.Mock).mockResolvedValue({
        id: '123',
        type: 'character',
        attributes: { name: 'TestCharacter' },
      });

      // First call with cache disabled
      const result1 = await resolveEntityLabel(
        EntityType.CHARACTER,
        '123',
        mockTenant,
        { useCache: false }
      );
      expect(result1.fromCache).toBe(false);

      // Second call with cache disabled should still fetch
      const result2 = await resolveEntityLabel(
        EntityType.CHARACTER,
        '123',
        mockTenant,
        { useCache: false }
      );
      expect(result2.fromCache).toBe(false);

      // Should call service twice
      expect(charactersService.getById).toHaveBeenCalledTimes(2);
    });

    it('should invalidate specific cache entries', async () => {
      const { charactersService } = await import('@/services/api');
      (charactersService.getById as jest.Mock).mockResolvedValue({
        id: '123',
        type: 'character',
        attributes: { name: 'TestCharacter' },
      });

      // Cache the result
      await resolveEntityLabel(EntityType.CHARACTER, '123', mockTenant);

      // Invalidate the cache
      invalidateEntityLabels(EntityType.CHARACTER, ['123'], mockTenant);

      // Next call should not use cache
      const result = await resolveEntityLabel(
        EntityType.CHARACTER,
        '123',
        mockTenant
      );
      expect(result.fromCache).toBe(false);

      // Should call service twice total
      expect(charactersService.getById).toHaveBeenCalledTimes(2);
    });

    it('should clear all cache when requested', async () => {
      const { charactersService } = await import('@/services/api');
      (charactersService.getById as jest.Mock).mockResolvedValue({
        id: '123',
        type: 'character',
        attributes: { name: 'TestCharacter' },
      });

      // Cache some results
      await resolveEntityLabel(EntityType.CHARACTER, '123', mockTenant);

      // Clear all cache
      clearResolverCache();

      // Next call should not use cache
      const result = await resolveEntityLabel(
        EntityType.CHARACTER,
        '123',
        mockTenant
      );
      expect(result.fromCache).toBe(false);

      // Should call service twice total
      expect(charactersService.getById).toHaveBeenCalledTimes(2);
    });
  });

  describe('Batch Resolution', () => {
    it('should resolve multiple entities', async () => {
      const { charactersService } = await import('@/services/api');
      (charactersService.getById as jest.Mock)
        .mockResolvedValueOnce({
          id: '123',
          type: 'character',
          attributes: { name: 'Character1' },
        })
        .mockResolvedValueOnce({
          id: '456',
          type: 'character',
          attributes: { name: 'Character2' },
        });

      const results = await resolveEntityLabels(
        EntityType.CHARACTER,
        ['123', '456'],
        mockTenant
      );

      expect(results).toEqual({
        '123': {
          label: 'Character1',
          fromCache: false,
          resolvedAt: expect.any(Number),
          isFallback: false,
        },
        '456': {
          label: 'Character2',
          fromCache: false,
          resolvedAt: expect.any(Number),
          isFallback: false,
        },
      });

      expect(charactersService.getById).toHaveBeenCalledTimes(2);
    });

    it('should handle mixed success and failure in batch', async () => {
      const { charactersService } = await import('@/services/api');
      (charactersService.getById as jest.Mock)
        .mockResolvedValueOnce({
          id: '123',
          type: 'character',
          attributes: { name: 'Character1' },
        })
        .mockRejectedValueOnce(new Error('Character not found'));

      const results = await resolveEntityLabels(
        EntityType.CHARACTER,
        ['123', '999'],
        mockTenant
      );

      expect(results['123'].isFallback).toBe(false);
      expect(results['123'].label).toBe('Character1');
      expect(results['999'].isFallback).toBe(true);
      expect(results['999'].label).toBe('Unknown');
    });
  });

  describe('Preloading', () => {
    it('should preload entities not in cache', async () => {
      const { charactersService } = await import('@/services/api');
      (charactersService.getById as jest.Mock).mockResolvedValue({
        id: '123',
        type: 'character',
        attributes: { name: 'TestCharacter' },
      });

      // Preload
      await preloadEntityLabels(
        EntityType.CHARACTER,
        ['123'],
        mockTenant
      );

      // Wait a bit for async resolution
      await new Promise(resolve => setTimeout(resolve, 10));

      // Verify it was cached
      const result = await resolveEntityLabel(
        EntityType.CHARACTER,
        '123',
        mockTenant
      );
      expect(result.fromCache).toBe(true);
    });

    it('should skip preloading for entities already in cache', async () => {
      const { charactersService } = await import('@/services/api');
      (charactersService.getById as jest.Mock).mockResolvedValue({
        id: '123',
        type: 'character',
        attributes: { name: 'TestCharacter' },
      });

      // First resolve to cache
      await resolveEntityLabel(EntityType.CHARACTER, '123', mockTenant);

      // Reset mock call count
      (charactersService.getById as jest.Mock).mockClear();

      // Preload should skip already cached entity
      await preloadEntityLabels(
        EntityType.CHARACTER,
        ['123'],
        mockTenant
      );

      // Should not make additional service calls
      expect(charactersService.getById).not.toHaveBeenCalled();
    });
  });

  describe('Cache Statistics', () => {
    it('should return cache statistics', async () => {
      const { charactersService } = await import('@/services/api');
      (charactersService.getById as jest.Mock).mockResolvedValue({
        id: '123',
        type: 'character',
        attributes: { name: 'TestCharacter' },
      });

      // Cache some data
      await resolveEntityLabel(EntityType.CHARACTER, '123', mockTenant);

      const stats = getResolverCacheStats();
      expect(stats).toHaveProperty('character');
      expect(stats.character.size).toBe(1);
      expect(stats.character.oldestEntry).toBeGreaterThan(0);
    });

    it('should return empty stats for empty cache', () => {
      clearResolverCache();
      const stats = getResolverCacheStats();
      expect(Object.keys(stats)).toHaveLength(0);
    });
  });
});

describe('Entity-Specific Resolution', () => {
  beforeEach(() => {
    clearResolverCache();
    jest.clearAllMocks();
  });

  it('should resolve NPC names with fallback to ID', async () => {
    const { npcsService } = await import('@/services/api');
    (npcsService.getById as jest.Mock).mockResolvedValue({
      id: '101112',
      type: 'npc',
      attributes: { name: 'Test NPC' },
    });

    const result = await resolveEntityLabel(
      EntityType.NPC,
      '101112',
      mockTenant
    );

    expect(result.label).toBe('Test NPC');
  });

  it('should resolve template names', async () => {
    const { templatesService } = await import('@/services/api');
    (templatesService.getById as jest.Mock).mockResolvedValue({
      id: 'template-123',
      type: 'template',
      attributes: { name: 'Test Template' },
    });

    const result = await resolveEntityLabel(
      EntityType.TEMPLATE,
      'template-123',
      mockTenant
    );

    expect(result.label).toBe('Test Template');
  });

  it('should resolve tenant names', async () => {
    const { tenantsService } = await import('@/services/api');
    (tenantsService.getById as jest.Mock).mockResolvedValue({
      id: 'tenant-123',
      type: 'tenant',
      attributes: { name: 'Test Tenant' },
    });

    const result = await resolveEntityLabel(
      EntityType.TENANT,
      'tenant-123',
      mockTenant
    );

    expect(result.label).toBe('Test Tenant');
  });

  it('should handle entities with missing name attribute', async () => {
    const { charactersService } = await import('@/services/api');
    (charactersService.getById as jest.Mock).mockResolvedValue({
      id: '123',
      type: 'character',
      attributes: {}, // No name attribute
    });

    const result = await resolveEntityLabel(
      EntityType.CHARACTER,
      '123',
      mockTenant
    );

    expect(result.label).toBe('Character 123');
    expect(result.isFallback).toBe(false);
  });

  it('should handle invalid entity type gracefully', async () => {
    const result = await resolveEntityLabel(
      'invalid' as EntityType,
      '123',
      mockTenant
    );

    expect(result.label).toBe('Unknown');
    expect(result.isFallback).toBe(true);
  });
});