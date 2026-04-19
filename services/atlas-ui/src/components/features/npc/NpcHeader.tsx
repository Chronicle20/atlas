import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

interface NpcHeaderProps {
  npcId: number;
  name?: string | undefined;
  iconUrl?: string | undefined;
}

export function NpcHeader({ npcId, name, iconUrl }: NpcHeaderProps) {
  const displayName = name || `NPC #${npcId}`;
  return (
    <div className="flex items-center gap-3 flex-wrap">
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <span
              tabIndex={0}
              className="inline-flex items-center gap-3 cursor-help focus:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded"
            >
              {iconUrl && (
                <img
                  src={iconUrl}
                  alt={displayName}
                  width={64}
                  height={64}
                  className="object-contain"
                />
              )}
              <h2 className="text-2xl font-bold tracking-tight">{displayName}</h2>
            </span>
          </TooltipTrigger>
          <TooltipContent copyable>
            <p>{npcId}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    </div>
  );
}
