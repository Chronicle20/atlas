import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { FieldDetailPage } from "@/pages/FieldDetailPage";
import type { Tenant } from "@/types/models/tenant";
import type { MapData } from "@/services/api/maps.service";
import type { WorldData } from "@/services/api/worlds.service";
import type { FieldData } from "@/services/api/fields.service";
import type { LiveMonsterData } from "@/services/api/live-monsters.service";
import type { MapObjectData } from "@/services/api/map-entities.service";
import * as toast from "@/lib/utils/toast";

const mockTenant: Tenant = {
  id: "tenant-1",
  attributes: {
    name: "Test Tenant",
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
  },
};

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: mockTenant,
    tenants: [mockTenant],
    loading: false,
    setActiveTenant: vi.fn(),
    refreshTenants: vi.fn(),
    refreshAndSelectTenant: vi.fn(),
    fetchTenantConfiguration: vi.fn(),
  }),
}));

const useMapMock = vi.fn();
vi.mock("@/lib/hooks/api/useMaps", () => ({
  useMap: (...args: unknown[]) => useMapMock(...args),
}));

const useWorldsMock = vi.fn();
vi.mock("@/lib/hooks/api/useWorlds", () => ({
  useWorlds: () => useWorldsMock(),
}));

const useFieldsMock = vi.fn();
vi.mock("@/lib/hooks/api/useFields", () => ({
  useFields: (...args: unknown[]) => useFieldsMock(...args),
}));

const useLiveMonstersMock = vi.fn();
vi.mock("@/lib/hooks/api/useFieldRuntime", () => ({
  useLiveMonsters: (...args: unknown[]) => useLiveMonstersMock(...args),
}));

const useMapObjectsMock = vi.fn();
vi.mock("@/lib/hooks/api/useMapEntities", () => ({
  useMapObjects: (...args: unknown[]) => useMapObjectsMock(...args),
}));

vi.mock("@/lib/utils/toast", () => ({
  success: vi.fn(),
  error: vi.fn(),
}));

function queryResult(data: unknown, overrides: Record<string, unknown> = {}) {
  return {
    data,
    isLoading: false,
    isError: false,
    error: null,
    isFetching: false,
    dataUpdatedAt: data === undefined ? 0 : 1_700_000_000_000,
    refetch: vi.fn().mockResolvedValue({ isError: false, error: null }),
    ...overrides,
  };
}

function makeMap(): MapData {
  return {
    id: "910340000",
    attributes: { name: "Henesys", streetName: "Victoria Road", mapArea: null },
  };
}

function makeWorld(id: string, name: string): WorldData {
  return {
    id,
    type: "worlds",
    attributes: {
      name,
      state: 0,
      message: "",
      eventMessage: "",
      recommended: false,
      recommendedMessage: "",
      capacityStatus: 0,
      expRate: 0,
      mesoRate: 0,
      itemDropRate: 0,
      questExpRate: 0,
    },
  };
}

const INSTANCE_ID = "00000000-0000-0000-0000-000000000000";

function makeField(characterCount: number): FieldData {
  return {
    id: `0:1:910340000:${INSTANCE_ID}`,
    type: "fields",
    attributes: {
      worldId: 0,
      channelId: 1,
      mapId: 910340000,
      instanceId: INSTANCE_ID,
      characterCount,
    },
  };
}

function makeLiveMonster(id: string, monsterId: number): LiveMonsterData {
  return {
    id,
    type: "live-monsters",
    attributes: {
      worldId: 0,
      channelId: 1,
      mapId: 910340000,
      instance: INSTANCE_ID,
      monsterId,
      controlCharacterId: 0,
      x: 0,
      y: 0,
      fh: 0,
      stance: 0,
      team: -1,
      maxHp: 100,
      hp: 100,
      maxMp: 0,
      mp: 0,
      damageEntries: [],
      experienceEntries: [],
      statusEffects: [],
      controllerHasAggro: false,
    },
  };
}

function makeMapObject(id: string): MapObjectData {
  return { id, type: "map-objects" } as MapObjectData;
}

function LocationSpy() {
  const location = useLocation();
  return <div data-testid="location-search">{location.search}</div>;
}

