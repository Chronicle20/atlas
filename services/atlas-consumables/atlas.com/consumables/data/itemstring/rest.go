package itemstring

type RestModel struct {
	Id   string `json:"-"`
	Name string `json:"name"`
}

func (r RestModel) GetName() string { return "item-strings" }

func (r RestModel) GetID() string { return r.Id }

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func (r *RestModel) SetToOneReferenceID(_, _ string) error { return nil }

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
