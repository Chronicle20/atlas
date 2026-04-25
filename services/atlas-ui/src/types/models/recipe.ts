export interface RecipeMaterial {
  itemId: number;
  quantity: number;
}

export interface Recipe {
  id: string;
  npcId: number;
  conversationId: string;
  stateId: string;
  itemId: number;
  materials: RecipeMaterial[];
  mesoCost: number;
  stimulatorId: number;
  stimulatorFailChance: number;
}

export function hasStimulator(recipe: Recipe): boolean {
  return recipe.stimulatorId > 0;
}
