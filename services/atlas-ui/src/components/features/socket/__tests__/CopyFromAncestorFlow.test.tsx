import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

// FillMissingValidatorsDialog renders a Radix Select - same polyfill
// GridToolbar.test.tsx / dialogs.test.tsx use for jsdom's missing DOM APIs.
Element.prototype.hasPointerCapture ||= () => false;
Element.prototype.scrollIntoView ||= () => {};

const mutateAsync = vi.fn();
vi.mock("@/lib/hooks/api/useSocketObjects", () => ({
  useSocketMutation: () => ({ mutateAsync, isPending: false }),
  socketKeys: { all: ["socket"] },
}));

import { CopyFromAncestorFlow } from "@/components/features/socket/CopyFromAncestorFlow";
import { FillMissingValidatorsDialog } from "@/components/features/socket/FillMissingValidatorsDialog";
import { fillMissingValidators } from "@/lib/socket/mutate";
import type { Binding, SocketObject } from "@/lib/socket/model";
import { parseOpcode } from "@/lib/socket/opcode";
import type { SocketConfig } from "@/types/models/socket";

function binding(opCode: string, extra: Partial<Binding> = {}): Binding {
  return {
    opCode,
    opCodeValue: parseOpcode(opCode),
    validator: "LoggedInValidator",
    services: ["channel"],
    index: 0,
    ...extra,
  };
}

function obj(
  key: string,
  source: SocketObject["source"],
  handlers: Record<string, Binding[]>,
  unsupportedHandlers: string[] = [],
): SocketObject {
  return {
    key,
    label: key === "tnt" ? "Tenant" : "GMS v83.1",
    source,
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
    handlers: new Map(Object.entries(handlers)),
    writers: new Map(),
    unsupportedHandlers: new Set(unsupportedHandlers),
    unsupportedWriters: new Set(),
  };
}

const tenant = obj("tnt", "tenant", { LoginHandle: [binding("0x01")] }, [
  "GuestLoginHandle",
]);
const ancestor = obj("t83", "template", {
  LoginHandle: [binding("0x01")],
  PongHandle: [binding("0x18")],
  MoveHandle: [binding("0x29", { options: { types: ["WALK"] } })],
  GuestLoginHandle: [binding("0x02")],
});

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

function renderFlow() {
  render(
    <CopyFromAncestorFlow
      open
      onOpenChange={vi.fn()}
      tenant={tenant}
      ancestor={ancestor}
      kind="handler"
      target={{ source: "tenant", id: "tnt" }}
    />,
    { wrapper },
  );
}

beforeEach(() => mutateAsync.mockReset().mockResolvedValue(undefined));

