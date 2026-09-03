import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import {
  MemoryRouter,
  Route,
  Routes,
  useLocation,
  useNavigate,
} from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { FieldDetailPage } from "@/pages/FieldDetailPage";
import type { Tenant } from "@/types/models/tenant";
import type { MapData } from "@/services/api/maps.service";
import type { WorldData } from "@/services/api/worlds.service";
import type { FieldCharacterData } from "@/services/api/fields.service";
import type { LiveMonsterData } from "@/services/api/live-monsters.service";
import type {
  MapNpcData,
  MapObjectData,
  MapPortalData,
  MapReactorData,
} from "@/services/api/map-entities.service";
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

const useFieldCharactersMock = vi.fn();
const useLiveMonstersMock = vi.fn();
vi.mock("@/lib/hooks/api/useFieldRuntime", () => ({
  useFieldCharacters: (...args: unknown[]) => useFieldCharactersMock(...args),
  useLiveMonsters: (...args: unknown[]) => useLiveMonstersMock(...args),
}));

const useMapObjectsMock = vi.fn();
const useMapPortalsMock = vi.fn();
const useMapNpcsMock = vi.fn();
const useMapReactorsMock = vi.fn();
vi.mock("@/lib/hooks/api/useMapEntities", () => ({
  useMapObjects: (...args: unknown[]) => useMapObjectsMock(...args),
  useMapPortals: (...args: unknown[]) => useMapPortalsMock(...args),
  useMapNpcs: (...args: unknown[]) => useMapNpcsMock(...args),
  useMapReactors: (...args: unknown[]) => useMapReactorsMock(...args),
}));

vi.mock("@/lib/utils/toast", () => ({
  success: vi.fn(),
  error: vi.fn(),
}));

// FR-19: capture the props MapImagePanel receives so B1's pin-passthrough
// (definition entities via hooks, live monsters via the adapter) can be
// asserted without re-testing MapImagePanel's own rendering.
const mapImagePanelPropsSpy = vi.fn();
vi.mock("@/components/features/maps/MapImagePanel", () => ({
  MapImagePanel: (props: Record<string, unknown>) => {
    mapImagePanelPropsSpy(props);
    return <div data-testid="map-image-panel" />;
  },
}));

