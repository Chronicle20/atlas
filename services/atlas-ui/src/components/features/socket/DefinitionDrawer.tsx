import type { GridSelection } from "@/components/features/socket/PacketGrid";
import { OptionsMatrixTable } from "@/components/features/socket/OptionsMatrix";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import type { Row } from "@/lib/socket/matrix";
import type { Binding, DefinitionKind, SocketObject } from "@/lib/socket/model";
import { stateOf } from "@/lib/socket/model";
import { formatOpcode } from "@/lib/socket/opcode";
import { classifyOptions } from "@/lib/socket/options";
import { cn } from "@/lib/utils";

export type DrawerActionType =
  | "add"
  | "edit"
  | "delete"
  | "mark-unsupported"
  | "clear-unsupported"
  | "copy"
  | "reset-to-ancestor"
  | "open-in";

export interface DrawerAction {
  type: DrawerActionType;
  /** The object the action targets (FR-5.2). Never implicit. */
  scopeKey: string;
  name: string;
  /** Present for binding-scoped actions (edit, delete, open-in). */
  opCodeValue?: number;
}

export interface DefinitionDrawerProps {
  row: Row;
  objects: SocketObject[];
  kind: DefinitionKind;
  baselineKey: string;
  selection: GridSelection;
  onClose: () => void;
  onAction: (action: DrawerAction) => void;
  /** Tenant pages only: enables Reset to Ancestor. */
  ancestor?: SocketObject;
  /**
   * When set, every MUTATING action on the current scope (Add/Edit/Delete/
   * Copy/Mark-Unsupported/Clear-Unsupported, including each per-binding row)
   * is disabled and this string is shown as the button's `title` - e.g. the
   * FR-7.2 ancestor column on a Tenant page, which DefinitionGridPage passes
   * whenever `selection.scopeKey` is the inferred ancestor's key. Open (a
   * navigation, not a mutation) is unaffected, and Reset to Ancestor is
   * unaffected too - it never renders for the ancestor scope itself, since
   * it is already gated on `scope.source === "tenant"`. Absent means fully
   * editable, matching every other scope's default.
   */
  readOnlyReason?: string;
}

/**
 * Every action button's accessible name ends with the scoped object's label
 * (FR-5.2/5.3), built through this one helper so the wording cannot drift
 * between buttons. `ellipsis` marks the actions that open a follow-up
 * dialog/form rather than firing immediately. `opcode`, when given, is
 * appended AFTER the scope label (never spliced between the verb and the
 * preposition) so a caller matching `/edit in GMS v87\.1/i` still matches
 * once a binding-scoped action starts naming its target opcode too.
 */
function actionLabel(
  verb: string,
  preposition: string,
  scopeLabel: string,
  options: { ellipsis?: boolean; opcode?: number | null } = {},
): string {
  const { ellipsis = true, opcode = null } = options;
  const opcodeSuffix = opcode !== null ? ` (${formatOpcode(opcode)})` : "";
  return `${verb} ${preposition} ${scopeLabel}${opcodeSuffix}${ellipsis ? "…" : ""}`;
}

function buildAction(
  type: DrawerActionType,
  scopeKey: string,
  name: string,
  opCodeValue: number | null | undefined,
): DrawerAction {
  return {
    type,
    scopeKey,
    name,
    ...(opCodeValue !== null && opCodeValue !== undefined
      ? { opCodeValue }
      : {}),
  };
}

interface FieldsTableProps {
  row: Row;
  objects: SocketObject[];
  kind: DefinitionKind;
  baselineKey: string;
}

/** FR-5.1: one row per object - label, state, opcodes, validator (handlers
 * only), services, and the options shape from `classifyOptions`. */
