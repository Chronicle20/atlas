// Mirrors global/rest.go — no weight.
export interface GlobalRewardItemAttributes {
  itemId: number;
  quantity: number;
  tier: string;
}

export interface GlobalRewardItemData {
  id: string;
  type: string;
  attributes: GlobalRewardItemAttributes;
}
