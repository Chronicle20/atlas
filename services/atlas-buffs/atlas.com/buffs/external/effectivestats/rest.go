package effectivestats

// RestModel is the trimmed atlas-effective-stats projection: buff-inclusive
// max HP (JSON tag maxHP per services/atlas-effective-stats stat/rest.go).
type RestModel struct {
	Id    string `json:"-"`
	MaxHp uint32 `json:"maxHP"`
}

func (r RestModel) GetName() string {
	return "effective-stats"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}
