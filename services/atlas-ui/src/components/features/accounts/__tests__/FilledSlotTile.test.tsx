// services/atlas-ui/src/components/features/accounts/__tests__/FilledSlotTile.test.tsx
import { render, screen } from "@testing-library/react";
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
});
