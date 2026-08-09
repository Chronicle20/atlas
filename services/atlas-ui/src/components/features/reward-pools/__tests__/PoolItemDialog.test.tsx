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
        commodityId: 0,
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

  it("requires a commodity id for cash-surprise entries", async () => {
    const user = userEvent.setup();
    renderDialog({ kind: "cash-surprise" });
    await user.type(screen.getByLabelText(/item id/i), "2000000");
    await user.type(screen.getByLabelText(/quantity/i), "1");
    await user.type(screen.getByLabelText(/^weight/i), "50");
    // Left blank, the raw input coerces to NaN and zod reports a generic
    // "expected number" error before the field-level .positive() message
    // ever runs; typing the rejected value 0 is what exercises the schema's
    // custom "Commodity id is required" message (mirrors the "weight 0"
    // pattern the existing weight-required test above uses).
    await user.type(screen.getByLabelText(/commodity id/i), "0");
    await user.click(screen.getByRole("button", { name: /save|add/i }));
    await waitFor(() =>
      expect(screen.getByText(/commodity id is required/i)).toBeInTheDocument(),
    );
    expect(rewardPoolsService.createItem).not.toHaveBeenCalled();
  });

  it("submits commodityId alongside weight for cash-surprise entries", async () => {
    const user = userEvent.setup();
    renderDialog({ kind: "cash-surprise" });
    await user.type(screen.getByLabelText(/item id/i), "2000000");
    await user.type(screen.getByLabelText(/quantity/i), "1");
    await user.type(screen.getByLabelText(/^weight/i), "50");
    await user.type(screen.getByLabelText(/commodity id/i), "100200300");
    await user.click(screen.getByRole("button", { name: /save|add/i }));
    await waitFor(() =>
      expect(rewardPoolsService.createItem).toHaveBeenCalledWith("4170001", {
        itemId: 2000000,
        quantity: 1,
        tier: "common",
        weight: 50,
        commodityId: 100200300,
      }),
    );
  });

  it("does not send a commodityId for incubator entries", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.type(screen.getByLabelText(/item id/i), "2000000");
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