// FieldCharactersTab (rendered by FieldTabs' `characters` slot) enriches
// each id via useCharacter/useJobNameLookup; stub both so this page's tests
// stay focused on the page shell, not the tab's own row rendering (that's
// FieldCharactersTab.test.tsx's job).
vi.mock("@/lib/hooks/api/useCharacters", () => ({
  useCharacter: () => ({
    data: undefined,
    isLoading: false,
    isError: false,
    error: null,
  }),
}));
vi.mock("@/lib/hooks/api/useJobGraph", () => ({
  useJobNameLookup: () => (id: number) => `Job ${id}`,
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

function makeFieldCharacters(count: number): FieldCharacterData[] {
  return Array.from({ length: count }, (_, i) => ({
    id: `${100 + i}`,
    type: "characters",
  }));
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

function makePortal(id: string): MapPortalData {
  return {
    id,
    type: "portals",
    attributes: {
      name: id,
      target: "",
      type: 0,
      x: 0,
      y: 0,
      targetMapId: 999999999,
      scriptName: "",
    },
  };
}

function makeNpc(id: string): MapNpcData {
  return {
    id,
    type: "npcs",
    attributes: {
      template: 9000000,
      name: id,
      cy: 0,
      x: 0,
      y: 0,
      f: 0,
      fh: 0,
      rx0: 0,
      rx1: 0,
      hide: false,
    },
  };
}

function makeReactor(id: string): MapReactorData {
  return {
    id,
    type: "reactors",
    attributes: {
      classification: 1,
      name: id,
      x: 0,
      y: 0,
      delay: 0,
      direction: 0,
    },
  };
}

function makeMapObject(id: string): MapObjectData {
  return {
    id,
    type: "map-objects",
    attributes: {
      kind: "ENVIRONMENT",
      name: id,
      objectSource: "effect",
      l0: "quest",
      l1: "gate",
      l2: "1",
      x: 0,
      y: 0,
      z: 0,
      layer: 0,
    },
  };
}

function LocationSpy() {
  const location = useLocation();
  return <div data-testid="location-search">{location.search}</div>;
}

// Navigates externally (not via a tab-trigger click) so a test can verify
// FieldTabs stays in sync with an incoming `tab` prop change — the one thing
// a `defaultValue={tab}` regression would silently break (findings, task 18).
function NavigateButton({ to }: { to: string }) {
  const navigate = useNavigate();
  return <button onClick={() => navigate(to)}>External tab navigation</button>;
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
      <NavigateButton
        to={`/fields/0/1/910340000/${INSTANCE_ID}?tab=monsters`}
      />
    </>,
    { wrapper },
  );
}

describe("FieldDetailPage", () => {
  beforeEach(() => {
    useMapMock.mockReset();
    useWorldsMock.mockReset();
    useFieldCharactersMock.mockReset();
    useLiveMonstersMock.mockReset();
    useMapObjectsMock.mockReset();
    useMapPortalsMock.mockReset();
    useMapNpcsMock.mockReset();
    useMapReactorsMock.mockReset();
    mapImagePanelPropsSpy.mockReset();
    vi.mocked(toast.error).mockReset();
    vi.mocked(toast.success).mockReset();

    useMapMock.mockReturnValue(queryResult(makeMap()));
    useWorldsMock.mockReturnValue(
      queryResult([makeWorld("1", "Bera"), makeWorld("2", "Kradia")]),
    );
    useFieldCharactersMock.mockReturnValue(queryResult(makeFieldCharacters(2)));
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
    useMapPortalsMock.mockReturnValue(queryResult([makePortal("p1")]));
    useMapNpcsMock.mockReturnValue(queryResult([makeNpc("n1")]));
    useMapReactorsMock.mockReturnValue(queryResult([makeReactor("r1")]));
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
    useFieldCharactersMock.mockReturnValue(queryResult(makeFieldCharacters(3)));
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
    useFieldCharactersMock.mockReturnValue(queryResult(makeFieldCharacters(3)));
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

  it("tab stays in sync with an externally-changed ?tab= (controlled, not defaultValue)", async () => {
    const user = userEvent.setup();
    renderPage();

    expect(screen.getByRole("tab", { name: /characters/i })).toHaveAttribute(
      "data-state",
      "active",
    );

    // Navigates via a plain link, not by clicking the tab trigger — only a
    // `value={tab}` binding reacts to this; `defaultValue={tab}` would leave
    // the previously-active tab showing.
    await user.click(
      screen.getByRole("button", { name: /external tab navigation/i }),
    );

    expect(screen.getByRole("tab", { name: /monsters/i })).toHaveAttribute(
      "data-state",
      "active",
    );
    expect(screen.getByRole("tab", { name: /characters/i })).toHaveAttribute(
      "data-state",
      "inactive",
    );
  });

  it("torn-down field", () => {
    useFieldCharactersMock.mockReturnValue(queryResult([]));
    renderPage();

    expect(screen.getByText(/may have been torn down/i)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /fields/i })).toHaveAttribute(
      "href",
      "/fields",
    );
    expect(toast.error).not.toHaveBeenCalled();
  });

  it("FR-19: passes definition entity pins and live-monster pins to MapImagePanel", () => {
    renderPage();

    expect(mapImagePanelPropsSpy).toHaveBeenCalled();
    const props = mapImagePanelPropsSpy.mock.calls[0]?.[0] as {
      portals?: MapPortalData[];
      npcs?: MapNpcData[];
      reactors?: MapReactorData[];
      monsters?: { id: string; attributes: { template: number } }[];
    };

    expect(props.portals).toEqual([makePortal("p1")]);
    expect(props.npcs).toEqual([makeNpc("n1")]);
    expect(props.reactors).toEqual([makeReactor("r1")]);
    // Monster pins come from LIVE monsters (monstersQuery.data), not
    // declared spawn points, adapted to { id, attributes: { template, x, y } }.
    expect(props.monsters).toEqual([
      { id: "m1", attributes: { template: 100100, x: 0, y: 0 } },
      { id: "m2", attributes: { template: 100100, x: 0, y: 0 } },
      { id: "m3", attributes: { template: 100101, x: 0, y: 0 } },
    ]);
  });

  it("exposes refresh and last updated", () => {
    renderPage();

    expect(
      screen.getByRole("button", { name: /refresh/i }),
    ).toBeInTheDocument();
    expect(screen.getByText(/last updated/i)).toBeInTheDocument();
  });

  it("refresh also refetches the FR-19 pin queries (portals, npcs, reactors), not just map/characters/monsters/objects", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: /refresh/i }));

    const mapResult = useMapMock.mock.results[0]?.value as {
      refetch: () => Promise<unknown>;
    };
    const charactersResult = useFieldCharactersMock.mock.results[0]?.value as {
      refetch: () => Promise<unknown>;
    };
    const monstersResult = useLiveMonstersMock.mock.results[0]?.value as {
      refetch: () => Promise<unknown>;
    };
    const objectsResult = useMapObjectsMock.mock.results[0]?.value as {
      refetch: () => Promise<unknown>;
    };
    const portalsResult = useMapPortalsMock.mock.results[0]?.value as {
      refetch: () => Promise<unknown>;
    };
    const npcsResult = useMapNpcsMock.mock.results[0]?.value as {
      refetch: () => Promise<unknown>;
    };
    const reactorsResult = useMapReactorsMock.mock.results[0]?.value as {
      refetch: () => Promise<unknown>;
    };

    expect(mapResult.refetch).toHaveBeenCalled();
    expect(charactersResult.refetch).toHaveBeenCalled();
    expect(monstersResult.refetch).toHaveBeenCalled();
    expect(objectsResult.refetch).toHaveBeenCalled();
    expect(portalsResult.refetch).toHaveBeenCalled();
    expect(npcsResult.refetch).toHaveBeenCalled();
    expect(reactorsResult.refetch).toHaveBeenCalled();
  });
});
