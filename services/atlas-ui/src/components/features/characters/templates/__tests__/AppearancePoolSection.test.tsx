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

import { AppearancePoolSection } from "../AppearancePoolSection";

const POOL = [20000, 21000];

function renderSection(over: Record<string, unknown> = {}) {
  return render(
    <AppearancePoolSection
      dimension="faces"
      title="Faces"
      pool={POOL}
      selectedIndex={0}
      variantLoadout={(_dim, id) => ({
        skin: 0,
        hair: 30030,
        face: id,
        equipment: {},
        gender: 0,
      })}
      onPick={vi.fn()}
      onRemoveEntry={vi.fn()}
      renderAddDialog={() => null}
      {...over}
    />,
  );
}

describe("AppearancePoolSection", () => {
  it("renders one thumb per pool entry with id captions", () => {
    renderSection();
    expect(screen.getByText("20000")).toBeInTheDocument();
    expect(screen.getByText("21000")).toBeInTheDocument();
  });

  it("clicking a thumb sets the preview pick (UI-only)", async () => {
    const onPick = vi.fn();
    renderSection({ onPick });
    await userEvent.click(
      screen.getByRole("button", { name: /preview face 21000/i }),
    );
    expect(onPick).toHaveBeenCalledWith(1);
  });

  it("the picked thumb is marked pressed", () => {
    renderSection({ selectedIndex: 1 });
    expect(
      screen.getByRole("button", { name: /preview face 21000/i }),
    ).toHaveAttribute("aria-pressed", "true");
  });

  it("each thumb has a remove affordance", async () => {
    const onRemoveEntry = vi.fn();
    renderSection({ onRemoveEntry });
    await userEvent.click(
      screen.getByRole("button", { name: /remove face 20000/i }),
    );
    expect(onRemoveEntry).toHaveBeenCalledWith(0);
  });

  it("empty pool shows the non-blocking factory warning", () => {
    renderSection({ pool: [] });
    expect(
      screen.getByText(
        /character creation will fail while this pool is empty/i,
      ),
    ).toBeInTheDocument();
  });

  it("renders the description when supplied", () => {
    renderSection({ description: <span>Faces are full item ids</span> });
    expect(screen.getByText("Faces are full item ids")).toBeInTheDocument();
  });
});
