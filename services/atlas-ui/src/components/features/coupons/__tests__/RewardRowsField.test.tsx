import { describe, it, expect, vi, beforeEach } from "vitest";
import { useState } from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RewardRowsField } from "@/components/features/coupons/RewardRowsField";
import {
  emptyRewardRow,
  type RewardRowInput,
} from "@/lib/schemas/coupons.schema";
import { itemsService } from "@/services/api/items.service";
import { commoditiesService } from "@/services/api/commodities.service";
import { itemStringsService } from "@/services/api/item-strings.service";
import type { ItemCashShopCommodity } from "@/types/models/npc";
import type { ItemSearchResult } from "@/types/models/item";

// Radix Select/Popover rely on DOM APIs jsdom does not implement.
Element.prototype.hasPointerCapture ||= () => false;
Element.prototype.scrollIntoView ||= () => {};

vi.mock("@/services/api/items.service", () => ({
  itemsService: { searchItems: vi.fn() },
}));

vi.mock("@/services/api/commodities.service", () => ({
  commoditiesService: { getByItem: vi.fn(), drainAll: vi.fn() },
}));

vi.mock("@/services/api/item-strings.service", () => ({
  itemStringsService: { getItemString: vi.fn() },
}));

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "test-tenant",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

function Harness({ initial }: { initial: RewardRowInput[] }) {
  const [rows, setRows] = useState(initial);
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <QueryClientProvider client={qc}>
      <RewardRowsField idPrefix="t" rows={rows} onChange={setRows} />
      <output data-testid="state">{JSON.stringify(rows)}</output>
    </QueryClientProvider>
  );
}

function state(): RewardRowInput[] {
  return JSON.parse(screen.getByTestId("state").textContent ?? "[]");
}

function cashItemRow(): RewardRowInput[] {
  return [{ ...emptyRewardRow(), type: "CASH_ITEM" }];
}

/**
 * The live catalog is mostly EQUIPMENT ids (serial 20000036 sells item
 * 1002077, a hat), which is why the picker filters on catalog membership
 * rather than on the "cash" compartment.
 */
function commodity(
  id: string,
  itemId: number,
  overrides: Partial<ItemCashShopCommodity> = {},
): ItemCashShopCommodity {
  return {
    id,
    itemId,
    count: 1,
    price: 3700,
    period: 90,
    priority: 5,
    gender: 2,
    onSale: true,
    ...overrides,
  };
}

function searchResult(id: string, name: string): ItemSearchResult {
  return {
    id,
    name,
    type: "Equipment",
    compartment: "equipment",
    subcategory: "hat",
  };
}

function searchPage(items: ItemSearchResult[]) {
  return {
    items,
    total: items.length,
    pageNumber: 1,
    pageSize: 50,
    lastPage: 1,
  };
}

describe("RewardRowsField", () => {
  beforeEach(() => {
    vi.mocked(itemsService.searchItems).mockReset();
    vi.mocked(commoditiesService.getByItem).mockReset();
    vi.mocked(commoditiesService.drainAll).mockReset();
    vi.mocked(itemStringsService.getItemString).mockReset();
    vi.mocked(commoditiesService.drainAll).mockResolvedValue([]);
    vi.mocked(itemStringsService.getItemString).mockRejectedValue(
      new Error("no name"),
    );
  });

  it("picks a currency by name and stores its numeric code", async () => {
    const user = userEvent.setup();
    render(<Harness initial={[emptyRewardRow()]} />);

    const trigger = screen.getByRole("combobox", { name: /currency/i });
    expect(trigger).toHaveTextContent("NX");

    await user.click(trigger);
    await user.click(await screen.findByRole("option", { name: "Prepaid" }));

    expect(state()[0]?.currency).toBe("3");
  });

  // The reported bug: clicking the searched item left the field blank, so the
  // form failed with "Serial number must be a positive whole number".
  it("selects the serial in one click when the item has exactly one", async () => {
    vi.mocked(commoditiesService.drainAll).mockResolvedValue([
      commodity("20000036", 1002077),
    ]);
    vi.mocked(itemsService.searchItems).mockResolvedValue(
      searchPage([searchResult("1002077", "Zeta Nova Hat")]),
    );

    const user = userEvent.setup();
    render(<Harness initial={cashItemRow()} />);

    await user.click(screen.getByRole("button", { name: "Cash item" }));
    await user.type(screen.getByLabelText("Search cash items"), "Zeta");
    await user.click(await screen.findByText("Zeta Nova Hat"));

    expect(state()[0]?.serialNumber).toBe("20000036");
  });

  it("asks which serial only when the item is sold under several", async () => {
    vi.mocked(commoditiesService.drainAll).mockResolvedValue([
      commodity("20000036", 1002077),
      commodity("20000037", 1002077, { count: 2, period: 0 }),
    ]);
    vi.mocked(itemsService.searchItems).mockResolvedValue(
      searchPage([searchResult("1002077", "Zeta Nova Hat")]),
    );

    const user = userEvent.setup();
    render(<Harness initial={cashItemRow()} />);

    await user.click(screen.getByRole("button", { name: "Cash item" }));
    await user.type(screen.getByLabelText("Search cash items"), "Zeta");
    await user.click(await screen.findByText("Zeta Nova Hat"));

    // Nothing is chosen yet — the item alone is ambiguous.
    expect(state()[0]?.serialNumber).toBe("");
    await user.click(await screen.findByText("20000037"));
    expect(state()[0]?.serialNumber).toBe("20000037");
  });

  // A sword or a red potion has no commodity, so it has no serial to grant.
  it("hides search matches the cash shop does not sell", async () => {
    vi.mocked(commoditiesService.drainAll).mockResolvedValue([
      commodity("20000036", 1002077),
    ]);
    vi.mocked(itemsService.searchItems).mockResolvedValue(
      searchPage([
        searchResult("1002077", "Zeta Nova Hat"),
        searchResult("1302000", "Sword"),
      ]),
    );

    const user = userEvent.setup();
    render(<Harness initial={cashItemRow()} />);

    await user.click(screen.getByRole("button", { name: "Cash item" }));
    await user.type(screen.getByLabelText("Search cash items"), "a");

    expect(await screen.findByText("Zeta Nova Hat")).toBeInTheDocument();
    expect(screen.queryByText("Sword")).not.toBeInTheDocument();
    expect(await screen.findByText(/1 match hidden/)).toBeInTheDocument();
  });

  it("accepts a serial number typed straight in", async () => {
    vi.mocked(commoditiesService.drainAll).mockResolvedValue([
      commodity("20000036", 1002077),
    ]);
    vi.mocked(itemsService.searchItems).mockResolvedValue(searchPage([]));

    const user = userEvent.setup();
    render(<Harness initial={cashItemRow()} />);

    await user.click(screen.getByRole("button", { name: "Cash item" }));
    await user.type(screen.getByLabelText("Search cash items"), "20000036");

    await user.click(await screen.findByText(/Use serial 20000036/));
    expect(state()[0]?.serialNumber).toBe("20000036");
  });
});
