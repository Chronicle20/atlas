import { render, screen, fireEvent } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ColumnDef } from "@tanstack/react-table";
import { DataTable } from "@/components/data-table";

type Row = { id: string; name: string };
const columns: ColumnDef<Row>[] = [{ accessorKey: "name", header: "Name" }];
const data: Row[] = [{ id: "1", name: "Alpha" }];

describe("DataTable refresh button", () => {
  it("spins and disables the button while refreshing", () => {
    render(
      <DataTable
        columns={columns}
        data={data}
        onRefresh={vi.fn()}
        isRefreshing
      />,
    );
    const button = screen.getByTitle("Refresh");
    expect(button).toBeDisabled();
    expect(button.querySelector("svg")).toHaveClass("animate-spin");
  });

  it("is enabled and not spinning when not refreshing, and clicking calls onRefresh", () => {
    const onRefresh = vi.fn();
    render(
      <DataTable
        columns={columns}
        data={data}
        onRefresh={onRefresh}
        isRefreshing={false}
      />,
    );
    const button = screen.getByTitle("Refresh");
    expect(button).not.toBeDisabled();
    expect(button.querySelector("svg")).not.toHaveClass("animate-spin");
    fireEvent.click(button);
    expect(onRefresh).toHaveBeenCalledTimes(1);
  });

  it("does not call onRefresh when disabled (refreshing) — no overlapping refetch", () => {
    const onRefresh = vi.fn();
    render(
      <DataTable
        columns={columns}
        data={data}
        onRefresh={onRefresh}
        isRefreshing
      />,
    );
    fireEvent.click(screen.getByTitle("Refresh"));
    expect(onRefresh).not.toHaveBeenCalled();
  });
});
