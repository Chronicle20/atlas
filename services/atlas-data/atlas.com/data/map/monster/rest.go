package monster

import "strconv"

type RestModel struct {
	Id       uint32 `json:"-"`
	Template uint32 `json:"template"`
	MobTime  uint32 `json:"mob_time"`
	Team     int32  `json:"team"`
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
	return "monsters"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:       m.ObjectId,
		Template: m.Id,
		MobTime:  m.MobTime,
		Team:     m.Team,
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