describe("CopyFromAncestorFlow", () => {
  // FR-9.1 + FR-9.5
  it("lists only definitions defined in the ancestor and undefined in the tenant", () => {
    renderFlow();
    const list = screen.getByRole("group", { name: /candidates/i });
    expect(
      within(list).getByRole("checkbox", { name: /PongHandle/ }),
    ).toBeInTheDocument();
    expect(
      within(list).getByRole("checkbox", { name: /MoveHandle/ }),
    ).toBeInTheDocument();
    // Already defined in the tenant.
    expect(
      within(list).queryByRole("checkbox", { name: /LoginHandle/ }),
    ).not.toBeInTheDocument();
    // Explicitly marked Unsupported in the tenant.
    expect(
      within(list).queryByRole("checkbox", { name: /GuestLoginHandle/ }),
    ).not.toBeInTheDocument();
  });

  // FR-9.3
  it("shows name, source opcode, target opcode, validator, services and option differences in review", async () => {
    renderFlow();
    await userEvent.click(screen.getByRole("checkbox", { name: /MoveHandle/ }));
    await userEvent.click(screen.getByRole("button", { name: /review/i }));

    const review = screen.getByRole("region", { name: /review/i });
    expect(within(review).getByText("MoveHandle")).toBeInTheDocument();
    expect(within(review).getByText(/source opcode/i)).toBeInTheDocument();
    expect(
      within(review).getByLabelText(/target opcode for MoveHandle/i),
    ).toHaveValue("0x29");
    expect(within(review).getByText("LoggedInValidator")).toBeInTheDocument();
    expect(within(review).getByText("channel")).toBeInTheDocument();
    expect(within(review).getByText(/types/)).toBeInTheDocument();
    expect(within(review).getByText(/undefined/i)).toBeInTheDocument();
  });

  it("lets the target opcode be adjusted before applying", async () => {
    renderFlow();
    await userEvent.click(screen.getByRole("checkbox", { name: /PongHandle/ }));
    await userEvent.click(screen.getByRole("button", { name: /review/i }));
    const field = screen.getByLabelText(/target opcode for PongHandle/i);
    await userEvent.clear(field);
    await userEvent.type(field, "0x1A");
    await userEvent.click(screen.getByRole("button", { name: /apply/i }));

    const { apply } = mutateAsync.mock.calls[0]![0] as {
      apply: (c: SocketConfig) => SocketConfig;
    };
    const out = apply({
      handlers: [],
      writers: [],
      unsupported: { handlers: [], writers: [] },
    });
    expect(out.handlers.find((h) => h.handler === "PongHandle")!.opCode).toBe(
      "0x1A",
    );
  });

  // FR-9.6
  it("applies the whole selection as a single write", async () => {
    renderFlow();
    await userEvent.click(screen.getByRole("checkbox", { name: /PongHandle/ }));
    await userEvent.click(screen.getByRole("checkbox", { name: /MoveHandle/ }));
    await userEvent.click(screen.getByRole("button", { name: /review/i }));
    await userEvent.click(screen.getByRole("button", { name: /apply/i }));

    expect(mutateAsync).toHaveBeenCalledTimes(1);
    const { apply } = mutateAsync.mock.calls[0]![0] as {
      apply: (c: SocketConfig) => SocketConfig;
    };
    const out = apply({
      handlers: [],
      writers: [],
      unsupported: { handlers: [], writers: [] },
    });
    expect(out.handlers.map((h) => h.handler).sort()).toEqual([
      "MoveHandle",
      "PongHandle",
    ]);
  });

  // FR-9.4
  it("never overwrites a definition the tenant gained since the scan", async () => {
    renderFlow();
    await userEvent.click(screen.getByRole("checkbox", { name: /PongHandle/ }));
    await userEvent.click(screen.getByRole("button", { name: /review/i }));
    await userEvent.click(screen.getByRole("button", { name: /apply/i }));

    const { apply } = mutateAsync.mock.calls[0]![0] as {
      apply: (c: SocketConfig) => SocketConfig;
    };
    const raced: SocketConfig = {
      handlers: [
        {
          opCode: "0xFF",
          validator: "NoOpValidator",
          handler: "PongHandle",
          services: ["channel"],
        },
      ],
      writers: [],
      unsupported: { handlers: [], writers: [] },
    };
    const out = apply(raced);
    const pong = out.handlers.filter((h) => h.handler === "PongHandle");
    expect(pong).toHaveLength(1);
    expect(pong[0]!.opCode).toBe("0xFF");
  });

  it("disables Review until at least one candidate is selected", () => {
    renderFlow();
    expect(screen.getByRole("button", { name: /review/i })).toBeDisabled();
  });

  it("goes through useSocketMutation only - never a service", async () => {
    renderFlow();
    await userEvent.click(screen.getByRole("checkbox", { name: /PongHandle/ }));
    await userEvent.click(screen.getByRole("button", { name: /review/i }));
    await userEvent.click(screen.getByRole("button", { name: /apply/i }));
    expect(mutateAsync).toHaveBeenCalledTimes(1);
    expect(mutateAsync.mock.calls[0]![0]).toMatchObject({
      target: { source: "tenant", id: "tnt" },
    });
  });
});

