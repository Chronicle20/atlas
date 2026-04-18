export interface GachaponAttributes {
  name: string;
  npcIds: number[];
  commonWeight: number;
  uncommonWeight: number;
  rareWeight: number;
}

export interface GachaponData {
  id: string;
  type: string;
  attributes: GachaponAttributes;
}
