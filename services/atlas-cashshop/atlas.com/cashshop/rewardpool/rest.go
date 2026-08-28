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

// SetToOneReferenceID and SetToManyReferenceIDs are required by
// jsonapi.Unmarshal for any type decoded through requests.PostRequest /
// requests.GetRequest, even when the type declares no relationships — see
// libs/atlas-rest/CLAUDE.md. RewardRestModel has no relationships to
// populate, so these are no-ops (task-207 EXT-01).
func (r *RewardRestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RewardRestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// TransformReward converts the domain Model into its wire representation.
// RewardRestModel carries fields (Tier, Weight, GachaponId) that Model does
// not model, per design.md's rule that only the fields an inverse actually
// maps are round-tripped; SelectReward (processor.go) is the existing
// wire→domain mapping this mirrors, and it maps only ItemId, Quantity, and
// CommodityId.
func TransformReward(m Model) (RewardRestModel, error) {
	return RewardRestModel{
		ItemId:      m.itemId,
		Quantity:    m.quantity,
		CommodityId: m.commodityId,
	}, nil
}
