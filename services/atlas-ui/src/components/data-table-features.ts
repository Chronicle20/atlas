import {
  columnSizingFeature,
  columnVisibilityFeature,
  rowSelectionFeature,
  tableFeatures,
  type CellData,
  type ColumnDef,
  type RowData,
} from "@tanstack/react-table";

/**
 * Feature set registered on every DataTable instance.
 *
 * v9 no longer bundles every feature automatically — each API the shared table
 * touches has to be listed here or it is simply absent at runtime:
 *
 * - `columnSizingFeature`     → `header.getSize()` / `column.getSize()`
 * - `columnVisibilityFeature` → `initialVisibilityState` / `row.getVisibleCells()`
 * - `rowSelectionFeature`     → `row.getIsSelected()`
 */
export const dataTableFeatures = tableFeatures({
  columnSizingFeature,
  columnVisibilityFeature,
  rowSelectionFeature,
});

export type DataTableFeatures = typeof dataTableFeatures;

/** Column definition bound to the shared DataTable feature set. */
export type DataTableColumnDef<
  TData extends RowData,
  TValue extends CellData = CellData,
> = ColumnDef<DataTableFeatures, TData, TValue>;
