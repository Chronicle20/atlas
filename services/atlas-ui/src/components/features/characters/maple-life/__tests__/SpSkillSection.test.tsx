import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

// Radix Select relies on DOM APIs jsdom does not implement.
Element.prototype.hasPointerCapture ||= () => false;
Element.prototype.scrollIntoView ||= () => {};

import type { MapleLifeClassDraft } from "../mapleLifeEditorState";
import { SpSkillSection } from "../SpSkillSection";

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

describe("SpSkillSection", () => {
  it("offers exactly three options", async () => {
    const user = userEvent.setup();
    render(<SpSkillSection draft={buildDraft()} onSetSpSkillId={vi.fn()} />);
    await user.click(screen.getByRole("combobox", { name: /sp skill/i }));
    const options = screen.getAllByRole("option");
    expect(options).toHaveLength(3);
    expect(options.map((o) => o.textContent)).toEqual([
      "None",
      "Improved Max HP Increase (Warrior)",
      "Improved Max MP Increase (Magician)",
    ]);
  });

  it("is disabled at ordinal 2", () => {
    render(
      <SpSkillSection
        draft={buildDraft({ ordinal: 2 })}
        onSetSpSkillId={vi.fn()}
      />,
    );
    expect(screen.getByRole("combobox", { name: /sp skill/i })).toBeDisabled();
  });

  it("is disabled at ordinals 3 and 4", () => {
    const { rerender } = render(
      <SpSkillSection
        draft={buildDraft({ ordinal: 3 })}
        onSetSpSkillId={vi.fn()}
      />,
    );
    expect(screen.getByRole("combobox", { name: /sp skill/i })).toBeDisabled();
    rerender(
      <SpSkillSection
        draft={buildDraft({ ordinal: 4 })}
        onSetSpSkillId={vi.fn()}
      />,
    );
    expect(screen.getByRole("combobox", { name: /sp skill/i })).toBeDisabled();
  });

  it("is enabled at ordinals 0 and 1", () => {
    const { rerender } = render(
      <SpSkillSection
        draft={buildDraft({ ordinal: 0 })}
        onSetSpSkillId={vi.fn()}
      />,
    );
    expect(
      screen.getByRole("combobox", { name: /sp skill/i }),
    ).not.toBeDisabled();
    rerender(
      <SpSkillSection
        draft={buildDraft({ ordinal: 1 })}
        onSetSpSkillId={vi.fn()}
      />,
    );
    expect(
      screen.getByRole("combobox", { name: /sp skill/i }),
    ).not.toBeDisabled();
  });

  it("the disabled control carries an accessible reason", () => {
    render(
      <SpSkillSection
        draft={buildDraft({ ordinal: 2 })}
        onSetSpSkillId={vi.fn()}
      />,
    );
    const select = screen.getByRole("combobox", { name: /sp skill/i });
    const describedById = select.getAttribute("aria-describedby");
    expect(describedById).toBeTruthy();
    const description = document.getElementById(describedById ?? "");
    expect(description).not.toBeNull();
    expect(description).toHaveTextContent(/client skips step 4/i);
  });

  it("a loaded non-zero value at ordinal >= 2 is visible, not hidden", () => {
    render(
      <SpSkillSection
        draft={buildDraft({ ordinal: 2, spSkillId: 1000001 })}
        onSetSpSkillId={vi.fn()}
      />,
    );
    expect(
      screen.getByText("Improved Max HP Increase (Warrior)"),
    ).toBeInTheDocument();
  });

  it("a loaded non-zero value at ordinal >= 2 shows its blocking error", () => {
    render(
      <SpSkillSection
        draft={buildDraft({ ordinal: 2, spSkillId: 1000001 })}
        onSetSpSkillId={vi.fn()}
        errors={{ spSkillId: ["blocked"] }}
      />,
    );
    expect(screen.getByText("blocked")).toBeInTheDocument();
  });

  it("states the prerequisite and the effective cap", () => {
    render(
      <SpSkillSection
        draft={buildDraft({
          ordinal: 0,
          spSkillId: 1000001,
          spBooks: [61, 0, 0, 0, 0, 0, 0, 0, 0, 0],
        })}
        onSetSpSkillId={vi.fn()}
      />,
    );
    expect(screen.getByText(/level-5 prerequisite/i)).toBeInTheDocument();
    expect(screen.getByText(/Effective player cap: 10/)).toBeInTheDocument();
  });

  it("the cap is spBooks[0] - 5 when that is below 10", () => {
    render(
      <SpSkillSection
        draft={buildDraft({
          ordinal: 0,
          spSkillId: 1000001,
          spBooks: [8, 0, 0, 0, 0, 0, 0, 0, 0, 0],
        })}
        onSetSpSkillId={vi.fn()}
      />,
    );
    expect(screen.getByText(/Effective player cap: 3/)).toBeInTheDocument();
  });

  it("an unknown skill id is preserved and warned about", () => {
    render(
      <SpSkillSection
        draft={buildDraft({ ordinal: 0, spSkillId: 9999999 })}
        onSetSpSkillId={vi.fn()}
        warnings={["no prerequisite"]}
      />,
    );
    expect(screen.getByText("Unknown skill 9999999")).toBeInTheDocument();
    expect(screen.getByText("no prerequisite")).toBeInTheDocument();
  });

  it("selecting None clears the value", async () => {
    const user = userEvent.setup();
    const onSetSpSkillId = vi.fn();
    render(
      <SpSkillSection
        draft={buildDraft({ ordinal: 0, spSkillId: 1000001 })}
        onSetSpSkillId={onSetSpSkillId}
      />,
    );
    await user.click(screen.getByRole("combobox", { name: /sp skill/i }));
    await user.click(screen.getByRole("option", { name: "None" }));
    expect(onSetSpSkillId).toHaveBeenCalledWith(undefined);
  });

  it("selecting a skill sets its id", async () => {
    const user = userEvent.setup();
    const onSetSpSkillId = vi.fn();
    render(
      <SpSkillSection
        draft={buildDraft({ ordinal: 0 })}
        onSetSpSkillId={onSetSpSkillId}
      />,
    );
    await user.click(screen.getByRole("combobox", { name: /sp skill/i }));
    await user.click(
      screen.getByRole("option", {
        name: "Improved Max MP Increase (Magician)",
      }),
    );
    expect(onSetSpSkillId).toHaveBeenCalledWith(2000001);
  });
});
