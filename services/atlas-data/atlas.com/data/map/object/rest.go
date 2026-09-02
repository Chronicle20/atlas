package object

// RestModel exposes a named WZ "obj" entry of a map. Name is the WZ "name"
// property, which is how the client addresses the object in
// CField::OnSetObjectState, and State is the object's declared default (the
// WZ "l2" property) — the state a field reset must restore it to.
type RestModel struct {
	Name  string `json:"name"`
	State uint32 `json:"state"`
}

func (r RestModel) GetName() string {
	return "objects"
}

func (r RestModel) GetID() string {
	return r.Name
}

func (r *RestModel) SetID(id string) error {
	r.Name = id
	return nil
}
