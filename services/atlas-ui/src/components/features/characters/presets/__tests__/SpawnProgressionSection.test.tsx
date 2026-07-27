import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { SpawnProgressionSection } from "../SpawnProgressionSection";
import { DEFAULT_PRESET_ATTRIBUTES } from "../presetEditorState";

vi.mock("../../templates/MapPicker", () => ({
  MapPicker: ({
    value,
    onChange,
  }: {
    value: number;
    onChange: (n: number) => void;
  }) => (
    <button aria-label="map-picker" onClick={() => onChange(100000000)}>
      map:{value}
    </button>
  ),
}));

describe("SpawnProgressionSection", () => {
  it("wires MapPicker to mapId", async () => {
    const onSetField = vi.fn();
    render(
      <SpawnProgressionSection
        attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }}
        onSetField={onSetField}
      />,
    );
    await userEvent.click(screen.getByLabelText("map-picker"));
    expect(onSetField).toHaveBeenCalledWith("mapId", 100000000);
  });

  it("edits level within 1..250", async () => {
    const onSetField = vi.fn();
    render(
      <SpawnProgressionSection
        attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }}
        onSetField={onSetField}
      />,
    );
    const level = screen.getByLabelText(/^level/i);
    await userEvent.clear(level);
    await userEvent.type(level, "30");
    expect(onSetField).toHaveBeenCalledWith("level", 30);
  });

  it("edits meso", async () => {
    const onSetField = vi.fn();
    render(
      <SpawnProgressionSection
        attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }}
        onSetField={onSetField}
      />,
    );
    const meso = screen.getByLabelText(/^meso/i);
    await userEvent.clear(meso);
    await userEvent.type(meso, "5000");
    expect(onSetField).toHaveBeenCalledWith("meso", 5000);
  });

  it("edits GM level", async () => {
    const onSetField = vi.fn();
    render(
      <SpawnProgressionSection
        attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }}
        onSetField={onSetField}
      />,
    );
    const gm = screen.getByLabelText(/^gm level/i);
    await userEvent.clear(gm);
    await userEvent.type(gm, "2");
    expect(onSetField).toHaveBeenCalledWith("gm", 2);
  });

  it("re-syncs local drafts when switching to a different preset (no stale value)", () => {
    const onSetField = vi.fn();
    const { rerender } = render(
      <SpawnProgressionSection
        attrs={{ ...DEFAULT_PRESET_ATTRIBUTES, level: 10, meso: 100, gm: 0 }}
        onSetField={onSetField}
      />,
    );
    expect(screen.getByLabelText(/^level/i)).toHaveValue(10);

    rerender(
      <SpawnProgressionSection
        attrs={{
          ...DEFAULT_PRESET_ATTRIBUTES,
          level: 200,
          meso: 999999,
          gm: 5,
        }}
        onSetField={onSetField}
      />,
    );
    expect(screen.getByLabelText(/^level/i)).toHaveValue(200);
    expect(screen.getByLabelText(/^meso/i)).toHaveValue(999999);
    expect(screen.getByLabelText(/^gm level/i)).toHaveValue(5);
  });
});
