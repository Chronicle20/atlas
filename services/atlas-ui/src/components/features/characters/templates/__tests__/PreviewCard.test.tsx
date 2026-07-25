import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

const useCharacterImageMock = vi.fn();
vi.mock("@/lib/hooks/useCharacterImage", () => ({
  useCharacterImage: (...a: unknown[]) => useCharacterImageMock(...a),
}));

const useItemNameMock = vi.fn();
vi.mock("@/lib/hooks/api/useItemStrings", () => ({
  useItemName: (...a: unknown[]) => useItemNameMock(...a),
  itemStringKeys: {
    all: ["item-strings"],
    byId: (id: string) => ["item-strings", "name", id],
  },
}));

import { normalizeTemplate, DEFAULT_PICKS } from "../editorState";
import { PreviewCard } from "../PreviewCard";

const tpl = normalizeTemplate({
  gender: 0,
  faces: [20000],
  hairs: [30030],
  hairColors: [2],
  skinColors: [1],
  tops: [1041002, 1041003],
  weapons: [1302000],
});

beforeEach(() => {
  useItemNameMock.mockReturnValue({ data: "Item", isError: false });
  useCharacterImageMock.mockReturnValue({
    imageUrl: "/api/assets/t1/GMS/83.1/character/abc.png",
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  });
});

describe("PreviewCard", () => {
  it("builds the loadout from picks + first-of-pool equipment", () => {
    render(<PreviewCard template={tpl} picks={DEFAULT_PICKS} />);
    const character = useCharacterImageMock.mock.calls[0]![0];
    expect(character).toMatchObject({
      tenant: "t1",
      region: "GMS",
      majorVersion: 83,
      minorVersion: 1,
      skinColor: 1,
      hair: 30032, // 30030 + digit 2
      face: 20000,
      gender: 0,
      equipment: { "-5": 1041002, "-11": 1302000 },
    });
    const options = useCharacterImageMock.mock.calls[0]![1];
    expect(options).toMatchObject({ stance: "stand1", resize: 2 });
  });

  it("renders the composited image and the worn-equipment icon strip", () => {
    render(<PreviewCard template={tpl} picks={DEFAULT_PICKS} />);
    expect(screen.getByRole("img", { name: /live preview/i })).toHaveAttribute(
      "src",
      "/api/assets/t1/GMS/83.1/character/abc.png",
    );
    // first-of-pool only: tops + weapons = 2 worn icons
    expect(screen.getAllByTestId("worn-icon")).toHaveLength(2);
  });

  it("shows the error + retry state when the render fails", () => {
    const refetch = vi.fn();
    useCharacterImageMock.mockReturnValue({
      imageUrl: undefined,
      isLoading: false,
      isError: true,
      refetch,
    });
    render(<PreviewCard template={tpl} picks={DEFAULT_PICKS} />);
    expect(screen.getByText(/preview failed/i)).toBeInTheDocument();
    screen.getByRole("button", { name: /retry/i }).click();
    expect(refetch).toHaveBeenCalled();
  });
});
