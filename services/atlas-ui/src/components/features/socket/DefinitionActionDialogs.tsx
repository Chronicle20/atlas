import { useEffect, useRef } from "react";
import { toast } from "sonner";

import { AddDefinitionDialog } from "@/components/features/socket/dialogs/AddDefinitionDialog";
import { CopyDefinitionDialog } from "@/components/features/socket/dialogs/CopyDefinitionDialog";
import { DeleteDefinitionDialog } from "@/components/features/socket/dialogs/DeleteDefinitionDialog";
import { EditDefinitionDialog } from "@/components/features/socket/dialogs/EditDefinitionDialog";
import { MarkUnsupportedDialog } from "@/components/features/socket/dialogs/MarkUnsupportedDialog";
import { ResetToAncestorDialog } from "@/components/features/socket/dialogs/ResetToAncestorDialog";
import type { DrawerAction } from "@/components/features/socket/DefinitionDrawer";
import {
  useSocketMutation,
  type SocketTarget,
} from "@/lib/hooks/api/useSocketObjects";
import { clearUnsupported } from "@/lib/socket/mutate";
import {
  entriesOf,
  type DefinitionKind,
  type SocketObject,
} from "@/lib/socket/model";
import { createErrorFromUnknown } from "@/types/api/errors";

export interface DefinitionActionDialogsProps {
  kind: DefinitionKind;
  /** Every object the acting scope may resolve to - both grid columns on a
   * per-object page, every selected column on the matrix. */
  objects: SocketObject[];
  action: DrawerAction | null;
  onClose: () => void;
  /** Tenant pages only: enables Reset to Ancestor. */
  ancestor?: SocketObject;
}

function targetOf(scope: SocketObject): SocketTarget {
  return { source: scope.source, id: scope.key };
}

/**
 * Renders whichever of the six Task-17 dialogs matches `action.type`, or
 * applies `clearUnsupported` directly - no confirmation dialog exists for it,
 * it is the reversible half of Mark Unsupported and the drawer already fires
 * it without an ellipsis (no follow-up form). One instance per grid
 * (`PacketMatrixPage` and `DefinitionGridPage` each mount exactly one), so
 * the drawer-action -> dialog switch lives in a single place instead of
 * being duplicated across both pages.
 *
 * `open-in` is never passed here - it is a navigation, not a mutation, and is
 * handled by the page before an action ever reaches this component.
 */
export function DefinitionActionDialogs({
  kind,
  objects,
  action,
  onClose,
  ancestor,
}: DefinitionActionDialogsProps) {
  const mutation = useSocketMutation();
  const handledRef = useRef<DrawerAction | null>(null);

  useEffect(() => {
    if (!action || action.type !== "clear-unsupported") return;
    if (handledRef.current === action) return;
    handledRef.current = action;

    const scope = objects.find((o) => o.key === action.scopeKey);
    if (!scope) {
      onClose();
      return;
    }

    let cancelled = false;
    void (async () => {
      try {
        await mutation.mutateAsync({
          target: targetOf(scope),
          apply: (cfg) => clearUnsupported(cfg, kind, action.name),
        });
        if (!cancelled) {
          toast.success(
            `Cleared unsupported for ${action.name} in ${scope.label}`,
          );
        }
      } catch (e) {
        if (!cancelled) toast.error(createErrorFromUnknown(e).message);
      } finally {
        if (!cancelled) onClose();
      }
    })();

    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [action]);

  if (
    !action ||
    action.type === "open-in" ||
    action.type === "clear-unsupported"
  ) {
    return null;
  }

  const scope = objects.find((o) => o.key === action.scopeKey);
  if (!scope) return null;

  const target = targetOf(scope);
  const onOpenChange = (open: boolean) => {
    if (!open) onClose();
  };

  switch (action.type) {
    case "add":
      return (
        <AddDefinitionDialog
          open
          onOpenChange={onOpenChange}
          target={target}
          targetLabel={scope.label}
          kind={kind}
        />
      );

    case "copy":
      return (
        <CopyDefinitionDialog
          open
          onOpenChange={onOpenChange}
          target={target}
          targetLabel={scope.label}
          kind={kind}
          name={action.name}
          sourceObjects={objects.filter((o) => o.key !== scope.key)}
        />
      );

    case "mark-unsupported": {
      const bindingCount = entriesOf(scope, kind).get(action.name)?.length ?? 0;
      return (
        <MarkUnsupportedDialog
          open
          onOpenChange={onOpenChange}
          target={target}
          targetLabel={scope.label}
          kind={kind}
          name={action.name}
          bindingCount={bindingCount}
        />
      );
    }

    case "edit":
    case "delete": {
      if (action.opCodeValue === undefined) return null;
      const opCodeValue = action.opCodeValue;
      const matched = entriesOf(scope, kind)
        .get(action.name)
        ?.find((b) => b.opCodeValue === opCodeValue);
      if (!matched) return null;

      if (action.type === "delete") {
        return (
          <DeleteDefinitionDialog
            open
            onOpenChange={onOpenChange}
            target={target}
            targetLabel={scope.label}
            kind={kind}
            name={action.name}
            opCodeValue={opCodeValue}
            scope={scope}
          />
        );
      }

      return (
        <EditDefinitionDialog
          open
          onOpenChange={onOpenChange}
          target={target}
          targetLabel={scope.label}
          kind={kind}
          name={action.name}
          opCodeValue={opCodeValue}
          initial={{
            opCode: matched.opCode,
            services: matched.services,
            ...(matched.validator !== undefined
              ? { validator: matched.validator }
              : {}),
            ...(matched.fname !== undefined ? { fname: matched.fname } : {}),
            ...(matched.options !== undefined
              ? { options: matched.options }
              : {}),
          }}
        />
      );
    }

    case "reset-to-ancestor":
      if (!ancestor) return null;
      return (
        <ResetToAncestorDialog
          open
          onOpenChange={onOpenChange}
          target={target}
          targetLabel={scope.label}
          kind={kind}
          name={action.name}
          tenant={scope}
          ancestor={ancestor}
        />
      );

    default:
      return null;
  }
}
