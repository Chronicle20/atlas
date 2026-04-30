import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { UserCircle2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useNpcData } from "@/lib/hooks/useNpcData";
import { useTenant } from "@/context/tenant-context";
import { itemsService } from "@/services/api/items.service";
import { hasStimulator, type Recipe } from "@/types/models/recipe";
import { EntityWidget } from "@/components/features/quests/EntityWidget";

interface RecipeWidgetProps {
  recipe: Recipe;
}

export function RecipeWidget({ recipe }: RecipeWidgetProps) {
  const { name, iconUrl, isLoading } = useNpcData(recipe.npcId);
  const { activeTenant } = useTenant();

  const stimulatorNameQuery = useQuery({
    queryKey: ["items", "name", activeTenant?.id ?? "no-tenant", String(recipe.stimulatorId)],
    queryFn: () => itemsService.getItemName(String(recipe.stimulatorId)),
    enabled: !!activeTenant && recipe.stimulatorId > 0,
    staleTime: 10 * 60 * 1000,
  });
  const stimulatorName = stimulatorNameQuery.data ?? `Item #${recipe.stimulatorId}`;
  const failPercent = Math.round(recipe.stimulatorFailChance * 100);

  return (
    <div className="flex flex-col gap-3 rounded-md border bg-card p-3">
      <div className="flex items-center gap-3">
        <Link
          to={`/npcs/${recipe.npcId}`}
          className="flex flex-1 items-center gap-3 hover:bg-accent transition-colors rounded-md p-1"
        >
          <div className="flex h-8 w-8 shrink-0 items-center justify-center">
            {iconUrl ? (
              <img
                src={iconUrl}
                alt={name || `NPC ${recipe.npcId}`}
                width={32}
                height={32}
                loading="lazy"
                className="max-h-full max-w-full object-contain"
              />
            ) : (
              <UserCircle2 className="h-7 w-7 text-muted-foreground" />
            )}
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium truncate">
              {isLoading && !name ? `NPC #${recipe.npcId}` : name || `NPC #${recipe.npcId}`}
            </p>
            <p className="text-xs text-muted-foreground truncate">
              Cost: {recipe.mesoCost.toLocaleString()} mesos
            </p>
          </div>
        </Link>
        {hasStimulator(recipe) && (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Badge variant="outline">With Stimulator</Badge>
              </TooltipTrigger>
              <TooltipContent>
                <p>{stimulatorName}</p>
                <p>Fail chance: {failPercent}%</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        )}
      </div>
      {recipe.materials.length > 0 && (
        <div className="flex flex-col gap-2">
          <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
            Ingredients
          </p>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
            {recipe.materials.map((mat, idx) => (
              <EntityWidget
                key={`${mat.itemId}-${idx}`}
                kind="item"
                id={mat.itemId}
                count={mat.quantity}
              />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
