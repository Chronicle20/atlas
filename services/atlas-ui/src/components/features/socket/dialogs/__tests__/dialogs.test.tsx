import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

// Radix Select (used by CopyDefinitionDialog's source pickers) relies on DOM
// APIs jsdom does not implement - same polyfill GridToolbar.test.tsx uses.
Element.prototype.hasPointerCapture ||= () => false;
Element.prototype.scrollIntoView ||= () => {};

// vi.mock factories are hoisted above every import in this file, so the
// values they close over must be created via vi.hoisted rather than plain
// top-level `const` - a bare `const mutateAsync = vi.fn()` referenced from a
// hoisted factory hits the TDZ (ReferenceError: Cannot access before
// initialization) the moment any dialog module transitively imports sonner
// or useSocketObjects.
const { mutateAsync, toastError, toastSuccess } = vi.hoisted(() => ({
  mutateAsync: vi.fn(),
  toastError: vi.fn(),
  toastSuccess: vi.fn(),
}));

vi.mock("@/lib/hooks/api/useSocketObjects", () => ({
  useSocketMutation: () => ({ mutateAsync, isPending: false }),
  socketKeys: { all: ["socket"] },
}));

vi.mock("sonner", () => ({
  toast: { error: toastError, success: toastSuccess },
}));

import { AddDefinitionDialog } from "@/components/features/socket/dialogs/AddDefinitionDialog";
import { EditDefinitionDialog } from "@/components/features/socket/dialogs/EditDefinitionDialog";
import { DeleteDefinitionDialog } from "@/components/features/socket/dialogs/DeleteDefinitionDialog";
import { MarkUnsupportedDialog } from "@/components/features/socket/dialogs/MarkUnsupportedDialog";
import { CopyDefinitionDialog } from "@/components/features/socket/dialogs/CopyDefinitionDialog";
import { ResetToAncestorDialog } from "@/components/features/socket/dialogs/ResetToAncestorDialog";
import {
  addBinding,
  copyBindings,
  deleteBinding,
  editBinding,
  markUnsupported,
  MutationError,
} from "@/lib/socket/mutate";
import type { Binding, SocketObject } from "@/lib/socket/model";
import { parseOpcode } from "@/lib/socket/opcode";
import {
  definitionFormSchema,
  definitionFormSchemaFor,
} from "@/lib/schemas/socket-definition";
import type { SocketConfig } from "@/types/models/socket";

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

const target = { source: "template" as const, id: "t1" };

function config(): SocketConfig {
  return {
    handlers: [
      {
        opCode: "0x17",
        validator: "LoggedInValidator",
        handler: "NoOpHandler",
        services: ["channel"],
      },
      {
        opCode: "0x19",
        validator: "LoggedInValidator",
        handler: "NoOpHandler",
        services: ["channel"],
      },
    ],
    writers: [],
    unsupported: { handlers: [], writers: [] },
  };
}

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

function socketObject(
  key: string,
  label: string,
  handlers: Record<string, Binding[]> = {},
): SocketObject {
  return {
    key,
    label,
    source: "template",
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
    handlers: new Map(Object.entries(handlers)),
    writers: new Map(),
    unsupportedHandlers: new Set(),
    unsupportedWriters: new Set(),
  };
}

beforeEach(() => {
  mutateAsync.mockReset().mockResolvedValue(undefined);
  toastError.mockReset();
  toastSuccess.mockReset();
});

