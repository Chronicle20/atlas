import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { DialogBaseProps } from "@/components/features/socket/dialogs/dialog-base";
import { useSocketMutation } from "@/lib/hooks/api/useSocketObjects";
import {
  copyBindings,
  deleteBinding,
  type BindingInput,
} from "@/lib/socket/mutate";
import {
  entriesOf,
  nameOfEntry,
  type Binding,
  type SocketObject,
} from "@/lib/socket/model";
import { formatOpcode, parseOpcode } from "@/lib/socket/opcode";
import { createErrorFromUnknown } from "@/types/api/errors";

export interface ResetToAncestorDialogProps extends DialogBaseProps {
  name: string;
  /**
   * The current Tenant object, so this dialog can render "current Tenant
   * values" for comparison. Display only - the write path (`apply` below)
   * always re-derives the bindings to remove from the freshly re-fetched
   * document `useSocketMutation` passes in, never from this (possibly
   * sparse or stale) prop.
   */
  tenant: SocketObject;
  ancestor: SocketObject;
}

function bindingSummary(b: Binding, kind: DialogBaseProps["kind"]): string {
  const opcode =
    b.opCodeValue !== null ? formatOpcode(b.opCodeValue) : b.opCode;
  const parts = [opcode];
  if (kind === "handler" && b.validator)
    parts.push(`validator: ${b.validator}`);
  if (b.services.length > 0) parts.push(`services: ${b.services.join(", ")}`);
  return parts.join(" • ");
}

function BindingsList({
  bindings,
  kind,
  emptyLabel,
}: {
  bindings: Binding[];
  kind: DialogBaseProps["kind"];
  emptyLabel: string;
}) {
  if (bindings.length === 0) {
    return <p className="text-muted-foreground text-xs">{emptyLabel}</p>;
  }
  return (
    <ul className="space-y-1 text-xs">
      {bindings.map((b, i) => (
        <li key={`${b.opCode}-${i}`} className="font-mono">
          {bindingSummary(b, kind)}
        </li>
      ))}
    </ul>
  );
}

/**
 * FR-9.4/FR-9.6 at single-Definition scope, Tenant only. Removes every
 * binding the Tenant currently holds for `name` (individually, via
 * `deleteBinding` - never `markUnsupported`, so no Unsupported marker is
 * added) and replaces them with the Ancestor Template's bindings for the
 * same name via `copyBindings`, which deep-clones every value so the
 * written binding shares no structure with the ancestor's in-memory copy.
 */
export function ResetToAncestorDialog({
  open,
  onOpenChange,
  target,
  targetLabel,
  kind,
  name,
  tenant,
  ancestor,
}: ResetToAncestorDialogProps) {
  const mutation = useSocketMutation();

  const currentBindings = entriesOf(tenant, kind).get(name) ?? [];
  const ancestorBindings = entriesOf(ancestor, kind).get(name) ?? [];

  const onConfirm = async () => {
    try {
      await mutation.mutateAsync({
        target,
        apply: (cfg) => {
          const list = kind === "handler" ? cfg.handlers : cfg.writers;
          const currentInFreshDoc = list.filter(
            (e) => nameOfEntry(e, kind) === name,
          );

          let out = cfg;
          for (const entry of currentInFreshDoc) {
            const value = parseOpcode(entry.opCode);
            if (value !== null) out = deleteBinding(out, kind, name, value);
          }

          const inputs: BindingInput[] = ancestorBindings.map((b) => ({
            opCode: b.opCode,
            services: b.services,
            ...(b.validator !== undefined ? { validator: b.validator } : {}),
            ...(b.fname ? { fname: b.fname } : {}),
            ...(b.options !== undefined ? { options: b.options } : {}),
          }));
          return copyBindings(out, kind, name, inputs);
        },
      });
      toast.success(`Reset ${name} to ${ancestor.label} in ${targetLabel}`);
      onOpenChange(false);
    } catch (e) {
      toast.error(createErrorFromUnknown(e).message);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            Reset {name} to {ancestor.label} in {targetLabel}
          </DialogTitle>
        </DialogHeader>

        <p className="text-muted-foreground text-sm">
          {`Every binding of "${name}" currently in ${targetLabel} will be removed and replaced with ${ancestor.label}'s bindings for the same name.`}
        </p>

        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-1">
            <h4 className="text-sm font-semibold">Current ({targetLabel})</h4>
            <BindingsList
              bindings={currentBindings}
              kind={kind}
              emptyLabel="No current bindings."
            />
          </div>
          <div className="space-y-1">
            <h4 className="text-sm font-semibold">
              Ancestor ({ancestor.label})
            </h4>
            <BindingsList
              bindings={ancestorBindings}
              kind={kind}
              emptyLabel="No ancestor bindings."
            />
          </div>
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
          >
            Cancel
          </Button>
          <Button
            type="button"
            disabled={mutation.isPending}
            onClick={onConfirm}
          >
            Reset to ancestor
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
