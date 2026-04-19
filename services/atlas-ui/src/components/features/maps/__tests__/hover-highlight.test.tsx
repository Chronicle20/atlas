import { act, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { HoverHighlightProvider, useHoverHighlight } from "../HoverHighlightContext";

function wrapper({ children }: { children: ReactNode }) {
  return <HoverHighlightProvider>{children}</HoverHighlightProvider>;
}

describe("HoverHighlightContext", () => {
  it("matches portals by exact portalId", () => {
    const { result } = renderHook(() => useHoverHighlight(), { wrapper });
    act(() => result.current.setHovered({ kind: "portal", portalId: "1" }));
    expect(result.current.isHovered({ kind: "portal", portalId: "1" })).toBe(true);
    expect(result.current.isHovered({ kind: "portal", portalId: "2" })).toBe(false);
  });

  it("matches reactors by exact reactorId", () => {
    const { result } = renderHook(() => useHoverHighlight(), { wrapper });
    act(() => result.current.setHovered({ kind: "reactor", reactorId: "r1" }));
    expect(result.current.isHovered({ kind: "reactor", reactorId: "r1" })).toBe(true);
    expect(result.current.isHovered({ kind: "reactor", reactorId: "r2" })).toBe(false);
  });

  it("matches monsters by template regardless of spawnIndex", () => {
    const { result } = renderHook(() => useHoverHighlight(), { wrapper });
    act(() => result.current.setHovered({ kind: "monster", template: 1234 }));
    expect(result.current.isHovered({ kind: "monster", template: 1234, spawnIndex: 0 })).toBe(true);
    expect(result.current.isHovered({ kind: "monster", template: 1234, spawnIndex: 5 })).toBe(true);
    expect(result.current.isHovered({ kind: "monster", template: 9999 })).toBe(false);
  });

  it("matches NPCs by template regardless of spawnIndex", () => {
    const { result } = renderHook(() => useHoverHighlight(), { wrapper });
    act(() => result.current.setHovered({ kind: "npc", template: 2001, spawnIndex: 3 }));
    expect(result.current.isHovered({ kind: "npc", template: 2001 })).toBe(true);
    expect(result.current.isHovered({ kind: "npc", template: 2001, spawnIndex: 0 })).toBe(true);
    expect(result.current.isHovered({ kind: "npc", template: 2002 })).toBe(false);
  });

  it("matches nothing when hovered is null", () => {
    const { result } = renderHook(() => useHoverHighlight(), { wrapper });
    expect(result.current.isHovered({ kind: "portal", portalId: "1" })).toBe(false);
    expect(result.current.isHovered({ kind: "monster", template: 1 })).toBe(false);
  });

  it("does not cross-match different kinds", () => {
    const { result } = renderHook(() => useHoverHighlight(), { wrapper });
    act(() => result.current.setHovered({ kind: "monster", template: 100 }));
    expect(result.current.isHovered({ kind: "npc", template: 100 })).toBe(false);
  });
});
