import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { PendingChangesPanel } from "@/components/features/characters/PendingChangesPanel";
import type { PendingChange } from "@/services/api/pending-changes.service";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: { id: "t" } }),
}));

let changes: PendingChange[] = [];
const mutateAsync = vi.fn().mockResolvedValue(undefined);

vi.mock("@/lib/hooks/api/usePendingChanges", () => ({
  usePendingChanges: () => ({ data: changes, isPending: false, error: null }),
  useCancelPendingChange: () => ({ mutateAsync, isPending: false }),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

function pendingNameChange(
  overrides: Partial<PendingChange> = {},
): PendingChange {
  return {
    id: "pc-1",
    characterId: 1,
    type: "NAME_CHANGE",
    status: "PENDING",
    requestedName: "Zulu",
    destinationWorldId: 0,
    sourceWorldId: 0,
    createdAt: "2026-08-14T00:00:00Z",
    expiresAt: "2026-08-21T00:00:00Z",
    ...overrides,
  };
}

function rejectedNameChange(
  overrides: Partial<PendingChange> = {},
): PendingChange {
  return {
    ...pendingNameChange(),
    status: "REJECTED",
    reason: "name_taken",
    resolvedAt: "2026-08-15T00:00:00Z",
    ...overrides,
  };
}

function appliedNameChange(
  overrides: Partial<PendingChange> = {},
): PendingChange {
  return {
    ...pendingNameChange(),
    status: "APPLIED",
    resolvedAt: "2026-08-15T00:00:00Z",
    ...overrides,
  };
}

function renderPanel() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <PendingChangesPanel characterId="1" />
    </QueryClientProvider>,
  );
}

describe("PendingChangesPanel", () => {
  beforeEach(() => {
    changes = [];
    vi.clearAllMocks();
    mutateAsync.mockResolvedValue(undefined);
  });

  it("lists a pending name change with its requested value and expiry", async () => {
    changes = [
      pendingNameChange({
        requestedName: "Zulu",
        expiresAt: "2026-08-21T00:00:00Z",
      }),
    ];
    renderPanel();

    expect(await screen.findByText("Name Change")).toBeInTheDocument();
    expect(screen.getByText("Zulu")).toBeInTheDocument();
    expect(screen.getByText(/PENDING/)).toBeInTheDocument();
  });

  it("shows the rejection reason on a resolved record so an operator can answer 'what happened to my coupon?'", async () => {
    changes = [
      rejectedNameChange({ requestedName: "Zulu", reason: "name_taken" }),
    ];
    renderPanel();

    expect(await screen.findByText(/REJECTED/)).toBeInTheDocument();
    expect(screen.getByText(/name.taken/i)).toBeInTheDocument();
  });

  it("offers Cancel only on a PENDING record", async () => {
    changes = [
      pendingNameChange({ id: "pc-1" }),
      appliedNameChange({ id: "pc-2" }),
    ];
    renderPanel();

    const buttons = await screen.findAllByRole("button", { name: /cancel/i });
    expect(buttons).toHaveLength(1);
  });

  it("names the character and the requested value in the confirm dialog before cancelling", async () => {
    changes = [pendingNameChange({ id: "pc-1", requestedName: "Zulu" })];
    renderPanel();

    await userEvent.click(
      await screen.findByRole("button", { name: /cancel/i }),
    );

    const dialog = await screen.findByRole("alertdialog");
    expect(dialog).toHaveTextContent("Zulu");
    expect(mutateAsync).not.toHaveBeenCalled(); // not until confirmed

    await userEvent.click(
      screen.getByRole("button", { name: /^cancel request$/i }),
    );
    expect(mutateAsync).toHaveBeenCalledWith({
      tenant: { id: "t" },
      characterId: "1",
      id: "pc-1",
    });
  });

  it("exposes no create or edit affordance (FR-2.10 is read + cancel only)", async () => {
    changes = [pendingNameChange({})];
    renderPanel();
    await screen.findByText("Name Change");

    expect(
      screen.queryByRole("button", { name: /new|create|add|edit/i }),
    ).toBeNull();
  });

  it("renders an empty state rather than a broken table when there are no records", async () => {
    changes = [];
    renderPanel();
    expect(await screen.findByText(/no pending changes/i)).toBeInTheDocument();
  });
});
