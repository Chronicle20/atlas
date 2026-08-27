import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

// Radix Select relies on DOM APIs jsdom does not implement.
Element.prototype.hasPointerCapture ||= () => false;
Element.prototype.scrollIntoView ||= () => {};

const useMapMock = vi.fn();
vi.mock("@/lib/hooks/api/useMaps", () => ({
  useMap: (...a: unknown[]) => useMapMock(...a),
  useMapsByName: () => ({ data: [], isLoading: false }),
}));

import type { MapleLifeClassDraft } from "../mapleLifeEditorState";
import { IdentitySection } from "../IdentitySection";

beforeEach(() => {
  useMapMock.mockReturnValue({ data: undefined, isError: false });
});

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

const readyJobs = {
  options: [{ id: 100, name: "Warrior" }],
  isPending: false,
  isError: false,
};

describe("IdentitySection", () => {
  it("the job field stays editable for ordinal 2", () => {
    render(
      <IdentitySection
        draft={buildDraft({ ordinal: 2 })}
        jobs={readyJobs}
        onSetIdentity={vi.fn()}
        onSetLevel={vi.fn()}
      />,
    );
    expect(screen.getByRole("combobox", { name: /job/i })).not.toBeDisabled();
    expect(screen.queryByRole("note")).not.toBeInTheDocument();
  });

  it("reports pending job names distinctly from an empty list", () => {
    render(
      <IdentitySection
        draft={buildDraft()}
        jobs={{ options: [], isPending: true, isError: false }}
        onSetIdentity={vi.fn()}
        onSetLevel={vi.fn()}
      />,
    );
    expect(screen.getByText(/loading job names/i)).toBeInTheDocument();
    expect(screen.getByRole("combobox", { name: /job/i })).toBeDisabled();
  });

  it("reports a job-name error distinctly", () => {
    render(
      <IdentitySection
        draft={buildDraft()}
        jobs={{ options: [], isPending: false, isError: true }}
        onSetIdentity={vi.fn()}
        onSetLevel={vi.fn()}
      />,
    );
    expect(screen.getByText(/job names unavailable/i)).toBeInTheDocument();
  });

  it("level is bounded 1..200", () => {
    render(
      <IdentitySection
        draft={buildDraft()}
        jobs={readyJobs}
        onSetIdentity={vi.fn()}
        onSetLevel={vi.fn()}
      />,
    );
    const level = screen.getByRole("spinbutton", { name: /level/i });
    expect(level).toHaveAttribute("min", "1");
    expect(level).toHaveAttribute("max", "200");
  });

  it("a loaded out-of-range level is displayed as loaded", () => {
    render(
      <IdentitySection
        draft={buildDraft({ level: 240 })}
        jobs={readyJobs}
        onSetIdentity={vi.fn()}
        onSetLevel={vi.fn()}
      />,
    );
    expect(screen.getByRole("spinbutton", { name: /level/i })).toHaveValue(240);
  });

  it("editing the level reports through onSetLevel", async () => {
    const user = userEvent.setup();
    const onSetLevel = vi.fn();
    render(
      <IdentitySection
        draft={buildDraft()}
        jobs={readyJobs}
        onSetIdentity={vi.fn()}
        onSetLevel={onSetLevel}
      />,
    );
    const level = screen.getByRole("spinbutton", { name: /level/i });
    await user.clear(level);
    await user.type(level, "35");
    expect(onSetLevel).toHaveBeenLastCalledWith(35);
  });

  it("renders field errors", () => {
    render(
      <IdentitySection
        draft={buildDraft()}
        jobs={readyJobs}
        onSetIdentity={vi.fn()}
        onSetLevel={vi.fn()}
        errors={{ jobId: ["bad job"] }}
      />,
    );
    expect(screen.getByText("bad job")).toBeInTheDocument();
  });
});
