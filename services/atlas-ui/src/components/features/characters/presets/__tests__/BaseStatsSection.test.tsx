import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { BaseStatsSection } from "../BaseStatsSection";
import { DEFAULT_PRESET_ATTRIBUTES } from "../presetEditorState";

describe("BaseStatsSection", () => {
  it("renders all six stats and edits STR", async () => {
    const onSetStat = vi.fn();
    render(<BaseStatsSection attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }} onSetStat={onSetStat} />);
    for (const s of ["STR", "DEX", "INT", "LUK", "HP", "MP"]) {
      expect(screen.getByLabelText(s)).toBeInTheDocument();
    }
    const str = screen.getByLabelText("STR");
    await userEvent.clear(str);
    await userEvent.type(str, "13");
    expect(onSetStat).toHaveBeenCalledWith("str", 13);
  });

  it("notes stats are written verbatim", () => {
    render(<BaseStatsSection attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }} onSetStat={vi.fn()} />);
    expect(screen.getByText(/written verbatim/i)).toBeInTheDocument();
  });

  it("re-syncs the draft values when the attrs switch (e.g. selecting a different preset)", () => {
    const { rerender } = render(
      <BaseStatsSection attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }} onSetStat={vi.fn()} />,
    );
    expect(screen.getByLabelText("STR")).toHaveValue(DEFAULT_PRESET_ATTRIBUTES.stats.str);

    const otherAttrs = {
      ...DEFAULT_PRESET_ATTRIBUTES,
      stats: { str: 99, dex: 88, int: 77, luk: 66, hp: 5555, mp: 4444 },
    };
    rerender(<BaseStatsSection attrs={otherAttrs} onSetStat={vi.fn()} />);

    expect(screen.getByLabelText("STR")).toHaveValue(99);
    expect(screen.getByLabelText("DEX")).toHaveValue(88);
    expect(screen.getByLabelText("INT")).toHaveValue(77);
    expect(screen.getByLabelText("LUK")).toHaveValue(66);
    expect(screen.getByLabelText("HP")).toHaveValue(5555);
    expect(screen.getByLabelText("MP")).toHaveValue(4444);
  });
});
