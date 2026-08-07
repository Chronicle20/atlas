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

import { ItemPicker } from "../ItemPicker";

function renderPicker(
  props: Partial<React.ComponentProps<typeof ItemPicker>> = {},
) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const onChange = props.onChange ?? vi.fn();
  const utils = render(
    <QueryClientProvider client={client}>
      <ItemPicker value={0} debounceMs={0} {...props} onChange={onChange} />
    </QueryClientProvider>,
  );
  return { ...utils, onChange };
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
  useItemNameMock.mockReset();
  useItemNameMock.mockReturnValue({ data: undefined, isError: false });
});

describe("ItemPicker", () => {
  it("renders the placeholder when unset and does not look up item 0", () => {
    renderPicker({ value: 0, placeholder: "None" });

    expect(screen.getByRole("button", { name: "None" })).toBeInTheDocument();
    expect(useItemNameMock).toHaveBeenCalledWith("");
  });

  it("renders the resolved name and id once the lookup settles", () => {
    useItemNameMock.mockReturnValue({ data: "Perfect Pitch", isError: false });
    renderPicker({ value: 4310000 });

    expect(
      screen.getByRole("button", { name: "Perfect Pitch · 4310000" }),
    ).toBeInTheDocument();
  });

  it("falls back to the raw id while loading", () => {
    useItemNameMock.mockReturnValue({ data: undefined, isError: false });
    renderPicker({ value: 4310000 });

    expect(
      screen.getByRole("button", { name: "Item 4310000" }),
    ).toBeInTheDocument();
    expect(screen.queryByText(/couldn't resolve/i)).not.toBeInTheDocument();
  });

  it("falls back to the raw id and hints when the lookup fails", () => {
    useItemNameMock.mockReturnValue({ data: undefined, isError: true });
    renderPicker({ value: 4310000 });

    expect(
      screen.getByRole("button", { name: "Item 4310000" }),
    ).toBeInTheDocument();
    expect(screen.getByText(/couldn't resolve/i)).toBeInTheDocument();
  });

  it("picking a row calls onChange with its id and closes the popover", async () => {
    const user = userEvent.setup();
    searchItemsMock.mockResolvedValue(
      page([{ id: "2022503", name: "Red Potion", subcategory: "potion" }]),
    );
    const { onChange } = renderPicker({ value: 0, placeholder: "None" });

    await user.click(screen.getByRole("button", { name: "None" }));
    await user.type(screen.getByPlaceholderText(/search by name/i), "red");

    await user.click(await screen.findByText("Red Potion"));

    expect(onChange).toHaveBeenCalledWith(2022503);
    await waitFor(() =>
      expect(screen.queryByRole("listbox")).not.toBeInTheDocument(),
    );
  });

  it("commits a raw id through the Use id escape hatch", async () => {
    const user = userEvent.setup();
    searchItemsMock.mockResolvedValue(page([]));
    const { onChange } = renderPicker({ value: 0, placeholder: "None" });

    await user.click(screen.getByRole("button", { name: "None" }));
    await user.type(screen.getByPlaceholderText(/search by name/i), "4310000");

    await user.click(await screen.findByText("Use id 4310000"));

    expect(onChange).toHaveBeenCalledWith(4310000);
  });

  it("renders a None row only when allowClear is set, and it clears to 0", async () => {
    const user = userEvent.setup();
    searchItemsMock.mockResolvedValue(page([]));
    useItemNameMock.mockReturnValue({ data: "Perfect Pitch", isError: false });

    const withClear = renderPicker({ value: 4310000, allowClear: true });
    await user.click(
      screen.getByRole("button", { name: "Perfect Pitch · 4310000" }),
    );
    await user.click(screen.getByText("None"));
    expect(withClear.onChange).toHaveBeenCalledWith(0);
    withClear.unmount();

    const withoutClear = renderPicker({ value: 4310000 });
    await user.click(
      screen.getByRole("button", { name: "Perfect Pitch · 4310000" }),
    );
    expect(screen.queryByText("None")).not.toBeInTheDocument();
    withoutClear.unmount();
  });
});
