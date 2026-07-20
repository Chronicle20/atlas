import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { IdentitySection } from "../IdentitySection";
import { DEFAULT_PRESET_ATTRIBUTES } from "../presetEditorState";

describe("IdentitySection", () => {
  it("edits name and calls onSetField", async () => {
    const onSetField = vi.fn();
    render(
      <IdentitySection
        attrs={{ ...DEFAULT_PRESET_ATTRIBUTES, name: "" }}
        onSetField={onSetField}
        onAddTag={vi.fn()}
        onRemoveTag={vi.fn()}
      />,
    );
    await userEvent.type(screen.getByLabelText(/^name/i), "Hero");
    expect(onSetField).toHaveBeenCalledWith("name", expect.any(String));
  });

  it("adds and removes tags", async () => {
    const onAddTag = vi.fn();
    const onRemoveTag = vi.fn();
    render(
      <IdentitySection
        attrs={{ ...DEFAULT_PRESET_ATTRIBUTES, tags: ["PvP"] }}
        onSetField={vi.fn()}
        onAddTag={onAddTag}
        onRemoveTag={onRemoveTag}
      />,
    );
    await userEvent.click(
      screen.getByRole("button", { name: /remove tag PvP/i }),
    );
    expect(onRemoveTag).toHaveBeenCalledWith("PvP");

    await userEvent.click(screen.getByRole("button", { name: /add tag/i }));
    await userEvent.type(screen.getByPlaceholderText(/tag/i), "Solo");
    await userEvent.click(
      screen.getByRole("button", { name: /^add$/i }),
    );
    expect(onAddTag).toHaveBeenCalledWith("Solo");
  });

  it("edits default name and description", async () => {
    const onSetField = vi.fn();
    render(
      <IdentitySection
        attrs={{ ...DEFAULT_PRESET_ATTRIBUTES, defaultName: "", description: "" }}
        onSetField={onSetField}
        onAddTag={vi.fn()}
        onRemoveTag={vi.fn()}
      />,
    );
    await userEvent.type(
      screen.getByLabelText(/default character name/i),
      "X",
    );
    expect(onSetField).toHaveBeenCalledWith("defaultName", expect.any(String));

    await userEvent.type(screen.getByLabelText(/description/i), "Y");
    expect(onSetField).toHaveBeenCalledWith("description", expect.any(String));
  });

  it("renders actions in the header", () => {
    render(
      <IdentitySection
        attrs={DEFAULT_PRESET_ATTRIBUTES}
        onSetField={vi.fn()}
        onAddTag={vi.fn()}
        onRemoveTag={vi.fn()}
        actions={<button>menu</button>}
      />,
    );
    expect(screen.getByRole("button", { name: /menu/i })).toBeInTheDocument();
  });
});
