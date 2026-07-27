export type RewardPoolKind = "gachapon" | "incubator";

export interface RewardPoolAttributes {
  name: string;
  kind: RewardPoolKind;
  npcIds: number[];
  commonWeight: number;
  uncommonWeight: number;
  rareWeight: number;
}

export interface RewardPoolData {
  id: string;
  type: string;
  attributes: RewardPoolAttributes;
}
