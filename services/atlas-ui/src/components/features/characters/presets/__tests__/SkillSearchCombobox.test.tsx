import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

const searchSkillsMock = vi.fn();
vi.mock("@/services/api/skills.service", () => ({
  skillsService: { searchSkills: (...a: unknown[]) => searchSkillsMock(...a) },
}));

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

import { SkillSearchCombobox } from "../SkillSearchCombobox";

function renderBox(
  props: Partial<React.ComponentProps<typeof SkillSearchCombobox>> = {},
) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <SkillSearchCombobox
        existingIds={[]}
        onAdd={vi.fn()}
        debounceMs={0}
        {...props}
      />
    </QueryClientProvider>,
  );
}

const page = (skills: { id: number; name: string }[]) => ({
  skills,
  pageNumber: 1,
  lastPage: 1,
});

beforeEach(() => searchSkillsMock.mockReset());

describe("SkillSearchCombobox", () => {
  it("searches by name and adds a clicked row with icon + name + id", async () => {
    searchSkillsMock.mockResolvedValue(
      page([{ id: 1001004, name: "Power Strike" }]),
    );
    const onAdd = vi.fn();
    renderBox({ onAdd });
    await userEvent.click(screen.getByRole("button", { name: /add/i }));
    await userEvent.type(screen.getByRole("textbox"), "power");
    await waitFor(() =>
      expect(searchSkillsMock).toHaveBeenCalledWith("power", {
        number: 1,
        size: 50,
      }),
    );
    const row = await screen.findByRole("option", { name: /power strike/i });
    expect(row).toHaveTextContent("1001004");
    await userEvent.click(row);
    expect(onAdd).toHaveBeenCalledWith(1001004);
  });

  it("offers a 'Use id N' escape hatch for numeric input", async () => {
    searchSkillsMock.mockResolvedValue(page([]));
    const onAdd = vi.fn();
    renderBox({ onAdd });
    await userEvent.click(screen.getByRole("button", { name: /add/i }));
    await userEvent.type(screen.getByRole("textbox"), "9101000");
    await userEvent.click(
      await screen.findByRole("option", { name: /use id 9101000/i }),
    );
    expect(onAdd).toHaveBeenCalledWith(9101000);
  });

  it("marks already-granted skills as Added and does not re-add them", async () => {
    searchSkillsMock.mockResolvedValue(
      page([{ id: 1001004, name: "Power Strike" }]),
    );
    const onAdd = vi.fn();
    renderBox({ onAdd, existingIds: [1001004] });
    await userEvent.click(screen.getByRole("button", { name: /add/i }));
    await userEvent.type(screen.getByRole("textbox"), "power");
    const row = await screen.findByRole("option", { name: /power strike/i });
    expect(row).toHaveTextContent(/added/i);
    await userEvent.click(row);
    expect(onAdd).not.toHaveBeenCalled();
  });
});
