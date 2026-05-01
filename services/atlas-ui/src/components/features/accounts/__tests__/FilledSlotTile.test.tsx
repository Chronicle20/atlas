// services/atlas-ui/src/components/features/accounts/__tests__/FilledSlotTile.test.tsx
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { Character } from "@/types/models/character";

vi.mock("@/components/features/characters/CharacterRenderer", () => ({
  CharacterRenderer: ({
    character,
    inventory,
  }: {
    character: Character;
    inventory?: unknown[];
  }) => (
    <div
      data-testid="renderer"
      data-name={character.attributes.name}
      data-inventory-count={inventory?.length ?? 0}
    />
  ),
}));

const inventoryMock = vi.fn();
vi.mock("@/lib/hooks/api/useInventory", () => ({
  useInventory: () => inventoryMock(),
}));

import { FilledSlotTile } from "../FilledSlotTile";

const tenant = {
  id: "t1",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
} as never;

const character = (worldId: number): Character =>
  ({
    id: "5",
    type: "characters",
    attributes: {
      accountId: 1,
      worldId,
      name: "Foo",
    },
  }) as unknown as Character;

const worlds = [
  { name: "Scania", flag: "0", serverMessage: "", eventMessage: "", whyAmIRecommended: "" },
];

function renderTile(props: { character: Character; worlds: typeof worlds }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={qc}>
        <FilledSlotTile tenant={tenant} {...props} />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("FilledSlotTile", () => {
  it("renders a link to /characters/{id} with character renderer + name", () => {
    inventoryMock.mockReturnValue({ data: undefined });
    renderTile({ character: character(0), worlds });
    const link = screen.getByRole("link", { name: /Foo/i });
    expect(link).toHaveAttribute("href", "/characters/5");
    expect(screen.getByTestId("renderer")).toBeInTheDocument();
    expect(screen.getByText("Foo")).toBeInTheDocument();
  });

  it("forwards equipped assets (slot < 0) from inventory.included to CharacterRenderer", () => {
    inventoryMock.mockReturnValue({
      data: {
        included: [
          { type: "assets", attributes: { slot: -1 } },
          { type: "assets", attributes: { slot: -11 } },
          { type: "assets", attributes: { slot: 5 } },
          { type: "compartments", attributes: { type: 1 } },
        ],
      },
    });
    renderTile({ character: character(0), worlds });
    expect(screen.getByTestId("renderer")).toHaveAttribute(
      "data-inventory-count",
      "2",
    );
  });

  it("renders the world name in the caption when the world is configured", () => {
    inventoryMock.mockReturnValue({ data: undefined });
    renderTile({ character: character(0), worlds });
    expect(screen.getByText("Scania")).toBeInTheDocument();
  });

  it("hides the world caption when worldId is out of range", () => {
    inventoryMock.mockReturnValue({ data: undefined });
    renderTile({ character: character(7), worlds });
    expect(screen.queryByText("Scania")).toBeNull();
  });

  it("renders the extracted world icon next to the world name", () => {
    inventoryMock.mockReturnValue({ data: undefined });
    const { container } = renderTile({ character: character(0), worlds });
    const img = container.querySelector("img");
    expect(img).not.toBeNull();
    // URL shape comes from getWorldIconUrl(tenantId, region, major, minor, worldId).
    expect(img!.getAttribute("src")).toContain(
      "/t1/GMS/83.1/world-icon/0/icon.png",
    );
  });

  it("hides the icon if the asset 404s (onError fallback)", () => {
    inventoryMock.mockReturnValue({ data: undefined });
    const { container } = renderTile({ character: character(0), worlds });
    const img = container.querySelector("img");
    expect(img).not.toBeNull();
    fireEvent.error(img!);
    expect(container.querySelector("img")).toBeNull();
    // World name still rendered.
    expect(screen.getByText("Scania")).toBeInTheDocument();
  });
});
