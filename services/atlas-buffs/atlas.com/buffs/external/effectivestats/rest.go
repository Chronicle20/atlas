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

// SetToOneReferenceID and SetToManyReferenceIDs are required by api2go
// (jsonapi.Unmarshal) if the upstream response ever carries a
// `relationships` block, even when this client doesn't care about the
// relationship payload. See libs/atlas-rest/CLAUDE.md.
func (r *RestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
