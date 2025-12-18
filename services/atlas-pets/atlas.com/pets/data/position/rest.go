package position

import (
	"atlas-pets/point"
	"strconv"
)

type FootholdRestModel struct {
	Id     uint32           `json:"-"`
	First  *point.RestModel `json:"first,omitempty"`
	Second *point.RestModel `json:"second,omitempty"`
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

func Extract(rm FootholdRestModel) (Model, error) {
	return Model{
		id: rm.Id,
		x1: rm.First.X,
		y1: rm.First.Y,
		x2: rm.Second.X,
		y2: rm.Second.Y,
	}, nil
}

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
