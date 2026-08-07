// services/atlas-ui/src/components/ui/__tests__/tooltip.test.tsx
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter, Link, Routes, Route } from "react-router-dom";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "../tooltip";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

function LinkTriggeredTooltip() {
  return (
    <MemoryRouter initialEntries={["/maps"]}>
      <Routes>
        <Route
          path="/maps"
          element={
            // Mirrors the map cells in MapDetailTabs/ConnectedMapsRow: the
            // Link wraps the whole tooltip, so it is a React-tree ancestor of
            // the portaled content.
            <Link to="/maps/100000000">
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span>Henesys</span>
                  </TooltipTrigger>
                  <TooltipContent copyable>
                    <p>100000000</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </Link>
          }
        />
        <Route path="/maps/:mapId" element={<div>map detail</div>} />
      </Routes>
    </MemoryRouter>
  );
}

describe("TooltipContent copyable", () => {
  const writeText = vi.fn().mockResolvedValue(undefined);

  async function openTooltip() {
    const user = userEvent.setup();
    // userEvent.setup() installs its own navigator.clipboard stub, so ours has
    // to land after it.
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText },
      configurable: true,
    });
    render(<LinkTriggeredTooltip />);
    await user.hover(screen.getByText("Henesys"));
    return user;
  }

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("copies the id without navigating when the tooltip is wrapped in a link", async () => {
    const user = await openTooltip();

    // Radix renders the content twice — visible, plus a visually-hidden copy
    // for screen readers. The first match is the visible one.
    const [copyButton] = await screen.findAllByRole("button", {
      name: /copy to clipboard/i,
    });
    await user.click(copyButton!);

    await waitFor(() => expect(writeText).toHaveBeenCalledWith("100000000"));
    expect(screen.queryByText("map detail")).toBeNull();
  });

  it("does not navigate when the tooltip text itself is clicked", async () => {
    const user = await openTooltip();

    const [idText] = await screen.findAllByText("100000000");
    await user.click(idText!);

    expect(screen.queryByText("map detail")).toBeNull();
    expect(writeText).not.toHaveBeenCalled();
  });
});