describe("FillMissingValidatorsDialog", () => {
  it("states how many entries it will repair and why one at a time will not work", () => {
    render(
      <FillMissingValidatorsDialog
        open
        onOpenChange={vi.fn()}
        target={{ source: "tenant", id: "tnt" }}
        targetLabel="GMS v95.1 tenant"
        emptyValidatorCount={32}
      />,
      { wrapper },
    );
    expect(screen.getByText(/32 handler/i)).toBeInTheDocument();
    expect(screen.getByText(/single configuration write/i)).toBeInTheDocument();
  });

  it("applies fillMissingValidators with the chosen validator in one write", async () => {
    render(
      <FillMissingValidatorsDialog
        open
        onOpenChange={vi.fn()}
        target={{ source: "tenant", id: "tnt" }}
        targetLabel="GMS v95.1 tenant"
        emptyValidatorCount={2}
      />,
      { wrapper },
    );
    await userEvent.click(
      screen.getByRole("button", { name: /fill validators/i }),
    );

    expect(mutateAsync).toHaveBeenCalledTimes(1);
    const { apply } = mutateAsync.mock.calls[0]![0] as {
      apply: (c: SocketConfig) => SocketConfig;
    };
    const broken: SocketConfig = {
      handlers: [
        { opCode: "0x01", validator: "", handler: "A", services: ["channel"] },
        { opCode: "0x02", validator: "", handler: "B", services: ["channel"] },
      ],
      writers: [],
    };
    expect(apply(broken)).toEqual(
      fillMissingValidators(broken, "NoOpValidator"),
    );
  });

  // The banner (and this dialog) must never surface itself for a document
  // with nothing to repair, regardless of whether a caller passes `open`.
  it("renders nothing when the empty-validator count is zero", () => {
    render(
      <FillMissingValidatorsDialog
        open
        onOpenChange={vi.fn()}
        target={{ source: "tenant", id: "tnt" }}
        targetLabel="GMS v95.1 tenant"
        emptyValidatorCount={0}
      />,
      { wrapper },
    );
    expect(
      screen.queryByText(/handler entries with no validator/i),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /fill validators/i }),
    ).not.toBeInTheDocument();
  });

  it("goes through useSocketMutation only - never a service", async () => {
    render(
      <FillMissingValidatorsDialog
        open
        onOpenChange={vi.fn()}
        target={{ source: "template", id: "t95" }}
        targetLabel="GMS v95.1 template"
        emptyValidatorCount={1}
      />,
      { wrapper },
    );
    await userEvent.click(
      screen.getByRole("button", { name: /fill validators/i }),
    );
    expect(mutateAsync).toHaveBeenCalledTimes(1);
    expect(mutateAsync.mock.calls[0]![0]).toMatchObject({
      target: { source: "template", id: "t95" },
    });
  });

  // Task 12's exact bug: fillMissingValidators threw on a validator key that
  // is ABSENT entirely or explicitly `null`, neither of which any seed-corpus
  // fixture can produce (the corpus has zero such entries - the 32 live in a
  // gms_95 TENANT, not seed data). Built by hand for that reason, mirroring
  // lib/socket/__tests__/mutate.test.ts's own hand-built regression fixture.
  it("counts and repairs handler entries whose validator is absent, null, or empty - and none throws", async () => {
    render(
      <FillMissingValidatorsDialog
        open
        onOpenChange={vi.fn()}
        target={{ source: "tenant", id: "gms95" }}
        targetLabel="GMS v95.1 tenant"
        emptyValidatorCount={3}
      />,
      { wrapper },
    );
    await userEvent.click(
      screen.getByRole("button", { name: /fill validators/i }),
    );

    const { apply } = mutateAsync.mock.calls[0]![0] as {
      apply: (c: SocketConfig) => SocketConfig;
    };
    const broken = {
      handlers: [
        // validator key absent entirely
        { opCode: "0x01", handler: "A", services: ["channel"] },
        // validator explicitly null
        {
          opCode: "0x02",
          handler: "B",
          services: ["channel"],
          validator: null,
        },
        // validator the empty string
        { opCode: "0x03", handler: "C", services: ["channel"], validator: "" },
        // untouched: already has a real validator
        {
          opCode: "0x04",
          handler: "D",
          services: ["channel"],
          validator: "LoggedInValidator",
        },
      ],
      writers: [],
    } as unknown as SocketConfig;

    expect(() => apply(broken)).not.toThrow();
    const out = apply(broken);
    expect(out.handlers.map((h) => h.validator)).toEqual([
      "NoOpValidator",
      "NoOpValidator",
      "NoOpValidator",
      "LoggedInValidator",
    ]);
  });
});
