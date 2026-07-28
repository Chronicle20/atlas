import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";

// Radix Select relies on DOM APIs jsdom does not implement.
Element.prototype.hasPointerCapture ||= () => false;
Element.prototype.scrollIntoView ||= () => {};

const useMapMock = vi.fn();
vi.mock("@/lib/hooks/api/useMaps", () => ({
  useMap: (...a: unknown[]) => useMapMock(...a),
  useMapsByName: () => ({ data: [], isLoading: false }),
}));

import { blankTemplate, normalizeTemplate } from "../editorState";
import { IdentitySection } from "../IdentitySection";

beforeEach(() => {
  useMapMock.mockReturnValue({ data: undefined, isError: false });
});

describe("IdentitySection", () => {
  it("selecting a known class sets jobIndex and subJobIndex", async () => {
    const onSetIdentity = vi.fn();
    render(
      <IdentitySection
        template={blankTemplate()}
        onSetIdentity={onSetIdentity}
      />,
    );
    await userEvent.click(screen.getByRole("combobox", { name: /class/i }));
    await userEvent.click(
      await screen.findByRole("option", { name: /Aran \(2\.0\)/ }),
    );
    expect(onSetIdentity).toHaveBeenCalledWith("jobIndex", 2);
    expect(onSetIdentity).toHaveBeenCalledWith("subJobIndex", 0);
  });

  it("Advanced mode accepts arbitrary numeric job/subJob (backend validates)", async () => {
    const onSetIdentity = vi.fn();
    render(
      <IdentitySection
        template={blankTemplate()}
        onSetIdentity={onSetIdentity}
      />,
    );
    await userEvent.click(screen.getByRole("button", { name: /advanced/i }));
    // "Sub job index" also matches /job index/i (substring), so this must
    // disambiguate: "Job index" renders first in the DOM.
    const jobInput = screen.getAllByLabelText(/job index/i)[0]!;
    await userEvent.clear(jobInput);
    await userEvent.type(jobInput, "1");
    const subInput = screen.getByLabelText(/sub job index/i);
    await userEvent.clear(subInput);
    await userEvent.type(subInput, "1"); // Dual Blade 1.1 — permitted
    expect(onSetIdentity).toHaveBeenCalledWith("jobIndex", 1);
    expect(onSetIdentity).toHaveBeenCalledWith("subJobIndex", 1);
  });

  it("gender select maps Male/Female to 0/1", async () => {
    const onSetIdentity = vi.fn();
    render(
      <IdentitySection
        template={blankTemplate()}
        onSetIdentity={onSetIdentity}
      />,
    );
    await userEvent.click(screen.getByRole("combobox", { name: /gender/i }));
    await userEvent.click(
      await screen.findByRole("option", { name: /female/i }),
    );
    expect(onSetIdentity).toHaveBeenCalledWith("gender", 1);
  });

  it("unknown class combos display in the closed class control", () => {
    render(
      <IdentitySection
        template={normalizeTemplate({ jobIndex: 1, subJobIndex: 1 })}
        onSetIdentity={vi.fn()}
      />,
    );
    expect(screen.getByText(/Adventurer \(1\.1\)/)).toBeInTheDocument();
  });

  it("renders the actions slot in the header", () => {
    render(
      <IdentitySection
        template={blankTemplate()}
        onSetIdentity={vi.fn()}
        actions={<button type="button">kebab-here</button>}
      />,
    );
    expect(
      screen.getByRole("button", { name: "kebab-here" }),
    ).toBeInTheDocument();
  });
});
