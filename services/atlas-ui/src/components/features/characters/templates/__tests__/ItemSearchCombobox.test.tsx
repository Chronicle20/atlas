import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

const searchItemsMock = vi.fn();
vi.mock("@/services/api/items.service", () => ({
  itemsService: { searchItems: (...a: unknown[]) => searchItemsMock(...a) },
}));

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

import { ItemSearchCombobox } from "../ItemSearchCombobox";

function renderBox(
  props: Partial<React.ComponentProps<typeof ItemSearchCombobox>> = {},
) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <ItemSearchCombobox
        poolKey="weapons"
        existingIds={[]}
        onAdd={vi.fn()}
        debounceMs={0}
        {...props}
      />
    </QueryClientProvider>,
  );
}

const page = (items: unknown[]) => ({
  items,
  total: items.length,
  pageNumber: 1,
  pageSize: 50,
  lastPage: 1,
});

beforeEach(() => searchItemsMock.mockReset());

describe("ItemSearchCombobox", () => {
  it("searches with the pool's server filters and adds a clicked row", async () => {
    searchItemsMock.mockResolvedValue(
      page([
        {
          id: "1302000",
          name: "Sword",
          compartment: "equipment",
          subcategory: "one-handed-sword",
          type: "Equipment",
        },
      ]),
    );
    const onAdd = vi.fn();
    renderBox({ poolKey: "bottoms", onAdd });
    await userEvent.click(screen.getByRole("button", { name: /add/i }));
    await userEvent.type(screen.getByRole("textbox"), "pants");
    await waitFor(() =>
      expect(searchItemsMock).toHaveBeenCalledWith(
        expect.objectContaining({
          q: "pants",
          compartment: "equipment",
          subcategory: "bottom",
          pageNumber: 1,
          pageSize: 50,
        }),
      ),
    );
    await userEvent.click(await screen.findByRole("option", { name: /Sword/ }));
    expect(onAdd).toHaveBeenCalledWith(1302000);
  });

  it("client-filters weapons to the 16 weapon subcategories", async () => {
    searchItemsMock.mockResolvedValue(
      page([
        {
          id: "1302000",
          name: "Sword",
          compartment: "equipment",
          subcategory: "one-handed-sword",
          type: "Equipment",
        },
        {
          id: "1802000",
          name: "Pet Leash",
          compartment: "equipment",
          subcategory: "pet-equip",
          type: "Equipment",
        },
      ]),
    );
    renderBox();
    await userEvent.click(screen.getByRole("button", { name: /add/i }));
    await userEvent.type(screen.getByRole("textbox"), "s");
    expect(
      await screen.findByRole("option", { name: /Sword/ }),
    ).toBeInTheDocument();
    expect(screen.queryByText(/Pet Leash/)).not.toBeInTheDocument();
  });

  it("offers the manual Use id fallback for numeric input", async () => {
    searchItemsMock.mockResolvedValue(page([]));
    const onAdd = vi.fn();
    renderBox({ onAdd });
    await userEvent.click(screen.getByRole("button", { name: /add/i }));
    await userEvent.type(screen.getByRole("textbox"), "1402001");
    await userEvent.click(
      await screen.findByRole("option", { name: /use id 1402001/i }),
    );
    expect(onAdd).toHaveBeenCalledWith(1402001);
  });

  it("marks rows already in the pool and does not re-add them", async () => {
    searchItemsMock.mockResolvedValue(
      page([
        {
          id: "1302000",
          name: "Sword",
          compartment: "equipment",
          subcategory: "one-handed-sword",
          type: "Equipment",
        },
      ]),
    );
    const onAdd = vi.fn();
    renderBox({ existingIds: [1302000], onAdd });
    await userEvent.click(screen.getByRole("button", { name: /add/i }));
    await userEvent.type(screen.getByRole("textbox"), "s");
    const row = await screen.findByRole("option", { name: /Sword/ });
    expect(row).toHaveAttribute("aria-disabled", "true");
    await userEvent.click(row);
    expect(onAdd).not.toHaveBeenCalled();
  });
});
