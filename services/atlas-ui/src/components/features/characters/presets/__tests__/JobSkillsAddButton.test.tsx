import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { FIXTURE_JOBS_SORTED } from "@/lib/jobs/__tests__/job-graph-fixtures";
import type {
  PresetJobOption,
  PresetJobOptionsResult,
} from "@/lib/hooks/usePresetJobOptions";

const getSkillsByJobIdMock = vi.fn();
vi.mock("@/services/api/jobs.service", () => ({
  jobsService: {
    getSkillsByJobId: (...a: unknown[]) => getSkillsByJobIdMock(...a),
  },
}));

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

// Version-gated option list — mocked to a controlled set (see JobCombobox test).
const optionsMock = vi.fn<() => PresetJobOptionsResult>();
vi.mock("@/lib/hooks/usePresetJobOptions", () => ({
  usePresetJobOptions: () => optionsMock(),
}));

/** Shape a usePresetJobOptions success result from a plain option list. */
function loaded(options: PresetJobOption[]): PresetJobOptionsResult {
  return { options, isPending: false, isError: false };
}

import { JobSkillsAddButton } from "../JobSkillsAddButton";

function renderButton(
  props: Partial<React.ComponentProps<typeof JobSkillsAddButton>> = {},
) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <JobSkillsAddButton onAddMany={vi.fn()} {...props} />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  getSkillsByJobIdMock.mockReset();
  optionsMock.mockReturnValue(loaded(FIXTURE_JOBS_SORTED));
});

describe("JobSkillsAddButton", () => {
  it("filters jobs by name and adds the picked family's skills", async () => {
    getSkillsByJobIdMock.mockResolvedValue([4111000, 4111001, 4111002]);
    const onAddMany = vi.fn();
    renderButton({ onAddMany });
    await userEvent.click(screen.getByRole("button", { name: /job skills/i }));
    await userEvent.type(
      screen.getByPlaceholderText(/add all skills for a job/i),
      "hermit",
    );
    await userEvent.click(
      await screen.findByRole("option", { name: /hermit/i }),
    );
    await waitFor(() => expect(getSkillsByJobIdMock).toHaveBeenCalledWith(411));
    expect(onAddMany).toHaveBeenCalledWith([4111000, 4111001, 4111002]);
  });

  it("adds an Aran family's skills when the tenant has that job", async () => {
    getSkillsByJobIdMock.mockResolvedValue([21100000, 21100001]);
    const onAddMany = vi.fn();
    renderButton({ onAddMany });
    await userEvent.click(screen.getByRole("button", { name: /job skills/i }));
    await userEvent.type(
      screen.getByPlaceholderText(/add all skills for a job/i),
      "aran 2",
    );
    await userEvent.click(
      await screen.findByRole("option", { name: /aran 2/i }),
    );
    await waitFor(() =>
      expect(getSkillsByJobIdMock).toHaveBeenCalledWith(2110),
    );
    expect(onAddMany).toHaveBeenCalledWith([21100000, 21100001]);
  });

  it("accepts a numeric id for a job not in the tenant's set", async () => {
    getSkillsByJobIdMock.mockResolvedValue([90000]);
    const onAddMany = vi.fn();
    renderButton({ onAddMany });
    await userEvent.click(screen.getByRole("button", { name: /job skills/i }));
    await userEvent.type(
      screen.getByPlaceholderText(/add all skills for a job/i),
      "99999",
    );
    await userEvent.click(
      await screen.findByRole("option", {
        name: /add all skills for job 99999/i,
      }),
    );
    await waitFor(() =>
      expect(getSkillsByJobIdMock).toHaveBeenCalledWith(99999),
    );
    expect(onAddMany).toHaveBeenCalledWith([90000]);
  });

  it("shows a loading affordance, not 'No matches', while options are pending", async () => {
    optionsMock.mockReturnValue({
      options: [],
      isPending: true,
      isError: false,
    });
    renderButton();
    await userEvent.click(screen.getByRole("button", { name: /job skills/i }));
    expect(screen.queryByText(/no matches/i)).toBeNull();
    expect(
      screen.getByText(/loading this tenant's job list/i),
    ).toBeInTheDocument();
  });

  it("shows a load-failure message, not 'No matches', when availability errors", async () => {
    optionsMock.mockReturnValue({
      options: [],
      isPending: false,
      isError: true,
    });
    renderButton();
    await userEvent.click(screen.getByRole("button", { name: /job skills/i }));
    expect(screen.queryByText(/no matches/i)).toBeNull();
    expect(
      screen.getByText(/couldn't load this tenant's job list/i),
    ).toBeInTheDocument();
  });

  it("still accepts manual numeric id entry while options are pending", async () => {
    optionsMock.mockReturnValue({
      options: [],
      isPending: true,
      isError: false,
    });
    getSkillsByJobIdMock.mockResolvedValue([90000]);
    const onAddMany = vi.fn();
    renderButton({ onAddMany });
    await userEvent.click(screen.getByRole("button", { name: /job skills/i }));
    await userEvent.type(
      screen.getByPlaceholderText(/add all skills for a job/i),
      "99999",
    );
    await userEvent.click(
      await screen.findByRole("option", {
        name: /add all skills for job 99999/i,
      }),
    );
    await waitFor(() =>
      expect(getSkillsByJobIdMock).toHaveBeenCalledWith(99999),
    );
    expect(onAddMany).toHaveBeenCalledWith([90000]);
  });
});
