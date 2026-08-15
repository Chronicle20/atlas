import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { eventsService } from "@/services/api/events.service";
import { EventOccurrenceDetailPage } from "../EventOccurrenceDetailPage";

// vi.mock factories are hoisted above top-level const declarations, so
// fixtures referenced inside the factory below must be wrapped in
// vi.hoisted (same idiom as RewardPoolsPage.test.tsx / EventOccurrencesPage.test.tsx).
const { completedBalrogOccurrence, anniversaryOccurrence } = vi.hoisted(() => ({
  completedBalrogOccurrence: {
    id: "o2",
    type: "event-occurrences",
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
}));

vi.mock("@/services/api/events.service", () => ({
  eventsService: {
    getOccurrence: vi.fn(),
  },
}));

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

  // FR-UI8
  it("shows Crimson Balrog scope and monster status", async () => {
    vi.mocked(eventsService.getOccurrence).mockResolvedValue(
      completedBalrogOccurrence,
    );
    renderAt("o2");
    expect(await screen.findByText(/voyage/i)).toBeInTheDocument();
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
