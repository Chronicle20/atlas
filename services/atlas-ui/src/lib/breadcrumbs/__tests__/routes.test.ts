/**
 * Tests for route configuration mapping
 */

import {
  findRouteConfig,
  matchesPattern,
  extractParams,
  getRouteHierarchy,
  getBreadcrumbsFromRoute,
  buildHrefFromPattern,
  getRoutesByEntityType,
  routeRequiresAuth,
  isRouteHidden,
  getChildRoutes,
  validateRouteConfig,
  ROUTE_PATTERNS,
  ROUTE_CONFIGS,
  RouteConfig,
} from '../routes';

describe('Route Configuration', () => {
  describe('findRouteConfig', () => {
    it('should find exact route matches', () => {
      const config = findRouteConfig('/tenants');
      expect(config).toBeTruthy();
      expect(config?.label).toBe('Tenants');
    });

    it('should find dynamic route matches', () => {
      const config = findRouteConfig('/tenants/123/properties');
      expect(config).toBeTruthy();
      expect(config?.label).toBe('Properties');
      expect(config?.parent).toBe('/tenants/[id]');
    });

    it('should return null for unknown routes', () => {
      const config = findRouteConfig('/unknown/route');
      expect(config).toBeNull();
    });
  });

  describe('matchesPattern', () => {
    it('should match static patterns', () => {
      expect(matchesPattern('/tenants', '/tenants')).toBe(true);
      expect(matchesPattern('/tenants', '/templates')).toBe(false);
    });

    it('should match dynamic patterns', () => {
      expect(matchesPattern('/tenants/123', '/tenants/[id]')).toBe(true);
      expect(matchesPattern('/tenants/abc-123', '/tenants/[id]')).toBe(true);
      expect(matchesPattern('/tenants', '/tenants/[id]')).toBe(false);
    });

    it('should match nested dynamic patterns', () => {
      expect(matchesPattern('/tenants/123/properties', '/tenants/[id]/properties')).toBe(true);
      expect(matchesPattern('/npcs/456/conversations', '/npcs/[id]/conversations')).toBe(true);
    });
  });

  describe('extractParams', () => {
    it('should extract single parameter', () => {
      const params = extractParams('/tenants/123', '/tenants/[id]');
      expect(params).toEqual({ id: '123' });
    });

    it('should extract multiple parameters', () => {
      // If we had a pattern like '/users/[userId]/posts/[postId]'
      const params = extractParams('/npcs/123', '/npcs/[id]');
      expect(params).toEqual({ id: '123' });
    });

    it('should return empty object for non-matching patterns', () => {
      const params = extractParams('/tenants/123/properties', '/tenants/[id]');
      expect(params).toEqual({});
    });

    it('should handle UUID parameters', () => {
      const uuid = '550e8400-e29b-41d4-a716-446655440000';
      const params = extractParams(`/tenants/${uuid}`, '/tenants/[id]');
      expect(params).toEqual({ id: uuid });
    });
  });

  describe('getRouteHierarchy', () => {
    it('should return hierarchy for root route', () => {
      const hierarchy = getRouteHierarchy('/');
      expect(hierarchy).toHaveLength(1);
      expect(hierarchy[0].pattern).toBe('/');
    });

    it('should return hierarchy for nested route', () => {
      const hierarchy = getRouteHierarchy('/tenants/123/properties');
      expect(hierarchy.length).toBeGreaterThan(1);
      expect(hierarchy.map(h => h.pattern)).toContain('/');
      expect(hierarchy.map(h => h.pattern)).toContain('/tenants');
      expect(hierarchy.map(h => h.pattern)).toContain('/tenants/[id]');
    });

    it('should return empty array for unknown routes', () => {
      const hierarchy = getRouteHierarchy('/unknown/route');
      expect(hierarchy).toEqual([]);
    });
  });

  describe('getBreadcrumbsFromRoute', () => {
    it('should generate breadcrumbs for static routes', () => {
      const breadcrumbs = getBreadcrumbsFromRoute('/tenants');
      expect(breadcrumbs.length).toBeGreaterThan(0);
      expect(breadcrumbs.some(b => b.label === 'Home')).toBe(true);
      expect(breadcrumbs.some(b => b.label === 'Tenants')).toBe(true);
    });

    it('should generate breadcrumbs for dynamic routes', () => {
      const breadcrumbs = getBreadcrumbsFromRoute('/tenants/123');
      expect(breadcrumbs.length).toBeGreaterThan(0);
      expect(breadcrumbs.some(b => b.entityType === 'tenant')).toBe(true);
    });

    it('should mark the last breadcrumb as current page', () => {
      const breadcrumbs = getBreadcrumbsFromRoute('/tenants/123/properties');
      const lastBreadcrumb = breadcrumbs[breadcrumbs.length - 1];
      expect(lastBreadcrumb.isCurrentPage).toBe(true);
    });
  });

  describe('buildHrefFromPattern', () => {
    it('should build href with parameters', () => {
      const href = buildHrefFromPattern('/tenants/[id]/properties', { id: '123' });
      expect(href).toBe('/tenants/123/properties');
    });

    it('should handle multiple parameters', () => {
      const href = buildHrefFromPattern('/tenants/[id]', { id: 'abc-123' });
      expect(href).toBe('/tenants/abc-123');
    });

    it('should return pattern if no matching params', () => {
      const href = buildHrefFromPattern('/tenants/[id]', {});
      expect(href).toBe('/tenants/[id]');
    });
  });

  describe('getRoutesByEntityType', () => {
    it('should return routes for entity type', () => {
      const routes = getRoutesByEntityType('tenant');
      expect(routes.length).toBeGreaterThan(0);
      expect(routes.every(r => r.entityType === 'tenant')).toBe(true);
    });

    it('should return empty array for unknown entity type', () => {
      const routes = getRoutesByEntityType('unknown');
      expect(routes).toEqual([]);
    });
  });

  describe('routeRequiresAuth', () => {
    it('should return false for routes without auth requirement', () => {
      expect(routeRequiresAuth('/tenants')).toBe(false);
    });
  });

  describe('isRouteHidden', () => {
    it('should return false for visible routes', () => {
      expect(isRouteHidden('/tenants')).toBe(false);
    });
  });

  describe('getChildRoutes', () => {
    it('should return child routes for parent', () => {
      const children = getChildRoutes('/tenants/[id]');
      expect(children.length).toBeGreaterThan(0);
      expect(children.every(c => c.parent === '/tenants/[id]')).toBe(true);
    });

    it('should return empty array for routes without children', () => {
      const children = getChildRoutes('/tenants/[id]/properties');
      expect(children).toEqual([]);
    });
  });

  describe('validateRouteConfig', () => {
    it('should validate correct route config', () => {
      const config: RouteConfig = {
        pattern: '/test',
        label: 'Test',
        parent: '/',
      };
      expect(validateRouteConfig(config)).toBe(true);
    });

    it('should reject config without pattern', () => {
      const config: RouteConfig = {
        pattern: '',
        label: 'Test',
      };
      expect(validateRouteConfig(config)).toBe(false);
    });

    it('should reject config without label', () => {
      const config: RouteConfig = {
        pattern: '/test',
        label: '',
      };
      expect(validateRouteConfig(config)).toBe(false);
    });
  });

  describe('ROUTE_PATTERNS constants', () => {
    it('should have all expected route patterns', () => {
      expect(ROUTE_PATTERNS.HOME).toBe('/');
      expect(ROUTE_PATTERNS.TENANTS).toBe('/tenants');
      expect(ROUTE_PATTERNS.TENANT_DETAIL).toBe('/tenants/[id]');
      expect(ROUTE_PATTERNS.TENANT_PROPERTIES).toBe('/tenants/[id]/properties');
    });
  });

  describe('Route configuration completeness', () => {
    it('should have configurations for all main entity routes', () => {
      const mainRoutes = ['/accounts', '/characters', '/guilds', '/npcs', '/templates', '/tenants'];
      mainRoutes.forEach(route => {
        const config = findRouteConfig(route);
        expect(config).toBeTruthy();
        expect(config?.label).toBeTruthy();
      });
    });

    it('should have detail routes for all main entities', () => {
      const detailPatterns = [
        '/accounts/[id]', 
        '/characters/[id]', 
        '/guilds/[id]', 
        '/npcs/[id]', 
        '/templates/[id]', 
        '/tenants/[id]'
      ];
      
      detailPatterns.forEach(pattern => {
        const config = ROUTE_CONFIGS.find(c => c.pattern === pattern);
        expect(config).toBeTruthy();
        expect(config?.entityType).toBeTruthy();
      });
    });

    it('should have proper parent-child relationships', () => {
      // Check that all parent references exist
      ROUTE_CONFIGS.forEach(config => {
        if (config.parent) {
          const parentConfig = ROUTE_CONFIGS.find(c => c.pattern === config.parent);
          expect(parentConfig).toBeTruthy();
        }
      });
    });

    it('should have unique patterns', () => {
      const patterns = ROUTE_CONFIGS.map(c => c.pattern);
      const uniquePatterns = new Set(patterns);
      expect(patterns.length).toBe(uniquePatterns.size);
    });
  });
});