import { useState } from "react";
import { Lock } from "lucide-react";
import type { Tenant } from "@/services/api/tenants.service";
import { getItemProtectorIconUrl } from "@/lib/utils/asset-url";
import { cn } from "@/lib/utils";

/**
 * SealIcon renders the sealed-item padlock. It prefers the authentic game
 * overlay (UI.wz/UIWindow.img/ItemProtector/Icon, served by atlas-assets under
 * the given tenant) and falls back to a lucide Lock glyph when the asset is
 * missing (onError) or no tenant is available — so the badge never vanishes.
 *
 * The tenant is passed in rather than read from context so the component stays
 * pure and testable; callers resolve it from their own tenant prop or the active
 * tenant. `data-testid="seal-icon"` is preserved across both render branches.
 */
export function SealIcon({ tenant, className }: { tenant?: Tenant | null; className?: string }) {
  const [failed, setFailed] = useState(false);

  if (!tenant || failed) {
    return <Lock data-testid="seal-icon" className={className} aria-label="Sealed item" />;
  }

  return (
    <img
      src={getItemProtectorIconUrl(
        tenant.id,
        tenant.attributes.region,
        tenant.attributes.majorVersion,
        tenant.attributes.minorVersion,
      )}
      data-testid="seal-icon"
      className={cn(className, "object-contain")}
      alt="Sealed item"
      aria-label="Sealed item"
      loading="lazy"
      onError={() => setFailed(true)}
    />
  );
}
