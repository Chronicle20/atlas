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

func (rm RestModel) GetID() string {
	return strconv.Itoa(int(rm.Id))
}

func (rm *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	rm.Id = uint32(id)
	return nil
}

func (rm RestModel) GetName() string {
	return "monsters"
}

func Extract(rm RestModel) (SpawnPoint, error) {
	return SpawnPoint{
		Id:       rm.Id,
		Template: rm.Template,
		MobTime:  rm.MobTime,
		Team:     rm.Team,
		Cy:       rm.CY,
		F:        rm.F,
		Fh:       rm.FH,
		Rx0:      rm.RX0,
		Rx1:      rm.RX1,
		X:        rm.X,
		Y:        rm.Y,
	}, nil
}
