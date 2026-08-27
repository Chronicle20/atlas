package foothold

import "strconv"

// pointRestModel is one endpoint of a foothold segment. It is a plain nested
// attribute object (not a JSON:API resource) under "first"/"second" in the
// foothold payload returned by atlas-data.
type pointRestModel struct {
	X int16 `json:"x"`
	Y int16 `json:"y"`
}

// FootholdRestModel mirrors atlas-data's foothold payload
// (services/atlas-data/.../map/rest.go FootholdRestModel) so the "foothold
// below a position" lookup can be decoded here.
type FootholdRestModel struct {
	Id     uint32          `json:"-"`
	First  *pointRestModel `json:"first,omitempty"`
	Second *pointRestModel `json:"second,omitempty"`
}

func (r FootholdRestModel) GetName() string {
	return "footholds"
}

func (r FootholdRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *FootholdRestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// PositionRestModel is the request body: the point to search below.
type PositionRestModel struct {
	Id uint32 `json:"-"`
	X  int16  `json:"x"`
	Y  int16  `json:"y"`
}

func (r PositionRestModel) GetName() string {
	return "positions"
}

func (r PositionRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *PositionRestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// Extract flattens the REST payload into the internal Model.
func Extract(rm FootholdRestModel) (Model, error) {
	return Model{
		id: rm.Id,
		x1: rm.First.X,
		y1: rm.First.Y,
		x2: rm.Second.X,
		y2: rm.Second.Y,
	}, nil
}

// Transform is the exact inverse of Extract: it expands the internal Model
// back into the nested First/Second REST payload.
func Transform(m Model) (FootholdRestModel, error) {
	return FootholdRestModel{
		Id:     m.id,
		First:  &pointRestModel{X: m.x1, Y: m.y1},
		Second: &pointRestModel{X: m.x2, Y: m.y2},
	}, nil
}
