export interface DropAttributes {
  monsterId: number;
  itemId: number;
  minimumQuantity: number;
  maximumQuantity: number;
  questId: number;
  chance: number;
}

export interface DropData {
  id: string;
  type: string;
  attributes: DropAttributes;
}

export interface ReactorDropAttributes {
  reactorId: number;
  itemId: number;
  questId: number;
  chance: number;
}

export interface ReactorDropData {
  id: string;
  type: string;
  attributes: ReactorDropAttributes;
}
