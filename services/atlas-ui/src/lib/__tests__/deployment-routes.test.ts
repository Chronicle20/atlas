import { describe, it, expect } from "vitest";
import { isDeploymentRoute } from "@/lib/deployment-routes";

describe("isDeploymentRoute", () => {
  it.each([
    "/templates",
    "/templates/abc123/writers",
    "/tenants",
    "/tenants/9f8e/properties",
    "/tenants/9f8e/character/presets",
    "/services",
    "/services/atlas-data",
    "/baselines",
  ])("returns true for deployment route %s", (path) => {
    expect(isDeploymentRoute(path)).toBe(true);
  });

  it.each([
    "/",
    "/setup",
    "/accounts",
    "/characters/42",
    "/servicesx", // prefix guard: no false positive on sibling names
    "/templatesfoo",
    "/baselines-old",
  ])("returns false for non-deployment route %s", (path) => {
    expect(isDeploymentRoute(path)).toBe(false);
  });
});
