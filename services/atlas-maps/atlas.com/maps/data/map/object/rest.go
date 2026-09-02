package object

// RestModel is atlas-data's projection of a named map object. The resource id
// is the object name, which is what SetObjectState addresses the object by.
type RestModel struct {
	Id    string `json:"-"`
	Name  string `json:"name"`
	State uint32 `json:"state"`
}

func (r RestModel) GetName() string {
	return "objects"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(strId string) error {
	r.Id = strId
	return nil
}

func (r *RestModel) SetToOneReferenceID(_ string, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return NewBuilder().
		SetName(rm.Name).
		SetState(rm.State).
		Build(), nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{Id: m.Name(), Name: m.Name(), State: m.State()}, nil
}
