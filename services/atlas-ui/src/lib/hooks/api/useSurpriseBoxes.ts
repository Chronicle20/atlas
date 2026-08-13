/**
 * The active tenant's Cash Shop Surprise box template ids.
 *
 * A cash-surprise reward pool is keyed by the BOX template id (the pool id IS
 * the box id), and atlas-cashshop only rolls a pool whose id appears in this
 * list — a pool keyed to anything else is never reachable. The dialog uses
 * this to warn, not to block: an operator may legitimately create the pool
 * first and add the id to the tenant configuration afterwards.
 *
 * The fallback mirrors atlas-cashshop configuration/registry.go
 * `GetSurpriseBoxTemplateIds`: an absent or empty list means "unconfigured",
 * which resolves to the stock box, NOT to "no box works".
 */

import { useTenantConfiguration } from "@/lib/hooks/api/useTenants";
import { useTenant } from "@/context/tenant-context";

/** atlas-cashshop `DefaultSurpriseBoxTemplateId` (configuration/registry.go). */
export const DEFAULT_SURPRISE_BOX_TEMPLATE_ID = 5222000;

export interface SurpriseBoxTemplateIds {
  ids: number[];
  /** False while the configuration is still loading or failed to load. */
  isResolved: boolean;
}

export function useSurpriseBoxTemplateIds(): SurpriseBoxTemplateIds {
  const { activeTenant } = useTenant();
  const config = useTenantConfiguration(activeTenant?.id ?? "");

  const configured = config.data?.attributes.cashShop?.surprise?.boxTemplateIds;
  return {
    ids:
      configured && configured.length > 0
        ? configured
        : [DEFAULT_SURPRISE_BOX_TEMPLATE_ID],
    isResolved: config.isSuccess,
  };
}
