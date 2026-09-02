/**
 * Tests for the /fields route configuration (runtime read model, FR-17/FR-18).
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

  it("full field detail trail", () => {
    const breadcrumbs = getBreadcrumbsFromRoute(
      "/fields/0/1/910340000/00000000-0000-0000-0000-000000000000",
      testCtx,
    );
    expect(breadcrumbs.map((b) => b.label)).toEqual([
      "Home",
      "Fields",
      "World 0",
      "Channel 1",
      "910340000",
      "Instance 00000000-0000-0000-0000-000000000000",
    ]);
  });

  it("intermediates are non-navigable", () => {
    const breadcrumbs = getBreadcrumbsFromRoute(
      "/fields/0/1/910340000/00000000-0000-0000-0000-000000000000",
      testCtx,
    );
    const byLabel = new Map(breadcrumbs.map((b) => [b.label, b]));

    expect(byLabel.get("World 0")?.nonNavigable).toBe(true);
    expect(byLabel.get("Channel 1")?.nonNavigable).toBe(true);
    expect(byLabel.get("910340000")?.nonNavigable).toBe(true);

    expect(byLabel.get("Fields")?.nonNavigable).toBeUndefined();
    expect(
      byLabel.get("Instance 00000000-0000-0000-0000-000000000000")
        ?.nonNavigable,
    ).toBeUndefined();
  });

  it("ROUTES constants", () => {
    expect(ROUTE_PATTERNS.FIELDS).toBe("/fields");
    expect(ROUTE_PATTERNS.FIELD_DETAIL).toBe(
      "/fields/[worldId]/[channelId]/[mapId]/[instanceId]",
    );
  });
});
