import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

const searchItemsMock = vi.fn();
vi.mock("@/services/api/items.service", () => ({
  itemsService: { searchItems: (...a: unknown[]) => searchItemsMock(...a) },
}));

const useItemNameMock = vi.fn();
vi.mock("@/lib/hooks/api/useItemStrings", () => ({
  useItemName: (...a: unknown[]) => useItemNameMock(...a),
}));

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

import { NpcShopCommodityDialog } from "../NpcShopCommodityDialog";
import type { CommodityAttributes } from "@/types/models/npc";

const EXISTING: CommodityAttributes = {
  templateId: 2022503,
  mesoPrice: 0,
  discountRate: 0,
  tokenTemplateId: 4310000,
  tokenPrice: 5,
  period: 0,
  levelLimit: 0,
};

function renderDialog(
  props: Partial<React.ComponentProps<typeof NpcShopCommodityDialog>> = {},
) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const onSubmit = props.onSubmit ?? vi.fn();
  const utils = render(
    <QueryClientProvider client={client}>
      <NpcShopCommodityDialog
        open
        onOpenChange={vi.fn()}
        mode="create"
        {...props}
        onSubmit={onSubmit}
      />
    </QueryClientProvider>,
  );
  return { ...utils, onSubmit };
}

const page = (items: unknown[]) => ({
  items,
  total: items.length,
  pageNumber: 1,
  pageSize: 50,
  lastPage: 1,
});

beforeEach(() => {
  searchItemsMock.mockReset();
  searchItemsMock.mockResolvedValue(page([]));
  useItemNameMock.mockReset();
  useItemNameMock.mockReturnValue({ data: undefined, isError: false });
});

describe("NpcShopCommodityDialog", () => {
  it("create mode submits the exact CommodityAttributes shape a picker-chosen id produces", async () => {
    const user = userEvent.setup();
    const { onSubmit } = renderDialog({ mode: "create" });

    // Choose the item via the Template ID picker's raw-id escape hatch.
    await user.click(screen.getByRole("button", { name: "Select an item…" }));
    await user.type(screen.getByPlaceholderText(/search by name/i), "2022503");
    await user.click(await screen.findByText("Use id 2022503"));

    // Choose the token item the same way.
    await user.click(screen.getByRole("button", { name: "None" }));
    await user.type(screen.getByPlaceholderText(/search by name/i), "4310000");
    await user.click(await screen.findByText("Use id 4310000"));

    const tokenPrice = screen.getByLabelText("Token Price");
    await user.clear(tokenPrice);
    await user.type(tokenPrice, "5");

    await user.click(screen.getByRole("button", { name: "Create" }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit).toHaveBeenCalledWith({
      templateId: 2022503,
      mesoPrice: 0,
      discountRate: 0,
      tokenTemplateId: 4310000,
      tokenPrice: 5,
      period: 0,
      levelLimit: 0,
    });
  });

  it("edit mode renders the template id as read-only text, not a picker", () => {
    useItemNameMock.mockReturnValue({ data: "Red Potion", isError: false });
    renderDialog({ mode: "edit", initial: EXISTING });

    expect(screen.getByText("Red Potion · 2022503")).toBeInTheDocument();
    // The only remaining item-picker trigger is the token one.
    expect(
      screen.queryByRole("button", { name: /Red Potion · 2022503/ }),
    ).not.toBeInTheDocument();
  });

  it("edit mode's token picker is interactive and clears to 0", async () => {
    const user = userEvent.setup();
    useItemNameMock.mockReturnValue({ data: "Perfect Pitch", isError: false });
    const { onSubmit } = renderDialog({ mode: "edit", initial: EXISTING });

    await user.click(
      screen.getByRole("button", { name: "Perfect Pitch · 4310000" }),
    );
    await user.click(screen.getByText("None"));
    await user.click(screen.getByRole("button", { name: "Update" }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit).toHaveBeenCalledWith({ ...EXISTING, tokenTemplateId: 0 });
  });

  it("submits an unedited commodity byte-for-byte unchanged", async () => {
    const user = userEvent.setup();
    const { onSubmit } = renderDialog({ mode: "edit", initial: EXISTING });

    await user.click(screen.getByRole("button", { name: "Update" }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit).toHaveBeenCalledWith(EXISTING);
  });
});
