import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { LoginHistoryPage } from "@/pages/LoginHistoryPage";

const activeTenant: { current: { id: string } | null } = {
  current: { id: "aaa" },
};
const searchMock = vi.fn();

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: activeTenant.current }),
}));

vi.mock("@/services/api/login-history.service", () => ({
  loginHistoryService: { search: (...args: unknown[]) => searchMock(...args) },
}));

vi.mock("sonner", () => ({
  Toaster: () => null,
  toast: { error: vi.fn(), info: vi.fn(), success: vi.fn() },
}));

vi.mock("@/components/features/bans/CreateBanDialog", () => ({
  CreateBanDialog: () => null,
}));

describe("LoginHistoryPage tenant switching", () => {
  beforeEach(() => {
    searchMock.mockReset();
    activeTenant.current = { id: "aaa" };
  });

  it("clears prior-tenant results when the active tenant changes", async () => {
    searchMock.mockResolvedValueOnce([
      {
        id: "1",
        attributes: {
          accountId: 1,
          accountName: "Alice",
          ipAddress: "1.1.1.1",
          hwid: "hw1",
          success: true,
          failureReason: "",
        },
      },
    ]);

    const { rerender } = render(<LoginHistoryPage />);

    fireEvent.change(screen.getByLabelText("IP Address"), {
      target: { value: "1.1.1.1" },
    });
    fireEvent.click(screen.getByRole("button", { name: /search/i }));

    await waitFor(() => {
      expect(screen.getByText("Results")).toBeInTheDocument();
    });
    expect(searchMock).toHaveBeenCalledTimes(1);

    // Switch tenant — the reset effect must drop the previous tenant's rows.
    activeTenant.current = { id: "bbb" };
    rerender(<LoginHistoryPage />);

    await waitFor(() => {
      expect(screen.queryByText("Results")).not.toBeInTheDocument();
    });
  });
});
