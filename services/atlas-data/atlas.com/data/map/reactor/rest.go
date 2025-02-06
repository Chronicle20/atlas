package reactor

import "strconv"

type RestModel struct {
	Id             uint32 `json:"-"`
	Classification uint32 `json:"classification"`
	Name           string `json:"name"`
	X              int16  `json:"x"`
	Y              int16  `json:"y"`
	Delay          uint32 `json:"delay"`
	Direction      byte   `json:"direction"`
}

func (r RestModel) GetName() string {
	return "reactors"
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

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:             m.Id,
		Classification: m.Classification,
		Name:           m.Name,
		X:              m.X,
		Y:              m.Y,
		Delay:          m.Delay,
		Direction:      m.Direction,
	}, nil
}
