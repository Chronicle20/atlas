import { useMemo, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import { toast } from "sonner";

import { DefinitionActionDialogs } from "@/components/features/socket/DefinitionActionDialogs";
import { CopyFromAncestorFlow } from "@/components/features/socket/CopyFromAncestorFlow";
import {
  DefinitionDrawer,
  type DrawerAction,
} from "@/components/features/socket/DefinitionDrawer";
import { FillMissingValidatorsDialog } from "@/components/features/socket/FillMissingValidatorsDialog";
import { GridLegend } from "@/components/features/socket/GridLegend";
import { GridToolbar } from "@/components/features/socket/GridToolbar";
import {
  PacketGrid,
  type GridSelection,
} from "@/components/features/socket/PacketGrid";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { ErrorDisplay, LoadingSpinner } from "@/components/common";
import { useSocketMatrixTemplates } from "@/lib/hooks/api/useSocketObjects";
import { useTemplate } from "@/lib/hooks/api/useTemplates";
import { useTenantConfiguration } from "@/lib/hooks/api/useTenants";
import {
  classifyAgainstAncestor,
  inferAncestor,
  type AncestryClass,
} from "@/lib/socket/ancestry";
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
import type { DefinitionKind } from "@/lib/socket/model";
import { fromTemplate, fromTenantConfig } from "@/lib/socket/normalize";
import { buildOpenInPath } from "@/lib/socket/routes";

export interface DefinitionGridPageProps {
  kind: DefinitionKind;
  scope: "template" | "tenant";
}

/**
 * The shared per-object page shell the four routes
 * (`/templates/:id/handlers`, `/templates/:id/writers`,
 * `/tenants/:id/handlers`, `/tenants/:id/writers`) render (FR-7.1). Renders
 * the same `PacketGrid` locked to one object - the mode switch, column
 * picker and baseline selector are absent (FR-7.3), not disabled, because
 * they have no meaning on a page locked to a single object.
 */
export function DefinitionGridPage({ kind, scope }: DefinitionGridPageProps) {
  const { id } = useParams();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();

  const templateQuery = useTemplate(scope === "template" ? (id ?? "") : "");
  const tenantQuery = useTenantConfiguration(
    scope === "tenant" ? (id ?? "") : "",
  );
  // The sparse matrix read, used only to infer a Tenant's ancestor. Always
  // called (Rules of Hooks) - unused on Template pages.
  const templatesForAncestor = useSocketMatrixTemplates();

  const isLoading =
    scope === "template" ? templateQuery.isLoading : tenantQuery.isLoading;
  const error = scope === "template" ? templateQuery.error : tenantQuery.error;

  const object = useMemo(() => {
    if (scope === "template") {
      return templateQuery.data ? fromTemplate(templateQuery.data) : null;
    }
    return tenantQuery.data ? fromTenantConfig(tenantQuery.data) : null;
  }, [scope, templateQuery.data, tenantQuery.data]);

  // FR-8.1/8.2: inferred by exact (region, majorVersion, minorVersion) match.
  // null when there is no match - the page then renders a single column with
  // every ancestry affordance absent, never disabled.
  const ancestor = useMemo(() => {
    if (scope !== "tenant" || !object || !templatesForAncestor.data)
      return null;
    return inferAncestor(object, templatesForAncestor.data);
  }, [scope, object, templatesForAncestor.data]);

  const objects = useMemo(() => {
    if (!object) return [];
    return ancestor ? [object, ancestor] : [object];
  }, [object, ancestor]);

  const baselineKey = object?.key ?? "";

  const [filters, setFilters] = useState<GridFilters>(() => {
    const def = searchParams.get("def");
    return def ? { ...emptyFilters(), query: def } : emptyFilters();
  });
  const [sort, setSort] = useState<{ key: SortKey; direction: SortDirection }>({
    key: "opcode",
    direction: "asc",
  });
  const [showFName, setShowFName] = useState(false);
  const [ancestryClasses, setAncestryClasses] = useState<AncestryClass[]>([]);

  const rows = useMemo(() => {
    const built = filterRows(
      buildRows({ objects, kind, baselineKey }),
      filters,
      baselineKey,
    );
    const withAncestry =
      object && ancestor && ancestryClasses.length > 0
        ? built.filter((row) =>
            ancestryClasses.includes(
              classifyAgainstAncestor(object, ancestor, kind, row.name),
            ),
          )
        : built;
    return sortRows(withAncestry, sort.key, sort.direction);
  }, [
    objects,
    kind,
    baselineKey,
    filters,
    sort,
    object,
    ancestor,
    ancestryClasses,
  ]);

  // Same gating as the matrix page: opcode order, and no filter narrowing the
  // set - the ancestry filter counts, since a blank row belongs to no
  // ancestry class.
  const showsOpcodeGaps =
    sort.key === "opcode" &&
    !hasActiveFilters(filters) &&
    ancestryClasses.length === 0;
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
  // Seeds `selection` from the `def` deep link once the object has loaded.
  // Adjusted during render (React's "you might not need an Effect" pattern,
  // already used by CopyDefinitionDialog/DeleteDefinitionDialog in this
  // codebase) rather than in a useEffect, since this is deriving state from
  // a prop-like input on a one-time transition, not synchronizing with an
  // external system.
  const [seeded, setSeeded] = useState(false);
  if (!seeded && object) {
    setSeeded(true);
    if (defParam) setSelection({ name: defParam, scopeKey: object.key });
  }

  // The drawer is a modal Sheet: Radix marks the rest of the page
  // aria-hidden while it is open. A `def` deep link highlights and filters
  // to the row (drives `selection`, independent of the drawer) without
  // forcing the modal open on load - the drawer only opens from an explicit
  // grid interaction.
  const [dialogAction, setDialogAction] = useState<DrawerAction | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [copyFlowOpen, setCopyFlowOpen] = useState(false);
  const [fillDialogOpen, setFillDialogOpen] = useState(false);

  const setDefParam = (name: string | null) => {
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        if (name) next.set("def", name);
        else next.delete("def");
        return next;
      },
      { replace: true },
    );
  };

  const handleGridSelect = (sel: GridSelection) => {
    setSelection(sel);
    setDrawerOpen(true);
    setDefParam(sel.name);
  };

  const handleDrawerClose = () => {
    setSelection(null);
    setDrawerOpen(false);
    setDefParam(null);
  };

  const handleDrawerAction = (action: DrawerAction) => {
    if (action.type === "open-in") {
      const target = objects.find((o) => o.key === action.scopeKey);
      if (target) navigate(buildOpenInPath(target, kind, action.name));
      return;
    }
    // The ancestor column is read-only (FR-7.2): mutating actions are only
    // ever wired to this page's own object, never the inferred ancestor.
    if (!object || action.scopeKey !== object.key) {
      toast.error("This column is read-only.");
      return;
    }
    setDialogAction(action);
  };

  const emptyValidatorCount = useMemo(() => {
    if (!object) return 0;
    let count = 0;
    for (const bindings of object.handlers.values()) {
      for (const b of bindings) {
        if ((b.validator ?? "").trim() === "") count++;
      }
    }
    return count;
  }, [object]);

  if (isLoading) return <LoadingSpinner />;
  if (error) return <ErrorDisplay error={error} />;
  if (!object) return <ErrorDisplay error="Not found." />;

  const selectedRow = selection
    ? (rows.find((r) => r.name === selection.name) ?? null)
    : null;

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4">
      {emptyValidatorCount > 0 && (
        <Alert variant="destructive">
          <AlertTitle>Missing validators</AlertTitle>
          <AlertDescription className="flex items-center justify-between gap-4">
            <span>
              {`${object.label} has ${emptyValidatorCount} handler entr${emptyValidatorCount === 1 ? "y" : "ies"} with no validator. Saves are rejected until every one is filled.`}
            </span>
            <Button size="sm" onClick={() => setFillDialogOpen(true)}>
              Fill missing validators…
            </Button>
          </AlertDescription>
        </Alert>
      )}

      {scope === "tenant" && ancestor && (
        <div className="flex justify-end">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setCopyFlowOpen(true)}
          >
            Copy missing from ancestor…
          </Button>
        </div>
      )}

      <div className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-md border">
        <GridToolbar
          kind={kind}
          objects={objects}
          selectedKeys={objects.map((o) => o.key)}
          baselineKey={baselineKey}
          showFName={showFName}
          onShowFNameChange={setShowFName}
          filters={filters}
          onFiltersChange={setFilters}
          sort={sort}
          onSortChange={setSort}
          {...(ancestor
            ? {
                ancestryFilterOptions: {
                  value: ancestryClasses,
                  onChange: setAncestryClasses,
                },
              }
            : {})}
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
          {...(ancestor ? { ancestor } : {})}
          {...(ancestor && selection.scopeKey === ancestor.key
            ? {
                readOnlyReason: `${ancestor.label} is the ancestor template and is read-only here.`,
              }
            : {})}
        />
      )}

      <DefinitionActionDialogs
        kind={kind}
        objects={objects}
        action={dialogAction}
        onClose={() => setDialogAction(null)}
        {...(ancestor ? { ancestor } : {})}
      />

      <FillMissingValidatorsDialog
        open={fillDialogOpen}
        onOpenChange={setFillDialogOpen}
        target={{ source: object.source, id: object.key }}
        targetLabel={object.label}
        emptyValidatorCount={emptyValidatorCount}
      />

      {ancestor && (
        <CopyFromAncestorFlow
          open={copyFlowOpen}
          onOpenChange={setCopyFlowOpen}
          tenant={object}
          ancestor={ancestor}
          kind={kind}
          target={{ source: object.source, id: object.key }}
        />
      )}
    </div>
  );
}
