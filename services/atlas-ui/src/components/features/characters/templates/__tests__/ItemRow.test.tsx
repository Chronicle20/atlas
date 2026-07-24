import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { ItemRow } from "../ItemRow";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: null }),
}));
vi.mock("@/lib/hooks/api/useItemStrings", () => ({
  useItemName: () => ({ data: "Test Item", isError: false }),
}));

describe("ItemRow", () => {
  it("renders trailing content when provided", () => {
    render(
      <ItemRow
        id={1040002}
        onRemove={() => {}}
        removeAriaLabel="Remove"
        trailing={<span data-testid="trailing">avg</span>}
      />,
    );
    expect(screen.getByTestId("trailing")).toBeInTheDocument();
  });

  it("omits trailing region cleanly when not provided", () => {
    render(
      <ItemRow id={1040002} onRemove={() => {}} removeAriaLabel="Remove" />,
    );
    expect(screen.queryByTestId("trailing")).toBeNull();
    expect(screen.getByText("1040002")).toBeInTheDocument();
  });
});
