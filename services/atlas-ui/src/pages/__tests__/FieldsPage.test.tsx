import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { MemoryRouter, useLocation } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { FieldsPage } from "@/pages/FieldsPage";
import type { Tenant } from "@/types/models/tenant";
import type { FieldData } from "@/services/api/fields.service";
import type { WorldData, ChannelData } from "@/services/api/worlds.service";

// Radix Select relies on DOM APIs jsdom does not implement.
Element.prototype.hasPointerCapture ||= () => false;
Element.prototype.scrollIntoView ||= () => {};

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

const useFieldsMock = vi.fn();
vi.mock("@/lib/hooks/api/useFields", () => ({
  useFields: (...args: unknown[]) => useFieldsMock(...args),
}));

const useWorldsMock = vi.fn();
const useChannelsMock = vi.fn();
vi.mock("@/lib/hooks/api/useWorlds", () => ({
  useWorlds: () => useWorldsMock(),
  useChannels: (worldId: number) => useChannelsMock(worldId),
}));

const useMapNamesMock = vi.fn();
vi.mock("@/lib/hooks/api/useMaps", () => ({
  useMapNames: (ids: number[]) => useMapNamesMock(ids),
}));

// bug-fields-ui item 6: FieldsPage dispatches to FieldDetailPage on
// `?instance=`. FieldDetailPage's own rendering is FieldDetailPage.test.tsx's
// job — stub it here to a marker so this file only asserts the dispatch.
vi.mock("@/pages/FieldDetailPage", () => ({
  FieldDetailPage: () => <div data-testid="field-detail-page" />,
}));

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

function makeChannel(worldId: number, channelId: number): ChannelData {
  return {
    id: `${worldId}-${channelId}`,
    type: "channels",
    attributes: {
      worldId,
      channelId,
      ipAddress: "127.0.0.1",
      port: 8484,
      currentCapacity: 0,
      maxCapacity: 0,
      createdAt: "",
      expRate: 0,
      mesoRate: 0,
      itemDropRate: 0,
      questExpRate: 0,
    },
  };
}

function makeField(
  worldId: number,
  channelId: number,
  mapId: number,
  instanceId: string,
  characterCount = 1,
): FieldData {
  return {
    id: `${worldId}:${channelId}:${mapId}:${instanceId}`,
    type: "fields",
    attributes: { worldId, channelId, mapId, instanceId, characterCount },
  };
}

const WORLDS: WorldData[] = [makeWorld("1", "Bera"), makeWorld("0", "Scania")];
const CHANNELS_WORLD_0: ChannelData[] = [
  makeChannel(0, 1),
  makeChannel(0, 2),
  makeChannel(0, 3),
];
const FIELDS: FieldData[] = [
  makeField(0, 1, 910340000, "instance-a"),
  makeField(0, 3, 100000000, "instance-b"),
];

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

function LocationSpy() {
  const location = useLocation();
  return <div data-testid="location-search">{location.search}</div>;
}

function renderPage(initialPath = "/fields") {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[initialPath]}>{children}</MemoryRouter>
    </QueryClientProvider>
  );
  return render(
    <>
      <FieldsPage />
      <LocationSpy />
    </>,
    { wrapper },
  );
}

