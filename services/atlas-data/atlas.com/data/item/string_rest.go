package item

type StringRestModel struct {
	Id   string `json:"-"`
	Name string `json:"name"`
}

func (r StringRestModel) GetName() string {
	return "item-strings"
}

func (r StringRestModel) GetID() string {
	return r.Id
}

func (r *StringRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func TransformString(m ItemString) (StringRestModel, error) {
	return StringRestModel{
		Id:   m.GetID(),
		Name: m.Name(),
	}, nil
}
