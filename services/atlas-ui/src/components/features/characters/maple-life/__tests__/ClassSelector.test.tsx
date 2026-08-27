import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { MapleLifeConfig } from "@/types/models/template";
import {
  initialMapleLifeState,
  mapleLifeReducer,
} from "../mapleLifeEditorState";
import { ClassSelector } from "../ClassSelector";

// Synthetic fixture (not shipped seed data): the minimum needed to exercise
// a present row at (0,0) and an unconfirmed present row at (2,0), leaving
// the other eight slots absent.
const SEED: MapleLifeConfig = {
  looks: [],
  classes: [
    {
      ordinal: 0,
      gender: 0,
      jobId: 100,
      level: 30,
      mapId: 102000000,
      stats: { str: 4, dex: 4, int: 4, luk: 4, hp: 50, mp: 50 },
      ap: 0,
      sp: "0,0,0,0,0,0,0,0,0,0",
      meso: 0,
      equipment: [],
      inventory: [],
    },
    {
      ordinal: 2,
      gender: 0,
      jobId: 200,
      level: 30,
      mapId: 102000000,
      stats: { str: 4, dex: 4, int: 4, luk: 4, hp: 50, mp: 50 },
      ap: 0,
      sp: "0,0,0,0,0,0,0,0,0,0",
      meso: 0,
      equipment: [],
      inventory: [],
    },
  ],
};

function buildDrafts() {
  const state = mapleLifeReducer(initialMapleLifeState(), {
    type: "load",
    config: SEED,
  });
  return state.drafts;
}

