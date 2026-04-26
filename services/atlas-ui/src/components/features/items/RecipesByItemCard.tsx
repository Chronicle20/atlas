import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { Recipe } from "@/types/models/recipe";
import { RecipeWidget } from "./RecipeWidget";

interface RecipesByItemCardProps {
  recipes: Recipe[] | undefined;
  isLoading: boolean;
  error: Error | null;
}

export function RecipesByItemCard({ recipes, isLoading, error }: RecipesByItemCardProps) {
  const count = recipes?.length ?? 0;
  const title = count > 0 ? `Craftable At (${count})` : "Craftable At";

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <p className="text-sm text-muted-foreground">Loading craft recipes...</p>
        ) : error ? (
          <p className="text-sm text-destructive">{error.message}</p>
        ) : count === 0 ? (
          <p className="text-sm text-muted-foreground">No NPCs craft this item.</p>
        ) : (
          <div className="space-y-3">
            {recipes!.map((recipe) => (
              <RecipeWidget key={recipe.id} recipe={recipe} />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
