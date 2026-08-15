import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { eventsService } from "@/services/api/events.service";
import { EventOccurrenceDetailPage } from "../EventOccurrenceDetailPage";

// vi.mock factories are hoisted above top-level const declarations, so
// fixtures referenced inside the factory below must be wrapped in
// vi.hoisted (same idiom as RewardPoolsPage.test.tsx / EventOccurrencesPage.test.tsx).
const {
  completedBalrogOccurrence,
  anniversaryOccurrence,
  crimsonBalrogDefinition,
} = vi.hoisted(() => ({
  completedBalrogOccurrence: {
    id: "o2",
    type: "event-occurrences",
    definitionId: "def-crimson-balrog",
    attributes: {
      type: "CRIMSON_BALROG",
      state: "COMPLETED",
      stage: "DONE",
      context: {
        routeId: "route-1",
        voyageId: "voyage-42",
        worldId: 1,
        channelId: 4,
        attackMaps: [
          { mapId: 200090001, spawnPositions: [{ x: 100, y: 200 }] },
        ],
        relatedMapIds: [200090000],
        monsterId: 200090010,
        monsterCount: 1,
        backgroundMusic: "Bgm00/FantasyReturn",
        visual: {
          name: "balrog",
          showState: "0",
          showSubState: "0",
          hideState: "1",
          hideSubState: "0",
        },
      },
      startedAt: "2024-01-01T00:00:00Z",
      completedAt: "2024-01-01T01:00:00Z",
    },
    transitions: [
      {
        id: "t1",
        occurrenceId: "o2",
        fromStage: "NONE",
        toStage: "SCHEDULED",
        occurredAt: "2024-01-01T00:00:00Z",
        triggerType: "OCCURRENCE_CREATED",
      },
      {
        id: "t2",
        occurrenceId: "o2",
        fromStage: "SCHEDULED",
        toStage: "ATTACKING",
        occurredAt: "2024-01-01T00:05:00Z",
        triggerType: "VESSEL_ARRIVED",
      },
    ],
  },
  anniversaryOccurrence: {
    id: "o3",
    type: "event-occurrences",
    attributes: {
      type: "ANNIVERSARY",
      state: "ACTIVE",
      stage: "RUNNING",
      context: {
        scheduledEnd: "2024-01-08T00:00:00Z",
        expMultiplier: 2.0,
        dropMultiplier: 1.5,
        buffSourceId: 12345,
      },
      startedAt: "2024-01-01T00:00:00Z",
    },
    transitions: [],
  },
  crimsonBalrogDefinition: {
    id: "def-crimson-balrog",
    type: "event-definitions",
    attributes: {
      type: "CRIMSON_BALROG",
      name: "Crimson Balrog Assault",
      enabled: true,
      configuration: {},
      singleOccurrence: false,
      createdAt: "2024-01-01T00:00:00Z",
      updatedAt: "2024-01-01T00:00:00Z",
    },
  },
}));

vi.mock("@/services/api/events.service", () => ({
  eventsService: {
    getOccurrence: vi.fn(),
    getDefinition: vi.fn(),
  },
}));

const useTenantMock = vi.hoisted(() => vi.fn());
vi.mock("@/context/tenant-context", () => ({
  useTenant: useTenantMock,
}));

const ACTIVE_TENANT = {
  id: "test-tenant",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
};

