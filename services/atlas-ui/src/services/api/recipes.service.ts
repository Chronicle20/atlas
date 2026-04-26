import { api } from "@/lib/api/client";
import type { Recipe, RecipeMaterial } from "@/types/models/recipe";

interface RecipeResource {
  id: string;
  attributes: {
    npcId: number;
    conversationId: string;
    stateId: string;
    itemId: number;
    materials: RecipeMaterial[];
    mesoCost: number;
    stimulatorId: number;
    stimulatorFailChance: number;
  };
}

function toRecipe(row: RecipeResource): Recipe {
  return {
    id: row.id,
    npcId: row.attributes.npcId,
    conversationId: row.attributes.conversationId,
    stateId: row.attributes.stateId,
    itemId: row.attributes.itemId,
    materials: row.attributes.materials ?? [],
    mesoCost: row.attributes.mesoCost,
    stimulatorId: row.attributes.stimulatorId,
    stimulatorFailChance: row.attributes.stimulatorFailChance,
  };
}

export const recipesService = {
  async getByItem(itemId: string | number): Promise<Recipe[]> {
    const rows = await api.getList<RecipeResource>(`/api/items/${itemId}/recipes`);
    return rows.map(toRecipe);
  },
  async getByNpc(npcId: string | number): Promise<Recipe[]> {
    const rows = await api.getList<RecipeResource>(`/api/npcs/${npcId}/recipes`);
    return rows.map(toRecipe);
  },
};
