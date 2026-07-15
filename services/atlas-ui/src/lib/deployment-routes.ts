/**
 * The single definition of "Deployment route" — pages whose changes affect
 * every tenant. The tenant switcher's inert state and the deployment scope
 * banner both consume this predicate; they can never disagree.
 */
export const DEPLOYMENT_ROUTE_PREFIXES = [
  '/templates',
  '/tenants',
  '/services',
  '/baselines',
] as const;

export function isDeploymentRoute(pathname: string): boolean {
  return DEPLOYMENT_ROUTE_PREFIXES.some(
    (prefix) => pathname === prefix || pathname.startsWith(prefix + '/'),
  );
}
