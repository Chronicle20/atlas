/**
 * Route configuration mapping for breadcrumb navigation
 * Defines all application routes and their breadcrumb metadata
 */

import { type BreadcrumbSegment } from "./utils";

/** A version-correct job-name resolver, supplied by the calling component. */
export type JobNameResolver = (id: number) => string;

/**
 * Context a labelResolver may need but cannot fetch itself: the route table
 * is a plain module-level array, not a component, so it cannot call hooks.
 * useBreadcrumbs supplies this from useJobNameLookup().
 */
export interface BreadcrumbResolverContext {
  jobName: JobNameResolver;
}

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
  /**
   * Grouping-only node with no page of its own — rendered as a plain label,
   * never a clickable link (e.g. the "Character" node whose only real pages
   * are its /templates and /presets children).
   */
  nonNavigable?: boolean;
  /** Entity type for dynamic routes */
  entityType?: string;
  /** Custom label resolver function */
  labelResolver?: (
    params: Record<string, string>,
    ctx: BreadcrumbResolverContext,
  ) => string;
}

// Comprehensive route configuration for all application routes
export const ROUTE_CONFIGS: RouteConfig[] = [
  // Root route
  {
    pattern: "/",
    label: "Home",
  },

  // Main entity list routes
  {
    pattern: "/accounts",
    label: "Accounts",
    parent: "/",
  },
  {
    pattern: "/baselines",
    label: "Baselines",
    parent: "/",
  },
  {
    pattern: "/characters",
    label: "Characters",
    parent: "/",
  },
  {
    pattern: "/guilds",
    label: "Guilds",
    parent: "/",
  },
  {
    pattern: "/npcs",
    label: "NPCs",
    parent: "/",
  },
  {
    pattern: "/quests",
    label: "Quests",
    parent: "/",
  },
  {
    pattern: "/quests/[id]",
    label: "Quest Details",
    parent: "/quests",
    entityType: "quest",
  },
  {
    pattern: "/services",
    label: "Services",
    parent: "/",
  },
  {
    pattern: "/services/[id]",
    label: "Service Details",
    parent: "/services",
    entityType: "service",
  },
  {
    pattern: "/templates",
    label: "Templates",
    parent: "/",
  },
  {
    pattern: "/tenants",
    label: "Tenants",
    parent: "/",
  },

  // Account detail routes
  {
    pattern: "/accounts/[id]",
    label: "Account Details",
    parent: "/accounts",
    entityType: "account",
  },

  // Character detail routes
  {
    pattern: "/characters/[id]",
    label: "Character Details",
    parent: "/characters",
    entityType: "character",
  },

  // Guild routes
  {
    pattern: "/guilds/[id]",
    label: "Guild Details",
    parent: "/guilds",
    entityType: "guild",
  },

  // NPC routes
  {
    pattern: "/npcs/[id]",
    label: "NPC Details",
    parent: "/npcs",
    entityType: "npc",
  },

  // Monster routes
  {
    pattern: "/monsters",
    label: "Monsters",
    parent: "/",
  },
  {
    pattern: "/monsters/[id]",
    label: "Monster Details",
    parent: "/monsters",
    entityType: "monster",
  },

  // Item routes
  {
    pattern: "/items",
    label: "Items",
    parent: "/",
  },
  {
    pattern: "/items/[id]",
    label: "Item Details",
    parent: "/items",
    entityType: "item",
  },

  // Job routes
  {
    pattern: "/jobs",
    label: "Jobs",
    parent: "/",
  },
  {
    pattern: "/jobs/[id]",
    label: "Job Details",
    parent: "/jobs",
    // Deliberately no `entityType`: job names come from the tenant's job
    // graph via labelResolver, not from an async entity resolver. Declaring
    // one would mark the crumb `dynamic` and the resolver lookup would miss,
    // overwriting the label with "Unknown". The graph is version-correct —
    // the static table this replaced named wire id 500 "Pirate" even on a
    // v0.48 tenant, where it is Gm (task-202 FR-4.2).
    labelResolver: (params, ctx) => ctx.jobName(Number(params.id)),
  },

  // Map routes
  {
    pattern: "/maps",
    label: "Maps",
    parent: "/",
  },
  {
    pattern: "/maps/[id]",
    label: "Map Details",
    parent: "/maps",
    entityType: "map",
  },
  {
    pattern: "/maps/[id]/portals/[portalId]",
    label: "Portal Details",
    parent: "/maps/[id]",
    entityType: "portal",
  },

  // Field routes (runtime read model — live occupancy, not a definition)
  {
    pattern: "/fields",
    label: "Fields",
    parent: "/",
  },
  {
    pattern: "/fields/[worldId]",
    label: "World",
    parent: "/fields",
    // Grouping node only — no page lives at this path (D11). The world name
    // is not available from params alone, so the label falls back to the id;
    // the resolved name is shown in the FieldDetailPage header instead.
    nonNavigable: true,
    labelResolver: (params) => `World ${params.worldId ?? ""}`,
  },
  {
    pattern: "/fields/[worldId]/[channelId]",
    label: "Channel",
    parent: "/fields/[worldId]",
    // Grouping node only — no page lives at this path (D11).
    nonNavigable: true,
    labelResolver: (params) => `Channel ${params.channelId ?? ""}`,
  },
  {
    pattern: "/fields/[worldId]/[channelId]/[mapId]",
    label: "Map",
    parent: "/fields/[worldId]/[channelId]",
    // Grouping node only — no page lives at this path (D11). The map
    // definition name is not available from params alone, so the label
    // falls back to the map id.
    nonNavigable: true,
    labelResolver: (params) => params.mapId ?? "",
  },
  {
    pattern: "/fields/[worldId]/[channelId]/[mapId]/[instanceId]",
    label: "Instance",
    parent: "/fields/[worldId]/[channelId]/[mapId]",
    labelResolver: (params) => `Instance ${params.instanceId ?? ""}`,
  },

  // Reactor routes
  {
    pattern: "/reactors",
    label: "Reactors",
    parent: "/",
  },
  {
    pattern: "/reactors/[id]",
    label: "Reactor Details",
    parent: "/reactors",
    entityType: "reactor",
  },

  // Merchant routes
  {
    pattern: "/merchants",
    label: "Merchants",
    parent: "/",
  },
  {
    pattern: "/merchants/[id]",
    label: "Merchant Details",
    parent: "/merchants",
    entityType: "merchant",
  },

  // Marketplace routes
  {
    pattern: "/marketplace",
    label: "Marketplace",
    parent: "/",
  },

  // Ranking routes
  {
    pattern: "/rankings",
    label: "Rankings",
    parent: "/",
  },

  // Ban routes
  {
    pattern: "/bans",
    label: "Bans",
    parent: "/",
  },
  {
    // Matches App.tsx's `/bans/:banId` — the param name differs but
    // matchesPattern/extractParams key off position, not name.
    pattern: "/bans/[id]",
    label: "Ban Details",
    parent: "/bans",
    entityType: "ban",
  },

  // Login history routes
  {
    pattern: "/login-history",
    label: "Login History",
    parent: "/",
  },

  // Packet matrix routes
  {
    pattern: "/packet-matrix",
    label: "Packet Matrix",
    parent: "/",
  },

  // Coupon routes
  {
    pattern: "/coupons",
    label: "Coupons",
    parent: "/",
  },
  {
    // Matches App.tsx's `/coupons/:couponId`. The label resolves to the
    // coupon's own code (a coupon has no name), via the "coupon" resolver.
    pattern: "/coupons/[id]",
    label: "Coupon",
    parent: "/coupons",
    entityType: "coupon",
  },

  // Event routes
  {
    pattern: "/events/definitions",
    label: "Definitions",
    parent: "/",
  },
  {
    pattern: "/events/occurrences",
    label: "Occurrences",
    parent: "/",
  },
  {
    // No `entityType`: occurrences have no name of their own to resolve
    // (unlike accounts/npcs/etc, whose EntityType resolvers live in
    // resolvers.ts) — a static crumb, same pattern as "/reports/[reportId]"
    // below. Registering an entityType with no matching resolver in
    // resolvers.ts would mark the crumb `dynamic` and resolve to "Unknown".
    pattern: "/events/occurrences/[id]",
    label: "Occurrence",
    parent: "/events/occurrences",
  },

  // Reward pool routes
  {
    pattern: "/reward-pools",
    label: "Reward Pools",
    parent: "/",
  },
  {
    pattern: "/reward-pools/[id]",
    label: "Pool",
    parent: "/reward-pools",
    entityType: "reward-pool",
  },

  // Setup routes
  {
    pattern: "/setup",
    label: "Setup",
    parent: "/",
  },

  // Transport routes
  {
    pattern: "/transports",
    label: "Transports",
    parent: "/",
  },
  {
    // Matches App.tsx's `/transports/routes/:routeId`. `parent` skips straight
    // back to the board — there is no page at `/transports/routes`.
    pattern: "/transports/routes/[id]",
    label: "Route",
    parent: "/transports",
    entityType: "transport-route",
  },

  // Template routes
  {
    pattern: "/templates/[id]",
    label: "Template Details",
    parent: "/templates",
    entityType: "template",
  },
  {
    pattern: "/templates/[id]/properties",
    label: "Properties",
    parent: "/templates/[id]",
  },
  {
    pattern: "/templates/[id]/handlers",
    label: "Socket Handlers",
    parent: "/templates/[id]",
  },
  {
    pattern: "/templates/[id]/writers",
    label: "Socket Writers",
    parent: "/templates/[id]",
  },
  {
    pattern: "/templates/[id]/worlds",
    label: "Worlds",
    parent: "/templates/[id]",
  },
  {
    pattern: "/templates/[id]/character",
    label: "Character",
    parent: "/templates/[id]",
    // Grouping node only — no page lives at this path (children are
    // /character/templates and /character/presets). Render as a label,
    // never a link.
    nonNavigable: true,
  },
  {
    pattern: "/templates/[id]/character/templates",
    label: "Templates",
    parent: "/templates/[id]/character",
  },
  {
    pattern: "/templates/[id]/character/presets",
    label: "Presets",
    parent: "/templates/[id]/character",
  },
  {
    pattern: "/templates/[id]/character/maple-life",
    label: "Maple Life",
    parent: "/templates/[id]/character",
  },

  // Tenant routes
  {
    pattern: "/tenants/[id]",
    label: "Tenant Details",
    parent: "/tenants",
    entityType: "tenant",
  },
  {
    pattern: "/tenants/[id]/properties",
    label: "Properties",
    parent: "/tenants/[id]",
  },
  {
    pattern: "/tenants/[id]/handlers",
    label: "Socket Handlers",
    parent: "/tenants/[id]",
  },
  {
    pattern: "/tenants/[id]/writers",
    label: "Socket Writers",
    parent: "/tenants/[id]",
  },
  {
    pattern: "/tenants/[id]/worlds",
    label: "Worlds",
    parent: "/tenants/[id]",
  },
  {
    pattern: "/tenants/[id]/character",
    label: "Character",
    parent: "/tenants/[id]",
    // Grouping node only — no page lives at this path (children are
    // /character/templates and /character/presets). Render as a label,
    // never a link.
    nonNavigable: true,
  },
  {
    pattern: "/tenants/[id]/character/templates",
    label: "Templates",
    parent: "/tenants/[id]/character",
  },
  {
    pattern: "/tenants/[id]/character/presets",
    label: "Presets",
    parent: "/tenants/[id]/character",
  },
  {
    pattern: "/tenants/[id]/character/maple-life",
    label: "Maple Life",
    parent: "/tenants/[id]/character",
  },

  // Report routes
  {
    pattern: "/reports",
    label: "Reports",
    parent: "/",
  },
  {
    pattern: "/reports/[reportId]",
    label: "Report Detail",
    parent: "/reports",
  },
];

