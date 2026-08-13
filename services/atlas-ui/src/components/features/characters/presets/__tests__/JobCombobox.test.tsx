import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  FIXTURE_JOB_TREE,
  FIXTURE_JOBS_SORTED,
} from "@/lib/jobs/__tests__/job-graph-fixtures";
import type {
  PresetJobOption,
  PresetJobOptionsResult,
} from "@/lib/hooks/usePresetJobOptions";

Element.prototype.scrollIntoView ||= () => {};

// The picker's option list is version-gated by usePresetJobOptions (tenant job
// set from GET /api/data/jobs). Mock it so these tests drive the filter/pick
// behaviour against a controlled list; the gating itself is covered in
// usePresetJobOptions.test.tsx.
const optionsMock = vi.fn<() => PresetJobOptionsResult>();
vi.mock("@/lib/hooks/usePresetJobOptions", () => ({
  usePresetJobOptions: () => optionsMock(),
}));

/** Shape a usePresetJobOptions success result from a plain option list. */
function loaded(options: PresetJobOption[]): PresetJobOptionsResult {
  return { options, isPending: false, isError: false };
}

// The trigger's fallback name (id outside the version-gated options) comes
// from the tenant's job graph via useJobNameLookup, not a component-local
// query — mock it to the same structural fixture so "falls back" coverage
// stays meaningful without standing up a QueryClientProvider.
vi.mock("@/lib/hooks/api/useJobGraph", () => ({
  useJobNameLookup: () => (id: number) =>
    FIXTURE_JOB_TREE[id]?.name ?? `Job ${id}`,
}));

import { JobCombobox } from "../JobCombobox";

beforeEach(() => optionsMock.mockReturnValue(loaded(FIXTURE_JOBS_SORTED)));

