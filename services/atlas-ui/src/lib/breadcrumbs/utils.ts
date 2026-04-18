/**
 * Breadcrumb utility functions for route parsing and navigation
 * Supports Next.js App Router with dynamic segments and multi-tenant context
 */

// Types for breadcrumb items
export interface BreadcrumbSegment {
  /** Original URL segment (e.g., 'users', '[id]', 'settings') */
  segment: string;
  /** Human-readable label (e.g., 'Users', 'John Doe', 'Settings') */
  label: string;
  /** Full URL path to this segment */
  href: string;
  /** Whether this segment represents a dynamic route parameter */
  dynamic: boolean;
  /** Whether this is the current/active page */
  isCurrentPage: boolean;
  /** Entity ID if this is a dynamic segment */
  entityId?: string;
  /** Entity type for dynamic segments (for label resolution) */
  entityType?: string;
}

// Configuration for mapping route segments to labels
export const STATIC_ROUTE_LABELS: Record<string, string> = {
  // Main sections
  'accounts': 'Accounts',
  'characters': 'Characters',
  'guilds': 'Guilds',
  'npcs': 'NPCs',
  'templates': 'Templates',
  'tenants': 'Tenants',
  
  // Sub-sections
  'properties': 'Properties',
  'handlers': 'Socket Handlers',
  'writers': 'Socket Writers',
  'worlds': 'Worlds',
  'conversations': 'Conversations',
  'shop': 'Shop',
  'character': 'Character',
  
  // Special cases
  '': 'Home',
};

// Patterns to identify entity types from route segments
export const ENTITY_TYPE_PATTERNS: Array<{
  pattern: RegExp;
  getEntityType: (segments: string[], index: number) => string | null;
}> = [
  {
    pattern: /^\/accounts\/([^/]+)$/,
    getEntityType: () => 'account'
  },
  {
    pattern: /^\/characters\/([^/]+)$/,
    getEntityType: () => 'character'
  },
  {
    pattern: /^\/guilds\/([^/]+)$/,
    getEntityType: () => 'guild'
  },
  {
    pattern: /^\/npcs\/([^/]+)/,
    getEntityType: () => 'npc'
  },
  {
    pattern: /^\/templates\/([^/]+)/,
    getEntityType: () => 'template'
  },
  {
    pattern: /^\/tenants\/([^/]+)/,
    getEntityType: () => 'tenant'
  },
];

/**
 * Checks if a route segment is dynamic (represents a parameter like [id])
 */
export function isDynamicSegment(segment: string, pathname: string): boolean {
  // First check: don't treat known static segments as dynamic
  if (STATIC_ROUTE_LABELS[segment]) {
    return false;
  }

  // Check if segment looks like an ID (UUID, number, or alphanumeric)
  const isId = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/.test(segment) || // UUID
           /^\d+$/.test(segment) || // Numeric ID
           /^[a-zA-Z0-9_-]+$/.test(segment); // Alphanumeric ID
  
  if (!isId) return false;

  // Additional validation: check if this segment follows a known entity pattern
  const segments = pathname.split('/').filter(Boolean);
  const segmentIndex = segments.indexOf(segment);
  
  if (segmentIndex === -1) return false;

  // Check if this ID appears in a known dynamic position
  const pathUpToSegment = '/' + segments.slice(0, segmentIndex + 1).join('/');
  
  return ENTITY_TYPE_PATTERNS.some(({ pattern }) => pattern.test(pathUpToSegment));
}

/**
 * Gets the entity type for a dynamic segment based on its position in the route
 */
export function getEntityType(segment: string, pathname: string): string | null {
  const segments = pathname.split('/').filter(Boolean);
  const segmentIndex = segments.indexOf(segment);
  
  if (segmentIndex === -1) return null;

  const pathUpToSegment = '/' + segments.slice(0, segmentIndex + 1).join('/');
  
  for (const { pattern, getEntityType } of ENTITY_TYPE_PATTERNS) {
    if (pattern.test(pathUpToSegment)) {
      return getEntityType(segments, segmentIndex);
    }
  }
  
  return null;
}

/**
 * Parses a pathname into breadcrumb segments
 */
