import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

vi.mock("../../templates/AppearanceBrowserDialog", () => ({
  AppearanceBrowserDialog: (props: { gender: number; open: boolean }) =>
    props.open ? <div data-gender={String(props.gender)} /> : null,
}));

import { AppearancePoolsSection } from "../AppearancePoolsSection";
import { DEFAULT_PICKS } from "../mapleLifeEditorState";
import type {
  MapleLifeClassDraft,
  MapleLifeLookDraft,
  PreviewPicks,
} from "../mapleLifeEditorState";

function draft(
  overrides: Partial<MapleLifeClassDraft> = {},
): MapleLifeClassDraft {
  return {
    ordinal: 0,
    gender: 0,
    jobId: 100,
    level: 30,
    mapId: 102000000,
    stats: { str: 35, dex: 4, int: 4, luk: 4, hp: 804, mp: 150 },
    ap: 123,
    spBooks: [61, 0, 0, 0, 0, 0, 0, 0, 0, 0],
    spRaw: "61,0,0,0,0,0,0,0,0,0",
    spSkillId: 1000001,
    meso: 100000,
    equipment: [],
    inventory: [],
    present: true,
    ...overrides,
  };
}

function look(overrides: Partial<MapleLifeLookDraft> = {}): MapleLifeLookDraft {
  return {
    gender: 0,
    faces: [20000, 20001, 20002],
    hairs: [30030, 30020, 30000],
    hairColors: [0, 7, 3, 2],
    skinColors: [0, 1, 2, 3],
    present: true,
    ...overrides,
  };
}

function picks(overrides: Partial<PreviewPicks> = {}): PreviewPicks {
  return { ...DEFAULT_PICKS, ...overrides };
}

function renderSection(over: Record<string, unknown> = {}) {
  return render(
    <AppearancePoolsSection
      look={look()}
      draft={draft()}
      picks={picks()}
      onPick={vi.fn()}
      onAddEntry={vi.fn()}
      onRemoveEntry={vi.fn()}
      {...over}
    />,
  );
}

describe("AppearancePoolsSection", () => {
  it("renders all four pool sections", () => {
    renderSection();
    expect(screen.getByRole("heading", { name: "Faces" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Hairs" })).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Hair colours" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Skin tones" }),
    ).toBeInTheDocument();
  });

  it("renders one thumb per entry", () => {
    renderSection({ look: look({ faces: [20000, 20001] }) });
    expect(screen.getAllByText("20000").length).toBeGreaterThan(0);
    expect(screen.getAllByText("20001").length).toBeGreaterThan(0);
  });

  it("states each dimension's value domain", () => {
    renderSection();
    expect(screen.getByText(/Normalised style ids/)).toBeInTheDocument();
    expect(screen.getByText(/Bare digits 0\.\.9/)).toBeInTheDocument();
    expect(screen.getByText(/Bare byte ordinals/)).toBeInTheDocument();
    expect(screen.getByText(/Full item ids/)).toBeInTheDocument();
  });

  it("states the allow-list semantics once per pool", () => {
    renderSection();
    expect(screen.getAllByText(/ErrLookInvalid rejections/)).toHaveLength(4);
  });

  it("clicking a thumb reports the pick for that dimension", async () => {
    const onPick = vi.fn();
    renderSection({ onPick });
    await userEvent.click(
      screen.getByRole("button", { name: /preview hair 30030/i }),
    );
    expect(onPick).toHaveBeenCalledWith("hairIdx", 0);
  });

  it("removing an entry reports the dimension and index", async () => {
    const onRemoveEntry = vi.fn();
    renderSection({ onRemoveEntry });
    await userEvent.click(
      screen.getByRole("button", { name: /remove face 20001/i }),
    );
    expect(onRemoveEntry).toHaveBeenCalledWith("faces", 1);
  });

  it("an emptied pool renders its blocking error", () => {
    renderSection({
      look: look({ faces: [] }),
      errors: { faces: ["This pool is empty…"] },
    });
    expect(screen.getByText("This pool is empty…")).toBeInTheDocument();
  });

  it("passes the look row's gender to the browser dialog", async () => {
    renderSection({ look: look({ gender: 1 }) });
    const addButtons = screen.getAllByRole("button", { name: /add/i });
    const faceAdd = addButtons[0];
    if (faceAdd === undefined) throw new Error("no add button found");
    await userEvent.click(faceAdd);
    expect(document.querySelector('[data-gender="1"]')).toBeInTheDocument();
  });
});