describe("JobCombobox", () => {
  it("shows the current job's name on the trigger", () => {
    render(<JobCombobox value={100} onChange={vi.fn()} />);
    expect(screen.getByRole("combobox", { name: /class/i })).toHaveTextContent(
      "Warrior",
    );
  });

  it("names Aran/Evan on the trigger, not a raw id", () => {
    render(<JobCombobox value={2100} onChange={vi.fn()} />);
    expect(screen.getByRole("combobox", { name: /class/i })).toHaveTextContent(
      "Aran 1",
    );
  });

  it("renders unmapped ids as Job <id>", () => {
    render(<JobCombobox value={4321} onChange={vi.fn()} />);
    expect(screen.getByRole("combobox", { name: /class/i })).toHaveTextContent(
      "Job 4321",
    );
  });

  it("uses the availability-provided name for a selected id present in the options", () => {
    // Regression coverage for the selectedName fallback (~JobCombobox.tsx
    // L31-34): when the id is in the version-gated options, the trigger must
    // show THAT name, not the static advancement-graph name — they can
    // diverge (e.g. wire id 500 is "Gm" pre-v0.61 but "Pirate" at v0.61+).
    optionsMock.mockReturnValue(
      loaded([{ id: 500, name: "Gm" }, ...FIXTURE_JOBS_SORTED]),
    );
    render(<JobCombobox value={500} onChange={vi.fn()} />);
    expect(screen.getByRole("combobox", { name: /class/i })).toHaveTextContent(
      "Gm",
    );
  });

  it("falls back to the job graph's name for a selected id NOT in the options", () => {
    // The id isn't in the availability-gated set (still loading, or a
    // manually-entered id the tenant hasn't released): fall back to
    // useJobNameLookup, which renders "Job <id>" for ids outside the graph
    // too.
    optionsMock.mockReturnValue(
      loaded(FIXTURE_JOBS_SORTED.filter((j) => j.id !== 100)),
    );
    render(<JobCombobox value={100} onChange={vi.fn()} />);
    expect(screen.getByRole("combobox", { name: /class/i })).toHaveTextContent(
      "Warrior",
    );
  });

  it("filters by name and picks a job as a number", async () => {
    const onChange = vi.fn();
    render(<JobCombobox value={0} onChange={onChange} />);
    await userEvent.click(screen.getByRole("combobox", { name: /class/i }));
    await userEvent.type(
      screen.getByPlaceholderText(/search by name/i),
      "bishop",
    );
    await userEvent.click(
      await screen.findByRole("option", { name: /bishop/i }),
    );
    expect(onChange).toHaveBeenCalledWith(232);
  });

  it("lets an Aran class be searched and picked when the tenant has it", async () => {
    const onChange = vi.fn();
    render(<JobCombobox value={0} onChange={onChange} />);
    await userEvent.click(screen.getByRole("combobox", { name: /class/i }));
    await userEvent.type(
      screen.getByPlaceholderText(/search by name/i),
      "aran",
    );
    await userEvent.click(
      await screen.findByRole("option", { name: /aran 2/i }),
    );
    expect(onChange).toHaveBeenCalledWith(2110);
  });

  it("hides Aran when the tenant's job set excludes it", async () => {
    // A version without Aran: usePresetJobOptions returns explorer jobs only.
    optionsMock.mockReturnValue(
      loaded(FIXTURE_JOBS_SORTED.filter((j) => j.id < 1000)),
    );
    render(<JobCombobox value={0} onChange={vi.fn()} />);
    await userEvent.click(screen.getByRole("combobox", { name: /class/i }));
    await userEvent.type(
      screen.getByPlaceholderText(/search by name/i),
      "aran",
    );
    expect(screen.queryByRole("option", { name: /aran/i })).toBeNull();
    expect(screen.getByText(/no matches/i)).toBeInTheDocument();
  });

  it("numeric input matching a listed id filters to it; unlisted ids get a Use-id row", async () => {
    const onChange = vi.fn();
    render(<JobCombobox value={0} onChange={onChange} />);
    await userEvent.click(screen.getByRole("combobox", { name: /class/i }));
    const input = screen.getByPlaceholderText(/search by name/i);
    await userEvent.type(input, "123456");
    await userEvent.click(
      await screen.findByRole("option", { name: /use id 123456/i }),
    );
    expect(onChange).toHaveBeenCalledWith(123456);
  });

  it("shows a loading affordance, not 'No matches', while options are pending", async () => {
    // Regression coverage: usePresetJobOptions' doc comment claims a pending
    // affordance exists; before this fix the empty options list fell through
    // to the same "No matches." shown for a genuinely empty search.
    optionsMock.mockReturnValue({
      options: [],
      isPending: true,
      isError: false,
    });
    render(<JobCombobox value={0} onChange={vi.fn()} />);
    await userEvent.click(screen.getByRole("combobox", { name: /class/i }));
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
    render(<JobCombobox value={0} onChange={vi.fn()} />);
    await userEvent.click(screen.getByRole("combobox", { name: /class/i }));
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
    const onChange = vi.fn();
    render(<JobCombobox value={0} onChange={onChange} />);
    await userEvent.click(screen.getByRole("combobox", { name: /class/i }));
    await userEvent.type(
      screen.getByPlaceholderText(/search by name/i),
      "123456",
    );
    await userEvent.click(
      await screen.findByRole("option", { name: /use id 123456/i }),
    );
    expect(onChange).toHaveBeenCalledWith(123456);
  });

  it("still accepts manual numeric id entry after options fail to load", async () => {
    optionsMock.mockReturnValue({
      options: [],
      isPending: false,
      isError: true,
    });
    const onChange = vi.fn();
    render(<JobCombobox value={0} onChange={onChange} />);
    await userEvent.click(screen.getByRole("combobox", { name: /class/i }));
    await userEvent.type(
      screen.getByPlaceholderText(/search by name/i),
      "123456",
    );
    await userEvent.click(
      await screen.findByRole("option", { name: /use id 123456/i }),
    );
    expect(onChange).toHaveBeenCalledWith(123456);
  });
});
