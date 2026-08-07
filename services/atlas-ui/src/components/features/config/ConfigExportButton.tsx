import { Download } from "lucide-react";
import { toast } from "sonner";

import { useDetailActionBarState } from "@/components/DetailActionBarContext";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useTemplate } from "@/lib/hooks/api/useTemplates";
import { useTenantConfiguration } from "@/lib/hooks/api/useTenants";
import {
  configExportFilename,
  toConfigExportPayload,
  type ConfigExportKind,
} from "@/lib/utils/config-export";
import { downloadJson } from "@/lib/utils/download-json";
import { createErrorFromUnknown } from "@/types/api/errors";

export interface ConfigExportButtonProps {
  kind: ConfigExportKind;
  id: string | undefined;
}

/**
 * Downloads the viewed Template / Tenant configuration as a seed-shaped JSON
 * file. Lives in the detail LAYOUT header, so it is present on every sub-tab
 * without per-page wiring.
 *
 * The payload is read from the React Query cache, so it is the last PERSISTED
 * document - never the page's unsaved form state. That is the point: the file
 * exists to be diffed against, or promoted into, a checked-in seed template.
 */
export function ConfigExportButton({ kind, id }: ConfigExportButtonProps) {
  // Rules of Hooks forbid calling one hook conditionally, so both are called
  // and the irrelevant one is disabled with an empty id (both guard with
  // `enabled: !!id`, so it issues no request). Same pattern as
  // DefinitionGridPage.tsx.
  const templateQuery = useTemplate(kind === "template" ? (id ?? "") : "");
  const tenantQuery = useTenantConfiguration(
    kind === "tenant" ? (id ?? "") : "",
  );
  const query = kind === "template" ? templateQuery : tenantQuery;
  const actionBar = useDetailActionBarState();

  const onExport = () => {
    const data = query.data;
    if (!data) return;
    try {
      downloadJson(
        configExportFilename(kind, {
          id: data.id,
          region: data.attributes.region,
          majorVersion: data.attributes.majorVersion,
          minorVersion: data.attributes.minorVersion,
        }),
        toConfigExportPayload(data.attributes),
      );
      toast.success(
        kind === "template" ? "Template exported" : "Tenant exported",
      );
    } catch (e) {
      // Route through createErrorFromUnknown so the real cause survives into
      // the toast (a JSON.stringify failure on a cyclic/BigInt payload reads
      // as its own message, not a generic string). Same convention as every
      // other async-catch site under components/features/.
      toast.error(createErrorFromUnknown(e, "Export failed").message);
    }
  };

  // Deriving from data presence covers loading, error, refetch-after-error and
  // the no-id case in one predicate.
  const button = (
    <Button
      type="button"
      variant="outline"
      size="sm"
      disabled={!query.data}
      onClick={onExport}
    >
      <Download className="h-4 w-4" aria-hidden="true" />
      Export
    </Button>
  );

  // The Tooltip root is ALWAYS mounted, even when the page is clean and the
  // content is suppressed - React reconciles by element type at a position,
  // so a component that returns a bare <Button> when clean and a <Tooltip>
  // wrapping that same button when dirty would remount the button's DOM node
  // on every dirty/clean flip, silently dropping focus if it was focused.
  // Keeping the root stable and gating only the TooltipContent avoids that.
  // `Tooltip` mounts its own TooltipProvider (components/ui/tooltip.tsx), so
  // none is added here.
  return (
    <Tooltip>
      <TooltipTrigger asChild>{button}</TooltipTrigger>
      {actionBar?.dirty ? (
        <TooltipContent>Exports the last saved configuration</TooltipContent>
      ) : null}
    </Tooltip>
  );
}
