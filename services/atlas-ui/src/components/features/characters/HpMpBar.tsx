import { cn } from "@/lib/utils";

interface Props {
  label: string;
  cur: number;
  max: number;
  colorClass: string;
}

export function HpMpBar({ label, cur, max, colorClass }: Props) {
  const pct = max > 0 ? Math.min(100, (cur / max) * 100) : 0;
  return (
    <div className="flex flex-col gap-1">
      <span className="text-xs"><strong>{label}:</strong> {cur} / {max}</span>
      <div className="h-2 rounded bg-muted overflow-hidden">
        <div className={cn("h-full rounded", colorClass)} style={{ width: `${pct}%` }} />
      </div>
    </div>
  );
}
