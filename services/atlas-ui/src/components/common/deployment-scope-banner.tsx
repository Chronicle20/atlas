import { Globe } from "lucide-react";
import { useLocation } from "react-router-dom";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { isDeploymentRoute } from "@/lib/deployment-routes";

/**
 * Slim, non-dismissible callout shown on every Deployment page (and all
 * their subpages). Mounted once in AppShell; self-conditions on the same
 * route predicate the tenant switcher uses, so the two scope signals can
 * never disagree.
 */
export function DeploymentScopeBanner() {
  const { pathname } = useLocation();
  if (!isDeploymentRoute(pathname)) return null;
  return (
    <Alert variant="warning" className="mx-2 w-auto py-2">
      <Globe className="h-4 w-4" />
      <AlertDescription>
        Changes on this page affect all tenants.
      </AlertDescription>
    </Alert>
  );
}
