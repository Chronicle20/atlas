import type { GridSelection } from "@/components/features/socket/PacketGrid";
import { OptionsMatrixTable } from "@/components/features/socket/OptionsMatrix";
import {
  SHORT_STATE_LABEL,
  STATE_CELL_CLASS,
  STATE_LABEL,
} from "@/components/features/socket/cell-state";
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
   * When set, every MUTATING action on the current scope (Add/Edit/Undefine/
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
 * The accessible name of every action button ends with the scoped object's
 * label (FR-5.2/5.3), built through this one helper so the wording cannot
 * drift between buttons.
 *
 * The VISIBLE label is deliberately shorter than the accessible one. Six
 * buttons all repeating "… in GMS v95.1" is a wall of near-identical text
 * you have to read to the end to tell apart, while the header already states
 * the scope once. So the button shows the verb, the accessible name and
 * tooltip carry the full "verb + scope (+ opcode)" phrase, and a screen
 * reader still hears which object it is about to change.
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

interface FieldsGridProps {
  row: Row;
  objects: SocketObject[];
  kind: DefinitionKind;
  baselineKey: string;
}

/**
 * FR-5.1: one card per object - the object, its opcodes, and (for handlers)
 * its validator.
 *
 * It used to be a six-column table that also repeated the state as a word,
 * the services (which the Services tab owns) and the options shape (which
 * the Options tab owns) - three columns of information available one click
 * away, crowding out the one thing only this view shows. State is now
 * carried by the card's tint, in the same colours as the grid cell it came
 * from.
 *
 * Every card is the same three lines - object, opcode, footnote - whatever
 * its state, so a row of them scans as a row rather than as cards of two
 * different heights. A card with no definition leaves the opcode line EMPTY
 * (there is no opcode to show) and spends its footnote on the state word,
 * which keeps the state non-colour-only exactly where the defined cards put
 * their validator.
 */
function FieldsGrid({ row, objects, kind, baselineKey }: FieldsGridProps) {
  return (
    <ul className="grid grid-cols-[repeat(auto-fill,minmax(11rem,1fr))] gap-1.5">
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
        return (
          <li
            key={o.key}
            aria-label={`${o.label}: ${STATE_LABEL[state]}`}
            className={cn(
              "rounded-md border px-2.5 py-1.5",
              STATE_CELL_CLASS[state],
              o.key === baselineKey && "border-primary/60",
            )}
          >
            <span className="text-muted-foreground block text-[10px] tracking-wide uppercase">
              {o.label}
            </span>
            {/* Line 2 is the opcode line, and stays reserved when there is no
                opcode - that empty line is what keeps an Undefined card the
                same shape as the defined ones beside it. */}
            <span className="block min-h-5 font-mono text-sm">
              {state === "defined" ? opcodes : " "}
            </span>
            <span className="text-muted-foreground block min-h-4 truncate text-[11px]">
              {state === "defined"
                ? kind === "handler"
                  ? validators
                  : " "
                : SHORT_STATE_LABEL[state]}
            </span>
          </li>
        );
      })}
    </ul>
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
 * Opens on one Definition, scoped to one object (FR-5.2/5.3), as a bottom
 * sheet: the matrix is a wide grid, so the detail that explains one of its
 * rows reads across the same width rather than in a narrow right-hand
 * column.
 *
 * Design §5.1: lists EVERY binding of the scoped object with its own action
 * row - the only place a multi-binding Definition (NoOpHandler bound to four
 * opcodes in gms_95_1) is individually addressable; the grid cell can only
 * show the lowest opcode plus a count.
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
  // FR-5.4: Edit/Undefine/Open have no meaning without a real Definition to
  // target; Add/Copy/Mark-unsupported still mean something on an Undefined
  // scope (that's precisely when you'd use them) so they are never disabled
  // on this basis - defining a definition that this template lacks is the
  // whole point of being able to click an empty cell.
  const canTargetDefinition = scopeState === "defined";

  // The top-level Edit/Undefine/Open buttons act on ONE binding, but a
  // Definition can carry several (NoOpHandler: four opcodes in gms_95_1).
  // `lowestOpCodeValue` used to be the implicit target here, which meant
  // "Undefine NoOpHandler in GMS v83.1" silently removed exactly one of four
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
        ? "This definition has more than one binding in this scope - edit or remove the specific binding on the right."
        : "This binding's opcode does not parse - edit or remove it directly on the right."
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

  // Only Open/Edit/Undefine carry an opcode in the accessible name - naming
  // an already-unambiguous single target (FR-5.2 extended to the binding
  // level), never spliced in a way that would break a caller matching
  // `verb in <label>`.
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

  const kindLabel = kind === "handler" ? "Handler" : "Writer";

  return (
    <Sheet
      open
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      {/* `flex flex-col` is load-bearing, not decoration: SheetContent's base
          class carries a `gap-*` that does nothing on a block container, so
          before this the header, the action row and the tabs sat flush
          against each other with no vertical rhythm at all. The min-height
          stops a short definition (one row of Fields cards) from collapsing
          the sheet to a band too shallow for its own header. */}
      <SheetContent
        side="bottom"
        className="flex max-h-[85vh] min-h-[26rem] flex-col gap-4 overflow-y-auto p-5"
      >
        <SheetHeader className="shrink-0 pr-8">
          <div className="flex flex-wrap items-center gap-2">
            <SheetTitle className="font-mono">{row.name}</SheetTitle>
            <span className="text-muted-foreground rounded-full border px-2 py-0.5 text-xs">
              {kindLabel}
            </span>
            {row.fname && (
              <span className="text-muted-foreground rounded-full border px-2 py-0.5 font-mono text-xs">
                {row.fname}
              </span>
            )}
            <span className="border-primary/60 text-primary bg-primary/10 rounded-full border px-2 py-0.5 text-xs">
              {`scope: ${scope.label}`}
              {scope.key === baselineKey && " · baseline"}
            </span>
          </div>
          <SheetDescription>
            {`Actions below apply to ${scope.label} only. Click another cell in this row to re-scope.`}
          </SheetDescription>
        </SheetHeader>

        <div className="flex shrink-0 flex-wrap gap-2">
          <Button
            type="button"
            size="sm"
            variant="outline"
            disabled={!canTargetSingleBinding}
            aria-label={targetLabel("Open", "in", false)}
            title={
              singleBindingDisabledReason ??
              `Go to ${scope.label}'s ${kindLabel.toLowerCase()}s page with this definition selected.`
            }
            onClick={() => fireTarget("open-in")}
          >
            Open ↗
          </Button>
          <Button
            type="button"
            size="sm"
            disabled={editDeleteDisabled}
            aria-label={targetLabel("Edit", "in")}
            title={
              editDeleteReason ??
              `Change this binding's opcode, services${kind === "handler" ? ", validator" : ""} or options in ${scope.label}.`
            }
            onClick={() => fireTarget("edit")}
          >
            Edit
          </Button>
          <Button
            type="button"
            size="sm"
            variant="outline"
            disabled={isReadOnly}
            aria-label={
              canTargetDefinition
                ? label("Add binding", "to")
                : label("Define", "in")
            }
            title={
              readOnlyReason ??
              (canTargetDefinition
                ? `Bind ${row.name} to a further opcode in ${scope.label}.`
                : `${scope.label} has no ${row.name} yet - add it here.`)
            }
            onClick={() => fire("add")}
          >
            {canTargetDefinition ? "Add binding" : "Define here"}
          </Button>
          <Button
            type="button"
            size="sm"
            variant="outline"
            disabled={isReadOnly}
            aria-label={label("Copy", "into")}
            title={
              readOnlyReason ??
              `Copy ${row.name} from another object into ${scope.label}.`
            }
            onClick={() => fire("copy")}
          >
            Copy from…
          </Button>
          {scopeState === "unsupported" ? (
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={isReadOnly}
              aria-label={label("Clear unsupported", "in", false)}
              title={
                readOnlyReason ??
                `Drop the audited-absent record, returning ${row.name} to Undefined in ${scope.label}.`
              }
              onClick={() => fire("clear-unsupported")}
            >
              Clear unsupported
            </Button>
          ) : (
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={isReadOnly}
              aria-label={label("Mark unsupported", "in")}
              title={
                readOnlyReason ??
                `Record that this packet does not exist in ${scope.label}. The cell reads "n/a".`
              }
              onClick={() => fire("mark-unsupported")}
            >
              Mark unsupported
            </Button>
          )}
          {ancestor && scope.source === "tenant" && (
            <Button
              type="button"
              size="sm"
              variant="outline"
              aria-label={label("Reset to Ancestor", "in")}
              title={`Replace ${row.name} in ${scope.label} with ${ancestor.label}'s version.`}
              onClick={() => fire("reset-to-ancestor")}
            >
              Reset to ancestor
            </Button>
          )}
          <Button
            type="button"
            size="sm"
            variant="destructive"
            className="ml-auto"
            disabled={editDeleteDisabled}
            aria-label={targetLabel("Undefine", "in")}
            title={
              editDeleteReason ??
              `Remove ${row.name} from ${scope.label}. The cell returns to Undefined.`
            }
            onClick={() => fireTarget("delete")}
          >
            Undefine
          </Button>
        </div>

        <div className="flex flex-col gap-4 md:flex-row">
          <Tabs defaultValue="fields" className="min-w-0 flex-1">
            <TabsList>
              <TabsTrigger value="fields">Fields</TabsTrigger>
              <TabsTrigger value="options">Options</TabsTrigger>
              <TabsTrigger value="services">Services</TabsTrigger>
            </TabsList>
            <TabsContent value="fields">
              <FieldsGrid
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

          <section className="md:w-72 md:shrink-0">
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
                        aria-label={`Edit ${binding.opCode} in ${scope.label}…`}
                        title={readOnlyReason ?? `Change this binding only.`}
                        onClick={() => fireBinding("edit", binding)}
                      >
                        Edit
                      </Button>
                      <Button
                        type="button"
                        size="sm"
                        variant="ghost"
                        disabled={isReadOnly}
                        aria-label={`Remove ${binding.opCode} from ${scope.label}…`}
                        title={
                          readOnlyReason ??
                          (bindings.length > 1
                            ? `Remove this binding only - ${row.name} stays defined in ${scope.label} through its other ${bindings.length - 1} binding(s).`
                            : `Remove this binding. ${row.name} becomes Undefined in ${scope.label}.`)
                        }
                        onClick={() => fireBinding("delete", binding)}
                      >
                        Remove
                      </Button>
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </section>
        </div>
      </SheetContent>
    </Sheet>
  );
}
