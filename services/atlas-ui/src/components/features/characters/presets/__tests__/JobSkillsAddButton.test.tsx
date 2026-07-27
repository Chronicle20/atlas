import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

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

beforeEach(() => getSkillsByJobIdMock.mockReset());

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

  it("accepts a numeric id for a job not in the curated list", async () => {
    getSkillsByJobIdMock.mockResolvedValue([90000]);
    const onAddMany = vi.fn();
    renderButton({ onAddMany });
    await userEvent.click(screen.getByRole("button", { name: /job skills/i }));
    await userEvent.type(
      screen.getByPlaceholderText(/add all skills for a job/i),
      "1000",
    );
    await userEvent.click(
      await screen.findByRole("option", {
        name: /add all skills for job 1000/i,
      }),
    );
    await waitFor(() =>
      expect(getSkillsByJobIdMock).toHaveBeenCalledWith(1000),
    );
    expect(onAddMany).toHaveBeenCalledWith([90000]);
  });
});
