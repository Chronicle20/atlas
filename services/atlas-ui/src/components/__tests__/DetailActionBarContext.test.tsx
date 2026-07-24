import { useState } from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import {
  DetailActionBar,
  DetailActionBarProvider,
  useRegisterDetailActionBar,
  type DetailActionBarConfig,
} from "../DetailActionBarContext";

function Harness({
  config,
  onSave,
  onDiscard,
}: {
  config: Omit<DetailActionBarConfig, "onSave" | "onDiscard"> | null;
  onSave: () => void;
  onDiscard: () => void;
}) {
  useRegisterDetailActionBar(config ? { ...config, onSave, onDiscard } : null);
  return null;
}

function renderBar(
  props: Partial<React.ComponentProps<typeof Harness>> = {},
  registered = true,
) {
  const onSave = props.onSave ?? vi.fn();
  const onDiscard = props.onDiscard ?? vi.fn();
  render(
    <DetailActionBarProvider>
      <Harness
        config={
          registered ? { dirty: true, isSaving: false, ...props.config } : null
        }
        onSave={onSave}
        onDiscard={onDiscard}
      />
      <DetailActionBar />
    </DetailActionBarProvider>,
  );
  return { onSave, onDiscard };
}

describe("DetailActionBar", () => {
  it("renders nothing until a page registers", () => {
    render(
      <DetailActionBarProvider>
        <DetailActionBar />
      </DetailActionBarProvider>,
    );
    expect(screen.queryByRole("button", { name: /^save$/i })).toBeNull();
  });

  it("surfaces a registered page's dirty state and save action", async () => {
    const { onSave } = renderBar({ config: { dirty: true, isSaving: false } });
    expect(screen.getByText(/unsaved changes/i)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: /^save$/i }));
    expect(onSave).toHaveBeenCalledTimes(1);
  });

  it("disables save/discard when not dirty", () => {
    renderBar({ config: { dirty: false, isSaving: false } });
    expect(screen.getByRole("button", { name: /^save$/i })).toBeDisabled();
    expect(screen.getByRole("button", { name: /discard/i })).toBeDisabled();
  });

  it("hides the bar again when a page registers null", () => {
    function Toggle() {
      const [on, setOn] = useState(true);
      useRegisterDetailActionBar(
        on
          ? { dirty: true, isSaving: false, onSave() {}, onDiscard() {} }
          : null,
      );
      return (
        <button type="button" onClick={() => setOn(false)}>
          unmount-bar
        </button>
      );
    }
    render(
      <DetailActionBarProvider>
        <Toggle />
        <DetailActionBar />
      </DetailActionBarProvider>,
    );
    expect(screen.getByRole("button", { name: /^save$/i })).toBeInTheDocument();
  });
});
