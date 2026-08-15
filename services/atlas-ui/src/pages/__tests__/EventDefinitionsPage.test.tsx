import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { eventsService } from "@/services/api/events.service";
import { EventDefinitionsPage } from "../EventDefinitionsPage";

// vi.mock factories are hoisted above top-level const declarations, so the
// definition fixtures referenced inside the factory below must be wrapped in
// vi.hoisted (see RewardPoolsPage.test.tsx for the same idiom).
const { balrogDefinition, anniversaryDefinition } = vi.hoisted(() => ({
  balrogDefinition: {
    id: "d1",
    type: "event-definitions",
    attributes: {
      type: "CRIMSON_BALROG",
      name: "Crimson Balrog",
      enabled: false,
      configuration: {},
      singleOccurrence: false,
      createdAt: "2024-01-01T00:00:00Z",
      updatedAt: "2024-01-01T00:00:00Z",
    },
  },
  anniversaryDefinition: {
    id: "d2",
    type: "event-definitions",
    attributes: {
      type: "ANNIVERSARY",
      name: "Anniversary",
      enabled: false,
      configuration: {},
      singleOccurrence: true,
      createdAt: "2024-01-01T00:00:00Z",
      updatedAt: "2024-01-01T00:00:00Z",
    },
  },
}));

vi.mock("@/services/api/events.service", () => ({
  eventsService: {
    getDefinitions: vi.fn(),
    setDefinitionEnabled: vi.fn(),
    getOccurrences: vi.fn(),
  },
}));

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={qc}>
        <EventDefinitionsPage />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("EventDefinitionsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Default: no active occurrences for any definition, overridden per-test.
    vi.mocked(eventsService.getOccurrences).mockResolvedValue({
      data: [],
      meta: null,
    });
  });

  it("shows name, type and enablement", async () => {
    vi.mocked(eventsService.getDefinitions).mockResolvedValue({
      data: [balrogDefinition, anniversaryDefinition],
      meta: null,
    });
    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Crimson Balrog")).toBeInTheDocument(),
    );
    expect(screen.getByText("CRIMSON_BALROG")).toBeInTheDocument();
  });

  it("toggles enablement", async () => {
    vi.mocked(eventsService.getDefinitions).mockResolvedValue({
      data: [balrogDefinition],
      meta: null,
    });
    const setEnabled = vi
      .mocked(eventsService.setDefinitionEnabled)
      .mockResolvedValue({
        ...balrogDefinition,
        attributes: { ...balrogDefinition.attributes, enabled: true },
      });
    const user = userEvent.setup();
    renderPage();
    await waitFor(() =>
      expect(
        screen.getByRole("switch", { name: /crimson balrog/i }),
      ).toBeInTheDocument(),
    );
    await user.click(screen.getByRole("switch", { name: /crimson balrog/i }));
    expect(setEnabled).toHaveBeenCalledWith(balrogDefinition.id, true);
  });

  // FR-UI4: "enabled" must never read as "occurring". A definition that can
  // have MANY concurrent occurrences (singleOccurrence: false) shows a count
  // linking to the filtered list, never a single live state.
  it("does not render enabled as occurring for a multi-occurrence type", async () => {
    vi.mocked(eventsService.getDefinitions).mockResolvedValue({
      data: [
        {
          ...balrogDefinition,
          attributes: {
            ...balrogDefinition.attributes,
            enabled: true,
            singleOccurrence: false,
          },
        },
      ],
      meta: null,
    });
    vi.mocked(eventsService.getOccurrences).mockResolvedValue({
      data: [],
      meta: null,
    });
    renderPage();
    await waitFor(() =>
      expect(screen.getByText(/0 active/i)).toBeInTheDocument(),
    );
    expect(screen.queryByText(/^occurring$/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/in progress/i)).not.toBeInTheDocument();
  });

  // A single-occurrence type MAY show live state, because at most one exists.
  it("shows live occurrence state for a single-occurrence type", async () => {
    vi.mocked(eventsService.getDefinitions).mockResolvedValue({
      data: [
        {
          ...anniversaryDefinition,
          attributes: {
            ...anniversaryDefinition.attributes,
            enabled: true,
            singleOccurrence: true,
          },
        },
      ],
      meta: null,
    });
    vi.mocked(eventsService.getOccurrences).mockResolvedValue({
      data: [
        {
          id: "o1",
          type: "event-occurrences",
          attributes: {
            type: "ANNIVERSARY",
            state: "ACTIVE",
            stage: "running",
            context: {},
            startedAt: "2024-01-01T00:00:00Z",
          },
          transitions: [],
        },
      ],
      meta: null,
    });
    renderPage();
    await waitFor(() =>
      expect(screen.getByText(/active now/i)).toBeInTheDocument(),
    );
  });
});
