package position

import "strconv"

type FootholdRestModel struct {
	Id uint32 `json:"-"`
	X1 int16  `json:"x1"`
	Y1 int16  `json:"y1"`
	X2 int16  `json:"x2"`
	Y2 int16  `json:"y2"`
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
		x1: rm.X1,
		y1: rm.Y1,
		x2: rm.X2,
		y2: rm.Y2,
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
