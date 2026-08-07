import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { DialogBaseProps } from "@/components/features/socket/dialogs/dialog-base";
import { useSocketMutation } from "@/lib/hooks/api/useSocketObjects";
import { markUnsupported } from "@/lib/socket/mutate";
import { createErrorFromUnknown } from "@/types/api/errors";

export interface MarkUnsupportedDialogProps extends DialogBaseProps {
  name: string;
  /** How many bindings `name` currently holds in this scope (>= 1). */
  bindingCount: number;
}

/**
 * FR-6.4/FR-1.1. `markUnsupported` removes EVERY binding of `name`,
 * necessarily: `unsupported` is name-scoped while bindings are
 * opcode-scoped, so a name cannot be half-marked. This dialog states that in
 * as many words, and names the count when it is more than one, before the
 * user can confirm - this is what stops someone destroying four live
 * `NoOpHandler` routes thinking they are hiding one.
 */
export function MarkUnsupportedDialog({
  open,
  onOpenChange,
  target,
  targetLabel,
  kind,
  name,
  bindingCount,
}: MarkUnsupportedDialogProps) {
  const mutation = useSocketMutation();

  const warning =
    bindingCount > 1
      ? `All ${bindingCount} bindings of "${name}" will be removed - a name cannot be both Defined and Unsupported at once, and Unsupported is name-scoped while bindings are opcode-scoped.`
      : `The existing binding of "${name}" will be removed - a name cannot be both Defined and Unsupported at once.`;

  const onConfirm = async () => {
    try {
      await mutation.mutateAsync({
        target,
        apply: (cfg) => markUnsupported(cfg, kind, name),
      });
      toast.success(`Marked ${name} unsupported in ${targetLabel}`);
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
            Mark {name} unsupported in {targetLabel}
          </DialogTitle>
          <DialogDescription>{warning}</DialogDescription>
        </DialogHeader>
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
            Mark unsupported
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