describe("AddDefinitionDialog", () => {
  it("rejects a malformed opcode without submitting", async () => {
    render(
      <AddDefinitionDialog
        open
        onOpenChange={vi.fn()}
        target={target}
        targetLabel="GMS v83.1"
        kind="handler"
      />,
      { wrapper },
    );
    await userEvent.type(
      screen.getByLabelText(/definition name/i),
      "PongHandle",
    );
    await userEvent.type(screen.getByLabelText(/operation code/i), "B8");
    await userEvent.click(
      screen.getByRole("button", { name: /^add definition$/i }),
    );
    expect(
      await screen.findByText(/0x followed by 1-4 hex digits/i),
    ).toBeInTheDocument();
    expect(mutateAsync).not.toHaveBeenCalled();
  });

  it("requires a validator for handlers", async () => {
    render(
      <AddDefinitionDialog
        open
        onOpenChange={vi.fn()}
        target={target}
        targetLabel="GMS v83.1"
        kind="handler"
      />,
      { wrapper },
    );
    await userEvent.type(
      screen.getByLabelText(/definition name/i),
      "PongHandle",
    );
    await userEvent.type(screen.getByLabelText(/operation code/i), "0x18");
    await userEvent.click(
      screen.getByRole("button", { name: /^add definition$/i }),
    );
    expect(
      await screen.findByText(/validator is required/i),
    ).toBeInTheDocument();
    expect(mutateAsync).not.toHaveBeenCalled();
  });

  it("submits a splice that adds the binding and clears the unsupported marker", async () => {
    render(
      <AddDefinitionDialog
        open
        onOpenChange={vi.fn()}
        target={target}
        targetLabel="GMS v83.1"
        kind="handler"
      />,
      { wrapper },
    );
    await userEvent.type(
      screen.getByLabelText(/definition name/i),
      "PongHandle",
    );
    await userEvent.type(screen.getByLabelText(/operation code/i), "0x18");
    await userEvent.type(screen.getByLabelText(/validator/i), "NoOpValidator");
    await userEvent.click(
      screen.getByRole("button", { name: /^add definition$/i }),
    );

    expect(mutateAsync).toHaveBeenCalledTimes(1);
    const { apply } = mutateAsync.mock.calls[0]![0] as {
      apply: (c: SocketConfig) => SocketConfig;
    };
    const before: SocketConfig = {
      ...config(),
      unsupported: { handlers: ["PongHandle"], writers: [] },
    };
    const after = apply(before);
    expect(after.handlers.some((h) => h.handler === "PongHandle")).toBe(true);
    expect(after.unsupported!.handlers).toEqual([]);
    // The pure function is the same one the unit tests cover.
    expect(after).toEqual(
      addBinding(before, "handler", "PongHandle", {
        opCode: "0x18",
        validator: "NoOpValidator",
        services: [],
      }),
    );
  });

  it("does not render a validator field for writers - writers carry no validator", () => {
    render(
      <AddDefinitionDialog
        open
        onOpenChange={vi.fn()}
        target={target}
        targetLabel="GMS v83.1"
        kind="writer"
      />,
      { wrapper },
    );
    expect(screen.queryByLabelText(/validator/i)).not.toBeInTheDocument();
  });
});

describe("EditDefinitionDialog", () => {
  it("renders the name read-only and submits an editBinding splice", async () => {
    render(
      <EditDefinitionDialog
        open
        onOpenChange={vi.fn()}
        target={target}
        targetLabel="GMS v83.1"
        kind="handler"
        name="NoOpHandler"
        opCodeValue={0x19}
        initial={{
          opCode: "0x19",
          validator: "LoggedInValidator",
          services: ["channel"],
        }}
      />,
      { wrapper },
    );

    const nameInput = screen.getByLabelText(
      /definition name/i,
    ) as HTMLInputElement;
    expect(nameInput).toBeDisabled();
    expect(nameInput.value).toBe("NoOpHandler");

    await userEvent.click(screen.getByRole("button", { name: /^save$/i }));

    expect(mutateAsync).toHaveBeenCalledTimes(1);
    const { apply } = mutateAsync.mock.calls[0]![0] as {
      apply: (c: SocketConfig) => SocketConfig;
    };
    expect(apply(config())).toEqual(
      editBinding(config(), "handler", "NoOpHandler", 0x19, {
        opCode: "0x19",
        validator: "LoggedInValidator",
        services: ["channel"],
      }),
    );
  });
});

