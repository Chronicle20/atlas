import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { CellContext, ColumnDef } from "@tanstack/react-table";
import { describe, it, expect, vi } from "vitest";
import { getColumns } from "@/pages/tenants-columns";
import type { Tenant } from "@/types/models/tenant";

function makeTenant(id: string): Tenant {
  return {
    id,
    attributes: {
      name: `Tenant ${id}`,
      region: "GMS",
      majorVersion: 83,
      minorVersion: 1,
    },
  } as unknown as Tenant;
}

type ActionsProps = {
  onDelete?: (id: string) => void;
  onRename?: (id: string) => void;
};

function renderActionsCell(props: ActionsProps) {
  const columns = getColumns(props);
  const actions = columns.find((c: ColumnDef<Tenant>) => c.id === "actions");
  if (!actions || typeof actions.cell !== "function") {
    throw new Error("actions column missing");
  }

  const tenant = makeTenant("abc");
  const ctx = {
    row: {
      original: tenant,
      getValue: (key: string) =>
        (tenant as unknown as Record<string, unknown>)[key],
    },
    column: { id: "actions" },
  } as unknown as CellContext<Tenant, unknown>;

  const CellComponent = actions.cell;
  const node = CellComponent(ctx) as React.ReactNode;
  return render(<>{node}</>);
}

describe("tenants-columns actions menu", () => {
  it("renders Rename menu item when onRename is provided and invokes with correct id", async () => {
    const onRename = vi.fn();
    const onDelete = vi.fn();
    renderActionsCell({ onDelete, onRename });

    await userEvent.click(screen.getByRole("button", { name: /open menu/i }));
    const renameItem = await screen.findByText("Rename");
    await userEvent.click(renameItem);

    expect(onRename).toHaveBeenCalledWith("abc");
    expect(onDelete).not.toHaveBeenCalled();
  });

  it("omits Rename when onRename is not provided", async () => {
    const onDelete = vi.fn();
    renderActionsCell({ onDelete });

    await userEvent.click(screen.getByRole("button", { name: /open menu/i }));
    expect(await screen.findByText("Delete")).toBeInTheDocument();
    expect(screen.queryByText("Rename")).not.toBeInTheDocument();
  });
});
