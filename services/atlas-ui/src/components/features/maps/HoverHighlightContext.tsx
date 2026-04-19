import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";

export type HoverTarget =
  | { kind: "portal"; portalId: string }
  | { kind: "monster"; template: number; spawnIndex?: number }
  | { kind: "reactor"; reactorId: string }
  | { kind: "npc"; template: number; spawnIndex?: number }
  | null;

interface HoverHighlightContextValue {
  hovered: HoverTarget;
  setHovered: (t: HoverTarget) => void;
  isHovered: (target: NonNullable<HoverTarget>) => boolean;
}

const HoverHighlightContext = createContext<HoverHighlightContextValue | null>(null);

export function HoverHighlightProvider({ children }: { children: ReactNode }) {
  const [hovered, setHovered] = useState<HoverTarget>(null);

  const isHovered = useCallback(
    (target: NonNullable<HoverTarget>) => {
      if (!hovered) return false;
      if (hovered.kind !== target.kind) return false;
      switch (target.kind) {
        case "portal":
          return hovered.kind === "portal" && hovered.portalId === target.portalId;
        case "reactor":
          return hovered.kind === "reactor" && hovered.reactorId === target.reactorId;
        case "monster":
          return hovered.kind === "monster" && hovered.template === target.template;
        case "npc":
          return hovered.kind === "npc" && hovered.template === target.template;
        default:
          return false;
      }
    },
    [hovered],
  );

  const value = useMemo<HoverHighlightContextValue>(
    () => ({ hovered, setHovered, isHovered }),
    [hovered, isHovered],
  );

  return <HoverHighlightContext.Provider value={value}>{children}</HoverHighlightContext.Provider>;
}

export function useHoverHighlight(): HoverHighlightContextValue {
  const ctx = useContext(HoverHighlightContext);
  if (!ctx) {
    throw new Error("useHoverHighlight must be used within a HoverHighlightProvider");
  }
  return ctx;
}
