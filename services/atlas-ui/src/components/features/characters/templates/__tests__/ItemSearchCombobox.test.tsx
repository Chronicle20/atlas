import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
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

  it("shows a distinct error message when the search fails, manual id fallback still works", async () => {
    // A default resolved fallback keeps any extra/lingering queryFn
    // invocation (e.g. TanStack Query's gc/cleanup teardown call, see the
    // debounce/page atomicity test below) from rejecting into an unhandled
    // promise rejection — only the one call the assertions care about
    // (mockRejectedValueOnce) needs to fail.
    searchItemsMock.mockResolvedValue(page([]));
    searchItemsMock.mockRejectedValueOnce(new Error("boom"));
    const onAdd = vi.fn();
    renderBox({ onAdd });
    await userEvent.click(screen.getByRole("button", { name: /add/i }));
    // fireEvent.change sets the whole value in one state update — unlike
    // userEvent.type's per-keystroke updates, this guarantees exactly one
    // debounce-settled query call (for "1402001"), so the mockRejectedValueOnce
    // above is guaranteed to cover that call rather than an intermediate
    // substring typed along the way.
    fireEvent.change(screen.getByRole("textbox"), {
      target: { value: "1402001" },
    });
    expect(
      await screen.findByText(/search failed — enter an id manually/i),
    ).toBeInTheDocument();
    expect(screen.queryByText("No matches.")).not.toBeInTheDocument();
    await userEvent.click(
      screen.getByRole("option", { name: /use id 1402001/i }),
    );
    expect(onAdd).toHaveBeenCalledWith(1402001);
  });

  describe("debounce/page atomicity", () => {
    afterEach(() => {
      vi.useRealTimers();
    });

    it("keeps the page reset atomic with the settled search term — no un-debounced query fires when refining after Load More", async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true });
      const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });

      // `filters` is typed optional and read with `?.` — TanStack Query's
      // gc/cleanup machinery can invoke a lingering queryFn one extra time
      // during unmount teardown after fake timers are torn down; that call
      // carries no args and must resolve harmlessly rather than throw
      // (which would surface as a false test failure unrelated to this
      // test's actual assertions, all of which run before teardown).
      searchItemsMock.mockImplementation(
        (filters?: { pageNumber: number; q?: string }) =>
          Promise.resolve({
            items: [
              {
                id: "1302000",
                name: "Sword",
                compartment: "equipment",
                subcategory: "one-handed-sword",
                type: "Equipment",
              },
            ],
            total: 2,
            pageNumber: filters?.pageNumber ?? 1,
            pageSize: 50,
            lastPage: 2,
          }),
      );

      // gcTime: 0 evicts the page-1 cache entry as soon as the observer
      // moves on to page 2, so a stray key-switch back to page 1 for the
      // (stale) settled term forces a genuine refetch — making the
      // un-debounced-query regression observable via the mock instead of
      // silently absorbed by TanStack Query's default cache reuse.
      const client = new QueryClient({
        defaultOptions: { queries: { retry: false, gcTime: 0 } },
      });
      render(
        <QueryClientProvider client={client}>
          <ItemSearchCombobox
            poolKey="weapons"
            existingIds={[]}
            onAdd={vi.fn()}
            debounceMs={300}
          />
        </QueryClientProvider>,
      );

      await user.click(screen.getByRole("button", { name: /add/i }));
      await user.type(screen.getByRole("textbox"), "sw");

      await act(async () => {
        await vi.advanceTimersByTimeAsync(300);
      });

      await waitFor(() =>
        expect(searchItemsMock).toHaveBeenCalledWith(
          expect.objectContaining({ q: "sw", pageNumber: 1 }),
        ),
      );

      const loadMore = await screen.findByRole("button", {
        name: /load more/i,
      });
      await user.click(loadMore);

      await waitFor(() =>
        expect(searchItemsMock).toHaveBeenCalledWith(
          expect.objectContaining({ q: "sw", pageNumber: 2 }),
        ),
      );

      // Let the now-inactive page-1 cache entry actually get garbage
      // collected before the refine keystroke below.
      await act(async () => {
        await vi.advanceTimersByTimeAsync(0);
      });

      searchItemsMock.mockClear();

      // Refine the term while parked on page 2. This must NOT fire an
      // un-debounced query at the STALE term with pageNumber:1 before the
      // debounce window elapses — that extra query is the regression.
      await user.type(screen.getByRole("textbox"), "o");
      expect(searchItemsMock).not.toHaveBeenCalled();

      await act(async () => {
        await vi.advanceTimersByTimeAsync(300);
      });

      await waitFor(() =>
        expect(searchItemsMock).toHaveBeenCalledWith(
          expect.objectContaining({ q: "swo", pageNumber: 1 }),
        ),
      );
      expect(searchItemsMock).toHaveBeenCalledTimes(1);
    });
  });
});
