import { useState } from "react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import type { DialogBaseProps } from "@/components/features/socket/dialogs/dialog-base";
import { useSocketMutation } from "@/lib/hooks/api/useSocketObjects";
import { deleteBinding, markUnsupported } from "@/lib/socket/mutate";
import { entriesOf, type SocketObject } from "@/lib/socket/model";
import { formatOpcode } from "@/lib/socket/opcode";
import { createErrorFromUnknown } from "@/types/api/errors";

export interface DeleteDefinitionDialogProps extends DialogBaseProps {
  name: string;
  opCodeValue: number;
  /**
   * The current scope object (the same `SocketObject` the drawer already
   * has in hand), so the "remove and mark unsupported" warning's binding
   * count is DERIVED rather than caller-supplied - a caller cannot
   * understate a four-binding removal as one by omitting or miscounting a
   * separate `bindingCount` prop. That choice removes ALL of `name`'s
   * bindings, not just the one addressed by `opCodeValue`, because
   * `markUnsupported` is name-scoped while bindings are opcode-scoped.
   */
  scope: SocketObject;
}

type DeleteChoice = "remove" | "remove-unsupported";

/**
 * FR-6.3. Two explicit, separately-chosen outcomes:
 *   - "Remove definition" -> `deleteBinding`, exactly the one binding
 *     addressed by `(name, opCodeValue)`.
 *   - "Remove and mark unsupported" -> `markUnsupported`, which necessarily
 *     removes EVERY binding of `name` (unsupported is name-scoped, bindings
 *     are opcode-scoped) - the warning states the count before it's chosen.
 */
export function DeleteDefinitionDialog({
  open,
  onOpenChange,
  target,
  targetLabel,
  kind,
  name,
  opCodeValue,
  scope,
}: DeleteDefinitionDialogProps) {
  const mutation = useSocketMutation();
  const [choice, setChoice] = useState<DeleteChoice>("remove");
  const opcodeLabel = formatOpcode(opCodeValue);
  // Derived from `scope`, never a caller-supplied number: it cannot drift
  // from what's actually there. Falls back to 1 only if `scope` somehow
  // doesn't carry the binding being deleted (should not happen - opCodeValue
  // addresses a real binding - but "singular" is the safe default wording).
  const count = entriesOf(scope, kind).get(name)?.length ?? 1;

  // Reset the choice back to "remove" whenever the dialog transitions from
  // closed to open (adjust state during render per
  // https://react.dev/learn/you-might-not-need-an-effect, instead of an
  // effect that would fire a synchronous setState on mount).
  const [wasOpen, setWasOpen] = useState(open);
  if (open !== wasOpen) {
    setWasOpen(open);
    if (open) setChoice("remove");
  }

  const onConfirm = async () => {
    try {
      await mutation.mutateAsync({
        target,
        apply: (cfg) =>
          choice === "remove"
            ? deleteBinding(cfg, kind, name, opCodeValue)
            : markUnsupported(cfg, kind, name),
      });
      toast.success(
        choice === "remove"
          ? `Removed ${name} (${opcodeLabel}) from ${targetLabel}`
          : `Marked ${name} unsupported in ${targetLabel}`,
      );
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
            Remove {name} ({opcodeLabel}) from {targetLabel}
          </DialogTitle>
        </DialogHeader>

        <RadioGroup
          value={choice}
          onValueChange={(v) => setChoice(v as DeleteChoice)}
          className="gap-4"
        >
          <div className="flex items-start gap-2">
            <RadioGroupItem
              value="remove"
              id="delete-choice-remove"
              className="mt-1"
            />
            <div>
              <Label htmlFor="delete-choice-remove" className="font-normal">
                Remove definition
              </Label>
              <p className="text-muted-foreground text-xs">
                {`Removes only the binding at ${opcodeLabel}. Other bindings of "${name}" in ${targetLabel}, if any, are untouched.`}
              </p>
            </div>
          </div>
          <div className="flex items-start gap-2">
            <RadioGroupItem
              value="remove-unsupported"
              id="delete-choice-unsupported"
              className="mt-1"
            />
            <div>
              <Label
                htmlFor="delete-choice-unsupported"
                className="font-normal"
              >
                Remove and mark unsupported
              </Label>
              <p className="text-muted-foreground text-xs">
                {count > 1
                  ? `All ${count} bindings of "${name}" will be removed and the name recorded as audited-absent - unsupported is name-scoped, not opcode-scoped.`
                  : `This binding of "${name}" will be removed and the name recorded as audited-absent.`}
              </p>
            </div>
          </div>
        </RadioGroup>

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
            variant="destructive"
            disabled={mutation.isPending}
            onClick={onConfirm}
          >
            {choice === "remove" ? "Remove" : "Remove and mark unsupported"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
