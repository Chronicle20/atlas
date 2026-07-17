import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { PoolFormDialog } from "../PoolFormDialog";
import { rewardPoolsService } from "@/services/api/reward-pools.service";

vi.mock("@/services/api/reward-pools.service", () => ({
  rewardPoolsService: {
    createPool: vi.fn().mockResolvedValue(undefined),
    updatePool: vi.fn().mockResolvedValue(undefined),
  },
}));
vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } }),
}));

function renderDialog(props: Partial<Parameters<typeof PoolFormDialog>[0]> = {}) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <PoolFormDialog open onOpenChange={() => {}} mode="create" {...props} />
    </QueryClientProvider>,
  );
}

describe("PoolFormDialog", () => {
  beforeEach(() => vi.clearAllMocks());

  it("create mode: choosing Incubator swaps tier-weight fields for egg fields", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.click(screen.getByRole("radio", { name: /incubator/i }));
    expect(screen.getByLabelText(/egg item id/i)).toBeInTheDocument();
    expect(screen.queryByLabelText(/common weight/i)).not.toBeInTheDocument();
  });

  it("creates an incubator pool with the egg id as the pool id and zero tier weights", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.click(screen.getByRole("radio", { name: /incubator/i }));
    await user.type(screen.getByLabelText(/egg item id/i), "4170001");
    await user.type(screen.getByLabelText(/name/i), "Pigmy Egg (Victoria)");
    await user.type(screen.getByLabelText(/success npc/i), "1012004");
    await user.click(screen.getByRole("button", { name: /create/i }));
    await waitFor(() =>
      expect(rewardPoolsService.createPool).toHaveBeenCalledWith("4170001", {
        name: "Pigmy Egg (Victoria)",
        kind: "incubator",
        npcIds: [1012004],
        commonWeight: 0, uncommonWeight: 0, rareWeight: 0,
      }),
    );
  });

  it("edit mode locks kind and prefills", () => {
    renderDialog({
      mode: "edit",
      pool: { id: "henesys", type: "gachapons", attributes: { name: "Henesys", kind: "gachapon", npcIds: [9100100], commonWeight: 70, uncommonWeight: 25, rareWeight: 5 } },
    });
    expect(screen.queryByRole("radio")).not.toBeInTheDocument();
    expect(screen.getByLabelText(/name/i)).toHaveValue("Henesys");
  });
});
