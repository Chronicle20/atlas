import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";

const useMapMock = vi.fn();
const useMapsByNameMock = vi.fn();
vi.mock("@/lib/hooks/api/useMaps", () => ({
  useMap: (...a: unknown[]) => useMapMock(...a),
  useMapsByName: (...a: unknown[]) => useMapsByNameMock(...a),
}));

import { MapPicker } from "../MapPicker";

const mushroomTown = {
  id: "10000",
  attributes: { name: "Mushroom Town", streetName: "Maple Road" },
};

beforeEach(() => {
  useMapMock.mockReset();
  useMapsByNameMock.mockReset();
  useMapsByNameMock.mockReturnValue({ data: [], isLoading: false });
});

describe("MapPicker", () => {
  it("shows <name> · <streetName> · <id> when the id resolves", () => {
    useMapMock.mockReturnValue({ data: mushroomTown, isError: false });
    render(<MapPicker value={10000} onChange={vi.fn()} />);
    expect(
      screen.getByRole("button", {
        name: /Mushroom Town · Maple Road · 10000/,
      }),
    ).toBeInTheDocument();
  });

  it("shows Map <id> with a warning hint when unresolvable (non-blocking)", () => {
    useMapMock.mockReturnValue({ data: undefined, isError: true });
    render(<MapPicker value={999999999} onChange={vi.fn()} />);
    expect(screen.getByText(/Map 999999999/)).toBeInTheDocument();
    expect(screen.getByText(/not found in map data/i)).toBeInTheDocument();
  });

  it("search results select a map by id", async () => {
    useMapMock.mockReturnValue({ data: undefined, isError: false });
    useMapsByNameMock.mockReturnValue({
      data: [mushroomTown],
      isLoading: false,
    });
    const onChange = vi.fn();
    render(<MapPicker value={0} onChange={onChange} debounceMs={0} />);
    await userEvent.click(screen.getByRole("button", { name: /Map 0/ }));
    await userEvent.type(screen.getByRole("textbox"), "mush");
    await userEvent.click(
      await screen.findByRole("option", { name: /Mushroom Town/ }),
    );
    expect(onChange).toHaveBeenCalledWith(10000);
  });

  it("numeric input offers the manual Use id fallback", async () => {
    useMapMock.mockReturnValue({ data: undefined, isError: false });
    const onChange = vi.fn();
    render(<MapPicker value={0} onChange={onChange} debounceMs={0} />);
    await userEvent.click(screen.getByRole("button", { name: /Map 0/ }));
    await userEvent.type(screen.getByRole("textbox"), "100000000");
    await userEvent.click(
      await screen.findByRole("option", { name: /use id 100000000/i }),
    );
    expect(onChange).toHaveBeenCalledWith(100000000);
  });
});
