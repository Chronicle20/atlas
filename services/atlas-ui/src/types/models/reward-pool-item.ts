// Mirrors item/rest.go; id = numeric record id as string.
export interface RewardPoolItemAttributes {
  gachaponId: string;
  itemId: number;
  quantity: number;
  tier: string; // placeholder "common" on incubator items (roll ignores it)
  weight: number; // 0 on classic gachapon items
}

export interface RewardPoolItemData {
  id: string;
  type: string;
  attributes: RewardPoolItemAttributes;
}
