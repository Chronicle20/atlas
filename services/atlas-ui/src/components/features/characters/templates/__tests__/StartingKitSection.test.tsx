import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

const useItemNameMock = vi.fn();
vi.mock("@/lib/hooks/api/useItemStrings", () => ({
  useItemName: (...a: unknown[]) => useItemNameMock(...a),
  itemStringKeys: {
    all: ["item-strings"],
    byId: (id: string) => ["item-strings", "name", id],
  },
}));

const useSkillDataMock = vi.fn();
vi.mock("@/lib/hooks/useSkillData", () => ({
  useSkillData: (...a: unknown[]) => useSkillDataMock(...a),
}));

vi.mock("../ItemSearchCombobox", () => ({
  ItemSearchCombobox: ({ onAdd }: { onAdd: (id: number) => void }) => (
    <button type="button" onClick={() => onAdd(2000000)}>
      mock-add-item
    </button>
  ),
}));

import { StartingKitSection } from "../StartingKitSection";

beforeEach(() => {
  useItemNameMock.mockReturnValue({ data: "Red Potion", isError: false });
  useSkillDataMock.mockReturnValue({
    name: "Three Snails",
    iconUrl: "/icon.png",
  });
});

describe("StartingKitSection", () => {
  it("items header shows <n> granted and rows render", () => {
    render(
      <StartingKitSection
        items={[2000000]}
        skills={[]}
        onAddItem={vi.fn()}
        onRemoveItem={vi.fn()}
        onAddSkill={vi.fn()}
        onRemoveSkill={vi.fn()}
      />,
    );
    expect(screen.getByText("1 granted")).toBeInTheDocument();
    expect(screen.getByText("Red Potion")).toBeInTheDocument();
  });

  it("empty skills shows the class-specific empty copy", () => {
    render(
      <StartingKitSection
        items={[]}
        skills={[]}
        onAddItem={vi.fn()}
        onRemoveItem={vi.fn()}
        onAddSkill={vi.fn()}
        onRemoveSkill={vi.fn()}
      />,
    );
    expect(
      screen.getByText(/this class starts with no granted skills/i),
    ).toBeInTheDocument();
  });

  it("skill rows resolve names; numeric add dispatches", async () => {
    const onAddSkill = vi.fn();
    render(
      <StartingKitSection
        items={[]}
        skills={[1000]}
        onAddItem={vi.fn()}
        onRemoveItem={vi.fn()}
        onAddSkill={onAddSkill}
        onRemoveSkill={vi.fn()}
      />,
    );
    expect(screen.getByText("Three Snails")).toBeInTheDocument();
    await userEvent.type(
      screen.getByRole("textbox", { name: /skill id/i }),
      "1001",
    );
    await userEvent.click(screen.getByRole("button", { name: /add skill/i }));
    expect(onAddSkill).toHaveBeenCalledWith(1001);
  });
});
