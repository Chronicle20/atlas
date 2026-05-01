// services/atlas-ui/src/components/common/CopyableIdHeader.tsx
import type { ReactNode } from "react";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

interface CopyableIdHeaderProps {
  title: string;
  id: string;
  actions?: ReactNode;
}

export function CopyableIdHeader({ title, id, actions }: CopyableIdHeaderProps) {
  return (
    <div className="flex flex-row items-center justify-between gap-4">
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <h2
              tabIndex={0}
              className="text-2xl font-bold tracking-tight cursor-help focus:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded"
            >
              {title}
            </h2>
          </TooltipTrigger>
          <TooltipContent copyable>
            <p>{id}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>

      {actions ? <div className="flex items-center gap-2">{actions}</div> : null}
    </div>
  );
}
