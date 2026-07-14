package frederick

// StatusRestModel is the JSON:API resource for a character's Fredrick
// standing: whether unclaimed items or mesos are waiting. The entrusted-shop
// permit check (atlas-channel) consults it to send the faithful
// "retrieve your items from Fredrick first" reply before allowing a new
// hired merchant.
type StatusRestModel struct {
	Id         string `json:"-"`
	HasPending bool   `json:"hasPending"`
}

func (r StatusRestModel) GetName() string {
	return "frederick-status"
}

func (r StatusRestModel) GetID() string {
	return r.Id
}

func (r *StatusRestModel) SetID(id string) error {
	r.Id = id
	return nil
}
