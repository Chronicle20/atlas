import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Pager } from "@/components/common/Pager";

describe("Pager", () => {
  it("renders status text 'Page N of M • T results'", () => {
    render(<Pager page={3} lastPage={25} total={1234} pageSize={50} onPageChange={() => {}} />);
    expect(screen.getByText(/Page 3 of 25/)).toBeInTheDocument();
    expect(screen.getByText(/1234 results/)).toBeInTheDocument();
  });

  it("disables First and Prev on page 1", () => {
    render(<Pager page={1} lastPage={5} total={100} pageSize={20} onPageChange={() => {}} />);
    expect(screen.getByRole("button", { name: /first/i })).toBeDisabled();
    expect(screen.getByRole("button", { name: /previous/i })).toBeDisabled();
    expect(screen.getByRole("button", { name: /next/i })).toBeEnabled();
    expect(screen.getByRole("button", { name: /last/i })).toBeEnabled();
  });

  it("disables Next and Last on the last page", () => {
    render(<Pager page={5} lastPage={5} total={100} pageSize={20} onPageChange={() => {}} />);
    expect(screen.getByRole("button", { name: /next/i })).toBeDisabled();
    expect(screen.getByRole("button", { name: /last/i })).toBeDisabled();
    expect(screen.getByRole("button", { name: /first/i })).toBeEnabled();
    expect(screen.getByRole("button", { name: /previous/i })).toBeEnabled();
  });

  it("disables all four boundary buttons on a single-page result", () => {
    render(<Pager page={1} lastPage={1} total={5} pageSize={50} onPageChange={() => {}} />);
    expect(screen.getByRole("button", { name: /first/i })).toBeDisabled();
    expect(screen.getByRole("button", { name: /previous/i })).toBeDisabled();
    expect(screen.getByRole("button", { name: /next/i })).toBeDisabled();
    expect(screen.getByRole("button", { name: /last/i })).toBeDisabled();
  });

  it("renders current ± 2 numbered window", () => {
    render(<Pager page={7} lastPage={25} total={1234} pageSize={50} onPageChange={() => {}} />);
    for (const n of [5, 6, 7, 8, 9]) {
      expect(screen.getByRole("button", { name: String(n) })).toBeInTheDocument();
    }
    expect(screen.queryByRole("button", { name: "4" })).toBeNull();
    expect(screen.queryByRole("button", { name: "10" })).toBeNull();
  });

  it("clips the window at boundaries", () => {
    render(<Pager page={1} lastPage={5} total={100} pageSize={20} onPageChange={() => {}} />);
    for (const n of [1, 2, 3]) {
      expect(screen.getByRole("button", { name: String(n) })).toBeInTheDocument();
    }
  });

  it("calls onPageChange with the clicked page", async () => {
    const fn = vi.fn();
    render(<Pager page={3} lastPage={10} total={200} pageSize={20} onPageChange={fn} />);
    await userEvent.click(screen.getByRole("button", { name: "5" }));
    expect(fn).toHaveBeenCalledWith(5);

    await userEvent.click(screen.getByRole("button", { name: /next/i }));
    expect(fn).toHaveBeenCalledWith(4);

    await userEvent.click(screen.getByRole("button", { name: /last/i }));
    expect(fn).toHaveBeenCalledWith(10);
  });

  it("formats zero-result state as 'No results'", () => {
    render(<Pager page={1} lastPage={1} total={0} pageSize={50} onPageChange={() => {}} />);
    expect(screen.getByText(/no results/i)).toBeInTheDocument();
  });
});
