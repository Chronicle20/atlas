package data

type RestModel struct {
	Id    string `json:"-"`
	Seats uint32 `json:"seats"`
}

func (r RestModel) GetName() string {
	return "maps"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func (r *RestModel) SetToOneReferenceID(name string, ID string) error {
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		seats: rm.Seats,
	}, nil
}
