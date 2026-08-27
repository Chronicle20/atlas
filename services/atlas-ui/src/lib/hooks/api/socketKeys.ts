/**
 * Query keys for the SPARSE socket reads that feed the Packet Matrix.
 *
 * These deliberately do NOT reuse templateKeys.detail / tenantKeys.configDetail.
 * A sparse document that reached a mutation's attribute spread would silently
 * erase characters, worlds and cashShop (see tenantsService.updateTenantConfiguration
 * and templatesService.update, both of which write the whole document), so the
 * two live under separate keys and the sparse one is never a write input -
 * useSocketMutation (useSocketObjects.ts) never reads from socketKeys.* at all.
 *
 * Lives in its own module (rather than in useSocketObjects.ts, where the
 * matrix hooks that consume it are defined) because useSocketObjects.ts
 * already imports templateKeys from useTemplates.ts and tenantKeys from
 * useTenants.ts. Every template/tenant-configuration mutation hook must also
 * invalidate socketKeys.all, so importing socketKeys from useSocketObjects.ts
 * in those two files would close an import cycle.
 */
export const socketKeys = {
  all: ["socket"] as const,
  matrix: () => [...socketKeys.all, "matrix", "templates"] as const,
  tenantMatrix: () => [...socketKeys.all, "matrix", "tenants"] as const,
};
