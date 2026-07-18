import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { TemplateSelector } from "../TemplateSelector";

const templates = [
  { jobIndex: 1, gender: 0 },
  { jobIndex: 1, gender: 1 },
  { jobIndex: 1, gender: 0 },
];

describe("TemplateSelector", () => {
  it("renders tablist segments with derived labels incl. ordinals", () => {
    render(
      <TemplateSelector
        templates={templates}
        selectedIndex={0}
        onSelect={vi.fn()}
        onAdd={vi.fn()}
      />,
    );
    expect(screen.getByRole("tablist")).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Adventurer · M" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    expect(screen.getByRole("tab", { name: "Adventurer · F" })).toHaveAttribute(
      "aria-selected",
      "false",
    );
    expect(
      screen.getByRole("tab", { name: "Adventurer · M (2)" }),
    ).toBeInTheDocument();
  });

  it("clicking a segment selects it; + New adds", async () => {
    const onSelect = vi.fn();
    const onAdd = vi.fn();
    render(
      <TemplateSelector
        templates={templates}
        selectedIndex={0}
        onSelect={onSelect}
        onAdd={onAdd}
      />,
    );
    await userEvent.click(screen.getByRole("tab", { name: "Adventurer · F" }));
    expect(onSelect).toHaveBeenCalledWith(1);
    await userEvent.click(screen.getByRole("button", { name: /new/i }));
    expect(onAdd).toHaveBeenCalled();
  });
});
