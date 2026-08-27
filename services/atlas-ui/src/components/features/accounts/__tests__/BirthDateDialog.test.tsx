// services/atlas-ui/src/components/features/accounts/__tests__/BirthDateDialog.test.tsx
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import type { Account } from "@/types/models/account";

const mutate = vi.fn();
vi.mock("@/lib/hooks/api/useAccounts", () => ({
  useUpdateAccountBirthDate: () => ({ mutate, isPending: false }),
}));

const toastError = vi.fn();
const toastSuccess = vi.fn();
vi.mock("sonner", () => ({
  toast: {
    error: (m: string) => toastError(m),
    success: (m: string) => toastSuccess(m),
  },
}));

import { BirthDateDialog } from "../BirthDateDialog";

const tenant = {
  id: "t1",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
} as never;

const account = (birthDate: number): Account =>
  ({
    id: "42",
    attributes: {
      name: "tester",
      pin: "",
      pic: "",
      pinAttempts: 1,
      picAttempts: 2,
      loggedIn: 0,
      lastLogin: 0,
      gender: 0,
      tos: true,
      language: "en",
      country: "US",
      birthDate,
    },
  }) as Account;

beforeEach(() => {
  mutate.mockReset();
  toastError.mockReset();
  toastSuccess.mockReset();
});

describe("BirthDateDialog", () => {
  it("shows the stored birthday", () => {
    render(<BirthDateDialog account={account(19900102)} tenant={tenant} />);
    expect(screen.getByText("1990-01-02")).toBeInTheDocument();
  });

  it("shows 'Not set' when the account has no birthday", () => {
    render(<BirthDateDialog account={account(0)} tenant={tenant} />);
    expect(screen.getByText("Not set")).toBeInTheDocument();
  });

  it("pre-fills the editor with the current value and saves the packed integer", () => {
    render(<BirthDateDialog account={account(19900102)} tenant={tenant} />);
    fireEvent.click(screen.getByRole("button", { name: /edit birthday/i }));

    const input = screen.getByLabelText("Birthday") as HTMLInputElement;
    expect(input.value).toBe("1990-01-02");

    fireEvent.change(input, { target: { value: "2001-03-04" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    expect(mutate).toHaveBeenCalledTimes(1);
    expect(mutate.mock.calls[0]?.[0]).toMatchObject({ birthDate: 20010304 });
  });

  // The whole account goes to the mutation, not just its id: atlas-account's
  // PATCH is a full-model diff, so the service resends every attribute.
  it("passes the whole account through to the mutation", () => {
    const a = account(19900102);
    render(<BirthDateDialog account={a} tenant={tenant} />);
    fireEvent.click(screen.getByRole("button", { name: /edit birthday/i }));
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    expect(mutate.mock.calls[0]?.[0]).toMatchObject({ account: a, tenant });
  });

  it("refuses an impossible date instead of sending it", () => {
    render(<BirthDateDialog account={account(0)} tenant={tenant} />);
    fireEvent.click(screen.getByRole("button", { name: /edit birthday/i }));

    fireEvent.change(screen.getByLabelText("Birthday"), {
      target: { value: "1990-02-30" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    expect(mutate).not.toHaveBeenCalled();
    expect(toastError).toHaveBeenCalledWith("Please enter a valid date");
  });
});
