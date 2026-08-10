import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { rewardPoolsService } from "@/services/api/reward-pools.service";
import { commoditiesService } from "@/services/api/commodities.service";

// Radix Popover relies on DOM APIs jsdom does not implement.
Element.prototype.hasPointerCapture ||= () => false;
Element.prototype.scrollIntoView ||= () => {};

const searchItemsMock = vi.fn();
vi.mock("@/services/api/items.service", () => ({
  itemsService: { searchItems: (...a: unknown[]) => searchItemsMock(...a) },
}));
vi.mock("@/services/api/commodities.service", () => ({
  commoditiesService: { drainAll: vi.fn() },
}));
vi.mock("@/lib/hooks/api/useItemStrings", () => ({
  useItemName: () => ({ data: undefined, isError: false }),
}));

vi.mock("@/services/api/reward-pools.service", () => ({
  rewardPoolsService: {
    createItem: vi.fn().mockResolvedValue(undefined),
    updateItem: vi.fn().mockResolvedValue(undefined),
    createGlobalItem: vi.fn().mockResolvedValue(undefined),
  },
}));
vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

import { PoolItemDialog } from "../PoolItemDialog";

/** Drives the ItemPicker: open the trigger, type an id, take the "Use id N"
 *  escape hatch (rendered from the raw box, so no debounce wait is needed). */
async function pickItem(user: ReturnType<typeof userEvent.setup>, id: number) {
  await user.click(screen.getByRole("button", { name: /select an item/i }));
  await user.type(screen.getByPlaceholderText(/search by name/i), String(id));
  await user.click(await screen.findByText(`Use id ${id}`));
}

function renderDialog(
  props: Partial<Parameters<typeof PoolItemDialog>[0]> = {},
) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <PoolItemDialog
        open
        onOpenChange={() => {}}
        kind="incubator"
        poolId="4170001"
        {...props}
      />
    </QueryClientProvider>,
  );
}

