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

import { normalizeTemplate, DEFAULT_PICKS } from "../editorState";
import { AppearancePoolSection } from "../AppearancePoolSection";

const tpl = normalizeTemplate({ faces: [20000, 21000] });

function renderSection(over: Record<string, unknown> = {}) {
  return render(
    <AppearancePoolSection
      dimension="faces"
      title="Faces"
      template={tpl}
      picks={DEFAULT_PICKS}
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
    expect(onPick).toHaveBeenCalledWith("faceIdx", 1);
  });

  it("the picked thumb is marked pressed", () => {
    renderSection({ picks: { ...DEFAULT_PICKS, faceIdx: 1 } });
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
    renderSection({ template: normalizeTemplate({}) });
    expect(
      screen.getByText(
        /character creation will fail while this pool is empty/i,
      ),
    ).toBeInTheDocument();
  });
});
