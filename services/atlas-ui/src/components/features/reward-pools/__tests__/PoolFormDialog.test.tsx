import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { PoolFormDialog } from "../PoolFormDialog";
import { rewardPoolsService } from "@/services/api/reward-pools.service";
import { tenantsService } from "@/services/api/tenants.service";

const searchItemsMock = vi.fn();

vi.mock("@/services/api/reward-pools.service", () => ({
  rewardPoolsService: {
    createPool: vi.fn().mockResolvedValue(undefined),
    updatePool: vi.fn().mockResolvedValue(undefined),
  },
}));
vi.mock("@/services/api/items.service", () => ({
  itemsService: { searchItems: (...a: unknown[]) => searchItemsMock(...a) },
}));
vi.mock("@/services/api/tenants.service", () => ({
  tenantsService: { getTenantConfigurationById: vi.fn() },
}));
vi.mock("@/lib/hooks/api/useItemStrings", () => ({
  useItemName: () => ({ data: undefined, isError: false }),
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
  props: Partial<Parameters<typeof PoolFormDialog>[0]> = {},
) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <PoolFormDialog open onOpenChange={() => {}} mode="create" {...props} />
    </QueryClientProvider>,
  );
}

/** A tenant configuration whose Surprise box list is exactly [5222000]. */
function configWithBoxes(boxTemplateIds: number[]) {
  return {
    id: "t1",
    attributes: {
      region: "GMS",
      majorVersion: 83,
      minorVersion: 1,
      cashShop: { commodities: {}, surprise: { boxTemplateIds } },
    },
  };
}

/** Drives the box ItemPicker via its "Use id N" escape hatch. */
async function pickBox(user: ReturnType<typeof userEvent.setup>, id: number) {
  await user.click(screen.getByRole("button", { name: /select a box item/i }));
  await user.type(screen.getByPlaceholderText(/search by name/i), String(id));
  await user.click(await screen.findByText(`Use id ${id}`));
}

describe("PoolFormDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    searchItemsMock.mockResolvedValue({
      items: [],
      total: 0,
      pageNumber: 1,
      pageSize: 50,
      lastPage: 1,
    });
    vi.mocked(tenantsService.getTenantConfigurationById).mockResolvedValue(
      configWithBoxes([5222000]) as Awaited<
        ReturnType<typeof tenantsService.getTenantConfigurationById>
      >,
    );
  });

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
        commonWeight: 0,
        uncommonWeight: 0,
        rareWeight: 0,
      }),
    );
  });

  it("offers a cash-surprise kind and submits the box item id as the pool id", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.click(screen.getByRole("radio", { name: /cash surprise/i }));
    await pickBox(user, 5222000);
    await user.type(screen.getByLabelText(/name/i), "Surprise Box");
    await user.click(screen.getByRole("button", { name: /create/i }));
    await waitFor(() =>
      expect(rewardPoolsService.createPool).toHaveBeenCalledWith("5222000", {
        name: "Surprise Box",
        kind: "cash-surprise",
        npcIds: [],
        commonWeight: 0,
        uncommonWeight: 0,
        rareWeight: 0,
      }),
    );
  });

  it("picks the box item rather than asking for a raw id", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.click(screen.getByRole("radio", { name: /cash surprise/i }));
    expect(
      screen.getByRole("group", { name: /box item/i }),
    ).toBeInTheDocument();
    expect(screen.queryByLabelText(/box item id/i)).not.toBeInTheDocument();
  });

  // A pool keyed to an unconfigured box is never rolled — atlas-cashshop
  // rejects the open with NOT_A_SURPRISE_BOX before reaching the pool.
  it("warns when the picked box is not one of the tenant's Surprise boxes", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.click(screen.getByRole("radio", { name: /cash surprise/i }));
    await pickBox(user, 5220000);
    expect(
      await screen.findByText(/not one of this tenant's surprise boxes/i),
    ).toBeInTheDocument();
  });

  it("does not warn for a configured box", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.click(screen.getByRole("radio", { name: /cash surprise/i }));
    await pickBox(user, 5222000);
    await waitFor(() =>
      expect(
        vi.mocked(tenantsService.getTenantConfigurationById),
      ).toHaveBeenCalled(),
    );
    expect(
      screen.queryByText(/not one of this tenant's surprise boxes/i),
    ).not.toBeInTheDocument();
  });

  it("does not render an NPC Ids field for cash-surprise pools", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.click(screen.getByRole("radio", { name: /cash surprise/i }));
    expect(screen.queryByLabelText(/NPC Ids/i)).not.toBeInTheDocument();
  });

  it("edit mode locks kind and prefills", () => {
    renderDialog({
      mode: "edit",
      pool: {
        id: "henesys",
        type: "gachapons",
        attributes: {
          name: "Henesys",
          kind: "gachapon",
          npcIds: [9100100],
          commonWeight: 70,
          uncommonWeight: 25,
          rareWeight: 5,
        },
      },
    });
    expect(screen.queryByRole("radio")).not.toBeInTheDocument();
    expect(screen.getByLabelText(/name/i)).toHaveValue("Henesys");
  });
});
