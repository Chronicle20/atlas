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

  it("roving tabindex: only the selected tab is a tab stop", () => {
    render(
      <TemplateSelector
        templates={templates}
        selectedIndex={1}
        onSelect={vi.fn()}
        onAdd={vi.fn()}
      />,
    );
    const tabs = screen.getAllByRole("tab");
    tabs.forEach((tab, index) => {
      expect(tab).toHaveAttribute("tabindex", index === 1 ? "0" : "-1");
    });
  });

  it("ArrowRight selects and focuses the next tab, wrapping from the last to the first", async () => {
    const onSelect = vi.fn();
    render(
      <TemplateSelector
        templates={templates}
        selectedIndex={2}
        onSelect={onSelect}
        onAdd={vi.fn()}
      />,
    );
    const tabs = screen.getAllByRole("tab");
    tabs[2]!.focus();
    await userEvent.keyboard("{ArrowRight}");
    expect(onSelect).toHaveBeenCalledWith(0);
    expect(tabs[0]).toHaveFocus();
  });

  it("Home/End jump to the first/last tab", async () => {
    const onSelect = vi.fn();
    render(
      <TemplateSelector
        templates={templates}
        selectedIndex={1}
        onSelect={onSelect}
        onAdd={vi.fn()}
      />,
    );
    const tabs = screen.getAllByRole("tab");
    tabs[1]!.focus();
    await userEvent.keyboard("{End}");
    expect(onSelect).toHaveBeenCalledWith(2);
    expect(tabs[2]).toHaveFocus();

    onSelect.mockClear();
    tabs[1]!.focus();
    await userEvent.keyboard("{Home}");
    expect(onSelect).toHaveBeenCalledWith(0);
    expect(tabs[0]).toHaveFocus();
  });
});
