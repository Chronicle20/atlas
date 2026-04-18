export interface GachaponRewardAttributes {
  itemId: number;
  quantity: number;
  tier: string;
  gachaponId: string;
}

export interface GachaponRewardData {
  id: string;
  type: string;
  attributes: GachaponRewardAttributes;
}
