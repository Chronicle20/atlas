import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, render, screen } from "@testing-library/react";

import { FreshnessIndicator } from "@/components/features/transports/FreshnessIndicator";

describe("FreshnessIndicator", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-08-06T12:00:00Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("shows the stale/failed message when isError is true, regardless of the other props", () => {
    render(
      <FreshnessIndicator
        dataUpdatedAt={Date.now()}
        isFetching={true}
        isError={true}
      />,
    );

    expect(screen.getByText("Stale — last refresh failed")).toBeInTheDocument();
    expect(screen.queryByText("Loading…")).not.toBeInTheDocument();
    expect(screen.queryByText(/Updated/)).not.toBeInTheDocument();
  });

  it("shows a loading message before the first successful fetch", () => {
    render(
      <FreshnessIndicator
        dataUpdatedAt={0}
        isFetching={true}
        isError={false}
      />,
    );

    expect(screen.getByText("Loading…")).toBeInTheDocument();
    expect(screen.queryByText(/Updated/)).not.toBeInTheDocument();
  });

  it("increments the rendered age as the shared clock ticks", () => {
    const dataUpdatedAt = Date.now() - 5000;

    render(
      <FreshnessIndicator
        dataUpdatedAt={dataUpdatedAt}
        isFetching={false}
        isError={false}
      />,
    );

    expect(screen.getByText("Updated 5s ago")).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(3000);
    });

    expect(screen.getByText("Updated 8s ago")).toBeInTheDocument();
  });

  it("toggles the pulse indicator on isFetching without changing which state renders", () => {
    const dataUpdatedAt = Date.now() - 2000;

    const { container, rerender } = render(
      <FreshnessIndicator
        dataUpdatedAt={dataUpdatedAt}
        isFetching={false}
        isError={false}
      />,
    );

    expect(screen.getByText("Updated 2s ago")).toBeInTheDocument();
    const dot = () => container.querySelector('[aria-hidden="true"]');
    expect(dot()?.className).not.toContain("animate-pulse");

    rerender(
      <FreshnessIndicator
        dataUpdatedAt={dataUpdatedAt}
        isFetching={true}
        isError={false}
      />,
    );

    expect(screen.getByText("Updated 2s ago")).toBeInTheDocument();
    expect(dot()?.className).toContain("animate-pulse");
  });
});
