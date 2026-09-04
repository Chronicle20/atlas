import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";

import { MapDetailTabs } from "../MapDetailTabs";

// FR-30: `?tab=monsters` on the map-definition route must open the
// Monster Spawns tab (not Portals) so a link from elsewhere (e.g. the
// field-detail Monsters tab's spawn cell) lands where it says it does.
function renderTabs(initialPath: string) {
  return render(
    <MemoryRouter initialEntries={[initialPath]}>
      <MapDetailTabs
        mapId="910340000"
        portals={[]}
        monsters={[]}
        reactors={[]}
        objects={[]}
      />
    </MemoryRouter>,
  );
}

describe("MapDetailTabs", () => {
  it("?tab=monsters opens the monsters tab", () => {
    renderTabs("/maps/910340000?tab=monsters");

    expect(
      screen.getByRole("tab", { name: /monster spawns/i }),
    ).toHaveAttribute("data-state", "active");
    expect(screen.getByRole("tab", { name: /^portals/i })).toHaveAttribute(
      "data-state",
      "inactive",
    );
  });

  it("no ?tab= still opens Portals", () => {
    renderTabs("/maps/910340000");

    expect(screen.getByRole("tab", { name: /^portals/i })).toHaveAttribute(
      "data-state",
      "active",
    );
  });
});
