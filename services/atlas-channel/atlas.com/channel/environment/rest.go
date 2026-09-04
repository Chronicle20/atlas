package environment

type RestModel struct {
	Id    string `json:"-"`
	Kind  string `json:"kind"`
	Name  string `json:"name"`
	State uint32 `json:"state"`
}

func (m RestModel) GetID() string {
	return m.Id
}

func (m RestModel) GetName() string {
	return "environment-objects"
}

func (m *RestModel) SetID(idStr string) error {
	m.Id = idStr
	return nil
}

// SetToOneReferenceID and SetToManyReferenceIDs are required by
// api2go/jsonapi's UnmarshalToOneRelations/UnmarshalToManyRelations
// interfaces even though environment objects have no relationships of
// their own -- without them, any response body carrying a `relationships`
// block fails decode instead of a clean fetch error.
func (m *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (m *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// Extract maps the wire RestModel to the domain Model.
func Extract(rm RestModel) (Model, error) {
	return Model{
		kind:  rm.Kind,
		name:  rm.Name,
		state: rm.State,
	}, nil
}

// Transform maps a domain Model to its REST projection.
func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:    m.kind + ":" + m.name,
		Kind:  m.kind,
		Name:  m.name,
		State: m.state,
	}, nil
}

// TransformSlice maps a slice of domain Models to their REST projections.
// Returns the first transform error encountered, if any.
func TransformSlice(ms []Model) ([]RestModel, error) {
	out := make([]RestModel, 0, len(ms))
	for _, m := range ms {
		rm, err := Transform(m)
		if err != nil {
			return nil, err
		}
		out = append(out, rm)
	}
	return out, nil
}
