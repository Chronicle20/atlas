import { useState } from "react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { SocketTarget } from "@/lib/hooks/api/useSocketObjects";
import { useSocketMutation } from "@/lib/hooks/api/useSocketObjects";
import { fillMissingValidators } from "@/lib/socket/mutate";
import { createErrorFromUnknown } from "@/types/api/errors";

export interface FillMissingValidatorsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  target: SocketTarget;
  /** The scoped object's label, so the dialog names it (matches DialogBaseProps' convention from Task 17). */
  targetLabel: string;
  /** How many handler entries in the loaded document have no validator. */
  emptyValidatorCount: number;
}

/**
 * The only two validator values the corpus carries (measured, not an
 * allow-list - validate.go's rule is presence-only; any other string a
 * caller ends up needing can still be typed into `fillMissingValidators`
 * directly).
 */
const VALIDATOR_OPTIONS = ["NoOpValidator", "LoggedInValidator"] as const;

/**
 * The strict FR-11.4 escape hatch (not in the PRD - required by the decision
 * to enforce FR-11.4 as a blocking 400, Tasks 3 and 5). Saves are
 * whole-document, so a configuration carrying ANY empty handler validator
 * cannot be saved at all: a single-definition edit can never repair one,
 * because the very PATCH that would fix it is itself rejected by the same
 * rule. The live gms_95 tenant carries 32 such entries - a hard deadlock
 * under strict blocking without this. `fillMissingValidators` repairs every
 * offender (blank, whitespace-only, absent, or null at runtime) in one
 * document write, submitted through `useSocketMutation` as exactly one call.
 *
 * Renders nothing when `emptyValidatorCount < 1`. The caller (a banner on
 * the per-object page) decides WHEN to offer this dialog, but the guard is
 * repeated here so it can never render for a document with nothing to
 * repair, regardless of caller.
 */
export function FillMissingValidatorsDialog({
  open,
  onOpenChange,
  target,
  targetLabel,
  emptyValidatorCount,
}: FillMissingValidatorsDialogProps) {
  const mutation = useSocketMutation();
  const [validator, setValidator] = useState<string>(VALIDATOR_OPTIONS[0]);

  if (emptyValidatorCount < 1) return null;

  const onConfirm = async () => {
    try {
      await mutation.mutateAsync({
        target,
        apply: (cfg) => fillMissingValidators(cfg, validator),
      });
      toast.success(
        `Filled ${emptyValidatorCount} validator${emptyValidatorCount === 1 ? "" : "s"} in ${targetLabel}`,
      );
      onOpenChange(false);
    } catch (e) {
      toast.error(createErrorFromUnknown(e).message);
    }
  };

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            Fill missing validators in {targetLabel}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {`This configuration has ${emptyValidatorCount} handler entries with no validator. The server rejects any save of a configuration containing one, and saves replace the whole document — so editing them one at a time is not possible. This repairs every one of them in a single configuration write.`}
          </AlertDialogDescription>
        </AlertDialogHeader>

        <div className="space-y-2">
          <Label htmlFor="fill-missing-validators-choice">
            Validator to apply
          </Label>
          <Select value={validator} onValueChange={setValidator}>
            <SelectTrigger id="fill-missing-validators-choice">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {VALIDATOR_OPTIONS.map((v) => (
                <SelectItem key={v} value={v}>
                  {v}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <AlertDialogFooter>
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
            Fill validators
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