describe("DeleteDefinitionDialog", () => {
  // Matches config()'s NoOpHandler bindings (0x17, 0x19), so tests that
  // assert against `apply(config())` stay consistent with the derived count.
  const twoBindingScope = socketObject("scope", "GMS v83.1", {
    NoOpHandler: [binding("0x17"), binding("0x19")],
  });

  // FR-6.3: two distinct outcomes, chosen explicitly.
  it("offers remove and remove-and-mark-unsupported as separate choices", () => {
    render(
      <DeleteDefinitionDialog
        open
        onOpenChange={vi.fn()}
        target={target}
        targetLabel="GMS v83.1"
        kind="handler"
        name="NoOpHandler"
        opCodeValue={0x19}
        scope={twoBindingScope}
      />,
      { wrapper },
    );
    expect(
      screen.getByRole("radio", { name: /remove this binding/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("radio", { name: /undefine and mark unsupported/i }),
    ).toBeInTheDocument();
  });

  it("names the opcode being deleted", () => {
    render(
      <DeleteDefinitionDialog
        open
        onOpenChange={vi.fn()}
        target={target}
        targetLabel="GMS v83.1"
        kind="handler"
        name="NoOpHandler"
        opCodeValue={0x19}
        scope={twoBindingScope}
      />,
      { wrapper },
    );
    expect(screen.getByRole("heading", { name: /0x19/ })).toBeInTheDocument();
  });

  it("removes exactly the addressed binding by default", async () => {
    render(
      <DeleteDefinitionDialog
        open
        onOpenChange={vi.fn()}
        target={target}
        targetLabel="GMS v83.1"
        kind="handler"
        name="NoOpHandler"
        opCodeValue={0x19}
        scope={twoBindingScope}
      />,
      { wrapper },
    );
    await userEvent.click(
      screen.getByRole("button", { name: /^remove binding$/i }),
    );
    const { apply } = mutateAsync.mock.calls[0]![0] as {
      apply: (c: SocketConfig) => SocketConfig;
    };
    expect(apply(config())).toEqual(
      deleteBinding(config(), "handler", "NoOpHandler", 0x19),
    );
  });

  it("warns that marking unsupported removes every binding of the name - count derived from scope, not caller-supplied", async () => {
    render(
      <DeleteDefinitionDialog
        open
        onOpenChange={vi.fn()}
        target={target}
        targetLabel="GMS v83.1"
        kind="handler"
        name="NoOpHandler"
        opCodeValue={0x19}
        scope={twoBindingScope}
      />,
      { wrapper },
    );
    await userEvent.click(
      screen.getByRole("radio", { name: /undefine and mark unsupported/i }),
    );
    expect(screen.getByText(/all 2 bindings/i)).toBeInTheDocument();
  });

  // Regression coverage for the hazard a caller-supplied `bindingCount` prop
  // would reintroduce: a four-binding definition must never be understated
  // as "this binding" (singular) just because a caller forgot to pass a
  // count. This would fail against a version of the dialog whose count
  // defaults to 1 when not derived from `scope`.
  it("names all four bindings when 'remove and mark unsupported' is chosen for a four-binding definition", async () => {
    const fourBindingScope = socketObject("scope", "GMS v83.1", {
      NoOpHandler: [
        binding("0x11"),
        binding("0x17"),
        binding("0x19"),
        binding("0x22"),
      ],
    });
    render(
      <DeleteDefinitionDialog
        open
        onOpenChange={vi.fn()}
        target={target}
        targetLabel="GMS v83.1"
        kind="handler"
        name="NoOpHandler"
        opCodeValue={0x19}
        scope={fourBindingScope}
      />,
      { wrapper },
    );
    await userEvent.click(
      screen.getByRole("radio", { name: /undefine and mark unsupported/i }),
    );
    expect(screen.getByText(/all 4 bindings/i)).toBeInTheDocument();
  });

  it("switches to markUnsupported when 'remove and mark unsupported' is chosen", async () => {
    render(
      <DeleteDefinitionDialog
        open
        onOpenChange={vi.fn()}
        target={target}
        targetLabel="GMS v83.1"
        kind="handler"
        name="NoOpHandler"
        opCodeValue={0x19}
        scope={twoBindingScope}
      />,
      { wrapper },
    );
    await userEvent.click(
      screen.getByRole("radio", { name: /undefine and mark unsupported/i }),
    );
    await userEvent.click(
      screen.getByRole("button", { name: /undefine and mark unsupported/i }),
    );
    const { apply } = mutateAsync.mock.calls[0]![0] as {
      apply: (c: SocketConfig) => SocketConfig;
    };
    expect(apply(config())).toEqual(
      markUnsupported(config(), "handler", "NoOpHandler"),
    );
  });
});

describe("MarkUnsupportedDialog", () => {
  // FR-6.4 - it must SAY it removes the existing definitions, plural.
  it("names the target version and states that existing bindings will be removed", () => {
    render(
      <MarkUnsupportedDialog
        open
        onOpenChange={vi.fn()}
        target={target}
        targetLabel="GMS v83.1"
        kind="handler"
        name="NoOpHandler"
        bindingCount={4}
      />,
      { wrapper },
    );
    expect(screen.getByText(/GMS v83\.1/)).toBeInTheDocument();
    expect(
      screen.getByText(/all 4 bindings.*will be removed/i),
    ).toBeInTheDocument();
  });

  it("applies markUnsupported on confirm", async () => {
    render(
      <MarkUnsupportedDialog
        open
        onOpenChange={vi.fn()}
        target={target}
        targetLabel="GMS v83.1"
        kind="handler"
        name="NoOpHandler"
        bindingCount={2}
      />,
      { wrapper },
    );
    await userEvent.click(
      screen.getByRole("button", { name: /mark unsupported/i }),
    );
    const { apply } = mutateAsync.mock.calls[0]![0] as {
      apply: (c: SocketConfig) => SocketConfig;
    };
    expect(apply(config())).toEqual(
      markUnsupported(config(), "handler", "NoOpHandler"),
    );
  });
});

describe("CopyDefinitionDialog", () => {
  it("loads the source binding and submits a copyBindings splice, never calling a service", async () => {
    const source = socketObject("src", "GMS v79.1", {
      PongHandle: [
        binding("0x30", { validator: "SourceValidator", services: [] }),
      ],
    });
    render(
      <CopyDefinitionDialog
        open
        onOpenChange={vi.fn()}
        target={target}
        targetLabel="GMS v83.1"
        kind="handler"
        name="PongHandle"
        sourceObjects={[source]}
      />,
      { wrapper },
    );

    await userEvent.click(screen.getByRole("button", { name: /^copy$/i }));

    expect(mutateAsync).toHaveBeenCalledTimes(1);
    const call = mutateAsync.mock.calls[0]![0] as {
      target: typeof target;
      apply: (c: SocketConfig) => SocketConfig;
    };
    expect(call.target).toEqual(target);
    expect(call.apply(config())).toEqual(
      copyBindings(config(), "handler", "PongHandle", [
        { opCode: "0x30", validator: "SourceValidator", services: [] },
      ]),
    );
  });
});

describe("ResetToAncestorDialog", () => {
  it("removes the tenant's own bindings and copies the ancestor's, never calling a service", async () => {
    const tenant = socketObject("tenant-obj", "Tenant A", {
      NoOpHandler: [binding("0x19")],
    });
    const ancestor = socketObject("anc", "GMS v83.1", {
      NoOpHandler: [
        binding("0x30", { validator: "AncestorValidator", services: [] }),
      ],
    });
    render(
      <ResetToAncestorDialog
        open
        onOpenChange={vi.fn()}
        target={target}
        targetLabel="Tenant A"
        kind="handler"
        name="NoOpHandler"
        tenant={tenant}
        ancestor={ancestor}
      />,
      { wrapper },
    );

    await userEvent.click(
      screen.getByRole("button", { name: /reset to ancestor/i }),
    );

    expect(mutateAsync).toHaveBeenCalledTimes(1);
    const { apply } = mutateAsync.mock.calls[0]![0] as {
      apply: (c: SocketConfig) => SocketConfig;
    };
    // Both of the tenant's own NoOpHandler bindings (0x17, 0x19) are gone,
    // replaced by the ancestor's single 0x30 binding.
    const after = apply(config());
    expect(after.handlers.filter((h) => h.handler === "NoOpHandler")).toEqual([
      {
        opCode: "0x30",
        validator: "AncestorValidator",
        handler: "NoOpHandler",
      },
    ]);
  });
});

describe("definitionFormSchema", () => {
  const base = {
    name: "PongHandle",
    services: [] as string[],
    validator: "LoggedInValidator",
  };

  it("accepts 0x9 and 0x0A5 - both present in the corpus", () => {
    expect(
      definitionFormSchema.safeParse({ ...base, opCode: "0x9" }).success,
    ).toBe(true);
    expect(
      definitionFormSchema.safeParse({ ...base, opCode: "0x0A5" }).success,
    ).toBe(true);
  });

  it("rejects a 5-digit opcode", () => {
    expect(
      definitionFormSchema.safeParse({ ...base, opCode: "0x12345" }).success,
    ).toBe(false);
  });

  it("rejects a non-hex opcode", () => {
    expect(
      definitionFormSchema.safeParse({ ...base, opCode: "banana" }).success,
    ).toBe(false);
  });

  it("accepts an empty services list - legal, and occurs 25 times in the corpus", () => {
    expect(
      definitionFormSchema.safeParse({ ...base, opCode: "0x18", services: [] })
        .success,
    ).toBe(true);
  });

  it("does not enumerate validators - an arbitrary validator string is accepted", () => {
    const result = definitionFormSchemaFor("handler").safeParse({
      ...base,
      opCode: "0x18",
      validator: "SomeCustomValidator",
    });
    expect(result.success).toBe(true);
  });

  it("requires a non-blank validator for handlers", () => {
    const result = definitionFormSchemaFor("handler").safeParse({
      ...base,
      opCode: "0x18",
      validator: "",
    });
    expect(result.success).toBe(false);
  });

  it("does not require a validator for writers - writers carry no validator field", () => {
    const result = definitionFormSchemaFor("writer").safeParse({
      ...base,
      opCode: "0x18",
      validator: "",
    });
    expect(result.success).toBe(true);
  });
});

describe("MutationError surfacing", () => {
  it("surfaces a MutationError's message verbatim rather than a generic failure", async () => {
    mutateAsync.mockRejectedValueOnce(
      new MutationError(
        'No handler named "NoOpHandler" at opcode 0x19 was found. It may have been changed or removed by another session - reload and try again.',
      ),
    );
    render(
      <DeleteDefinitionDialog
        open
        onOpenChange={vi.fn()}
        target={target}
        targetLabel="GMS v83.1"
        kind="handler"
        name="NoOpHandler"
        opCodeValue={0x19}
        scope={socketObject("scope", "GMS v83.1", {
          NoOpHandler: [binding("0x19")],
        })}
      />,
      { wrapper },
    );
    await userEvent.click(screen.getByRole("button", { name: /^undefine$/i }));
    await waitFor(() =>
      expect(toastError).toHaveBeenCalledWith(
        expect.stringMatching(
          /may have been changed or removed by another session/i,
        ),
      ),
    );
  });
});
