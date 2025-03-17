package npc

type RestModel struct {
	Id       string `json:"-"`
	Template uint32 `json:"template"`
	Name     string `json:"name"`
	CY       int16  `json:"cy"`
	F        uint32 `json:"f"`
	FH       uint16 `json:"fh"`
	RX0      int16  `json:"rx0"`
	RX1      int16  `json:"rx1"`
	X        int16  `json:"x"`
	Y        int16  `json:"y"`
	Hide     bool   `json:"hide"`
}

func (r RestModel) GetName() string {
	return "npcs"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:       m.Id,
		Template: m.Template,
		Name:     m.Name,
		CY:       m.CY,
		F:        m.F,
		FH:       m.FH,
		RX0:      m.RX0,
		RX1:      m.RX1,
		X:        m.X,
		Y:        m.Y,
		Hide:     m.Hide,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		Id:       rm.Id,
		Template: rm.Template,
		Name:     rm.Name,
		CY:       rm.CY,
		F:        rm.F,
		FH:       rm.FH,
		RX0:      rm.RX0,
		RX1:      rm.RX1,
		X:        rm.X,
		Y:        rm.Y,
		Hide:     rm.Hide,
	}, nil
}
