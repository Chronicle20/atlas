import { useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import { DefinitionActionDialogs } from "@/components/features/socket/DefinitionActionDialogs";
import {
  DefinitionDrawer,
  type DrawerAction,
} from "@/components/features/socket/DefinitionDrawer";
import { GridLegend } from "@/components/features/socket/GridLegend";
import { GridToolbar } from "@/components/features/socket/GridToolbar";
import {
  PacketGrid,
  type GridSelection,
} from "@/components/features/socket/PacketGrid";
import { ErrorDisplay, LoadingSpinner } from "@/components/common";
import { useSocketMatrixTemplates } from "@/lib/hooks/api/useSocketObjects";
import { buildOpenInPath } from "@/lib/socket/routes";
import {
  buildRows,
  emptyFilters,
  filterRows,
  hasActiveFilters,
  sortRows,
  withOpcodeGaps,
  type GridFilters,
  type SortDirection,
  type SortKey,
} from "@/lib/socket/matrix";
import type { DefinitionKind, SocketObject } from "@/lib/socket/model";

/** (region, majorVersion, minorVersion) ascending - FR-2.4. */
function compareObjects(a: SocketObject, b: SocketObject): number {
  if (a.region !== b.region) return a.region.localeCompare(b.region);
  if (a.majorVersion !== b.majorVersion) return a.majorVersion - b.majorVersion;
  return a.minorVersion - b.minorVersion;
}

/** The highest (majorVersion, minorVersion) among the given objects - FR-2.10's default baseline. */
function highestVersionKey(objects: SocketObject[]): string {
  if (objects.length === 0) return "";
  return objects.reduce((best, o) => {
    if (o.majorVersion !== best.majorVersion) {
      return o.majorVersion > best.majorVersion ? o : best;
    }
    return o.minorVersion > best.minorVersion ? o : best;
  }).key;
}

const MODE_TO_KIND: Record<string, DefinitionKind> = {
  handlers: "handler",
  writers: "writer",
};
const KIND_TO_MODE: Record<DefinitionKind, string> = {
  handler: "handlers",
  writer: "writers",
};

/**
 * The all-templates matrix (FR-2.1-2.12). Templates only - tenants never
 * appear as columns here (FR-2.2). View state (mode, baseline, selected
 * columns, selected definition) lives in the URL via `useSearchParams`, so
 * the view is shareable and survives a reload (FR-12.1).
 */
export function PacketMatrixPage() {
  const templatesQuery = useSocketMatrixTemplates();
  const [searchParams, setSearchParams] = useSearchParams();
  const navigate = useNavigate();

  const setParam = (key: string, value: string) => {
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        next.set(key, value);
        return next;
      },
      { replace: true },
    );
  };
  const removeParam = (key: string) => {
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        next.delete(key);
        return next;
      },
      { replace: true },
    );
  };

  const kind: DefinitionKind =
    MODE_TO_KIND[searchParams.get("mode") ?? ""] ?? "handler";

  const allObjects = useMemo(
    () => [...(templatesQuery.data ?? [])].sort(compareObjects),
    [templatesQuery.data],
  );
  const allKeys = useMemo(() => allObjects.map((o) => o.key), [allObjects]);

  const colsParam = searchParams.get("cols");
  const selectedKeys = useMemo(() => {
    if (!colsParam) return allKeys;
    const requested = new Set(colsParam.split(",").filter(Boolean));
    const filtered = allKeys.filter((k) => requested.has(k));
    return filtered.length > 0 ? filtered : allKeys;
  }, [colsParam, allKeys]);

  const objects = useMemo(
    () => allObjects.filter((o) => selectedKeys.includes(o.key)),
    [allObjects, selectedKeys],
  );

  const baselineParam = searchParams.get("baseline");
  const baselineKey =
    baselineParam && objects.some((o) => o.key === baselineParam)
      ? baselineParam
      : highestVersionKey(objects);

  const [filters, setFilters] = useState<GridFilters>(emptyFilters());
  const [sort, setSort] = useState<{ key: SortKey; direction: SortDirection }>({
    key: "opcode",
    direction: "asc",
  });
  const [showFName, setShowFName] = useState(false);

  const rows = useMemo(
    () =>
      sortRows(
        filterRows(
          buildRows({ objects, kind, baselineKey }),
          filters,
          baselineKey,
        ),
        sort.key,
        sort.direction,
      ),
    [objects, kind, baselineKey, filters, sort],
  );

  // Blank rows for the opcodes nothing in view defines (FR-2 addendum). Only
  // in an opcode-ordered, unfiltered view: a gap has no name and no state, so
  // it can neither be placed in a name/state ordering nor survive a filter.
  const showsOpcodeGaps = sort.key === "opcode" && !hasActiveFilters(filters);
  const gridRows = useMemo(
    () =>
      showsOpcodeGaps
        ? withOpcodeGaps(rows, {
            objects,
            kind,
            baselineKey,
            direction: sort.direction,
          })
        : rows,
    [showsOpcodeGaps, rows, objects, kind, baselineKey, sort.direction],
  );

  const defParam = searchParams.get("def");
  const [selection, setSelection] = useState<GridSelection | null>(null);
  // Seeds `selection` from the `def` deep link once the templates have
  // loaded. Adjusted during render (React's "you might not need an Effect"
  // pattern, already used by CopyDefinitionDialog/DeleteDefinitionDialog in
  // this codebase) rather than in a useEffect, since this is deriving state
  // from a prop-like input on a one-time transition, not synchronizing with
  // an external system.
  const [seeded, setSeeded] = useState(false);
  if (!seeded && objects.length > 0) {
    setSeeded(true);
    if (defParam) setSelection({ name: defParam, scopeKey: baselineKey });
  }

  // The drawer is a modal Sheet: Radix marks the rest of the page
  // aria-hidden while it is open. A `def` deep link should highlight and
  // scroll to the row (drives `selection`, which PacketGrid uses for
  // `aria-selected` independent of the drawer) without forcing the modal
  // open on load - the drawer only opens from an explicit grid interaction.
  const [dialogAction, setDialogAction] = useState<DrawerAction | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);

  const handleGridSelect = (sel: GridSelection) => {
    setSelection(sel);
    setDrawerOpen(true);
    setParam("def", sel.name);
  };

  const handleDrawerClose = () => {
    setSelection(null);
    setDrawerOpen(false);
    removeParam("def");
  };

  const handleDrawerAction = (action: DrawerAction) => {
    if (action.type === "open-in") {
      const scope = objects.find((o) => o.key === action.scopeKey);
      if (scope) navigate(buildOpenInPath(scope, kind, action.name));
      return;
    }
    setDialogAction(action);
  };

  if (templatesQuery.isLoading) return <LoadingSpinner />;
  if (templatesQuery.error)
    return <ErrorDisplay error={templatesQuery.error} />;

  const selectedRow = selection
    ? (rows.find((r) => r.name === selection.name) ?? null)
    : null;

  return (
    <div className="flex flex-1 flex-col overflow-hidden space-y-6 p-10 pb-6">
      <div className="space-y-0.5">
        <h2 className="text-2xl font-bold tracking-tight">Packet Matrix</h2>
        <p className="text-muted-foreground">
          Handler and writer definitions across every Template.
        </p>
      </div>

      <div className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-md border">
        <GridToolbar
          kind={kind}
          onKindChange={(k) => setParam("mode", KIND_TO_MODE[k])}
          objects={allObjects}
          selectedKeys={selectedKeys}
          onSelectedKeysChange={(keys) => setParam("cols", keys.join(","))}
          baselineKey={baselineKey}
          onBaselineChange={(key) => setParam("baseline", key)}
          showFName={showFName}
          onShowFNameChange={setShowFName}
          filters={filters}
          onFiltersChange={setFilters}
          sort={sort}
          onSortChange={setSort}
        />
        <PacketGrid
          rows={gridRows}
          objects={objects}
          baselineKey={baselineKey}
          showFName={showFName}
          selection={selection}
          onSelect={handleGridSelect}
        />
        <GridLegend rowCount={rows.length} showsOpcodeGaps={showsOpcodeGaps} />
      </div>

      {drawerOpen && selection && selectedRow && (
        <DefinitionDrawer
          row={selectedRow}
          objects={objects}
          kind={kind}
          baselineKey={baselineKey}
          selection={selection}
          onClose={handleDrawerClose}
          onAction={handleDrawerAction}
        />
      )}

      <DefinitionActionDialogs
        kind={kind}
        objects={objects}
        action={dialogAction}
        onClose={() => setDialogAction(null)}
      />
    </div>
  );
}
