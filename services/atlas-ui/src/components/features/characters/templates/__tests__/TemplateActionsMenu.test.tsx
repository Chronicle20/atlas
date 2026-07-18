import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { TemplateActionsMenu } from "../TemplateActionsMenu";

describe("TemplateActionsMenu", () => {
  it("offers Duplicate and Remove, and nothing else", async () => {
    render(
      <TemplateActionsMenu
        label="Adventurer · M"
        onDuplicate={vi.fn()}
        onRemove={vi.fn()}
      />,
    );
    await userEvent.click(
      screen.getByRole("button", { name: /template actions/i }),
    );
    expect(
      screen.getByRole("menuitem", { name: /duplicate template/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /remove template/i }),
    ).toBeInTheDocument();
    expect(screen.queryByText(/json/i)).not.toBeInTheDocument();
  });

  it("Duplicate fires immediately", async () => {
    const onDuplicate = vi.fn();
    render(
      <TemplateActionsMenu
        label="Adventurer · M"
        onDuplicate={onDuplicate}
        onRemove={vi.fn()}
      />,
    );
    await userEvent.click(
      screen.getByRole("button", { name: /template actions/i }),
    );
    await userEvent.click(
      screen.getByRole("menuitem", { name: /duplicate template/i }),
    );
    expect(onDuplicate).toHaveBeenCalled();
  });

  it("Remove requires confirm and shows the template label", async () => {
    const onRemove = vi.fn();
    render(
      <TemplateActionsMenu
        label="Adventurer · M"
        onDuplicate={vi.fn()}
        onRemove={onRemove}
      />,
    );
    await userEvent.click(
      screen.getByRole("button", { name: /template actions/i }),
    );
    await userEvent.click(
      screen.getByRole("menuitem", { name: /remove template/i }),
    );
    // confirm dialog: not yet removed
    expect(onRemove).not.toHaveBeenCalled();
    expect(screen.getByText(/Adventurer · M/)).toBeInTheDocument();
    expect(
      screen.getByText(/players can no longer create this class\/gender/i),
    ).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: /^remove$/i }));
    expect(onRemove).toHaveBeenCalled();
  });

  it("Remove confirm button uses destructive styling", async () => {
    render(
      <TemplateActionsMenu
        label="Adventurer · M"
        onDuplicate={vi.fn()}
        onRemove={vi.fn()}
      />,
    );
    await userEvent.click(
      screen.getByRole("button", { name: /template actions/i }),
    );
    await userEvent.click(
      screen.getByRole("menuitem", { name: /remove template/i }),
    );
    expect(screen.getByRole("button", { name: /^remove$/i })).toHaveClass(
      "bg-destructive",
      "text-destructive-foreground",
      "hover:bg-destructive/90",
    );
  });
});
