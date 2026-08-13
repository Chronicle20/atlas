import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { stateLabel } from "@/components/features/transports/transport-format";
import type { RouteState } from "@/types/models/transport";

/**
 * State is never encoded by colour alone — the pill always carries its text
 * label. `out_of_service` gets the destructive treatment because it means the
 * scheduler produced no trips for the route today, which is a fault, not a
 * quiet status.
 */
const STATE_VARIANT: Record<
  RouteState,
  {
    variant: "default" | "secondary" | "outline" | "destructive";
    className?: string;
  }
> = {
  out_of_service: { variant: "destructive" },
  in_transit: { variant: "default" },
  locked_entry: { variant: "secondary" },
  open_entry: { variant: "outline", className: "border-primary text-primary" },
  awaiting_return: { variant: "outline" },
};

export function RouteStatePill({ state }: { state: RouteState }) {
  const { variant, className } = STATE_VARIANT[state];
  return (
    <Badge variant={variant} className={cn("whitespace-nowrap", className)}>
      {stateLabel(state)}
    </Badge>
  );
}