function FieldsTable({ row, objects, kind, baselineKey }: FieldsTableProps) {
  return (
    <table className="w-full border-collapse text-left text-sm">
      <thead>
        <tr>
          <th scope="col" className="border-b px-2 py-1">
            Object
          </th>
          <th scope="col" className="border-b px-2 py-1">
            State
          </th>
          <th scope="col" className="border-b px-2 py-1">
            Opcode
          </th>
          {kind === "handler" && (
            <th scope="col" className="border-b px-2 py-1">
              Validator
            </th>
          )}
          <th scope="col" className="border-b px-2 py-1">
            Services
          </th>
          <th scope="col" className="border-b px-2 py-1">
            Options
          </th>
        </tr>
      </thead>
      <tbody>
        {objects.map((o) => {
          const cell = row.cells.get(o.key);
          const bindings = cell?.bindings ?? [];
          const state = cell?.state ?? "undefined";
          const opcodes = bindings
            .map((b) =>
              b.opCodeValue !== null ? formatOpcode(b.opCodeValue) : b.opCode,
            )
            .join(", ");
          const validators = Array.from(
            new Set(
              bindings.map((b) => b.validator).filter((v): v is string => !!v),
            ),
          ).join(", ");
          const services = Array.from(
            new Set(bindings.flatMap((b) => b.services)),
          ).join(", ");
          const shape = classifyOptions(bindings[0]?.options);
          return (
            <tr
              key={o.key}
              className={cn(o.key === baselineKey && "bg-muted/40")}
            >
              <td className="border-b px-2 py-1">{o.label}</td>
              <td className="border-b px-2 py-1">{state}</td>
              <td className="border-b px-2 py-1 font-mono">{opcodes}</td>
              {kind === "handler" && (
                <td className="border-b px-2 py-1">{validators}</td>
              )}
              <td className="border-b px-2 py-1">{services}</td>
              <td className="border-b px-2 py-1">{shape}</td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}

interface ServicesTableProps {
  row: Row;
  objects: SocketObject[];
  baselineKey: string;
}

/** Per object, the union of its bindings' service tags. */
function ServicesTable({ row, objects, baselineKey }: ServicesTableProps) {
  return (
    <table className="w-full border-collapse text-left text-sm">
      <thead>
        <tr>
          <th scope="col" className="border-b px-2 py-1">
            Object
          </th>
          <th scope="col" className="border-b px-2 py-1">
            Services
          </th>
        </tr>
      </thead>
      <tbody>
        {objects.map((o) => {
          const bindings = row.cells.get(o.key)?.bindings ?? [];
          const services = Array.from(
            new Set(bindings.flatMap((b) => b.services)),
          ).sort();
          return (
            <tr
              key={o.key}
              className={cn(o.key === baselineKey && "bg-muted/40")}
            >
              <td className="border-b px-2 py-1">{o.label}</td>
              <td className="border-b px-2 py-1">
                {services.length > 0 ? (
                  services.join(", ")
                ) : (
                  <span className="text-muted-foreground">—</span>
                )}
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}

/**
 * Opens on one Definition, scoped to one object (FR-5.2/5.3). Design §5.1:
 * lists EVERY binding of the scoped object with its own action row - the
 * only place a multi-binding Definition (NoOpHandler bound to four opcodes
 * in gms_95_1) is individually addressable; the grid cell can only show the
 * lowest opcode plus a count.
 */
export function DefinitionDrawer({
  row,
  objects,
  kind,
  baselineKey,
  selection,
  onClose,
  onAction,
  ancestor,
  readOnlyReason,
}: DefinitionDrawerProps) {
  const scope = objects.find((o) => o.key === selection.scopeKey);
  if (!scope) return null;

  const scopeState = stateOf(scope, kind, row.name);
  const scopeCell = row.cells.get(scope.key);
  const bindings = scopeCell?.bindings ?? [];
  // FR-5.4: Edit/Delete/Open have no meaning without a real Definition to
  // target; Add/Copy/Mark-unsupported still mean something on an Undefined
  // scope (that's precisely when you'd use them) so they are never disabled
  // on this basis.
  const canTargetDefinition = scopeState === "defined";

  // The top-level Edit/Delete/Open buttons act on ONE binding, but a
  // Definition can carry several (NoOpHandler: four opcodes in gms_95_1).
  // `lowestOpCodeValue` used to be the implicit target here, which meant
  // "Delete NoOpHandler in GMS v83.1" silently deleted exactly one of four
  // live routes while reading as if it removed the whole definition - a
  // data-loss hazard, not an ambiguity a throw could catch, because
  // resolving (name, opcode) to one binding always succeeds. The fix: these
  // buttons are only ever wired to a SPECIFIC, UNAMBIGUOUS binding - the
  // scope's only one, and only when its opcode actually parses. Anywhere
  // else (more than one binding, or a single binding whose opcode does not
  // parse) they are disabled and the per-binding rows below - already
  // unambiguous by construction (Design §5.1) - are the only way to act.
  const singleResolvableBinding: Binding | null =
    bindings.length === 1 && bindings[0]!.opCodeValue !== null
      ? bindings[0]!
      : null;
  const canTargetSingleBinding =
    canTargetDefinition && singleResolvableBinding !== null;
  const singleBindingDisabledReason: string | undefined =
    canTargetDefinition && !canTargetSingleBinding
      ? bindings.length > 1
        ? "This definition has more than one binding in this scope - edit or delete the specific binding below."
        : "This binding's opcode does not parse - edit or delete it directly below."
      : undefined;

  // FR-7.2: a read-only scope (the Tenant page's ancestor column) disables
  // every MUTATING action, taking priority over the single-binding-ambiguity
  // reason above when both would otherwise apply - "this column is read-only"
  // is the more specific, more actionable explanation.
  const isReadOnly = !!readOnlyReason;
  const editDeleteDisabled = !canTargetSingleBinding || isReadOnly;
  const editDeleteReason = readOnlyReason ?? singleBindingDisabledReason;

  const label = (verb: string, preposition: string, ellipsis = true) =>
    actionLabel(verb, preposition, scope.label, { ellipsis });

  // Only Open/Edit/Delete carry an opcode in the label - naming an already-
  // unambiguous single target (FR-5.2 extended to the binding level), never
  // spliced in a way that would break a caller matching `verb in <label>`.
  const targetLabel = (verb: string, preposition: string, ellipsis = true) =>
    actionLabel(verb, preposition, scope.label, {
      ellipsis,
      opcode: canTargetSingleBinding
        ? singleResolvableBinding!.opCodeValue
        : null,
    });

  const fire = (type: DrawerActionType, opCodeValue?: number | null) =>
    onAction(buildAction(type, scope.key, row.name, opCodeValue));

  const fireTarget = (type: "open-in" | "edit" | "delete") => {
    if (!canTargetSingleBinding) return;
    onAction(
      buildAction(
        type,
        scope.key,
        row.name,
        singleResolvableBinding!.opCodeValue,
      ),
    );
  };

  const fireBinding = (type: "edit" | "delete", binding: Binding) =>
    onAction(buildAction(type, scope.key, row.name, binding.opCodeValue));

  return (
    <Sheet
      open
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <SheetContent side="right" className="w-full overflow-y-auto sm:max-w-xl">
        <SheetHeader>
          <SheetTitle>{row.name}</SheetTitle>
          <SheetDescription>
            Scoped to{" "}
            <span className="text-foreground font-medium">{scope.label}</span>
            {scope.key === baselineKey && (
              <span className="bg-primary/10 text-primary ml-2 rounded px-1 text-[10px] uppercase">
                baseline
              </span>
            )}
          </SheetDescription>
        </SheetHeader>

        <div className="mt-4 flex flex-wrap gap-2">
          <Button
            type="button"
            size="sm"
            variant="outline"
            disabled={!canTargetSingleBinding}
            title={singleBindingDisabledReason}
            onClick={() => fireTarget("open-in")}
          >
            {targetLabel("Open", "in", false)}
          </Button>
          <Button
            type="button"
            size="sm"
            disabled={editDeleteDisabled}
            title={editDeleteReason}
            onClick={() => fireTarget("edit")}
          >
            {targetLabel("Edit", "in")}
          </Button>
          <Button
            type="button"
            size="sm"
            variant="destructive"
            disabled={editDeleteDisabled}
            title={editDeleteReason}
            onClick={() => fireTarget("delete")}
          >
            {targetLabel("Delete", "in")}
          </Button>
          <Button
            type="button"
            size="sm"
            variant="outline"
            disabled={isReadOnly}
            title={readOnlyReason}
            onClick={() => fire("add")}
          >
            {label("Add", "to")}
          </Button>
          <Button
            type="button"
            size="sm"
            variant="outline"
            disabled={isReadOnly}
            title={readOnlyReason}
            onClick={() => fire("copy")}
          >
            {label("Copy", "into")}
          </Button>
          {scopeState === "unsupported" ? (
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={isReadOnly}
              title={readOnlyReason}
              onClick={() => fire("clear-unsupported")}
            >
              {label("Clear unsupported", "in", false)}
            </Button>
          ) : (
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={isReadOnly}
              title={readOnlyReason}
              onClick={() => fire("mark-unsupported")}
            >
              {label("Mark unsupported", "in")}
            </Button>
          )}
          {ancestor && scope.source === "tenant" && (
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={() => fire("reset-to-ancestor")}
            >
              {label("Reset to Ancestor", "in")}
            </Button>
          )}
        </div>

        <section className="mt-4">
          <h3 className="text-sm font-semibold">Bindings in {scope.label}</h3>
          {bindings.length === 0 ? (
            <p className="text-muted-foreground mt-1 text-sm">
              No bindings in {scope.label}.
            </p>
          ) : (
            <ul
              aria-label={`Bindings in ${scope.label}`}
              className="mt-2 space-y-1"
            >
              {bindings.map((binding, i) => (
                <li
                  key={`${binding.opCode}-${i}`}
                  className="flex items-center justify-between gap-2 rounded border px-2 py-1 text-sm"
                >
                  <span className="font-mono">{binding.opCode}</span>
                  <span className="flex gap-1">
                    <Button
                      type="button"
                      size="sm"
                      variant="ghost"
                      disabled={isReadOnly}
                      title={readOnlyReason}
                      onClick={() => fireBinding("edit", binding)}
                    >
                      Edit
                    </Button>
                    <Button
                      type="button"
                      size="sm"
                      variant="ghost"
                      disabled={isReadOnly}
                      title={readOnlyReason}
                      onClick={() => fireBinding("delete", binding)}
                    >
                      Delete
                    </Button>
                  </span>
                </li>
              ))}
            </ul>
          )}
        </section>

        <Tabs defaultValue="fields" className="mt-4">
          <TabsList>
            <TabsTrigger value="fields">Fields</TabsTrigger>
            <TabsTrigger value="options">Options</TabsTrigger>
            <TabsTrigger value="services">Services</TabsTrigger>
          </TabsList>
          <TabsContent value="fields">
            <FieldsTable
              row={row}
              objects={objects}
              kind={kind}
              baselineKey={baselineKey}
            />
          </TabsContent>
          <TabsContent value="options">
            <OptionsMatrixTable
              objects={objects}
              kind={kind}
              name={row.name}
              baselineKey={baselineKey}
            />
          </TabsContent>
          <TabsContent value="services">
            <ServicesTable
              row={row}
              objects={objects}
              baselineKey={baselineKey}
            />
          </TabsContent>
        </Tabs>
      </SheetContent>
    </Sheet>
  );
}
