import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";

import { RouteStatePill } from "@/components/features/transports/RouteStatePill";
import type { RouteState } from "@/types/models/transport";

describe("RouteStatePill", () => {
  it.each<[RouteState, string]>([
    ["out_of_service", "Out of service"],
    ["in_transit", "In transit"],
    ["locked_entry", "Boarding closed"],
    ["open_entry", "Boarding"],
    ["awaiting_return", "Awaiting return"],
  ])("labels %s in text, never colour alone", (state, label) => {
    render(<RouteStatePill state={state} />);
    expect(screen.getByText(label)).toBeInTheDocument();
  });

  it("gives out_of_service a fault treatment distinct from the rest", () => {
    const { container: fault } = render(
      <RouteStatePill state="out_of_service" />,
    );
    const { container: normal } = render(<RouteStatePill state="open_entry" />);

    expect(fault.firstElementChild?.className).not.toBe(
      normal.firstElementChild?.className,
    );
  });
});
