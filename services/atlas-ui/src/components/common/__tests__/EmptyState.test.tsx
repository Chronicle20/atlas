import { render, screen, fireEvent } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { EmptyState } from "@/components/common/EmptyState";

const TS = 1_735_732_920_000;

describe("EmptyState refresh control", () => {
  it("renders no refresh button when onRefresh is absent", () => {
    render(<EmptyState title="Empty" />);
    expect(screen.queryByTestId("empty-state-refresh")).toBeNull();
    expect(screen.getByTestId("empty-state")).toBeInTheDocument();
  });

  it("renders a Refresh button when onRefresh is supplied", () => {
    const onRefresh = vi.fn();
    render(<EmptyState title="Empty" onRefresh={onRefresh} />);
    const button = screen.getByTestId("empty-state-refresh");
    expect(button).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /refresh/i })).toBe(button);
    fireEvent.click(button);
    expect(onRefresh).toHaveBeenCalledTimes(1);
  });

  it("disables the button and spins the icon while refreshing", () => {
    const onRefresh = vi.fn();
    render(<EmptyState title="Empty" onRefresh={onRefresh} isRefreshing />);
    const button = screen.getByTestId("empty-state-refresh");
    expect(button).toBeDisabled();
    expect(button).toHaveAttribute("aria-busy", "true");
    expect(button.querySelector("svg")).toHaveClass("animate-spin");
  });

  it("does not spin or disable when isRefreshing is false", () => {
    const onRefresh = vi.fn();
    render(
      <EmptyState title="Empty" onRefresh={onRefresh} isRefreshing={false} />,
    );
    const button = screen.getByTestId("empty-state-refresh");
    expect(button).not.toBeDisabled();
    expect(button.querySelector("svg")).not.toHaveClass("animate-spin");
  });

  it("does not invoke onRefresh while disabled", () => {
    const onRefresh = vi.fn();
    render(<EmptyState title="Empty" onRefresh={onRefresh} isRefreshing />);
    const button = screen.getByTestId("empty-state-refresh");
    fireEvent.click(button);
    expect(onRefresh).not.toHaveBeenCalled();
  });
});

describe("EmptyState action precedence", () => {
  it("renders the custom action alone when onRefresh is absent", () => {
    const onClick = vi.fn();
    render(
      <EmptyState
        title="Empty"
        action={{ label: "Create Account", onClick }}
      />,
    );
    const buttons = screen.getAllByRole("button");
    expect(buttons).toHaveLength(1);
    expect(buttons[0]).toHaveAccessibleName("Create Account");
  });

  it("renders both, action first, refresh second", () => {
    const a = vi.fn();
    const r = vi.fn();
    render(
      <EmptyState
        title="Empty"
        action={{ label: "Create Account", onClick: a }}
        onRefresh={r}
      />,
    );
    const buttons = screen.getAllByRole("button");
    expect(buttons).toHaveLength(2);
    expect(buttons[0]).toHaveTextContent("Create Account");
    expect(buttons[1]).toBe(screen.getByTestId("empty-state-refresh"));
  });

  it("keeps refresh on the outline variant with and without a sibling action", () => {
    const r = vi.fn();
    const { unmount } = render(<EmptyState title="Empty" onRefresh={r} />);
    const withoutActionClass = screen.getByTestId(
      "empty-state-refresh",
    ).className;
    expect(withoutActionClass).toContain("border");
    unmount();

    render(
      <EmptyState
        title="Empty"
        action={{ label: "Create Account", onClick: vi.fn() }}
        onRefresh={r}
      />,
    );
    const withActionClass = screen.getByTestId("empty-state-refresh").className;
    expect(withActionClass).toContain("border");
    expect(withActionClass).toBe(withoutActionClass);
  });
});

describe("EmptyState last-updated caption", () => {
  it("renders the caption for a positive timestamp", () => {
    render(<EmptyState title="Empty" lastUpdatedAt={TS} />);
    const caption = screen.getByTestId("empty-state-last-updated");
    expect(caption).toBeInTheDocument();
    expect(caption.textContent).toMatch(/^Last updated/);
    expect(caption).toHaveAttribute("title", new Date(TS).toISOString());
  });

  it("renders no caption for null", () => {
    render(<EmptyState title="Empty" lastUpdatedAt={null} />);
    expect(screen.queryByTestId("empty-state-last-updated")).toBeNull();
  });

  it("renders no caption for zero", () => {
    render(<EmptyState title="Empty" lastUpdatedAt={0} />);
    expect(screen.queryByTestId("empty-state-last-updated")).toBeNull();
  });

  it("renders no caption when absent", () => {
    render(<EmptyState title="Empty" />);
    expect(screen.queryByTestId("empty-state-last-updated")).toBeNull();
  });

  it("advances the caption when the timestamp changes", () => {
    const { rerender } = render(
      <EmptyState title="Empty" lastUpdatedAt={TS} />,
    );
    rerender(<EmptyState title="Empty" lastUpdatedAt={TS + 3_600_000} />);
    expect(screen.getByTestId("empty-state-last-updated")).toHaveAttribute(
      "title",
      new Date(TS + 3_600_000).toISOString(),
    );
  });
});
