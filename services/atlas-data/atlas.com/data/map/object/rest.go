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
