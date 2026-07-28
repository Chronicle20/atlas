import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { PoolItemDialog } from "../PoolItemDialog";
import { rewardPoolsService } from "@/services/api/reward-pools.service";

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
  beforeEach(() => vi.clearAllMocks());

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
    await user.type(screen.getByLabelText(/item id/i), "2000000");
    await user.type(screen.getByLabelText(/quantity/i), "1");
    await user.type(screen.getByLabelText(/weight/i), "50");
    await user.click(screen.getByRole("button", { name: /save|add/i }));
    await waitFor(() =>
      expect(rewardPoolsService.createItem).toHaveBeenCalledWith("4170001", {
        itemId: 2000000,
        quantity: 1,
        tier: "common",
        weight: 50,
      }),
    );
  });

  it("rejects weight 0 before calling the service", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.type(screen.getByLabelText(/item id/i), "2000000");
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
});
