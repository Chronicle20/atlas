import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  MemoryRouter,
  Route,
  Routes,
  useLocation,
  useNavigationType,
} from "react-router-dom";
import type { Tenant } from "@/services/api/tenants.service";
import type { SkillDefinitionWithIcon } from "@/lib/hooks/api/useSkillDefinition";
import type { SkillEffect } from "@/services/api/skills.service";

const useTenantMock = vi.fn();
vi.mock("@/context/tenant-context", () => ({
  useTenant: () => useTenantMock(),
}));

const useJobSkillsMock = vi.fn();
vi.mock("@/lib/hooks/api/useJobSkills", () => ({
  useJobSkills: (...args: unknown[]) => useJobSkillsMock(...args),
}));

const useJobSkillDefinitionsMock = vi.fn();
vi.mock("@/lib/hooks/api/useJobSkillDefinitions", () => ({
  useJobSkillDefinitions: (...args: unknown[]) =>
    useJobSkillDefinitionsMock(...args),
}));

const useMediaQueryMock = vi.fn();
vi.mock("@/hooks/use-media-query", () => ({
  useMediaQuery: () => useMediaQueryMock(),
}));

import { JobsPage } from "@/pages/JobsPage";

const tenant = (major: number) =>
  ({
    id: "t1",
    attributes: { region: "GMS", majorVersion: major, minorVersion: 1 },
  }) as unknown as Tenant;

function def(
  id: number,
  name: string,
  over?: Partial<SkillDefinitionWithIcon>,
): SkillDefinitionWithIcon {
  return {
    id,
    name,
    description: "",
    action: true,
    element: "",
    animationTime: 0,
    maxLevel: 20,
    effects: Array.from(
      { length: 20 },
      (_, i) => ({ damage: 10 + i }) as SkillEffect,
    ),
    iconUrl: `http://assets.test/skills/${id}/icon`,
    ...over,
  };
}

const warriorDefs = [def(1001004, "Power Strike"), def(1001005, "Slash Blast")];
const fighterDefs = [def(1101007, "Power Guard"), def(1101006, "Rage")];

function LocationProbe() {
  const location = useLocation();
  const navType = useNavigationType();
  return (
    <>
      <div data-testid="location">{location.pathname + location.search}</div>
      <div data-testid="nav-type">{navType}</div>
    </>
  );
}

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/jobs" element={<JobsPage />} />
        <Route path="/jobs/:jobId" element={<JobsPage />} />
      </Routes>
      <LocationProbe />
    </MemoryRouter>,
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  useTenantMock.mockReturnValue({ activeTenant: tenant(83) });
  useMediaQueryMock.mockReturnValue(true); // wide by default in these tests
  useJobSkillsMock.mockImplementation((_t: unknown, jobId: number) => ({
    data:
      jobId === 110
        ? fighterDefs.map((d) => d.id)
        : warriorDefs.map((d) => d.id),
    isLoading: false,
    isError: false,
  }));
  useJobSkillDefinitionsMock.mockImplementation(
    (_t: unknown, ids: number[]) => ({
      definitions: [...warriorDefs, ...fighterDefs].filter((d) =>
        ids.includes(d.id),
      ),
      isLoading: false,
      isError: false,
    }),
  );
});

