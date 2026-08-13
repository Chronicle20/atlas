import { useClock } from "@/lib/utils/clock";
import { formatCountdown } from "@/components/features/transports/transport-format";

interface CountdownProps {
  /** Absolute RFC3339 instant to count down to; null renders an em dash. */
  targetAt: string | null;
  /** What the transition is, e.g. "departs in". Hidden when there is no target. */
  label?: string;
}

/**
 * A countdown leaf. It subscribes to the shared clock on its own so a tick
 * re-renders this cell and nothing else.
 *
 * Reaching zero is not a state change: the pill next to this component keeps
 * showing the server's state until the next refetch supplies a new one.
 */
export function Countdown({ targetAt, label }: CountdownProps) {
  const now = useClock();

  if (!targetAt) {
    return <span className="text-muted-foreground">—</span>;
  }

  const target = Date.parse(targetAt);
  if (Number.isNaN(target)) {
    return <span className="text-muted-foreground">—</span>;
  }

  return (
    <span className="inline-flex items-baseline gap-1.5 tabular-nums">
      {label ? (
        <span className="text-xs text-muted-foreground">{label}</span>
      ) : null}
      <span>{formatCountdown(target - now)}</span>
    </span>
  );
}
