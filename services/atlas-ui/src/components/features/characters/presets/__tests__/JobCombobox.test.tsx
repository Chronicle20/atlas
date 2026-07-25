import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { JobCombobox } from "../JobCombobox";

Element.prototype.scrollIntoView ||= () => {};

describe("JobCombobox", () => {
  it("shows the current job's name on the trigger", () => {
    render(<JobCombobox value={100} onChange={vi.fn()} />);
    expect(screen.getByRole("combobox", { name: /class/i })).toHaveTextContent(
      "Warrior",
    );
  });

  it("renders unmapped ids as Job <id>", () => {
    render(<JobCombobox value={4321} onChange={vi.fn()} />);
    expect(screen.getByRole("combobox", { name: /class/i })).toHaveTextContent(
      "Job 4321",
    );
  });

  it("filters by name and picks a job as a number", async () => {
    const onChange = vi.fn();
    render(<JobCombobox value={0} onChange={onChange} />);
    await userEvent.click(screen.getByRole("combobox", { name: /class/i }));
    await userEvent.type(
      screen.getByPlaceholderText(/search by name/i),
      "bishop",
    );
    await userEvent.click(
      await screen.findByRole("option", { name: /bishop/i }),
    );
    expect(onChange).toHaveBeenCalledWith(232);
  });

  it("numeric input matching a curated id filters to it; unmapped ids get a Use-id row", async () => {
    const onChange = vi.fn();
    render(<JobCombobox value={0} onChange={onChange} />);
    await userEvent.click(screen.getByRole("combobox", { name: /class/i }));
    const input = screen.getByPlaceholderText(/search by name/i);
    await userEvent.type(input, "123456");
    await userEvent.click(
      await screen.findByRole("option", { name: /use id 123456/i }),
    );
    expect(onChange).toHaveBeenCalledWith(123456);
  });
});