function renderPage(initialPath = `/fields/0/1/910340000/${INSTANCE_ID}`) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[initialPath]}>{children}</MemoryRouter>
    </QueryClientProvider>
  );
  return render(
    <>
      <Routes>
        <Route
          path="/fields/:worldId/:channelId/:mapId/:instanceId"
          element={<FieldDetailPage />}
        />
      </Routes>
      <LocationSpy />
    </>,
    { wrapper },
  );
}

describe("FieldDetailPage", () => {
  beforeEach(() => {
    useMapMock.mockReset();
    useWorldsMock.mockReset();
    useFieldsMock.mockReset();
    useLiveMonstersMock.mockReset();
    useMapObjectsMock.mockReset();
    vi.mocked(toast.error).mockReset();
    vi.mocked(toast.success).mockReset();

    useMapMock.mockReturnValue(queryResult(makeMap()));
    useWorldsMock.mockReturnValue(
      queryResult([makeWorld("1", "Bera"), makeWorld("2", "Kradia")]),
    );
    useFieldsMock.mockReturnValue(queryResult([makeField(2)]));
    useLiveMonstersMock.mockReturnValue(
      queryResult([
        makeLiveMonster("m1", 100100),
        makeLiveMonster("m2", 100100),
        makeLiveMonster("m3", 100101),
      ]),
    );
    useMapObjectsMock.mockReturnValue(
      queryResult([makeMapObject("o1"), makeMapObject("o2")]),
    );
  });

  it("map name is the primary title", () => {
    renderPage();

    expect(
      screen.getByRole("heading", { name: "Henesys" }),
    ).toBeInTheDocument();
  });

  it("world/channel/instance outrank the map id", () => {
    renderPage();

    expect(screen.getByText(/World 0/)).toBeInTheDocument();
    expect(screen.getByText(/Channel 1/)).toBeInTheDocument();
    expect(screen.getByText(new RegExp(INSTANCE_ID))).toBeInTheDocument();
    expect(
      screen.queryByRole("heading", { name: /910340000/ }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /view map definition/i }),
    ).toHaveAttribute("href", "/maps/910340000");
  });

  it("renders a Runtime badge", () => {
    renderPage();

    expect(screen.getByText("Runtime")).toBeInTheDocument();
  });

  it("View Map Definition link", () => {
    renderPage();

    expect(
      screen.getByRole("link", { name: /view map definition/i }),
    ).toHaveAttribute("href", "/maps/910340000");
  });

  it("live summary shows character count", () => {
    useFieldsMock.mockReturnValue(queryResult([makeField(3)]));
    renderPage();

    expect(screen.getByTestId("field-character-count")).toHaveTextContent("3");
  });

  it("live summary groups monsters by name", () => {
    renderPage();

    const groups = screen.getAllByTestId("field-monster-group");
    expect(groups).toHaveLength(2);
    expect(screen.getByText("×2")).toBeInTheDocument();
    expect(screen.getByText("×1")).toBeInTheDocument();
  });

  it("live summary shows tracked object count", () => {
    renderPage();

    expect(screen.getByTestId("field-tracked-object-count")).toHaveTextContent(
      "—",
    );
  });

  it("tabs render with counts", () => {
    useFieldsMock.mockReturnValue(queryResult([makeField(3)]));
    renderPage();

    expect(screen.getByText("Characters (3)")).toBeInTheDocument();
    expect(screen.getByText("Monsters (3)")).toBeInTheDocument();
    expect(screen.getByText("Map Objects (2)")).toBeInTheDocument();
  });

  it("tab state syncs to the query string", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("tab", { name: /monsters/i }));

    expect(screen.getByTestId("location-search")).toHaveTextContent(
      "tab=monsters",
    );
  });

  it("?tab= selects the tab on load", () => {
    renderPage(`/fields/0/1/910340000/${INSTANCE_ID}?tab=objects`);

    expect(screen.getByRole("tab", { name: /map objects/i })).toHaveAttribute(
      "data-state",
      "active",
    );
  });

  it("torn-down field", () => {
    useFieldsMock.mockReturnValue(queryResult([]));
    renderPage();

    expect(screen.getByText(/may have been torn down/i)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /fields/i })).toHaveAttribute(
      "href",
      "/fields",
    );
    expect(toast.error).not.toHaveBeenCalled();
  });

  it("exposes refresh and last updated", () => {
    renderPage();

    expect(
      screen.getByRole("button", { name: /refresh/i }),
    ).toBeInTheDocument();
    expect(screen.getByText(/last updated/i)).toBeInTheDocument();
  });
});