function renderAt(id: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter initialEntries={[`/events/occurrences/${id}`]}>
      <QueryClientProvider client={qc}>
        <Routes>
          <Route
            path="/events/occurrences/:id"
            element={<EventOccurrenceDetailPage />}
          />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("EventOccurrenceDetailPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useTenantMock.mockReturnValue({ activeTenant: ACTIVE_TENANT });
    vi.mocked(eventsService.getDefinition).mockResolvedValue(
      crimsonBalrogDefinition,
    );
  });

  // FE-09: a direct navigation before TenantProvider resolves the active
  // tenant must not fire a request with no tenant headers, for either the
  // occurrence query or the lazy definition-name query it feeds.
  it("does not fetch the occurrence or definition before the tenant resolves", async () => {
    useTenantMock.mockReturnValue({ activeTenant: null });
    vi.mocked(eventsService.getOccurrence).mockResolvedValue(
      completedBalrogOccurrence,
    );
    renderAt("o2");
    await Promise.resolve();
    expect(eventsService.getOccurrence).not.toHaveBeenCalled();
    expect(eventsService.getDefinition).not.toHaveBeenCalled();
  });

  // FR-UI7
  it("shows the transition history", async () => {
    vi.mocked(eventsService.getOccurrence).mockResolvedValue(
      completedBalrogOccurrence,
    );
    renderAt("o2");
    expect(await screen.findByText("OCCURRENCE_CREATED")).toBeInTheDocument();
    expect(screen.getByText("ATTACKING")).toBeInTheDocument();
    expect(screen.getByText("VESSEL_ARRIVED")).toBeInTheDocument();
  });

  // Fetches the routed occurrence, not a stale/default one.
  it("fetches the occurrence for the routed id", async () => {
    vi.mocked(eventsService.getOccurrence).mockResolvedValue(
      completedBalrogOccurrence,
    );
    renderAt("o2");
    await screen.findByText("OCCURRENCE_CREATED");
    expect(eventsService.getOccurrence).toHaveBeenCalledWith("o2");
  });

  // FR-UI7 (fix round 1, finding 1): the occurrence's owning definition —
  // surfaced via the `definition` relationship id (events.service.ts) and
  // resolved to a name via getDefinition — is rendered on the page.
  it("shows the associated definition", async () => {
    vi.mocked(eventsService.getOccurrence).mockResolvedValue(
      completedBalrogOccurrence,
    );
    renderAt("o2");
    expect(
      await screen.findByText("Crimson Balrog Assault"),
    ).toBeInTheDocument();
    expect(eventsService.getDefinition).toHaveBeenCalledWith(
      "def-crimson-balrog",
    );
  });

  // FR-UI8
  it("shows Crimson Balrog scope and monster status", async () => {
    vi.mocked(eventsService.getOccurrence).mockResolvedValue(
      completedBalrogOccurrence,
    );
    renderAt("o2");
    // Exact node match (not a substring regex) so this pins panel selection
    // specifically, not just "some element mentions voyage somewhere" — the
    // always-present full-context JSON view (finding 2) also contains
    // "voyageId", so a loose /voyage/i match would be satisfied by either.
    expect(await screen.findByText("Voyage: voyage-42")).toBeInTheDocument();
    expect(screen.getByText("200090010")).toBeInTheDocument();
  });

  it("shows Anniversary schedule and multipliers", async () => {
    vi.mocked(eventsService.getOccurrence).mockResolvedValue(
      anniversaryOccurrence,
    );
    renderAt("o3");
    expect(await screen.findByText("2.0x")).toBeInTheDocument();
    expect(screen.getByText(/scheduled end/i)).toBeInTheDocument();
  });

  // FR-UI7 + FR-UI8 (fix round 1, finding 2): the bespoke panel supplements
  // the full context, it does not substitute for it — both must be
  // reachable for a registered type.
  it("renders both the bespoke panel and the full context for a registered type", async () => {
    vi.mocked(eventsService.getOccurrence).mockResolvedValue(
      completedBalrogOccurrence,
    );
    renderAt("o2");
    expect(await screen.findByText("Voyage: voyage-42")).toBeInTheDocument();
    expect(
      await screen.findByTestId("occurrence-context-json"),
    ).toBeInTheDocument();
  });

  // An occurrence of a type with no bespoke panel still renders: the generic
  // context view is the fallback, so a third event needs no edit here to be
  // usable. Overrides `attributes.type` (the domain event type the panel
  // lookup keys on), not the top-level `type` (the fixed JSON:API resource
  // type, always "event-occurrences" — see event-occurrence-panels.tsx).
  it("falls back to the generic context view for an unknown type", async () => {
    vi.mocked(eventsService.getOccurrence).mockResolvedValue({
      ...completedBalrogOccurrence,
      attributes: {
        ...completedBalrogOccurrence.attributes,
        type: "MYSTERIOUS_MERCHANT",
      },
    });
    renderAt("o2");
    expect(
      await screen.findByTestId("occurrence-context-json"),
    ).toBeInTheDocument();
  });
});
