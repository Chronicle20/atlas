import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter, Route, Routes, useSearchParams } from "react-router-dom";

// Radix AlertDialog/DropdownMenu rely on DOM APIs jsdom does not implement.
Element.prototype.hasPointerCapture ||= () => false;
Element.prototype.scrollIntoView ||= () => {};

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

// Sections with their own data needs are exercised in their own suites; stub
// them here so the editor test focuses on assembly, URL sync, and save flow.
vi.mock("../IdentitySection", () => ({
  IdentitySection: ({ actions }: { actions?: React.ReactNode }) => (
    <div data-testid="identity">{actions}</div>
  ),
}));
vi.mock("../AppearancePoolSection", () => ({
  AppearancePoolSection: () => <div data-testid="appearance-pool" />,
}));
vi.mock("../EquipmentPoolSection", () => ({
  EquipmentPoolSection: () => <div data-testid="equipment-pool" />,
}));
vi.mock("../StartingKitSection", () => ({
  StartingKitSection: () => <div data-testid="starting-kit" />,
}));
vi.mock("../PreviewCard", () => ({
  PreviewCard: () => <div data-testid="preview-card" />,
}));

import { normalizeTemplate } from "../editorState";
import {
  CharacterTemplatesEditor,
  type TemplatesEditorAdapter,
} from "../CharacterTemplatesEditor";

function TplProbe() {
  const [params] = useSearchParams();
  return <output data-testid="tpl-param">{params.get("tpl") ?? ""}</output>;
}

function renderEditor(
  adapter: Partial<TemplatesEditorAdapter> = {},
  initialEntry = "/edit",
) {
  const full: TemplatesEditorAdapter = {
    templates: [
      normalizeTemplate({ jobIndex: 1, gender: 0 }),
      normalizeTemplate({ jobIndex: 1, gender: 1 }),
    ],
    isLoading: false,
    error: null,
    save: vi.fn(),
    isSaving: false,
    ...adapter,
  };
  render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route
          path="/edit"
          element={
            <>
              <CharacterTemplatesEditor adapter={full} />
              <TplProbe />
            </>
          }
        />
      </Routes>
    </MemoryRouter>,
  );
  return full;
}

beforeEach(() => vi.clearAllMocks());

describe("CharacterTemplatesEditor", () => {
  it("shows skeleton while loading and ErrorDisplay on error (both contexts identical)", () => {
    renderEditor({ templates: undefined, isLoading: true });
    expect(screen.getByTestId("form-skeleton")).toBeInTheDocument();
  });

  it("renders ErrorDisplay for load errors", () => {
    renderEditor({
      templates: undefined,
      isLoading: false,
      error: new Error("boom"),
    });
    expect(screen.getByTestId("error-display")).toBeInTheDocument();
  });

  it("empty configuration shows the explanatory empty state with Add", async () => {
    renderEditor({ templates: [] });
    expect(screen.getByTestId("empty-state")).toBeInTheDocument();
    await userEvent.click(
      screen.getByRole("button", { name: /add template/i }),
    );
    expect(screen.getByRole("tablist")).toBeInTheDocument();
  });

  it("restores selection from ?tpl= deep link", () => {
    renderEditor({}, "/edit?tpl=1");
    expect(screen.getByRole("tab", { name: "Adventurer · F" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
  });

  it("clamps out-of-range ?tpl= to 0 and writes it back", async () => {
    renderEditor({}, "/edit?tpl=99");
    expect(screen.getByRole("tab", { name: "Adventurer · M" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    await waitFor(() =>
      expect(screen.getByTestId("tpl-param")).toHaveTextContent("0"),
    );
  });

  it("selecting a template syncs ?tpl=", async () => {
    renderEditor();
    await userEvent.click(screen.getByRole("tab", { name: "Adventurer · F" }));
    await waitFor(() =>
      expect(screen.getByTestId("tpl-param")).toHaveTextContent("1"),
    );
  });

  it("save passes the working array; success resets the dirty bar", async () => {
    const save = vi.fn((_tpls: unknown, onSuccess: () => void) => onSuccess());
    renderEditor({ save });
    // + New makes it dirty
    await userEvent.click(screen.getByRole("button", { name: /new/i }));
    expect(screen.getByText(/unsaved changes/i)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: /^save$/i }));
    expect(save).toHaveBeenCalled();
    expect((save.mock.calls[0]![0] as unknown[]).length).toBe(3);
    expect(screen.getByText(/no unsaved changes/i)).toBeInTheDocument();
  });

  it("discard reverts to baseline after confirm", async () => {
    renderEditor();
    await userEvent.click(screen.getByRole("button", { name: /new/i }));
    expect(screen.getAllByRole("tab")).toHaveLength(3);
    await userEvent.click(screen.getByRole("button", { name: /discard/i }));
    await userEvent.click(
      screen.getByRole("button", { name: /discard changes/i }),
    );
    expect(screen.getAllByRole("tab")).toHaveLength(2);
  });

  it("discard restores the nearest valid tab (reducer selection), not tab 0", async () => {
    renderEditor();
    // Select the last baseline template, then + New (selection -> new last
    // index, dirty). The added template's URL param is the pre-discard index.
    await userEvent.click(screen.getByRole("tab", { name: "Adventurer · F" }));
    await userEvent.click(screen.getByRole("button", { name: /new/i }));
    expect(screen.getAllByRole("tab")).toHaveLength(3);
    await userEvent.click(screen.getByRole("button", { name: /discard/i }));
    await userEvent.click(
      screen.getByRole("button", { name: /discard changes/i }),
    );
    const tabs = screen.getAllByRole("tab");
    expect(tabs).toHaveLength(2);
    // Reducer clamps to baseline.length-1 (the last real tab), NOT tab 0, and
    // the URL param must agree with that selection.
    expect(screen.getByRole("tab", { name: "Adventurer · F" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    await waitFor(() =>
      expect(screen.getByTestId("tpl-param")).toHaveTextContent("1"),
    );
  });
});
