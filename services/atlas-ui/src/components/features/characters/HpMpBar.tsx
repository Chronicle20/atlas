import { cn } from "@/lib/utils";

interface Props {
  label: string;
  cur: number;
  max: number;
  colorClass: string;
  /**
   * Equipment / buff bonus contributing to `max`. When present, rendered as
   * `+bonus` in green next to the cur/max text so it's visually obvious that
   * effective-stats data is reaching the bar.
   */
  bonus?: number | undefined;
}

export function HpMpBar({ label, cur, max, colorClass, bonus }: Props) {
  const pct = max > 0 ? Math.min(100, (cur / max) * 100) : 0;
  return (
    <div className="flex flex-col gap-1">
      <span className="text-xs">
        <strong>{label}:</strong> {cur} / {max}
        {bonus != null && bonus > 0 && (
          <span className="text-emerald-600 dark:text-emerald-400"> +{bonus}</span>
        )}
      </span>
      <div className="h-2 rounded bg-muted overflow-hidden">
        <div className={cn("h-full rounded", colorClass)} style={{ width: `${pct}%` }} />
      </div>
    </div>
  );
}
