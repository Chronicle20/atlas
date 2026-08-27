import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { MapleLifeClassDraft } from "../mapleLifeEditorState";
import { ProgressionSection } from "../ProgressionSection";

// Reused from IdentitySection.test.tsx.
function buildDraft(
  overrides: Partial<MapleLifeClassDraft> = {},
): MapleLifeClassDraft {
  return {
    ordinal: 0,
    gender: 0,
    jobId: 100,
    level: 30,
    mapId: 102000000,
    stats: { str: 4, dex: 4, int: 4, luk: 4, hp: 50, mp: 50 },
    ap: 0,
    spBooks: [0, 0, 0, 0, 0, 0, 0, 0, 0, 0],
    spRaw: "0,0,0,0,0,0,0,0,0,0",
    spSkillId: undefined,
    meso: 0,
    equipment: [],
    inventory: [],
    present: true,
    ...overrides,
  };
}

describe("ProgressionSection", () => {
  it("labels stats as the skill-excluded midpoint", () => {
    render(
      <ProgressionSection
        draft={buildDraft()}
        onSetStat={vi.fn()}
        onSetScalar={vi.fn()}
        onSetSpBook={vi.fn()}
      />,
    );
    expect(
      screen.getByText(/EXCLUDE the SP skill's own 29 × effectX/),
    ).toBeInTheDocument();
  });

  it("labels ap and sp as unspent at level", () => {
    render(
      <ProgressionSection
        draft={buildDraft()}
        onSetStat={vi.fn()}
        onSetScalar={vi.fn()}
        onSetSpBook={vi.fn()}
      />,
    );
    expect(
      screen.getByText(/remains UNSPENT at the configured level/i),
    ).toBeInTheDocument();
  });

  it("renders ten book inputs", () => {
    render(
      <ProgressionSection
        draft={buildDraft()}
        onSetStat={vi.fn()}
        onSetScalar={vi.fn()}
        onSetSpBook={vi.fn()}
      />,
    );
    const bookInputs = screen.getAllByRole("spinbutton", {
      name: /^Book [0-9]$/,
    });
    expect(bookInputs).toHaveLength(10);
  });

  it("distinguishes book 0", () => {
    render(
      <ProgressionSection
        draft={buildDraft()}
        onSetStat={vi.fn()}
        onSetScalar={vi.fn()}
        onSetSpBook={vi.fn()}
      />,
    );
    expect(
      screen.getByText(/only book Maple Life reads or spends/i),
    ).toBeInTheDocument();
  });

  it("editing a book reports its index and value", async () => {
    const user = userEvent.setup();
    const onSetSpBook = vi.fn();
    render(
      <ProgressionSection
        draft={buildDraft()}
        onSetStat={vi.fn()}
        onSetScalar={vi.fn()}
        onSetSpBook={onSetSpBook}
      />,
    );
    const book0 = screen.getByRole("spinbutton", { name: "Book 0" });
    await user.clear(book0);
    await user.type(book0, "75");
    expect(onSetSpBook).toHaveBeenLastCalledWith(0, 75);
  });

  it("a mid-edit keystroke is not clobbered", async () => {
    const user = userEvent.setup();
    render(
      <ProgressionSection
        draft={buildDraft()}
        onSetStat={vi.fn()}
        onSetScalar={vi.fn()}
        onSetSpBook={vi.fn()}
      />,
    );
    const book0 = screen.getByRole("spinbutton", { name: "Book 0" });
    await user.clear(book0);
    expect(book0).toHaveValue(null);
  });

  it("an unparseable pool disables the books and shows the raw value", () => {
    render(
      <ProgressionSection
        draft={buildDraft({ spBooks: [], spRaw: "61,0" })}
        onSetStat={vi.fn()}
        onSetScalar={vi.fn()}
        onSetSpBook={vi.fn()}
      />,
    );
    const bookInputs = screen.getAllByRole("spinbutton", {
      name: /^Book [0-9]$/,
    });
    for (const input of bookInputs) {
      expect(input).toBeDisabled();
    }
    expect(screen.getByText(/preserved as loaded: 61,0/)).toBeInTheDocument();
  });

  it("renders the six stat inputs", () => {
    render(
      <ProgressionSection
        draft={buildDraft()}
        onSetStat={vi.fn()}
        onSetScalar={vi.fn()}
        onSetSpBook={vi.fn()}
      />,
    );
    for (const label of ["STR", "DEX", "INT", "LUK", "HP", "MP"]) {
      expect(
        screen.getByRole("spinbutton", { name: label }),
      ).toBeInTheDocument();
    }
  });

  it("editing a stat reports its key", async () => {
    const user = userEvent.setup();
    const onSetStat = vi.fn();
    render(
      <ProgressionSection
        draft={buildDraft()}
        onSetStat={onSetStat}
        onSetScalar={vi.fn()}
        onSetSpBook={vi.fn()}
      />,
    );
    const str = screen.getByRole("spinbutton", { name: "STR" });
    await user.clear(str);
    await user.type(str, "36");
    expect(onSetStat).toHaveBeenLastCalledWith("str", 36);
  });

  it("editing ap reports the scalar key", async () => {
    const user = userEvent.setup();
    const onSetScalar = vi.fn();
    render(
      <ProgressionSection
        draft={buildDraft()}
        onSetStat={vi.fn()}
        onSetScalar={onSetScalar}
        onSetSpBook={vi.fn()}
      />,
    );
    const ap = screen.getByRole("spinbutton", { name: "AP" });
    await user.clear(ap);
    await user.type(ap, "124");
    expect(onSetScalar).toHaveBeenLastCalledWith("ap", 124);
  });

  it("renders field errors", () => {
    render(
      <ProgressionSection
        draft={buildDraft()}
        onSetStat={vi.fn()}
        onSetScalar={vi.fn()}
        onSetSpBook={vi.fn()}
        errors={{ sp: ["bad pool"] }}
      />,
    );
    expect(screen.getByText("bad pool")).toBeInTheDocument();
  });
});
