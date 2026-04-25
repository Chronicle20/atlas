import type { ReactNode } from "react";
import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AssetTooltipContent } from "../AssetTooltipContent";
import type { Asset } from "@/services/api/inventory.service";

const useItemDataMock = vi.fn();
const useEquipmentDataMock = vi.fn();

vi.mock("@/lib/hooks/useItemData", () => ({
  useItemData: (...a: unknown[]) => useItemDataMock(...a),
}));
vi.mock("@/lib/hooks/api/useEquipmentData", () => ({
  useEquipmentData: (...a: unknown[]) => useEquipmentDataMock(...a),
}));

const baseAsset = (overrides: Partial<Asset["attributes"]> = {}): Asset => ({
  type: "assets",
  id: "999",
  attributes: {
    id: 999,
    slot: -1,
    templateId: 1062001, // Brown Cotton Shorts (BOTTOM)
    expiration: "0001-01-01T00:00:00Z",
    createdAt: "0001-01-01T00:00:00Z",
    quantity: 1,
    ownerId: 0,
    flag: 0,
    rechargeable: 0,
    strength: 0,
    dexterity: 0,
    intelligence: 0,
    luck: 0,
    hp: 0,
    mp: 0,
    weaponAttack: 0,
    magicAttack: 0,
    weaponDefense: 0,
    magicDefense: 0,
    accuracy: 0,
    avoidability: 0,
    hands: 0,
    speed: 0,
    jump: 0,
    slots: 7,
    levelType: 0,
    level: 0,
    experience: 0,
    hammersApplied: 0,
    equippedSince: "0001-01-01T00:00:00Z",
    cashId: "",
    commodityId: 0,
    purchaseBy: 0,
    petId: 0,
    ...overrides,
  },
});

const baseEquipmentData = (overrides: Record<string, number> = {}) => ({
  data: {
    id: "1062001",
    attributes: {
      strength: 0,
      dexterity: 0,
      intelligence: 0,
      luck: 0,
      hp: 0,
      mp: 0,
      weaponAttack: 0,
      magicAttack: 0,
      weaponDefense: 3,
      magicDefense: 0,
      accuracy: 0,
      avoidability: 0,
      speed: 0,
      jump: 0,
      slots: 7,
      reqLevel: 0,
      reqJob: 0,
      reqStr: 0,
      reqDex: 0,
      reqInt: 0,
      reqLuk: 0,
      reqPop: 0,
      reqFame: 0,
      cash: false,
      price: 0,
      timeLimited: false,
      ...overrides,
    },
  },
  isSuccess: true,
  isLoading: false,
  isError: false,
  error: null,
});

function renderTooltip(node: ReactNode) {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(<QueryClientProvider client={qc}>{node}</QueryClientProvider>);
}

