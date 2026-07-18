import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import type { Tenant } from "@/services/api/tenants.service";
import type { SkillDefinitionWithIcon } from "@/lib/hooks/api/useSkillDefinition";

const useTenantMock = vi.fn();
const useJobSkillsMock = vi.fn();
const useJobSkillDefsMock = vi.fn();

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => useTenantMock(),
}));
vi.mock("@/lib/hooks/api/useJobSkills", () => ({
  useJobSkills: (...a: unknown[]) => useJobSkillsMock(...a),
  jobSkillsKeys: { all: ["job-skills"], detail: () => [] },
}));
vi.mock("@/lib/hooks/api/useJobSkillDefinitions", () => ({
  useJobSkillDefinitions: (...a: unknown[]) => useJobSkillDefsMock(...a),
}));

import { JobDetailPage } from "@/pages/JobDetailPage";

const v83 = {
  id: "t1",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
} as unknown as Tenant;

function def(over: Partial<SkillDefinitionWithIcon>): SkillDefinitionWithIcon {
  return {
    id: 1101004,
    name: "Iron Body",
    description: "Hardens the body.",
    action: false,
    element: "",
    animationTime: 0,
    maxLevel: 20,
    effects: [{ weaponDefense: 16 }],
    iconUrl: "/api/assets/x/GMS/83.1/skill/1101004/icon.png",
    ...over,
  } as SkillDefinitionWithIcon;
}

function renderAt(jobId = "112") {
  return render(
    <MemoryRouter initialEntries={[`/jobs/${jobId}`]}>
      <Routes>
        <Route path="/jobs/:jobId" element={<JobDetailPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("JobDetailPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useTenantMock.mockReturnValue({ activeTenant: v83 });
  });

  it("shows a skeleton while skill ids are loading", () => {
    useJobSkillsMock.mockReturnValue({
      data: undefined,
      isLoading: true,
      isError: false,
    });
    useJobSkillDefsMock.mockReturnValue({
      definitions: [],
      isLoading: true,
      isError: false,
    });
    renderAt();
    expect(screen.getByTestId("job-detail-loading")).toBeInTheDocument();
  });

  it("shows an empty state when the job grants no skills", () => {
    useJobSkillsMock.mockReturnValue({
      data: [],
      isLoading: false,
      isError: false,
    });
    useJobSkillDefsMock.mockReturnValue({
      definitions: [],
      isLoading: false,
      isError: false,
    });
    renderAt();
    expect(screen.getByText(/grants no skills/i)).toBeInTheDocument();
  });

  it("renders a skill row with a Master Lv indicator and a type badge", () => {
    useJobSkillsMock.mockReturnValue({
      data: [1101004],
      isLoading: false,
      isError: false,
    });
    useJobSkillDefsMock.mockReturnValue({
      definitions: [def({})],
      isLoading: false,
      isError: false,
    });
    renderAt();
    expect(screen.getByText("Iron Body")).toBeInTheDocument();
    expect(screen.getByText(/Master Lv/i)).toBeInTheDocument();
    expect(screen.getByText("20")).toBeInTheDocument();
    expect(screen.getByText("Passive")).toBeInTheDocument();
  });

  it("renders Beginner skills with a curated fallback when the server name is blank", () => {
    useJobSkillsMock.mockReturnValue({
      data: [1000, 1004],
      isLoading: false,
      isError: false,
    });
    useJobSkillDefsMock.mockReturnValue({
      definitions: [def({ id: 1000, name: "" }), def({ id: 1004, name: "" })],
      isLoading: false,
      isError: false,
    });
    renderAt("0");
    expect(screen.getByText("Three Snails")).toBeInTheDocument();
    expect(screen.getByText("Monster Riding")).toBeInTheDocument();
  });

  it("renders the job id as a copyable (focusable) header element", () => {
    useJobSkillsMock.mockReturnValue({
      data: [],
      isLoading: false,
      isError: false,
    });
    useJobSkillDefsMock.mockReturnValue({
      definitions: [],
      isLoading: false,
      isError: false,
    });
    renderAt("112");
    expect(screen.getByText("112")).toHaveAttribute("tabindex", "0");
  });

  it("uses a page-local overflow container for scrolling", () => {
    useJobSkillsMock.mockReturnValue({
      data: [],
      isLoading: false,
      isError: false,
    });
    useJobSkillDefsMock.mockReturnValue({
      definitions: [],
      isLoading: false,
      isError: false,
    });
    const { container } = renderAt();
    expect(container.firstChild).toHaveClass("overflow-y-auto");
  });

  it("renders a markup description with line breaks and no raw '#'", () => {
    useJobSkillsMock.mockReturnValue({
      data: [1101004],
      isLoading: false,
      isError: false,
    });
    useJobSkillDefsMock.mockReturnValue({
      definitions: [def({ description: "Line one\n#cColored#" })],
      isLoading: false,
      isError: false,
    });
    renderAt();
    fireEvent.click(screen.getByRole("button", { name: /iron body/i }));
    expect(screen.getByText("Line one")).toBeInTheDocument();
    expect(screen.getByText("Colored")).toBeInTheDocument();
    expect(screen.queryByText(/#c/)).toBeNull();
  });

  it("expanding a skill reveals its per-level table", () => {
    useJobSkillsMock.mockReturnValue({
      data: [1101004],
      isLoading: false,
      isError: false,
    });
    useJobSkillDefsMock.mockReturnValue({
      definitions: [
        def({ effects: [{ weaponDefense: 16 }, { weaponDefense: 18 }] }),
      ],
      isLoading: false,
      isError: false,
    });
    renderAt();
    fireEvent.click(screen.getByRole("button", { name: /iron body/i }));
    expect(screen.getByText("Weapon Def")).toBeInTheDocument();
  });
});
