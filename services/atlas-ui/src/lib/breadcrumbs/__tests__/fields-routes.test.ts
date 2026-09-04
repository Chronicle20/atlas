/**
 * Tests for the /fields route configuration (runtime read model, FR-17/FR-18).
 *
 * The field-detail view is a query-param (`?instance=`) variant of the same
 * `/fields` route, not a separate path, so there is no nested
 * world/channel/map/instance breadcrumb chain to resolve (bug-fields-ui
 * items 5/6).
 */

import {
  getBreadcrumbsFromRoute,
  ROUTE_PATTERNS,
  type BreadcrumbResolverContext,
} from "../routes";

// Static stub — none of the /fields resolvers consume jobName, but the
// context is a required argument.
const testCtx: BreadcrumbResolverContext = {
  jobName: (id) => `Job ${id}`,
};

describe("/fields routes", () => {
  it("locator", () => {
    const breadcrumbs = getBreadcrumbsFromRoute("/fields", testCtx);
    expect(breadcrumbs.map((b) => b.label)).toEqual(["Home", "Fields"]);
  });

  it("field detail view collapses to the same locator trail (query params carry the rest)", () => {
    const breadcrumbs = getBreadcrumbsFromRoute("/fields", testCtx);
    expect(breadcrumbs.map((b) => b.label)).toEqual(["Home", "Fields"]);
    expect(breadcrumbs.every((b) => !b.nonNavigable)).toBe(true);
  });

  it("ROUTES constants", () => {
    expect(ROUTE_PATTERNS.FIELDS).toBe("/fields");
    expect("FIELD_DETAIL" in ROUTE_PATTERNS).toBe(false);
  });
});
