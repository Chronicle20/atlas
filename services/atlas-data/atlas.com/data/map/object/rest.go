package object

// RestModel is one named WZ object declared on a map. Id is "{KIND}:{name}",
// deliberately the same composite key task-278's environment-object resource
// uses, so the UI merges the two collections by id rather than by heuristic.
type RestModel struct {
	Id           string `json:"-"`
	Kind         string `json:"kind"`
	Name         string `json:"name"`
	ObjectSource string `json:"objectSource"`
	L0           string `json:"l0"`
	L1           string `json:"l1"`
	L2           string `json:"l2"`
	X            int16  `json:"x"`
	Y            int16  `json:"y"`
	Z            int32  `json:"z"`
	Layer        uint32 `json:"layer"`
}

func (r RestModel) GetName() string {
	return "map-objects"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(strId string) error {
	r.Id = strId
	return nil
}

// Transform maps a domain Model to its REST projection.
func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:           m.Id(),
		Kind:         m.Kind(),
		Name:         m.Name(),
		ObjectSource: m.ObjectSource(),
		L0:           m.L0(),
		L1:           m.L1(),
		L2:           m.L2(),
		X:            m.X(),
		Y:            m.Y(),
		Z:            m.Z(),
		Layer:        m.Layer(),
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
