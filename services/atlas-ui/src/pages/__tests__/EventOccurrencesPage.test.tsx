import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, within, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { eventsService } from "@/services/api/events.service";
import { EventOccurrencesPage } from "../EventOccurrencesPage";

// vi.mock factories are hoisted above top-level const declarations, so the
// occurrence fixtures referenced inside the factory below must be wrapped in
// vi.hoisted (see RewardPoolsPage.test.tsx for the same idiom).
const { activeBalrogOccurrence, completedBalrogOccurrence } = vi.hoisted(
  () => ({
    activeBalrogOccurrence: {
      id: "o1",
      type: "event-occurrences",
      attributes: {
        type: "CRIMSON_BALROG",
        state: "ACTIVE",
        stage: "ATTACKING",
        context: { worldId: 1, channelId: 4 },
        startedAt: "2024-01-01T00:00:00Z",
      },
      transitions: [],
    },
    completedBalrogOccurrence: {
      id: "o2",
      type: "event-occurrences",
      attributes: {
        type: "CRIMSON_BALROG",
        state: "COMPLETED",
        stage: "DONE",
        context: { worldId: 1, channelId: 4 },
        startedAt: "2024-01-01T00:00:00Z",
        completedAt: "2024-01-01T01:00:00Z",
        completionReason: "VESSEL_ARRIVED",
      },
      transitions: [],
    },
  }),
);

vi.mock("@/services/api/events.service", () => ({
  eventsService: {
    getOccurrences: vi.fn(),
  },
}));

vi.mock("@/lib/utils/toast", () => ({ success: vi.fn(), error: vi.fn() }));

const useTenantMock = vi.hoisted(() => vi.fn());
vi.mock("@/context/tenant-context", () => ({
  useTenant: useTenantMock,
}));

const ACTIVE_TENANT = {
  id: "test-tenant",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
};

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={qc}>
        <EventOccurrencesPage />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("EventOccurrencesPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useTenantMock.mockReturnValue({ activeTenant: ACTIVE_TENANT });
  });

  // FE-09: a direct navigation before TenantProvider resolves the active
  // tenant must not fire a request with no tenant headers.
  it("does not fetch occurrences before the tenant resolves", async () => {
    useTenantMock.mockReturnValue({ activeTenant: null });
    vi.mocked(eventsService.getOccurrences).mockResolvedValue({
      data: [completedBalrogOccurrence],
      meta: null,
    });
    renderPage();
    await Promise.resolve();
    expect(eventsService.getOccurrences).not.toHaveBeenCalled();
  });

  // FR-UI5: id, name/type, state, stage, scope, start, completion, reason.
  it("renders the occurrence summary columns", async () => {
    vi.mocked(eventsService.getOccurrences).mockResolvedValue({
      data: [completedBalrogOccurrence],
      meta: null,
    });
    renderPage();
    expect(await screen.findByText("VESSEL_ARRIVED")).toBeInTheDocument();
    const table = screen.getByRole("table");
    expect(within(table).getByText("COMPLETED")).toBeInTheDocument();
    expect(within(table).getByText(/w1 ch4/i)).toBeInTheDocument();
  });

  // FR-UI5: active occurrences must be readily distinguishable from history.
  it("distinguishes active from historical occurrences", async () => {
    vi.mocked(eventsService.getOccurrences).mockResolvedValue({
      data: [activeBalrogOccurrence, completedBalrogOccurrence],
      meta: null,
    });
    renderPage();
    const active = await screen.findByTestId(
      `occurrence-${activeBalrogOccurrence.id}`,
    );
    const historical = screen.getByTestId(
      `occurrence-${completedBalrogOccurrence.id}`,
    );
    expect(within(active).getByText("ACTIVE")).toBeInTheDocument();
    expect(within(historical).getByText("COMPLETED")).toBeInTheDocument();
    expect(active.className).not.toEqual(historical.className);
  });

  // Fix round 1, finding 3: the list must link to the detail page — nothing
  // reached /events/occurrences/:id without a hand-typed URL otherwise.
  it("links each occurrence's id to its detail page", async () => {
    vi.mocked(eventsService.getOccurrences).mockResolvedValue({
      data: [completedBalrogOccurrence],
      meta: null,
    });
    renderPage();
    const link = await screen.findByRole("link", {
      name: completedBalrogOccurrence.id.slice(0, 8),
    });
    expect(link).toHaveAttribute(
      "href",
      `/events/occurrences/${completedBalrogOccurrence.id}`,
    );
  });

  // FR-UI6: filterable by type, state, active-vs-historical, date range, world
  // and channel. There is no backend "active vs historical" filter — the
  // distinction is expressed through filter[state].
  it("passes the selected filters to the service", async () => {
    const get = vi.mocked(eventsService.getOccurrences).mockResolvedValue({
      data: [],
      meta: null,
    });
    renderPage();
    await userEvent.selectOptions(
      await screen.findByLabelText(/state/i),
      "ACTIVE",
    );
    await waitFor(() =>
      expect(get).toHaveBeenLastCalledWith(
        expect.objectContaining({ state: "ACTIVE" }),
      ),
    );
  });

  it("offers a refresh control when no occurrences match", async () => {
    const get = vi.mocked(eventsService.getOccurrences).mockResolvedValue({
      data: [],
      meta: null,
    });
    const user = userEvent.setup();
    renderPage();
    await screen.findByText("No event occurrences found");
    const refreshButton = screen.getByTestId("empty-state-refresh");
    await user.click(refreshButton);
    await waitFor(() => expect(get).toHaveBeenCalledTimes(2));
  });
});
