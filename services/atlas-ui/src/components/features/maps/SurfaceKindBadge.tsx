import { Badge } from "@/components/ui/badge";

interface SurfaceKindBadgeProps {
  kind: "definition" | "runtime";
}

/**
 * Marks a surface as static configuration ("Definition") or live instance
 * state ("Runtime"). FR-3: the distinction must never be carried by colour
 * alone, so the badge always renders its word; colour is a secondary cue.
 */
export function SurfaceKindBadge({ kind }: SurfaceKindBadgeProps) {
  const label = kind === "definition" ? "Definition" : "Runtime";
  return (
    <Badge variant={kind === "definition" ? "outline" : "default"}>
      {label}
    </Badge>
  );
}
