import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  getClockSnapshot,
  subscribeToClock,
  useClock,
} from "@/lib/utils/clock";

describe("clock store", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-08-06T12:00:00Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("getClockSnapshot returns a cached value, not a live Date.now() reading", () => {
    // useSyncExternalStore calls getSnapshot repeatedly within the same
    // render pass to detect an unstable source; if the store read a fresh
    // Date.now() on every call it would report a different value each time
    // even though no tick occurred, and React would loop trying to reach a
    // stable snapshot. Stub Date.now to prove the store never touches it
    // between ticks: every call in between must return the exact same
    // cached number.
    const unsubscribe = subscribeToClock(() => {});
    const dateNowSpy = vi
      .spyOn(Date, "now")
      .mockImplementation(
        () => 1_700_000_000_000 + dateNowSpy.mock.calls.length,
      );

    const first = getClockSnapshot();
    const second = getClockSnapshot();
    const third = getClockSnapshot();

    expect(second).toBe(first);
    expect(third).toBe(first);

    dateNowSpy.mockRestore();
    unsubscribe();
  });

  it("advances the snapshot only when the shared interval ticks", () => {
    const unsubscribe = subscribeToClock(() => {});
    const before = getClockSnapshot();

    vi.advanceTimersByTime(1000);

    expect(getClockSnapshot()).toBe(before + 1000);
    unsubscribe();
  });

  it("notifies subscribers once per tick", () => {
    const listener = vi.fn();
    const unsubscribe = subscribeToClock(listener);

    vi.advanceTimersByTime(3000);

    expect(listener).toHaveBeenCalledTimes(3);
    unsubscribe();
  });

  it("starts the interval on the first subscriber and clears it on the last unsubscribe", () => {
    const setIntervalSpy = vi.spyOn(globalThis, "setInterval");
    const clearIntervalSpy = vi.spyOn(globalThis, "clearInterval");

    const unsubscribeA = subscribeToClock(() => {});
    const unsubscribeB = subscribeToClock(() => {});

    expect(setIntervalSpy).toHaveBeenCalledTimes(1);

    unsubscribeA();
    expect(clearIntervalSpy).not.toHaveBeenCalled();

    unsubscribeB();
    expect(clearIntervalSpy).toHaveBeenCalledTimes(1);

    setIntervalSpy.mockRestore();
    clearIntervalSpy.mockRestore();
  });

  it("useClock is exported for components to subscribe with", () => {
    expect(typeof useClock).toBe("function");
  });
});
