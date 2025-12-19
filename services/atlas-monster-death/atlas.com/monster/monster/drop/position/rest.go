package position

import "strconv"

type RestModel struct {
	Id uint32 `json:"-"`
	X  int16  `json:"x"`
	Y  int16  `json:"y"`
}

func (r RestModel) GetName() string {
	return "points"
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

func Extract(rm RestModel) (Model, error) {
	return Model{
		x: rm.X,
		y: rm.Y,
	}, nil
}

type DropPositionRestModel struct {
	Id        uint32 `json:"-"`
	InitialX  int16  `json:"initialX"`
	InitialY  int16  `json:"initialY"`
	FallbackX int16  `json:"fallbackX"`
	FallbackY int16  `json:"fallbackY"`
}

func (r DropPositionRestModel) GetName() string {
	return "positions"
}

func (r DropPositionRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *DropPositionRestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}