describe("AssetTooltipContent", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useItemDataMock.mockReturnValue({
      name: "Brown Cotton Shorts",
      iconUrl: "https://example.test/icon.png",
    });
    useEquipmentDataMock.mockReturnValue(baseEquipmentData());
  });

  it("renders the resolved item name from useItemData", () => {
    renderTooltip(<AssetTooltipContent asset={baseAsset()} />);
    expect(screen.getByText("Brown Cotton Shorts")).toBeInTheDocument();
  });

  it("falls back to 'Item #<templateId>' when name is unavailable", () => {
    useItemDataMock.mockReturnValue({ name: undefined, iconUrl: undefined });
    renderTooltip(<AssetTooltipContent asset={baseAsset()} />);
    expect(screen.getByText("Item #1062001")).toBeInTheDocument();
  });

  it("shows the fixed required-stats list (LEV/POP/STR/DEX/INT/LUK) for equipment, even when all are 0", () => {
    renderTooltip(<AssetTooltipContent asset={baseAsset()} />);
    ["REQ LEV", "REQ POP", "REQ STR", "REQ DEX", "REQ INT", "REQ LUK"].forEach((label) => {
      expect(screen.getByText(label)).toBeInTheDocument();
    });
  });

  it("hides REQ FAM when reqFame is 0", () => {
    renderTooltip(<AssetTooltipContent asset={baseAsset()} />);
    expect(screen.queryByText("REQ FAM")).not.toBeInTheDocument();
  });

  it("highlights all six job badges when reqJob is 0 (Beginner-tier)", () => {
    renderTooltip(<AssetTooltipContent asset={baseAsset()} />);
    ["BEGINNER", "WARRIOR", "MAGICIAN", "BOWMAN", "THIEF", "PIRATE"].forEach((j) => {
      const badge = screen.getByText(j);
      expect(badge).toHaveClass("bg-orange-500/80");
    });
  });

  it("highlights only the bitmask-active classes when reqJob is non-zero", () => {
    useEquipmentDataMock.mockReturnValue(baseEquipmentData({ reqJob: 8 })); // Thief only
    renderTooltip(<AssetTooltipContent asset={baseAsset()} />);
    expect(screen.getByText("THIEF")).toHaveClass("bg-orange-500/80");
    expect(screen.getByText("WARRIOR")).not.toHaveClass("bg-orange-500/80");
    expect(screen.getByText("BEGINNER")).not.toHaveClass("bg-orange-500/80");
  });

  it("renders the derived equipment category (BOTTOM for 1062xxx)", () => {
    renderTooltip(<AssetTooltipContent asset={baseAsset()} />);
    expect(screen.getByText("CATEGORY:")).toBeInTheDocument();
    expect(screen.getByText("BOTTOM")).toBeInTheDocument();
  });

  it("shows non-zero stats from the asset and omits zero ones", () => {
    renderTooltip(
      <AssetTooltipContent asset={baseAsset({ weaponDefense: 3, strength: 5 })} />,
    );
    expect(screen.getByText("WEAPON DEF")).toBeInTheDocument();
    expect(screen.getByText("STR")).toBeInTheDocument();
    expect(screen.queryByText("MAGIC DEF")).not.toBeInTheDocument();
    expect(screen.queryByText("DEX")).not.toBeInTheDocument();
  });

  it("always shows UPGRADES AVAILABLE and HAMMERS APPLIED for equipment", () => {
    renderTooltip(<AssetTooltipContent asset={baseAsset({ slots: 7, hammersApplied: 0 })} />);
    expect(screen.getByText(/UPGRADES AVAILABLE:/)).toBeInTheDocument();
    expect(screen.getByText(/HAMMERS APPLIED:/)).toBeInTheDocument();
  });

  it("does not render Asset ID, Slot, or Quantity for an equipment asset", () => {
    renderTooltip(<AssetTooltipContent asset={baseAsset()} />);
    expect(screen.queryByText(/Asset ID/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/^Slot:/)).not.toBeInTheDocument();
    expect(screen.queryByText(/QUANTITY/i)).not.toBeInTheDocument();
  });

  it("shows quantity for non-equipment items when > 1", () => {
    renderTooltip(
      <AssetTooltipContent asset={baseAsset({ templateId: 2000000, quantity: 50 })} />,
    );
    expect(screen.getByText(/QUANTITY:/)).toBeInTheDocument();
    expect(screen.getByText("50")).toBeInTheDocument();
  });

  it("hides expiration row for the zero-date sentinel and shows real dates", () => {
    const { rerender } = renderTooltip(<AssetTooltipContent asset={baseAsset()} />);
    expect(screen.queryByText(/EXPIRES:/)).not.toBeInTheDocument();

    rerender(
      <QueryClientProvider client={new QueryClient()}>
        <AssetTooltipContent asset={baseAsset({ expiration: "2027-01-01T00:00:00Z" })} />
      </QueryClientProvider>,
    );
    expect(screen.getByText(/EXPIRES:/)).toBeInTheDocument();
  });

  it("renders the slotName override beside the name", () => {
    renderTooltip(<AssetTooltipContent asset={baseAsset()} slotName="Bottom" />);
    expect(screen.getByText("(Bottom)")).toBeInTheDocument();
  });
});
