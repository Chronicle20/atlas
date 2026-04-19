import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

interface MonsterHeaderProps {
  monsterId: string;
  name: string;
  iconUrl?: string | undefined;
  boss: boolean;
  undead: boolean;
  friendly: boolean;
}

export function MonsterHeader({
  monsterId,
  name,
  iconUrl,
  boss,
  undead,
  friendly,
}: MonsterHeaderProps) {
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
                  alt={name}
                  width={64}
                  height={64}
                  className="object-contain"
                />
              )}
              <h2 className="text-2xl font-bold tracking-tight">{name}</h2>
            </span>
          </TooltipTrigger>
          <TooltipContent copyable>
            <p>{monsterId}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
      {boss && <Badge variant="destructive">Boss</Badge>}
      {undead && <Badge variant="secondary">Undead</Badge>}
      {friendly && <Badge variant="outline">Friendly</Badge>}
    </div>
  );
}
