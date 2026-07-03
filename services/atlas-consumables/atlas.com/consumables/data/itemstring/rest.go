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
