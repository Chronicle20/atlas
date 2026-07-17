package configuration

// RestModel is the rankings configuration resource served by atlas-tenants
// at /tenants/{tenantId}/configurations/rankings.
type RestModel struct {
	Id                       string `json:"-"`
	RecomputeIntervalMinutes uint32 `json:"recomputeIntervalMinutes"`
}

func (r RestModel) GetName() string {
	return "rankings"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}