describe("FieldsPage", () => {
  beforeEach(() => {
    useFieldsMock.mockReset();
    useWorldsMock.mockReset();
    useChannelsMock.mockReset();
    useMapNamesMock.mockReset();

    useFieldsMock.mockReturnValue(queryResult(FIELDS));
    useWorldsMock.mockReturnValue(queryResult(WORLDS));
    useChannelsMock.mockReturnValue(queryResult(CHANNELS_WORLD_0));
    useMapNamesMock.mockReturnValue({});
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("selects no field on load", async () => {
    renderPage();

    expect(
      screen.queryByText(/field detail|selected field/i),
    ).not.toBeInTheDocument();
    expect(screen.getByLabelText("World")).toBeInTheDocument();
    expect(screen.getByLabelText("Map filter")).toBeInTheDocument();
  });

  it("defaults to the lowest-numbered world", async () => {
    renderPage();

    expect(screen.getByLabelText("World")).toHaveTextContent("Scania");
  });

  it("channel defaults to Any channel", async () => {
    renderPage();

    expect(screen.getByLabelText("Channel")).toHaveTextContent("Any channel");
  });

  it("world and channel options come from the API", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByLabelText("World"));
    expect(
      await screen.findByRole("option", { name: "Scania" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "Bera" })).toBeInTheDocument();
    await user.click(screen.getByRole("option", { name: "Scania" }));

    // CHANNELS_WORLD_0 has channelId 1/2/3 (0-based); options display
    // 1-indexed (item 15) — the underlying values stay 0-based.
    await user.click(screen.getByLabelText("Channel"));
    expect(
      await screen.findByRole("option", { name: "2" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "3" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "4" })).toBeInTheDocument();
  });

  it("does not render a hard-coded world list when useWorlds returns []", async () => {
    useWorldsMock.mockReturnValue(queryResult([]));
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByLabelText("World"));
    expect(
      screen.queryByRole("option", { name: "Scania" }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("option", { name: "Bera" }),
    ).not.toBeInTheDocument();
  });

  it("map filter searches name and id", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.type(screen.getByLabelText("Map filter"), "9103");

    await waitFor(() => {
      expect(screen.getByText("910340000")).toBeInTheDocument();
      expect(screen.queryByText("100000000")).not.toBeInTheDocument();
    });
  });

  it("map filter matches on map name", async () => {
    useMapNamesMock.mockReturnValue({ 910340000: "Henesys" });
    const user = userEvent.setup();
    renderPage();

    await user.type(screen.getByLabelText("Map filter"), "Henesys");

    await waitFor(() => {
      expect(screen.getByText("Henesys")).toBeInTheDocument();
      expect(screen.queryByText("100000000")).not.toBeInTheDocument();
    });
  });

  it("map filter is a text input, not a select", () => {
    renderPage();

    const mapFilter = screen.getByLabelText("Map filter");
    expect(mapFilter.tagName).toBe("INPUT");
    expect(screen.queryAllByRole("option").length).toBe(0);
  });

  it("?map= pre-fills the filter", async () => {
    renderPage("/fields?map=910340000");

    expect(screen.getByLabelText("Map filter")).toHaveValue("910340000");
    await waitFor(() => {
      expect(screen.getByText("910340000")).toBeInTheDocument();
      expect(screen.queryByText("100000000")).not.toBeInTheDocument();
    });
  });

  it("changing the filter writes back to the URL", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.type(screen.getByLabelText("Map filter"), "100000000");

    await waitFor(() => {
      expect(screen.getByTestId("location-search")).toHaveTextContent(
        "map=100000000",
      );
    });
  });

  it("empty state echoes the filters by name, not a missing map", async () => {
    useFieldsMock.mockReturnValue(queryResult([]));
    const user = userEvent.setup();
    renderPage();

    // Channel option labels display 1-indexed (item 15): raw channelId 3
    // is the option labelled "4".
    await user.click(screen.getByLabelText("Channel"));
    await user.click(await screen.findByRole("option", { name: "4" }));

    await waitFor(() => {
      const empty = screen.getByTestId("empty-state");
      expect(empty).toHaveTextContent("Scania");
      // Channel label is 1-indexed (item 4/7/15 convention): raw channelId 3
      // is displayed as "4", matching the selected option.
      expect(empty).toHaveTextContent("4");
      expect(empty).not.toHaveTextContent(/map.*missing/i);
    });
  });

  it("empty state offers clear filters", async () => {
    useFieldsMock.mockReturnValue(queryResult([]));
    const user = userEvent.setup();
    renderPage();

    // Channel option labels display 1-indexed (item 15): raw channelId 3
    // is the option labelled "4".
    await user.click(screen.getByLabelText("Channel"));
    await user.click(await screen.findByRole("option", { name: "4" }));
    await user.type(screen.getByLabelText("Map filter"), "910340000");

    const clearButton = await screen.findByRole("button", {
      name: "Clear filters",
    });
    await user.click(clearButton);

    await waitFor(() => {
      expect(screen.getByLabelText("World")).toHaveTextContent("Scania");
      expect(screen.getByLabelText("Channel")).toHaveTextContent("Any channel");
      expect(screen.getByLabelText("Map filter")).toHaveValue("");
    });
  });

  it("result columns are Channel, Map, Instance, Characters (in that order) and Map links to the field", async () => {
    useMapNamesMock.mockReturnValue({ 910340000: "Henesys" });
    renderPage();

    const headers = screen
      .getAllByRole("columnheader")
      .map((h) => h.textContent);
    expect(headers).toEqual(["Channel", "Map", "Instance", "Characters"]);

    const link = screen.getByRole("link", { name: "Henesys" });
    expect(link).toHaveAttribute(
      "href",
      "/fields?world=0&channel=1&map=910340000&instance=instance-a",
    );
  });

  it("channel is shown 1-indexed in the results grid (value stays 0-based)", async () => {
    renderPage();

    // FIELDS[0] has channelId 1 (0-based) — displayed as "2".
    expect(screen.getByRole("cell", { name: "2" })).toBeInTheDocument();
  });

  it("exposes refresh and last updated", () => {
    renderPage();

    expect(
      screen.getByRole("button", { name: /refresh/i }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("fields-last-updated")).toBeInTheDocument();
  });

  it("Refresh is an icon-only button, not a labelled button", () => {
    renderPage();

    const button = screen.getByRole("button", { name: /refresh/i });
    expect(button).not.toHaveTextContent(/refresh/i);
  });

  it("title and runtime badge each render on their own row (item 18)", () => {
    renderPage();

    const heading = screen.getByRole("heading", { name: "Fields" });
    const badge = screen.getByText("Runtime");
    expect(heading).toBeInTheDocument();
    expect(badge).toBeInTheDocument();
    // Not siblings in the same flex row — the badge's row is a distinct
    // element from the one containing the heading (item 18: badge below the
    // title, not inline with it).
    expect(badge.closest("h1")).toBeNull();
  });

  it("filter bar is a Search Fields card with a result count and a Clear button (item 19)", async () => {
    renderPage();

    expect(screen.getByText("Search Fields")).toBeInTheDocument();
    expect(screen.getByText(/fields match/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Clear" })).toBeInTheDocument();
  });

  it("dispatches to the field-detail view when ?instance= is present (item 6)", () => {
    renderPage("/fields?world=0&channel=1&map=910340000&instance=instance-a");

    expect(screen.getByTestId("field-detail-page")).toBeInTheDocument();
    expect(screen.queryByLabelText("World")).not.toBeInTheDocument();
  });

  it("does not poll", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    renderPage();

    const callCountBefore = useFieldsMock.mock.calls.length;
    vi.advanceTimersByTime(60_000);
    const callCountAfter = useFieldsMock.mock.calls.length;

    // No new render cycle should be triggered purely by the passage of time
    // — useFields itself is mocked, so a poll would show up as additional
    // invocations of the hook with no user or prop-driven cause.
    expect(callCountAfter).toBe(callCountBefore);
  });
});
