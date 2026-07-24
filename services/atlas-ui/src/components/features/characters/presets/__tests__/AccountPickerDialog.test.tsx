import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, afterEach } from "vitest";
import { AccountPickerDialog } from "../AccountPickerDialog";

const useAccountSearchMock = vi.fn();
vi.mock("@/lib/hooks/api/useAccounts", () => ({
  useAccountSearch: (...a: unknown[]) => useAccountSearchMock(...a),
}));

const tenant = {
  id: "t1",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
} as never;

describe("AccountPickerDialog", () => {
  it("searches and picks an account (id → number)", async () => {
    useAccountSearchMock.mockReturnValue({
      data: [{ id: "42", attributes: { name: "operator" } }],
      isLoading: false,
      isError: false,
    });
    const onPick = vi.fn();
    render(
      <AccountPickerDialog
        tenant={tenant}
        open
        onOpenChange={vi.fn()}
        onPick={onPick}
      />,
    );
    await userEvent.type(screen.getByRole("searchbox"), "oper");
    await userEvent.click(
      await screen.findByRole("button", { name: /operator/i }),
    );
    expect(onPick).toHaveBeenCalledWith(42);
  });

  it("shows empty state when no results", () => {
    useAccountSearchMock.mockReturnValue({
      data: [],
      isLoading: false,
      isError: false,
    });
    render(
      <AccountPickerDialog
        tenant={tenant}
        open
        onOpenChange={vi.fn()}
        onPick={vi.fn()}
      />,
    );
    expect(screen.getByText(/no accounts/i)).toBeInTheDocument();
  });

  it("shows the loading skeleton while a search is in flight", () => {
    useAccountSearchMock.mockReturnValue({
      data: undefined,
      isLoading: true,
      isError: false,
    });
    render(
      <AccountPickerDialog
        tenant={tenant}
        open
        onOpenChange={vi.fn()}
        onPick={vi.fn()}
      />,
    );
    // Dialog content renders into a portal on document.body, not under the
    // render() container, so query the document for the skeleton rows.
    expect(document.querySelectorAll(".animate-pulse")).toHaveLength(2);
    expect(screen.queryByText(/no accounts/i)).not.toBeInTheDocument();
  });

  describe("debounce wiring", () => {
    afterEach(() => {
      vi.useRealTimers();
    });

    it("passes the debounced pattern to useAccountSearch, not each keystroke", async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true });
      const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
      useAccountSearchMock.mockReturnValue({
        data: [],
        isLoading: false,
        isError: false,
      });

      render(
        <AccountPickerDialog
          tenant={tenant}
          open
          onOpenChange={vi.fn()}
          onPick={vi.fn()}
        />,
      );
      useAccountSearchMock.mockClear();

      await user.type(screen.getByRole("searchbox"), "oper");

      // Before the ~200ms debounce settles, the hook must not yet have
      // seen the fully-typed pattern.
      expect(useAccountSearchMock).not.toHaveBeenCalledWith(tenant, "oper");

      await act(async () => {
        await vi.advanceTimersByTimeAsync(200);
      });

      expect(useAccountSearchMock).toHaveBeenCalledWith(tenant, "oper");
    });
  });
});
