import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { BaselineTargetPicker } from "@/components/features/baselines/BaselineTargetPicker";
import {
  dedupeSelections,
  parseCustomSelection,
  selectionKey,
} from "@/components/features/baselines/BaselineTargetPicker.utils";

vi.mock("@/lib/hooks/api/useTemplates", () => ({
  useTemplates: () => ({
    data: [
      {
        id: "t1",
        attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
      },
      {
        id: "t2",
        attributes: { region: "JMS", majorVersion: 185, minorVersion: 1 },
      },
    ],
  }),
}));

vi.mock("@/lib/hooks/api/useTenants", () => ({
  useTenants: () => ({
    data: [
      // Duplicate of the GMS template combo — must dedupe.
      {
        id: "x1",
        attributes: {
          name: "gms",
          region: "GMS",
          majorVersion: 83,
          minorVersion: 1,
        },
      },
      {
        id: "x2",
        attributes: {
          name: "v87",
          region: "GMS",
          majorVersion: 87,
          minorVersion: 1,
        },
      },
    ],
  }),
}));

describe("selectionKey", () => {
  it("formats region/major.minor", () => {
    expect(
      selectionKey({ region: "GMS", majorVersion: 83, minorVersion: 1 }),
    ).toBe("GMS/83.1");
  });
});

describe("dedupeSelections", () => {
  it("unions templates and tenants, dedupes, and sorts", () => {
    const out = dedupeSelections(
      [
        { attributes: { region: "JMS", majorVersion: 185, minorVersion: 1 } },
        { attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } },
      ],
      [
        { attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } },
        { attributes: { region: "GMS", majorVersion: 87, minorVersion: 1 } },
      ],
    );
    expect(out.map(selectionKey)).toEqual([
      "GMS/83.1",
      "GMS/87.1",
      "JMS/185.1",
    ]);
  });

  it("works with zero tenants (templates only)", () => {
    const out = dedupeSelections(
      [{ attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } }],
      [],
    );
    expect(out.map(selectionKey)).toEqual(["GMS/83.1"]);
  });
});

describe("parseCustomSelection", () => {
  it("accepts a valid custom entry", () => {
    expect(parseCustomSelection("GMS", "92", "1")).toEqual({
      region: "GMS",
      majorVersion: 92,
      minorVersion: 1,
    });
  });
  it.each([
    ["", "92", "1"],
    ["  ", "92", "1"],
    ["GMS", "", "1"],
    ["GMS", "-1", "1"],
    ["GMS", "9.5", "1"],
    ["GMS", "abc", "1"],
    ["GMS", "92", "-2"],
    ["GMS", "92", ""],
  ])("rejects region=%j major=%j minor=%j", (region, major, minor) => {
    expect(parseCustomSelection(region, major, minor)).toBeNull();
  });
});

describe("BaselineTargetPicker render", () => {
  it("renders the trigger with a placeholder when nothing is selected", () => {
    render(<BaselineTargetPicker value={null} onChange={() => {}} />);
    expect(screen.getByText(/select region and version/i)).toBeInTheDocument();
  });
});
