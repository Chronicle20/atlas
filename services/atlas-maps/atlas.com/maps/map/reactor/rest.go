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

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:        rm.Id,
		name:      rm.Name,
		x:         rm.X,
		y:         rm.Y,
		delay:     rm.Delay,
		direction: rm.FacingDirection,
	}, nil
}
