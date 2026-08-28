import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { SaveBar } from "../SaveBar";

describe("SaveBar", () => {
  it("enables Save with 'Unsaved changes' when there is no gate and it's dirty", () => {
    render(
      <SaveBar
        dirty={true}
        isSaving={false}
        onSave={vi.fn()}
        onDiscard={vi.fn()}
      />,
    );
    expect(screen.getByRole("button", { name: /^save$/i })).not.toBeDisabled();
    expect(screen.getByText(/unsaved changes/i)).toBeInTheDocument();
  });

  it("enables Save with 'Unsaved changes' when the gate is satisfied", () => {
    render(
      <SaveBar
        dirty={true}
        isSaving={false}
        blockingIssues={0}
        onSave={vi.fn()}
        onDiscard={vi.fn()}
      />,
    );
    expect(screen.getByRole("button", { name: /^save$/i })).not.toBeDisabled();
    expect(screen.getByText(/unsaved changes/i)).toBeInTheDocument();
  });

  it("disables Save and reports one blocking error", () => {
    render(
      <SaveBar
        dirty={true}
        isSaving={false}
        blockingIssues={1}
        onSave={vi.fn()}
        onDiscard={vi.fn()}
      />,
    );
    expect(screen.getByRole("button", { name: /^save$/i })).toBeDisabled();
    expect(screen.getByText(/1 blocking error/i)).toBeInTheDocument();
    expect(screen.queryByText(/unsaved changes/i)).not.toBeInTheDocument();
  });

  it("reports three blocking errors", () => {
    render(
      <SaveBar
        dirty={true}
        isSaving={false}
        blockingIssues={3}
        onSave={vi.fn()}
        onDiscard={vi.fn()}
      />,
    );
    expect(screen.getByText(/3 blocking errors/i)).toBeInTheDocument();
  });

  it("keeps Discard usable while Save is blocked", async () => {
    const onDiscard = vi.fn();
    render(
      <SaveBar
        dirty={true}
        isSaving={false}
        blockingIssues={2}
        onSave={vi.fn()}
        onDiscard={onDiscard}
      />,
    );
    expect(screen.getByRole("button", { name: /discard/i })).not.toBeDisabled();
    await userEvent.click(screen.getByRole("button", { name: /discard/i }));
    await userEvent.click(
      screen.getByRole("button", { name: /discard changes/i }),
    );
    expect(onDiscard).toHaveBeenCalledTimes(1);
  });

  it("disables Save and shows 'No unsaved changes' when clean", () => {
    render(
      <SaveBar
        dirty={false}
        isSaving={false}
        blockingIssues={0}
        onSave={vi.fn()}
        onDiscard={vi.fn()}
      />,
    );
    expect(screen.getByRole("button", { name: /^save$/i })).toBeDisabled();
    expect(screen.getByText(/no unsaved changes/i)).toBeInTheDocument();
  });
});
