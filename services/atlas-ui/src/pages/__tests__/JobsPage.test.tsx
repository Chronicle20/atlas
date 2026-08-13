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
import { FIXTURE_JOBS_SORTED } from "@/lib/jobs/__tests__/job-graph-fixtures";
import { buildJobGraph, type JobGraph } from "@/lib/jobs/job-graph";
import type { JobAvailabilityEntry } from "@/services/api/availability.service";
import type { JobGraphResult } from "@/lib/hooks/api/useJobGraph";

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

// JobsPage's single hierarchy source (task-202): mock it directly rather than
// the two queries it composes (useJobAvailability + useJobs), since that is
// the actual boundary JobsPage now depends on.
const useJobGraphMock = vi.fn();
vi.mock("@/lib/hooks/api/useJobGraph", () => ({
  useJobGraph: (...args: unknown[]) => useJobGraphMock(...args),
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

// The structural fixture source predates identity/wire-id divergence, so
// identity === id here; the graph-level wire-id-vs-identity divergence
// itself is covered by job-graph.test.ts and rail-groups.test.ts, not by
// this page-level suite.
const FULL_AVAILABILITY: JobAvailabilityEntry[] = FIXTURE_JOBS_SORTED.map(
  (e) => ({
    id: e.id,
    name: e.name,
    parent: e.parent,
    identity: e.id,
  }),
);

function graphOf(ids: readonly number[]): JobGraph {
  return buildJobGraph(FULL_AVAILABILITY, new Set(ids));
}

const ALL_JOBS = [
  0, 100, 110, 111, 112, 120, 121, 122, 130, 131, 132, 200, 300, 400, 500, 900,
  910, 800, 1000, 2000, 2001,
];

function jobGraphQuery(
  state: "pending" | "success" | "error",
  ids: readonly number[] = ALL_JOBS,
): JobGraphResult {
  return {
    graph: state === "success" ? graphOf(ids) : new Map(),
    isPending: state === "pending",
    isSuccess: state === "success",
    isError: state === "error",
  };
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
    useJobGraphMock.mockReturnValue(jobGraphQuery("success"));
    renderAt("/jobs");
    expect(screen.getByText(/select a tenant/i)).toBeInTheDocument();
    expect(screen.queryByText("Branches")).not.toBeInTheDocument();
  });

  it("defaults /jobs to the Warrior entry with no skill selected", () => {
    useJobGraphMock.mockReturnValue(jobGraphQuery("success"));
    renderAt("/jobs");
    expect(screen.getByRole("button", { name: /Warrior 10/ })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    expect(screen.getByText("Warrior — Skills")).toBeInTheDocument();
    expect(useJobSkillsMock).toHaveBeenCalledWith(expect.anything(), 100);
    expect(
      screen.getByText("Select a skill to inspect it"),
    ).toBeInTheDocument();
  });

  it("deep-links /jobs/110 to Fighter in the Warrior branch", () => {
    useJobGraphMock.mockReturnValue(jobGraphQuery("success"));
    renderAt("/jobs/110");
    expect(screen.getByRole("button", { name: /Warrior 10/ })).toHaveAttribute(
      "aria-pressed",
      "true",
    ); // rail highlights the branch
    expect(screen.getByRole("button", { name: /Fighter/ })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    expect(screen.getByText("Fighter — Skills")).toBeInTheDocument();
  });

  it("deep-links ?skill= to an open detail panel once definitions load", () => {
    useJobGraphMock.mockReturnValue(jobGraphQuery("success"));
    renderAt("/jobs/110?skill=1101007");
    expect(screen.getByText("ID 1101007")).toBeInTheDocument();
    expect(screen.getByLabelText("Skill level")).toBeInTheDocument();
  });

  it("selecting a job pushes /jobs/:id and clears the skill selection", () => {
    useJobGraphMock.mockReturnValue(jobGraphQuery("success"));
    renderAt("/jobs/100?skill=1001004");
    fireEvent.click(screen.getByRole("button", { name: /Fighter/ }));
    expect(screen.getByTestId("location")).toHaveTextContent("/jobs/110");
    expect(screen.getByTestId("location")).not.toHaveTextContent("skill=");
  });

  it("selecting a skill writes ?skill= to the URL", () => {
    useJobGraphMock.mockReturnValue(jobGraphQuery("success"));
    renderAt("/jobs/110");
    fireEvent.click(screen.getByRole("button", { name: /Power Guard/ }));
    expect(screen.getByTestId("location")).toHaveTextContent(
      "/jobs/110?skill=1101007",
    );
  });

  it("selecting a job pushes (not replaces) so Back works", () => {
    useJobGraphMock.mockReturnValue(jobGraphQuery("success"));
    renderAt("/jobs/100");
    fireEvent.click(screen.getByRole("button", { name: /Fighter/ }));
    expect(screen.getByTestId("nav-type")).toHaveTextContent("PUSH");
    expect(screen.getByTestId("location")).toHaveTextContent("/jobs/110");
  });

  it("selecting a skill pushes", () => {
    useJobGraphMock.mockReturnValue(jobGraphQuery("success"));
    renderAt("/jobs/110");
    fireEvent.click(screen.getByRole("button", { name: /Power Guard/ }));
    expect(screen.getByTestId("nav-type")).toHaveTextContent("PUSH");
    expect(screen.getByTestId("location")).toHaveTextContent("?skill=1101007");
  });

  it("normalizes an unknown jobId to /jobs with the default selection", async () => {
    useJobGraphMock.mockReturnValue(jobGraphQuery("success"));
    renderAt("/jobs/99999");
    await waitFor(() =>
      expect(screen.getByTestId("location")).toHaveTextContent(/^\/jobs$/),
    );
    expect(screen.getByText("Warrior — Skills")).toBeInTheDocument();
  });

  it("normalizing an unknown jobId replaces (Back does not bounce)", async () => {
    useJobGraphMock.mockReturnValue(jobGraphQuery("success"));
    renderAt("/jobs/99999");
    await waitFor(() =>
      expect(screen.getByTestId("location")).toHaveTextContent(/^\/jobs$/),
    );
    expect(screen.getByTestId("nav-type")).toHaveTextContent("REPLACE");
  });

  it("normalizes a version-hidden jobId (Evan on v83) to /jobs", async () => {
    useJobGraphMock.mockReturnValue(jobGraphQuery("success"));
    renderAt("/jobs/2200");
    await waitFor(() =>
      expect(screen.getByTestId("location")).toHaveTextContent(/^\/jobs$/),
    );
  });

  it("strips a ?skill= that does not resolve for the job", async () => {
    useJobGraphMock.mockReturnValue(jobGraphQuery("success"));
    renderAt("/jobs/110?skill=424242");
    await waitFor(() =>
      expect(screen.getByTestId("location")).not.toHaveTextContent("skill="),
    );
    expect(screen.getByTestId("location")).toHaveTextContent("/jobs/110");
  });

  it("stripping a stale ?skill= replaces", async () => {
    useJobGraphMock.mockReturnValue(jobGraphQuery("success"));
    renderAt("/jobs/110?skill=424242");
    await waitFor(() =>
      expect(screen.getByTestId("location")).not.toHaveTextContent("skill="),
    );
    expect(screen.getByTestId("nav-type")).toHaveTextContent("REPLACE");
  });

  it("gates rail entries by tenant version (GMS v12)", () => {
    useTenantMock.mockReturnValue({ activeTenant: tenant(12) });
    // A GMS v12-shaped job set: the four launch-era explorer branches (no
    // Pirate) plus the GM line — no Cygnus Knights, Legends, or Brigadier.
    // Visibility is driven by the tenant's actual graph (useJobGraph), not
    // by majorVersion, so this fixture stands in for what v12 would ingest.
    useJobGraphMock.mockReturnValue(
      jobGraphQuery(
        "success",
        [
          0, 100, 110, 111, 112, 120, 121, 122, 130, 131, 132, 200, 300, 400,
          900, 910,
        ],
      ),
    );
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
    useJobGraphMock.mockReturnValue(jobGraphQuery("success"));
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
    useJobGraphMock.mockReturnValue(jobGraphQuery("success"));
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

describe("JobsPage — tenant job graph (useJobGraph)", () => {
  it("does not redirect a valid /jobs/112 while either query is pending (task-182 D10)", async () => {
    useTenantMock.mockReturnValue({ activeTenant: tenant(83) });
    useJobGraphMock.mockReturnValue(jobGraphQuery("pending"));
    useJobSkillsMock.mockReturnValue({
      data: [],
      isLoading: true,
      isError: false,
    });
    useJobSkillDefinitionsMock.mockReturnValue({
      definitions: [],
      isLoading: true,
      isError: false,
    });
    useMediaQueryMock.mockReturnValue(true);

    renderAt("/jobs/112");

    // Give any stray effect a tick to fire, then assert the location never
    // moved off the deep link — a pending graph must not read as "absent".
    await waitFor(() =>
      expect(screen.getByTestId("location").textContent).toBe("/jobs/112"),
    );
  });

  it("renders the load-error card when either query fails", () => {
    useTenantMock.mockReturnValue({ activeTenant: tenant(83) });
    useJobGraphMock.mockReturnValue(jobGraphQuery("error"));
    useJobSkillsMock.mockReturnValue({
      data: [],
      isLoading: false,
      isError: false,
    });
    useJobSkillDefinitionsMock.mockReturnValue({
      definitions: [],
      isLoading: false,
      isError: false,
    });
    useMediaQueryMock.mockReturnValue(true);

    renderAt("/jobs/112");
    expect(screen.getByTestId("jobs-load-error")).toBeInTheDocument();
    expect(
      screen.queryByTestId("branch-rail-skeleton"),
    ).not.toBeInTheDocument();
  });

  it("renders the rail skeleton while the graph is loading", () => {
    useTenantMock.mockReturnValue({ activeTenant: tenant(83) });
    useJobGraphMock.mockReturnValue(jobGraphQuery("pending"));
    useJobSkillsMock.mockReturnValue({
      data: [],
      isLoading: true,
      isError: false,
    });
    useJobSkillDefinitionsMock.mockReturnValue({
      definitions: [],
      isLoading: true,
      isError: false,
    });
    useMediaQueryMock.mockReturnValue(true);

    renderAt("/jobs/112");
    expect(screen.getByTestId("branch-rail-skeleton")).toBeInTheDocument();
  });

  it("shows only the branches present in the tenant job graph", () => {
    useTenantMock.mockReturnValue({ activeTenant: tenant(48) });
    // A GMS 48-shaped graph: explorers only, no Pirate/GM/Cygnus/Legends/Brigadier.
    useJobGraphMock.mockReturnValue(
      jobGraphQuery("success", [0, 100, 110, 111, 112, 200, 300, 400]),
    );
    useJobSkillsMock.mockReturnValue({
      data: [1121000],
      isLoading: false,
      isError: false,
    });
    useJobSkillDefinitionsMock.mockReturnValue({
      definitions: [def(1121000, "Brandish")],
      isLoading: false,
      isError: false,
    });
    useMediaQueryMock.mockReturnValue(true);

    renderAt("/jobs/112");

    // "Warrior" renders twice (rail entry + AdvancementFlow anchor chip), so
    // getByText would throw on ambiguity — getAllByText confirms presence
    // without assuming a single match.
    expect(screen.getAllByText("Warrior").length).toBeGreaterThan(0);
    expect(screen.queryByText("Pirate")).not.toBeInTheDocument();
    expect(screen.queryByText("Noblesse")).not.toBeInTheDocument();
  });

  it("redirects a jobId absent from the tenant job graph once the query succeeds", async () => {
    useTenantMock.mockReturnValue({ activeTenant: tenant(48) });
    useJobGraphMock.mockReturnValue(
      jobGraphQuery("success", [0, 100, 110, 111, 112, 200, 300, 400]),
    );
    useJobSkillsMock.mockReturnValue({
      data: [],
      isLoading: false,
      isError: false,
    });
    useJobSkillDefinitionsMock.mockReturnValue({
      definitions: [],
      isLoading: false,
      isError: false,
    });
    useMediaQueryMock.mockReturnValue(true);

    renderAt("/jobs/1000"); // Noblesse — not in this tenant's graph

    await waitFor(() =>
      expect(screen.getByTestId("location").textContent).toBe("/jobs"),
    );
  });
});
