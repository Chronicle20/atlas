import type { ReactNode } from "react";
import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { SkillWidget } from "../SkillWidget";

const fakeTenant = { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } as never;

const useSkillDefinitionMock = vi.fn();
vi.mock("@/lib/hooks/api/useSkillDefinition", () => ({
  useSkillDefinition: (...a: unknown[]) => useSkillDefinitionMock(...a),
}));

function renderW(node: ReactNode) {
  const qc = new QueryClient();
  return render(<QueryClientProvider client={qc}>{node}</QueryClientProvider>);
}

describe("SkillWidget", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders learned level / master level when learned", () => {
    useSkillDefinitionMock.mockReturnValue({ data: { id: 1101000, name: "Iron Body", description: "", effects: [], iconUrl: "x.png" }, isSuccess: true });
    renderW(<SkillWidget skillId={1101000} learnedLevel={5} learnedMasterLevel={20} tenant={fakeTenant} />);
    expect(screen.getByText("5 / 20")).toBeInTheDocument();
  });

  it("renders 0/master and faded when not learned", () => {
    useSkillDefinitionMock.mockReturnValue({ data: { id: 1101000, name: "Iron Body", description: "", effects: [{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{}], iconUrl: "x.png" }, isSuccess: true });
    const { container } = renderW(<SkillWidget skillId={1101000} tenant={fakeTenant} />);
    expect(screen.getByText("0 / 20")).toBeInTheDocument();
    expect(container.querySelector(".opacity-50")).toBeTruthy();
  });

  it("falls back to effects.length as master when learnedMasterLevel is 0 and still renders x/y", () => {
    useSkillDefinitionMock.mockReturnValue({ data: { id: 1101000, name: "Iron Body", description: "", effects: [{}], iconUrl: "x.png" }, isSuccess: true });
    renderW(<SkillWidget skillId={1101000} learnedLevel={1} learnedMasterLevel={0} tenant={fakeTenant} />);
    expect(screen.getByText("1 / 1")).toBeInTheDocument();
  });
});
