/**
 * Route configuration mapping for breadcrumb navigation
 * Defines all application routes and their breadcrumb metadata
 */

import { BreadcrumbSegment } from './utils';

// Types for route configuration
export interface RouteConfig {
  /** Route pattern (e.g., '/tenants/[id]/properties') */
  pattern: string;
  /** Static label for the route */
  label: string;
  /** Whether this route requires authentication */
  requiresAuth?: boolean;
  /** Whether this route should be hidden from breadcrumbs */
  hidden?: boolean;
  /** Parent route pattern for hierarchical navigation */
  parent?: string;
  /** Entity type for dynamic routes */
  entityType?: string;
  /** Custom label resolver function */
  labelResolver?: (params: Record<string, string>) => string;
}

// Comprehensive route configuration for all application routes
export const ROUTE_CONFIGS: RouteConfig[] = [
  // Root route
  {
    pattern: '/',
    label: 'Home',
  },

  // Main entity list routes
  {
    pattern: '/accounts',
    label: 'Accounts',
    parent: '/',
  },
  {
    pattern: '/characters',
    label: 'Characters', 
    parent: '/',
  },
  {
    pattern: '/guilds',
    label: 'Guilds',
    parent: '/',
  },
  {
    pattern: '/npcs',
    label: 'NPCs',
    parent: '/',
  },
  {
    pattern: '/templates',
    label: 'Templates',
    parent: '/',
  },
  {
    pattern: '/tenants',
    label: 'Tenants',
    parent: '/',
  },

  // Account detail routes
  {
    pattern: '/accounts/[id]',
    label: 'Account Details',
    parent: '/accounts',
    entityType: 'account',
  },

  // Character detail routes
  {
    pattern: '/characters/[id]',
    label: 'Character Details',
    parent: '/characters',
    entityType: 'character',
  },

  // Guild routes
  {
    pattern: '/guilds/[id]',
    label: 'Guild Details',
    parent: '/guilds',
    entityType: 'guild',
  },

  // NPC routes
  {
    pattern: '/npcs/[id]',
    label: 'NPC Details',
    parent: '/npcs',
    entityType: 'npc',
  },
  {
    pattern: '/npcs/[id]/conversations',
    label: 'Conversations',
    parent: '/npcs/[id]',
  },
  {
    pattern: '/npcs/[id]/shop',
    label: 'Shop',
    parent: '/npcs/[id]',
  },

  // Template routes
  {
    pattern: '/templates/[id]',
    label: 'Template Details',
    parent: '/templates',
    entityType: 'template',
  },
  {
    pattern: '/templates/[id]/properties',
    label: 'Properties',
    parent: '/templates/[id]',
  },
  {
    pattern: '/templates/[id]/handlers',
    label: 'Socket Handlers',
    parent: '/templates/[id]',
  },
  {
    pattern: '/templates/[id]/writers',
    label: 'Socket Writers',
    parent: '/templates/[id]',
  },
  {
    pattern: '/templates/[id]/worlds',
    label: 'Worlds',
    parent: '/templates/[id]',
  },
  {
    pattern: '/templates/[id]/character',
    label: 'Character',
    parent: '/templates/[id]',
  },
  {
    pattern: '/templates/[id]/character/templates',
    label: 'Templates',
    parent: '/templates/[id]/character',
  },

  // Tenant routes
  {
    pattern: '/tenants/[id]',
    label: 'Tenant Details',
    parent: '/tenants',
    entityType: 'tenant',
  },
  {
    pattern: '/tenants/[id]/properties',
    label: 'Properties',
    parent: '/tenants/[id]',
  },
  {
    pattern: '/tenants/[id]/handlers',
    label: 'Socket Handlers',
    parent: '/tenants/[id]',
  },
  {
    pattern: '/tenants/[id]/writers',
    label: 'Socket Writers',
    parent: '/tenants/[id]',
  },
  {
    pattern: '/tenants/[id]/worlds',
    label: 'Worlds',
    parent: '/tenants/[id]',
  },
  {
    pattern: '/tenants/[id]/character',
    label: 'Character',
    parent: '/tenants/[id]',
  },
  {
    pattern: '/tenants/[id]/character/templates',
    label: 'Templates',
    parent: '/tenants/[id]/character',
  },
];

// Helper function to find route config by pathname
export function findRouteConfig(pathname: string): RouteConfig | null {
  // Direct match first
  const directMatch = ROUTE_CONFIGS.find(config => config.pattern === pathname);
  if (directMatch) return directMatch;

  // Pattern matching for dynamic routes
  for (const config of ROUTE_CONFIGS) {
    if (matchesPattern(pathname, config.pattern)) {
      return config;
    }
  }

  return null;
}

// Helper function to check if a pathname matches a route pattern
export function matchesPattern(pathname: string, pattern: string): boolean {
  // Convert pattern to regex
  const regexPattern = pattern
    .replace(/\[([^\]]+)\]/g, '([^/]+)') // Replace [id] with capture groups
    .replace(/\//g, '\\/'); // Escape forward slashes

  const regex = new RegExp(`^${regexPattern}$`);
  return regex.test(pathname);
}

// Get all matching patterns for a pathname (useful for nested routes)
export function getMatchingPatterns(pathname: string): RouteConfig[] {
  return ROUTE_CONFIGS.filter(config => 
    matchesPattern(pathname, config.pattern)
  );
}

