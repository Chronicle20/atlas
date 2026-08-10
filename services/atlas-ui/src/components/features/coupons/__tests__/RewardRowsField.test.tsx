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

// Radix Select/Popover rely on DOM APIs jsdom does not implement.
Element.prototype.hasPointerCapture ||= () => false;
Element.prototype.scrollIntoView ||= () => {};

vi.mock("@/services/api/items.service", () => ({
  itemsService: { searchItems: vi.fn() },
}));

vi.mock("@/services/api/commodities.service", () => ({
  commoditiesService: { getByItem: vi.fn(), getBySerialNumber: vi.fn() },
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

describe("RewardRowsField", () => {
  beforeEach(() => {
    vi.mocked(itemsService.searchItems).mockReset();
    vi.mocked(commoditiesService.getByItem).mockReset();
    vi.mocked(commoditiesService.getBySerialNumber).mockReset();
    vi.mocked(itemStringsService.getItemString).mockReset();
    vi.mocked(commoditiesService.getBySerialNumber).mockRejectedValue(
      new Error("no commodity"),
    );
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

  it("translates an item search pick into a commodity serial number", async () => {
    vi.mocked(itemsService.searchItems).mockResolvedValue({
      items: [
        {
          id: "5010000",
          name: "Wizet Hat",
          type: "Cash",
          compartment: "cash",
          subcategory: "",
        },
      ],
      total: 1,
      pageNumber: 1,
      pageSize: 50,
      lastPage: 1,
    });
    vi.mocked(commoditiesService.getByItem).mockResolvedValue([
      {
        id: "50200004",
        itemId: 5010000,
        count: 1,
        price: 1800,
        period: 30,
        priority: 0,
        gender: 2,
        onSale: true,
      },
    ]);

    const user = userEvent.setup();
    render(<Harness initial={[{ ...emptyRewardRow(), type: "CASH_ITEM" }]} />);

    await user.click(screen.getByRole("button", { name: "Cash item" }));
    await user.type(screen.getByLabelText("Search cash items"), "Wizet");

    // Item first…
    await user.click(await screen.findByText("Wizet Hat"));
    // …then the specific commodity, because one item can have several serials.
    await user.click(await screen.findByText("50200004"));

    expect(state()[0]?.serialNumber).toBe("50200004");
    expect(commoditiesService.getByItem).toHaveBeenCalledWith("5010000");
  });

  it("accepts a serial number typed straight in", async () => {
    const user = userEvent.setup();
    render(<Harness initial={[{ ...emptyRewardRow(), type: "CASH_ITEM" }]} />);

    await user.click(screen.getByRole("button", { name: "Cash item" }));
    await user.type(
      screen.getByLabelText(/enter a serial number directly/i),
      "50200009",
    );
    await user.click(screen.getByRole("button", { name: "Use" }));

    expect(state()[0]?.serialNumber).toBe("50200009");
  });
});
