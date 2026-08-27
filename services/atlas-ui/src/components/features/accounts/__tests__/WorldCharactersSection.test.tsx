// services/atlas-ui/src/components/features/accounts/__tests__/WorldCharactersSection.test.tsx
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import type { Character } from "@/types/models/character";
import type { Account } from "@/types/models/account";
import type { Tenant } from "@/types/models/tenant";

vi.mock("../FilledSlotTile", () => ({
  FilledSlotTile: ({ character }: { character: Character }) => (
    <div data-testid="filled-slot">{character.attributes.name}</div>
  ),
}));

vi.mock("../EmptySlotTile", () => ({
  EmptySlotTile: () => <div data-testid="empty-slot" />,
}));

const useCharacterSlotsMock = vi.fn();
vi.mock("@/lib/hooks/api/useCharacterSlots", () => ({
  useCharacterSlots: (...a: unknown[]) => useCharacterSlotsMock(...a),
}));

import { WorldCharactersSection } from "../WorldCharactersSection";

const tenant = {
  id: "t1",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
} as unknown as Tenant;

const account = { id: "1" } as unknown as Account;

const character = (id: string, worldId: number, name = "Foo"): Character =>
  ({
    id,
    type: "characters",
    attributes: { accountId: 1, worldId, name },
  }) as unknown as Character;

function renderSection(worldId = 0) {
  return render(
    <MemoryRouter>
      <WorldCharactersSection
        tenant={tenant}
        account={account}
        worldId={worldId}
        worldName="Scania"
        characters={[character("5", worldId)]}
        charactersLoading={false}
        charactersError={null}
        hasPresets
        onAddClick={() => {}}
      />
    </MemoryRouter>,
  );
}

describe("WorldCharactersSection", () => {
  it("renders the world icon next to the world name in the section heading", () => {
    useCharacterSlotsMock.mockReturnValue({
      data: { attributes: { worldId: 0, slots: 3 } },
      isLoading: false,
      error: null,
    });
    const { container } = renderSection();
    const heading = screen.getByText("Scania").closest("h3")!;
    const img = heading.querySelector("img");
    expect(img).not.toBeNull();
    // URL shape comes from getWorldIconUrl(tenantId, region, major, minor, worldId).
    expect(img!.getAttribute("src")).toContain(
      "/t1/GMS/83.1/world-icon/0/icon.png",
    );
    expect(
      container.querySelector("[data-testid='filled-slot']"),
    ).toBeInTheDocument();
  });

  it("hides the icon if the asset 404s (onError fallback), world name still shown", () => {
    useCharacterSlotsMock.mockReturnValue({
      data: { attributes: { worldId: 0, slots: 3 } },
      isLoading: false,
      error: null,
    });
    renderSection();
    const heading = screen.getByText("Scania").closest("h3")!;
    const img = heading.querySelector("img")!;
    fireEvent.error(img);
    expect(heading.querySelector("img")).toBeNull();
    expect(screen.getByText("Scania")).toBeInTheDocument();
  });
});
