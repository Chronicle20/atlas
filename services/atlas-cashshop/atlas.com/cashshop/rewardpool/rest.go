package rewardpool

// RewardRestModel mirrors atlas-reward-pools reward/rest.go. CommodityId is
// the cash shop commodity (serial number) a cash-surprise pool awards; the
// commodity catalog owns the reward's itemId/count/period, so the grant path
// resolves the commodity rather than trusting ItemId here.
type RewardRestModel struct {
	Id          string `json:"-"`
	ItemId      uint32 `json:"itemId"`
	Quantity    uint32 `json:"quantity"`
	Tier        string `json:"tier"`
	Weight      uint32 `json:"weight"`
	CommodityId uint32 `json:"commodityId"`
	GachaponId  string `json:"gachaponId"`
}

func (r RewardRestModel) GetName() string { return "gachapon-rewards" }
func (r RewardRestModel) GetID() string   { return r.Id }
func (r *RewardRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}
