import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, render, screen } from "@testing-library/react";

import { Countdown } from "@/components/features/transports/Countdown";

describe("Countdown", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-08-06T12:00:00Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders mm:ss and ticks down once per second", () => {
    render(<Countdown targetAt="2026-08-06T12:00:30Z" label="departs in" />);

    expect(screen.getByText("0:30")).toBeInTheDocument();
    expect(screen.getByText("departs in")).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(5000);
    });

    expect(screen.getByText("0:25")).toBeInTheDocument();
  });

  it("switches to h:mm:ss past one hour", () => {
    render(<Countdown targetAt="2026-08-06T13:30:05Z" />);
    expect(screen.getByText("1:30:05")).toBeInTheDocument();
  });

  it("clamps at 0:00 and never goes negative", () => {
    render(<Countdown targetAt="2026-08-06T12:00:02Z" />);

    act(() => {
      vi.advanceTimersByTime(10_000);
    });

    expect(screen.getByText("0:00")).toBeInTheDocument();
  });

  it("renders an em dash when there is no target", () => {
    render(<Countdown targetAt={null} label="departs in" />);

    expect(screen.getByText("—")).toBeInTheDocument();
    expect(screen.queryByText("departs in")).not.toBeInTheDocument();
  });
});
