import { useState } from "react";
import { Loader2, RotateCcw } from "lucide-react";
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
import { Button, buttonVariants } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { useReseedTemplate, useTemplate } from "@/lib/hooks/api/useTemplates";
import { createErrorFromUnknown } from "@/types/api/errors";

export interface TemplateReseedButtonProps {
  id: string | undefined;
}

/**
 * Resets the viewed template to the configuration shipped in the currently
 * deployed image.
 *
 * Lives in the detail LAYOUT header beside ConfigExportButton, for the same
 * reason that component does: it acts on the whole template document, not on
 * the sub-tab being viewed, so putting it in the layout gives it every sub-tab
 * with no per-page wiring.
 *
 * Disabled when the template's shippedRevision is empty or absent - this image
 * ships no file for its region/version, so there is nothing to reset to and the
 * server would 409 (FR-5.4).
 */
export function TemplateReseedButton({ id }: TemplateReseedButtonProps) {
  const [open, setOpen] = useState(false);
  const [isReseeding, setIsReseeding] = useState(false);
  const query = useTemplate(id ?? "");
  const reseed = useReseedTemplate();

  const shippedRevision = query.data?.attributes.shippedRevision;
  const shipsSeedFile = Boolean(shippedRevision);
  const disabled = !id || !query.data || !shipsSeedFile;

  const onConfirm = async () => {
    if (!id) return;
    setIsReseeding(true);
    try {
      await reseed.mutateAsync({ id });
      toast.success("Template reset to shipped defaults");
      setOpen(false);
    } catch (e) {
      // Route through createErrorFromUnknown so the server's JSON:API error
      // detail survives into the toast rather than a generic string. Same
      // convention as ConfigExportButton and every other async-catch site
      // under components/features/. The dialog stays open and the displayed
      // template is untouched (FR-5.7).
      toast.error(createErrorFromUnknown(e, "Reset failed").message);
    } finally {
      setIsReseeding(false);
    }
  };

  const trigger = (
    <Button
      type="button"
      variant="outline"
      size="sm"
      disabled={disabled}
      onClick={() => setOpen(true)}
    >
      <RotateCcw className="h-4 w-4" aria-hidden="true" />
      Reset to shipped defaults
    </Button>
  );

  return (
    <>
      {/*
        The Tooltip root is ALWAYS mounted and only its content is gated -
        React reconciles by element type at a position, so swapping between a
        bare Button and a Tooltip-wrapped one would remount the button's DOM
        node and silently drop focus. Same reasoning as ConfigExportButton.
      */}
      <Tooltip>
        <TooltipTrigger asChild>{trigger}</TooltipTrigger>
        {disabled ? (
          <TooltipContent>
            This image ships no seed file for this template&apos;s region and
            version, so there is nothing to reset to.
          </TooltipContent>
        ) : null}
      </Tooltip>

      <AlertDialog open={open} onOpenChange={setOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Reset to shipped defaults?</AlertDialogTitle>
            <AlertDialogDescription>
              This template will be overwritten with the version shipped in this
              image. Edits made through the UI will be lost. The template&apos;s
              id, region and version are unchanged.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            {/*
              Cancel renders FIRST so it holds the dialog's default focus - the
              destructive action must never be what Enter triggers (FR-5.5).
            */}
            <AlertDialogCancel disabled={isReseeding}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={(e) => {
                e.preventDefault();
                void onConfirm();
              }}
              disabled={isReseeding}
              className={cn(buttonVariants({ variant: "destructive" }))}
            >
              {isReseeding ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Resetting...
                </>
              ) : (
                "Reset Template"
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