describe("JobsPage", () => {
  it("shows the select-a-tenant card when no tenant is active", () => {
    useTenantMock.mockReturnValue({ activeTenant: null });
    renderAt("/jobs");
    expect(screen.getByText(/select a tenant/i)).toBeInTheDocument();
    expect(screen.queryByText("Branches")).not.toBeInTheDocument();
  });

  it("defaults /jobs to the Warrior entry with no skill selected", () => {
    renderAt("/jobs");
    expect(
      screen.getByRole("button", { name: /Warrior 10/ }),
    ).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByText("Warrior — Skills")).toBeInTheDocument();
    expect(useJobSkillsMock).toHaveBeenCalledWith(expect.anything(), 100);
    expect(
      screen.getByText("Select a skill to inspect it"),
    ).toBeInTheDocument();
  });

  it("deep-links /jobs/110 to Fighter in the Warrior branch", () => {
    renderAt("/jobs/110");
    expect(
      screen.getByRole("button", { name: /Warrior 10/ }),
    ).toHaveAttribute("aria-pressed", "true"); // rail highlights the branch
    expect(screen.getByRole("button", { name: /Fighter/ })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    expect(screen.getByText("Fighter — Skills")).toBeInTheDocument();
  });

  it("deep-links ?skill= to an open detail panel once definitions load", () => {
    renderAt("/jobs/110?skill=1101007");
    expect(screen.getByText("ID 1101007")).toBeInTheDocument();
    expect(screen.getByLabelText("Skill level")).toBeInTheDocument();
  });

  it("selecting a job pushes /jobs/:id and clears the skill selection", () => {
    renderAt("/jobs/100?skill=1001004");
    fireEvent.click(screen.getByRole("button", { name: /Fighter/ }));
    expect(screen.getByTestId("location")).toHaveTextContent("/jobs/110");
    expect(screen.getByTestId("location")).not.toHaveTextContent("skill=");
  });

  it("selecting a skill writes ?skill= to the URL", () => {
    renderAt("/jobs/110");
    fireEvent.click(screen.getByRole("button", { name: /Power Guard/ }));
    expect(screen.getByTestId("location")).toHaveTextContent(
      "/jobs/110?skill=1101007",
    );
  });

  it("selecting a job pushes (not replaces) so Back works", () => {
    renderAt("/jobs/100");
    fireEvent.click(screen.getByRole("button", { name: /Fighter/ }));
    expect(screen.getByTestId("nav-type")).toHaveTextContent("PUSH");
    expect(screen.getByTestId("location")).toHaveTextContent("/jobs/110");
  });

  it("selecting a skill pushes", () => {
    renderAt("/jobs/110");
    fireEvent.click(screen.getByRole("button", { name: /Power Guard/ }));
    expect(screen.getByTestId("nav-type")).toHaveTextContent("PUSH");
    expect(screen.getByTestId("location")).toHaveTextContent(
      "?skill=1101007",
    );
  });

  it("normalizes an unknown jobId to /jobs with the default selection", async () => {
    renderAt("/jobs/99999");
    await waitFor(() =>
      expect(screen.getByTestId("location")).toHaveTextContent(/^\/jobs$/),
    );
    expect(screen.getByText("Warrior — Skills")).toBeInTheDocument();
  });

  it("normalizing an unknown jobId replaces (Back does not bounce)", async () => {
    renderAt("/jobs/99999");
    await waitFor(() =>
      expect(screen.getByTestId("location")).toHaveTextContent(/^\/jobs$/),
    );
    expect(screen.getByTestId("nav-type")).toHaveTextContent("REPLACE");
  });

  it("normalizes a version-hidden jobId (Evan on v83) to /jobs", async () => {
    renderAt("/jobs/2200");
    await waitFor(() =>
      expect(screen.getByTestId("location")).toHaveTextContent(/^\/jobs$/),
    );
  });

  it("strips a ?skill= that does not resolve for the job", async () => {
    renderAt("/jobs/110?skill=424242");
    await waitFor(() =>
      expect(screen.getByTestId("location")).not.toHaveTextContent("skill="),
    );
    expect(screen.getByTestId("location")).toHaveTextContent("/jobs/110");
  });

  it("stripping a stale ?skill= replaces", async () => {
    renderAt("/jobs/110?skill=424242");
    await waitFor(() =>
      expect(screen.getByTestId("location")).not.toHaveTextContent("skill="),
    );
    expect(screen.getByTestId("nav-type")).toHaveTextContent("REPLACE");
  });

  it("gates rail entries by tenant version (GMS v12)", () => {
    useTenantMock.mockReturnValue({ activeTenant: tenant(12) });
    renderAt("/jobs");
    // "Warrior 10" is the rail entry; the flow chip reads "Warrior 1st"
    expect(
      screen.getByRole("button", { name: /Warrior 10/ }),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^GM 2$/ })).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Pirate/ }),
    ).not.toBeInTheDocument();
    expect(screen.queryByText("Cygnus Knights")).not.toBeInTheDocument();
    expect(screen.queryByText("Legends")).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Maple Leaf Brigadier/ }),
    ).not.toBeInTheDocument();
  });

  it("renders skill-list error state from the hook", () => {
    useJobSkillsMock.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
    });
    useJobSkillDefinitionsMock.mockReturnValue({
      definitions: [],
      isLoading: false,
      isError: false,
    });
    renderAt("/jobs/100");
    expect(
      screen.getByText("Failed to load this job's skills."),
    ).toBeInTheDocument();
  });

  it("below 1150px renders the detail in a dismissible sheet that clears ?skill=", async () => {
    useMediaQueryMock.mockReturnValue(false); // narrow
    renderAt("/jobs/110?skill=1101007");
    // detail content is in the sheet (dialog), not a third column
    const dialog = await screen.findByRole("dialog");
    expect(dialog).toHaveTextContent("ID 1101007");
    fireEvent.click(screen.getByRole("button", { name: /close/i }));
    await waitFor(() =>
      expect(screen.getByTestId("location")).not.toHaveTextContent("skill="),
    );
    expect(screen.getByTestId("location")).toHaveTextContent("/jobs/110");
  });
});
