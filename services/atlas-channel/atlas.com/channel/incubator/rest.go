package incubator

// RewardRestModel is the JSON:API attribute payload for the atlas-reward-pools
// gachapon-rewards resource returned by POST
// /gachapons/{gachaponId}/rewards/select.
type RewardRestModel struct {
	Id         string `json:"-"`
	ItemId     uint32 `json:"itemId"`
	Quantity   uint32 `json:"quantity"`
	Tier       string `json:"tier"`
	GachaponId string `json:"gachaponId"`
}

func (r RewardRestModel) GetName() string { return "gachapon-rewards" }
func (r RewardRestModel) GetID() string   { return r.Id }
func (r *RewardRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// SetToOneReferenceID satisfies the api2go UnmarshalToOneRelations interface.
func (r *RewardRestModel) SetToOneReferenceID(_ string, _ string) error { return nil }

// SetToManyReferenceIDs satisfies the api2go UnmarshalToManyRelations interface.
func (r *RewardRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

// Reward is the incubator reward selected by atlas-reward-pools for one roll.
type Reward struct {
	itemId   uint32
	quantity uint32
}

func (r Reward) ItemId() uint32   { return r.itemId }
func (r Reward) Quantity() uint32 { return r.quantity }

// Extract converts a RewardRestModel into a Reward, defaulting an unset
// quantity to 1.
func Extract(rm RewardRestModel) (Reward, error) {
	q := rm.Quantity
	if q == 0 {
		q = 1
	}
	return Reward{itemId: rm.ItemId, quantity: q}, nil
}
