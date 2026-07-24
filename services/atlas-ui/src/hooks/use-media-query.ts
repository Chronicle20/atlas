import * as React from "react";

// Generalized matchMedia subscription — same useSyncExternalStore pattern as
// use-mobile.tsx (which stays hard-coded to the sidebar's 768px breakpoint):
// the snapshot is read synchronously on first render instead of flashing a
// default until an effect commits.
export function useMediaQuery(query: string): boolean {
  const subscribe = React.useCallback(
    (onChange: () => void) => {
      const mql = window.matchMedia(query);
      mql.addEventListener("change", onChange);
      return () => mql.removeEventListener("change", onChange);
    },
    [query],
  );
  const getSnapshot = React.useCallback(
    () => window.matchMedia(query).matches,
    [query],
  );
  return React.useSyncExternalStore(subscribe, getSnapshot);
}
