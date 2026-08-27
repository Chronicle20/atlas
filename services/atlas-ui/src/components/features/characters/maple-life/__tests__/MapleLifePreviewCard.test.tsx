import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

import { MapleLifePreviewCard } from "../MapleLifePreviewCard";
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

function renderCard(over: Record<string, unknown> = {}) {
  return render(
    <MapleLifePreviewCard
      draft={draft()}
      look={look()}
      picks={picks()}
      {...over}
    />,
  );
}

function getImgSrc(): string {
  const img = screen.getByAltText("Live preview of the selected look");
  return img.getAttribute("src") ?? "";
}

describe("MapleLifePreviewCard", () => {
  it("composes hair as color + 10*floor(style/10)", () => {
    renderCard({
      look: look({ hairs: [30035], hairColors: [2] }),
      picks: picks({ hairIdx: 0, hairColorIdx: 0 }),
    });
    const src = getImgSrc();
    expect(src).toContain("hair=30032");
    expect(src).not.toContain("hair=30037");
  });

  it("passes the look row's gender", () => {
    renderCard({ look: look({ gender: 1 }) });
    expect(getImgSrc()).toContain("gender=1");
  });

  it("reports the combination count", () => {
    renderCard({
      look: look({
        faces: [20000, 20001, 20002],
        hairs: [30030, 30020, 30000],
        hairColors: [0, 7, 3, 2],
        skinColors: [0, 1, 2, 3],
      }),
    });
    expect(screen.getByText(/144 combinations offered/)).toBeInTheDocument();
  });

  it("reports the four factors", () => {
    renderCard({
      look: look({
        faces: [20000, 20001, 20002],
        hairs: [30030, 30020, 30000],
        hairColors: [0, 7, 3, 2],
        skinColors: [0, 1, 2, 3],
      }),
    });
    expect(
      screen.getByText(/3 faces × 3 hairs × 4 hair colours × 4 skin tones/),
    ).toBeInTheDocument();
  });

  it("a zero-combination pool reads zero", () => {
    renderCard({ look: look({ faces: [] }) });
    expect(screen.getByText(/0 combinations offered/)).toBeInTheDocument();
  });

  it("the render image is lazy", () => {
    renderCard();
    const img = screen.getByAltText("Live preview of the selected look");
    expect(img).toHaveAttribute("loading", "lazy");
  });
});
