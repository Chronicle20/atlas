import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { PoolItemActions } from "../PoolItemActions";

describe("PoolItemActions", () => {
  it("renders a trigger button with the accessible name 'Open menu'", () => {
    render(<PoolItemActions onEdit={() => {}} onDelete={() => {}} />);
    expect(screen.getByRole("button", { name: "Open menu" })).toBeInTheDocument();
  });

  it("opens the menu and shows Edit and Delete items, calling the right handler for each", async () => {
    const user = userEvent.setup();
    const onEdit = vi.fn();
    const onDelete = vi.fn();
    render(<PoolItemActions onEdit={onEdit} onDelete={onDelete} />);

    await user.click(screen.getByRole("button", { name: "Open menu" }));

    const editItem = await screen.findByText("Edit");
    const deleteItem = await screen.findByText("Delete");
    expect(editItem).toBeInTheDocument();
    expect(deleteItem).toBeInTheDocument();

    await user.click(editItem);
    expect(onEdit).toHaveBeenCalledTimes(1);
    expect(onDelete).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Open menu" }));
    const deleteItemAgain = await screen.findByText("Delete");
    await user.click(deleteItemAgain);
    expect(onDelete).toHaveBeenCalledTimes(1);
  });
});
