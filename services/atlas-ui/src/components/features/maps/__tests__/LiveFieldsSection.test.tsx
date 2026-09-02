import { render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi, beforeEach } from "vitest";
import type { FieldData } from "@/services/api/fields.service";

const useFieldsForMapMock = vi.fn();
vi.mock("@/lib/hooks/api/useFields", () => ({
  useFieldsForMap: (...a: unknown[]) => useFieldsForMapMock(...a),
}));

const useLiveMonstersMock = vi.fn();
vi.mock("@/lib/hooks/api/useFieldRuntime", () => ({
  useLiveMonsters: (...a: unknown[]) => useLiveMonstersMock(...a),
  fieldRuntimeKeys: {
    monsters: (w: number, c: number, m: number, i: string) => [
      "fields",
      w,
      c,
      m,
      i,
      "monsters",
    ],
  },
}));

import { LiveFieldsSection } from "../LiveFieldsSection";

const MAP_ID = "910340000";

function makeFields(n: number): FieldData[] {
  return Array.from({ length: n }, (_, i) => ({
    id: `0:${i + 1}:${MAP_ID}:00000000-0000-0000-0000-00000000000${i}`,
    type: "fields",
    attributes: {
      worldId: 0,
      channelId: i + 1,
      mapId: Number(MAP_ID),
      instanceId: `00000000-0000-0000-0000-00000000000${i}`,
      characterCount: i + 1,
    },
  }));
}

function renderSection() {
  return render(
    <MemoryRouter>
      <LiveFieldsSection mapId={MAP_ID} />
    </MemoryRouter>,
  );
}

describe("LiveFieldsSection", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useLiveMonstersMock.mockReturnValue({
      data: [1, 2],
      isLoading: false,
      error: null,
    });
  });

  it("renders one row per live field", () => {
    useFieldsForMapMock.mockReturnValue({
      data: makeFields(3),
      isLoading: false,
      error: null,
    });
    renderSection();
    const rows = screen.getAllByRole("row");
    expect(rows).toHaveLength(4); // header + 3
    const cells = within(rows[1]!).getAllByRole("cell");
    expect(cells[3]).toHaveTextContent("1"); // characterCount
  });

  it("empty state is explicit and never hidden", () => {
    useFieldsForMapMock.mockReturnValue({
      data: [],
      isLoading: false,
      error: null,
    });
    renderSection();
    expect(
      screen.getByRole("heading", { name: /live fields/i }),
    ).toBeInTheDocument();
    expect(screen.getByText(/no live fields/i)).toBeInTheDocument();
  });

  it("each row links to the field page", () => {
    useFieldsForMapMock.mockReturnValue({
      data: makeFields(1),
      isLoading: false,
      error: null,
    });
    renderSection();
    const link = screen.getByRole("link", {
      name: "00000000-0000-0000-0000-000000000000",
    });
    expect(link).toHaveAttribute(
      "href",
      `/fields/0/1/${MAP_ID}/00000000-0000-0000-0000-000000000000`,
    );
  });

  it("offers a pre-filtered link into the locator", () => {
    useFieldsForMapMock.mockReturnValue({
      data: makeFields(3),
      isLoading: false,
      error: null,
    });
    renderSection();
    const link = screen.getByRole("link", { name: /view all in fields/i });
    expect(link).toHaveAttribute("href", `/fields?map=${MAP_ID}`);
  });

  it("fan-out is capped at 12", () => {
    useFieldsForMapMock.mockReturnValue({
      data: makeFields(15),
      isLoading: false,
      error: null,
    });
    renderSection();
    const enabledCalls = useLiveMonstersMock.mock.calls.filter(
      (call) => call[4] === true,
    );
    expect(enabledCalls).toHaveLength(12);
    expect(screen.getAllByText("—").length).toBeGreaterThanOrEqual(3);
    expect(screen.getByText(/first 12/i)).toBeInTheDocument();
  });

  it("a failed monster query does not unmount its row", () => {
    useFieldsForMapMock.mockReturnValue({
      data: makeFields(2),
      isLoading: false,
      error: null,
    });
    useLiveMonstersMock.mockImplementation((w: number, c: number) => {
      if (c === 1) {
        return { data: undefined, isLoading: false, error: new Error("boom") };
      }
      return { data: [1], isLoading: false, error: null };
    });
    renderSection();
    const rows = screen.getAllByRole("row");
    const cells = within(rows[1]!).getAllByRole("cell");
    expect(cells[3]).toHaveTextContent("1"); // characterCount
    expect(cells[4]).toHaveTextContent("—"); // monster count unavailable
  });

  it("spans all worlds and channels", () => {
    const fields: FieldData[] = [
      {
        id: "0:1:910340000:a",
        type: "fields",
        attributes: {
          worldId: 0,
          channelId: 1,
          mapId: 910340000,
          instanceId: "a",
          characterCount: 1,
        },
      },
      {
        id: "0:2:910340000:b",
        type: "fields",
        attributes: {
          worldId: 0,
          channelId: 2,
          mapId: 910340000,
          instanceId: "b",
          characterCount: 2,
        },
      },
      {
        id: "1:1:910340000:c",
        type: "fields",
        attributes: {
          worldId: 1,
          channelId: 1,
          mapId: 910340000,
          instanceId: "c",
          characterCount: 3,
        },
      },
    ];
    useFieldsForMapMock.mockReturnValue({
      data: fields,
      isLoading: false,
      error: null,
    });
    renderSection();
    expect(useFieldsForMapMock).toHaveBeenCalledWith(MAP_ID);
    expect(screen.getAllByRole("row")).toHaveLength(4); // header + 3
  });
});
