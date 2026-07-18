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

// 20000-20009 male, 21000-21009 female
const faceIds = [
  ...Array.from({ length: 30 }, (_, i) => 20000 + i),
  ...Array.from({ length: 10 }, (_, i) => 21000 + i),
];

function renderDialog(over: Record<string, unknown> = {}) {
  return render(
    <AppearanceBrowserDialog
      dimension="faces"
      template={normalizeTemplate({ gender: 0, faces: [20000] })}
      picks={DEFAULT_PICKS}
      open
      onOpenChange={vi.fn()}
      onAdd={vi.fn()}
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
    const onAdd = vi.fn();
    renderDialog({ onAdd });
    await userEvent.click(
      screen.getByRole("button", { name: /add face 20001/i }),
    );
    expect(onAdd).toHaveBeenCalledWith(20001);
  });

  it("resolves names for the current page", () => {
    renderDialog();
    expect(useItemNamesMock).toHaveBeenCalled();
    expect(screen.getByText("Male 2")).toBeInTheDocument();
  });

  it("hairColors offers digits 0-7 on the current base hair (no enumeration)", () => {
    renderDialog({
      dimension: "hairColors",
      template: normalizeTemplate({ hairs: [30030], hairColors: [0] }),
    });
    expect(
      screen.getAllByRole("button", { name: /add hair color/i }),
    ).toHaveLength(8);
    expect(
      screen.getByRole("button", { name: /add hair color 0/i }),
    ).toBeDisabled();
  });

  it("skinColors offers 0-9 rendered previews", () => {
    renderDialog({
      dimension: "skinColors",
      template: normalizeTemplate({}),
    });
    expect(
      screen.getAllByRole("button", { name: /add skin tone/i }),
    ).toHaveLength(10);
  });
});