export function parsePathname(pathname: string): BreadcrumbSegment[] {
  // Handle root path
  if (pathname === '/') {
    return [{
      segment: '',
      label: STATIC_ROUTE_LABELS[''] || 'Home',
      href: '/',
      dynamic: false,
      isCurrentPage: true,
    }];
  }

  const segments = pathname.split('/').filter(Boolean);
  const breadcrumbs: BreadcrumbSegment[] = [];
  
  // Add home breadcrumb for non-root paths
  breadcrumbs.push({
    segment: '',
    label: STATIC_ROUTE_LABELS[''] || 'Home',
    href: '/',
    dynamic: false,
    isCurrentPage: false,
  });

  // Process each segment
  segments.forEach((segment, index) => {
    const isLast = index === segments.length - 1;
    const currentPath = '/' + segments.slice(0, index + 1).join('/');
    const isDynamic = isDynamicSegment(segment, pathname);
    
    const breadcrumbSegment: BreadcrumbSegment = {
      segment,
      label: isDynamic ? segment : (STATIC_ROUTE_LABELS[segment] || capitalizeFirst(segment)),
      href: currentPath,
      dynamic: isDynamic,
      isCurrentPage: isLast,
    };

    // Add entity metadata for dynamic segments
    if (isDynamic) {
      breadcrumbSegment.entityId = segment;
      const entityType = getEntityType(segment, pathname);
      if (entityType) {
        breadcrumbSegment.entityType = entityType;
      }
    }

    breadcrumbs.push(breadcrumbSegment);
  });

  return breadcrumbs;
}

/**
 * Builds a breadcrumb path from segments with optional filtering
 */
export function buildBreadcrumbPath(
  segments: BreadcrumbSegment[],
  options: {
    maxItems?: number;
    showEllipsis?: boolean;
  } = {}
): BreadcrumbSegment[] {
  const { maxItems = 5, showEllipsis = true } = options;
  
  if (segments.length <= maxItems) {
    return segments;
  }

  if (!showEllipsis) {
    return segments.slice(segments.length - maxItems);
  }

  // Keep first, last, and some middle items with ellipsis
  const firstItem = segments[0];
  if (!firstItem) return segments; // Guard against undefined
  
  const lastItems = segments.slice(-2); // Keep last 2 items
  const middleItems = segments.slice(1, -(Math.min(2, segments.length - 1)));
  
  // If we have room, include some middle items
  const availableSlots = maxItems - 3; // -3 for first, ellipsis, and last items
  const selectedMiddleItems = availableSlots > 0 
    ? middleItems.slice(-availableSlots) 
    : [];

  const result: BreadcrumbSegment[] = [firstItem];
  
  // Add ellipsis if we're skipping items
  if (middleItems.length > selectedMiddleItems.length) {
    result.push({
      segment: '...',
      label: '...',
      href: '',
      dynamic: false,
      isCurrentPage: false,
    });
  }

  result.push(...selectedMiddleItems, ...lastItems);
  
  return result;
}

/**
 * Gets the parent breadcrumb for navigation
 */
export function getParentBreadcrumb(segments: BreadcrumbSegment[]): BreadcrumbSegment | null {
  if (segments.length <= 1) return null;
  
  // Return the second-to-last segment (parent of current page)
  const parentIndex = segments.length - 2;
  return segments[parentIndex] || null;
}

/**
 * Validates if a pathname matches expected route patterns
 */
export function isValidRoute(pathname: string): boolean {
  if (pathname === '/') return true;
  
  const segments = pathname.split('/').filter(Boolean);
  
  // Check if first segment is a known route
  if (segments.length === 0) return true;
  
  const firstSegment = segments[0];
  if (!firstSegment) return true; // Empty segments is valid (home)
  
  const knownRoutes = Object.keys(STATIC_ROUTE_LABELS).filter(key => key !== '');
  
  return knownRoutes.includes(firstSegment);
}

/**
 * Capitalizes the first letter of a string
 */
export function capitalizeFirst(str: string): string {
  if (!str) return str;
  return str.charAt(0).toUpperCase() + str.slice(1);
}

/**
 * Generates a breadcrumb key for caching/comparison
 */
export function getBreadcrumbKey(segments: BreadcrumbSegment[]): string {
  return segments.map(s => `${s.segment}:${s.dynamic ? 'dynamic' : 'static'}`).join('|');
}

/**
 * Filters breadcrumbs based on user permissions or visibility rules
 */
export function filterVisibleBreadcrumbs(
  segments: BreadcrumbSegment[],
  options: {
    hiddenRoutes?: string[];
    userPermissions?: string[];
  } = {}
): BreadcrumbSegment[] {
  const { hiddenRoutes = [] } = options;
  
  return segments.filter(segment => {
    // Skip hidden routes
    if (hiddenRoutes.includes(segment.segment)) return false;
    
    // Add permission checks here if needed
    // For now, show all segments
    return true;
  });
}