// services/atlas-ui/src/components/features/characters/presets/useSyncedNumberInput.ts
import { useState } from "react";

/**
 * Local echo for a numeric field so the DOM value reflects keystrokes as
 * they land — the canonical value only updates once the reducer round-trips
 * the corresponding onSet call. Re-synced whenever the underlying value
 * changes from outside this input (e.g. switching presets), via React's
 * "adjust state during render" pattern rather than a useEffect
 * (https://react.dev/learn/you-might-not-need-an-effect#adjusting-some-state-when-a-prop-changes),
 * so it doesn't trip react-hooks/set-state-in-effect.
 */
export function useSyncedNumberInput(
  value: number,
): [string, (v: string) => void] {
  const [draft, setDraft] = useState(String(value));
  const [prevValue, setPrevValue] = useState(value);
  if (prevValue !== value) {
    setPrevValue(value);
    setDraft(String(value));
  }
  return [draft, setDraft];
}
