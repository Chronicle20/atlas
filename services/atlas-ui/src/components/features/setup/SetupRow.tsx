import type { ReactNode } from "react";

interface SetupRowProps {
  icon: ReactNode;
  label: ReactNode;
  badge: ReactNode;
  action: ReactNode;
  warning?: ReactNode;
}

export function SetupRow({ icon, label, badge, action, warning }: SetupRowProps) {
  return (
    <div className="flex flex-col gap-2 border-b last:border-0 py-3">
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <div className="text-muted-foreground">{icon}</div>
          <div>
            <p className="font-medium text-sm">{label}</p>
            <p
              className="text-xs text-muted-foreground"
              aria-live="polite"
            >
              {badge}
            </p>
          </div>
        </div>
        {action}
      </div>
      {warning}
    </div>
  );
}

export function formatCount(n: number): string {
  return new Intl.NumberFormat().format(n);
}

export function pluralize(n: number, singular: string, plural: string): string {
  return n === 1 ? singular : plural;
}
