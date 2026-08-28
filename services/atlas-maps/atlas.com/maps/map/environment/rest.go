package environment

import "fmt"

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

func Transform(e ObjectEntry) (RestModel, error) {
	return RestModel{
		Id:    fmt.Sprintf("%s:%s", e.Kind, e.Name),
		Kind:  string(e.Kind),
		Name:  e.Name,
		State: e.State,
	}, nil
}

// TransformSlice maps a slice of ObjectEntry to their REST projections.
// Returns the first transform error encountered, if any.
func TransformSlice(es []ObjectEntry) ([]RestModel, error) {
	out := make([]RestModel, 0, len(es))
	for _, e := range es {
		rm, err := Transform(e)
		if err != nil {
			return nil, err
		}
		out = append(out, rm)
	}
	return out, nil
}
