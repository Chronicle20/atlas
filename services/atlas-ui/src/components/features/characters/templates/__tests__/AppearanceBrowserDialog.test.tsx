import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";

// Radix Dialog/Select/Switch rely on DOM APIs jsdom does not implement.
Element.prototype.hasPointerCapture ||= () => false;
Element.prototype.scrollIntoView ||= () => {};

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

const useFaceIdsMock = vi.fn();
const useHairIdsMock = vi.fn();
vi.mock("@/lib/hooks/api/useCosmetics", () => ({
  useFaceIds: (...a: unknown[]) => useFaceIdsMock(...a),
  useHairIds: (...a: unknown[]) => useHairIdsMock(...a),
}));

const useItemNamesMock = vi.fn();
vi.mock("@/lib/hooks/api/useItemNames", () => ({
  useItemNames: (...a: unknown[]) => useItemNamesMock(...a),
}));

import { normalizeTemplate, DEFAULT_PICKS } from "../editorState";
import { AppearanceBrowserDialog, PAGE_SIZE } from "../AppearanceBrowserDialog";
import { collapseHairBases } from "../hairBases";
import { buildVariantLoadout } from "../previewLoadout";

// 20000-20009 male, 21000-21009 female
const faceIds = [
  ...Array.from({ length: 30 }, (_, i) => 20000 + i),
  ...Array.from({ length: 10 }, (_, i) => 21000 + i),
];

function renderDialog(over: Record<string, unknown> = {}) {
  const template = normalizeTemplate({ gender: 0, faces: [20000] });
  return render(
    <AppearanceBrowserDialog
      dimension="faces"
      gender={template.gender}
      variantLoadout={(dim, id) =>
        buildVariantLoadout(template, DEFAULT_PICKS, dim, id)
      }
      open
      onOpenChange={vi.fn()}
      onSelect={vi.fn()}
      markedIds={template.faces}
      {...over}
    />,
  );
}

beforeEach(() => {
  useFaceIdsMock.mockReturnValue({
    data: faceIds,
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  });
  useHairIdsMock.mockReturnValue({
    data: [],
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  });
  useItemNamesMock.mockReturnValue({ 20001: "Male 2" });
});

describe("AppearanceBrowserDialog", () => {
  it("gender-filters candidates by the id convention, with a show-all toggle", async () => {
    renderDialog();
    // male template: female id 21000 hidden
    expect(screen.queryByText("21000")).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole("switch", { name: /show all/i }));
    expect(screen.getByText("21000")).toBeInTheDocument();
  });

  it("caps the grid at PAGE_SIZE per page and pages through candidates", async () => {
    renderDialog();
    // 30 male faces → page 1 shows PAGE_SIZE, 20029 is on page 2
    expect(screen.getAllByRole("button", { name: /add face/i })).toHaveLength(
      PAGE_SIZE,
    );
    expect(screen.queryByText("20029")).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: /next/i }));
    expect(screen.getByText("20029")).toBeInTheDocument();
  });

  it("marks already-in-pool ids as disabled", () => {
    renderDialog();
    expect(
      screen.getByRole("button", { name: /add face 20000/i }),
    ).toBeDisabled();
  });

  it("clicking a candidate adds it", async () => {
    const onSelect = vi.fn();
    renderDialog({ onSelect });
    await userEvent.click(
      screen.getByRole("button", { name: /add face 20001/i }),
    );
    expect(onSelect).toHaveBeenCalledWith(20001);
  });

  it("resolves names for the current page", () => {
    renderDialog();
    expect(useItemNamesMock).toHaveBeenCalled();
    expect(screen.getByText("Male 2")).toBeInTheDocument();
  });

  it("hairColors offers digits 0-7 on the current base hair (no enumeration)", () => {
    const template = normalizeTemplate({ hairs: [30030], hairColors: [0] });
    renderDialog({
      dimension: "hairColors",
      gender: template.gender,
      variantLoadout: (dim: string, id: number) =>
        buildVariantLoadout(template, DEFAULT_PICKS, dim as "hairColors", id),
      markedIds: template.hairColors,
    });
    expect(
      screen.getAllByRole("button", { name: /add hair color/i }),
    ).toHaveLength(8);
    expect(
      screen.getByRole("button", { name: /add hair color 0/i }),
    ).toBeDisabled();
  });

  it("skinColors offers 0-9 rendered previews", () => {
    const template = normalizeTemplate({});
    renderDialog({
      dimension: "skinColors",
      gender: template.gender,
      variantLoadout: (dim: string, id: number) =>
        buildVariantLoadout(template, DEFAULT_PICKS, dim as "skinColors", id),
      markedIds: template.skinColors,
    });
    expect(
      screen.getAllByRole("button", { name: /add skin tone/i }),
    ).toHaveLength(10);
  });

  it("replace mode: clicking a thumb calls onSelect and shows a selection ring on selectedId", async () => {
    const onSelect = vi.fn();
    render(
      <AppearanceBrowserDialog
        dimension="skinColors"
        gender={0}
        variantLoadout={(_d, id) => ({
          skin: id,
          hair: 30030,
          face: 20000,
          equipment: {},
          gender: 0,
        })}
        open
        onOpenChange={() => {}}
        onSelect={onSelect}
        selectMode="replace"
        selectedId={2}
      />,
    );
    // skin candidates 0-9 render; the selectedId thumb is marked selected
    const selected = await screen.findByRole("button", {
      name: /skin tone 2/i,
    });
    expect(selected.className).toMatch(/ring/);
    const other = screen.getByRole("button", { name: /skin tone 5/i });
    await userEvent.click(other);
    expect(onSelect).toHaveBeenCalledWith(5);
  });
});

describe("collapseHairBases", () => {
  it("collapses color variants to one base entry rendering the lowest digit", () => {
    expect(collapseHairBases([30000, 30002, 30030, 30031, 31000])).toEqual([
      { value: 30000, renderId: 30000 },
      { value: 30030, renderId: 30030 },
      { value: 31000, renderId: 31000 },
    ]);
  });

  it("renders the lowest EXISTING variant when the black one is absent", () => {
    expect(collapseHairBases([30033, 30031])).toEqual([
      { value: 30030, renderId: 30031 },
    ]);
  });
});
