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
import {
  useResetTenantConfiguration,
  useTenantConfiguration,
} from "@/lib/hooks/api/useTenants";
import type { TenantResetSection } from "@/services/api/tenants.service";
import { createErrorFromUnknown } from "@/types/api/errors";

export interface TenantResetButtonProps {
  id: string | undefined;
  /** Omit for a whole-document reset. */
  sections?: TenantResetSection[];
  /** Human label for the scoped copy, e.g. "socket handlers and writers". */
  sectionLabel?: string;
}

/**
 * Resets a tenant's configuration to its resolved template baseline.
 *
 * Cut from TemplateReseedButton - same conventions: the Tooltip root is
 * always mounted (only its content is gated on `disabled`) so the button's
 * DOM node is never remounted and focus is never dropped; Cancel renders
 * first in the dialog footer so Enter never fires the destructive action;
 * errors route through createErrorFromUnknown so the server's JSON:API error
 * detail survives into the toast, the dialog stays open, and the displayed
 * tenant is untouched.
 *
 * Disabled when no baseline template resolves for this tenant's region and
 * version - there is nothing to reset to and the server would 422.
 */
export function TenantResetButton({
  id,
  sections,
  sectionLabel,
}: TenantResetButtonProps) {
  const [open, setOpen] = useState(false);
  const [isResetting, setIsResetting] = useState(false);
  const query = useTenantConfiguration(id ?? "");
  const reset = useResetTenantConfiguration();

  const hasBaseline = Boolean(query.data?.attributes.baselineTemplateId);
  const disabled = !id || !query.data || !hasBaseline;

  const onConfirm = async () => {
    if (!id) return;
    setIsResetting(true);
    try {
      await reset.mutateAsync({ id, sections });
      toast.success(
        sections
          ? `${sectionLabel ?? "This section"} reset to template`
          : "Tenant reset to template",
      );
      setOpen(false);
    } catch (e) {
      // Route through createErrorFromUnknown so the server's JSON:API error
      // detail survives into the toast rather than a generic string. Same
      // convention as TemplateReseedButton and every other async-catch site
      // under components/features/. The dialog stays open and the displayed
      // tenant is untouched.
      toast.error(createErrorFromUnknown(e, "Reset failed").message);
    } finally {
      setIsResetting(false);
    }
  };

  const triggerLabel = sections
    ? `Reset ${sectionLabel ?? "this section"}`
    : "Reset to template";
  const titleLabel = sections ? `Reset ${sectionLabel}?` : "Reset to template?";
  const confirmLabel = sections ? `Reset ${sectionLabel}` : "Reset Tenant";

  const trigger = (
    <Button
      type="button"
      variant="outline"
      size="sm"
      disabled={disabled}
      onClick={() => setOpen(true)}
    >
      <RotateCcw className="h-4 w-4" aria-hidden="true" />
      {triggerLabel}
    </Button>
  );

  return (
    <>
      {/*
        The Tooltip root is ALWAYS mounted and only its content is gated -
        React reconciles by element type at a position, so swapping between a
        bare Button and a Tooltip-wrapped one would remount the button's DOM
        node and silently drop focus. Same reasoning as TemplateReseedButton.
      */}
      <Tooltip>
        <TooltipTrigger asChild>{trigger}</TooltipTrigger>
        {disabled ? (
          <TooltipContent>
            No configuration template resolves for this tenant&apos;s region and
            version, so there is nothing to reset to.
          </TooltipContent>
        ) : null}
      </Tooltip>

      <AlertDialog open={open} onOpenChange={setOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{titleLabel}</AlertDialogTitle>
            <AlertDialogDescription>
              {sections ? (
                <>
                  This replaces {sectionLabel} with the template&apos;s. Edits
                  you have made through the UI to those sections will be lost.
                  The tenant&apos;s id, region, version, world configuration and
                  diagnostics are unchanged, and no game data - accounts,
                  characters, inventories - is affected.
                </>
              ) : (
                <>
                  This replaces every comparable section of this tenant&apos;s
                  configuration with its template&apos;s. Edits you have made
                  through the UI to those sections will be lost. The
                  tenant&apos;s id, region, version, world configuration and
                  diagnostics are unchanged, and no game data - accounts,
                  characters, inventories - is affected.
                </>
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            {/*
              Cancel renders FIRST so it holds the dialog's default focus -
              the destructive action must never be what Enter triggers.
            */}
            <AlertDialogCancel disabled={isResetting}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={(e) => {
                e.preventDefault();
                void onConfirm();
              }}
              disabled={isResetting}
              className={cn(buttonVariants({ variant: "destructive" }))}
            >
              {isResetting ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Resetting...
                </>
              ) : (
                confirmLabel
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
