import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { IncubatorRewardsForm } from "../tenants-incubator-rewards-form";
import * as hooks from "@/lib/hooks/api/useIncubatorRewards";

vi.mock("@/lib/hooks/api/useIncubatorRewards");

// Stub ItemNameCell so the test doesn't need item-data; render the raw itemId.
vi.mock("@/components/item-name-cell", () => ({
  ItemNameCell: ({ itemId }: { itemId: string }) => <span>item:{itemId}</span>,
}));

// The page reads the active tenant from context to pass into ItemNameCell;
// a single stubbed tenant is enough since ItemNameCell itself is mocked.
vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: { id: "t1" } }),
}));

function renderForm() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={["/tenants/t1/incubator-rewards"]}>
        <Routes>
          <Route path="/tenants/:id/incubator-rewards" element={<IncubatorRewardsForm />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const noopMut = { mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false } as any;

describe("IncubatorRewardsForm", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (hooks.useIncubatorRewards as any).mockReturnValue({
      data: [
        { id: "r1", attributes: { itemId: 2000000, quantity: 1, weight: 30 } },
        { id: "r2", attributes: { itemId: 2000001, quantity: 2, weight: 10 } },
      ],
      isLoading: false,
    });
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (hooks.useCreateIncubatorReward as any).mockReturnValue(noopMut);
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (hooks.useUpdateIncubatorReward as any).mockReturnValue(noopMut);
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (hooks.useDeleteIncubatorReward as any).mockReturnValue(noopMut);
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (hooks.useSeedIncubatorRewards as any).mockReturnValue(noopMut);
  });

  it("renders rows with computed chance %", () => {
    renderForm();
    expect(screen.getByText("item:2000000")).toBeInTheDocument();
    expect(screen.getByText("item:2000001")).toBeInTheDocument();
    expect(screen.getByText("75.0%")).toBeInTheDocument(); // 30/40
    expect(screen.getByText("25.0%")).toBeInTheDocument(); // 10/40
  });

  it("opens the add dialog", async () => {
    renderForm();
    await userEvent.click(screen.getByRole("button", { name: /add/i }));
    await waitFor(() => expect(screen.getByLabelText(/item id/i)).toBeInTheDocument());
  });

  it("opens the edit dialog prefilled with the row's values", async () => {
    renderForm();
    const editButtons = screen.getAllByRole("button", { name: /edit/i });
    await userEvent.click(editButtons[0]!);
    await waitFor(() => expect(screen.getByLabelText(/item id/i)).toBeInTheDocument());
    expect(screen.getByLabelText(/item id/i)).toHaveValue(2000000);
    expect(screen.getByLabelText(/quantity/i)).toHaveValue(1);
    expect(screen.getByLabelText(/weight/i)).toHaveValue(30);
  });

  it("submits the add dialog by calling create", async () => {
    renderForm();
    await userEvent.click(screen.getByRole("button", { name: /add/i }));
    await waitFor(() => expect(screen.getByLabelText(/item id/i)).toBeInTheDocument());

    await userEvent.clear(screen.getByLabelText(/item id/i));
    await userEvent.type(screen.getByLabelText(/item id/i), "2000002");
    await userEvent.clear(screen.getByLabelText(/quantity/i));
    await userEvent.type(screen.getByLabelText(/quantity/i), "1");
    await userEvent.clear(screen.getByLabelText(/weight/i));
    await userEvent.type(screen.getByLabelText(/weight/i), "5");
    await userEvent.click(screen.getByRole("button", { name: /save/i }));

    await waitFor(() =>
      expect(noopMut.mutate).toHaveBeenCalledWith(
        { tenantId: "t1", attributes: { itemId: 2000002, quantity: 1, weight: 5 } },
        expect.anything(),
      ),
    );
  });

  it("opens the seed confirmation and seeds on confirm", async () => {
    renderForm();
    await userEvent.click(screen.getByRole("button", { name: /seed defaults/i }));
    await waitFor(() => expect(screen.getByRole("alertdialog")).toBeInTheDocument());
    await userEvent.click(screen.getByRole("button", { name: /confirm/i }));
    await waitFor(() => expect(noopMut.mutate).toHaveBeenCalledWith({ tenantId: "t1" }, expect.anything()));
  });

  it("opens the delete confirmation and deletes on confirm", async () => {
    renderForm();
    const deleteButtons = screen.getAllByRole("button", { name: /delete/i });
    await userEvent.click(deleteButtons[0]!);
    await waitFor(() => expect(screen.getByRole("alertdialog")).toBeInTheDocument());
    await userEvent.click(screen.getByRole("button", { name: /confirm/i }));
    await waitFor(() =>
      expect(noopMut.mutate).toHaveBeenCalledWith({ tenantId: "t1", id: "r1" }, expect.anything()),
    );
  });

  it("shows an empty state with no rows and a dash for chance", () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (hooks.useIncubatorRewards as any).mockReturnValue({ data: [], isLoading: false });
    renderForm();
    expect(screen.getByText(/no incubator rewards/i)).toBeInTheDocument();
  });
});
