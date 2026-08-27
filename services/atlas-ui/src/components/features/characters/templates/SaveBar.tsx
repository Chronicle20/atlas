import { useState } from "react";
import { Button } from "@/components/ui/button";
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

interface SaveBarProps {
  dirty: boolean;
  isSaving: boolean;
  onSave: () => void;
  onDiscard: () => void;
  blockingIssues?: number | undefined;
}

export function SaveBar({
  dirty,
  isSaving,
  onSave,
  onDiscard,
  blockingIssues,
}: SaveBarProps) {
  const [confirmOpen, setConfirmOpen] = useState(false);
  const blocked = (blockingIssues ?? 0) > 0;
  const blockingText = `${blockingIssues} blocking ${
    blockingIssues === 1 ? "error" : "errors"
  }`;

  return (
    <div className="sticky bottom-0 z-10 mt-4 flex items-center justify-between gap-3 rounded-lg border bg-background/95 p-3 backdrop-blur">
      <p
        className={
          blocked
            ? "text-sm font-medium text-destructive"
            : dirty
              ? "text-sm font-medium"
              : "text-sm text-muted-foreground"
        }
      >
        {blocked
          ? blockingText
          : dirty
            ? "Unsaved changes"
            : "No unsaved changes"}
      </p>
      <div className="flex gap-2">
        <Button
          type="button"
          variant="outline"
          disabled={!dirty || isSaving}
          onClick={() => setConfirmOpen(true)}
        >
          Discard
        </Button>
        <Button
          type="button"
          disabled={!dirty || isSaving || blocked}
          onClick={onSave}
        >
          {isSaving ? "Saving…" : "Save"}
        </Button>
      </div>
      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Discard unsaved changes?</AlertDialogTitle>
            <AlertDialogDescription>
              All edits since the last save will be reverted.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Keep editing</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                setConfirmOpen(false);
                onDiscard();
              }}
            >
              Discard changes
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