describe("PoolItemDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(commoditiesService.drainAll).mockResolvedValue([]);
    searchItemsMock.mockResolvedValue({
      items: [],
      total: 0,
      pageNumber: 1,
      pageSize: 50,
      lastPage: 1,
    });
  });

  it("renders an item picker rather than a raw id input", () => {
    renderDialog();
    expect(screen.getByRole("group", { name: /^item$/i })).toBeInTheDocument();
    expect(screen.queryByLabelText(/item id/i)).not.toBeInTheDocument();
  });

  it("searches by name and submits the picked item's id", async () => {
    const user = userEvent.setup();
    searchItemsMock.mockResolvedValue({
      items: [{ id: "2022503", name: "Red Potion", subcategory: "potion" }],
      total: 1,
      pageNumber: 1,
      pageSize: 50,
      lastPage: 1,
    });
    renderDialog();

    await user.click(screen.getByRole("button", { name: /select an item/i }));
    await user.type(screen.getByPlaceholderText(/search by name/i), "red");
    // Settles after ItemPicker's 300ms debounce, well inside findBy's timeout.
    await user.click(await screen.findByText("Red Potion"));

    await user.type(screen.getByLabelText(/quantity/i), "1");
    await user.type(screen.getByLabelText(/weight/i), "50");
    await user.click(screen.getByRole("button", { name: /save|add/i }));
    await waitFor(() =>
      expect(rewardPoolsService.createItem).toHaveBeenCalledWith("4170001", {
        itemId: 2022503,
        quantity: 1,
        tier: "common",
        weight: 50,
        commodityId: 0,
      }),
    );
  });

  it("blocks submit with the authored message when no item was picked", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.type(screen.getByLabelText(/quantity/i), "1");
    await user.type(screen.getByLabelText(/weight/i), "50");
    await user.click(screen.getByRole("button", { name: /save|add/i }));
    await waitFor(() =>
      expect(screen.getByText(/item id is required/i)).toBeInTheDocument(),
    );
    expect(rewardPoolsService.createItem).not.toHaveBeenCalled();
  });

  it("incubator mode shows a Weight field and no Tier select", () => {
    renderDialog();
    expect(screen.getByLabelText(/weight/i)).toBeInTheDocument();
    expect(screen.queryByLabelText(/tier/i)).not.toBeInTheDocument();
  });

  it("gachapon mode shows a Tier select and no Weight field", () => {
    renderDialog({ kind: "gachapon" });
    expect(screen.getByLabelText(/tier/i)).toBeInTheDocument();
    expect(screen.queryByLabelText(/weight/i)).not.toBeInTheDocument();
  });

  it("submits an incubator item with tier 'common' and the entered weight", async () => {
    const user = userEvent.setup();
    renderDialog();
    await pickItem(user, 2000000);
    await user.type(screen.getByLabelText(/quantity/i), "1");
    await user.type(screen.getByLabelText(/weight/i), "50");
    await user.click(screen.getByRole("button", { name: /save|add/i }));
    await waitFor(() =>
      expect(rewardPoolsService.createItem).toHaveBeenCalledWith("4170001", {
        itemId: 2000000,
        quantity: 1,
        tier: "common",
        weight: 50,
        commodityId: 0,
      }),
    );
  });

  it("rejects weight 0 before calling the service", async () => {
    const user = userEvent.setup();
    renderDialog();
    await pickItem(user, 2000000);
    await user.type(screen.getByLabelText(/quantity/i), "1");
    await user.type(screen.getByLabelText(/weight/i), "0");
    await user.click(screen.getByRole("button", { name: /save|add/i }));
    await waitFor(() =>
      expect(
        screen.getByText(/weight must be at least 1/i),
      ).toBeInTheDocument(),
    );
    expect(rewardPoolsService.createItem).not.toHaveBeenCalled();
  });

  it("asks a cash-surprise entry for a cash item, not a raw commodity id", () => {
    renderDialog({ kind: "cash-surprise" });
    expect(
      screen.getByRole("group", { name: /cash item/i }),
    ).toBeInTheDocument();
    expect(screen.queryByLabelText(/commodity id/i)).not.toBeInTheDocument();
    // Derived from the commodity, so it must not be typeable.
    expect(screen.getByLabelText(/quantity/i)).toHaveAttribute("readonly");
  });

  it("blocks a cash-surprise submit until a cash item is chosen", async () => {
    const user = userEvent.setup();
    renderDialog({ kind: "cash-surprise" });
    await user.type(screen.getByLabelText(/^weight/i), "50");
    await user.click(screen.getByRole("button", { name: /save|add/i }));
    await waitFor(() =>
      expect(screen.getByText(/choose a cash item/i)).toBeInTheDocument(),
    );
    expect(rewardPoolsService.createItem).not.toHaveBeenCalled();
  });

  // The whole point of the picker: itemId and quantity are the COMMODITY's,
  // because that is what atlas-cashshop grants (surprise/processor.go grants
  // ci.ItemId() x ci.Count(), ignoring the entry's own values).
  it("derives itemId and quantity from the chosen commodity", async () => {
    vi.mocked(commoditiesService.drainAll).mockResolvedValue([
      {
        id: "50200133",
        itemId: 5222000,
        count: 11,
        price: 5000,
        period: 90,
        priority: 16,
        gender: 2,
        onSale: false,
      },
    ]);
    searchItemsMock.mockResolvedValue({
      items: [
        {
          id: "5222000",
          name: "Cash Shop Surprise",
          subcategory: "other-cash",
        },
      ],
      total: 1,
      pageNumber: 1,
      pageSize: 50,
      lastPage: 1,
    });

    const user = userEvent.setup();
    renderDialog({ kind: "cash-surprise" });

    await user.click(
      screen.getByRole("button", { name: /select a cash item/i }),
    );
    await user.type(screen.getByPlaceholderText(/search by name/i), "surprise");
    await user.click(await screen.findByText("Cash Shop Surprise"));

    expect(screen.getByLabelText(/quantity/i)).toHaveValue(11);

    await user.type(screen.getByLabelText(/^weight/i), "50");
    await user.click(screen.getByRole("button", { name: /save|add/i }));
    await waitFor(() =>
      expect(rewardPoolsService.createItem).toHaveBeenCalledWith("4170001", {
        itemId: 5222000,
        quantity: 11,
        tier: "common",
        weight: 50,
        commodityId: 50200133,
      }),
    );
  });

  it("does not send a commodityId for incubator entries", async () => {
    const user = userEvent.setup();
    renderDialog();
    await pickItem(user, 2000000);
    await user.type(screen.getByLabelText(/quantity/i), "1");
    await user.type(screen.getByLabelText(/^weight/i), "50");
    expect(screen.queryByLabelText(/commodity id/i)).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /save|add/i }));
    await waitFor(() =>
      expect(rewardPoolsService.createItem).toHaveBeenCalledWith("4170001", {
        itemId: 2000000,
        quantity: 1,
        tier: "common",
        weight: 50,
        commodityId: 0,
      }),
    );
  });
});