// Extract parameters from a pathname using a pattern
export function extractParams(pathname: string, pattern: string): Record<string, string> {
  const params: Record<string, string> = {};
  
  const patternSegments = pattern.split('/').filter(Boolean);
  const pathSegments = pathname.split('/').filter(Boolean);
  
  if (patternSegments.length !== pathSegments.length) {
    return params;
  }
  
  patternSegments.forEach((segment, index) => {
    if (segment.startsWith('[') && segment.endsWith(']')) {
      const paramName = segment.slice(1, -1);
      const pathSegment = pathSegments[index];
      if (pathSegment) {
        params[paramName] = pathSegment;
      }
    }
  });
  
  return params;
}

// Get the hierarchical route structure for a pathname
export function getRouteHierarchy(pathname: string): RouteConfig[] {
  const config = findRouteConfig(pathname);
  if (!config) return [];

  const hierarchy: RouteConfig[] = [config];
  
  let currentConfig = config;
  while (currentConfig.parent) {
    const parentConfig = ROUTE_CONFIGS.find(c => c.pattern === currentConfig.parent);
    if (parentConfig) {
      hierarchy.unshift(parentConfig);
      currentConfig = parentConfig;
    } else {
      break;
    }
  }
  
  return hierarchy;
}

// Get breadcrumb segments with route-specific configuration
export function getBreadcrumbsFromRoute(pathname: string): Partial<BreadcrumbSegment>[] {
  const hierarchy = getRouteHierarchy(pathname);
  const params = findRouteConfig(pathname) 
    ? extractParams(pathname, findRouteConfig(pathname)!.pattern)
    : {};

  return hierarchy.map((config, index) => {
    const isLast = index === hierarchy.length - 1;
    const href = buildHrefFromPattern(config.pattern, params);
    
    const breadcrumb: Partial<BreadcrumbSegment> = {
      segment: config.pattern.split('/').pop() || '',
      label: config.labelResolver ? config.labelResolver(params) : config.label,
      href,
      dynamic: config.entityType !== undefined,
      isCurrentPage: isLast,
    };

    // Only set entityId and entityType if they exist
    if (config.entityType && params.id) {
      breadcrumb.entityId = params.id;
      breadcrumb.entityType = config.entityType;
    }

    return breadcrumb;
  });
}

// Build href from pattern and parameters
export function buildHrefFromPattern(pattern: string, params: Record<string, string>): string {
  let href = pattern;
  
  Object.entries(params).forEach(([key, value]) => {
    href = href.replace(`[${key}]`, value);
  });
  
  return href;
}

// Get route config for a specific entity type
export function getRoutesByEntityType(entityType: string): RouteConfig[] {
  return ROUTE_CONFIGS.filter(config => config.entityType === entityType);
}

// Check if a route requires authentication
export function routeRequiresAuth(pathname: string): boolean {
  const config = findRouteConfig(pathname);
  return config?.requiresAuth ?? false;
}

// Check if a route should be hidden from breadcrumbs
export function isRouteHidden(pathname: string): boolean {
  const config = findRouteConfig(pathname);
  return config?.hidden ?? false;
}

// Get all available routes (useful for navigation generation)
export function getAllRoutes(): RouteConfig[] {
  return [...ROUTE_CONFIGS];
}

// Get child routes for a given parent pattern
export function getChildRoutes(parentPattern: string): RouteConfig[] {
  return ROUTE_CONFIGS.filter(config => config.parent === parentPattern);
}

// Validate if a route pattern is properly configured
export function validateRouteConfig(config: RouteConfig): boolean {
  // Basic validation
  if (!config.pattern || !config.label) return false;
  
  // Check parent exists if specified
  if (config.parent && !ROUTE_CONFIGS.some(c => c.pattern === config.parent)) {
    return false;
  }
  
  return true;
}

// Export route patterns as constants for type safety
export const ROUTE_PATTERNS = {
  HOME: '/',
  ACCOUNTS: '/accounts',
  ACCOUNT_DETAIL: '/accounts/[id]',
  CHARACTERS: '/characters',
  CHARACTER_DETAIL: '/characters/[id]',
  GUILDS: '/guilds',
  GUILD_DETAIL: '/guilds/[id]',
  NPCS: '/npcs',
  NPC_DETAIL: '/npcs/[id]',
  NPC_CONVERSATIONS: '/npcs/[id]/conversations',
  NPC_SHOP: '/npcs/[id]/shop',
  TEMPLATES: '/templates',
  TEMPLATE_DETAIL: '/templates/[id]',
  TEMPLATE_PROPERTIES: '/templates/[id]/properties',
  TEMPLATE_HANDLERS: '/templates/[id]/handlers',
  TEMPLATE_WRITERS: '/templates/[id]/writers',
  TEMPLATE_WORLDS: '/templates/[id]/worlds',
  TEMPLATE_CHARACTER: '/templates/[id]/character',
  TEMPLATE_CHARACTER_TEMPLATES: '/templates/[id]/character/templates',
  TENANTS: '/tenants',
  TENANT_DETAIL: '/tenants/[id]',
  TENANT_PROPERTIES: '/tenants/[id]/properties',
  TENANT_HANDLERS: '/tenants/[id]/handlers',
  TENANT_WRITERS: '/tenants/[id]/writers',
  TENANT_WORLDS: '/tenants/[id]/worlds',
  TENANT_CHARACTER: '/tenants/[id]/character',
  TENANT_CHARACTER_TEMPLATES: '/tenants/[id]/character/templates',
} as const;