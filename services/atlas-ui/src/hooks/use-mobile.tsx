import * as React from "react";

const MOBILE_BREAKPOINT = 768;

function subscribe(onChange: () => void) {
  const mql = window.matchMedia(`(max-width: ${MOBILE_BREAKPOINT - 1}px)`);
  mql.addEventListener("change", onChange);
  return () => mql.removeEventListener("change", onChange);
}

function getSnapshot() {
  return window.innerWidth < MOBILE_BREAKPOINT;
}

// useSyncExternalStore (rather than an effect that calls setState) is the
// React-recommended way to subscribe to an external source like matchMedia:
// it reads the snapshot synchronously on first render instead of flashing
// a default value until the effect commits.
export function useIsMobile() {
  return React.useSyncExternalStore(subscribe, getSnapshot);
}
