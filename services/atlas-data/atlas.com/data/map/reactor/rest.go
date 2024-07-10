package reactor

type RestModel struct {
	Id              string `json:"-"`
	Name            string `json:"name"`
	X               int16  `json:"x"`
	Y               int16  `json:"y"`
	Delay           uint32 `json:"delay"`
	FacingDirection byte   `json:"facingDirection"`
}

func (r RestModel) GetName() string {
	return "reactors"
}

func (r RestModel) GetID() string {
	return r.Id
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:              m.Id,
		Name:            m.Name,
		X:               m.X,
		Y:               m.Y,
		Delay:           m.Delay,
		FacingDirection: m.FacingDirection,
	}, nil
}
