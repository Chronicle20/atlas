import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import type { Character } from "@/types/models/character";

const getByIdMock = vi.fn();
vi.mock("@/services/api/characters.service", () => ({
  charactersService: {
    getById: (characterId: string) => getByIdMock(characterId),
  },
}));

import { OwnerNameCell } from "../owner-name-cell";

const tenant = {
  id: "t1",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
} as never;

const character = (id: string, name: string): Character =>
  ({ id, attributes: { name } }) as unknown as Character;

describe("OwnerNameCell", () => {
  beforeEach(() => {
    getByIdMock.mockReset();
  });

  it("resolves and renders the character name", async () => {
    getByIdMock.mockResolvedValue(character("1001", "Hero"));
    render(<OwnerNameCell characterId="1001" tenant={tenant} />);
    // Falls back to the id until the name resolves.
    expect(screen.getByText("1001")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByText("Hero")).toBeInTheDocument());
    expect(getByIdMock).toHaveBeenCalledWith("1001");
  });

  it("falls back to the id when the lookup fails", async () => {
    getByIdMock.mockRejectedValue(new Error("boom"));
    render(<OwnerNameCell characterId="2002" tenant={tenant} />);
    await waitFor(() => expect(getByIdMock).toHaveBeenCalledWith("2002"));
    expect(screen.getByText("2002")).toBeInTheDocument();
  });

  it("does not fetch when no tenant is selected", () => {
    render(<OwnerNameCell characterId="3003" tenant={null} />);
    expect(screen.getByText("3003")).toBeInTheDocument();
    expect(getByIdMock).not.toHaveBeenCalled();
  });
});
