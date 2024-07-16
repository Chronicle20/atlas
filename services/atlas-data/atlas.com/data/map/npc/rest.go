package npc

import "strconv"

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

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:       strconv.Itoa(int(m.ObjectId)),
		Template: m.Id,
		Name:     m.Name,
		CY:       m.Cy,
		F:        m.F,
		FH:       m.Fh,
		RX0:      m.Rx0,
		RX1:      m.Rx1,
		X:        m.X,
		Y:        m.Y,
		Hide:     m.Hide,
	}, nil
}
