import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { MtsConfig } from "@/services/api/mts-config.service";

const sampleConfig: MtsConfig = {
  id: "cfg-1",
  attributes: {
    listingFee: 5000,
    commissionRate: 0.07,
    maxActiveListings: 10,
    minLevel: 10,
    auctionMinHours: 24,
    auctionMaxHours: 168,
    priceFloor: 110,
    pageSize: 16,
    minBidIncrement: 100,
  },
};

// Mutable holders so a test can render the "no config" branch and inspect the
// mutation call, without re-declaring the hook mock per test.
const { configHolder, mutateMock } = vi.hoisted(() => ({
  configHolder: { data: undefined as MtsConfig | null | undefined, isLoading: false },
  mutateMock: vi.fn(),
}));

vi.mock("@/lib/hooks/api/useMtsConfig", () => ({
  useMtsConfig: () => configHolder,
  useUpdateMtsConfig: () => ({ mutate: mutateMock, isPending: false }),
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

import { MtsConfigForm } from "@/pages/tenants-mts-config-form";

function renderForm() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={["/tenants/t-1/mts-config"]}>
        <Routes>
          <Route path="/tenants/:id/mts-config" element={<MtsConfigForm />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("MtsConfigForm", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    configHolder.data = sampleConfig;
    configHolder.isLoading = false;
  });

  it("hydrates the form inputs from the loaded config", async () => {
    renderForm();
    await waitFor(() => {
      // form.reset(config.attributes) populates the number inputs.
      expect(screen.getByDisplayValue("5000")).toBeInTheDocument();
      expect(screen.getByDisplayValue("168")).toBeInTheDocument();
    });
  });

  it("shows the empty state when no config exists for the tenant", () => {
    configHolder.data = null;
    renderForm();
    expect(screen.getByText(/no mts configuration found/i)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /save/i })).toBeNull();
  });

  it("submits the tenant id and the current attributes on Save", async () => {
    const user = userEvent.setup();
    renderForm();
    await screen.findByDisplayValue("5000");

    await user.click(screen.getByRole("button", { name: /save/i }));

    await waitFor(() => expect(mutateMock).toHaveBeenCalledTimes(1));
    const [vars] = mutateMock.mock.calls[0]!;
    expect(vars.tenantId).toBe("t-1");
    expect(vars.config).toBe(sampleConfig);
    expect(vars.updates.listingFee).toBe(5000);
    expect(vars.updates.commissionRate).toBe(0.07);
  });
});
