package map_

import (
	"strconv"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// RectangleRestModel mirrors atlas-data's map rectangle shape (see
// services/atlas-data/atlas.com/data/map/rest.go:270-275).
type RectangleRestModel struct {
	X      int16 `json:"x"`
	Y      int16 `json:"y"`
	Width  int16 `json:"width"`
	Height int16 `json:"height"`
}

// RestModel mirrors the subset of atlas-data's "maps" resource that
// positioning needs.
type RestModel struct {
	Id      uint32              `json:"-"`
	MapArea *RectangleRestModel `json:"mapArea"`
}

func (r RestModel) GetName() string {
	return "maps"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// SetToOneReferenceID and SetToManyReferenceIDs are required even though
// this client no longer requests any relationship — see
// libs/atlas-rest/CLAUDE.md: api2go errors decoding any resource whose
// response carries a relationships block unless the target struct
// implements these, whether or not the caller cares about the data.
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func Extract(rm RestModel) (Model, error) {
	var area *Rectangle
	if rm.MapArea != nil {
		area = &Rectangle{
			x:      rm.MapArea.X,
			y:      rm.MapArea.Y,
			width:  rm.MapArea.Width,
			height: rm.MapArea.Height,
		}
	}
	return Model{
		id:      _map.Id(rm.Id),
		mapArea: area,
	}, nil
}

// GroundPointRestModel mirrors atlas-data's ground query point shape (see
// services/atlas-data/atlas.com/data/map/rest.go:330-333).
type GroundPointRestModel struct {
	X int16 `json:"x"`
	Y int16 `json:"y"`
}

// GroundRequestRestModel is the POST data/maps/{id}/ground request body
// (Task 4, design §5.3 D-2).
type GroundRequestRestModel struct {
	Id     uint32                 `json:"-"`
	Points []GroundPointRestModel `json:"points"`
}

func (r GroundRequestRestModel) GetName() string {
	return "grounds"
}

func (r GroundRequestRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *GroundRequestRestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func (r *GroundRequestRestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *GroundRequestRestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// GroundResultRestModel mirrors atlas-data's ground response shape (see
// services/atlas-data/atlas.com/data/map/rest.go:357-363).
type GroundResultRestModel struct {
	Id    uint32 `json:"-"`
	X     int16  `json:"x"`
	Y     int16  `json:"y"`
	Fh    uint32 `json:"fh"`
	Found bool   `json:"found"`
}

func (r GroundResultRestModel) GetName() string {
	return "grounds"
}

func (r GroundResultRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *GroundResultRestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func (r *GroundResultRestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *GroundResultRestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func ExtractGroundResult(rm GroundResultRestModel) (GroundResult, error) {
	return GroundResult{
		x:     rm.X,
		y:     rm.Y,
		fh:    rm.Fh,
		found: rm.Found,
	}, nil
}
