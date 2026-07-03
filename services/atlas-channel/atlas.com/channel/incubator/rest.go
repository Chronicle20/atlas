package incubator

// RewardRestModel is the JSON:API attribute payload for the atlas-tenants
// incubator-rewards configuration resource.
type RewardRestModel struct {
	Id       string `json:"-"`
	ItemId   uint32 `json:"itemId"`
	Quantity uint32 `json:"quantity"`
	Weight   uint32 `json:"weight"`
}

func (r RewardRestModel) GetName() string { return "incubator-rewards" }
func (r RewardRestModel) GetID() string   { return r.Id }
func (r *RewardRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// SetToOneReferenceID satisfies the api2go UnmarshalToOneRelations interface.
func (r *RewardRestModel) SetToOneReferenceID(_ string, _ string) error { return nil }

// SetToManyReferenceIDs satisfies the api2go UnmarshalToManyRelations interface.
func (r *RewardRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

// Reward is one weighted incubator reward entry.
type Reward struct {
	itemId   uint32
	quantity uint32
	weight   uint32
}

func (r Reward) ItemId() uint32   { return r.itemId }
func (r Reward) Quantity() uint32 { return r.quantity }
func (r Reward) Weight() uint32   { return r.weight }

// Extract converts a RewardRestModel into a Reward, defaulting an unset
// quantity to 1.
func Extract(rm RewardRestModel) (Reward, error) {
	q := rm.Quantity
	if q == 0 {
		q = 1
	}
	return Reward{itemId: rm.ItemId, quantity: q, weight: rm.Weight}, nil
}
