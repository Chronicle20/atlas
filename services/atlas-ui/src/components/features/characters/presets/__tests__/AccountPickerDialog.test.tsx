import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { AccountPickerDialog } from "../AccountPickerDialog";

const useAccountSearchMock = vi.fn();
vi.mock("@/lib/hooks/api/useAccounts", () => ({
  useAccountSearch: (...a: unknown[]) => useAccountSearchMock(...a),
}));

const tenant = { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } as never;

describe("AccountPickerDialog", () => {
  it("searches and picks an account (id → number)", async () => {
    useAccountSearchMock.mockReturnValue({
      data: [{ id: "42", attributes: { name: "operator" } }],
      isLoading: false, isError: false,
    });
    const onPick = vi.fn();
    render(<AccountPickerDialog tenant={tenant} open onOpenChange={vi.fn()} onPick={onPick} />);
    await userEvent.type(screen.getByRole("searchbox"), "oper");
    await userEvent.click(await screen.findByRole("button", { name: /operator/i }));
    expect(onPick).toHaveBeenCalledWith(42);
  });

  it("shows empty state when no results", () => {
    useAccountSearchMock.mockReturnValue({ data: [], isLoading: false, isError: false });
    render(<AccountPickerDialog tenant={tenant} open onOpenChange={vi.fn()} onPick={vi.fn()} />);
    expect(screen.getByText(/no accounts/i)).toBeInTheDocument();
  });
});
