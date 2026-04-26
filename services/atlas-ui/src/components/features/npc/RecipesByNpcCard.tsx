import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useTenant } from "@/context/tenant-context";
import { useNpcRecipes } from "@/lib/hooks/api/useNpcRecipes";
import { itemsService } from "@/services/api/items.service";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import { hasStimulator, type Recipe } from "@/types/models/recipe";

interface RecipesByNpcCardProps {
  npcId: number;
}

export function RecipesByNpcCard({ npcId }: RecipesByNpcCardProps) {
  const { data: recipes, isLoading, error } = useNpcRecipes(npcId);

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium">Crafts</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">Loading craft recipes...</p>
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium">Crafts</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-destructive">{error.message}</p>
        </CardContent>
      </Card>
    );
  }

  if (!recipes || recipes.length === 0) {
    return null;
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium">Crafts ({recipes.length})</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
          {recipes.map((recipe) => (
            <CraftedItemRow key={recipe.id} recipe={recipe} />
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

function CraftedItemRow({ recipe }: { recipe: Recipe }) {
  const { activeTenant } = useTenant();
  const nameQuery = useQuery({
    queryKey: ["items", "name", activeTenant?.id ?? "no-tenant", String(recipe.itemId)],
    queryFn: () => itemsService.getItemName(String(recipe.itemId)),
    enabled: !!activeTenant && recipe.itemId > 0,
    staleTime: 10 * 60 * 1000,
  });
  const itemName = nameQuery.data ?? `Item #${recipe.itemId}`;
  const iconUrl = activeTenant
    ? getAssetIconUrl(
        activeTenant.id,
        activeTenant.attributes.region,
        activeTenant.attributes.majorVersion,
        activeTenant.attributes.minorVersion,
        "item",
        recipe.itemId,
      )
    : undefined;

  return (
    <Link
      to={`/items/${recipe.itemId}`}
      className="flex items-center gap-3 rounded-md border bg-card p-3 hover:bg-accent transition-colors"
    >
      <div className="flex h-8 w-8 shrink-0 items-center justify-center">
        {iconUrl && (
          <img
            src={iconUrl}
            alt={itemName}
            width={32}
            height={32}
            loading="lazy"
            className="max-h-full max-w-full object-contain"
          />
        )}
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium truncate">{itemName}</p>
        <p className="text-xs text-muted-foreground truncate">
          Cost: {recipe.mesoCost.toLocaleString()} mesos
        </p>
      </div>
      {hasStimulator(recipe) && (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <Badge variant="outline">With Stimulator</Badge>
            </TooltipTrigger>
            <TooltipContent>
              <p>Stimulator item #{recipe.stimulatorId}</p>
              <p>Fail chance: {Math.round(recipe.stimulatorFailChance * 100)}%</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      )}
    </Link>
  );
}
