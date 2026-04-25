import { Shield, MapPin } from "lucide-react";
import type { Character } from "@/types/models/character";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

interface Props {
  character: Character;
  onChangeGm: () => void;
  onChangeMap: () => void;
}

export function CharacterPageHeader({ character, onChangeGm, onChangeMap }: Props) {
  const gm = character.attributes.gm ?? 0;
  return (
    <div className="flex flex-row items-center justify-between gap-4">
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <h2
              tabIndex={0}
              className="text-2xl font-bold tracking-tight cursor-help focus:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded"
            >
              {character.attributes.name}
            </h2>
          </TooltipTrigger>
          <TooltipContent copyable>
            <p>{character.id}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>

      <div className="flex items-center gap-2">
        {gm > 0 && <Badge variant="destructive">GM {gm}</Badge>}
        <Button variant="outline" size="sm" onClick={onChangeGm}>
          <Shield className="mr-1 h-4 w-4" />
          {gm > 0 ? "Change GM" : "Promote to GM"}
        </Button>
        <Button variant="outline" size="sm" onClick={onChangeMap}>
          <MapPin className="mr-1 h-4 w-4" />
          Change Map
        </Button>
      </div>
    </div>
  );
}