// Helper function to find route config by pathname
export function findRouteConfig(pathname: string): RouteConfig | null {
  // Direct match first
  const directMatch = ROUTE_CONFIGS.find(
    (config) => config.pattern === pathname,
  );
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
    .replace(/\[([^\]]+)\]/g, "([^/]+)") // Replace [id] with capture groups
    .replace(/\//g, "\\/"); // Escape forward slashes

  const regex = new RegExp(`^${regexPattern}$`);
  return regex.test(pathname);
}

// Get all matching patterns for a pathname (useful for nested routes)
export function getMatchingPatterns(pathname: string): RouteConfig[] {
  return ROUTE_CONFIGS.filter((config) =>
    matchesPattern(pathname, config.pattern),
  );
}

// Extract parameters from a pathname using a pattern
export function extractParams(
  pathname: string,
  pattern: string,
): Record<string, string> {
  const params: Record<string, string> = {};

  const patternSegments = pattern.split("/").filter(Boolean);
  const pathSegments = pathname.split("/").filter(Boolean);

  if (patternSegments.length !== pathSegments.length) {
    return params;
  }

  patternSegments.forEach((segment, index) => {
    if (segment.startsWith("[") && segment.endsWith("]")) {
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
    const parentConfig = ROUTE_CONFIGS.find(
      (c) => c.pattern === currentConfig.parent,
    );
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
export function getBreadcrumbsFromRoute(
  pathname: string,
  ctx: BreadcrumbResolverContext,
): Partial<BreadcrumbSegment>[] {
  const hierarchy = getRouteHierarchy(pathname);
  const params = findRouteConfig(pathname)
    ? extractParams(pathname, findRouteConfig(pathname)!.pattern)
    : {};

  return hierarchy.map((config, index) => {
    const isLast = index === hierarchy.length - 1;
    const href = buildHrefFromPattern(config.pattern, params);

    const breadcrumb: Partial<BreadcrumbSegment> = {
      segment: config.pattern.split("/").pop() || "",
      label: config.labelResolver
        ? config.labelResolver(params, ctx)
        : config.label,
      href,
      dynamic: config.entityType !== undefined,
      isCurrentPage: isLast,
    };

    // Only set entityId and entityType if they exist
    if (config.entityType && params.id) {
      breadcrumb.entityId = params.id;
      breadcrumb.entityType = config.entityType;
    }

    if (config.nonNavigable) {
      breadcrumb.nonNavigable = true;
    }

    return breadcrumb;
  });
}

// Build href from pattern and parameters
export function buildHrefFromPattern(
  pattern: string,
  params: Record<string, string>,
): string {
  let href = pattern;

  Object.entries(params).forEach(([key, value]) => {
    href = href.replace(`[${key}]`, value);
  });

  return href;
}

// Get route config for a specific entity type
export function getRoutesByEntityType(entityType: string): RouteConfig[] {
  return ROUTE_CONFIGS.filter((config) => config.entityType === entityType);
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
  return ROUTE_CONFIGS.filter((config) => config.parent === parentPattern);
}

// Validate if a route pattern is properly configured
export function validateRouteConfig(config: RouteConfig): boolean {
  // Basic validation
  if (!config.pattern || !config.label) return false;

  // Check parent exists if specified
  if (
    config.parent &&
    !ROUTE_CONFIGS.some((c) => c.pattern === config.parent)
  ) {
    return false;
  }

  return true;
}

// Export route patterns as constants for type safety
export const ROUTE_PATTERNS = {
  HOME: "/",
  ACCOUNTS: "/accounts",
  ACCOUNT_DETAIL: "/accounts/[id]",
  BASELINES: "/baselines",
  CHARACTERS: "/characters",
  CHARACTER_DETAIL: "/characters/[id]",
  GUILDS: "/guilds",
  GUILD_DETAIL: "/guilds/[id]",
  NPCS: "/npcs",
  NPC_DETAIL: "/npcs/[id]",
  QUESTS: "/quests",
  QUEST_DETAIL: "/quests/[id]",
  SERVICES: "/services",
  SERVICE_DETAIL: "/services/[id]",
  TEMPLATES: "/templates",
  TEMPLATE_DETAIL: "/templates/[id]",
  TEMPLATE_PROPERTIES: "/templates/[id]/properties",
  TEMPLATE_HANDLERS: "/templates/[id]/handlers",
  TEMPLATE_WRITERS: "/templates/[id]/writers",
  TEMPLATE_WORLDS: "/templates/[id]/worlds",
  TEMPLATE_CHARACTER: "/templates/[id]/character",
  TEMPLATE_CHARACTER_TEMPLATES: "/templates/[id]/character/templates",
  TEMPLATE_CHARACTER_PRESETS: "/templates/[id]/character/presets",
  TEMPLATE_CHARACTER_MAPLE_LIFE: "/templates/[id]/character/maple-life",
  MONSTERS: "/monsters",
  MONSTER_DETAIL: "/monsters/[id]",
  ITEMS: "/items",
  ITEM_DETAIL: "/items/[id]",
  JOBS: "/jobs",
  JOB_DETAIL: "/jobs/[id]",
  MAPS: "/maps",
  MAP_DETAIL: "/maps/[id]",
  MAP_PORTAL_DETAIL: "/maps/[id]/portals/[portalId]",
  FIELDS: "/fields",
  FIELD_DETAIL: "/fields/[worldId]/[channelId]/[mapId]/[instanceId]",
  REACTORS: "/reactors",
  REACTOR_DETAIL: "/reactors/[id]",
  MERCHANTS: "/merchants",
  MERCHANT_DETAIL: "/merchants/[id]",
  MARKETPLACE: "/marketplace",
  RANKINGS: "/rankings",
  BANS: "/bans",
  BAN_DETAIL: "/bans/[id]",
  LOGIN_HISTORY: "/login-history",
  PACKET_MATRIX: "/packet-matrix",
  COUPONS: "/coupons",
  COUPON_DETAIL: "/coupons/[id]",
  EVENT_DEFINITIONS: "/events/definitions",
  EVENT_OCCURRENCES: "/events/occurrences",
  EVENT_OCCURRENCE_DETAIL: "/events/occurrences/[id]",
  REWARD_POOLS: "/reward-pools",
  REWARD_POOL_DETAIL: "/reward-pools/[id]",
  SETUP: "/setup",
  TRANSPORTS: "/transports",
  TRANSPORT_ROUTE_DETAIL: "/transports/routes/[id]",
  TENANTS: "/tenants",
  TENANT_DETAIL: "/tenants/[id]",
  TENANT_PROPERTIES: "/tenants/[id]/properties",
  TENANT_HANDLERS: "/tenants/[id]/handlers",
  TENANT_WRITERS: "/tenants/[id]/writers",
  TENANT_WORLDS: "/tenants/[id]/worlds",
  TENANT_CHARACTER: "/tenants/[id]/character",
  TENANT_CHARACTER_TEMPLATES: "/tenants/[id]/character/templates",
  TENANT_CHARACTER_PRESETS: "/tenants/[id]/character/presets",
  TENANT_CHARACTER_MAPLE_LIFE: "/tenants/[id]/character/maple-life",
} as const;
