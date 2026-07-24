import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { PresetPreviewCard } from "../PresetPreviewCard";
import { DEFAULT_PRESET_ATTRIBUTES } from "../presetEditorState";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));
vi.mock("@/lib/hooks/useCharacterImage", () => ({
  useCharacterImage: () => ({
    isLoading: false,
    isError: false,
    imageUrl: "http://img/preview.png",
    refetch: vi.fn(),
  }),
}));
vi.mock("@/lib/hooks/api/useItemStrings", () => ({
  useItemName: () => ({ data: "Item" }),
}));

describe("PresetPreviewCard", () => {
  it("renders the composited preview image", () => {
    render(<PresetPreviewCard attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }} />);
    expect(screen.getByRole("img", { name: /live preview/i })).toHaveAttribute(
      "src",
      "http://img/preview.png",
    );
  });

  it("shows a worn-icon for each placeable equipment id", () => {
    render(
      <PresetPreviewCard
        attrs={{
          ...DEFAULT_PRESET_ATTRIBUTES,
          equipment: [{ templateId: 1040002, useAverageStats: true }],
        }}
      />,
    );
    expect(screen.getAllByTestId("worn-icon")).toHaveLength(1);
  });
});
