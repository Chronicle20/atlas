import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { SkillsSection } from "../SkillsSection";

const fakeTenant = { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } as never;

vi.mock("@/lib/hooks/api/useCharacterSkills", () => ({
  useCharacterSkills: vi.fn().mockReturnValue({ data: [{ id: "1101004", level: 5, masterLevel: 20, expiration: "0001-01-01T00:00:00Z", cooldownExpiresAt: "0001-01-01T00:00:00Z" }] }),
}));
vi.mock("@/lib/hooks/api/useJobSkills", () => ({
  useJobSkills: vi.fn().mockReturnValue({ data: [1101000, 1101004], isSuccess: true }),
}));
vi.mock("@/lib/hooks/api/useSkillDefinition", () => ({
  useSkillDefinition: vi.fn().mockReturnValue({ data: { id: 1, name: "Stub", description: "", effects: [], iconUrl: "x.png" }, isSuccess: true, isLoading: false }),
}));

const baseChar = { id: "42", attributes: { jobId: 112, name: "Hero" } } as never;

function renderS() {
  const qc = new QueryClient();
  return render(<QueryClientProvider client={qc}><SkillsSection character={baseChar} tenant={fakeTenant} /></QueryClientProvider>);
}

describe("SkillsSection", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders one tab per job in path", () => {
    renderS();
    ["Beginner", "Warrior", "Fighter", "Crusader", "Hero"].forEach((label) => {
      expect(screen.getByRole("tab", { name: label })).toBeInTheDocument();
    });
  });

  it("defaults to the current job tab", () => {
    renderS();
    expect(screen.getByRole("tab", { name: "Hero", selected: true })).toBeInTheDocument();
  });

  it("renders empty state when jobTreePath is empty", () => {
    render(<QueryClientProvider client={new QueryClient()}>
      <SkillsSection character={{ id: "1", attributes: { jobId: 99999, name: "X" } } as never} tenant={fakeTenant} />
    </QueryClientProvider>);
    expect(screen.getByText(/No skill book available/i)).toBeInTheDocument();
  });
});
