import { useSyncExternalStore } from "react";

/**
 * A single 1-second clock shared by every countdown on the page.
 *
 * Countdowns must tick between the 30-second polls without dragging a table
 * re-render along with them. One module-level interval plus
 * useSyncExternalStore gives that: only the leaf components that subscribe
 * re-render on a tick. A context provider would re-render every consumer's
 * subtree; a timer per cell would multiply intervals.
 *
 * The interval starts on the first subscriber and is cleared on the last, so
 * a page with no countdowns runs no timer.
 */

const TICK_MS = 1000;

let listeners: Array<() => void> = [];
let intervalId: ReturnType<typeof setInterval> | null = null;
let snapshot = Date.now();

function tick(): void {
  snapshot = Date.now();
  for (const listener of listeners) {
    listener();
  }
}

export function subscribeToClock(listener: () => void): () => void {
  listeners = [...listeners, listener];
  if (intervalId === null) {
    snapshot = Date.now();
    intervalId = setInterval(tick, TICK_MS);
  }
  return () => {
    listeners = listeners.filter((registered) => registered !== listener);
    if (listeners.length === 0 && intervalId !== null) {
      clearInterval(intervalId);
      intervalId = null;
    }
  };
}

/**
 * The cached tick value. It changes only when `tick` runs, which is what
 * useSyncExternalStore requires — returning `Date.now()` here would loop.
 */
export function getClockSnapshot(): number {
  return snapshot;
}

/** Current epoch milliseconds, re-rendering the caller once per second. */
export function useClock(): number {
  return useSyncExternalStore(
    subscribeToClock,
    getClockSnapshot,
    getClockSnapshot,
  );
}
