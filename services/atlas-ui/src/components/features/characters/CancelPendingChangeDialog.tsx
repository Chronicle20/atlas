import { useState } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useTenant } from "@/context/tenant-context";
import { useCancelPendingChange } from "@/lib/hooks/api/usePendingChanges";
import { createErrorFromUnknown } from "@/types/api/errors";
import type { PendingChange } from "@/services/api/pending-changes.service";

function requestedValue(change: PendingChange): string {
  return change.type === "NAME_CHANGE"
    ? (change.requestedName ?? "")
    : `world ${change.destinationWorldId}`;
}

interface CancelPendingChangeDialogProps {
  change: PendingChange;
  characterName: string;
  characterId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/**
 * Confirmation dialog for cancelling a pending cash-shop name change or
 * world transfer request. Names both the character and the requested
 * value so an operator cannot cancel the wrong record by muscle memory.
 */
export function CancelPendingChangeDialog({
  change,
  characterName,
  characterId,
  open,
  onOpenChange,
}: CancelPendingChangeDialogProps) {
  const { activeTenant } = useTenant();
  const [isCancelling, setIsCancelling] = useState(false);
  const { mutateAsync: cancelChange } = useCancelPendingChange();

  const handleCancel = async () => {
    setIsCancelling(true);
    try {
      await cancelChange({
        tenant: activeTenant,
        characterId,
        id: change.id,
      });
      toast.success("Pending change cancelled");
      onOpenChange(false);
    } catch (error: unknown) {
      toast.error(createErrorFromUnknown(error).message);
    } finally {
      setIsCancelling(false);
    }
  };

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Cancel pending change?</AlertDialogTitle>
          <AlertDialogDescription>
            This will cancel the pending{" "}
            {change.type === "NAME_CHANGE" ? "name change" : "world transfer"}{" "}
            to <strong>{requestedValue(change)}</strong> for{" "}
            <strong>{characterName}</strong>. This action cannot be undone.
          </AlertDialogDescription>
        </AlertDialogHeader>

        <AlertDialogFooter>
          <AlertDialogCancel disabled={isCancelling}>
            Keep Request
          </AlertDialogCancel>
          <AlertDialogAction
            onClick={(e) => {
              e.preventDefault();
              handleCancel();
            }}
            disabled={isCancelling}
            className={cn(buttonVariants({ variant: "destructive" }))}
          >
            {isCancelling ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Cancelling...
              </>
            ) : (
              "Cancel Request"
            )}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
