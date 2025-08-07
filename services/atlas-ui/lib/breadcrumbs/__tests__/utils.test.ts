import {
  parsePathname,
  isDynamicSegment,
  getEntityType,
  buildBreadcrumbPath,
  getParentBreadcrumb,
  isValidRoute,
  capitalizeFirst,
  getBreadcrumbKey,
  filterVisibleBreadcrumbs,
  BreadcrumbSegment,
} from '../utils';

describe('Breadcrumb Utils', () => {
  describe('parsePathname', () => {
    it('should parse root path correctly', () => {
      const result = parsePathname('/');
      expect(result).toHaveLength(1);
      expect(result[0]).toEqual({
        segment: '',
        label: 'Home',
        href: '/',
        dynamic: false,
        isCurrentPage: true,
      });
    });

    it('should parse simple static path correctly', () => {
      const result = parsePathname('/accounts');
      expect(result).toHaveLength(2);
      expect(result[0].label).toBe('Home');
      expect(result[1]).toEqual({
        segment: 'accounts',
        label: 'Accounts',
        href: '/accounts',
        dynamic: false,
        isCurrentPage: true,
      });
    });

    it('should parse dynamic character path correctly', () => {
      const result = parsePathname('/characters/123');
      expect(result).toHaveLength(3);
      expect(result[2]).toEqual({
        segment: '123',
        label: '123',
        href: '/characters/123',
        dynamic: true,
        isCurrentPage: true,
        entityId: '123',
        entityType: 'character',
      });
    });

    it('should parse nested path correctly', () => {
      const result = parsePathname('/npcs/456/conversations');
      expect(result).toHaveLength(4);
      expect(result[1].label).toBe('NPCs');
      expect(result[2].dynamic).toBe(true);
      expect(result[2].entityType).toBe('npc');
      expect(result[3].label).toBe('Conversations');
    });

    it('should parse tenant path correctly', () => {
      const result = parsePathname('/tenants/uuid-123/handlers');
      expect(result).toHaveLength(4);
      expect(result[2].entityType).toBe('tenant');
      expect(result[3].label).toBe('Socket Handlers');
    });
  });

  describe('isDynamicSegment', () => {
    it('should identify numeric IDs as dynamic', () => {
      expect(isDynamicSegment('123', '/characters/123')).toBe(true);
      expect(isDynamicSegment('456', '/npcs/456/shop')).toBe(true);
    });

    it('should identify UUIDs as dynamic', () => {
      const uuid = '12345678-1234-1234-1234-123456789012';
      expect(isDynamicSegment(uuid, `/tenants/${uuid}`)).toBe(true);
    });

    it('should not identify static segments as dynamic', () => {
      expect(isDynamicSegment('accounts', '/accounts')).toBe(false);
      expect(isDynamicSegment('conversations', '/npcs/123/conversations')).toBe(false);
    });

    it('should not identify IDs in wrong contexts as dynamic', () => {
      expect(isDynamicSegment('123', '/unknown/123')).toBe(false);
    });
  });

  describe('getEntityType', () => {
    it('should return correct entity types', () => {
      expect(getEntityType('123', '/characters/123')).toBe('character');
      expect(getEntityType('456', '/npcs/456')).toBe('npc');
      expect(getEntityType('789', '/guilds/789')).toBe('guild');
      expect(getEntityType('uuid', '/tenants/uuid')).toBe('tenant');
      expect(getEntityType('template-id', '/templates/template-id')).toBe('template');
    });

    it('should return null for unknown patterns', () => {
      expect(getEntityType('123', '/unknown/123')).toBe(null);
    });
  });

  describe('buildBreadcrumbPath', () => {
    const longPath: BreadcrumbSegment[] = [
      { segment: '', label: 'Home', href: '/', dynamic: false, isCurrentPage: false },
      { segment: 'tenants', label: 'Tenants', href: '/tenants', dynamic: false, isCurrentPage: false },
      { segment: 'uuid-1', label: 'Tenant 1', href: '/tenants/uuid-1', dynamic: true, isCurrentPage: false },
      { segment: 'character', label: 'Character', href: '/tenants/uuid-1/character', dynamic: false, isCurrentPage: false },
      { segment: 'templates', label: 'Templates', href: '/tenants/uuid-1/character/templates', dynamic: false, isCurrentPage: true },
    ];

    it('should return all items if under limit', () => {
      const result = buildBreadcrumbPath(longPath, { maxItems: 10 });
      expect(result).toHaveLength(5);
    });

    it('should truncate items when over limit', () => {
      const result = buildBreadcrumbPath(longPath, { maxItems: 3, showEllipsis: false });
      expect(result).toHaveLength(3);
      // Should show last 3 items: Tenant 1, Character, Templates
      expect(result[0].label).toBe('Tenant 1');
      expect(result[1].label).toBe('Character');
      expect(result[2].label).toBe('Templates');
    });

    it('should add ellipsis when truncating', () => {
      const result = buildBreadcrumbPath(longPath, { maxItems: 4, showEllipsis: true });
      expect(result.some(item => item.label === '...')).toBe(true);
    });
  });

  describe('getParentBreadcrumb', () => {
    const breadcrumbs: BreadcrumbSegment[] = [
      { segment: '', label: 'Home', href: '/', dynamic: false, isCurrentPage: false },
      { segment: 'accounts', label: 'Accounts', href: '/accounts', dynamic: false, isCurrentPage: true },
    ];

    it('should return parent breadcrumb', () => {
      const parent = getParentBreadcrumb(breadcrumbs);
      expect(parent?.label).toBe('Home');
    });

    it('should return null for root or single item', () => {
      expect(getParentBreadcrumb([breadcrumbs[0]])).toBe(null);
      expect(getParentBreadcrumb([])).toBe(null);
    });
  });

  describe('isValidRoute', () => {
    it('should validate known routes', () => {
      expect(isValidRoute('/')).toBe(true);
      expect(isValidRoute('/accounts')).toBe(true);
      expect(isValidRoute('/characters/123')).toBe(true);
      expect(isValidRoute('/npcs/456/shop')).toBe(true);
    });

    it('should reject unknown routes', () => {
      expect(isValidRoute('/unknown')).toBe(false);
      expect(isValidRoute('/invalid/path')).toBe(false);
    });
  });

  describe('capitalizeFirst', () => {
    it('should capitalize first letter', () => {
      expect(capitalizeFirst('hello')).toBe('Hello');
      expect(capitalizeFirst('world')).toBe('World');
    });

    it('should handle edge cases', () => {
      expect(capitalizeFirst('')).toBe('');
      expect(capitalizeFirst('a')).toBe('A');
    });
  });

  describe('getBreadcrumbKey', () => {
    it('should generate unique keys', () => {
      const segments1: BreadcrumbSegment[] = [
        { segment: 'accounts', label: 'Accounts', href: '/accounts', dynamic: false, isCurrentPage: true },
      ];
      const segments2: BreadcrumbSegment[] = [
        { segment: '123', label: 'Character', href: '/characters/123', dynamic: true, isCurrentPage: true },
      ];

      expect(getBreadcrumbKey(segments1)).toBe('accounts:static');
      expect(getBreadcrumbKey(segments2)).toBe('123:dynamic');
      expect(getBreadcrumbKey(segments1)).not.toBe(getBreadcrumbKey(segments2));
    });
  });

  describe('filterVisibleBreadcrumbs', () => {
    const segments: BreadcrumbSegment[] = [
      { segment: '', label: 'Home', href: '/', dynamic: false, isCurrentPage: false },
      { segment: 'admin', label: 'Admin', href: '/admin', dynamic: false, isCurrentPage: false },
      { segment: 'accounts', label: 'Accounts', href: '/admin/accounts', dynamic: false, isCurrentPage: true },
    ];

    it('should filter out hidden routes', () => {
      const result = filterVisibleBreadcrumbs(segments, { hiddenRoutes: ['admin'] });
      expect(result).toHaveLength(2);
      expect(result.find(s => s.segment === 'admin')).toBeUndefined();
    });

    it('should return all items when no filters', () => {
      const result = filterVisibleBreadcrumbs(segments);
      expect(result).toHaveLength(3);
    });
  });
});