describe("ClassSelector", () => {
  it("renders two tablists", () => {
    render(
      <ClassSelector
        ordinal={0}
        gender={0}
        drafts={buildDrafts()}
        jobNameById={null}
        onSelect={vi.fn()}
      />,
    );
    const tablists = screen.getAllByRole("tablist");
    expect(tablists).toHaveLength(2);
    expect(tablists.map((t) => t.getAttribute("aria-label"))).toEqual([
      "Gender",
      "Class ordinal",
    ]);
  });

  it("renders five ordinal tabs and two gender tabs", () => {
    render(
      <ClassSelector
        ordinal={0}
        gender={0}
        drafts={buildDrafts()}
        jobNameById={null}
        onSelect={vi.fn()}
      />,
    );
    const genderList = screen.getByRole("tablist", { name: "Gender" });
    const ordinalList = screen.getByRole("tablist", { name: "Class ordinal" });
    expect(within(ordinalList).getAllByRole("tab")).toHaveLength(5);
    expect(within(genderList).getAllByRole("tab")).toHaveLength(2);
  });

  it("labels an ordinal with its job name", () => {
    render(
      <ClassSelector
        ordinal={0}
        gender={0}
        drafts={buildDrafts()}
        jobNameById={new Map([[100, "Warrior"]])}
        onSelect={vi.fn()}
      />,
    );
    expect(
      screen.getByRole("tab", { name: /0 · Warrior/ }),
    ).toBeInTheDocument();
  });

  it("falls back to the raw jobId when job names are unknown", () => {
    render(
      <ClassSelector
        ordinal={0}
        gender={0}
        drafts={buildDrafts()}
        jobNameById={null}
        onSelect={vi.fn()}
      />,
    );
    expect(screen.getByRole("tab", { name: /0 · 100/ })).toBeInTheDocument();
  });

  it("badges ordinals 0 and 1 as derived", () => {
    render(
      <ClassSelector
        ordinal={0}
        gender={0}
        drafts={buildDrafts()}
        jobNameById={null}
        onSelect={vi.fn()}
      />,
    );
    const ordinalList = screen.getByRole("tablist", { name: "Class ordinal" });
    const tabs = within(ordinalList).getAllByRole("tab");
    expect(within(tabs[0]!).getByText("derived")).toBeInTheDocument();
    expect(within(tabs[1]!).getByText("derived")).toBeInTheDocument();
  });

  it("badges ordinals 2, 3 and 4 as unconfirmed", () => {
    render(
      <ClassSelector
        ordinal={0}
        gender={0}
        drafts={buildDrafts()}
        jobNameById={null}
        onSelect={vi.fn()}
      />,
    );
    const ordinalList = screen.getByRole("tablist", { name: "Class ordinal" });
    const tabs = within(ordinalList).getAllByRole("tab");
    expect(within(tabs[2]!).getByText("unconfirmed")).toBeInTheDocument();
    expect(within(tabs[3]!).getByText("unconfirmed")).toBeInTheDocument();
    expect(within(tabs[4]!).getByText("unconfirmed")).toBeInTheDocument();
  });

  it("marks an absent row", () => {
    render(
      <ClassSelector
        ordinal={0}
        gender={0}
        drafts={buildDrafts()}
        jobNameById={null}
        onSelect={vi.fn()}
      />,
    );
    const ordinalList = screen.getByRole("tablist", { name: "Class ordinal" });
    const tabs = within(ordinalList).getAllByRole("tab");
    expect(within(tabs[1]!).getByText("not configured")).toBeInTheDocument();
    expect(
      within(tabs[0]!).queryByText("not configured"),
    ).not.toBeInTheDocument();
  });

  it("only the selected tab is in the tab order", () => {
    render(
      <ClassSelector
        ordinal={2}
        gender={0}
        drafts={buildDrafts()}
        jobNameById={null}
        onSelect={vi.fn()}
      />,
    );
    const ordinalList = screen.getByRole("tablist", { name: "Class ordinal" });
    const tabs = within(ordinalList).getAllByRole("tab");
    tabs.forEach((tab, index) => {
      expect(tab).toHaveAttribute("tabIndex", index === 2 ? "0" : "-1");
    });
  });

  it("ArrowRight moves to the next ordinal", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    render(
      <ClassSelector
        ordinal={0}
        gender={0}
        drafts={buildDrafts()}
        jobNameById={null}
        onSelect={onSelect}
      />,
    );
    const ordinalList = screen.getByRole("tablist", { name: "Class ordinal" });
    const tabs = within(ordinalList).getAllByRole("tab");
    tabs[0]!.focus();
    await user.keyboard("{ArrowRight}");
    expect(onSelect).toHaveBeenCalledWith(1, 0);
  });

  it("ArrowLeft wraps from the first to the last", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    render(
      <ClassSelector
        ordinal={0}
        gender={0}
        drafts={buildDrafts()}
        jobNameById={null}
        onSelect={onSelect}
      />,
    );
    const ordinalList = screen.getByRole("tablist", { name: "Class ordinal" });
    const tabs = within(ordinalList).getAllByRole("tab");
    tabs[0]!.focus();
    await user.keyboard("{ArrowLeft}");
    expect(onSelect).toHaveBeenCalledWith(4, 0);
  });

  it("Home and End jump to the ends", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    render(
      <ClassSelector
        ordinal={2}
        gender={0}
        drafts={buildDrafts()}
        jobNameById={null}
        onSelect={onSelect}
      />,
    );
    const ordinalList = screen.getByRole("tablist", { name: "Class ordinal" });
    const tabs = within(ordinalList).getAllByRole("tab");
    tabs[2]!.focus();
    await user.keyboard("{Home}");
    expect(onSelect).toHaveBeenCalledWith(0, 0);
    tabs[2]!.focus();
    await user.keyboard("{End}");
    expect(onSelect).toHaveBeenCalledWith(4, 0);
  });

  it("the gender tablist has its own roving tabindex", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    render(
      <ClassSelector
        ordinal={3}
        gender={0}
        drafts={buildDrafts()}
        jobNameById={null}
        onSelect={onSelect}
      />,
    );
    const genderList = screen.getByRole("tablist", { name: "Gender" });
    const tabs = within(genderList).getAllByRole("tab");
    tabs[0]!.focus();
    await user.keyboard("{ArrowRight}");
    expect(onSelect).toHaveBeenCalledWith(3, 1);
  });

  it("clicking a tab selects it", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    render(
      <ClassSelector
        ordinal={3}
        gender={0}
        drafts={buildDrafts()}
        jobNameById={null}
        onSelect={onSelect}
      />,
    );
    await user.click(screen.getByRole("tab", { name: "Female" }));
    expect(onSelect).toHaveBeenCalledWith(3, 1);
  });
});
