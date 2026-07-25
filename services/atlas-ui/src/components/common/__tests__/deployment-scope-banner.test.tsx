import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { DeploymentScopeBanner } from "@/components/common/deployment-scope-banner";

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <DeploymentScopeBanner />
    </MemoryRouter>,
  );
}

describe("DeploymentScopeBanner", () => {
  it.each([
    "/templates",
    "/templates/abc/writers",
    "/tenants/9f8e/character/presets",
    "/services",
    "/baselines",
  ])("shows the banner on deployment route %s (including subpages)", (path) => {
    renderAt(path);
    expect(
      screen.getByText("Changes on this page affect all tenants."),
    ).toBeInTheDocument();
  });

  it.each(["/", "/setup", "/accounts", "/characters/42"])(
    "renders nothing on %s",
    (path) => {
      renderAt(path);
      expect(
        screen.queryByText("Changes on this page affect all tenants."),
      ).not.toBeInTheDocument();
    },
  );
});
